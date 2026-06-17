package dependencygovernance

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"scenario-dependency-analyzer/internal/installgateway"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// WithInstaller overrides the package installer (tests inject a fake that
// records the resolved plan without running a real pnpm/go/pip).
func WithInstaller(installer installgateway.PackageInstaller) ConnectOption {
	return func(h *connectHandler) { h.installer = installer }
}

// InstallDependency is the governed install gateway. It resolves the surface's
// package manager + manifest, enforces governance (denied/deprecated/out-of-
// range/surface-not-allowed/unrecorded all block, fail-closed), and on apply runs
// the install behind the PackageInstaller seam. Dry-run (the default) returns the
// exact command + verdict without mutating anything.
func (h *connectHandler) InstallDependency(ctx context.Context, req *connect.Request[governancev1.InstallDependencyRequest]) (*connect.Response[governancev1.InstallDependencyResponse], error) {
	msg := req.Msg
	repoRoot := ""
	if h != nil && h.scenariosDir != nil {
		repoRoot = filepath.Dir(h.scenariosDir())
	}

	resolution, err := installgateway.Resolve(repoRoot, msg.GetScenario(), msg.GetSurface(), msg.GetEcosystem(), msg.GetPackageName(), msg.GetVersion())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	verdict, blocked, nextSteps, secNotes := h.registry().governInstall(msg, resolution)

	resp := &governancev1.InstallDependencyResponse{
		DryRun:         !msg.GetApply(),
		Verdict:        verdict,
		Blocked:        blocked,
		Command:        resolution.Command(),
		PackageManager: resolution.PackageManager,
		ManifestPath:   resolution.ManifestPath,
		NextSteps:      nextSteps,
		SecurityNotes:  secNotes,
		Guidance:       Guidance,
	}

	switch {
	case blocked:
		resp.Message = fmt.Sprintf("Blocked (%s): %s/%s is not installable into scenarios/%s/%s. The install was NOT run.",
			verdict, msg.GetEcosystem(), msg.GetPackageName(), msg.GetScenario(), msg.GetSurface())
	case !msg.GetApply():
		resp.Message = fmt.Sprintf("Dry run (%s): would run `%s` in scenarios/%s/%s. Re-run with --apply to install.",
			verdict, resolution.Command(), msg.GetScenario(), msg.GetSurface())
	default:
		installer := h.installer
		if installer == nil {
			installer = installgateway.ExecInstaller{}
		}
		output, runErr := installer.Install(ctx, resolution)
		if runErr != nil {
			resp.Message = fmt.Sprintf("Install failed (%s): `%s` — %v\n%s", verdict, resolution.Command(), runErr, strings.TrimSpace(output))
			resp.NextSteps = append(resp.NextSteps, "Fix the error above and re-run, or run the dry-run first to inspect the command.")
			return connect.NewResponse(resp), nil
		}
		resp.Installed = true
		resp.Message = fmt.Sprintf("Installed (%s): %s/%s into scenarios/%s/%s via %s.",
			verdict, msg.GetEcosystem(), msg.GetPackageName(), msg.GetScenario(), msg.GetSurface(), resolution.PackageManager)
		resp.NextSteps = append(resp.NextSteps,
			fmt.Sprintf("Re-validate governance: %s deps approved validate %s", scenarioCLIName(), msg.GetScenario()),
			"Confirm no new vulnerabilities: security-health deps status --json")
	}

	return connect.NewResponse(resp), nil
}

// governInstall applies the governance decision for an install request against
// the resolved surface. It returns the verdict, whether the install is blocked
// (fail-closed), agent-facing next steps, and any recorded security notes.
func (r *Registry) governInstall(msg *governancev1.InstallDependencyRequest, _ installgateway.Resolution) (verdict string, blocked bool, nextSteps, secNotes []string) {
	ecosystem := strings.TrimSpace(msg.GetEcosystem())
	pkg := strings.TrimSpace(msg.GetPackageName())
	surface := strings.ToLower(strings.TrimSpace(msg.GetSurface()))

	record, found, err := r.Explain(ecosystem, pkg)
	if err != nil {
		return "registry_error", true, []string{fmt.Sprintf("Approved dependency registry unreadable: %v. Run `%s deps approved list` to surface the parse error.", err, scenarioCLIName())}, nil
	}
	if !found {
		return "unrecorded", true, []string{
			fmt.Sprintf("Record it first: %s deps approved approve-observed %s/%s --from-findings --apply", scenarioCLIName(), ecosystem, pkg),
			fmt.Sprintf("Or find an approved alternative: %s deps approved search \"%s\"", scenarioCLIName(), pkg),
		}, nil
	}

	if notes := strings.TrimSpace(record.GetSecurityNotes()); notes != "" {
		secNotes = append(secNotes, notes)
	}

	switch normalize(record.GetState()) {
	case "denied", "blocked":
		return "denied", true, []string{firstNonEmpty(record.GetReplacement(), "This package is denied. Choose an approved alternative or file a reviewed governance exception.")}, secNotes
	case "deprecated":
		return "deprecated", true, []string{firstNonEmpty(record.GetReplacement(), "This package is deprecated. Migrate to the recorded replacement.")}, secNotes
	}

	// Approved family. Enforce allowed_surfaces and the version range.
	if allowed := record.GetAllowedSurfaces(); len(allowed) > 0 && !containsFold(allowed, surface) {
		return "surface_not_allowed", true, []string{
			fmt.Sprintf("%s/%s is approved only for surfaces: %s. Install into one of those, or widen the approval.", ecosystem, pkg, strings.Join(allowed, ", ")),
		}, secNotes
	}
	if v := strings.TrimSpace(msg.GetVersion()); v != "" {
		if decision := evaluateVersionPolicy(ecosystem, v, record.GetVersionRange(), record.GetRangePolicy()); !decision.Allowed {
			return "out_of_range", true, []string{
				fmt.Sprintf("Version %s is outside the approved range %s. Use a version in range, or widen it: %s deps approved widen-range %s/%s --to-major-line --apply", v, record.GetVersionRange(), scenarioCLIName(), ecosystem, pkg),
			}, secNotes
		}
	}

	return firstNonEmpty(normalize(record.GetState()), "approved"), false, nil, secNotes
}

// scenarioCLIName is the CLI binary that fronts this service.
func scenarioCLIName() string { return "scenario-dependency-analyzer" }
