// Package install owns the bootstrap/version-pin reconciliation for the kopia
// host binary. The binary itself is provisioned declaratively by the manifest's
// install.platforms block; this package is the Go-side reconciler that the
// `version` / `status` commands use to report whether the installed binary
// satisfies the pinned version and to warn on drift (e.g. brew/winget
// installing a version that differs from the pin).
package install

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/discovery"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/version"
)

// EngineReport summarizes the state of the provisioned kopia binary.
type EngineReport struct {
	BinaryPath string
	Installed  string // installed kopia version (empty if not installed)
	Pinned     string
	Satisfies  bool // installed >= pinned
	Present    bool // binary present on PATH
}

// Inspect resolves the kopia binary and compares its version to the pin.
func Inspect(ctx context.Context, loc discovery.Locator, run func(ctx context.Context, name string, args ...string) (string, error)) EngineReport {
	report := EngineReport{Pinned: version.Pinned}

	path, err := loc.Resolve()
	if err != nil {
		return report
	}
	report.BinaryPath = path
	report.Present = true

	installed, satisfies, err := loc.ProbeVersion(ctx, run)
	if err != nil {
		return report
	}
	report.Installed = installed.String()
	report.Satisfies = satisfies
	return report
}

// DriftWarning returns a human warning when the installed version drifts below
// the pin, or empty string when the report is healthy.
func (r EngineReport) DriftWarning() string {
	if !r.Present {
		return fmt.Sprintf("kopia binary not installed (pinned %s); run `vrooli setup --resources kopia`", r.Pinned)
	}
	if !r.Satisfies {
		return fmt.Sprintf("installed kopia %s is below pinned %s; reproducibility not guaranteed", r.Installed, r.Pinned)
	}
	return ""
}
