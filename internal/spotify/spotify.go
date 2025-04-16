package spotify

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"tony-gony/internal/scraping"
)

const ApiBaseUrl = "https://api.spotify.com/v1"
const AccountsBaseUrl = "https://accounts.spotify.com/api"

type ISpotifyClient interface {
	loadBasicToken()
	loadAccessToken()
	GetMyPlaylists()
}

type SpotifyClient struct {
	clientId     string
	clientSecret string
	refreshToken string
	basicToken   string
	accessToken  string
}

type Secrets struct {
	ClientId     string
	ClientSecret string
	RefreshToken string
}

func NewClient(secrets Secrets) SpotifyClient {
	return SpotifyClient{
		clientId:     secrets.ClientId,
		clientSecret: secrets.ClientSecret,
		refreshToken: secrets.RefreshToken,
	}
}

func (s *SpotifyClient) loadBasicToken() {
	apiUrl := AccountsBaseUrl + "/token"

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", s.clientId)
	data.Set("client_secret", s.clientSecret)

	postData := strings.NewReader(data.Encode())

	resp, err := http.Post(apiUrl, "application/x-www-form-urlencoded", postData)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode > 299 {
		b := new(strings.Builder)
		io.Copy(b, resp.Body)
		log.Print(b.String())
		log.Fatalf("\nLoadBasicToken failed: \"%s\"", resp.Status)
	}

	tokenResponse := SpotifyTokenResponse{}
	json.NewDecoder(resp.Body).Decode(&tokenResponse)

	s.basicToken = tokenResponse.AccessToken
}

func (s *SpotifyClient) loadAccessToken() {
	apiUrl := AccountsBaseUrl + "/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", s.refreshToken)
	postData := strings.NewReader(data.Encode())

	req, _ := http.NewRequest("POST", apiUrl, postData)

	req.SetBasicAuth(s.clientId, s.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode > 299 {
		b := new(strings.Builder)
		io.Copy(b, resp.Body)
		log.Print(b.String())
		log.Fatalf("\nLoadAccessToken failed: \"%s\"", resp.Status)
	}

	tokenResponse := SpotifyTokenResponse{}
	json.NewDecoder(resp.Body).Decode(&tokenResponse)

	s.accessToken = tokenResponse.AccessToken
}

func (s *SpotifyClient) GetMyPlaylists() []SpotifyPlaylist {
	if s.accessToken == "" {
		s.loadAccessToken()
	}

	apiUrl := ApiBaseUrl + "/me/playlists"

	queryPart := url.Values{}
	queryPart.Set("limit", "50")

	apiUrl += "?" + queryPart.Encode()

	playlists := getPaginatedItems[SpotifyPlaylist](apiUrl, s.accessToken)

	return playlists
}

func getPaginatedItems[T any](startUrl string, token string) []T {
	apiUrl := startUrl

	items := []T{}

	for apiUrl != "" {
		req, _ := http.NewRequest("GET", apiUrl, nil)

		authHeaderValue := "Bearer " + token
		req.Header.Set("Authorization", authHeaderValue)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		if resp.StatusCode > 299 {
			b := new(strings.Builder)
			io.Copy(b, resp.Body)
			log.Print(b.String())
			log.Fatalf("\ngetPaginatedItems failed: \"%s\"\n%s\n", resp.Status, apiUrl)
		}

		responseBody := PaginatedResponse[T]{}
		json.NewDecoder(resp.Body).Decode(&responseBody)

		items = append(items, responseBody.Items...)
		apiUrl = responseBody.Next
	}

	return items
}

func (s *SpotifyClient) GetPlaylistItems(playlistId string) []SpotifyPlaylistItem {
	if s.accessToken == "" {
		s.loadAccessToken()
	}

	apiUrl := ApiBaseUrl + "/playlists/" + playlistId + "/tracks"

	items := getPaginatedItems[SpotifyPlaylistItem](apiUrl, s.accessToken)

	return items
}

func (s *SpotifyClient) FindTrack(t scraping.ScrapedTrack) []SpotifyTrack {
	if s.accessToken == "" {
		s.loadAccessToken()
	}

	trackQuery := "track:" + t.Title
	trackQuery += " artist:" + t.Artist
	// year is a bit sketchy
	// mostly should be fine
	// but if it's january - maybe Tony's added songs from previous year?
	// found that with "My Golden Years - Lemon Twigs"
	// trackQuery += " year:" + strconv.Itoa(t.Year)
	// Noticed cases where apple music adds " - EP" | " - Single" to album suffix
	// and that doesn't match Spotify records
	// So I think that would break the spotify query
	// could trim those suffixes, but prefer to not use album at all
	// trust that title and artist will be more consistent between apple/spotify
	// if t.Album != "" {
	// trackQuery += " album:" + t.Album
	// }

	queryPart := url.Values{}
	queryPart.Set("q", trackQuery)
	queryPart.Set("type", "track")
	queryPart.Set("limit", "1")

	apiUrl := ApiBaseUrl + "/search?" + queryPart.Encode()

	req, _ := http.NewRequest("GET", apiUrl, nil)

	authHeaderValue := "Bearer " + s.accessToken
	req.Header.Set("Authorization", authHeaderValue)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode > 299 {
		b := new(strings.Builder)
		io.Copy(b, resp.Body)
		log.Print(b.String())
		log.Fatalf("\nFindTrack failed: \"%s\"\n%s\n", resp.Status, apiUrl)
	}

	trackResponse := SpotifyTrackSearchResults{}
	json.NewDecoder(resp.Body).Decode(&trackResponse)

	return trackResponse.Tracks.Items
}

func (s *SpotifyClient) CreatePlaylist(name string) SpotifyPlaylist {
	if s.accessToken == "" {
		s.loadAccessToken()
	}

	apiUrl := ApiBaseUrl + "/users/" + JvbSpotifyId + "/playlists"

	data := map[string]any{
		"name":          name,
		"description":   "", // TODO: description
		"public":        true,
		"collaborative": false,
	}
	b, _ := json.Marshal(data)
	postData := strings.NewReader(string(b))

	req, _ := http.NewRequest("POST", apiUrl, postData)

	authHeaderValue := "Bearer " + s.accessToken
	req.Header.Set("Authorization", authHeaderValue)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode > 299 {
		b := new(strings.Builder)
		io.Copy(b, resp.Body)
		log.Print(b.String())
		log.Fatalf("\nCreatePlaylist failed: \"%s\"", resp.Status)
	}

	responseBody := SpotifyPlaylist{}
	json.NewDecoder(resp.Body).Decode(&responseBody)

	return responseBody
}

func (s *SpotifyClient) AddPlaylistItems(playlistId string, trackUris []string) {
	if s.accessToken == "" {
		s.loadAccessToken()
	}

	apiUrl := ApiBaseUrl + "/playlists/" + playlistId + "/tracks"

	l := len(trackUris)

	for i := 0; i < l; i += 100 {
		// there is a math.Min() method but it takes floats
		// so i need to convert to float then from float, sack that off
		upper := i + 100
		if upper > l {
			upper = l
		}

		uris := trackUris[i:upper]

		data := map[string]any{
			"uris": uris,
		}
		b, _ := json.Marshal(data)
		postData := strings.NewReader(string(b))

		req, _ := http.NewRequest("POST", apiUrl, postData)

		authHeaderValue := "Bearer " + s.accessToken
		req.Header.Set("Authorization", authHeaderValue)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}

		if resp.StatusCode > 299 {
			b := new(strings.Builder)
			io.Copy(b, resp.Body)
			log.Print(b.String())
			log.Fatalf("\nAddPlaylistItems failed: \"%s\"", resp.Status)
		}
	}
}
