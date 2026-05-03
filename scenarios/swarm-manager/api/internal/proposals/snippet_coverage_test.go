package proposals_test

import (
	"strings"
	"testing"

	"swarm-manager/internal/operatingmode/promptcatalog"
	"swarm-manager/internal/proposals"
)

// TestSharedSnippet_CoversAllProposalOps reverse-couples the shared
// reconcile/feedback snippet to the canonical op list in `proposals`. The
// test lives here (not in operatingmode/promptcatalog) to avoid an import
// cycle: proposals' transitive deps reach back to operatingmode, which
// imports operatingmode/promptcatalog. Tests in operatingmode that import
// proposals would close the loop. Hosting the test in `package proposals_test`
// (an external test package alongside `package proposals`) keeps the
// dependency one-way: proposals_test → both proposals and
// operatingmode/promptcatalog, but neither of those imports back into the
// other.
//
// Invariant: every op the apply path accepts MUST appear in the snippet's
// supported-ops table so agents and operators see the same surface. Adding
// a new op in proposals/types.go fails this test until the snippet is
// extended — exactly the reverse-coupling we want.
func TestSharedSnippet_CoversAllProposalOps(t *testing.T) {
	snippet := promptcatalog.BacklogSyncProposalSnippet()
	for _, op := range proposals.AllOps() {
		needle := "`" + string(op) + "`"
		if !strings.Contains(snippet, needle) {
			t.Errorf("BacklogSyncProposalSnippet missing op %q (looked for %q) — add this op to the snippet's supported-ops table", op, needle)
		}
	}
}
