package agentinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveURLRejectsUnsupportedPlatformBeforeDownload(t *testing.T) {
	_, err := ResolveURL(Spec{Binary: "k6", URLTemplates: map[string]string{"linux": "https://example.invalid/${arch}"}}, "windows", "arm64")
	var unsupported UnsupportedPlatformError
	if !errors.As(err, &unsupported) || unsupported.OS != "windows" || unsupported.Arch != "arm64" {
		t.Fatalf("error=%v, want typed unsupported-platform error", err)
	}
}

func TestResolveURLExpandsDeclaredTarget(t *testing.T) {
	got, err := ResolveURL(Spec{Version: "v1.2.3", URLTemplates: map[string]string{"darwin": "https://example.invalid/${platform}/${arch}/v${version}"}}, "darwin", "arm64")
	if err != nil || got != "https://example.invalid/macos/arm64/v1.2.3" {
		t.Fatalf("url=%q err=%v", got, err)
	}
}

func TestParseStatusArgs(t *testing.T) {
	got := ParseStatusArgs([]string{"--json", "--verbose", "--fast", "--format", "text"})
	if got.Format != "text" || !got.Verbose || !got.Fast {
		t.Fatalf("unexpected args: %#v", got)
	}
}

func TestBlockingSystemInstallAndShadowWarning(t *testing.T) {
	root := t.TempDir()
	managed := filepath.Join(root, "bin")
	if err := os.MkdirAll(managed, 0o755); err != nil {
		t.Fatal(err)
	}
	system := filepath.Join(root, "system", "agent")
	blocked, err := BlockingSystemInstall("agent", managed, func(string) (string, error) { return system, nil }, func(string) (os.FileInfo, error) {
		return fakeFileInfo{mode: 0o555}, nil
	})
	if err != nil || blocked != system {
		t.Fatalf("blocked=%q err=%v", blocked, err)
	}
	warning := WarnIfShadowed("agent", managed, func(string) (string, error) { return system, nil })
	if !strings.Contains(warning, "precedes") {
		t.Fatalf("warning=%q", warning)
	}
}

type fakeFileInfo struct{ mode os.FileMode }

func (f fakeFileInfo) Name() string       { return "system" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return true }
func (f fakeFileInfo) Sys() any           { return nil }
