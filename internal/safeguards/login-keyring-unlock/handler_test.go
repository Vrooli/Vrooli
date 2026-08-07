package loginkeyringunlock

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func testRequirement(choice hostreqspec.OperatorChoice) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "login_keyring_unlock", Kind: hostreqspec.KindSafeguard,
		OperatorChoice: choice,
	}
}

func linuxHost() hostreqkit.Host { return hostreqkit.Host{OS: "linux"} }

func restoreSeams(t *testing.T) {
	t.Helper()
	oldFacts, oldPath, oldPasswordless, oldBackup, oldRun, oldRunOutput := collectFactsFn, keyringPathFn, passwordlessFn, backupFn, runUserFn, runUserOutputFn
	t.Cleanup(func() {
		collectFactsFn, keyringPathFn, passwordlessFn, backupFn, runUserFn, runUserOutputFn = oldFacts, oldPath, oldPasswordless, oldBackup, oldRun, oldRunOutput
	})
}

func TestLoginKeyringUnlockIsInertWithoutOptIn(t *testing.T) {
	restoreSeams(t)
	called := false
	collectFactsFn = func() hostinventory.Snapshot { called = true; return hostinventory.Snapshot{AutoLoginUser: "alice"} }
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "login_keyring_unlock"})
	status := h.Inspect(linuxHost(), testRequirement(hostreqspec.OperatorChoiceNotRecorded))
	if status.ExecutionState != hostreqkit.ExecutionPending || status.Applied {
		t.Fatalf("not-recorded inspect = %+v, want pending and unapplied", status)
	}
	if called {
		t.Fatal("non-opted-in inspection touched host facts")
	}
	status, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil || status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("not-recorded apply = %+v, %v", status, err)
	}
}

func TestLoginKeyringUnlockNotApplicableWithoutAutologin(t *testing.T) {
	restoreSeams(t)
	collectFactsFn = func() hostinventory.Snapshot { return hostinventory.Snapshot{} }
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "login_keyring_unlock"}).Inspect(linuxHost(), testRequirement(hostreqspec.OperatorChoiceOptedIn))
	if status.SupportClass != hostreqkit.SupportNotApplicable || status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("no-autologin status = %+v, want not applicable", status)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "no autologin") {
		t.Fatalf("no-autologin notes = %v", status.Notes)
	}
}

func TestLoginKeyringUnlockRefusesToWriteWhenBackupFails(t *testing.T) {
	restoreSeams(t)
	collectFactsFn = func() hostinventory.Snapshot { return hostinventory.Snapshot{AutoLoginUser: "alice"} }
	keyringPathFn = func(string) (string, error) { return "/home/alice/.local/share/keyrings/login.keyring", nil }
	passwordlessFn = func(string) (bool, error) { return false, nil }
	backupFn = func(string) (string, error) { return "", errors.New("read-only filesystem") }
	runCalled := false
	runUserFn = func(string, []string, hostreqkit.EnsureOptions) error { runCalled = true; return nil }

	h := NewHandler(hostreqkit.SafeguardManifest{Name: "login_keyring_unlock"})
	status := h.Inspect(linuxHost(), testRequirement(hostreqspec.OperatorChoiceOptedIn))
	status, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("backup failure state = %+v, want failed", status)
	}
	if runCalled {
		t.Fatal("keyring command ran after backup failure")
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "backup failed") {
		t.Fatalf("backup failure notes = %v", status.Notes)
	}
}

func TestLoginKeyringUnlockOpensPromptAndWaitsForVerification(t *testing.T) {
	restoreSeams(t)
	collectFactsFn = func() hostinventory.Snapshot { return hostinventory.Snapshot{AutoLoginUser: "alice"} }
	keyringPathFn = func(string) (string, error) { return "/home/alice/.local/share/keyrings/login.keyring", nil }
	passwordlessFn = func(string) (bool, error) { return false, nil }
	backupFn = func(string) (string, error) {
		return "/home/alice/.local/share/keyrings/login.keyring.vrooli-login-unlock-backup", nil
	}

	var outputName string
	var outputArgs []string
	runUserOutputFn = func(name string, args []string, _ hostreqkit.EnsureOptions) ([]byte, error) {
		outputName, outputArgs = name, append([]string(nil), args...)
		return []byte("(objectpath '/org/gnome/keyring/Prompt/p1',)"), nil
	}
	var promptArgs []string
	runUserFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		if name != "gdbus" {
			t.Fatalf("prompt command = %q, want gdbus", name)
		}
		promptArgs = append([]string(nil), args...)
		return nil
	}

	h := NewHandler(hostreqkit.SafeguardManifest{Name: "login_keyring_unlock"})
	status := h.Inspect(linuxHost(), testRequirement(hostreqspec.OperatorChoiceOptedIn))
	status, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.Applied || status.ExecutionState != hostreqkit.ExecutionManualActionRequired || status.BlockingReason != hostreqkit.BlockingManual {
		t.Fatalf("prompt status = %+v, want manual verification", status)
	}
	if outputName != "gdbus" || !strings.Contains(strings.Join(outputArgs, " "), "ChangeWithPrompt") || strings.Contains(strings.Join(outputArgs, " "), "Collection.ChangeLock") {
		t.Fatalf("prompt request = %s %v", outputName, outputArgs)
	}
	joinedPrompt := strings.Join(promptArgs, " ")
	if !strings.Contains(joinedPrompt, "org.freedesktop.Secret.Prompt.Prompt") || !strings.Contains(joinedPrompt, "/org/gnome/keyring/Prompt/p1") {
		t.Fatalf("prompt display command = %v", promptArgs)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "choose a blank new password") {
		t.Fatalf("prompt status notes = %v", status.Notes)
	}
}
