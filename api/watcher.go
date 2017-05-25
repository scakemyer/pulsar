package api

import (
	"github.com/op/go-logging"
	"github.com/scakemyer/quasar/broadcast"
	"github.com/scakemyer/quasar/bittorrent"
)

var (
	watcherLog = logging.MustGetLogger("watcher")
)

func LibraryListener() {
	broadcaster := broadcast.LocalBroadcasters[broadcast.WATCHED]

	c, done := broadcaster.Listen()
	defer close(done)

	for {
		select {
		case v, ok := <-c:
			if !ok {
				return
			}

			updateWatchedForItem(v.(*bittorrent.PlayingItem))
		}
	}
}

func updateWatchedForItem(item *bittorrent.PlayingItem) {
	if item.Duration == 0 || item.WatchedTime == 0 || item.DBID == 0 {
		return
	}

	if item.DBTYPE == "movie" {
		UpdateMovieWatched(item.DBID, item.WatchedTime, item.Duration)
	} else if item.DBTYPE == "episode" {
		UpdateEpisodeWatched(item.DBID, item.WatchedTime, item.Duration)
	}
}
