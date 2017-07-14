package trakt

import (
	"fmt"
	"time"
	"bytes"
	"errors"
	"strconv"
	"net/url"
	"net/http"

	"github.com/op/go-logging"
	"github.com/jmcvetta/napping"
	"github.com/scakemyer/quasar/cloudhole"
	"github.com/scakemyer/quasar/config"
	"github.com/scakemyer/quasar/util"
	"github.com/scakemyer/quasar/xbmc"
)

const (
	ApiUrl       = "https://api.trakt.tv"
	ClientId     = "4407ab20a3a971e7c92d4996b36b76d0312ea085cb139d7c38a1a4c9f8428f60"
	ClientSecret = "83f5993015942fe1320772c9c9886dce08252fa95445afab81a1603f8671e490"
	ApiVersion   = "2"
)

var log = logging.MustGetLogger("trakt")

var (
	clearance, _            = cloudhole.GetClearance()
	retriesLeft             = 3
	PagesAtOnce             = 5
	burstRate               = 50
	burstTime               = 10 * time.Second
	simultaneousConnections = 25
	cacheExpiration         = 6 * 24 * time.Hour
	recentExpiration        = 15 * time.Minute
	userlistExpiration      = 1 * time.Minute
)

var rateLimiter = util.NewRateLimiter(burstRate, burstTime, simultaneousConnections)

type Object struct {
	Title string `json:"title"`
	Year  int    `json:"year"`
	IDs   *IDs   `json:"ids"`
}

type Movie struct {
	Object

	Released      string      `json:"released"`
	URL           string      `json:"homepage"`
	Trailer       string      `json:"trailer"`
	Runtime       int         `json:"runtime"`
	TagLine       string      `json:"tagline"`
	Overview      string      `json:"overview"`
	Certification string      `json:"certification"`
	Rating        float32     `json:"rating"`
	Votes         int         `json:"votes"`
	Genres        []string    `json:"genres"`
	Language      string      `json:"language"`
	Translations  []string    `json:"available_translations"`

	Images        *Images     `json:"images"`
}

type Show struct {
	Object

	FirstAired    string      `json:"first_aired"`
	URL           string      `json:"homepage"`
	Trailer       string      `json:"trailer"`
	Runtime       int         `json:"runtime"`
	Overview      string      `json:"overview"`
	Certification string      `json:"certification"`
	Status        string      `json:"status"`
	Network       string      `json:"network"`
	AiredEpisodes int         `json:"aired_episodes"`
	Airs          *Airs       `json:"airs"`
	Rating        float32     `json:"rating"`
	Votes         int         `json:"votes"`
	Genres        []string    `json:"genres"`
	Country       string      `json:"country"`
	Language      string      `json:"language"`
	Translations  []string    `json:"available_translations"`

	Images        *Images     `json:"images"`
}

type Season struct {
	// Show          *Show   `json:"-"`
	Number        int         `json:"number"`
	Overview      string      `json:"overview"`
	EpisodeCount  int         `json:"episode_count"`
	AiredEpisodes int         `json:"aired_episodes"`
	Rating        float32     `json:"rating"`
	Votes         int         `json:"votes"`

	Images        *Images     `json:"images"`
	IDs           *IDs        `json:"ids"`
}

type Episode struct {
	// Show          *Show       `json:"-"`
	// Season        *ShowSeason `json:"-"`
	Number        int         `json:"number"`
	Season        int         `json:"season"`
	Title         string      `json:"title"`
	Overview      string      `json:"overview"`
	Absolute      int         `json:"number_abs"`
	FirstAired    string      `json:"first_aired"`
	Translations  []string    `json:"available_translations"`

	Rating        float32     `json:"rating"`
	Votes         int         `json:"votes"`

	Images        *Images     `json:"images"`
	IDs           *IDs        `json:"ids"`
}

type Airs struct {
	Day           string      `json:"day"`
	Time          string      `json:"time"`
	Timezone      string      `json:"timezone"`
}

type Movies struct {
	Watchers int    `json:"watchers"`
	Movie    *Movie `json:"movie"`
}

type Shows struct {
	Watchers int   `json:"watchers"`
	Show     *Show `json:"show"`
}

type Watchlist struct {
	Movies   []*Movie   `json:"movies"`
	Shows    []*Show    `json:"shows"`
	Episodes []*Episode `json:"episodes"`
}

type WatchlistMovie struct {
	ListedAt string  `json:"listed_at"`
	Type     string  `json:"type"`
	Movie    *Movie  `json:"movie"`
}

type WatchlistShow struct {
	ListedAt string  `json:"listed_at"`
	Type     string  `json:"type"`
	Show     *Show   `json:"show"`
}

type WatchlistSeason struct {
	ListedAt string  `json:"listed_at"`
	Type     string  `json:"type"`
	Season   *Object `json:"season"`
	Show     *Object `json:"show"`
}

type WatchlistEpisode struct {
	ListedAt string   `json:"listed_at"`
	Type     string   `json:"type"`
	Episode  *Episode `json:"episode"`
	Show     *Object  `json:"show"`
}

type WatchedShow struct {
	Plays			int		`json:"plays"`
	LastWatchedAt	string	`json:"last_watched_at"`
	Show 			*Show	`json:"show"`
}

type WatchedProgressShow struct {
	Aired			int			`json:"aired"`
	Completed		int			`json:"completed"`
	LastWatchedAt	string		`json:"last_watched_at"`
	Seasons			[]*Season	`json:"seasons"`
	HiddenSeasons	[]*Season	`json:"hidden_seasons"`
	NextEpisode		Episode		`json:"next_episode"`
}

type ProgressShow struct {
	Episode     *Episode `json:"episode"`
	Show        *Show    `json:"show"`
}

type CollectionMovie struct {
	CollectedAt string `json:"collected_at"`
	Movie       *Movie `json:"movie"`
}

type CollectionShow struct {
	CollectedAt string             `json:"last_collected_at"`
	Show        *Show              `json:"show"`
	Seasons     []*CollectedSeason `json:"seasons"`
}

type CollectedSeason struct {
	Number   int                 `json:"number"`
	Episodes []*CollectedEpisode `json:"episodes"`
}

type CollectedEpisode struct {
	CollectedAt string `json:"collected_at"`
	Number      int    `json:"number"`
}

type Images struct {
	Poster     *Sizes `json:"poster"`
	FanArt     *Sizes `json:"fanart"`
	ScreenShot *Sizes `json:"screenshot"`
	HeadShot   *Sizes `json:"headshot"`
	Logo       *Sizes `json:"logo"`
	ClearArt   *Sizes `json:"clearart"`
	Banner     *Sizes `json:"banner"`
	Thumbnail  *Sizes `json:"thumb"`
	Avatar     *Sizes `json:"avatar"`
}

type Sizes struct {
	Full      string `json:"full"`
	Medium    string `json:"medium"`
	Thumbnail string `json:"thumb"`
}

type IDs struct {
  Trakt  int    `json:"trakt"`
  IMDB   string `json:"imdb"`
	TMDB   int    `json:"tmdb"`
  TVDB   int    `json:"tvdb"`
	TVRage int    `json:"tvrage"`
  Slug   string `json:"slug"`
}

type Code struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type TokenRefresh struct {
	RefreshToken string `json:"refresh_token"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
	GrantType    string `json:"grant_type"`
}

type List struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Privacy        string `json:"privacy"`
	DisplayNumbers bool   `json:"display_numbers"`
	AllowComments  bool   `json:"allow_comments"`
	SortBy         string `json:"sort_by"`
	SortHow        string `json:"sort_how"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	ItemCount      int    `json:"item_count"`
	CommentCount   int    `json:"comment_count"`
	Likes          int    `json:"likes"`
	IDs            *IDs
}

type ListItem struct {
	Rank      int      `json:"rank"`
	ListedAt  string   `json:"listed_at"`
	Type      string   `json:"type"`
	Movie     *Movie   `json:"movie"`
	Show      *Show    `json:"show"`
	// Season    *Season  `json:"season"`
	// Episode   *Episode `json:"episode"`
}

type CalendarShow struct {
	FirstAired  string   `json:"first_aired"`
	Episode     *Episode `json:"episode"`
	Show        *Show    `json:"show"`
}

type CalendarMovie struct {
	Released   string `json:"released"`
	Movie      *Movie `json:"movie"`
}

func totalFromHeaders(headers http.Header) (total int, err error) {
	if len(headers) > 0 {
		if itemCount, exists := headers["X-Pagination-Item-Count"]; exists {
			if itemCount != nil {
				total, err = strconv.Atoi(itemCount[0])
				return
			}
			return -1, errors.New("X-Pagination-Item-Count was empty")
		}
		return -1, errors.New("No X-Pagination-Item-Count header found")
	}
	return -1, errors.New("No valid headers in request")
}

func newClearance() (err error) {
	retriesLeft -= 1
	log.Warningf("CloudFlared! User-Agent: %s - Cookies: %s", clearance.UserAgent, clearance.Cookies)

	if config.Get().UseCloudHole == false {
		retriesLeft = 0
		return errors.New("CloudFlared! Enable CloudHole in the add-on's settings.")
	}

	clearance, err = cloudhole.GetClearance()
	if err == nil {
		log.Noticef("New clearance: %s - %s", clearance.UserAgent, clearance.Cookies)
	} else {
		retriesLeft = 0
	}
	return err
}

func Get(endPoint string, params url.Values) (resp *napping.Response, err error) {
	header := http.Header{
		"Content-type": []string{"application/json"},
		"trakt-api-key": []string{ClientId},
		"trakt-api-version": []string{ApiVersion},
		"User-Agent": []string{clearance.UserAgent},
		"Cookie": []string{clearance.Cookies},
	}

	req := napping.Request{
		Url: fmt.Sprintf("%s/%s", ApiUrl, endPoint),
		Method: "GET",
		Params: &params,
		Header: &header,
	}

	rateLimiter.Call(func() {
		resp, err = napping.Send(&req)
		if err != nil {
			return
		} else if resp.Status() == 429 {
			log.Warningf("Rate limit exceeded getting %s, cooling down...", endPoint)
			rateLimiter.CoolDown(resp.HttpResponse().Header)
		} else if resp.Status() == 403 && retriesLeft > 0 {
			err = newClearance()
			if err == nil {
				resp, err = Get(endPoint, params)
			}
		}
	})
	return
}

func GetWithAuth(endPoint string, params url.Values) (resp *napping.Response, err error) {
	header := http.Header{
		"Content-type": []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Bearer %s", config.Get().TraktToken)},
		"trakt-api-key": []string{ClientId},
		"trakt-api-version": []string{ApiVersion},
		"User-Agent": []string{clearance.UserAgent},
		"Cookie": []string{clearance.Cookies},
	}

	req := napping.Request{
		Url: fmt.Sprintf("%s/%s", ApiUrl, endPoint),
		Method: "GET",
		Params: &params,
		Header: &header,
	}

	rateLimiter.Call(func() {
		resp, err = napping.Send(&req)
		if err != nil {
			return
		} else if resp.Status() == 429 {
			log.Warningf("Rate limit exceeded getting %s, cooling down...", endPoint)
			rateLimiter.CoolDown(resp.HttpResponse().Header)
		} else if resp.Status() == 403 && retriesLeft > 0 {
			err = newClearance()
			if err == nil {
				resp, err = GetWithAuth(endPoint, params)
			}
		}
	})
	return
}

func Post(endPoint string, payload *bytes.Buffer) (resp *napping.Response, err error) {
	header := http.Header{
		"Content-type": []string{"application/json"},
		"Authorization": []string{fmt.Sprintf("Bearer %s", config.Get().TraktToken)},
		"trakt-api-key": []string{ClientId},
		"trakt-api-version": []string{ApiVersion},
		"User-Agent": []string{clearance.UserAgent},
		"Cookie": []string{clearance.Cookies},
	}

	req := napping.Request{
		Url: fmt.Sprintf("%s/%s", ApiUrl, endPoint),
		Method: "POST",
		RawPayload: true,
		Payload: payload,
		Header: &header,
	}

	rateLimiter.Call(func() {
		resp, err = napping.Send(&req)
		if err != nil {
			return
		} else if resp.Status() == 429 {
			log.Warningf("Rate limit exceeded getting %s, cooling down...", endPoint)
			rateLimiter.CoolDown(resp.HttpResponse().Header)
		} else if resp.Status() == 403 && retriesLeft > 0 {
			err = newClearance()
			if err == nil {
				resp, err = Post(endPoint, payload)
			}
		}
	})
	return
}

func GetCode() (code *Code, err error) {
	endPoint := "oauth/device/code"
	header := http.Header{
		"Content-type": []string{"application/json"},
		"User-Agent": []string{clearance.UserAgent},
		"Cookie": []string{clearance.Cookies},
	}
	params := napping.Params{
		"client_id": ClientId,
	}.AsUrlValues()

	req := napping.Request{
		Url: fmt.Sprintf("%s/%s", ApiUrl, endPoint),
		Method: "POST",
		Params: &params,
		Header: &header,
	}

	var resp *napping.Response
	rateLimiter.Call(func() {
		resp, err = napping.Send(&req)
		if err != nil {
			err = resp.Unmarshal(&code)
			return
		} else if resp.Status() == 429 {
			log.Warningf("Rate limit exceeded getting Trakt code %s, cooling down...", code)
			rateLimiter.CoolDown(resp.HttpResponse().Header)
		} else if resp.Status() == 403 && retriesLeft > 0 {
			err = newClearance()
			if err == nil {
				code, err = GetCode()
			}
		} else {
			resp.Unmarshal(&code)
		}
	})
	if err == nil && resp.Status() != 200 {
		err = errors.New(fmt.Sprintf("Unable to get Trakt code: %d", resp.Status()))
	}
	return
}

func GetToken(code string) (resp *napping.Response, err error) {
	endPoint := "oauth/device/token"
	header := http.Header{
		"Content-type": []string{"application/json"},
		"User-Agent": []string{clearance.UserAgent},
		"Cookie": []string{clearance.Cookies},
	}
	params := napping.Params{
		"code": code,
		"client_id": ClientId,
		"client_secret": ClientSecret,
	}.AsUrlValues()

	req := napping.Request{
		Url: fmt.Sprintf("%s/%s", ApiUrl, endPoint),
		Method: "POST",
		Params: &params,
		Header: &header,
	}


	rateLimiter.Call(func() {
		resp, err = napping.Send(&req)
		if err != nil {
			return
		} else if resp.Status() == 429 {
			log.Warningf("Rate limit exceeded getting Trakt token with code %s, cooling down...", code)
			rateLimiter.CoolDown(resp.HttpResponse().Header)
		} else if resp.Status() == 403 && retriesLeft > 0 {
			err = newClearance()
			if err == nil {
				resp, err = GetToken(code)
			}
		}
	})
	return
}

func PollToken(code *Code) (token *Token, err error) {
	startInterval := code.Interval
	interval := time.NewTicker(time.Duration(startInterval) * time.Second)
	defer interval.Stop()
	expired := time.NewTicker(time.Duration(code.ExpiresIn) * time.Second)
	defer expired.Stop()

	for {
		select {
		case <-interval.C:
			resp, err := GetToken(code.DeviceCode)
			if err != nil {
				return nil, err
			}
			if resp.Status() == 200 {
				resp.Unmarshal(&token)
				return token, err
			} else if resp.Status() == 400 {
				break
			} else if resp.Status() == 404 {
				err = errors.New("Invalid device code.")
				return nil, err
			} else if resp.Status() == 409 {
				err = errors.New("Code already used.")
				return nil, err
			} else if resp.Status() == 410 {
				err = errors.New("Code expired.")
				return nil, err
			} else if resp.Status() == 418 {
				err = errors.New("Code denied.")
				return nil, err
			} else if resp.Status() == 429 {
				// err = errors.New("Polling too quickly.")
				interval.Stop()
				interval = time.NewTicker(time.Duration(startInterval + 5) * time.Second)
				break
			}

		case <-expired.C:
			err = errors.New("Code expired, please try again.")
			return nil, err
		}
	}
}

func RefreshToken() (resp *napping.Response, err error) {
	endPoint := "oauth/token"
	header := http.Header{
		"Content-type": []string{"application/json"},
		"User-Agent": []string{clearance.UserAgent},
		"Cookie": []string{clearance.Cookies},
	}
	params := napping.Params{
		"refresh_token": config.Get().TraktRefreshToken,
		"client_id": ClientId,
		"client_secret": ClientSecret,
		"redirect_uri": "urn:ietf:wg:oauth:2.0:oob",
		"grant_type": "refresh_token",
	}.AsUrlValues()

	req := napping.Request{
		Url: fmt.Sprintf("%s/%s", ApiUrl, endPoint),
		Method: "POST",
		Params: &params,
		Header: &header,
	}


	resp, err = napping.Send(&req)
	if err != nil {
		return
	} else if resp.Status() == 403 && retriesLeft > 0 {
		err = newClearance()
		if err == nil {
			resp, err = RefreshToken()
		}
	}
	return
}

func TokenRefreshHandler() {
	if config.Get().TraktToken == "" {
		return
	}

	var token *Token
	ticker := time.NewTicker(12 * time.Hour)
	for {
		select {
		case <-ticker.C:
			if time.Now().Unix() > int64(config.Get().TraktTokenExpiry) - int64(259200) {
				resp, err := RefreshToken()
				if err != nil {
					xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
					log.Error(err)
					return
				} else {
					if resp.Status() == 200 {
						if err := resp.Unmarshal(&token); err != nil {
							xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
							log.Error(err)
						} else {
							expiry := time.Now().Unix() + int64(token.ExpiresIn)
							xbmc.SetSetting("trakt_token_expiry", strconv.Itoa(int(expiry)))
							xbmc.SetSetting("trakt_token", token.AccessToken)
							xbmc.SetSetting("trakt_refresh_token", token.RefreshToken)
							log.Noticef("Token refreshed for Trakt authorization, next refresh in %s", time.Duration(token.ExpiresIn - 259200) * time.Second)
						}
					} else {
						err = errors.New(fmt.Sprintf("Bad status while refreshing Trakt token: %d", resp.Status()))
						xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
						log.Error(err)
					}
				}
			}
		}
	}
}

func Authorize(fromSettings bool) error {
	code, err := GetCode()

	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
		return err
	}
	log.Noticef("Got code for %s: %s", code.VerificationURL, code.UserCode)

	if xbmc.Dialog("LOCALIZE[30058]", fmt.Sprintf("Visit %s and enter your code: %s", code.VerificationURL, code.UserCode)) == false {
		return errors.New("Authentication canceled.")
	}

	token, err := PollToken(code)

	if err != nil {
		xbmc.Notify("Quasar", err.Error(), config.AddonIcon())
		return err
	}

	success := "Woohoo!"
	if fromSettings {
		success += " (Save your settings!)"
	}

	expiry := time.Now().Unix() + int64(token.ExpiresIn)
	xbmc.SetSetting("trakt_token_expiry", strconv.Itoa(int(expiry)))
	xbmc.SetSetting("trakt_token", token.AccessToken)
	xbmc.SetSetting("trakt_refresh_token", token.RefreshToken)

	xbmc.Notify("Quasar", success, config.AddonIcon())
	return nil
}

func Authorized() error {
	if config.Get().TraktToken == "" {
		err := Authorize(false)
		if err != nil {
			return err
		}
	}
	return nil
}

func AddToWatchlist(itemType string, tmdbId string) (resp *napping.Response, err error) {
	if err := Authorized(); err != nil {
		return nil, err
	}

	endPoint := "sync/watchlist"
	return Post(endPoint, bytes.NewBufferString(fmt.Sprintf(`{"%s": [{"ids": {"tmdb": %s}}]}`, itemType, tmdbId)))
}

func RemoveFromWatchlist(itemType string, tmdbId string) (resp *napping.Response, err error) {
	if err := Authorized(); err != nil {
		return nil, err
	}

	endPoint := "sync/watchlist/remove"
	return Post(endPoint, bytes.NewBufferString(fmt.Sprintf(`{"%s": [{"ids": {"tmdb": %s}}]}`, itemType, tmdbId)))
}

func AddToCollection(itemType string, tmdbId string) (resp *napping.Response, err error) {
	if err := Authorized(); err != nil {
		return nil, err
	}

	endPoint := "sync/collection"
	return Post(endPoint, bytes.NewBufferString(fmt.Sprintf(`{"%s": [{"ids": {"tmdb": %s}}]}`, itemType, tmdbId)))
}

func RemoveFromCollection(itemType string, tmdbId string) (resp *napping.Response, err error) {
	if err := Authorized(); err != nil {
		return nil, err
	}

	endPoint := "sync/collection/remove"
	return Post(endPoint, bytes.NewBufferString(fmt.Sprintf(`{"%s": [{"ids": {"tmdb": %s}}]}`, itemType, tmdbId)))
}

func Scrobble(action string, contentType string, tmdbId int, watched float64, runtime float64) {
	if err := Authorized(); err != nil {
		return
	}

	if runtime < 1 {
		return
	}
	progress := watched / runtime * 100

	log.Noticef("%s %s: %f%%, watched: %fs, duration: %fs", action, contentType, progress, watched, runtime)

	endPoint := fmt.Sprintf("scrobble/%s", action)
	payload := fmt.Sprintf(`{"%s": {"ids": {"tmdb": %d}}, "progress": %f, "app_version": "%s"}`,
	                       contentType, tmdbId, progress, util.Version[1:len(util.Version) - 1])
	resp, err := Post(endPoint, bytes.NewBufferString(payload))
	if err != nil {
		log.Error(err.Error())
		xbmc.Notify("Quasar", "Scrobble failed, check your logs.", config.AddonIcon())
	} else if resp.Status() != 201 {
		log.Errorf("Failed to scrobble %s #%d to %s at %f: %d", contentType, tmdbId, action, progress, resp.Status())
	}
}
