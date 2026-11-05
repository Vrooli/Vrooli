package server

import (
	"net/http"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/middleware"
)

func buildMiddleware(cfg *config.Config, handler http.Handler, logger logx.Logger) http.Handler {
	handler = middleware.MaxBodySize(1<<20, logger)(handler) // 1MB body limit
	handler = middleware.CORS(handler)

	handler = middleware.Logging(logger)(handler)

	handler = middleware.Recovery(logger)(handler)

	handler = middleware.RequestID(handler)

	return handler
}
