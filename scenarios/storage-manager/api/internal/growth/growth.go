// Package growth turns persisted census samples into attributed velocity.
// It never walks the filesystem: the census writer owns observation and this
// package owns the read-side fit and projection contract.
package growth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/vrooli/api-core/database"
)

// Sample is one observed byte count in a time series.
type Sample struct {
	ObservedAt time.Time
	Bytes      int64
}

// Point is retained as a readable compatibility alias for callers that use
// the older growth vocabulary.
type Point = Sample

var ErrInsufficientSamples = errors.New("growth: at least three samples are required")

type Fit struct {
	InterceptBytes    float64   `json:"intercept_bytes"`
	SlopeBytesPerHour float64   `json:"slope_bytes_per_hour"`
	R2                float64   `json:"r_squared"`
	RSquared          float64   `json:"-"`
	SampleCount       int       `json:"sample_count"`
	Confidence        string    `json:"confidence"`
	WindowStart       time.Time `json:"window_start,omitempty"`
	WindowEnd         time.Time `json:"window_end,omitempty"`
}

type Projection struct {
	Kind           string  `json:"kind"`
	LimitBytes     int64   `json:"limit_bytes"`
	HoursRemaining float64 `json:"hours_remaining"`
}

type OwnerGrowth struct {
	OwnerKind         string   `json:"owner_kind,omitempty"`
	OwnerID           string   `json:"owner_id"`
	EntryName         string   `json:"entry_name"`
	CurrentBytes      int64    `json:"current_bytes"`
	SlopeBytesPerHour float64  `json:"slope_bytes_per_hour"`
	R2                float64  `json:"r_squared"`
	SampleCount       int      `json:"sample_count"`
	Confidence        string   `json:"confidence"`
	CeilingBytes      *int64   `json:"ceiling_bytes,omitempty"`
	HoursToCeiling    *float64 `json:"hours_to_ceiling,omitempty"`
	CeilingStatus     string   `json:"ceiling_status"`
}

type DeviceGrowth struct {
	Fit
	CurrentBytes   int64    `json:"current_bytes"`
	CapacityBytes  int64    `json:"capacity_bytes"`
	AvailableBytes int64    `json:"available_bytes"`
	HoursToFull    *float64 `json:"hours_to_full,omitempty"`
	DaysToFull     *float64 `json:"days_to_full,omitempty"`
}

type Report struct {
	Root        string        `json:"root"`
	Window      string        `json:"window"`
	WindowHours float64       `json:"window_hours"`
	Device      DeviceGrowth  `json:"device"`
	Owners      []OwnerGrowth `json:"owners"`
}

type Store struct{ db *database.RoutedDB }

func NewStore(db *database.RoutedDB) *Store { return &Store{db: db} }

// Analyze is the read-side growth analysis entry point. Build is retained as
// a compatibility name for callers that predate the public contract wording.
func (s *Store) Analyze(ctx context.Context, root string, window time.Duration, ceilings map[string]int64) (Report, error) {
	return s.Build(ctx, root, window, ceilings)
}

// FitSamples performs ordinary least squares on hours since the first sample.
// A two-point difference is deliberately not a fit: it cannot expose noise or
// report a meaningful goodness-of-fit value.
func FitSamples(samples []Sample) (Fit, error) {
	if len(samples) < 3 {
		return Fit{SampleCount: len(samples), Confidence: "insufficient_samples"}, ErrInsufficientSamples
	}
	fit := fitSeries(samples, 3)
	return fit, nil
}

// FitSeries performs ordinary least squares on hours since the first sample.
// Three points are the minimum because a two-point difference is not a trend.
func FitSeries(points []Point) Fit {
	fit, _ := FitSamples(points)
	return fit
}

func ProjectToCeiling(current, maxBytes int64, fit Fit) (Projection, bool) {
	if fit.SlopeBytesPerHour <= 0 || current >= maxBytes {
		return Projection{}, false
	}
	return Projection{Kind: "ceiling", LimitBytes: maxBytes, HoursRemaining: float64(maxBytes-current) / fit.SlopeBytesPerHour}, true
}

func ProjectToFull(available int64, fit Fit) (Projection, bool) {
	if available <= 0 || fit.SlopeBytesPerHour <= 0 {
		return Projection{}, false
	}
	return Projection{Kind: "device_full", LimitBytes: available, HoursRemaining: float64(available) / fit.SlopeBytesPerHour}, true
}

func fitSeries(points []Sample, minimum int) Fit {
	fit := Fit{SampleCount: len(points), Confidence: "insufficient_samples"}
	if len(points) < minimum {
		return fit
	}
	meanX, meanY := 0.0, 0.0
	for _, point := range points {
		meanX += point.ObservedAt.Sub(points[0].ObservedAt).Hours()
		meanY += float64(point.Bytes)
	}
	meanX /= float64(len(points))
	meanY /= float64(len(points))
	var numerator, denominator, totalVariance float64
	for _, point := range points {
		x := point.ObservedAt.Sub(points[0].ObservedAt).Hours() - meanX
		y := float64(point.Bytes) - meanY
		numerator += x * y
		denominator += x * x
		totalVariance += y * y
	}
	if denominator == 0 {
		return fit
	}
	fit.SlopeBytesPerHour = numerator / denominator
	fit.InterceptBytes = meanY - fit.SlopeBytesPerHour*meanX
	fit.WindowStart = points[0].ObservedAt
	fit.WindowEnd = points[len(points)-1].ObservedAt
	if totalVariance == 0 {
		fit.R2 = 1
	} else {
		var residual float64
		for _, point := range points {
			x := point.ObservedAt.Sub(points[0].ObservedAt).Hours()
			predicted := meanY + fit.SlopeBytesPerHour*(x-meanX)
			residual += math.Pow(float64(point.Bytes)-predicted, 2)
		}
		fit.R2 = 1 - residual/totalVariance
		if fit.R2 < 0 {
			fit.R2 = 0
		}
	}
	fit.RSquared = fit.R2
	switch {
	case fit.R2 >= 0.8:
		fit.Confidence = "high"
	case fit.R2 >= 0.5:
		fit.Confidence = "medium"
	default:
		fit.Confidence = "low"
	}
	return fit
}

func (s *Store) Build(ctx context.Context, root string, window time.Duration, ceilings map[string]int64) (Report, error) {
	if s == nil || s.db == nil {
		return Report{}, fmt.Errorf("growth store is not configured")
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	cutoff := time.Now().UTC().Add(-window).Format(time.RFC3339Nano)
	rows, err := s.db.QueryContext(ctx, `SELECT owner_kind, owner_id, entry_name, observed_at, bytes FROM census_entry_samples WHERE root = ? AND observed_at >= ? ORDER BY owner_kind, owner_id, entry_name, observed_at`, root, cutoff)
	if err != nil {
		return Report{}, fmt.Errorf("load growth samples: %w", err)
	}
	defer func() { _ = rows.Close() }()
	type key struct{ kind, owner, entry string }
	series := map[key][]Point{}
	for rows.Next() {
		var kind, owner, entry, raw string
		var bytes int64
		if err := rows.Scan(&kind, &owner, &entry, &raw, &bytes); err != nil {
			return Report{}, err
		}
		at, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return Report{}, fmt.Errorf("parse growth sample time: %w", err)
		}
		series[key{kind, owner, entry}] = append(series[key{kind, owner, entry}], Point{ObservedAt: at, Bytes: bytes})
	}
	if err := rows.Err(); err != nil {
		return Report{}, err
	}
	out := Report{Root: root, Window: window.String(), WindowHours: window.Hours(), Owners: make([]OwnerGrowth, 0, len(series))}
	for k, points := range series {
		fit := FitSeries(points)
		row := OwnerGrowth{OwnerKind: k.kind, OwnerID: k.owner, EntryName: k.entry, CurrentBytes: points[len(points)-1].Bytes, SlopeBytesPerHour: fit.SlopeBytesPerHour, R2: fit.R2, SampleCount: fit.SampleCount, Confidence: fit.Confidence, CeilingStatus: "unbounded"}
		ceilingKey := k.kind + "/" + k.owner + "/" + k.entry
		if max, ok := ceilings[ceilingKey]; ok {
			row.CeilingBytes = &max
			row.CeilingStatus = "binding"
			if row.CurrentBytes >= max {
				row.CeilingStatus = "over_ceiling"
			} else if fit.SlopeBytesPerHour <= 0 {
				row.CeilingStatus = "non_binding"
			} else if projection, ok := ProjectToCeiling(row.CurrentBytes, max, fit); ok {
				row.HoursToCeiling = &projection.HoursRemaining
			}
		}
		out.Owners = append(out.Owners, row)
	}
	device, err := s.deviceGrowth(ctx, root, cutoff)
	if err != nil {
		return Report{}, err
	}
	out.Device = device
	sort.Slice(out.Owners, func(i, j int) bool {
		if out.Owners[i].SlopeBytesPerHour != out.Owners[j].SlopeBytesPerHour {
			return out.Owners[i].SlopeBytesPerHour > out.Owners[j].SlopeBytesPerHour
		}
		if out.Owners[i].OwnerID != out.Owners[j].OwnerID {
			return out.Owners[i].OwnerID < out.Owners[j].OwnerID
		}
		return out.Owners[i].EntryName < out.Owners[j].EntryName
	})
	return out, nil
}

func (s *Store) deviceGrowth(ctx context.Context, root, cutoff string) (DeviceGrowth, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT observed_at, measured_bytes, report_json FROM census_snapshots WHERE root = ? AND observed_at >= ? ORDER BY observed_at ASC`, root, cutoff)
	if err != nil {
		return DeviceGrowth{}, fmt.Errorf("load device growth samples: %w", err)
	}
	defer func() { _ = rows.Close() }()
	points := make([]Point, 0)
	var capacity int64
	var latest int64
	for rows.Next() {
		var raw, payload string
		var bytes int64
		if err := rows.Scan(&raw, &bytes, &payload); err != nil {
			return DeviceGrowth{}, err
		}
		at, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return DeviceGrowth{}, err
		}
		points = append(points, Point{ObservedAt: at, Bytes: bytes})
		var envelope struct {
			ScanCoverage struct {
				DeviceTotalBytes int64 `json:"device_total_bytes"`
			} `json:"scan_coverage"`
		}
		if json.Unmarshal([]byte(payload), &envelope) == nil && envelope.ScanCoverage.DeviceTotalBytes > 0 {
			capacity = envelope.ScanCoverage.DeviceTotalBytes
		}
		latest = bytes
	}
	if err := rows.Err(); err != nil {
		return DeviceGrowth{}, err
	}
	// Device capacity is the safety projection used by infra-health. Require a
	// wider sample than an owner trend so one noisy statfs observation cannot
	// move the time-to-full alarm.
	fit := fitSeries(points, 6)
	device := DeviceGrowth{Fit: fit, CurrentBytes: latest, CapacityBytes: capacity}
	if capacity > latest {
		device.AvailableBytes = capacity - latest
	}
	if projection, ok := ProjectToFull(device.AvailableBytes, fit); ok {
		hours := projection.HoursRemaining
		days := hours / 24
		device.HoursToFull = &hours
		device.DaysToFull = &days
	}
	return device, nil
}
