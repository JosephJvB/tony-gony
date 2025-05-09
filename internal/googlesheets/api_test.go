package googlesheets

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestGoogleSheets(t *testing.T) {

	t.Run("append to undefined map key", func(t *testing.T) {
		m := map[int][]string{}

		// m[20] is not set - will append throw?
		m[20] = append(m[20], "123")

		// t.Logf("%v", m)

		if len(m[20]) != 1 {
			t.Error("something went wrong")
		}
	})
	t.Run("can load videos from google sheets", func(t *testing.T) {
		t.Skip("skip test calling real google sheets api")

		err := godotenv.Load("../../.env")
		if err != nil {
			panic(err)
		}

		// .env file does not handle private keys gracefully
		// probably would be better saved to a file than in .env. Oh well.
		invalidKey := os.Getenv("GOOGLE_SHEETS_PRIVATE_KEY")
		fixedKey := strings.ReplaceAll(invalidKey, "__n__", "\n")

		os.Setenv("GOOGLE_SHEETS_PRIVATE_KEY", fixedKey)

		gs := NewClient(Secrets{
			Email:      os.Getenv("GOOGLE_SHEETS_EMAIL"),
			PrivateKey: fixedKey,
		})

		gs.LoadScrapedTracks()

		if len(gs.scrapedTracks) == 0 {
			t.Errorf("Expected parsed videos to be loaded")
		}
		if len(gs.ScrapedTracksMap) == 0 {
			t.Errorf("Expected parsed videos map to be loaded")
		}

		b, err := json.MarshalIndent(gs.scrapedTracks, "", "	")
		if err != nil {
			panic(err)
		}

		err = os.WriteFile("../../data/scraped-tracks.json", b, 0666)
		if err != nil {
			panic(err)
		}
	})

	t.Run("can append tracks to google sheets", func(t *testing.T) {
		t.Skip("skip test calling real google sheets api")

		err := godotenv.Load("../../.env")
		if err != nil {
			panic(err)
		}

		// .env file does not handle private keys gracefully
		// probably would be better saved to a file than in .env. Oh well.
		invalidKey := os.Getenv("GOOGLE_SHEETS_PRIVATE_KEY")
		fixedKey := strings.ReplaceAll(invalidKey, "__n__", "\n")

		os.Setenv("GOOGLE_SHEETS_PRIVATE_KEY", fixedKey)

		gs := NewClient(Secrets{
			Email:      os.Getenv("GOOGLE_SHEETS_EMAIL"),
			PrivateKey: fixedKey,
		})

		toAdd := []ScrapedTrackRow{
			{
				Id:      "",
				Title:   "song 9",
				Artist:  "artist 9",
				Album:   "album 9",
				Year:    2023,
				Found:   true,
				AddedAt: "2024-04-16T00:00:00.000Z",
			},
			{
				Id:      "",
				Title:   "song 2",
				Artist:  "artist 2",
				Album:   "album 2",
				Year:    2025,
				Found:   true,
				AddedAt: "2025-04-16T00:00:00.000Z",
			},
		}

		gs.AddNextRows(toAdd)
	})
}
