package credentials

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

// Credential-store repair is a ladder, not an operation.
//
// The file-level repair that came first fixed exactly one fault: a multi-line
// value that made GNOME Keyring reject a plaintext keyring. That is a real
// fault and it is still rung 2 here. But it is one of at least four states an
// operator lands in, and for the other three the command did nothing and said
// nothing — so the answer to "my credentials are broken, what do I run" was a
// command that reported success while the host stayed broken.
//
// The states, in the order they must be attempted:
//
//	1. store-identity   what backend does this host actually use
//	2. keyring-file     is the on-disk keyring parseable
//	3. store-response   does the live store answer
//	4. daemon-reload    if not, restart it and prove the restart helped
//	5. unlock-state     is it locked, and can that be lifted without a human
//
// Two rules hold across every rung. A rung that cannot observe its subject
// reports that it could not, never that the subject is healthy — the defect
// this ladder replaced was exactly a false green. And a rung that ends blocked
// must name what the operator does next, either as a Vrooli command or as an
// explicit statement that no automated remedy exists.

// Rung statuses.
const (
	// RungHealthy means the rung looked and found nothing wrong.
	RungHealthy = "healthy"
	// RungRepaired means the rung changed host state and proved the change
	// worked. It is never set on the strength of a command's exit code.
	RungRepaired = "repaired"
	// RungNotApplicable means this host has no such subject. A macOS host has
	// no GNOME keyring file; that is not a fault.
	RungNotApplicable = "not-applicable"
	// RungBlocked means a fault exists and no automated remedy reaches it.
	// A blocked rung always carries a remedy.
	RungBlocked = "blocked"
	// RungFailed means the rung attempted a repair and the attempt errored.
	RungFailed = "failed"
	// RungSkipped means an earlier rung made this one moot.
	RungSkipped = "skipped"
	// RungUnknown means the rung ran but could not determine an answer. It is
	// deliberately distinct from healthy.
	RungUnknown = "unknown"

	// reloadApplied is internal to the reload seam: the command was accepted,
	// but whether it fixed anything is still unproven. It never escapes into a
	// report — the engine converts it to repaired or blocked after re-probing.
	reloadApplied = "applied"
)

// Rung names.
const (
	RungStoreIdentity = "store-identity"
	RungKeyringFile   = "keyring-file"
	RungStoreResponse = "store-response"
	RungDaemonReload  = "daemon-reload"
	RungUnlockState   = "unlock-state"
)

// Rung is one step of the ladder.
type Rung struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	// Action is the command this rung executed against the host. Empty when the
	// rung only observed.
	Action string `json:"action,omitempty"`
}

// RepairReport is the whole ladder's result.
type RepairReport struct {
	Platform string `json:"platform"`
	Adapter  string `json:"adapter,omitempty"`
	// StateBefore and StateAfter are the live store states either side of the
	// ladder. They are the honest measure of whether anything improved; the
	// per-rung statuses explain why.
	StateBefore string `json:"stateBefore"`
	StateAfter  string `json:"stateAfter"`
	Rungs       []Rung `json:"rungs"`
	// Resolved is true only when the live store answers at the end. A ladder
	// that repaired a file but left the store unreachable is not resolved.
	Resolved bool `json:"resolved"`
	// Remedy is what the operator does next. Empty when Resolved.
	Remedy []string `json:"remedy,omitempty"`
	// File is the rung-2 detail, present only when a keyring file was examined.
	File *KeyringReport `json:"file,omitempty"`
}

func (r *RepairReport) add(rung Rung) { r.Rungs = append(r.Rungs, rung) }

// storeReady is the one state that means the credential store works. Every
// other state the host-fact probe can return is a fault or an unknown.
const storeReady = "ready"

// probeCredentialStore is the live-store seam. It is a variable so the ladder's
// decision table can be tested without a Secret Service, and it points at
// hostinventory because that package is the project's single authority on what
// this host is — this file must not grow a second opinion.
var probeCredentialStore = hostinventory.CredentialStoreStatus

// repairKeyringFileAt is the rung-2 seam, held separately so a test can drive
// the ladder against a fixture file without a platform keyring.
var repairKeyringFileAt = Repair

// storeAdapterName is the rung-1 seam.
var storeAdapterName = func() string { return securestore.AdapterName(securestore.Default()) }

// RepairStore walks the ladder and returns what it found and what it changed.
//
// It never prints, reads, or returns a credential value. path selects a keyring
// file for rung 2; empty means the platform default, and on a platform with no
// keyring file the rung reports not-applicable.
func RepairStore(ctx context.Context, path string) (RepairReport, error) {
	report := RepairReport{Platform: runtime.GOOS}

	// Rung 1 — identity. Named first because every later rung's meaning depends
	// on which backend this host actually uses, and because an operator reading
	// a repair transcript needs to know what was being repaired.
	report.Adapter = storeAdapterName()
	report.add(Rung{
		Name:   RungStoreIdentity,
		Status: RungHealthy,
		Detail: fmt.Sprintf("platform %s, credential adapter %s", runtime.GOOS, report.Adapter),
	})

	// Rung 3's probe is taken first so StateBefore describes the host as found,
	// not as it looked after a file rewrite.
	before := probeCredentialStore(ctx)
	report.StateBefore = before.State

	// Rung 2 — the on-disk keyring file.
	report.add(repairKeyringFile(path, &report))

	// Rung 3 — does the live store answer.
	if before.State == storeReady {
		report.add(Rung{Name: RungStoreResponse, Status: RungHealthy, Detail: "the credential store answered a bounded property read"})
		report.add(Rung{Name: RungDaemonReload, Status: RungSkipped, Detail: "the store already answers; a restart would drop a working session for nothing"})
		report.add(Rung{Name: RungUnlockState, Status: RungHealthy, Detail: "the store is unlocked"})
		report.StateAfter = before.State
		report.Resolved = true
		return report, nil
	}
	if !before.Supported {
		report.add(Rung{Name: RungStoreResponse, Status: RungNotApplicable, Detail: reasonOr(before.Reason, "this platform has no probeable credential store")})
		report.add(Rung{Name: RungDaemonReload, Status: RungSkipped, Detail: "no probeable store to reload"})
		report.add(Rung{Name: RungUnlockState, Status: RungSkipped, Detail: "no probeable store to unlock"})
		report.StateAfter = before.State
		report.Remedy = []string{"Run `vrooli credentials doctor` to see which backend this platform uses and whether it is writable."}
		return report, nil
	}
	report.add(Rung{
		Name:   RungStoreResponse,
		Status: faultStatus(before.State),
		Detail: fmt.Sprintf("the credential store reported %q: %s", before.State, reasonOr(before.Reason, "no reason given")),
	})

	// Rung 4 — reload, but only for a fault a reload can actually clear. A
	// locked store is not wedged; restarting it would discard an unlock the
	// operator may already have supplied and gains nothing.
	if before.State == "locked" {
		report.add(Rung{Name: RungDaemonReload, Status: RungSkipped, Detail: "the store answers but is locked; a restart cannot supply a passphrase and would discard any unlock already held"})
		return finishLocked(ctx, &report), nil
	}

	outcome := reloadCredentialDaemon(ctx)
	switch outcome.Status {
	case reloadApplied:
		after := probeCredentialStore(ctx)
		report.StateAfter = after.State
		// The re-probe, not the exit code, decides. This is the same discipline
		// the phase-cache audit uses: a claim that the work was done is not
		// evidence that it was.
		if after.State == storeReady {
			report.add(Rung{Name: RungDaemonReload, Status: RungRepaired, Action: outcome.Action, Detail: "restarted the credential daemon; the store now answers"})
			report.add(Rung{Name: RungUnlockState, Status: RungHealthy, Detail: "the store is unlocked"})
			report.Resolved = true
			return report, nil
		}
		if after.State == "locked" {
			report.add(Rung{Name: RungDaemonReload, Status: RungRepaired, Action: outcome.Action, Detail: "restarted the credential daemon; the store answers again but is locked"})
			return finishLocked(ctx, &report), nil
		}
		report.add(Rung{
			Name:   RungDaemonReload,
			Status: RungFailed,
			Action: outcome.Action,
			Detail: fmt.Sprintf("restarted the credential daemon, but the store still reports %q", after.State),
		})
		report.add(Rung{Name: RungUnlockState, Status: RungSkipped, Detail: "the store does not answer, so its lock state cannot be read"})
		report.Remedy = relogRemedy("restarting the credential daemon did not make the store answer")
		return report, nil
	case RungBlocked, RungNotApplicable, RungFailed:
		report.add(Rung{Name: RungDaemonReload, Status: outcome.Status, Action: outcome.Action, Detail: outcome.Detail})
		report.add(Rung{Name: RungUnlockState, Status: RungSkipped, Detail: "the store does not answer, so its lock state cannot be read"})
		report.StateAfter = before.State
		report.Remedy = outcome.Remedy
		if len(report.Remedy) == 0 {
			report.Remedy = relogRemedy("no automated reload reached this host's credential store")
		}
		return report, nil
	default:
		// A reload implementation returning something this engine does not know
		// is a defect in that implementation, and saying so beats guessing.
		report.add(Rung{Name: RungDaemonReload, Status: RungUnknown, Detail: fmt.Sprintf("the platform reload seam returned an unrecognized status %q", outcome.Status)})
		report.StateAfter = before.State
		return report, fmt.Errorf("credential daemon reload returned unrecognized status %q", outcome.Status)
	}
}

// finishLocked closes out the ladder for a store that answers but is locked.
// This is the one fault with a real Vrooli command behind it, so the remedy
// names it rather than sending the operator to a login screen.
func finishLocked(ctx context.Context, report *RepairReport) RepairReport {
	after := probeCredentialStore(ctx)
	report.StateAfter = after.State
	report.add(Rung{
		Name:   RungUnlockState,
		Status: RungBlocked,
		Detail: "the store is locked; unlocking it requires a passphrase this process does not hold and must never store",
	})
	report.Remedy = []string{
		"Pipe the login-keyring passphrase to `vrooli credentials keyring unlock`, then rerun `vrooli credentials keyring repair` to confirm.",
		"Or log out and back in: PAM unlocks the login keyring with your login password automatically.",
	}
	return *report
}

// repairKeyringFile runs rung 2 and attaches its detail to the report.
//
// A missing keyring file is not-applicable rather than an error: a host using
// the encrypted file store, the macOS Keychain, or the Windows Credential
// Manager has no keyring file and is not broken for lacking one.
func repairKeyringFile(path string, report *RepairReport) Rung {
	file, err := repairKeyringFileAt(path)
	switch {
	case err == nil:
	case errors.Is(err, fs.ErrNotExist), isNoKeyringFiles(err):
		return Rung{Name: RungKeyringFile, Status: RungNotApplicable, Detail: "this host has no GNOME keyring file; its credentials live in another backend"}
	default:
		return Rung{Name: RungKeyringFile, Status: RungFailed, Detail: err.Error()}
	}
	report.File = &file

	switch {
	case file.Repaired > 0:
		detail := fmt.Sprintf("rewrote %d malformed entr%s in %s", file.Repaired, plural(file.Repaired), file.Path)
		if file.BackupPath != "" {
			detail += "; original backed up to " + file.BackupPath
		}
		if file.StaleDaemon {
			detail += ". The running daemon still holds the pre-repair state, which the next rung addresses"
		}
		return Rung{Name: RungKeyringFile, Status: RungRepaired, Detail: detail}
	case !file.Assessed:
		// The false green this ladder was built to remove. An encrypted keyring
		// is opaque, so the honest report is "could not look", not "looks fine".
		return Rung{
			Name:   RungKeyringFile,
			Status: RungUnknown,
			Detail: fmt.Sprintf("%s is an %s keyring; its bytes are opaque to file inspection, so no verdict on its contents is possible", file.Path, formatLabel(file.Format)),
		}
	case len(file.Defects) > 0:
		return Rung{
			Name:   RungKeyringFile,
			Status: RungBlocked,
			Detail: fmt.Sprintf("%s holds %d malformed entr%s Vrooli does not own and will not rewrite; collapsing another application's value means guessing how it reads that value back", file.Path, len(file.Defects), plural(len(file.Defects))),
		}
	case file.StaleDaemon:
		return Rung{Name: RungKeyringFile, Status: RungHealthy, Detail: file.StaleDaemonDetail}
	default:
		return Rung{Name: RungKeyringFile, Status: RungHealthy, Detail: file.Path + " parses cleanly"}
	}
}

// faultStatus maps a live store state onto a rung status. "unresponsive" is a
// fault a reload may clear; the rest are conditions the ladder reports but does
// not treat as repairable here.
func faultStatus(state string) string {
	switch state {
	case "unresponsive":
		return RungFailed
	case "unsupported":
		return RungNotApplicable
	default:
		return RungBlocked
	}
}

func isNoKeyringFiles(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no keyring files found")
}

func formatLabel(format string) string {
	if strings.TrimSpace(format) == "" {
		return "unrecognized"
	}
	return format
}

func reasonOr(reason, fallback string) string {
	if strings.TrimSpace(reason) == "" {
		return fallback
	}
	return reason
}

func plural(count int) string {
	if count == 1 {
		return "y"
	}
	return "ies"
}
