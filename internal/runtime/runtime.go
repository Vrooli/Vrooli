package runtime

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Tool string

const (
	ToolDocker Tool = "docker"
	ToolGo     Tool = "go"
	ToolNode   Tool = "node"
	ToolPython Tool = "python"
	ToolHelm   Tool = "helm"
)

type ToolStatus struct {
	Name             Tool     `json:"name"`
	Command          string   `json:"command,omitempty"`
	Version          string   `json:"version,omitempty"`
	Installed        bool     `json:"installed"`
	Required         bool     `json:"required"`
	InstallSupported bool     `json:"install_supported"`
	PackageName      string   `json:"package_name,omitempty"`
	Notes            []string `json:"notes,omitempty"`
}

type Report struct {
	Environment     string       `json:"environment"`
	Host            Host         `json:"host"`
	Tools           []ToolStatus `json:"tools"`
	MissingRequired []string     `json:"missing_required,omitempty"`
	MissingOptional []string     `json:"missing_optional,omitempty"`
}

type EnsureOptions struct {
	Environment string
	SudoMode    string
	DryRun      bool
	AutoInstall bool
	Stdout      io.Writer
	Stderr      io.Writer
}

type Host struct {
	OS              string   `json:"os"`
	PackageManager  string   `json:"package_manager,omitempty"`
	SupportsSetup   bool     `json:"supports_setup"`
	SupportsDevelop bool     `json:"supports_develop"`
	SupportsSysctl  bool     `json:"supports_sysctl"`
	SupportsSystemd bool     `json:"supports_systemd"`
	Notes           []string `json:"notes,omitempty"`
}

type toolSpec struct {
	Name         Tool
	Commands     []string
	VersionArgs  []string
	PackageName  string
	RequiredEnvs map[string]bool
	InstallHint  string
}

var ErrUnsupportedPlatform = errors.New("unsupported platform")

var (
	lookPathFn       = exec.LookPath
	combinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).CombinedOutput()
	}
	installToolFn = installTool
)

func Current() Host {
	return currentHost()
}

func Inspect(environment string) (Report, error) {
	host := Current()
	env := normalizedEnvironment(environment)
	tools := make([]ToolStatus, 0, len(toolSpecs()))
	for _, spec := range toolSpecs() {
		tools = append(tools, probeTool(host, env, spec))
	}
	return summarizeReport(Report{
		Environment: env,
		Host:        host,
		Tools:       tools,
	}), nil
}

func Ensure(opts EnsureOptions) (Report, error) {
	report, err := Inspect(opts.Environment)
	if err != nil {
		return Report{}, err
	}
	if !opts.AutoInstall {
		return report, missingRequiredError(report)
	}

	for index, status := range report.Tools {
		if status.Installed || !status.Required {
			continue
		}
		updated, installErr := installToolFn(report.Host, status, opts)
		if installErr != nil {
			return Report{}, installErr
		}
		report.Tools[index] = updated
	}

	report = summarizeReport(report)
	return report, missingRequiredError(report)
}

func (h Host) ValidateSetup() error {
	if h.SupportsSetup {
		return nil
	}
	return h.unsupportedError("setup")
}

func (h Host) ValidateDevelop() error {
	if h.SupportsDevelop {
		return nil
	}
	return h.unsupportedError("develop")
}

func (h Host) unsupportedError(command string) error {
	if len(h.Notes) == 0 {
		return fmt.Errorf("%w: vrooli %s is not supported on %s", ErrUnsupportedPlatform, command, defaultOS(h.OS))
	}
	return fmt.Errorf("%w: vrooli %s is not supported on %s (%s)", ErrUnsupportedPlatform, command, defaultOS(h.OS), strings.Join(h.Notes, "; "))
}

func defaultOS(value string) string {
	if strings.TrimSpace(value) == "" {
		return "this platform"
	}
	return value
}

func normalizedEnvironment(environment string) string {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "production":
		return "production"
	case "minimal":
		return "minimal"
	default:
		return "development"
	}
}

func toolSpecs() []toolSpec {
	return []toolSpec{
		{
			Name:        ToolDocker,
			Commands:    []string{"docker"},
			VersionArgs: []string{"--version"},
			PackageName: "docker.io",
			RequiredEnvs: map[string]bool{
				"development": true,
				"production":  true,
				"minimal":     true,
			},
			InstallHint: "Install Docker Engine or Docker Desktop",
		},
		{
			Name:        ToolGo,
			Commands:    []string{"go"},
			VersionArgs: []string{"version"},
			PackageName: "golang-go",
			RequiredEnvs: map[string]bool{
				"development": true,
			},
			InstallHint: "Install Go 1.22+",
		},
		{
			Name:        ToolNode,
			Commands:    []string{"node"},
			VersionArgs: []string{"--version"},
			PackageName: "nodejs",
			RequiredEnvs: map[string]bool{
				"development": true,
			},
			InstallHint: "Install Node.js 20+",
		},
		{
			Name:        ToolPython,
			Commands:    []string{"python3", "python"},
			VersionArgs: []string{"--version"},
			PackageName: "python3",
			RequiredEnvs: map[string]bool{
				"development": true,
			},
			InstallHint: "Install Python 3.10+",
		},
		{
			Name:         ToolHelm,
			Commands:     []string{"helm"},
			VersionArgs:  []string{"version", "--short"},
			PackageName:  "helm",
			RequiredEnvs: map[string]bool{},
			InstallHint:  "Install Helm for Kubernetes packaging flows",
		},
	}
}

func probeTool(host Host, environment string, spec toolSpec) ToolStatus {
	status := ToolStatus{
		Name:             spec.Name,
		Required:         spec.RequiredEnvs[environment],
		InstallSupported: host.PackageManager != "",
		PackageName:      packageNameForHost(host, spec),
	}
	command, installed := resolveCommand(spec.Commands)
	status.Command = command
	status.Installed = installed
	if !installed {
		if spec.InstallHint != "" {
			status.Notes = append(status.Notes, spec.InstallHint)
		}
		return status
	}

	version := readVersion(command, spec.VersionArgs)
	if version != "" {
		status.Version = version
	}
	return status
}

func resolveCommand(candidates []string) (string, bool) {
	for _, candidate := range candidates {
		if _, err := lookPathFn(candidate); err == nil {
			return candidate, true
		}
	}
	return "", false
}

func readVersion(command string, args []string) string {
	if command == "" || len(args) == 0 {
		return ""
	}
	output, err := combinedOutputFn(command, args...)
	if err != nil {
		return ""
	}
	return firstLine(strings.TrimSpace(string(output)))
}

func firstLine(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}

func packageNameForHost(host Host, spec toolSpec) string {
	if host.PackageManager != "brew" {
		return spec.PackageName
	}
	switch spec.Name {
	case ToolDocker:
		return "docker"
	case ToolGo:
		return "go"
	case ToolNode:
		return "node"
	case ToolPython:
		return "python"
	default:
		return spec.PackageName
	}
}

func summarizeReport(report Report) Report {
	report.MissingRequired = report.MissingRequired[:0]
	report.MissingOptional = report.MissingOptional[:0]
	for _, tool := range report.Tools {
		name := string(tool.Name)
		if tool.Installed {
			continue
		}
		if tool.Required {
			report.MissingRequired = append(report.MissingRequired, name)
		} else {
			report.MissingOptional = append(report.MissingOptional, name)
		}
	}
	return report
}

func missingRequiredError(report Report) error {
	if len(report.MissingRequired) == 0 {
		return nil
	}
	return fmt.Errorf("missing required tools for %s: %s", normalizedEnvironment(report.Environment), strings.Join(report.MissingRequired, ", "))
}

func installTool(host Host, status ToolStatus, opts EnsureOptions) (ToolStatus, error) {
	if !status.InstallSupported || strings.TrimSpace(status.PackageName) == "" {
		status.Notes = append(status.Notes, "automatic install unavailable on this host")
		return status, nil
	}
	command, args, err := installCommand(host, status.PackageName, opts.SudoMode)
	if err != nil {
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if opts.DryRun {
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would run %s %s", command, strings.Join(args, " ")))
		return status, nil
	}

	cmd := exec.Command(command, args...)
	cmd.Stdout = writerOrDiscard(opts.Stdout)
	cmd.Stderr = writerOrDiscard(opts.Stderr)
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	commandName, installed := resolveCommand([]string{status.Command, string(status.Name)})
	status.Command = commandName
	status.Installed = installed
	if installed {
		status.Version = readVersion(commandName, defaultVersionArgs(status.Name))
	}
	return status, nil
}

func defaultVersionArgs(name Tool) []string {
	switch name {
	case ToolGo:
		return []string{"version"}
	case ToolHelm:
		return []string{"version", "--short"}
	default:
		return []string{"--version"}
	}
}

func writerOrDiscard(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return io.Discard
}
