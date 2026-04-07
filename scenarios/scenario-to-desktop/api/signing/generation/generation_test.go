package generation

import (
	"strings"
	"testing"

	"scenario-to-desktop-api/signing/types"
)

// --- generateWindowsConfig tests ---

func TestGenerateWindowsConfig_Nil(t *testing.T) {
	got := generateWindowsConfig(nil, nil)
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestGenerateWindowsConfig_FileSource(t *testing.T) {
	cfg := &types.WindowsSigningConfig{
		CertificateSource:      types.CertSourceFile,
		CertificateFile:        "/path/to/cert.pfx",
		CertificatePasswordEnv: "WIN_CERT_PASS",
	}
	got := generateWindowsConfig(cfg, nil)

	if got.CertificateFile != "/path/to/cert.pfx" {
		t.Errorf("CertificateFile = %q, want /path/to/cert.pfx", got.CertificateFile)
	}
	if got.CertificatePassword != "${WIN_CERT_PASS}" {
		t.Errorf("CertificatePassword = %q, want ${WIN_CERT_PASS}", got.CertificatePassword)
	}
	if !got.SignAndEditExecutable {
		t.Error("SignAndEditExecutable should be true")
	}
	if !got.SignDlls {
		t.Error("SignDlls should be true")
	}
}

func TestGenerateWindowsConfig_StoreSource(t *testing.T) {
	cfg := &types.WindowsSigningConfig{
		CertificateSource:     types.CertSourceStore,
		CertificateThumbprint: "AABB1122",
	}
	got := generateWindowsConfig(cfg, nil)

	if got.CertificateSha1 != "AABB1122" {
		t.Errorf("CertificateSha1 = %q, want AABB1122", got.CertificateSha1)
	}
}

func TestGenerateWindowsConfig_DefaultTimestamp(t *testing.T) {
	cfg := &types.WindowsSigningConfig{}
	got := generateWindowsConfig(cfg, nil)

	if got.Rfc3161TimeStampServer != types.DefaultTimestampServerDigiCert {
		t.Errorf("timestamp = %q, want DigiCert default", got.Rfc3161TimeStampServer)
	}
}

func TestGenerateWindowsConfig_CustomTimestamp(t *testing.T) {
	cfg := &types.WindowsSigningConfig{
		TimestampServer: "http://custom.ts.server",
	}
	got := generateWindowsConfig(cfg, nil)

	if got.Rfc3161TimeStampServer != "http://custom.ts.server" {
		t.Errorf("timestamp = %q, want custom server", got.Rfc3161TimeStampServer)
	}
}

func TestGenerateWindowsConfig_DualSign(t *testing.T) {
	cfg := &types.WindowsSigningConfig{DualSign: true}
	got := generateWindowsConfig(cfg, nil)

	if len(got.SigningHashAlgorithms) != 2 {
		t.Fatalf("SigningHashAlgorithms len = %d, want 2", len(got.SigningHashAlgorithms))
	}
	if got.SigningHashAlgorithms[0] != "sha1" || got.SigningHashAlgorithms[1] != "sha256" {
		t.Errorf("algorithms = %v, want [sha1, sha256]", got.SigningHashAlgorithms)
	}
}

func TestGenerateWindowsConfig_SingleAlgorithm(t *testing.T) {
	tests := []struct {
		name     string
		algo     string
		expected string
	}{
		{"explicit sha512", "sha512", "sha512"},
		{"default to sha256", "", types.SignAlgorithmSHA256},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &types.WindowsSigningConfig{SignAlgorithm: tc.algo}
			got := generateWindowsConfig(cfg, nil)
			if len(got.SigningHashAlgorithms) != 1 || got.SigningHashAlgorithms[0] != tc.expected {
				t.Errorf("algorithms = %v, want [%s]", got.SigningHashAlgorithms, tc.expected)
			}
		})
	}
}

func TestGenerateWindowsConfig_EmptyPasswordEnv(t *testing.T) {
	cfg := &types.WindowsSigningConfig{
		CertificateSource:      types.CertSourceFile,
		CertificateFile:        "/cert.pfx",
		CertificatePasswordEnv: "",
	}
	got := generateWindowsConfig(cfg, nil)
	if got.CertificatePassword != "" {
		t.Errorf("CertificatePassword = %q, want empty when env not set", got.CertificatePassword)
	}
}

// --- generateMacOSConfig tests ---

func TestGenerateMacOSConfig_Nil(t *testing.T) {
	got := generateMacOSConfig(nil, nil)
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestGenerateMacOSConfig_BasicIdentity(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		Identity:        "Developer ID Application: Test (TEAMID)",
		HardenedRuntime: true,
	}
	got := generateMacOSConfig(cfg, nil)

	if got.Identity != cfg.Identity {
		t.Errorf("Identity = %q, want %q", got.Identity, cfg.Identity)
	}
	if !got.HardenedRuntime {
		t.Error("HardenedRuntime should be true")
	}
	if !got.GatekeeperAssess {
		t.Error("GatekeeperAssess should always be true")
	}
}

func TestGenerateMacOSConfig_ExplicitEntitlements(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		EntitlementsFile: "/custom/entitlements.plist",
	}
	got := generateMacOSConfig(cfg, nil)

	if got.Entitlements != "/custom/entitlements.plist" {
		t.Errorf("Entitlements = %q, want custom path", got.Entitlements)
	}
	if got.EntitlementsInherit != "/custom/entitlements.plist" {
		t.Errorf("EntitlementsInherit = %q, want custom path", got.EntitlementsInherit)
	}
}

func TestGenerateMacOSConfig_GeneratedEntitlements(t *testing.T) {
	cfg := &types.MacOSSigningConfig{}
	opts := &Options{
		OutputDir:        "out",
		EntitlementsPath: "ent.plist",
	}
	got := generateMacOSConfig(cfg, opts)

	if got.Entitlements != "out/ent.plist" {
		t.Errorf("Entitlements = %q, want out/ent.plist", got.Entitlements)
	}
}

func TestGenerateMacOSConfig_NotarizeWithTeamID(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		Notarize: true,
		TeamID:   "ABC123",
	}
	got := generateMacOSConfig(cfg, nil)

	nc, ok := got.Notarize.(*types.NotarizeConfig)
	if !ok {
		t.Fatalf("Notarize type = %T, want *NotarizeConfig", got.Notarize)
	}
	if nc.TeamID != "ABC123" {
		t.Errorf("TeamID = %q, want ABC123", nc.TeamID)
	}
}

func TestGenerateMacOSConfig_NotarizeWithoutTeamID(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		Notarize: true,
	}
	got := generateMacOSConfig(cfg, nil)

	if got.Notarize != true {
		t.Errorf("Notarize = %v, want true (bool)", got.Notarize)
	}
}

func TestGenerateMacOSConfig_ProvisioningProfile(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		ProvisioningProfile: "/path/to/profile.provisionprofile",
	}
	got := generateMacOSConfig(cfg, nil)

	if got.ProvisioningProfile != "/path/to/profile.provisionprofile" {
		t.Errorf("ProvisioningProfile = %q", got.ProvisioningProfile)
	}
}

// --- GenerateElectronBuilderJSON tests ---

func TestGenerateElectronBuilderJSON_Nil(t *testing.T) {
	got, err := GenerateElectronBuilderJSON(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestGenerateElectronBuilderJSON_Disabled(t *testing.T) {
	got, err := GenerateElectronBuilderJSON(&types.SigningConfig{Enabled: false}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for disabled config")
	}
}

func TestGenerateElectronBuilderJSON_WindowsOnly(t *testing.T) {
	cfg := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{
			CertificateSource: types.CertSourceFile,
			CertificateFile:   "/cert.pfx",
		},
	}
	got, err := GenerateElectronBuilderJSON(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["win"] == nil {
		t.Error("expected 'win' key in result")
	}
	if got["mac"] != nil {
		t.Error("unexpected 'mac' key in result")
	}
}

func TestGenerateElectronBuilderJSON_MacWithNotarization(t *testing.T) {
	cfg := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Notarize: true,
			TeamID:   "T123",
		},
	}
	opts := DefaultOptions()
	got, err := GenerateElectronBuilderJSON(cfg, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["mac"] == nil {
		t.Error("expected 'mac' key")
	}
	if got["afterSign"] != opts.NotarizeScriptPath {
		t.Errorf("afterSign = %v, want %q", got["afterSign"], opts.NotarizeScriptPath)
	}
}

func TestGenerateElectronBuilderJSON_DefaultOpts(t *testing.T) {
	cfg := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{},
	}
	// Pass nil opts to trigger DefaultOptions()
	got, err := GenerateElectronBuilderJSON(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result")
	}
}

// --- entitlements tests ---

func TestGenerateEntitlementsPlist_DefaultElectron(t *testing.T) {
	cfg := &types.MacOSSigningConfig{HardenedRuntime: true}
	got, err := generateEntitlementsPlist(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got)
	// Should contain default Electron entitlements
	for _, ent := range DefaultElectronEntitlements {
		if !strings.Contains(content, string(ent)) {
			t.Errorf("missing default entitlement: %s", ent)
		}
	}
	// Should be valid plist structure
	if !strings.Contains(content, "<?xml version=") {
		t.Error("missing XML declaration")
	}
	if !strings.Contains(content, "<plist version=") {
		t.Error("missing plist declaration")
	}
}

func TestGenerateEntitlementsPlist_Capabilities(t *testing.T) {
	cfg := &types.MacOSSigningConfig{}

	tests := []struct {
		capability string
		expected   []EntitlementKey
	}{
		{"network", []EntitlementKey{EntitlementNetworkClient}},
		{"network-client", []EntitlementKey{EntitlementNetworkClient}},
		{"network-server", []EntitlementKey{EntitlementNetworkServer}},
		{"audio", []EntitlementKey{EntitlementDeviceAudio}},
		{"microphone", []EntitlementKey{EntitlementDeviceAudio}},
		{"camera", []EntitlementKey{EntitlementDeviceCamera}},
		{"bluetooth", []EntitlementKey{EntitlementDeviceBluetooth}},
		{"usb", []EntitlementKey{EntitlementDeviceUSB}},
		{"location", []EntitlementKey{EntitlementPersonalInformationLocation}},
		{"files", []EntitlementKey{EntitlementFilesUserSelected, EntitlementFilesDownloads}},
		{"filesystem", []EntitlementKey{EntitlementFilesUserSelected, EntitlementFilesDownloads}},
		{"debugger", []EntitlementKey{EntitlementDebugger}},
		{"inherit", []EntitlementKey{EntitlementInheritSecurityScope}},
	}

	for _, tc := range tests {
		t.Run(tc.capability, func(t *testing.T) {
			got, err := generateEntitlementsPlist(cfg, []string{tc.capability})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			content := string(got)
			for _, ent := range tc.expected {
				if !strings.Contains(content, string(ent)) {
					t.Errorf("capability %q should include %s", tc.capability, ent)
				}
			}
		})
	}
}

func TestGenerateEntitlementsPlist_CaseInsensitive(t *testing.T) {
	cfg := &types.MacOSSigningConfig{}
	got, err := generateEntitlementsPlist(cfg, []string{"CAMERA", "Network"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(got)
	if !strings.Contains(content, string(EntitlementDeviceCamera)) {
		t.Error("CAMERA (uppercase) should map to camera entitlement")
	}
	if !strings.Contains(content, string(EntitlementNetworkClient)) {
		t.Error("Network (mixed case) should map to network entitlement")
	}
}

func TestGenerateEntitlementsPlist_SortedOutput(t *testing.T) {
	cfg := &types.MacOSSigningConfig{}
	got, err := generateEntitlementsPlist(cfg, []string{"usb", "camera", "network"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := string(got)

	// Keys should appear in sorted order (alphabetical by full entitlement key)
	cameraIdx := strings.Index(content, string(EntitlementDeviceCamera))
	usbIdx := strings.Index(content, string(EntitlementDeviceUSB))

	if cameraIdx == -1 || usbIdx == -1 {
		t.Fatal("not all entitlements present")
	}
	// com.apple.security.device.camera < com.apple.security.device.usb
	if cameraIdx > usbIdx {
		t.Error("camera should sort before usb")
	}
}

func TestGenerateEntitlementsPlist_UnknownCapabilityIgnored(t *testing.T) {
	cfg := &types.MacOSSigningConfig{}
	got, err := generateEntitlementsPlist(cfg, []string{"nonexistent-cap"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still produce valid plist with just defaults
	if !strings.Contains(string(got), string(EntitlementAllowJIT)) {
		t.Error("should still contain default entitlements")
	}
}

// --- ParseCapabilities tests ---

func TestParseCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "network", []string{"network"}},
		{"multiple", "network,camera,usb", []string{"network", "camera", "usb"}},
		{"with spaces", " network , camera , usb ", []string{"network", "camera", "usb"}},
		{"trailing comma", "network,camera,", []string{"network", "camera"}},
		{"empty items filtered", ",,network,,camera,,", []string{"network", "camera"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseCapabilities(tc.input)
			if tc.expected == nil {
				if got != nil {
					t.Errorf("ParseCapabilities(%q) = %v, want nil", tc.input, got)
				}
				return
			}
			if len(got) != len(tc.expected) {
				t.Fatalf("len = %d, want %d; got %v", len(got), len(tc.expected), got)
			}
			for i, v := range got {
				if v != tc.expected[i] {
					t.Errorf("[%d] = %q, want %q", i, v, tc.expected[i])
				}
			}
		})
	}
}

// --- notarize script tests ---

func TestGenerateNotarizeJS_Nil(t *testing.T) {
	got, err := generateNotarizeJS(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestGenerateNotarizeJS_NotarizeDisabled(t *testing.T) {
	cfg := &types.MacOSSigningConfig{Notarize: false}
	got, err := generateNotarizeJS(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil when notarize=false")
	}
}

func TestGenerateNotarizeJS_APIKeyMethod(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		Notarize:         true,
		AppleAPIKeyID:    "KEY123",
		AppleAPIIssuerID: "ISSUER456",
		TeamID:           "TEAM789",
	}
	got, err := generateNotarizeJS(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got)
	if !strings.Contains(content, "KEY123") {
		t.Error("should contain API key ID")
	}
	if !strings.Contains(content, "ISSUER456") {
		t.Error("should contain issuer ID")
	}
	if !strings.Contains(content, "TEAM789") {
		t.Error("should contain team ID")
	}
	if !strings.Contains(content, "APPLE_API_KEY_ID") {
		t.Error("should reference APPLE_API_KEY_ID env var")
	}
	if !strings.Contains(content, "notarize") {
		t.Error("should contain notarize import")
	}
}

func TestGenerateNotarizeJS_AppPasswordMethod(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		Notarize: true,
		TeamID:   "TEAM789",
		// No API key set → falls back to app-password method
	}
	got, err := generateNotarizeJS(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got)
	if !strings.Contains(content, "APPLE_ID") {
		t.Error("should reference APPLE_ID env var")
	}
	if !strings.Contains(content, "APPLE_ID_PASSWORD") {
		t.Error("should reference APPLE_ID_PASSWORD env var")
	}
}

func TestGenerateNotarizeJS_CustomEnvVars(t *testing.T) {
	cfg := &types.MacOSSigningConfig{
		Notarize:           true,
		TeamID:             "TEAM",
		AppleIDEnv:         "MY_APPLE_ID",
		AppleIDPasswordEnv: "MY_APPLE_PASS",
	}
	got, err := generateNotarizeJS(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got)
	if !strings.Contains(content, "MY_APPLE_ID") {
		t.Error("should use custom Apple ID env var")
	}
	if !strings.Contains(content, "MY_APPLE_PASS") {
		t.Error("should use custom Apple ID password env var")
	}
}

// --- NotarizeCredentialMethod tests ---

func TestNotarizeCredentialMethod(t *testing.T) {
	tests := []struct {
		name     string
		config   *types.MacOSSigningConfig
		expected string
	}{
		{"nil", nil, ""},
		{"notarize disabled", &types.MacOSSigningConfig{Notarize: false}, ""},
		{
			"api key method",
			&types.MacOSSigningConfig{Notarize: true, AppleAPIKeyID: "KEY"},
			"api_key",
		},
		{
			"app password via AppleIDEnv",
			&types.MacOSSigningConfig{Notarize: true, AppleIDEnv: "MY_ID"},
			"app_password",
		},
		{
			"app password via AppleIDPasswordEnv",
			&types.MacOSSigningConfig{Notarize: true, AppleIDPasswordEnv: "MY_PASS"},
			"app_password",
		},
		{
			"notarize enabled but no creds",
			&types.MacOSSigningConfig{Notarize: true},
			"",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NotarizeCredentialMethod(tc.config)
			if got != tc.expected {
				t.Errorf("NotarizeCredentialMethod() = %q, want %q", got, tc.expected)
			}
		})
	}
}

// --- Generator interface tests ---

func TestNewGenerator_DefaultOpts(t *testing.T) {
	g := NewGenerator(nil)
	if g == nil {
		t.Fatal("NewGenerator(nil) returned nil")
	}
}

func TestDefaultGenerator_GenerateElectronBuilder_Nil(t *testing.T) {
	g := NewGenerator(nil)
	got, err := g.GenerateElectronBuilder(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestDefaultGenerator_GenerateElectronBuilder_Disabled(t *testing.T) {
	g := NewGenerator(nil)
	got, err := g.GenerateElectronBuilder(&types.SigningConfig{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for disabled config")
	}
}

func TestDefaultGenerator_GenerateElectronBuilder_Full(t *testing.T) {
	g := NewGenerator(nil)
	cfg := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{CertificateSource: types.CertSourceFile},
		MacOS:   &types.MacOSSigningConfig{Identity: "Test"},
	}
	got, err := g.GenerateElectronBuilder(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Win == nil {
		t.Error("expected Win config")
	}
	if got.Mac == nil {
		t.Error("expected Mac config")
	}
}

func TestDefaultGenerator_GenerateEntitlements_Nil(t *testing.T) {
	g := NewGenerator(nil)
	got, err := g.GenerateEntitlements(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestDefaultGenerator_GenerateNotarizeScript_Nil(t *testing.T) {
	g := NewGenerator(nil)
	got, err := g.GenerateNotarizeScript(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestDefaultGenerator_GenerateNotarizeScript_Disabled(t *testing.T) {
	g := NewGenerator(nil)
	got, err := g.GenerateNotarizeScript(&types.MacOSSigningConfig{Notarize: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil when notarize disabled")
	}
}

func TestDefaultGenerator_GenerateAll_Nil(t *testing.T) {
	g := NewGenerator(nil)
	got, err := g.GenerateAll(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil for nil config")
	}
}

func TestDefaultGenerator_GenerateAll_MacOS(t *testing.T) {
	opts := &Options{
		OutputDir:          "build",
		EntitlementsPath:   "entitlements.plist",
		NotarizeScriptPath: "scripts/notarize.js",
	}
	g := NewGenerator(opts)
	cfg := &types.SigningConfig{
		Enabled: true,
		MacOS: &types.MacOSSigningConfig{
			Notarize:      true,
			TeamID:        "TEAM",
			AppleAPIKeyID: "KEY",
		},
	}
	got, err := g.GenerateAll(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have entitlements file
	entPath := "build/entitlements.plist"
	if got[entPath] == nil {
		t.Errorf("missing entitlements at %q", entPath)
	}

	// Should have notarize script
	if got["scripts/notarize.js"] == nil {
		t.Error("missing notarize script")
	}
}

func TestDefaultGenerator_GenerateAll_NoMac(t *testing.T) {
	g := NewGenerator(nil)
	cfg := &types.SigningConfig{
		Enabled: true,
		Windows: &types.WindowsSigningConfig{},
		// No MacOS config
	}
	got, err := g.GenerateAll(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map for Windows-only, got %d files", len(got))
	}
}

// --- DefaultOptions tests ---

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.OutputDir != "build" {
		t.Errorf("OutputDir = %q, want 'build'", opts.OutputDir)
	}
	if opts.EntitlementsPath != "entitlements.mac.plist" {
		t.Errorf("EntitlementsPath = %q, want 'entitlements.mac.plist'", opts.EntitlementsPath)
	}
	if opts.NotarizeScriptPath != "scripts/notarize.js" {
		t.Errorf("NotarizeScriptPath = %q, want 'scripts/notarize.js'", opts.NotarizeScriptPath)
	}
}
