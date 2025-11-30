package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// GenerationEvent represents a single generation event for analytics
type GenerationEvent struct {
	EventID     string    `json:"event_id"`
	TemplateID  string    `json:"template_id"`
	ScenarioID  string    `json:"scenario_id,omitempty"`
	IsDryRun    bool      `json:"is_dry_run"`
	Success     bool      `json:"success"`
	ErrorReason string    `json:"error_reason,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Duration    int64     `json:"duration_ms"`
}

// AnalyticsSummary provides aggregate statistics about generation usage
type AnalyticsSummary struct {
	TotalGenerations  int               `json:"total_generations"`
	SuccessfulCount   int               `json:"successful_count"`
	FailedCount       int               `json:"failed_count"`
	DryRunCount       int               `json:"dry_run_count"`
	SuccessRate       float64           `json:"success_rate"`
	ByTemplate        map[string]int    `json:"by_template"`
	AverageDurationMs int64             `json:"average_duration_ms"`
	RecentEvents      []GenerationEvent `json:"recent_events"`
	FirstGeneration   *time.Time        `json:"first_generation,omitempty"`
	LastGeneration    *time.Time        `json:"last_generation,omitempty"`
	GenerationsByDay  map[string]int    `json:"generations_by_day,omitempty"`
}

// AnalyticsService tracks template usage and success metrics at factory level
type AnalyticsService struct {
	events    []GenerationEvent
	eventsMux sync.RWMutex
	dataDir   string
}

// NewAnalyticsService creates a new analytics service instance
func NewAnalyticsService() *AnalyticsService {
	as := &AnalyticsService{
		events:  make([]GenerationEvent, 0),
		dataDir: getAnalyticsDataDir(),
	}
	as.loadEvents()
	return as
}

func getAnalyticsDataDir() string {
	if testDir := os.Getenv("ANALYTICS_DATA_DIR"); testDir != "" {
		return testDir
	}

	execPath, err := os.Executable()
	if err != nil {
		return "/tmp/landing-manager-analytics"
	}
	execDir := filepath.Dir(execPath)
	return filepath.Join(execDir, "analytics")
}

func (as *AnalyticsService) loadEvents() {
	eventsFile := filepath.Join(as.dataDir, "events.json")
	data, err := os.ReadFile(eventsFile)
	if err != nil {
		return
	}

	var events []GenerationEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return
	}

	as.eventsMux.Lock()
	as.events = events
	as.eventsMux.Unlock()
}

func (as *AnalyticsService) saveEvents() error {
	if err := os.MkdirAll(as.dataDir, 0755); err != nil {
		return err
	}

	as.eventsMux.RLock()
	data, err := json.MarshalIndent(as.events, "", "  ")
	as.eventsMux.RUnlock()
	if err != nil {
		return err
	}

	eventsFile := filepath.Join(as.dataDir, "events.json")
	return os.WriteFile(eventsFile, data, 0644)
}

// RecordGeneration records a new generation event
func (as *AnalyticsService) RecordGeneration(templateID, scenarioID string, isDryRun, success bool, errorReason string, durationMs int64) {
	event := GenerationEvent{
		EventID:     generateEventID(),
		TemplateID:  templateID,
		ScenarioID:  scenarioID,
		IsDryRun:    isDryRun,
		Success:     success,
		ErrorReason: errorReason,
		Timestamp:   time.Now(),
		Duration:    durationMs,
	}

	as.eventsMux.Lock()
	as.events = append(as.events, event)
	as.eventsMux.Unlock()

	go func() {
		_ = as.saveEvents()
	}()
}

// GetSummary returns aggregate analytics summary
func (as *AnalyticsService) GetSummary() AnalyticsSummary {
	as.eventsMux.RLock()
	defer as.eventsMux.RUnlock()

	summary := AnalyticsSummary{
		ByTemplate:       make(map[string]int),
		GenerationsByDay: make(map[string]int),
		RecentEvents:     make([]GenerationEvent, 0),
	}

	if len(as.events) == 0 {
		return summary
	}

	var totalDuration int64
	for _, event := range as.events {
		summary.TotalGenerations++

		if event.Success {
			summary.SuccessfulCount++
		} else {
			summary.FailedCount++
		}

		if event.IsDryRun {
			summary.DryRunCount++
		}

		summary.ByTemplate[event.TemplateID]++
		totalDuration += event.Duration

		if summary.FirstGeneration == nil || event.Timestamp.Before(*summary.FirstGeneration) {
			t := event.Timestamp
			summary.FirstGeneration = &t
		}
		if summary.LastGeneration == nil || event.Timestamp.After(*summary.LastGeneration) {
			t := event.Timestamp
			summary.LastGeneration = &t
		}

		dayKey := event.Timestamp.Format("2006-01-02")
		summary.GenerationsByDay[dayKey]++
	}

	if summary.TotalGenerations > 0 {
		summary.AverageDurationMs = totalDuration / int64(summary.TotalGenerations)
		summary.SuccessRate = float64(summary.SuccessfulCount) / float64(summary.TotalGenerations) * 100
	}

	startIdx := 0
	if len(as.events) > 10 {
		startIdx = len(as.events) - 10
	}
	summary.RecentEvents = as.events[startIdx:]

	return summary
}

// GetEvents returns all recorded events
func (as *AnalyticsService) GetEvents() []GenerationEvent {
	as.eventsMux.RLock()
	defer as.eventsMux.RUnlock()

	events := make([]GenerationEvent, len(as.events))
	copy(events, as.events)
	return events
}

func generateEventID() string {
	return time.Now().Format("20060102-150405.000")
}
