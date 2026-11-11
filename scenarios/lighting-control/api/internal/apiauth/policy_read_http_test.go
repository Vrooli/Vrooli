package apiauth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy/policyconnect"
	policyH "lighting-control/handlers/policy"
	internalpolicy "lighting-control/internal/policy"
)

type readPolicyInputs struct{ input internalpolicy.Inputs }

func (s readPolicyInputs) Read(context.Context) internalpolicy.Inputs { return s.input }

func TestReadOnlyHTTPGetPolicyNeverCommitsInitializationOrEngineTransitions(t *testing.T) {
	// Old receipt time forces every HTTP read through the cache-miss path,
	// without sleeping or changing clocks concurrently with the server.
	at := time.Now().UTC().Add(-20 * time.Second)
	for _, kind := range []string{"default", "legacy", "expired manual", "pending ownership", "debounce elapsed"} {
		t.Run(kind, func(t *testing.T) {
			principal := identity.Principal{Kind: identity.ActorHuman, Subject: "reader", Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: []string{"lighting-control:read"}, ExpiresAt: time.Now().Add(time.Hour)}
			f := newFixture(t, principal, true, false)
			if _, err := f.db.Exec(`DELETE FROM lighting_policy_state`); err != nil {
				t.Fatal(err)
			}
			config := internalpolicy.DefaultConfig()
			state := internalpolicy.State{}
			switch kind {
			case "legacy":
				config.HealthPalette = nil
				config.ActivityEffects = nil
			case "expired manual":
				state.ManualPreset = policyv1.PresetKind_PRESET_KIND_STATIC
				state.ManualExpiry = at.Add(-time.Second)
				state.ManualSelectionPending = true
			case "pending ownership":
				state.ManualPreset = policyv1.PresetKind_PRESET_KIND_STATIC
				state.ManualExpiry = at.Add(time.Hour)
				state.ManualSelectionPending = true
			case "debounce elapsed":
				config.DefaultPreset = policyv1.PresetKind_PRESET_KIND_LOAD
				state.ActivePreset = policyv1.PresetKind_PRESET_KIND_STATIC
				state.Candidate = policyv1.PresetKind_PRESET_KIND_LOAD
				state.CandidateAt = at.Add(-10 * time.Second)
			}
			store := internalpolicy.NewSQLiteStore(f.db)
			if kind != "default" {
				if err := store.Save(context.Background(), config, state); err != nil {
					t.Fatal(err)
				}
			}
			inputs := internalpolicy.Inputs{Now: at, SystemMonitor: internalpolicy.SourceInput{Name: "system-monitor", State: policyv1.EvidenceState_EVIDENCE_STATE_FRESH, SourceTime: at, ObservedAt: at, Measurements: map[string]float64{"cpu_percent": 95}}}
			service := policyH.NewService(store, readPolicyInputs{inputs}, f.controlDomain)
			_, handler := policyconnect.NewPolicyServiceHandler(policyH.NewConnectHandler(service, nil))
			server := httptest.NewServer(authn.Middleware(authn.Config{Providers: []authn.Provider{credentialProvider{principal: principal}}})(handler))
			defer server.Close()
			client := policyconnect.NewPolicyServiceClient(http.DefaultClient, server.URL)
			before := databaseSnapshot(t, f.db)
			f.backend.calls.Store(0)
			for i := 0; i < 3; i++ {
				response, err := client.GetPolicy(context.Background(), request(f, &policyv1.GetPolicyRequest{}))
				if err != nil {
					t.Fatal(err)
				}
				if response.Msg.Config.HealthPalette == nil || response.Msg.Config.ActivityEffects == nil {
					t.Fatal("read lost normalized presentation defaults")
				}
				if kind == "pending ownership" && !strings.Contains(response.Msg.Effective.Reason, "control selection pending") {
					t.Fatalf("pending ownership hidden: %+v", response.Msg.Effective)
				}
				if kind == "expired manual" && response.Msg.Effective.ManualPriority {
					t.Fatal("expired manual selection remained active in preview")
				}
				if kind == "debounce elapsed" && response.Msg.Effective.Preset != policyv1.PresetKind_PRESET_KIND_LOAD {
					t.Fatalf("read did not preserve current engine preview: %+v", response.Msg.Effective)
				}
				if after := databaseSnapshot(t, f.db); after != before {
					t.Fatal("read-only HTTP GetPolicy committed initialization, normalization, engine state or manual ownership")
				}
				if f.backend.calls.Load() != 0 {
					t.Fatal("read-only HTTP GetPolicy called lighting backend")
				}
			}
		})
	}
}
