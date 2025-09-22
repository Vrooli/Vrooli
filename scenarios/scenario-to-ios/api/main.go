package main

import (
	"encoding/json\"
	\"log\"
	\"net/http\"
	\"time\"
)

type Server struct {
	router *http.ServeMux
}

func (s *Server) respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set(\"Content-Type\", \"application/json\")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		\"status\": \"healthy\",
		\"service\": \"scenario-to-ios\",
		\"timestamp\": time.Now().UTC().Format(time.RFC3339),
	}
	s.respondJSON(w, response)
}

func main() {
	s := &Server{router: http.NewServeMux()}
	s.router.HandleFunc(\"/api/v1/health\", s.handleHealth)
	log.Println(\"Starting server on :8080\")
	http.ListenAndServe(\":8080\", s.router)
}