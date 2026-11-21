package monetization

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/api-core/consumeridentity"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

const testUpgradePath = "/upgrade"

func TestStatusDecision(t *testing.T) {
	for _, test := range []struct {
		status  string
		allowed bool
		warning bool
		reason  string
	}{
		{status: "active", allowed: true, reason: ReasonAllowed},
		{status: "trialing", allowed: true, reason: ReasonAllowed},
		{status: "past_due", allowed: true, warning: true, reason: ReasonPastDue},
		{status: "canceled", reason: ReasonSubscriptionInactive},
		{status: "inactive", reason: ReasonSubscriptionInactive},
		{status: "unknown", reason: ReasonSubscriptionInactive},
	} {
		t.Run(test.status, func(t *testing.T) {
			got := StatusDecision(entitlementclient.Payload{Status: test.status}, testUpgradePath)
			if got.Allowed != test.allowed || got.Warning != test.warning || got.Reason != test.reason || got.UpgradePath != testUpgradePath {
				t.Fatalf("decision = %+v, want allowed=%v warning=%v reason=%q", got, test.allowed, test.warning, test.reason)
			}
		})
	}
}

func TestLeaseValuesDriveRankFeaturesAndLimits(t *testing.T) {
	payload := entitlementclient.Payload{
		PlanRank: 3,
		Features: []string{"ai"},
		Limits:   []entitlementclient.Limit{{Key: "workflow_executions", Value: 12, BundleKey: "business_suite"}},
	}
	if !HasFeature(payload, "ai") || HasFeature(payload, "recording") {
		t.Fatalf("feature lookup did not use lease payload")
	}
	for rank, want := range map[int32]bool{2: true, 3: true, 4: false} {
		if got := AtLeastRank(payload, rank); got != want {
			t.Errorf("rank %d = %v, want %v", rank, got, want)
		}
	}
	if value, ok := Limit(payload, "workflow_executions", "business_suite"); !ok || value != 12 {
		t.Fatalf("lease limit = %d,%v, want 12,true", value, ok)
	}
	if _, ok := Limit(payload, "workflow_executions", "other_bundle"); ok {
		t.Fatal("limit from another bundle was accepted")
	}
}

func TestAppEntitlementsIsolateOffersAndExpandBundleGrants(t *testing.T) {
	payload := entitlementclient.Payload{
		Status:   "active",
		PlanRank: 99,
		Features: []string{"audio:export", "web:admin"},
		Limits:   []entitlementclient.Limit{{Key: "ai_credits", Value: 999}},
		AppEntitlements: []entitlementclient.AppEntitlement{
			{AppKey: "audio-tools", Status: "active", PlanRank: 1, Features: []string{"audio:export"}, Limits: []entitlementclient.Limit{{Key: "ai_credits", Value: 100}}},
			{AppKey: "web-console", BundleKey: "vrooli-suite", Status: "active", PlanRank: 3, Features: []string{"web:admin"}, Limits: []entitlementclient.Limit{{Key: "ai_credits", Value: 500}}},
		},
	}
	audio := NewGateForApp(nil, nil, "vrooli-suite", "audio-tools")
	web := NewGateForApp(nil, nil, "vrooli-suite", "web-console")
	missingApp := NewGate(nil, nil, "vrooli-suite")
	decision := FeatureDecisionForApp(payload, audio.AppKey, "audio:export", 1, StatusDecision(payload, testUpgradePath))
	if !decision.Allowed {
		t.Fatalf("audio app grant denied: %+v", decision)
	}
	decision = FeatureDecisionForApp(payload, audio.AppKey, "web:admin", 0, StatusDecision(payload, testUpgradePath))
	if decision.Allowed || decision.Reason != ReasonFeatureMissing {
		t.Fatalf("audio offer unlocked another app: %+v", decision)
	}
	decision = FeatureDecisionForApp(payload, missingApp.AppKey, "web:admin", 0, StatusDecision(payload, testUpgradePath))
	if decision.Allowed || decision.Reason != ReasonFeatureMissing {
		t.Fatalf("unbound gate accepted a scoped entitlement: %+v", decision)
	}
	if decision := FeatureDecision(payload, "web:admin", 0, StatusDecision(payload, testUpgradePath)); decision.Allowed {
		t.Fatalf("unscoped helper accepted an app-scoped lease: %+v", decision)
	}
	if decision := FeatureDecisionForApp(payload, audio.AppKey, "audio:export", 2, StatusDecision(payload, testUpgradePath)); decision.Allowed || decision.Reason != ReasonRankInsufficient {
		t.Fatalf("another app's global plan rank leaked: %+v", decision)
	}
	if decision := FeatureDecisionForApp(payload, audio.AppKey, "audio:export", 1, StatusDecision(payload, testUpgradePath)); !decision.Allowed {
		t.Fatalf("standalone offer denied after sibling bundle grant: %+v", decision)
	}
	if value, ok := LimitForApp(payload, audio.AppKey, "ai_credits", "vrooli-suite"); !ok || value != 100 {
		t.Fatalf("audio limit = %d,%v, want app-specific 100", value, ok)
	}
	if value, ok := LimitForApp(payload, web.AppKey, "ai_credits", "vrooli-suite"); !ok || value != 500 {
		t.Fatalf("bundle-expanded web limit = %d,%v, want 500", value, ok)
	}
	if _, ok := LimitForApp(payload, "other-app", "ai_credits", "vrooli-suite"); ok {
		t.Fatal("limit leaked to app absent from bundle expansion")
	}
	if _, ok := LimitForApp(payload, "", "ai_credits", "vrooli-suite"); ok {
		t.Fatal("unbound gate accepted scoped limit")
	}
	if HasFeatureForApp(payload, "audio-tools", "web:admin") || HasFeatureForApp(payload, "", "audio:export") {
		t.Fatal("app-scoped feature leaked across app boundary or missing identity")
	}
}

func TestAppGrantStatusAndLegacyLeaseBehavior(t *testing.T) {
	payload := entitlementclient.Payload{Status: "active", Features: []string{"ai"}, AppEntitlements: []entitlementclient.AppEntitlement{{AppKey: "audio-tools", Status: "canceled", Features: []string{"ai"}}}}
	if decision := FeatureDecisionForApp(payload, "audio-tools", "ai", 0, StatusDecision(payload, testUpgradePath)); decision.Allowed || decision.Reason != ReasonSubscriptionInactive {
		t.Fatalf("canceled app grant remained active from global status: %+v", decision)
	}
	legacy := entitlementclient.Payload{Status: "active", Features: []string{"ai"}, Limits: []entitlementclient.Limit{{Key: "ai_credits", Value: 7}}}
	if decision := FeatureDecisionForApp(legacy, "audio-tools", "ai", 0, StatusDecision(legacy, testUpgradePath)); !decision.Allowed {
		t.Fatalf("legacy bundle lease lost compatibility: %+v", decision)
	}
	if value, ok := LimitForApp(legacy, "audio-tools", "ai_credits", "bundle"); !ok || value != 7 {
		t.Fatalf("legacy limit = %d,%v", value, ok)
	}
}

func TestCachedFeatureAtUsesExactAppGrantAfterOfflineLeaseRestore(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "issuer", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	token, err := entitlementclient.Sign(entitlementclient.Payload{
		UserIdentity: "member@example.com",
		Status:       "active",
		Features:     []string{"audio:export", "web:admin"},
		AppEntitlements: []entitlementclient.AppEntitlement{
			{AppKey: "audio-tools", Status: "active", PlanRank: 1, Features: []string{"audio:export"}},
			{AppKey: "web-console", Status: "active", PlanRank: 1, Features: []string{"web:admin"}},
		},
		NotAfter: now.Add(time.Hour),
	}, "issuer", key)
	if err != nil {
		t.Fatal(err)
	}
	leaseStore, err := entitlementclient.NewFileLeaseStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := leaseStore.Save("member@example.com", entitlementclient.LeaseRecord{Lease: token}); err != nil {
		t.Fatal(err)
	}
	client := entitlementclient.NewClient("", nil, nil)
	client.SetStore(leaseStore)
	client.SetIssuerKeyStore(staticIssuerKeyStore{jwks: jwks})
	audioGate := NewGateForApp(client, nil, "suite", "audio-tools")
	webGate := NewGateForApp(client, nil, "suite", "web-console")
	if got := audioGate.CachedFeatureAt("member@example.com", "audio:export", 1, now); !got.Allowed {
		t.Fatalf("restored scoped lease did not grant audio: %+v", got)
	}
	if got := audioGate.CachedFeatureAt("member@example.com", "web:admin", 0, now); got.Allowed || got.Reason != ReasonFeatureMissing {
		t.Fatalf("restored scoped lease leaked web feature to audio: %+v", got)
	}
	if got := webGate.CachedFeatureAt("member@example.com", "web:admin", 0, now); !got.Allowed {
		t.Fatalf("restored scoped lease did not grant web app: %+v", got)
	}
}

type staticIssuerKeyStore struct{ jwks []byte }

func (s staticIssuerKeyStore) Load(context.Context, string) ([]byte, error) {
	return append([]byte(nil), s.jwks...), nil
}

func (staticIssuerKeyStore) Save(context.Context, string, []byte) error { return nil }

func TestFeatureDecisionCoversCommercialReasonVocabulary(t *testing.T) {
	active := entitlementclient.Payload{Status: "active", PlanRank: 2, Features: []string{"ai"}}
	for _, test := range []struct {
		name     string
		payload  entitlementclient.Payload
		feature  string
		rank     int32
		decision Decision
		want     string
	}{
		{name: "allowed", payload: active, decision: StatusDecision(active, testUpgradePath), want: ReasonAllowed},
		{name: "past due", payload: entitlementclient.Payload{Status: "past_due"}, decision: StatusDecision(entitlementclient.Payload{Status: "past_due"}, testUpgradePath), want: ReasonPastDue},
		{name: "feature missing", payload: active, feature: "recording", decision: StatusDecision(active, testUpgradePath), want: ReasonFeatureMissing},
		{name: "rank insufficient", payload: active, rank: 3, decision: StatusDecision(active, testUpgradePath), want: ReasonRankInsufficient},
		{name: "subscription inactive", payload: entitlementclient.Payload{Status: "canceled"}, decision: StatusDecision(entitlementclient.Payload{Status: "canceled"}, testUpgradePath), want: ReasonSubscriptionInactive},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := FeatureDecision(test.payload, test.feature, test.rank, test.decision)
			if got.Reason != test.want || got.UpgradePath != testUpgradePath {
				t.Fatalf("decision = %+v, want reason=%q with upgrade path", got, test.want)
			}
		})
	}

	for _, test := range []struct {
		name string
		err  error
		want string
	}{
		{name: "lease unavailable", err: entitlementclient.ErrLeaseUnavailable, want: ReasonLeaseUnavailable},
		{name: "lease expired", err: entitlementclient.ErrLeaseExpired, want: ReasonLeaseExpired},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := fallbackDecision(test.err, testUpgradePath)
			if got.Reason != test.want {
				t.Fatalf("decision = %+v, want reason=%q", got, test.want)
			}
		})
	}
}

func TestGateFallsBackToFreeTierWhenLeaseUnavailableOrExpired(t *testing.T) {
	for _, err := range []error{entitlementclient.ErrLeaseUnavailable, entitlementclient.ErrLeaseExpired} {
		decision := fallbackDecision(err, testUpgradePath)
		if decision.Allowed || decision.UpgradePath != testUpgradePath {
			t.Fatalf("fallback decision = %+v", decision)
		}
		if errors.Is(err, entitlementclient.ErrLeaseExpired) && decision.Reason != ReasonLeaseExpired {
			t.Fatalf("expired fallback reason = %q", decision.Reason)
		}
	}
}

func TestGateWithoutClientDoesNotGrantAccess(t *testing.T) {
	gate := NewGate(nil, nil, "business_suite")
	decision := gate.Feature(context.Background(), "user@example.com", "ai", 1)
	if decision.Allowed || decision.Reason != ReasonLeaseUnavailable {
		t.Fatalf("decision = %+v", decision)
	}
}

func TestMeterGateUsesCommittedInventoryAndOfflineClass(t *testing.T) {
	bas := NewGateForApp(nil, nil, "business_suite", "browser-automation-studio")
	if got := bas.CachedMeterAt("user@example.com", "workflow_executions", time.Now()); got.Reason != ReasonLeaseUnavailable {
		t.Fatalf("declared Class B offline meter = %+v", got)
	}
	if got := bas.CachedMeterAt("user@example.com", "ai_credits", time.Now()); got.Reason != ReasonMeterRequiresAuthority {
		t.Fatalf("Class A offline meter = %+v", got)
	}
	if got := bas.CachedMeterAt("user@example.com", "not-a-meter", time.Now()); got.Reason != ReasonMeterUndeclared {
		t.Fatalf("undeclared offline meter = %+v", got)
	}
	webConsole := NewGateForApp(nil, nil, "business_suite", "web-console")
	if got := webConsole.Meter(context.Background(), "user@example.com", "workflow_executions"); got.Reason != ReasonMeterUndeclared {
		t.Fatalf("meter declared by another app = %+v", got)
	}
	if got := NewGate(nil, nil, "business_suite").Meter(context.Background(), "user@example.com", "workflow_executions"); got.Reason != ReasonMeterUndeclared {
		t.Fatalf("meter gate without app declaration = %+v", got)
	}
}

func TestAtLeastRankDoesNotDependOnTimeOrTierNames(t *testing.T) {
	if !AtLeastRank(entitlementclient.Payload{PlanRank: 1, NotAfter: time.Now().Add(time.Hour)}, 1) {
		t.Fatal("rank boundary denied")
	}
}
