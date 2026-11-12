package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func configureFrozenOpenRGBObserver(t *testing.T, x *openRGBExecFixture, accessErr, contextErr error, mutateConfig bool) {
	t.Helper()
	catalog, err := readManagedUpdateConfigFile(filepath.Join(projectRoot(t), "resources", openRGBResourceName), x.update.Target.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	policy := hostreqkit.OpenRGBLinuxPolicy{
		MachineID: strings.Repeat("a", 32), ArtifactSHA256: x.update.Target.Artifact.SHA256,
		DetectorSHA256: updateTestDigest(string(catalog)), RuleSHA256: updateTestDigest("static rule"), GroupID: 1001,
		Devices: []hostreqkit.OpenRGBLinuxDevice{{Detector: "ASUS Aura Motherboard", Kind: "hidraw", Topology: "pci0000:00/usb1/1-5/1-5:1.0", HardwareID: "0b05:19af:00"}},
	}
	x.owner.openRGBObserve = func(ctx context.Context, contract ManagedUpdateContract, invocation managedUpdateInvocation, fence managedUpdateExecFence, ref platform.ServiceRef, incarnation string) (openRGBExecWitness, error) {
		configBytes := contract.ConfigBytes
		if mutateConfig {
			configBytes = []byte(strings.Replace(invocation.ConfigContent, `"ENE SMBus DRAM": false`, `"ENE SMBus DRAM": true`, 1))
		}
		return observeOpenRGBExecWitness(ctx, policy, contract, invocation, catalog, configBytes, fence, ref, incarnation, 1000,
			func(platform.ServiceRef, string) (string, error) {
				if contextErr != nil {
					return "", contextErr
				}
				return updateTestDigest("same final native context"), nil
			},
			func(_ context.Context, _ hostreqkit.OpenRGBLinuxPolicy, _ hostreqkit.OpenRGBLinuxCheck, _ func(int) (string, error)) (hostreqkit.OpenRGBLinuxEvidence, error) {
				if accessErr != nil {
					return hostreqkit.OpenRGBLinuxEvidence{}, accessErr
				}
				return hostreqkit.OpenRGBLinuxEvidence{DeviceGenerationSHA256: updateTestDigest("selected kernel device generation"), StaticGrantConfigured: true, CurrentScopeVerified: true, SelectedNodes: []string{"/dev/hidraw9"}}, nil
			})
	}
}

func TestOpenRGBExecRefusalDoesNotPublishConfigOrLaunch(t *testing.T) {
	for _, tc := range []struct {
		name          string
		accessErr     error
		contextErr    error
		mutateConfig  bool
		missing       bool
		changedPolicy bool
	}{
		{name: "missing frozen policy", missing: true},
		{name: "changed frozen policy", changedPolicy: true},
		{name: "same-context observation fails", contextErr: errors.New("native context changed")},
		{name: "selected device access fails", accessErr: errors.New("unselected HID node is accessible")},
		{name: "intended config changed", mutateConfig: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := newOpenRGBExecFixture(t)
			configureFrozenOpenRGBObserver(t, x, tc.accessErr, tc.contextErr, tc.mutateConfig)
			if _, err := x.store.Prepare(context.Background(), x.fence); err != nil {
				t.Fatal(err)
			}
			if tc.missing {
				legacy := x.fence
				legacy.OpenRGBPolicy = nil
				if err := x.store.save(legacy); err != nil {
					t.Fatal(err)
				}
			} else if tc.changedPolicy {
				changed := x.fence
				copy := *changed.OpenRGBPolicy
				copy.ID = updateTestDigest("different selected machine policy")
				changed.OpenRGBPolicy = &copy
				if err := x.store.save(changed); err != nil {
					t.Fatal(err)
				}
			}
			err := x.run()
			if err == nil || x.replaced != 0 || x.observed > 1 {
				t.Fatalf("refused witness advanced launch: err=%v observations=%d execs=%d", err, x.observed, x.replaced)
			}
			roleRoot, err := x.fence.Intent.Runtime.roleRoot(x.update.ID, x.fence.Intent.InstanceID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(roleRoot, "config")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("refused witness published role config directory: %v", err)
			}
		})
	}
}

func TestOpenRGBExecClaimsFrozenPolicyBeforePublishingConfig(t *testing.T) {
	x := newOpenRGBExecFixture(t)
	configureFrozenOpenRGBObserver(t, x, nil, nil, false)
	if _, err := x.store.Prepare(context.Background(), x.fence); err != nil {
		t.Fatal(err)
	}
	claimed := false
	x.owner.replace = func(_ *os.File, argv, _ []string) error {
		f, err := x.store.read()
		if err != nil || f.State != managedUpdateExecFenceClaimed || f.OpenRGBWitness == nil || f.OpenRGBPolicy == nil {
			t.Fatalf("resource launch preceded the durable witness claim: %+v, %v", f, err)
		}
		claimed = true
		roleRoot, err := x.fence.Intent.Runtime.roleRoot(x.update.ID, x.fence.Intent.InstanceID)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(roleRoot, "config", "OpenRGB", "OpenRGB.json"))
		if err != nil || string(data) != string(x.update.Target.ConfigBytes) {
			t.Fatalf("claimed launch lacks exact frozen config: %v", err)
		}
		if len(argv) == 0 || argv[0] != x.update.Target.Artifact.Path {
			t.Fatalf("resource target changed after claim: %v", argv)
		}
		return nil
	}
	if err := x.run(); err != nil || !claimed {
		t.Fatalf("verified witness did not publish config and launch: claimed=%t err=%v", claimed, err)
	}
}
