package authn_test

// KEEP-SYNC boundary test. It guards against a future over-generalization of the
// operator-decision consolidation (the shared operatordecision rail): the gates
// enumerated here are the BASE OF TRUST and MUST stay synchronous-human-live.
// None of them may be converted to the async `*.decision_requested.v1` rail.
//
// The rule they encode (see docs/concepts/AUTHORIZATION-KEEP-SYNC-BOUNDARY.md):
// a gate stays synchronous-human-live iff it (a) mints/revokes a machine
// identity, a standing grant, or the live-human approval itself, or (b) releases
// raw secret material / opens an unbounded interactive session. Routing the
// issuance of a credential through the very rail that credential would authorize
// is CIRCULAR — that is exactly what this test forbids.
//
// Scope — assert only what EXISTS in code, state the rest as doctrine in the doc.
// This file exercises the base-of-trust gates that live in packages/api-core
// (importable here): the human-only guards (`RequireHuman`,
// `RequireIndependentHuman`, the `RequireOperation` agent branch), the pure
// standing-grant decision (`AuthorizeStandingGrant`), and the Cloudflare Access
// verifier contract (`cloudflareaccess.NewVerifier` forces RequireUser=true and
// refuses service tokens). The scenario-local gates on the same boundary —
// service-principal issuance (scenario-authenticator), standing-grant /
// authorization mint-and-revoke (agent-manager), and the synchronous human
// release/secret check (deployment-manager's `RequireIndependentHuman`, a
// Mechanism-2 use, NOT a standalone function) — are enumerated in the boundary
// doc and pinned by their own scenario suites; they cannot be imported here
// without a dependency inversion. There is deliberately NO dedicated break-glass
// or SSH/interactive-terminal gate in code today; those are DOCTRINE in the doc
// so a future implementer keeps them synchronous — this test invents no stub for
// them. The property proven here is the load-bearing half: each api-core gate
// REFUSES a non-human caller, and the authn package depends on no
// operatordecision publisher/raiser — so none of them can route a decision onto
// the rail.

import (
	"context"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/cloudflareaccess"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// agentPrincipal / servicePrincipal are the two non-human callers every
// base-of-trust gate must refuse. A verified agent even carries a plausible
// scope so the refusal is about the KIND of caller, not a missing scope.
func agentPrincipal() identity.Principal {
	return identity.Principal{Kind: identity.ActorAgent, Subject: "run-123", Verified: true, Scopes: []string{"agent-manager:write", "deployment-manager:write", "swarm-manager:write"}}
}

func servicePrincipal() identity.Principal {
	return identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write", "deployment-manager:write"}}
}

// TestKeepSyncBoundaryGatesRefuseNonHumanCallers enumerates the api-core
// base-of-trust gates and asserts each refuses an agent and a service principal.
// Each gate here is one that MUST NOT move to the decision rail; keeping the
// refusal pinned makes a future "just raise a decision instead" regression fail
// loudly in api-core's own suite.
func TestKeepSyncBoundaryGatesRefuseNonHumanCallers(t *testing.T) {
	cases := []struct {
		name string
		// why documents which base-of-trust clause (mint/revoke identity, approval
		// itself, secret release, interactive session) puts this gate on the
		// synchronous-human-live side of the boundary.
		why  string
		call func(ctx context.Context) error
	}{
		{
			name: "RequireHuman",
			why:  "the live-human approval itself — the root the whole rail is attested against",
			call: func(ctx context.Context) error { _, err := authn.RequireHuman(ctx); return err },
		},
		{
			name: "RequireIndependentHuman",
			why:  "a human decision that must not be captured by a co-presented agent credential",
			call: func(ctx context.Context) error { _, err := authn.RequireIndependentHuman(ctx); return err },
		},
		{
			name: "RequireOperation(human-only base op, no grant)",
			why:  "the standing-grant/authorization evaluation seat; a bare agent with no covering grant is refused (not asked)",
			call: func(ctx context.Context) error {
				// No RequestOperationAuthority in context and the agent carries no
				// OperationGrant, so RequireOperation falls to its agent branch and
				// refuses — it does NOT raise a decision.
				_, err := authn.RequireOperation(ctx, "agent-manager:write", "run.create", "agent-manager:*")
				return err
			},
		},
	}

	callers := []struct {
		name string
		p    identity.Principal
	}{
		{"agent", agentPrincipal()},
		{"service", servicePrincipal()},
	}

	for _, tc := range cases {
		for _, caller := range callers {
			t.Run(tc.name+"/"+caller.name, func(t *testing.T) {
				ctx := identity.WithPrincipal(context.Background(), caller.p)
				if err := tc.call(ctx); err == nil {
					t.Fatalf("%s must REFUSE a %s caller (base-of-trust gate: %s), but it admitted", tc.name, caller.name, tc.why)
				}
			})
		}
	}
}

// TestKeepSyncBoundaryIndependentHumanRefusesHumanBehindAgentProvenance pins the
// subtler half of the approval root: even a VERIFIED human is refused when the
// request arrived with agent provenance, so an agent can never borrow a human's
// approval by co-presenting it. If this ever "fell through to a decision" it
// would be the circular issuance the boundary forbids.
func TestKeepSyncBoundaryIndependentHumanRefusesHumanBehindAgentProvenance(t *testing.T) {
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true}
	ctx := identity.WithPrincipal(context.Background(), human)
	if _, err := authn.RequireIndependentHuman(ctx); err != nil {
		t.Fatalf("a clean verified human must be admitted by RequireIndependentHuman: %v", err)
	}
	// Same human, now behind agent provenance — must be refused.
	agentCtx := provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run"})
	if _, err := authn.RequireIndependentHuman(agentCtx); !errors.Is(err, authn.ErrHumanRequired) {
		t.Fatalf("RequireIndependentHuman must refuse a human carried behind agent provenance, got %v", err)
	}
}

// TestKeepSyncBoundaryStandingGrantAuthorizeIsPureDecision pins that the
// standing-grant AUTHORIZATION decision (the evaluator the rail's approvals mint
// INTO) is a pure synchronous function of (grant, limits, usage) — it takes no
// event Publisher and cannot itself raise a decision. A nil/empty grant is
// refused, so issuance can never be bootstrapped by asking the rail that the
// grant would authorize.
func TestKeepSyncBoundaryStandingGrantAuthorizeIsPureDecision(t *testing.T) {
	now := time.Now()
	// An empty grant allows nothing.
	if err := authn.AuthorizeStandingGrant(&identity.OperationGrant{}, authn.StandingGrantLimits{}, authn.StandingGrantUsage{}, "run.create", "agent-manager:*", now); err == nil {
		t.Fatal("an empty standing grant must authorize nothing")
	}
}

// TestKeepSyncBoundaryCloudflareAccessConstructsAsHumanProof pins that the
// Cf-Access verifier — the live-human proof the whole decision rail is attested
// against — constructs from the shared constructor. The constructor forces
// RequireUser=true internally (cloudflareaccess.go), and the verifier refuses
// service tokens and missing-email assertions at verify time; those refusals are
// pinned by cloudflareaccess's own suite and by notification-hub's access_test.go
// (which drives a signed assertion end to end). Here we assert only the
// reachable api-core fact: the human-proof verifier is a real, constructable gate
// (not a stub) that this boundary doc enumerates. The signed-token refusal is not
// re-driven here to avoid duplicating heavy JWKS fixtures.
func TestKeepSyncBoundaryCloudflareAccessConstructsAsHumanProof(t *testing.T) {
	if _, err := cloudflareaccess.NewVerifier(cloudflareaccess.Config{
		TeamDomain: "https://team.example.test", Audience: "aud", JWKSURL: "https://example.test/jwks",
	}); err != nil {
		t.Fatalf("Cf-Access human-proof verifier must construct: %v", err)
	}
}

// TestKeepSyncBoundaryAuthnTakesNoDecisionRail is the STRUCTURAL guard behind the
// whole boundary: the authn package (where the base-of-trust gates live) must not
// depend on the operator-decision rail at all. If a future change imported
// operatordecision (or any event publisher/raiser) into an authn gate, that gate
// could "ask instead of refuse" — the exact circular issuance this boundary
// forbids. Asserting the absence of the dependency makes that regression
// impossible to introduce silently.
func TestKeepSyncBoundaryAuthnTakesNoDecisionRail(t *testing.T) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read authn dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(path, "operatordecision") || strings.HasSuffix(path, "/eventbus") {
				t.Fatalf("%s imports %q — a base-of-trust gate must never depend on the operator-decision rail (it could ask instead of refuse)", name, path)
			}
		}
	}
}
