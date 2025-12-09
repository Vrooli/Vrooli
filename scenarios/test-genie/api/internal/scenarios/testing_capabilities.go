package scenarios

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// TestingCapabilities captures how to run tests for a scenario using Go-native or scenario-local entrypoints.
type TestingCapabilities struct {
	Genie     bool             `json:"genie"`
	HasTests  bool             `json:"hasTests"`
	Phased    bool             `json:"phased"`
	Lifecycle bool             `json:"lifecycle"`
	Legacy    bool             `json:"legacy"`
	Preferred string           `json:"preferred,omitempty"`
	Commands  []TestingCommand `json:"commands,omitempty"`
}

// TestingCommand captures how to run a particular scenario test mode.
type TestingCommand struct {
	Type        string   `json:"type"`
	Command     []string `json:"command"`
	WorkingDir  string   `json:"workingDir,omitempty"`
	Description string   `json:"description,omitempty"`
}

// DetectTestingCapabilities inspects a scenario directory and reports which testing entrypoints exist.
func DetectTestingCapabilities(scenarioDir string) TestingCapabilities {
	if strings.TrimSpace(scenarioDir) == "" {
		return TestingCapabilities{}
	}
	caps := TestingCapabilities{}
	appRoot := projectRootFromScenario(scenarioDir)
	scenarioName := filepath.Base(scenarioDir)

	if cmd := detectTestGenieCommand(); len(cmd) > 0 && appRoot != "" {
		caps.Genie = true
		caps.Commands = append(caps.Commands, TestingCommand{
			Type:        "genie",
			Command:     append(cmd, "execute", scenarioName, "--preset", "smoke"),
			WorkingDir:  appRoot,
			Description: "Runs the Go-native test-genie orchestrator (smoke preset).",
		})
	}
	if hasExecutable(filepath.Join(scenarioDir, "test", "run-tests.sh")) {
		caps.Phased = true
		caps.Commands = append(caps.Commands, TestingCommand{
			Type:        "phased",
			Command:     []string{"./test/run-tests.sh", "--quick"},
			WorkingDir:  scenarioDir,
			Description: "Runs the scenario-local phased suite (new format).",
		})
	}
	if hasLifecycleTest(filepath.Join(scenarioDir, ".vrooli", "service.json")) {
		caps.Lifecycle = true
		if appRoot != "" {
			caps.Commands = append(caps.Commands, TestingCommand{
				Type:        "lifecycle",
				Command:     []string{filepath.Join(appRoot, "scripts", "manage.sh"), "test"},
				WorkingDir:  scenarioDir,
				Description: "Runs lifecycle-managed tests declared in service.json.",
			})
		}
	}
	if fileExists(filepath.Join(scenarioDir, "scenario-test.yaml")) {
		caps.Legacy = true
		if appRoot != "" {
			caps.Commands = append(caps.Commands, TestingCommand{
				Type:        "legacy",
				Command:     []string{"vrooli", "test", "scenarios", "--scenario", scenarioName},
				WorkingDir:  appRoot,
				Description: "Runs legacy scenario-test.yaml harness via vrooli CLI (consider migrating).",
			})
		}
	}
	caps.HasTests = caps.Genie || caps.Phased || caps.Lifecycle || caps.Legacy
	switch {
	case caps.Genie:
		caps.Preferred = "genie"
	case caps.Phased:
		caps.Preferred = "phased"
	case caps.Lifecycle:
		caps.Preferred = "lifecycle"
	case caps.Legacy:
		caps.Preferred = "legacy"
	}
	return caps
}

// CommandByType finds the first command matching the provided type.
func (caps TestingCapabilities) CommandByType(kind string) *TestingCommand {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == "" {
		return nil
	}
	for _, cmd := range caps.Commands {
		if strings.ToLower(cmd.Type) == kind {
			c := cmd
			return &c
		}
	}
	return nil
}

// PreferredCommand returns the preferred testing command, falling back by priority order.
func (caps TestingCapabilities) PreferredCommand() *TestingCommand {
	if caps.Preferred != "" {
		if cmd := caps.CommandByType(caps.Preferred); cmd != nil {
			return cmd
		}
	}
	for _, kind := range []string{"genie", "phased", "lifecycle", "legacy"} {
		if cmd := caps.CommandByType(kind); cmd != nil {
			return cmd
		}
	}
	return nil
}

// SelectCommand picks a command using the caller's preferred type when provided.
func (caps TestingCapabilities) SelectCommand(preferred string) *TestingCommand {
	if cmd := caps.CommandByType(preferred); cmd != nil {
		return cmd
	}
	return caps.PreferredCommand()
}

func hasExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

func hasLifecycleTest(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var manifest struct {
		Lifecycle struct {
			Test string `json:"test"`
		} `json:"lifecycle"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false
	}
	return strings.TrimSpace(manifest.Lifecycle.Test) != ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func detectTestGenieCommand() []string {
	if disable := strings.TrimSpace(os.Getenv("TEST_GENIE_DISABLE")); disable != "" {
		return nil
	}
	if custom := strings.TrimSpace(os.Getenv("TEST_GENIE_BIN")); custom != "" {
		if hasExecutable(custom) {
			return []string{custom}
		}
		return nil
	}
	path, err := exec.LookPath("test-genie")
	if err == nil && path != "" {
		return []string{path}
	}
	if errors.Is(err, exec.ErrDot) {
		return nil
	}
	return nil
}

func projectRootFromScenario(scenarioDir string) string {
	dir := filepath.Clean(scenarioDir)
	if dir == "." || dir == "" {
		return ""
	}
	parent := filepath.Dir(dir)
	if parent == "." || parent == "/" {
		return ""
	}
	root := filepath.Dir(parent)
	if root == "." || root == "/" {
		return ""
	}
	return root
}
