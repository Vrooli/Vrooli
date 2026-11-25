package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

const maxPackageConfigBytes = 1 << 20

type prerequisiteCheck struct {
	Tool     string `json:"tool"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
	NextStep string `json:"next_step,omitempty"`
}

type buildPrerequisiteReport struct {
	TargetPlatform      string              `json:"target_platform"`
	BuildHost           string              `json:"build_host"`
	Status              string              `json:"status"`
	PackageConfigStatus string              `json:"package_config_status"`
	PackageConfigSource string              `json:"package_config_source"`
	PackageFormats      []string            `json:"package_formats"`
	Scope               string              `json:"scope"`
	Checks              []prerequisiteCheck `json:"checks"`
	Limitations         []string            `json:"limitations,omitempty"`
}

type executableLookup func(string) (string, error)

type packageTargetConfig struct {
	Formats []string
	Status  string
	Source  string
	Problem string
}

func buildPrerequisitesPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(
		func(ctx cliapp.OperationContext) (buildPrerequisiteReport, error) {
			return inspectBuildPrerequisites(ctx.Flag("platform"), runtime.GOOS, exec.LookPath, ctx.Flag("package-json"))
		},
		renderBuildPrerequisitesHuman,
		func(ctx cliapp.RunContext, report buildPrerequisiteReport) error {
			return cliapp.PrintJSON(ctx.Stdout(), report)
		},
	)
}

// inspectBuildPrerequisites uses the generated application's documented default
// target formats unless the operator supplies that app's package.json. It does
// not execute electron-builder, install tools, or claim that packaging will pass.
func inspectBuildPrerequisites(targetPlatform, hostGOOS string, lookPath executableLookup, packageJSONPath ...string) (buildPrerequisiteReport, error) {
	targetPlatform = strings.ToLower(strings.TrimSpace(targetPlatform))
	if targetPlatform != "linux" && targetPlatform != "win" && targetPlatform != "mac" {
		return buildPrerequisiteReport{}, fmt.Errorf("unsupported platform %q; choose linux, win, or mac", targetPlatform)
	}
	if lookPath == nil {
		return buildPrerequisiteReport{}, errors.New("executable lookup is not configured")
	}

	hostPlatform := normalizeBuildHost(hostGOOS)
	configPath := ""
	if len(packageJSONPath) > 0 {
		configPath = strings.TrimSpace(packageJSONPath[0])
	}
	config := resolvePackageTargets(targetPlatform, configPath)
	report := buildPrerequisiteReport{
		TargetPlatform:      targetPlatform,
		BuildHost:           hostPlatform,
		Status:              "ready",
		PackageConfigStatus: config.Status,
		PackageConfigSource: config.Source,
		PackageFormats:      append([]string(nil), config.Formats...),
		Scope:               "checks common host tools for the selected electron-builder targets; it does not prove a build or inspect signing, target runtime, or all package behavior",
		Checks:              []prerequisiteCheck{},
		Limitations:         []string{},
	}
	if config.Problem != "" {
		report.Status = "limited"
		report.Limitations = append(report.Limitations, config.Problem)
	}

	checked := make(map[string]bool)
	checkTool := func(name, reason, next string) {
		if checked[name] {
			return
		}
		checked[name] = true
		_, err := lookPath(name)
		check := prerequisiteCheck{Tool: name, Required: true, Reason: reason, NextStep: next}
		if err != nil {
			check.Status = "missing"
			report.Status = "blocked"
		} else {
			check.Status = "present"
		}
		report.Checks = append(report.Checks, check)
	}
	limited := func(reason string) {
		if report.Status == "ready" {
			report.Status = "limited"
		}
		report.Limitations = append(report.Limitations, reason)
	}
	if len(config.Formats) == 0 {
		return report, nil
	}

	for _, format := range config.Formats {
		formatName := strings.ToLower(format)
		switch targetPlatform {
		case "linux":
			switch formatName {
			case "appimage":
				if hostPlatform != "linux" {
					limited("The Linux AppImage target is cross-host on this machine; the cross-platform guide documents Docker or WSL, which this command does not validate.")
				}
			case "deb":
				if hostPlatform == "linux" {
					checkTool("fpm", "The selected DEB target requires fpm.", "Provide fpm through the host's approved setup path, then rerun this check.")
					checkTool("ruby", "The selected DEB target uses the Ruby-based fpm package builder.", "Provide Ruby through the host's approved setup path, then rerun this check.")
				} else {
					limited("The selected Linux DEB target is cross-host; the cross-platform guide documents Docker or WSL, which this command does not validate.")
				}
			case "rpm":
				limited("The owner has no validated host-prerequisite rule for the selected RPM target; this result is unknown until that rule is established.")
			default:
				limited(fmt.Sprintf("The selected Linux package format %q has no owner prerequisite rule.", format))
			}
		case "win":
			switch formatName {
			case "nsis", "portable":
				if hostPlatform == "linux" {
					checkTool("wine", "The selected Windows target uses Wine when built on Linux.", "Provide working Wine through the host's approved setup path, then rerun this check.")
				} else if hostPlatform != "win" {
					limited("This Windows target is cross-host on this machine; only Linux-hosted Wine is covered by the owner prerequisite rules.")
				}
			case "msi":
				if hostPlatform == "linux" {
					checkTool("wine", "The selected MSI target uses Wine on Linux, but the owner guide reports the MSI/WiX path as unstable there.", "Use the supported Windows build host for MSI packaging.")
					limited("The owner guide describes MSI packaging through Linux Wine as unstable; presence of Wine does not establish MSI readiness.")
				} else if hostPlatform != "win" {
					limited("The selected Windows MSI target is unsupported on this host by the owner guide.")
				}
			default:
				limited(fmt.Sprintf("The selected Windows package format %q has no owner prerequisite rule.", format))
			}
		case "mac":
			switch formatName {
			case "zip":
				if hostPlatform == "linux" || hostPlatform == "win" {
					checkTool("docker", "The owner cross-platform guide documents Docker for the selected macOS ZIP target on this host.", "Provide Docker through the host's approved setup path, then rerun this check.")
				}
			case "dmg", "pkg":
				if hostPlatform != "mac" {
					limited(fmt.Sprintf("The selected macOS %s target requires a macOS build host per the owner guide.", strings.ToUpper(format)))
				}
			default:
				limited(fmt.Sprintf("The selected macOS package format %q has no owner prerequisite rule.", format))
			}
		}
	}
	return report, nil
}

func resolvePackageTargets(platform, packageJSONPath string) packageTargetConfig {
	if packageJSONPath == "" {
		defaults := map[string][]string{"linux": {"AppImage"}, "win": {"nsis"}, "mac": {"zip"}}
		return packageTargetConfig{Formats: defaults[platform], Status: "template-default", Source: "generated-template"}
	}

	info, err := os.Stat(packageJSONPath)
	if err != nil {
		return packageTargetConfig{Status: "unavailable", Source: "selected-package-json", Problem: "The selected package.json could not be read; no per-app target was inferred."}
	}
	if !info.Mode().IsRegular() {
		return packageTargetConfig{Status: "invalid", Source: "selected-package-json", Problem: "The selected package.json is not a regular file; no per-app target was inferred."}
	}
	if info.Size() > maxPackageConfigBytes {
		return packageTargetConfig{Status: "invalid", Source: "selected-package-json", Problem: "The selected package.json exceeds the 1 MiB read limit; no per-app target was inferred."}
	}
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return packageTargetConfig{Status: "unavailable", Source: "selected-package-json", Problem: "The selected package.json could not be read; no per-app target was inferred."}
	}

	var root struct {
		Build map[string]json.RawMessage `json:"build"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return packageTargetConfig{Status: "invalid", Source: "selected-package-json", Problem: "The selected package.json is not valid JSON; no per-app target was inferred."}
	}
	platformConfig, ok := root.Build[platform]
	if !ok {
		return packageTargetConfig{Status: "unconfigured", Source: "selected-package-json", Problem: "The selected package.json has no explicit build." + platform + ".target; electron-builder defaults or other config may apply, so prerequisites are unknown."}
	}
	var entry struct {
		Target json.RawMessage `json:"target"`
	}
	if err := json.Unmarshal(platformConfig, &entry); err != nil || len(entry.Target) == 0 || string(entry.Target) == "null" {
		return packageTargetConfig{Status: "unconfigured", Source: "selected-package-json", Problem: "The selected package.json has no explicit build." + platform + ".target; electron-builder defaults or other config may apply, so prerequisites are unknown."}
	}
	formats, err := parsePackageTarget(entry.Target)
	if err != nil || len(formats) == 0 {
		return packageTargetConfig{Status: "invalid", Source: "selected-package-json", Problem: "The selected package.json build." + platform + ".target has an unsupported shape; no per-app target was inferred."}
	}
	return packageTargetConfig{Formats: formats, Status: "ready", Source: "selected-package-json"}
}

func parsePackageTarget(raw json.RawMessage) ([]string, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	var values []json.RawMessage
	if len(raw) > 0 && raw[0] == '[' {
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
	} else {
		values = []json.RawMessage{raw}
	}
	formats := make([]string, 0, len(values))
	for _, value := range values {
		var name string
		if err := json.Unmarshal(value, &name); err == nil {
			name = strings.TrimSpace(name)
			if name == "" {
				return nil, errors.New("empty package target")
			}
			formats = append(formats, name)
			continue
		}
		var object struct {
			Target string `json:"target"`
		}
		if err := json.Unmarshal(value, &object); err != nil || strings.TrimSpace(object.Target) == "" {
			return nil, errors.New("unsupported package target")
		}
		formats = append(formats, strings.TrimSpace(object.Target))
	}
	return formats, nil
}

func renderBuildPrerequisitesHuman(ctx cliapp.RunContext, report buildPrerequisiteReport) error {
	lines := []string{
		fmt.Sprintf("Target: %s · build host: %s", report.TargetPlatform, report.BuildHost),
		fmt.Sprintf("Package formats: %s (config: %s)", strings.Join(report.PackageFormats, ", "), report.PackageConfigStatus),
	}
	for _, check := range report.Checks {
		line := fmt.Sprintf("%s: %s — %s", check.Tool, check.Status, check.Reason)
		if check.NextStep != "" {
			line += " Next: " + check.NextStep
		}
		lines = append(lines, line)
	}
	for _, limitation := range report.Limitations {
		lines = append(lines, "Limitation: "+limitation)
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status:    []string{strings.ToUpper(report.Status), report.Scope},
		Triage:    []cliapp.TriageGroup{{Heading: "Prerequisites", Items: lines}},
		NextSteps: []string{"scenario-to-desktop pipeline run <scenario> --platforms " + report.TargetPlatform},
	})
}

func normalizeBuildHost(goos string) string {
	switch strings.ToLower(strings.TrimSpace(goos)) {
	case "linux":
		return "linux"
	case "windows":
		return "win"
	case "darwin":
		return "mac"
	default:
		return strings.ToLower(strings.TrimSpace(goos))
	}
}
