package platform

import (
	"fmt"
	"strings"
	"time"
)

// windowsTaskNamespace is the Task Scheduler XML namespace every task
// document must declare; the validator checks it on every host.
const windowsTaskNamespace = "http://schemas.microsoft.com/windows/2004/02/mit/task"

// RenderWindowsTaskXML renders a definition as a Task Scheduler task
// document for `schtasks /Create /XML`. A daemon is a boot-triggered task the
// scheduler restarts per the restart policy; a oneshot is a boot-triggered
// task that runs once; a timer adds a repeating time trigger at
// Schedule.Every. The principal is the definition's Username under S4U, so
// the task runs without a stored password and without an interactive logon.
//
// The task name itself is not part of the document: the installer passes it
// as `/TN`, so the file is named after the definition's Name.
//
// Element order follows what `schtasks /Query /XML` exports, which is the
// order the scheduler is known to accept.
func RenderWindowsTaskXML(d ServiceDefinition) (RenderedArtifact, error) {
	if err := d.Validate(); err != nil {
		return RenderedArtifact{}, err
	}
	if d.Kind == KindSlice {
		return RenderedArtifact{}, fmt.Errorf("platform: Task Scheduler has no slice equivalent for %s", d.Name)
	}
	if strings.TrimSpace(d.Username) == "" {
		return RenderedArtifact{}, fmt.Errorf("platform: service %s needs a username for the Windows task principal", d.Name)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&b, `<Task version="1.4" xmlns="%s">`+"\n", windowsTaskNamespace)
	fmt.Fprintf(&b, "  <RegistrationInfo>\n    <Description>%s</Description>\n    <Author>Vrooli</Author>\n    <Documentation>%s</Documentation>\n  </RegistrationInfo>\n", xmlText(d.Description), xmlText(d.DocumentationURL))
	b.WriteString("  <Triggers>\n")
	b.WriteString("    <BootTrigger>\n      <Enabled>true</Enabled>\n      <Delay>PT30S</Delay>\n    </BootTrigger>\n")
	if d.Kind == KindTimer {
		// A fixed past StartBoundary is required by the schema; the repetition
		// with no Duration repeats indefinitely from that boundary.
		fmt.Fprintf(&b, "    <TimeTrigger>\n      <Enabled>true</Enabled>\n      <StartBoundary>2026-01-01T00:00:00</StartBoundary>\n      <Repetition>\n        <Interval>%s</Interval>\n        <StopAtDurationEnd>false</StopAtDurationEnd>\n      </Repetition>\n    </TimeTrigger>\n", iso8601Duration(d.Schedule.Every))
	}
	b.WriteString("  </Triggers>\n")
	fmt.Fprintf(&b, "  <Principals>\n    <Principal id=\"Author\">\n      <UserId>%s</UserId>\n      <LogonType>S4U</LogonType>\n      <RunLevel>HighestAvailable</RunLevel>\n    </Principal>\n  </Principals>\n", xmlText(d.Username))
	b.WriteString("  <Settings>\n")
	b.WriteString("    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>\n")
	b.WriteString("    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>\n")
	b.WriteString("    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>\n")
	b.WriteString("    <AllowHardTerminate>true</AllowHardTerminate>\n")
	b.WriteString("    <StartWhenAvailable>true</StartWhenAvailable>\n")
	b.WriteString("    <AllowStartOnDemand>true</AllowStartOnDemand>\n")
	b.WriteString("    <Enabled>true</Enabled>\n")
	b.WriteString("    <Hidden>false</Hidden>\n")
	if d.Protections.CPUWeight > 0 {
		fmt.Fprintf(&b, "    <Priority>%d</Priority>\n", windowsPriorityForWeight(d.Protections.CPUWeight))
	}
	if d.Kind == KindDaemon {
		b.WriteString("    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>\n")
	} else {
		b.WriteString("    <ExecutionTimeLimit>PT30M</ExecutionTimeLimit>\n")
	}
	if d.Kind == KindDaemon && d.Restart.Mode != RestartNever {
		count := d.Restart.BurstLimit
		if d.Restart.Mode == RestartAlways || count <= 0 {
			count = 999
		}
		fmt.Fprintf(&b, "    <RestartOnFailure>\n      <Interval>%s</Interval>\n      <Count>%d</Count>\n    </RestartOnFailure>\n", iso8601Duration(maxDuration(d.Restart.Delay, time.Minute)), count)
	}
	b.WriteString("  </Settings>\n")
	b.WriteString("  <Actions Context=\"Author\">\n    <Exec>\n")
	fmt.Fprintf(&b, "      <Command>%s</Command>\n", xmlText(d.Executable))
	if len(d.Args) > 0 {
		fmt.Fprintf(&b, "      <Arguments>%s</Arguments>\n", xmlText(windowsArguments(d.Args)))
	}
	if d.WorkingDirectory != "" {
		fmt.Fprintf(&b, "      <WorkingDirectory>%s</WorkingDirectory>\n", xmlText(d.WorkingDirectory))
	}
	b.WriteString("    </Exec>\n  </Actions>\n</Task>\n")
	return RenderedArtifact{Target: "windows", Files: []RenderedFile{{Name: d.Name + ".xml", Content: b.String()}}}, nil
}

// WindowsServiceCommandLine is the binPath the Service Control Manager runs
// for a daemon installed as a Windows service rather than a task.
func WindowsServiceCommandLine(d ServiceDefinition) string {
	return windowsArguments(append([]string{d.Executable}, d.Args...))
}

// windowsArguments joins tokens the way CommandLineToArgvW splits them:
// tokens with whitespace or quotes are double-quoted with inner quotes
// backslash-escaped.
func windowsArguments(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\"") {
			arg = `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}

// iso8601Duration renders a duration as the PT#H#M#S form Task Scheduler
// expects.
func iso8601Duration(d time.Duration) string {
	d = d.Round(time.Second)
	hours := int64(d / time.Hour)
	minutes := int64((d % time.Hour) / time.Minute)
	seconds := int64((d % time.Minute) / time.Second)
	var b strings.Builder
	b.WriteString("PT")
	if hours > 0 {
		fmt.Fprintf(&b, "%dH", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&b, "%dM", minutes)
	}
	if seconds > 0 || (hours == 0 && minutes == 0) {
		fmt.Fprintf(&b, "%dS", seconds)
	}
	return b.String()
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
