package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// NativeResolver resolves workflows using pure Go (no Python dependency).
// Since BAS now handles all variable interpolation natively (${@params/...}, ${@store/...}, etc.),
// this resolver simply loads workflow JSON files and returns them as-is.
// Subflow resolution (via workflowPath) and variable interpolation are handled by BAS at execution time.
type NativeResolver struct {
	scenarioDir string
}

// NewNativeResolver creates a new native workflow resolver.
func NewNativeResolver(scenarioDir string) *NativeResolver {
	return &NativeResolver{
		scenarioDir: scenarioDir,
	}
}

// ResolveWorkflow loads a workflow definition from disk.
// The workflow is returned as-is - BAS handles all variable interpolation and subflow resolution
// at execution time using the ${@params/...}, ${@store/...}, and ${@env/...} namespaces.
func (r *NativeResolver) ResolveWorkflow(workflowPath string) (map[string]any, error) {
	// Ensure absolute path
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(r.scenarioDir, filepath.FromSlash(workflowPath))
	}

	// Load workflow
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}

	var workflow map[string]any
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	// Ensure metadata.reset has a default value
	ensureResetMetadata(workflow)

	return workflow, nil
}

// LoadSeedState loads the seed state file if it exists.
// Seed state values are passed to BAS as initial_params for the ${@params/...} namespace.
func (r *NativeResolver) LoadSeedState() (map[string]any, error) {
	seedPath := sharedartifacts.SeedStatePath(r.scenarioDir)
	data, err := os.ReadFile(seedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse seed state: %w", err)
	}
	return state, nil
}

// ensureResetMetadata ensures the workflow has a reset value in metadata.
func ensureResetMetadata(workflow map[string]any) {
	metadata, ok := workflow["metadata"].(map[string]any)
	if !ok {
		metadata = make(map[string]any)
		workflow["metadata"] = metadata
	}

	if _, hasReset := metadata["reset"]; !hasReset {
		metadata["reset"] = "none"
	}
}
