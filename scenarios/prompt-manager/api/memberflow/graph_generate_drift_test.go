package memberflow

import (
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
)

// TestCheckedInOperatingGraphsMatchTheGenerator is the drift gate for Phase 9.
//
// Each team's `## Operating Graph` block and `## Topic Catalog` table are
// rendered from topics.json, team.json, and that team's graph-presentation.json.
// A hand edit inside either is drift by definition: the declarations are the
// source, and a generated block that disagrees with them is the duplication this
// phase removed. Regenerate with:
//
//	go run ./cmd/gen-operating-graph <repo-root> --apply
func TestCheckedInOperatingGraphsMatchTheGenerator(t *testing.T) {
	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("resolve test filename")
	}
	pkgDir := filepath.Dir(filename)
	repoRoot := filepath.Clean(filepath.Join(pkgDir, "..", "..", "..", ".."))
	storeDir := filepath.Clean(filepath.Join(pkgDir, "..", "..", "store"))

	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("load members: %v", err)
	}
	contracts, err := LoadAllTeamContracts(storeDir)
	if err != nil {
		t.Fatalf("load team contracts: %v", err)
	}
	runtime := OperatingGraphRuntime{RepoRoot: repoRoot, StoreDir: storeDir, Members: members, Contracts: contracts}

	blocks, err := LoadOperatingGraphBlocks(repoRoot)
	if err != nil {
		t.Fatalf("load blocks: %v", err)
	}

	checked := 0
	for _, block := range blocks {
		if block.Metadata.Mode != OperatingGraphModeContract || block.Metadata.Team == "" {
			continue
		}
		team := block.Metadata.Team
		presentation, err := LoadGraphPresentation(storeDir, team)
		if err != nil {
			t.Errorf("%s: load presentation: %v", team, err)
			continue
		}
		generated, err := GenerateOperatingGraphBlock(GenerateOperatingGraphInput{
			TeamID: team, Runtime: runtime, Presentation: presentation,
		})
		if err != nil {
			t.Errorf("%s: generate: %v", team, err)
			continue
		}
		checked++

		// Compare the parsed shape rather than the bytes: what must not drift
		// is the set of nodes and edges the declarations imply, not the
		// whitespace of the rendering.
		lines := strings.Split(strings.TrimSpace(generated), "\n")
		want, err := ParseOperatingMermaid(team, lines[1:len(lines)-1], 1)
		if err != nil {
			t.Errorf("%s: generated block does not parse: %v", team, err)
			continue
		}
		if diff := describeGraphNodeDiff(want, block.Graph); diff != "" {
			t.Errorf("%s operating graph has drifted from its declarations: %s\n"+
				"Regenerate with: go run ./cmd/gen-operating-graph <repo-root> --apply", team, diff)
		}
	}
	if checked == 0 {
		t.Fatal("no contract graph blocks were checked; the discovery is broken, not clean")
	}
}

// describeGraphNodeDiff names the first node present in one graph and absent
// from the other, so a failure points at a topic rather than at a byte offset.
func describeGraphNodeDiff(want, have OperatingGraph) string {
	index := func(g OperatingGraph) map[string]bool {
		out := map[string]bool{}
		for _, node := range g.Nodes {
			if node.Kind != "" && node.Value != "" {
				out[string(node.Kind)+":"+node.Value] = true
			}
		}
		return out
	}
	wantSet, haveSet := index(want), index(have)
	var missing, extra []string
	for key := range wantSet {
		if !haveSet[key] {
			missing = append(missing, key)
		}
	}
	for key := range haveSet {
		if !wantSet[key] {
			extra = append(extra, key)
		}
	}
	switch {
	case len(missing) > 0 && len(extra) > 0:
		return "declared but not drawn: " + strings.Join(missing, ", ") + "; drawn but not declared: " + strings.Join(extra, ", ")
	case len(missing) > 0:
		return "declared but not drawn: " + strings.Join(missing, ", ")
	case len(extra) > 0:
		return "drawn but not declared: " + strings.Join(extra, ", ")
	}
	return ""
}
