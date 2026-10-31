package services

import (
	"context"

	"app-monitor-api/repository"
)

type mockAppRepository struct {
	apps []repository.App
}

func (m *mockAppRepository) GetApps(ctx context.Context) ([]repository.App, error) {
	return m.apps, nil
}

func (m *mockAppRepository) GetApp(ctx context.Context, id string) (*repository.App, error) {
	for _, app := range m.apps {
		if app.ID == id || app.ScenarioName == id {
			return &app, nil
		}
	}
	return nil, ErrAppNotFound
}

func (m *mockAppRepository) CreateApp(ctx context.Context, app *repository.App) error { return nil }
func (m *mockAppRepository) UpdateApp(ctx context.Context, app *repository.App) error { return nil }
func (m *mockAppRepository) UpdateAppStatus(ctx context.Context, id string, status string) error {
	return nil
}
func (m *mockAppRepository) DeleteApp(ctx context.Context, id string) error { return nil }
func (m *mockAppRepository) CreateAppStatus(ctx context.Context, status *repository.AppStatus) error {
	return nil
}

func (m *mockAppRepository) GetAppStatus(ctx context.Context, appID string) (*repository.AppStatus, error) {
	return nil, nil
}

func (m *mockAppRepository) GetAppStatusHistory(ctx context.Context, appID string, hours int) ([]repository.AppStatus, error) {
	return nil, nil
}

func (m *mockAppRepository) CreateAppLog(ctx context.Context, log *repository.AppLog) error {
	return nil
}

func (m *mockAppRepository) GetAppLogs(ctx context.Context, appID string, limit, offset int) ([]repository.AppLog, error) {
	return nil, nil
}

func (m *mockAppRepository) GetAppLogsByLevel(ctx context.Context, appID string, level string, limit int) ([]repository.AppLog, error) {
	return nil, nil
}

func (m *mockAppRepository) RecordAppView(ctx context.Context, scenarioName string) (*repository.AppViewStats, error) {
	return nil, nil
}

func (m *mockAppRepository) GetAppViewStats(ctx context.Context) (map[string]repository.AppViewStats, error) {
	return map[string]repository.AppViewStats{}, nil
}
