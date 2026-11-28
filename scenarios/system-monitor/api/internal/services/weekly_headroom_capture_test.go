package services

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/memory"
)

type capturedWeeklyTimeline struct {
	CapturedAt        time.Time               `json:"captured_at"`
	WindowStart       time.Time               `json:"window_start"`
	WindowEnd         time.Time               `json:"window_end"`
	ReturnedGroups    int                     `json:"returned_groups"`
	TopLimit          int                     `json:"top_limit"`
	PossiblyTruncated bool                    `json:"possibly_truncated"`
	Entries           []retainedTimelineEntry `json:"entries"`
}

type capturedWeeklyRepository struct {
	*memory.MemoryRepository
	entries []repository.ProcessTimelineEntry
}

func (r *capturedWeeklyRepository) QueryProcessTimeline(context.Context, repository.ProcessTimelineQuery) ([]repository.ProcessTimelineEntry, error) {
	return r.entries, nil
}

type partialWeeklyOwnerRow struct {
	Owner                 string   `json:"owner"`
	ProcessGroups         int      `json:"process_groups"`
	Samples               int64    `json:"samples"`
	CoreHours             float64  `json:"core_hours"`
	MeanRSSGiBProcessSum  float64  `json:"mean_rss_gib_process_sum"`
	MeanSwapGiBProcessSum *float64 `json:"mean_swap_gib_process_sum"`
	PeakObservedVRAMGiB   float64  `json:"peak_observed_vram_gib"`
}

type partialWeeklyHeadroomReport struct {
	Status                string                  `json:"status"`
	WindowStart           time.Time               `json:"window_start"`
	WindowEnd             time.Time               `json:"window_end"`
	SampleCoverageStart   time.Time               `json:"sample_coverage_start"`
	SampleCoverageEnd     time.Time               `json:"sample_coverage_end"`
	TimeCoverageComplete  bool                    `json:"time_coverage_complete"`
	AttributionComplete   bool                    `json:"attribution_complete"`
	PossiblyTruncated     bool                    `json:"possibly_truncated"`
	ReturnedProcessGroups int                     `json:"returned_process_groups"`
	MissingSourceFields   []string                `json:"missing_source_fields"`
	Owners                []partialWeeklyOwnerRow `json:"owners"`
}

// This opt-in report applies the production aggregation to a bounded weekly
// source snapshot captured from the live API. It marks fields absent from the
// serving API explicitly instead of interpreting them as zero.
func TestGenerateWeeklyHeadroomFromCapturedLiveTimeline(t *testing.T) {
	if os.Getenv("LM16_RUN_WEEKLY_HEADROOM") != "1" {
		t.Skip("set LM16_RUN_WEEKLY_HEADROOM=1 and LM16_WEEKLY_SOURCE_FILE to report the captured live window")
	}
	path := os.Getenv("LM16_WEEKLY_SOURCE_FILE")
	if path == "" {
		t.Fatal("LM16_WEEKLY_SOURCE_FILE is required")
	}
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read weekly timeline source: %v", err)
	}
	var snapshot capturedWeeklyTimeline
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		t.Fatalf("decode weekly timeline source: %v", err)
	}
	if snapshot.ReturnedGroups != len(snapshot.Entries) || snapshot.PossiblyTruncated || snapshot.TopLimit != repository.MaxProcessTimelineGroups {
		t.Fatalf("weekly source count/cap = %d/%d truncated=%v, entries=%d", snapshot.ReturnedGroups, snapshot.TopLimit, snapshot.PossiblyTruncated, len(snapshot.Entries))
	}
	entries := make([]repository.ProcessTimelineEntry, 0, len(snapshot.Entries))
	unknownGroups := 0
	for _, row := range snapshot.Entries {
		if row.Owner == "" || row.Owner == "unknown" {
			unknownGroups++
		}
		entries = append(entries, repository.ProcessTimelineEntry{
			Owner: row.Owner, Comm: row.Comm, PID: row.PID, Aggregated: row.Aggregated,
			CPUPct: row.CPUPct, CPUSeconds: row.CPUSeconds, MaxCPUPct: row.MaxCPUPct,
			RSSKB: row.RSSKB, GPUVRAMMB: row.GPUVRAMMB, SampleCount: row.SampleCount,
			FirstSeen: row.FirstSeen, LastSeen: row.LastSeen,
		})
	}
	repo := &capturedWeeklyRepository{MemoryRepository: memory.NewRepository(), entries: entries}
	service := NewReportService(&config.Config{}, repo, WithReportClock(NewStubClock(snapshot.WindowEnd)))
	resources, err := service.ownerResourceReport(context.Background(), repository.TimeRange{
		StartTime: snapshot.WindowStart, EndTime: snapshot.WindowEnd,
	})
	if err != nil {
		t.Fatalf("aggregate weekly owner report: %v", err)
	}
	result := partialWeeklyHeadroomReport{
		Status: "partial", WindowStart: resources.WindowStart, WindowEnd: resources.WindowEnd,
		SampleCoverageStart: resources.SampleCoverageStart, SampleCoverageEnd: resources.SampleCoverageEnd,
		TimeCoverageComplete: resources.Complete, AttributionComplete: resources.AttributionComplete,
		PossiblyTruncated: snapshot.PossiblyTruncated, ReturnedProcessGroups: resources.ReturnedProcessGroups,
		MissingSourceFields: []string{"swap_kb", "per-process GPU inventory completeness", "per-process disk I/O"},
		Owners: make([]partialWeeklyOwnerRow, 0, len(resources.Owners)),
	}
	if unknownGroups > 0 {
		result.AttributionComplete = false
	}
	for _, owner := range resources.Owners {
		result.Owners = append(result.Owners, partialWeeklyOwnerRow{
			Owner: owner.Owner, ProcessGroups: owner.ProcessGroups, Samples: owner.Samples,
			CoreHours: owner.CoreHours, MeanRSSGiBProcessSum: owner.MeanRSSGiBProcessSum,
			MeanSwapGiBProcessSum: nil, PeakObservedVRAMGiB: owner.PeakSingleProcessVRAMGiB,
		})
	}
	encodedResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("encode weekly owner report: %v", err)
	}
	t.Logf("WEEKLY_OWNER_HEADROOM_REPORT=%s", encodedResult)
	if outputPath := os.Getenv("LM16_WEEKLY_REPORT_OUT"); outputPath != "" {
		if err := os.WriteFile(outputPath, encodedResult, 0o600); err != nil {
			t.Fatalf("write weekly report artifact: %v", err)
		}
	}
}
