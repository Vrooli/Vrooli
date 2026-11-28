package runtimeapp

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"runtime"
	"strings"

	platform "github.com/vrooli/platform-go"

	"github.com/vrooli/vrooli/internal/resources"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
)

func (a LightingActions) Devices(ctx context.Context) (*setupv1.LightingDeviceDiscovery, error) {
	if err := a.validateResourceStorage(ctx); err != nil {
		return nil, err
	}
	inventory, err := a.Resources.DiscoverOpenRGBDeviceSetup(ctx)
	if err != nil {
		return nil, err
	}
	result := &setupv1.LightingDeviceDiscovery{State: "available", Message: inventory.Message}
	for _, device := range inventory.Devices {
		result.Candidates = append(result.Candidates, &setupv1.LightingDeviceCandidate{Id: device.ID, Label: device.Label, Kind: device.Kind, Detail: device.Detail, Selected: device.Selected})
	}
	return result, nil
}

func (a LightingActions) PlanDevices(ctx context.Context, ids []string) (*setupv1.LightingDevicePlan, error) {
	if err := a.validateResourceStorage(ctx); err != nil {
		return nil, err
	}
	if len(ids) == 0 || len(ids) > 32 {
		return nil, errors.New("select between one and 32 discovered lighting devices")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if len(id) == 0 || len(id) > 128 || seen[id] {
			return nil, errors.New("select distinct discovered lighting device identities")
		}
		seen[id] = true
	}
	plan, err := a.Resources.PlanOpenRGBDeviceSetup(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := &setupv1.LightingDevicePlan{Id: plan.PlanID, Changes: plan.Changes, RequiresPrivilege: plan.NeedsSudo, Message: plan.Summary}
	// Labels come from current authoritative discovery, never caller text.
	discovery, err := a.Resources.DiscoverOpenRGBDeviceSetup(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range discovery.Devices {
		if seen[device.ID] {
			result.Devices = append(result.Devices, &setupv1.LightingDeviceCandidate{Id: device.ID, Label: device.Label, Kind: device.Kind, Detail: device.Detail, Selected: true})
		}
	}
	return result, nil
}

func (a LightingActions) ApplyDevices(ctx context.Context, id, mode string) (*setupv1.LightingDeviceAccessResult, error) {
	return a.changeDeviceAccess(ctx, id, mode, false)
}

func (a LightingActions) RollbackDevices(ctx context.Context, id, mode string) (*setupv1.LightingDeviceAccessResult, error) {
	return a.changeDeviceAccess(ctx, id, mode, true)
}

func (a LightingActions) changeDeviceAccess(ctx context.Context, id, mode string, rollback bool) (*setupv1.LightingDeviceAccessResult, error) {
	if err := a.validateResourceStorage(ctx); err != nil {
		return nil, err
	}
	digest, digestErr := hex.DecodeString(id)
	if digestErr != nil || len(digest) != 32 || id != strings.ToLower(id) {
		return nil, errors.New("select an exact permission plan identity")
	}
	if mode != "error" && mode != "ask" {
		return nil, errors.New("sudo mode must be error or ask")
	}
	var result resources.OpenRGBDeviceSetupResult
	change := func() error {
		var err error
		if rollback {
			result, err = a.Resources.RollbackOpenRGBDeviceSetup(ctx, id, mode)
		} else {
			result, err = a.Resources.ApplyOpenRGBDeviceSetup(ctx, id, mode)
		}
		return err
	}
	var err error
	// Elevation reexecs the fixed command. Only the actual privileged executor
	// holds the intent fence, so its waiting parent cannot deadlock that child.
	elevated := os.Geteuid() == 0
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		security, securityErr := platform.ProcessSecurity(os.Getpid())
		if securityErr != nil {
			return nil, securityErr
		}
		elevated = security.Elevated
	}
	if elevated {
		err = a.Control.controller.PrepareDisabled(ctx, change)
	} else {
		err = change()
	}
	if err != nil && !result.Pending && !result.Applied && !result.RolledBack && !result.NeedsSudo {
		return nil, err
	}
	state := "unchanged"
	if result.Pending {
		state = "pending"
	} else if result.NeedsSudo {
		state = "needs_sudo"
	} else if result.Applied || result.RolledBack {
		state = "applied"
		if rollback {
			state = "rolled_back"
		}
	}
	if err != nil {
		state = "failed"
		if result.Pending || result.Applied || result.RolledBack {
			state = "pending"
		}
		if result.Message == "" {
			result.Message = err.Error()
		}
	}
	return &setupv1.LightingDeviceAccessResult{State: state, Message: result.Message, PlanId: result.PlanID, NeedsSudo: result.NeedsSudo, DevicePolicyId: result.PolicyID}, err
}
