package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"system-monitor-api/internal/collectors"
	"system-monitor-api/internal/handlers"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/services"
)

type HealthResponse struct {
	Status           string                 `json:"status"`
	Service          string                 `json:"service"`
	Timestamp        int64                  `json:"timestamp"`
	Uptime           float64                `json:"uptime"`
	Checks           map[string]interface{} `json:"checks"`
	Version          string                 `json:"version"`
	ProcessorActive  bool                   `json:"processor_active"`
	MaintenanceState string                 `json:"maintenance_state"`
}

type MetricsResponse struct {
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	TCPConnections int     `json:"tcp_connections"`
	Timestamp      string  `json:"timestamp"`
}

// metricHistorySample tracks raw samples for rolling history retention
type metricHistorySample struct {
	timestamp      time.Time
	cpuUsage       float64
	memoryUsage    float64
	tcpConnections int
}

// MetricTimelineSample is returned to the UI for sparkline rendering
type MetricTimelineSample struct {
	Timestamp      string  `json:"timestamp"`
	CPUUsage       float64 `json:"cpu_usage"`
	MemoryUsage    float64 `json:"memory_usage"`
	TCPConnections int     `json:"tcp_connections"`
}

// MetricsTimelineResponse bundles recent samples for the UI graphs
type MetricsTimelineResponse struct {
	WindowSeconds         int                    `json:"window_seconds"`
	SampleIntervalSeconds int                    `json:"sample_interval_seconds"`
	Samples               []MetricTimelineSample `json:"samples"`
}

// Enhanced metric structures for expandable cards
type ProcessInfo struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryMB    float64 `json:"memory_mb"`
	Connections int     `json:"connections"`
	Threads     int     `json:"threads"`
	FDs         int     `json:"file_descriptors"`
	Status      string  `json:"status"`
	Goroutines  int     `json:"goroutines,omitempty"`
}

type TriggerConfig struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Icon        string  `json:"icon"`
	Enabled     bool    `json:"enabled"`
	AutoFix     bool    `json:"auto_fix"`
	Threshold   float64 `json:"threshold"`
	Unit        string  `json:"unit"`
	Condition   string  `json:"condition"`
}

type investigationScriptManifestEntry struct {
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	File                  string   `json:"file"`
	Category              string   `json:"category"`
	Author                string   `json:"author"`
	Created               string   `json:"created"`
	LastModified          string   `json:"last_modified"`
	SafetyLevel           string   `json:"safety_level"`
	Enabled               *bool    `json:"enabled,omitempty"`
	Tags                  []string `json:"tags,omitempty"`
	ExecutionTimeEstimate string   `json:"execution_time_estimate,omitempty"`
}

type investigationScriptListItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Author      string   `json:"author"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Enabled     bool     `json:"enabled"`
	Tags        []string `json:"tags,omitempty"`
}

type TCPConnectionStates struct {
	Established int `json:"established"`
	TimeWait    int `json:"time_wait"`
	CloseWait   int `json:"close_wait"`
	FinWait1    int `json:"fin_wait1"`
	FinWait2    int `json:"fin_wait2"`
	SynSent     int `json:"syn_sent"`
	SynRecv     int `json:"syn_recv"`
	Closing     int `json:"closing"`
	LastAck     int `json:"last_ack"`
	Listen      int `json:"listen"`
	Total       int `json:"total"`
}

type ConnectionPoolInfo struct {
	Name     string `json:"name"`
	Active   int    `json:"active"`
	Idle     int    `json:"idle"`
	MaxSize  int    `json:"max_size"`
	Waiting  int    `json:"waiting"`
	Healthy  bool   `json:"healthy"`
	LeakRisk string `json:"leak_risk"`
}

type NetworkStats struct {
	BandwidthInMbps  float64 `json:"bandwidth_in_mbps"`
	BandwidthOutMbps float64 `json:"bandwidth_out_mbps"`
	PacketLoss       float64 `json:"packet_loss"`
	DNSSuccessRate   float64 `json:"dns_success_rate"`
	DNSLatencyMs     float64 `json:"dns_latency_ms"`
}

type SystemHealthDetails struct {
	FileDescriptors struct {
		Used    int     `json:"used"`
		Max     int     `json:"max"`
		Percent float64 `json:"percent"`
	} `json:"file_descriptors"`
	ServiceDependencies []ServiceHealth   `json:"service_dependencies"`
	Certificates        []CertificateInfo `json:"certificates"`
}

type ServiceHealth struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	LatencyMs float64 `json:"latency_ms"`
	LastCheck string  `json:"last_check"`
	Endpoint  string  `json:"endpoint"`
}

type CertificateInfo struct {
	Domain       string `json:"domain"`
	DaysToExpiry int    `json:"days_to_expiry"`
	Status       string `json:"status"`
}

// Detailed metric responses for expandable cards
type DetailedMetricsResponse struct {
	CPUDetails struct {
		Usage           float64       `json:"usage"`
		TopProcesses    []ProcessInfo `json:"top_processes"`
		LoadAverage     []float64     `json:"load_average"`
		ContextSwitches int64         `json:"context_switches"`
		Goroutines      int           `json:"total_goroutines"`
	} `json:"cpu_details"`
	MemoryDetails struct {
		Usage          float64       `json:"usage"`
		TopProcesses   []ProcessInfo `json:"top_processes"`
		GrowthPatterns []struct {
			Process         string  `json:"process"`
			GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
			RiskLevel       string  `json:"risk_level"`
		} `json:"growth_patterns"`
		SwapUsage struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		} `json:"swap_usage"`
		DiskUsage struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		} `json:"disk_usage"`
	} `json:"memory_details"`
	NetworkDetails struct {
		TCPStates TCPConnectionStates `json:"tcp_states"`
		PortUsage struct {
			Used  int `json:"used"`
			Total int `json:"total"`
		} `json:"port_usage"`
		NetworkStats    NetworkStats         `json:"network_stats"`
		ConnectionPools []ConnectionPoolInfo `json:"connection_pools"`
	} `json:"network_details"`
	SystemDetails SystemHealthDetails `json:"system_details"`
	Timestamp     string              `json:"timestamp"`
}

type ProcessMonitorResponse struct {
	ProcessHealth struct {
		TotalProcesses  int           `json:"total_processes"`
		ZombieProcesses []ProcessInfo `json:"zombie_processes"`
		HighThreadCount []ProcessInfo `json:"high_thread_count"`
		LeakCandidates  []ProcessInfo `json:"leak_candidates"`
	} `json:"process_health"`
	ResourceMatrix []ProcessInfo `json:"resource_matrix"`
	Timestamp      string        `json:"timestamp"`
}

type InfrastructureMonitorResponse struct {
	DatabasePools   []ConnectionPoolInfo `json:"database_pools"`
	HTTPClientPools []ConnectionPoolInfo `json:"http_client_pools"`
	MessageQueues   struct {
		RedisPubSub struct {
			Subscribers int `json:"subscribers"`
			Channels    int `json:"channels"`
		} `json:"redis_pubsub"`
		BackgroundJobs struct {
			Pending int `json:"pending"`
			Active  int `json:"active"`
			Failed  int `json:"failed"`
		} `json:"background_jobs"`
	} `json:"message_queues"`
	StorageIO struct {
		DiskQueueDepth float64 `json:"disk_queue_depth"`
		IOWaitPercent  float64 `json:"io_wait_percent"`
		ReadMBPerSec   float64 `json:"read_mb_per_sec"`
		WriteMBPerSec  float64 `json:"write_mb_per_sec"`
	} `json:"storage_io"`
	Timestamp string `json:"timestamp"`
}

type Investigation struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"` // queued, in_progress, completed, failed
	AnomalyID string                 `json:"anomaly_id"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time,omitempty"`
	Findings  string                 `json:"findings,omitempty"`
	Progress  int                    `json:"progress"`          // 0-100
	Details   map[string]interface{} `json:"details,omitempty"` // Structured data
	Steps     []InvestigationStep    `json:"steps,omitempty"`
}

type InvestigationStep struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Findings  string    `json:"findings,omitempty"`
}

// ErrorLog represents an error entry for the system output
type ErrorLog struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // "error", "warning", "info"
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	Details   string    `json:"details,omitempty"`
	Unread    bool      `json:"unread"`
}

// GeneratedReport represents a generated report for listing
type GeneratedReport struct {
	ReportID    string    `json:"report_id"`
	ReportType  string    `json:"report_type"`
	GeneratedAt time.Time `json:"generated_at"`
	DateRange   string    `json:"date_range"`
	Status      string    `json:"status"`
	// Full report content
	Content *FullReportContent `json:"content,omitempty"`
}

// FullReportContent represents the detailed report data
type FullReportContent struct {
	ExecutiveSummary    ReportExecutiveSummary    `json:"executive_summary"`
	PerformanceAnalysis ReportPerformanceAnalysis `json:"performance"`
	Trends              []ReportTrend             `json:"trends"`
	Recommendations     []string                  `json:"recommendations"`
	Highlights          []string                  `json:"highlights"`
	MetricsCount        int                       `json:"metrics_count"`
	AlertsCount         int                       `json:"alerts_count"`
	ActualDuration      string                    `json:"actual_duration"`
	DateRangeDisplay    string                    `json:"date_range_display"`
}

type ReportExecutiveSummary struct {
	OverallHealth   string   `json:"overall_health"`
	KeyFindings     []string `json:"key_findings"`
	TimeDescription string   `json:"time_description"`
	MetricsAnalyzed int      `json:"metrics_analyzed"`
}

type ReportPerformanceAnalysis struct {
	CPU       ReportMetricStats `json:"cpu"`
	Memory    ReportMetricStats `json:"memory"`
	TimeRange string            `json:"time_range"`
}

type ReportMetricStats struct {
	Average   float64   `json:"average"`
	Min       float64   `json:"min"`
	Max       float64   `json:"max"`
	StdDev    float64   `json:"std_dev"`
	PeakValue float64   `json:"peak_value"`
	PeakTime  time.Time `json:"peak_time"`
	MinTime   time.Time `json:"min_time"`
}

type ReportTrend struct {
	Name          string  `json:"name"`
	Direction     string  `json:"direction"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
}

// Global variables to store investigations and app state
var (
	latestInvestigation *Investigation
	investigations      = make(map[string]*Investigation)
	investigationsMutex sync.RWMutex

	// Cooldown configuration
	cooldownPeriodSeconds = 300 // Default 5 minutes
	lastTriggerTime       time.Time
	cooldownMutex         sync.RWMutex

	// Trigger configurations
	triggers = map[string]*TriggerConfig{
		"high_cpu": {
			ID:          "high_cpu",
			Name:        "High CPU Usage",
			Description: "Triggers when CPU usage exceeds threshold",
			Icon:        "cpu",
			Enabled:     true,
			AutoFix:     false,
			Threshold:   85,
			Unit:        "%",
			Condition:   "above",
		},
		"memory_pressure": {
			ID:          "memory_pressure",
			Name:        "Memory Pressure",
			Description: "Triggers when available memory falls below threshold",
			Icon:        "database",
			Enabled:     true,
			AutoFix:     true,
			Threshold:   10,
			Unit:        "%",
			Condition:   "below",
		},
		"disk_space": {
			ID:          "disk_space",
			Name:        "Low Disk Space",
			Description: "Triggers when disk usage exceeds threshold",
			Icon:        "hard-drive",
			Enabled:     true,
			AutoFix:     true,
			Threshold:   90,
			Unit:        "%",
			Condition:   "above",
		},
		"network_connections": {
			ID:          "network_connections",
			Name:        "Excessive Network Connections",
			Description: "Triggers when active connections exceed normal levels",
			Icon:        "network",
			Enabled:     false,
			AutoFix:     false,
			Threshold:   1000,
			Unit:        " connections",
			Condition:   "above",
		},
		"process_anomaly": {
			ID:          "process_anomaly",
			Name:        "Process Anomaly",
			Description: "Triggers when zombie or high-resource processes detected",
			Icon:        "zap",
			Enabled:     true,
			AutoFix:     false,
			Threshold:   5,
			Unit:        " processes",
			Condition:   "above",
		},
	}

	// Sliding window history for sparkline graphs
	metricsHistory          = make([]metricHistorySample, 0, 120)
	metricsHistoryMutex     sync.RWMutex
	metricsHistoryRetention = 5 * time.Minute
	metricsSampleInterval   = 5 * time.Second

	triggersMutex       sync.RWMutex
	startTime           = time.Now()
	monitoringProcessor *MonitoringProcessor

	// Settings management
	settingsManager   *services.SettingsManager
	settingsHandler   *handlers.SettingsHandler
	monitoringService *services.MonitoringService

	// Error tracking for system output
	errorLogs         []ErrorLog
	errorLogsMutex    sync.RWMutex
	unreadErrorCount  int
	lastErrorReadTime time.Time

	// Generated reports storage
	generatedReports []GeneratedReport
	reportsMutex     sync.RWMutex
)

type ReportRequest struct {
	Type string `json:"type"`
}

// loadTriggerConfig loads trigger configuration from JSON file
func loadTriggerConfig() error {
	configPath := filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios/system-monitor/initialization/configuration/investigation-triggers.json")
	if configPath == "scenarios/system-monitor/initialization/configuration/investigation-triggers.json" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/home/matthalloran8"
		}
		configPath = filepath.Join(homeDir, "Vrooli/scenarios/system-monitor/initialization/configuration/investigation-triggers.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config struct {
		Triggers []struct {
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Icon        string  `json:"icon"`
			Enabled     bool    `json:"enabled"`
			AutoFix     bool    `json:"auto_fix"`
			Threshold   float64 `json:"threshold"`
			Unit        string  `json:"unit"`
			Condition   string  `json:"condition"`
		} `json:"triggers"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Update triggers map
	triggersMutex.Lock()
	defer triggersMutex.Unlock()

	for _, t := range config.Triggers {
		triggers[t.ID] = &TriggerConfig{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Icon:        t.Icon,
			Enabled:     t.Enabled,
			AutoFix:     t.AutoFix,
			Threshold:   t.Threshold,
			Unit:        t.Unit,
			Condition:   t.Condition,
		}
	}

	return nil
}

// saveTriggerConfig saves trigger configuration to JSON file
func saveTriggerConfig() error {
	configPath := filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios/system-monitor/initialization/configuration/investigation-triggers.json")
	if configPath == "scenarios/system-monitor/initialization/configuration/investigation-triggers.json" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/home/matthalloran8"
		}
		configPath = filepath.Join(homeDir, "Vrooli/scenarios/system-monitor/initialization/configuration/investigation-triggers.json")
	}

	// Copy trigger data while holding the lock
	triggersCopy := make(map[string]*TriggerConfig)
	triggersMutex.Lock()
	for id, trigger := range triggers {
		triggersCopy[id] = &TriggerConfig{
			ID:          trigger.ID,
			Name:        trigger.Name,
			Description: trigger.Description,
			Icon:        trigger.Icon,
			Enabled:     trigger.Enabled,
			AutoFix:     trigger.AutoFix,
			Threshold:   trigger.Threshold,
			Unit:        trigger.Unit,
			Condition:   trigger.Condition,
		}
	}
	triggersMutex.Unlock()

	// Now do file I/O without holding the lock
	// Read existing config to preserve extra fields
	existingData, err := os.ReadFile(configPath)
	var existingConfig map[string]interface{}
	if err == nil {
		json.Unmarshal(existingData, &existingConfig)
	} else {
		existingConfig = make(map[string]interface{})
	}

	// Build triggers array
	var triggerList []map[string]interface{}

	// First, preserve any existing triggers with their extra fields
	if existingTriggers, ok := existingConfig["triggers"].([]interface{}); ok {
		for _, et := range existingTriggers {
			if triggerMap, ok := et.(map[string]interface{}); ok {
				if id, ok := triggerMap["id"].(string); ok {
					// Check if we have an updated version
					if updatedTrigger, exists := triggersCopy[id]; exists {
						// Update core fields while preserving extras
						triggerMap["enabled"] = updatedTrigger.Enabled
						triggerMap["auto_fix"] = updatedTrigger.AutoFix
						triggerMap["threshold"] = updatedTrigger.Threshold
						triggerMap["name"] = updatedTrigger.Name
						triggerMap["description"] = updatedTrigger.Description
						triggerMap["icon"] = updatedTrigger.Icon
						triggerMap["unit"] = updatedTrigger.Unit
						triggerMap["condition"] = updatedTrigger.Condition
					}
					triggerList = append(triggerList, triggerMap)
				}
			}
		}
	}

	// Add any new triggers that weren't in the file
	for id, trigger := range triggersCopy {
		found := false
		for _, t := range triggerList {
			if t["id"] == id {
				found = true
				break
			}
		}
		if !found {
			triggerList = append(triggerList, map[string]interface{}{
				"id":          trigger.ID,
				"name":        trigger.Name,
				"description": trigger.Description,
				"icon":        trigger.Icon,
				"enabled":     trigger.Enabled,
				"auto_fix":    trigger.AutoFix,
				"threshold":   trigger.Threshold,
				"unit":        trigger.Unit,
				"condition":   trigger.Condition,
			})
		}
	}

	// Update the config
	existingConfig["triggers"] = triggerList

	// Update metadata
	if metadata, ok := existingConfig["metadata"].(map[string]interface{}); ok {
		metadata["last_modified"] = time.Now().Format(time.RFC3339)
	}

	// Marshal with indentation
	data, err := json.MarshalIndent(existingConfig, "", "    ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0644)
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start system-monitor

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT for compatibility
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}

	// Initialize database connection (optional - will use mock data if unavailable)
	var db *sql.DB
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		var err error
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to database: %v", err)
		} else {
			// Configure connection pool
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)

			// Implement exponential backoff for database connection
			maxRetries := 10
			baseDelay := 1 * time.Second
			maxDelay := 30 * time.Second

			log.Println("🔄 Attempting database connection with exponential backoff...")

			var pingErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				pingErr = db.Ping()
				if pingErr == nil {
					log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
					break
				}

				// Calculate exponential backoff delay
				delay := time.Duration(math.Min(
					float64(baseDelay)*math.Pow(2, float64(attempt)),
					float64(maxDelay),
				))

				// Add progressive jitter to prevent thundering herd
				jitterRange := float64(delay) * 0.25
				jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
				actualDelay := delay + jitter

				log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
				log.Printf("⏳ Waiting %v before next attempt", actualDelay)

				time.Sleep(actualDelay)
			}

			if pingErr != nil {
				log.Printf("⚠️ Database connection failed after %d attempts, continuing without database: %v", maxRetries, pingErr)
				db.Close()
				db = nil
			}
		}
	} else {
		log.Printf("DATABASE_URL not set, using mock data")
	}

	// Initialize MonitoringProcessor
	monitoringProcessor = NewMonitoringProcessor(db)

	// Initialize Settings Manager
	settingsManager = services.NewSettingsManager()
	settingsHandler = handlers.NewSettingsHandler(settingsManager)

	// Initialize Monitoring Service
	monitoringService = services.NewMonitoringService(settingsManager)

	// Set up monitoring callbacks
	monitoringService.SetThresholdChecker(func(ctx context.Context) error {
		log.Println("🔍 Running threshold check...")
		// Use the existing MonitoringProcessor
		req := ThresholdMonitorRequest{ForceCheck: false}
		_, err := monitoringProcessor.MonitorThresholds(ctx, req)
		if err != nil {
			log.Printf("Threshold check failed: %v", err)
			return err
		}
		log.Println("✅ Threshold check completed")
		return nil
	})

	monitoringService.SetAnomalyDetector(func(ctx context.Context) error {
		log.Println("🔍 Running anomaly detection...")
		// Use existing anomaly detection logic
		req := AnomalyInvestigationRequest{
			AnomalyType: "system_check",
			Severity:    "low",
			Context:     make(map[string]interface{}),
			TimeRange:   "5m",
			AutoResolve: false,
		}
		_, err := monitoringProcessor.InvestigateAnomaly(ctx, req)
		if err != nil {
			log.Printf("Anomaly detection failed: %v", err)
			return err
		}
		log.Println("✅ Anomaly detection completed")
		return nil
	})

	monitoringService.SetReportGenerator(func(ctx context.Context) error {
		log.Println("📊 Generating system report...")
		// Use existing report generation logic
		req := ReportGenerationRequest{
			ReportType: "hourly",
			TimeRange:  "1h",
			Format:     "json",
		}
		_, err := monitoringProcessor.GenerateReport(ctx, req)
		if err != nil {
			log.Printf("Report generation failed: %v", err)
			return err
		}
		log.Println("✅ System report generated")
		return nil
	})

	// Set callback for when active status changes
	settingsManager.SetActiveChangedCallback(func(active bool) {
		if active {
			log.Println("🟢 System monitor activated - monitoring loops will start")
		} else {
			log.Println("🔴 System monitor deactivated - monitoring loops paused")
		}
	})

	// Start the monitoring service
	if err := monitoringService.Start(); err != nil {
		log.Printf("Failed to start monitoring service: %v", err)
	}

	// Begin background sampling for metrics history
	startMetricsHistoryCollector()

	// Load trigger configuration from file
	if err := loadTriggerConfig(); err != nil {
		log.Printf("Failed to load trigger config, using defaults: %v", err)
	}

	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Metrics endpoints
	r.HandleFunc("/api/metrics/current", getCurrentMetricsHandler).Methods("GET")
	r.HandleFunc("/api/metrics/timeline", getMetricsTimelineHandler).Methods("GET")
	r.HandleFunc("/api/metrics/detailed", getDetailedMetricsHandler).Methods("GET")
	r.HandleFunc("/api/metrics/processes", getProcessMonitorHandler).Methods("GET")
	r.HandleFunc("/api/metrics/infrastructure", getInfrastructureMonitorHandler).Methods("GET")
	r.HandleFunc("/api/metrics/disk/details", getDiskDetailsHandler).Methods("GET")

	// Settings endpoints
	r.HandleFunc("/api/settings", settingsHandler.GetSettings).Methods("GET")
	r.HandleFunc("/api/settings", settingsHandler.UpdateSettings).Methods("PUT")
	r.HandleFunc("/api/settings/reset", settingsHandler.ResetSettings).Methods("POST")
	r.HandleFunc("/api/maintenance/state", settingsHandler.GetMaintenanceState).Methods("GET")
	r.HandleFunc("/api/maintenance/state", settingsHandler.SetMaintenanceState).Methods("POST")

	// Investigation endpoints
	r.HandleFunc("/api/investigations/latest", getLatestInvestigationHandler).Methods("GET")
	r.HandleFunc("/api/investigations/trigger", triggerInvestigationHandler).Methods("POST")
	r.HandleFunc("/api/investigations/scripts", listInvestigationScriptsHandler).Methods("GET")
	r.HandleFunc("/api/investigations/scripts/{id}", getInvestigationScriptHandler).Methods("GET")
	r.HandleFunc("/api/investigations/scripts/{id}/execute", executeInvestigationScriptHandler).Methods("POST")
	r.HandleFunc("/api/investigations/agent/spawn", spawnAgentHandler).Methods("POST")
	r.HandleFunc("/api/investigations/agent/{id}/status", getAgentStatusHandler).Methods("GET")
	r.HandleFunc("/api/investigations/agent/current", getCurrentAgentHandler).Methods("GET")
	r.HandleFunc("/api/investigations/cooldown", getCooldownStatusHandler).Methods("GET")
	r.HandleFunc("/api/investigations/cooldown/reset", resetCooldownHandler).Methods("POST")
	r.HandleFunc("/api/investigations/cooldown/period", updateCooldownPeriodHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/triggers", getTriggersHandler).Methods("GET")
	r.HandleFunc("/api/investigations/triggers/{id}", updateTriggerHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/triggers/{id}/threshold", updateTriggerThresholdHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}", getInvestigationHandler).Methods("GET")
	r.HandleFunc("/api/investigations/{id}/status", updateInvestigationStatusHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/findings", updateInvestigationFindingsHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/progress", updateInvestigationProgressHandler).Methods("PUT")
	r.HandleFunc("/api/investigations/{id}/step", addInvestigationStepHandler).Methods("POST")

	// Report endpoints
	r.HandleFunc("/api/reports", listReportsHandler).Methods("GET")
	r.HandleFunc("/api/reports/{id}", getReportHandler).Methods("GET")
	r.HandleFunc("/api/reports/generate", generateReportHandler).Methods("POST")

	// New MonitoringProcessor endpoints
	r.HandleFunc("/api/monitoring/threshold-check", thresholdMonitorHandler).Methods("POST")
	r.HandleFunc("/api/monitoring/investigate-anomaly", anomalyInvestigationHandler).Methods("POST")
	r.HandleFunc("/api/monitoring/generate-report", systemReportHandler).Methods("POST")

	// Debug logs endpoint for UI troubleshooting
	r.HandleFunc("/api/logs", getLogsHandler).Methods("GET")

	// Error logs endpoint for system output
	r.HandleFunc("/api/errors", getErrorsHandler).Methods("GET")
	r.HandleFunc("/api/errors/mark-read", markErrorsReadHandler).Methods("POST")

	// Process management endpoints
	r.HandleFunc("/api/processes/{pid}/kill", killProcessHandler).Methods("POST")

	// Enable CORS
	r.Use(corsMiddleware)

	// Enable comprehensive request/response logging
	r.Use(loggingMiddleware)

	log.Printf("System Monitor API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

// Removed getEnv function - no defaults allowed

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log incoming request
		reqMsg := fmt.Sprintf(">>> INCOMING REQUEST: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		log.Printf(reqMsg)
		addLogEntry("INFO", reqMsg, "http-server")

		headerMsg := fmt.Sprintf("    Headers: %+v", r.Header)
		log.Printf(headerMsg)
		addLogEntry("DEBUG", headerMsg, "http-server")

		userAgentMsg := fmt.Sprintf("    User-Agent: %s", r.UserAgent())
		log.Printf(userAgentMsg)
		addLogEntry("DEBUG", userAgentMsg, "http-server")

		// Create a response writer that captures status and size
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Log response
		duration := time.Since(start)
		respMsg := fmt.Sprintf("<<< RESPONSE: %s %s - Status: %d - Duration: %v",
			r.Method, r.URL.Path, rw.statusCode, duration)
		log.Printf(respMsg)
		addLogEntry("INFO", respMsg, "http-server")

		// Log any errors
		if rw.statusCode >= 400 {
			errorMsg := fmt.Sprintf("!!! ERROR RESPONSE: %s %s returned %d", r.Method, r.URL.Path, rw.statusCode)
			log.Printf(errorMsg)
			addLogEntry("ERROR", errorMsg, "http-server")
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Perform health checks
	checks := make(map[string]interface{})
	overallStatus := "healthy"

	// Check metrics collection capability
	cpuUsage := getCPUUsage()
	memUsage := getMemoryUsage()

	if cpuUsage >= 0 && memUsage >= 0 {
		checks["metrics_collection"] = map[string]interface{}{
			"status":  "healthy",
			"message": fmt.Sprintf("Collecting metrics - CPU: %.1f%%, Memory: %.1f%%", cpuUsage, memUsage),
		}
	} else {
		checks["metrics_collection"] = map[string]interface{}{
			"status":  "degraded",
			"message": "Unable to collect some metrics",
		}
		overallStatus = "degraded"
	}

	// Check investigation system
	investigationsMutex.RLock()
	investigationCount := len(investigations)
	investigationsMutex.RUnlock()

	checks["investigation_system"] = map[string]interface{}{
		"status":  "healthy",
		"message": fmt.Sprintf("Investigation system operational - %d investigations tracked", investigationCount),
	}

	// Check filesystem access (for logs/reports)
	if _, err := os.Stat("/tmp"); err == nil {
		checks["filesystem"] = map[string]interface{}{
			"status":  "healthy",
			"message": "Filesystem accessible",
		}
	} else {
		checks["filesystem"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  "Cannot access filesystem",
		}
		overallStatus = "unhealthy"
	}

	// Calculate uptime
	uptime := time.Since(startTime).Seconds()

	// Get processor status
	processorActive := settingsManager.IsActive()
	maintenanceState := settingsManager.GetMaintenanceState()

	response := HealthResponse{
		Status:           overallStatus,
		Service:          "system-monitor",
		Timestamp:        time.Now().Unix(),
		Uptime:           uptime,
		Checks:           checks,
		Version:          "2.0.0",
		ProcessorActive:  processorActive,
		MaintenanceState: maintenanceState,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func collectMetricSample() (MetricsResponse, metricHistorySample) {
	now := time.Now()
	cpu := getCPUUsage()
	mem := getMemoryUsage()
	tcp := getTCPConnections()

	metrics := MetricsResponse{
		CPUUsage:       cpu,
		MemoryUsage:    mem,
		TCPConnections: tcp,
		Timestamp:      now.Format(time.RFC3339),
	}

	sample := metricHistorySample{
		timestamp:      now,
		cpuUsage:       cpu,
		memoryUsage:    mem,
		tcpConnections: tcp,
	}

	return metrics, sample
}

func storeMetricSample(sample metricHistorySample) {
	metricsHistoryMutex.Lock()
	defer metricsHistoryMutex.Unlock()

	metricsHistory = append(metricsHistory, sample)
	cutoff := time.Now().Add(-metricsHistoryRetention)

	// drop stale samples beyond retention window
	trimIndex := 0
	for ; trimIndex < len(metricsHistory); trimIndex++ {
		if metricsHistory[trimIndex].timestamp.After(cutoff) {
			break
		}
	}
	if trimIndex > 0 && trimIndex < len(metricsHistory) {
		metricsHistory = append([]metricHistorySample{}, metricsHistory[trimIndex:]...)
	} else if trimIndex >= len(metricsHistory) {
		metricsHistory = metricsHistory[:0]
	}
}

func getMetricHistory(window time.Duration) []metricHistorySample {
	metricsHistoryMutex.RLock()
	defer metricsHistoryMutex.RUnlock()

	if len(metricsHistory) == 0 {
		return nil
	}

	cutoff := time.Now().Add(-window)
	result := make([]metricHistorySample, 0, len(metricsHistory))
	for _, sample := range metricsHistory {
		if sample.timestamp.After(cutoff) || sample.timestamp.Equal(cutoff) {
			result = append(result, sample)
		}
	}
	return result
}

func getLatestMetricSample() (metricHistorySample, bool) {
	metricsHistoryMutex.RLock()
	defer metricsHistoryMutex.RUnlock()

	if len(metricsHistory) == 0 {
		return metricHistorySample{}, false
	}
	return metricsHistory[len(metricsHistory)-1], true
}

func metricsResponseFromSample(sample metricHistorySample) MetricsResponse {
	return MetricsResponse{
		CPUUsage:       sample.cpuUsage,
		MemoryUsage:    sample.memoryUsage,
		TCPConnections: sample.tcpConnections,
		Timestamp:      sample.timestamp.Format(time.RFC3339),
	}
}

func startMetricsHistoryCollector() {
	if metricsSampleInterval <= 0 {
		metricsSampleInterval = 5 * time.Second
	}

	log.Printf("🟢 Starting metrics history collector (interval: %v, retention: %v)", metricsSampleInterval, metricsHistoryRetention)

	go func() {
		// Capture an initial sample so the UI has immediate data
		_, sample := collectMetricSample()
		storeMetricSample(sample)

		ticker := time.NewTicker(metricsSampleInterval)
		defer ticker.Stop()

		for range ticker.C {
			_, sample := collectMetricSample()
			storeMetricSample(sample)
		}
	}()
}

func getCurrentMetricsHandler(w http.ResponseWriter, r *http.Request) {
	sample, ok := getLatestMetricSample()
	if !ok || time.Since(sample.timestamp) > metricsSampleInterval*2 {
		metrics, newSample := collectMetricSample()
		storeMetricSample(newSample)
		sample = newSample
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(metrics); err != nil {
			log.Printf("Failed to encode current metrics: %v", err)
		}
		return
	}

	metrics := metricsResponseFromSample(sample)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Failed to encode current metrics: %v", err)
	}
}

func getMetricsTimelineHandler(w http.ResponseWriter, r *http.Request) {
	windowSeconds := 60
	if raw := r.URL.Query().Get("window"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			windowSeconds = parsed
		}
	}

	maxWindow := int(metricsHistoryRetention.Seconds())
	if windowSeconds > maxWindow {
		windowSeconds = maxWindow
	}
	if windowSeconds <= 0 {
		windowSeconds = 60
	}

	samples := getMetricHistory(time.Duration(windowSeconds) * time.Second)
	if len(samples) == 0 {
		_, sample := collectMetricSample()
		storeMetricSample(sample)
		samples = getMetricHistory(time.Duration(windowSeconds) * time.Second)
	}

	response := MetricsTimelineResponse{
		WindowSeconds:         windowSeconds,
		SampleIntervalSeconds: int(metricsSampleInterval / time.Second),
		Samples:               make([]MetricTimelineSample, 0, len(samples)),
	}

	for _, sample := range samples {
		response.Samples = append(response.Samples, MetricTimelineSample{
			Timestamp:      sample.timestamp.Format(time.RFC3339),
			CPUUsage:       sample.cpuUsage,
			MemoryUsage:    sample.memoryUsage,
			TCPConnections: sample.tcpConnections,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode metrics timeline: %v", err)
	}
}

func getLatestInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	if latestInvestigation == nil {
		// Default mock investigation if none triggered yet
		investigation := Investigation{
			ID:        "inv_default",
			Status:    "pending",
			AnomalyID: "none",
			StartTime: time.Now(),
			Findings:  "No investigations have been triggered yet. Click 'RUN ANOMALY CHECK' to start an investigation.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(investigation)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(latestInvestigation)
}

func generateReportHandler(w http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Mock report generation
	reportID := "report_" + strconv.FormatInt(time.Now().Unix(), 10)
	now := time.Now()

	// Store the generated report
	reportsMutex.Lock()
	generatedReport := GeneratedReport{
		ReportID:    reportID,
		ReportType:  req.Type,
		GeneratedAt: now,
		DateRange:   fmt.Sprintf("%s to %s", now.Add(-24*time.Hour).Format("Jan 02, 3:04 PM"), now.Format("Jan 02, 3:04 PM")),
		Status:      "success",
	}
	generatedReports = append(generatedReports, generatedReport)

	// Keep only last 100 reports
	if len(generatedReports) > 100 {
		generatedReports = generatedReports[len(generatedReports)-100:]
	}
	reportsMutex.Unlock()

	response := map[string]interface{}{
		"status":       "success",
		"report_id":    reportID,
		"report_type":  req.Type,
		"generated_at": now.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// logError logs an error to the system output
func logError(level, message, source, details string) {
	errorLogsMutex.Lock()
	defer errorLogsMutex.Unlock()

	errorLog := ErrorLog{
		ID:        fmt.Sprintf("err_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    source,
		Details:   details,
		Unread:    true,
	}

	errorLogs = append(errorLogs, errorLog)
	unreadErrorCount++

	// Keep only last 1000 errors
	if len(errorLogs) > 1000 {
		errorLogs = errorLogs[len(errorLogs)-1000:]
	}

	// Also log to stdout for debugging
	log.Printf("[%s] %s: %s (source: %s)", level, message, details, source)
}

func getReportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reportID := vars["id"]

	if reportID == "" {
		logError("error", "Report ID is required", "getReportHandler", "Missing report ID in URL")
		http.Error(w, "Report ID is required", http.StatusBadRequest)
		return
	}

	reportsMutex.RLock()
	defer reportsMutex.RUnlock()

	// Find the report by ID
	var foundReport *GeneratedReport
	for i, report := range generatedReports {
		if report.ReportID == reportID {
			foundReport = &generatedReports[i]
			break
		}
	}

	if foundReport == nil {
		logError("warning", "Report not found", "getReportHandler", fmt.Sprintf("Report ID: %s", reportID))
		http.Error(w, "Report not found", http.StatusNotFound)
		return
	}

	// If report doesn't have content, generate it now
	if foundReport.Content == nil {
		content := generateReportContent(foundReport.ReportType)
		foundReport.Content = content
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(foundReport); err != nil {
		logError("error", "Failed to encode report response", "getReportHandler", err.Error())
	}
}

func listReportsHandler(w http.ResponseWriter, r *http.Request) {
	reportsMutex.RLock()
	defer reportsMutex.RUnlock()

	// Return the list of generated reports (without full content)
	reports := make([]GeneratedReport, len(generatedReports))
	for i, report := range generatedReports {
		reports[i] = GeneratedReport{
			ReportID:    report.ReportID,
			ReportType:  report.ReportType,
			GeneratedAt: report.GeneratedAt,
			DateRange:   report.DateRange,
			Status:      report.Status,
			// Don't include content in list view for performance
		}
	}

	response := map[string]interface{}{
		"reports": reports,
		"count":   len(reports),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logError("error", "Failed to encode reports response", "listReportsHandler", err.Error())
	}
}

// generateReportContent creates realistic report content
func generateReportContent(reportType string) *FullReportContent {
	now := time.Now()

	// Get current metrics for realistic data
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()
	tcpConnections := getTCPConnections()

	// Generate realistic historical data points
	baseTime := now.Add(-24 * time.Hour)
	if reportType == "weekly" {
		baseTime = now.Add(-7 * 24 * time.Hour)
	}

	// Simulate metrics with some variation
	cpuMin := cpuUsage * 0.3
	cpuMax := cpuUsage * 1.8
	memMin := memoryUsage * 0.8
	memMax := memoryUsage * 1.2

	executiveSummary := ReportExecutiveSummary{
		OverallHealth: "healthy",
		KeyFindings: []string{
			fmt.Sprintf("Average CPU usage: %.1f%%", cpuUsage),
			fmt.Sprintf("Average memory usage: %.1f%%", memoryUsage),
			fmt.Sprintf("Average TCP connections: %d", tcpConnections),
			"System performance within normal parameters",
		},
		TimeDescription: "24 hours",
		MetricsAnalyzed: 288, // 24 hours * 12 (5-min intervals)
	}

	if reportType == "weekly" {
		executiveSummary.TimeDescription = "7 days"
		executiveSummary.MetricsAnalyzed = 2016 // 7 days * 288
	}

	performanceAnalysis := ReportPerformanceAnalysis{
		CPU: ReportMetricStats{
			Average:   cpuUsage,
			Min:       cpuMin,
			Max:       cpuMax,
			StdDev:    (cpuMax - cpuMin) / 4, // Rough estimate
			PeakValue: cpuMax,
			PeakTime:  baseTime.Add(time.Duration(rand.Intn(1440)) * time.Minute),
			MinTime:   baseTime.Add(time.Duration(rand.Intn(1440)) * time.Minute),
		},
		Memory: ReportMetricStats{
			Average:   memoryUsage,
			Min:       memMin,
			Max:       memMax,
			StdDev:    (memMax - memMin) / 4,
			PeakValue: memMax,
			PeakTime:  baseTime.Add(time.Duration(rand.Intn(1440)) * time.Minute),
			MinTime:   baseTime.Add(time.Duration(rand.Intn(1440)) * time.Minute),
		},
		TimeRange: fmt.Sprintf("%s to %s",
			baseTime.Format("2006-01-02 15:04"),
			now.Format("2006-01-02 15:04")),
	}

	// Determine health status
	if cpuUsage > 80 || memoryUsage > 85 {
		executiveSummary.OverallHealth = "warning"
	}
	if cpuUsage > 95 || memoryUsage > 95 {
		executiveSummary.OverallHealth = "critical"
	}

	// Generate trends
	trends := []ReportTrend{}
	if cpuUsage > 15 {
		trends = append(trends, ReportTrend{
			Name:          "CPU Usage",
			Direction:     "stable",
			Change:        0.2,
			ChangePercent: 1.5,
		})
	}

	// Generate recommendations
	recommendations := []string{"System performance appears normal - continue regular monitoring"}
	if cpuUsage > 80 {
		recommendations = []string{"Consider scaling CPU resources or optimizing high-CPU processes"}
	}
	if memoryUsage > 85 {
		recommendations = append(recommendations, "Monitor memory usage closely and consider increasing available memory")
	}

	// Generate highlights
	highlights := []string{
		fmt.Sprintf("Analyzed %d metric data points over %s", executiveSummary.MetricsAnalyzed, executiveSummary.TimeDescription),
	}
	if cpuMax > 90 {
		highlights = append(highlights, fmt.Sprintf("Peak CPU usage: %.1f%% at %s", cpuMax, performanceAnalysis.CPU.PeakTime.Format("Jan 02 15:04")))
	}
	if memMax > 90 {
		highlights = append(highlights, fmt.Sprintf("Peak memory usage: %.1f%% at %s", memMax, performanceAnalysis.Memory.PeakTime.Format("Jan 02 15:04")))
	}

	duration := now.Sub(baseTime)
	actualDuration := formatDuration(duration)
	dateRangeDisplay := fmt.Sprintf("%s to %s",
		baseTime.Format("January 2, 2006 3:04 PM MST"),
		now.Format("January 2, 2006 3:04 PM MST"))

	return &FullReportContent{
		ExecutiveSummary:    executiveSummary,
		PerformanceAnalysis: performanceAnalysis,
		Trends:              trends,
		Recommendations:     recommendations,
		Highlights:          highlights,
		MetricsCount:        executiveSummary.MetricsAnalyzed,
		AlertsCount:         0,
		ActualDuration:      actualDuration,
		DateRangeDisplay:    dateRangeDisplay,
	}
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
		}
		return fmt.Sprintf("%d hours", hours)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		if hours > 0 {
			return fmt.Sprintf("%d days, %d hours", days, hours)
		}
		return fmt.Sprintf("%d days", days)
	}
}

func getCPUUsage() float64 {
	// Get CPU usage using system commands
	if runtime.GOOS == "linux" {
		// Use top command for accurate CPU usage
		cmd := exec.Command("bash", "-c", "top -bn2 -d 0.5 | grep '^%Cpu' | tail -1 | awk '{print 100-$8}'")
		output, err := cmd.Output()
		if err != nil {
			// Fallback to simpler method
			cmd = exec.Command("bash", "-c", "top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\\([0-9.]*\\)%* id.*/\\1/' | awk '{print 100 - $1}'")
			output, err = cmd.Output()
			if err != nil {
				return 25.0 // Default fallback
			}
		}

		usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil {
			return 25.0
		}
		return usage
	}

	// Fallback for non-Linux systems
	return float64(15 + (time.Now().Second() % 30)) // Mock varying CPU usage
}

func getMemoryUsage() float64 {
	// Get memory usage using system commands
	if runtime.GOOS == "linux" {
		// Use (total - available) / total for accurate memory usage
		cmd := exec.Command("bash", "-c", "free | grep Mem | awk '{print (($2-$7)/$2) * 100.0}'")
		output, err := cmd.Output()
		if err != nil {
			return 30.0 // Default fallback
		}

		usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
		if err != nil {
			return 30.0
		}
		// Ensure non-negative value
		if usage < 0 {
			return 0.0
		}
		return usage
	}

	// Fallback for non-Linux systems
	return float64(45 + (time.Now().Second() % 20)) // Mock varying memory usage
}

func getTCPConnections() int {
	// Get TCP connection count using netstat
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | grep ESTABLISHED | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return 0
	}
	return count
}

// Get detailed TCP connection states
func getTCPConnectionStates() TCPConnectionStates {
	states := TCPConnectionStates{}

	// Parse netstat output for connection states
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | awk 'NR>2 {print $6}' | sort | uniq -c")
	output, err := cmd.Output()
	if err != nil {
		return states
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		count, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		state := strings.ToUpper(parts[1])
		switch state {
		case "ESTABLISHED":
			states.Established = count
		case "TIME_WAIT":
			states.TimeWait = count
		case "CLOSE_WAIT":
			states.CloseWait = count
		case "FIN_WAIT1":
			states.FinWait1 = count
		case "FIN_WAIT2":
			states.FinWait2 = count
		case "SYN_SENT":
			states.SynSent = count
		case "SYN_RECV":
			states.SynRecv = count
		case "CLOSING":
			states.Closing = count
		case "LAST_ACK":
			states.LastAck = count
		case "LISTEN":
			states.Listen = count
		}

		states.Total += count
	}

	return states
}

// Get top processes by CPU usage
func getTopProcessesByCPU(limit int) []ProcessInfo {
	var processes []ProcessInfo

	// Use ps to get process info sorted by CPU
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("ps -eo pid,comm,%%cpu,%%mem,nlwp --sort=-%%cpu --no-headers | head -%d", limit))
	output, err := cmd.Output()
	if err != nil {
		return processes
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		memoryPercent, _ := strconv.ParseFloat(fields[3], 64)
		threads, _ := strconv.Atoi(fields[4])

		// Get memory in MB (rough calculation)
		memoryMB := memoryPercent * getSystemMemoryGB() * 1024 / 100

		// Get file descriptors for this process
		fdCount := getProcessFileDescriptors(pid)

		processes = append(processes, ProcessInfo{
			PID:        pid,
			Name:       fields[1],
			CPUPercent: cpuPercent,
			MemoryMB:   memoryMB,
			Threads:    threads,
			FDs:        fdCount,
			Status:     "running",
			Goroutines: getProcessGoroutines(fields[1], pid),
		})
	}

	return processes
}

// Get top processes by memory usage
func getTopProcessesByMemory(limit int) []ProcessInfo {
	var processes []ProcessInfo

	// Use ps to get process info sorted by memory
	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("ps -eo pid,comm,%%cpu,%%mem,nlwp,rss --sort=-%%mem --no-headers | head -%d", limit))
	output, err := cmd.Output()
	if err != nil {
		return processes
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpuPercent, _ := strconv.ParseFloat(fields[2], 64)
		_ = fields[3] // ignore memory percent, we use RSS instead
		threads, _ := strconv.Atoi(fields[4])
		rssKB, _ := strconv.ParseFloat(fields[5], 64)

		// Convert RSS from KB to MB
		memoryMB := rssKB / 1024

		// Get file descriptors for this process
		fdCount := getProcessFileDescriptors(pid)

		processes = append(processes, ProcessInfo{
			PID:        pid,
			Name:       fields[1],
			CPUPercent: cpuPercent,
			MemoryMB:   memoryMB,
			Threads:    threads,
			FDs:        fdCount,
			Status:     "running",
			Goroutines: getProcessGoroutines(fields[1], pid),
		})
	}

	return processes
}

// Helper functions
func getSystemMemoryGB() float64 {
	cmd := exec.Command("bash", "-c", "free -g | grep Mem | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return 8.0 // fallback
	}

	memGB, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 8.0
	}
	return memGB
}

func getProcessFileDescriptors(pid int) int {
	// Count files in /proc/PID/fd/
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	files, err := os.ReadDir(fdDir)
	if err != nil {
		return 0
	}
	return len(files)
}

func getProcessGoroutines(processName string, pid int) int {
	// Only check for Go processes
	if !strings.Contains(processName, "api") && !strings.Contains(processName, "go") {
		return 0
	}

	// Try to get goroutine count from pprof endpoint (if available)
	// This is a mock implementation - in real world you'd query the actual pprof endpoint
	return 0
}

func getLoadAverage() []float64 {
	cmd := exec.Command("bash", "-c", "cat /proc/loadavg | awk '{print $1, $2, $3}'")
	output, err := cmd.Output()
	if err != nil {
		return []float64{0.0, 0.0, 0.0}
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 3 {
		return []float64{0.0, 0.0, 0.0}
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)

	return []float64{load1, load5, load15}
}

func getContextSwitches() int64 {
	cmd := exec.Command("bash", "-c", "grep '^ctxt' /proc/stat | awk '{print $2}'")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	ctxt, _ := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	return ctxt
}

// Cache for file descriptor metrics
var (
	fdCacheMutex sync.RWMutex
	fdCacheUsed  int
	fdCacheMax   int
	fdCacheTime  time.Time
)

// Simple rate limiter for expensive endpoints
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Clean old requests
	var validRequests []time.Time
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// Check if under limit
	if len(validRequests) >= rl.limit {
		rl.requests[key] = validRequests
		return false
	}

	// Add new request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests
	return true
}

// Initialize rate limiters for expensive endpoints
var (
	detailedMetricsLimiter = NewRateLimiter(10, time.Minute) // 10 requests per minute
	processMetricsLimiter  = NewRateLimiter(10, time.Minute) // 10 requests per minute
	infraMetricsLimiter    = NewRateLimiter(10, time.Minute) // 10 requests per minute
)

func getSystemFileDescriptors() (int, int) {
	// Check cache (valid for 60 seconds)
	fdCacheMutex.RLock()
	if time.Since(fdCacheTime) < 60*time.Second && fdCacheUsed > 0 {
		used, max := fdCacheUsed, fdCacheMax
		fdCacheMutex.RUnlock()
		return used, max
	}
	fdCacheMutex.RUnlock()

	// Get current FD count from /proc/sys/fs/file-nr
	// Format: allocated-fds allocated-but-unused max-fds
	data, err := os.ReadFile("/proc/sys/fs/file-nr")
	used := 0
	if err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 1 {
			used, _ = strconv.Atoi(parts[0])
		}
	}

	// Get max FD limit
	data2, err2 := os.ReadFile("/proc/sys/fs/file-max")
	max := 65536 // fallback
	if err2 == nil {
		max, _ = strconv.Atoi(strings.TrimSpace(string(data2)))
	}

	// Update cache
	fdCacheMutex.Lock()
	fdCacheUsed = used
	fdCacheMax = max
	fdCacheTime = time.Now()
	fdCacheMutex.Unlock()

	return used, max
}

func spawnAgentHandler(w http.ResponseWriter, r *http.Request) {
	// This is the endpoint the UI calls when spawning an agent
	// It's essentially the same as triggerInvestigationHandler
	triggerInvestigationHandler(w, r)
}

func getAgentStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	investigationsMutex.RLock()
	investigation, exists := investigations[id]
	investigationsMutex.RUnlock()

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Investigation not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       investigation.ID,
		"status":   investigation.Status,
		"progress": investigation.Progress,
		"findings": investigation.Findings,
	})
}

func getCurrentAgentHandler(w http.ResponseWriter, r *http.Request) {
	investigationsMutex.RLock()
	var currentAgent *Investigation

	// Find the most recent running investigation
	for _, inv := range investigations {
		if inv.Status == "in_progress" || inv.Status == "queued" {
			if currentAgent == nil || inv.StartTime.After(currentAgent.StartTime) {
				currentAgent = inv
			}
		}
	}
	investigationsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if currentAgent != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agent": map[string]interface{}{
				"id":        currentAgent.ID,
				"status":    currentAgent.Status,
				"progress":  currentAgent.Progress,
				"startTime": currentAgent.StartTime.Format(time.RFC3339),
			},
		})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"agent": nil,
		})
	}
}

func triggerInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body for auto_fix parameter and optional note
	var reqBody struct {
		AutoFix bool   `json:"auto_fix"`
		Note    string `json:"note,omitempty"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&reqBody)
	}

	// Generate new investigation ID
	investigationID := "inv_" + strconv.FormatInt(time.Now().Unix(), 10)

	// Create investigation with queued status
	investigation := &Investigation{
		ID:        investigationID,
		Status:    "queued",
		AnomalyID: "ai_investigation_" + strconv.FormatInt(time.Now().Unix(), 10),
		StartTime: time.Now(),
		Findings:  "Investigation queued for processing...",
		Progress:  0,
		Details:   make(map[string]interface{}),
		Steps:     []InvestigationStep{},
	}

	// Store auto_fix mode in investigation details
	investigation.Details["auto_fix"] = reqBody.AutoFix
	investigation.Details["operation_mode"] = "auto-fix"
	if !reqBody.AutoFix {
		investigation.Details["operation_mode"] = "report-only"
	}

	// Store optional user note if provided
	if reqBody.Note != "" {
		investigation.Details["user_note"] = reqBody.Note
	}

	// Store investigation
	investigationsMutex.Lock()
	investigations[investigationID] = investigation
	latestInvestigation = investigation
	investigationsMutex.Unlock()

	// Start investigation in background
	go func() {
		runClaudeInvestigation(investigationID, reqBody.AutoFix, reqBody.Note)
	}()

	// Return immediate response with API info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "queued",
		"investigation_id": investigationID,
		"api_base_url":     "http://localhost:8080",
		"message":          "Investigation queued for Claude Code processing",
	})
}

// getCooldownStatusHandler returns the current cooldown status
func getCooldownStatusHandler(w http.ResponseWriter, r *http.Request) {
	cooldownMutex.RLock()
	defer cooldownMutex.RUnlock()

	remainingSeconds := 0
	isReady := true

	if !lastTriggerTime.IsZero() {
		elapsed := time.Since(lastTriggerTime)
		cooldownDuration := time.Duration(cooldownPeriodSeconds) * time.Second
		if elapsed < cooldownDuration {
			remainingSeconds = int((cooldownDuration - elapsed).Seconds())
			isReady = false
		}
	}

	response := map[string]interface{}{
		"cooldown_period_seconds": cooldownPeriodSeconds,
		"remaining_seconds":       remainingSeconds,
		"last_trigger_time":       lastTriggerTime,
		"is_ready":                isReady,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// resetCooldownHandler resets the cooldown timer
func resetCooldownHandler(w http.ResponseWriter, r *http.Request) {
	cooldownMutex.Lock()
	defer cooldownMutex.Unlock()

	lastTriggerTime = time.Time{} // Reset to zero time

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cooldown_reset"})
}

// updateCooldownPeriodHandler updates the cooldown period
func updateCooldownPeriodHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CooldownPeriodSeconds int `json:"cooldown_period_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cooldownMutex.Lock()
	cooldownPeriodSeconds = req.CooldownPeriodSeconds
	cooldownMutex.Unlock()

	// TODO: Save to configuration file if needed

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// getTriggersHandler returns all trigger configurations
func getTriggersHandler(w http.ResponseWriter, r *http.Request) {
	triggersMutex.RLock()
	defer triggersMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(triggers)
}

// updateTriggerHandler updates a trigger configuration
func updateTriggerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["id"]

	var req struct {
		Enabled   *bool    `json:"enabled,omitempty"`
		AutoFix   *bool    `json:"auto_fix,omitempty"`
		Threshold *float64 `json:"threshold,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update trigger with lock
	triggersMutex.Lock()
	trigger, exists := triggers[triggerID]
	if !exists {
		triggersMutex.Unlock()
		http.Error(w, "Trigger not found", http.StatusNotFound)
		return
	}

	if req.Enabled != nil {
		trigger.Enabled = *req.Enabled
	}
	if req.AutoFix != nil {
		trigger.AutoFix = *req.AutoFix
	}
	if req.Threshold != nil {
		trigger.Threshold = *req.Threshold
	}
	triggersMutex.Unlock() // Release lock before file I/O

	// Save to configuration file (without holding lock)
	if err := saveTriggerConfig(); err != nil {
		log.Printf("Failed to save trigger config: %v", err)
		// Continue anyway - in-memory update succeeded
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// updateTriggerThresholdHandler updates just the threshold of a trigger
func updateTriggerThresholdHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	triggerID := vars["id"]

	var req struct {
		Threshold float64 `json:"threshold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	triggersMutex.Lock()
	defer triggersMutex.Unlock()

	trigger, exists := triggers[triggerID]
	if !exists {
		http.Error(w, "Trigger not found", http.StatusNotFound)
		return
	}

	trigger.Threshold = req.Threshold

	// TODO: Save to configuration file if needed

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func runClaudeInvestigation(investigationID string, autoFix bool, userNote string) {
	// Update status to in_progress
	updateInvestigationField(investigationID, "Status", "in_progress")
	updateInvestigationField(investigationID, "Progress", 10)

	// Get current system metrics for context
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()
	tcpConnections := getTCPConnections()
	timestamp := time.Now().Format(time.RFC3339)

	// Try Claude Code first, but have a good fallback
	var findings string
	var details map[string]interface{}

	// Determine operation mode
	operationMode := "report-only"
	if autoFix {
		operationMode = "auto-fix"
	}

	// Load investigation prompt with system context and script capabilities
	prompt, err := loadAndProcessPromptWithInvestigation(cpuUsage, memoryUsage, tcpConnections, timestamp, investigationID, operationMode, userNote)
	if err != nil {
		log.Printf("Failed to load prompt template: %v", err)
		// Enhanced fallback prompt with investigation scripts context
		userNoteSection := ""
		if userNote != "" {
			userNoteSection = fmt.Sprintf(`

## User Instructions
**Note from user**: %s`, userNote)
		}

		prompt = fmt.Sprintf(`# System Anomaly Investigation

## Investigation Context
- **Investigation ID**: %s
- **Operation Mode**: %s
- **Trigger**: Manual investigation request
- **Timestamp**: %s
- **API Base URL**: http://localhost:8080%s

## Current System Metrics
- **CPU Usage**: %.2f%%
- **Memory Usage**: %.2f%%  
- **TCP Connections**: %d

## Your Investigation Capabilities

### Investigation Scripts System
You have **full read/write access** to the investigation scripts directory at:
- **../investigations/active/** - Ready-to-use investigation scripts
- **../investigations/manifest.json** - Script registry
- **../investigations/templates/** - Script templates

### Available Investigation Tools
Check for existing investigation scripts that may help:
1. **high-cpu-analysis.sh** - Identifies excessive CPU consumers and traces origins
2. **process-genealogy.sh** - Traces parent-child relationships for spawn issues

### Your Task
1. **Immediate Analysis**: Assess current system state for anomalies
2. **Pattern Detection**: Look for known problematic patterns (excessive lsof, worker spawning, etc.)
3. **Root Cause Investigation**: Use available tools and create new ones as needed
4. **Recommendations**: Provide specific, actionable solutions
5. **Tool Enhancement**: Improve existing investigation scripts or create new ones

Please begin your systematic investigation now, utilizing the investigation scripts framework to build lasting diagnostic capabilities.`,
			investigationID, operationMode, timestamp, userNoteSection, cpuUsage, memoryUsage, tcpConnections)
	}

	// Try Claude Code investigation with proper timeout and settings
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			vrooliRoot = filepath.Join(homeDir, "Vrooli")
		}
	}
	if vrooliRoot == "" {
		vrooliRoot = "." // Fallback to current directory
	}

	// Set timeout (10 minutes for investigation)
	timeoutDuration := 10 * time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	// Use resource-claude-code directly with proper arguments
	cmd := exec.CommandContext(ctx, "resource-claude-code", "run", "-")
	cmd.Dir = vrooliRoot

	// Apply Claude execution settings via environment variables
	cmd.Env = append(os.Environ(),
		"MAX_TURNS=75",
		"ALLOWED_TOOLS=Read,Write,Edit,Bash,LS,Glob,Grep",
		"TIMEOUT=600", // 10 minutes in seconds
		"SKIP_PERMISSIONS=yes",
	)

	// Set up pipes and execute Claude Code
	var output []byte
	var claudeWorked bool

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Failed to create stdin pipe for Claude Code: %v", err)
		claudeWorked = false
	} else {
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Failed to create stdout pipe for Claude Code: %v", err)
			claudeWorked = false
		} else {
			// Start the command
			if err := cmd.Start(); err != nil {
				log.Printf("Failed to start Claude Code: %v", err)
				claudeWorked = false
			} else {
				// Write prompt to stdin in a goroutine
				go func() {
					defer stdinPipe.Close()
					if _, err := stdinPipe.Write([]byte(prompt)); err != nil {
						log.Printf("Failed to write prompt to Claude Code stdin: %v", err)
					}
				}()

				// Read output
				output, err = io.ReadAll(stdoutPipe)

				// Wait for command to complete
				waitErr := cmd.Wait()

				// Check if Claude Code actually worked
				claudeWorked = err == nil && waitErr == nil && !strings.Contains(string(output), "USAGE:") && !strings.Contains(string(output), "Failed to load library")

				if !claudeWorked {
					log.Printf("Claude Code execution failed - err: %v, waitErr: %v, output preview: %.500s", err, waitErr, string(output))
				}
			}
		}
	}

	if claudeWorked {
		findings = string(output)
		details = map[string]interface{}{
			"source":     "claude_code",
			"risk_level": "low",
		}
	} else {
		// Fallback: Perform basic system analysis
		log.Printf("Claude Code unavailable, using fallback investigation")
		updateInvestigationField(investigationID, "Progress", 25)

		// Analyze system metrics
		riskLevel := "low"
		anomalies := []string{}
		recommendations := []string{}

		if cpuUsage > 80 {
			riskLevel = "high"
			anomalies = append(anomalies, fmt.Sprintf("High CPU usage: %.2f%%", cpuUsage))
			recommendations = append(recommendations, "Investigate top CPU-consuming processes with 'top' or 'htop'")
		} else if cpuUsage > 60 {
			riskLevel = "medium"
			anomalies = append(anomalies, fmt.Sprintf("Elevated CPU usage: %.2f%%", cpuUsage))
		}

		updateInvestigationField(investigationID, "Progress", 50)

		if memoryUsage > 90 {
			if riskLevel == "low" {
				riskLevel = "high"
			}
			anomalies = append(anomalies, fmt.Sprintf("Critical memory usage: %.2f%%", memoryUsage))
			recommendations = append(recommendations, "Check for memory leaks and consider increasing system RAM")
		} else if memoryUsage > 75 {
			if riskLevel == "low" {
				riskLevel = "medium"
			}
			anomalies = append(anomalies, fmt.Sprintf("High memory usage: %.2f%%", memoryUsage))
		}

		updateInvestigationField(investigationID, "Progress", 75)

		if tcpConnections > 500 {
			if riskLevel == "low" {
				riskLevel = "medium"
			}
			anomalies = append(anomalies, fmt.Sprintf("High number of TCP connections: %d", tcpConnections))
			recommendations = append(recommendations, "Review network connections with 'netstat -tuln'")
		}

		// Build findings report
		findings = fmt.Sprintf(`### Investigation Summary

**Status**: %s
**Investigation ID**: %s
**Timestamp**: %s

**Key Findings**:
- CPU Usage: %.2f%%
- Memory Usage: %.2f%%
- TCP Connections: %d

**Anomalies Detected**: %d
%s

**Risk Level**: %s

**Recommendations**:
%s

**Technical Details**:
System metrics are %s. %s`,
			func() string {
				if len(anomalies) > 0 {
					return "Warning"
				}
				return "Normal"
			}(),
			investigationID,
			timestamp,
			cpuUsage,
			memoryUsage,
			tcpConnections,
			len(anomalies),
			func() string {
				if len(anomalies) == 0 {
					return "- No anomalies detected"
				}
				result := ""
				for _, a := range anomalies {
					result += fmt.Sprintf("- %s\n", a)
				}
				return result
			}(),
			riskLevel,
			func() string {
				if len(recommendations) == 0 {
					return "- Continue normal monitoring"
				}
				result := ""
				for _, r := range recommendations {
					result += fmt.Sprintf("- %s\n", r)
				}
				return result
			}(),
			func() string {
				if len(anomalies) == 0 {
					return "within normal parameters"
				}
				return "showing some concerns"
			}(),
			func() string {
				if riskLevel == "high" {
					return "Immediate attention recommended."
				} else if riskLevel == "medium" {
					return "Monitor closely for escalation."
				}
				return "No immediate action required."
			}(),
		)

		details = map[string]interface{}{
			"source":                "fallback_analysis",
			"risk_level":            riskLevel,
			"anomalies_found":       len(anomalies),
			"recommendations_count": len(recommendations),
			"critical_issues":       riskLevel == "high",
		}
	}

	// Update investigation with findings
	investigationsMutex.Lock()
	if inv, exists := investigations[investigationID]; exists {
		inv.Findings = findings
		inv.Details = details
		inv.Status = "completed"
		inv.EndTime = time.Now()
		inv.Progress = 100
	}
	investigationsMutex.Unlock()
}

func loadAndProcessPromptWithInvestigation(cpuUsage, memoryUsage float64, tcpConnections int, timestamp string, investigationID string, operationMode string, userNote string) (string, error) {
	// Get VROOLI_ROOT with proper fallback
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/root" // fallback for containers
		}
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}

	// Construct scenarios directory path
	scenariosDir := filepath.Join(vrooliRoot, "scenarios")

	// Construct path to prompt file
	// Try multiple locations where the prompt file might exist
	promptPaths := []string{
		filepath.Join(scenariosDir, "system-monitor", "initialization", "claude-code", "anomaly-check.md"),
		filepath.Join("initialization", "claude-code", "anomaly-check.md"), // Relative path as fallback
	}

	var promptContent []byte
	var err error

	// Try each path until we find the file
	for _, path := range promptPaths {
		promptContent, err = os.ReadFile(path)
		if err == nil {
			log.Printf("Loaded prompt from: %s", path)
			break
		}
	}

	if err != nil {
		return "", fmt.Errorf("could not read prompt file from any location: %v", err)
	}

	// Replace placeholders with actual values
	prompt := string(promptContent)
	prompt = strings.ReplaceAll(prompt, "{{CPU_USAGE}}", fmt.Sprintf("%.2f", cpuUsage))
	prompt = strings.ReplaceAll(prompt, "{{MEMORY_USAGE}}", fmt.Sprintf("%.2f", memoryUsage))
	prompt = strings.ReplaceAll(prompt, "{{TCP_CONNECTIONS}}", strconv.Itoa(tcpConnections))
	prompt = strings.ReplaceAll(prompt, "{{TIMESTAMP}}", timestamp)
	prompt = strings.ReplaceAll(prompt, "{{INVESTIGATION_ID}}", investigationID)
	prompt = strings.ReplaceAll(prompt, "{{API_BASE_URL}}", "http://localhost:8080")
	prompt = strings.ReplaceAll(prompt, "{{OPERATION_MODE}}", operationMode)

	// Add user note if provided
	if userNote != "" {
		prompt = strings.ReplaceAll(prompt, "{{USER_NOTE}}", userNote)
	} else {
		prompt = strings.ReplaceAll(prompt, "{{USER_NOTE}}", "No specific instructions provided.")
	}

	// Process conditional sections based on operation mode
	if operationMode == "auto-fix" {
		// Remove report-only sections and their markers
		reportOnlyStart := strings.Index(prompt, "{{#IF_REPORT_ONLY}}")
		reportOnlyEnd := strings.Index(prompt, "{{/IF_REPORT_ONLY}}")
		if reportOnlyStart != -1 && reportOnlyEnd != -1 {
			prompt = prompt[:reportOnlyStart] + prompt[reportOnlyEnd+len("{{/IF_REPORT_ONLY}}"):]
		}
		// Remove auto-fix markers but keep content
		prompt = strings.ReplaceAll(prompt, "{{#IF_AUTO_FIX}}", "")
		prompt = strings.ReplaceAll(prompt, "{{/IF_AUTO_FIX}}", "")
	} else {
		// Remove auto-fix sections and their markers
		autoFixStart := strings.Index(prompt, "{{#IF_AUTO_FIX}}")
		autoFixEnd := strings.Index(prompt, "{{/IF_AUTO_FIX}}")
		if autoFixStart != -1 && autoFixEnd != -1 {
			prompt = prompt[:autoFixStart] + prompt[autoFixEnd+len("{{/IF_AUTO_FIX}}"):]
		}
		// Remove report-only markers but keep content
		prompt = strings.ReplaceAll(prompt, "{{#IF_REPORT_ONLY}}", "")
		prompt = strings.ReplaceAll(prompt, "{{/IF_REPORT_ONLY}}", "")
	}

	return prompt, nil
}

// Helper function to update investigation fields
func updateInvestigationField(id string, field string, value interface{}) {
	investigationsMutex.Lock()
	defer investigationsMutex.Unlock()

	if inv, exists := investigations[id]; exists {
		switch field {
		case "Status":
			inv.Status = value.(string)
		case "Findings":
			inv.Findings = value.(string)
		case "Progress":
			inv.Progress = value.(int)
		case "EndTime":
			inv.EndTime = value.(time.Time)
		}
	}
}

// New handler functions for investigation updates
func getInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	investigationsMutex.RLock()
	inv, exists := investigations[id]
	investigationsMutex.RUnlock()

	if !exists {
		http.Error(w, "Investigation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inv)
}

func updateInvestigationStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updateInvestigationField(id, "Status", req.Status)

	if req.Status == "completed" || req.Status == "failed" {
		updateInvestigationField(id, "EndTime", time.Now())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func updateInvestigationFindingsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Findings string                 `json:"findings"`
		Details  map[string]interface{} `json:"details,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	investigationsMutex.Lock()
	if inv, exists := investigations[id]; exists {
		inv.Findings = req.Findings
		if req.Details != nil {
			for k, v := range req.Details {
				inv.Details[k] = v
			}
		}
	}
	investigationsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func updateInvestigationProgressHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Progress int `json:"progress"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updateInvestigationField(id, "Progress", req.Progress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func addInvestigationStepHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var step InvestigationStep
	if err := json.NewDecoder(r.Body).Decode(&step); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	step.StartTime = time.Now()

	investigationsMutex.Lock()
	if inv, exists := investigations[id]; exists {
		inv.Steps = append(inv.Steps, step)
	}
	investigationsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "step_added"})
}

func listInvestigationScriptsHandler(w http.ResponseWriter, r *http.Request) {
	scripts, _, err := loadInvestigationScripts()
	if err != nil {
		log.Printf("Failed to load investigation scripts: %v", err)
		http.Error(w, "Failed to load investigation scripts", http.StatusInternalServerError)
		return
	}

	ids := make([]string, 0, len(scripts))
	for id := range scripts {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool {
		a := scripts[ids[i]]
		b := scripts[ids[j]]
		an := strings.ToLower(a.Name)
		bn := strings.ToLower(b.Name)
		if an == bn {
			return ids[i] < ids[j]
		}
		return an < bn
	})

	result := make([]investigationScriptListItem, 0, len(ids))
	for _, id := range ids {
		result = append(result, manifestEntryToListItem(id, scripts[id]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"scripts": result})
}

func getInvestigationScriptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scriptID := vars["id"]

	scripts, baseDir, err := loadInvestigationScripts()
	if err != nil {
		log.Printf("Failed to load investigation scripts: %v", err)
		http.Error(w, "Failed to load investigation scripts", http.StatusInternalServerError)
		return
	}

	entry, exists := scripts[scriptID]
	if !exists {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	scriptPath := filepath.Join(baseDir, filepath.Clean(entry.File))
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		log.Printf("Failed to resolve investigation scripts directory: %v", err)
		http.Error(w, "Failed to load script", http.StatusInternalServerError)
		return
	}
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		log.Printf("Failed to resolve script path for %s: %v", scriptID, err)
		http.Error(w, "Failed to load script", http.StatusInternalServerError)
		return
	}

	prefix := absBase
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix = prefix + string(os.PathSeparator)
	}
	if absScript != absBase && !strings.HasPrefix(absScript, prefix) {
		http.Error(w, "Invalid script path", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(absScript)
	if err != nil {
		log.Printf("Failed to read script %s: %v", scriptID, err)
		http.Error(w, "Failed to read script", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"script":  manifestEntryToListItem(scriptID, entry),
		"content": string(content),
	})
}

func executeInvestigationScriptHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scriptID := vars["id"]

	var req struct {
		TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
		Args           []string `json:"args,omitempty"`
	}

	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	}

	scripts, baseDir, err := loadInvestigationScripts()
	if err != nil {
		log.Printf("Failed to load investigation scripts: %v", err)
		http.Error(w, "Failed to load investigation scripts", http.StatusInternalServerError)
		return
	}

	entry, exists := scripts[scriptID]
	if !exists {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	scriptPath := filepath.Join(baseDir, filepath.Clean(entry.File))
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		log.Printf("Failed to resolve script base path: %v", err)
		http.Error(w, "Failed to execute script", http.StatusInternalServerError)
		return
	}
	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		log.Printf("Failed to resolve script path for %s: %v", scriptID, err)
		http.Error(w, "Failed to execute script", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(absScript, absBase+string(os.PathSeparator)) && absScript != absBase {
		http.Error(w, "Invalid script path", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(absScript); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "Script file not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to stat script %s: %v", scriptID, err)
			http.Error(w, "Failed to execute script", http.StatusInternalServerError)
		}
		return
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultScriptTimeout(entry)
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmdArgs := append([]string{absScript}, req.Args...)
	cmd := exec.CommandContext(ctx, "bash", cmdArgs...)
	cmd.Dir = filepath.Dir(absScript)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	started := time.Now()
	execErr := cmd.Run()
	completed := time.Now()

	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	timedOut := ctx.Err() == context.DeadlineExceeded
	durationSeconds := completed.Sub(started).Seconds()

	response := map[string]interface{}{
		"script_id":        scriptID,
		"stdout":           stdoutBuf.String(),
		"stderr":           stderrBuf.String(),
		"exit_code":        exitCode,
		"duration_seconds": durationSeconds,
		"timed_out":        timedOut,
		"started_at":       started.Format(time.RFC3339),
		"completed_at":     completed.Format(time.RFC3339),
	}

	if execErr != nil {
		response["error"] = execErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	if timedOut {
		w.WriteHeader(http.StatusGatewayTimeout)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode script execution response: %v", err)
	}
}

func manifestEntryToListItem(id string, entry investigationScriptManifestEntry) investigationScriptListItem {
	enabled := true
	if entry.Enabled != nil {
		enabled = *entry.Enabled
	} else if strings.EqualFold(entry.SafetyLevel, "deprecated") {
		enabled = false
	}

	return investigationScriptListItem{
		ID:          id,
		Name:        entry.Name,
		Description: entry.Description,
		Category:    entry.Category,
		Author:      entry.Author,
		CreatedAt:   formatManifestTimestamp(entry.Created),
		UpdatedAt:   formatManifestTimestamp(entry.LastModified),
		Enabled:     enabled,
		Tags:        entry.Tags,
	}
}

func formatManifestTimestamp(value string) string {
	if value == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339, value); err == nil {
		return value
	}
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return value + "T00:00:00Z"
	}
	return value
}

func loadInvestigationScripts() (map[string]investigationScriptManifestEntry, string, error) {
	manifestPath, err := findInvestigationManifest()
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read manifest: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, "", fmt.Errorf("failed to parse manifest: %w", err)
	}

	scripts := make(map[string]investigationScriptManifestEntry)
	if scriptsRaw, ok := raw["scripts"]; ok {
		var manifestEntries map[string]investigationScriptManifestEntry
		if err := json.Unmarshal(scriptsRaw, &manifestEntries); err != nil {
			return nil, "", fmt.Errorf("failed to parse manifest scripts: %w", err)
		}
		for id, entry := range manifestEntries {
			if entry.Name != "" && entry.File != "" {
				scripts[id] = entry
			}
		}
	}

	for key, value := range raw {
		if _, exists := scripts[key]; exists {
			continue
		}
		switch key {
		case "scripts", "categories", "templates", "settings", "version", "description", "last_updated":
			continue
		}
		var entry investigationScriptManifestEntry
		if err := json.Unmarshal(value, &entry); err == nil {
			if entry.Name != "" && entry.File != "" {
				scripts[key] = entry
			}
		}
	}

	if len(scripts) == 0 {
		return nil, "", fmt.Errorf("no scripts found in manifest")
	}

	return scripts, filepath.Dir(manifestPath), nil
}

func findInvestigationManifest() (string, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = "/root"
		}
		vrooliRoot = filepath.Join(homeDir, "Vrooli")
	}

	candidates := []string{
		filepath.Join(vrooliRoot, "scenarios", "system-monitor", "investigations", "manifest.json"),
		filepath.Join("investigations", "manifest.json"),
	}

	for _, path := range candidates {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
	}

	return "", fmt.Errorf("investigation scripts manifest not found")
}

func defaultScriptTimeout(entry investigationScriptManifestEntry) time.Duration {
	if entry.ExecutionTimeEstimate != "" {
		if estimate := parseExecutionEstimate(entry.ExecutionTimeEstimate); estimate > 0 {
			// Add a buffer and ensure a sensible minimum window
			timeout := estimate + 15*time.Second
			if timeout < 5*time.Minute {
				return 5 * time.Minute
			}
			return timeout
		}
	}
	return 5 * time.Minute
}

func parseExecutionEstimate(value string) time.Duration {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return 0
	}
	if strings.Contains(clean, "-") {
		parts := strings.Split(clean, "-")
		clean = parts[len(parts)-1]
	}
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return 0
	}
	if !strings.HasSuffix(clean, "s") && !strings.HasSuffix(clean, "m") && !strings.HasSuffix(clean, "h") {
		clean += "s"
	}
	dur, err := time.ParseDuration(clean)
	if err != nil {
		return 0
	}
	return dur
}

// Log storage for debugging
var (
	logBuffer []LogEntry
	logMutex  sync.RWMutex
	maxLogs   = 1000 // Keep last 1000 log entries
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Source    string `json:"source"`
}

func addLogEntry(level, message, source string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Source:    source,
	}

	logBuffer = append(logBuffer, entry)
	if len(logBuffer) > maxLogs {
		logBuffer = logBuffer[1:] // Remove oldest entry
	}
}

func getErrorsHandler(w http.ResponseWriter, r *http.Request) {
	errorLogsMutex.RLock()
	defer errorLogsMutex.RUnlock()

	// Get query parameters
	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	var filteredLogs []ErrorLog
	if unreadOnly {
		for _, log := range errorLogs {
			if log.Unread {
				filteredLogs = append(filteredLogs, log)
			}
		}
	} else {
		filteredLogs = errorLogs
	}

	response := map[string]interface{}{
		"errors":       filteredLogs,
		"total_count":  len(filteredLogs),
		"unread_count": unreadErrorCount,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode errors response: %v", err)
	}
}

func markErrorsReadHandler(w http.ResponseWriter, r *http.Request) {
	errorLogsMutex.Lock()
	defer errorLogsMutex.Unlock()

	// Mark all errors as read
	for i := range errorLogs {
		errorLogs[i].Unread = false
	}
	unreadErrorCount = 0
	lastErrorReadTime = time.Now()

	response := map[string]interface{}{
		"status":  "success",
		"message": "All errors marked as read",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode mark-read response: %v", err)
	}
}

// killProcessHandler handles process termination requests
func killProcessHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pidStr := vars["pid"]

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid PID format",
		})
		return
	}

	// Kill the process with SIGKILL
	cmd := exec.Command("kill", "-9", strconv.Itoa(pid))
	if err := cmd.Run(); err != nil {
		// Check if process exists
		checkCmd := exec.Command("kill", "-0", strconv.Itoa(pid))
		if checkErr := checkCmd.Run(); checkErr != nil {
			// Process doesn't exist
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "Process not found",
			})
			return
		}

		// Process exists but couldn't be killed (permission issue)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Permission denied. Try running with elevated privileges.",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Process %d terminated successfully", pid),
	})
}

func getLogsHandler(w http.ResponseWriter, r *http.Request) {
	logMutex.RLock()
	defer logMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":      logBuffer,
		"count":     len(logBuffer),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// New handlers for enhanced metrics
func getDetailedMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Rate limiting
	clientIP := r.RemoteAddr
	if !detailedMetricsLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded. Please wait before making another request.", http.StatusTooManyRequests)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	timestamp := time.Now().Format(time.RFC3339)

	// Gather all detailed metrics
	cpuUsage := getCPUUsage()
	memoryUsage := getMemoryUsage()
	tcpStates := getTCPConnectionStates()
	topCPUProcesses := getTopProcessesByCPU(5)
	topMemoryProcesses := getTopProcessesByMemory(5)
	loadAvg := getLoadAverage()
	contextSwitches := getContextSwitches()
	fdUsed, fdMax := getSystemFileDescriptors()

	response := DetailedMetricsResponse{
		CPUDetails: struct {
			Usage           float64       `json:"usage"`
			TopProcesses    []ProcessInfo `json:"top_processes"`
			LoadAverage     []float64     `json:"load_average"`
			ContextSwitches int64         `json:"context_switches"`
			Goroutines      int           `json:"total_goroutines"`
		}{
			Usage:           cpuUsage,
			TopProcesses:    topCPUProcesses,
			LoadAverage:     loadAvg,
			ContextSwitches: contextSwitches,
			Goroutines:      getTotalGoroutines(),
		},
		MemoryDetails: struct {
			Usage          float64       `json:"usage"`
			TopProcesses   []ProcessInfo `json:"top_processes"`
			GrowthPatterns []struct {
				Process         string  `json:"process"`
				GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
				RiskLevel       string  `json:"risk_level"`
			} `json:"growth_patterns"`
			SwapUsage struct {
				Used    int64   `json:"used"`
				Total   int64   `json:"total"`
				Percent float64 `json:"percent"`
			} `json:"swap_usage"`
			DiskUsage struct {
				Used    int64   `json:"used"`
				Total   int64   `json:"total"`
				Percent float64 `json:"percent"`
			} `json:"disk_usage"`
		}{
			Usage:          memoryUsage,
			TopProcesses:   topMemoryProcesses,
			GrowthPatterns: getMemoryGrowthPatterns(),
			SwapUsage:      getSwapUsage(),
			DiskUsage:      getDiskUsage(),
		},
		NetworkDetails: struct {
			TCPStates TCPConnectionStates `json:"tcp_states"`
			PortUsage struct {
				Used  int `json:"used"`
				Total int `json:"total"`
			} `json:"port_usage"`
			NetworkStats    NetworkStats         `json:"network_stats"`
			ConnectionPools []ConnectionPoolInfo `json:"connection_pools"`
		}{
			TCPStates:       tcpStates,
			PortUsage:       getPortUsage(),
			NetworkStats:    getNetworkStats(),
			ConnectionPools: getHTTPConnectionPools(),
		},
		SystemDetails: SystemHealthDetails{
			FileDescriptors: struct {
				Used    int     `json:"used"`
				Max     int     `json:"max"`
				Percent float64 `json:"percent"`
			}{
				Used:    fdUsed,
				Max:     fdMax,
				Percent: float64(fdUsed) / float64(fdMax) * 100,
			},
			ServiceDependencies: checkServiceDependencies(),
			Certificates:        checkCertificates(),
		},
		Timestamp: timestamp,
	}

	json.NewEncoder(w).Encode(response)
}

func getProcessMonitorHandler(w http.ResponseWriter, r *http.Request) {
	// Rate limiting
	clientIP := r.RemoteAddr
	if !processMetricsLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded. Please wait before making another request.", http.StatusTooManyRequests)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	zombieProcesses := getZombieProcesses()
	highThreadProcesses := getHighThreadCountProcesses()
	leakCandidates := getResourceLeakCandidates()
	resourceMatrix := getTopProcessesByCPU(10) // Top 10 for resource matrix

	response := ProcessMonitorResponse{
		ProcessHealth: struct {
			TotalProcesses  int           `json:"total_processes"`
			ZombieProcesses []ProcessInfo `json:"zombie_processes"`
			HighThreadCount []ProcessInfo `json:"high_thread_count"`
			LeakCandidates  []ProcessInfo `json:"leak_candidates"`
		}{
			TotalProcesses:  getTotalProcessCount(),
			ZombieProcesses: zombieProcesses,
			HighThreadCount: highThreadProcesses,
			LeakCandidates:  leakCandidates,
		},
		ResourceMatrix: resourceMatrix,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

func getInfrastructureMonitorHandler(w http.ResponseWriter, r *http.Request) {
	// Rate limiting
	clientIP := r.RemoteAddr
	if !infraMetricsLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded. Please wait before making another request.", http.StatusTooManyRequests)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := InfrastructureMonitorResponse{
		DatabasePools:   getDatabasePools(),
		HTTPClientPools: getHTTPConnectionPools(),
		MessageQueues: struct {
			RedisPubSub struct {
				Subscribers int `json:"subscribers"`
				Channels    int `json:"channels"`
			} `json:"redis_pubsub"`
			BackgroundJobs struct {
				Pending int `json:"pending"`
				Active  int `json:"active"`
				Failed  int `json:"failed"`
			} `json:"background_jobs"`
		}{
			RedisPubSub:    getRedisPubSubStats(),
			BackgroundJobs: getBackgroundJobStats(),
		},
		StorageIO: getStorageIOStats(),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

func getDiskDetailsHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	if !detailedMetricsLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded. Please wait before making another request.", http.StatusTooManyRequests)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	partitionsRaw, err := collectors.GetDiskPartitions()
	if err != nil {
		log.Printf("Failed to gather disk partitions: %v", err)
	}

	partitions := make([]models.DiskPartitionInfo, 0, len(partitionsRaw))
	mountLookup := make(map[string]struct{})
	for _, raw := range partitionsRaw {
		partition := models.DiskPartitionInfo{
			Device:         toString(raw["device"]),
			MountPoint:     toString(raw["mount_point"]),
			SizeBytes:      toInt64(raw["size_bytes"]),
			SizeHuman:      toString(raw["size_human"]),
			UsedBytes:      toInt64(raw["used_bytes"]),
			UsedHuman:      toString(raw["used_human"]),
			AvailableBytes: toInt64(raw["available_bytes"]),
			AvailableHuman: toString(raw["available_human"]),
			UsePercent:     toFloat64(raw["use_percent"]),
		}
		if partition.MountPoint == "" {
			partition.MountPoint = "/"
		}
		partitions = append(partitions, partition)
		mountLookup[partition.MountPoint] = struct{}{}
	}

	mount := r.URL.Query().Get("mount")
	if mount == "" {
		if _, ok := mountLookup["/"]; ok {
			mount = "/"
		} else if len(partitions) > 0 {
			mount = partitions[0].MountPoint
		} else {
			mount = "/"
		}
	} else {
		if _, ok := mountLookup[mount]; !ok {
			if len(partitions) > 0 {
				mount = partitions[0].MountPoint
			} else {
				mount = "/"
			}
		}
	}

	depth := 2
	if rawDepth := r.URL.Query().Get("depth"); rawDepth != "" {
		if parsed, err := strconv.Atoi(rawDepth); err == nil && parsed >= 1 && parsed <= 5 {
			depth = parsed
		}
	}

	limit := 8
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed >= 3 && parsed <= 25 {
			limit = parsed
		}
	}

	includeFiles := false
	if raw := r.URL.Query().Get("include_files"); raw != "" {
		includeFiles = raw == "true" || raw == "1" || strings.EqualFold(raw, "yes")
	}

	directories, dirErr := collectors.GetLargestDirectories(mount, depth, limit)
	if dirErr != nil {
		log.Printf("Failed to analyze directories for %s: %v", mount, dirErr)
	}

	var files []models.DiskUsageEntry
	if includeFiles {
		var fileErr error
		files, fileErr = collectors.GetLargestFiles(mount, limit)
		if fileErr != nil {
			log.Printf("Failed to analyze files for %s: %v", mount, fileErr)
		}
	}

	response := models.DiskDetailResponse{
		Partitions:     partitions,
		ActiveMount:    mount,
		Depth:          depth,
		TopDirectories: directories,
		LargestFiles:   files,
		Timestamp:      time.Now(),
	}

	if dirErr != nil {
		response.Notes = append(response.Notes, fmt.Sprintf("Directory scan error: %v", dirErr))
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode disk detail response: %v", err)
	}
}

func toInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return parsed
		}
	}
	return 0
}

func toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	case json.Number:
		if parsed, err := v.Float64(); err == nil {
			return parsed
		}
	}
	return 0
}

func toString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case fmt.Stringer:
		return v.String()
	case json.Number:
		return v.String()
	}
	return fmt.Sprintf("%v", value)
}

// Additional helper functions for detailed metrics
func getTotalGoroutines() int {
	// Mock implementation - in reality you'd sum up goroutines from all Go processes
	return 0
}

func getMemoryGrowthPatterns() []struct {
	Process         string  `json:"process"`
	GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
	RiskLevel       string  `json:"risk_level"`
} {
	// Mock implementation - in reality you'd track memory usage over time
	return []struct {
		Process         string  `json:"process"`
		GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
		RiskLevel       string  `json:"risk_level"`
	}{
		{Process: "scenario-api-1", GrowthMBPerHour: 15.0, RiskLevel: "medium"},
		{Process: "postgres", GrowthMBPerHour: 2.0, RiskLevel: "low"},
	}
}

func getSwapUsage() struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
} {
	cmd := exec.Command("bash", "-c", "free -b | grep Swap | awk '{print $2, $3}'")
	output, err := cmd.Output()
	if err != nil {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}

	total, _ := strconv.ParseInt(fields[0], 10, 64)
	used, _ := strconv.ParseInt(fields[1], 10, 64)
	percent := 0.0
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return struct {
		Used    int64   `json:"used"`
		Total   int64   `json:"total"`
		Percent float64 `json:"percent"`
	}{used, total, percent}
}

func getDiskUsage() struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
} {
	cmd := exec.Command("bash", "-c", "df -B1 / | tail -1 | awk '{print $2, $3}'")
	output, err := cmd.Output()
	if err != nil {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 {
		return struct {
			Used    int64   `json:"used"`
			Total   int64   `json:"total"`
			Percent float64 `json:"percent"`
		}{0, 0, 0.0}
	}

	total, _ := strconv.ParseInt(fields[0], 10, 64)
	used, _ := strconv.ParseInt(fields[1], 10, 64)
	percent := 0.0
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}

	return struct {
		Used    int64   `json:"used"`
		Total   int64   `json:"total"`
		Percent float64 `json:"percent"`
	}{used, total, percent}
}

func getPortUsage() struct {
	Used  int `json:"used"`
	Total int `json:"total"`
} {
	// Count used ephemeral ports
	cmd := exec.Command("bash", "-c", "netstat -tn 2>/dev/null | grep -E ':([3-6][0-9]{4})' | wc -l")
	output, err := cmd.Output()
	used := 0
	if err == nil {
		used, _ = strconv.Atoi(strings.TrimSpace(string(output)))
	}

	// Typical ephemeral port range is 32768-65535
	return struct {
		Used  int `json:"used"`
		Total int `json:"total"`
	}{used, 32767}
}

func getNetworkStats() NetworkStats {
	// Mock implementation - in reality you'd read from /proc/net/dev
	return NetworkStats{
		BandwidthInMbps:  12.5,
		BandwidthOutMbps: 8.2,
		PacketLoss:       0.1,
		DNSSuccessRate:   99.2,
		DNSLatencyMs:     15.0,
	}
}

func getHTTPConnectionPools() []ConnectionPoolInfo {
	// Mock implementation - in reality you'd query application metrics
	return []ConnectionPoolInfo{
		{
			Name:     "scenario-api-1->ollama",
			Active:   3,
			Idle:     7,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
		{
			Name:     "scenario-api-2->qdrant",
			Active:   6,
			Idle:     4,
			MaxSize:  12,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
	}
}

func checkServiceDependencies() []ServiceHealth {
	// Check common services
	services := []struct {
		name     string
		endpoint string
	}{
		{"postgres", "localhost:5432"},
		{"redis", "localhost:6379"},
		{"qdrant", "localhost:6333"},
		{"ollama", "localhost:11434"},
	}

	var results []ServiceHealth
	for _, service := range services {
		status := "healthy"
		latency := 0.0

		// Quick TCP connection test
		start := time.Now()
		cmd := exec.Command("timeout", "2", "bash", "-c",
			fmt.Sprintf("echo > /dev/tcp/%s", service.endpoint))
		err := cmd.Run()
		latency = float64(time.Since(start).Milliseconds())

		if err != nil {
			status = "unhealthy"
			latency = -1
		}

		results = append(results, ServiceHealth{
			Name:      service.name,
			Status:    status,
			LatencyMs: latency,
			LastCheck: time.Now().Format(time.RFC3339),
			Endpoint:  service.endpoint,
		})
	}

	return results
}

func checkCertificates() []CertificateInfo {
	// Mock implementation - in reality you'd check actual certificates
	return []CertificateInfo{
		{
			Domain:       "api.local",
			DaysToExpiry: 89,
			Status:       "valid",
		},
	}
}

func getZombieProcesses() []ProcessInfo {
	// Find zombie processes
	cmd := exec.Command("bash", "-c", "ps -eo pid,comm,stat | grep ' Z' | head -10")
	output, err := cmd.Output()
	if err != nil {
		return []ProcessInfo{}
	}

	var zombies []ProcessInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid, _ := strconv.Atoi(fields[0])
			zombies = append(zombies, ProcessInfo{
				PID:    pid,
				Name:   fields[1],
				Status: "zombie",
			})
		}
	}

	return zombies
}

func getHighThreadCountProcesses() []ProcessInfo {
	// Find processes with high thread counts
	cmd := exec.Command("bash", "-c", "ps -eo pid,comm,nlwp --sort=-nlwp --no-headers | head -5")
	output, err := cmd.Output()
	if err != nil {
		return []ProcessInfo{}
	}

	var processes []ProcessInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, _ := strconv.Atoi(fields[0])
			threads, _ := strconv.Atoi(fields[2])

			if threads > 20 { // Only include processes with >20 threads
				processes = append(processes, ProcessInfo{
					PID:     pid,
					Name:    fields[1],
					Threads: threads,
					Status:  "high_threads",
				})
			}
		}
	}

	return processes
}

func getResourceLeakCandidates() []ProcessInfo {
	// Mock implementation - in reality you'd analyze FD growth, memory growth, etc.
	return []ProcessInfo{
		{
			PID:    1234,
			Name:   "scenario-api-1",
			Status: "fd_leak_risk",
			FDs:    512,
		},
	}
}

func getTotalProcessCount() int {
	cmd := exec.Command("bash", "-c", "ps -e --no-headers | wc -l")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

func getDatabasePools() []ConnectionPoolInfo {
	// Mock implementation - in reality you'd query database metrics
	return []ConnectionPoolInfo{
		{
			Name:     "postgres-main",
			Active:   8,
			Idle:     2,
			MaxSize:  10,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
		{
			Name:     "redis-main",
			Active:   45,
			Idle:     55,
			MaxSize:  100,
			Waiting:  0,
			Healthy:  true,
			LeakRisk: "low",
		},
	}
}

func getRedisPubSubStats() struct {
	Subscribers int `json:"subscribers"`
	Channels    int `json:"channels"`
} {
	// Mock implementation
	return struct {
		Subscribers int `json:"subscribers"`
		Channels    int `json:"channels"`
	}{12, 5}
}

func getBackgroundJobStats() struct {
	Pending int `json:"pending"`
	Active  int `json:"active"`
	Failed  int `json:"failed"`
} {
	// Mock implementation
	return struct {
		Pending int `json:"pending"`
		Active  int `json:"active"`
		Failed  int `json:"failed"`
	}{3, 1, 0}
}

func getStorageIOStats() struct {
	DiskQueueDepth float64 `json:"disk_queue_depth"`
	IOWaitPercent  float64 `json:"io_wait_percent"`
	ReadMBPerSec   float64 `json:"read_mb_per_sec"`
	WriteMBPerSec  float64 `json:"write_mb_per_sec"`
} {
	// Get I/O wait from /proc/stat
	cmd := exec.Command("bash", "-c", "grep '^cpu ' /proc/stat | awk '{print ($5/($2+$3+$4+$5+$6+$7+$8))*100}'")
	output, err := cmd.Output()
	iowait := 0.0
	if err == nil {
		iowait, _ = strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	}

	return struct {
		DiskQueueDepth float64 `json:"disk_queue_depth"`
		IOWaitPercent  float64 `json:"io_wait_percent"`
		ReadMBPerSec   float64 `json:"read_mb_per_sec"`
		WriteMBPerSec  float64 `json:"write_mb_per_sec"`
	}{0.2, iowait, 15.0, 8.0}
}

// New handler functions using MonitoringProcessor

func thresholdMonitorHandler(w http.ResponseWriter, r *http.Request) {
	var req ThresholdMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	result, err := monitoringProcessor.MonitorThresholds(ctx, req)
	if err != nil {
		log.Printf("Error monitoring thresholds: %v", err)
		http.Error(w, "Failed to monitor thresholds", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func anomalyInvestigationHandler(w http.ResponseWriter, r *http.Request) {
	var req AnomalyInvestigationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	result, err := monitoringProcessor.InvestigateAnomaly(ctx, req)
	if err != nil {
		log.Printf("Error investigating anomaly: %v", err)
		http.Error(w, "Failed to investigate anomaly", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func systemReportHandler(w http.ResponseWriter, r *http.Request) {
	var req ReportGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	result, err := monitoringProcessor.GenerateReport(ctx, req)
	if err != nil {
		log.Printf("Error generating report: %v", err)
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
