package orchestration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// A consent request that names its packet shows the operator what approving
// starts in the packet's own words, carries the packet path to the Hub, and
// keeps the path itself out of the quoted reason.
func TestConsentPageShowsWhatThePacketStarts(t *testing.T) {
	dir := t.TempDir()
	brief := "# W10 — Start three windows\n\n## Destination\n\nOne consent.\n\n## What each commission builds\n\n| Commission | What it builds |\n|---|---|\n| `models-monetization-music` | Complete original songs |\n\n## On approval: what executes\n\nOwner A launches the built commission.\n\n## Technical details\n\nvrooli agent authorize ...\n"
	if err := os.WriteFile(filepath.Join(dir, "BRIEF.md"), []byte(brief), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	g := &domain.AuthorizationGrant{
		ID: uuid.New(), RunID: uuid.New(), State: domain.AuthorizationPending, Policy: "swarm-bookkeeping",
		Targets: []string{"swarm-manager:backlog/execute/obsolete"}, DurationSeconds: 7 * 24 * 3600, HarnessKind: "claude-code",
		CreatedAt: now, RequestExpiresAt: now.Add(24 * time.Hour),
		RequesterContext: &domain.AuthorizationRequesterContext{BigPicture: "Launch three research windows.\nPacket: " + dir},
	}
	f := consentFacts(g, "https://agent-manager.example.test/api/v1/authorizations/"+g.ID.String()+"/approve", "")
	if f.Packet != dir {
		t.Fatalf("packet path = %q, want %q", f.Packet, dir)
	}
	for _, must := range []string{"What each commission builds", "Complete original songs", "On approval: what executes", "Owner A launches the built commission.", "Launch three research windows."} {
		if !strings.Contains(f.Explainer, must) {
			t.Fatalf("the page does not show %q", must)
		}
	}
	for _, mustNot := range []string{"Packet: " + dir, "vrooli agent authorize"} {
		if strings.Contains(f.Explainer[:strings.Index(f.Explainer, "The exact permission")], mustNot) {
			t.Fatalf("the reading path shows %q", mustNot)
		}
	}
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
}
