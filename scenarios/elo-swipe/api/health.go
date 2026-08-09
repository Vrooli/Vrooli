package main

import (
	"net/http"

	"github.com/vrooli/api-core/health"
)

// HealthCheck is the dependency-free test seam for the health endpoint. The
// production route adds the database check; unit tests exercise the response
// contract without requiring PostgreSQL.
func (a *App) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health.New("elo-swipe-api").Version("1.0.0").Handler()(w, r)
}
