package platform

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// VerdictState is the outcome of running a rendered artifact through the
// native validator.
type VerdictState string

const (
	// VerdictAccepted means the native validator ran and found nothing.
	VerdictAccepted VerdictState = "accepted"
	// VerdictUnavailable means no native validator could run here; the
	// artifact is unproven, which is never the same as accepted.
	VerdictUnavailable VerdictState = "unavailable"
	// VerdictRejected means the validator ran and reported a finding.
	VerdictRejected VerdictState = "rejected"
)

// Verdict is the validator's answer, kept as safeguard evidence so a later
// readiness inspection can read it back from `vrooli setup status --json`.
type Verdict struct {
	State     VerdictState `json:"state"`
	Validator string       `json:"validator"`
	Output    string       `json:"output,omitempty"`
}

// Rejected reports whether the artifact must not be enabled.
func (v Verdict) Rejected() bool { return v.State == VerdictRejected }

// String renders the verdict for a status note.
func (v Verdict) String() string {
	if v.Output == "" {
		return fmt.Sprintf("%s: %s", v.Validator, v.State)
	}
	return fmt.Sprintf("%s: %s: %s", v.Validator, v.State, v.Output)
}

const validatorTimeout = 30 * time.Second

// ValidateArtifact dispatches on the artifact's target.
func ValidateArtifact(artifact RenderedArtifact, scope Scope) Verdict {
	switch artifact.Target {
	case "linux":
		return ValidateSystemd(artifact, scope)
	case "darwin":
		return ValidateLaunchd(artifact)
	case "windows":
		return ValidateWindowsTask(artifact)
	default:
		return Verdict{State: VerdictUnavailable, Validator: "none", Output: "no validator for target " + artifact.Target}
	}
}

// ValidateSystemd writes the artifact into a private directory and runs
// `systemd-analyze --user verify` (or --system) on it. A finding is any
// output line that names one of the files, or a non-zero exit; systemd 255
// reports unknown directives on stdout with exit 0 and a missing executable
// with exit 1, so both signals count. Lines about other units on the host
// are ignored: the verifier loads the whole unit path and reports on all of
// it.
func ValidateSystemd(artifact RenderedArtifact, scope Scope) Verdict {
	const validator = "systemd-analyze verify"
	analyze, err := exec.LookPath("systemd-analyze")
	if err != nil {
		return Verdict{State: VerdictUnavailable, Validator: validator, Output: "systemd-analyze is not installed"}
	}
	dir, err := os.MkdirTemp("", "vrooli-unit-verify-")
	if err != nil {
		return Verdict{State: VerdictUnavailable, Validator: validator, Output: "temp dir: " + err.Error()}
	}
	defer os.RemoveAll(dir)
	paths := make([]string, 0, len(artifact.Files))
	for _, file := range artifact.Files {
		path := filepath.Join(dir, file.Name)
		if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
			return Verdict{State: VerdictUnavailable, Validator: validator, Output: "write " + path + ": " + err.Error()}
		}
		paths = append(paths, path)
	}
	scopeFlag := "--user"
	if scope == ScopeSystem {
		scopeFlag = "--system"
	}
	ctx, cancel := context.WithTimeout(context.Background(), validatorTimeout)
	defer cancel()
	output, runErr := exec.CommandContext(ctx, analyze, append([]string{scopeFlag, "verify"}, paths...)...).CombinedOutput()
	findings := systemdFindings(string(output), paths)
	if runErr != nil && len(findings) == 0 {
		if ctx.Err() != nil {
			return Verdict{State: VerdictUnavailable, Validator: validator, Output: "timed out after " + validatorTimeout.String()}
		}
		findings = append(findings, strings.TrimSpace(runErr.Error()))
		if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
			findings = append(findings, trimmed)
		}
	}
	if len(findings) > 0 {
		return Verdict{State: VerdictRejected, Validator: validator, Output: strings.Join(findings, "\n")}
	}
	return Verdict{State: VerdictAccepted, Validator: validator}
}

// systemdFindings keeps the verifier lines that are about our files: those
// naming a written path, or prefixed with the bare unit name the way
// "vrooli-x.service: Command /y is not executable" is.
func systemdFindings(output string, paths []string) []string {
	var findings []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, path := range paths {
			if strings.Contains(line, path) || strings.HasPrefix(line, filepath.Base(path)+":") {
				findings = append(findings, line)
				break
			}
		}
	}
	return findings
}

// ValidateLaunchd runs `plutil -lint` on the plist when the tool exists.
// Elsewhere the plist is still parsed as XML so a malformed document is
// rejected everywhere, but a well-formed one stays unavailable rather than
// accepted: only launchd's own tooling can say it is a valid property list.
func ValidateLaunchd(artifact RenderedArtifact) Verdict {
	const validator = "plutil -lint"
	for _, file := range artifact.Files {
		if err := xmlWellFormed(file.Content); err != nil {
			return Verdict{State: VerdictRejected, Validator: validator, Output: file.Name + ": " + err.Error()}
		}
	}
	plutil, err := exec.LookPath("plutil")
	if err != nil {
		return Verdict{State: VerdictUnavailable, Validator: validator, Output: "plutil is not installed; the plist is well-formed XML but unproven"}
	}
	dir, err := os.MkdirTemp("", "vrooli-plist-lint-")
	if err != nil {
		return Verdict{State: VerdictUnavailable, Validator: validator, Output: "temp dir: " + err.Error()}
	}
	defer os.RemoveAll(dir)
	ctx, cancel := context.WithTimeout(context.Background(), validatorTimeout)
	defer cancel()
	for _, file := range artifact.Files {
		path := filepath.Join(dir, file.Name)
		if err := os.WriteFile(path, []byte(file.Content), 0o644); err != nil {
			return Verdict{State: VerdictUnavailable, Validator: validator, Output: "write " + path + ": " + err.Error()}
		}
		output, runErr := exec.CommandContext(ctx, plutil, "-lint", path).CombinedOutput()
		if runErr != nil {
			return Verdict{State: VerdictRejected, Validator: validator, Output: file.Name + ": " + strings.TrimSpace(string(output))}
		}
	}
	return Verdict{State: VerdictAccepted, Validator: validator}
}

// windowsTaskElements is the Task Scheduler schema element set the renderer
// draws from. Anything outside it is a rendering defect the scheduler would
// reject at import.
var windowsTaskElements = map[string]bool{
	"Task": true, "RegistrationInfo": true, "Description": true, "Author": true, "Documentation": true, "URI": true, "Date": true,
	"Triggers": true, "BootTrigger": true, "TimeTrigger": true, "LogonTrigger": true, "Enabled": true, "Delay": true, "StartBoundary": true,
	"Repetition": true, "Interval": true, "Duration": true, "StopAtDurationEnd": true,
	"Principals": true, "Principal": true, "UserId": true, "LogonType": true, "RunLevel": true,
	"Settings": true, "MultipleInstancesPolicy": true, "DisallowStartIfOnBatteries": true, "StopIfGoingOnBatteries": true,
	"AllowHardTerminate": true, "StartWhenAvailable": true, "RunOnlyIfNetworkAvailable": true, "AllowStartOnDemand": true,
	"Hidden": true, "RunOnlyIfIdle": true, "WakeToRun": true, "ExecutionTimeLimit": true, "Priority": true,
	"RestartOnFailure": true, "Count": true,
	"Actions": true, "Exec": true, "Command": true, "Arguments": true, "WorkingDirectory": true,
}

// windowsTaskDocument is the subset of the task schema the validator reads
// back to prove the parts that matter are present.
type windowsTaskDocument struct {
	XMLName    xml.Name `xml:"Task"`
	Namespace  string   `xml:"xmlns,attr"`
	Principals struct {
		Principal struct {
			UserId    string `xml:"UserId"`
			LogonType string `xml:"LogonType"`
		} `xml:"Principal"`
	} `xml:"Principals"`
	Actions struct {
		Exec struct {
			Command string `xml:"Command"`
		} `xml:"Exec"`
	} `xml:"Actions"`
	Triggers struct {
		Boot *struct{} `xml:"BootTrigger"`
		Time *struct{} `xml:"TimeTrigger"`
	} `xml:"Triggers"`
}

// ValidateWindowsTask parses the task document with encoding/xml against the
// Task Scheduler element set: the namespace must match, every element must
// belong to the schema, and the principal, trigger and exec command must be
// present. The check is pure Go, so it is available on every host.
func ValidateWindowsTask(artifact RenderedArtifact) Verdict {
	const validator = "task-scheduler-xml"
	for _, file := range artifact.Files {
		if err := windowsTaskFindings(file.Content); err != nil {
			return Verdict{State: VerdictRejected, Validator: validator, Output: file.Name + ": " + err.Error()}
		}
	}
	return Verdict{State: VerdictAccepted, Validator: validator}
}

func windowsTaskFindings(content string) error {
	if !strings.HasPrefix(content, `<?xml version="1.0" encoding="UTF-8"?>`) {
		return fmt.Errorf("missing UTF-8 XML declaration")
	}
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("malformed XML: %w", err)
		}
		if start, ok := token.(xml.StartElement); ok {
			if !windowsTaskElements[start.Name.Local] {
				return fmt.Errorf("element <%s> is not in the Task Scheduler schema", start.Name.Local)
			}
			if start.Name.Space != windowsTaskNamespace {
				return fmt.Errorf("element <%s> is outside the task namespace", start.Name.Local)
			}
		}
	}
	var doc windowsTaskDocument
	if err := xml.NewDecoder(bytes.NewReader([]byte(content))).Decode(&doc); err != nil {
		return fmt.Errorf("decode task: %w", err)
	}
	if strings.TrimSpace(doc.Principals.Principal.UserId) == "" {
		return fmt.Errorf("task principal has an empty <UserId>")
	}
	if doc.Principals.Principal.LogonType == "" {
		return fmt.Errorf("task principal has no <LogonType>")
	}
	if strings.TrimSpace(doc.Actions.Exec.Command) == "" {
		return fmt.Errorf("task has no <Exec><Command>")
	}
	if doc.Triggers.Boot == nil && doc.Triggers.Time == nil {
		return fmt.Errorf("task has no trigger")
	}
	return nil
}

func xmlWellFormed(content string) error {
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("malformed XML: %w", err)
		}
	}
}
