//go:build !windows

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinEnrolledResourceRootAbsentProtectedAndChangedUnit(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, configured, err := e.enrolledResourceStorageOptions(context.Background()); err != nil || configured {
		t.Fatalf("initial absence not recognized: %t %v", configured, err)
	}
	o.SystemBroker.GlobalResourceStorageRoot = t.TempDir()
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	actual, configured, err := e.enrolledResourceStorageOptions(context.Background())
	if err != nil || !configured || actual != *o.SystemBroker {
		t.Fatalf("installed root not retained: %+v %t %v", actual, configured, err)
	}
	original := e.check
	reads := 0
	e.check = func(path string, directory bool) error {
		err := original(path, directory)
		if path == f.roots.Unit {
			reads++
			if reads == 2 {
				data, _ := os.ReadFile(path)
				os.WriteFile(path, append(data, byte(' ')), 0o600)
			}
		}
		return err
	}
	if _, configured, err = e.enrolledResourceStorageOptions(context.Background()); err == nil || configured {
		t.Fatal("changed protected boot definition accepted")
	}
	e.check = original
	if err := os.WriteFile(f.roots.Unit, []byte("malformed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, configured, err = e.enrolledResourceStorageOptions(context.Background()); err == nil || configured {
		t.Fatal("malformed definition treated as absent")
	}
	e.check = func(string, bool) error { return os.ErrPermission }
	if _, configured, err = e.enrolledResourceStorageOptions(context.Background()); !errors.Is(err, os.ErrPermission) || configured {
		t.Fatalf("query denial treated as absent %t %v", configured, err)
	}
}

func TestDarwinRetainedNamespaceSurvivesBootUninstall(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	o.SystemBroker.GlobalResourceStorageRoot = t.TempDir()
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if _, err := e.uninstall(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if f.active {
		t.Fatal("fixture boot still active")
	}
	if _, err := os.Lstat(f.roots.Unit); !os.IsNotExist(err) {
		t.Fatalf("unit not removed: %v", err)
	}
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", t.TempDir())
	actual, configured, err := e.enrolledResourceStorageOptions(context.Background())
	if err != nil || !configured || actual != *o.SystemBroker {
		t.Fatalf("retained owner lost after uninstall: %+v %t %v", actual, configured, err)
	}
	st, err := os.Stat(f.roots.State)
	if err != nil || st.Mode().Perm() != 0o755 {
		t.Fatalf("readonly ownership journal not traversable: %v %v", st, err)
	}
	lock, err := os.Stat(filepath.Join(f.roots.State, "native-broker.lock"))
	if err != nil || lock.Mode().Perm()&0o077 != 0 {
		t.Fatalf("native lock exposed: %v %v", lock, err)
	}
	plan, err := e.journal()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(plan.ImagePath); err != nil {
		t.Fatal(err)
	}
	if actual, configured, err = e.enrolledResourceStorageOptions(context.Background()); err != nil || !configured || actual != *o.SystemBroker {
		t.Fatalf("code cleanup erased namespace ownership: %+v %t %v", actual, configured, err)
	}
	if err = os.Remove(plan.DescriptorPath); err != nil {
		t.Fatal(err)
	}
	if _, configured, err = e.enrolledResourceStorageOptions(context.Background()); err == nil || configured {
		t.Fatal("retained pointer with missing descriptor became unenrolled")
	}
}

func TestDarwinRetainedJournalDuplicateIsUnknown(t *testing.T) {
	e, f, o := newDarwinBrokerFixture(t)
	if _, err := e.install(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	if _, err := e.uninstall(context.Background(), o); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(f.roots.State, "installed-descriptor.json")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	duplicate := append([]byte(`{"descriptor":"/foreign","descriptor":`), original[len(`{"descriptor":`):]...)
	if err = os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, duplicate, 0o444); err != nil {
		t.Fatal(err)
	}
	if err = os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	if _, configured, err := e.enrolledResourceStorageOptions(context.Background()); err == nil || configured {
		t.Fatal("duplicate protected journal pointer accepted")
	}
}
