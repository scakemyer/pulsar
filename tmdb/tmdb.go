package tmdb

import (
	"fmt"
	"math/rand"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/charly3pins/magnetar/cache"
	"github.com/charly3pins/magnetar/config"
	"github.com/charly3pins/magnetar/util"
	"github.com/charly3pins/magnetar/xbmc"

	"github.com/jmcvetta/napping"
	"github.com/op/go-logging"
)

const (
	PagesAtOnce    = 5
	ResultsPerPage = 20
)

var (
	log = logging.MustGetLogger("tmdb")
)

type Movies []*Movie
type Shows []*Show
type SeasonList []*Season
type EpisodeList []*Episode

type Movie struct {
	Entity

	IMDBId              string       `json:"imdb_id"`
	Overview            string       `json:"overview"`
	ProductionCompanies []*IdName    `json:"production_companies"`
	Runtime             int          `json:"runtime"`
	TagLine             string       `json:"tagline"`
	RawPopularity       interface{}  `json:"popularity"`
	Popularity          float64      `json:"-"`
	SpokenLanguages     []*Language  `json:"spoken_languages"`
	ExternalIDs         *ExternalIDs `json:"external_ids"`

	AlternativeTitles *struct {
		Titles []*AlternativeTitle `json:"titles"`
	} `json:"alternative_titles"`

	Translations *struct {
		Translations []*Language `json:"translations"`
	} `json:"translations"`

	Trailers *struct {
		Youtube []*Trailer `json:"youtube"`
	} `json:"trailers"`

	Credits *Credits `json:"credits,omitempty"`
	Images  *Images  `json:"images,omitempty"`

	ReleaseDates *ReleaseDatesResults `json:"release_dates"`
}

type Show struct {
	Entity

	EpisodeRunTime      []int        `json:"episode_run_time"`
	Genres              []*Genre     `json:"genres"`
	Homepage            string       `json:"homepage"`
	InProduction        bool         `json:"in_production"`
	FirstAirDate        string       `json:"first_air_date"`
	LastAirDate         string       `json:"last_air_date"`
	Networks            []*IdName    `json:"networks"`
	NumberOfEpisodes    int          `json:"number_of_episodes"`
	NumberOfSeasons     int          `json:"number_of_seasons"`
	OriginalName        string       `json:"original_name"`
	OriginCountry       []string     `json:"origin_country"`
	Overview            string       `json:"overview"`
	RawPopularity       interface{}  `json:"popularity"`
	Popularity          float64      `json:"-"`
	ProductionCompanies []*IdName    `json:"production_companies"`
	Status              string       `json:"status"`
	ExternalIDs         *ExternalIDs `json:"external_ids"`
	Translations        *struct {
		Translations []*Language `json:"translations"`
	} `json:"translations"`

	Credits *Credits `json:"credits,omitempty"`
	Images  *Images  `json:"images,omitempty"`

	Seasons SeasonList `json:"seasons"`
}

type Season struct {
	Id           int          `json:"id"`
	Name         string       `json:"name,omitempty"`
	Season       int          `json:"season_number"`
	EpisodeCount int          `json:"episode_count,omitempty"`
	AirDate      string       `json:"air_date"`
	Poster       string       `json:"poster_path"`
	ExternalIDs  *ExternalIDs `json:"external_ids"`

	Episodes EpisodeList `json:"episodes"`
}

type Episode struct {
	Id            int          `json:"id"`
	Name          string       `json:"name"`
	Overview      string       `json:"overview"`
	AirDate       string       `json:"air_date"`
	SeasonNumber  int          `json:"season_number"`
	EpisodeNumber int          `json:"episode_number"`
	VoteAverage   float32      `json:"vote_average"`
	StillPath     string       `json:"still_path"`
	ExternalIDs   *ExternalIDs `json:"external_ids"`
}

type Entity struct {
	IsAdult       bool      `json:"adult"`
	BackdropPath  string    `json:"backdrop_path"`
	Id            int       `json:"id"`
	Genres        []*IdName `json:"genres"`
	OriginalTitle string    `json:"original_title,omitempty"`
	ReleaseDate   string    `json:"release_date"`
	FirstAirDate  string    `json:"first_air_date"`
	PosterPath    string    `json:"poster_path"`
	Title         string    `json:"title,omitempty"`
	VoteAverage   float32   `json:"vote_average"`
	VoteCount     int       `json:"vote_count"`
	OriginalName  string    `json:"original_name,omitempty"`
	Name          string    `json:"name,omitempty"`
}

type EntityList struct {
	Page         int       `json:"page"`
	Results      []*Entity `json:"results"`
	TotalPages   int       `json:"total_pages"`
	TotalResults int       `json:"total_results"`
}

type IdName struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type Genre IdName

type GenreList struct {
	Genres []*Genre `json:"genres"`
}

type Image struct {
	FilePath  string `json:"file_path"`
	Height    int    `json:"height"`
	ISO_639_1 string `json:"iso_639_1"`
	Width     int    `json:"width"`
}

type Images struct {
	Backdrops []*Image `json:"backdrops"`
	Posters   []*Image `json:"posters"`
	Stills    []*Image `json:"stills"`
}

type Cast struct {
	IdName
	CastId      int    `json:"cast_id"`
	Character   string `json:"character"`
	CreditId    string `json:"credit_id"`
	Order       int    `json:"order"`
	ProfilePath string `json:"profile_path"`
}

type Crew struct {
	IdName
	CreditId    string `json:"credit_id"`
	Department  string `json:"department"`
	Job         string `json:"job"`
	ProfilePath string `json:"profile_path"`
}

type Credits struct {
	Cast []*Cast `json:"cast"`
	Crew []*Crew `json:"crew"`
}

type ExternalIDs struct {
	IMDBId      string      `json:"imdb_id"`
	FreeBaseID  string      `json:"freebase_id"`
	FreeBaseMID string      `json:"freebase_mid"`
	TVDBID      interface{} `json:"tvdb_id"`
}

type AlternativeTitle struct {
	ISO_3166_1 string `json:"iso_3166_1"`
	Title      string `json:"title"`
}

type Language struct {
	ISO_639_1   string `json:"iso_639_1"`
	Name        string `json:"name"`
	EnglishName string `json:"english_name,omitempty"`
}

type FindResult struct {
	MovieResults     []*Entity `json:"movie_results"`
	PersonResults    []*Entity `json:"person_results"`
	TVResults        []*Entity `json:"tv_results"`
	TVEpisodeResults []*Entity `json:"tv_episode_results"`
	TVSeasonResults  []*Entity `json:"tv_season_results"`
}

type List struct {
	CreatedBy     string    `json:"created_by"`
	Description   string    `json:"description"`
	FavoriteCount int       `json:"favorite_count"`
	Id            string    `json:"id"`
	ItemCount     int       `json:"item_count"`
	ISO_639_1     string    `json:"iso_639_1"`
	Name          string    `json:"name"`
	PosterPath    string    `json:"poster_path"`
	Items         []*Entity `json:"items"`
}

type Trailer struct {
	Name   string `json:"name"`
	Size   string `json:"size"`
	Source string `json:"source"`
	Type   string `json:"type"`
}

type ReleaseDatesResults struct {
	Results []*ReleaseDates `json:"results"`
}

type ReleaseDates struct {
	ISO_3166_1   string         `json:"iso_3166_1"`
	ReleaseDates []*ReleaseDate `json:"release_dates"`
}

type ReleaseDate struct {
	Certification string `json:"certification"`
	ISO_639_1     string `json:"iso_639_1"`
	Note          string `json:"note"`
	ReleaseDate   string `json:"release_date"`
	Type          int    `json:"type"`
}

const (
	tmdbEndpoint            = "https://api.themoviedb.org/3/"
	imageEndpoint           = "http://image.tmdb.org/t/p/"
	burstRate               = 35
	burstTime               = 15 * time.Second
	simultaneousConnections = 20
	cacheExpiration         = 6 * 24 * time.Hour
	recentExpiration        = 15 * time.Minute
	imagesCacheExpiration   = 14 * 24 * time.Hour
)

var (
	apiKeys = []string{
		"8cf43ad9c085135b9479ad5cf6bbcbda",
		"ae4bd1b6fce2a5648671bfc171d15ba4",
	}
	apiKey    = apiKeys[rand.Intn(len(apiKeys))]
	WarmingUp = true
)

var rateLimiter = util.NewRateLimiter(burstRate, burstTime, simultaneousConnections)

func CheckApiKey() {
	log.Info("Checking TMDB API key...")

	customApiKey := config.Get().TMDBApiKey
	if customApiKey != "" {
		apiKeys = append(apiKeys, customApiKey)
		apiKey = customApiKey
	}

	result := false
	for index := len(apiKeys) - 1; index >= 0; index-- {
		result = tmdbCheck(apiKey)
		if result {
			log.Noticef("TMDB API key check passed, using %s...", apiKey[:7])
			break
		} else {
			log.Warningf("TMDB API key failed: %s", apiKey)
			if apiKey == apiKeys[index] {
				apiKeys = append(apiKeys[:index], apiKeys[index+1:]...)
			}
			if len(apiKeys) > 0 {
				apiKey = apiKeys[rand.Intn(len(apiKeys))]
			} else {
				result = false
				break
			}
		}
	}
	if result == false {
		log.Error("No valid TMDB API key found")
	}
}

func tmdbCheck(key string) bool {
	var result *Entity

	urlValues := napping.Params{
		"api_key": key,
	}.AsUrlValues()

	resp, err := napping.Get(
		tmdbEndpoint+"movie/550",
		&urlValues,
		&result,
		nil,
	)

	if err != nil {
		log.Error(err.Error())
		xbmc.Notify("Magnetar", "TMDB check failed, check your logs.", config.AddonIcon())
		return false
	} else if resp.Status() != 200 {
		return false
	}

	return true
}

func ImageURL(uri string, size string) string {
	return imageEndpoint + size + uri
}

// TODO Unused...
func ListEntities(endpoint string, params napping.Params) []*Entity {
	var wg sync.WaitGroup
	resultsPerPage := config.Get().ResultsPerPage
	entities := make([]*Entity, PagesAtOnce*resultsPerPage)
	params["api_key"] = apiKey
	params["language"] = "en"

	wg.Add(PagesAtOnce)
	for i := 0; i < PagesAtOnce; i++ {
		go func(page int) {
			defer wg.Done()
			var tmp *EntityList
			tmpParams := napping.Params{
				"page": strconv.Itoa(page),
			}
			for k, v := range params {
				tmpParams[k] = v
			}
			urlValues := tmpParams.AsUrlValues()
			rateLimiter.Call(func() {
				resp, err := napping.Get(
					tmdbEndpoint+endpoint,
					&urlValues,
					&tmp,
					nil,
				)
				if err != nil {
					log.Error(err.Error())
					xbmc.Notify("Magnetar", "Failed listing entities, check your logs.", config.AddonIcon())
				} else if resp.Status() != 200 {
					message := fmt.Sprintf("Bad status listing entities: %d", resp.Status())
					log.Error(message)
					xbmc.Notify("Magnetar", message, config.AddonIcon())
				}
			})
			for i, entity := range tmp.Results {
				entities[page*resultsPerPage+i] = entity
			}
		}(i)
	}
	wg.Wait()

	return entities
}

// TODO Actually use this somewhere?
func Find(externalId string, externalSource string) *FindResult {
	var result *FindResult

	cacheStore := cache.NewFileStore(path.Join(config.Get().ProfilePath, "cache"))
	key := fmt.Sprintf("com.tmdb.find.%s.%s", externalSource, externalId)
	if err := cacheStore.Get(key, &result); err != nil {
		rateLimiter.Call(func() {
			urlValues := napping.Params{
				"api_key":         apiKey,
				"external_source": externalSource,
			}.AsUrlValues()
			resp, err := napping.Get(
				tmdbEndpoint+"find/"+externalId,
				&urlValues,
				&result,
				nil,
			)
			if err != nil {
				log.Error(err.Error())
				xbmc.Notify("Magnetar", "Failed Find call, check your logs.", config.AddonIcon())
			} else if resp.Status() != 200 {
				message := fmt.Sprintf("Find call bad status: %d", resp.Status())
				log.Error(message)
				xbmc.Notify("Magnetar", message, config.AddonIcon())
			}
			cacheStore.Set(key, result, 15*time.Minute)
		})
	}

	return result
}
