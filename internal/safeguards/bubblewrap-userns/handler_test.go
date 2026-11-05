package bubblewrapuserns

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func fakeHost(t *testing.T) {
	t.Helper()
	lookup, read, output := hostreqkit.LookPathFn, hostreqkit.ReadFileFn, hostreqkit.CombinedOutputFn
	run, temp, root := hostreqkit.RunCommandFn, hostreqkit.WriteTempFileFn, hostreqkit.RunningAsRootFn
	t.Cleanup(func() {
		hostreqkit.LookPathFn, hostreqkit.ReadFileFn, hostreqkit.CombinedOutputFn = lookup, read, output
		hostreqkit.RunCommandFn, hostreqkit.WriteTempFileFn, hostreqkit.RunningAsRootFn = run, temp, root
	})
	hostreqkit.RunningAsRootFn = func() bool { return false }
	hostreqkit.LookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == "/proc/sys/kernel/apparmor_restrict_unprivileged_userns" {
			return []byte("1\n"), nil
		}
		return nil, os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name != "/usr/bin/bwrap" || !reflect.DeepEqual(args, []string{"--unshare-user", "--unshare-pid", "--unshare-net", "--die-with-parent", "--ro-bind", "/", "/", "--proc", "/proc", "--dev", "/dev", "--", "/bin/true"}) {
			t.Fatalf("probe changed: %s %v", name, args)
		}
		return []byte("namespace denied"), errors.New("exit 1")
	}
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		t.Fatalf("unexpected host mutation: %s %v", name, args)
		return nil
	}
}

func inspect() hostreqkit.ItemStatus {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "bubblewrap_userns"}).Inspect(
		hostreqkit.Host{OS: "linux"}, hostreqspec.ResolvedRequirement{Name: "bubblewrap_userns", Kind: hostreqspec.KindSafeguard})
}

func apply(status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) hostreqkit.ItemStatus {
	result, _ := NewHandler(hostreqkit.SafeguardManifest{Name: "bubblewrap_userns"}).Apply(hostreqkit.Host{OS: "linux"}, status, opts)
	return result
}

func TestNamespaceProbeAndIdempotency(t *testing.T) {
	fakeHost(t)
	status := inspect()
	if status.Applied || status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("failed namespaces must remain pending: %+v", status)
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return nil, nil }
	status = apply(inspect(), hostreqkit.EnsureOptions{})
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("working distribution policy must need no mutation: %+v", status)
	}
}

func TestElevatedInspectionStillProbesAsInvokingUser(t *testing.T) {
	fakeHost(t)
	hostreqkit.RunningAsRootFn = func() bool { return true }
	t.Setenv("SUDO_USER", "operator")
	t.Setenv("SUDO_UID", "1001")
	t.Setenv("SUDO_GID", "1001")
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name != "sudo" || len(args) < 6 || !reflect.DeepEqual(args[:5], []string{"-u", "operator", "-H", "--", "/usr/bin/bwrap"}) {
			t.Fatalf("probe did not drop root: %s %v", name, args)
		}
		return nil, nil
	}
	status := inspect()
	if !status.Applied {
		t.Fatalf("operator probe not performed: %+v", status)
	}
}

func TestApplyInstallsLoadsAndValidates(t *testing.T) {
	fakeHost(t)
	status := inspect()
	status.Notes = append(status.Notes, "operator context must be preserved")
	temp := t.TempDir() + "/profile"
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		if content != profileContent || !strings.Contains(content, "/{,usr/}bin/bwrap") {
			t.Fatal("profile is not scoped to system bubblewrap")
		}
		return temp, os.WriteFile(temp, []byte(content), 0600)
	}
	var commands []string
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if name == "sudo" && len(args) > 0 && args[0] == "apparmor_parser" {
			hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) { return nil, nil }
		}
		return nil
	}
	status = apply(status, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if !status.Applied || status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("namespace validation failed: %+v", status)
	}
	if !strings.HasPrefix(status.Notes[0], "resolved pre-repair failure: bubblewrap namespace probe:") {
		t.Fatalf("successful repair must distinguish the old failure from current validation: %v", status.Notes)
	}
	if status.Notes[1] != "operator context must be preserved" {
		t.Fatalf("unrelated inspection notes changed: %v", status.Notes)
	}
	want := []string{"sudo install -m 644 " + temp + " " + profilePath, "sudo apparmor_parser -r " + profilePath}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("unexpected host effects: %v", commands)
	}
}

func TestApplyFailureBoundaries(t *testing.T) {
	for _, name := range []string{"dry-run", "custom-binary", "missing-binary", "root", "no-restriction", "foreign-profile", "write-failure", "load-failure", "probe-failure"} {
		t.Run(name, func(t *testing.T) {
			fakeHost(t)
			opts := hostreqkit.EnsureOptions{SudoMode: "ask"}
			want := hostreqkit.ExecutionFailed
			switch name {
			case "dry-run":
				opts.DryRun, want = true, hostreqkit.ExecutionWouldApply
			case "custom-binary":
				hostreqkit.LookPathFn = func(string) (string, error) { return "/tmp/bwrap", nil }
			case "missing-binary":
				hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
			case "root":
				hostreqkit.RunningAsRootFn = func() bool { return true }
				t.Setenv("SUDO_USER", "")
				want = hostreqkit.ExecutionManualActionRequired
			case "no-restriction":
				hostreqkit.ReadFileFn = func(string) ([]byte, error) { return []byte("0"), nil }
			case "foreign-profile":
				read := hostreqkit.ReadFileFn
				hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
					if path == profilePath {
						return []byte("operator policy"), nil
					}
					return read(path)
				}
			case "write-failure", "load-failure", "probe-failure":
				hostreqkit.WriteTempFileFn = func(string) (string, error) {
					if name == "write-failure" {
						return "", errors.New("disk full")
					}
					return t.TempDir() + "/profile", nil
				}
				hostreqkit.RunCommandFn = func(command string, args []string, _ hostreqkit.EnsureOptions) error {
					if command == "sudo" && len(args) > 0 && args[0] == "apparmor_parser" && name == "load-failure" {
						return errors.New("invalid policy")
					}
					return nil
				}
			}
			status := apply(inspect(), opts)
			if status.Applied || status.ExecutionState != want {
				t.Fatalf("want %s without success: %+v", want, status)
			}
			if strings.Contains(strings.Join(status.Notes, "\n"), "resolved pre-repair failure:") {
				t.Fatalf("unvalidated repair must not report a resolved failure: %+v", status)
			}
		})
	}
}

func TestNonLinuxIsNotApplicable(t *testing.T) {
	fakeHost(t)
	for _, osName := range []string{"darwin", "windows"} {
		status := NewHandler(hostreqkit.SafeguardManifest{Name: "bubblewrap_userns"}).Inspect(hostreqkit.Host{OS: osName}, hostreqspec.ResolvedRequirement{})
		if status.SupportClass != hostreqkit.SupportNotApplicable || status.ExecutionState != hostreqkit.ExecutionNotApplicable {
			t.Fatalf("%s: %+v", osName, status)
		}
	}
}
