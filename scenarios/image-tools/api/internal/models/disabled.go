package models

import (
	"fmt"
	"strings"
)

// DisabledKind says why a catalog row is switched off. Licensing is deliberately
// not a kind: the operator's rule is the best model for normal use whatever its
// licence, so rights only gate monetized, hosted or redistributed use (the rights
// record and commercial_use notes carry that, and the shared rights owner enforces
// it at use time). A row that is off must be off for a reason a reader can act on.
type DisabledKind string

const (
	// DisabledRuntime: it cannot run on this build today (no runner, no usable
	// backend, no loadable weight format). Its verdict reads broken.
	DisabledRuntime DisabledKind = "runtime"
	// DisabledNeedsInstall: the weights or backend need a manual download, export
	// or install the catalog cannot perform, so enabling it would only fail a job.
	DisabledNeedsInstall DisabledKind = "needs_install"
	// DisabledHardware: it does not fit this host's GPU or memory.
	DisabledHardware DisabledKind = "hardware"
	// DisabledSuperseded: a recommended model covers the operation and this row adds
	// cost without a gain anyone has measured.
	DisabledSuperseded DisabledKind = "superseded"
	// DisabledCandidate: registered for comparison and not yet judged.
	DisabledCandidate DisabledKind = "candidate"
)

var disabledKinds = map[DisabledKind]bool{
	DisabledRuntime: true, DisabledNeedsInstall: true, DisabledHardware: true,
	DisabledSuperseded: true, DisabledCandidate: true,
}

// DisabledReason is the structured, visible reason a row is off. Verdicts show it
// for a disabled model that has no other evidence, so "not enabled" always comes
// with its why.
type DisabledReason struct {
	Kind   DisabledKind `json:"kind"`
	Reason string       `json:"reason"`
}

// Breaks reports whether the reason means the model cannot run (a broken verdict).
func (d DisabledReason) Breaks() bool { return d.Kind == DisabledRuntime }

// Validate checks the kind is known and the reason is one non-empty line.
func (d DisabledReason) Validate() error {
	if !disabledKinds[d.Kind] {
		return fmt.Errorf("disabled kind %q must be one of runtime, needs_install, hardware, superseded, candidate (licensing is never a reason to disable)", d.Kind)
	}
	if strings.TrimSpace(d.Reason) == "" || strings.ContainsAny(d.Reason, "\r\n") {
		return fmt.Errorf("disabled reason must be one non-empty line")
	}
	return nil
}
