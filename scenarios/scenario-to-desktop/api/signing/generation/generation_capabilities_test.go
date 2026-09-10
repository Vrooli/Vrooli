package generation

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"scenario-to-desktop-api/signing/types"
)

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
		AppleAPIKeyFile:  "/tmp/AuthKey_FIXTURE.p8",
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
	if !strings.Contains(content, "APPLE_API_KEY_FILE") {
		t.Error("should reference the declared private-key file environment variable")
	}
	if !strings.Contains(content, "APPLE_API_ISSUER_ID") {
		t.Error("should reference the canonical issuer ID environment variable")
	}
	if strings.Contains(content, "appleApiKey: process.env.APPLE_API_KEY_ID") {
		t.Error("must not pass the key ID as the private-key file path")
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
	if !strings.Contains(content, "APPLE_APP_SPECIFIC_PASSWORD") {
		t.Error("should reference the configured app-password env var")
	}
}

func executeGeneratedNotarize(t *testing.T, config *types.MacOSSigningConfig, environment map[string]string) map[string]any {
	t.Helper()
	if environment == nil {
		environment = map[string]string{}
	}
	script, err := generateNotarizeJS(config)
	if err != nil {
		t.Fatal(err)
	}
	envJSON, err := json.Marshal(environment)
	if err != nil {
		t.Fatal(err)
	}
	// Execute the generated artifact against an independent mock of the
	// library boundary. This proves the options passed to @electron/notarize,
	// rather than only proving that template text contains familiar names.
	runner := `const fs=require('fs'),vm=require('vm'),hostRequire=require;let observed;const box={exports:{},require:(name)=>name==='@electron/notarize'?{notarize:async(options)=>{observed=options}}:hostRequire(name),process:{env:` + string(envJSON) + `},console:{log:()=>{},error:()=>{}}};vm.runInNewContext(fs.readFileSync(0,'utf8'),box);box.exports.default({electronPlatformName:'darwin',appOutDir:'/tmp/audit',packager:{appInfo:{productFilename:'Fixture'}}}).then(()=>process.stdout.write(JSON.stringify(observed))).catch(()=>process.exit(2));`
	cmd := exec.Command("node", "-e", runner)
	cmd.Stdin = bytes.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execute generated notarization hook: %v\n%s", err, output)
	}
	var observed map[string]any
	if err := json.Unmarshal(output, &observed); err != nil {
		t.Fatalf("decode notarization options %q: %v", output, err)
	}
	return observed
}

func TestGeneratedNotarizeJSUsesPrivateKeyFileAndKeyIDSeparately(t *testing.T) {
	got := executeGeneratedNotarize(t, &types.MacOSSigningConfig{
		Notarize:         true,
		AppleAPIKeyID:    "FIXTURE123",
		AppleAPIKeyFile:  "/audit/AuthKey_FIXTURE.p8",
		AppleAPIIssuerID: "fixture-issuer",
		TeamID:           "FIXTURETEAM",
	}, nil)
	if got["appleApiKey"] != "/audit/AuthKey_FIXTURE.p8" || got["appleApiKeyId"] != "FIXTURE123" || got["appleApiIssuer"] != "fixture-issuer" {
		t.Fatalf("notarization options = %#v, want separate path, key ID, and issuer", got)
	}
}

func TestGeneratedNotarizeJSUsesDeclaredAppPasswordBinding(t *testing.T) {
	got := executeGeneratedNotarize(t, &types.MacOSSigningConfig{Notarize: true, TeamID: "FIXTURETEAM"}, map[string]string{
		"APPLE_ID":                    "fixture@example.test",
		"APPLE_APP_SPECIFIC_PASSWORD": "synthetic-fixture",
	})
	if got["appleId"] != "fixture@example.test" || got["appleIdPassword"] != "synthetic-fixture" {
		t.Fatalf("notarization options = %#v, want declared Apple ID and app-password bindings", got)
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
			&types.MacOSSigningConfig{Notarize: true, AppleAPIKeyID: "KEY", AppleAPIKeyFile: "key.p8", AppleAPIIssuerID: "ISSUER"},
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
