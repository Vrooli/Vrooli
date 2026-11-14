package runtimeapp

import (
	"context"
	"errors"
	"fmt"

	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/resources"
)

const openRGBResource = "openrgb"

type LightingState string

const (
	LightingStateUnknown      LightingState = "unknown"
	LightingStateDisabled     LightingState = "disabled"
	LightingStateNotInstalled LightingState = "not_installed"
	LightingStateStopped      LightingState = "stopped"
	LightingStateReady        LightingState = "ready"
	LightingStateDegraded     LightingState = "degraded"
)

// LightingResourceStatusReader supplies the resource owner's explicit status
// observation. It is intentionally separate from lifecycle ownership: a status
// read cannot install, enable, or start OpenRGB.
type LightingResourceStatusReader func(context.Context) (resources.Status, error)

// LightingStatus is the typed control-plane view consumed by an authenticated
// caller. Installed, running and ready are separate facts; ready is populated
// only from the resource owner's serving observation.
type LightingStatus struct {
	Intent           lightingruntime.Intent `json:"intent"`
	State            LightingState          `json:"state"`
	Installed        bool                   `json:"installed"`
	Running          bool                   `json:"running"`
	Ready            bool                   `json:"ready"`
	Degraded         bool                   `json:"degraded"`
	Health           string                 `json:"health,omitempty"`
	Message          string                 `json:"message,omitempty"`
	ResourceObserved bool                   `json:"resource_observed"`
}

// LightingControl exposes only fixed lighting operations. Transport and
// authentication belong to the caller; there is no caller-selected command or
// argument vector. Setup verifies an already staged selection and does not
// acquire bytes, edit host permissions, or start a process.
type LightingControl struct {
	controller *lightingruntime.Controller
	owners     lightingruntime.Owners
	status     LightingResourceStatusReader
}

// NewLightingControl composes the fixed application operations from the same
// owner set used by native reconciliation. Callers must still supply the
// resource's truthful status reader; no owner is inferred from caller input.
func (app *Service) NewLightingControl(home string, status LightingResourceStatusReader) (*LightingControl, error) {
	if app == nil || app.Lighting == nil {
		return nil, errors.New("lighting control owners are unavailable")
	}
	return NewLightingControl(home, *app.Lighting, status)
}

func NewLightingControl(home string, owners lightingruntime.Owners, status LightingResourceStatusReader) (*LightingControl, error) {
	controller, err := lightingruntime.NewController(home, owners)
	if err != nil {
		return nil, err
	}
	return &LightingControl{controller: controller, owners: owners, status: status}, nil
}

// Status reads durable intent and one explicit OpenRGB resource observation.
func (c *LightingControl) Status(ctx context.Context) (LightingStatus, error) {
	if c == nil || c.controller == nil {
		return LightingStatus{}, errors.New("lighting control is unavailable")
	}
	intent, err := c.controller.Current()
	if err != nil {
		return LightingStatus{}, fmt.Errorf("read lighting intent: %w", err)
	}
	status := LightingStatus{Intent: intent, State: LightingStateUnknown}
	if intent.Phase == lightingruntime.Disabled {
		status.State = LightingStateDisabled
		status.Message = "Lighting is disabled. Use explicit setup to select an installation."
		return status, nil
	}
	if c.status == nil {
		status.State = LightingStateDegraded
		status.Degraded = true
		status.Message = "OpenRGB resource status owner is unavailable; compose the typed host resource owner before setup or enable"
		return status, nil
	}
	resource, err := c.status(ctx)
	if err != nil {
		return LightingStatus{}, fmt.Errorf("read OpenRGB resource status: %w", err)
	}
	status.Installed = resource.Installed
	status.ResourceObserved = true
	status.Running = resource.Running
	status.Ready = resource.Serving != nil && *resource.Serving
	status.Degraded = resource.Running && ((resource.Healthy != nil && !*resource.Healthy) || !status.Ready)
	status.Health = resource.Health
	status.Message = resource.Message
	switch {
	case status.Degraded:
		status.State = LightingStateDegraded
	case !resource.Installed:
		status.State = LightingStateNotInstalled
	case !resource.Running:
		status.State = LightingStateStopped
	case status.Ready:
		status.State = LightingStateReady
	default:
		// A running process without a serving observation is degraded, even if
		// the generic resource adapter omitted its health result.
		status.State = LightingStateDegraded
		status.Degraded = true
	}
	return status, nil
}

// Setup validates selected installation and enrolled policy metadata without
// persisting enablement. Native access and confinement are checked at launch.
func (c *LightingControl) Setup(ctx context.Context, selection lightingruntime.Selection) error {
	if c == nil || c.owners.Resource == nil {
		return errors.New("lighting resource owner is unavailable")
	}
	if err := c.owners.Resource.VerifyLightingSetup(ctx, selection); err != nil {
		return fmt.Errorf("verify lighting setup: %w", err)
	}
	return nil
}

// Enable durably admits the exact verified selection for native reconciliation.
func (c *LightingControl) Enable(ctx context.Context, selection lightingruntime.Selection) error {
	if c == nil || c.controller == nil {
		return errors.New("lighting control is unavailable")
	}
	if err := c.controller.Enable(ctx, selection); err != nil {
		return fmt.Errorf("enable lighting: %w", err)
	}
	return nil
}

// Disable persists the start fence, quiesces the scenario writer, and then
// delegates exact backend shutdown to the resource owner.
func (c *LightingControl) Disable(ctx context.Context) error {
	if c == nil || c.controller == nil {
		return errors.New("lighting control is unavailable")
	}
	if err := c.controller.Disable(ctx); err != nil {
		return fmt.Errorf("disable lighting: %w", err)
	}
	return nil
}
