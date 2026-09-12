package aisearch

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeSampleFile drops sampleSearchJSON into a temp dir and returns the path.
func writeSampleFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "search.json")
	if err := os.WriteFile(path, []byte(sampleSearchJSON), 0o644); err != nil {
		t.Fatalf("seed search.json: %v", err)
	}
	return path
}

func TestWriteProviderTuningQueryTimeOnly(t *testing.T) {
	path := writeSampleFile(t)

	// Flip a QUERY-TIME factor only (rerank_shortlist 50 -> 80).
	newTuning := TuningConfig{
		Engine: EngineDense, EmbedModel: DefaultEmbedModel, EmbedTaskPrefix: true,
		RerankEnabled: true, RerankBlend: true, RerankShortlist: 80,
	}
	eff, idxChanged, written, err := WriteProviderTuning(path, "demo.commands", newTuning, false)
	if err != nil {
		t.Fatalf("WriteProviderTuning: %v", err)
	}
	if idxChanged {
		t.Fatalf("query-time-only change must not report index-time changed")
	}
	if !written {
		t.Fatalf("a real change must report written=true")
	}
	if eff.RerankShortlist != 80 {
		t.Fatalf("effective shortlist = %d, want 80", eff.RerankShortlist)
	}

	// The persisted file must round-trip with the new value and preserve the rest.
	reloaded, err := LoadSearchFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	p, ok := reloaded.Provider("demo.commands")
	if !ok {
		t.Fatalf("provider missing after write")
	}
	if p.Tuning.RerankShortlist != 80 {
		t.Fatalf("persisted shortlist = %d, want 80", p.Tuning.RerankShortlist)
	}
	// Descriptor + tests survived the rewrite.
	if len(p.Endpoint) == 0 {
		t.Fatalf("endpoint sub-object dropped by write")
	}
	if len(p.Tests.Cases) != 2 || p.Tests.Cases[0].ID != "c1" {
		t.Fatalf("tests corpus not preserved: %+v", p.Tests)
	}
	if !p.Tests.Cases[1].ExpectNoStrongHit {
		t.Fatalf("negative case not preserved")
	}
}

func TestWriteProviderTuningIndexTimeChange(t *testing.T) {
	path := writeSampleFile(t)

	// Flip an INDEX-TIME factor (embed_task_prefix true -> false).
	newTuning := TuningConfig{
		Engine: EngineDense, EmbedModel: DefaultEmbedModel, EmbedTaskPrefix: false,
		RerankEnabled: true, RerankBlend: true, RerankShortlist: 50,
	}
	_, idxChanged, written, err := WriteProviderTuning(path, "demo.commands", newTuning, false)
	if err != nil {
		t.Fatalf("WriteProviderTuning: %v", err)
	}
	if !idxChanged {
		t.Fatalf("embed_task_prefix flip must report index-time changed")
	}
	if !written {
		t.Fatalf("a real change must report written=true")
	}
}

func TestWriteProviderTuningEngineChangeIsIndexTime(t *testing.T) {
	path := writeSampleFile(t)
	newTuning := CommandCorpusTuning()
	newTuning.Engine = EngineHybrid
	_, idxChanged, written, err := WriteProviderTuning(path, "demo.commands", newTuning, false)
	if err != nil {
		t.Fatalf("WriteProviderTuning: %v", err)
	}
	if !idxChanged || !written {
		t.Fatalf("engine dense->hybrid must be an index-time write (idx=%v written=%v)", idxChanged, written)
	}
}

func TestWriteProviderTuningDryRun(t *testing.T) {
	path := writeSampleFile(t)
	before, _ := os.ReadFile(path)

	newTuning := CommandCorpusTuning()
	newTuning.RerankShortlist = 100
	eff, idxChanged, written, err := WriteProviderTuning(path, "demo.commands", newTuning, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if written {
		t.Fatalf("dry run must not write")
	}
	if idxChanged {
		t.Fatalf("dry run query-time change must not be index-time")
	}
	if eff.RerankShortlist != 100 {
		t.Fatalf("dry run must still resolve the proposed effective tuning")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("dry run mutated the file on disk")
	}
}

func TestWriteProviderTuningNoOp(t *testing.T) {
	path := writeSampleFile(t)
	before, _ := os.ReadFile(path)

	// Submit the current tuning verbatim (with defaults filled).
	current, _ := LoadSearchFile(path)
	p, _ := current.Provider("demo.commands")
	_, _, written, err := WriteProviderTuning(path, "demo.commands", p.ResolvedTuning(), false)
	if err != nil {
		t.Fatalf("no-op write: %v", err)
	}
	if written {
		t.Fatalf("submitting the current tuning must be a no-op (written=false)")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("no-op write mutated the file")
	}
}

func TestWriteProviderTuningInvalid(t *testing.T) {
	path := writeSampleFile(t)
	bad := CommandCorpusTuning()
	bad.Engine = "trie" // not a known engine
	_, _, written, err := WriteProviderTuning(path, "demo.commands", bad, false)
	if err == nil {
		t.Fatalf("invalid tuning must be rejected")
	}
	if written {
		t.Fatalf("invalid tuning must not be written")
	}
}

func TestWriteProviderTuningUnknownProvider(t *testing.T) {
	path := writeSampleFile(t)
	_, _, _, err := WriteProviderTuning(path, "does.not.exist", CommandCorpusTuning(), false)
	if err == nil {
		t.Fatalf("unknown provider must error")
	}
	var notIn ErrProviderNotInFile
	if !errors.As(err, &notIn) {
		t.Fatalf("expected ErrProviderNotInFile, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// WriteProviderCorpus tests
// ---------------------------------------------------------------------------

func sampleCorpus() TestSuite {
	return TestSuite{
		Name:        "demo suite",
		Description: "round-trip corpus",
		Cases: []TestCase{
			{ID: "q1", Query: "restart a scenario", ExpectIDs: []string{"restart"}, ExpectWithinTopK: 3, Tags: []string{"strong"}},
			{ID: "n1", Query: "asdf qwer", ExpectNoStrongHit: true, ExpectMaxScore: 0.1, Tags: []string{"gibberish"}},
		},
	}
}

func TestWriteProviderCorpusRoundTrip(t *testing.T) {
	path := writeSampleFile(t)
	suite := sampleCorpus()

	eff, written, err := WriteProviderCorpus(path, "demo.commands", suite, false)
	if err != nil {
		t.Fatalf("WriteProviderCorpus: %v", err)
	}
	if !written {
		t.Fatalf("a real change must report written=true")
	}
	if len(eff.Cases) != 2 || eff.Cases[0].ID != "q1" {
		t.Fatalf("effective corpus wrong: %+v", eff)
	}

	// Reload and verify the tests block was persisted and tuning was untouched.
	reloaded, err := LoadSearchFile(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	p, ok := reloaded.Provider("demo.commands")
	if !ok {
		t.Fatalf("provider missing after write")
	}
	if len(p.Tests.Cases) != 2 || p.Tests.Cases[0].ID != "q1" {
		t.Fatalf("persisted corpus wrong: %+v", p.Tests)
	}
	// Tuning block must survive the rewrite.
	if p.Tuning.Engine != EngineDense {
		t.Fatalf("tuning block corrupted by corpus write: engine=%q", p.Tuning.Engine)
	}
	// Descriptor must survive.
	if len(p.Endpoint) == 0 {
		t.Fatalf("endpoint dropped by corpus write")
	}
}

func TestWriteProviderCorpusDryRun(t *testing.T) {
	path := writeSampleFile(t)
	before, _ := os.ReadFile(path)

	suite := sampleCorpus()
	eff, written, err := WriteProviderCorpus(path, "demo.commands", suite, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if written {
		t.Fatalf("dry run must not write")
	}
	if len(eff.Cases) != 2 || eff.Cases[0].ID != "q1" {
		t.Fatalf("dry run must still return the proposed corpus: %+v", eff)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("dry run mutated the file on disk")
	}
}

func TestWriteProviderCorpusNoOp(t *testing.T) {
	path := writeSampleFile(t)
	before, _ := os.ReadFile(path)

	// Load the current corpus and submit it verbatim.
	current, _ := LoadSearchFile(path)
	p, _ := current.Provider("demo.commands")
	_, written, err := WriteProviderCorpus(path, "demo.commands", p.Tests, false)
	if err != nil {
		t.Fatalf("no-op write: %v", err)
	}
	if written {
		t.Fatalf("submitting the current corpus must be a no-op (written=false)")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("no-op write mutated the file")
	}
}

func TestWriteProviderCorpusUnknownProvider(t *testing.T) {
	path := writeSampleFile(t)
	_, _, err := WriteProviderCorpus(path, "does.not.exist", sampleCorpus(), false)
	if err == nil {
		t.Fatalf("unknown provider must error")
	}
	var notIn ErrProviderNotInFile
	if !errors.As(err, &notIn) {
		t.Fatalf("expected ErrProviderNotInFile, got %T: %v", err, err)
	}
}

func TestWriteProviderCorpusValidationError(t *testing.T) {
	path := writeSampleFile(t)
	bad := TestSuite{Cases: []TestCase{
		{ID: "q1", Query: ""}, // missing query — Validate rejects it
	}}
	_, written, err := WriteProviderCorpus(path, "demo.commands", bad, false)
	if err == nil {
		t.Fatalf("invalid corpus must be rejected")
	}
	if written {
		t.Fatalf("invalid corpus must not be written")
	}
}

func TestIndexTimeChanged(t *testing.T) {
	base := CommandCorpusTuning()
	// Query-time-only deltas are not index-time.
	qt := base
	qt.RerankEnabled = !base.RerankEnabled
	qt.RerankShortlist = base.RerankShortlist + 10
	qt.Floor.MaxGap = 0.3
	if base.IndexTimeChanged(qt) {
		t.Fatalf("query-time deltas must not be index-time changes")
	}
	// Each index-time factor independently flips the verdict.
	for _, mut := range []func(*TuningConfig){
		func(c *TuningConfig) { c.Engine = EngineHybrid },
		func(c *TuningConfig) { c.EmbedTaskPrefix = !c.EmbedTaskPrefix },
		func(c *TuningConfig) { c.EmbedModel = "some-other-model" },
	} {
		c := base
		mut(&c)
		if !base.IndexTimeChanged(c) {
			t.Fatalf("index-time factor change not detected: %+v", c)
		}
	}
}
