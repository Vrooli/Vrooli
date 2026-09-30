package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
)

// FleetEntry is the machine-readable result for one execution declaration.
type FleetEntry struct {
	Resource     string   `json:"resource"`
	RunnerType   string   `json:"runnerType,omitempty"`
	Codec        string   `json:"codec,omitempty"`
	InstallOnly  bool     `json:"installOnly"`
	Conforming   bool     `json:"conforming"`
	FindingCodes []string `json:"findingCodes,omitempty"`
}

// FleetReport is the read-only coding-agent fleet contract result. It never
// reconciles manifests, starts agents, or touches the host filesystem beyond
// reading resource.json files.
type FleetReport struct {
	Entries  []FleetEntry `json:"entries"`
	Findings []Finding    `json:"findings,omitempty"`
}

type fleetExecution struct {
	RunnerType      string `json:"runner_type"`
	Codec           string `json:"codec"`
	DetectionSignal string `json:"detection_signal"`
	HookSurface     string `json:"hook_surface"`
	InstallOnly     bool   `json:"install_only"`
	Rationale       string `json:"rationale"`
}

type fleetManifest struct {
	Name      string          `json:"name"`
	Execution *fleetExecution `json:"execution"`
}

// ValidateFleet reads every resource execution declaration and compares the
// declared runner/codec pair to the domain and codec registries. Resources
// without an execution block are ordinary non-agent resources and are not
// subjects. An install_only declaration is the explicit escape hatch for a
// resource such as Ollama that supplies a runner but is not itself a runner.
func ValidateFleet(repoRoot string) (FleetReport, error) {
	root := filepath.Join(repoRoot, "resources")
	entries, err := os.ReadDir(root)
	if err != nil {
		return FleetReport{}, fmt.Errorf("read resources: %w", err)
	}

	knownRunners := make(map[string]struct{}, len(domain.ValidRunnerTypes()))
	for _, runnerType := range domain.ValidRunnerTypes() {
		knownRunners[string(runnerType)] = struct{}{}
	}
	knownCodecs := map[string]string{
		"claude":      string(codecs.NewClaudeForTest().Type()),
		"codex":       string(codecs.NewCodexForTest().Type()),
		"opencode":    string(codecs.NewOpenCodeForTest().Type()),
		"grok":        string(codecs.NewGrokForTest().Type()),
		"antigravity": string(codecs.NewAntigravityForTest().Type()),
	}

	report := FleetReport{Entries: []FleetEntry{}}
	declaredRunners := map[string]string{}
	declaredCodecs := map[string]string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(root, entry.Name(), "resource.json")
		contents, readErr := os.ReadFile(manifestPath)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return FleetReport{}, fmt.Errorf("read %s: %w", manifestPath, readErr)
		}
		var manifest fleetManifest
		if err := json.Unmarshal(contents, &manifest); err != nil {
			return FleetReport{}, fmt.Errorf("parse %s: %w", manifestPath, err)
		}
		if manifest.Execution == nil {
			continue
		}
		contract := manifest.Execution
		fleetEntry := FleetEntry{Resource: entry.Name(), RunnerType: contract.RunnerType, Codec: contract.Codec, InstallOnly: contract.InstallOnly, Conforming: true}
		addFinding := func(code, title, message string) {
			fleetEntry.Conforming = false
			fleetEntry.FindingCodes = append(fleetEntry.FindingCodes, code)
			report.Findings = append(report.Findings, Finding{Code: code, Title: title, Message: message, Location: manifestPath, Severity: "error"})
		}
		if contract.InstallOnly {
			if strings.TrimSpace(contract.Rationale) == "" {
				addFinding("agent_fleet.install_only_rationale_missing", "Install-only execution declaration has no rationale", "Add execution.rationale explaining why this resource is not an agent-manager runner.")
			}
			if contract.RunnerType != "" || contract.Codec != "" || contract.DetectionSignal != "" || contract.HookSurface != "" {
				addFinding("agent_fleet.install_only_fields_present", "Install-only resource declares runnable fields", "Remove runner_type, codec, detection_signal, and hook_surface from an install-only declaration.")
			}
		} else {
			if contract.RunnerType == "" {
				addFinding("agent_fleet.runner_type_missing", "Execution declaration has no runner type", "Declare the domain RunnerType implemented by this coding-agent resource.")
			} else if _, ok := knownRunners[contract.RunnerType]; !ok {
				addFinding("agent_fleet.runner_type_unknown", "Execution declaration names an unknown runner type", fmt.Sprintf("Runner type %q is not present in domain.ValidRunnerTypes().", contract.RunnerType))
			} else if previous, exists := declaredRunners[contract.RunnerType]; exists {
				addFinding("agent_fleet.runner_type_duplicate", "Runner type is declared by multiple resources", fmt.Sprintf("Runner type %q is already declared by %s.", contract.RunnerType, previous))
			} else {
				declaredRunners[contract.RunnerType] = entry.Name()
			}
			if contract.Codec == "" {
				addFinding("agent_fleet.codec_missing", "Execution declaration has no codec", "Declare the codec that translates this harness's output and controls.")
			} else if runnerType, ok := knownCodecs[contract.Codec]; !ok {
				addFinding("agent_fleet.codec_unknown", "Execution declaration names an unknown codec", fmt.Sprintf("Codec %q is not registered in the codec catalog.", contract.Codec))
			} else if contract.RunnerType != "" && runnerType != contract.RunnerType {
				addFinding("agent_fleet.codec_runner_mismatch", "Codec and runner type disagree", fmt.Sprintf("Codec %q implements %q, not %q.", contract.Codec, runnerType, contract.RunnerType))
			} else if previous, exists := declaredCodecs[contract.Codec]; exists {
				addFinding("agent_fleet.codec_duplicate", "Codec is declared by multiple resources", fmt.Sprintf("Codec %q is already declared by %s.", contract.Codec, previous))
			} else {
				declaredCodecs[contract.Codec] = entry.Name()
			}
			if strings.TrimSpace(contract.DetectionSignal) == "" {
				addFinding("agent_fleet.detection_signal_missing", "Execution declaration has no detection signal", "Declare the runtime-self environment signal documented in agent-detection-signals.md.")
			}
			if strings.TrimSpace(contract.HookSurface) == "" {
				addFinding("agent_fleet.hook_surface_missing", "Execution declaration has no hook surface", "Declare the native or broker-backed hook/permission surface for this agent.")
			}
		}
		sort.Strings(fleetEntry.FindingCodes)
		report.Entries = append(report.Entries, fleetEntry)
	}

	for runnerType := range knownRunners {
		if _, ok := declaredRunners[runnerType]; !ok {
			report.Findings = append(report.Findings, Finding{Code: "agent_fleet.resource_missing", Title: "Runner type has no resource declaration", Message: fmt.Sprintf("Runner type %q has no matching coding-agent resource.json execution declaration.", runnerType), Location: "resources/*/resource.json", Severity: "error"})
		}
	}
	for codecName, runnerType := range knownCodecs {
		if _, ok := declaredCodecs[codecName]; !ok {
			report.Findings = append(report.Findings, Finding{Code: "agent_fleet.codec_resource_missing", Title: "Codec has no resource declaration", Message: fmt.Sprintf("Codec %q for runner %q has no matching resource.json execution declaration.", codecName, runnerType), Location: "resources/*/resource.json", Severity: "error"})
		}
	}
	sort.Slice(report.Entries, func(i, j int) bool { return report.Entries[i].Resource < report.Entries[j].Resource })
	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Code != report.Findings[j].Code {
			return report.Findings[i].Code < report.Findings[j].Code
		}
		return report.Findings[i].Location < report.Findings[j].Location
	})
	return report, nil
}

// FleetConforms is the gate-shaped convenience used by tests and lifecycle
// checks that only need a boolean verdict.
func FleetConforms(repoRoot string) (FleetReport, error) {
	report, err := ValidateFleet(repoRoot)
	if err != nil {
		return FleetReport{}, err
	}
	if len(report.Findings) > 0 {
		return report, fmt.Errorf("coding-agent fleet has %d conformance finding(s)", len(report.Findings))
	}
	return report, nil
}
