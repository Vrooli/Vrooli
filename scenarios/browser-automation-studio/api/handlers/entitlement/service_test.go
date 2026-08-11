package entitlement

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/services/credits"
	entsvc "github.com/vrooli/browser-automation-studio/services/entitlement"
	entitlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement"
	entitlementconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement/entitlementconnect"
)

// ---------------------------------------------------------------------------
// fakes
// ---------------------------------------------------------------------------

type fakeSettings struct {
	store map[string]string
	getEr error
	setEr error
}

func newFakeSettings() *fakeSettings { return &fakeSettings{store: map[string]string{}} }

func (f *fakeSettings) GetSetting(_ context.Context, key string) (string, error) {
	if f.getEr != nil {
		return "", f.getEr
	}
	return f.store[key], nil
}

func (f *fakeSettings) SetSetting(_ context.Context, key, value string) error {
	if f.setEr != nil {
		return f.setEr
	}
	f.store[key] = value
	return nil
}

type fakeProvider struct {
	ent            *entsvc.Entitlement
	getErr         error
	overrideTier   entsvc.Tier
	invalidated    []string
	apiSourceLast  string
	apiPortLast    int
	aiCreditsLimit int

	tierWatermark map[entsvc.Tier]bool
	tierAI        map[entsvc.Tier]bool
	tierRecording map[entsvc.Tier]bool

	requiresWatermark bool
	canUseAI          bool
	canUseRecording   bool
}

func (f *fakeProvider) GetEntitlement(_ context.Context, _ string) (*entsvc.Entitlement, error) {
	return f.ent, f.getErr
}

func (f *fakeProvider) BuildOverrideEntitlement(user string, tier entsvc.Tier) *entsvc.Entitlement {
	f.overrideTier = tier
	return &entsvc.Entitlement{UserIdentity: user, Status: entsvc.StatusActive, Tier: tier}
}

func (f *fakeProvider) InvalidateCache(user string) { f.invalidated = append(f.invalidated, user) }

func (f *fakeProvider) SetApiSource(source string, port int) {
	f.apiSourceLast = source
	f.apiPortLast = port
}

func (f *fakeProvider) GetAICreditsLimit(_ entsvc.Tier) int { return f.aiCreditsLimit }

func (f *fakeProvider) TierRequiresWatermark(t entsvc.Tier) bool { return f.tierWatermark[t] }
func (f *fakeProvider) TierCanUseAI(t entsvc.Tier) bool          { return f.tierAI[t] }
func (f *fakeProvider) TierCanUseRecording(t entsvc.Tier) bool   { return f.tierRecording[t] }

func (f *fakeProvider) RequiresWatermark(context.Context, string) bool { return f.requiresWatermark }
func (f *fakeProvider) CanUseAI(context.Context, string) bool          { return f.canUseAI }
func (f *fakeProvider) CanUseRecording(context.Context, string) bool   { return f.canUseRecording }

func (f *fakeProvider) MinTierForAI() entsvc.Tier            { return entsvc.TierPro }
func (f *fakeProvider) MinTierForRecording() entsvc.Tier     { return entsvc.TierSolo }
func (f *fakeProvider) MinTierWithoutWatermark() entsvc.Tier { return entsvc.TierPro }

type fakeCredits struct {
	usage         *credits.UsageSummary
	usageErr      error
	history       []credits.UsageSummary
	historyMore   bool
	historyErr    error
	log           *credits.OperationLogPage
	logErr        error
	lastHistoryN  int
	lastHistoryOf int
	lastLogLimit  int
	lastLogOffset int
}

func (f *fakeCredits) CanCharge(context.Context, string, credits.OperationType) (bool, int, error) {
	return true, 0, nil
}

func (f *fakeCredits) Charge(context.Context, credits.ChargeRequest) (*credits.ChargeResult, error) {
	return nil, nil
}

func (f *fakeCredits) ChargeIfAllowed(context.Context, credits.ChargeRequest) (*credits.ChargeResult, error) {
	return nil, nil
}

func (f *fakeCredits) GetUsage(_ context.Context, _ string) (*credits.UsageSummary, error) {
	return f.usage, f.usageErr
}

func (f *fakeCredits) GetOperationCost(credits.OperationType) int { return 0 }

func (f *fakeCredits) LogFailedOperation(context.Context, credits.ChargeRequest, error) error {
	return nil
}

func (f *fakeCredits) GetUsageHistory(_ context.Context, _ string, months, offset int) ([]credits.UsageSummary, bool, error) {
	f.lastHistoryN, f.lastHistoryOf = months, offset
	return f.history, f.historyMore, f.historyErr
}

func (f *fakeCredits) GetOperationLog(_ context.Context, _, _, _ string, limit, offset int) (*credits.OperationLogPage, error) {
	f.lastLogLimit, f.lastLogOffset = limit, offset
	return f.log, f.logErr
}

func (f *fakeCredits) CanPerformAIOperation(context.Context, string, credits.OperationType, bool) (bool, string, string, int, error) {
	return true, "", "", 0, nil
}

// ---------------------------------------------------------------------------
// test client
// ---------------------------------------------------------------------------

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

type clientDeps struct {
	provider *fakeProvider
	credits  *fakeCredits
	settings *fakeSettings
}

func newTestClient(t *testing.T, d clientDeps) entitlementconnect.EntitlementServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	var credSvc credits.CreditService
	if d.credits != nil {
		credSvc = d.credits
	}
	var settings SettingsStore
	if d.settings != nil {
		settings = d.settings
	}
	mount := Module(Deps{Provider: d.provider, Credits: credSvc, Settings: settings, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return entitlementconnect.NewEntitlementServiceClient(srv.Client(), srv.URL)
}

func newDefaultProvider() *fakeProvider {
	return &fakeProvider{
		ent: &entsvc.Entitlement{
			UserIdentity: "alice@example.com",
			Status:       entsvc.StatusActive,
			Tier:         entsvc.TierPro,
			Features:     []string{"ai", "recording"},
		},
		aiCreditsLimit:    100,
		tierWatermark:     map[entsvc.Tier]bool{entsvc.TierFree: true},
		tierAI:            map[entsvc.Tier]bool{entsvc.TierPro: true, entsvc.TierStudio: true, entsvc.TierBusiness: true},
		tierRecording:     map[entsvc.Tier]bool{entsvc.TierSolo: true, entsvc.TierPro: true, entsvc.TierStudio: true, entsvc.TierBusiness: true},
		requiresWatermark: false,
		canUseAI:          true,
		canUseRecording:   true,
	}
}

// ---------------------------------------------------------------------------
// Module / wiring
// ---------------------------------------------------------------------------

func TestModule_RequiresLogger(t *testing.T) {
	require.Panics(t, func() { Module(Deps{Provider: newDefaultProvider()}) })
}

func TestModule_RequiresProvider(t *testing.T) {
	require.Panics(t, func() { Module(Deps{Logger: logrus.New()}) })
}

// ---------------------------------------------------------------------------
// GetStatus
// ---------------------------------------------------------------------------

func TestGetStatus_HappyPath(t *testing.T) {
	prov := newDefaultProvider()
	client := newTestClient(t, clientDeps{provider: prov})

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&entitlementv1.GetStatusRequest{User: "alice@example.com"}))
	require.NoError(t, err)

	s := resp.Msg.GetStatus()
	require.Equal(t, "alice@example.com", s.UserIdentity)
	require.Equal(t, string(entsvc.TierPro), s.Tier)
	require.True(t, s.IsActive)
	require.True(t, s.CanUseAi)
	require.True(t, s.EntitlementsEnabled)
	require.Equal(t, int32(100), s.MonthlyLimit)
	require.Equal(t, int32(100), s.MonthlyRemaining)
	require.Equal(t, "", s.OverrideTier)
	require.Len(t, s.FeatureAccess, 3)
}

func TestGetStatus_UsesStoredSettingsFallback(t *testing.T) {
	prov := newDefaultProvider()
	settings := newFakeSettings()
	settings.store["user_identity"] = "stored@example.com"
	client := newTestClient(t, clientDeps{provider: prov, settings: settings})

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&entitlementv1.GetStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, "stored@example.com", resp.Msg.GetStatus().UserIdentity)
}

func TestGetStatus_AnonymousFallback(t *testing.T) {
	prov := newDefaultProvider()
	client := newTestClient(t, clientDeps{provider: prov})

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&entitlementv1.GetStatusRequest{}))
	require.NoError(t, err)
	require.Equal(t, "anonymous", resp.Msg.GetStatus().UserIdentity)
}

func TestGetStatus_OverrideTierWins(t *testing.T) {
	prov := newDefaultProvider()
	settings := newFakeSettings()
	settings.store[entsvc.OverrideTierSettingKey] = string(entsvc.TierBusiness)
	client := newTestClient(t, clientDeps{provider: prov, settings: settings})

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&entitlementv1.GetStatusRequest{User: "x"}))
	require.NoError(t, err)
	require.Equal(t, string(entsvc.TierBusiness), resp.Msg.GetStatus().Tier)
	require.Equal(t, string(entsvc.TierBusiness), resp.Msg.GetStatus().OverrideTier)
}

func TestGetStatus_ProviderErrorFallsBackToDefaultEntitlement(t *testing.T) {
	prov := newDefaultProvider()
	prov.getErr = errors.New("upstream down")
	client := newTestClient(t, clientDeps{provider: prov})

	resp, err := client.GetStatus(context.Background(), connect.NewRequest(&entitlementv1.GetStatusRequest{User: "x"}))
	require.NoError(t, err)
	s := resp.Msg.GetStatus()
	require.Equal(t, string(entsvc.TierFree), s.Tier)
	require.Equal(t, string(entsvc.StatusInactive), s.Status)
}

// ---------------------------------------------------------------------------
// Identity
// ---------------------------------------------------------------------------

func TestGetIdentity_ReturnsStoredEmail(t *testing.T) {
	settings := newFakeSettings()
	settings.store["user_identity"] = "bob@example.com"
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: settings})

	resp, err := client.GetIdentity(context.Background(), connect.NewRequest(&entitlementv1.GetIdentityRequest{}))
	require.NoError(t, err)
	require.Equal(t, "bob@example.com", resp.Msg.Email)
}

func TestSetIdentity_PersistsAndReturnsStatus(t *testing.T) {
	prov := newDefaultProvider()
	settings := newFakeSettings()
	client := newTestClient(t, clientDeps{provider: prov, settings: settings})

	resp, err := client.SetIdentity(context.Background(), connect.NewRequest(&entitlementv1.SetIdentityRequest{Email: "Carol@Example.com"}))
	require.NoError(t, err)
	require.Equal(t, "carol@example.com", settings.store["user_identity"])
	require.Contains(t, prov.invalidated, "carol@example.com")
	require.Equal(t, "carol@example.com", resp.Msg.GetStatus().UserIdentity)
}

func TestSetIdentity_RejectsInvalidEmail(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: newFakeSettings()})
	_, err := client.SetIdentity(context.Background(), connect.NewRequest(&entitlementv1.SetIdentityRequest{Email: "not-an-email"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestSetIdentity_EmptyEmailAllowed(t *testing.T) {
	prov := newDefaultProvider()
	settings := newFakeSettings()
	settings.store["user_identity"] = "old@example.com"
	client := newTestClient(t, clientDeps{provider: prov, settings: settings})

	_, err := client.SetIdentity(context.Background(), connect.NewRequest(&entitlementv1.SetIdentityRequest{Email: ""}))
	require.NoError(t, err)
	require.Equal(t, "", settings.store["user_identity"])
}

func TestClearIdentity_ClearsStoredValue(t *testing.T) {
	settings := newFakeSettings()
	settings.store["user_identity"] = "drop@example.com"
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: settings})

	resp, err := client.ClearIdentity(context.Background(), connect.NewRequest(&entitlementv1.ClearIdentityRequest{}))
	require.NoError(t, err)
	require.Equal(t, "cleared", resp.Msg.Status)
	require.Equal(t, "", settings.store["user_identity"])
}

func TestRefreshStatus_RequiresUser(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider()})
	_, err := client.RefreshStatus(context.Background(), connect.NewRequest(&entitlementv1.RefreshStatusRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestRefreshStatus_InvalidatesAndReturns(t *testing.T) {
	prov := newDefaultProvider()
	client := newTestClient(t, clientDeps{provider: prov})
	resp, err := client.RefreshStatus(context.Background(), connect.NewRequest(&entitlementv1.RefreshStatusRequest{User: "alice@example.com"}))
	require.NoError(t, err)
	require.Contains(t, prov.invalidated, "alice@example.com")
	require.Equal(t, "alice@example.com", resp.Msg.GetStatus().UserIdentity)
}

// ---------------------------------------------------------------------------
// Usage
// ---------------------------------------------------------------------------

func TestGetUsage_FailedPreconditionWithoutCredits(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider()})
	_, err := client.GetUsage(context.Background(), connect.NewRequest(&entitlementv1.GetUsageRequest{User: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestGetUsage_RoundTripsTimestamps(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	cred := &fakeCredits{usage: &credits.UsageSummary{
		UserIdentity:     "alice@example.com",
		BillingMonth:     "2026-05",
		TotalCreditsUsed: 42,
		TotalOperations:  7,
		ByOperation:      map[credits.OperationType]int{credits.OperationType("ai.workflow_generate"): 15},
		OperationCounts:  map[credits.OperationType]int{credits.OperationType("ai.workflow_generate"): 3},
		CreditsLimit:     100,
		CreditsRemaining: 58,
		PeriodStart:      now,
		PeriodEnd:        now.AddDate(0, 1, 0).Add(-time.Nanosecond),
		ResetDate:        now.AddDate(0, 1, 0),
	}}
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), credits: cred})

	resp, err := client.GetUsage(context.Background(), connect.NewRequest(&entitlementv1.GetUsageRequest{User: "alice@example.com"}))
	require.NoError(t, err)
	require.Equal(t, int32(42), resp.Msg.TotalCreditsUsed)
	require.Equal(t, int32(15), resp.Msg.ByOperation["ai.workflow_generate"])
	require.NotNil(t, resp.Msg.PeriodStart)
	require.Equal(t, now.Unix(), resp.Msg.PeriodStart.AsTime().Unix())
}

func TestGetUsageHistory_DefaultsAndOffset(t *testing.T) {
	cred := &fakeCredits{historyMore: true}
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), credits: cred})

	resp, err := client.GetUsageHistory(context.Background(), connect.NewRequest(&entitlementv1.GetUsageHistoryRequest{User: "alice@example.com"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.HasMore)
	require.Equal(t, 6, cred.lastHistoryN) // default
	require.Equal(t, 0, cred.lastHistoryOf)

	_, err = client.GetUsageHistory(context.Background(), connect.NewRequest(&entitlementv1.GetUsageHistoryRequest{User: "alice@example.com", Months: 3, Offset: 2}))
	require.NoError(t, err)
	require.Equal(t, 3, cred.lastHistoryN)
	require.Equal(t, 2, cred.lastHistoryOf)
}

func TestGetOperationLog_DefaultsLimit(t *testing.T) {
	cred := &fakeCredits{log: &credits.OperationLogPage{
		UserIdentity: "alice@example.com",
		BillingMonth: "2026-05",
		Operations: []credits.OperationLogEntry{{
			ID:             "op-1",
			OperationType:  credits.OperationType("ai.workflow_generate"),
			CreditsCharged: 5,
			Success:        true,
			CreatedAt:      time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
			Metadata:       map[string]interface{}{"model": "claude"},
		}},
		Total: 1, Limit: 20, Offset: 0,
	}}
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), credits: cred})

	resp, err := client.GetOperationLog(context.Background(), connect.NewRequest(&entitlementv1.GetOperationLogRequest{User: "alice@example.com"}))
	require.NoError(t, err)
	require.Equal(t, 20, cred.lastLogLimit)
	require.Len(t, resp.Msg.Operations, 1)
	require.Equal(t, "op-1", resp.Msg.Operations[0].Id)
	require.NotNil(t, resp.Msg.Operations[0].Metadata)
	require.Equal(t, "claude", resp.Msg.Operations[0].Metadata.Fields["model"].GetStringValue())
}

// ---------------------------------------------------------------------------
// Override
// ---------------------------------------------------------------------------

func TestGetOverride_EmptyByDefault(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: newFakeSettings()})
	resp, err := client.GetOverride(context.Background(), connect.NewRequest(&entitlementv1.GetOverrideRequest{}))
	require.NoError(t, err)
	require.Equal(t, "", resp.Msg.Tier)
}

func TestSetOverride_PersistsTier(t *testing.T) {
	settings := newFakeSettings()
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: settings})
	resp, err := client.SetOverride(context.Background(), connect.NewRequest(&entitlementv1.SetOverrideRequest{Tier: "PRO"}))
	require.NoError(t, err)
	require.Equal(t, string(entsvc.TierPro), resp.Msg.Tier)
	require.Equal(t, string(entsvc.TierPro), settings.store[entsvc.OverrideTierSettingKey])
}

func TestSetOverride_EmptyClears(t *testing.T) {
	settings := newFakeSettings()
	settings.store[entsvc.OverrideTierSettingKey] = string(entsvc.TierPro)
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: settings})
	resp, err := client.SetOverride(context.Background(), connect.NewRequest(&entitlementv1.SetOverrideRequest{Tier: ""}))
	require.NoError(t, err)
	require.Equal(t, "", resp.Msg.Tier)
	require.Equal(t, "", settings.store[entsvc.OverrideTierSettingKey])
}

func TestSetOverride_RejectsUnknownTier(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: newFakeSettings()})
	_, err := client.SetOverride(context.Background(), connect.NewRequest(&entitlementv1.SetOverrideRequest{Tier: "platinum"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestSetOverride_RequiresSettings(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider()})
	_, err := client.SetOverride(context.Background(), connect.NewRequest(&entitlementv1.SetOverrideRequest{Tier: "pro"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestClearOverride_RemovesValue(t *testing.T) {
	settings := newFakeSettings()
	settings.store[entsvc.OverrideTierSettingKey] = string(entsvc.TierPro)
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: settings})
	_, err := client.ClearOverride(context.Background(), connect.NewRequest(&entitlementv1.ClearOverrideRequest{}))
	require.NoError(t, err)
	require.Equal(t, "", settings.store[entsvc.OverrideTierSettingKey])
}

// ---------------------------------------------------------------------------
// API source
// ---------------------------------------------------------------------------

func TestGetApiSource_DefaultsToProduction(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: newFakeSettings()})
	resp, err := client.GetApiSource(context.Background(), connect.NewRequest(&entitlementv1.GetApiSourceRequest{}))
	require.NoError(t, err)
	require.Equal(t, "production", resp.Msg.Source)
	require.Equal(t, int32(15000), resp.Msg.LocalPort)
}

func TestSetApiSource_PersistsAndPropagates(t *testing.T) {
	prov := newDefaultProvider()
	settings := newFakeSettings()
	client := newTestClient(t, clientDeps{provider: prov, settings: settings})

	resp, err := client.SetApiSource(context.Background(), connect.NewRequest(&entitlementv1.SetApiSourceRequest{Source: "LOCAL", LocalPort: 15123}))
	require.NoError(t, err)
	require.Equal(t, "local", resp.Msg.Source)
	require.Equal(t, int32(15123), resp.Msg.LocalPort)
	require.Equal(t, "local", settings.store[entsvc.ApiSourceSettingKey])
	require.Equal(t, "15123", settings.store[entsvc.LocalApiPortSettingKey])
	require.Equal(t, "local", prov.apiSourceLast)
	require.Equal(t, 15123, prov.apiPortLast)
}

func TestSetApiSource_RejectsUnknown(t *testing.T) {
	client := newTestClient(t, clientDeps{provider: newDefaultProvider(), settings: newFakeSettings()})
	_, err := client.SetApiSource(context.Background(), connect.NewRequest(&entitlementv1.SetApiSourceRequest{Source: "staging"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestControlPlaneGuardRecognizesOnlyLoopbackControlRequests(t *testing.T) {
	for _, test := range []struct {
		name string
		peer string
		want bool
	}{
		{name: "ipv4 loopback", peer: "127.0.0.1:8080", want: true},
		{name: "ipv6 loopback", peer: "[::1]:8080", want: true},
		{name: "remote", peer: "198.51.100.10:8080", want: false},
		{name: "spoofed host is irrelevant", peer: "198.51.100.10:8080", want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := isLoopbackPeer(test.peer); got != test.want {
				t.Fatalf("isLoopbackPeer(%q) = %v, want %v", test.peer, got, test.want)
			}
		})
	}
	if !isProtectedControlPath("/browser-automation-studio.v1.EntitlementService/SetOverride") {
		t.Fatal("SetOverride must be protected")
	}
	if isProtectedControlPath("/browser-automation-studio.v1.EntitlementService/GetStatus") {
		t.Fatal("GetStatus must remain available to authenticated remote clients")
	}
}

func TestClearApiSource_ResetsToProduction(t *testing.T) {
	prov := newDefaultProvider()
	settings := newFakeSettings()
	settings.store[entsvc.ApiSourceSettingKey] = "local"
	client := newTestClient(t, clientDeps{provider: prov, settings: settings})

	_, err := client.ClearApiSource(context.Background(), connect.NewRequest(&entitlementv1.ClearApiSourceRequest{}))
	require.NoError(t, err)
	require.Equal(t, "production", settings.store[entsvc.ApiSourceSettingKey])
	require.Equal(t, "production", prov.apiSourceLast)
	require.Equal(t, 0, prov.apiPortLast)
}
