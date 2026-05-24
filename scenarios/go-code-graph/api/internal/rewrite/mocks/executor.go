// Package mocks holds test fakes for the rewrite domain seams. Lives
// outside _test.go files so sibling packages (handlers/rewrite_test,
// handlers/graph_test, cli/domains/rewrite_test) can import the same
// fake without copying.
//
// Production code never imports this package; testutil's
// no_prod_import_test asserts it.
package mocks

import (
	"context"

	intrewrite "go-code-graph/internal/rewrite"
)

// FakeExecutor is the canned RewriteExecutor for tests. Set
// ExecuteFunc to the behaviour the test wants; the zero value returns
// nil (success) for every op.
type FakeExecutor struct {
	ExecuteFunc func(ctx context.Context, scenarioRoot string, op intrewrite.Operation) error
}

// Execute satisfies intrewrite.RewriteExecutor.
func (f *FakeExecutor) Execute(ctx context.Context, scenarioRoot string, op intrewrite.Operation) error {
	if f.ExecuteFunc != nil {
		return f.ExecuteFunc(ctx, scenarioRoot, op)
	}
	return nil
}

// Compile-time assertion.
var _ intrewrite.RewriteExecutor = (*FakeExecutor)(nil)
