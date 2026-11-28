// Package hubtest builds a hub.Service for a test and stops it when the test
// ends. Every test that needs a Service uses New, so none leaves a background
// worker running against a database the test has already closed.
package hubtest

import (
	"log"
	"testing"

	"github.com/vrooli/api-core/database"

	"notification-hub/internal/hub"
)

// New returns a Service over db and registers its Close with t.Cleanup.
// Cleanups run last-in first-out, so a database the test registered before
// calling New is still open when the Service closes.
func New(t testing.TB, db *database.RoutedDB, clock hub.Clock, logger *log.Logger) *hub.Service {
	t.Helper()
	service := hub.New(db, clock, logger)
	t.Cleanup(service.Close)
	return service
}
