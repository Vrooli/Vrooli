package repository

import (
	"context"
	"time"
)

// App represents an application in the system
type App struct {
	ID           string                 `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	ScenarioName string                 `json:"scenario_name" db:"scenario_name"`
	Path         string                 `json:"path" db:"path"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	Status       string                 `json:"status" db:"status"`
	PortMappings map[string]interface{} `json:"port_mappings" db:"port_mappings"`
	Environment  map[string]interface{} `json:"environment" db:"environment"`
	Config       map[string]interface{} `json:"config" db:"config"`
	Description  string                 `json:"description,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	Runtime      string                 `json:"runtime,omitempty"`
	Uptime       string                 `json:"uptime,omitempty"`
	Type         string                 `json:"type,omitempty"`
	HealthStatus string                 `json:"health_status,omitempty"`
	IsPartial    bool                   `json:"is_partial,omitempty" db:"-"`
}

// AppStatus represents the current status of an application
type AppStatus struct {
	ID         string    `json:"id" db:"id"`
	AppID      string    `json:"app_id" db:"app_id"`
	Status     string    `json:"status" db:"status"`
	CPUUsage   float64   `json:"cpu_usage" db:"cpu_usage"`
	MemUsage   float64   `json:"memory_usage" db:"memory_usage"`
	DiskUsage  float64   `json:"disk_usage" db:"disk_usage"`
	NetworkIn  int64     `json:"network_in" db:"network_in"`
	NetworkOut int64     `json:"network_out" db:"network_out"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

// AppLog represents a log entry for an application
type AppLog struct {
	ID        string    `json:"id" db:"id"`
	AppID     string    `json:"app_id" db:"app_id"`
	Level     string    `json:"level" db:"level"`
	Message   string    `json:"message" db:"message"`
	Source    string    `json:"source" db:"source"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

// AppRepository defines the interface for app-related database operations
type AppRepository interface {
	// App CRUD operations
	GetApps(ctx context.Context) ([]App, error)
	GetApp(ctx context.Context, id string) (*App, error)
	CreateApp(ctx context.Context, app *App) error
	UpdateApp(ctx context.Context, app *App) error
	UpdateAppStatus(ctx context.Context, id string, status string) error
	DeleteApp(ctx context.Context, id string) error

	// App Status operations
	CreateAppStatus(ctx context.Context, status *AppStatus) error
	GetAppStatus(ctx context.Context, appID string) (*AppStatus, error)
	GetAppStatusHistory(ctx context.Context, appID string, hours int) ([]AppStatus, error)

	// App Logs operations
	CreateAppLog(ctx context.Context, log *AppLog) error
	GetAppLogs(ctx context.Context, appID string, limit, offset int) ([]AppLog, error)
	GetAppLogsByLevel(ctx context.Context, appID string, level string, limit int) ([]AppLog, error)
}

// MetricsRepository defines the interface for metrics-related database operations
type MetricsRepository interface {
	RecordMetric(ctx context.Context, appID string, metricType string, value float64, metadata map[string]interface{}) error
	GetMetrics(ctx context.Context, appID string, metricType string, since time.Time) ([]map[string]interface{}, error)
	GetAggregatedMetrics(ctx context.Context, appID string, interval string) (map[string]interface{}, error)
}

// HealthRepository defines the interface for health check operations
type HealthRepository interface {
	UpdateHealthStatus(ctx context.Context, appID string, status string, responseTime int) error
	GetHealthStatus(ctx context.Context, appID string) (map[string]interface{}, error)
	RecordHealthAlert(ctx context.Context, appID string, alertType string, severity string, message string) error
	GetUnresolvedAlerts(ctx context.Context) ([]map[string]interface{}, error)
}
