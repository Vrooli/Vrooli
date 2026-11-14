package main

import "github.com/gorilla/mux"

func registerSearchRoutes(router *mux.Router) {
	router.HandleFunc("/api/search", searchHandler).Methods("POST")
}
