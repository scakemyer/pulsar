package util

import (
	"fmt"

	"github.com/charly3pins/libtorrent-go"
)

var (
	Version string
)

func UserAgent() string {
	return fmt.Sprintf("Magnetar/%s libtorrent/%s", Version[1:len(Version)-1], libtorrent.Version())
}
