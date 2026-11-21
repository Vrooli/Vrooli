// Package chaos contains deliberately bounded writers used by the recovery
// proof. It is not a production cleanup provider: callers must supply a
// governed temporary root and must keep the returned report with the test
// evidence.
package chaos

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	chunkSize       = 1 << 20
	defaultDuration = 8 * time.Minute
	maxDuration     = 10 * time.Minute
	maxRate         = int64(100) << 30 // 100 GiB/hour
)

// Config describes one synthetic runaway writer. Root is intentionally not
// resolved or broadened here; the caller owns governed-root validation.
type Config struct {
	Root         string
	BytesPerHour int64
	Duration     time.Duration
	FileName     string
	OpenFile     func(string, int, os.FileMode) (io.WriteCloser, error)
	Sleep        func(context.Context, time.Duration) error
	Now          func() time.Time
}

// Report is the durable-friendly result of a writer run.
type Report struct {
	Root          string        `json:"root"`
	Path          string        `json:"path"`
	RequestedRate int64         `json:"requested_bytes_per_hour"`
	Duration      time.Duration `json:"duration"`
	BytesWritten  int64         `json:"bytes_written"`
	StartedAt     time.Time     `json:"started_at"`
	FinishedAt    time.Time     `json:"finished_at"`
}

// Run writes real data blocks at approximately the requested rate. It stops
// on context cancellation or when Duration elapses, and rejects unbounded or
// unexpectedly broad inputs before opening a file.
func Run(ctx context.Context, cfg Config) (Report, error) {
	if strings.TrimSpace(cfg.Root) == "" || !filepath.IsAbs(cfg.Root) {
		return Report{}, errors.New("chaos writer: Root must be an absolute path")
	}
	if cfg.BytesPerHour <= 0 || cfg.BytesPerHour > maxRate {
		return Report{}, fmt.Errorf("chaos writer: BytesPerHour must be within (0,%d]", maxRate)
	}
	if cfg.Duration <= 0 {
		cfg.Duration = defaultDuration
	}
	if cfg.Duration > maxDuration {
		return Report{}, fmt.Errorf("chaos writer: Duration must not exceed %s", maxDuration)
	}
	if cfg.FileName == "" {
		cfg.FileName = ".vrooli-chaos-writer"
	}
	if filepath.Base(cfg.FileName) != cfg.FileName || cfg.FileName == "." || cfg.FileName == ".." {
		return Report{}, errors.New("chaos writer: FileName must be a single file name")
	}
	if cfg.OpenFile == nil {
		cfg.OpenFile = func(path string, flags int, mode os.FileMode) (io.WriteCloser, error) {
			return os.OpenFile(path, flags, mode)
		}
	}
	if cfg.Sleep == nil {
		cfg.Sleep = sleepContext
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	if err := os.MkdirAll(cfg.Root, 0o700); err != nil {
		return Report{}, fmt.Errorf("chaos writer: create root: %w", err)
	}
	path := filepath.Join(cfg.Root, cfg.FileName)
	f, err := cfg.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return Report{}, fmt.Errorf("chaos writer: open %s: %w", path, err)
	}
	defer f.Close()

	started := cfg.Now()
	deadline := started.Add(cfg.Duration)
	interval := time.Duration(float64(time.Hour*chunkSize) / float64(cfg.BytesPerHour))
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	buf := make([]byte, chunkSize)
	report := Report{Root: cfg.Root, Path: path, RequestedRate: cfg.BytesPerHour, StartedAt: started}
	for {
		if err := ctx.Err(); err != nil {
			return finish(report, cfg.Now()), err
		}
		if !cfg.Now().Before(deadline) {
			break
		}
		written, err := f.Write(buf)
		if err != nil {
			return finish(report, cfg.Now()), fmt.Errorf("chaos writer: write: %w", err)
		}
		report.BytesWritten += int64(written)
		if err := cfg.Sleep(ctx, interval); err != nil {
			return finish(report, cfg.Now()), err
		}
	}
	return finish(report, cfg.Now()), nil
}

func finish(report Report, at time.Time) Report {
	report.FinishedAt = at
	report.Duration = at.Sub(report.StartedAt)
	return report
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
