package control

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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
	case "package-state":
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
	if request.Package == "" || len(request.Package) > 256 || !androidPackageName.MatchString(request.Package) {
		return fmt.Errorf("package must be a fully-qualified application identity")
	}
	if request.Executable != "" || len(request.Arguments) > 0 {
		return fmt.Errorf("executable paths and arguments require a strategy-owned allowlist")
	}
	if len(request.Permission) > 256 || len(request.Value) > 4096 {
		return fmt.Errorf("lifecycle parameters exceed bounded limits")
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
	session, err := s.sessionForLease(ctx, deviceID, request.LeaseToken)
	if err != nil {
		return AppLifecycleResponse{}, err
	}
	adapter, ok := s.strategyForFlow(deviceID, "")
	if !ok {
		return AppLifecycleResponse{}, fmt.Errorf("device %q has no available transport", deviceID)
	}
	declaration, err := adapter.Describe(ctx)
	if err != nil {
		return AppLifecycleResponse{}, fmt.Errorf("describe lifecycle strategy: %w", err)
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
		return AppLifecycleResponse{}, &strategy.AvailabilityError{Reason: reason, NextAction: nextAction}
	}
	lifecycle, ok := adapter.(strategy.AppLifecycle)
	if !ok {
		return AppLifecycleResponse{}, &strategy.UnsupportedCapabilityError{Capability: strategy.CapAppLifecycle, Operation: request.Operation}
	}
	commandID := uuid.NewString()
	result, operationErr := lifecycle.AppLifecycle(ctx, request.AppLifecycleRequest)
	outcome := "success"
	if operationErr != nil {
		outcome = "failed"
	}
	audit := Audit{ID: uuid.NewString(), Actor: request.Actor, DeviceID: deviceID, Transport: declaration.Transport, OperationID: commandID, LeaseID: session.ID, Verb: "app-" + request.Operation, Outcome: outcome, RedactionVerified: true, Interactive: true, EvidenceBacked: false}
	s.persistDirectAudit(ctx, audit)
	response := AppLifecycleResponse{CommandID: commandID, Result: result, Audit: audit}
	if operationErr != nil {
		return response, operationErr
	}
	return response, nil
}
