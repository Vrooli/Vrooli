package nextaction

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// The client decides whether a control warns "this dispatches an agent" by
// reading the Effect this package puts on every next-action projection. The
// mapping lives in ui/src/lib/action-semantics.ts.
//
// An effect the UI does not map falls through to a classification that
// promises no agent involvement, so a new effect would silently ship buttons
// that spend minutes of agent time while looking like an ordinary save. This
// test makes that drift fail here, where the effect is introduced.
//
// The sibling assertion for transition kinds lives in
// internal/transitions/uicontract_test.go.
const uiActionSemanticsPath = "../../../ui/src/lib/action-semantics.ts"

// Matches `case "agent_run":` inside the UI's effect mapping switch.
var uiMappedEffect = regexp.MustCompile(`case "([a-z_]+)":`)

func readUIMappedEffects(t *testing.T) map[string]struct{} {
	t.Helper()
	path := filepath.Clean(uiActionSemanticsPath)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	matches := uiMappedEffect.FindAllSubmatch(source, -1)
	if len(matches) == 0 {
		t.Fatalf("no `case \"effect\":` arms found in %s; the UI effect mapping must stay machine-readable", path)
	}
	mapped := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		mapped[string(match[1])] = struct{}{}
	}
	return mapped
}

func declaredEffects() []Effect {
	return []Effect{EffectNone, EffectStateChange, EffectAgentRun, EffectAgentSession}
}

func TestUIMapsEveryDeclaredEffect(t *testing.T) {
	mapped := readUIMappedEffects(t)
	var missing []string
	for _, effect := range declaredEffects() {
		if _, ok := mapped[string(effect)]; !ok {
			missing = append(missing, string(effect))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("effects %v are declared here but unmapped in %s; controls for those actions would render without the right consequence marker", missing, uiActionSemanticsPath)
	}
}

// EffectFor must only ever return a declared constant; a typo'd string literal
// would reach the client as an unmapped value and be treated as unknown.
func TestEffectForOnlyReturnsDeclaredEffects(t *testing.T) {
	declared := make(map[Effect]struct{}, len(declaredEffects()))
	for _, effect := range declaredEffects() {
		declared[effect] = struct{}{}
	}
	all := []ID{
		None, AcceptSuggestion, AuthorPlan, AcceptPlan, RepairPlan, ResolveDependencies,
		Review, ViewExecution, Run, Retry, Archive, Decide, DispatchFollowup,
		AuthorFollowup, PlanGoal, DefineCriteria, CloseOut, Chain,
		ID("an-action-nobody-declared"),
	}
	for _, id := range all {
		if _, ok := declared[EffectFor(id)]; !ok {
			t.Errorf("EffectFor(%q) returned undeclared effect %q", id, EffectFor(id))
		}
	}
}
