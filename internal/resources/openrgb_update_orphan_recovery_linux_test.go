//go:build linux

package resources

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Reproduce the operator's old terminal ledger with an admitted role and no
// exec file. Exercise the real retirement/store/native-owner code; only the
// manager's fixed read-only command response is a disposable fixture.
func TestOpenRGBOrphanedTerminalRoleUsesSupportedDisableRetirement(t *testing.T) {
	c, manifest, _ := openRGBHeadlessFixture(t)
	owner := NewOpenRGBHeadless(c, func(_ context.Context, _ OpenRGBDevicePolicyCheck) error {
		t.Fatal("disabled recovery invoked hardware qualification")
		return nil
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	port := listener.Addr().(*net.TCPAddr).Port
	manifest.Ports[0].Host, manifest.Ports[0].Container = port, port
	writeResourceManifest(t, c.Root, openRGBResourceName, manifest)
	x := newOpenRGBExecFixtureAtPort(t, port)
	u := updateTestClone(t, x.update)
	u.PrimaryStopVerified, u.PrimaryQuiescent, u.CandidateQuiescent = true, true, true
	u.Restored.Quiescence = &ManagedUpdateQuiescence{InstanceID: u.Restored.Intent.InstanceID, OwnershipTokenHash: u.Restored.Intent.OwnershipTokenHash, ObservedAt: time.Now().UTC(), NativeAbsent: true, ProcessesAbsent: true, ListenerAbsent: true}
	store, err := newManagedUpdateStore(openRGBResourceName)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.save(managedUpdateLedger{SchemaVersion: 1, Resource: u.Resource, Records: []managedUpdateEntry{{State: x.update}}}); err != nil {
		t.Fatal(err)
	}
	u, err = store.Commit(t.Context(), u, UpdateFinishDisabled)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureOpenRGBUpdateFencesRetired(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing fence was accepted: %v", err)
	}
	unit, err := u.Restored.Intent.NativeIntent.UnitName()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	// No unit, service, device or payload is launched. An invalid manager
	// response first proves revocation alone cannot qualify permission edits.
	manager := filepath.Join(bin, "systemctl")
	if err := os.WriteFile(manager, []byte("#!/bin/sh\nexit 91\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if err := owner.DisableOpenRGB(t.Context()); err == nil {
		t.Fatal("uncertain native absence was retired")
	}
	if err := ensureOpenRGBUpdateFencesRetired(); err == nil || !strings.Contains(err.Error(), "cleanup acknowledgement") {
		t.Fatalf("revoked tombstone bypassed failed cleanup: %v", err)
	}
	// The platform owner requires not-found unit plus absent exact cgroup.
	// This test-only namespace is beneath /sys/fs/cgroup and does not exist.
	base := "/vrooli-test-absent-" + filepath.Base(bin) + "/vrooli-services.slice"
	if _, err := os.Stat(filepath.Join("/sys/fs/cgroup", base, unit)); !os.IsNotExist(err) {
		t.Fatalf("fixture cgroup absence unknown: %v", err)
	}
	script := "#!/bin/sh\nfor arg do last=$arg; done\ncase \"$last\" in\n" + unit + ")\nprintf '%s\\n' 'LoadState=not-found' 'ActiveState=inactive' 'SubState=dead' 'ControlGroup=' 'InvocationID=' 'Result=success' 'MainPID=0' 'ExecMainCode=0' 'ExecMainStatus=0' 'Job='\n;;\nvrooli-services.slice)\nprintf '%s\\n' 'LoadState=loaded' 'ControlGroup=" + base + "'\n;;\n*) exit 92;;\nesac\n"
	if err := os.WriteFile(manager, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	// A fixture-owned live listener must block retirement even after the exact
	// native owner is absent. The real host OpenRGB port is never involved.
	if err := owner.DisableOpenRGB(t.Context()); err == nil || !strings.Contains(err.Error(), "listener absence") {
		t.Fatalf("occupied fixture port was accepted as retired: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := owner.DisableOpenRGB(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := ensureOpenRGBUpdateFencesRetired(); err != nil {
			t.Fatal(err)
		}
	}
	after, err := os.ReadFile(store.path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("recovery rewrote terminal historical ledger: %v", err)
	}
	fences, err := newManagedUpdateExecStore(u.Resource, u.ID, u.Restored.Intent.InstanceID)
	if err != nil {
		t.Fatal(err)
	}
	f, err := fences.read()
	if err != nil || f.Claim != nil || f.NativeRetiredAt == nil || f.State != managedUpdateExecFenceRevoked {
		t.Fatalf("recovery lacks exact retired tombstone: %+v %v", f, err)
	}
	if _, err := fences.Claim(t.Context(), x.fence, x.ref); !errors.Is(err, ErrManagedUpdateExecRevoked) {
		t.Fatalf("recovery created execution permission: %v", err)
	}
}

func TestManagedUpdateNativeRetirementAcknowledgementIsDurableAndFailClosed(t *testing.T) {
	for _, visible := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-write", true: "after-write"}[visible], func(t *testing.T) {
			f, _ := execFenceTestInitial(t)
			s := execFenceTestStore(filepath.Join(t.TempDir(), "fence.json"), f)
			if err := s.recordNativeRetired(t.Context(), f); !os.IsNotExist(err) {
				t.Fatalf("retirement invented missing revocation: %v", err)
			}
			if _, err := s.Prepare(t.Context(), f); err != nil {
				t.Fatal(err)
			}
			if err := s.recordNativeRetired(t.Context(), f); !errors.Is(err, ErrManagedUpdateConflict) {
				t.Fatalf("unrevoked exec acknowledged as retired: %v", err)
			}
			if _, err := s.Revoke(t.Context(), f); err != nil {
				t.Fatal(err)
			}
			s.write = func(path string, data []byte, mode os.FileMode) error {
				if visible {
					if err := writeResourceOwnedFileDurable(path, data, mode); err != nil {
						return err
					}
				}
				return errors.New("lost native retirement acknowledgement")
			}
			if err := s.recordNativeRetired(t.Context(), f); !errors.Is(err, ErrManagedUpdateCommitUnknown) {
				t.Fatalf("uncertain persistence claimed success: %v", err)
			}
			s = execFenceTestStore(s.path, f)
			writes := 0
			s.write = func(path string, data []byte, mode os.FileMode) error {
				writes++
				return writeResourceOwnedFileDurable(path, data, mode)
			}
			if err := s.recordNativeRetired(t.Context(), f); err != nil {
				t.Fatal(err)
			}
			if writes != 1 {
				t.Fatal("visible uncertain acknowledgement was not durably reconfirmed")
			}
			after, err := s.read()
			if err != nil || after.NativeRetiredAt == nil || after.State != managedUpdateExecFenceRevoked {
				t.Fatal("retry lost revocation or completed cleanup acknowledgement")
			}
		})
	}
}
