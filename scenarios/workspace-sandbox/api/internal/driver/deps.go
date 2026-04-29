// Package driver — shared dependency container.
//
// Deps carries the seam dependencies every driver constructor needs.
// Round 4 Phase 7 introduced fsmount.Mounter and process.Starter as
// required deps; bundling them with clock.Clock keeps factory call
// sites readable (one struct argument instead of three positional
// parameters) and lets new seams be added without re-flowing every
// driver factory signature.
//
// Validation panics on nil because the workspace-sandbox API has only
// one wiring path (main.go) and drivers without a clock/mounter/
// starter are unusable — a fail-loud panic at startup beats a
// nil-deref at the first overlay mount.
package driver

import (
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/process"
)

// Deps is the bundle of seam dependencies every driver factory takes.
// Every field is required; Validate panics on nil.
type Deps struct {
	// Clock is the time source for change-detection timestamps and
	// driver-internal scheduling. Required.
	Clock clock.Clock

	// Mounter handles every mount/unmount/IsMountPoint syscall. Required.
	Mounter fsmount.Mounter

	// Starter handles every external process invocation (fuse-overlayfs
	// version probe, modprobe overlay, mountpoint, fusermount, bwrap,
	// prlimit, the agent process itself). Required.
	Starter process.Starter
}

// Validate panics when any required field is nil. Called from every
// driver factory; the panic message names the missing field and the
// caller so wiring bugs surface at startup rather than first-mount.
func (d Deps) Validate(caller string) {
	if d.Clock == nil {
		panic(caller + ": Deps.Clock is required")
	}
	if d.Mounter == nil {
		panic(caller + ": Deps.Mounter is required")
	}
	if d.Starter == nil {
		panic(caller + ": Deps.Starter is required")
	}
}
