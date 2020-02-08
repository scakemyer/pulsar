package providers

import (
	"github.com/charly3pins/magnetar/bittorrent"
	"github.com/charly3pins/magnetar/tmdb"
)

type Searcher interface {
	SearchLinks(query string) []*bittorrent.Torrent
}

type MovieSearcher interface {
	SearchMovieLinks(movie *tmdb.Movie) []*bittorrent.Torrent
}

type SeasonSearcher interface {
	SearchSeasonLinks(show *tmdb.Show, season *tmdb.Season) []*bittorrent.Torrent
}

type EpisodeSearcher interface {
	SearchEpisodeLinks(show *tmdb.Show, episode *tmdb.Episode) []*bittorrent.Torrent
}
