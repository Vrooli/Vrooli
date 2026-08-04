package dependencyhealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

// registryDescription is the stable JSON contract emitted by each consumer's
// capability handler. Keeping this DTO here avoids importing a consumer's Go
// package into the generic analyzer.
type registryDescription struct {
	Definitions []registryDefinition `json:"definitions"`
	States      []registryState      `json:"states"`
}

type registryDefinition struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	DependencyKind  string `json:"dependencyKind"`
	DependencySlug  string `json:"dependencySlug"`
	ActionKind      string `json:"actionKind"`
	ActionLabel     string `json:"actionLabel"`
	OperatorCommand string `json:"operatorCommand"`
}

type registryState struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	ActionKind      string `json:"actionKind"`
	ActionLabel     string `json:"actionLabel"`
	OperatorCommand string `json:"operatorCommand"`
}

// evaluateIntegrationConformance checks the declared scenario graph against
// the registry's public description. A running target is authoritative: it
// proves that the handler is wired and that Registry.Describe passed its own
// checker/state validation. Stopped or generated targets use the source
// contract as a deterministic fallback so the gate remains useful offline.
func (h *connectHandler) evaluateIntegrationConformance(ctx context.Context, scenario string) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DegradedDependency) {
	manifest, err := loadDependencyManifest(filepath.Join(h.scenarioDir(ctx, scenario), ".vrooli", "service.json"))
	if err != nil {
		return section("integration_conformance", "Integration conformance", "degraded", "Integration declarations could not be read."), nil, nil
	}
	if len(manifest.Dependencies.Scenarios) == 0 {
		return section("integration_conformance", "Integration conformance", "not_applicable", "No scenario dependency is declared."), nil, nil
	}

	var evidence registryDescription
	runtimeVerified := false
	if scenarioPathFrom(ctx) == "" {
		if data, fetchErr := fetchRuntimeRegistryDescription(ctx, scenario); fetchErr == nil {
			if jsonErr := json.Unmarshal(data, &evidence); jsonErr == nil {
				runtimeVerified = true
			}
		}
	}
	registryText := ""
	if !runtimeVerified {
		registryDir := filepath.Join(h.scenarioDir(ctx, scenario), "api", "internal", "capabilities")
		registryText = readRegistryText(registryDir)
	}

	var findings []*healthv1.DependencyHealthFinding
	scenariosRoot := h.resolveScenariosDir()
	if explicit := scenarioPathFrom(ctx); explicit != "" {
		// Deep validation may inspect a generated scenario outside the repository
		// scenarios tree; its sibling directories are the dependency universe.
		scenariosRoot = filepath.Dir(explicit)
	}
	for slug, declaration := range manifest.Dependencies.Scenarios {
		if _, statErr := os.Stat(filepath.Join(scenariosRoot, slug)); statErr != nil {
			findings = append(findings, integrationFinding("INTEGRATION_DEPENDENCY_UNRESOLVED", "Scenario dependency does not resolve", fmt.Sprintf("The declared scenario dependency %q is not present locally.", slug), "local scenario directory", slug))
			continue
		}

		var def registryDefinition
		var state registryState
		found := false
		if runtimeVerified {
			for _, candidate := range evidence.Definitions {
				if candidate.DependencySlug == slug || candidate.ID == slug {
					def = candidate
					found = true
					break
				}
			}
			for _, candidate := range evidence.States {
				if candidate.ID == def.ID || candidate.ID == slug {
					state = candidate
					break
				}
			}
		} else {
			found = strings.Contains(registryText, slug)
		}
		if !found {
			findings = append(findings, integrationFinding("INTEGRATION_REGISTRY_ENTRY_MISSING", "Capability registry entry missing", fmt.Sprintf("The scenario declares %q but its capability registry does not mention that dependency.", slug), "registry definition for dependency", slug))
			continue
		}

		if runtimeVerified {
			if strings.TrimSpace(def.Description) == "" || (def.DependencyKind != "scenario" && def.DependencyKind != "resource") || def.DependencySlug != slug {
				findings = append(findings, integrationFinding("INTEGRATION_REGISTRY_INCOMPLETE", "Capability registry entry incomplete", fmt.Sprintf("The runtime registry entry for %q does not expose a valid description, kind, and matching slug.", slug), "description, dependency kind, and matching slug", slug))
			}
			if strings.TrimSpace(def.ActionKind) == "" && strings.TrimSpace(state.ActionKind) == "" {
				findings = append(findings, integrationFinding("INTEGRATION_NO_OPERATOR_ACTION", "Operator action is not declared", fmt.Sprintf("Registry entry %q has no operator recovery action.", slug), "operator action metadata", slug))
			}
		} else {
			if !strings.Contains(registryText, "Description") || !strings.Contains(registryText, "DependencySlug") || (!strings.Contains(registryText, "Checker") && !strings.Contains(registryText, "Check(")) {
				findings = append(findings, integrationFinding("INTEGRATION_REGISTRY_INCOMPLETE", "Capability registry entry incomplete", fmt.Sprintf("The registry entry for %q does not expose a description, matching dependency slug, and reachability checker.", slug), "description, dependency kind, matching slug, and reachability checker", slug))
			}
			if !strings.Contains(registryText, "ActionKind") && !strings.Contains(registryText, "operator_command") {
				findings = append(findings, integrationFinding("INTEGRATION_NO_OPERATOR_ACTION", "Operator action is not declared", fmt.Sprintf("Registry entry %q has no operator recovery action.", slug), "operator action metadata", slug))
			}
		}
		if strings.TrimSpace(declaration.DegradedBehavior) == "" {
			findings = append(findings, integrationFinding("INTEGRATION_DEGRADED_BEHAVIOR_UNDECLARED", "Degraded behavior is not declared", fmt.Sprintf("Dependency %q has no actionable degraded_behavior declaration.", slug), "non-empty degraded_behavior", slug))
		}
	}
	status := statusFromFindings(findings, "integration_conformance")
	return sectionWithFindingIDs("integration_conformance", "Integration conformance", status, fmt.Sprintf("Checked %d declared scenario integration(s) using %s registry evidence.", len(manifest.Dependencies.Scenarios), evidenceSource(runtimeVerified)), findingIDs(findings, "integration")), findings, nil
}

func evidenceSource(runtimeVerified bool) string {
	if runtimeVerified {
		return "runtime"
	}
	return "source fallback"
}

func fetchRuntimeRegistryDescription(ctx context.Context, scenario string) ([]byte, error) {
	statusCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(statusCtx, "vrooli", "scenario", "status", scenario, "--json").Output()
	if err != nil {
		return nil, err
	}
	var payload struct {
		Scenario struct {
			Ports []struct {
				Key  string `json:"key"`
				Port int    `json:"port"`
			} `json:"ports"`
		} `json:"scenario"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, err
	}
	var apiPort int
	for _, port := range payload.Scenario.Ports {
		if strings.EqualFold(port.Key, "API_PORT") {
			apiPort = port.Port
			break
		}
	}
	if apiPort == 0 {
		return nil, fmt.Errorf("scenario %q has no API_PORT", scenario)
	}
	requestCtx, cancelRequest := context.WithTimeout(ctx, 3*time.Second)
	defer cancelRequest()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/capabilities/describe", apiPort), nil)
	if err != nil {
		return nil, err
	}
	resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("capability description returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func readRegistryText(dir string) string {
	var b strings.Builder
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil {
			// Source fallback is structural evidence, not a text grep. Render
			// the parsed AST without comments so a stale example or comment
			// mentioning a dependency cannot satisfy the registry gate.
			fileSet := token.NewFileSet()
			file, parseErr := parser.ParseFile(fileSet, path, data, 0)
			if parseErr == nil {
				var rendered bytes.Buffer
				if formatErr := format.Node(&rendered, fileSet, file); formatErr == nil {
					b.Write(rendered.Bytes())
					b.WriteByte('\n')
					return nil
				}
			}
			// Preserve deterministic fallback behavior for a source file the
			// parser cannot understand; malformed source is separately reported
			// by dependency health and must not make this reader panic.
			b.Write(data)
			b.WriteByte('\n')
		}
		return nil
	})
	return b.String()
}

func integrationFinding(rule, title, description, expected, observed string) *healthv1.DependencyHealthFinding {
	severity := "ERROR"
	if rule == "INTEGRATION_NO_OPERATOR_ACTION" || rule == "INTEGRATION_DEGRADED_BEHAVIOR_UNDECLARED" {
		severity = "WARNING"
	}
	return &healthv1.DependencyHealthFinding{Id: strings.ToLower(strings.ReplaceAll(rule, "_", "-")), Severity: severity, SourceDomain: "integration_conformance", Title: title, Description: description, Remediation: "Add or repair the shared capability-registry entry and rerun the dependencies phase.", RuleId: rule, Observed: observed, Expected: expected}
}
