package manifestschema

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type environmentContractManifest struct {
	Components map[string]struct {
		Run struct {
			Env map[string]string `json:"env"`
		} `json:"run"`
	} `json:"components"`
	Dependencies struct {
		Scenarios map[string]peerDependency `json:"scenarios"`
		Resources map[string]struct {
			Enabled  bool `json:"enabled"`
			Required bool `json:"required"`
		} `json:"resources"`
	} `json:"dependencies"`
}

func decodeEnvironmentContract(content []byte, filePath string) (environmentContractManifest, bool) {
	if !ShouldCheck(filePath) {
		return environmentContractManifest{}, false
	}
	var manifest environmentContractManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return environmentContractManifest{}, false
	}
	return manifest, true
}

func CheckScenarioHardcodedPeerAddress(content []byte, filePath string) []Violation {
	manifest, ok := decodeEnvironmentContract(content, filePath)
	if !ok || len(manifest.Dependencies.Scenarios) == 0 {
		return nil
	}
	var messages []string
	for componentName, component := range manifest.Components {
		for key, value := range component.Run.Env {
			if hasLoopbackPortLiteral(value) {
				messages = append(messages, fmt.Sprintf("component %q run.env.%s hardcodes a loopback peer address; resolve the peer through discovery", componentName, key))
			}
		}
	}
	return environmentContractViolations(filePath, "Scenario hardcodes a peer address", messages)
}

func CheckScenarioSecretLiteral(content []byte, filePath string) []Violation {
	manifest, ok := decodeEnvironmentContract(content, filePath)
	if !ok {
		return nil
	}
	var messages []string
	for componentName, component := range manifest.Components {
		for key, value := range component.Run.Env {
			upper := strings.ToUpper(strings.TrimSpace(key))
			if isSecretKey(upper) {
				messages = append(messages, fmt.Sprintf("component %q run.env.%s is a secret-bearing key; declare it through credentials", componentName, key))
				continue
			}
			if looksLikeHighEntropyLiteral(value) {
				messages = append(messages, fmt.Sprintf("component %q run.env.%s contains a high-entropy literal; declare it through credentials", componentName, key))
			}
		}
	}
	return environmentContractViolations(filePath, "Scenario contains a secret literal", messages)
}

func CheckScenarioRedeclaresResourceEnv(content []byte, filePath string) []Violation {
	manifest, ok := decodeEnvironmentContract(content, filePath)
	if !ok {
		return nil
	}
	repoRoot, ok := repositoryRootFromScenarioManifest(filePath)
	if !ok {
		return nil
	}
	exports := map[string]string{}
	for name, dependency := range manifest.Dependencies.Resources {
		if !dependency.Enabled && !dependency.Required {
			continue
		}
		for _, key := range loadResourceExportKeys(repoRoot, name) {
			exports[key] = name
		}
	}
	var messages []string
	for componentName, component := range manifest.Components {
		for key := range component.Run.Env {
			if resourceName, exists := exports[key]; exists {
				messages = append(messages, fmt.Sprintf("component %q run.env.%s redeclares value exported by resource %q", componentName, key, resourceName))
			}
		}
	}
	return environmentContractViolations(filePath, "Scenario redeclares a resource environment value", messages)
}

func hasLoopbackPortLiteral(value string) bool {
	for _, prefix := range []string{"localhost:", "127.0.0.1:"} {
		index := strings.Index(strings.ToLower(value), prefix)
		if index < 0 {
			continue
		}
		rest := value[index+len(prefix):]
		if rest != "" && rest[0] >= '0' && rest[0] <= '9' {
			return true
		}
	}
	return false
}

func isSecretKey(key string) bool {
	for _, suffix := range []string{"_PASSWORD", "_SECRET", "_TOKEN", "_KEY"} {
		if strings.HasSuffix(key, suffix) {
			return true
		}
	}
	return false
}

func looksLikeHighEntropyLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) < 24 || strings.ContainsAny(value, "$ {}") {
		return false
	}
	classes := 0
	hasLower, hasUpper, hasDigit, hasSymbol := false, false, false, false
	counts := map[rune]int{}
	for _, char := range value {
		counts[char]++
		switch {
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsDigit(char):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}
	for _, present := range []bool{hasLower, hasUpper, hasDigit, hasSymbol} {
		if present {
			classes++
		}
	}
	if classes < 3 {
		return false
	}
	entropy := 0.0
	for _, count := range counts {
		probability := float64(count) / float64(len([]rune(value)))
		entropy -= probability * math.Log2(probability)
	}
	return entropy >= 3.5
}

func loadResourceExportKeys(repoRoot, name string) []string {
	payload, err := os.ReadFile(filepath.Join(repoRoot, "resources", name, "resource.json"))
	if err != nil {
		return nil
	}
	var manifest struct {
		EnvironmentExports struct {
			Static         map[string]string `json:"static"`
			FromPorts      map[string]string `json:"from_ports"`
			FromRuntimeEnv []string          `json:"from_runtime_env"`
			Derived        map[string]any    `json:"derived"`
		} `json:"environment_exports"`
	}
	if json.Unmarshal(payload, &manifest) != nil {
		return nil
	}
	keys := make([]string, 0)
	for key := range manifest.EnvironmentExports.Static {
		keys = append(keys, key)
	}
	for key := range manifest.EnvironmentExports.FromPorts {
		keys = append(keys, key)
	}
	keys = append(keys, manifest.EnvironmentExports.FromRuntimeEnv...)
	for key := range manifest.EnvironmentExports.Derived {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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

func environmentContractViolations(filePath, title string, messages []string) []Violation {
	sort.Strings(messages)
	out := make([]Violation, 0, len(messages))
	for _, message := range messages {
		out = append(out, Violation{
			Type:           "scenario_environment_contract",
			Severity:       "high",
			Title:          title,
			Description:    message,
			FilePath:       filePath,
			LineNumber:     1,
			Recommendation: "Use discovery, resource exports, and credential descriptors as the environment authorities.",
			Standard:       "configuration-v1",
		})
	}
	return out
}
