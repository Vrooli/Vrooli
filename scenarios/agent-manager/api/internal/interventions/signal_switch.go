package interventions

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/usagemode"
)

// SignalSwitchFile is the operator's runtime off switch for operator signals.
// It lives beside the usage mode file so it is machine-local state, and it is
// read on every pass, so turning signals off or on needs no restart.
const signalSwitchName = "operator-signals.json"

// SignalSwitch is the stored operator choice.
type SignalSwitch struct {
	Enabled   bool      `json:"enabled"`
	ChangedAt time.Time `json:"changed_at"`
	ChangedBy string    `json:"changed_by"`
}

// SignalSwitchPath resolves the switch file next to the usage mode file.
func SignalSwitchPath() string {
	path := usagemode.Path()
	if path == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(path), signalSwitchName)
}

// Sources of the signal state.
const (
	SignalSourceEnvironment = "environment"
	SignalSourceOperator    = "operator"
	SignalSourceDefault     = "default"
)

// SignalState reports whether operator signals are on and what decides it. The
// environment switch (SignalsSwitchEnv) set to off always wins, so a deployment
// can force them off. Otherwise the operator's stored choice applies. A switch
// file that exists but cannot be read counts as off: this is a privacy control,
// so it fails closed, unlike the usage mode file.
func SignalState() (enabled bool, source string, stored SignalSwitch) {
	if envOff() {
		return false, SignalSourceEnvironment, SignalSwitch{}
	}
	path := SignalSwitchPath()
	if path == "" {
		return true, SignalSourceDefault, SignalSwitch{}
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return true, SignalSourceDefault, SignalSwitch{}
	}
	if err != nil || json.Unmarshal(raw, &stored) != nil {
		return false, SignalSourceOperator, SignalSwitch{}
	}
	return stored.Enabled, SignalSourceOperator, stored
}

func envOff() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(SignalsSwitchEnv))) {
	case "off", "0", "false", "no", "disabled":
		return true
	}
	return false
}

// SignalsEnabled reports whether operator signals are switched on. It reads the
// environment and the operator's stored switch on every call.
func SignalsEnabled() bool {
	enabled, _, _ := SignalState()
	return enabled
}

// SignalControl is the operator surface over the switch and the stored labels.
type SignalControl struct {
	Store *Store
	Now   func() time.Time
}

func (c SignalControl) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

// SignalStatus is what the status command shows.
type SignalStatus struct {
	Enabled   bool
	Source    string
	ChangedAt time.Time
	ChangedBy string
	Stored    int64
	Note      string
}

// Status reads the switch and counts stored signal rows.
func (c SignalControl) Status(ctx context.Context) (SignalStatus, error) {
	enabled, source, stored := SignalState()
	status := SignalStatus{Enabled: enabled, Source: source, ChangedAt: stored.ChangedAt, ChangedBy: stored.ChangedBy}
	switch {
	case !enabled && source == SignalSourceEnvironment:
		status.Note = SignalsSwitchEnv + " forces signals off; `on` cannot override it"
	case !enabled && stored.ChangedBy == "":
		status.Note = "the switch file is unreadable, so signals stay off until `on` rewrites it"
	}
	if c.Store != nil {
		count, err := c.Store.CountSignals(ctx)
		if err != nil {
			return status, err
		}
		status.Stored = count
	}
	return status, nil
}

// Set writes the operator's choice. Turning signals on while the environment
// forces them off records the choice and reports the environment as the cause.
func (c SignalControl) Set(enabled bool, by string) error {
	path := SignalSwitchPath()
	if path == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(SignalSwitch{Enabled: enabled, ChangedAt: c.now(), ChangedBy: by}, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".signals-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if err := tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(append(raw, '\n'))
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Rename(name, path)
}

// Purge deletes every stored signal and returns how many rows went.
func (c SignalControl) Purge(ctx context.Context) (int64, error) {
	if c.Store == nil {
		return 0, errors.New("operator signal store is unavailable")
	}
	return c.Store.PurgeSignals(ctx)
}
