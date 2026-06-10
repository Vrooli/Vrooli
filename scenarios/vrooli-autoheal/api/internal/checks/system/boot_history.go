// Package system: boot history check.
//
// Detects unclean shutdowns by reading `journalctl --list-boots` and looking
// for missing systemd shutdown markers in the previous boot's tail. Two
// recent unclean shutdowns in 24h elevates the warning to a critical-adjacent
// signal — the operator has had at least one unexplained reset.
package system

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// shutdownMarkers are kernel/systemd messages that confirm a clean shutdown.
// Presence of any one is enough to declare the boot clean.
var shutdownMarkers = []string{
	"Reached target Shutdown",
	"systemd-shutdown",
	"Powering off",
	"Reboot Requested",
	"reboot: System halted",
	"reboot: Power down",
	"reboot: Restarting system",
}

// BootHistoryCheck reports unclean previous-boot shutdowns.
type BootHistoryCheck struct {
	reader     *journal.Reader
	now        func() time.Time
	uncleanWin time.Duration // how far back to count unclean events
}

// BootHistoryCheckOption configures a BootHistoryCheck.
type BootHistoryCheckOption func(*BootHistoryCheck)

// WithBootHistoryReader injects the journal reader (for testing).
func WithBootHistoryReader(r *journal.Reader) BootHistoryCheckOption {
	return func(c *BootHistoryCheck) { c.reader = r }
}

// WithBootHistoryNow overrides the time source (for testing).
func WithBootHistoryNow(now func() time.Time) BootHistoryCheckOption {
	return func(c *BootHistoryCheck) { c.now = now }
}

// NewBootHistoryCheck builds a BootHistoryCheck.
func NewBootHistoryCheck(opts ...BootHistoryCheckOption) *BootHistoryCheck {
	c := &BootHistoryCheck{
		reader:     journal.NewReader(checks.DefaultExecutor),
		now:        time.Now,
		uncleanWin: 24 * time.Hour,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *BootHistoryCheck) ID() string    { return "system-boot-history" }
func (c *BootHistoryCheck) Title() string { return "Boot History" }
func (c *BootHistoryCheck) Description() string {
	return "Detects unclean shutdowns by inspecting journalctl boot list and shutdown markers"
}

func (c *BootHistoryCheck) Importance() string {
	return "Repeated unclean shutdowns indicate hardware, firmware, or thermal issues — never just \"the OS rebooted\""
}
func (c *BootHistoryCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *BootHistoryCheck) IntervalSeconds() int       { return 600 }
func (c *BootHistoryCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *BootHistoryCheck) Run(ctx context.Context) checks.Result {
	r := checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	if runtime.GOOS != "linux" {
		r.Status = checks.StatusOK
		r.Message = "Boot history is Linux-only"
		return r
	}

	if !c.reader.Available(ctx) {
		r.Status = checks.StatusWarning
		r.Message = "journalctl unavailable; cannot inspect boot history"
		r.Details["journalAvailable"] = false
		return r
	}
	r.Details["journalAvailable"] = true

	boots, err := c.reader.ListBoots(ctx)
	if err != nil {
		r.Status = checks.StatusWarning
		r.Message = "Failed to list boots: " + err.Error()
		r.Details["error"] = err.Error()
		return r
	}
	r.Details["bootCount"] = len(boots)
	if len(boots) < 2 {
		r.Status = checks.StatusOK
		r.Message = "Only the current boot is recorded — no history yet"
		return r
	}

	// Count how many recent previous boots ended uncleanly.
	uncleanRecent := 0
	uncleanTotal := 0
	uncleanBootIDs := []string{}
	recentUncleanBootIDs := []string{}
	var latestUnclean journal.BootRecord
	cutoff := c.now().Add(-c.uncleanWin)
	for _, b := range boots {
		if b.Index >= 0 {
			continue // skip current boot
		}
		entries, err := c.reader.QueryLogs(ctx, journal.QueryOpts{
			Boot:    b.BootID,
			Tail:    200,
			Reverse: true,
		})
		if err != nil {
			continue
		}
		if !hasShutdownMarker(entries) {
			uncleanTotal++
			uncleanBootIDs = append(uncleanBootIDs, b.BootID)
			if latestUnclean.BootID == "" || b.LastEntry.After(latestUnclean.LastEntry) {
				latestUnclean = b
			}
			if !b.LastEntry.IsZero() && b.LastEntry.After(cutoff) {
				uncleanRecent++
				recentUncleanBootIDs = append(recentUncleanBootIDs, b.BootID)
			}
		}
	}
	r.Details["uncleanBootsTotal"] = uncleanTotal
	r.Details["uncleanBootsRecent24h"] = uncleanRecent
	r.Details["uncleanBootIds"] = uncleanBootIDs
	r.Details["recentUncleanBootIds"] = recentUncleanBootIDs
	if latestUnclean.BootID != "" {
		r.Details["latestUncleanBootId"] = latestUnclean.BootID
		r.Details["latestUncleanBootLastEntry"] = latestUnclean.LastEntry.UTC().Format(time.RFC3339Nano)
	}

	switch {
	case uncleanRecent >= 2:
		r.Status = checks.StatusCritical
		r.Message = "Multiple unclean shutdowns in last 24 hours — investigate hardware/firmware"
	case uncleanRecent == 1, uncleanTotal > 0:
		r.Status = checks.StatusWarning
		r.Message = "At least one unclean shutdown recorded — check pstore / dmesg for evidence"
	default:
		r.Status = checks.StatusOK
		r.Message = "All recorded shutdowns were clean"
	}
	r.Timestamp = c.now()
	return r
}

func hasShutdownMarker(entries []journal.LogEntry) bool {
	for _, e := range entries {
		for _, m := range shutdownMarkers {
			if strings.Contains(e.Message, m) {
				return true
			}
		}
	}
	return false
}
