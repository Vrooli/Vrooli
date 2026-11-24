// Package rootstatus reports the state of the root control plane in the terms
// an operator can act on: is the serving image healthy, are automatic updates
// paused, and is an interrupted transition waiting to be finished.
//
// It is read-only. Every finding names the one command that resolves it, so
// `vrooli doctor` never requires the operator to reconstruct a bundle path or
// remember which of several transition commands applies.
package rootstatus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	runtimesupervisorsafeguard "github.com/vrooli/vrooli/internal/safeguards/runtime-supervisor"
)

// Check is one finding. Status is "ok", "attention" or "unknown"; Message says
// what to do, not only what is wrong.
type Check struct {
	Name    string
	Status  string
	Message string
}

// Checks inspects the runtime home. It never mutates host state.
func Checks(ctx context.Context, home string) []Check {
	store, storeErr := coreactivation.NewStore(home)
	enrollment, enrolled, enrollmentErr := coreactivation.Enrollment{}, false, storeErr
	if storeErr == nil {
		enrollment, enrolled, enrollmentErr = store.Enrollment()
	}

	progress, progressErr := readProgress(home)

	checks := []Check{updatesCheck(enrolled, enrollment, enrollmentErr, progress, progressErr)}
	if progress != nil && !progress.Complete {
		checks = append(checks, Check{
			Name:    "root_rebaseline",
			Status:  "attention",
			Message: "an interrupted root transition is waiting; run `make rebaseline` to finish it (the same command starts one when none is pending)",
		})
	}
	if check, present := ordinaryBundleCheck(home); present {
		checks = append(checks, check)
	}
	checks = append(checks, supervisorCheck(ctx, home), stateCoverageCheck(ctx, home))
	return checks
}

// ordinaryBundleCheck reports a retained ordinary-bundle transaction that never
// finished. Without this row the operator only learns about it from a refused
// install, and the refusal has to be translated into a bundle path by hand.
func ordinaryBundleCheck(home string) (Check, bool) {
	const name = "root_bundle_recovery"
	_, bundle, err := buildinfo.PreparedBundlePaths(home)
	if err != nil {
		return Check{Name: name, Status: "unknown", Message: "cannot resolve the ordinary bundle transaction: " + err.Error()}, true
	}
	intent, err := buildinfo.ReadPreparedBundleIntent(home)
	if err != nil {
		return Check{Name: name, Status: "attention", Message: "the ordinary bundle recovery intent is unreadable: " + err.Error() + "; resume or discard it from " + bundle}, true
	}
	if intent == nil {
		return Check{}, false
	}
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return Check{Name: name, Status: "unknown", Message: "cannot resolve the runtime bin directory: " + err.Error()}, true
	}
	installer := filepath.Join(bundle, "vrooli-atomic-install")
	return Check{
		Name:    name,
		Status:  "attention",
		Message: fmt.Sprintf("an ordinary bundle install did not finish; finish it with %q --resume-bundle %q, or discard it with %q --abort-bundle %q when the installed images still match the recorded prior state", installer, bin, installer, bin),
	}, true
}

// updatesCheck answers the one question an operator asks first: will a new
// build install by itself, and if not, what unblocked it?
func updatesCheck(enrolled bool, enrollment coreactivation.Enrollment, err error, progress *rebaselineprogress.Progress, progressErr error) Check {
	const name = "root_updates"
	switch {
	case err != nil:
		return Check{Name: name, Status: "attention", Message: "cannot read the root update enrollment: " + err.Error()}
	case progressErr != nil:
		return Check{Name: name, Status: "attention", Message: progressErr.Error()}
	case progress != nil && !progress.Complete:
		// The pending record owns the action; this row reports the effect.
		return Check{Name: name, Status: "attention", Message: "paused by an unfinished root transition"}
	case !enrolled:
		return Check{Name: name, Status: "attention", Message: "automatic root updates are not enrolled; run `vrooli runtime update enroll`"}
	case enrollment.RebaselinePaused:
		return Check{Name: name, Status: "attention", Message: "automatic root updates are paused; run `make rebaseline` to finish the transition"}
	case !enrollment.Enabled:
		return Check{Name: name, Status: "ok", Message: "disabled; `make install` installs directly and keeps its own rollback"}
	default:
		return Check{Name: name, Status: "ok", Message: "enrolled; compatible builds activate automatically"}
	}
}

// supervisorCheck is the same verification a root transition performs, under a
// short deadline. It is the difference between "the unit is active" and "the
// unit is serving the installed image with a fresh lease".
func supervisorCheck(ctx context.Context, home string) Check {
	const name = "root_supervisor"
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return Check{Name: name, Status: "unknown", Message: "cannot resolve the runtime bin directory: " + err.Error()}
	}
	installed := filepath.Join(bin, "vrooli")
	identity, err := coreactivation.ProbeRootIdentity(ctx, installed)
	if err != nil {
		message := "the installed root CLI cannot report its identity: " + oneLine(err) + "; a local build may have replaced the managed image"
		if baseline := newestRetainedBaseline(ctx, home); baseline != "" {
			message += fmt.Sprintf("; the newest retained qualified baseline is %s, so `cp %s %s` restores a serving image (or rebuild with `make install`)", baseline, baseline, installed)
		}
		return Check{Name: name, Status: "attention", Message: message}
	}
	image, err := coreactivation.ImageDigest(ctx, installed)
	if err != nil {
		return Check{Name: name, Status: "attention", Message: "cannot read the installed root CLI: " + err.Error()}
	}
	probe, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := runtimesupervisorsafeguard.WaitServing(probe, home, image, identity.GetBuildIdentity()); err != nil {
		return Check{Name: name, Status: "attention", Message: "the runtime supervisor is not serving the installed image: " + oneLine(err) + "; restart it with `systemctl --user restart vrooli-runtime-supervisor.service`"}
	}
	return Check{Name: name, Status: "ok", Message: "serving the installed image with a fresh lease"}
}

// stateCoverageCheck answers what a forced transition can actually restore. It
// reads the installed image's own manifest, so it reports the deployed truth
// rather than the source tree's, and it needs no store declarations itself.
func stateCoverageCheck(ctx context.Context, home string) Check {
	const name = "root_state_coverage"
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return Check{Name: name, Status: "unknown", Message: "cannot resolve the runtime bin directory: " + err.Error()}
	}
	probe, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	manifest, err := coreactivation.ProbeStateFormat(probe, filepath.Join(bin, "vrooli"))
	if err != nil {
		return Check{Name: name, Status: "unknown", Message: "the installed root CLI cannot report its state format: " + oneLine(err)}
	}
	coverage := manifest.Coverage()
	declared, contractOnly, discardable, external, unmigrated, total := coverage.Declared, coverage.ContractOnly, coverage.Discardable, coverage.External, coverage.Unmigrated, coverage.Total
	message := fmt.Sprintf("a forced transition restores %d of %d stores; %d own no state by contract, %d hold only derived data and %d own bytes outside the runtime home", declared, total, contractOnly, discardable, external)
	if unmigrated > 0 {
		return Check{
			Name:    name,
			Status:  "attention",
			Message: message + fmt.Sprintf("; %d declare no paths yet, so their state is not snapshotted", unmigrated),
		}
	}
	return Check{Name: name, Status: "ok", Message: message + "; every store declares what it owns"}
}

// newestRetainedBaseline finds the newest retained prior image that still
// reports a qualified identity. It gives an operator a concrete recovery when a
// local build has replaced the managed root CLI.
func newestRetainedBaseline(ctx context.Context, home string) string {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return ""
	}
	entries, err := os.ReadDir(filepath.Join(state, "root-rebaseline"))
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "20") {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		candidate := filepath.Join(state, "root-rebaseline", name, "vrooli")
		probe, cancel := context.WithTimeout(ctx, 2*time.Second)
		_, probeErr := coreactivation.ProbeRootIdentity(probe, candidate)
		cancel()
		if probeErr == nil {
			return candidate
		}
	}
	return ""
}

func readProgress(home string) (*rebaselineprogress.Progress, error) {
	state, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return nil, err
	}
	return rebaselineprogress.Read(filepath.Join(state, "root-rebaseline", "progress.json"))
}

func oneLine(err error) string {
	line, _, _ := strings.Cut(err.Error(), "\n")
	return strings.TrimSpace(fmt.Sprint(line))
}
