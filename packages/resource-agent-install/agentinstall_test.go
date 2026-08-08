package agentinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBridgeExposesStatusAndInstallPolicy(t *testing.T) {
	if got := ParseStatusArgs([]string{"--json", "--fast"}); got.Format != "json" || !got.Fast {
		t.Fatalf("unexpected status args: %#v", got)
	}
	root := t.TempDir()
	managed := filepath.Join(root, "bin")
	path := filepath.Join(root, "system", "agent")
	blocked, err := BlockingSystemInstall("agent", managed, func(string) (string, error) { return path, nil }, func(string) (os.FileInfo, error) {
		return bridgeFileInfo{}, nil
	})
	if err != nil || blocked != path {
		t.Fatalf("blocked=%q err=%v", blocked, err)
	}
	if !strings.Contains(WarnIfShadowed("agent", managed, func(string) (string, error) { return path, nil }), "precedes") {
		t.Fatal("expected shadow warning")
	}
}

type bridgeFileInfo struct{}

func (bridgeFileInfo) Name() string       { return "system" }
func (bridgeFileInfo) Size() int64        { return 0 }
func (bridgeFileInfo) Mode() os.FileMode  { return 0o555 }
func (bridgeFileInfo) ModTime() time.Time { return time.Time{} }
func (bridgeFileInfo) IsDir() bool        { return true }
func (bridgeFileInfo) Sys() any           { return nil }
