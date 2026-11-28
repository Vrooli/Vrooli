package codexrollout

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCodexNativeUsageDeduplicatesAndPreservesUnknown(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	record := `{"timestamp":"2026-01-01T00:01:00Z","type":"token_usage_record","payload":{"thread_id":"one","response_id":"r1","usage":{"input_tokens":100,"cached_input_tokens":80,"output_tokens":4}}}` + "\n"
	missing := `{"timestamp":"2026-01-01T00:02:00Z","type":"token_usage_record","payload":{"thread_id":"one","response_id":"r2","usage":{"input_tokens":0,"output_tokens":0}}}` + "\n"
	zero := `{"timestamp":"2026-01-01T00:03:00Z","type":"token_usage_record","payload":{"thread_id":"two","response_id":"r1","usage":{"input_tokens":0,"cached_input_tokens":0,"output_tokens":0}}}` + "\n"
	var a CodexNativeUsageAccumulator
	if err := a.Read(context.Background(), strings.NewReader(record+record+missing), "one", start, end, 10000); err != nil {
		t.Fatal(err)
	}
	if err := a.Read(context.Background(), strings.NewReader(zero), "two", start, end, 10000); err != nil {
		t.Fatal(err)
	}
	got := a.Report()
	if got.Responses != 2 || got.Duplicates != 1 || got.Incomplete != 1 || got.Usage == nil || *got.Usage != (NativeUsage{InputTokens: 100, CachedInputTokens: 80, UncachedInputTokens: 20, OutputTokens: 4}) {
		t.Fatalf("report: %+v usage=%+v", got, got.Usage)
	}
	var empty CodexNativeUsageAccumulator
	if err := empty.Read(context.Background(), strings.NewReader(missing), "one", start, end, 10000); err != nil {
		t.Fatal(err)
	}
	if empty.Report().Usage != nil || empty.Report().Incomplete != 1 {
		t.Fatalf("unknown became zero: %+v", empty.Report())
	}
}

func TestCodexNativeUsageRejectsConflictAndLimits(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	record := `{"timestamp":"2026-01-01T00:01:00Z","type":"token_usage_record","payload":{"thread_id":"one","response_id":"r1","usage":{"input_tokens":100,"cached_input_tokens":80,"output_tokens":4}}}` + "\n"
	var a CodexNativeUsageAccumulator
	if err := a.Read(context.Background(), strings.NewReader(record+strings.Replace(record, `"output_tokens":4`, `"output_tokens":5`, 1)), "one", start, start.Add(time.Hour), 10000); err == nil {
		t.Fatal("conflicting duplicate accepted")
	}
	var b CodexNativeUsageAccumulator
	if err := b.Read(context.Background(), strings.NewReader(record), "one", start, start.Add(time.Hour), 10); err == nil {
		t.Fatal("oversize read accepted")
	}
}

func TestNativeUsageCountsObservedModelContextsWithinWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	records := `{"timestamp":"2026-01-01T00:01:00Z","type":"turn_context","payload":{"model":"gpt-6-luna","effort":"high"}}` + "\n" +
		`{"timestamp":"2026-01-01T00:02:00Z","type":"turn_context","payload":{"model":"gpt-6-luna","reasoning_effort":"high"}}` + "\n" +
		`{"timestamp":"2026-01-01T00:03:00Z","type":"turn_context","payload":{"model":"gpt-6.1-terra"}}` + "\n" +
		`{"timestamp":"2026-01-01T00:04:00Z","type":"other","payload":{"model":{"nested":"ignored"}}}` + "\n" +
		`{"timestamp":"2026-01-01T01:00:00Z","type":"turn_context","payload":{"model":"outside"}}` + "\n"
	var a CodexNativeUsageAccumulator
	if err := a.Read(context.Background(), strings.NewReader(records), "one", start, start.Add(time.Hour), 10000); err != nil {
		t.Fatal(err)
	}
	got := a.Report()
	if len(got.ModelContexts) != 2 || got.ModelContexts[0] != (NativeModelContext{Model: "gpt-6-luna", Effort: "high", Count: 2}) || got.ModelContexts[1] != (NativeModelContext{Model: "gpt-6.1-terra", Count: 1}) || got.Usage != nil {
		t.Fatalf("model contexts = %+v, usage = %+v", got.ModelContexts, got.Usage)
	}
	got.ModelContexts[0].Count = 99
	if a.Report().ModelContexts[0].Count != 2 {
		t.Fatal("report mutated accumulator")
	}
}

// TestCodexNativeUsageRetainedAudit is an opt-in replay of the six explicitly
// selected native rollouts, not a model invocation or a billing reconciliation.
func TestCodexNativeUsageRetainedAudit(t *testing.T) {
	path := os.Getenv("CODEX_NATIVE_USAGE_AUDIT")
	if path == "" {
		t.Skip("set CODEX_NATIVE_USAGE_AUDIT to replay retained evidence")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var audit struct {
		Threads map[string]struct {
			Thread    string `json:"thread_id"`
			Source    string `json:"source"`
			Responses int    `json:"unique_response_records"`
			Usage     struct {
				Input  int64 `json:"input_tokens"`
				Cached int64 `json:"cached_input_tokens"`
				Output int64 `json:"output_tokens"`
			} `json:"usage"`
		} `json:"threads"`
	}
	if err := json.Unmarshal(data, &audit); err != nil {
		t.Fatal(err)
	}
	if len(audit.Threads) != 6 {
		t.Fatalf("expected six selected threads, got %d", len(audit.Threads))
	}
	start := time.Date(2026, 10, 27, 4, 30, 0, 0, time.UTC)
	end := time.Date(2026, 10, 27, 12, 32, 0, 0, time.UTC)
	var all CodexNativeUsageAccumulator
	for name, thread := range audit.Threads {
		f, err := os.Open(thread.Source)
		if err != nil {
			t.Fatal(err)
		}
		var one CodexNativeUsageAccumulator
		err = one.Read(context.Background(), f, thread.Thread, start, end, 256<<20)
		_ = f.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := one.Report()
		if got.Incomplete != 0 || got.Responses != thread.Responses || got.Usage == nil || got.Usage.InputTokens != thread.Usage.Input || got.Usage.CachedInputTokens != thread.Usage.Cached || got.Usage.OutputTokens != thread.Usage.Output {
			t.Fatalf("%s: report %+v usage %+v", name, got, got.Usage)
		}
		f, err = os.Open(thread.Source)
		if err != nil {
			t.Fatal(err)
		}
		err = all.Read(context.Background(), f, thread.Thread, start, end, 256<<20)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("six native threads: %+v usage=%+v", all.Report(), all.Report().Usage)
}
