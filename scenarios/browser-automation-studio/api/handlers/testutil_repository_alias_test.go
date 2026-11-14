package handlers

import "github.com/vrooli/browser-automation-studio/internal/testutil/databasemocks"

type MockRepository = databasemocks.MockRepository

func NewMockRepository() *MockRepository {
	return databasemocks.NewMockRepository()
}
