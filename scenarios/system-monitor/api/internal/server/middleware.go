package server

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/middleware"
)

func buildMiddleware(cfg *config.Config, router *mux.Router) http.Handler {
	handler := http.Handler(router)

	handler = middleware.CORS(handler)

	logger := log.New(os.Stdout, "[HTTP] ", log.LstdFlags)
	handler = middleware.Logging(logger)(handler)

	if cfg.Alerts.EnableWebhooks {
		// API key auth could be enforced here.
	}

	return handler
}
