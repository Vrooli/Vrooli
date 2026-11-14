package orchestration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/orchestration/testutil"
)

// A refused authorization request has to tell the requesting agent which
// component could not prove its session and what to do next. The bare
// "live launcher session binding is unavailable" did neither, and it hid the
// owner's categorized reason that the proof reader had already classified.
func TestRequestAuthorizationNamesWhyTheLiveSessionCouldNotBeProved(t *testing.T) {
	categorized := func(category string) error {
		return fmt.Errorf("%w: %w", errBindingProofUnavailable, bindingProofReason{category: category})
	}
	for _, tc := range []struct {
		name string
		err  error
		want []string
	}{
		{"scope membership", categorized("scope_membership_unverified"), []string{"scope_membership_unverified", "launcher process", "relaunched"}},
		{"lease gone", categorized("lease_unavailable"), []string{"lease_unavailable", "vrooli agent list", "retry"}},
		{"web console", categorized("web_console_session_unverified"), []string{"Web Console", "retry"}},
		{"registry schema", categorized("runtime_registry"), []string{"runtime registry", "operator"}},
		{"timeout", errBindingProofTimeout, []string{"did not answer", "retry"}},
		{"uncategorized failure", errBindingProofUnavailable, []string{"without a recognised reason", "operator"}},
		{"proof for another tuple", errBindingProofMismatch, []string{"different session", "vrooli agent list --json"}},
		{"unclassified error", errors.New("secret subprocess stderr /home/x/.token"), []string{"unusable", "retry"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repos, events, _ := testutil.SetupTestReposWithDB(t, db)
			o := New(nil, nil, repos.Runs,
				WithEvents(events), WithIdentitySecret([]byte("binding-failure-secret")),
				WithAuthorizations(database.NewAuthorizationRepository(db)),
				WithSessionBindingProofReader(SessionBindingProofReaderFunc(func(context.Context, SessionBindingRequest) (SessionBindingProof, error) {
					return SessionBindingProof{}, tc.err
				})))
			_, err := o.RequestAuthorization(ctx, AuthorizationRequest{
				PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "claude-code-host-session",
				Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:*"}, DurationSeconds: 3600,
			})
			if err == nil {
				t.Fatal("an unprovable session created a consent request")
			}
			message := err.Error()
			if !strings.Contains(message, "live launcher session binding is unavailable") {
				t.Fatalf("refusal no longer names the failed requirement: %s", message)
			}
			for _, want := range tc.want {
				if !strings.Contains(message, want) {
					t.Fatalf("refusal %q does not say %q", message, want)
				}
			}
			if strings.Contains(message, "secret subprocess stderr") || strings.Contains(message, ".token") {
				t.Fatalf("refusal leaked raw proof-reader output: %s", message)
			}
			grants, err := o.authorizations.List(ctx, os.Getpid())
			if err != nil || len(grants) != 0 {
				t.Fatalf("refused request left a consent row: grants=%d err=%v", len(grants), err)
			}
		})
	}
}

func TestSessionBindingFailureTextCoversEveryClassifiedCategory(t *testing.T) {
	// classifiedBindingProofFailure admits exactly these categories; each must
	// have actionable text rather than the generic fallback.
	for _, category := range []string{
		"kernel_binding_unverified", "launch_owner_unverified", "lease_changed_during_proof",
		"lease_identity_mismatch", "lease_time_unverified", "lease_unavailable",
		"process_changed_during_proof", "process_incarnation_unverified", "process_started_after_lease",
		"proof_cancelled", "proof_reader_unavailable", "run_binding_missing", "run_process_mismatch",
		"scope_membership_unverified", "selector_missing", "session_scope_unverified",
		"web_console_session_unverified", "runtime_home", "runtime_registry",
	} {
		if _, ok := sessionBindingFailureNext[category]; !ok {
			t.Errorf("category %q has no actionable next step", category)
		}
		err := classifiedBindingProofFailure("Error: identity binding proof unavailable: " + category)
		if got := describeSessionBindingFailure(err); strings.Contains(got, "retry once, and ask the operator if it repeats") {
			t.Errorf("category %q fell through to the generic text: %s", category, got)
		}
	}
}
