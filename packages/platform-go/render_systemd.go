package platform

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RenderSystemd renders a definition as systemd unit files. A daemon or
// oneshot renders one .service; a timer renders its .service and .timer.
//
// Quoting is per directive and getting it wrong is fatal rather than
// cosmetic: Environment= and ExecStart= go through systemd's quote-aware
// parser, so values with spaces are quoted with strconv.Quote; WorkingDirectory=
// and the log paths take the rest of the line verbatim, and a quoted value
// there is rejected as "path is not absolute". TestSystemdRenderPassesSystemdAnalyze
// asserts this against systemd itself rather than against our reading of it.
func RenderSystemd(d ServiceDefinition) (RenderedArtifact, error) {
	if err := d.Validate(); err != nil {
		return RenderedArtifact{}, err
	}
	if d.Kind == KindSlice {
		return RenderSystemdSlice(d)
	}
	serviceName := d.Name + ".service"
	artifact := RenderedArtifact{Target: "linux", Files: []RenderedFile{{Name: serviceName, Content: systemdService(d)}}}
	if d.Kind == KindTimer {
		artifact.Files = append(artifact.Files, RenderedFile{Name: d.Name + ".timer", Content: systemdTimer(d, serviceName)})
	}
	return artifact, nil
}

func systemdService(d ServiceDefinition) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s\n", d.Description)
	fmt.Fprintf(&b, "Documentation=%s\n", d.DocumentationURL)
	// network-online.target exists only in the system manager; a user unit
	// that orders after it waits on a target the user manager never reaches.
	if d.Scope == ScopeSystem && d.Kind == KindDaemon {
		b.WriteString("After=network-online.target\nWants=network-online.target\n")
	}
	if d.OnFailureUnit != "" {
		fmt.Fprintf(&b, "OnFailure=%s\n", d.OnFailureUnit)
	}
	if d.Restart.BurstLimit > 0 && d.Restart.BurstWindow > 0 {
		fmt.Fprintf(&b, "StartLimitIntervalSec=%s\nStartLimitBurst=%d\n", systemdSeconds(d.Restart.BurstWindow), d.Restart.BurstLimit)
	}
	b.WriteString("\n[Service]\n")
	if d.Kind == KindDaemon {
		b.WriteString("Type=simple\n")
	} else {
		b.WriteString("Type=oneshot\n")
	}
	if d.Scope == ScopeSystem && d.Username != "" {
		fmt.Fprintf(&b, "User=%s\n", d.Username)
	}
	for _, key := range d.envKeys() {
		fmt.Fprintf(&b, "Environment=%s=%s\n", key, strconv.Quote(d.Env[key]))
	}
	if d.WorkingDirectory != "" {
		fmt.Fprintf(&b, "WorkingDirectory=%s\n", d.WorkingDirectory)
	}
	if d.Logs.Stdout != "" {
		fmt.Fprintf(&b, "StandardOutput=append:%s\n", d.Logs.Stdout)
	}
	if d.Logs.Stderr != "" {
		fmt.Fprintf(&b, "StandardError=append:%s\n", d.Logs.Stderr)
	}
	fmt.Fprintf(&b, "ExecStart=%s\n", systemdExecLine(d))
	if d.Kind == KindDaemon {
		fmt.Fprintf(&b, "Restart=%s\n", systemdRestart(d.Restart.Mode))
		if d.Restart.Delay > 0 {
			fmt.Fprintf(&b, "RestartSec=%s\n", systemdSeconds(d.Restart.Delay))
		}
	}
	if d.StopTimeout > 0 {
		fmt.Fprintf(&b, "TimeoutStopSec=%s\n", systemdSeconds(d.StopTimeout))
	}
	writeSystemdContainment(&b, d.Protections.Containment)
	if d.Protections.MemoryMin != "" {
		fmt.Fprintf(&b, "MemoryMin=%s\n", d.Protections.MemoryMin)
	}
	if d.Protections.OOMScoreAdjust != 0 {
		fmt.Fprintf(&b, "OOMScoreAdjust=%d\n", d.Protections.OOMScoreAdjust)
	}
	if d.Kind == KindDaemon {
		fmt.Fprintf(&b, "\n[Install]\nWantedBy=%s\n", systemdWantedBy(d.Scope))
	}
	return b.String()
}

// writeSystemdContainment emits the ceiling directives shared by services
// and slices, in a fixed order, omitting unset fields.
func writeSystemdContainment(b *strings.Builder, c Containment) {
	if c.Slice != "" {
		fmt.Fprintf(b, "Slice=%s\n", c.Slice)
	}
	if c.CPUWeight > 0 {
		fmt.Fprintf(b, "CPUWeight=%d\n", c.CPUWeight)
	}
	if c.MemoryHigh != "" {
		fmt.Fprintf(b, "MemoryHigh=%s\n", c.MemoryHigh)
	}
	if c.MemoryMax != "" {
		fmt.Fprintf(b, "MemoryMax=%s\n", c.MemoryMax)
	}
	if c.TasksMax > 0 {
		fmt.Fprintf(b, "TasksMax=%d\n", c.TasksMax)
	}
}

// RenderSystemdSlice renders a KindSlice definition as <name>.slice: a
// resource-control parent whose ceilings every scope started inside it
// inherits. systemd-oomd is asked to kill inside it under memory pressure
// when a memory ceiling is set, so a storm is reclaimed in the slice and not
// from the desktop or the supervisors.
func RenderSystemdSlice(d ServiceDefinition) (RenderedArtifact, error) {
	if err := d.Validate(); err != nil {
		return RenderedArtifact{}, err
	}
	if d.Kind != KindSlice {
		return RenderedArtifact{}, fmt.Errorf("platform: %s is a %s, not a slice", d.Name, d.Kind)
	}
	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s\n", d.Description)
	if d.DocumentationURL != "" {
		fmt.Fprintf(&b, "Documentation=%s\n", d.DocumentationURL)
	}
	b.WriteString("\n[Slice]\n")
	c := d.Protections.Containment
	c.Slice = ""
	writeSystemdContainment(&b, c)
	if c.MemoryHigh != "" || c.MemoryMax != "" {
		b.WriteString("ManagedOOMMemoryPressure=kill\n")
	}
	return RenderedArtifact{Target: "linux", Files: []RenderedFile{{Name: d.Name + ".slice", Content: b.String()}}}, nil
}

func systemdTimer(d ServiceDefinition, serviceName string) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s (timer)\n", d.Description)
	fmt.Fprintf(&b, "Documentation=%s\n", d.DocumentationURL)
	b.WriteString("\n[Timer]\n")
	if d.Schedule.OnBoot > 0 {
		fmt.Fprintf(&b, "OnBootSec=%s\n", systemdSeconds(d.Schedule.OnBoot))
	}
	fmt.Fprintf(&b, "OnUnitActiveSec=%s\n", systemdSeconds(d.Schedule.Every))
	if d.Schedule.Persistent {
		b.WriteString("Persistent=true\n")
	}
	fmt.Fprintf(&b, "Unit=%s\n", serviceName)
	b.WriteString("\n[Install]\nWantedBy=timers.target\n")
	return b.String()
}

func systemdWantedBy(scope Scope) string {
	if scope == ScopeSystem {
		return "multi-user.target"
	}
	return "default.target"
}

func systemdRestart(mode RestartMode) string {
	switch mode {
	case RestartAlways:
		return "always"
	case RestartOnFailure:
		return "on-failure"
	default:
		return "no"
	}
}

func systemdSeconds(d time.Duration) string {
	return strconv.FormatInt(int64(d/time.Second), 10) + "s"
}

// systemdExecLine renders ExecStart=. The executable is always quoted; an
// argument is quoted only when systemd's word splitter would otherwise
// misread it, so the common case stays readable in `systemctl cat`.
func systemdExecLine(d ServiceDefinition) string {
	parts := []string{strconv.Quote(d.Executable)}
	for _, arg := range d.Args {
		parts = append(parts, systemdArg(arg))
	}
	return strings.Join(parts, " ")
}

func systemdArg(arg string) string {
	if arg == "" || strings.ContainsAny(arg, " \t\"'\\;") {
		return strconv.Quote(arg)
	}
	return arg
}
