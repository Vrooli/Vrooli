package manifestschema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var knownBuildKinds = map[string]bool{
	"go_module":   true,
	"pnpm_vite":   true,
	"node_bundle": true,
	"reuse":       true,
	"python_uv":   false,
	"cargo":       false,
}

type componentManifest struct {
	Ports        map[string]json.RawMessage     `json:"ports"`
	Components   map[string]componentDefinition `json:"components"`
	Dependencies struct {
		Scenarios map[string]peerDependency `json:"scenarios"`
	} `json:"dependencies"`
}

type componentDefinition struct {
	Role  string `json:"role"`
	Build struct {
		Kind  string `json:"kind"`
		Reuse string `json:"reuse"`
	} `json:"build"`
	Run struct {
		Argv         []string `json:"argv"`
		Port         string   `json:"port"`
		SupervisedBy string   `json:"supervised_by"`
		DependsOn    []struct {
			Component string `json:"component"`
		} `json:"depends_on"`
	} `json:"run"`
}

func CheckScenarioUIServesBuild(content []byte, filePath string) []Violation {
	manifest, ok := decodeComponentManifest(content, filePath)
	if !ok {
		return nil
	}
	var messages []string
	for name, component := range manifest.Components {
		if component.Role != "ui" {
			continue
		}
		command := strings.ToLower(strings.Join(component.Run.Argv, " "))
		for _, forbidden := range []string{"vite", "npm run dev", "pnpm run dev", "npm run preview", "pnpm run preview"} {
			if strings.Contains(command, forbidden) {
				messages = append(messages, fmt.Sprintf("UI component %q runs development server %q instead of serving its production build", name, forbidden))
				break
			}
		}
	}
	return componentViolations(filePath, "Scenario UI does not serve its production build", messages)
}

type peerDependency struct {
	StartupPolicy    string        `json:"startup_policy"`
	DegradedBehavior string        `json:"degraded_behavior"`
	Bindings         []peerBinding `json:"bindings"`
}

type peerBinding struct {
	EnvVar          string `json:"env_var"`
	Port            string `json:"port"`
	WhenUnavailable string `json:"when_unavailable"`
}

// CheckScenarioComponents validates references inside one component graph.
// It deliberately does not require components yet; Phase 5 promotes adoption.
func CheckScenarioComponents(content []byte, filePath string) []Violation {
	manifest, ok := decodeComponentManifest(content, filePath)
	if !ok {
		return nil
	}
	if len(manifest.Components) == 0 {
		return componentViolations(filePath, "Scenario component contract is invalid", []string{"scenario must declare at least one component"})
	}

	var messages []string
	graph := make(map[string][]string, len(manifest.Components))
	for name, component := range manifest.Components {
		kind := strings.TrimSpace(component.Build.Kind)
		reuse := strings.TrimSpace(component.Build.Reuse)
		switch {
		case kind != "" && reuse != "":
			messages = append(messages, fmt.Sprintf("component %q build must declare exactly one of kind or reuse", name))
		case kind == "" && reuse == "":
			messages = append(messages, fmt.Sprintf("component %q build must declare kind or reuse", name))
		case kind != "":
			if implemented, known := knownBuildKinds[kind]; !known || !implemented {
				messages = append(messages, fmt.Sprintf("component %q build kind %q is not executable", name, kind))
			}
		case reuse != "":
			if _, exists := manifest.Components[reuse]; !exists {
				messages = append(messages, fmt.Sprintf("component %q reuses missing component %q", name, reuse))
			} else {
				graph[name] = append(graph[name], reuse)
			}
		}

		if port := strings.TrimSpace(component.Run.Port); port != "" {
			if _, exists := manifest.Ports[port]; !exists {
				messages = append(messages, fmt.Sprintf("component %q run.port names missing ports key %q", name, port))
			}
		}
		if supervisor := strings.TrimSpace(component.Run.SupervisedBy); supervisor != "" {
			if _, exists := manifest.Components[supervisor]; !exists {
				messages = append(messages, fmt.Sprintf("component %q supervised_by names missing component %q", name, supervisor))
			} else {
				graph[name] = append(graph[name], supervisor)
			}
		}
		for _, dependency := range component.Run.DependsOn {
			target := strings.TrimSpace(dependency.Component)
			if _, exists := manifest.Components[target]; !exists {
				messages = append(messages, fmt.Sprintf("component %q depends_on names missing component %q", name, target))
				continue
			}
			graph[name] = append(graph[name], target)
		}
	}
	if cycle := componentCycle(graph); len(cycle) > 0 {
		messages = append(messages, "component dependency cycle: "+strings.Join(cycle, " -> "))
	}
	return componentViolations(filePath, "Scenario component contract is invalid", messages)
}

// CheckScenarioPeerBindings validates each edge against the peer manifest it
// names. This is intentionally cross-manifest: a local schema cannot prove the
// referenced port exists on the peer.
func CheckScenarioPeerBindings(content []byte, filePath string) []Violation {
	manifest, ok := decodeComponentManifest(content, filePath)
	if !ok {
		return nil
	}
	var messages []string
	peerNames := make([]string, 0, len(manifest.Dependencies.Scenarios))
	for name := range manifest.Dependencies.Scenarios {
		peerNames = append(peerNames, name)
	}
	sort.Strings(peerNames)
	for _, peerName := range peerNames {
		dependency := manifest.Dependencies.Scenarios[peerName]
		if len(dependency.Bindings) == 0 {
			continue
		}
		for index, binding := range dependency.Bindings {
			switch strings.TrimSpace(binding.WhenUnavailable) {
			case "fail":
				if strings.TrimSpace(dependency.StartupPolicy) != "must_start" {
					messages = append(messages, fmt.Sprintf("peer %q binding[%d] uses when_unavailable=fail without startup_policy=must_start", peerName, index))
				}
			case "omit":
				if strings.TrimSpace(dependency.DegradedBehavior) == "" {
					messages = append(messages, fmt.Sprintf("peer %q binding[%d] uses when_unavailable=omit without degraded_behavior", peerName, index))
				}
			}
		}
		peerPorts, err := loadPeerPorts(filePath, peerName)
		if err != nil {
			messages = append(messages, fmt.Sprintf("peer %q bindings cannot be checked: %v", peerName, err))
			continue
		}
		for index, binding := range dependency.Bindings {
			if _, exists := peerPorts[strings.TrimSpace(binding.Port)]; !exists {
				messages = append(messages, fmt.Sprintf("peer %q binding[%d] port %q is not declared by that peer", peerName, index, binding.Port))
			}
		}
	}
	return componentViolations(filePath, "Scenario peer binding is invalid", messages)
}

func CheckScenarioBuildKinds(content []byte, filePath string) []Violation {
	manifest, ok := decodeComponentManifest(content, filePath)
	if !ok {
		return nil
	}
	var messages []string
	for name, component := range manifest.Components {
		kind := strings.TrimSpace(component.Build.Kind)
		if kind == "" {
			continue
		}
		if _, known := knownBuildKinds[kind]; !known {
			messages = append(messages, fmt.Sprintf("component %q declares unknown build kind %q", name, kind))
		}
	}
	return componentViolations(filePath, "Scenario build kind is unknown", messages)
}

func decodeComponentManifest(content []byte, filePath string) (componentManifest, bool) {
	if !ShouldCheck(filePath) {
		return componentManifest{}, false
	}
	var manifest componentManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return componentManifest{}, false
	}
	return manifest, true
}

func componentViolations(filePath, title string, messages []string) []Violation {
	sort.Strings(messages)
	out := make([]Violation, 0, len(messages))
	for _, message := range messages {
		out = append(out, Violation{
			Type:           "component_contract",
			Severity:       "warning",
			Title:          title,
			Description:    message,
			FilePath:       filePath,
			LineNumber:     1,
			Recommendation: "Repair the component or peer-binding declaration in .vrooli/service.json.",
			Standard:       "configuration-v1",
		})
	}
	return out
}

func loadPeerPorts(servicePath, peerName string) (map[string]json.RawMessage, error) {
	repoRoot, ok := repositoryRootFromScenarioManifest(servicePath)
	if !ok {
		return nil, fmt.Errorf("cannot locate repository root from %s", servicePath)
	}
	peerPath := filepath.Join(repoRoot, "scenarios", peerName, ".vrooli", "service.json")
	raw, err := os.ReadFile(peerPath)
	if err != nil {
		return nil, err
	}
	var peer struct {
		Ports map[string]json.RawMessage `json:"ports"`
	}
	if err := json.Unmarshal(raw, &peer); err != nil {
		return nil, err
	}
	return peer.Ports, nil
}

func repositoryRootFromScenarioManifest(servicePath string) (string, bool) {
	dir := filepath.Clean(filepath.Dir(servicePath))
	for {
		if filepath.Base(dir) == "scenarios" {
			return filepath.Dir(dir), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func componentCycle(graph map[string][]string) []string {
	state := make(map[string]uint8, len(graph))
	stack := make([]string, 0, len(graph))
	var visit func(string) []string
	visit = func(node string) []string {
		switch state[node] {
		case 1:
			for index, candidate := range stack {
				if candidate == node {
					return append(append([]string(nil), stack[index:]...), node)
				}
			}
			return []string{node, node}
		case 2:
			return nil
		}
		state[node] = 1
		stack = append(stack, node)
		neighbors := append([]string(nil), graph[node]...)
		sort.Strings(neighbors)
		for _, neighbor := range neighbors {
			if cycle := visit(neighbor); len(cycle) > 0 {
				return cycle
			}
		}
		stack = stack[:len(stack)-1]
		state[node] = 2
		return nil
	}
	nodes := make([]string, 0, len(graph))
	for node := range graph {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if cycle := visit(node); len(cycle) > 0 {
			return cycle
		}
	}
	return nil
}
