package main

import "net/http"

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/events", s.handleIngest)
	mux.HandleFunc("GET /api/v1/events", s.handleQuery)
	mux.HandleFunc("GET /api/v1/events/subscribe", s.handleSubscribe)
	mux.HandleFunc("GET /health", s.handleHealth)
	return mux
}
