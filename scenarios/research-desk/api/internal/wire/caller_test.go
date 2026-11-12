package wire_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"

	"research-desk/handlers/activityapi"
	"research-desk/handlers/launchapi"
	"research-desk/handlers/operatorapi"
	"research-desk/internal/wire"
)

var (
	agent = provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-7"}
	// An agent identity that was offered but could not be verified.
	unverified = provenance.Provenance{Actor: provenance.ActorOperator, VerificationStatus: provenance.VerificationUnavailable}
)

// Who may act for the operator.
func TestOnlyACallerWithoutAnAgentOrServiceIdentityActsForTheOperator(t *testing.T) {
	bg := context.Background()
	for name, ctx := range map[string]context.Context{
		"verified agent":          provenance.NewContext(bg, agent),
		"unverifiable agent":      provenance.NewContext(bg, unverified),
		"service principal":       identity.WithPrincipal(bg, identity.Principal{Kind: identity.ActorService, Subject: "agent-manager", Verified: true}),
		"agent principal (authn)": identity.WithPrincipal(bg, identity.Principal{Kind: identity.ActorAgent, Subject: "run-7", Owner: "operator", Verified: true}),
	} {
		if err := wire.Caller(ctx); !errors.Is(err, wire.ErrNotOperator) {
			t.Errorf("%s: may act for the operator (%v)", name, err)
		}
	}
	for name, ctx := range map[string]context.Context{
		"local shell, no token": provenance.NewContext(bg, provenance.Provenance{Actor: provenance.ActorOperator, VerificationStatus: provenance.VerificationAbsent}),
		"verified human":        identity.WithPrincipal(bg, identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true}),
	} {
		if err := wire.Caller(ctx); err != nil {
			t.Errorf("%s: refused: %v", name, err)
		}
	}
}

// Each procedure that acts for the operator refuses an agent through the
// mounted module, with the reason and what to do instead.
func TestEveryOperatorActionRefusesAnAgent(t *testing.T) {
	want := []string{"TestLaunch", "Answer", "AddEntry", "EnableSteering", "AddVerdict", "SetCourier", "TestCourier", "MarkSeen"}
	router := mux.NewRouter()
	launchapi.Module(launchapi.Deps{}).Mount(router)
	operatorapi.Module(operatorapi.Deps{}).Mount(router)
	asAgent := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r.WithContext(provenance.NewContext(r.Context(), agent)))
	})
	var got []string
	for _, procedure := range append(slices.Clone(launchapi.OperatorProcedures), operatorapi.OperatorProcedures...) {
		got = append(got, procedure[strings.LastIndex(procedure, "/")+1:])
		req := httptest.NewRequest(http.MethodPost, procedure, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		asAgent.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), wire.OperatorAction) || !strings.Contains(rec.Body.String(), "run-7") {
			t.Errorf("%s: %d %s", procedure, rec.Code, rec.Body.String())
		}
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("operator actions = %v, want %v (reads, Build and Preflight stay open)", got, want)
	}
}

func TestResearchOperationRequiresPairedLiveGrantCoverage(t *testing.T) {
	for _, tc := range []struct {
		name, operation, target string
		expired, want           bool
	}{
		{"launch", "research.launch", "research-desk:commission/new", false, true},
		{"other target", "research.launch", "research-desk:commission/other", false, false},
		{"cross-rule control", "research.control", "research-desk:commission/new", false, false},
		{"superseded control", "research.control", "research-desk:commission/old", false, true},
		{"expired", "research.launch", "research-desk:commission/new", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expires := time.Now().Add(time.Hour)
			if tc.expired {
				expires = time.Now().Add(-time.Hour)
			}
			p := identity.Principal{Kind: identity.ActorAgent, Subject: "run-7", Verified: true, Scopes: []string{"research-desk:write"}, Grant: &identity.OperationGrant{
				ID: "approved", Approver: "operator", ExpiresAt: expires,
				Rules: []identity.OperationRule{
					{Operations: []string{"research.launch"}, Targets: []string{"research-desk:commission/new"}},
					{Operations: []string{"research.control"}, Targets: []string{"research-desk:commission/old"}},
				},
			}}
			ctx := identity.WithPrincipal(provenance.NewContext(context.Background(), agent), p)
			err := wire.ResearchOperation(ctx, tc.operation, strings.TrimPrefix(tc.target, "research-desk:commission/"))
			if (err == nil) != tc.want {
				t.Fatalf("authorization = %v, want allowed %v", err, tc.want)
			}
		})
	}
	ctx := provenance.NewContext(context.Background(), agent)
	err := wire.ResearchOperation(ctx, "research.launch", "new")
	if !errors.Is(err, authn.ErrOperation) || !strings.Contains(authn.RequestAuthority(err), "--policy research-management --target research-desk:commission/new") {
		t.Fatalf("missing grant must provide the canonical request: %v", err)
	}
	// Availability access cannot erase known agent attribution.
	ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Verified: true, Scopes: []string{"*"}})
	if wire.ResearchOperation(ctx, "research.launch", "new") == nil {
		t.Fatal("ambient human principal authorized the agent")
	}
}

func TestDelegatedLaunchAndControlExposeCanonicalApprovalRequest(t *testing.T) {
	router := mux.NewRouter()
	launchapi.Module(launchapi.Deps{}).Mount(router)
	for _, method := range []string{"Launch", "Control"} {
		req := httptest.NewRequest(http.MethodPost, "/vrooli.research_desk.v1.launch.LaunchService/"+method, strings.NewReader(`{"commissionId":"new"}`))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(provenance.NewContext(req.Context(), agent))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "--policy research-management --target research-desk:commission/new") {
			t.Fatalf("%s must return an actionable shared refusal before effects: %d %s", method, rec.Code, rec.Body.String())
		}
	}
}

// The CLI manifest marks a command runnable by an agent exactly when the API
// lets an agent call it, so the two cannot drift: each bound procedure is
// called as an agent with no grant, and a refusal must be run_eligible false.
func TestTheManifestMarksEveryRefusedCommandNotRunnable(t *testing.T) {
	data, err := os.ReadFile("../../../cli/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Groups []struct {
			Commands []struct {
				Name    string `json:"name"`
				Binding struct {
					Service string `json:"service"`
					Method  string `json:"method"`
				} `json:"binding"`
				Governance struct {
					RunEligible bool `json:"run_eligible"`
				} `json:"governance"`
			} `json:"commands"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	activityapi.Module(activityapi.Deps{}).Mount(router)
	launchapi.Module(launchapi.Deps{}).Mount(router)
	operatorapi.Module(operatorapi.Deps{}).Mount(router)
	refused := func(procedure string) (refused bool) {
		// A handler with no dependencies panics once it is past the
		// refusal; that call was let through.
		defer func() {
			if recover() != nil {
				refused = false
			}
		}()
		req := httptest.NewRequest(http.MethodPost, procedure, strings.NewReader(`{"commissionId":"new"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req.WithContext(provenance.NewContext(req.Context(), agent)))
		return rec.Code == http.StatusForbidden
	}
	checked := 0
	for _, g := range manifest.Groups {
		for _, c := range g.Commands {
			if c.Binding.Service == "" {
				continue
			}
			checked++
			if r := refused("/" + c.Binding.Service + "/" + c.Binding.Method); r == c.Governance.RunEligible {
				t.Errorf("%s: the API refuses an agent = %v, but run_eligible = %v", c.Name, r, c.Governance.RunEligible)
			}
		}
	}
	if checked < 20 {
		t.Fatalf("only %d bound commands checked", checked)
	}
}
