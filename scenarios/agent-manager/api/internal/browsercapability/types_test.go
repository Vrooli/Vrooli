package browsercapability

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type callerResolverFunc func(context.Context) (TrustedCaller, error)

func (f callerResolverFunc) ResolveTrustedCaller(ctx context.Context) (TrustedCaller, error) {
	return f(ctx)
}

type runReaderFunc func(context.Context, string) (Capability, error)

func (f runReaderFunc) BrowserCapability(ctx context.Context, runID string) (Capability, error) {
	return f(ctx, runID)
}

type grantIssuerFunc func(context.Context, TrustedCaller, Capability, time.Time) (Grant, error)

func (f grantIssuerFunc) IssueBrowserGrant(ctx context.Context, caller TrustedCaller, capability Capability, expires time.Time) (Grant, error) {
	return f(ctx, caller, capability, expires)
}

func validCapability() Capability {
	return Capability{
		Enabled:     true,
		Candidates:  []Candidate{{Harness: "claude-code", ResourceRole: "browser.session"}},
		Mode:        ModeAttached,
		Permissions: PermissionProfileSpec{ProfileID: "profile-act", ProfileMode: SessionProfileUserAttached, Level: ProfileAct, AllowedOrigins: []string{"https://example.test"}, CredentialRefs: []string{"credential-owner:site-login"}},
		Budget:      Budget{SessionMinutes: 15, Actions: 200, Bytes: 8_000_000, InferenceTokens: 40_000},
	}
}

func TestValidateRequiresBoundedRunScopedCapability(t *testing.T) {
	if err := Validate(validCapability()); err != nil {
		t.Fatalf("valid capability rejected: %v", err)
	}
	cases := []struct {
		name string
		edit func(*Capability)
		want string
	}{
		{"unknown mode", func(c *Capability) { c.Mode = "stealth" }, "unsupported browser mode"},
		{"profile mode mismatches backend", func(c *Capability) { c.Permissions.ProfileMode = SessionProfileFresh }, "does not match backend mode"},
		{"missing profile id", func(c *Capability) { c.Permissions.ProfileID = " " }, "requires a stable profile ID"},
		{"missing budget", func(c *Capability) { c.Budget.Actions = 0 }, "positive session, action and byte limits"},
		{"oversized budget", func(c *Capability) { c.Budget.Bytes = MaxBytes + 1 }, "exceeds the hard session"},
		{"path in origin scope", func(c *Capability) { c.Permissions.AllowedOrigins = []string{"https://example.test/private"} }, "must be an http(s) origin"},
		{"observe history", func(c *Capability) { c.Permissions.Level = ProfileObserve; c.Permissions.AllowHistory = true }, "observe profile cannot grant browser history"},
		{"identity in attached mode", func(c *Capability) { c.Permissions.IdentityProfile = &IdentityProfile{Preset: "balanced"} }, "unsupported for user-attached and remote"},
		{"extension install in attached mode", func(c *Capability) { c.Permissions.ExtensionInstall = true }, "only in managed browser mode"},
		{"extension load in attached mode", func(c *Capability) { c.Permissions.ExtensionLoad = true }, "only in managed browser mode"},
		{"App Monitor metrics in attached mode", func(c *Capability) { c.Permissions.AppMonitorMetrics = true }, "only in managed browser mode"},
		{"App Monitor trace without ceilings", func(c *Capability) {
			c.Mode = ModeManaged
			c.Permissions.ProfileMode = SessionProfileFresh
			c.Permissions.View = true
			c.Permissions.AppMonitorTrace = true
		}, "requires positive duration and byte limits"},
		{"extension install missing profile mode", func(c *Capability) { c.Mode = ModeManaged; c.Permissions.ExtensionInstall = true }, "require the persisted fresh managed profile mode"},
		{"extension install without allowlist", func(c *Capability) {
			c.Mode = ModeManaged
			c.Permissions.ProfileMode = SessionProfileFresh
			c.Permissions.ExtensionInstall = true
			c.Permissions.ExtensionIDs = nil
		}, "require an existing exact extension ID allowlist"},
		{"extension load without allowlist", func(c *Capability) {
			c.Mode = ModeManaged
			c.Permissions.ProfileMode = SessionProfileFresh
			c.Permissions.ExtensionLoad = true
			c.Permissions.ExtensionIDs = nil
		}, "require an existing exact extension ID allowlist"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capability := validCapability()
			tc.edit(&capability)
			if err := Validate(capability); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestAppMonitorCapturePermissionsAreDefaultDenyBoundedAndDelegationSafe(t *testing.T) {
	legacy := validCapability()
	if legacy.Permissions.AppMonitorMetrics || legacy.Permissions.AppMonitorTrace || legacy.Permissions.AppMonitorTraceMaxDurationMS != 0 || legacy.Permissions.AppMonitorTraceMaxBytes != 0 {
		t.Fatalf("legacy profile did not default App Monitor capture to deny: %+v", legacy.Permissions)
	}
	parent := validCapability()
	parent.Mode = ModeManaged
	parent.Permissions.ProfileMode = SessionProfileFresh
	parent.Permissions.View = true
	parent.Permissions.AppMonitorMetrics = true
	parent.Permissions.AppMonitorTrace = true
	parent.Permissions.AppMonitorTraceMaxDurationMS = 30_000
	parent.Permissions.AppMonitorTraceMaxBytes = 8 << 20
	if err := Validate(parent); err != nil {
		t.Fatalf("valid bounded App Monitor grant rejected: %v", err)
	}
	child := parent.Clone()
	child.Permissions.AppMonitorMetrics = false
	child.Permissions.AppMonitorTrace = false
	child.Permissions.AppMonitorTraceMaxDurationMS = 0
	child.Permissions.AppMonitorTraceMaxBytes = 0
	if !Within(&parent, &child) {
		t.Fatal("delegation may narrow App Monitor capture permissions")
	}
	deniedParent := parent.Clone()
	deniedParent.Permissions.AppMonitorMetrics = false
	deniedParent.Permissions.AppMonitorTrace = false
	deniedParent.Permissions.AppMonitorTraceMaxDurationMS = 0
	deniedParent.Permissions.AppMonitorTraceMaxBytes = 0
	if Within(&deniedParent, &parent) {
		t.Fatal("delegation widened App Monitor capture permission")
	}
	broaderChild := parent.Clone()
	broaderChild.Permissions.AppMonitorTraceMaxBytes = parent.Permissions.AppMonitorTraceMaxBytes + 1
	if Within(&parent, &broaderChild) {
		t.Fatal("delegation widened the App Monitor trace byte ceiling")
	}
	tooLarge := parent.Clone()
	tooLarge.Permissions.AppMonitorTraceMaxBytes = MaxAppMonitorTraceBytes + 1
	if err := Validate(tooLarge); err == nil {
		t.Fatal("oversized App Monitor trace ceiling was accepted")
	}
}

func TestExtensionInstallCapabilityIsExplicitBoundedAndDelegationCannotBroaden(t *testing.T) {
	parent := validCapability()
	parent.Mode = ModeManaged
	parent.Permissions.ProfileMode = SessionProfileFresh
	parent.Permissions.ExtensionIDs = []string{"extension-approved"}
	parent.Permissions.ExtensionInstall = true
	parent.Permissions.ExtensionLoad = true
	if err := Validate(parent); err != nil {
		t.Fatalf("valid managed install capability rejected: %v", err)
	}
	clone := parent.Clone()
	if !clone.Permissions.ExtensionInstall || !clone.Permissions.ExtensionLoad {
		t.Fatal("clone lost explicit extension lifecycle permission")
	}
	clone.Permissions.ExtensionInstall = false
	clone.Permissions.ExtensionLoad = false
	if !parent.Permissions.ExtensionInstall || !parent.Permissions.ExtensionLoad {
		t.Fatal("clone mutation changed persisted extension lifecycle permission")
	}

	child := parent.Clone()
	child.Permissions.ExtensionInstall = false
	if !Within(&parent, &child) {
		t.Fatal("delegation may remove extension install permission")
	}
	child = parent.Clone()
	child.Permissions.ExtensionLoad = false
	if !Within(&parent, &child) {
		t.Fatal("delegation may remove extension load permission")
	}
	parentDenied := parent.Clone()
	parentDenied.Permissions.ExtensionInstall = false
	if Within(&parentDenied, &parent) {
		t.Fatal("delegation broadened extension install permission")
	}
	parentDenied = parent.Clone()
	parentDenied.Permissions.ExtensionLoad = false
	if Within(&parentDenied, &parent) {
		t.Fatal("delegation broadened extension load permission")
	}
	child = parent.Clone()
	child.Permissions.ExtensionIDs = []string{"extension-unapproved"}
	if Within(&parent, &child) {
		t.Fatal("delegation broadened exact extension ID allowlist")
	}
}

func TestManagedIdentityProfileIsBoundedAndCloned(t *testing.T) {
	capability := validCapability()
	capability.Mode = ModeManaged
	capability.Permissions.ProfileMode = SessionProfileFresh
	capability.Permissions.IdentityProfile = &IdentityProfile{
		Preset: "balanced", ViewportWidth: 1440, ViewportHeight: 900,
		DeviceScaleFactor: 1.25, Locale: "en-GB", TimezoneID: "Europe/London", ColorScheme: "dark",
	}
	if err := Validate(capability); err != nil {
		t.Fatalf("valid managed identity profile rejected: %v", err)
	}

	clone := capability.Clone()
	clone.Permissions.IdentityProfile.Locale = "fr-FR"
	if capability.Permissions.IdentityProfile.Locale != "en-GB" {
		t.Fatal("cloned capability shares mutable identity profile")
	}

	for name, identity := range map[string]*IdentityProfile{
		"partial viewport": {ViewportWidth: 1200},
		"bad locale":       {Locale: "en_US"},
		"malformed locale": {Locale: "en--US"},
		"bad timezone":     {TimezoneID: "Mars/Olympus"},
		"host timezone":    {TimezoneID: "Local"},
		"stale UA persona": {Preset: "firefox-linux"},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := capability.Clone()
			candidate.Permissions.IdentityProfile = identity
			if err := Validate(candidate); err == nil {
				t.Fatal("invalid identity profile unexpectedly accepted")
			}
		})
	}
}

func TestIdentityProfileModeMatrixAndDelegationSubset(t *testing.T) {
	installed := validCapability()
	installed.Mode = ModeInstalled
	installed.Permissions.ProfileMode = SessionProfileIsolated
	installed.Permissions.IdentityProfile = &IdentityProfile{ViewportWidth: 1365, ViewportHeight: 768}
	if err := Validate(installed); err != nil {
		t.Fatalf("installed viewport identity profile rejected: %v", err)
	}
	for name, identity := range map[string]*IdentityProfile{
		"preset": {Preset: "balanced"}, "scale": {DeviceScaleFactor: 1.25}, "locale": {Locale: "en-GB"},
		"timezone": {TimezoneID: "UTC"}, "color scheme": {ColorScheme: "dark"},
	} {
		t.Run("installed rejects "+name, func(t *testing.T) {
			candidate := installed.Clone()
			candidate.Permissions.IdentityProfile = identity
			if err := Validate(candidate); err == nil || !strings.Contains(err.Error(), "viewport only") {
				t.Fatalf("unsupported installed identity %s error = %v", name, err)
			}
		})
	}
	for _, mode := range []Mode{ModeAttached, ModeRemote} {
		candidate := installed.Clone()
		candidate.Mode = mode
		candidate.Permissions.ProfileMode, _ = ProfileModeForMode(mode)
		if err := Validate(candidate); err == nil || !strings.Contains(err.Error(), "unsupported for user-attached and remote") {
			t.Fatalf("identity customization for %s error = %v", mode, err)
		}
	}

	parent := validCapability()
	parent.Mode = ModeManaged
	parent.Permissions.ProfileMode = SessionProfileFresh
	parent.Permissions.IdentityProfile = &IdentityProfile{Preset: "balanced", ViewportWidth: 1440, ViewportHeight: 900, Locale: "en-GB"}
	child := parent.Clone()
	child.Permissions.IdentityProfile = &IdentityProfile{ViewportWidth: 1440, ViewportHeight: 900}
	if !Within(&parent, &child) {
		t.Fatal("delegation could not narrow identity fields to an exact parent subset")
	}
	child.Permissions.IdentityProfile.Locale = "fr-FR"
	if Within(&parent, &child) {
		t.Fatal("delegation changed a persisted identity field")
	}
	child.Permissions.IdentityProfile = &IdentityProfile{Preset: "fast"}
	if Within(&parent, &child) {
		t.Fatal("delegation added an identity field absent from parent")
	}
	if Within(nil, &child) {
		t.Fatal("identity profile was added without a parent browser capability")
	}
	child.Permissions.IdentityProfile = nil
	if !Within(&parent, &child) {
		t.Fatal("delegation could not omit the parent identity profile")
	}
}

func TestIdentityProfileRequiresPersistedBackendProfileMode(t *testing.T) {
	candidate := validCapability()
	candidate.Mode = ModeManaged
	candidate.Permissions.ProfileMode = ""
	candidate.Permissions.IdentityProfile = &IdentityProfile{Locale: "en-GB"}
	if err := Validate(candidate); err == nil || !strings.Contains(err.Error(), "explicit persisted profile mode") {
		t.Fatalf("identity profile without explicit mode error = %v", err)
	}
}

func TestManagedIdentityAcceptsBCP47AndExplicitTimezoneAliases(t *testing.T) {
	validate := func(identity *IdentityProfile) error {
		capability := validCapability()
		capability.Mode = ModeManaged
		capability.Permissions.ProfileMode = SessionProfileFresh
		capability.Permissions.IdentityProfile = identity
		return Validate(capability)
	}
	for _, locale := range []string{"en-GB", "zh-Hant-TW", "en-US-u-ca-gregory", "i-klingon"} {
		identity := &IdentityProfile{Locale: locale}
		if err := validate(identity); err != nil {
			t.Errorf("valid BCP 47 locale %q rejected: %v", locale, err)
		}
		if identity.Locale != locale {
			t.Errorf("valid locale was normalized from %q to %q", locale, identity.Locale)
		}
	}
	for _, locale := range []string{"en_US", "en--US", "en-GB-", "en-US-abc-abc", "not_a_tag", "en-GB\n"} {
		if err := validate(&IdentityProfile{Locale: locale}); err == nil || !strings.Contains(err.Error(), "identity locale") {
			t.Errorf("malformed locale %q error = %v, want locale validation failure", locale, err)
		}
	}
	for _, zone := range []string{"UTC", "Etc/UTC", "Etc/GMT", "GMT", "America/New_York", "US/Eastern"} {
		identity := &IdentityProfile{TimezoneID: zone}
		if err := validate(identity); err != nil {
			t.Errorf("supported timezone %q rejected: %v", zone, err)
		}
		if identity.TimezoneID != zone {
			t.Errorf("supported timezone was normalized from %q to %q", zone, identity.TimezoneID)
		}
	}
	for _, zone := range []string{"Local", "Mars/Olympus", "../../etc/passwd", " UTC"} {
		if err := validate(&IdentityProfile{TimezoneID: zone}); err == nil || !strings.Contains(err.Error(), "identity timezone") {
			t.Errorf("invalid timezone %q error = %v, want timezone validation failure", zone, err)
		}
	}
}

func TestAudioPermissionRequiresExplicitManagedControlAndCannotBeBroadened(t *testing.T) {
	audio := validCapability()
	audio.Mode = ModeManaged
	audio.Permissions.ProfileMode = SessionProfileFresh
	audio.Permissions.View = true
	audio.Permissions.Control = true
	audio.Permissions.AudioMode = AudioProfileFakeMicrophone
	if err := Validate(audio); err != nil {
		t.Fatalf("valid explicit fake-microphone capability rejected: %v", err)
	}

	for name, mutate := range map[string]func(*Capability){
		"attached backend": func(value *Capability) {
			value.Mode = ModeAttached
			value.Permissions.ProfileMode = SessionProfileUserAttached
		},
		"no view":      func(value *Capability) { value.Permissions.View = false },
		"no control":   func(value *Capability) { value.Permissions.Control = false },
		"no origin":    func(value *Capability) { value.Permissions.AllowedOrigins = nil },
		"unknown mode": func(value *Capability) { value.Permissions.AudioMode = "raw_microphone" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := audio.Clone()
			mutate(&candidate)
			if err := Validate(candidate); err == nil {
				t.Fatal("invalid audio capability was accepted")
			}
		})
	}
	pipewire := audio.Clone()
	pipewire.Permissions.AudioMode = AudioProfilePipeWire
	if Within(&audio, &pipewire) {
		t.Fatal("a fake-microphone grant was broadened to host PipeWire evidence")
	}
}

func TestProfileModeForModeIsExplicitAndStable(t *testing.T) {
	for _, tc := range []struct {
		mode Mode
		want SessionProfileMode
	}{
		{ModeManaged, SessionProfileFresh},
		{ModeInstalled, SessionProfileIsolated},
		{ModeAttached, SessionProfileUserAttached},
		{ModeRemote, SessionProfileRemote},
	} {
		got, ok := ProfileModeForMode(tc.mode)
		if !ok || got != tc.want {
			t.Errorf("ProfileModeForMode(%q) = %q,%v; want %q,true", tc.mode, got, ok, tc.want)
		}
	}
	if _, ok := ProfileModeForMode("unknown"); ok {
		t.Fatal("unknown backend must not gain a session profile mode")
	}
}

func TestAuthorizeUsesTrustedRunAndExactHarnessRoleCandidate(t *testing.T) {
	capability := validCapability()
	issued := false
	authorizer := &Authorizer{
		Callers: callerResolverFunc(func(context.Context) (TrustedCaller, error) {
			return TrustedCaller{RunID: "run-1", Harness: "claude-code", ResourceRole: "browser.session"}, nil
		}),
		Runs: runReaderFunc(func(_ context.Context, id string) (Capability, error) {
			if id != "run-1" {
				t.Fatalf("reader run id = %q", id)
			}
			return capability, nil
		}),
		Issuer: grantIssuerFunc(func(_ context.Context, caller TrustedCaller, got Capability, expiry time.Time) (Grant, error) {
			issued = true
			if caller.RunID != "run-1" || got.Mode != capability.Mode || got.Budget != capability.Budget || !expiry.After(time.Now()) {
				t.Fatalf("issuer got caller=%+v, capability=%+v, expiry=%s", caller, got, expiry)
			}
			return Grant{ID: "grant-1", RunID: caller.RunID, ExpiresAt: expiry}, nil
		}),
		Now: func() time.Time { return time.Now().UTC() },
		TTL: 10 * time.Minute,
	}
	grant, err := authorizer.Authorize(context.Background(), Request{})
	if err != nil || grant.ID != "grant-1" || !issued {
		t.Fatalf("Authorize() = %+v, %v, issued=%v", grant, err, issued)
	}
}

func TestAuthorizeRejectsUntrustedWideningAndWrongCandidate(t *testing.T) {
	capability := validCapability()
	issuerCalled := false
	authorizer := &Authorizer{
		Callers: callerResolverFunc(func(context.Context) (TrustedCaller, error) {
			return TrustedCaller{RunID: "run-1", Harness: "codex", ResourceRole: "browser.session"}, nil
		}),
		Runs: runReaderFunc(func(context.Context, string) (Capability, error) { return capability, nil }),
		Issuer: grantIssuerFunc(func(context.Context, TrustedCaller, Capability, time.Time) (Grant, error) {
			issuerCalled = true
			return Grant{}, nil
		}),
	}
	if _, err := authorizer.Authorize(context.Background(), Request{}); !errors.Is(err, ErrNoEligibleCandidate) {
		t.Fatalf("wrong candidate error = %v", err)
	}
	authorizer.Callers = callerResolverFunc(func(context.Context) (TrustedCaller, error) {
		return TrustedCaller{RunID: "run-1", Harness: "claude-code", ResourceRole: "browser.session"}, nil
	})
	authorizer.Runs = runReaderFunc(func(context.Context, string) (Capability, error) {
		denied := capability
		denied.Enabled = false
		return denied, nil
	})
	if _, err := authorizer.Authorize(context.Background(), Request{}); !errors.Is(err, ErrRunNotAuthorized) {
		t.Fatalf("disabled run error = %v", err)
	}
	if issuerCalled {
		t.Fatal("issuer was called for a rejected capability")
	}
}

func TestCapabilityDelegationOnlyNarrows(t *testing.T) {
	parent := validCapability()
	parent.Permissions.View = true
	parent.Permissions.Control = true
	child := parent.Clone()
	child.Permissions.Level = ProfileObserve
	child.Budget.Actions = 40
	if !Within(&parent, &child) {
		t.Fatal("valid narrower child capability was rejected")
	}
	child = parent.Clone()
	child.Permissions.AllowedOrigins = []string{"https://outside.test"}
	if Within(&parent, &child) {
		t.Fatal("child URL scope widened parent capability")
	}
	child = parent.Clone()
	child.Budget.SessionMinutes++
	if Within(&parent, &child) {
		t.Fatal("child budget widened parent capability")
	}
	child = parent.Clone()
	child.Permissions.ProfileMode = SessionProfileFresh
	if Within(&parent, &child) {
		t.Fatal("child changed the persisted session isolation mode")
	}
	child = parent.Clone()
	child.Permissions.View = false
	child.Permissions.Control = true
	if Within(&parent, &child) {
		t.Fatal("control without view was accepted")
	}
	child = parent.Clone()
	child.Permissions.Control = false
	child.Permissions.View = false
	if !Within(&parent, &child) {
		t.Fatal("child could not attenuate stream permissions")
	}
}

func TestValidateControlRequiresExplicitView(t *testing.T) {
	capability := validCapability()
	capability.Permissions.Control = true
	if err := Validate(capability); err == nil {
		t.Fatal("control without explicit view was accepted")
	}
	capability.Permissions.View = true
	if err := Validate(capability); err != nil {
		t.Fatalf("explicit view+control rejected: %v", err)
	}
}

func TestAuthorizeOpenBindsGrantToRunModeProfileAndOrigin(t *testing.T) {
	grant := Grant{ID: "grant-1", RunID: "run-1", Mode: ModeAttached,
		PermissionProfile: validCapability().Permissions, ExpiresAt: time.Now().Add(time.Minute)}
	authorizer := &Authorizer{
		Callers: callerResolverFunc(func(context.Context) (TrustedCaller, error) { return TrustedCaller{RunID: "run-1"}, nil }),
		Grants: grantReaderFunc(func(_ context.Context, id string) (Grant, error) {
			if id != grant.ID {
				t.Fatalf("grant id = %q", id)
			}
			return grant, nil
		}),
	}
	if _, err := authorizer.AuthorizeOpen(context.Background(), OpenRequest{GrantReceipt: "grant-1", Mode: ModeAttached, ProfileID: "profile-act", InitialURL: "https://example.test/login"}); err != nil {
		t.Fatalf("authorized open rejected: %v", err)
	}
	for _, request := range []OpenRequest{
		{GrantReceipt: "grant-1", Mode: ModeRemote, ProfileID: "profile-act", InitialURL: "https://example.test"},
		{GrantReceipt: "grant-1", Mode: ModeAttached, ProfileID: "profile-observe", InitialURL: "https://example.test"},
		{GrantReceipt: "grant-1", Mode: ModeAttached, ProfileID: "profile-act", InitialURL: "https://outside.test"},
	} {
		if _, err := authorizer.AuthorizeOpen(context.Background(), request); err == nil || (!errors.Is(err, ErrRunNotAuthorized) && !strings.Contains(err.Error(), "outside the granted origin")) {
			t.Fatalf("widened open request %+v error = %v", request, err)
		}
	}
}

type grantReaderFunc func(context.Context, string) (Grant, error)

func (f grantReaderFunc) ReadBrowserGrant(ctx context.Context, grantID string) (Grant, error) {
	return f(ctx, grantID)
}
