package restroomfinder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/neurodrone/restroomfinder/googleapi"
)

const (
	approxTotalLooCount = 1000
)

var (
	ErrUnimplemented = errors.New("not implemented")

	validRestroomInfoIndex = []string{
		"",
		"UUID", "",
		"CreatedAtUnix", "",
		"UpdatedAtUnix", "", "",
		"Name",
		"Location",
		"OpenAllYear",
		"HandicapOk",
		"Borough", "",
	}
)

type LocationDiff struct {
	*LooInfo
	Distance float64
}

func NewLocationDiff(info *LooInfo, distance float64) LocationDiff {
	return LocationDiff{info, distance}
}

type LooInfo struct {
	UUID          string
	Coordinates   *Geo
	CreatedAtUnix int64
	UpdatedAtUnix int64
	Name          string
	Location      string
	Borough       string
	HandicapOk    bool
	OpenAllYear   bool
}

type LooInfos []*LooInfo

func (l LooInfos) Len() int      { return len(l) }
func (l LooInfos) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l LooInfos) Less(i, j int) bool {
	if l[i].Coordinates.Latitude == l[j].Coordinates.Longitude {
		return l[i].Coordinates.Longitude < l[j].Coordinates.Longitude
	}
	return l[i].Coordinates.Latitude < l[j].Coordinates.Latitude
}

func (l LooInfos) FindClosest(geo *Geo, k int) []LocationDiff {
	ld := make([]LocationDiff, 0, k)

	if k >= len(l) {
		for _, loo := range l {
			dh := l.haversine(geo, loo.Coordinates)
			ld = append(ld, NewLocationDiff(loo, dh))
		}
		return ld
	}

	index := sort.Search(len(l), func(i int) bool {
		return !l[i].Coordinates.Less(geo)
	})

	dh := l.haversine(geo, l[index].Coordinates)
	ld = append(ld, NewLocationDiff(l[index], dh))

	forward, back := index+1, index-1

	for i := 1; i < k; i++ {
		if back < 0 {
			fh := l.haversine(l[forward].Coordinates, geo)
			ld = append(ld, NewLocationDiff(l[forward], fh))
			forward++
			continue
		}

		if forward >= len(l) {
			bh := l.haversine(l[back].Coordinates, geo)
			ld = append(ld, NewLocationDiff(l[back], bh))
			back--
			continue
		}

		fh := l.haversine(l[forward].Coordinates, geo)
		bh := l.haversine(l[back].Coordinates, geo)

		fmt.Println("comparing:", l[back].Name, bh, l[forward].Name, fh)

		if bh < fh {
			ld = append(ld, NewLocationDiff(l[back], bh))
			back--
		} else {
			ld = append(ld, NewLocationDiff(l[forward], fh))
			forward++
		}
	}

	return ld
}

func (l LooInfos) haversine(geo1, geo2 *Geo) float64 {
	lat1, lng1 := geo1.Latitude, geo1.Longitude
	lat2, lng2 := geo2.Latitude, geo2.Longitude

	const er = float64(3961)

	var dlat = (lat2 - lat1) * (math.Pi / 180)
	var dlng = (lng2 - lng1) * (math.Pi / 180)

	var a = math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dlng/2)*math.Sin(dlng/2)
	var c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return er * c
}

type Geo struct {
	Latitude, Longitude float64
}

func (l *Geo) Less(j *Geo) bool {
	if l.Latitude == j.Latitude {
		return l.Longitude < j.Longitude
	}
	return l.Latitude < j.Latitude
}

type LooRaw struct {
	Raw json.RawMessage `json:"data"`
}

func (l *LooInfo) CreatedAt() time.Time {
	return time.Unix(l.CreatedAtUnix, 0)
}

func (l *LooInfo) UpdatedAt() time.Time {
	return time.Unix(l.UpdatedAtUnix, 0)
}

func (l *LooInfo) FillGeo() error {
	location, err := googleapi.LocationInfo(strings.Join([]string{
		l.Name,
		l.Location,
		l.Borough,
	}, ", "))
	if err != nil {
		return err
	}

	l.Coordinates = &Geo{location.Latitude, location.Longitude}
	return nil
}

func DecodeLoo(decoder *json.Decoder) (*LooInfo, error) {
	looi := new(LooInfo)
	loov := reflect.ValueOf(looi).Elem()

	for _, field := range validRestroomInfoIndex {
		if field == "" {
			var temp json.RawMessage
			if err := decoder.Decode(&temp); err != nil {
				return nil, err
			}
			continue
		}

		if f := loov.FieldByName(field); f.IsValid() && f.CanSet() {
			var temp interface{}

			switch f.Kind() {
			case reflect.Bool, reflect.String:
				var tempVar string
				temp = &tempVar
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				var tempVar int64
				temp = &tempVar
			}

			if err := decoder.Decode(temp); err != nil {
				return nil, err
			}

			switch f.Kind() {
			case reflect.Bool:
				var tempVar bool
				if strings.ToLower(reflect.Indirect(reflect.ValueOf(temp)).String()) == "yes" {
					tempVar = true
				}
				temp = &tempVar
			}

			f.Set(reflect.Indirect(reflect.ValueOf(temp)))
		}
	}

	if err := looi.FillGeo(); err != nil {
		return nil, err
	}
	return looi, nil
}

func DecodeLoos(rd io.Reader) ([]*LooInfo, error) {
	data := &bytes.Buffer{}
	loos := make([]*LooInfo, 0, approxTotalLooCount)

	var lr LooRaw
	if err := json.NewDecoder(rd).Decode(&lr); err != nil {
		return nil, err
	}

	data.Write(lr.Raw)
	decoder := json.NewDecoder(data)

	if _, err := decoder.Token(); err != nil {
		return nil, err
	}

	var i int
	for decoder.More() {
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}

		loo, err := DecodeLoo(decoder)
		if err == nil {
			loos = append(loos, loo)
		}
		log.Printf("Completed %d", i)

		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		i++
	}

	if _, err := decoder.Token(); err != nil {
		return nil, err
	}

	return loos, nil
}
