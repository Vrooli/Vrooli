package dependencygovernance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"

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
	if msg.GetWorkspacePath() != "" {
		if msg.GetScenario() != "" || msg.GetSurface() != "" || msg.GetRefreshLock() || strings.HasPrefix(msg.GetVersion(), "override:") {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("workspace_path cannot be combined with scenario/surface, refresh_lock, or overrides"))
		}
		msg = proto.Clone(msg).(*governancev1.InstallDependencyRequest)
		msg.Scenario, msg.Surface = "external-workspace", "workspace"
	}
	repoRoot := ""
	if h != nil && h.scenariosDir != nil {
		repoRoot = filepath.Dir(h.scenariosDir())
	}

	overrideVersion := strings.TrimPrefix(strings.TrimSpace(msg.GetVersion()), "override:")
	isNpmOverride := strings.HasPrefix(strings.TrimSpace(msg.GetVersion()), "override:")
	jsEcosystem := false
	switch strings.ToLower(strings.TrimSpace(msg.GetEcosystem())) {
	case "npm", "node", "js", "ts", "typescript":
		jsEcosystem = true
	}
	if msg.GetDevelopment() && (!jsEcosystem || isNpmOverride || msg.GetSurface() == "host-tool" || msg.GetRefreshLock()) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("development placement requires a mutating JavaScript surface and cannot be combined with overrides or host-tool operations"))
	}
	if msg.GetRefreshLock() && (msg.GetSurface() != "host-tool" || msg.GetEcosystem() != "npm" || msg.GetPackageName() != "hyperframes" || !msg.GetExactVersion() || msg.GetDevelopment() || isNpmOverride) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("refresh_lock requires npm/hyperframes on host-tool with an explicit exact version and development=false"))
	}
	var resolution installgateway.Resolution
	var err error
	if msg.GetWorkspacePath() != "" {
		resolution, err = installgateway.ResolveWorkspace(repoRoot, msg.GetWorkspacePath(), msg.GetEcosystem(), msg.GetPackageName(), msg.GetVersion())
	} else if msg.GetRefreshLock() {
		resolution, err = installgateway.ResolveHostToolRefresh(repoRoot, msg.GetScenario(), msg.GetEcosystem(), msg.GetPackageName(), msg.GetVersion())
	} else if isNpmOverride {
		if strings.ToLower(strings.TrimSpace(msg.GetEcosystem())) != "npm" || strings.TrimSpace(overrideVersion) == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("override: requires npm and a version"))
		}
		resolution, err = installgateway.ResolveNpmOverride(repoRoot, msg.GetScenario(), msg.GetSurface())
	} else {
		resolution, err = installgateway.Resolve(repoRoot, msg.GetScenario(), msg.GetSurface(), msg.GetEcosystem(), msg.GetPackageName(), msg.GetVersion())
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if msg.GetDevelopment() {
		if err := installgateway.ApplyDevelopmentDependency(&resolution); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	if msg.GetExactVersion() {
		if err := installgateway.ApplyExactVersion(&resolution, msg.GetVersion()); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	verdict, blocked, nextSteps, secNotes := h.registry().governInstall(msg, resolution)
	if resolution.Profile.ScriptsDisabled {
		secNotes = append(secNotes, "lifecycle scripts are disabled by default for this install profile")
	}
	secNotes = append(secNotes, fmt.Sprintf("install profile: lifecycle=%s frozen_lockfile=%t governance=%s", resolution.Profile.LifecycleMode, resolution.Profile.FrozenLockfile, resolution.Profile.Governance))

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
	target := fmt.Sprintf("scenarios/%s/%s", msg.GetScenario(), msg.GetSurface())
	if resolution.ExternalWorkspace {
		target = "external workspace " + resolution.SurfaceRoot
	}
	if msg.GetScenario() == "vrooli" && msg.GetSurface() == "control-plane" {
		target = "Vrooli control-plane root Go module"
	}
	if resolution.Profile.LifecycleMode == "managed-service-artifact" {
		target = "resource-owned verified artifact " + msg.GetScenario() + " (staging only; serving process unchanged)"
	} else if resolution.Profile.LifecycleMode == installgateway.MT7927SourceLifecycle {
		target = "local pinned MT7927 build inputs"
	} else if resolution.Profile.LifecycleMode == "resource-agent" {
		target = "resource-owned harness " + msg.GetScenario() + " (exact user-prefix installation)"
	} else if resolution.HostTool || resolution.HostToolRefresh {
		target = "managed host tool " + msg.GetScenario() + " (private frozen candidate)"
		if resolution.HostToolRefresh {
			target = "host-tool dependency source " + msg.GetScenario() + " (private lock-refresh candidate; no runtime installation)"
		}
	}

	switch {
	case blocked:
		resp.Message = fmt.Sprintf("Blocked (%s): %s/%s is not installable into %s. The install was NOT run.",
			verdict, msg.GetEcosystem(), msg.GetPackageName(), target)
	case !msg.GetApply():
		action := "install"
		if resolution.Profile.LifecycleMode == "managed-service-artifact" {
			action = "stage the verified artifact; serving process remains unchanged"
		}
		if resolution.Profile.LifecycleMode == installgateway.MT7927SourceLifecycle {
			action = "stage verified data for exact-ABI build qualification"
		}
		if resolution.HostToolRefresh {
			action = "refresh source dependency inputs only; no runtime will be installed"
		}
		resp.Message = fmt.Sprintf("Dry run (%s): would run `%s` in %s. Re-run with --apply to %s.",
			verdict, resolution.Command(), target, action)
	default:
		lease, plan, admissionErr := reserveInstallDiskJob(ctx, h.diskBroker, h.diskPlanner, resolution)
		if admissionErr != nil {
			resp.Blocked = true
			resp.Verdict = "disk_admission_denied"
			resp.Message = fmt.Sprintf("Blocked (disk_admission_denied): %v. The install was NOT run and no manifest was changed.", admissionErr)
			resp.NextSteps = append(resp.NextSteps, "Restore the shared capacity broker or resolve the uncovered path/size estimate, then retry the dry-run and apply request.")
			return connect.NewResponse(resp), nil
		}
		resolution.DiskCachePath = plan.CachePath
		resolution.DiskBuildCachePath = plan.BuildCachePath
		resolution.DiskPythonExecutable = plan.PythonExecutable
		resolution.DiskPythonWheelURL = plan.PythonWheelURL
		resolution.DiskPythonWheelSHA256 = plan.PythonWheelSHA256
		resolution.DiskPythonWheelBytes = plan.PythonWheelBytes
		resolution.DiskPythonTarget = plan.PythonTarget
		resolution.DiskPythonStage = plan.PythonStage
		installer := h.installer
		if installer == nil {
			installer = installgateway.ExecInstaller{}
		}
		job := runWithDiskJob(ctx, lease, resolution, func(installCtx context.Context) (string, error) {
			if isNpmOverride {
				if err := installgateway.SetNpmOverride(resolution.ManifestPath, msg.GetPackageName(), overrideVersion); err != nil {
					return "", err
				}
			}
			return installer.Install(installCtx, resolution)
		})
		output, runErr := job.Output, job.RunErr
		if job.Receipt.ID != "" {
			resp.SecurityNotes = append(resp.SecurityNotes, diskReceiptNote(job.Receipt))
		}
		if job.LifeErr != nil {
			runErr = errors.Join(runErr, job.LifeErr)
		}
		if runErr != nil {
			resp.Message = fmt.Sprintf("Install failed (%s): `%s` — %v\n%s", verdict, resolution.Command(), runErr, strings.TrimSpace(output))
			resp.NextSteps = append(resp.NextSteps, "Fix the error above and re-run, or run the dry-run first to inspect the command.")
			return connect.NewResponse(resp), nil
		}
		if resolution.HostToolRefresh {
			resp.SourceUpdated = true
			resp.Message = "Updated host-tool dependency source inputs; no managed runtime was installed.\n" + strings.TrimSpace(output)
			resp.NextSteps = append(resp.NextSteps, "Review the generated manifest/lock diff and align the owner version metadata before separately admitting frozen provisioning. Preserve the previous immutable installation.")
			return connect.NewResponse(resp), nil
		}
		if resolution.PythonBootstrap != nil {
			resp.SourceUpdated = true
			resp.Message = "Bootstrapped the hashed resource Python lock; no runtime was installed.\n" + strings.TrimSpace(output)
			resp.NextSteps = append(resp.NextSteps, "Review the generated complete lock and source hashes before separately admitting normal resource acquisition and runtime qualification.")
			return connect.NewResponse(resp), nil
		}
		if len(resolution.PythonLockfiles) > 0 {
			resp.SourceUpdated = true
			resp.Message = "Updated canonical Python dependency inputs and hashed locks; no runtime was installed.\n" + strings.TrimSpace(output)
			resp.NextSteps = append(resp.NextSteps, "Verify the owning input/lock conformance, then use the normal scenario or resource lifecycle to converge its private environment.")
			return connect.NewResponse(resp), nil
		}
		if resolution.Profile.LifecycleMode == "managed-service-artifact" {
			resp.Installed = true
			resp.Message = "Staged verified resource artifact (" + verdict + "); serving process unchanged.\n" + strings.TrimSpace(output)
			resp.NextSteps = append(resp.NextSteps, "Inspect desired, staged and running versions with vrooli resource status "+msg.GetScenario()+" --json; connector promotion and rollback are separate lifecycle operations.")
			return connect.NewResponse(resp), nil
		}
		if resolution.Profile.LifecycleMode == installgateway.MT7927SourceLifecycle {
			resp.Installed = true
			resp.Message = strings.TrimSpace(output)
			resp.NextSteps = append(resp.NextSteps, "Use the staged source inputs for an isolated build against the exact target kernel headers.")
			return connect.NewResponse(resp), nil
		}
		resp.Installed = true
		verb := "Installed"
		if resolution.TemplateVersion != "" {
			verb = "Updated template declaration and lockfile"
		}
		if isNpmOverride {
			verb = "Applied governed npm override"
		}
		resp.Message = fmt.Sprintf("%s (%s): %s/%s into %s via %s.", verb,
			verdict, msg.GetEcosystem(), msg.GetPackageName(), target, resolution.PackageManager)
		if resolution.Profile.LifecycleMode == "resource-agent" {
			resp.NextSteps = append(resp.NextSteps, "Verify selected executable/version through the resource owner before launch; retain the prior installation for rollback.")
			return connect.NewResponse(resp), nil
		}
		if resolution.HostTool {
			resp.NextSteps = append(resp.NextSteps, "Verify managed bytes: vrooli host install hyperframes --dry-run", "Qualify one bounded local composition before fresh-film production; no browser was downloaded.")
			return connect.NewResponse(resp), nil
		}
		if resolution.PackageManager == "pip-local-wheel" {
			resp.NextSteps = append(resp.NextSteps, "Run the local research tool with `PYTHONPATH="+resolution.DiskPythonTarget+" python3 -m yt_dlp` or the target's `bin/yt-dlp` entry point; keep the artifact labeled local-research-only.")
			return connect.NewResponse(resp), nil
		}
		if resolution.ExternalWorkspace {
			resp.NextSteps = append(resp.NextSteps, "Re-run this deps install request without --apply to validate all direct workspace dependencies.", "Review the generated lockfile and retained package integrity values before executing upstream code.")
			return connect.NewResponse(resp), nil
		}
		resp.NextSteps = append(resp.NextSteps,
			fmt.Sprintf("Re-validate governance: %s deps approved validate %s", scenarioCLIName(), msg.GetScenario()),
			"Confirm no new vulnerabilities: security-health deps status --json")
	}

	return connect.NewResponse(resp), nil
}

// governInstall applies the governance decision for an install request against
// the resolved surface. It returns the verdict, whether the install is blocked
// (fail-closed), agent-facing next steps, and any recorded security notes.
func (r *Registry) governInstall(msg *governancev1.InstallDependencyRequest, resolution installgateway.Resolution) (verdict string, blocked bool, nextSteps, secNotes []string) {
	if resolution.ExternalWorkspace {
		declared, err := scanPackageJSON(resolution.ManifestPath)
		if err != nil {
			return "manifest_error", true, []string{fmt.Sprintf("Read external dependency closure: %v", err)}, nil
		}
		for _, dependency := range declared {
			if dependency.PackageName == msg.GetPackageName() {
				continue // The requested replacement is evaluated below.
			}
			peer := &governancev1.InstallDependencyRequest{Scenario: "external-workspace", Surface: "workspace", Ecosystem: "npm", PackageName: dependency.PackageName, Version: dependency.Version}
			peerVerdict, peerBlocked, steps, notes := r.governInstall(peer, installgateway.Resolution{})
			secNotes = append(secNotes, notes...)
			if peerBlocked {
				return peerVerdict, true, append([]string{"Existing external workspace dependency is not approved: " + dependency.PackageName}, steps...), secNotes
			}
		}
	}
	if resolution.PythonBootstrap != nil {
		for _, requirement := range resolution.PythonBootstrap.Requirements {
			peer := &governancev1.InstallDependencyRequest{Scenario: msg.GetScenario(), Surface: msg.GetSurface(), Ecosystem: "pip", PackageName: requirement.Name, Version: requirement.Version}
			peerVerdict, peerBlocked, steps, notes := r.governInstall(peer, installgateway.Resolution{})
			secNotes = append(secNotes, notes...)
			if peerBlocked {
				return peerVerdict, true, append([]string{fmt.Sprintf("Python bootstrap direct requirement %s@%s is not approved", requirement.Name, requirement.Version)}, steps...), secNotes
			}
		}
	}
	ecosystem := strings.TrimSpace(msg.GetEcosystem())
	pkg := strings.TrimSpace(msg.GetPackageName())
	version := strings.TrimSpace(msg.GetVersion())
	if strings.HasPrefix(version, "override:") {
		version = strings.TrimSpace(strings.TrimPrefix(version, "override:"))
	}
	// npm aliases replace a legacy package name with a separately governed
	// successor while preserving consumers that import the original name. The
	// successor—not the legacy alias key—is the package whose approval and
	// version policy must be evaluated.
	if ecosystem == "npm" {
		if aliasPkg, aliasVersion, ok := npmAliasTarget(version); ok {
			pkg, version = aliasPkg, aliasVersion
		}
	}
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

	// Apply the same scenario policy used by registry validation before any writes.
	if violation := scenarioExceptionViolation(msg.GetScenario(), record); violation.reason != "" {
		return "scenario_not_allowed", true, []string{violation.reason}, secNotes
	}

	// Approved family. Enforce allowed_surfaces and the version range.
	if allowed := record.GetAllowedSurfaces(); len(allowed) > 0 && !containsFold(allowed, surface) {
		return "surface_not_allowed", true, []string{
			fmt.Sprintf("%s/%s is approved only for surfaces: %s. Install into one of those, or widen the approval.", ecosystem, pkg, strings.Join(allowed, ", ")),
		}, secNotes
	}
	if v := version; v != "" {
		if decision := evaluateVersionPolicy(ecosystem, v, record.GetVersionRange(), record.GetRangePolicy()); !decision.Allowed {
			if decision.FindingClass == "VERSION_RANGE_UNPARSEABLE" {
				return "range_unparseable", true, []string{decision.Description, decision.Remediation}, secNotes
			}
			return "out_of_range", true, []string{
				fmt.Sprintf("Version %s is outside the approved range %s. Use a version in range, or widen it: %s deps approved widen-range %s/%s --to-major-line --apply", v, record.GetVersionRange(), scenarioCLIName(), ecosystem, pkg),
			}, secNotes
		}
	}

	return firstNonEmpty(normalize(record.GetState()), "approved"), false, nil, secNotes
}

// npmAliasTarget parses npm's supported "npm:<package>@<version>" alias
// specifier. Scoped package names are handled by splitting at the final @.
func npmAliasTarget(version string) (packageName, targetVersion string, ok bool) {
	alias := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(version), "npm:"))
	if alias == "" || !strings.HasPrefix(strings.TrimSpace(version), "npm:") {
		return "", "", false
	}
	index := strings.LastIndex(alias, "@")
	if index <= 0 || index == len(alias)-1 {
		return "", "", false
	}
	packageName = strings.TrimSpace(alias[:index])
	targetVersion = strings.TrimSpace(alias[index+1:])
	return packageName, targetVersion, packageName != "" && targetVersion != ""
}

// scenarioCLIName is the CLI binary that fronts this service.
func scenarioCLIName() string { return "scenario-dependency-analyzer" }
