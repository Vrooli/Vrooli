package dependencyhealth

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

// evaluateIntegrationConformance checks authored service.json integrations
// and ensures optional capability registries do not duplicate them. A running
// target is authoritative for its optional capability description: it proves
// that the handler is wired and that Registry.Describe passed its own
// checker/state validation. Stopped or generated targets use the source
// contract as a deterministic fallback so the gate remains useful offline.
func (h *connectHandler) evaluateIntegrationConformance(ctx context.Context, scenario string) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DegradedDependency) {
	manifest, err := loadDependencyManifest(filepath.Join(h.scenarioDir(ctx, scenario), ".vrooli", "service.json"))
	if err != nil {
		return section("integration_conformance", "Integration conformance", "degraded", "Integration declarations could not be read."), nil, nil
	}
	if len(manifest.Dependencies.Scenarios) == 0 && len(manifest.Dependencies.Resources) == 0 {
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
	var sourceDefinitions []registryDefinition
	if !runtimeVerified {
		registryDir := filepath.Join(h.scenarioDir(ctx, scenario), "api", "internal", "capabilities")
		sourceDefinitions = readRegistryDefinitions(registryDir)
	}

	var findings []*healthv1.DependencyHealthFinding
	manifestDependencies := make(map[string]struct{}, len(manifest.Dependencies.Resources)+len(manifest.Dependencies.Scenarios))
	for slug := range manifest.Dependencies.Resources {
		manifestDependencies["resource:"+slug] = struct{}{}
	}
	for slug := range manifest.Dependencies.Scenarios {
		manifestDependencies["scenario:"+slug] = struct{}{}
	}
	definitions := sourceDefinitions
	if runtimeVerified {
		definitions = evidence.Definitions
	}
	for _, definition := range definitions {
		key := strings.TrimSpace(definition.DependencyKind) + ":" + strings.TrimSpace(definition.DependencySlug)
		if _, duplicate := manifestDependencies[key]; duplicate {
			findings = append(findings, integrationFinding(
				"INTEGRATION_REGISTRY_DUPLICATES_MANIFEST",
				"Capability registry duplicates a manifest dependency",
				fmt.Sprintf("Registry entry %q repeats a dependency already authored in service.json.", definition.DependencySlug),
				"capability entry with no manifest dependency duplicate",
				definition.DependencySlug,
			))
		}
	}
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

		if strings.TrimSpace(declaration.DegradedBehavior) == "" {
			findings = append(findings, integrationFinding("INTEGRATION_DEGRADED_BEHAVIOR_UNDECLARED", "Degraded behavior is not declared", fmt.Sprintf("Dependency %q has no actionable degraded_behavior declaration.", slug), "non-empty degraded_behavior", slug))
		}
	}
	status := statusFromFindings(findings, "integration_conformance")
	return sectionWithFindingIDs("integration_conformance", "Integration conformance", status, fmt.Sprintf("Checked %d declared scenario integration(s) and %d resource integration(s) against service.json using %s registry evidence.", len(manifest.Dependencies.Scenarios), len(manifest.Dependencies.Resources), evidenceSource(runtimeVerified)), findingIDs(findings, "integration")), findings, nil
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

// readRegistryDefinitions extracts dependency-bearing struct literals from a
// source fallback registry. It uses the AST so comments, examples, checker
// commands, and unrelated strings cannot satisfy or evade the no-duplicate
// contract when the scenario is not running.
func readRegistryDefinitions(dir string) []registryDefinition {
	var definitions []registryDefinition
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, data, 0)
		if parseErr != nil {
			return nil
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			definition := registryDefinition{}
			for _, element := range literal.Elts {
				field, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, keyOK := field.Key.(*ast.Ident)
				value, valueOK := field.Value.(*ast.BasicLit)
				if !keyOK || !valueOK || value.Kind != token.STRING {
					continue
				}
				decoded, unquoteErr := strconv.Unquote(value.Value)
				if unquoteErr != nil {
					continue
				}
				switch key.Name {
				case "DependencyKind":
					definition.DependencyKind = decoded
				case "DependencySlug":
					definition.DependencySlug = decoded
				}
			}
			if definition.DependencyKind != "" && definition.DependencySlug != "" {
				definitions = append(definitions, definition)
			}
			return true
		})
		return nil
	})
	return definitions
}

func integrationFinding(rule, title, description, expected, observed string) *healthv1.DependencyHealthFinding {
	severity := "ERROR"
	if rule == "INTEGRATION_NO_OPERATOR_ACTION" || rule == "INTEGRATION_DEGRADED_BEHAVIOR_UNDECLARED" {
		severity = "WARNING"
	}
	remediation := "Repair the authored service.json integration or rerun the dependencies phase."
	if rule == "INTEGRATION_REGISTRY_DUPLICATES_MANIFEST" {
		remediation = "Remove the duplicated dependency from the optional capability registry and rerun the dependencies phase."
	}
	return &healthv1.DependencyHealthFinding{Id: strings.ToLower(strings.ReplaceAll(rule, "_", "-")), Severity: severity, SourceDomain: "integration_conformance", Title: title, Description: description, Remediation: remediation, RuleId: rule, Observed: observed, Expected: expected}
}
