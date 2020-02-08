package api

import (
	"strconv"

	"github.com/charly3pins/magnetar/bittorrent"
	"github.com/charly3pins/magnetar/broadcast"

	"github.com/op/go-logging"
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
	if item.Duration == 0 || item.WatchedTime == 0 {
		return
	}

	if item.DBItem.Type == "movie" {
		xbmcItem := FindByIdMovieInLibrary(strconv.Itoa(item.DBItem.ID))
		if xbmcItem != nil {
			UpdateMovieWatched(xbmcItem, item.WatchedTime, item.Duration)
		}
	} else if item.DBItem.Type == "episode" {
		xbmcItem := FindByIdEpisodeInLibrary(item.DBItem.ShowID, item.DBItem.Season, item.DBItem.Episode)
		if xbmcItem != nil {
			UpdateEpisodeWatched(xbmcItem, item.WatchedTime, item.Duration)
		}
	}
}
