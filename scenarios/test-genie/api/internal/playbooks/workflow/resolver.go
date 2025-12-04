package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	// DefaultResolverScript is the path to the Python workflow resolver script.
	DefaultResolverScript = "scripts/scenarios/testing/playbooks/resolve-workflow.py"
)

// Resolver defines the interface for resolving workflow definitions.
type Resolver interface {
	// Resolve loads and resolves a workflow definition from the given path.
	Resolve(ctx context.Context, workflowPath string) (map[string]any, error)
}

// FileResolver resolves workflows from the filesystem, optionally using a Python resolver.
type FileResolver struct {
	appRoot     string
	scenarioDir string
}

// NewResolver creates a new workflow resolver.
func NewResolver(appRoot, scenarioDir string) *FileResolver {
	return &FileResolver{
		appRoot:     appRoot,
		scenarioDir: scenarioDir,
	}
}

// Resolve loads and resolves a workflow definition.
// It first tries the Python resolver script if available, then falls back to direct file read.
func (r *FileResolver) Resolve(ctx context.Context, workflowPath string) (map[string]any, error) {
	// Ensure absolute path
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(r.scenarioDir, filepath.FromSlash(workflowPath))
	}

	// Verify file exists
	if _, err := os.Stat(workflowPath); err != nil {
		return nil, fmt.Errorf("workflow file not found: %s", workflowPath)
	}

	// Try Python resolver first
	output, err := r.tryPythonResolver(ctx, workflowPath)
	if err == nil && len(output) > 0 {
		return parseWorkflow(output)
	}

	// Fall back to direct file read
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}

	return parseWorkflow(data)
}

// tryPythonResolver attempts to use the Python resolver script.
func (r *FileResolver) tryPythonResolver(ctx context.Context, workflowPath string) ([]byte, error) {
	script := filepath.Join(r.appRoot, DefaultResolverScript)
	stat, err := os.Stat(script)
	if err != nil || stat.IsDir() {
		return nil, fmt.Errorf("resolver script not found")
	}

	pythonCmd := findPython()
	if pythonCmd == "" {
		return nil, fmt.Errorf("python not found")
	}

	cmd := exec.CommandContext(ctx, pythonCmd, script,
		"--workflow", workflowPath,
		"--scenario", r.scenarioDir,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("resolver failed: %s", stderr.String())
	}

	return stdout.Bytes(), nil
}

// findPython returns the Python command name, or empty if not found.
func findPython() string {
	for _, cmd := range []string{"python3", "python"} {
		if _, err := exec.LookPath(cmd); err == nil {
			return cmd
		}
	}
	return ""
}

// parseWorkflow parses JSON bytes into a workflow definition map.
func parseWorkflow(data []byte) (map[string]any, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON: %w", err)
	}
	return doc, nil
}
