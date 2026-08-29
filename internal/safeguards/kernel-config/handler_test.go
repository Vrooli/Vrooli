package kernelconfig

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	kernelInotifyWatches   = "/proc/sys/fs/inotify/max_user_watches"
	kernelInotifyInstances = "/proc/sys/fs/inotify/max_user_instances"
)

var stubLookups = kernelConfigStubLookups

func kernelConfigStubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
	}
}

var newTestHandler = kernelConfigTestHandler

func kernelConfigTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "kernel_config",
		Handler: "kernel_config",
	})
}

var linuxReq = kernelConfigLinuxReq

func kernelConfigLinuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "kernel_config",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

var linuxHost = kernelConfigLinuxHost

func kernelConfigLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:             "linux",
		PackageManager: "apt-get",
		SupportsSysctl: true,
	}
}

func TestInspectAllParametersMet(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case kernelInotifyWatches:
			return []byte("1048576\n"), nil
		case kernelInotifyInstances:
			return []byte("2048\n"), nil
		case configPath:
			return []byte(buildConfigContent()), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectParametersBelowMinimum(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case kernelInotifyWatches:
			return []byte("8192\n"), nil
		case kernelInotifyInstances:
			return []byte("128\n"), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false")
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "below minimum") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected note about minimum values, got: %v", status.Notes)
	}
}

func TestInspectConfigFileMismatch(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case kernelInotifyWatches:
			return []byte("1048576\n"), nil
		case kernelInotifyInstances:
			return []byte("2048\n"), nil
		case configPath:
			return []byte("# wrong content\n"), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false when config file doesn't match")
	}
}

func TestApplyUsesSysctlAndSurfacesFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail bool
		want hostreqkit.ExecutionState
	}{
		{name: "success", want: hostreqkit.ExecutionApplied},
		{name: "sysctl failure", fail: true, want: hostreqkit.ExecutionFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			restore := stubLookups(t)
			defer restore()
			hostreqkit.LookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }
			var calls []string
			hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
				calls = append(calls, name+" "+strings.Join(args, " "))
				if tc.fail && strings.Contains(strings.Join(args, " "), "sysctl") {
					return fmt.Errorf("sysctl: permission denied")
				}
				return nil
			}
			status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}, hostreqkit.EnsureOptions{SudoMode: "ask"})
			if err != nil {
				t.Fatal(err)
			}
			if status.ExecutionState != tc.want {
				t.Fatalf("ExecutionState = %q, want %q", status.ExecutionState, tc.want)
			}
			if !tc.fail {
				if !status.Applied {
					t.Fatalf("expected Applied = true, notes: %v", status.Notes)
				}
				foundSysctl := false
				for _, call := range calls {
					foundSysctl = foundSysctl || strings.Contains(call, "sysctl") && strings.Contains(call, "--system")
				}
				if !foundSysctl {
					t.Fatalf("expected sysctl --system call, got: %v", calls)
				}
			}
		})
	}
}
