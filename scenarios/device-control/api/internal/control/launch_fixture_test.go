package control

import (
	"context"
	"testing"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
)

// lifecycleFixture answers package-state and foreground-state like a device
// with one installed app.
type lifecycleFixture struct {
	*fakes.Strategy
	installed  string
	foreground string
	// installVersion is the version code package-state reports after an install;
	// installRefused makes the install a silent no-op (accepted, nothing installed).
	installVersion int64
	installRefused bool
	installs       int
}

func (f *lifecycleFixture) Actuate(ctx context.Context, action strategy.Actuation) error {
	if action.Action == "install" {
		f.installs++
		if !f.installRefused {
			f.installed = action.Package
		}
		return nil
	}
	return f.Strategy.Actuate(ctx, action)
}

func (f *lifecycleFixture) AppLifecycle(_ context.Context, request strategy.AppLifecycleRequest) (strategy.AppLifecycleResult, error) {
	result := strategy.AppLifecycleResult{Operation: request.Operation, Package: request.Package, Status: strategy.StatusAvailable}
	switch request.Operation {
	case "package-state":
		result.Installed = request.Package == f.installed
		result.VersionCode = f.installVersion
	case "launch":
		f.foreground = request.Package
	case "foreground-state":
		result.ForegroundPackage = f.foreground
	}
	return result, nil
}

func launchService(t *testing.T, installed string) (*Service, *lifecycleFixture) {
	t.Helper()
	fixture := &lifecycleFixture{Strategy: fakes.New("launch-fixture", strategy.StatusAvailable, strategy.CapAppLifecycle), installed: installed}
	svc := New(strategyregistry.New(fixture))
	svc.devices.Upsert(devicedomain.Record{ID: "target", Name: "Living room", Kind: "physical", StrategyID: fixture.ID(), Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	return svc, fixture
}
