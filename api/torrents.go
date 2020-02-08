package api

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charly3pins/magnetar/bittorrent"
	"github.com/charly3pins/magnetar/config"
	"github.com/charly3pins/magnetar/util"
	"github.com/charly3pins/magnetar/xbmc"

	"github.com/charly3pins/libtorrent-go"

	"github.com/cloudflare/ahocorasick"
	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
	"github.com/zeebo/bencode"
)

var torrentsLog = logging.MustGetLogger("torrents")

type TorrentsWeb struct {
	Name          string  `json:"name"`
	Size          string  `json:"size"`
	Status        string  `json:"status"`
	Progress      float64 `json:"progress"`
	Ratio         float64 `json:"ratio"`
	TimeRatio     float64 `json:"time_ratio"`
	SeedingTime   string  `json:"seeding_time"`
	SeedTime      float64 `json:"seed_time"`
	SeedTimeLimit int     `json:"seed_time_limit"`
	DownloadRate  float64 `json:"download_rate"`
	UploadRate    float64 `json:"upload_rate"`
	Seeders       int     `json:"seeders"`
	SeedersTotal  int     `json:"seeders_total"`
	Peers         int     `json:"peers"`
	PeersTotal    int     `json:"peers_total"`
}

type TorrentMap struct {
	tmdbId  string
	torrent *bittorrent.Torrent
}

var TorrentsMap []*TorrentMap

func AddToTorrentsMap(tmdbId string, torrent *bittorrent.Torrent) {
	inTorrentsMap := false
	for _, torrentMap := range TorrentsMap {
		if tmdbId == torrentMap.tmdbId {
			inTorrentsMap = true
		}
	}
	if inTorrentsMap == false {
		torrentMap := &TorrentMap{
			tmdbId:  tmdbId,
			torrent: torrent,
		}
		TorrentsMap = append(TorrentsMap, torrentMap)
	}
}

func InTorrentsMap(tmdbId string) (torrents []*bittorrent.Torrent) {
	for index, torrentMap := range TorrentsMap {
		if tmdbId == torrentMap.tmdbId {
			if xbmc.DialogConfirm("Magnetar", "LOCALIZE[30260]") {
				torrents = append(torrents, torrentMap.torrent)
			} else {
				TorrentsMap = append(TorrentsMap[:index], TorrentsMap[index+1:]...)
			}
		}
	}
	return torrents
}

func nameMatch(torrentName string, itemName string) bool {
	patterns := strings.FieldsFunc(strings.ToLower(itemName), func(r rune) bool {
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsMark(r) {
			return true
		}
		return false
	})

	m := ahocorasick.NewStringMatcher(patterns)

	found := m.Match([]byte(strings.ToLower(torrentName)))

	return len(found) >= len(patterns)
}

func ExistingTorrent(btService *bittorrent.BTService, longName string) (existingTorrent string) {
	btService.Session.GetHandle().GetTorrents()
	torrentsVector := btService.Session.GetHandle().GetTorrents()
	torrentsVectorSize := int(torrentsVector.Size())

	for i := 0; i < torrentsVectorSize; i++ {
		torrentHandle := torrentsVector.Get(i)
		if torrentHandle.IsValid() == false {
			continue
		}
		torrentStatus := torrentHandle.Status()
		torrentName := torrentStatus.GetName()

		if nameMatch(torrentName, longName) {
			shaHash := torrentStatus.GetInfoHash().ToString()
			infoHash := hex.EncodeToString([]byte(shaHash))

			torrentFile := filepath.Join(config.Get().TorrentsPath, fmt.Sprintf("%s.torrent", infoHash))
			return torrentFile
		}
	}
	return ""
}

func ListTorrents(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		btService.Session.GetHandle().GetTorrents()
		torrentsVector := btService.Session.GetHandle().GetTorrents()
		torrentsVectorSize := int(torrentsVector.Size())
		items := make(xbmc.ListItems, 0, torrentsVectorSize)

		torrentsLog.Info("Currently downloading:")
		for i := 0; i < torrentsVectorSize; i++ {
			torrentHandle := torrentsVector.Get(i)
			if torrentHandle.IsValid() == false {
				continue
			}

			torrentStatus := torrentHandle.Status()

			torrentName := torrentStatus.GetName()
			progress := float64(torrentStatus.GetProgress()) * 100
			status := bittorrent.StatusStrings[int(torrentStatus.GetState())]

			ratio := float64(0)
			allTimeDownload := float64(torrentStatus.GetAllTimeDownload())
			if allTimeDownload > 0 {
				ratio = float64(torrentStatus.GetAllTimeUpload()) / allTimeDownload
			}

			timeRatio := float64(0)
			finishedTime := float64(torrentStatus.GetFinishedTime())
			downloadTime := float64(torrentStatus.GetActiveTime()) - finishedTime
			if downloadTime > 1 {
				timeRatio = finishedTime / downloadTime
			}

			seedingTime := time.Duration(torrentStatus.GetSeedingTime()) * time.Second
			if progress == 100 && seedingTime == 0 {
				seedingTime = time.Duration(finishedTime) * time.Second
			}

			torrentAction := []string{"LOCALIZE[30231]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/torrents/pause/%d", i))}
			sessionAction := []string{"LOCALIZE[30233]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/torrents/pause"))}

			if btService.Session.GetHandle().IsPaused() {
				status = "Paused"
				sessionAction = []string{"LOCALIZE[30234]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/torrents/resume"))}
			} else if torrentStatus.GetPaused() && status != "Finished" {
				if progress == 100 {
					status = "Finished"
				} else {
					status = "Paused"
				}
				torrentAction = []string{"LOCALIZE[30235]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/torrents/resume/%d", i))}
			} else if !torrentStatus.GetPaused() && (status == "Finished" || progress == 100) {
				status = "Seeding"
			}

			color := "white"
			switch status {
			case "Paused":
				fallthrough
			case "Finished":
				color = "grey"
			case "Seeding":
				color = "green"
			case "Buffering":
				color = "blue"
			case "Finding":
				color = "orange"
			case "Checking":
				color = "teal"
			case "Queued":
			case "Allocating":
				color = "black"
			case "Stalled":
				color = "red"
			}
			torrentsLog.Infof("- %.2f%% - %s - %.2f:1 / %.2f:1 (%s) - %s", progress, status, ratio, timeRatio, seedingTime.String(), torrentName)

			var (
				tmdb        string
				show        string
				season      string
				episode     string
				contentType string
			)
			shaHash := torrentStatus.GetInfoHash().ToString()
			infoHash := hex.EncodeToString([]byte(shaHash))
			dbItem := btService.GetDBItem(infoHash)
			if dbItem != nil && dbItem.Type != "" {
				contentType = dbItem.Type
				if contentType == "movie" {
					tmdb = strconv.Itoa(dbItem.ID)
				} else {
					show = strconv.Itoa(dbItem.ShowID)
					season = strconv.Itoa(dbItem.Season)
					episode = strconv.Itoa(dbItem.Episode)
				}
			}

			playUrl := UrlQuery(UrlForXBMC("/play"),
				"resume", strconv.Itoa(i),
				"type", contentType,
				"tmdb", tmdb,
				"show", show,
				"season", season,
				"episode", episode)

			item := xbmc.ListItem{
				Label: fmt.Sprintf("%.2f%% - [COLOR %s]%s[/COLOR] - %.2f:1 / %.2f:1 (%s) - %s", progress, color, status, ratio, timeRatio, seedingTime.String(), torrentName),
				Path:  playUrl,
				Info: &xbmc.ListItemInfo{
					Title: torrentName,
				},
			}
			item.ContextMenu = [][]string{
				[]string{"LOCALIZE[30230]", fmt.Sprintf("XBMC.PlayMedia(%s)", playUrl)},
				torrentAction,
				[]string{"LOCALIZE[30232]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/torrents/delete/%d", i))},
				[]string{"LOCALIZE[30276]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/torrents/delete/%d?files=1", i))},
				[]string{"LOCALIZE[30308]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/torrents/move/%d", i))},
				sessionAction,
			}
			item.IsPlayable = true
			items = append(items, &item)
		}

		ctx.JSON(200, xbmc.NewView("", items))
	}
}

func ListTorrentsWeb(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		btService.Session.GetHandle().GetTorrents()
		torrentsVector := btService.Session.GetHandle().GetTorrents()
		torrentsVectorSize := int(torrentsVector.Size())
		torrents := make([]*TorrentsWeb, 0, torrentsVectorSize)
		seedTimeLimit := config.Get().SeedTimeLimit

		for i := 0; i < torrentsVectorSize; i++ {
			torrentHandle := torrentsVector.Get(i)
			if torrentHandle.IsValid() == false {
				continue
			}

			torrentStatus := torrentHandle.Status()

			torrentName := torrentStatus.GetName()
			progress := float64(torrentStatus.GetProgress()) * 100

			status := bittorrent.StatusStrings[int(torrentStatus.GetState())]
			if btService.Session.GetHandle().IsPaused() {
				status = "Paused"
			} else if torrentStatus.GetPaused() && status != "Finished" {
				if progress == 100 {
					status = "Finished"
				} else {
					status = "Paused"
				}
			} else if !torrentStatus.GetPaused() && (status == "Finished" || progress == 100) {
				status = "Seeding"
			}

			ratio := float64(0)
			allTimeDownload := float64(torrentStatus.GetAllTimeDownload())
			if allTimeDownload > 0 {
				ratio = float64(torrentStatus.GetAllTimeUpload()) / allTimeDownload
			}

			timeRatio := float64(0)
			finishedTime := float64(torrentStatus.GetFinishedTime())
			downloadTime := float64(torrentStatus.GetActiveTime()) - finishedTime
			if downloadTime > 1 {
				timeRatio = finishedTime / downloadTime
			}
			seedingTime := time.Duration(torrentStatus.GetSeedingTime()) * time.Second
			if progress == 100 && seedingTime == 0 {
				seedingTime = time.Duration(finishedTime) * time.Second
			}

			torrentInfo := torrentHandle.TorrentFile()
			size := ""
			if torrentInfo != nil && torrentInfo.Swigcptr() != 0 {
				size = humanize.Bytes(uint64(torrentInfo.TotalSize()))
			}
			downloadRate := float64(torrentStatus.GetDownloadRate()) / 1024
			uploadRate := float64(torrentStatus.GetUploadRate()) / 1024
			seeders := torrentStatus.GetNumSeeds()
			seedersTotal := torrentStatus.GetNumComplete()
			peers := torrentStatus.GetNumPeers() - seeders
			peersTotal := torrentStatus.GetNumIncomplete()

			torrent := TorrentsWeb{
				Name:          torrentName,
				Size:          size,
				Status:        status,
				Progress:      progress,
				Ratio:         ratio,
				TimeRatio:     timeRatio,
				SeedingTime:   seedingTime.String(),
				SeedTime:      seedingTime.Seconds(),
				SeedTimeLimit: seedTimeLimit,
				DownloadRate:  downloadRate,
				UploadRate:    uploadRate,
				Seeders:       seeders,
				SeedersTotal:  seedersTotal,
				Peers:         peers,
				PeersTotal:    peersTotal,
			}
			torrents = append(torrents, &torrent)
		}

		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.JSON(200, torrents)
	}
}

func PauseSession(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		btService.Session.GetHandle().Pause()
		xbmc.Refresh()
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.String(200, "")
	}
}

func ResumeSession(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		btService.Session.GetHandle().Resume()
		xbmc.Refresh()
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.String(200, "")
	}
}

func AddTorrent(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		uri := ctx.Query("uri")
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")

		if uri == "" {
			ctx.String(404, "Missing torrent URI")
		}
		torrentsLog.Infof("Adding torrent from %s", uri)

		if config.Get().DownloadPath == "." {
			xbmc.Notify("Magnetar", "LOCALIZE[30113]", config.AddonIcon())
			ctx.String(404, "Download path empty")
			return
		}

		torrentParams := libtorrent.NewAddTorrentParams()
		defer libtorrent.DeleteAddTorrentParams(torrentParams)

		var infoHash string

		loadFromFile := false
		torrent := bittorrent.NewTorrent(uri)
		if strings.HasPrefix(uri, "magnet") || strings.HasPrefix(uri, "http") {
			if torrent.IsMagnet() {
				torrent.Magnet()
				torrentsLog.Infof("Parsed magnet: %s", torrent.URI)
				if err := torrent.IsValidMagnet(); err == nil {
					torrentParams.SetUrl(torrent.URI)
				} else {
					ctx.String(404, err.Error())
					return
				}
			} else {
				if err := torrent.Resolve(); err == nil {
					loadFromFile = true
				} else {
					ctx.String(404, err.Error())
					return
				}
			}
			infoHash = torrent.InfoHash
		} else {
			loadFromFile = true
		}

		if loadFromFile {
			if _, err := os.Stat(torrent.URI); err != nil {
				ctx.String(404, err.Error())
				return
			}

			file, err := os.Open(torrent.URI)
			if err != nil {
				ctx.String(404, err.Error())
				return
			}
			dec := bencode.NewDecoder(file)
			var torrentFile *bittorrent.TorrentFileRaw
			if err := dec.Decode(&torrentFile); err != nil {
				errMsg := fmt.Sprintf("Invalid torrent file %s, failed to decode with: %s", torrent.URI, err.Error())
				torrentsLog.Error(errMsg)
				ctx.String(404, errMsg)
				return
			}

			info := libtorrent.NewTorrentInfo(torrent.URI)
			torrentParams.SetTorrentInfo(info)

			shaHash := info.InfoHash().ToString()
			infoHash = hex.EncodeToString([]byte(shaHash))
		}

		torrentsLog.Infof("Setting save path to %s", config.Get().DownloadPath)
		torrentParams.SetSavePath(config.Get().DownloadPath)

		torrentsLog.Infof("Checking for fast resume data in %s.fastresume", infoHash)
		fastResumeFile := filepath.Join(config.Get().TorrentsPath, fmt.Sprintf("%s.fastresume", infoHash))
		if _, err := os.Stat(fastResumeFile); err == nil {
			torrentsLog.Info("Found fast resume data...")
			fastResumeData, err := ioutil.ReadFile(fastResumeFile)
			if err != nil {
				torrentsLog.Error(err.Error())
				ctx.String(404, err.Error())
				return
			}
			fastResumeVector := libtorrent.NewStdVectorChar()
			defer libtorrent.DeleteStdVectorChar(fastResumeVector)
			for _, c := range fastResumeData {
				fastResumeVector.Add(c)
			}
			torrentParams.SetResumeData(fastResumeVector)
		}

		torrentHandle := btService.Session.GetHandle().AddTorrent(torrentParams)

		if torrentHandle == nil {
			ctx.String(404, fmt.Sprintf("Unable to add torrent with URI %s", uri))
			return
		}

		torrentsLog.Infof("Downloading %s", uri)
		btService.SpaceChecked[infoHash] = false

		xbmc.Refresh()
		ctx.String(200, "")
	}
}

func ResumeTorrent(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		torrentsVector := btService.Session.GetHandle().GetTorrents()
		torrentId := ctx.Params.ByName("torrentId")
		torrentIndex, _ := strconv.Atoi(torrentId)
		torrentHandle := torrentsVector.Get(torrentIndex)

		if torrentHandle == nil {
			ctx.Error(errors.New(fmt.Sprintf("Unable to resume torrent with index %d", torrentIndex)))
		}

		status := torrentHandle.Status(uint(libtorrent.TorrentHandleQueryName))

		torrentName := status.GetName()
		torrentsLog.Infof("Resuming %s", torrentName)

		torrentHandle.AutoManaged(true)

		xbmc.Refresh()
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.String(200, "")
	}
}

func MoveTorrent(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		btService.Session.GetHandle().GetTorrents()
		torrentsVector := btService.Session.GetHandle().GetTorrents()
		torrentId := ctx.Params.ByName("torrentId")
		torrentIndex, _ := strconv.Atoi(torrentId)
		torrentHandle := torrentsVector.Get(torrentIndex)
		if torrentHandle.IsValid() == false {
			ctx.Error(errors.New("Invalid torrent handle"))
		}

		torrentsLog.Infof("Marking %s to be moved...", torrentHandle.Status(uint(libtorrent.TorrentHandleQueryName)).GetName())
		btService.MarkedToMove = torrentIndex

		xbmc.Refresh()
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.String(200, "")
	}
}

func PauseTorrent(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		btService.Session.GetHandle().GetTorrents()
		torrentsVector := btService.Session.GetHandle().GetTorrents()
		torrentId := ctx.Params.ByName("torrentId")
		torrentIndex, _ := strconv.Atoi(torrentId)
		torrentHandle := torrentsVector.Get(torrentIndex)
		if torrentHandle.IsValid() == false {
			ctx.Error(errors.New("Invalid torrent handle"))
		}

		torrentsLog.Infof("Pausing torrent %s", torrentHandle.Status(uint(libtorrent.TorrentHandleQueryName)).GetName())
		torrentHandle.AutoManaged(false)
		torrentHandle.Pause(1)

		xbmc.Refresh()
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.String(200, "")
	}
}

func RemoveTorrent(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		deleteFiles := ctx.Query("files")
		torrentId := ctx.Params.ByName("torrentId")
		torrentIndex, _ := strconv.Atoi(torrentId)

		torrentsPath := config.Get().TorrentsPath

		btService.Session.GetHandle().GetTorrents()
		torrentsVector := btService.Session.GetHandle().GetTorrents()
		torrentHandle := torrentsVector.Get(torrentIndex)
		if torrentHandle.IsValid() == false {
			ctx.Error(errors.New("Invalid torrent handle"))
		}

		torrentStatus := torrentHandle.Status(uint(libtorrent.TorrentHandleQuerySavePath) | uint(libtorrent.TorrentHandleQueryName))
		shaHash := torrentStatus.GetInfoHash().ToString()
		infoHash := hex.EncodeToString([]byte(shaHash))

		// Delete torrent file
		torrentFile := filepath.Join(torrentsPath, fmt.Sprintf("%s.torrent", infoHash))
		if _, err := os.Stat(torrentFile); err == nil {
			torrentsLog.Infof("Deleting torrent file at %s", torrentFile)
			defer os.Remove(torrentFile)
		}

		// Delete fast resume data
		fastResumeFile := filepath.Join(torrentsPath, fmt.Sprintf("%s.fastresume", infoHash))
		if _, err := os.Stat(fastResumeFile); err == nil {
			torrentsLog.Infof("Deleting fast resume data at %s", fastResumeFile)
			defer os.Remove(fastResumeFile)
		}

		btService.UpdateDB(bittorrent.Delete, infoHash, 0, "")
		torrentsLog.Infof("Removed %s from database", infoHash)

		askedToDelete := false
		if config.Get().KeepFilesAsk == true && deleteFiles == "" {
			if xbmc.DialogConfirm("Magnetar", "LOCALIZE[30269]") {
				askedToDelete = true
			}
		}

		if config.Get().KeepFilesAfterStop == false || askedToDelete == true || deleteFiles == "true" {
			torrentsLog.Info("Removing the torrent and deleting files...")
			btService.Session.GetHandle().RemoveTorrent(torrentHandle, int(libtorrent.SessionHandleDeleteFiles))
			partsFile := filepath.Join(config.Get().DownloadPath, fmt.Sprintf(".%s.parts", infoHash))
			defer os.Remove(partsFile)
		} else {
			torrentsLog.Info("Removing the torrent without deleting files...")
			btService.Session.GetHandle().RemoveTorrent(torrentHandle, 0)
		}

		xbmc.Refresh()
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.String(200, "")
	}
}

func Versions(btService *bittorrent.BTService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		type Versions struct {
			Version    string `json:"version"`
			Libtorrent string `json:"libtorrent"`
			UserAgent  string `json:"user-agent"`
		}
		versions := Versions{
			Version:    util.Version[1 : len(util.Version)-1],
			Libtorrent: libtorrent.Version(),
			UserAgent:  btService.UserAgent,
		}
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.JSON(200, versions)
	}
}
