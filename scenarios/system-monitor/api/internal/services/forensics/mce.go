package forensics

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MCEReport is the data payload for /api/v1/forensics/mce.
type MCEReport struct {
	Window      string `json:"window"` // e.g. "1 hour ago"
	Uncorrected int    `json:"uncorrected"`
	Corrected   int    `json:"corrected"`
	RawSummary  string `json:"rawSummary,omitempty"`
}

// MCE summary lines look like:
//
//	"2 Uncorrected error(s)"
//	"17 Corrected error(s)"
//
// We accept either ordering (digits before or after the keyword) to be
// resilient across ras-mc-ctl versions.
var (
	uncorrectedRe = regexp.MustCompile(`(?i)(?:(\d+)\s+(?:uncorrected|UE)|(?:uncorrected|UE)[^\d\n]*?(\d+))`)
	correctedRe   = regexp.MustCompile(`(?i)(?:(\d+)\s+corrected|corrected[^\d\n]*?(\d+))`)
)

// MCE returns the parsed `ras-mc-ctl --summary --since=...` output.
// When ras-mc-ctl is not installed or fails, returns available=false.
func (s *Service) MCE(ctx context.Context) Envelope {
	const key = "mce"
	now := s.now()
	if cached, ok := s.cache.get(key, now); ok {
		return cached
	}
	env := s.computeMCE(ctx, now)
	s.cache.set(key, env, now)
	return env
}

func (s *Service) computeMCE(ctx context.Context, now time.Time) Envelope {
	env := Envelope{GeneratedAt: now}

	if s.exec == nil {
		env.Reason = "command executor not configured"
		return env
	}

	const window = "1 hour ago"
	out, err := s.exec.CombinedOutput(ctx, "ras-mc-ctl", "--summary", "--since="+window)
	if err != nil {
		// Distinguish "not installed" from "ran but failed"; both are 200/available=false.
		msg := err.Error()
		switch {
		case strings.Contains(msg, "executable file not found"),
			strings.Contains(msg, "no such file"),
			strings.Contains(msg, "not found in $PATH"):
			env.Reason = "ras-mc-ctl not installed"
		default:
			env.Reason = "ras-mc-ctl failed: " + msg
		}
		return env
	}

	report := MCEReport{Window: window, RawSummary: string(out)}
	report.Uncorrected = sumMatches(uncorrectedRe.FindAllStringSubmatch(string(out), -1))
	report.Corrected = sumMatches(correctedRe.FindAllStringSubmatch(string(out), -1))

	env.Available = true
	env.Data = report
	return env
}

// sumMatches sums the first non-empty captured group across each match.
// The regexes have two alternative capture groups (digits-before vs
// digits-after the keyword), so we take whichever fired.
func sumMatches(matches [][]string) int {
	total := 0
	for _, m := range matches {
		for i := 1; i < len(m); i++ {
			if m[i] == "" {
				continue
			}
			if v, err := strconv.Atoi(m[i]); err == nil {
				total += v
			}
			break
		}
	}
	return total
}
