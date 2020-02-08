package tmdb

import (
	"fmt"
	"math/rand"
	"path"
	"time"

	"github.com/charly3pins/magnetar/cache"
	"github.com/charly3pins/magnetar/config"
	"github.com/charly3pins/magnetar/xbmc"

	"github.com/jmcvetta/napping"
)

func GetSeason(showId int, seasonNumber int, language string) *Season {
	var season *Season
	cacheStore := cache.NewFileStore(path.Join(config.Get().ProfilePath, "cache"))
	key := fmt.Sprintf("com.tmdb.season.%d.%d.%s", showId, seasonNumber, language)
	if err := cacheStore.Get(key, &season); err != nil {
		rateLimiter.Call(func() {
			urlValues := napping.Params{
				"api_key":            apiKey,
				"append_to_response": "credits,images,videos,external_ids",
				"language":           language,
			}.AsUrlValues()
			resp, err := napping.Get(
				fmt.Sprintf("%stv/%d/season/%d", tmdbEndpoint, showId, seasonNumber),
				&urlValues,
				&season,
				nil,
			)
			if err != nil {
				log.Error(err.Error())
				xbmc.Notify("Magnetar", err.Error(), config.AddonIcon())
			} else if resp.Status() == 429 {
				log.Warningf("Rate limit exceeded getting season %d of show %d, cooling down...", seasonNumber, showId)
				rateLimiter.CoolDown(resp.HttpResponse().Header)
			} else if resp.Status() != 200 {
				message := fmt.Sprintf("Bad status getting season %d of show %d: %d", seasonNumber, showId, resp.Status())
				log.Error(message)
				xbmc.Notify("Magnetar", message, config.AddonIcon())
			}
		})
		season.EpisodeCount = len(season.Episodes)
		if season.EpisodeCount > 0 {
			for index := 0; index < season.EpisodeCount; index++ {
				if season.Episodes[index].Name == "" {
					season.Episodes[index] = GetEpisode(showId, seasonNumber, index+1, "")
				}
			}
		}

		if season != nil {
			updateFrequency := config.Get().UpdateFrequency * 60
			traktFrequency := config.Get().TraktSyncFrequency * 60
			if updateFrequency == 0 && traktFrequency == 0 {
				updateFrequency = 1440
			} else if updateFrequency > traktFrequency && traktFrequency != 0 {
				updateFrequency = traktFrequency - 1
			} else {
				updateFrequency = updateFrequency - 1
			}
			cacheStore.Set(key, season, time.Duration(updateFrequency)*time.Minute)
		}
	}
	return season
}

func (seasons SeasonList) ToListItems(show *Show) []*xbmc.ListItem {
	items := make([]*xbmc.ListItem, 0, len(seasons))

	fanarts := make([]string, 0)
	for _, backdrop := range show.Images.Backdrops {
		fanarts = append(fanarts, ImageURL(backdrop.FilePath, "w1280"))
	}

	now := time.Now().UTC()
	for _, season := range seasons {
		if season.EpisodeCount == 0 {
			continue
		}
		if config.Get().ShowUnairedSeasons == false {
			firstAired, _ := time.Parse("2006-01-02", season.AirDate)
			if firstAired.After(now) {
				continue
			}
		}

		item := season.ToListItem(show)

		if len(fanarts) > 0 {
			item.Art.FanArt = fanarts[rand.Intn(len(fanarts))]
		}

		items = append(items, item)
	}
	return items
}

func (season *Season) ToListItem(show *Show) *xbmc.ListItem {
	name := fmt.Sprintf("Season %d", season.Season)
	if season.Season == 0 {
		name = "Specials"
	}

	item := &xbmc.ListItem{
		Label: name,
		Info: &xbmc.ListItemInfo{
			Count:         rand.Int(),
			Title:         name,
			OriginalTitle: name,
			Season:        season.Season,
			TVShowTitle:   show.OriginalName,
			Plot:          show.Overview,
			PlotOutline:   show.Overview,
			DBTYPE:        "season",
			Mediatype:     "season",
			Code:          show.ExternalIDs.IMDBId,
			IMDBNumber:    show.ExternalIDs.IMDBId,
		},
		Art: &xbmc.ListItemArt{},
	}

	if season.Poster != "" {
		item.Art.Poster = ImageURL(season.Poster, "w500")
		item.Art.Thumbnail = ImageURL(season.Poster, "w500")
	}

	fanarts := make([]string, 0)
	for _, backdrop := range show.Images.Backdrops {
		fanarts = append(fanarts, ImageURL(backdrop.FilePath, "w1280"))
	}
	if len(fanarts) > 0 {
		item.Art.FanArt = fanarts[rand.Intn(len(fanarts))]
	}

	if len(show.Genres) > 0 {
		item.Info.Genre = show.Genres[0].Name
	}

	return item
}

func (s SeasonList) Len() int           { return len(s) }
func (s SeasonList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s SeasonList) Less(i, j int) bool { return s[i].Season < s[j].Season }
