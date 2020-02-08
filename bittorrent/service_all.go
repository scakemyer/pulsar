// +build !arm

package bittorrent

import "github.com/charly3pins/libtorrent-go"

// Nothing to do on regular devices
func setPlatformSpecificSettings(settings libtorrent.SettingsPack) {
}
