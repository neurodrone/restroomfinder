package googleapi

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	googleGeoCodeAPI = "https://maps.googleapis.com/maps/api/geocode/json"
)

var (
	digitRegex = regexp.MustCompile("[0-9]*")

	client = http.DefaultClient
)

type LocationJSON struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat json.Number `json:"lat"`
				Lng json.Number `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
	Status string `json:"status"`
}

func (lj *LocationJSON) Location() (*Location, error) {
	if len(lj.Results) == 0 {
		return nil, errors.New("no results found")
	}

	result := lj.Results[0]

	latitude, err := result.Geometry.Location.Lat.Float64()
	if err != nil {
		return nil, err
	}

	longitude, err := result.Geometry.Location.Lng.Float64()
	if err != nil {
		return nil, err
	}

	return &Location{
		FormattedAddress: result.FormattedAddress,
		Latitude:         latitude,
		Longitude:        longitude,
	}, nil
}

type Location struct {
	FormattedAddress    string
	Latitude, Longitude float64
}

func LocationInfo(address string) (*Location, error) {
	u, err := url.Parse(googleGeoCodeAPI)
	if err != nil {
		return nil, err
	}

	uv := &url.Values{}
	uv.Add("address", formatURL(address))
	uv.Add("key", os.Getenv("GOOGLE_GEOCODE_API_KEY"))

	u.RawQuery = uv.Encode()

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lj LocationJSON
	if err := json.NewDecoder(resp.Body).Decode(&lj); err != nil {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		return nil, err
	}

	return lj.Location()
}

func formatURL(address string) string {
	address = strings.Replace(address, "&", "and", -1)
	address = strings.Replace(address, " ", "+", -1)
	return digitRegex.ReplaceAllStringFunc(address, func(s string) string {
		switch n, err := strconv.Atoi(s); {
		case err != nil:
			return s
		case n%10 == 1:
			return s + "st"
		case n%10 == 2:
			return s + "nd"
		case n%10 == 3:
			return s + "rd"
		default:
			return s + "th"
		}
		return s
	})
}
