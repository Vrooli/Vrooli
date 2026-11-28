package safety

import (
	"strings"
	"testing"

	db "github.com/vrooli/api-core/databasetest"
)

// The route-locality rule: local OSS use stays unrestricted, but a request whose
// route leaves this machine needs a consent affirmation on every tier: a high-weight
// identity-altering one always, a low-weight one (caption, tags, boxes, colourise)
// because the image itself is sent.
func TestConsentForRouteLocality(t *testing.T) {
	cases := []struct {
		name     string
		policy   Policy
		weight   Weight
		tier     string
		required bool
		basis    string
	}{
		{"local run on the local tier is unrestricted", PolicyFor(TierLocal), WeightHigh, "local-gpu", false, ""},
		{"local CPU run is unrestricted", PolicyFor(TierLocal), WeightHigh, "local-cpu", false, ""},
		{"cloud route on the local tier needs consent", PolicyFor(TierLocal), WeightHigh, "byok-cloud", true, ConsentBasisRoute},
		{"an unknown route tier leaves the machine until proven local", PolicyFor(TierLocal), WeightHigh, "somewhere-else", true, ConsentBasisRoute},
		{"an unresolved tier fails closed", PolicyFor(TierLocal), WeightHigh, "", true, ConsentBasisRoute},
		{"low weight asks only when the image leaves the machine", PolicyFor(TierLocal), WeightLow, "byok-cloud", true, ConsentBasisRoute},
		{"a model the gateway serves on this machine stays local", PolicyFor(TierLocal), WeightHigh, "gateway-local", false, ""},
		{"low weight never asks for a local run", PolicyFor(TierLocal), WeightLow, "local-cpu", false, ""},
		{"low weight on the public tier asks only for the route, not the tier", PolicyFor(TierPublic), WeightLow, "local-gpu", false, ""},
		{"no weight never asks", PolicyFor(TierLocal), WeightNone, "byok-cloud", false, ""},
		{"public tier asks for a local run too", PolicyFor(TierPublic), WeightHigh, "local-gpu", true, ConsentBasisTier},
		{"public tier names the tier rule on a cloud route", PolicyFor(TierPublic), WeightHigh, "byok-cloud", true, ConsentBasisTier},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.policy.ConsentFor(tc.weight, "face_swap", tc.tier)
			if got.Required != tc.required || got.Basis != tc.basis {
				t.Fatalf("ConsentFor = required %v basis %q, want %v %q (%+v)", got.Required, got.Basis, tc.required, tc.basis, got)
			}
			if got.Required && got.Reason == "" {
				t.Error("a required consent must explain itself")
			}
		})
	}
}

func TestConsentNonLocalOverrideTurnsTheRouteRuleOff(t *testing.T) {
	off := ApplyOverrides(PolicyFor(TierLocal), mapLookup(map[string]string{ConsentNonLocalEnv: "0"}))
	if got := off.ConsentFor(WeightHigh, "face_swap", "byok-cloud"); got.Required {
		t.Errorf("route rule overridden off still required consent: %+v", got)
	}
	if !PolicyFor(TierLocal).ConsentNonLocal || !PolicyFor(TierPublic).ConsentNonLocal {
		t.Error("the route rule must default on for every tier")
	}
	bad := ApplyOverrides(PolicyFor(TierLocal), mapLookup(map[string]string{ConsentNonLocalEnv: "maybe"}))
	if !bad.ConsentNonLocal {
		t.Error("a malformed override must not silently disable the control")
	}
}

func TestEvaluateRouteRefusesCloudWithoutConsentAndAdmitsWithIt(t *testing.T) {
	p := PolicyFor(TierLocal)
	refused := p.EvaluateRoute(WeightHigh, "face_swap", "byok-cloud", false)
	if refused.Allowed {
		t.Fatal("a cloud face swap without consent must be refused on the local tier")
	}
	for _, want := range []string{"outside this machine", "face_swap"} {
		if !strings.Contains(refused.Reason, want) {
			t.Errorf("reason %q lacks %q", refused.Reason, want)
		}
	}
	for _, want := range []string{"consent_affirmed=true", "--consent", "local", ConsentNonLocalEnv} {
		if !strings.Contains(refused.RecoveryHint, want) {
			t.Errorf("recovery hint %q lacks %q", refused.RecoveryHint, want)
		}
	}
	admitted := p.EvaluateRoute(WeightHigh, "face_swap", "byok-cloud", true)
	if !admitted.Allowed || !admitted.RecordConsent {
		t.Fatalf("an affirmed cloud request must be admitted and audited, got %+v", admitted)
	}
	local := p.EvaluateRoute(WeightHigh, "face_swap", "local-gpu", false)
	if !local.Allowed || local.RecordConsent {
		t.Fatalf("a local run needs no affirmation and records nothing, got %+v", local)
	}
	// Affirming when nothing asked is not an audit event.
	if d := p.EvaluateRoute(WeightHigh, "face_swap", "local-gpu", true); d.RecordConsent {
		t.Error("an unneeded affirmation must not be logged")
	}
}

func TestConsentLogRecordsTheRouteAndMigratesOldTables(t *testing.T) {
	d := db.NewSQLite(t)
	// An install created before `route` existed.
	if _, err := d.Exec(`CREATE TABLE consent_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT, operation TEXT NOT NULL, weight TEXT NOT NULL,
		tier TEXT NOT NULL, affirmed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')))`); err != nil {
		t.Fatalf("old schema: %v", err)
	}
	log := NewConsentLog(d)
	for i := 0; i < 2; i++ { // the duplicate-column steady state must not fail
		if err := log.EnsureSchema(t.Context()); err != nil {
			t.Fatalf("EnsureSchema #%d: %v", i, err)
		}
	}
	gate := NewGate(TierLocal, log)
	dec := gate.EvaluateRoute(WeightHigh, "face_swap", "byok-cloud", true)
	if err := gate.RecordConsent(t.Context(), "face_swap", dec.Weight, "byok-cloud:openrouter-image"); err != nil {
		t.Fatalf("RecordConsent: %v", err)
	}
	var route, tier string
	if err := d.QueryRow(`SELECT route, tier FROM consent_log`).Scan(&route, &tier); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if route != "byok-cloud:openrouter-image" || tier != string(TierLocal) {
		t.Errorf("audit row = route %q tier %q", route, tier)
	}
}

// Every identity-altering operation carries the high weight the route rule keys
// on; a new one cannot slip past by inheriting none.
func TestIdentityAlteringOperationsAreHighWeight(t *testing.T) {
	for _, op := range []string{"face_swap", "garment_replace", "edit_instruct", "inpaint", "outpaint", "image_to_image", "object_removal", "background_replace"} {
		if got := OpWeight(op); got != WeightHigh {
			t.Errorf("OpWeight(%q) = %q, want high", op, got)
		}
	}
}
