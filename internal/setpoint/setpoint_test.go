package setpoint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func repoSetpointPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", filepath.FromSlash(RelativePath))
}

func TestSetpointRejectsDuplicateCellRef(t *testing.T) {
	doc := `{"schema_version":"1.0.0","confidence":{"level":"SKETCH","rationale":"r","recorded_on":"2026-09-02"},"bars":[
	 {"id":"a","cell_ref":"availability/A2","projection":"availability","target_kind":"x","deadband":"d","sustain":"1h","actuator":"a","decision_ref":"r","gradeable":false,"not_gradeable_reason":"test"},
	 {"id":"b","cell_ref":"availability/A2","projection":"availability","target_kind":"y","deadband":"d","sustain":"1h","actuator":"a","decision_ref":"r","unit":"u","max":1,"gradeable":true}]}`
	if _, err := Parse("dup.json", []byte(doc)); err == nil || !strings.Contains(err.Error(), "share cell_ref availability/A2") {
		t.Fatalf("duplicate cell_ref accepted: %v", err)
	}
}

func TestRepositorySetpointHasUniqueCellsAndValidates(t *testing.T) {
	sp, err := Load(repoSetpointPath(t))
	if err != nil {
		t.Fatalf("repository setpoint: %v", err)
	}
	for _, cell := range []string{CellCPUPressure, CellStrandedMemory, CellForkRate, CellMemoryPSI, CellCrashLoop} {
		if _, ok := sp.Bar(cell); !ok {
			t.Errorf("repository setpoint lacks %s", cell)
		}
	}
}

func TestSetpointBarCarriesMaxUnitAndSustain(t *testing.T) {
	sp, err := Load(repoSetpointPath(t))
	if err != nil {
		t.Fatal(err)
	}
	bar, ok := sp.Bar(CellForkRate)
	if !ok || bar.Max == nil || *bar.Max != 200 || bar.Unit != "forks per second" || bar.Window != 10*time.Minute {
		t.Fatalf("fork-rate bar = %+v", bar)
	}
	if sp.Max(CellCPUPressure, 1) != 50 || sp.Sustain(CellCPUPressure, time.Second) != 10*time.Minute {
		t.Fatalf("cpu bar accessors: max=%v sustain=%v", sp.Max(CellCPUPressure, 1), sp.Sustain(CellCPUPressure, time.Second))
	}
	if sp.Max("nowhere/Z9", 7) != 7 || sp.Sustain("nowhere/Z9", time.Second) != time.Second {
		t.Fatal("absent cell must return the fallback")
	}
}

func TestParseRejectsUngradeableBarWithoutReason(t *testing.T) {
	doc := `{"schema_version":"1.0.0","confidence":{"level":"SKETCH","rationale":"r","recorded_on":"2026-09-02"},"bars":[
	 {"id":"a","cell_ref":"substrate/SB1","projection":"substrate","target_kind":"x","deadband":"d","sustain":"1h","actuator":"a","decision_ref":"r","gradeable":false}]}`
	if _, err := Parse("bad.json", []byte(doc)); err == nil || !strings.Contains(err.Error(), "states no reason") {
		t.Fatalf("ungradeable bar without a reason accepted: %v", err)
	}
}

func TestDocumentCarriesConfidenceConstantsAndUnmapped(t *testing.T) {
	sp, err := Load(repoSetpointPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if sp.Confidence.Level == "" || sp.Constants.RetentionMarginDays <= 0 || len(sp.Unmapped) == 0 {
		t.Fatalf("document = %+v", sp.Document)
	}
}

func TestSchemaRejectsGradeableBarWithoutThreshold(t *testing.T) {
	doc := `{"schema_version":"1.0.0","confidence":{"level":"SKETCH","rationale":"r","recorded_on":"2026-09-02"},"bars":[
	 {"id":"a","cell_ref":"substrate/SB1","projection":"substrate","target_kind":"x","deadband":"d","sustain":"1h","actuator":"a","decision_ref":"r","unit":"u","gradeable":true}]}`
	if _, err := Parse("bad.json", []byte(doc)); err == nil || !strings.Contains(err.Error(), "authors no min or max") {
		t.Fatalf("gradeable bar without min/max accepted: %v", err)
	}
}

func TestParseSustain(t *testing.T) {
	cases := map[string]time.Duration{"10m": 10 * time.Minute, "24h": 24 * time.Hour, "30d": 30 * 24 * time.Hour, "one read": 0, "2 windows": 0, "": 0}
	for raw, want := range cases {
		got, ok := ParseSustain(raw)
		if got != want || ok != (want > 0) {
			t.Errorf("ParseSustain(%q) = %v, %v; want %v", raw, got, ok, want)
		}
	}
}

func TestResolveOrderAndFallback(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom.json")
	data, err := os.ReadFile(repoSetpointPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(custom, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sp, err := Resolve([]string{PathEnv + "=" + custom}, dir)
	if err != nil || sp.Path != custom {
		t.Fatalf("PathEnv not honored: %v %v", sp.Path, err)
	}
	sp, err = Resolve(nil, dir)
	if err != nil || sp.Path != FallbackPath {
		t.Fatalf("missing file must fall back: %v %v", sp.Path, err)
	}
	if sp.Max(CellForkRate, 0) != 200 || sp.Sustain(CellForkRate, 0) != 10*time.Minute {
		t.Fatalf("fallback bars: %+v", sp.Bars)
	}
	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte(`{"bars":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve([]string{PathEnv + "=" + broken}, dir); err == nil {
		t.Fatal("an unreadable bar file must be an error, not a fallback")
	}
}

func TestSustainerHonorsAuthoredWindow(t *testing.T) {
	for name, state := range map[string]SustainState{"memory": NewMemoryState(), "file": FileState{Dir: t.TempDir(), Prefix: "t."}} {
		t.Run(name, func(t *testing.T) {
			clock := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
			s := NewSustainer(state).WithClock(func() time.Time { return clock })
			if s.Breach("fork", true, 10*time.Minute) {
				t.Fatal("first breach fired before the window")
			}
			clock = clock.Add(61 * time.Second)
			if s.Breach("fork", true, 10*time.Minute) {
				t.Fatal("61 seconds is not a 10 minute sustain")
			}
			clock = clock.Add(9 * time.Minute)
			if !s.Breach("fork", true, 10*time.Minute) {
				t.Fatal("sustained breach did not fire")
			}
			if s.Breach("fork", false, 10*time.Minute) {
				t.Fatal("a clear reading must not fire")
			}
			if _, ok := s.Since("fork"); ok {
				t.Fatal("clear did not reset the window")
			}
			if !s.Breach("instant", true, 0) {
				t.Fatal("a zero sustain fires on the first breach")
			}
		})
	}
}
