package proposals

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

// The UI mirrors this package's op vocabulary in
// ui/src/types/proposal.ts. That mirror is load-bearing: the decision stream
// picks a renderer per op, and an op the UI does not know about renders as a
// card with no payload — which is how operators came to approve mutations
// whose contents were never displayed.
//
// This test makes the drift a build failure at the point it is introduced,
// rather than a blank card discovered in production. The UI side has the
// matching assertion (every op maps to an archetype) in
// lib/mutation-archetypes.test.ts.
const uiProposalTypesPath = "../../../ui/src/types/proposal.ts"

var proposalOpsBlock = regexp.MustCompile(`(?s)export const PROPOSAL_OPS:[^=]*=\s*\[(.*?)\]`)

var quotedOp = regexp.MustCompile(`"([a-z_]+)"`)

func readUIProposalOps(t *testing.T) []string {
	t.Helper()
	path := filepath.Clean(uiProposalTypesPath)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	block := proposalOpsBlock.FindSubmatch(source)
	if block == nil {
		t.Fatalf("PROPOSAL_OPS array not found in %s; the UI op mirror must stay machine-readable", path)
	}
	matches := quotedOp.FindAllSubmatch(block[1], -1)
	ops := make([]string, 0, len(matches))
	for _, match := range matches {
		ops = append(ops, string(match[1]))
	}
	if len(ops) == 0 {
		t.Fatalf("PROPOSAL_OPS in %s parsed to zero ops", path)
	}
	return ops
}

func TestUIMirrorsEveryServerOp(t *testing.T) {
	uiOps := make(map[string]bool)
	for _, op := range readUIProposalOps(t) {
		uiOps[op] = true
	}

	var missing []string
	for _, op := range AllOps() {
		if !uiOps[string(op)] {
			missing = append(missing, string(op))
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("ops accepted by the server but absent from %s: %v\n"+
			"Add them to ProposalOp and PROPOSAL_OPS, then map each to an archetype in ui/src/lib/mutation-archetypes.ts.",
			uiProposalTypesPath, missing)
	}
}

func TestUIDeclaresNoOpTheServerRejects(t *testing.T) {
	serverOps := make(map[string]bool)
	for _, op := range AllOps() {
		serverOps[string(op)] = true
	}

	var extra []string
	for _, op := range readUIProposalOps(t) {
		if !serverOps[op] {
			extra = append(extra, op)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("ops declared in %s that the server does not accept: %v\n"+
			"An op the apply layer rejects must not be offered as reviewable.",
			uiProposalTypesPath, extra)
	}
}

// AllOps is the vocabulary rendered into agent prompts as well as the UI, so a
// duplicate entry silently doubles an op in both.
func TestAllOpsHasNoDuplicates(t *testing.T) {
	seen := make(map[Op]bool, len(AllOps()))
	for _, op := range AllOps() {
		if seen[op] {
			t.Errorf("AllOps() lists %q more than once", op)
		}
		seen[op] = true
	}
}

// Every goal op must also be an accepted op; ValidateGoal draws from GoalOps
// while the apply layer draws from AllOps, and a goal op missing from AllOps
// would validate and then fail to apply.
func TestGoalOpsAreASubsetOfAllOps(t *testing.T) {
	all := make(map[Op]bool, len(AllOps()))
	for _, op := range AllOps() {
		all[op] = true
	}
	for _, op := range GoalOps() {
		if !all[op] {
			t.Errorf("GoalOps() includes %q, which AllOps() does not accept", op)
		}
	}
}
