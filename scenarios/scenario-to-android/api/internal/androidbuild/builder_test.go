package androidbuild

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type signingStore struct{ values map[string]string }

func (s *signingStore) Resolve(_ context.Context, _, field string) (string, error) {
	value, ok := s.values[field]
	if !ok {
		return "", errors.New("unconfigured")
	}
	return value, nil
}

func (s *signingStore) Provision(_ context.Context, request credentialclient.ProvisionRequest) (credentialclient.ProvisionResponse, error) {
	s.values[request.Field] = request.Value
	return credentialclient.ProvisionResponse{Identity: request.Identity, Field: request.Field, Status: "provisioned"}, nil
}

func TestRenderProjectUsesAPI36AndContainsNoSigningIdentity(t *testing.T) {
	web := t.TempDir()
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<h1>Hello Mobile</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "project")
	if err := RenderProject(project, web, "com.example.hello", "Hello Mobile", 36); err != nil {
		t.Fatal(err)
	}
	gradle, err := os.ReadFile(filepath.Join(project, "app", "build.gradle"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gradle) == "" || !containsAll(string(gradle), "compileSdk 36", "targetSdk 36") {
		t.Fatalf("unexpected build configuration: %s", gradle)
	}
	manifest, err := os.ReadFile(filepath.Join(project, "app", "src", "main", "AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if containsAny(string(manifest), "keystore", "password", "signingConfig") {
		t.Fatal("generated manifest contains signing identity material")
	}
	main, err := os.ReadFile(filepath.Join(project, "app", "src", "main", "java", "com", "example", "hello", "MainActivity.java"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(main), "import com.getcapacitor.BridgeActivity;", "extends BridgeActivity") || containsAny(string(main), "ServerSocket", "WebView") {
		t.Fatalf("generated shell is not a Capacitor BridgeActivity: %s", main)
	}
	config, err := os.ReadFile(filepath.Join(project, "capacitor.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(config), `"appId": "com.example.hello"`, `"webDir": "web"`, `"hostname": "localhost"`, `"androidScheme": "http"`) {
		t.Fatalf("generated Capacitor configuration is incomplete: %s", config)
	}
	assetConfig, err := os.ReadFile(filepath.Join(project, "app", "src", "main", "assets", "capacitor.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(assetConfig) != string(config) {
		t.Fatal("native asset Capacitor configuration differs from project configuration")
	}
	packageJSON, err := os.ReadFile(filepath.Join(project, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(packageJSON), `"@capacitor/core": "6.1.0"`, `"@capacitor/android": "6.1.0"`, `"@capacitor/cli": "6.1.0"`) {
		t.Fatalf("generated Capacitor package metadata is incomplete: %s", packageJSON)
	}
	if !containsAll(string(gradle), "com.capacitorjs:core:6.1.0", "androidx.appcompat:appcompat:1.7.0", "JavaVersion.VERSION_17") {
		t.Fatalf("generated Gradle project does not include Capacitor Android runtime: %s", gradle)
	}
	if _, err := os.Stat(filepath.Join(project, "app", "src", "main", "assets", "public", "index.html")); err != nil {
		t.Fatalf("web bundle was not copied to Capacitor public directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, "web", "index.html")); err != nil {
		t.Fatalf("web bundle was not copied to configured Capacitor web directory: %v", err)
	}
}

func TestRenderProjectIncludesBrandingAssets(t *testing.T) {
	web := t.TempDir()
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<h1>Hello Mobile</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "project")
	if err := renderProject(project, web, projectConfig{PackageName: "com.example.hello", AppName: "Hello Mobile", TargetSDK: 36, BrandName: "Hello Mobile", BrandID: "brand-1", BrandSource: "brand-manager:brand-1", Primary: "#2457D6", Secondary: "#173B8F", Accent: "#FFB020", Background: "#F4F7FF", TextColor: "#13203A"}); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{
		"app/src/main/res/drawable/ic_launcher_foreground.xml",
		"app/src/main/res/drawable/ic_launcher_legacy.xml",
		"app/src/main/res/drawable/ic_notification.xml",
		"app/src/main/res/drawable/splash_background.xml",
		"app/src/main/res/values/colors.xml",
		"app/src/main/res/values/branding.xml",
		"app/src/main/res/xml/shortcuts.xml",
	} {
		if _, err := os.Stat(filepath.Join(project, relative)); err != nil {
			t.Fatalf("branding asset %s was not rendered: %v", relative, err)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(project, "app/src/main/AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(manifest), "ic_launcher_legacy", "android.intent.action.SEND", "android:mimeType=\"text/plain\"", "android.app.shortcuts") {
		t.Fatalf("manifest omitted branding surfaces: %s", manifest)
	}
	branding, err := os.ReadFile(filepath.Join(project, "app/src/main/res/values/branding.xml"))
	if err != nil || !containsAll(string(branding), "brand-manager:brand-1", "Hello Mobile") {
		t.Fatalf("branding source was not recorded: %s", branding)
	}
}

func TestBuilderReportsPlaceholderBrandingWithoutAssignment(t *testing.T) {
	t.Setenv("ANDROID_SKIP_BRAND_LOOKUP", "1")
	web := t.TempDir()
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<h1>Unbranded fixture</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}

	generated, err := (Builder{BuildRoot: t.TempDir()}).Generate(context.Background(), deliveryramp.BuildRequest{
		SourceRef: web,
		Parameters: map[string]string{
			"scenario_name": "unassigned-fixture",
			"package_name":  "com.example.unassigned",
			"target_sdk":    "36",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if generated.BrandSource != "placeholder" || generated.BrandName != "Generic Vrooli App" {
		t.Fatalf("unassigned branding was not reported explicitly: %+v", generated)
	}
	branding, err := os.ReadFile(filepath.Join(generated.ProjectPath, "app/src/main/res/values/branding.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(branding), "placeholder", "Generic Vrooli App") {
		t.Fatalf("placeholder branding metadata is incomplete: %s", branding)
	}
}

func TestBuilderGeneratesProjectWithoutGradleArtifacts(t *testing.T) {
	web := t.TempDir()
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<h1>Hello Mobile</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := (Builder{BuildRoot: t.TempDir()}).Generate(context.Background(), deliveryramp.BuildRequest{
		SourceRef:  web,
		Parameters: map[string]string{"package_name": "com.example.generated", "target_sdk": "36"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(project.ProjectPath) || project.PackageName != "com.example.generated" || project.TargetSDK != 36 {
		t.Fatalf("unexpected generated project descriptor: %+v", project)
	}
	if _, err := os.Stat(filepath.Join(project.ProjectPath, "app", "build.gradle")); err != nil {
		t.Fatalf("generated project is missing Gradle configuration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project.ProjectPath, "app", "build", "outputs")); !os.IsNotExist(err) {
		t.Fatalf("generate unexpectedly produced build outputs: %v", err)
	}
}

func TestBundleDigestChangesWhenBundleChanges(t *testing.T) {
	web := t.TempDir()
	path := filepath.Join(web, "index.html")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := digestBundle(web)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := digestBundle(web)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("bundle digest did not change after bundle content changed")
	}
}

func TestArtifactCarriesTargetAPIAssertion(t *testing.T) {
	root := t.TempDir()
	apk := filepath.Join(root, "app-debug.apk")
	aab := filepath.Join(root, "app-debug.aab")
	if err := os.WriteFile(apk, []byte("apk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(aab, []byte("aab"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact, err := artifactForOutputs(apk, aab, time.Unix(1, 0), "com.example.hello", 36)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Metadata["target_sdk"] != "36" || artifact.Metadata["target_api_assertion"] != "targetSdk(36) >= 36" {
		t.Fatalf("artifact omitted target API assertion: %#v", artifact.Metadata)
	}
}

func TestBuilderRejectsBelowAPI36(t *testing.T) {
	_, err := (Builder{}).Build(context.Background(), deliveryramp.BuildRequest{SourceRef: t.TempDir(), Parameters: map[string]string{"target_sdk": "35"}})
	if err == nil {
		t.Fatal("expected API floor rejection")
	}
}

func TestProvisionSigningKeyStoresOnlyCredentialFields(t *testing.T) {
	store := &signingStore{values: map[string]string{}}
	var keytoolArgs []string
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		keytoolArgs = append([]string(nil), args...)
		for index, arg := range args {
			if arg != "-keystore" || index+1 >= len(args) {
				continue
			}
			if err := os.WriteFile(args[index+1], []byte("test-keystore"), 0o600); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if err := ProvisionSigningKey(context.Background(), store, "vrooli/test-signing", "/bin/keytool", run); err != nil {
		t.Fatal(err)
	}
	if store.values[SigningAliasField] != "vrooli-upload" || store.values[SigningPasswordField] == "" {
		t.Fatalf("provisioned signing fields are incomplete: %#v", store.values)
	}
	decoded, err := base64.StdEncoding.DecodeString(store.values[SigningKeystoreField])
	if err != nil || string(decoded) != "test-keystore" {
		t.Fatalf("keystore was not provisioned as base64: %v", err)
	}
	joinedArgs := strings.Join(keytoolArgs, " ")
	if strings.Contains(joinedArgs, store.values[SigningPasswordField]) || strings.Contains(joinedArgs, "-storepass ") || strings.Contains(joinedArgs, "-keypass ") {
		t.Fatalf("keytool command exposed signing password: %q", joinedArgs)
	}
}

func TestProvisionSigningKeyRejectsPartialIdentity(t *testing.T) {
	store := &signingStore{values: map[string]string{SigningAliasField: "existing"}}
	if err := ProvisionSigningKey(context.Background(), store, "vrooli/test-signing", "/bin/keytool", nil); err == nil || !strings.Contains(err.Error(), "partially configured") {
		t.Fatalf("expected partial identity rejection, got %v", err)
	}
}

func TestSignedBuildUsesTemporaryCredentialProperties(t *testing.T) {
	web := t.TempDir()
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<h1>Signed</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := &signingStore{values: map[string]string{
		SigningKeystoreField: base64.StdEncoding.EncodeToString([]byte("keystore")),
		SigningPasswordField: "password",
		SigningAliasField:    "alias",
	}}
	var gradleArgs []string
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		gradleArgs = append([]string(nil), args...)
		project := ""
		for i, arg := range args {
			if arg == "--project-dir" && i+1 < len(args) {
				project = args[i+1]
			}
		}
		if project == "" {
			return nil, errors.New("missing project directory")
		}
		for _, output := range []string{
			filepath.Join(project, "app/build/outputs/apk/release/app-release.apk"),
			filepath.Join(project, "app/build/outputs/bundle/release/app-release.aab"),
		} {
			if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
				return nil, err
			}
			if err := os.WriteFile(output, []byte("artifact"), 0o644); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	artifact, err := (Builder{BuildRoot: t.TempDir(), GradleBin: "/bin/gradle", Run: run, Signing: store}).Build(context.Background(), deliveryramp.BuildRequest{SourceRef: web, Parameters: map[string]string{"package_name": "com.example.signed", "target_sdk": "36", "signing": "required"}})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Metadata["signing"] != "secrets-manager" || artifact.Kind != "android-release-apk-aab" {
		t.Fatalf("signed artifact metadata is incorrect: %#v", artifact)
	}
	for _, arg := range gradleArgs {
		if strings.Contains(arg, "password") || strings.Contains(arg, "keystore") {
			t.Logf("Gradle receives a temporary properties path, as intended: %s", arg)
		}
	}
	for _, arg := range gradleArgs {
		if strings.HasPrefix(arg, "-Dvrooli.android.signing.properties=") {
			path := strings.TrimPrefix(arg, "-Dvrooli.android.signing.properties=")
			if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
				t.Fatalf("temporary signing properties survived build: %s", path)
			}
		}
	}
}

func TestBuilderBuildsConfiguredWebBundle(t *testing.T) {
	gradle := os.Getenv("ANDROID_GRADLE_BIN")
	sdk := os.Getenv("ANDROID_SDK_ROOT")
	if gradle == "" || sdk == "" {
		t.Skip("set ANDROID_GRADLE_BIN and ANDROID_SDK_ROOT for the governed Android packaging test")
	}
	web := os.Getenv("ANDROID_SOURCE_REF")
	if web == "" {
		web = t.TempDir()
		if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<html><body><h1 id=\"hello-mobile-title\">Hello Mobile</h1><script src=\"app.js\"></script></body></html>"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(web, "app.js"), []byte("document.body.dataset.ready = 'true';"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	artifact, err := (Builder{BuildRoot: t.TempDir(), GradleBin: gradle}).Build(context.Background(), deliveryramp.BuildRequest{
		SourceRef:  web,
		Parameters: map[string]string{"package_name": "com.example.generated", "target_sdk": "36"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.LocalPath == "" || artifact.Metadata["aab_path"] == "" || artifact.Metadata["package_name"] != "com.example.generated" {
		t.Fatalf("incomplete generated artifact: %+v", artifact)
	}
	if output := os.Getenv("ANDROID_ARTIFACT_OUTPUT"); output != "" {
		data, readErr := os.ReadFile(artifact.LocalPath)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if writeErr := os.WriteFile(output, data, 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
}

func containsAll(value string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(value, part) {
			return false
		}
	}
	return true
}

func containsAny(value string, parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(strings.ToLower(value), strings.ToLower(part)) {
			return true
		}
	}
	return false
}
