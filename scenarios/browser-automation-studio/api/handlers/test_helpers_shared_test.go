package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

// createTestHandler returns a Handler with mock dependencies wired for unit
// tests in this package. It used to live in the (now-deleted) executions
// REST test file; it is shared across artifacts, exports, schedules,
// uxmetrics, and other REST handler tests in this package.
func createTestHandler() (*Handler, *MockCatalogService, *MockExecutionService, *MockRepository, *MockHub, *MockStorage) {
	repo := NewMockRepository()
	hub := NewMockHub()
	catalogSvc := NewMockCatalogService()
	execSvc := NewMockExecutionService()
	storageMock := NewMockStorage()
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	handler := &Handler{
		catalogService:   catalogSvc,
		executionService: execSvc,
		repo:             repo,
		wsHub:            hub,
		storage:          storageMock,
		log:              log,
	}

	return handler, catalogSvc, execSvc, repo, hub, storageMock
}

// withURLParam attaches a chi URL parameter to the request context. Used
// when invoking REST handlers directly in unit tests without going through
// the chi router.
func withURLParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}
