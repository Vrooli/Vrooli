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
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
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
	checks = append(checks, supervisorCheck(ctx, home))
	return checks
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
		return Check{Name: name, Status: "attention", Message: "the installed root CLI cannot report its identity: " + err.Error()}
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
