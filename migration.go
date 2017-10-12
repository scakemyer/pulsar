package main

import (
	"os"
	"path/filepath"

	"github.com/charly3pins/quasar/xbmc"
	"github.com/charly3pins/quasar/config"
	"github.com/charly3pins/quasar/repository"
)

func Migrate() bool {
	firstRun := filepath.Join(config.Get().Info.Path, ".firstrun")
	if _, err := os.Stat(firstRun); err == nil {
		return false
	}
	file, _ := os.Create(firstRun)
	defer file.Close()

	log.Info("Preparing for first run...")

	log.Info("Creating Quasar repository add-on...")
	if err := repository.MakeQuasarRepositoryAddon(); err != nil {
		log.Errorf("Unable to create repository add-on: %s", err)
	} else {
		xbmc.UpdateLocalAddons()
		for _, addon := range xbmc.GetAddons("xbmc.addon.repository", "unknown", "all", []string{"name", "version", "enabled"}).Addons {
			if addon.ID == "repository.quasar" && addon.Enabled == true {
				log.Info("Found enabled Quasar repository add-on")
				return false
			}
		}
		log.Info("Quasar repository not installed, installing...")
		xbmc.InstallAddon("repository.quasar")
		xbmc.SetAddonEnabled("repository.quasar", true)
		xbmc.UpdateLocalAddons()
		xbmc.UpdateAddonRepos()
	}

	return true
}
