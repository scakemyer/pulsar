package main

import (
	"os"
	"path/filepath"

	"github.com/charly3pins/magnetar/config"
	"github.com/charly3pins/magnetar/repository"
	"github.com/charly3pins/magnetar/xbmc"
)

func Migrate() bool {
	firstRun := filepath.Join(config.Get().Info.Path, ".firstrun")
	if _, err := os.Stat(firstRun); err == nil {
		return false
	}
	file, _ := os.Create(firstRun)
	defer file.Close()

	log.Info("Preparing for first run...")

	log.Info("Creating Magnetar repository add-on...")
	if err := repository.MakeMagnetarRepositoryAddon(); err != nil {
		log.Errorf("Unable to create repository add-on: %s", err)
	} else {
		xbmc.UpdateLocalAddons()
		for _, addon := range xbmc.GetAddons("xbmc.addon.repository", "unknown", "all", []string{"name", "version", "enabled"}).Addons {
			if addon.ID == "repository.magnetar" && addon.Enabled == true {
				log.Info("Found enabled Magnetar repository add-on")
				return false
			}
		}
		log.Info("Magnetar repository not installed, installing...")
		xbmc.InstallAddon("repository.magnetar")
		xbmc.SetAddonEnabled("repository.magnetar", true)
		xbmc.UpdateLocalAddons()
		xbmc.UpdateAddonRepos()
	}

	return true
}
