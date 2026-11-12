package resources

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

)

func updateCredentialTestStore(t *testing.T, u ManagedServiceUpdate) (*managedUpdateCredentialStore, *string, *int) {
	t.Helper()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	paths, err := resourceStoragePaths(u.Resource)
	if err != nil {
		t.Fatal(err)
	}
	s, err := newManagedUpdateCredentialStore(paths.StateDir, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	value := new(string)
	*value = "private-operation-fixture"
	calls := new(int)
	s.resolve = func(context.Context, ManagedUpdateCredentialItem) (string, error) { *calls++; return *value, nil }
	return s, value, calls
}

func TestManagedUpdateCredentialsRemainPinnedAcrossAuthorityRotationAndRoleCopies(t *testing.T) {
	u := updateTestInitial(t)
	s, value, calls := updateCredentialTestStore(t, u)
	snapshot, err := s.Prepare(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	u.Credentials = snapshot
	encoded, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), *value) {
		t.Fatal("credential value entered checkpoint")
	}
	*value = "rotated-authority-fixture"
	retried, err := s.Prepare(context.Background(), u)
	if err != nil || !reflect.DeepEqual(snapshot, retried) {
		t.Fatal("snapshot changed after authority rotation")
	}
	if *calls != 1 {
		t.Fatal("committed snapshot refetched live authority")
	}
	r := updateLaunchTestRuntime(t)
	r.StateDir = filepath.Dir(filepath.Dir(filepath.Dir(s.dir)))
	for _, instance := range []string{"candidate", "primary", "restored"} {
		invocation, err := renderManagedUpdateInvocation(u.Target, r, u.ID, instance, "role-private-token")
		if err != nil {
			t.Fatal(err)
		}
		if err := materializeManagedUpdateInvocation(u, u.Target, invocation); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(managedServiceEnvValues(invocation.Env)["RESOURCE_STATE_DIR"], u.Target.Manifest.ManagedService.CredentialFiles[0].Path)
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "private-operation-fixture\n" {
			t.Fatal("role did not receive operation-pinned credential")
		}
	}
}

func TestManagedUpdateCredentialSnapshotCorruptionNeverRefetchesOrReplacesCommittedInputs(t *testing.T) {
	for _, mode := range []string{"value", "key", "missing-value", "missing-key", "metadata", "alias"} {
		t.Run(mode, func(t *testing.T) {
			u := updateTestInitial(t)
			s, _, calls := updateCredentialTestStore(t, u)
			snapshot, err := s.Prepare(context.Background(), u)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(s.dir, snapshot.Items[0].FileKey+".value")
			switch mode {
			case "value":
				err = os.WriteFile(path, []byte("corrupt-value"), 0o600)
			case "key":
				err = os.WriteFile(filepath.Join(s.dir, "key"), make([]byte, 32), 0o600)
			case "missing-value":
				err = os.Remove(path)
			case "missing-key":
				err = os.Remove(filepath.Join(s.dir, "key"))
			case "metadata":
				err = os.WriteFile(filepath.Join(s.dir, "snapshot.json"), []byte(`{"schema_version":1,"schema_version":1}`), 0o600)
			case "alias":
				err = os.Rename(path, path+".original")
				if err == nil {
					err = os.Symlink(path+".original", path)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Prepare(context.Background(), u); err == nil {
				t.Fatal("corrupt committed snapshot was replaced or accepted")
			}
			if *calls != 1 {
				t.Fatal("corruption silently selected a new authority value")
			}
		})
	}
}

func TestManagedUpdateCredentialSnapshotLostAcknowledgementKeepsSameCapture(t *testing.T) {
	for _, visible := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-rename", true: "after-rename"}[visible], func(t *testing.T) {
			u := updateTestInitial(t)
			s, value, calls := updateCredentialTestStore(t, u)
			originalWrite := s.write
			s.write = func(path string, data []byte, mode os.FileMode) error {
				if filepath.Base(path) != "snapshot.json" {
					return originalWrite(path, data, mode)
				}
				if visible {
					if err := originalWrite(path, data, mode); err != nil {
						return err
					}
				}
				return errors.New("injected snapshot acknowledgement failure")
			}
			if receipt, err := s.Prepare(context.Background(), u); !errors.Is(err, ErrManagedUpdateCommitUnknown) || receipt != nil {
				t.Fatal("unconfirmed snapshot returned admission permission")
			}
			*value = "new-authority-after-lost-reply"
			s.write = writeResourceOwnedFileDurable
			snapshot, err := s.Prepare(context.Background(), u)
			if err != nil {
				t.Fatal(err)
			}
			data, err := readManagedUpdatePrivateFile(filepath.Join(s.dir, snapshot.Items[0].FileKey+".value"), 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			if visible && (string(data) != "private-operation-fixture" || *calls != 1) {
				t.Fatal("visible committed snapshot was recaptured")
			}
			if !visible && (string(data) != *value || *calls != 2) {
				t.Fatal("unadmitted partial capture did not rebuild from current authority")
			}
		})
	}
}

func TestManagedUpdateCredentialSnapshotCannotBeRecreatedAfterAdmission(t *testing.T) {
	u := updateTestInitial(t)
	s, _, calls := updateCredentialTestStore(t, u)
	snapshot, err := s.Prepare(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	u.Credentials = snapshot
	owner, err := newManagedUpdateStore(u.Resource)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := owner.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(s.dir, "snapshot.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prepare(context.Background(), u); err == nil {
		t.Fatal("admitted operation recreated lost credential capture")
	}
	if *calls != 1 {
		t.Fatal("admitted operation refetched authority")
	}
	changed := updateTestClone(t, u)
	changed.Credentials.Digest = updateTestDigest("other-credential-binding")
	if _, _, err := owner.Create(context.Background(), changed, false); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatal("request replay replaced pinned credential binding")
	}
}
