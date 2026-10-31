package pricing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ClaudeStatuslineProvenance labels quota observations read from the
// resource-owned Claude Code statusline feed (resources/claude-code
// `statusline run`).
const ClaudeStatuslineProvenance = "claude-code:statusline"

// ClaudeStatuslineSamplesEnv matches the resource's sample-path override.
const ClaudeStatuslineSamplesEnv = "VROOLI_CLAUDE_QUOTA_SAMPLES"

// claudeWindows maps Claude's rate-limit pools to their window length.
var claudeWindows = map[string]int64{"five_hour": 300, "seven_day": 10080}

// ClaudeWindowMinutes returns the window length of a Claude rate-limit pool,
// or 0 for an unknown pool.
func ClaudeWindowMinutes(pool string) int64 { return claudeWindows[pool] }

// DefaultClaudeStatuslineSamplesPath is the resource's documented sample file.
func DefaultClaudeStatuslineSamplesPath() string {
	if override := strings.TrimSpace(os.Getenv(ClaudeStatuslineSamplesEnv)); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".vrooli", "state", "vrooli", "claude-code", "quota-samples.ndjson")
}

type claudeStatuslineSample struct {
	ObservedAt time.Time               `json:"observed_at"`
	SessionID  string                  `json:"session_id"`
	FiveHour   *claudeStatuslineWindow `json:"five_hour"`
	SevenDay   *claudeStatuslineWindow `json:"seven_day"`
}

type claudeStatuslineWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       int64    `json:"resets_at"`
}

// ClaudeStatuslineReader ingests new statusline samples into quota
// observations. It tails the file from its last offset and follows one
// rotation (<file>.1). Observation ids are stable, so a restart that re-reads
// the file from the start inserts nothing new.
type ClaudeStatuslineReader struct {
	Path     string
	Store    QuotaObservationRepository
	Now      func() time.Time
	Interval time.Duration

	mu     sync.Mutex
	file   os.FileInfo
	offset int64
}

// Start ingests on an interval until ctx ends.
func (r *ClaudeStatuslineReader) Start(ctx context.Context) {
	if r == nil || r.Store == nil || r.Path == "" {
		return
	}
	interval := r.Interval
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			_, _ = r.Ingest(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// Ingest records every complete sample written since the last call and
// returns the number of observations offered to the store.
func (r *ClaudeStatuslineReader) Ingest(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, err := os.Stat(r.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	recorded := 0
	if r.file != nil && !os.SameFile(r.file, info) {
		// Rotated: finish the previous generation, then start the new file.
		if rotated, statErr := os.Stat(r.Path + ".1"); statErr == nil && os.SameFile(r.file, rotated) {
			n, _, err := r.ingestFrom(ctx, r.Path+".1", r.offset)
			recorded += n
			if err != nil {
				return recorded, err
			}
		}
		r.offset = 0
	}
	if info.Size() < r.offset {
		r.offset = 0
	}
	r.file = info
	n, next, err := r.ingestFrom(ctx, r.Path, r.offset)
	r.offset = next
	return recorded + n, err
}

func (r *ClaudeStatuslineReader) ingestFrom(ctx context.Context, path string, offset int64) (int, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, offset, err
	}
	defer file.Close()
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return 0, offset, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return 0, offset, err
	}
	end := bytes.LastIndexByte(data, '\n')
	if end < 0 {
		return 0, offset, nil
	}
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	recorded := 0
	for _, line := range bytes.Split(data[:end], []byte{'\n'}) {
		var sample claudeStatuslineSample
		if len(bytes.TrimSpace(line)) == 0 || json.Unmarshal(line, &sample) != nil || sample.ObservedAt.IsZero() {
			continue
		}
		for _, observation := range claudeStatuslineObservations(sample, path, now().UTC()) {
			if err := r.Store.RecordQuotaObservation(ctx, observation); err != nil {
				return recorded, offset, err
			}
			recorded++
		}
	}
	return recorded, offset + int64(end) + 1, nil
}

func claudeStatuslineObservations(sample claudeStatuslineSample, path string, now time.Time) []*QuotaObservation {
	var out []*QuotaObservation
	for _, w := range []struct {
		pool   string
		window *claudeStatuslineWindow
	}{{"five_hour", sample.FiveHour}, {"seven_day", sample.SevenDay}} {
		if w.window == nil || w.window.UsedPercentage == nil || *w.window.UsedPercentage < 0 || *w.window.UsedPercentage > 100 {
			continue
		}
		// An idle session renders its cached rate_limits until its next API
		// response; a window whose reset had passed when the sample was taken
		// says nothing about the current window (the store rejects it too, with
		// ErrResetElapsed). The resource feed no longer writes these, but
		// samples written before that fix remain on disk.
		if w.window.ResetsAt > 0 && ResetElapsed(time.Unix(w.window.ResetsAt, 0), sample.ObservedAt) {
			continue
		}
		percent, minutes := *w.window.UsedPercentage, claudeWindows[w.pool]
		standing := QuotaStandingAvailable
		if percent >= 100 {
			standing = QuotaStandingExhausted
		}
		o := &QuotaObservation{
			Provider:      "anthropic",
			Pool:          w.pool,
			Window:        fmt.Sprintf("%dm", minutes),
			ObservedAt:    sample.ObservedAt.UTC(),
			Standing:      standing,
			UsedPercent:   &percent,
			WindowMinutes: &minutes,
			Provenance:    ClaudeStatuslineProvenance,
			Uncertainty:   "account scope is the Claude Code login that rendered the status line; consumed amount and ceiling are unavailable",
			EvidenceRef:   "file://" + path + "#session=" + sample.SessionID,
		}
		if w.window.ResetsAt > 0 {
			reset := time.Unix(w.window.ResetsAt, 0).UTC()
			o.ResetAt = &reset
		}
		o.ID = QuotaObservationID(o)
		o.Freshness = Freshness(o, now, QuotaFreshnessMaxAge)
		if o.Validate() == nil {
			out = append(out, o)
		}
	}
	return out
}
