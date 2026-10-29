package transcriptindex_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/transcriptindex"
)

// writeLines writes a transcript and returns each line's byte offset.
func writeLines(t *testing.T, lines ...string) (string, []int64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	var offsets []int64
	var body strings.Builder
	for _, line := range lines {
		offsets = append(offsets, int64(body.Len()))
		body.WriteString(line + "\n")
	}
	if err := os.WriteFile(path, []byte(body.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path, offsets
}

func at(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

// Legacy rows carry the indexing time and an unknown model; the repair takes
// both from the transcript by line offset. A row before the first model line
// takes the file's first model.
func TestScanUsageFixesReadsLineTimeAndTurnModel(t *testing.T) {
	huge := `{"timestamp":"2026-10-28T09:00:03Z","type":"response_item","payload":{"type":"function_call_output","output":"` + strings.Repeat("x", 3<<20) + `"}}`
	path, offsets := writeLines(t,
		`{"timestamp":"2026-10-28T09:00:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":10}}}}`,
		`{"timestamp":"2026-10-28T09:00:01Z","type":"turn_context","payload":{"cwd":"/w","model":"gpt-6.1-terra"}}`,
		`{"timestamp":"2026-10-28T09:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":20},"model_context_window":258400}}}`,
		huge,
		`{"timestamp":"2026-10-28T09:05:00Z","type":"turn_context","payload":{"model":"gpt-6-luna"}}`,
		`{"timestamp":"2026-10-28T09:05:01Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":30}}}}`,
	)
	fixes, err := transcriptindex.ScanUsageFixes(path, []int64{offsets[5], offsets[0], offsets[2]})
	if err != nil {
		t.Fatal(err)
	}
	want := []transcriptindex.UsageFix{
		{LineOffset: offsets[0], ObservedAt: at("2026-10-28T09:00:00Z"), Model: "gpt-6.1-terra"},
		{LineOffset: offsets[2], ObservedAt: at("2026-10-28T09:00:02Z"), Model: "gpt-6.1-terra"},
		{LineOffset: offsets[5], ObservedAt: at("2026-10-28T09:05:01Z"), Model: "gpt-6-luna"},
	}
	if len(fixes) != len(want) {
		t.Fatalf("fixes = %+v, want %+v", fixes, want)
	}
	for i := range want {
		if fixes[i].LineOffset != want[i].LineOffset || !fixes[i].ObservedAt.Equal(want[i].ObservedAt) || fixes[i].Model != want[i].Model {
			t.Errorf("fix %d = %+v, want %+v", i, fixes[i], want[i])
		}
	}

	claude, claudeOffsets := writeLines(t,
		`{"parentUuid":null,"sessionId":"s","message":{"model":"claude-opus-5-5","id":"msg_1","role":"assistant","usage":{"input_tokens":3}},"type":"assistant","timestamp":"2026-10-28T10:00:00.5Z"}`,
		`{"parentUuid":"a","sessionId":"s","message":{"model":"<synthetic>","id":"msg_2","role":"assistant","usage":{"input_tokens":0}},"type":"assistant","timestamp":"2026-10-28T10:01:00Z"}`,
	)
	fixes, err = transcriptindex.ScanUsageFixes(claude, claudeOffsets)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixes) != 2 || fixes[0].Model != "claude-opus-5-5" || !fixes[0].ObservedAt.Equal(at("2026-10-28T10:00:00.5Z")) || fixes[1].Model != "claude-opus-5-5" {
		t.Fatalf("claude fixes = %+v, want the message model and line time", fixes)
	}
}

type repairStore struct {
	files    []transcriptindex.FileRow
	offsets  map[string][]int64
	repaired map[string][]transcriptindex.UsageFix
	batches  int
	busy     int // RepairUsageRows fails with SQLITE_BUSY this many times first
	done     map[string]transcriptindex.RepairReport
}

func (s *repairStore) UsageRepairDone(_ context.Context, name string) (bool, error) {
	_, ok := s.done[name]
	return ok, nil
}

func (s *repairStore) MarkUsageRepairDone(_ context.Context, name string, report transcriptindex.RepairReport) error {
	s.done[name] = report
	return nil
}

func (s *repairStore) UsageFilesToRepair(context.Context) ([]transcriptindex.FileRow, error) {
	return s.files, nil
}

func (s *repairStore) UsageRowOffsets(_ context.Context, key string) ([]int64, error) {
	return s.offsets[key], nil
}

func (s *repairStore) RepairUsageRows(_ context.Context, key string, fixes []transcriptindex.UsageFix) error {
	if s.busy > 0 {
		s.busy--
		return errors.New("database error during repair_usage_rows: database is locked (5) (SQLITE_BUSY)")
	}
	s.batches++
	s.repaired[key] = append(s.repaired[key], fixes...)
	return nil
}

// The repair writes in short batches, retries a busy database, reports
// unreadable files, and records its completion so it never runs again.
func TestRepairUsageBatchesRetriesBusyAndRunsOnce(t *testing.T) {
	lines := []string{`{"timestamp":"2026-10-28T09:00:01Z","type":"turn_context","payload":{"model":"gpt-6-luna"}}`}
	for i := 0; i < 1200; i++ {
		lines = append(lines, `{"timestamp":"2026-10-28T09:00:02Z","type":"event_msg","payload":{"type":"token_count"}}`)
	}
	path, offsets := writeLines(t, lines...)
	store := &repairStore{
		files:    []transcriptindex.FileRow{{Key: "1:1", Path: path}, {Key: "1:2", Path: filepath.Join(t.TempDir(), "gone.jsonl")}},
		offsets:  map[string][]int64{"1:1": offsets[1:], "1:2": {0}},
		repaired: map[string][]transcriptindex.UsageFix{},
		busy:     2,
		done:     map[string]transcriptindex.RepairReport{},
	}
	restore := transcriptindex.SetRepairRetryDelayForTest(time.Millisecond)
	defer restore()
	report, err := transcriptindex.RepairUsage(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	if report.Files != 1 || report.Rows != 1200 || len(report.Failures) != 1 {
		t.Fatalf("report = %+v, want one repaired file of 1200 rows and one failure", report)
	}
	if store.batches != 3 || len(store.repaired["1:1"]) != 1200 || store.repaired["1:1"][0].Model != "gpt-6-luna" {
		t.Fatalf("batches = %d rows = %d, want 3 batches of at most 500", store.batches, len(store.repaired["1:1"]))
	}
	if len(store.done) != 1 {
		t.Fatalf("completion marker = %v", store.done)
	}
	store.batches = 0
	if again, err := transcriptindex.RepairUsage(context.Background(), store); err != nil || again.Files != 0 || store.batches != 0 {
		t.Fatalf("a completed repair ran again: %+v %v", again, err)
	}
}

// Database contention is not a property of the file: a busy failure leaves
// the file pending for the next tick instead of unreadable.
func TestTransientStoreFailureIsContentionOnly(t *testing.T) {
	for message, want := range map[string]bool{
		"database error during begin_immediate_transaction: database is locked (5) (SQLITE_BUSY)": true,
		"database table is locked (6) (SQLITE_LOCKED)":                                            true,
		"open /x.jsonl: no such file or directory":                                                false,
		"unsupported transcript format":                                                           false,
	} {
		if got := transcriptindex.TransientStoreFailure(message); got != want {
			t.Errorf("TransientStoreFailure(%q) = %v, want %v", message, got, want)
		}
	}
}
