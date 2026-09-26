package platform

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// RenderLaunchd renders a definition as one launchd property list. A daemon
// keeps itself alive per its restart policy; a oneshot runs at load; a timer
// runs at load and every Schedule.Every via StartInterval.
func RenderLaunchd(d ServiceDefinition) (RenderedArtifact, error) {
	if err := d.Validate(); err != nil {
		return RenderedArtifact{}, err
	}
	if d.Kind == KindSlice {
		return RenderedArtifact{}, fmt.Errorf("platform: launchd has no slice equivalent for %s", d.Name)
	}
	if strings.TrimSpace(d.Label) == "" {
		return RenderedArtifact{}, fmt.Errorf("platform: service %s needs a launchd label", d.Name)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	fmt.Fprintf(&b, "  <key>Label</key>\n  <string>%s</string>\n", xmlText(d.Label))
	b.WriteString("  <key>ProgramArguments</key>\n  <array>\n")
	for _, arg := range append([]string{d.Executable}, d.Args...) {
		fmt.Fprintf(&b, "    <string>%s</string>\n", xmlText(arg))
	}
	b.WriteString("  </array>\n")
	if len(d.Env) > 0 {
		b.WriteString("  <key>EnvironmentVariables</key>\n  <dict>\n")
		for _, key := range d.envKeys() {
			fmt.Fprintf(&b, "    <key>%s</key>\n    <string>%s</string>\n", xmlText(key), xmlText(d.Env[key]))
		}
		b.WriteString("  </dict>\n")
	}
	if d.WorkingDirectory != "" {
		fmt.Fprintf(&b, "  <key>WorkingDirectory</key>\n  <string>%s</string>\n", xmlText(d.WorkingDirectory))
	}
	// UserName is honored by LaunchDaemons only; a LaunchAgent already runs
	// as the user whose domain loaded it.
	if d.Scope == ScopeSystem && d.Username != "" {
		fmt.Fprintf(&b, "  <key>UserName</key>\n  <string>%s</string>\n", xmlText(d.Username))
	}
	if d.Logs.Stdout != "" {
		fmt.Fprintf(&b, "  <key>StandardOutPath</key>\n  <string>%s</string>\n", xmlText(d.Logs.Stdout))
	}
	if d.Logs.Stderr != "" {
		fmt.Fprintf(&b, "  <key>StandardErrorPath</key>\n  <string>%s</string>\n", xmlText(d.Logs.Stderr))
	}
	b.WriteString("  <key>RunAtLoad</key>\n  <true/>\n")
	switch d.Kind {
	case KindDaemon:
		b.WriteString("  <key>KeepAlive</key>\n")
		switch d.Restart.Mode {
		case RestartAlways:
			b.WriteString("  <true/>\n")
		case RestartOnFailure:
			b.WriteString("  <dict>\n    <key>SuccessfulExit</key>\n    <false/>\n  </dict>\n")
		default:
			b.WriteString("  <false/>\n")
		}
	case KindTimer:
		fmt.Fprintf(&b, "  <key>StartInterval</key>\n  <integer>%d</integer>\n", int64(d.Schedule.Every/time.Second))
	}
	if d.Restart.Delay > 0 && d.Kind == KindDaemon {
		fmt.Fprintf(&b, "  <key>ThrottleInterval</key>\n  <integer>%d</integer>\n", int64(d.Restart.Delay/time.Second))
	}
	if d.StopTimeout > 0 {
		fmt.Fprintf(&b, "  <key>ExitTimeOut</key>\n  <integer>%d</integer>\n", int64(d.StopTimeout/time.Second))
	}
	writeLaunchdContainment(&b, d.Protections.Containment)
	b.WriteString("</dict>\n</plist>\n")
	return RenderedArtifact{Target: "darwin", Files: []RenderedFile{{Name: d.Label + ".plist", Content: b.String()}}}, nil
}

// writeLaunchdContainment renders the ceiling vocabulary in launchd terms:
// CPUWeight becomes Nice (see niceForWeight), TasksMax becomes the
// NumberOfProcesses soft limit and an absolute MemoryMax becomes
// ResidentSetSize. A percentage ceiling has no launchd meaning (limits are
// per process, not per tree) and is omitted here; the session body applies
// it through the rlimit shim against the host's physical memory instead.
func writeLaunchdContainment(b *strings.Builder, c Containment) {
	if c.CPUWeight > 0 {
		fmt.Fprintf(b, "  <key>Nice</key>\n  <integer>%d</integer>\n", niceForWeight(c.CPUWeight))
	}
	maxBytes, maxPercent, _ := parseMemoryCeiling(c.MemoryMax)
	if c.TasksMax <= 0 && (maxPercent || maxBytes <= 0) {
		return
	}
	b.WriteString("  <key>SoftResourceLimits</key>\n  <dict>\n")
	if c.TasksMax > 0 {
		fmt.Fprintf(b, "    <key>NumberOfProcesses</key>\n    <integer>%d</integer>\n", c.TasksMax)
	}
	if maxBytes > 0 && !maxPercent {
		fmt.Fprintf(b, "    <key>ResidentSetSize</key>\n    <integer>%d</integer>\n", maxBytes)
	}
	b.WriteString("  </dict>\n")
}

// xmlText escapes a value for an XML text node.
func xmlText(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}
