package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/charly3pins/magnetar/providers"
	"github.com/charly3pins/magnetar/tmdb"
	"github.com/charly3pins/magnetar/xbmc"

	"github.com/gin-gonic/gin"
	"github.com/op/go-logging"
)

var providerLog = logging.MustGetLogger("provider")

type providerDebugResponse struct {
	Payload interface{} `json:"payload"`
	Results interface{} `json:"results"`
}

func ProviderGetMovie(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("tmdbId")
	provider := ctx.Params.ByName("provider")
	providerLog.Infof("Searching links for: %s", tmdbId)
	movie := tmdb.GetMovieById(tmdbId, "en")
	providerLog.Infof("Resolved %s to %s", tmdbId, movie.Title)

	searcher := providers.NewAddonSearcher(provider)
	torrents := searcher.SearchMovieLinks(movie)
	if ctx.Query("resolve") == "true" {
		for _, torrent := range torrents {
			torrent.Resolve()
		}
	}
	data, err := json.MarshalIndent(providerDebugResponse{
		Payload: searcher.GetMovieSearchObject(movie),
		Results: torrents,
	}, "", "    ")
	if err != nil {
		xbmc.AddonFailure(provider)
		ctx.Error(err)
	}
	ctx.Data(200, "application/json", data)
}

func ProviderGetEpisode(ctx *gin.Context) {
	provider := ctx.Params.ByName("provider")
	showId, _ := strconv.Atoi(ctx.Params.ByName("showId"))
	seasonNumber, _ := strconv.Atoi(ctx.Params.ByName("season"))
	episodeNumber, _ := strconv.Atoi(ctx.Params.ByName("episode"))

	providerLog.Infof("Searching links for TMDB Id: %s", showId)

	show := tmdb.GetShow(showId, "en")
	season := tmdb.GetSeason(showId, seasonNumber, "en")
	if season == nil {
		ctx.Error(errors.New(fmt.Sprintf("Unable to get season %d", seasonNumber)))
		return
	}
	episode := season.Episodes[episodeNumber-1]

	providerLog.Infof("Resolved %d to %s", showId, show.Name)

	searcher := providers.NewAddonSearcher(provider)
	torrents := searcher.SearchEpisodeLinks(show, episode)
	if ctx.Query("resolve") == "true" {
		for _, torrent := range torrents {
			torrent.Resolve()
		}
	}
	data, err := json.MarshalIndent(providerDebugResponse{
		Payload: searcher.GetEpisodeSearchObject(show, episode),
		Results: torrents,
	}, "", "    ")
	if err != nil {
		xbmc.AddonFailure(provider)
		ctx.Error(err)
	}
	ctx.Data(200, "application/json", data)
}
