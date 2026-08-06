package actspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/repo-contract-go"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	internalbindings "program-runtime/internal/bindings"
)

func liveActDefinition(t *testing.T) (*internalbindings.Registry, *spacedoc.SpaceDefinition) {
	t.Helper()
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatal(err)
	}
	registry, err := internalbindings.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "scenarios", "program-runtime", "docs", "spaces", "act-space.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := spacedoc.Parse(spacedoc.ProjectionAct, data)
	if err != nil {
		t.Fatal(err)
	}
	return registry, definition
}

func TestEveryCellResolvesAgainstLiveRegistry(t *testing.T) { // [REQ:PRT-P1-009]
	registry, definition := liveActDefinition(t)
	verdicts := Audit(context.Background(), registry, definition)
	if len(verdicts) != len(definition.Cells) || len(verdicts) != 28 {
		t.Fatalf("verdicts=%d cells=%d, want the complete 28-cell denominator", len(verdicts), len(definition.Cells))
	}
	for _, verdict := range verdicts {
		if verdict == nil || !verdict.GetAudited() {
			t.Fatalf("cell was not audited: %v", verdict)
		}
		if verdict.GetVerdict() == bindingsv1.ActVerdict_ACT_VERDICT_UNSPECIFIED {
			t.Fatalf("cell has no decided verdict: %v", verdict)
		}
	}
}

func TestConfidenceReflectsAuditCoverage(t *testing.T) { // [REQ:PRT-P1-009]
	full := []*bindingsv1.ActCellVerdict{{Audited: true}, {Audited: true}}
	if got := Confidence(full); got != spacedoc.ConfidencePartial {
		t.Fatalf("full audit confidence=%q, want partial", got)
	}
	incomplete := []*bindingsv1.ActCellVerdict{{Audited: true}, {Audited: false}}
	if got := Confidence(incomplete); got != spacedoc.ConfidenceSketch {
		t.Fatalf("incomplete audit confidence=%q, want sketch", got)
	}
}
