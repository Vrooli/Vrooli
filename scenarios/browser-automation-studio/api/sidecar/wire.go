// Package sidecar provides playwright-driver process supervision, health monitoring,
// and session recovery for the browser-automation-studio API.
//
// The sidecar system manages the playwright-driver as a child process, automatically
// restarting it on crashes and providing real-time health status to the UI.
package sidecar

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/sidecar/health"
	"github.com/vrooli/browser-automation-studio/sidecar/recovery"
	"github.com/vrooli/browser-automation-studio/sidecar/supervisor"
	"github.com/vrooli/browser-automation-studio/websocket"
)

// Dependencies holds all sidecar-related dependencies.
// These are nil when sidecar management is disabled (external driver configured).
type Dependencies struct {
	// Supervisor manages the playwright-driver process lifecycle.
	// Nil when using an external driver (PLAYWRIGHT_DRIVER_URL set).
	Supervisor supervisor.Supervisor

	// HealthMonitor tracks driver health and broadcasts status updates.
	// Nil when supervisor is disabled.
	HealthMonitor health.Monitor

	// CheckpointManager handles recording session checkpointing for crash recovery.
	// Nil when supervisor is disabled.
	CheckpointManager recovery.Manager

	// Store is the checkpoint persistence layer.
	// Nil when supervisor is disabled.
	Store recovery.Store
}

// BuildDependencies wires up all sidecar-related dependencies.
//
// If PLAYWRIGHT_DRIVER_URL is set, sidecar management is disabled and all
// fields in Dependencies will be nil. This allows the API to work with
// an externally-managed playwright-driver instance.
//
// Parameters:
//   - db: database connection for checkpoint storage
//   - driverClient: existing driver client (used for health checking)
//   - hub: WebSocket hub for broadcasting driver status
//   - log: logger for sidecar events
func BuildDependencies(
	db *sqlx.DB,
	driverClient *driver.Client,
	hub *websocket.Hub,
	log *logrus.Logger,
) (*Dependencies, error) {
	// Load configurations - this handles the BAS_SIDECAR_ENABLED vs PLAYWRIGHT_DRIVER_URL logic
	supervisorCfg := supervisor.LoadConfig()
	healthCfg := health.LoadConfig()
	recoveryCfg := recovery.LoadConfig()

	// Check if supervision is disabled (either explicitly or due to external driver)
	if !supervisorCfg.Enabled {
		if externalURL := strings.TrimSpace(os.Getenv(driver.PlaywrightDriverEnv)); externalURL != "" {
			log.WithField("url", externalURL).Info("Sidecar supervision disabled (external driver or explicit disable)")
		} else {
			log.Info("Sidecar supervision disabled via configuration")
		}
		return &Dependencies{}, nil
	}

	// 1. Build checkpoint store (SQLite)
	checkpointStore := recovery.NewSQLiteStore(db, log)

	// Run migrations for checkpoint table
	if err := checkpointStore.Migrate(context.Background()); err != nil {
		log.WithError(err).Warn("Failed to migrate checkpoint table, recovery may not work")
	}

	// 2. Build health checker (wraps driver client)
	healthChecker := &driverHealthChecker{
		client: driverClient,
		log:    log,
	}

	// 3. Build process (mirrors service.json: cd playwright-driver && node dist/server.js)
	process := supervisor.NewNodeProcess(
		supervisorCfg.NodePath,
		supervisorCfg.DriverDir,
		supervisorCfg.DriverScript,
		supervisorCfg.DriverPort,
		log,
	)

	// 4. Build supervisor
	sup := supervisor.NewProcessSupervisor(
		supervisorCfg,
		process,
		func(ctx context.Context) error {
			return driverClient.Health(ctx)
		},
		log,
	)

	// 5. Build health monitor
	monitor := health.NewHealthMonitor(
		healthCfg,
		healthChecker,
		sup,
		driverClient.CircuitBreakerState,
		log,
	)

	// 6. Build checkpoint manager
	actionSource := func(ctx context.Context, sessionID string) ([]recovery.RecordedAction, string, error) {
		resp, err := driverClient.GetRecordedActions(ctx, sessionID, false)
		if err != nil {
			return nil, "", err
		}

		// Convert driver.RecordedAction to recovery.RecordedAction
		actions := make([]recovery.RecordedAction, len(resp.Actions))
		var currentURL string
		for i, a := range resp.Actions {
			// Extract selector string from SelectorSet
			var selector string
			if a.Selector != nil {
				selector = a.Selector.Primary
			}

			// Extract value from payload if present
			var value string
			if payload, ok := a.Payload["value"]; ok {
				if s, ok := payload.(string); ok {
					value = s
				}
			}

			// Parse timestamp
			timestamp := time.Now()
			if a.Timestamp != "" {
				if parsed, err := time.Parse(time.RFC3339, a.Timestamp); err == nil {
					timestamp = parsed
				}
			}

			actions[i] = recovery.RecordedAction{
				Type:      a.ActionType,
				Selector:  selector,
				Value:     value,
				URL:       a.URL,
				Timestamp: timestamp,
			}
			if a.URL != "" {
				currentURL = a.URL
			}
		}
		return actions, currentURL, nil
	}

	checkpointMgr := recovery.NewCheckpointManager(
		recoveryCfg,
		checkpointStore,
		actionSource,
		log,
	)

	// 7. Wire monitor → existing hub
	go func() {
		for h := range monitor.Subscribe() {
			hub.BroadcastDriverStatus(h)
		}
	}()

	log.WithFields(logrus.Fields{
		"driver_dir":          supervisorCfg.DriverDir,
		"driver_port":         supervisorCfg.DriverPort,
		"max_restarts":        supervisorCfg.MaxRestarts,
		"checkpoint_enabled":  recoveryCfg.Enabled,
		"checkpoint_interval": recoveryCfg.CheckpointInterval,
	}).Info("Sidecar dependencies initialized")

	return &Dependencies{
		Supervisor:        sup,
		HealthMonitor:     monitor,
		CheckpointManager: checkpointMgr,
		Store:             checkpointStore,
	}, nil
}

// Start starts all sidecar services.
// Safe to call when sidecar management is disabled (returns nil).
func (d *Dependencies) Start(ctx context.Context) error {
	if d.Supervisor == nil {
		return nil
	}

	// Start supervisor (spawns playwright-driver)
	if err := d.Supervisor.Start(ctx); err != nil {
		return err
	}

	// Start health monitor
	if err := d.HealthMonitor.Start(ctx); err != nil {
		return err
	}

	// Start checkpoint manager
	if err := d.CheckpointManager.Start(ctx); err != nil {
		return err
	}

	return nil
}

// Stop stops all sidecar services gracefully.
// Safe to call when sidecar management is disabled (returns nil).
func (d *Dependencies) Stop(ctx context.Context) error {
	if d.Supervisor == nil {
		return nil
	}

	// Stop checkpoint manager first (stops background workers)
	if d.CheckpointManager != nil {
		if err := d.CheckpointManager.Stop(); err != nil {
			// Log but continue shutdown
		}
	}

	// Stop health monitor
	if d.HealthMonitor != nil {
		if err := d.HealthMonitor.Stop(); err != nil {
			// Log but continue shutdown
		}
	}

	// Stop supervisor (kills playwright-driver)
	if d.Supervisor != nil {
		if err := d.Supervisor.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

// IsEnabled returns true if sidecar management is active.
func (d *Dependencies) IsEnabled() bool {
	return d.Supervisor != nil
}

// driverHealthChecker implements health.Checker by wrapping the driver client.
type driverHealthChecker struct {
	client *driver.Client
	log    *logrus.Logger
}

// driverHealthEndpointResponse represents the response from playwright-driver's /health endpoint.
type driverHealthEndpointResponse struct {
	Status           string `json:"status"`
	Ready            bool   `json:"ready"`
	Sessions         int    `json:"sessions"`
	ActiveRecordings int    `json:"active_recordings"`
	Version          string `json:"version"`
	UptimeMS         int64  `json:"uptime_ms"`
	Browser          struct {
		Healthy bool    `json:"healthy"`
		Version string  `json:"version"`
		Error   *string `json:"error,omitempty"`
	} `json:"browser"`
}

// Check implements health.Checker.
func (c *driverHealthChecker) Check(ctx context.Context) (*health.DriverHealthResponse, error) {
	// Make request to driver's /health endpoint
	resp, err := c.fetchDriverHealth(ctx)
	if err != nil {
		return nil, err
	}

	// Map driver status to health status
	var status string
	switch resp.Status {
	case "ok":
		status = "healthy"
	case "degraded":
		status = "degraded"
	default:
		status = "unhealthy"
	}

	// Map browser status
	var browserStatus string
	if resp.Browser.Healthy {
		browserStatus = "running"
	} else if resp.Browser.Error != nil && *resp.Browser.Error == "Browser not yet verified" {
		browserStatus = "starting"
	} else {
		browserStatus = "crashed"
	}

	return &health.DriverHealthResponse{
		Status:         status,
		ActiveSessions: resp.Sessions,
		BrowserStatus:  browserStatus,
		Uptime:         resp.UptimeMS,
		LastError:      resp.Browser.Error,
	}, nil
}

// fetchDriverHealth makes a request to the driver's /health endpoint.
func (c *driverHealthChecker) fetchDriverHealth(ctx context.Context) (*driverHealthEndpointResponse, error) {
	// First use the simple health check to verify connectivity
	if err := c.client.Health(ctx); err != nil {
		return nil, err
	}

	// For now, return a basic response since we verified connectivity
	// In the future, we can parse the full /health response body
	return &driverHealthEndpointResponse{
		Status:   "ok",
		Ready:    true,
		Sessions: 0,
		Browser: struct {
			Healthy bool    `json:"healthy"`
			Version string  `json:"version"`
			Error   *string `json:"error,omitempty"`
		}{
			Healthy: true,
		},
		UptimeMS: 0,
	}, nil
}

// compile-time check
var _ health.Checker = (*driverHealthChecker)(nil)
