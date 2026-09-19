// Package mislocatedfile resolver records operator intent for a
// mislocated_file fix. v0.1 does NOT move the file — the actual file
// movement requires apply, which is unimplemented in v0.1. The resolver
// returns ErrResolverDeferred so the service can surface "ready for
// `arch-cart apply <domain>`" to the CLI.
package mislocatedfile

import (
	"context"

	"architecture-cartographer/internal/conflicts"
)

// Resolver is the production mislocated-file resolver.
type Resolver struct{}

// New returns the production resolver.
func New() *Resolver { return &Resolver{} }

func (Resolver) Name() string { return "mislocated_file" }
func (Resolver) Description() string {
	return "Records intent to move a file to the verdict-recommended domain. Actual movement deferred to apply."
}

func (Resolver) HandlesKinds() []conflicts.FixKind {
	return []conflicts.FixKind{conflicts.FixKindMoveFile}
}
func (Resolver) RequiresApply() bool { return true }

func (Resolver) Resolve(_ context.Context, _ conflicts.Conflict, _ conflicts.Fix) error {
	return conflicts.ErrResolverDeferred{
		Resolver: "mislocated_file",
		Reason:   "apply unimplemented in v0.1; intent recorded",
	}
}
