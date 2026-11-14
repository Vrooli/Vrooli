package growth

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestFitSeriesUsesWindowedLeastSquares(t *testing.T) {
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	points := []Point{
		{ObservedAt: base, Bytes: 100},
		{ObservedAt: base.Add(time.Hour), Bytes: 210},
		{ObservedAt: base.Add(2 * time.Hour), Bytes: 320},
		{ObservedAt: base.Add(3 * time.Hour), Bytes: 430},
	}
	fit := FitSeries(points)
	if math.Abs(fit.SlopeBytesPerHour-110) > 0.01 || fit.R2 < 0.99 || fit.SampleCount != 4 || fit.Confidence != "high" {
		t.Fatalf("fit = %#v, want 110 bytes/hour, high confidence", fit)
	}
	if got := FitSeries(points[:2]); got.Confidence != "insufficient_samples" || got.SlopeBytesPerHour != 0 {
		t.Fatalf("two-point fit = %#v, want no slope", got)
	}
}

func TestFitSamplesReturnsTypedInsufficientSamples(t *testing.T) {
	fit, err := FitSamples([]Sample{{ObservedAt: time.Now(), Bytes: 10}, {ObservedAt: time.Now().Add(time.Hour), Bytes: 20}})
	if !errors.Is(err, ErrInsufficientSamples) {
		t.Fatalf("error = %v, want ErrInsufficientSamples", err)
	}
	if fit.SlopeBytesPerHour != 0 || fit.SampleCount != 2 {
		t.Fatalf("fit = %#v, want no slope and two samples", fit)
	}
}

func TestFlatSeriesDoesNotProject(t *testing.T) {
	base := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	fit, err := FitSamples([]Sample{{base, 100}, {base.Add(time.Hour), 100}, {base.Add(2 * time.Hour), 100}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ProjectToCeiling(100, 200, fit); ok {
		t.Fatal("flat series projected a ceiling")
	}
	if _, ok := ProjectToFull(100, fit); ok {
		t.Fatal("flat series projected device full")
	}
}

type goldenSample struct {
	ObservedAt time.Time `json:"observed_at"`
	Bytes      int64     `json:"bytes"`
}

func loadGoldenSeries(t *testing.T) (map[string][]Sample, []Sample) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot resolve home directory for golden series: %v", err)
	}
	path := filepath.Join(home, ".vrooli", "plan-artifacts", "storage-growth-2026-08-25", "baseline-series.json")
	body, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skipf("golden series is absent: %s", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Entries map[string][]goldenSample `json:"entries"`
		Device  []goldenSample            `json:"device"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode golden series: %v", err)
	}
	series := make(map[string][]Sample, len(envelope.Entries))
	for key, points := range envelope.Entries {
		for _, point := range points {
			series[key] = append(series[key], Sample{ObservedAt: point.ObservedAt, Bytes: point.Bytes})
		}
	}
	device := make([]Sample, 0, len(envelope.Device))
	for _, point := range envelope.Device {
		device = append(device, Sample{ObservedAt: point.ObservedAt, Bytes: point.Bytes})
	}
	if len(device) == 0 {
		for key, points := range series {
			if strings.Contains(strings.ToLower(key), "device") {
				device = points
				break
			}
		}
	}
	if len(device) == 0 {
		t.Fatalf("golden series has no device samples")
	}
	return series, device
}

func TestGoldenSeriesReproducesRecordedGrowth(t *testing.T) {
	series, device := loadGoldenSeries(t)
	deviceFit, err := FitSamples(device)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(deviceFit.SlopeBytesPerHour/(3.87e9)-1) > 0.10 {
		t.Fatalf("device slope = %g, want within 10%% of 3.87 GB/h", deviceFit.SlopeBytesPerHour)
	}
	expectations := map[string]float64{
		"browser-automation-studio": 1.19e9,
		"go/build_cache":            1.57e9,
	}
	for owner, expected := range expectations {
		var found bool
		for key, points := range series {
			if !strings.Contains(key, owner) {
				continue
			}
			fit, fitErr := FitSamples(points)
			if fitErr != nil {
				t.Fatal(fitErr)
			}
			if math.Abs(fit.SlopeBytesPerHour/expected-1) > 0.10 {
				t.Fatalf("%s slope = %g, want within 10%% of %g", key, fit.SlopeBytesPerHour, expected)
			}
			found = true
		}
		if !found {
			t.Fatalf("golden series has no %s entry", owner)
		}
	}
}

func TestGoldenSeriesSlopeIsStableWhenOneObservationIsPerturbed(t *testing.T) {
	_, device := loadGoldenSeries(t)
	original, err := FitSamples(device)
	if err != nil {
		t.Fatal(err)
	}
	perturbed := append([]Sample(nil), device...)
	perturbed[len(perturbed)/2].Bytes += int64(float64(perturbed[len(perturbed)/2].Bytes) * 0.01)
	changed, err := FitSamples(perturbed)
	if err != nil {
		t.Fatal(err)
	}
	if delta := math.Abs(changed.SlopeBytesPerHour/original.SlopeBytesPerHour - 1); delta >= 0.25 {
		t.Fatalf("perturbed slope changed by %.2f%%, want <25%%", delta*100)
	}
}

func TestBuildRanksOwnersAndProjectsBindingCeiling(t *testing.T) {
	db := database.NewFromPrimary(dbtest.NewSQLite(t))
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE census_snapshots (id TEXT PRIMARY KEY, observed_at TEXT, root TEXT, measured_bytes INTEGER, report_json TEXT); CREATE TABLE census_entry_samples (snapshot_id TEXT, observed_at TEXT, root TEXT, owner_kind TEXT, owner_id TEXT, entry_name TEXT, bytes INTEGER, PRIMARY KEY(snapshot_id, owner_kind, owner_id, entry_name));`); err != nil {
		t.Fatal(err)
	}
	root := "/"
	base := time.Now().UTC().Add(-3 * time.Hour)
	for i, bytes := range []int64{100, 200, 300, 400, 500, 600} {
		at := base.Add(time.Duration(i) * time.Hour).Format(time.RFC3339Nano)
		id := string(rune('a' + i))
		if _, err := db.ExecContext(context.Background(), `INSERT INTO census_snapshots (id, observed_at, root, measured_bytes, report_json) VALUES (?, ?, ?, ?, ?)`, id, at, root, 1000+bytes, `{"scan_coverage":{"device_total_bytes":10000}}`); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(context.Background(), `INSERT INTO census_entry_samples VALUES (?, ?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?, ?)`, id, at, root, "scenario", "fast", "data", bytes, id, at, root, "scenario", "flat", "data", 50); err != nil {
			t.Fatal(err)
		}
	}
	store := NewStore(db)
	ceiling := int64(800)
	report, err := store.Build(context.Background(), root, 24*time.Hour, map[string]int64{"scenario/fast/data": ceiling})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Owners) != 2 || report.Owners[0].OwnerID != "fast" {
		t.Fatalf("owners = %#v, want fast first", report.Owners)
	}
	if report.Owners[0].HoursToCeiling == nil || *report.Owners[0].HoursToCeiling < 1.8 || *report.Owners[0].HoursToCeiling > 2.2 {
		t.Fatalf("projection = %#v, want about 2 hours", report.Owners[0])
	}
	if report.Device.SampleCount != 6 || report.Device.DaysToFull == nil {
		t.Fatalf("device growth = %#v, want six-sample fitted full projection", report.Device)
	}
}
