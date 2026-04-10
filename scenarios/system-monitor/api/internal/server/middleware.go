package server

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/middleware"
)

func buildMiddleware(cfg *config.Config, router *mux.Router) http.Handler {
	handler := http.Handler(router)

	handler = middleware.MaxBodySize(1<<20, slog.Default())(handler) // 1MB body limit
	handler = middleware.CORS(handler)

	handler = middleware.Logging(slog.Default())(handler)

	handler = middleware.Recovery(slog.Default())(handler)

	handler = middleware.RequestID(handler)

	return handler
}
