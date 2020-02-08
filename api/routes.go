package api

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"time"

	"github.com/charly3pins/magnetar/api/repository"
	"github.com/charly3pins/magnetar/bittorrent"
	"github.com/charly3pins/magnetar/cache"
	"github.com/charly3pins/magnetar/config"
	"github.com/charly3pins/magnetar/providers"
	"github.com/charly3pins/magnetar/util"

	"github.com/gin-gonic/gin"
)

const (
	DefaultCacheExpiration    = 6 * time.Hour
	RecentCacheExpiration     = 5 * time.Minute
	RepositoryCacheExpiration = 20 * time.Minute
	IndexCacheExpiration      = 15 * 24 * time.Hour // 15 days caching for index
)

func Routes(btService *bittorrent.BTService) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/torrents/list", "/notification"))

	gin.SetMode(gin.ReleaseMode)

	store := cache.NewFileStore(path.Join(config.Get().ProfilePath, "cache"))

	r.GET("/", Index)
	r.GET("/search", Search(btService))
	r.GET("/playtorrent", PlayTorrent)
	r.GET("/infolabels", InfoLabelsStored(btService))

	r.LoadHTMLGlob(filepath.Join(config.Get().Info.Path, "resources", "web", "*.html"))
	web := r.Group("/web")
	{
		web.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", nil)
		})
		web.Static("/static", filepath.Join(config.Get().Info.Path, "resources", "web", "static"))
		web.StaticFile("/favicon.ico", filepath.Join(config.Get().Info.Path, "resources", "web", "favicon.ico"))
	}

	torrents := r.Group("/torrents")
	{
		torrents.GET("/", ListTorrents(btService))
		torrents.GET("/add", AddTorrent(btService))
		torrents.GET("/pause", PauseSession(btService))
		torrents.GET("/resume", ResumeSession(btService))
		torrents.GET("/move/:torrentId", MoveTorrent(btService))
		torrents.GET("/pause/:torrentId", PauseTorrent(btService))
		torrents.GET("/resume/:torrentId", ResumeTorrent(btService))
		torrents.GET("/delete/:torrentId", RemoveTorrent(btService))

		// Web UI json
		torrents.GET("/list", ListTorrentsWeb(btService))
	}

	movies := r.Group("/movies")
	{
		movies.GET("/", cache.Cache(store, IndexCacheExpiration), MoviesIndex)
		movies.GET("/search", SearchMovies)
		movies.GET("/popular", PopularMovies)
		movies.GET("/popular/:genre", PopularMovies)
		movies.GET("/recent", RecentMovies)
		movies.GET("/recent/:genre", RecentMovies)
		movies.GET("/top", TopRatedMovies)
		movies.GET("/imdb250", IMDBTop250)
		movies.GET("/mostvoted", MoviesMostVoted)
		movies.GET("/genres", MovieGenres)

		trakt := movies.Group("/trakt")
		{
			trakt.GET("/", cache.Cache(store, IndexCacheExpiration), MoviesTrakt)
			trakt.GET("/watchlist", WatchlistMovies)
			trakt.GET("/collection", CollectionMovies)
			trakt.GET("/popular", TraktPopularMovies)
			trakt.GET("/trending", TraktTrendingMovies)
			trakt.GET("/played", TraktMostPlayedMovies)
			trakt.GET("/watched", TraktMostWatchedMovies)
			trakt.GET("/collected", TraktMostCollectedMovies)
			trakt.GET("/anticipated", TraktMostAnticipatedMovies)
			trakt.GET("/boxoffice", TraktBoxOffice)

			lists := trakt.Group("/lists")
			{
				lists.GET("/", cache.Cache(store, RecentCacheExpiration), MoviesTraktLists)
				lists.GET("/id/:listId", UserlistMovies)
			}

			calendars := trakt.Group("/calendars")
			{
				calendars.GET("/", CalendarMovies)
				calendars.GET("/movies", TraktMyMovies)
				calendars.GET("/releases", TraktMyReleases)
				calendars.GET("/allmovies", TraktAllMovies)
				calendars.GET("/allreleases", TraktAllReleases)
			}
		}
	}
	movie := r.Group("/movie")
	{
		movie.GET("/:tmdbId/infolabels", InfoLabelsMovie(btService))
		movie.GET("/:tmdbId/links", MovieLinks(btService, false))
		movie.GET("/:tmdbId/play", MoviePlay(btService, false))
		movie.GET("/:tmdbId/watchlist/add", AddMovieToWatchlist)
		movie.GET("/:tmdbId/watchlist/remove", RemoveMovieFromWatchlist)
		movie.GET("/:tmdbId/collection/add", AddMovieToCollection)
		movie.GET("/:tmdbId/collection/remove", RemoveMovieFromCollection)
		movie.GET("/:tmdbId/trakt/watched", MarkMovieWatchedInTrakt)
		movie.GET("/:tmdbId/trakt/unwatched", MarkMovieUnwatchedInTrakt)
	}

	shows := r.Group("/shows")
	{
		shows.GET("/", cache.Cache(store, IndexCacheExpiration), TVIndex)
		shows.GET("/search", SearchShows)
		shows.GET("/popular", PopularShows)
		shows.GET("/popular/:genre", PopularShows)
		shows.GET("/recent/shows", RecentShows)
		shows.GET("/recent/shows/:genre", RecentShows)
		shows.GET("/recent/episodes", RecentEpisodes)
		shows.GET("/recent/episodes/:genre", RecentEpisodes)
		shows.GET("/top", TopRatedShows)
		shows.GET("/mostvoted", TVMostVoted)
		shows.GET("/genres", TVGenres)

		trakt := shows.Group("/trakt")
		{
			trakt.GET("/", cache.Cache(store, IndexCacheExpiration), TVTrakt)
			trakt.GET("/watchlist", WatchlistShows)
			trakt.GET("/collection", CollectionShows)
			trakt.GET("/popular", TraktPopularShows)
			trakt.GET("/trending", TraktTrendingShows)
			trakt.GET("/played", TraktMostPlayedShows)
			trakt.GET("/watched", TraktMostWatchedShows)
			trakt.GET("/collected", TraktMostCollectedShows)
			trakt.GET("/anticipated", TraktMostAnticipatedShows)
			trakt.GET("/history", TraktHistoryShows)
			trakt.GET("/progress", TraktProgressShows)

			lists := trakt.Group("/lists")
			{
				lists.GET("/", cache.Cache(store, RecentCacheExpiration), TVTraktLists)
				lists.GET("/id/:listId", UserlistShows)
			}

			calendars := trakt.Group("/calendars")
			{
				calendars.GET("/", CalendarShows)
				calendars.GET("/shows", TraktMyShows)
				calendars.GET("/newshows", TraktMyNewShows)
				calendars.GET("/premieres", TraktMyPremieres)
				calendars.GET("/allshows", TraktAllShows)
				calendars.GET("/allnewshows", TraktAllNewShows)
				calendars.GET("/allpremieres", TraktAllPremieres)
			}
		}
	}
	show := r.Group("/show")
	{
		show.GET("/:showId/seasons", ShowSeasons)
		show.GET("/:showId/season/:season/links", ShowSeasonLinks(btService, false))
		show.GET("/:showId/season/:season/episodes", ShowEpisodes)
		show.GET("/:showId/season/:season/episode/:episode/infolabels", InfoLabelsEpisode(btService))
		show.GET("/:showId/season/:season/episode/:episode/play", ShowEpisodePlay(btService, false))
		show.GET("/:showId/season/:season/episode/:episode/links", ShowEpisodeLinks(btService, false))
		show.GET("/:showId/watchlist/add", AddShowToWatchlist)
		show.GET("/:showId/watchlist/remove", RemoveShowFromWatchlist)
		show.GET("/:showId/collection/add", AddShowToCollection)
		show.GET("/:showId/collection/remove", RemoveShowFromCollection)
		show.GET("/:showId/trakt/watched", MarkShowWatchedInTrakt)
		show.GET("/:showId/trakt/unwatched", MarkShowUnwatchedInTrakt)
		show.GET("/:showId/season/:season/trakt/watched", MarkSeasonWatchedInTrakt)
		show.GET("/:showId/season/:season/trakt/unwatched", MarkSeasonUnwatchedInTrakt)
		show.GET("/:showId/season/:season/episode/:episode/trakt/watched", MarkEpisodeWatchedInTrakt)
		show.GET("/:showId/season/:season/episode/:episode/trakt/unwatched", MarkEpisodeUnwatchedInTrakt)
	}
	// TODO
	// episode := r.Group("/episode")
	// {
	// 	episode.GET("/:episodeId/watchlist/add", AddEpisodeToWatchlist)
	// }

	library := r.Group("/library")
	{
		library.GET("/movie/add/:tmdbId", AddMovie)
		library.GET("/movie/remove/:tmdbId", RemoveMovie)
		library.GET("/movie/list/add/:listId", AddMoviesList)
		library.GET("/movie/play/:tmdbId", PlayMovie(btService))
		library.GET("/show/add/:tmdbId", AddShow)
		library.GET("/show/remove/:tmdbId", RemoveShow)
		library.GET("/show/list/add/:listId", AddShowsList)
		library.GET("/show/play/:showId/:season/:episode", PlayShow(btService))

		library.GET("/update", UpdateLibrary)

		// DEPRECATED
		library.GET("/play/movie/:tmdbId", PlayMovie(btService))
		library.GET("/play/show/:showId/season/:season/episode/:episode", PlayShow(btService))
	}

	provider := r.Group("/provider")
	{
		provider.GET("/", ProviderList)
		provider.GET("/:provider/check", ProviderCheck)
		provider.GET("/:provider/enable", ProviderEnable)
		provider.GET("/:provider/disable", ProviderDisable)
		provider.GET("/:provider/failure", ProviderFailure)
		provider.GET("/:provider/settings", ProviderSettings)

		provider.GET("/:provider/movie/:tmdbId", ProviderGetMovie)
		provider.GET("/:provider/show/:showId/season/:season/episode/:episode", ProviderGetEpisode)
	}

	allproviders := r.Group("/providers")
	{
		allproviders.GET("/enable", ProvidersEnableAll)
		allproviders.GET("/disable", ProvidersDisableAll)
	}

	repo := r.Group("/repository")
	{
		repo.GET("/:user/:repository/*filepath", repository.GetAddonFiles)
		repo.HEAD("/:user/:repository/*filepath", repository.GetAddonFilesHead)
	}

	trakt := r.Group("/trakt")
	{
		trakt.GET("/authorize", AuthorizeTrakt)
	}

	r.GET("/setviewmode/:content_type", SetViewMode)

	r.GET("/subtitles", SubtitlesIndex)
	r.GET("/subtitle/:id", SubtitleGet)

	r.GET("/play", Play(btService))
	r.GET("/playuri", PlayURI(btService))

	r.POST("/callbacks/:cid", providers.CallbackHandler)

	r.GET("/notification", Notification(btService))

	r.GET("/versions", Versions(btService))

	cmd := r.Group("/cmd")
	{
		cmd.GET("/clear_cache", ClearCache)
		cmd.GET("/clear_page_cache", ClearPageCache)
		cmd.GET("/reset_clearances", ResetClearances)
	}

	return r
}

func UrlForHTTP(pattern string, args ...interface{}) string {
	u, _ := url.Parse(fmt.Sprintf(pattern, args...))
	return util.GetHTTPHost() + u.String()
}

func UrlForXBMC(pattern string, args ...interface{}) string {
	u, _ := url.Parse(fmt.Sprintf(pattern, args...))
	return "plugin://" + config.Get().Info.Id + u.String()
}

func UrlQuery(route string, query ...string) string {
	v := url.Values{}
	for i := 0; i < len(query); i += 2 {
		v.Add(query[i], query[i+1])
	}
	return route + "?" + v.Encode()
}
