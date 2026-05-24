package rewrite

import "context"

// RewriteExecutor applies one operation to the filesystem rooted at
// scenarioRoot. Production wires FSExecutor (executor_fs.go); tests
// wire FakeExecutor from mocks/executor.go.
//
// seam: This is the ONLY edge in the rewrite domain that touches disk.
// Anything else needing filesystem access must route through here so
// the no_external_command_test.go invariant (no git/go subprocess
// invocation) holds.
type RewriteExecutor interface {
	Execute(ctx context.Context, scenarioRoot string, op Operation) error
}
