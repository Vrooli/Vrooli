package runtimesupervisor

import (
	"strings"
	"testing"
)

func TestLaunchAgentPlistContentCarriesSourceRoot(t *testing.T) {
	content := launchAgentPlistContent("/opt/vrooli/bin/vrooli", "/Users/tester", "/srv/vrooli")
	for _, want := range []string{
		"<key>Label</key>\n  <string>com.vrooli.runtime-supervisor</string>",
		"<string>/opt/vrooli/bin/vrooli</string>",
		"<string>--no-stale-check</string>",
		"<string>runtime</string>",
		"<string>supervisor</string>",
		"<string>run</string>",
		"<key>HOME</key>\n    <string>/Users/tester</string>",
		"<key>VROOLI_RUNTIME_SUPERVISOR</key>\n    <string>on</string>",
		"<key>VROOLI_SOURCE_ROOT</key>\n    <string>/srv/vrooli</string>",
		"<key>WorkingDirectory</key>\n  <string>/srv/vrooli</string>",
		"<key>RunAtLoad</key>\n  <true/>",
		"<key>SuccessfulExit</key>\n    <false/>",
		"<key>StandardOutPath</key>\n  <string>/Users/tester/Library/Logs/com.vrooli.runtime-supervisor.log</string>",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("plist content missing %q:\n%s", want, content)
		}
	}
}

func TestLaunchAgentPlistContentOmitsSourceRootWhenEmpty(t *testing.T) {
	content := launchAgentPlistContent("/opt/vrooli/bin/vrooli", "/Users/tester", "")
	for _, unwanted := range []string{"VROOLI_SOURCE_ROOT", "WorkingDirectory"} {
		if strings.Contains(content, unwanted) {
			t.Fatalf("plist content should omit %q when source root is empty:\n%s", unwanted, content)
		}
	}
}

func TestLaunchAgentPlistContentEscapesXML(t *testing.T) {
	content := launchAgentPlistContent("/opt/dir with <spaces> & such/vrooli", "/Users/tester", "")
	if !strings.Contains(content, "/opt/dir with &lt;spaces&gt; &amp; such/vrooli") {
		t.Fatalf("plist content should XML-escape the executable path:\n%s", content)
	}
	if strings.Contains(content, "<spaces>") {
		t.Fatalf("plist content leaked unescaped XML:\n%s", content)
	}
}

func TestLaunchdServicePlanUsesConfiguredUserHome(t *testing.T) {
	plan := newLaunchdServicePlan("/Users/bridge", 501)
	if plan.PlistPath != "/Users/bridge/Library/LaunchAgents/com.vrooli.runtime-supervisor.plist" {
		t.Fatalf("plist path = %q", plan.PlistPath)
	}
	if plan.DomainTarget != "gui/501" || plan.ServiceTarget != "gui/501/com.vrooli.runtime-supervisor" {
		t.Fatalf("launchd targets = %#v", plan)
	}
}
