package wire_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"

	"research-desk/handlers/launchapi"
	"research-desk/handlers/operatorapi"
	"research-desk/internal/wire"
)

var (
	agent = provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-7"}
	// An agent identity that was offered but could not be verified.
	unverified = provenance.Provenance{Actor: provenance.ActorOperator, VerificationStatus: provenance.VerificationUnavailable}
)

// F13: who may act for the operator.
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
	want := []string{"Launch", "Control", "TestLaunch", "Answer", "AddEntry", "EnableSteering", "AddVerdict", "SetCourier", "TestCourier", "MarkSeen"}
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
