package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Resolver defines the interface for resolving workflow definitions.
type Resolver interface {
	// Resolve loads and resolves a workflow definition from the given path.
	Resolve(ctx context.Context, workflowPath string) (map[string]any, error)
}

// FileResolver resolves workflows from the filesystem using the native Go resolver.
type FileResolver struct {
	scenarioDir    string
	nativeResolver *NativeResolver
}

// NewResolver creates a new workflow resolver that uses native Go resolution.
func NewResolver(appRoot, scenarioDir string) *FileResolver {
	return &FileResolver{
		scenarioDir:    scenarioDir,
		nativeResolver: NewNativeResolver(scenarioDir),
	}
}

// Resolve loads and resolves a workflow definition.
// It uses the native Go resolver to expand @fixture/, @selector/, and @seed/ tokens.
// Returns an error if resolution fails - no silent fallbacks.
func (r *FileResolver) Resolve(ctx context.Context, workflowPath string) (map[string]any, error) {
	// Ensure absolute path
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(r.scenarioDir, filepath.FromSlash(workflowPath))
	}

	// Verify file exists
	if _, err := os.Stat(workflowPath); err != nil {
		return nil, fmt.Errorf("workflow file not found: %s", workflowPath)
	}

	// Use native Go resolver - no fallbacks
	result, err := r.nativeResolver.ResolveWorkflow(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("workflow resolution failed for %s: %w", filepath.Base(workflowPath), err)
	}

	return result, nil
}

// parseWorkflow parses JSON bytes into a workflow definition map.
func parseWorkflow(data []byte) (map[string]any, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("invalid workflow JSON: %w", err)
	}
	return doc, nil
}
