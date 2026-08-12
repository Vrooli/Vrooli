package onboard

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/cprev"
	"vrooli-bridge/internal/machines"
	"vrooli-bridge/internal/onboard"

	"connectrpc.com/connect"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

// dryRunHeader is the canonical header cli-core sets on `--dry-run`
// (cli-core/cliutil.DryRunHeader). StartOnboarding honours it by validating then
// short-circuiting before the first side effect.
const dryRunHeader = "X-Dry-Run"

// Deps wires the seams the Connect onboard handler needs. All verbs are
// owner-gated operator verbs.
type Deps struct {
	Service  onboard.Service
	Attempts attemptLookup
	Machines machineReader
	Logger   *log.Logger
}

type attemptLookup interface {
	GetAttemptByCorrelation(context.Context, string) (onboard.EnrollmentAttempt, error)
}

type connectHandler struct {
	deps Deps
}

// machineReader validates an explicitly selected Machine before onboarding can
// use its Bridge-owned SSH key. Locator matching here is validation, not
// identity discovery: the caller must provide the durable Machine ID.
type machineReader interface {
	Get(context.Context, string) (machines.Machine, error)
}

// NewConnectHandler constructs the handler, defaulting the logger.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// StartOnboarding validates the request, then (unless a dry-run) creates a
// durable onboarding op and launches the server-owned orchestration. Owner-gated.
// The SSH password is consumed request-scoped: it is copied into an owned []byte
// handed to the service (which zeroes it) and never retained here.
func (h *connectHandler) StartOnboarding(ctx context.Context, req *connect.Request[onboardv1.StartOnboardingRequest]) (*connect.Response[onboardv1.StartOnboardingResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	if actor == "" {
		actor = "owner"
	}

	dryRun := req.Header().Get(dryRunHeader) == "true"
	if !dryRun && req.Msg.GetMachineId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("machine_id is required for onboarding; create or select a Machine first"))
	}
	if !dryRun && h.deps.Machines != nil {
		if err := validateMachineTarget(ctx, h.deps.Machines, req.Msg.GetMachineId(), req.Msg.GetHost()); err != nil {
			return nil, err
		}
	}

	// Copy the password into an owned mutable slice for the service to zero; the
	// proto string itself is immutable and cannot be wiped, so we never hold it.
	var password []byte
	if pw := req.Msg.GetSshPassword(); pw != "" {
		password = []byte(pw)
	}
	var setupPassphrase []byte
	if passphrase := req.Msg.GetSetupPassphrase(); passphrase != "" {
		setupPassphrase = []byte(passphrase)
	}

	input := onboard.StartInput{
		Actor:                actor,
		MachineID:            req.Msg.GetMachineId(),
		Host:                 req.Msg.GetHost(),
		Port:                 int(req.Msg.GetPort()),
		User:                 req.Msg.GetUser(),
		Password:             password,
		SetupPassphrase:      setupPassphrase,
		NodeName:             req.Msg.GetNodeName(),
		TargetRevision:       req.Msg.GetTargetRevision(),
		RepoURL:              req.Msg.GetRepoUrl(),
		CheckoutDir:          req.Msg.GetCheckoutDir(),
		ControlPlaneURL:      req.Msg.GetControlPlaneUrl(),
		ReachabilityMode:     req.Msg.GetReachabilityMode(),
		Capabilities:         req.Msg.GetCapabilities(),
		VerifyTimeoutSeconds: req.Msg.GetVerifyTimeoutSeconds(),
		SkipSetup:            req.Msg.GetSkipSetup(),
		SkipPrereqs:          req.Msg.GetSkipPrereqs(),
		ProvisionSudo:        req.Msg.GetProvisionSudo(),
		SetupEnvironment:     req.Msg.GetSetupEnvironment(),
		SetupResources:       req.Msg.GetSetupResources(),
		SetupScenarios:       req.Msg.GetSetupScenarios(),
		IncludeOptional:      req.Msg.GetIncludeOptional(),
		SourceMode:           sourceModeFromProto(req.Msg.GetSourceMode()),
		DryRun:               dryRun,
	}
	var dec onboard.Decision
	machineID, attemptID := "", ""
	if req.Msg.GetMachineId() != "" && !dryRun {
		var machineDecision onboard.MachineEnrollmentDecision
		var machineErr error
		if priorAttemptID := req.Msg.GetRetryOfEnrollmentAttemptId(); priorAttemptID != "" {
			machineDecision, machineErr = h.deps.Service.RetryMachineEnrollment(ctx, req.Msg.GetMachineId(), priorAttemptID, input)
		} else {
			machineDecision, machineErr = h.deps.Service.StartMachineEnrollment(ctx, req.Msg.GetMachineId(), input)
		}
		dec, err = machineDecision.Decision, machineErr
		machineID, attemptID = req.Msg.GetMachineId(), machineDecision.Attempt.ID
	} else {
		dec, err = h.deps.Service.Start(ctx, input)
	}
	if err != nil {
		// Revision-resolution failures (unsafe ref, unpushed commit, no CP commit)
		// carry their own friendly Connect codes; fall back to the domain mapping
		// for everything else.
		if ce := cprev.ConnectError(err); ce != nil {
			return nil, ce
		}
		connectErr := onboard.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("onboard.StartOnboarding(host=%q): %v", req.Msg.GetHost(), err)
		}
		return nil, connectErr
	}

	return connect.NewResponse(&onboardv1.StartOnboardingResponse{
		OpId:                dec.OpID,
		DryRun:              dec.DryRun,
		Host:                dec.Host,
		Port:                int32(dec.Port),
		User:                dec.User,
		MachineId:           machineID,
		EnrollmentAttemptId: attemptID,
	}), nil
}

func validateMachineTarget(ctx context.Context, reader machineReader, machineID, host string) error {
	machine, err := reader.Get(ctx, strings.TrimSpace(machineID))
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("selected Machine %q is unavailable: %w", machineID, err))
	}
	if machine.Lifecycle != machines.LifecycleActive {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("selected Machine %q is %s and cannot be onboarded", machineID, machine.Lifecycle))
	}
	target := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	for _, locator := range machine.Locators {
		kind := strings.ToLower(strings.TrimSpace(locator.Kind))
		value := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(locator.Value), "."))
		if (kind == "hostname" || kind == "ip" || kind == "ssh") && value == target {
			return nil
		}
	}
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("host %q does not match any locator on selected Machine %q; select the correct durable Machine instead of creating a replacement", host, machineID))
}

func (h *connectHandler) GetOnboarding(ctx context.Context, req *connect.Request[onboardv1.GetOnboardingRequest]) (*connect.Response[onboardv1.GetOnboardingResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	op, events, err := h.deps.Service.GetOp(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.mapErr("GetOnboarding", req.Msg.GetId(), err)
	}
	resp := &onboardv1.GetOnboardingResponse{
		Op:     domainOpToProto(op),
		Events: make([]*onboardv1.OnboardingStepEvent, 0, len(events)),
	}
	h.composeAttempt(ctx, op, resp.Op)
	for _, ev := range events {
		resp.Events = append(resp.Events, domainStepEventToProto(ev))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListOnboardings(ctx context.Context, req *connect.Request[onboardv1.ListOnboardingsRequest]) (*connect.Response[onboardv1.ListOnboardingsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	list, err := h.deps.Service.ListOps(ctx, onboard.ListFilter{
		Host:  req.Msg.GetHost(),
		Limit: int(req.Msg.GetLimit()),
	})
	if err != nil {
		h.deps.Logger.Printf("onboard.ListOnboardings: %v", err)
		return nil, onboard.ToConnectError(err)
	}
	resp := &onboardv1.ListOnboardingsResponse{Ops: make([]*onboardv1.OnboardingOp, 0, len(list))}
	for _, op := range list {
		wireOp := domainOpToProto(op)
		h.composeAttempt(ctx, op, wireOp)
		resp.Ops = append(resp.Ops, wireOp)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) composeAttempt(ctx context.Context, op onboard.Op, wire *onboardv1.OnboardingOp) {
	if h.deps.Attempts == nil || op.CorrelationID == "" || wire == nil {
		return
	}
	attempt, err := h.deps.Attempts.GetAttemptByCorrelation(ctx, op.CorrelationID)
	if err != nil {
		return // Legacy operations have no immutable Machine attempt.
	}
	wire.MachineId = attempt.MachineID
	wire.EnrollmentAttemptId = attempt.ID
}

func (h *connectHandler) WaitOnboarding(ctx context.Context, req *connect.Request[onboardv1.WaitOnboardingRequest]) (*connect.Response[onboardv1.WaitOnboardingResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	timeout := time.Duration(req.Msg.GetTimeoutSeconds()) * time.Second
	op, timedOut, err := h.deps.Service.Wait(ctx, req.Msg.GetId(), timeout)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, h.mapErr("WaitOnboarding", req.Msg.GetId(), err)
	}
	return connect.NewResponse(&onboardv1.WaitOnboardingResponse{
		Op:       domainOpToProto(op),
		TimedOut: timedOut,
	}), nil
}

func (h *connectHandler) CancelOnboarding(ctx context.Context, req *connect.Request[onboardv1.CancelOnboardingRequest]) (*connect.Response[onboardv1.CancelOnboardingResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	op, err := h.deps.Service.Cancel(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.mapErr("CancelOnboarding", req.Msg.GetId(), err)
	}
	return connect.NewResponse(&onboardv1.CancelOnboardingResponse{Op: domainOpToProto(op)}), nil
}

func (h *connectHandler) RemoveFailedOnboarding(ctx context.Context, req *connect.Request[onboardv1.RemoveFailedOnboardingRequest]) (*connect.Response[onboardv1.RemoveFailedOnboardingResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	if err := h.deps.Service.RemoveFailed(ctx, req.Msg.GetId()); err != nil {
		return nil, h.mapErr("RemoveFailedOnboarding", req.Msg.GetId(), err)
	}
	return connect.NewResponse(&onboardv1.RemoveFailedOnboardingResponse{}), nil
}

// mapErr logs internal errors and returns the Connect translation.
func (h *connectHandler) mapErr(op, id string, err error) error {
	connectErr := onboard.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("onboard.%s(%q): %v", op, id, err)
	}
	return connectErr
}
