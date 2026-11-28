//go:build linux

package setupprivilege

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/apparmorprofiles"
)

func TestInstallAppArmorProfileAtUsesFixedFilePolicy(t *testing.T) {
	uid := uint32(os.Getuid())
	for _, id := range []string{string(apparmorprofiles.BubblewrapUserns), string(apparmorprofiles.WorkspaceSandboxUserns)} {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chmod(dir, 0755); err != nil {
				t.Fatal(err)
			}
			profile, _ := apparmorprofiles.Lookup(id)
			if id == string(apparmorprofiles.BubblewrapUserns) {
				if err := os.WriteFile(filepath.Join(dir, filepath.Base(profile.Path)), []byte(profile.Content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := installAppArmorProfileAt(dir, profile, uid); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(filepath.Join(dir, filepath.Base(profile.Path)))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0644 {
				t.Fatalf("mode=%o", info.Mode().Perm())
			}
			content, err := os.ReadFile(filepath.Join(dir, filepath.Base(profile.Path)))
			if err != nil || string(content) != profile.Content {
				t.Fatalf("content mismatch: err=%v", err)
			}
		})
	}
}

func TestInstallAppArmorProfileAtRejectsUnsafeParentAndTargets(t *testing.T) {
	profile, _ := apparmorprofiles.Lookup(string(apparmorprofiles.WorkspaceSandboxUserns))
	base := t.TempDir()
	unsafe := filepath.Join(base, "unsafe")
	if err := os.Mkdir(unsafe, 0777); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(unsafe, 0777); err != nil {
		t.Fatal(err)
	}
	if err := installAppArmorProfileAt(unsafe, profile, uint32(os.Getuid())); err == nil {
		t.Fatal("writable parent accepted")
	}
	if err := installAppArmorProfileAt(filepath.Join(base, "missing"), profile, uint32(os.Getuid())); err == nil {
		t.Fatal("missing fixed parent accepted")
	}

	real := filepath.Join(base, "real")
	if err := os.Mkdir(real, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "parent-link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := installAppArmorProfileAt(link, profile, uint32(os.Getuid())); err == nil {
		t.Fatal("symlink parent accepted")
	}

	target := filepath.Join(real, filepath.Base(profile.Path))
	if err := os.Symlink("elsewhere", target); err != nil {
		t.Fatal(err)
	}
	if err := installAppArmorProfileAt(real, profile, uint32(os.Getuid())); err == nil {
		t.Fatal("symlink target accepted")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := installAppArmorProfileAt(real, profile, uint32(os.Getuid())); err == nil {
		t.Fatal("non-regular target accepted")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("unsafe mode"), 0666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0666); err != nil {
		t.Fatal(err)
	}
	if err := installAppArmorProfileAt(real, profile, uint32(os.Getuid())); err == nil {
		t.Fatal("group/world-writable target accepted")
	}
}

func TestBubblewrapRefusesDifferentContentWorkspaceReplacesManagedFile(t *testing.T) {
	for _, id := range []string{string(apparmorprofiles.BubblewrapUserns), string(apparmorprofiles.WorkspaceSandboxUserns)} {
		t.Run(id, func(t *testing.T) {
			profile, _ := apparmorprofiles.Lookup(id)
			dir := t.TempDir()
			if err := os.Chmod(dir, 0755); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, filepath.Base(profile.Path))
			if err := os.WriteFile(target, []byte("different"), 0644); err != nil {
				t.Fatal(err)
			}
			err := installAppArmorProfileAt(dir, profile, uint32(os.Getuid()))
			if id == string(apparmorprofiles.BubblewrapUserns) && err == nil {
				t.Fatal("bubblewrap replaced differing policy")
			}
			if id == string(apparmorprofiles.WorkspaceSandboxUserns) && err != nil {
				t.Fatal(err)
			}
			if id == string(apparmorprofiles.WorkspaceSandboxUserns) {
				content, readErr := os.ReadFile(target)
				if readErr != nil || string(content) != profile.Content {
					t.Fatalf("workspace replacement content=%q err=%v", content, readErr)
				}
			}
		})
	}
}

func TestInstallAndLoadAppArmorProfileOrdersWritesBeforeReload(t *testing.T) {
	var calls []string
	err := installAndLoadAppArmorProfile(
		func() error { calls = append(calls, "write"); return nil },
		func() error { calls = append(calls, "reload"); return nil },
	)
	if err != nil || !reflect.DeepEqual(calls, []string{"write", "reload"}) {
		t.Fatalf("calls=%v err=%v", calls, err)
	}
}

func TestInstallAndLoadAppArmorProfileSkipsReloadAfterWriteFailure(t *testing.T) {
	writeErr := errors.New("write failed")
	var calls []string
	err := installAndLoadAppArmorProfile(
		func() error { calls = append(calls, "write"); return writeErr },
		func() error { calls = append(calls, "reload"); return nil },
	)
	if !errors.Is(err, writeErr) || !reflect.DeepEqual(calls, []string{"write"}) {
		t.Fatalf("calls=%v err=%v", calls, err)
	}
}

func TestInstallAndLoadAppArmorProfileReportsReloadFailureAfterWrite(t *testing.T) {
	reloadErr := errors.New("parser failed")
	var calls []string
	err := installAndLoadAppArmorProfile(
		func() error { calls = append(calls, "write"); return nil },
		func() error { calls = append(calls, "reload"); return reloadErr },
	)
	if !errors.Is(err, reloadErr) || !strings.Contains(err.Error(), "profile written but apparmor_parser did not verify it") || !reflect.DeepEqual(calls, []string{"write", "reload"}) {
		t.Fatalf("calls=%v err=%v", calls, err)
	}
}

func TestResolveAppArmorParserRejectsPathLookupCandidates(t *testing.T) {
	dir := t.TempDir()
	pathParser := filepath.Join(dir, "trusted-apparmor-parser")
	if err := os.WriteFile(pathParser, []byte("parser"), 0755); err != nil {
		t.Fatal(err)
	}
	pathParser = filepath.Join(dir, "apparmor_parser")
	if err := os.WriteFile(pathParser, []byte("untrusted PATH parser"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if got := resolveAppArmorParser([]string{"apparmor_parser"}); got != "" {
		t.Fatalf("PATH parser resolved as root: %q", got)
	}
	if got := resolveAppArmorParser([]string{filepath.Join(dir, "trusted-apparmor-parser")}); got != filepath.Join(dir, "trusted-apparmor-parser") {
		t.Fatalf("absolute fixed parser resolution = %q", got)
	}
}
