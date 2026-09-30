package handlers

import (
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/internal/testutil/hubmocks"
)

type MockHub = hubmocks.Hub

func NewMockHub() *MockHub { return hubmocks.New() }

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

func NewTestHandler() (*Handler, *MockCatalogService, *MockExecutionService, *MockRepository, *MockHub, *MockStorage) {
	return createTestHandler()
}
