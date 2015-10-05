package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/neurodrone/restroomfinder"
)

func main() {
	var (
		port        = flag.String("port", "", "http server port")
		pollTimeout = flag.Duration("polltimeout", 30*time.Second, "read timeout for polling opendata url")
		dbName      = flag.String("dbname", "geo_restroom_finder.db", "sqlite database name")
	)
	flag.Parse()

	h, err := restroomfinder.NewHandler(*pollTimeout, *dbName)
	if err != nil {
		log.Fatalln("cannot open handler:", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/poll", h.PollLoos).Methods("GET")
	r.HandleFunc("/closest/{lat}/{lng}/{count}/", h.FindClosest).Methods("GET")

	log.Printf("Starting server on :%s", *port)
	if err := http.ListenAndServe(":"+*port, r); err != nil {
		log.Fatalln("cannot start server:", err)
	}
}
