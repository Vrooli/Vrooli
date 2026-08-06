package actspace

import (
	"context"
	"testing"

	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
)

func TestReportsPerCellBindingResolution(t *testing.T) { // [REQ:PRT-P0-008]
	registry, _ := liveActDefinition(t)
	verdicts := registry.ResolveActCells(context.Background(), []*bindingsv1.ActCell{{Id: "bound", Operations: []string{"program-runtime/sessions/list"}, AuthoredStatus: "NOW"}})
	if len(verdicts) != 1 || verdicts[0].GetVerdict() != bindingsv1.ActVerdict_ACT_VERDICT_NOW || !verdicts[0].GetAudited() {
		t.Fatalf("bound verdict=%v", verdicts)
	}
}

func TestPartiallyBoundCellIsInReach(t *testing.T) { // [REQ:PRT-P0-008]
	registry, _ := liveActDefinition(t)
	verdicts := registry.ResolveActCells(context.Background(), []*bindingsv1.ActCell{{Id: "partial", Operations: []string{"program-runtime/sessions/list", "missing-owner"}, AuthoredStatus: "NOW"}})
	if len(verdicts) != 1 || verdicts[0].GetVerdict() != bindingsv1.ActVerdict_ACT_VERDICT_IN_REACH || len(verdicts[0].GetReasons()) == 0 {
		t.Fatalf("partial verdict=%v", verdicts)
	}
}

func TestUnresolvableCellKeepsAuthoredStatus(t *testing.T) { // [REQ:PRT-P0-008]
	registry, _ := liveActDefinition(t)
	verdicts := registry.ResolveActCells(context.Background(), []*bindingsv1.ActCell{{Id: "unresolved", Operations: []string{"missing-owner"}, AuthoredStatus: "AUTHORED"}})
	if len(verdicts) != 1 || verdicts[0].GetVerdict() != bindingsv1.ActVerdict_ACT_VERDICT_AUTHORED || verdicts[0].GetAuthoredStatus() != "AUTHORED" {
		t.Fatalf("unresolved verdict=%v", verdicts)
	}
}
