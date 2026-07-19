package runtimesupervisor

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// launchdLabel is deliberately distinct from Bridge's agent label. The runtime
// supervisor owns local scenario lifecycle; the Bridge agent owns node pairing
// and control-plane connectivity.
const launchdLabel = "com.vrooli.runtime-supervisor"

// launchdServicePlan contains only deterministic user-domain launchd values.
// Keeping it platform-neutral lets Linux CI verify macOS service semantics
// without invoking launchctl.
type launchdServicePlan struct {
	PlistPath     string
	DomainTarget  string
	ServiceTarget string
}

func newLaunchdServicePlan(home string, uid int) launchdServicePlan {
	domain := fmt.Sprintf("gui/%d", uid)
	return launchdServicePlan{
		PlistPath:     filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist"),
		DomainTarget:  domain,
		ServiceTarget: domain + "/" + launchdLabel,
	}
}

// launchAgentPlistContent mirrors the systemd user unit: same argv, HOME and
// VROOLI_RUNTIME_SUPERVISOR environment, optional source root + working
// directory, and restart-on-failure (KeepAlive SuccessfulExit=false ≈
// Restart=on-failure). launchd has no journal, so output goes to
// ~/Library/Logs.
func launchAgentPlistContent(executable string, home string, sourceRoot string) string {
	sourceRoot = strings.TrimSpace(sourceRoot)
	logPath := filepath.Join(home, "Library", "Logs", launchdLabel+".log")

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	fmt.Fprintf(&b, "  <key>Label</key>\n  <string>%s</string>\n", plistValue(launchdLabel))
	b.WriteString("  <key>ProgramArguments</key>\n  <array>\n")
	for _, arg := range []string{executable, "--no-stale-check", "runtime", "supervisor", "run"} {
		fmt.Fprintf(&b, "    <string>%s</string>\n", plistValue(arg))
	}
	b.WriteString("  </array>\n")
	b.WriteString("  <key>EnvironmentVariables</key>\n  <dict>\n")
	fmt.Fprintf(&b, "    <key>HOME</key>\n    <string>%s</string>\n", plistValue(home))
	b.WriteString("    <key>VROOLI_RUNTIME_SUPERVISOR</key>\n    <string>on</string>\n")
	if sourceRoot != "" {
		fmt.Fprintf(&b, "    <key>VROOLI_SOURCE_ROOT</key>\n    <string>%s</string>\n", plistValue(sourceRoot))
	}
	b.WriteString("  </dict>\n")
	if sourceRoot != "" {
		fmt.Fprintf(&b, "  <key>WorkingDirectory</key>\n  <string>%s</string>\n", plistValue(sourceRoot))
	}
	b.WriteString("  <key>RunAtLoad</key>\n  <true/>\n")
	b.WriteString("  <key>KeepAlive</key>\n  <dict>\n    <key>SuccessfulExit</key>\n    <false/>\n  </dict>\n")
	b.WriteString("  <key>ThrottleInterval</key>\n  <integer>5</integer>\n")
	fmt.Fprintf(&b, "  <key>StandardOutPath</key>\n  <string>%s</string>\n", plistValue(logPath))
	fmt.Fprintf(&b, "  <key>StandardErrorPath</key>\n  <string>%s</string>\n", plistValue(logPath))
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func plistValue(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}
