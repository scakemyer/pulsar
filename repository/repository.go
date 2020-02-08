package repository

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charly3pins/magnetar/config"
	"github.com/charly3pins/magnetar/util"
	"github.com/charly3pins/magnetar/xbmc"
)

func copyFile(from string, to string) error {
	input, err := os.Open(from)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(to)
	if err != nil {
		return err
	}
	defer output.Close()
	io.Copy(output, input)
	return nil
}

func MakeMagnetarRepositoryAddon() error {
	addonId := "repository.magnetar"
	addonName := "Magnetar Repository"

	magnetarHost := fmt.Sprintf("http://localhost:%d", config.ListenPort)
	addon := &xbmc.Addon{
		Id:           addonId,
		Name:         addonName,
		Version:      util.Version[2 : len(util.Version)-1],
		ProviderName: config.Get().Info.Author,
		Extensions: []*xbmc.AddonExtension{
			&xbmc.AddonExtension{
				Point: "xbmc.addon.repository",
				Name:  addonName,
				Info: &xbmc.AddonRepositoryInfo{
					Text:       magnetarHost + "/repository/charly3pins/plugin.video.magnetar/addons.xml",
					Compressed: false,
				},
				Checksum: magnetarHost + "/repository/charly3pins/plugin.video.magnetar/addons.xml.md5",
				Datadir: &xbmc.AddonRepositoryDataDir{
					Text: magnetarHost + "/repository/charly3pins/",
					Zip:  true,
				},
			},
			&xbmc.AddonExtension{
				Point: "xbmc.addon.metadata",
				Summaries: []*xbmc.AddonText{
					&xbmc.AddonText{
						Text: "GitHub repository for Magnetar updates",
						Lang: "en",
					},
				},
				Platform: "all",
			},
		},
	}

	addonPath := filepath.Clean(filepath.Join(config.Get().Info.Path, "..", addonId))
	if err := os.MkdirAll(addonPath, 0777); err != nil {
		return err
	}

	if err := copyFile(filepath.Join(config.Get().Info.Path, "icon.png"), filepath.Join(addonPath, "icon.png")); err != nil {
		return err
	}

	if err := copyFile(filepath.Join(config.Get().Info.Path, "fanart.jpg"), filepath.Join(addonPath, "fanart.jpg")); err != nil {
		return err
	}

	addonXmlFile, err := os.Create(filepath.Join(addonPath, "addon.xml"))
	if err != nil {
		return err
	}
	defer addonXmlFile.Close()
	return xml.NewEncoder(addonXmlFile).Encode(addon)
}
