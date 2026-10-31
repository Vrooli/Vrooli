package pricing

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func statuslineSample(at time.Time, five, seven float64) string {
	return fmt.Sprintf(`{"observed_at":%q,"session_id":"s-1","model":"claude-opus","five_hour":{"used_percentage":%g,"resets_at":1793268600},"seven_day":{"used_percentage":%g,"resets_at":1793804400}}`+"\n", at.Format(time.RFC3339Nano), five, seven)
}

func appendFile(t *testing.T, path, text string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(text); err != nil {
		t.Fatal(err)
	}
}

func TestClaudeStatuslineReaderIngestsIncrementallyAcrossRotationAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quota-samples.ndjson")
	at := time.Date(2026, 10, 29, 4, 0, 0, 0, time.UTC)
	store := NewMemoryRepository()
	reader := &ClaudeStatuslineReader{Path: path, Store: store, Now: func() time.Time { return at.Add(time.Hour) }}
	ingest := func(want int) {
		t.Helper()
		got, err := reader.Ingest(t.Context())
		if err != nil || got != want {
			t.Fatalf("ingested %d (%v), want %d", got, err, want)
		}
	}
	ingest(0) // no feed yet is not an error
	appendFile(t, path, statuslineSample(at, 18, 11)+`{"observed_at":"2026-10-29T04:01:00Z","five_hour":{"used_percentage":19`)
	ingest(2) // a partial trailing line waits for its newline
	appendFile(t, path, `,"resets_at":1793268600}}`+"\n")
	ingest(1)
	ingest(0)
	if err := os.Rename(path, path+".1"); err != nil {
		t.Fatal(err)
	}
	appendFile(t, path+".1", statuslineSample(at.Add(2*time.Minute), 20, 12))
	appendFile(t, path, statuslineSample(at.Add(3*time.Minute), 21, 12))
	ingest(4) // the rotated generation's tail, then the new file

	restarted := &ClaudeStatuslineReader{Path: path, Store: store}
	if _, err := restarted.Ingest(t.Context()); err != nil {
		t.Fatal(err)
	}
	five, _ := store.ListQuotaObservations(t.Context(), "anthropic", "five_hour", "300m", 100)
	seven, _ := store.ListQuotaObservations(t.Context(), "anthropic", "seven_day", "10080m", 100)
	if len(five) != 4 || len(seven) != 3 {
		t.Fatalf("observations five_hour=%d seven_day=%d, want 4/3 (re-reading must not duplicate)", len(five), len(seven))
	}
	latest := five[0]
	if *latest.UsedPercent != 21 || latest.ResetAt == nil || latest.ResetAt.Unix() != 1793268600 || latest.Provenance != ClaudeStatuslineProvenance || latest.Standing != QuotaStandingAvailable || *latest.WindowMinutes != 300 {
		t.Fatalf("latest five_hour observation = %+v", latest)
	}
}

// Live 2026-10-29 06:41Z: an idle session still renders five_hour resets_at
// Oct 22 and seven_day resets_at Oct 28; a current session renders the
// Oct 29 10:10Z and Nov 4 15:00Z windows. Only current windows are ingested.
func TestClaudeStatuslineReaderSkipsWindowsWhoseResetHadPassed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quota-samples.ndjson")
	appendFile(t, path, `{"observed_at":"2026-10-29T06:41:09Z","session_id":"idle","five_hour":{"used_percentage":57,"resets_at":1792692600},"seven_day":{"used_percentage":18,"resets_at":1793199600}}`+"\n"+
		`{"observed_at":"2026-10-29T06:41:11Z","session_id":"live","five_hour":{"used_percentage":63,"resets_at":1793268600},"seven_day":{"used_percentage":23,"resets_at":1793804400}}`+"\n")
	store := NewMemoryRepository()
	reader := &ClaudeStatuslineReader{Path: path, Store: store, Now: func() time.Time { return time.Date(2026, 10, 29, 6, 42, 0, 0, time.UTC) }}
	if got, err := reader.Ingest(t.Context()); err != nil || got != 2 {
		t.Fatalf("ingested %d (%v), want the live session's 2 windows", got, err)
	}
	five, _ := store.ListQuotaObservations(t.Context(), "anthropic", "five_hour", "300m", 10)
	seven, _ := store.ListQuotaObservations(t.Context(), "anthropic", "seven_day", "10080m", 10)
	if len(five) != 1 || *five[0].UsedPercent != 63 || five[0].ResetAt.Unix() != 1793268600 || len(seven) != 1 || *seven[0].UsedPercent != 23 {
		t.Fatalf("five_hour=%+v seven_day=%+v", five, seven)
	}
}
