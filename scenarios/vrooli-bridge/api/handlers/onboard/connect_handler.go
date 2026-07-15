package onboard

import (
	"context"
	"errors"
	"log"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/cprev"
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
	Service onboard.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
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

	// Copy the password into an owned mutable slice for the service to zero; the
	// proto string itself is immutable and cannot be wiped, so we never hold it.
	var password []byte
	if pw := req.Msg.GetSshPassword(); pw != "" {
		password = []byte(pw)
	}

	dec, err := h.deps.Service.Start(ctx, onboard.StartInput{
		Actor:                actor,
		Host:                 req.Msg.GetHost(),
		Port:                 int(req.Msg.GetPort()),
		User:                 req.Msg.GetUser(),
		Password:             password,
		NodeName:             req.Msg.GetNodeName(),
		TargetRevision:       req.Msg.GetTargetRevision(),
		RepoURL:              req.Msg.GetRepoUrl(),
		CheckoutDir:          req.Msg.GetCheckoutDir(),
		ControlPlaneURL:      req.Msg.GetControlPlaneUrl(),
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
	})
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
		OpId:   dec.OpID,
		DryRun: dec.DryRun,
		Host:   dec.Host,
		Port:   int32(dec.Port),
		User:   dec.User,
	}), nil
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
		resp.Ops = append(resp.Ops, domainOpToProto(op))
	}
	return connect.NewResponse(resp), nil
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

// mapErr logs internal errors and returns the Connect translation.
func (h *connectHandler) mapErr(op, id string, err error) error {
	connectErr := onboard.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("onboard.%s(%q): %v", op, id, err)
	}
	return connectErr
}
