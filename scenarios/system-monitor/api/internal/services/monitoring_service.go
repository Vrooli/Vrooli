package services

import (
	"context"
	"log"
	"sync"
	"time"
)

// MonitoringService manages the continuous monitoring loops
type MonitoringService struct {
	settingsManager *SettingsManager
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
	mu              sync.RWMutex
	
	// Monitoring functions
	thresholdChecker   func(ctx context.Context) error
	anomalyDetector    func(ctx context.Context) error
	reportGenerator    func(ctx context.Context) error
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(settingsManager *SettingsManager) *MonitoringService {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MonitoringService{
		settingsManager: settingsManager,
		ctx:             ctx,
		cancel:          cancel,
		running:         false,
	}
}

// SetThresholdChecker sets the function to call for threshold checking
func (ms *MonitoringService) SetThresholdChecker(checker func(ctx context.Context) error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.thresholdChecker = checker
}

// SetAnomalyDetector sets the function to call for anomaly detection
func (ms *MonitoringService) SetAnomalyDetector(detector func(ctx context.Context) error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.anomalyDetector = detector
}

// SetReportGenerator sets the function to call for report generation
func (ms *MonitoringService) SetReportGenerator(generator func(ctx context.Context) error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.reportGenerator = generator
}

// Start begins the monitoring loops
func (ms *MonitoringService) Start() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if ms.running {
		return nil // Already running
	}
	
	ms.running = true
	
	// Start threshold monitoring loop
	ms.wg.Add(1)
	go ms.thresholdMonitoringLoop()
	
	// Start anomaly detection loop
	ms.wg.Add(1)
	go ms.anomalyDetectionLoop()
	
	// Start report generation loop
	ms.wg.Add(1)
	go ms.reportGenerationLoop()
	
	log.Println("🟢 Monitoring service started - loops will run when system is active")
	return nil
}

// Stop gracefully shuts down the monitoring service
func (ms *MonitoringService) Stop() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	if !ms.running {
		return // Already stopped
	}
	
	log.Println("🔴 Stopping monitoring service...")
	ms.running = false
	ms.cancel()
	
	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		ms.wg.Wait()
		close(done)
	}()
	
	// Wait for graceful shutdown or timeout
	select {
	case <-done:
		log.Println("✅ Monitoring service stopped gracefully")
	case <-time.After(30 * time.Second):
		log.Println("⚠️ Monitoring service shutdown timed out")
	}
}

// thresholdMonitoringLoop runs threshold checking based on settings
func (ms *MonitoringService) thresholdMonitoringLoop() {
	defer ms.wg.Done()
	
	for {
		// Get current settings
		settings := ms.settingsManager.GetSettings()
		interval := time.Duration(settings.ThresholdCheckInterval) * time.Second
		
		// Create timer
		timer := time.NewTimer(interval)
		
		select {
		case <-ms.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Only check thresholds if system is active
			if settings.Active && ms.thresholdChecker != nil {
				if err := ms.thresholdChecker(ms.ctx); err != nil {
					log.Printf("❌ Threshold check error: %v", err)
				}
			}
		}
		
		timer.Stop()
	}
}

// anomalyDetectionLoop runs anomaly detection based on settings
func (ms *MonitoringService) anomalyDetectionLoop() {
	defer ms.wg.Done()
	
	for {
		// Get current settings
		settings := ms.settingsManager.GetSettings()
		interval := time.Duration(settings.AnomalyDetectionInterval) * time.Second
		
		// Create timer
		timer := time.NewTimer(interval)
		
		select {
		case <-ms.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Only detect anomalies if system is active
			if settings.Active && ms.anomalyDetector != nil {
				if err := ms.anomalyDetector(ms.ctx); err != nil {
					log.Printf("❌ Anomaly detection error: %v", err)
				}
			}
		}
		
		timer.Stop()
	}
}

// reportGenerationLoop runs report generation (less frequent)
func (ms *MonitoringService) reportGenerationLoop() {
	defer ms.wg.Done()
	
	// Reports run every hour regardless of metric collection interval
	const reportInterval = 1 * time.Hour
	
	for {
		timer := time.NewTimer(reportInterval)
		
		select {
		case <-ms.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			// Only generate reports if system is active
			settings := ms.settingsManager.GetSettings()
			if settings.Active && ms.reportGenerator != nil {
				if err := ms.reportGenerator(ms.ctx); err != nil {
					log.Printf("❌ Report generation error: %v", err)
				}
			}
		}
		
		timer.Stop()
	}
}

// IsRunning returns whether the monitoring service is running
func (ms *MonitoringService) IsRunning() bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.running
}

// GetStatus returns the current status of the monitoring service
func (ms *MonitoringService) GetStatus() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	settings := ms.settingsManager.GetSettings()
	
	return map[string]interface{}{
		"service_running":              ms.running,
		"system_active":               settings.Active,
		"threshold_check_interval":    settings.ThresholdCheckInterval,
		"anomaly_detection_interval":  settings.AnomalyDetectionInterval,
		"metric_collection_interval":  settings.MetricCollectionInterval,
		"monitoring_loops_active":     ms.running && settings.Active,
	}
}