package services

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

type retainedTimelineFrame struct {
	ElapsedSeconds int                     `json:"elapsed_seconds"`
	Entries        []retainedTimelineEntry `json:"entries"`
}

type retainedTimelineEntry struct {
	Owner       string    `json:"owner"`
	Comm        string    `json:"comm"`
	PID         int       `json:"pid"`
	Aggregated  bool      `json:"aggregated"`
	CPUPct      float64   `json:"cpu_pct"`
	CPUSeconds  float64   `json:"cpu_seconds"`
	MaxCPUPct   float64   `json:"max_cpu_pct"`
	RSSKB       int64     `json:"rss_kb"`
	GPUVRAMMB   float64   `json:"gpu_vram_mb"`
	SampleCount int64     `json:"sample_count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

type capturedTimelineReader struct {
	frames []retainedTimelineFrame
	next   int
}

func (r *capturedTimelineReader) GetProcessTimelineInRange(_ context.Context, _, _ time.Time, _ string, _ int, _ string) ([]repository.ProcessTimelineEntry, error) {
	if r.next >= len(r.frames) {
		return nil, context.DeadlineExceeded
	}
	frame := r.frames[r.next]
	r.next++
	entries := make([]repository.ProcessTimelineEntry, 0, len(frame.Entries))
	for _, entry := range frame.Entries {
		entries = append(entries, repository.ProcessTimelineEntry{
			Owner: entry.Owner, Comm: entry.Comm, PID: entry.PID, Aggregated: entry.Aggregated,
			CPUPct: entry.CPUPct, CPUSeconds: entry.CPUSeconds, MaxCPUPct: entry.MaxCPUPct,
			RSSKB: entry.RSSKB, GPUVRAMMB: entry.GPUVRAMMB, SampleCount: entry.SampleCount,
			FirstSeen: entry.FirstSeen, LastSeen: entry.LastSeen,
		})
	}
	return entries, nil
}

// This opt-in replay consumes per-tick rows captured from the running system-
// monitor's read-only historical timeline API. It drives the real rule and
// AlertEvaluator without touching the server's database or alert repository.
func TestReplayNovember12AgainstCapturedRetainedTimeline(t *testing.T) {
	if os.Getenv("LM16_RUN_RETAINED_REPLAY") != "1" {
		t.Skip("set LM16_RUN_RETAINED_REPLAY=1 and LM16_REPLAY_SNAPSHOT_FILE to replay retained history")
	}
	path := os.Getenv("LM16_REPLAY_SNAPSHOT_FILE")
	if path == "" {
		t.Fatal("LM16_REPLAY_SNAPSHOT_FILE is required for the opt-in retained replay")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read retained timeline snapshots: %v", err)
	}
	var frames []retainedTimelineFrame
	if err := json.Unmarshal(encoded, &frames); err != nil {
		t.Fatalf("decode retained timeline snapshots: %v", err)
	}
	if len(frames) != 37 {
		t.Fatalf("captured frames=%d, want 37 passes covering 15 minutes at 25s cadence", len(frames))
	}
	for i, frame := range frames {
		if frame.ElapsedSeconds != i*25 {
			t.Fatalf("frame %d elapsed=%d, want %d", i, frame.ElapsedSeconds, i*25)
		}
	}
	reader := &capturedTimelineReader{frames: frames}
	start := time.Date(2026, 11, 12, 22, 22, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	result, err := ReplayProcessCPUAlert(context.Background(), reader, start, end)
	if err != nil {
		t.Fatalf("replay retained Nov 12 interval: %v", err)
	}
	if len(result.Alerts) == 0 {
		t.Fatalf("retained Nov 12 replay produced no alert: %+v", result)
	}
	var target *models.Alert
	for _, alert := range result.Alerts {
		if alert.Details["owner"] == "image-tools" && alert.Details["command"] == "sd-cli" && alert.Details["rule_id"] == RuleSustainedProcessCPU {
			target = alert
			break
		}
	}
	if target == nil {
		t.Fatalf("retained replay did not flag image-tools/sd-cli: %+v", result.Alerts)
	}
	if result.PossiblyTruncated || result.RowsAtEvaluation >= result.TopLimit {
		t.Fatalf("retained replay hit process-group cap: %+v", result)
	}
	delay := target.Timestamp.Sub(start)
	if delay < 0 || delay > 15*time.Minute || result.EvaluatedUntil.After(end) {
		t.Fatalf("retained replay delay=%s evaluated_until=%s, outside the 15m window", delay, result.EvaluatedUntil)
	}
	encodedResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("encode retained replay result: %v", err)
	}
	t.Logf("RETAINED_NOV12_REPLAY delay=%s result=%s", delay, string(encodedResult))
}
