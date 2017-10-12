package repository

import (
	"os"
	"fmt"
	"bufio"
	"errors"
	"regexp"
	"strings"
	"context"
	"net/http"
	"io/ioutil"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"path/filepath"

	"github.com/op/go-logging"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/charly3pins/quasar/config"
	"github.com/charly3pins/quasar/xbmc"
)

const (
	githubUserContentURL = "https://raw.githubusercontent.com/%s/%s/%s"
	burstWebsiteURL      = "https://burst.surge.sh/release/%s"
	backupRepositoryURL  = "https://offshoregit.com/%s/%s"
	releaseChangelog     = "[B]%s[/B] - %s\n%s\n\n"
)

var (
	addonZipRE       = regexp.MustCompile(`[\w]+\.[\w]+(\.[\w]+)?-\d+\.\d+\.\d+(-[\w]+\.?\d+)?\.zip`)
	addonChangelogRE = regexp.MustCompile(`changelog.*\.txt`)
	log              = logging.MustGetLogger("repository")
)

func getLastRelease(user string, repository string) (string, string) {
	client := github.NewClient(nil)
	releases, _, _ := client.Repositories.ListReleases(context.TODO(), user, repository, nil)
	if len(releases) > 0 {
		lastRelease := releases[0]
		return *lastRelease.TagName, *lastRelease.TargetCommitish
	}
	return "", "master"
}

func getReleaseByTag(user string, repository string, tagName string) *github.RepositoryRelease {
	client := github.NewClient(nil)
	releases, _, _ := client.Repositories.ListReleases(context.TODO(), user, repository, nil)
	for _, release := range releases {
		if *release.TagName == tagName {
			return release
		}
	}
	return nil
}

func getAddons(user string, repository string) (*xbmc.AddonList, error) {
	var addons []xbmc.Addon

	_, lastReleaseBranch := getLastRelease(user, "plugin.video.quasar")
	resp, err := http.Get(fmt.Sprintf(githubUserContentURL, user, "plugin.video.quasar", lastReleaseBranch) + "/addon.xml")
	if err == nil && resp.StatusCode != 200 {
		err = errors.New(resp.Status)
	}
	if err != nil {
		log.Warning("Unable to retrieve the addon.xml file, checking backup repository...")
		resp, err = http.Get(fmt.Sprintf(backupRepositoryURL, user, "plugin.video.quasar") + "/addon.xml")
		if err == nil && resp.StatusCode != 200 {
			err = errors.New(resp.Status)
		}
		if err != nil {
			log.Error("Unable to retrieve the backup's addon.xml file.")
			return nil, err
		}
	}

	respBurst, errBurst := http.Get(fmt.Sprintf(burstWebsiteURL, "addon.xml"))
	if errBurst == nil && respBurst.StatusCode != 200 {
		err = errors.New(resp.Status)
	}
	if err != nil {
		log.Warning("Unable to retrieve Burst's addon.xml file, checking backup repository...")
		respBurst, errBurst := http.Get(fmt.Sprintf(backupRepositoryURL, user, "script.quasar.burst") + "/addon.xml")
		if errBurst == nil && respBurst.StatusCode != 200 {
			errBurst = errors.New(respBurst.Status)
		}
		if errBurst != nil {
			log.Error("Unable to retrieve Burst's addon.xml file.")
		}
	}

	addon := xbmc.Addon{}
	xml.NewDecoder(resp.Body).Decode(&addon)
	addons = append(addons, addon)

	if errBurst == nil {
		burst := xbmc.Addon{}
		xml.NewDecoder(respBurst.Body).Decode(&burst)
		addons = append(addons, burst)
	}

	return &xbmc.AddonList{
		Addons: addons,
	}, nil
}

func GetAddonsXML(ctx *gin.Context) {
	user := ctx.Params.ByName("user")
	repository := ctx.Params.ByName("repository")
	addons, err := getAddons(user, repository)
	if err != nil {
		ctx.AbortWithError(404, errors.New("Unable to retrieve the remote's addons.xml file."))
	}
	ctx.XML(200, addons)
}

func GetAddonsXMLChecksum(ctx *gin.Context) {
	user := ctx.Params.ByName("user")
	repository := ctx.Params.ByName("repository")
	addons, err := getAddons(user, repository)
	if len(addons.Addons) > 0 {
		log.Infof("Last available release of %s: v%s", repository, addons.Addons[0].Version)
	}
	if len(addons.Addons) > 1 {
		log.Infof("Last available release of script.quasar.burst: v%s", addons.Addons[1].Version)
	}
	if err != nil {
		ctx.Error(errors.New("Unable to retrieve the remote's addon.xml file."))
	}
	hasher := md5.New()
	xml.NewEncoder(hasher).Encode(addons)
	ctx.String(200, hex.EncodeToString(hasher.Sum(nil)))
}

func GetAddonFiles(ctx *gin.Context) {
	user := ctx.Params.ByName("user")
	repository := ctx.Params.ByName("repository")
	filepath := ctx.Params.ByName("filepath")[1:] // strip the leading "/"

	lastReleaseTag := ""
	if repository == "plugin.video.quasar" {
		lastReleaseTag, _ = getLastRelease(user, repository)
		if lastReleaseTag == "" {
			// Get last release from addons.xml on master
			log.Warning("Unable to find a last tag, using master.")
			addons, err := getAddons(user, repository)
			if err != nil || len(addons.Addons) < 1 || addons.Addons[0].Version == "" {
				ctx.AbortWithError(404, errors.New("Unable to retrieve the remote's addon.xml file."))
				return
			}
			lastReleaseTag = "v" + addons.Addons[0].Version
		}
	} else {
		addons, err := getAddons(user, repository)
		if err != nil || len(addons.Addons) < 2 || addons.Addons[1].Version == "" {
			ctx.AbortWithError(404, errors.New("Unable to retrieve the addon.xml file."))
			return
		}
		lastReleaseTag = "v" + addons.Addons[1].Version
	}

	switch filepath {
	case "addons.xml":
		GetAddonsXML(ctx)
		return
	case "addons.xml.md5":
		go writeChangelog(user, "plugin.video.quasar")
		go writeChangelog(user, "script.quasar.burst")
		GetAddonsXMLChecksum(ctx)
		return
	case "fanart.jpg":
		fallthrough
	case "icon.png":
		if repository == "plugin.video.quasar" {
			ctx.Redirect(302, fmt.Sprintf(githubUserContentURL + "/" + filepath, user, repository, lastReleaseTag))
		} else {
			ctx.Redirect(302, fmt.Sprintf(burstWebsiteURL, filepath))
		}
		return
	}

	switch {
	case addonZipRE.MatchString(filepath):
		addonZip(ctx, user, repository, lastReleaseTag)
	case addonChangelogRE.MatchString(filepath):
		writeChangelog(user, repository)
		addonChangelog(ctx, user, repository)
	default:
		ctx.AbortWithError(404, errors.New(filepath))
	}
}

func GetAddonFilesHead(ctx *gin.Context) {
	ctx.String(200, "")
}

func addonZip(ctx *gin.Context, user string, repository string, lastReleaseTag string) {
	if repository == "plugin.video.quasar" {
		release := getReleaseByTag(user, repository, lastReleaseTag)
		// if there a release with an asset that matches a addon zip, use it
		if release != nil {
			client := github.NewClient(nil)
			assets, _, _ := client.Repositories.ListReleaseAssets(context.TODO(), user, repository, *release.ID, nil)
			platformStruct := xbmc.GetPlatform()
			platform := platformStruct.OS + "_" + platformStruct.Arch
			var assetAllPlatforms string
			for _, asset := range assets {
				if strings.HasSuffix(*asset.Name, platform + ".zip") {
					assetPlatform := *asset.BrowserDownloadURL
					log.Infof("Using release asset for %s: %s", platform, assetPlatform)
					ctx.Redirect(302, assetPlatform)
					return
				}
				if addonZipRE.MatchString(*asset.Name) {
					assetAllPlatforms = *asset.BrowserDownloadURL
					log.Infof("Found all platforms release asset: %s", assetAllPlatforms)
					continue
				}
			}
			if assetAllPlatforms != "" {
				log.Infof("Using release asset for all platforms: %s", assetAllPlatforms)
				ctx.Redirect(302, assetAllPlatforms)
				return
			}
		} else {
			log.Warning("Release not found on main repository, using backup...")
			platformStruct := xbmc.GetPlatform()
			platform := platformStruct.OS + "_" + platformStruct.Arch
			backupRepoPath := fmt.Sprintf(backupRepositoryURL, user, repository)
			addonFile := fmt.Sprintf("%s-%s.%s.zip", repository, lastReleaseTag[1:], platform)
			assetPlatform := fmt.Sprintf("%s/%s/%s", backupRepoPath, lastReleaseTag, addonFile)
			log.Infof("Backup release asset: %s", assetPlatform)
			ctx.Redirect(302, assetPlatform)
			return
		}
	} else {
		addonFile := fmt.Sprintf("%s-%s.zip", repository, lastReleaseTag[1:])
		assetURI := fmt.Sprintf(burstWebsiteURL, addonFile)
		if resp, err := http.Head(assetURI); err == nil && resp.StatusCode == 200 {
			ctx.Redirect(302, assetURI)
			return
		} else {
			log.Warningf("Release of %s not found on main repository, using backup...", repository)
			repoURI := fmt.Sprintf(backupRepositoryURL, user, repository)
			assetURI = fmt.Sprintf("%s/%s", repoURI, addonFile)
			log.Infof("Release asset: %s", assetURI)
			ctx.Redirect(302, assetURI)
		}
	}
}

func fetchChangelog(user string, repository string) string {
	log.Infof("Fetching add-on changelog for %s...", repository)
	changelog := ""
	if repository == "plugin.video.quasar" {
		client := github.NewClient(nil)
		releases, _, _ := client.Repositories.ListReleases(context.TODO(), user, repository, nil)
		changelog = "Quasar changelog\n======\n\n"
		for _, release := range releases {
			changelog += fmt.Sprintf(releaseChangelog, *release.TagName, release.PublishedAt.Format("Jan 2 2006"), *release.Body)
		}
	} else {
		resp, err := http.Get(fmt.Sprintf(burstWebsiteURL, "changelog.txt"))
		if err != nil || resp.StatusCode != 200 {
			log.Warning("Unable to fetch changelog, trying backup repository...")
			resp, err = http.Get(fmt.Sprintf(backupRepositoryURL, user, repository) + "/changelog.txt")
			if err != nil || resp.StatusCode != 200 {
				changelog = "Unable to fetch changelog, try again later."
				return changelog
			}
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			changelog = err.Error()
		} else {
			changelog = string(data)
		}
	}
	return changelog
}

func writeChangelog(user string, repository string) error {
	changelog := fetchChangelog(user, repository)
	lines := strings.Split(changelog, "\n")
	path := filepath.Clean(filepath.Join(config.Get().Info.Path, "..", repository, "changelog.txt"))

  file, err := os.Create(path)
  if err != nil {
    return err
  }
  defer file.Close()

  w := bufio.NewWriter(file)
  for _, line := range lines {
    fmt.Fprintln(w, line)
  }
  return w.Flush()
}

func addonChangelog(ctx *gin.Context, user string, repository string) {
	changelog := fetchChangelog(user, repository)
	ctx.String(200, changelog)
}
