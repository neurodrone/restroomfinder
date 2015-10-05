package restroomfinder

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"time"
	"unsafe"

	"github.com/gorilla/mux"
	"github.com/neurodrone/restroomfinder/sqlite"
)

const (
	restroomDataURL = "https://data.cityofnewyork.us/api/views/hjae-yuav/rows.json"

	float64Bits = int(unsafe.Sizeof(struct{ float64 }{}) * 8)
)

type Handler struct {
	pollTimeout    time.Duration
	db             *sqlite.DB
	recentlyPulled bool

	loos LooInfos
}

func NewHandler(pollTimeout time.Duration, dbName string) (*Handler, error) {
	db, err := sqlite.NewDB(dbName)
	if err != nil {
		return nil, err
	}

	return &Handler{
		pollTimeout: pollTimeout,
		db:          db,
	}, nil
}

func (h *Handler) PollLoos(w http.ResponseWriter, r *http.Request) {
	if err := h.clearAndSaveLoos(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "success")
}

func (h *Handler) FindClosest(w http.ResponseWriter, r *http.Request) {
	if err := h.loadLoos(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	lat, _ := strconv.ParseFloat(mux.Vars(r)["lat"], float64Bits)
	lng, _ := strconv.ParseFloat(mux.Vars(r)["lng"], float64Bits)
	count, _ := strconv.Atoi(mux.Vars(r)["count"])

	for _, loc := range h.loos.FindClosest(&Geo{
		Latitude:  lat,
		Longitude: lng,
	}, count) {
		fmt.Fprintf(w, h.formatLocation(loc))
	}
}

func (h *Handler) formatLocation(loc LocationDiff) string {
	return fmt.Sprintf("%s,%f,%f,%.2f miles;",
		loc.Name,
		loc.Coordinates.Latitude,
		loc.Coordinates.Longitude,
		loc.Distance)
}

func (h *Handler) loadLoos() error {
	if h.recentlyPulled {
		return nil
	}

	count := h.db.CountGeo()
	if count <= 0 {
		return errors.New("no geo data found")
	}

	loos := make(LooInfos, 0, count)

	rows, err := h.db.QueryAllStmt.Query()
	if err != nil {
		return err
	}

	for rows.Next() {
		var info LooInfo
		var geo Geo
		if err := rows.Scan(
			&info.UUID,
			&info.CreatedAtUnix,
			&info.UpdatedAtUnix,
			&info.Name,
			&info.Location,
			&info.Borough,
			&info.HandicapOk,
			&info.OpenAllYear,
			&geo.Latitude,
			&geo.Longitude,
		); err != nil {
			return err
		}

		info.Coordinates = &geo

		loos = append(loos, &info)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	sort.Sort(loos)

	h.loos = loos
	h.recentlyPulled = true
	return nil
}

func (h *Handler) clearAndSaveLoos() error {
	client := &http.Client{
		Timeout:   h.pollTimeout,
		Transport: http.DefaultTransport,
	}

	resp, err := client.Get(restroomDataURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	loos, err := DecodeLoos(resp.Body)
	if err != nil {
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		return err
	}

	if err := h.db.ClearGeo(); err != nil {
		return err
	}

	for _, loo := range loos {
		if _, err := h.db.InsertStmt.Exec(
			loo.UUID,
			loo.CreatedAtUnix,
			loo.UpdatedAtUnix,
			loo.Name,
			loo.Location,
			loo.Borough,
			loo.HandicapOk,
			loo.OpenAllYear,
			loo.Coordinates.Latitude,
			loo.Coordinates.Longitude,
		); err != nil {
			return err
		}
	}
	return nil
}
