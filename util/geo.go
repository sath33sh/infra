package util

import (
	"fmt"
	"github.com/sath33sh/infra/log"
	"sync"
	"time"
)

// Geometry types.
const (
	POINT = "Point"
)

// Geometry in GeoJSON format: http://geojson.org.
type Geometry struct {
	Type        string     `json:"type,omitempty"`        // Geometry type: "Point", etc.
	Coordinates [2]float64 `json:"coordinates,omitempty"` // Coordinates: [lat, lon]
}

// Rate limit for Google Geocode API calls.
var rateLimit struct {
	sync.Mutex           // Lock.
	lastCall   time.Time // Last call timestamp.
}

// Google maps geocode API result.
type GoogleGeocodeResult struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Bounds struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"bounds"`
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PlaceID string   `json:"place_id"`
		Types   []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

func LookupAddress(address string) (geo Geometry, err error) {
	var gr GoogleGeocodeResult

	// Rate limit the API call.
	rateLimit.Lock()
	defer func() {
		rateLimit.Unlock()
	}()

	retry := 0
	for retry < 3 {
		// Google allows about 5 calls per second, but let's be conservative.
		intvl := time.Now().Sub(rateLimit.lastCall)
		if intvl < (500 * time.Millisecond) {
			time.Sleep(500 * time.Millisecond)
		}

		url := fmt.Sprintf("http://maps.googleapis.com/maps/api/geocode/json?address=%s", address)
		err = HttpJsonGet(url, &gr)
		rateLimit.lastCall = time.Now()
		if err != nil {
			return geo, err
		}

		if gr.Status != "OK" {
			if gr.Status == "OVER_QUERY_LIMIT" {
				time.Sleep(time.Second)
			} else {
				log.Errorf("Invalid status %s", gr.Status)
				return geo, ErrInternal
			}
		} else {
			// Got result.
			break
		}
		retry++
	}

	geo.Type = POINT
	geo.Coordinates[0] = gr.Results[0].Geometry.Location.Lat
	geo.Coordinates[1] = gr.Results[0].Geometry.Location.Lng

	return geo, nil
}
