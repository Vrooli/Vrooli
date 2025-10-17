package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// SystemMonitor handles all monitoring activities
type SystemMonitor struct {
	db              *sql.DB
	ctx             context.Context
	cancel          context.CancelFunc
	collectors      map[string]MetricCollector
	anomalyDetector *AnomalyDetector
	alertManager    *AlertManager
	mu              sync.RWMutex
	httpClient      *http.Client
}

// MetricCollector interface for different metric types
type MetricCollector interface {
	Collect() (*MetricData, error)
	GetName() string
}

// MetricData represents collected metrics
type MetricData struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Values    map[string]interface{} `json:"values"`
}

// AnomalyDetector detects anomalies in metrics
type AnomalyDetector struct {
	thresholds map[string]Threshold
	history    map[string][]float64
	mu         sync.RWMutex
}

// Threshold defines limits for anomaly detection
type Threshold struct {
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	StdDevs  float64 `json:"std_devs"`
	Duration int     `json:"duration_seconds"`
}

// AlertManager handles alert generation and routing
type AlertManager struct {
	db         *sql.DB
	httpClient *http.Client
	webhookURL string
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor(db *sql.DB) *SystemMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	sm := &SystemMonitor{
		db:              db,
		ctx:             ctx,
		cancel:          cancel,
		collectors:      make(map[string]MetricCollector),
		anomalyDetector: NewAnomalyDetector(),
		alertManager:    NewAlertManager(db),
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
	
	// Register collectors
	sm.RegisterCollector("cpu", &CPUCollector{})
	sm.RegisterCollector("memory", &MemoryCollector{})
	sm.RegisterCollector("disk", &DiskCollector{})
	sm.RegisterCollector("network", &NetworkCollector{})
	sm.RegisterCollector("process", &ProcessCollector{})
	sm.RegisterCollector("scenario", &ScenarioCollector{db: db})
	
	return sm
}

// Start begins monitoring
func (sm *SystemMonitor) Start() error {
	log.Println("Starting system monitor...")
	
	// Start metric collection
	go sm.collectionLoop()
	
	// Start anomaly detection
	go sm.anomalyDetectionLoop()
	
	// Start scheduled reports
	go sm.scheduledReportsLoop()
	
	// Start threshold monitoring
	go sm.thresholdMonitoringLoop()
	
	log.Println("System monitor started")
	return nil
}

// Stop gracefully shuts down the monitor
func (sm *SystemMonitor) Stop() {
	log.Println("Stopping system monitor...")
	sm.cancel()
	log.Println("System monitor stopped")
}

// RegisterCollector adds a new metric collector
func (sm *SystemMonitor) RegisterCollector(name string, collector MetricCollector) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.collectors[name] = collector
}

// collectionLoop continuously collects metrics
func (sm *SystemMonitor) collectionLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.collectAllMetrics()
		}
	}
}

// collectAllMetrics collects metrics from all collectors
func (sm *SystemMonitor) collectAllMetrics() {
	sm.mu.RLock()
	collectors := make(map[string]MetricCollector)
	for k, v := range sm.collectors {
		collectors[k] = v
	}
	sm.mu.RUnlock()
	
	var wg sync.WaitGroup
	for name, collector := range collectors {
		wg.Add(1)
		go func(n string, c MetricCollector) {
			defer wg.Done()
			
			data, err := c.Collect()
			if err != nil {
				log.Printf("Error collecting %s metrics: %v", n, err)
				return
			}
			
			// Store metrics
			sm.storeMetrics(n, data)
			
			// Check for anomalies
			sm.anomalyDetector.Check(n, data)
		}(name, collector)
	}
	wg.Wait()
}

// storeMetrics stores collected metrics in the database
func (sm *SystemMonitor) storeMetrics(collectorName string, data *MetricData) {
	jsonData, _ := json.Marshal(data.Values)
	
	query := `
		INSERT INTO metrics (collector, timestamp, data)
		VALUES ($1, $2, $3)
	`
	
	if _, err := sm.db.Exec(query, collectorName, data.Timestamp, jsonData); err != nil {
		log.Printf("Failed to store metrics: %v", err)
	}
}

// anomalyDetectionLoop runs anomaly detection
func (sm *SystemMonitor) anomalyDetectionLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.detectAnomalies()
		}
	}
}

// detectAnomalies analyzes recent metrics for anomalies
func (sm *SystemMonitor) detectAnomalies() {
	// Query recent metrics
	query := `
		SELECT collector, timestamp, data
		FROM metrics
		WHERE timestamp > NOW() - INTERVAL '5 minutes'
		ORDER BY timestamp DESC
	`
	
	rows, err := sm.db.Query(query)
	if err != nil {
		log.Printf("Failed to query metrics for anomaly detection: %v", err)
		return
	}
	defer rows.Close()
	
	// Analyze metrics
	for rows.Next() {
		var collector string
		var timestamp time.Time
		var data json.RawMessage
		
		if err := rows.Scan(&collector, &timestamp, &data); err != nil {
			continue
		}
		
		// Check for anomalies
		if anomaly := sm.anomalyDetector.Analyze(collector, data); anomaly != nil {
			sm.handleAnomaly(anomaly)
		}
	}
}

// handleAnomaly processes detected anomalies
func (sm *SystemMonitor) handleAnomaly(anomaly *Anomaly) {
	log.Printf("Anomaly detected: %s - %s", anomaly.Type, anomaly.Description)
	
	// Store anomaly
	query := `
		INSERT INTO anomalies (type, severity, description, metric_data, detected_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	
	metricData, _ := json.Marshal(anomaly.MetricData)
	sm.db.Exec(query, anomaly.Type, anomaly.Severity, anomaly.Description, metricData)
	
	// Send alert if severity is high
	if anomaly.Severity >= 3 {
		sm.alertManager.SendAlert(anomaly)
	}
}

// scheduledReportsLoop handles scheduled report generation
func (sm *SystemMonitor) scheduledReportsLoop() {
	// Run hourly reports
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.generateReport("hourly")
		}
	}
}

// generateReport creates a system health report
func (sm *SystemMonitor) generateReport(reportType string) {
	log.Printf("Generating %s report", reportType)
	
	report := &SystemReport{
		Type:      reportType,
		Timestamp: time.Now(),
		Metrics:   make(map[string]interface{}),
	}
	
	// Collect summary statistics
	query := `
		SELECT 
			collector,
			COUNT(*) as count,
			AVG((data->>'cpu_usage')::float) as avg_cpu,
			MAX((data->>'cpu_usage')::float) as max_cpu
		FROM metrics
		WHERE timestamp > NOW() - INTERVAL '1 hour'
		GROUP BY collector
	`
	
	rows, err := sm.db.Query(query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var collector string
			var count int
			var avgCPU, maxCPU sql.NullFloat64
			
			if err := rows.Scan(&collector, &count, &avgCPU, &maxCPU); err == nil {
				report.Metrics[collector] = map[string]interface{}{
					"count":   count,
					"avg_cpu": avgCPU.Float64,
					"max_cpu": maxCPU.Float64,
				}
			}
		}
	}
	
	// Store report
	reportData, _ := json.Marshal(report)
	sm.db.Exec(
		"INSERT INTO reports (type, timestamp, data) VALUES ($1, $2, $3)",
		reportType, report.Timestamp, reportData,
	)
	
	// Send report notification
	sm.alertManager.SendReport(report)
}

// thresholdMonitoringLoop monitors metrics against thresholds
func (sm *SystemMonitor) thresholdMonitoringLoop() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-sm.ctx.Done():
			return
		case <-ticker.C:
			sm.checkThresholds()
		}
	}
}

// checkThresholds checks current metrics against configured thresholds
func (sm *SystemMonitor) checkThresholds() {
	// Get current metrics
	metrics := sm.getCurrentMetrics()
	
	// Check each threshold
	for metric, value := range metrics {
		threshold := sm.anomalyDetector.GetThreshold(metric)
		if threshold == nil {
			continue
		}
		
		if value < threshold.Min || value > threshold.Max {
			// Threshold violation
			sm.handleThresholdViolation(metric, value, threshold)
		}
	}
}

// getCurrentMetrics gets the latest metric values
func (sm *SystemMonitor) getCurrentMetrics() map[string]float64 {
	metrics := make(map[string]float64)
	
	// CPU usage
	if cpuPercent, err := cpu.Percent(0, false); err == nil && len(cpuPercent) > 0 {
		metrics["cpu_usage"] = cpuPercent[0]
	}
	
	// Memory usage
	if vmStat, err := mem.VirtualMemory(); err == nil {
		metrics["memory_usage"] = vmStat.UsedPercent
	}
	
	// Disk usage
	if diskStat, err := disk.Usage("/"); err == nil {
		metrics["disk_usage"] = diskStat.UsedPercent
	}
	
	return metrics
}

// handleThresholdViolation handles threshold violations
func (sm *SystemMonitor) handleThresholdViolation(metric string, value float64, threshold *Threshold) {
	log.Printf("Threshold violation: %s = %.2f (min: %.2f, max: %.2f)", 
		metric, value, threshold.Min, threshold.Max)
	
	// Create alert
	alert := &Alert{
		Type:        "threshold_violation",
		Severity:    2,
		Message:     fmt.Sprintf("%s threshold violated: %.2f", metric, value),
		MetricName:  metric,
		MetricValue: value,
		Threshold:   threshold,
		Timestamp:   time.Now(),
	}
	
	sm.alertManager.SendAlert(&Anomaly{
		Type:        alert.Type,
		Severity:    alert.Severity,
		Description: alert.Message,
		MetricData:  map[string]interface{}{"value": value, "threshold": threshold},
	})
}

// Collector implementations

type CPUCollector struct{}

func (c *CPUCollector) Collect() (*MetricData, error) {
	cpuPercent, err := cpu.Percent(1*time.Second, false)
	if err != nil {
		return nil, err
	}
	
	cpuInfo, _ := cpu.Info()
	
	return &MetricData{
		Timestamp: time.Now(),
		Type:      "cpu",
		Values: map[string]interface{}{
			"usage_percent": cpuPercent[0],
			"cores":         runtime.NumCPU(),
			"model":         cpuInfo[0].ModelName,
		},
	}, nil
}

func (c *CPUCollector) GetName() string { return "cpu" }

type MemoryCollector struct{}

func (c *MemoryCollector) Collect() (*MetricData, error) {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	
	return &MetricData{
		Timestamp: time.Now(),
		Type:      "memory",
		Values: map[string]interface{}{
			"total":         vmStat.Total,
			"used":          vmStat.Used,
			"available":     vmStat.Available,
			"used_percent":  vmStat.UsedPercent,
			"cached":        vmStat.Cached,
			"buffers":       vmStat.Buffers,
		},
	}, nil
}

func (c *MemoryCollector) GetName() string { return "memory" }

type DiskCollector struct{}

func (c *DiskCollector) Collect() (*MetricData, error) {
	diskStat, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}
	
	ioCounters, _ := disk.IOCounters()
	
	return &MetricData{
		Timestamp: time.Now(),
		Type:      "disk",
		Values: map[string]interface{}{
			"total":        diskStat.Total,
			"used":         diskStat.Used,
			"free":         diskStat.Free,
			"used_percent": diskStat.UsedPercent,
			"io_counters":  ioCounters,
		},
	}, nil
}

func (c *DiskCollector) GetName() string { return "disk" }

type NetworkCollector struct{}

func (c *NetworkCollector) Collect() (*MetricData, error) {
	netStats, err := net.IOCounters(false)
	if err != nil {
		return nil, err
	}
	
	if len(netStats) == 0 {
		return nil, fmt.Errorf("no network stats available")
	}
	
	stat := netStats[0]
	return &MetricData{
		Timestamp: time.Now(),
		Type:      "network",
		Values: map[string]interface{}{
			"bytes_sent":   stat.BytesSent,
			"bytes_recv":   stat.BytesRecv,
			"packets_sent": stat.PacketsSent,
			"packets_recv": stat.PacketsRecv,
			"errors_in":    stat.Errin,
			"errors_out":   stat.Errout,
		},
	}, nil
}

func (c *NetworkCollector) GetName() string { return "network" }

type ProcessCollector struct{}

func (c *ProcessCollector) Collect() (*MetricData, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, err
	}
	
	topProcesses := make([]map[string]interface{}, 0, 10)
	for i, p := range processes {
		if i >= 10 {
			break
		}
		
		name, _ := p.Name()
		cpuPercent, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		
		topProcesses = append(topProcesses, map[string]interface{}{
			"pid":         p.Pid,
			"name":        name,
			"cpu_percent": cpuPercent,
			"memory_rss":  memInfo.RSS,
		})
	}
	
	return &MetricData{
		Timestamp: time.Now(),
		Type:      "process",
		Values: map[string]interface{}{
			"total_count":   len(processes),
			"top_processes": topProcesses,
		},
	}, nil
}

func (c *ProcessCollector) GetName() string { return "process" }

type ScenarioCollector struct {
	db *sql.DB
}

func (c *ScenarioCollector) Collect() (*MetricData, error) {
	// Collect Vrooli scenario metrics
	query := `
		SELECT COUNT(*) as total,
		       COUNT(CASE WHEN status = 'running' THEN 1 END) as running,
		       COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM scenarios
	`
	
	var total, running, failed int
	err := c.db.QueryRow(query).Scan(&total, &running, &failed)
	if err != nil {
		return nil, err
	}
	
	return &MetricData{
		Timestamp: time.Now(),
		Type:      "scenario",
		Values: map[string]interface{}{
			"total":   total,
			"running": running,
			"failed":  failed,
		},
	}, nil
}

func (c *ScenarioCollector) GetName() string { return "scenario" }

// Anomaly detection implementation

type Anomaly struct {
	Type        string                 `json:"type"`
	Severity    int                    `json:"severity"`
	Description string                 `json:"description"`
	MetricData  map[string]interface{} `json:"metric_data"`
	DetectedAt  time.Time              `json:"detected_at"`
}

func NewAnomalyDetector() *AnomalyDetector {
	ad := &AnomalyDetector{
		thresholds: make(map[string]Threshold),
		history:    make(map[string][]float64),
	}
	
	// Set default thresholds
	ad.thresholds["cpu_usage"] = Threshold{Min: 0, Max: 80, StdDevs: 2}
	ad.thresholds["memory_usage"] = Threshold{Min: 0, Max: 90, StdDevs: 2}
	ad.thresholds["disk_usage"] = Threshold{Min: 0, Max: 85, StdDevs: 2}
	
	return ad
}

func (ad *AnomalyDetector) Check(collector string, data *MetricData) {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	// Add to history
	if _, ok := ad.history[collector]; !ok {
		ad.history[collector] = make([]float64, 0, 100)
	}
	
	// Extract numeric value for analysis
	if val, ok := data.Values["usage_percent"].(float64); ok {
		ad.history[collector] = append(ad.history[collector], val)
		
		// Keep only last 100 values
		if len(ad.history[collector]) > 100 {
			ad.history[collector] = ad.history[collector][1:]
		}
	}
}

func (ad *AnomalyDetector) Analyze(collector string, data json.RawMessage) *Anomaly {
	// Statistical anomaly detection
	var values map[string]interface{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil
	}
	
	// Check for sudden spikes
	if usage, ok := values["cpu_usage"].(float64); ok {
		if usage > 95 {
			return &Anomaly{
				Type:        "cpu_spike",
				Severity:    3,
				Description: fmt.Sprintf("CPU usage spike detected: %.2f%%", usage),
				MetricData:  values,
				DetectedAt:  time.Now(),
			}
		}
	}
	
	return nil
}

func (ad *AnomalyDetector) GetThreshold(metric string) *Threshold {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	
	if threshold, ok := ad.thresholds[metric]; ok {
		return &threshold
	}
	return nil
}

// Alert management

type Alert struct {
	Type        string     `json:"type"`
	Severity    int        `json:"severity"`
	Message     string     `json:"message"`
	MetricName  string     `json:"metric_name"`
	MetricValue float64    `json:"metric_value"`
	Threshold   *Threshold `json:"threshold"`
	Timestamp   time.Time  `json:"timestamp"`
}

type SystemReport struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

func NewAlertManager(db *sql.DB) *AlertManager {
	return &AlertManager{
		db:         db,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		webhookURL: os.Getenv("ALERT_WEBHOOK_URL"),
	}
}

func (am *AlertManager) SendAlert(anomaly *Anomaly) {
	// Log alert
	log.Printf("ALERT: %s (severity: %d) - %s", anomaly.Type, anomaly.Severity, anomaly.Description)
	
	// Store in database
	data, _ := json.Marshal(anomaly)
	am.db.Exec(
		"INSERT INTO alerts (type, severity, data, created_at) VALUES ($1, $2, $3, NOW())",
		anomaly.Type, anomaly.Severity, data,
	)
	
	// Send webhook if configured
	if am.webhookURL != "" {
		go am.sendWebhook(anomaly)
	}
}

func (am *AlertManager) SendReport(report *SystemReport) {
	log.Printf("Report generated: %s", report.Type)
	
	// Send to configured endpoints
	if am.webhookURL != "" {
		data, _ := json.Marshal(report)
		am.httpClient.Post(am.webhookURL, "application/json", bytes.NewBuffer(data))
	}
}

func (am *AlertManager) sendWebhook(anomaly *Anomaly) {
	payload, _ := json.Marshal(map[string]interface{}{
		"type":        "system_alert",
		"anomaly":     anomaly,
		"timestamp":   time.Now(),
		"system_name": "vrooli-system-monitor",
	})
	
	am.httpClient.Post(am.webhookURL, "application/json", bytes.NewBuffer(payload))
}

// Helper functions

func calculateStdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	
	// Calculate variance
	var variance float64
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	
	return math.Sqrt(variance)
}