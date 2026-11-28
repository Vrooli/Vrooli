package runtimeapp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
)

func TestNativeLightingChangedGlobalRootCannotDisableOrReconnectDifferentNamespace(t *testing.T) {
	storagetest.IsolateStorage(t)
	ctx := context.Background()
	home := t.TempDir()
	rootA, rootB := t.TempDir(), t.TempDir()
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", rootA)
	resource, scenario := &lightingResourceOwnerFake{}, &lightingScenarioOwnerFake{}
	control, err := NewLightingControl(home, lightingruntime.Owners{Resource: resource, Scenario: scenario}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection := lightingruntime.Selection{Version: "1.1", SHA256: strings.Repeat("a", 64), DevicePolicyID: "policy"}
	if err = control.Enable(ctx, selection); err != nil {
		t.Fatal(err)
	}
	if err = control.controller.Reconcile(ctx); err != nil {
		t.Fatal(err)
	}
	oldEnsured, oldScenarioEnsured := resource.ensured, scenario.ensured
	source, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	denied := errors.New("restore enrolled VROOLI_RESOURCE_STORAGE_ROOT=" + rootA)
	actions := LightingActions{Control: control, Resources: resources.NewController(source, home), ResourceStorageAuthority: func(context.Context) error {
		if os.Getenv("VROOLI_RESOURCE_STORAGE_ROOT") != rootA {
			return denied
		}
		return nil
	}}
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", rootB)
	for name, run := range map[string]func() error{
		"disable":     func() error { _, e := actions.Disable(ctx); return e },
		"reconnect":   func() error { _, e := actions.Enable(ctx, nil); return e },
		"setup":       func() error { _, e := actions.Setup(ctx, nil); return e },
		"prepare":     func() error { return actions.PrepareNative(ctx) },
		"permissions": func() error { _, e := actions.ApplyDevices(ctx, "invalid", "error"); return e },
		"status":      func() error { _, e := actions.Status(ctx); return e },
	} {
		t.Run(name, func(t *testing.T) {
			if e := run(); !errors.Is(e, denied) {
				t.Fatalf("changed root entered effects: %v", e)
			}
		})
	}
	intent, err := control.controller.Current()
	if err != nil || intent.Phase != lightingruntime.Enabled {
		t.Fatalf("false disabled acknowledgement: %+v %v", intent, err)
	}
	if resource.disabled != 0 || scenario.quiesced != 0 || resource.ensured != oldEnsured || scenario.ensured != oldScenarioEnsured {
		t.Fatalf("foreign namespace touched original owner: resource=%+v scenario=%+v", resource, scenario)
	}
	t.Setenv("VROOLI_RESOURCE_STORAGE_ROOT", rootA)
	if _, err = actions.Disable(ctx); err != nil {
		t.Fatal(err)
	}
	intent, err = control.controller.Current()
	if err != nil || intent.Phase != lightingruntime.Disabled || resource.disabled != 1 || scenario.quiesced != 1 {
		t.Fatalf("restore did not clean exact owner: %+v %v resource=%+v scenario=%+v", intent, err, resource, scenario)
	}
}
