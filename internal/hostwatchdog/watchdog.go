// Package hostwatchdog implements the platform-neutral floor watchdog.
//
// It owns only observation and the typed hand-off to storage-manager. It does
// not execute cleanup commands or elevate privileges.
package hostwatchdog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Mount      string
	FloorBytes uint64
	Sustain    time.Duration
	StatePath  string
	Now        func() time.Time
	// FreeSpace is injectable so the floor state machine can be validated
	// without manufacturing disk pressure. Production uses the platform
	// syscall implementation when this is nil.
	FreeSpace      func(string) (uint64, float64, error)
	ReportPressure func(context.Context, Report) error
}

type Report struct {
	Mount          string  `json:"mount"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedPercent    float64 `json:"used_percent"`
	BelowFloor     bool    `json:"below_floor"`
	Sustained      bool    `json:"sustained"`
}

type state struct {
	FirstBelow time.Time `json:"first_below"`
}

func Tick(ctx context.Context, cfg Config) (Report, error) {
	if strings.TrimSpace(cfg.Mount) == "" {
		cfg.Mount = "/"
	}
	if cfg.FloorBytes == 0 {
		return Report{}, fmt.Errorf("floor bytes must be positive")
	}
	if cfg.Sustain <= 0 {
		cfg.Sustain = 120 * time.Second
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	spaceReader := cfg.FreeSpace
	if spaceReader == nil {
		spaceReader = freeSpace
	}
	available, used, err := spaceReader(cfg.Mount)
	if err != nil {
		return Report{}, err
	}
	r := Report{Mount: cfg.Mount, AvailableBytes: available, UsedPercent: used, BelowFloor: available < cfg.FloorBytes}
	var st state
	if cfg.StatePath != "" {
		if raw, readErr := os.ReadFile(cfg.StatePath); readErr == nil {
			_ = json.Unmarshal(raw, &st)
		}
	}
	if r.BelowFloor {
		if st.FirstBelow.IsZero() {
			st.FirstBelow = cfg.Now().UTC()
		}
		r.Sustained = !cfg.Now().UTC().Before(st.FirstBelow.Add(cfg.Sustain))
	} else {
		st.FirstBelow = time.Time{}
	}
	if cfg.StatePath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.StatePath), 0o700); err != nil {
			return r, err
		}
		data, _ := json.Marshal(st)
		if err := os.WriteFile(cfg.StatePath, data, 0o600); err != nil {
			return r, err
		}
	}
	if r.Sustained && cfg.ReportPressure != nil {
		if err := cfg.ReportPressure(ctx, r); err != nil {
			return r, err
		}
	}
	return r, nil
}
