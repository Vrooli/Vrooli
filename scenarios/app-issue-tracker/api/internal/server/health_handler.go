package server

import (
	"net/http"

	"github.com/vrooli/api-core/health"
)

// createHealthHandler returns a standardized health check handler using api-core/health.
// App Issue Tracker uses file-based storage, so no database check is needed.
func (s *Server) createHealthHandler() http.HandlerFunc {
	return health.New().
		Version("2.0.0").
		Handler()
}
