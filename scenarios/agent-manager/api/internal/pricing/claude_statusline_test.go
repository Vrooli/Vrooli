package pricing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corestorage "github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
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

func TestClaudeStatuslineReaderCarriesOnlyOpaqueForwardAccountScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quota-samples.ndjson")
	const scope = "anthropic:sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	appendFile(t, path,
		`{"observed_at":"2026-10-29T06:41:11Z","session_id":"legacy","five_hour":{"used_percentage":63,"resets_at":1793268600}}`+"\n"+
			`{"observed_at":"2026-10-29T06:42:11Z","account_scope":"`+scope+`","session_id":"live","five_hour":{"used_percentage":64,"resets_at":1793268600}}`+"\n")
	store := NewMemoryRepository()
	reader := &ClaudeStatuslineReader{Path: path, Store: store, Now: func() time.Time { return time.Date(2026, 10, 29, 6, 43, 0, 0, time.UTC) }}
	if got, err := reader.Ingest(t.Context()); err != nil || got != 2 {
		t.Fatalf("ingested %d (%v), want both old unknown and new bound samples", got, err)
	}
	rows, err := store.ListQuotaObservations(t.Context(), "anthropic", "five_hour", "300m", 10)
	if err != nil || len(rows) != 2 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if rows[0].AccountScope != scope || rows[1].AccountScope != "" {
		t.Fatalf("forward binding must retain scoped new evidence and unknown legacy evidence: %+v", rows)
	}
}

func TestClaudeStatuslineReaderCarriesValidForwardSourceRunAndLeavesLegacyUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quota-samples.ndjson")
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	appendFile(t, path,
		`{"observed_at":"2026-10-29T06:41:11Z","five_hour":{"used_percentage":63,"resets_at":1793268600}}`+"\n"+
			`{"observed_at":"2026-10-29T06:42:11Z","source_run_id":"`+runID+`","five_hour":{"used_percentage":64,"resets_at":1793268600}}`+"\n"+
			`{"observed_at":"2026-10-29T06:43:11Z","source_run_id":"not-a-uuid","five_hour":{"used_percentage":65,"resets_at":1793268600}}`+"\n")
	store := NewMemoryRepository()
	reader := &ClaudeStatuslineReader{Path: path, Store: store, Now: func() time.Time { return time.Date(2026, 10, 29, 6, 44, 0, 0, time.UTC) }}
	if got, err := reader.Ingest(t.Context()); err != nil || got != 3 {
		t.Fatalf("ingested %d (%v), want three forward readings", got, err)
	}
	rows, err := store.ListQuotaObservations(t.Context(), "anthropic", "five_hour", "300m", 10)
	if err != nil || len(rows) != 3 {
		t.Fatalf("rows=%d err=%v, want 3", len(rows), err)
	}
	if rows[0].SourceRunID != "" || rows[1].SourceRunID != runID || rows[2].SourceRunID != "" {
		t.Fatalf("source run association must be forward and UUID-validated: %+v", rows)
	}
	for _, row := range rows {
		if row.Used != nil || row.Limit != nil || row.Uncertainty != "consumed amount and ceiling are unavailable" {
			t.Fatalf("percent-only statusline evidence must remain incomplete: %+v", row)
		}
	}
}

func TestClaudeStatuslineReaderRejectsMalformedAccountScopeAsUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quota-samples.ndjson")
	malformed := []string{
		"org-12345678-90ab-cdef-1234-567890abcdef",
		"anthropic:sha256:" + strings.Repeat("a", 63),
		"anthropic:sha256:" + strings.Repeat("A", 64),
		"anthropic:sha256:" + strings.Repeat("a", 65),
		"other:sha256:" + strings.Repeat("a", 64),
	}
	for i, scope := range malformed {
		observed := time.Date(2026, 10, 29, 7, i, 0, 0, time.UTC)
		line := fmt.Sprintf(`{"observed_at":%q,"account_scope":%q,"five_hour":{"used_percentage":63,"resets_at":1793268600}}`, observed.Format(time.RFC3339), scope)
		appendFile(t, path, line+"\n")
	}
	store := NewMemoryRepository()
	reader := &ClaudeStatuslineReader{Path: path, Store: store, Now: func() time.Time { return time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC) }}
	if got, err := reader.Ingest(t.Context()); err != nil || got != len(malformed) {
		t.Fatalf("ingested %d (%v), want %d malformed-but-usable readings", got, err, len(malformed))
	}
	rows, err := store.ListQuotaObservations(t.Context(), "anthropic", "five_hour", "300m", 10)
	if err != nil || len(rows) != len(malformed) {
		t.Fatalf("rows=%d err=%v, want %d", len(rows), err, len(malformed))
	}
	for i, row := range rows {
		if row.AccountScope != "" {
			t.Fatalf("malformed account scope %d was accepted", i)
		}
	}
}

func TestClaudeWindowMinutesReadsTheResourceDescriptor(t *testing.T) {
	for pool, want := range map[string]int64{"five_hour": 300, "seven_day": 10080, "unknown": 0} {
		if got := ClaudeWindowMinutes(pool); got != want {
			t.Errorf("%s: got %d, want %d", pool, got, want)
		}
	}
}

// The feed path is the claude-code owner's storage state file: identical to
// the documented ~/.vrooli location by default, moved by a custom
// VROOLI_STORAGE_ROOT, and replaced outright by the resource's override.
func TestClaudeStatuslineSamplesPathFollowsTheStorageAuthority(t *testing.T) {
	home := t.TempDir()
	resolve := func(env map[string]string) string {
		t.Helper()
		path, err := ClaudeStatuslineSamplesPath(corestorage.ResolverConfig{
			EnvGet:      func(key string) string { return env[key] },
			UserHomeDir: func() (string, error) { return home, nil },
		})
		if err != nil {
			t.Fatal(err)
		}
		return path
	}
	stateRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resolve(nil), filepath.Join(stateRoot, "vrooli", "claude-code", "quota-samples.ndjson"); got != want {
		t.Errorf("default samples path = %q, want %q", got, want)
	}
	custom := t.TempDir()
	if got, want := resolve(map[string]string{"VROOLI_STORAGE_ROOT": custom}), filepath.Join(custom, "state", "vrooli", "claude-code", "quota-samples.ndjson"); got != want {
		t.Errorf("custom storage root samples path = %q, want %q", got, want)
	}
	override := filepath.Join(t.TempDir(), "feed.ndjson")
	if got := resolve(map[string]string{"VROOLI_STORAGE_ROOT": custom, ClaudeStatuslineSamplesEnv: override}); got != override {
		t.Errorf("override samples path = %q, want %q", got, override)
	}
}
