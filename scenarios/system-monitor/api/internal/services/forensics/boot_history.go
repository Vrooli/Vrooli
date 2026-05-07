package forensics

import (
	"context"
	"regexp"
	"time"

	"system-monitor-api/internal/services/journal"
)

// BootEntry is one boot in the history with cleanliness annotation.
type BootEntry struct {
	Index      int       `json:"index"`
	BootID     string    `json:"bootId"`
	FirstEntry time.Time `json:"firstEntry"`
	LastEntry  time.Time `json:"lastEntry"`
	Clean      bool      `json:"clean"`
	Reason     string    `json:"reason,omitempty"` // when not clean
}

// BootHistoryReport is the data payload for /api/v1/forensics/boot-history.
type BootHistoryReport struct {
	Boots []BootEntry `json:"boots"`
}

// shutdownMarker matches lines indicating a clean shutdown sequence.
var shutdownMarker = regexp.MustCompile(`(?i)(Reached target.*Shutdown|systemd-shutdown|Shutdown initiated|Power-Off|Reboot)`)

// BootHistory returns parsed boot list with unclean-boot annotations.
func (s *Service) BootHistory(ctx context.Context) Envelope {
	const key = "boot-history"
	now := s.now()
	if cached, ok := s.cache.get(key, now); ok {
		return cached
	}

	env := s.computeBootHistory(ctx, now)
	s.cache.set(key, env, now)
	return env
}

func (s *Service) computeBootHistory(ctx context.Context, now time.Time) Envelope {
	env := Envelope{GeneratedAt: now}

	if s.journal == nil {
		env.Reason = "journal reader not configured"
		return env
	}

	boots, err := s.journal.ListBoots(ctx)
	if err != nil {
		env.Reason = "journalctl --list-boots failed: " + err.Error()
		return env
	}

	report := BootHistoryReport{Boots: []BootEntry{}}
	for _, b := range boots {
		entry := BootEntry{
			Index:      b.Index,
			BootID:     b.BootID,
			FirstEntry: b.FirstEntry,
			LastEntry:  b.LastEntry,
		}
		// Current boot (index 0) is always "in progress" — mark clean to avoid
		// false positives on the most recent entry.
		if b.Index == 0 {
			entry.Clean = true
			report.Boots = append(report.Boots, entry)
			continue
		}
		clean, reason := s.bootIsClean(ctx, b.BootID)
		entry.Clean = clean
		if !clean {
			entry.Reason = reason
		}
		report.Boots = append(report.Boots, entry)
	}

	env.Available = true
	env.Data = report
	return env
}

// bootIsClean returns true if the journal for the given boot contains a
// shutdown marker. Errors degrade to "unknown" (treated as clean to avoid
// false alarms).
func (s *Service) bootIsClean(ctx context.Context, bootID string) (bool, string) {
	if bootID == "" {
		return true, ""
	}
	entries, err := s.journal.QueryLogs(ctx, journal.QueryOpts{
		Boot:    bootID,
		Grep:    `Reached target.*Shutdown|systemd-shutdown|Shutdown initiated|Power-Off|Reboot`,
		Tail:    5,
		Reverse: true,
	})
	if err != nil {
		return true, ""
	}
	for _, e := range entries {
		if shutdownMarker.MatchString(e.Message) {
			return true, ""
		}
	}
	return false, "no shutdown marker found in boot log"
}
