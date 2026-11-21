package runtimeapp

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"

	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

func TestNativeLightingPrepareDoesNotProvisionWhileEnabled(t *testing.T) {
	control, err := NewLightingControl(t.TempDir(), lightingruntime.Owners{Resource: &lightingResourceOwnerFake{}, Scenario: &lightingScenarioOwnerFake{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = control.Enable(context.Background(), lightingruntime.Selection{Version: "1.1", SHA256: strings.Repeat("a", 64), DevicePolicyID: "policy"}); err != nil {
		t.Fatal(err)
	}
	prepared := 0
	actions := LightingActions{Control: control, Resources: &resources.Controller{}, NativePrepare: func(context.Context) error { prepared++; return nil }}
	if err = actions.PrepareNative(context.Background()); err == nil || prepared != 0 {
		t.Fatalf("enabled setup crossed preparation boundary count=%d err=%v", prepared, err)
	}
}

func TestNativeLightingElevationPreservesPublicPrepareOperation(t *testing.T) {
	err := nativeLightingElevationCommand(hostreqkit.ErrElevationRequired, cliinvoke.RuntimeLightingPrepare())
	if !errors.Is(err, hostreqkit.ErrElevationRequired) {
		t.Fatal("lost typed elevation reason")
	}
	for _, part := range []string{"runtime", "lighting", "prepare", "--json"} {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("missing public prepare %q: %v", part, err)
		}
	}
	denied := errors.New("artifact corruption")
	if nativeLightingElevationCommand(denied, cliinvoke.RuntimeLightingPrepare()) != denied {
		t.Fatal("corruption disguised as elevation")
	}
}

func TestNativeLightingPreparationInstallsSealedHostBeforeDeviceOrLauncher(t *testing.T) {
	var events []string
	e := nativeLightingPreparationEffects{
		inspect: func(context.Context) error { events = append(events, "inspect-pinned-artifact"); return nil },
		install: func(context.Context) (runtimesupervisor.ServiceInstallResult, error) {
			events = append(events, "sealed-install")
			return runtimesupervisor.ServiceInstallResult{Executable: "/sealed/final-host", ExecutableIsCanonical: true}, nil
		},
		device:   func(context.Context) error { events = append(events, "device-context"); return nil },
		artifact: func(context.Context) error { events = append(events, "seal-pinned-artifact"); return nil },
		launcher: func(_ context.Context, path string) error {
			if path != "/sealed/final-host" {
				t.Fatal("launcher lost installed identity")
			}
			events = append(events, "actual-service-launcher")
			return nil
		},
	}
	if err := prepareNativeLightingMachineWith(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	expected := "inspect-pinned-artifact,sealed-install,device-context,seal-pinned-artifact,actual-service-launcher"
	if strings.Join(events, ",") != expected {
		t.Fatalf("unsafe preparation order %v", events)
	}
	install := e.install
	events = nil
	denied := errors.New("sealed host install failed")
	e.install = func(context.Context) (runtimesupervisor.ServiceInstallResult, error) {
		events = append(events, "install-failed")
		return runtimesupervisor.ServiceInstallResult{}, denied
	}
	if err := prepareNativeLightingMachineWith(context.Background(), e); !errors.Is(err, denied) || strings.Join(events, ",") != "inspect-pinned-artifact,install-failed" {
		t.Fatalf("failure crossed native publication boundary %v %v", events, err)
	}
	events = nil
	e.install = func(context.Context) (runtimesupervisor.ServiceInstallResult, error) {
		events = append(events, "unsealed-install")
		return runtimesupervisor.ServiceInstallResult{Executable: "/mutable/caller", ExecutableIsCanonical: false}, nil
	}
	if err := prepareNativeLightingMachineWith(context.Background(), e); err == nil || strings.Join(events, ",") != "inspect-pinned-artifact,unsealed-install" {
		t.Fatalf("unsealed host reached device/artifact effects %v %v", events, err)
	}
	events = nil
	e.install = install
	if err := prepareNativeLightingMachineWith(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if strings.Join(events, ",") != expected {
		t.Fatalf("retry skipped sealed authority %v", events)
	}
}

func TestNativeLightingLifecycleRejectsPrivilegeBeforeAnyOwnerEffect(t *testing.T) {
	denied := errors.New("elevated operator refused")
	a := LightingActions{OperatorAuthority: func() error { return denied }}
	// All owners are nil: an effect before identity validation would panic.
	for name, run := range map[string]func() error{
		"setup":   func() error { _, e := a.Setup(context.Background(), nil); return e },
		"enable":  func() error { _, e := a.Enable(context.Background(), nil); return e },
		"disable": func() error { _, e := a.Disable(context.Background()); return e },
	} {
		t.Run(name, func(t *testing.T) {
			if e := run(); !errors.Is(e, denied) {
				t.Fatalf("identity boundary lost: %v", e)
			}
		})
	}
}

func TestNativeLightingOrdinarySetupAcquiresWithoutPrivilegedPreparation(t *testing.T) {
	storagetest.IsolateStorage(t)
	home := t.TempDir()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	c := resources.NewController(root, home)
	control, err := NewLightingControl(home, lightingruntime.Owners{Resource: &lightingResourceOwnerFake{}, Scenario: &lightingScenarioOwnerFake{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	denied := errors.New("operator SDA acquisition fixture")
	acquired := 0
	a := LightingActions{Control: control, Resources: c, NativePrepare: func(context.Context) error { t.Fatal("ordinary Setup invoked privileged publication"); return nil }, AcquireManaged: func(context.Context, string) (resources.ManagedServiceArtifact, error) {
		acquired++
		return resources.ManagedServiceArtifact{}, denied
	}}
	pinned, err := a.pinnedSelection()
	if err != nil {
		t.Skipf("no governed artifact for fixture target: %v", err)
	}
	_, err = a.Setup(context.Background(), pinned)
	if !errors.Is(err, denied) || acquired != 1 {
		t.Fatalf("ordinary acquisition path lost: %v calls=%d", err, acquired)
	}
}

func TestNativeLightingPrepareReadOnlyAcquisitionFailurePrecedesBootPublication(t *testing.T) {
	denied := errors.New("selected operator has no acquired pinned artifact")
	err := prepareNativeLightingMachineWith(context.Background(), nativeLightingPreparationEffects{inspect: func(context.Context) error { return denied }, install: func(context.Context) (runtimesupervisor.ServiceInstallResult, error) {
		t.Fatal("missing operator cache published boot enrollment")
		return runtimesupervisor.ServiceInstallResult{}, nil
	}})
	if !errors.Is(err, denied) {
		t.Fatal(err)
	}
}

func TestNativeLightingElevationFreezesOnlyGlobalRootForCustomAndDefault(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "lighting ' root")
	for _, goos := range []string{"darwin", "windows"} {
		for _, root := range []string{custom, ""} {
			t.Run(goos+root, func(t *testing.T) {
				err := nativeLightingElevationCommandFor(goos, root, hostreqkit.ErrElevationRequired, cliinvoke.RuntimeLightingPrepare())
				if !errors.Is(err, hostreqkit.ErrElevationRequired) {
					t.Fatal(err)
				}
				text := err.Error()
				if !strings.Contains(text, "VROOLI_RESOURCE_STORAGE_ROOT") || !strings.Contains(text, "prepare") || strings.Contains(text, "preserve-env") {
					t.Fatalf("namespace not explicitly transferred: %s", text)
				}
				if goos == "darwin" {
					quoted := "'" + strings.ReplaceAll("VROOLI_RESOURCE_STORAGE_ROOT="+root, "'", "'\"'\"'") + "'"
					if !strings.Contains(text, "sudo "+quoted) {
						t.Fatalf("sudo lost exact root: %s", text)
					}
				} else {
					quoted := "'" + strings.ReplaceAll(root, "'", "''") + "'"
					if !strings.Contains(text, quoted+", 'Process'") || !strings.Contains(text, "finally") || !strings.Contains(text, "same operator") {
						t.Fatalf("PowerShell root transfer/restoration lost: %s", text)
					}
				}
			})
		}
	}
}
