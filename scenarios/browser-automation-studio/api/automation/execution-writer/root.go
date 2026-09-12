package executionwriter

import (
	"context"
	"fmt"
	"path/filepath"
)

// RootProvider resolves the execution-artifact root for one request. Implementations
// may select a lease-owned test root when the request is in test mode.
type RootProvider interface {
	Root(context.Context) (string, error)
	RecordWrite(context.Context)
}

type staticRoot struct{ root string }

// NewStaticRoot is appropriate for explicitly local tools and unit tests. Server
// wiring uses the scenario-owned routed provider instead.
func NewStaticRoot(root string) RootProvider { return staticRoot{root: root} }

func (r staticRoot) Root(context.Context) (string, error) {
	if r.root == "" {
		return "", fmt.Errorf("execution artifact root is empty")
	}
	return filepath.Clean(r.root), nil
}

func (staticRoot) RecordWrite(context.Context) {}
