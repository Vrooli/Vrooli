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
