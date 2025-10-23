package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"app-issue-tracker-api/internal/automation"
	"app-issue-tracker-api/internal/server/services"
)

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

func WithCommandFactory(factory commandFactory) Option {
	return func(s *Server) {
		s.commandFactory = factory
	}
}

type Server struct {
	config         *Config
	store          IssueStorage
	processor      *automation.Processor
	hub            *Hub
	issues         *services.IssueService
	investigations *InvestigationService
	content        *IssueContentService
	commandFactory commandFactory
	startOnce      sync.Once
	stopOnce       sync.Once
	stopMu         sync.Mutex
	stopCtx        context.CancelFunc
	started        bool
	wsUpgrader     websocket.Upgrader
}

func (s *Server) Config() *Config {
	return s.config
}

func (s *Server) storeIssueArtifacts(issue *Issue, issueDir string, payloads []ArtifactPayload, replaceExisting bool) error {
	return s.issues.StoreArtifacts(issue, issueDir, payloads, replaceExisting)
}

func (s *Server) Start() {
	s.startOnce.Do(func() {
		if s.hub != nil {
			go s.hub.Run()
		}

		ctx, cancel := context.WithCancel(context.Background())
		s.stopMu.Lock()
		s.stopCtx = cancel
		s.started = true
		s.stopMu.Unlock()

		s.processor.Start(ctx)
	})
}

func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		s.stopMu.Lock()
		cancel := s.stopCtx
		s.stopCtx = nil
		started := s.started
		s.stopMu.Unlock()

		if cancel != nil {
			cancel()
		}

		if started && s.processor != nil {
			s.processor.Stop()
		}
		if started && s.hub != nil {
			s.hub.Shutdown()
		}
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const allowedMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD"
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", allowedMethods)

		if requestHeaders := r.Header.Get("Access-Control-Request-Headers"); requestHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", requestHeaders)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Token, Accept")
		}

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

func (s *Server) ensureCommandFactory() {
	if s.commandFactory == nil {
		s.commandFactory = defaultCommandFactory
	}
}

func (s *Server) ensureStore(defaultIssuesDir string) bool {
	if s.store == nil {
		s.store = NewFileIssueStore(defaultIssuesDir)
		return true
	}
	return false
}
