package desktopcompanion

import (
	"strings"
	"testing"
)

func TestRenderLaunchAgentIsUserScopedAndCredentialFree(t *testing.T) {
	spec := Spec{Label: "com.vrooli.device-control.desktop", Program: "/Users/alice/.vrooli/bin/device-control-companion", Config: "/Users/alice/Library/Application Support/Vrooli/desktop.json", UserHome: "/Users/alice"}
	plist, err := RenderLaunchAgent(spec)
	if err != nil {
		t.Fatal(err)
	}
	text := string(plist)
	for _, forbidden := range []string{"sudo", "ssh", "Authorization", "private_key", "BRIDGE_CONTROL_PLANE_URL"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("plist contains forbidden %q", forbidden)
		}
	}
	if !strings.Contains(text, "com.vrooli.device-control.desktop") || !strings.Contains(text, "RunAtLoad") {
		t.Fatalf("incomplete plist: %s", text)
	}
	path, err := Path(spec)
	if err != nil || !strings.HasSuffix(path, "Library/LaunchAgents/com.vrooli.device-control.desktop.plist") {
		t.Fatalf("path=%q err=%v", path, err)
	}
}

func TestLaunchAgentRejectsPathEscapingLabelsAndRootHomes(t *testing.T) {
	base := Spec{Label: "com.vrooli.device-control.desktop", Program: "/Users/alice/.vrooli/bin/device-control-companion", Config: "/Users/alice/Library/Application Support/Vrooli/desktop.json", UserHome: "/Users/alice"}
	for _, label := range []string{"../escape", "/tmp/escape", "", ".."} {
		spec := base
		spec.Label = label
		if _, err := RenderLaunchAgent(spec); err != ErrInvalidSpec {
			t.Fatalf("label %q render error=%v", label, err)
		}
		if _, err := Path(spec); err != ErrInvalidSpec {
			t.Fatalf("label %q path error=%v", label, err)
		}
	}
	root := base
	root.UserHome = "/"
	if _, err := RenderLaunchAgent(root); err != ErrInvalidSpec {
		t.Fatalf("root home render error=%v", err)
	}
}
