package credentials

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

// stubLadder replaces every seam so a case describes one host shape exactly.
// Each field is restored by the returned cleanup, because these are package
// variables and a leaked stub would make an unrelated test pass for the wrong
// reason.
func stubLadder(t *testing.T, probe func(context.Context) hostinventory.CredentialStoreCapability, file func(string) (KeyringReport, error), reload func(context.Context) ReloadOutcome) {
	t.Helper()
	origProbe, origFile, origReload, origAdapter := probeCredentialStore, repairKeyringFileAt, reloadCredentialDaemon, storeAdapterName
	probeCredentialStore = probe
	repairKeyringFileAt = file
	reloadCredentialDaemon = reload
	storeAdapterName = func() string { return "test-adapter" }
	t.Cleanup(func() {
		probeCredentialStore, repairKeyringFileAt, reloadCredentialDaemon, storeAdapterName = origProbe, origFile, origReload, origAdapter
	})
}

func constProbe(state string, supported bool) func(context.Context) hostinventory.CredentialStoreCapability {
	return func(context.Context) hostinventory.CredentialStoreCapability {
		return hostinventory.CredentialStoreCapability{State: state, Supported: supported, Observed: true}
	}
}

// sequenceProbe answers each call with the next state, so a case can express
// "unresponsive before the reload, ready after it".
func sequenceProbe(states ...string) func(context.Context) hostinventory.CredentialStoreCapability {
	call := 0
	return func(context.Context) hostinventory.CredentialStoreCapability {
		state := states[len(states)-1]
		if call < len(states) {
			state = states[call]
		}
		call++
		return hostinventory.CredentialStoreCapability{State: state, Supported: true, Observed: true}
	}
}

func cleanFile(string) (KeyringReport, error) {
	return KeyringReport{Path: "/fixture/login.keyring", Format: "plaintext", Assessed: true, Loadable: true}, nil
}

func noFile(string) (KeyringReport, error) {
	return KeyringReport{}, errors.New("no keyring files found")
}

func rungByName(t *testing.T, report RepairReport, name string) Rung {
	t.Helper()
	for _, rung := range report.Rungs {
		if rung.Name == name {
			return rung
		}
	}
	t.Fatalf("report has no %s rung; rungs=%+v", name, report.Rungs)
	return Rung{}
}

func neverReload(context.Context) ReloadOutcome {
	return ReloadOutcome{Status: RungFailed, Detail: "reload should not have been attempted"}
}

func TestRepairStoreHealthyStoreDoesNotRestartAnything(t *testing.T) {
	reloaded := false
	stubLadder(t, constProbe("ready", true), cleanFile, func(context.Context) ReloadOutcome {
		reloaded = true
		return ReloadOutcome{Status: reloadApplied}
	})

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	if reloaded {
		t.Fatal("a healthy store was restarted; a working session must never be dropped for nothing")
	}
	if !report.Resolved {
		t.Fatalf("Resolved = false for a ready store; report=%+v", report)
	}
	if got := rungByName(t, report, RungDaemonReload).Status; got != RungSkipped {
		t.Fatalf("daemon-reload status = %q, want %q", got, RungSkipped)
	}
	if len(report.Remedy) != 0 {
		t.Fatalf("a resolved ladder carried a remedy: %v", report.Remedy)
	}
}

func TestRepairStoreReloadProvenByReprobeNotExitCode(t *testing.T) {
	stubLadder(t, sequenceProbe("unresponsive", "ready"), cleanFile, func(context.Context) ReloadOutcome {
		return ReloadOutcome{Status: reloadApplied, Action: "systemctl --user restart gnome-keyring-daemon.service"}
	})

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	if !report.Resolved {
		t.Fatalf("Resolved = false after a reload that made the store answer; report=%+v", report)
	}
	if report.StateBefore != "unresponsive" || report.StateAfter != "ready" {
		t.Fatalf("state transition = %q -> %q, want unresponsive -> ready", report.StateBefore, report.StateAfter)
	}
	rung := rungByName(t, report, RungDaemonReload)
	if rung.Status != RungRepaired {
		t.Fatalf("daemon-reload status = %q, want %q", rung.Status, RungRepaired)
	}
	if rung.Action == "" {
		t.Fatal("a rung that mutated the host recorded no action; the operator cannot see what was run")
	}
}

// The whole point of re-probing: a reload command that succeeds while the store
// stays broken must not be reported as a repair.
func TestRepairStoreReloadThatDidNotHelpIsNotRepaired(t *testing.T) {
	stubLadder(t, constProbe("unresponsive", true), cleanFile, func(context.Context) ReloadOutcome {
		return ReloadOutcome{Status: reloadApplied, Action: "systemctl --user restart gnome-keyring-daemon.service"}
	})

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	if report.Resolved {
		t.Fatal("Resolved = true while the store still reported unresponsive")
	}
	if got := rungByName(t, report, RungDaemonReload).Status; got != RungFailed {
		t.Fatalf("daemon-reload status = %q, want %q", got, RungFailed)
	}
	if len(report.Remedy) == 0 {
		t.Fatal("an unresolved ladder carried no remedy; that is a dead end")
	}
}

func TestRepairStoreLockedStoreIsNotRestarted(t *testing.T) {
	stubLadder(t, constProbe("locked", true), cleanFile, neverReload)

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	if got := rungByName(t, report, RungDaemonReload).Status; got != RungSkipped {
		t.Fatalf("daemon-reload status = %q, want %q — restarting a locked store discards an unlock it may already hold", got, RungSkipped)
	}
	if got := rungByName(t, report, RungUnlockState).Status; got != RungBlocked {
		t.Fatalf("unlock-state status = %q, want %q", got, RungBlocked)
	}
	if !remedyMentions(report.Remedy, "vrooli credentials keyring unlock") {
		t.Fatalf("locked remedy = %v, want the unlock command", report.Remedy)
	}
}

// An encrypted keyring is opaque to file inspection. Reporting it as healthy is
// the false green that let a wedged daemon read as fine for four days.
func TestRepairStoreOpaqueKeyringIsUnknownNotHealthy(t *testing.T) {
	opaque := func(string) (KeyringReport, error) {
		return KeyringReport{Path: "/fixture/login.keyring", Format: "encrypted", Assessed: false}, nil
	}
	stubLadder(t, constProbe("ready", true), opaque, neverReload)

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	rung := rungByName(t, report, RungKeyringFile)
	if rung.Status != RungUnknown {
		t.Fatalf("keyring-file status = %q, want %q for an opaque file", rung.Status, RungUnknown)
	}
	if !strings.Contains(rung.Detail, "encrypted") {
		t.Fatalf("keyring-file detail = %q, want it to name the format", rung.Detail)
	}
}

// A host with no GNOME keyring file is a normal host, not a broken one.
func TestRepairStoreMissingKeyringFileIsNotApplicable(t *testing.T) {
	stubLadder(t, constProbe("ready", true), noFile, neverReload)

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	if got := rungByName(t, report, RungKeyringFile).Status; got != RungNotApplicable {
		t.Fatalf("keyring-file status = %q, want %q", got, RungNotApplicable)
	}
	if !report.Resolved {
		t.Fatal("a ready store with no keyring file should resolve")
	}
}

func TestRepairStoreUnsupportedPlatformReportsBackendNotFault(t *testing.T) {
	stubLadder(t, constProbe("unsupported", false), noFile, neverReload)

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	if got := rungByName(t, report, RungStoreResponse).Status; got != RungNotApplicable {
		t.Fatalf("store-response status = %q, want %q", got, RungNotApplicable)
	}
	if !remedyMentions(report.Remedy, "vrooli credentials doctor") {
		t.Fatalf("remedy = %v, want it to point at the backend diagnostic", report.Remedy)
	}
}

// A blocked reload must carry a remedy. A rung that says "cannot proceed" and
// nothing else is the failure mode this ladder exists to remove.
func TestRepairStoreBlockedReloadAlwaysCarriesRemedy(t *testing.T) {
	stubLadder(t, constProbe("unresponsive", true), cleanFile, func(context.Context) ReloadOutcome {
		return ReloadOutcome{Status: RungBlocked, Detail: "no systemd user manager on this host"}
	})

	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	if report.Resolved {
		t.Fatal("Resolved = true for a blocked reload")
	}
	if len(report.Remedy) == 0 {
		t.Fatal("blocked reload produced no remedy")
	}
}

func TestRepairStoreRejectsUnknownReloadStatus(t *testing.T) {
	stubLadder(t, constProbe("unresponsive", true), cleanFile, func(context.Context) ReloadOutcome {
		return ReloadOutcome{Status: "made-up"}
	})

	if _, err := RepairStore(context.Background(), ""); err == nil {
		t.Fatal("an unrecognized reload status was accepted; a platform seam defect must surface, not be guessed at")
	}
}

// Every rung the engine can emit must be one of the declared statuses. A typo
// would otherwise reach an operator as a status nothing renders.
func TestRepairStoreEmitsOnlyDeclaredStatuses(t *testing.T) {
	declared := map[string]bool{
		RungHealthy: true, RungRepaired: true, RungNotApplicable: true,
		RungBlocked: true, RungFailed: true, RungSkipped: true, RungUnknown: true,
	}
	cases := []struct {
		name   string
		probe  func(context.Context) hostinventory.CredentialStoreCapability
		reload func(context.Context) ReloadOutcome
	}{
		{"ready", constProbe("ready", true), neverReload},
		{"locked", constProbe("locked", true), neverReload},
		{"unsupported", constProbe("unsupported", false), neverReload},
		{"reload-helps", sequenceProbe("unresponsive", "ready"), func(context.Context) ReloadOutcome { return ReloadOutcome{Status: reloadApplied} }},
		{"reload-locks", sequenceProbe("unresponsive", "locked"), func(context.Context) ReloadOutcome { return ReloadOutcome{Status: reloadApplied} }},
		{"reload-fails", constProbe("unresponsive", true), func(context.Context) ReloadOutcome { return ReloadOutcome{Status: RungFailed, Detail: "x"} }},
		{"empty", constProbe("empty", true), func(context.Context) ReloadOutcome { return ReloadOutcome{Status: RungBlocked, Detail: "x"} }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stubLadder(t, testCase.probe, cleanFile, testCase.reload)
			report, err := RepairStore(context.Background(), "")
			if err != nil {
				t.Fatalf("RepairStore: %v", err)
			}
			for _, rung := range report.Rungs {
				if !declared[rung.Status] {
					t.Fatalf("rung %s carried undeclared status %q", rung.Name, rung.Status)
				}
			}
			if !report.Resolved && len(report.Remedy) == 0 {
				t.Fatalf("unresolved ladder for %s carried no remedy", testCase.name)
			}
		})
	}
}

// The engine must never leak its internal "applied" marker into a report; a
// consumer switching on rung status would have no case for it.
func TestRepairStoreNeverEmitsInternalAppliedStatus(t *testing.T) {
	stubLadder(t, sequenceProbe("unresponsive", "ready"), cleanFile, func(context.Context) ReloadOutcome {
		return ReloadOutcome{Status: reloadApplied}
	})
	report, err := RepairStore(context.Background(), "")
	if err != nil {
		t.Fatalf("RepairStore: %v", err)
	}
	for _, rung := range report.Rungs {
		if rung.Status == reloadApplied {
			t.Fatalf("rung %s leaked the internal %q status", rung.Name, reloadApplied)
		}
	}
}

func remedyMentions(remedy []string, want string) bool {
	for _, line := range remedy {
		if strings.Contains(line, want) {
			return true
		}
	}
	return false
}

// The keyring format classifier is what decides whether Loadable carries a
// verdict at all, so it is asserted directly against real leading bytes.
func TestKeyringFormatClassification(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name     string
		contents string
		format   string
		assessed bool
	}{
		{"plaintext", "[keyring]\ndisplay-name=login\n", "plaintext", true},
		{"encrypted", "GnomeKeyring\n\r\x00\n\x00\x00\x00\x00", "encrypted", false},
		{"unrecognized", "not a keyring at all", "unknown", false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			path := dir + "/" + testCase.name + ".keyring"
			writeFixture(t, path, testCase.contents)
			report, err := securestore.InspectKeyringFile(path)
			if err != nil {
				t.Fatalf("InspectKeyringFile: %v", err)
			}
			if report.Format != testCase.format {
				t.Fatalf("Format = %q, want %q", report.Format, testCase.format)
			}
			if report.Assessed != testCase.assessed {
				t.Fatalf("Assessed = %t, want %t", report.Assessed, testCase.assessed)
			}
			if !report.Assessed && report.Loadable {
				t.Fatal("Loadable = true on a file that was never assessed; that is the false green this field exists to prevent")
			}
		})
	}
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
