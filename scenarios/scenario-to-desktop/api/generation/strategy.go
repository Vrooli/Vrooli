package generation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

const (
	DefaultStrategyID   = "electron-sidecar"
	OptimizedStrategyID = "electron-optimized"
	WailsStrategyID     = "wails-inprocess"
	SelectionPolicyV1   = "selection-policy.v1"
)

func PackagingStrategies() *deliveryramp.PackagingStrategyRegistry {
	registry, _ := deliveryramp.NewPackagingStrategyRegistry([]deliveryramp.PackagingStrategy{
		{ID: DefaultStrategyID, Version: "1", Ramp: "desktop", Shell: "electron", Topology: "sidecar", UITransport: "loopback-http", Platforms: []string{"linux-x64", "macos-x64"}, Default: true},
		{ID: OptimizedStrategyID, Version: "1", Ramp: "desktop", Shell: "electron", Topology: "sidecar", UITransport: "custom-scheme-or-loopback", Platforms: []string{"linux-x64", "macos-x64"}, Profile: "independently-toggleable"},
		{ID: WailsStrategyID, Version: "1", Ramp: "desktop", Shell: "wails", Topology: "in-process", UITransport: "asset-server-handler", Platforms: []string{"linux-x64", "macos-x64"}, Predicates: []deliveryramp.EligibilityPredicate{{Kind: "build.kind", Role: "api", Value: "go_module"}, {Kind: "language", Role: "api", Value: "go"}, {Kind: "embeddable.server", Role: "api", Value: "true"}}, Capabilities: deliveryramp.StrategyCapabilities{HostTools: []string{"webkitgtk", "webkit2gtk-driver"}}},
	})
	return registry
}

// ValidateSelectedStrategyHostTools enforces strategy-only host prerequisites
// at build preflight. It intentionally does not run during strategy discovery,
// so unavailable Wails tools do not make Electron ineligible.
func ValidateSelectedStrategyHostTools(strategyID, platform string) error {
	strategy, ok := PackagingStrategies().Resolve(strategyID)
	if !ok {
		return fmt.Errorf("unknown packaging strategy %q", strategyID)
	}
	if !strings.HasPrefix(strings.ToLower(platform), "linux") {
		return nil
	}
	for _, tool := range strategy.Capabilities.HostTools {
		switch tool {
		case "webkitgtk":
			if _, err := exec.LookPath("pkg-config"); err != nil {
				return fmt.Errorf("strategy %q requires host tool webkitgtk: pkg-config is unavailable", strategyID)
			}
			if err := exec.Command("pkg-config", "--exists", "gtk+-3.0", "webkit2gtk-4.1", "libsoup-3.0").Run(); err != nil {
				return fmt.Errorf("strategy %q requires host tool webkitgtk: pkg-config could not find gtk+-3.0, webkit2gtk-4.1, and libsoup-3.0", strategyID)
			}
		case "webkit2gtk-driver":
			if _, err := exec.LookPath("WebKitWebDriver"); err != nil {
				return fmt.Errorf("strategy %q requires host tool webkit2gtk-driver: WebKitWebDriver is unavailable", strategyID)
			}
		}
	}
	return nil
}

func StrategyForShell(shell string) (deliveryramp.PackagingStrategy, bool) {
	if strings.EqualFold(strings.TrimSpace(shell), "electron") {
		strategy, ok := PackagingStrategies().Resolve(DefaultStrategyID)
		return strategy, ok
	}
	if strings.EqualFold(strings.TrimSpace(shell), "wails") {
		strategy, ok := PackagingStrategies().Resolve(WailsStrategyID)
		return strategy, ok
	}
	for _, strategy := range PackagingStrategies().List() {
		if strategy.Shell == strings.TrimSpace(shell) {
			return strategy, true
		}
	}
	return deliveryramp.PackagingStrategy{}, false
}

type PackagingSelectionRecord = deliveryramp.PackagingSelectionRecord

func ResolvePackagingSelection(scenarioDir, override, platform string) (*PackagingSelectionRecord, error) {
	if platform == "" {
		platform = "linux-x64"
	}
	facts := ReadStackFacts(scenarioDir)
	registry := PackagingStrategies()
	eligible := registry.Evaluate(facts, platform)
	selection, err := deliveryramp.EvaluateSelection(deliveryramp.PackagingSelectionRequest{
		Policy:   deliveryramp.SelectionPolicy{Version: SelectionPolicyV1, DefaultStrategy: DefaultStrategyID},
		Eligible: eligible,
		Override: strings.TrimSpace(override),
	})
	if err != nil {
		return nil, err
	}
	return &PackagingSelectionRecord{PolicyVersion: SelectionPolicyV1, StackFacts: facts, Eligibility: eligible, Selected: selection}, nil
}

func ReadStackFacts(scenarioDir string) deliveryramp.StackFacts {
	path := filepath.Join(scenarioDir, ".vrooli", "service.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return deliveryramp.StackFacts{}
	}
	var manifest struct {
		Components map[string]struct {
			Role  string `json:"role"`
			Build struct {
				Kind string `json:"kind"`
				Dir  string `json:"dir"`
			} `json:"build"`
			Run struct {
				CWD string `json:"cwd"`
			} `json:"run"`
		} `json:"components"`
		Dependencies struct {
			Resources map[string]json.RawMessage `json:"resources"`
		} `json:"dependencies"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return deliveryramp.StackFacts{}
	}
	facts := deliveryramp.StackFacts{}
	for id, component := range manifest.Components {
		dir := component.Build.Dir
		if dir == "" {
			dir = component.Run.CWD
		}
		if dir == "" {
			dir = "."
		}
		componentDir := filepath.Join(scenarioDir, dir)
		language, framework, bundler := stackHints(componentDir)
		embedDescriptor := filepath.Join(componentDir, "desktop", "embed.go")
		_, embedErr := os.Stat(embedDescriptor)
		facts.Components = append(facts.Components, deliveryramp.StackComponent{ID: id, Role: component.Role, BuildKind: component.Build.Kind, Directory: dir, Language: language, Framework: framework, Bundler: bundler, EmbeddableServer: embedErr == nil, Evidence: []string{path, id, embedDescriptor}})
	}
	sort.Slice(facts.Components, func(i, j int) bool {
		if facts.Components[i].Role != facts.Components[j].Role {
			return facts.Components[i].Role < facts.Components[j].Role
		}
		return facts.Components[i].ID < facts.Components[j].ID
	})
	for id := range manifest.Dependencies.Resources {
		facts.Resources = append(facts.Resources, id)
	}
	sort.Strings(facts.Resources)
	return facts
}

func stackHints(dir string) (language, framework, bundler string) {
	for _, candidate := range []struct{ name, value string }{{"go.mod", "go"}, {"Cargo.toml", "rust"}, {"pyproject.toml", "python"}, {"tsconfig.json", "typescript"}} {
		if _, err := os.Stat(filepath.Join(dir, candidate.name)); err == nil {
			language = candidate.value
			break
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}
	deps := map[string]bool{}
	for n := range pkg.Dependencies {
		deps[n] = true
	}
	for n := range pkg.DevDependencies {
		deps[n] = true
	}
	for _, candidate := range [][2]string{{"react", "react"}, {"vue", "vue"}, {"svelte", "svelte"}, {"@angular/core", "angular"}} {
		if deps[candidate[0]] {
			framework = candidate[1]
			break
		}
	}
	for _, candidate := range [][2]string{{"vite", "vite"}, {"webpack", "webpack"}, {"esbuild", "esbuild"}, {"parcel", "parcel"}} {
		if deps[candidate[0]] {
			bundler = candidate[1]
			break
		}
	}
	if language == "" {
		language = "javascript"
	}
	return
}

func StrategyOutputName(id string) (string, error) {
	if id == "electron" {
		id = DefaultStrategyID
	}
	strategy, ok := PackagingStrategies().Resolve(id)
	if !ok {
		return "", fmt.Errorf("unknown packaging strategy %q", id)
	}
	// Preserve the existing canonical output directory for the baseline.
	if strategy.ID == DefaultStrategyID {
		return "electron", nil
	}
	return strategy.ID, nil
}
