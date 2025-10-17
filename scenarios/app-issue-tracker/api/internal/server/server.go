package server

import "net/http"

type Option func(*Server)

func WithIssueStore(store IssueStorage) Option {
	return func(s *Server) {
		s.store = store
	}
}

func WithHub(h *Hub) Option {
	return func(s *Server) {
		s.hub = h
	}
}

type Server struct {
	config    *Config
	store     IssueStorage
	processor *AutomationProcessor
	hub       *Hub
}

func (s *Server) Config() *Config {
	return s.config
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) ensureHub() bool {
	if s.hub == nil {
		s.hub = NewHub()
		return true
	}
	return false
}

func (s *Server) ensureStore(defaultIssuesDir string) bool {
	if s.store == nil {
		s.store = NewFileIssueStore(defaultIssuesDir)
		return true
	}
	return false
}
