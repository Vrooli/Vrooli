package androidbuild

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

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
