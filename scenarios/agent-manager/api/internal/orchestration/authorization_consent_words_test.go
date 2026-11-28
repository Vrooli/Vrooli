package orchestration

import (
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

func consentGrant(policy, target, purpose string, life time.Duration) *domain.AuthorizationGrant {
	now := time.Now().UTC()
	g := &domain.AuthorizationGrant{
		ID: uuid.New(), RunID: uuid.New(), State: domain.AuthorizationPending, Policy: policy, Targets: []string{target},
		DurationSeconds: int64(life / time.Second), HarnessKind: "claude-code", HarnessSession: "claude-code-5896e82fe2317d8d4fc2c4551a1a4a8c",
		CreatedAt: now, RequestExpiresAt: now.Add(7 * 24 * time.Hour),
	}
	if purpose != "" {
		g.RequesterContext = &domain.AuthorizationRequesterContext{BigPicture: purpose}
	}
	return g
}

func consentLink(g *domain.AuthorizationGrant) string {
	return "https://agent-manager.example.test/api/v1/authorizations/" + g.ID.String() + "/approve"
}

// The live failure this guards: a run-creation request on every item read as
// "run creation on every item", dropped the requester's reason, did not say
// what the agent was working on and recommended decline for a wildcard that
// run creation cannot avoid.
func TestConsentDecisionSaysWhoWhyWhatAndWeighsThePurpose(t *testing.T) {
	const purpose = "Codex Luna dry-run eval of the notification-hub skill (no decisions sent)"
	g := consentGrant("run-creation", "agent-manager:*", purpose, 24*time.Hour)
	f := consentFacts(g, consentLink(g), "Notification Hub skill eval")

	for _, bad := range []string{"every item", "run creation", "Run create"} {
		for name, text := range map[string]string{"question": f.Question, "subject": f.Subject, "big picture": f.BigPicture} {
			if strings.Contains(text, bad) {
				t.Errorf("%s still says %q: %s", name, bad, text)
			}
		}
	}
	if !strings.Contains(f.Question, "start new agent runs") || !strings.Contains(f.Question, "Notification Hub skill eval") {
		t.Errorf("the question must say what and which session: %s", f.Question)
	}
	if strings.Contains(f.Question, purpose) || strings.Contains(f.Subject, purpose) {
		t.Errorf("the requester's unverified words reached the push preview: %s / %s", f.Subject, f.Question)
	}
	for _, must := range []string{"“" + purpose + "”", "own words", "Claude or Codex run", "24 hours"} {
		if !strings.Contains(f.BigPicture, must) {
			t.Errorf("the cover lacks %q: %s", must, f.BigPicture)
		}
	}
	if f.Recommended != "approve" {
		t.Errorf("a stated purpose on a permission that cannot be narrowed, for 24 hours, should be approved: %s", f.RecommendationReason)
	}
	for _, must := range []string{"cannot be narrowed", "ends by itself after 24 hours", purpose, "not checked"} {
		if !strings.Contains(f.RecommendationReason, must) {
			t.Errorf("the recommendation does not weigh %q: %s", must, f.RecommendationReason)
		}
	}
	for _, must := range []string{"What it says it is doing", purpose, "Notification Hub skill eval", "Start a new Claude or Codex run", "Stop, change or delete runs that already exist", "only comes in this wide form"} {
		if !strings.Contains(f.Explainer, must) {
			t.Errorf("the page lacks %q", must)
		}
	}
	if problems := operatordecision.CoverProblems(f); len(problems) > 0 {
		t.Errorf("the cover fails a cold reader: %v", problems)
	}
	if err := f.Validate(); err != nil {
		t.Error(err)
	}
}

func TestConsentRecommendationDeclinesWhatItCannotJustify(t *testing.T) {
	cases := []struct {
		name, policy, target, purpose string
		life                          time.Duration
		want                          string
		reasonHas                     []string
	}{
		{"no reason on a wide-only permission", "run-creation", "agent-manager:*", "", 24 * time.Hour, "decline", []string{"gives no reason", "what it is for"}},
		{"week-long wide-only permission", "run-creation", "agent-manager:*", "eval", 7 * 24 * time.Hour, "decline", []string{"limit is time", "24 hours"}},
		{"a wildcard that could be narrowed", "agent-management", "agent-manager:run/*", "tidy up", time.Hour, "decline", []string{"naming only the ones it needs"}},
		{"exact target", "agent-management", "agent-manager:run/abc", "tidy up", time.Hour, "approve", []string{"only what it names", "tidy up"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := consentGrant(c.policy, c.target, c.purpose, c.life)
			f := consentFacts(g, consentLink(g), "")
			if f.Recommended != c.want {
				t.Fatalf("recommended = %s, want %s: %s", f.Recommended, c.want, f.RecommendationReason)
			}
			for _, must := range c.reasonHas {
				if !strings.Contains(f.RecommendationReason, must) {
					t.Errorf("reason lacks %q: %s", must, f.RecommendationReason)
				}
			}
			if c.purpose == "" && !strings.Contains(f.BigPicture, "gave no reason") {
				t.Errorf("a request without a reason must say so: %s", f.BigPicture)
			}
		})
	}
}

// With no run label the decision still names the harness and which session.
func TestConsentDecisionNamesTheSessionWithoutALabel(t *testing.T) {
	g := consentGrant("run-creation", "agent-manager:*", "eval", time.Hour)
	f := consentFacts(g, consentLink(g), "")
	if !strings.Contains(f.BigPicture, "session ending "+g.HarnessSession[len(g.HarnessSession)-8:]) {
		t.Fatalf("the session is not identified: %s", f.BigPicture)
	}
}
