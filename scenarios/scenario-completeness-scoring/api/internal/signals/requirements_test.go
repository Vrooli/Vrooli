package signals

import (
	"math"
	"strings"
	"testing"
)

func collectRequirements(t *testing.T, root string) (RequirementsSignals, error) {
	t.Helper()
	snap := Snapshot{Root: root}
	err := requirementsCollector{}.Collect(&snap)
	return snap.Requirements, err
}

func TestRequirementsMissingDirNotCollected(t *testing.T) {
	sig, err := collectRequirements(t, t.TempDir())
	if err != nil {
		t.Fatalf("missing requirements/ must not error, got %v", err)
	}
	if sig.Collected {
		t.Fatal("missing requirements/ must report Collected=false")
	}
}

func TestRequirementsMalformedIndexErrors(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "requirements/index.json", `{"imports": [`)

	if _, err := collectRequirements(t, root); err == nil {
		t.Fatal("malformed index.json must error")
	}

	// Via the Service the error becomes a degradation, not a failure.
	snap := newService(requirementsCollector{}).Collect("demo", root)
	if len(snap.Degradations) != 1 || snap.Degradations[0].State != "failed" {
		t.Fatalf("degradations = %+v, want one failed entry", snap.Degradations)
	}
	if snap.Requirements.Collected {
		t.Fatal("failed collection must not mark requirements collected")
	}
}

func TestRequirementsFlatAndNestedShapes(t *testing.T) {
	tests := []struct {
		name           string
		files          map[string]string
		wantTotal      int
		wantPassing    int
		wantValidation int
		wantAvgDepth   float64
	}{
		{
			name: "flat list",
			files: map[string]string{
				"requirements/index.json": `{"imports":["01-a/module.json"]}`,
				"requirements/01-a/module.json": `{"requirements":[
					{"id":"REQ-1","status":"passed","validation":[{"type":"test"}]},
					{"id":"REQ-2","status":"draft"},
					{"id":"REQ-3","status":"Complete"}
				]}`,
			},
			wantTotal: 3, wantPassing: 2, wantValidation: 1, wantAvgDepth: 1,
		},
		{
			name: "id-reference children form a tree",
			files: map[string]string{
				"requirements/01-a/module.json": `{"requirements":[
					{"id":"REQ-1","children":["REQ-1a","REQ-1b"]},
					{"id":"REQ-1a","status":"passed","children":["REQ-1a-1"]},
					{"id":"REQ-1a-1","status":"passed","validation":[{"type":"test"}]},
					{"id":"REQ-1b","status":"pending"}
				]}`,
			},
			// REQ-1 is grouping-only (children, no status): skipped.
			wantTotal: 3, wantPassing: 2, wantValidation: 1, wantAvgDepth: 3,
		},
		{
			name: "inline nested children",
			files: map[string]string{
				"requirements/01-a/module.json": `{"requirements":[
					{"id":"REQ-1","children":[
						{"id":"REQ-1a","status":"done","children":[
							{"id":"REQ-1a-1","status":"validated","validation":[{"type":"test"}]}
						]}
					]}
				]}`,
			},
			wantTotal: 2, wantPassing: 2, wantValidation: 1, wantAvgDepth: 3,
		},
		{
			name: "requirements inline in index.json",
			files: map[string]string{
				"requirements/index.json": `{"requirements":[{"id":"REQ-1","status":"passed"}]}`,
			},
			wantTotal: 1, wantPassing: 1, wantValidation: 0, wantAvgDepth: 1,
		},
		{
			name: "module scan without index",
			files: map[string]string{
				"requirements/01-a/module.json": `{"requirements":[{"id":"REQ-1","status":"draft"}]}`,
			},
			wantTotal: 1, wantPassing: 0, wantValidation: 0, wantAvgDepth: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tt.files {
				writeFile(t, root, rel, content)
			}

			sig, err := collectRequirements(t, root)
			if err != nil {
				t.Fatal(err)
			}
			if !sig.Collected {
				t.Fatal("want Collected=true")
			}
			if sig.Total != tt.wantTotal || sig.Passing != tt.wantPassing {
				t.Fatalf("total/passing = %d/%d, want %d/%d", sig.Total, sig.Passing, tt.wantTotal, tt.wantPassing)
			}
			if sig.WithValidation != tt.wantValidation {
				t.Fatalf("withValidation = %d, want %d", sig.WithValidation, tt.wantValidation)
			}
			if math.Abs(sig.AvgDepth-tt.wantAvgDepth) > 1e-9 {
				t.Fatalf("avgDepth = %v, want %v", sig.AvgDepth, tt.wantAvgDepth)
			}
		})
	}
}

func TestRequirementsSyncMetadataOverridesRegistry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "requirements/01-a/module.json", `{"requirements":[
		{"id":"REQ-1","status":"passed"},
		{"id":"REQ-2","status":"draft"},
		{"id":"REQ-3","status":"failed"}
	]}`)
	// Sync wins in both directions: demote REQ-1, promote REQ-2.
	writeFile(t, root, "coverage/requirements-sync/latest.json", `{
		"requirements":{
			"REQ-1":{"status":"failed"},
			"REQ-2":{"status":"passed"}
		}}`)

	sig, err := collectRequirements(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if sig.Total != 3 || sig.Passing != 1 {
		t.Fatalf("total/passing = %d/%d, want 3/1", sig.Total, sig.Passing)
	}
}

func TestRequirementsSyncPathOrderAndUnusableCandidates(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "requirements/01-a/module.json",
		`{"requirements":[{"id":"REQ-1","status":"draft"}]}`)
	// The real coverage/sync/latest.json is a sync-run log with no
	// statuses; it must be skipped rather than masking the registry.
	writeFile(t, root, "coverage/sync/latest.json",
		`{"synced_at":"2026-06-03T22:04:11Z","files_updated":0}`)
	writeFile(t, root, "coverage/requirements-sync.json",
		`{"requirements":{"REQ-1":{"status":"passed"}}}`)

	sig, err := collectRequirements(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if sig.Passing != 1 {
		t.Fatalf("passing = %d, want 1 (requirements-sync.json applied)", sig.Passing)
	}
}

func TestRequirementsTargets(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		wantTotal   int
		wantPassing int
	}{
		{
			name: "sync operational_targets preferred, current writer shape",
			files: map[string]string{
				"requirements/01-a/module.json": `{"requirements":[
					{"id":"REQ-1","prd_ref":"OT-P0-001","status":"draft"}
				]}`,
				"coverage/requirements-sync/latest.json": `{
					"operational_targets":[
						{"id":"OT-P0-001","status":"complete","requirement_ids":["REQ-1"],"completion_rate":100},
						{"id":"OT-P0-002","status":"pending","requirement_ids":["REQ-2"],"completion_rate":0},
						{"id":"OT-P0-003","status":"in_progress","requirement_ids":["REQ-3"],"completion_rate":75}
					]}`,
			},
			// complete passes; pending 0% fails; 75% completion passes.
			wantTotal: 3, wantPassing: 2,
		},
		{
			name: "sync operational_targets legacy shape with counts",
			files: map[string]string{
				"requirements/01-a/module.json": `{"requirements":[{"id":"REQ-1","status":"draft"}]}`,
				"coverage/requirements-sync/latest.json": `{
					"operational_targets":[
						{"target_id":"OT-P0-001","status":"pending","counts":{"total":4,"complete":3}},
						{"key":"OT-P0-002","status":"pending","counts":{"total":4,"complete":1}}
					]}`,
			},
			wantTotal: 2, wantPassing: 1,
		},
		{
			name: "fallback grouping by prd_ref at the 50% threshold",
			files: map[string]string{
				"requirements/01-a/module.json": `{"requirements":[
					{"id":"REQ-1","prd_ref":"OT-P0-001","status":"passed"},
					{"id":"REQ-2","prd_ref":"OT-P0-001","status":"draft"},
					{"id":"REQ-3","prd_ref":"OT-P0-002","status":"draft"},
					{"id":"REQ-4","prd_ref":"OT-P0-002","status":"draft"},
					{"id":"REQ-5","operational_target_id":"OT-P0-003","status":"passed"},
					{"id":"REQ-6","status":"passed"}
				]}`,
			},
			// OT-1: 1/2 = 50% passes; OT-2: 0/2 fails; OT-3: 1/1 passes.
			// REQ-6 has no target link and joins no group.
			wantTotal: 3, wantPassing: 2,
		},
		{
			name: "no targets derivable",
			files: map[string]string{
				"requirements/01-a/module.json": `{"requirements":[{"id":"REQ-1","status":"passed"}]}`,
			},
			wantTotal: 0, wantPassing: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			for rel, content := range tt.files {
				writeFile(t, root, rel, content)
			}

			sig, err := collectRequirements(t, root)
			if err != nil {
				t.Fatal(err)
			}
			if sig.TargetsTotal != tt.wantTotal || sig.TargetsPassing != tt.wantPassing {
				t.Fatalf("targets = %d/%d, want %d/%d",
					sig.TargetsPassing, sig.TargetsTotal, tt.wantPassing, tt.wantTotal)
			}
		})
	}
}

func TestRequirementsImportDedupeAndStaleImports(t *testing.T) {
	root := t.TempDir()
	// The import and the recursive module scan find the same file; it must
	// count once. A stale import to a missing file is skipped silently.
	writeFile(t, root, "requirements/index.json",
		`{"imports":["01-a/module.json","99-gone/module.json"]}`)
	writeFile(t, root, "requirements/01-a/module.json",
		`{"requirements":[{"id":"REQ-1","status":"passed"}]}`)

	sig, err := collectRequirements(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if sig.Total != 1 {
		t.Fatalf("total = %d, want 1 (deduplicated)", sig.Total)
	}
}

func TestRequirementsChildCycleDoesNotHang(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "requirements/01-a/module.json", `{"requirements":[
		{"id":"REQ-1","status":"passed","children":["REQ-2"]},
		{"id":"REQ-2","status":"passed","children":["REQ-1"]}
	]}`)

	sig, err := collectRequirements(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if sig.Total != 2 {
		t.Fatalf("total = %d, want 2", sig.Total)
	}
}

func TestRequirementsMalformedModuleErrorMentionsFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "requirements/01-a/module.json", `not json`)

	_, err := collectRequirements(t, root)
	if err == nil || !strings.Contains(err.Error(), "module.json") {
		t.Fatalf("err = %v, want decode error naming the module file", err)
	}
}
