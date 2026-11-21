//go:build linux

package target

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestJ11_UsageSliceDrivesUsageBars(t *testing.T) {
	source := os.Getenv("J11_USAGE_RELEASE_ROOT")
	if source == "" {
		t.Skip("J11_USAGE_RELEASE_ROOT selects the real signed Agent Manager usage slice")
	}
	root, _ := startJ11ReleaseFrom(t, source)
	artifact := buildJ11Host(t, filepath.Join(root, "host-input.json"))
	home := t.TempDir()
	installed := filepath.Join(home, "Aquila.AppImage")
	digest := copyJ10Artifact(t, artifact, installed)
	session := startJ11Desktop(t, installed, home, digest)
	runUsageComponentBrowser(t, session, "install")
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Reuse a genuine installed harness in this isolated home. No credentials
	// are copied and no agent/model is launched; the real version probe makes
	// its card eligible, just as an existing user installation would be.
	claude := os.Getenv("J11_CLAUDE_BINARY")
	if claude == "" {
		t.Fatal("J11_CLAUDE_BINARY must identify a real native Claude executable")
	}
	bin := filepath.Join(home, ".local/bin/claude")
	if err := os.MkdirAll(filepath.Dir(bin), 0700); err != nil {
		t.Fatal(err)
	}
	t.Logf("real native Claude executable sha256=%s", copyJ10Artifact(t, claude, bin))
	// Seed the resource's documented durable feed, never a replacement usage
	// server. On restart the installed owner executable parses these facts.
	path := filepath.Join(home, ".vrooli", "state", "vrooli", "claude-code", "quota-samples.ndjson")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	sample := fmt.Sprintf(`{"observed_at":%q,"session_id":"j11-local","five_hour":{"used_percentage":23,"resets_at":%d},"seven_day":{"used_percentage":11,"resets_at":%d}}`+"\n", now.Format(time.RFC3339Nano), now.Add(time.Hour).Unix(), now.Add(24*time.Hour).Unix())
	if err := os.WriteFile(path, []byte(sample), 0600); err != nil {
		t.Fatal(err)
	}
	session = startJ11Desktop(t, installed, home, digest)
	if code, _ := j11RuntimeRead(t, home, "/scenario-port?scenario=agent-manager&port_name=API_PORT"); code != http.StatusNotFound {
		t.Fatalf("bare scenario lookup selected optional usage slice: HTTP%d", code)
	}
	state := runUsageComponentBrowser(t, session, "usage")
	if code, body := j11RuntimeRead(t, home, "/features?scenario=agent-manager&feature=usage.live"); code != http.StatusOK || body["source"] != "component" || body["component_id"] != "agent-manager-usage" {
		t.Fatalf("real usage did not resolve to installed owner slice: HTTP%d %+v", code, body)
	}
	t.Logf("real installed Agent Manager statusline parser → live quota API → Web Console usage bars: %s", state)
}

func runUsageComponentBrowser(t *testing.T, session *ElectronSession, mode string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "python3", "desktop_usage_component_journey.py", session.endpoint, mode).CombinedOutput()
	if err != nil {
		t.Fatalf("real usage component UI %s: %v\n%s", mode, err, out)
	}
	var state map[string]any
	if err := json.Unmarshal(out, &state); err != nil {
		t.Fatalf("usage component UI output: %v\n%s", err, out)
	}
	return string(out)
}
