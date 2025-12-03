package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deployment-manager/shared"
)

// Summary holds telemetry summary data for a scenario.
type Summary struct {
	Scenario       string                   `json:"scenario"`
	Path           string                   `json:"path"`
	TotalEvents    int                      `json:"total_events"`
	LastEvent      string                   `json:"last_event,omitempty"`
	LastTimestamp  string                   `json:"last_timestamp,omitempty"`
	FailureCounts  map[string]int           `json:"failure_counts,omitempty"`
	RecentFailures []map[string]interface{} `json:"recent_failures,omitempty"`
	RecentEvents   []map[string]interface{} `json:"recent_events,omitempty"`
}

// Handler handles telemetry requests.
type Handler struct {
	log func(string, map[string]interface{})
}

// NewHandler creates a new telemetry handler.
func NewHandler(log func(string, map[string]interface{})) *Handler {
	return &Handler{log: log}
}

func telemetryDir() string {
	return shared.GetConfigResolver().ResolveTelemetryDir()
}

// Upload handles telemetry upload requests.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Parse events from request body (supports JSON and JSONL formats)
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	events, req, err := ParseEventsFromRequest(r.Body, contentType, 4<<20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(events) == 0 {
		http.Error(w, "no telemetry events provided", http.StatusBadRequest)
		return
	}

	// Resolve parameters from query string and request body
	params := ResolveUploadParams(r.URL.Query(), req)

	// Enrich events with default fields
	now := shared.GetTimeProvider().Now().UTC().Format(time.RFC3339)
	EnrichEvents(events, params, now)

	// Ensure telemetry directory exists
	dir := telemetryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, fmt.Sprintf("prepare telemetry dir: %s", err), http.StatusInternalServerError)
		return
	}

	// Append events to scenario-specific file
	path := filepath.Join(dir, fmt.Sprintf("%s.jsonl", params.Scenario))
	if err := appendEventsToFile(path, events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"status":          "ok",
		"events_ingested": len(events),
		"scenario":        params.Scenario,
		"path":            path,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// List handles telemetry list requests.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	dir := telemetryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]Summary{})
			return
		}
		http.Error(w, fmt.Sprintf("read telemetry dir: %s", err), http.StatusInternalServerError)
		return
	}

	summaries := []Summary{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		summary, err := summarizeTelemetryFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summaries)
}

func summarizeTelemetryFile(path string) (Summary, error) {
	file, err := os.Open(path)
	if err != nil {
		return Summary{}, err
	}
	defer file.Close()

	scenario := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	summary := Summary{
		Scenario:       scenario,
		Path:           path,
		FailureCounts:  map[string]int{},
		RecentFailures: []map[string]interface{}{},
		RecentEvents:   []map[string]interface{}{},
	}

	const maxRecentEvents = 5
	const maxRecentFailures = 5

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*64), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		summary.TotalEvents++
		if ts, ok := evt["timestamp"].(string); ok {
			summary.LastTimestamp = ts
		}
		if ev, ok := evt["event"].(string); ok {
			summary.LastEvent = ev
			if IsFailureEvent(ev) {
				summary.FailureCounts[ev]++
				if len(summary.RecentFailures) < maxRecentFailures {
					summary.RecentFailures = append(summary.RecentFailures, evt)
				}
			}
		}

		// Keep the most recent N events using a sliding window
		summary.RecentEvents = append(summary.RecentEvents, evt)
		if len(summary.RecentEvents) > maxRecentEvents {
			summary.RecentEvents = summary.RecentEvents[1:] // drop oldest
		}
	}

	return summary, nil
}

// appendEventsToFile writes events as JSONL to the specified file, creating it if needed.
func appendEventsToFile(path string, events []map[string]interface{}) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open telemetry file: %w", err)
	}
	defer file.Close()

	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("write telemetry: %w", err)
		}
	}
	return nil
}
