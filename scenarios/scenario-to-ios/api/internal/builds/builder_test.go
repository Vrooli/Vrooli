package builds

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestGenerateDeterministicCapacitorProject(t *testing.T) {
	web := t.TempDir()
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<main>hello</main>"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := Builder{BuildRoot: t.TempDir(), GOOS: "linux"}
	a, err := b.Generate(context.Background(), request(web))
	if err != nil {
		t.Fatal(err)
	}
	c, err := b.Generate(context.Background(), request(web))
	if err != nil {
		t.Fatal(err)
	}
	if a.ProjectPath != c.ProjectPath {
		t.Fatalf("project paths differ: %q %q", a.ProjectPath, c.ProjectPath)
	}
	for _, path := range []string{"capacitor.config.json", "ios/Podfile", "ios/App/Info.plist", "ios/App/App.entitlements", "ios/App.xcodeproj/project.pbxproj", "public/index.html"} {
		if _, err := os.Stat(filepath.Join(a.ProjectPath, path)); err != nil {
			t.Fatalf("missing generated %s: %v", path, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(a.ProjectPath, "ios/App.xcodeproj/project.pbxproj"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "IPHONEOS_DEPLOYMENT_TARGET = 26.0") {
		t.Fatalf("SDK floor missing: %s", data)
	}
}

func TestBuildOnLinuxIsHonestUnavailable(t *testing.T) {
	web := t.TempDir()
	_ = os.WriteFile(filepath.Join(web, "index.html"), []byte("ok"), 0o644)
	_, err := (Builder{BuildRoot: t.TempDir(), GOOS: "linux"}).Build(context.Background(), request(web))
	if err == nil || !strings.Contains(err.Error(), "macOS xcodebuild host") {
		t.Fatalf("err = %v", err)
	}
}

func request(web string) deliveryramp.BuildRequest { return deliveryramp.BuildRequest{SourceRef: web} }
