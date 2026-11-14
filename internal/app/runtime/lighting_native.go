package runtimeapp

import (
	"context"
	"errors"
	"os"
	"runtime"

	platform "github.com/vrooli/platform-go"

	"github.com/vrooli/repo-contract-go/cliinvoke"

	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
)

// NewNativeLightingActions is the shared root composition for explicit user
// operations and the native supervisor. Construction observes no hardware.
func (app *Service) NewNativeLightingActions(root, home string, ensureStarted func(context.Context) error) (*LightingActions, error) {
	controller := resources.NewController(root, home)
	owners := app.Lighting
	if owners == nil {
		scenario, err := lightingruntime.NewNativeScenarioOwner(home, ensureStarted)
		if err != nil {
			return nil, err
		}
		owners = &lightingruntime.Owners{
			Resource: resources.NewOpenRGBHeadless(controller, controller.OpenRGBDevicePolicyVerifier()),
			Scenario: scenario,
		}
	}
	control, err := NewLightingControl(home, *owners, func(context.Context) (resources.Status, error) {
		return controller.Status(openRGBResource, true)
	})
	if err != nil {
		return nil, err
	}
	actions := &LightingActions{Control: control, Resources: controller, AcquireManaged: func(ctx context.Context, version string) (resources.ManagedServiceArtifact, error) {
		return acquireManagedLighting(ctx, controller, home, version)
	}}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		actions.ResourceStorageAuthority = controller.ValidateOpenRGBNativeResourceStorage
		actions.OperatorAuthority = requireNativeLightingOperator
		actions.NativePrepare = func(ctx context.Context) error { return prepareNativeLightingMachine(ctx, controller, home, root) }
		actions.NativePublishArtifact = func(ctx context.Context) error { return prepareNativeLightingUpdateArtifact(ctx, controller) }
	}
	return actions, nil
}

// EnsureNativeLightingScenario uses the supported lifecycle for this sole live
// owner. It is called by reconciliation, never by status or quiescence.
func EnsureNativeLightingScenario(ctx context.Context) error {
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{})
	if err != nil {
		return err
	}
	return cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: binary, Args: cliinvoke.ScenarioLifecycle("start", "lighting-control", false)}).Error()
}

// Ordinary native lifecycle commands may execute operator-owned scenario source.
// Machine publication has a distinct explicit elevated prepare operation.
func requireNativeLightingOperator() error {
	security, err := platform.ProcessSecurity(os.Getpid())
	if err != nil {
		return err
	}
	if security.Elevated {
		return errors.New("native lighting lifecycle must run as the non-elevated operator; elevated runtime lighting prepare is the separate machine publication operation")
	}
	return nil
}
