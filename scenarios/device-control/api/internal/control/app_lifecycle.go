package control

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"device-control/strategy"
	"github.com/google/uuid"
)

var androidPackageName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)

// AppLifecycleOperation is the authenticated control-plane request for one
// app operation. It deliberately carries no shell command or raw executable
// path from the caller.
type AppLifecycleOperation struct {
	Actor      string `json:"actor"`
	LeaseToken string `json:"lease_token"`
	Confirmed  bool   `json:"confirmed"`
	strategy.AppLifecycleRequest
}

type AppLifecycleResponse struct {
	CommandID string                      `json:"command_id"`
	Result    strategy.AppLifecycleResult `json:"result"`
	Audit     Audit                       `json:"audit"`
}

func lifecycleNeedsConfirmation(operation string) bool {
	switch operation {
	case "home-state", "package-state", "package-signing-state", "network-state", "accessibility-state", "foreground-state", "process-state", "app-status", "performance-state":
		return false
	default:
		return true
	}
}

func validateAppLifecycleRequest(request strategy.AppLifecycleRequest) error {
	request.Operation = strings.ToLower(strings.TrimSpace(request.Operation))
	if request.Operation == "" {
		return fmt.Errorf("app lifecycle operation is required")
	}
	if request.Operation == "network-state" {
		if request.Package != "" {
			return fmt.Errorf("network-state does not accept a package identity")
		}
	} else if request.Package == "" || len(request.Package) > 256 || !androidPackageName.MatchString(request.Package) {
		return fmt.Errorf("package must be a fully-qualified application identity")
	}
	if request.Executable != "" || len(request.Arguments) > 0 {
		return fmt.Errorf("executable paths and arguments require a strategy-owned allowlist")
	}
	if len(request.Permission) > 256 || len(request.Value) > 4096 {
		return fmt.Errorf("lifecycle parameters exceed bounded limits")
	}
	if len(request.ExpectedValue) > 64 || (request.ExpectedValue != "" && request.Operation != "package-enabled-set") {
		return fmt.Errorf("expected_value is only supported for checked package enabled-state changes")
	}
	return nil
}

// ExecuteAppLifecycle enforces the lease, declared capability, typed adapter
// seam, and confirmation policy before a lifecycle effect. A strategy that
// only declares the capability but lacks AppLifecycle is unavailable rather
// than being silently routed through a generic input command.
func (s *Service) ExecuteAppLifecycle(ctx context.Context, deviceID string, request AppLifecycleOperation) (AppLifecycleResponse, error) {
	request.Operation = strings.ToLower(strings.TrimSpace(request.Operation))
	if err := validateAppLifecycleRequest(request.AppLifecycleRequest); err != nil {
		return AppLifecycleResponse{}, err
	}
	if lifecycleNeedsConfirmation(request.Operation) && !request.Confirmed {
		return AppLifecycleResponse{}, fmt.Errorf("app lifecycle operation %q requires explicit confirmation", request.Operation)
	}
	target, err := s.resolveLifecycleTarget(ctx, deviceID, request.LeaseToken, request.Operation)
	if err != nil {
		return AppLifecycleResponse{}, err
	}
	return s.runLifecycle(ctx, target, request)
}

// lifecycleTarget is a lease-checked, capability-probed route to one device's
// app-lifecycle adapter. Resolving it once lets a multi-operation caller (such
// as LaunchApp) skip repeating the adapter's capability probe per operation.
type lifecycleTarget struct {
	deviceID  string
	session   Session
	adapter   strategy.AppLifecycle
	transport string
}

func (s *Service) resolveLifecycleTarget(ctx context.Context, deviceID, leaseToken, operation string) (lifecycleTarget, error) {
	session, err := s.sessionForLease(ctx, deviceID, leaseToken)
	if err != nil {
		return lifecycleTarget{}, err
	}
	// A composed identity (Cast plus ADB, say) may be reachable by several
	// transports of which only one can run app lifecycle. Try the default route
	// first, then each retained profile, and keep the first that declares it.
	routes := []string{""}
	if record, found := s.devices.Get(deviceID); found {
		for _, profile := range record.Transports {
			routes = append(routes, profile.Name)
		}
	}
	var firstErr error
	for _, route := range routes {
		target, err := s.lifecycleTargetOn(ctx, deviceID, session, route, operation)
		if err == nil {
			return target, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return lifecycleTarget{}, firstErr
}

func (s *Service) lifecycleTargetOn(ctx context.Context, deviceID string, session Session, route, operation string) (lifecycleTarget, error) {
	transport, err := s.requireCurrentEffectRoute(deviceID, route)
	if err != nil {
		return lifecycleTarget{}, err
	}
	adapter, ok := s.strategyForFlow(deviceID, transport)
	if !ok {
		return lifecycleTarget{}, fmt.Errorf("device %q has no available transport", deviceID)
	}
	if operation == "home-set" || operation == "package-enabled-set" {
		reader, readable := adapter.(strategy.StateReader)
		if !readable {
			return lifecycleTarget{}, fmt.Errorf("device configuration requires observable unlocked state")
		}
		state, err := reader.ReadState(ctx)
		if err != nil || state.LockState != "unlocked" || state.ScreenState != "on" {
			return lifecycleTarget{}, fmt.Errorf("device configuration requires an unlocked, on-screen device")
		}
	}
	var declaration strategy.Declaration
	if prober, ok := adapter.(interface {
		DescribePerformance(context.Context) (strategy.Declaration, error)
	}); ok && operation == "performance-state" {
		declaration, err = prober.DescribePerformance(ctx)
	} else {
		declaration, err = adapter.Describe(ctx)
	}
	if err != nil {
		return lifecycleTarget{}, fmt.Errorf("describe lifecycle strategy: %w", err)
	}
	capability, declared := declaration.Capabilities[strategy.CapAppLifecycle]
	if !declared || capability.Status != strategy.StatusAvailable {
		reason := capability.Reason
		if reason == "" {
			reason = "app lifecycle capability is not available on the selected transport"
		}
		nextAction := capability.NextAction
		if nextAction == "" {
			nextAction = "select a transport that declares executable app lifecycle support"
		}
		return lifecycleTarget{}, &strategy.AvailabilityError{Reason: reason, NextAction: nextAction}
	}
	lifecycle, ok := adapter.(strategy.AppLifecycle)
	if !ok {
		return lifecycleTarget{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapAppLifecycle, Operation: operation}
	}
	// Endpoint-bound adapters may omit transport from Describe. Use the route
	// that passed the Device Control inventory guard for the audit record.
	return lifecycleTarget{deviceID: deviceID, session: session, adapter: lifecycle, transport: firstNonEmpty(declaration.Transport, transport)}, nil
}

func (s *Service) runLifecycle(ctx context.Context, target lifecycleTarget, request AppLifecycleOperation) (AppLifecycleResponse, error) {
	commandID := uuid.NewString()
	result, operationErr := target.adapter.AppLifecycle(ctx, request.AppLifecycleRequest)
	outcome := "success"
	if result.Outcome != "" {
		outcome = result.Outcome
	}
	if operationErr != nil {
		outcome = "failed"
	}
	audit := Audit{ID: uuid.NewString(), Actor: request.Actor, DeviceID: target.deviceID, Transport: target.transport, OperationID: commandID, LeaseID: target.session.ID, Verb: "app-" + request.Operation, Outcome: outcome, CreatedAt: time.Now().UTC(), RedactionVerified: true, Interactive: true, EvidenceBacked: false}
	s.persistDirectAudit(ctx, audit)
	response := AppLifecycleResponse{CommandID: commandID, Result: result, Audit: audit}
	return response, operationErr
}
