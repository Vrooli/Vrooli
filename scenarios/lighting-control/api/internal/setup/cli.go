// Package setup transports only fixed typed lighting operations to the host owner.
package setup

import (
	"context"
	"errors"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/setup"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"io"
)

type CLI struct{}
type bounded struct {
	data     []byte
	overflow bool
}

func (b *bounded) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 65536 - len(b.data)
	if len(p) > remaining {
		p = p[:remaining]
		b.overflow = true
	}
	b.data = append(b.data, p...)
	return n, nil
}
func (CLI) Status(ctx context.Context) (*setupv1.LightingRuntimeStatus, error) {
	return invoke(ctx, cliinvoke.RuntimeLightingStatus())
}
func (CLI) Disable(ctx context.Context) (*setupv1.LightingRuntimeStatus, error) {
	return invoke(ctx, cliinvoke.RuntimeLightingDisable())
}
func (CLI) Setup(ctx context.Context, s *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error) {
	return selected(ctx, s, false)
}
func (CLI) Enable(ctx context.Context, s *setupv1.LightingSelection) (*setupv1.LightingRuntimeStatus, error) {
	return selected(ctx, s, true)
}
func selected(ctx context.Context, s *setupv1.LightingSelection, enable bool) (*setupv1.LightingRuntimeStatus, error) {
	if s == nil {
		return nil, errors.New("an exact installation selection is required")
	}
	source := ""
	switch s.Source {
	case setupv1.InstallationSource_INSTALLATION_SOURCE_MANAGED:
		source = "managed"
	case setupv1.InstallationSource_INSTALLATION_SOURCE_EXISTING:
		source = "existing"
	default:
		return nil, errors.New("select managed or existing installation")
	}
	argv := cliinvoke.RuntimeLightingSetup(source, s.Version, s.Sha256, s.DevicePolicyId)
	if enable {
		argv = cliinvoke.RuntimeLightingEnable(source, s.Version, s.Sha256, s.DevicePolicyId)
	}
	return invoke(ctx, argv)
}
func invokeMessage(ctx context.Context, argv []string, message proto.Message) error {
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{})
	if err != nil {
		return err
	}
	output := &bounded{}
	result := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: binary, Args: argv, Stdout: output, Stderr: io.Discard})
	if output.overflow {
		return errors.New("lighting runtime response exceeds size limit")
	}
	parseErr := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(output.data, message)
	if parseErr != nil {
		proto.Reset(message)
	}
	if result.Class != cliinvoke.OK {
		return result.Error()
	}
	return parseErr
}
func invoke(ctx context.Context, argv []string) (*setupv1.LightingRuntimeStatus, error) {
	status := &setupv1.LightingRuntimeStatus{}
	if err := invokeMessage(ctx, argv, status); err != nil {
		return nil, err
	}
	switch status.Phase {
	case "disabled", "enabled", "disabling":
	default:
		return nil, errors.New("lighting runtime returned an invalid desired-state phase")
	}
	if status.State == "" {
		return nil, errors.New("lighting runtime returned no observation state")
	}
	return status, nil
}

func (CLI) Devices(ctx context.Context) (*setupv1.LightingDeviceDiscovery, error) {
	m := &setupv1.LightingDeviceDiscovery{}
	err := invokeMessage(ctx, cliinvoke.RuntimeLightingDevices(), m)
	if err != nil {
		return nil, err
	}
	return m, err
}
func (CLI) PlanDevices(ctx context.Context, ids []string) (*setupv1.LightingDevicePlan, error) {
	m := &setupv1.LightingDevicePlan{}
	err := invokeMessage(ctx, cliinvoke.RuntimeLightingPermissionPlan(ids), m)
	if err != nil {
		return nil, err
	}
	return m, err
}
func (CLI) ApplyDevices(ctx context.Context, id string) (*setupv1.LightingDeviceAccessResult, error) {
	m := &setupv1.LightingDeviceAccessResult{}
	err := invokeMessage(ctx, cliinvoke.RuntimeLightingPermissionApply(id), m)
	if err != nil && m.PlanId == "" {
		return nil, err
	}
	return m, err
}
func (CLI) RollbackDevices(ctx context.Context, id string) (*setupv1.LightingDeviceAccessResult, error) {
	m := &setupv1.LightingDeviceAccessResult{}
	err := invokeMessage(ctx, cliinvoke.RuntimeLightingPermissionRollback(id), m)
	if err != nil && m.PlanId == "" {
		return nil, err
	}
	return m, err
}
