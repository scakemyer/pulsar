package api

import (
	"os"
	"fmt"
	"path"
	"time"
	"strconv"
	"strings"
	"path/filepath"

	"github.com/op/go-logging"
	"github.com/gin-gonic/gin"
	"github.com/charly3pins/quasar/config"
	"github.com/charly3pins/quasar/cache"
	"github.com/charly3pins/quasar/trakt"
	"github.com/charly3pins/quasar/xbmc"
	"github.com/charly3pins/quasar/tmdb"
)

var traktLog = logging.MustGetLogger("trakt")

func inMoviesWatched(tmdbId int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	if len(trakt.WatchedMoviesMap) == 0 {
		traktLog.Info("inMoviesWatched fetching cache for the first time")
		trakt.WatchedMovies()
	}

	if _, ok := trakt.WatchedMoviesMap[trakt.WatchedMoviesCache{tmdbId}]; ok {
		return true
	}

	return false
}

func addToMoviesWatched(tmdbId int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	if _, ok := trakt.WatchedMoviesMap[trakt.WatchedMoviesCache{tmdbId}]; !ok {
		traktLog.Infof("adding %d to cache", tmdbId)
		trakt.WatchedMoviesMap[trakt.WatchedMoviesCache{tmdbId}] = 1
	}

	return true
}

func removeFromMoviesWatched(tmdbId int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	if _, ok := trakt.WatchedMoviesMap[trakt.WatchedMoviesCache{tmdbId}]; ok {
		traktLog.Infof("removing %d from cache", tmdbId)
		delete(trakt.WatchedMoviesMap, trakt.WatchedMoviesCache{tmdbId})
	}

	return true
}

func inEpisodesWatched(showId, seasonNumber, episodeNumber int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	if len(trakt.WatchedEpisodesMap) == 0 {
		traktLog.Info("inEpisodesWatched fetching cache for the first time")
		trakt.WatchedEpisodes()
	}

	if _, ok := trakt.WatchedEpisodesMap[trakt.WatchedEpisodesCache{showId, seasonNumber, episodeNumber}]; ok {
		return true
	}

	return false
}

func addToEpisodesWatched(showId, seasonNumber, episodeNumber int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	if _, ok := trakt.WatchedEpisodesMap[trakt.WatchedEpisodesCache{showId, seasonNumber, episodeNumber}]; !ok {
		traktLog.Infof("adding show:%d season:%d episode:%d to cache", showId, seasonNumber, episodeNumber)
		trakt.WatchedEpisodesMap[trakt.WatchedEpisodesCache{showId, seasonNumber, episodeNumber}] = 1
	}

	return true
}

func removeFromEpisodesWatched(showId, seasonNumber, episodeNumber int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	if _, ok := trakt.WatchedEpisodesMap[trakt.WatchedEpisodesCache{showId, seasonNumber, episodeNumber}]; ok {
		traktLog.Infof("removing show:%d season:%d episode:%d from cache", showId, seasonNumber, episodeNumber)
		delete(trakt.WatchedEpisodesMap, trakt.WatchedEpisodesCache{showId, seasonNumber, episodeNumber})
	}

	return true
}

func inMoviesWatchlist(tmdbId int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	var movies []*trakt.Movies

	cacheStore := cache.NewFileStore(path.Join(config.Get().ProfilePath, "cache"))
	key := fmt.Sprintf("com.trakt.watchlist.movies")
	if err := cacheStore.Get(key, &movies); err != nil {
		movies, _ := trakt.WatchlistMovies()
		cacheStore.Set(key, movies, 30 * time.Second)
	}

	for _, movie := range movies {
		if tmdbId == movie.Movie.IDs.TMDB {
			return true
		}
	}
	return false
}

func inShowsWatchlist(tmdbId int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	var shows []*trakt.Shows

	cacheStore := cache.NewFileStore(path.Join(config.Get().ProfilePath, "cache"))
	key := fmt.Sprintf("com.trakt.watchlist.shows")
	if err := cacheStore.Get(key, &shows); err != nil {
		shows, _ := trakt.WatchlistShows()
		cacheStore.Set(key, shows, 30 * time.Second)
	}

	for _, show := range shows {
		if tmdbId == show.Show.IDs.TMDB {
			return true
		}
	}
	return false
}

func inMoviesCollection(tmdbId int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	var movies []*trakt.Movies

	cacheStore := cache.NewFileStore(path.Join(config.Get().ProfilePath, "cache"))
	key := fmt.Sprintf("com.trakt.collection.movies")
	if err := cacheStore.Get(key, &movies); err != nil {
		movies, _ := trakt.CollectionMovies()
		cacheStore.Set(key, movies, 30 * time.Second)
	}

	for _, movie := range movies {
		if tmdbId == movie.Movie.IDs.TMDB {
			return true
		}
	}
	return false
}

func inShowsCollection(tmdbId int) bool {
	if config.Get().TraktToken == "" {
		return false
	}

	var shows []*trakt.Shows

	cacheStore := cache.NewFileStore(path.Join(config.Get().ProfilePath, "cache"))
	key := fmt.Sprintf("com.trakt.collection.shows")
	if err := cacheStore.Get(key, &shows); err != nil {
		shows, _ := trakt.CollectionShows()
		cacheStore.Set(key, shows, 30 * time.Second)
	}

	for _, show := range shows {
		if tmdbId == show.Show.IDs.TMDB {
			return true
		}
	}
	return false
}

//
// Authorization
//
func AuthorizeTrakt(ctx *gin.Context) {
	err := trakt.Authorize(true)
	if err == nil {
		ctx.String(200, "")
	} else {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
		ctx.String(200, "")
	}
}

//
// Main lists
//
func WatchlistMovies(ctx *gin.Context) {
	movies, err := trakt.WatchlistMovies()
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, -1, 0)
}

func WatchlistShows(ctx *gin.Context) {
	shows, err := trakt.WatchlistShows()
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, -1, 0)
}

func CollectionMovies(ctx *gin.Context) {
	movies, err := trakt.CollectionMovies()
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, -1, 0)
}

func CollectionShows(ctx *gin.Context) {
	shows, err := trakt.CollectionShows()
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, -1, 0)
}

func UserlistMovies(ctx *gin.Context) {
	listId := ctx.Params.ByName("listId")
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, err := trakt.ListItemsMovies(listId, true)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, -1, page)
}

func UserlistShows(ctx *gin.Context) {
	listId := ctx.Params.ByName("listId")
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, err := trakt.ListItemsShows(listId, true)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, -1, page)
}

// func WatchlistSeasons(ctx *gin.Context) {
// 	renderTraktSeasons(trakt.Watchlist("seasons", pageParam), ctx, page)
// }

// func WatchlistEpisodes(ctx *gin.Context) {
// 	renderTraktEpisodes(trakt.Watchlist("episodes", pageParam), ctx, page)
// }

//
// Main lists actions
//
func AddMovieToWatchlist(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("tmdbId")
	resp, err := trakt.AddToWatchlist("movies", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Movie added to watchlist", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.watchlist.movies"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.movies.watchlist"))
		clearPageCache(ctx)
	}
}

func RemoveMovieFromWatchlist(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("tmdbId")
	resp, err := trakt.RemoveFromWatchlist("movies", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Movie removed from watchlist", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.watchlist.movies"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.movies.watchlist"))
		clearPageCache(ctx)
	}
}

func AddShowToWatchlist(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("showId")
	resp, err := trakt.AddToWatchlist("shows", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed %d", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Show added to watchlist", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.watchlist.shows"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.shows.watchlist"))
		clearPageCache(ctx)
	}
}

func RemoveShowFromWatchlist(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("showId")
	resp, err := trakt.RemoveFromWatchlist("shows", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Show removed from watchlist", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.watchlist.shows"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.shows.watchlist"))
		clearPageCache(ctx)
	}
}

func AddMovieToCollection(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("tmdbId")
	resp, err := trakt.AddToCollection("movies", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Movie added to collection", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.collection.movies"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.movies.collection"))
		clearPageCache(ctx)
	}
}

func RemoveMovieFromCollection(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("tmdbId")
	resp, err := trakt.RemoveFromCollection("movies", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Movie removed from collection", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.collection.movies"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.movies.collection"))
		clearPageCache(ctx)
	}
}

func AddShowToCollection(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("showId")
	resp, err := trakt.AddToCollection("shows", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Show added to collection", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.collection.shows"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.shows.collection"))
		clearPageCache(ctx)
	}
}

func RemoveShowFromCollection(ctx *gin.Context) {
	tmdbId := ctx.Params.ByName("showId")
	resp, err := trakt.RemoveFromCollection("shows", tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Show removed from collection", config.AddonIcon())
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.collection.shows"))
		os.Remove(filepath.Join(config.Get().Info.Profile, "cache", "com.trakt.shows.collection"))
		clearPageCache(ctx)
	}
}

func MarkMovieWatchedInTrakt(ctx *gin.Context) {
	tmdbId, _ := strconv.Atoi(ctx.Params.ByName("tmdbId"))
	resp, err := trakt.AddMovieToWatchedHistory2(tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Movie marked as watched in Trakt", config.AddonIcon())
	}
	addToMoviesWatched(tmdbId)
	xbmc.Refresh()
}

func MarkMovieUnwatchedInTrakt(ctx *gin.Context) {
	tmdbId, _ := strconv.Atoi(ctx.Params.ByName("tmdbId"))
	resp, err := trakt.RemoveMovieFromWatchedHistory2(tmdbId)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Movie marked as unwatched in Trakt", config.AddonIcon())
	}
	removeFromMoviesWatched(tmdbId)
	xbmc.Refresh()
}

func MarkShowWatchedInTrakt(ctx *gin.Context) {
	showId, _ := strconv.Atoi(ctx.Params.ByName("showId"))
	resp, err := trakt.AddEpisodeToWatchedHistory2(showId, -1, -1)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Episode marked as watched in Trakt", config.AddonIcon())
	}
}

func MarkShowUnwatchedInTrakt(ctx *gin.Context) {
	showId, _ := strconv.Atoi(ctx.Params.ByName("showId"))
	resp, err := trakt.RemoveEpisodeFromWatchedHistory2(showId, -1, -1)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Episode marked as unwatched in Trakt", config.AddonIcon())
	}
}

func MarkSeasonWatchedInTrakt(ctx *gin.Context) {
	showId, _ := strconv.Atoi(ctx.Params.ByName("showId"))
	season, _ := strconv.Atoi(ctx.Params.ByName("season"))
	resp, err := trakt.AddEpisodeToWatchedHistory2(showId, season, -1)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Episode marked as watched in Trakt", config.AddonIcon())
	}
}

func MarkSeasonUnwatchedInTrakt(ctx *gin.Context) {
	showId, _ := strconv.Atoi(ctx.Params.ByName("showId"))
	season, _ := strconv.Atoi(ctx.Params.ByName("season"))
	resp, err := trakt.RemoveEpisodeFromWatchedHistory2(showId, season, -1)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Episode marked as unwatched in Trakt", config.AddonIcon())
	}
}

func MarkEpisodeWatchedInTrakt(ctx *gin.Context) {
	showId, _ := strconv.Atoi(ctx.Params.ByName("showId"))
	season, _ := strconv.Atoi(ctx.Params.ByName("season"))
	episode, _ := strconv.Atoi(ctx.Params.ByName("episode"))
	resp, err := trakt.AddEpisodeToWatchedHistory2(showId, season, episode)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 201 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Episode marked as watched in Trakt", config.AddonIcon())
	}
	addToEpisodesWatched(showId, season, episode)
	xbmc.Refresh()
}

func MarkEpisodeUnwatchedInTrakt(ctx *gin.Context) {
	showId, _ := strconv.Atoi(ctx.Params.ByName("showId"))
	season, _ := strconv.Atoi(ctx.Params.ByName("season"))
	episode, _ := strconv.Atoi(ctx.Params.ByName("episode"))
	resp, err := trakt.RemoveEpisodeFromWatchedHistory2(showId, season, episode)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	} else if resp.Status() != 200 {
		xbmc.Notify("Quasar", fmt.Sprintf("Failed with %d status code", resp.Status()), config.AddonIcon())
	} else {
		xbmc.Notify("Quasar", "Episode marked as unwatched in Trakt", config.AddonIcon())
	}
	removeFromEpisodesWatched(showId, season, episode)
	xbmc.Refresh()
}

// func AddEpisodeToWatchlist(ctx *gin.Context) {
// 	tmdbId := ctx.Params.ByName("episodeId")
// 	resp, err := trakt.AddToWatchlist("episodes", tmdbId)
// 	if err != nil {
// 		xbmc.Notify("Quasar", fmt.Sprintf("Failed: %s", err), config.AddonIcon())
// 	} else if resp.Status() != 201 {
// 		xbmc.Notify("Quasar", fmt.Sprintf("Failed: %d", resp.Status()), config.AddonIcon())
// 	} else {
// 		xbmc.Notify("Quasar", "Episode added to watchlist", config.AddonIcon())
// 	}
// }

func renderTraktMovies(ctx *gin.Context, movies []*trakt.Movies, total int, page int) {
	hasNextPage := 0
	if page > 0 {
		resultsPerPage := config.Get().ResultsPerPage

		if total == -1 {
			total = len(movies)
		}
		if total > resultsPerPage {
			if page * resultsPerPage < total {
				hasNextPage = 1
			}
		}

		if len(movies) > resultsPerPage {
			start := (page - 1) % trakt.PagesAtOnce * resultsPerPage
			movies = movies[start:start + resultsPerPage]
		}
	}

	items := make(xbmc.ListItems, 0, len(movies) + hasNextPage)

	for _, movieListing := range movies {
		if movieListing == nil {
			continue
		}
		movie := movieListing.Movie
		if movie == nil {
			continue
		}
		item := movie.ToListItem()

		playLabel := "LOCALIZE[30023]"
		playURL := UrlForXBMC("/movie/%d/play", movie.IDs.TMDB)
		linksLabel := "LOCALIZE[30202]"
		linksURL := UrlForXBMC("/movie/%d/links", movie.IDs.TMDB)
		markWatchedLabel := "LOCALIZE[30313]"
		markWatchedURL := UrlForXBMC("/movie/%d/trakt/watched", movie.IDs.TMDB)
		markUnwatchedLabel := "LOCALIZE[30314]"
		markUnwatchedURL := UrlForXBMC("/movie/%d/trakt/unwatched", movie.IDs.TMDB)
 

		defaultURL := linksURL
		contextLabel := playLabel
		contextURL := playURL
		if config.Get().ChooseStreamAuto == true {
			defaultURL = playURL
			contextLabel = linksLabel
			contextURL = linksURL
		}

		item.Path = defaultURL
		if item.Info.Trailer != "" {
			if strings.Contains(item.Info.Trailer, "?v=") {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", strings.Split(item.Info.Trailer, "?v=")[1])
			} else {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", item.Info.Trailer)
			}
		}

		tmdbId := strconv.Itoa(movie.IDs.TMDB)
		libraryAction := []string{"LOCALIZE[30252]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/movie/add/%d", movie.IDs.TMDB))}
		if _, err := isDuplicateMovie(tmdbId); err != nil || isAddedToLibrary(tmdbId, Movie) {
			libraryAction = []string{"LOCALIZE[30253]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/movie/remove/%d", movie.IDs.TMDB))}
		}

		watchlistAction := []string{"LOCALIZE[30255]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/watchlist/add", movie.IDs.TMDB))}
		if inMoviesWatchlist(movie.IDs.TMDB) {
			watchlistAction = []string{"LOCALIZE[30256]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/watchlist/remove", movie.IDs.TMDB))}
		}

		collectionAction := []string{"LOCALIZE[30258]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/collection/add", movie.IDs.TMDB))}
		if inMoviesCollection(movie.IDs.TMDB) {
			collectionAction = []string{"LOCALIZE[30259]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/collection/remove", movie.IDs.TMDB))}
		}

		markAction := []string{markWatchedLabel, fmt.Sprintf("XBMC.RunPlugin(%s)", markWatchedURL)}
		if inMoviesWatched(movie.IDs.TMDB) {
			item.Info.Overlay = xbmc.IconOverlayWatched
			item.Info.PlayCount = 1
			markAction = []string{markUnwatchedLabel, fmt.Sprintf("XBMC.RunPlugin(%s)", markUnwatchedURL)}
		}

		item.ContextMenu = [][]string{
			[]string{contextLabel, fmt.Sprintf("XBMC.PlayMedia(%s)", contextURL)},
			libraryAction,
			watchlistAction,
			collectionAction,
			[]string{"LOCALIZE[30034]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/setviewmode/movies"))},
			markAction,
		}
		if config.Get().Platform.Kodi < 17 {
			item.ContextMenu = append(item.ContextMenu, []string{"LOCALIZE[30203]", "XBMC.Action(Info)"})
			item.ContextMenu = append(item.ContextMenu, []string{"LOCALIZE[30268]", "XBMC.Action(ToggleWatched)"})
		}
		item.IsPlayable = true
		items = append(items, item)
	}
	if page >= 0 && hasNextPage > 0 {
		path := ctx.Request.URL.Path
		nextpage := &xbmc.ListItem{
			Label: "LOCALIZE[30218]",
			Path: UrlForXBMC(fmt.Sprintf("%s?page=%d", path, page + 1)),
			Thumbnail: config.AddonResource("img", "nextpage.png"),
		}
		items = append(items, nextpage)
	}
	ctx.JSON(200, xbmc.NewView("movies", items))
}

func TraktPopularMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.TopMovies("popular", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, total, page)
}

func TraktTrendingMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.TopMovies("trending", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, total, page)
}

func TraktMostPlayedMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.TopMovies("played", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, total, page)
}

func TraktMostWatchedMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.TopMovies("watched", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, total, page)
}

func TraktMostCollectedMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.TopMovies("collected", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, total, page)
}

func TraktMostAnticipatedMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.TopMovies("anticipated", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, total, page)
}

func TraktBoxOffice(ctx *gin.Context) {
	movies, _, err := trakt.TopMovies("boxoffice", "1")
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktMovies(ctx, movies, -1, 0)
}


func renderTraktShows(ctx *gin.Context, shows []*trakt.Shows, total int, page int) {
	hasNextPage := 0
	if page > 0 {
		resultsPerPage := config.Get().ResultsPerPage

		if total == -1 {
			total = len(shows)
		}
		if total > resultsPerPage {
			if page * resultsPerPage < total {
				hasNextPage = 1
			}
		}

		if len(shows) >= resultsPerPage {
			start := (page - 1) % trakt.PagesAtOnce * resultsPerPage
			shows = shows[start:start + resultsPerPage]
		}
	}

	items := make(xbmc.ListItems, 0, len(shows) + hasNextPage)

	for _, showListing := range shows {
		if showListing == nil {
			continue
		}
		show := showListing.Show
		if show == nil {
			continue
		}
		item := show.ToListItem()
		item.Path = UrlForXBMC("/show/%d/seasons", show.IDs.TMDB)
		if item.Info.Trailer != "" {
			if strings.Contains(item.Info.Trailer, "?v=") {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", strings.Split(item.Info.Trailer, "?v=")[1])
			} else {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", item.Info.Trailer)
			}
		}

		tmdbId := strconv.Itoa(show.IDs.TMDB)

		libraryAction := []string{"LOCALIZE[30252]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/show/add/%d", show.IDs.TMDB))}
		if _, err := isDuplicateShow(tmdbId); err != nil || isAddedToLibrary(tmdbId, Show) {
			libraryAction = []string{"LOCALIZE[30253]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/show/remove/%d", show.IDs.TMDB))}
		}
		mergeAction := []string{"LOCALIZE[30283]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/show/add/%d?merge=true", show.IDs.TMDB))}

		watchlistAction := []string{"LOCALIZE[30255]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/watchlist/add", show.IDs.TMDB))}
		if inShowsWatchlist(show.IDs.TMDB) {
			watchlistAction = []string{"LOCALIZE[30256]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/watchlist/remove", show.IDs.TMDB))}
		}

		collectionAction := []string{"LOCALIZE[30258]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/collection/add", show.IDs.TMDB))}
		if inShowsCollection(show.IDs.TMDB) {
			collectionAction = []string{"LOCALIZE[30259]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/collection/remove", show.IDs.TMDB))}
		}

		item.ContextMenu = [][]string{
			libraryAction,
			mergeAction,
			watchlistAction,
			collectionAction,
			[]string{"LOCALIZE[30035]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/setviewmode/tvshows"))},
		}
		if config.Get().Platform.Kodi < 17 {
			item.ContextMenu = append(item.ContextMenu, []string{"LOCALIZE[30203]", "XBMC.Action(Info)"})
		}
		items = append(items, item)
	}
	if page >= 0 && hasNextPage > 0 {
		path := ctx.Request.URL.Path
		nextpage := &xbmc.ListItem{
			Label: "LOCALIZE[30218]",
			Path: UrlForXBMC(fmt.Sprintf("%s?page=%d", path, page + 1)),
			Thumbnail: config.AddonResource("img", "nextpage.png"),
		}
		items = append(items, nextpage)
	}
	ctx.JSON(200, xbmc.NewView("tvshows", items))
}

func TraktPopularShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.TopShows("popular", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, total, page)
}

func TraktTrendingShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.TopShows("trending", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, total, page)
}

func TraktMostPlayedShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.TopShows("played", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, total, page)
}

func TraktMostWatchedShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.TopShows("watched", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, total, page)
}

func TraktMostCollectedShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.TopShows("collected", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, total, page)
}

func TraktMostAnticipatedShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.TopShows("anticipated", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, total, page)
}

//
// Calendars
//
func TraktMyShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.CalendarShows("my/shows", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarShows(ctx, shows, total, page)
}

func TraktMyNewShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.CalendarShows("my/shows/new", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarShows(ctx, shows, total, page)
}

func TraktMyPremieres(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.CalendarShows("my/shows/premieres", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarShows(ctx, shows, total, page)
}

func TraktMyMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.CalendarMovies("my/movies", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarMovies(ctx, movies, total, page)
}

func TraktMyReleases(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.CalendarMovies("my/dvd", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarMovies(ctx, movies, total, page)
}

func TraktAllShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.CalendarShows("all/shows", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarShows(ctx, shows, total, page)
}

func TraktAllNewShows(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.CalendarShows("all/shows/new", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarShows(ctx, shows, total, page)
}

func TraktAllPremieres(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	shows, total, err := trakt.CalendarShows("all/shows/premieres", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarShows(ctx, shows, total, page)
}

func TraktAllMovies(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.CalendarMovies("all/movies", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarMovies(ctx, movies, total, page)
}

func TraktAllReleases(ctx *gin.Context) {
	pageParam := ctx.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageParam)
	movies, total, err := trakt.CalendarMovies("all/dvd", pageParam)
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderCalendarMovies(ctx, movies, total, page)
}

func TraktHistoryShows(ctx *gin.Context) {
	shows, err := trakt.WatchedShows()
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderTraktShows(ctx, shows, -1, 0)
}

func TraktProgressShows(ctx *gin.Context) {
	shows, err := trakt.WatchedProgressShows()
	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
	}
	renderProgressShows(ctx, shows, -1, 0)
}

func renderCalendarMovies(ctx *gin.Context, movies []*trakt.CalendarMovie, total int, page int) {
	hasNextPage := 0
	if page > 0 {
		resultsPerPage := config.Get().ResultsPerPage

		if total == -1 {
			total = len(movies)
		}
		if total > resultsPerPage {
			if page * resultsPerPage < total {
				hasNextPage = 1
			}
		}

		if len(movies) > resultsPerPage {
			start := (page - 1) % trakt.PagesAtOnce * resultsPerPage
			movies = movies[start:start + resultsPerPage]
		}
	}

	items := make(xbmc.ListItems, 0, len(movies) + hasNextPage)

	for _, movieListing := range movies {
		if movieListing == nil {
			continue
		}
		movie := movieListing.Movie
		if movie == nil {
			continue
		}
		item := movie.ToListItem()
		label := fmt.Sprintf("%s - %s", movieListing.Released, movie.Title)
		item.Label = label
		item.Info.Title = label
		if item.Info.Trailer != "" {
			if strings.Contains(item.Info.Trailer, "?v=") {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", strings.Split(item.Info.Trailer, "?v=")[1])
			} else {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", item.Info.Trailer)
			}
		}

		playLabel := "LOCALIZE[30023]"
		playURL := UrlForXBMC("/movie/%d/play", movie.IDs.TMDB)
		linksLabel := "LOCALIZE[30202]"
		linksURL := UrlForXBMC("/movie/%d/links", movie.IDs.TMDB)
		markWatchedLabel := "LOCALIZE[30313]"
		markWatchedURL := UrlForXBMC("/movie/%d/trakt/watched", movie.IDs.TMDB)
		markUnwatchedLabel := "LOCALIZE[30314]"
		markUnwatchedURL := UrlForXBMC("/movie/%d/trakt/unwatched", movie.IDs.TMDB)


		defaultURL := linksURL
		contextLabel := playLabel
		contextURL := playURL
		if config.Get().ChooseStreamAuto == true {
			defaultURL = playURL
			contextLabel = linksLabel
			contextURL = linksURL
		}

		item.Path = defaultURL

		tmdbId := strconv.Itoa(movie.IDs.TMDB)
		libraryAction := []string{"LOCALIZE[30252]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/movie/add/%d", movie.IDs.TMDB))}
		if _, err := isDuplicateMovie(tmdbId); err != nil || isAddedToLibrary(tmdbId, Movie) {
			libraryAction = []string{"LOCALIZE[30253]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/movie/remove/%d", movie.IDs.TMDB))}
		}

		watchlistAction := []string{"LOCALIZE[30255]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/watchlist/add", movie.IDs.TMDB))}
		if inMoviesWatchlist(movie.IDs.TMDB) {
			watchlistAction = []string{"LOCALIZE[30256]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/watchlist/remove", movie.IDs.TMDB))}
		}

		collectionAction := []string{"LOCALIZE[30258]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/collection/add", movie.IDs.TMDB))}
		if inMoviesCollection(movie.IDs.TMDB) {
			collectionAction = []string{"LOCALIZE[30259]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/movie/%d/collection/remove", movie.IDs.TMDB))}
		}

		markAction := []string{markWatchedLabel, fmt.Sprintf("XBMC.RunPlugin(%s)", markWatchedURL)}
		if inMoviesWatched(movie.IDs.TMDB) {
			item.Info.Overlay = xbmc.IconOverlayWatched
			item.Info.PlayCount = 1
			markAction = []string{markUnwatchedLabel, fmt.Sprintf("XBMC.RunPlugin(%s)", markUnwatchedURL)}
		}

		item.ContextMenu = [][]string{
			[]string{contextLabel, fmt.Sprintf("XBMC.PlayMedia(%s)", contextURL)},
			libraryAction,
			watchlistAction,
			collectionAction,
			[]string{"LOCALIZE[30034]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/setviewmode/movies"))},
			markAction,
		}
		if config.Get().Platform.Kodi < 17 {
			item.ContextMenu = append(item.ContextMenu, []string{"LOCALIZE[30203]", "XBMC.Action(Info)"})
			item.ContextMenu = append(item.ContextMenu, []string{"LOCALIZE[30268]", "XBMC.Action(ToggleWatched)"})
		}
		item.IsPlayable = true
		items = append(items, item)
	}
	if page >= 0 && hasNextPage > 0 {
		path := ctx.Request.URL.Path
		nextpage := &xbmc.ListItem{
			Label: "LOCALIZE[30218]",
			Path: UrlForXBMC(fmt.Sprintf("%s?page=%d", path, page + 1)),
			Thumbnail: config.AddonResource("img", "nextpage.png"),
		}
		items = append(items, nextpage)
	}
	ctx.JSON(200, xbmc.NewView("movies", items))
}

func renderCalendarShows(ctx *gin.Context, shows []*trakt.CalendarShow, total int, page int) {
	hasNextPage := 0
	if page > 0 {
		resultsPerPage := config.Get().ResultsPerPage

		if total == -1 {
			total = len(shows)
		}
		if total > resultsPerPage {
			if page * resultsPerPage < total {
				hasNextPage = 1
			}
		}

		if len(shows) >= resultsPerPage {
			start := (page - 1) % trakt.PagesAtOnce * resultsPerPage
			shows = shows[start:start + resultsPerPage]
		}
	}

	items := make(xbmc.ListItems, 0, len(shows) + hasNextPage)

	for _, showListing := range shows {
		if showListing == nil {
			continue
		}
		show := showListing.Show
		if show == nil {
			continue
		}
		item := show.ToListItem()
		episode := showListing.Episode
		label := fmt.Sprintf("%s - %s | %dx%02d %s", []byte(showListing.FirstAired)[:10], item.Label, episode.Season, episode.Number, episode.Title)
		item.Label = label
		item.Info.Title = label
		if item.Info.Trailer != "" {
			if strings.Contains(item.Info.Trailer, "?v=") {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", strings.Split(item.Info.Trailer, "?v=")[1])
			} else {
				item.Info.Trailer = fmt.Sprintf("plugin://plugin.video.youtube/play/?video_id=%s", item.Info.Trailer)
			}
		}

		itemPath := UrlQuery(UrlForXBMC("/search"), "q", fmt.Sprintf("%s S%02dE%02d", show.Title, episode.Season, episode.Number))
		if episode.Season > 100 {
			itemPath = UrlQuery(UrlForXBMC("/search"), "q", fmt.Sprintf("%s %d %d", show.Title, episode.Number, episode.Season))
		}
		item.Path = itemPath

		tmdbId := strconv.Itoa(show.IDs.TMDB)
		libraryAction := []string{"LOCALIZE[30252]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/show/add/%d", show.IDs.TMDB))}
		if _, err := isDuplicateShow(tmdbId); err != nil || isAddedToLibrary(tmdbId, Show) {
			libraryAction = []string{"LOCALIZE[30253]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/show/remove/%d", show.IDs.TMDB))}
		}
		mergeAction := []string{"LOCALIZE[30283]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/library/show/add/%d?merge=true", show.IDs.TMDB))}

		watchlistAction := []string{"LOCALIZE[30255]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/watchlist/add", show.IDs.TMDB))}
		if inShowsWatchlist(show.IDs.TMDB) {
			watchlistAction = []string{"LOCALIZE[30256]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/watchlist/remove", show.IDs.TMDB))}
		}

		collectionAction := []string{"LOCALIZE[30258]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/collection/add", show.IDs.TMDB))}
		if inShowsCollection(show.IDs.TMDB) {
			collectionAction = []string{"LOCALIZE[30259]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/show/%d/collection/remove", show.IDs.TMDB))}
		}

		item.ContextMenu = [][]string{
			libraryAction,
			mergeAction,
			watchlistAction,
			collectionAction,
			[]string{"LOCALIZE[30203]", "XBMC.Action(Info)"},
			[]string{"LOCALIZE[30268]", "XBMC.Action(ToggleWatched)"},
			[]string{"LOCALIZE[30035]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/setviewmode/tvshows"))},
		}
		item.IsPlayable = true

		items = append(items, item)
	}
	if page >= 0 && hasNextPage > 0 {
		path := ctx.Request.URL.Path
		nextpage := &xbmc.ListItem{
			Label: "LOCALIZE[30218]",
			Path: UrlForXBMC(fmt.Sprintf("%s?page=%d", path, page + 1)),
			Thumbnail: config.AddonResource("img", "nextpage.png"),
		}
		items = append(items, nextpage)
	}
	ctx.JSON(200, xbmc.NewView("tvshows", items))
}

func renderProgressShows(ctx *gin.Context, shows []*trakt.ProgressShow, total int, page int) {
	language := config.Get().Language
	hasNextPage := 0
	if page > 0 {
		resultsPerPage := config.Get().ResultsPerPage

		if total == -1 {
			total = len(shows)
		}
		if total > resultsPerPage {
			if page * resultsPerPage < total {
				hasNextPage = 1
			}
		}

		if len(shows) >= resultsPerPage {
			start := (page - 1) % trakt.PagesAtOnce * resultsPerPage
			shows = shows[start:start + resultsPerPage]
		}
	}

	items := make(xbmc.ListItems, 0, len(shows) + hasNextPage)

	for _, showListing := range shows {
		if showListing == nil {
			continue
		}
		
		show := showListing.Show
		epi := showListing.Episode
		if show == nil || epi == nil{
			continue
		}

		episode := tmdb.GetEpisode(show.IDs.TMDB, epi.Season, epi.Number, language)

		episodeLabel := fmt.Sprintf("%s | %dx%02d %s", show.Title, episode.SeasonNumber, episode.EpisodeNumber, episode.Name)

		item := &xbmc.ListItem{
			Label: episodeLabel,
			Label2: fmt.Sprintf("%f", episode.VoteAverage),
			Info: &xbmc.ListItemInfo{
				Title:         episodeLabel,
				OriginalTitle: episode.Name,
				Season:        episode.SeasonNumber,
				Episode:       episode.EpisodeNumber,
				TVShowTitle:   show.Title,
				Plot:          episode.Overview,
				PlotOutline:   episode.Overview,
				Rating:        episode.VoteAverage,
				Aired:         episode.AirDate,
				Code:          show.IDs.IMDB,
				IMDBNumber:    show.IDs.IMDB,
				DBTYPE:        "episode",
				Mediatype:     "episode",
			},
			Art: &xbmc.ListItemArt{},
		}

		if episode.StillPath != "" {
			item.Art.FanArt = tmdb.ImageURL(episode.StillPath, "w1280")
			item.Art.Thumbnail = tmdb.ImageURL(episode.StillPath, "w500")
			item.Art.Poster = tmdb.ImageURL(episode.StillPath, "w500")
		}

		playLabel := "LOCALIZE[30023]"
		playURL := UrlForXBMC("/show/%d/season/%d/episode/%d/play",
			show.IDs.TMDB,
			episode.SeasonNumber,
			episode.EpisodeNumber,
		)
		linksLabel := "LOCALIZE[30202]"
		linksURL := UrlForXBMC("/show/%d/season/%d/episode/%d/links",
			show.IDs.TMDB,
			episode.SeasonNumber,
			episode.EpisodeNumber,
		)
		markWatchedLabel := "LOCALIZE[30313]"
		markWatchedURL := UrlForXBMC("/show/%d/season/%d/episode/%d/trakt/watched", 
			show.IDs.TMDB,
			episode.SeasonNumber,
			episode.EpisodeNumber,
		)

		defaultURL := linksURL
		contextLabel := playLabel
		contextURL := playURL
		if config.Get().ChooseStreamAuto == true {
			defaultURL = playURL
			contextLabel = linksLabel
			contextURL = linksURL
		}

		item.Path = defaultURL

		markAction := []string{markWatchedLabel, fmt.Sprintf("XBMC.RunPlugin(%s)", markWatchedURL)}

		item.ContextMenu = [][]string{
			[]string{contextLabel, fmt.Sprintf("XBMC.PlayMedia(%s)", contextURL)},
			[]string{"LOCALIZE[30037]", fmt.Sprintf("XBMC.RunPlugin(%s)", UrlForXBMC("/setviewmode/episodes"))},
			markAction,
		}
		if config.Get().Platform.Kodi < 17 {
			item.ContextMenu = append(item.ContextMenu, []string{"LOCALIZE[30203]", "XBMC.Action(Info)"})
			item.ContextMenu = append(item.ContextMenu, []string{"LOCALIZE[30268]", "XBMC.Action(ToggleWatched)"})
		}
		item.IsPlayable = true
		items = append(items, item)
	}
	if page >= 0 && hasNextPage > 0 {
		path := ctx.Request.URL.Path
		nextpage := &xbmc.ListItem{
			Label: "LOCALIZE[30218]",
			Path: UrlForXBMC(fmt.Sprintf("%s?page=%d", path, page + 1)),
			Thumbnail: config.AddonResource("img", "nextpage.png"),
		}
		items = append(items, nextpage)
	}
	ctx.JSON(200, xbmc.NewView("tvshows", items))
}
