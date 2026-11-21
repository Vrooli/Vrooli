package codexrollout

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// MaxLineBytes bounds rollout records, including compacted history.
const MaxLineBytes = 32 << 20

// NativeUsage is operational usage observed in native rollout records. It is
// neither an invoice nor a statement of subscription quota consumption.
type NativeUsage struct {
	InputTokens         int64 `json:"input_tokens"`
	CachedInputTokens   int64 `json:"cached_input_tokens"`
	UncachedInputTokens int64 `json:"uncached_input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
}

type NativeUsageReport struct {
	Usage         *NativeUsage         `json:"usage"` // nil means no measurable response was observed.
	Responses     int                  `json:"responses"`
	Duplicates    int                  `json:"duplicates"`
	Incomplete    int                  `json:"incomplete"`
	ModelContexts []NativeModelContext `json:"model_contexts,omitempty"`
}

// NativeModelContext counts model-bearing native context records in the selected
// interval. It is not a count of responses and does not attribute usage.
type NativeModelContext struct {
	Model  string `json:"model"`
	Effort string `json:"effort,omitempty"`
	Count  int    `json:"count"`
}

type nativeUsageRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		ThreadID        string          `json:"thread_id"`
		ResponseID      string          `json:"response_id"`
		Model           json.RawMessage `json:"model"`
		Effort          json.RawMessage `json:"effort"`
		ReasoningEffort json.RawMessage `json:"reasoning_effort"`
		Usage           *struct {
			Input  *int64 `json:"input_tokens"`
			Cached *int64 `json:"cached_input_tokens"`
			Output *int64 `json:"output_tokens"`
		} `json:"usage"`
	} `json:"payload"`
}

// CodexNativeUsageAccumulator adds selected threads to a single report.
// Reuse the accumulator across rollout segments to deduplicate response IDs.
// The caller resolves and authorizes the source, and supplies a finite byte
// limit. The time interval is [start,end). Incomplete records remain unknown.
type CodexNativeUsageAccumulator struct {
	report NativeUsageReport
	seen   map[string]NativeUsage
}

func (a *CodexNativeUsageAccumulator) Report() NativeUsageReport {
	report := a.report
	if report.Usage != nil {
		usage := *report.Usage
		report.Usage = &usage
	}
	report.ModelContexts = append([]NativeModelContext(nil), report.ModelContexts...)
	return report
}

func (a *CodexNativeUsageAccumulator) Read(ctx context.Context, r io.Reader, thread string, start, end time.Time, maxBytes int64) error {
	if thread == "" || start.IsZero() || !end.After(start) || maxBytes <= 0 {
		return fmt.Errorf("thread, bounded window and byte limit are required")
	}
	// Stage changes so a truncated, cancelled, or conflicting read cannot
	// publish partial totals as a successful observation.
	pending := make(map[string]NativeUsage)
	report := a.Report()
	limited := &io.LimitedReader{R: r, N: maxBytes + 1}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 0, 64<<10), MaxLineBytes)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		var line nativeUsageRecord
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			return fmt.Errorf("invalid rollout record: %w", err)
		}
		if line.Type == "turn_context" || line.Type == "session_meta" {
			at, err := time.Parse(time.RFC3339Nano, line.Timestamp)
			var model, effort string
			_ = json.Unmarshal(line.Payload.Model, &model)
			_ = json.Unmarshal(line.Payload.Effort, &effort)
			if err == nil && !at.Before(start) && at.Before(end) && model != "" {
				if effort == "" {
					_ = json.Unmarshal(line.Payload.ReasoningEffort, &effort)
				}
				found := false
				for i := range report.ModelContexts {
					item := &report.ModelContexts[i]
					if item.Model == model && item.Effort == effort {
						item.Count++
						found = true
						break
					}
				}
				if !found {
					report.ModelContexts = append(report.ModelContexts, NativeModelContext{Model: model, Effort: effort, Count: 1})
				}
			}
			continue
		}
		if line.Type != "token_usage_record" {
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, line.Timestamp)
		if err != nil {
			report.Incomplete++
			continue
		}
		if at.Before(start) || !at.Before(end) {
			continue
		}
		p := line.Payload
		if p.ThreadID != thread {
			report.Incomplete++
			continue
		}
		u := p.Usage
		if p.ResponseID == "" || u == nil || u.Input == nil || u.Cached == nil || u.Output == nil || *u.Input < 0 || *u.Cached < 0 || *u.Output < 0 || *u.Cached > *u.Input {
			report.Incomplete++
			continue
		}
		value := NativeUsage{InputTokens: *u.Input, CachedInputTokens: *u.Cached, UncachedInputTokens: *u.Input - *u.Cached, OutputTokens: *u.Output}
		key := thread + "\x00" + p.ResponseID
		previous, exists := a.seen[key]
		if !exists {
			previous, exists = pending[key]
		}
		if exists {
			if previous != value {
				return fmt.Errorf("conflicting usage for response %s", p.ResponseID)
			}
			report.Duplicates++
			continue
		}
		pending[key] = value
		if report.Usage == nil {
			report.Usage = &NativeUsage{}
		}
		report.Usage.InputTokens += value.InputTokens
		report.Usage.CachedInputTokens += value.CachedInputTokens
		report.Usage.UncachedInputTokens += value.UncachedInputTokens
		report.Usage.OutputTokens += value.OutputTokens
		report.Responses++
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if limited.N == 0 {
		return fmt.Errorf("rollout exceeds byte limit")
	}
	if a.seen == nil {
		a.seen = make(map[string]NativeUsage)
	}
	for key, value := range pending {
		a.seen[key] = value
	}
	a.report = report
	return nil
}
