package dependencygovernance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

const maxListedLockChanges = 20

// lockChange names one staged lockfile operation (removal or update) for the
// shared preview/admission/apply routine.
type lockChange struct {
	noun           string // "removal" or "update", for messages
	verb           string // "remove" or "update"
	okVerdict      string
	blockedVerdict string
	failedVerdict  string
	// currentIfUnchanged reports a no-op lockfile delta as the "current" verdict
	// instead of applying it.
	currentIfUnchanged bool
	// highlight names packages whose locked versions are shown first, as
	// "name: old -> new", before the (possibly long) closure listing.
	highlight []string
	// fixes, when given, lets the outcome say which highlighted packages stay inside
	// an advisory after the change (exposure notes).
	fixes map[string][]advisoryFix
	// doneText says what a successful apply did: "<what> <names> from|in <target> via <manager>".
	doneText string
}

// lockChangeOutcome is the transport-independent result of a lock change.
type lockChangeOutcome struct {
	Done           bool
	DryRun         bool
	SourceUpdated  bool
	Blocked        bool
	Verdict        string
	Command        string
	PackageManager string
	ManifestPath   string
	Message        string
	NextSteps      []string
	Notes          []string
}

// RemoveDependency takes direct JavaScript dependencies off a surface through
// the same staged path as an install: the package manager removes them in a
// staging copy, the preview reports what leaves the lockfile, and apply
// publishes the staged inputs and relinks node_modules from the frozen lock.
// Removal needs no governance verdict (it narrows the dependency surface), but
// it shares the install's disk admission and rollback.
func (h *connectHandler) RemoveDependency(ctx context.Context, req *connect.Request[governancev1.RemoveDependencyRequest]) (*connect.Response[governancev1.RemoveDependencyResponse], error) {
	msg := req.Msg
	resolution, err := installgateway.ResolveRemoval(h.repoRoot(), msg.GetScenario(), msg.GetSurface(), msg.GetEcosystem(), msg.GetPackageNames())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := h.runLockChange(ctx, lockChange{
		noun: "removal", verb: "remove", okVerdict: "removable", blockedVerdict: "removal_blocked", failedVerdict: "removal_failed",
		doneText: "Removed %s from %s via %s",
	}, msg.GetScenario(), msg.GetSurface(), msg.GetApply(), resolution)
	return connect.NewResponse(&governancev1.RemoveDependencyResponse{
		Removed: out.Done, DryRun: out.DryRun, SourceUpdated: out.SourceUpdated, Blocked: out.Blocked,
		Verdict: out.Verdict, Command: out.Command, PackageManager: out.PackageManager, ManifestPath: out.ManifestPath,
		Message: out.Message, NextSteps: out.NextSteps, Notes: out.Notes,
	}), nil
}

func (h *connectHandler) repoRoot() string {
	if h != nil && h.scenariosDir != nil {
		return filepath.Dir(h.scenariosDir())
	}
	return ""
}

// runLockChange previews or applies one staged lockfile operation. A refusal
// from admission or staging is returned as a blocked outcome, never an error, so
// the caller reports it with its next steps.
func (h *connectHandler) runLockChange(ctx context.Context, change lockChange, scenario, surface string, apply bool, resolution installgateway.Resolution) lockChangeOutcome {
	target := fmt.Sprintf("scenarios/%s/%s", scenario, surface)
	out := lockChangeOutcome{
		DryRun:         !apply,
		Verdict:        change.okVerdict,
		Command:        resolution.Command(),
		PackageManager: resolution.PackageManager,
		ManifestPath:   resolution.ManifestPath,
	}
	blockedBy := func(cause error) lockChangeOutcome {
		out.Blocked = true
		out.Verdict = change.blockedVerdict
		lead := "Blocked (" + change.blockedVerdict + ")"
		if !apply {
			lead = "Dry run blocked (" + change.blockedVerdict + ")"
		}
		out.Message = fmt.Sprintf("%s: %v. Nothing was changed.", lead, cause)
		out.NextSteps = append(out.NextSteps, admissionNextSteps(cause)...)
		return out
	}

	admission, err := planInstallDisk(ctx, h.diskBroker, h.diskPlanner, resolution)
	defer admission.Release()
	if err != nil {
		return blockedBy(err)
	}
	stage := admission.plan.JSStage
	if stage != nil {
		removed, added, deltaErr := stage.LockDelta()
		if deltaErr != nil {
			return blockedBy(deltaErr)
		}
		locked, lockedErr := stage.LockedVersions(change.highlight)
		if lockedErr != nil {
			return blockedBy(lockedErr)
		}
		out.Notes = append(out.Notes, highlightedVersionNotes(change.highlight, removed, added, locked)...)
		out.Notes = append(out.Notes, exposureNotes(locked, change.fixes)...)
		out.Notes = append(out.Notes, lockDeltaNotes(removed, added)...)
		if change.currentIfUnchanged && len(removed) == 0 && len(added) == 0 {
			out.Verdict = "current"
			out.Message = fmt.Sprintf("Already current: nothing newer than the locked version of %s is allowed by the declared ranges in %s, so the lockfile is unchanged.", resolution.PackageName, target)
			return out
		}
	}
	out.Notes = append(out.Notes, admission.plan.Notes...)

	if !apply {
		out.Notes = append(out.Notes, admission.summary())
		out.Message = fmt.Sprintf("Dry run (%s): would run `%s` in %s. Re-run with --apply to %s.", change.okVerdict, resolution.Command(), target, change.verb)
		return out
	}

	lease, reserveErr := admission.reserve(ctx, h.diskBroker)
	if reserveErr != nil {
		return blockedBy(reserveErr)
	}
	resolution.DiskJSStage = stage
	installer := h.installer
	if installer == nil {
		installer = installgateway.ExecInstaller{}
	}
	job := runWithDiskJob(ctx, lease, resolution, func(installCtx context.Context) (string, error) {
		return installer.Install(installCtx, resolution)
	})
	runErr := job.RunErr
	if job.Receipt.ID != "" {
		out.Notes = append(out.Notes, diskReceiptNote(job.Receipt))
	}
	if job.LifeErr != nil {
		runErr = errors.Join(runErr, job.LifeErr)
	}
	output := strings.TrimSpace(job.Output)
	if runErr != nil {
		out.Verdict = change.failedVerdict
		out.Message = fmt.Sprintf("%s failed: `%s` — %v\n%s", strings.ToUpper(change.noun[:1])+change.noun[1:], resolution.Command(), runErr, output)
		out.NextSteps = append(out.NextSteps, "The error above says whether the previous manifest, lockfile and node_modules were restored. Fix the cause and repeat the dry run first.")
		return out
	}
	if stage != nil && stage.LinkSkipped != "" {
		out.SourceUpdated = true
		out.Message = fmt.Sprintf("Changed the dependency inputs of %s (%s: %s); node_modules was not relinked.\n%s", target, change.noun, resolution.PackageName, output)
		out.NextSteps = append(out.NextSteps, fmt.Sprintf("Converge node_modules through the lifecycle: vrooli scenario setup %s (pnpm would otherwise recreate it).", scenario))
		return out
	}
	out.Done = true
	out.Message = fmt.Sprintf(change.doneText, resolution.PackageName, target, resolution.PackageManager) + ".\n" + output
	out.NextSteps = append(out.NextSteps,
		fmt.Sprintf("Re-validate governance: %s deps approved validate %s", scenarioCLIName(), scenario),
		fmt.Sprintf("Confirm the surface still builds and its imports resolve: vrooli scenario test %s --phases unit", scenario))
	return out
}

// highlightedVersionNotes states, for each named package, which locked versions
// leave and enter, so the version the request is about is never lost in a long
// closure listing.
func highlightedVersionNotes(names, removed, added []string, locked map[string][]string) []string {
	versions := func(entries []string, name string) string {
		var found []string
		for _, entry := range entries {
			if version, ok := strings.CutPrefix(entry, name+"@"); ok {
				found = append(found, version)
			}
		}
		return strings.Join(found, ", ")
	}
	var notes []string
	for _, name := range names {
		from, to := versions(removed, name), versions(added, name)
		switch {
		case from == "" && to == "" && len(locked[name]) > 0:
			notes = append(notes, fmt.Sprintf("%s: locked at %s (unchanged)", name, strings.Join(locked[name], ", ")))
		case from == "" && to == "":
			notes = append(notes, name+": locked version unchanged")
		case from == "":
			notes = append(notes, fmt.Sprintf("%s: now also locked at %s", name, to))
		case to == "":
			notes = append(notes, fmt.Sprintf("%s: %s no longer locked", name, from))
		default:
			notes = append(notes, fmt.Sprintf("%s: %s -> %s", name, from, to))
		}
	}
	return notes
}

// lockDeltaNotes summarizes the closure change a staged operation makes.
func lockDeltaNotes(removed, added []string) []string {
	notes := []string{fmt.Sprintf("lockfile closure: %d package(s) leave, %d enter", len(removed), len(added))}
	list := func(label string, names []string) {
		if len(names) == 0 {
			return
		}
		shown := names
		if len(shown) > maxListedLockChanges {
			shown = shown[:maxListedLockChanges]
		}
		line := label + ": " + strings.Join(shown, ", ")
		if len(names) > len(shown) {
			line += fmt.Sprintf(", and %d more", len(names)-len(shown))
		}
		notes = append(notes, line)
	}
	list("leaving the lockfile", removed)
	list("entering the lockfile", added)
	return notes
}
