package apply

import "context"

// BuildGuard is the seam between apply and the language toolchain. v0.1
// ships the interface only; production implementations (go, tsc) land
// in the apply execution phase.
type BuildGuard interface {
	// Name identifies the toolchain ("go", "tsc").
	Name() string

	// Baseline captures the pre-apply build state. Returns Green=false
	// if the toolchain is already red, telling the caller to refuse
	// apply (per the build-green guardrail).
	Baseline(ctx context.Context, scenario string) (BuildBaseline, error)

	// Verify runs the toolchain over the post-apply tree.
	Verify(ctx context.Context, scenario string) (BuildBaseline, error)
}
