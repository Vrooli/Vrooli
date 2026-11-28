package sessionpermission

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type receiptVerifier struct{ accepted bool }

type credentialVerifier struct{ accepted bool }

type browserTabConsentVerifier struct {
	accepted bool
	requests []ActionRequest
}

func (v *browserTabConsentVerifier) VerifyBrowserTabConsent(_ context.Context, receipt string, req ActionRequest) error {
	v.requests = append(v.requests, req)
	if receipt != "consent" || !v.accepted {
		return errors.New("consent expired or revoked")
	}
	return nil
}

func (v credentialVerifier) AuthorizeBrowserCredential(_ context.Context, req ActionRequest, credentialID string) error {
	if !v.accepted || credentialID != "vault-ref-1" || req.CurrentURL != "https://shop.example.test/app" {
		return errors.New("credential scope mismatch")
	}
	return nil
}

func (v receiptVerifier) VerifyBrowserApproval(_ context.Context, receipt string, req ActionRequest) error {
	if !v.accepted || receipt != "approved" || req.SessionID != "s1" || req.LeaseID != "l1" || req.OwnerID != "o1" {
		return errors.New("receipt mismatch")
	}
	return nil
}

func TestAuthorizerEnforcesCredentialExtensionHistoryAndActionBudgets(t *testing.T) {
	profile := Profile{
		Effect: EffectAct, Approval: ApprovalNone, Control: true, View: true,
		URLAllow: []string{"https://shop.example.test/*"}, MaxActions: 5,
		CredentialIDs: []string{"vault-ref-1"}, ExtensionAccess: true,
		ExtensionPermissions: map[string][]string{"extension-1": {"https://shop.example.test/*"}},
		HistoryRead:          true,
	}
	p, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	p.SetCredentialUseVerifier(credentialVerifier{accepted: true})
	base := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", CurrentURL: "https://shop.example.test/app"}
	credential := base
	credential.Kind, credential.CredentialID = "credential_use", "vault-ref-1"
	if err := p.Authorize(context.Background(), credential); err != nil {
		t.Fatalf("granted credential denied: %v", err)
	}
	denied := base
	denied.Kind, denied.CredentialID = "credential_use", "other-secret"
	if err := p.Authorize(context.Background(), denied); !errors.Is(err, ErrDenied) {
		t.Fatalf("ungranted credential accepted: %v", err)
	}

	// A fresh policy proves the extension allowlist and history controls without
	// being masked by the one-action budget used above.
	p, err = New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, URLAllow: profile.URLAllow, ExtensionAccess: true, ExtensionPermissions: profile.ExtensionPermissions, HistoryRead: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	extension := base
	extension.Kind, extension.ExtensionID, extension.ExtensionPermission = "read", "extension-1", "https://shop.example.test/*"
	if err := p.Authorize(context.Background(), extension); err != nil {
		t.Fatalf("granted extension permission denied: %v", err)
	}
	extension.ExtensionPermission = "tabs"
	if err := p.Authorize(context.Background(), extension); !errors.Is(err, ErrDenied) {
		t.Fatalf("ungranted extension permission accepted: %v", err)
	}
	unknownExtension := base
	unknownExtension.Kind, unknownExtension.ExtensionID = "extension_load", "unlisted-extension"
	if err := p.Authorize(context.Background(), unknownExtension); !errors.Is(err, ErrDenied) {
		t.Fatalf("unlisted extension id accepted without a permission string: %v", err)
	}
	missingExtension := base
	missingExtension.Kind = "extension_load"
	if err := p.Authorize(context.Background(), missingExtension); !errors.Is(err, ErrDenied) {
		t.Fatalf("extension operation without an id accepted: %v", err)
	}
	inventory := base
	inventory.Kind = "extension_inventory"
	if err := p.Authorize(context.Background(), inventory); err != nil {
		t.Fatalf("allowlisted extension inventory admission denied: %v", err)
	}
	history := base
	history.Kind = "history_read"
	if err := p.Authorize(context.Background(), history); err != nil {
		t.Fatalf("granted history read denied: %v", err)
	}

	budget, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, URLAllow: profile.URLAllow, MaxActions: 1, MaxDuration: time.Hour}, nil)
	if err != nil {
		t.Fatal(err)
	}
	read := base
	read.Kind = "screenshot"
	if err := budget.Authorize(context.Background(), read); err != nil {
		t.Fatalf("first budgeted action denied: %v", err)
	}
	if err := budget.Authorize(context.Background(), read); !errors.Is(err, ErrDenied) {
		t.Fatalf("action budget exceeded: %v", err)
	}
}

func TestRecordingStopRequiresControlHistoryScopeAndSupportedApproval(t *testing.T) {
	request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1",
		Kind: "recording_stop", CurrentURL: "https://shop.example.test/app"}
	base := Profile{Effect: EffectAct, Approval: ApprovalNone, Control: true, View: true,
		HistoryRead: true, URLAllow: []string{"https://shop.example.test/*"}}
	cases := []struct {
		name    string
		profile Profile
		allowed bool
	}{
		{name: "missing control", profile: func() Profile { p := base; p.Control = false; return p }()},
		{name: "missing history permission for terminal receipt", profile: func() Profile { p := base; p.HistoryRead = false; return p }()},
		{name: "out of scope current page", profile: func() Profile { p := base; p.URLAllow = []string{"https://other.example/*"}; return p }()},
		{name: "valid scoped stop", profile: base, allowed: true},
		{name: "every action requires unavailable route receipt", profile: func() Profile { p := base; p.Approval = ApprovalEveryAction; return p }()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := New(tc.profile, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = policy.Authorize(context.Background(), request)
			if tc.allowed && err != nil {
				t.Fatalf("valid stop denied: %v", err)
			}
			if !tc.allowed && !errors.Is(err, ErrDenied) {
				t.Fatalf("stop authorization error = %v, want denial", err)
			}
		})
	}
}

func TestAppMonitorCapturePermissionIsDefaultDenyAndManagedOnly(t *testing.T) {
	base := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", CurrentURL: "https://monitor.example.test/apps/workspace"}
	denied, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, ProfileMode: "fresh", URLAllow: []string{"https://monitor.example.test/*"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	base.Kind = "app_monitor_metrics"
	if err := denied.Authorize(context.Background(), base); !errors.Is(err, ErrDenied) {
		t.Fatalf("legacy default-deny profile authorized App Monitor metrics: %v", err)
	}
	granted, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, ProfileMode: "fresh", URLAllow: []string{"https://monitor.example.test/*"}, AppMonitorMetrics: true, AppMonitorTrace: true, AppMonitorTraceMaxDurationMS: 15_000, AppMonitorTraceMaxBytes: 4 << 20}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := granted.Authorize(context.Background(), base); err != nil {
		t.Fatalf("explicit managed metrics grant denied: %v", err)
	}
	base.Kind = "app_monitor_trace"
	if err := granted.Authorize(context.Background(), base); err != nil {
		t.Fatalf("explicit managed trace grant denied: %v", err)
	}
	if _, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, ProfileMode: "user_attached", AppMonitorMetrics: true}, nil); err == nil {
		t.Fatal("user-attached App Monitor metrics profile was accepted")
	}
	if _, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, ProfileMode: "fresh", AppMonitorTrace: true, AppMonitorTraceMaxDurationMS: 60_001, AppMonitorTraceMaxBytes: 1}, nil); err == nil {
		t.Fatal("oversized App Monitor trace budget was accepted")
	}
}

func TestIdentityProfileIsModeAwareAndBounded(t *testing.T) {
	identity := &BrowserIdentityProfile{Preset: "balanced", ViewportWidth: 1440, ViewportHeight: 900, DeviceScaleFactor: 1.25, Locale: "en-GB", TimezoneID: "Europe/London", ColorScheme: "dark"}
	if _, err := New(Profile{ID: "managed", ProfileMode: "fresh", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: identity}, nil); err != nil {
		t.Fatalf("bounded managed identity profile rejected: %v", err)
	}
	if _, err := New(Profile{ID: "attached", ProfileMode: "user_attached", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: identity}, nil); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("identity customization on a real attached tab = %v, want fail-closed invalid profile", err)
	}
	if _, err := New(Profile{ID: "remote", ProfileMode: "remote", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{ViewportWidth: 1365, ViewportHeight: 768}}, nil); !errors.Is(err, ErrInvalidProfile) {
		t.Fatal("remote identity customization must remain unavailable")
	}
	installed := Profile{ID: "installed", ProfileMode: "isolated", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{ViewportWidth: 1365, ViewportHeight: 768}}
	if _, err := New(installed, nil); err != nil {
		t.Fatalf("installed viewport-only identity profile rejected: %v", err)
	}
	for _, identity := range []*BrowserIdentityProfile{{Preset: "balanced"}, {DeviceScaleFactor: 1.25}, {Locale: "en-GB"}, {TimezoneID: "UTC"}, {ColorScheme: "dark"}} {
		installed.IdentityProfile = identity
		if _, err := New(installed, nil); !errors.Is(err, ErrInvalidProfile) || !strings.Contains(err.Error(), "viewport only") {
			t.Fatalf("installed unsupported identity profile error = %v", err)
		}
	}
	if _, err := New(Profile{ID: "managed", ProfileMode: "fresh", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{Preset: "firefox"}}, nil); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("unsupported engine persona = %v, want fail-closed invalid profile", err)
	}
}

func TestIdentityProfileValidatesBCP47AndExplicitTimezonesWithoutNormalization(t *testing.T) {
	for _, locale := range []string{"en-GB", "zh-Hant-TW", "en-us-u-ca-gregory", "i-klingon"} {
		profile := Profile{ID: "managed", ProfileMode: "fresh", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{Locale: locale}}
		policy, err := New(profile, nil)
		if err != nil {
			t.Errorf("valid BCP 47 locale %q rejected: %v", locale, err)
			continue
		}
		if got := policy.ProfileSnapshot().IdentityProfile.Locale; got != locale {
			t.Errorf("locale was normalized from %q to %q", locale, got)
		}
	}
	for _, locale := range []string{"en_US", "en--US", "en-GB-", "en-US-abc-abc", "not_a_tag", "en-GB\n"} {
		if _, err := New(Profile{ID: "managed", ProfileMode: "fresh", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{Locale: locale}}, nil); !errors.Is(err, ErrInvalidProfile) || !strings.Contains(err.Error(), "identity locale") {
			t.Errorf("invalid locale %q error = %v, want field-specific profile rejection", locale, err)
		}
	}
	for _, zone := range []string{"UTC", "Etc/UTC", "Etc/GMT", "GMT", "America/New_York", "US/Eastern"} {
		policy, err := New(Profile{ID: "managed", ProfileMode: "fresh", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{TimezoneID: zone}}, nil)
		if err != nil {
			t.Errorf("supported timezone %q rejected: %v", zone, err)
			continue
		}
		if got := policy.ProfileSnapshot().IdentityProfile.TimezoneID; got != zone {
			t.Errorf("timezone was normalized from %q to %q", zone, got)
		}
	}
	for _, zone := range []string{"Local", "Mars/Olympus", "../../etc/passwd", " UTC"} {
		if _, err := New(Profile{ID: "managed", ProfileMode: "fresh", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{TimezoneID: zone}}, nil); !errors.Is(err, ErrInvalidProfile) || !strings.Contains(err.Error(), "identity timezone") {
			t.Errorf("invalid timezone %q error = %v, want field-specific profile rejection", zone, err)
		}
	}
}

func TestOpenRequestRejectsIdentityBackendMismatchBeforeSessionAdmission(t *testing.T) {
	profile := Profile{ID: "managed", ProfileMode: "fresh", Effect: EffectObserve, Approval: ApprovalNone, View: true, IdentityProfile: &BrowserIdentityProfile{Locale: "en-GB"}}
	request := OpenRequest{OwnerID: "owner", CallerID: "caller", ProfileID: "managed", Backend: BackendUserAttached, TabID: "tab", ConsentReceipt: "consent"}
	if err := request.Validate(profile); err == nil || !strings.Contains(err.Error(), "user-attached open requires") {
		t.Fatalf("attached backend accepted managed identity profile: %v", err)
	}
}

func TestAttachedBrowserRevalidatesConsentForEveryScopedOperation(t *testing.T) {
	profile := Profile{ID: "attached", ProfileMode: "user_attached", Effect: EffectAct, Approval: ApprovalNone, View: true, Control: true, URLAllow: []string{"https://example.test/account/*"}}
	verifier := &browserTabConsentVerifier{accepted: true}
	policy, err := NewWithVerifiers(profile, nil, verifier)
	if err != nil {
		t.Fatal(err)
	}
	req := ActionRequest{SessionID: "session", LeaseID: "lease", OwnerID: "owner", CallerID: "run:owner", Kind: "observe", CurrentURL: "https://example.test/account/home", TabID: "7", ConsentReceipt: "consent"}
	if err := policy.Authorize(context.Background(), req); err != nil {
		t.Fatalf("active tab consent denied observation: %v", err)
	}
	if len(verifier.requests) != 1 || verifier.requests[0] != req {
		t.Fatalf("consent owner did not receive exact operation binding: %#v", verifier.requests)
	}
	verifier.accepted = false
	req.Kind = "click"
	if err := policy.Authorize(context.Background(), req); !errors.Is(err, ErrDenied) {
		t.Fatalf("revoked consent allowed action: %v", err)
	}
	if len(verifier.requests) != 2 || verifier.requests[1] != req {
		t.Fatalf("revocation verification lost exact action binding: %#v", verifier.requests)
	}
}

func TestAudioPolicyRequiresRunGrantedModeAndScopesPlayback(t *testing.T) {
	base := Profile{ID: "audio-profile", ProfileID: "audio-profile", ProfileMode: "fresh", Effect: EffectAct, Approval: ApprovalNone, View: true, Control: true, URLAllow: []string{"https://dictation.example.test/*"}}
	base.AudioMode = AudioModePipeWire
	p, err := New(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", CurrentURL: "https://dictation.example.test/app", Kind: "audio_playback_restart"}
	if err := p.Authorize(context.Background(), req); err != nil {
		t.Fatalf("granted PipeWire playback restart denied: %v", err)
	}
	req.Kind = "audio_playback_stop"
	if err := p.Authorize(context.Background(), req); err != nil {
		t.Fatalf("granted PipeWire playback stop denied: %v", err)
	}
	req.CurrentURL = "https://outside.example.test/"
	if err := p.Authorize(context.Background(), req); !errors.Is(err, ErrDenied) {
		t.Fatalf("out-of-scope audio command should be denied, got %v", err)
	}
	base.AudioMode = AudioModeFakeMicrophone
	if _, err := New(base, nil); err != nil {
		t.Fatalf("valid fake-microphone profile rejected: %v", err)
	}
	policy, err := New(base, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.CurrentURL, req.Kind = "https://dictation.example.test/app", "audio_playback_restart"
	if err := policy.Authorize(context.Background(), req); !errors.Is(err, ErrDenied) {
		t.Fatalf("fake-microphone mode must not control host playback, got %v", err)
	}
}

func TestProfileURLScopeIsDenyByDefaultAndDenyWins(t *testing.T) {
	p := Profile{Effect: EffectAct, Approval: ApprovalNone, URLAllow: []string{"https://*.example.test/app/*"}, URLDeny: []string{"https://private.example.test/*"}}
	for _, tc := range []struct {
		raw     string
		allowed bool
	}{
		{"https://shop.example.test/app/cart", true},
		{"https://example.test/app/cart", false},
		{"https://shop.example.test/other", false},
		{"https://private.example.test/app/cart", false},
		{"javascript:alert(1)", false},
		{"https://user:pass@shop.example.test/app/cart", false},
	} {
		if got := p.AllowsURL(tc.raw); got != tc.allowed {
			t.Errorf("AllowsURL(%q)=%v, want %v", tc.raw, got, tc.allowed)
		}
	}
}

func TestAuthorizerEnforcesObserveScopeApprovalAndRawCDP(t *testing.T) {
	request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: "click", CurrentURL: "https://shop.example.test/app/home"}
	observe, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, URLAllow: []string{"https://shop.example.test/*"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := observe.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("observe should deny click, got %v", err)
	}

	approval, err := New(Profile{Effect: EffectAct, Approval: ApprovalSafetyPass, Control: true, URLAllow: []string{"https://shop.example.test/*"}}, receiptVerifier{accepted: true})
	if err != nil {
		t.Fatal(err)
	}
	request.Kind = "purchase"
	if err := approval.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("purchase should require receipt, got %v", err)
	}
	request.ApprovalReceipt = "approved"
	if err := approval.Authorize(context.Background(), request); err != nil {
		t.Fatalf("scoped receipt should authorize, got %v", err)
	}
	request.Kind, request.ApprovalReceipt = "raw_cdp", ""
	if err := approval.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("raw CDP should require unrestricted, got %v", err)
	}
}

func TestNavigateChecksTargetAndProfileValidation(t *testing.T) {
	p, err := New(Profile{Effect: EffectAct, Approval: ApprovalNone, URLAllow: []string{"https://good.example.test/*"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: "navigate", TargetURL: "https://evil.example.test/"}
	if err := p.Authorize(context.Background(), req); !errors.Is(err, ErrDenied) {
		t.Fatalf("out-of-scope navigation should be denied, got %v", err)
	}
	if _, err := New(Profile{Effect: "magic", Approval: ApprovalNone}, nil); !errors.Is(err, ErrInvalidProfile) {
		t.Fatalf("invalid effect accepted: %v", err)
	}
}

func TestObserveRequiresCurrentURLInsideProfileScope(t *testing.T) {
	policy, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, URLAllow: []string{"https://allowed.example.test/*"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: "observe", CurrentURL: "https://denied.example.test/private"}
	if err := policy.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("out-of-scope observation = %v, want denial", err)
	}
	request.CurrentURL = "https://allowed.example.test/path"
	if err := policy.Authorize(context.Background(), request); err != nil {
		t.Fatalf("in-scope observation was denied: %v", err)
	}
}

func TestCredentialUseRequiresVerifierAndExactCurrentLocation(t *testing.T) {
	profile := Profile{Effect: EffectAct, Approval: ApprovalNone, Control: true, URLAllow: []string{"https://shop.example.test/*"}, CredentialIDs: []string{"vault-ref-1"}}
	request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: "credential_use", CredentialID: "vault-ref-1", CurrentURL: "https://shop.example.test/app"}
	withoutVerifier, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := withoutVerifier.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("credential use without an authority verifier = %v, want denial", err)
	}
	withVerifier, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	withVerifier.SetCredentialUseVerifier(credentialVerifier{accepted: true})
	request.CurrentURL = "https://shop.example.test.attacker.invalid/app"
	if err := withVerifier.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("credential use at a host-confusion origin = %v, want denial", err)
	}
	request.CurrentURL = "https://shop.example.test/app"
	if err := withVerifier.Authorize(context.Background(), request); err != nil {
		t.Fatalf("exact-origin credential use was denied: %v", err)
	}
}

func TestApprovalProfileRequiresOwnerReceiptVerifier(t *testing.T) {
	for _, tc := range []struct {
		name      string
		approval  Approval
		kind      string
		audioMode AudioMode
	}{
		{name: "safety pass high impact", approval: ApprovalSafetyPass, kind: "submit"},
		{name: "safety pass typed input", approval: ApprovalSafetyPass, kind: "input"},
		{name: "safety pass reset", approval: ApprovalSafetyPass, kind: "reset"},
		{name: "safety pass audio playback", approval: ApprovalSafetyPass, kind: "audio_playback_restart", audioMode: AudioModePipeWire},
		{name: "every action", approval: ApprovalEveryAction, kind: "click"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policy, err := New(Profile{Effect: EffectAct, Approval: tc.approval, Control: true, View: true, AudioMode: tc.audioMode, ProfileMode: "fresh", URLAllow: []string{"https://shop.example.test/*"}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: tc.kind, CurrentURL: "https://shop.example.test/app", ApprovalReceipt: "approved"}
			if err := policy.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
				t.Fatalf("approval without the owner verifier = %v, want denial", err)
			}
		})
	}
}

func TestCredentialScopedProfileCannotTypeRawInput(t *testing.T) {
	policy, err := New(Profile{Effect: EffectAct, Approval: ApprovalNone, Control: true, URLAllow: []string{"https://shop.example.test/*"}, CredentialIDs: []string{"vault-ref-1"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: "input", CurrentURL: "https://shop.example.test/login"}
	if err := policy.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("raw input bypassed credential reference flow: %v", err)
	}
}

func TestCreditLimitedProfileRequiresOwnerBudgetVerifier(t *testing.T) {
	policy, err := New(Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, URLAllow: []string{"https://shop.example.test/*"}, MaxCredits: 10}, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: "observe", CurrentURL: "https://shop.example.test/app"}
	if err := policy.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("credit-limited profile without budget authority = %v, want denial", err)
	}
}

type byteBudgetVerifier struct{ accepted bool }

func (v byteBudgetVerifier) AuthorizeBrowserBytes(_ context.Context, request ActionRequest, maximum int64) error {
	if !v.accepted || request.SessionID != "s1" || maximum != 32 {
		return errors.New("byte budget scope mismatch")
	}
	return nil
}

func TestByteLimitedProfileRequiresOwnerMeasuredBudgetVerifier(t *testing.T) {
	profile := Profile{Effect: EffectObserve, Approval: ApprovalNone, View: true, URLAllow: []string{"https://shop.example.test/*"}, MaxBytes: 32}
	request := ActionRequest{SessionID: "s1", LeaseID: "l1", OwnerID: "o1", CallerID: "caller-1", Kind: "observe", CurrentURL: "https://shop.example.test/app"}
	withoutVerifier, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := withoutVerifier.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("byte-limited profile without authority = %v, want denial", err)
	}
	withVerifier, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	withVerifier.SetByteBudgetVerifier(byteBudgetVerifier{accepted: true})
	if err := withVerifier.Authorize(context.Background(), request); err != nil {
		t.Fatalf("owner-authorized byte-limited observe denied: %v", err)
	}
	denied, err := New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	denied.SetByteBudgetVerifier(byteBudgetVerifier{accepted: false})
	if err := denied.Authorize(context.Background(), request); !errors.Is(err, ErrDenied) {
		t.Fatalf("owner-denied byte-limited observe = %v, want denial", err)
	}
}
