package main

// machines_catalog.go is the operator-facing fleet surface. It answers the two
// questions a person actually has about another computer — does it answer, and
// what may it do — and owns the join flow so that linking a machine never
// requires opening the control plane's own interface.
//
// It is a projection, not a second authority. Reachability comes from the same
// readiness vocabulary the session launcher renders, permission presets are
// served by the control plane, and the scopes behind a preset are never
// re-derived here. Bridge owner credentials stay on this side of the wire.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/nodeclient"
	"google.golang.org/protobuf/types/known/timestamppb"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/machines/machines_v1connect"
)

// bridgeCallTimeout bounds every control-plane call this surface makes. The
// machines list is rendered on open, so a slow control plane must degrade to a
// stated reason rather than an unbounded spinner.
const bridgeCallTimeout = 3 * time.Second

type machineRPC struct {
	server *Server
}

func (s *Server) mountMachines() {
	path, handler := machinesconnect.NewMachineServiceHandler(&machineRPC{server: s})
	connectx.RegisterServices(s.router, connectx.ServiceMount{Path: path, Handler: handler})
}

// fleet is one consistent read of the control plane: the machines it holds and
// the enrollment state an owner may act on. Both are fetched together because
// the surface renders them together, and a partial read would show a pending
// request with no presets to answer it with.
type fleet struct {
	machines   []*machinesv1.Machine
	requests   []*machinesv1.JoinRequest
	presets    []*machinesv1.PermissionPreset
	state      machinesv1.FleetState
	message    string
	recovery   string
	controlURL string
	consoleURL string
	reachable  bool
	detail     string
}

func (h *machineRPC) List(ctx context.Context, _ *connect.Request[machinesv1.ListRequest]) (*connect.Response[machinesv1.ListResponse], error) {
	view := h.readFleet(ctx)
	return connect.NewResponse(&machinesv1.ListResponse{
		State:          view.state,
		Machines:       view.machines,
		JoinRequests:   view.requests,
		Presets:        view.presets,
		Message:        view.message,
		RecoveryAction: view.recovery,
		ControlPlane: &machinesv1.ControlPlane{
			Reachable:  view.reachable,
			Endpoint:   view.controlURL,
			Detail:     view.detail,
			ConsoleUrl: view.consoleURL,
		},
	}), nil
}

func (h *machineRPC) IssueCode(ctx context.Context, req *connect.Request[machinesv1.IssueCodeRequest]) (*connect.Response[machinesv1.IssueCodeResponse], error) {
	client, _, err := h.client(ctx)
	if err != nil {
		return nil, err
	}
	issued, issueErr := client.IssueCode(ctx, nodeclient.CodeRequest{Name: req.Msg.GetLabel()}, bridgeCallTimeout)
	if issueErr != nil {
		return nil, controlPlaneError("issue a join code", issueErr)
	}
	expiresAt := issued.GetExpiresAt()
	var remaining int64
	if expiresAt != nil {
		remaining = int64(time.Until(expiresAt.AsTime()).Seconds())
		if remaining < 0 {
			remaining = 0
		}
	}
	return connect.NewResponse(&machinesv1.IssueCodeResponse{
		Code:             issued.GetCode(),
		ExpiresAt:        expiresAt,
		ExpiresInSeconds: remaining,
	}), nil
}

func (h *machineRPC) Decide(ctx context.Context, req *connect.Request[machinesv1.DecideRequest]) (*connect.Response[machinesv1.DecideResponse], error) {
	client, _, err := h.client(ctx)
	if err != nil {
		return nil, err
	}
	requestID := strings.TrimSpace(req.Msg.GetRequestId())
	if requestID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a join request is required"))
	}
	scopes := req.Msg.GetScopes()
	if preset := strings.TrimSpace(req.Msg.GetPreset()); preset != "" {
		resolved, resolveErr := h.scopesForPreset(ctx, client, preset)
		if resolveErr != nil {
			return nil, resolveErr
		}
		scopes = resolved
	}
	nodeID, decideErr := client.Decide(ctx, nodeclient.DecideRequest{
		RequestID:         requestID,
		Approve:           req.Msg.GetApprove(),
		Scopes:            scopes,
		ConfirmationWords: req.Msg.GetConfirmationWords(),
	}, bridgeCallTimeout)
	if decideErr != nil {
		return nil, controlPlaneError("answer this join request", decideErr)
	}
	if !req.Msg.GetApprove() {
		return connect.NewResponse(&machinesv1.DecideResponse{Message: "This machine was not linked."}), nil
	}
	return connect.NewResponse(&machinesv1.DecideResponse{
		Machine: h.machineByNodeID(ctx, nodeID),
		Message: "This machine is linked.",
	}), nil
}

func (h *machineRPC) SetGrant(ctx context.Context, req *connect.Request[machinesv1.SetGrantRequest]) (*connect.Response[machinesv1.SetGrantResponse], error) {
	client, _, err := h.client(ctx)
	if err != nil {
		return nil, err
	}
	nodeID, err := nodeIDForMachine(req.Msg.GetMachineId())
	if err != nil {
		return nil, err
	}
	scopes := req.Msg.GetScopes()
	if preset := strings.TrimSpace(req.Msg.GetPreset()); preset != "" {
		resolved, resolveErr := h.scopesForPreset(ctx, client, preset)
		if resolveErr != nil {
			return nil, resolveErr
		}
		scopes = resolved
	}
	node, setErr := client.SetScopes(ctx, nodeID, scopes, bridgeCallTimeout)
	if setErr != nil {
		return nil, controlPlaneError("change what this machine may do", setErr)
	}
	presets := h.presetsOnly(ctx, client)
	return connect.NewResponse(&machinesv1.SetGrantResponse{Machine: h.machineForNode(node, presets)}), nil
}

func (h *machineRPC) Forget(ctx context.Context, req *connect.Request[machinesv1.ForgetRequest]) (*connect.Response[machinesv1.ForgetResponse], error) {
	client, _, err := h.client(ctx)
	if err != nil {
		return nil, err
	}
	nodeID, err := nodeIDForMachine(req.Msg.GetMachineId())
	if err != nil {
		return nil, err
	}
	if forgetErr := client.Forget(ctx, nodeID, bridgeCallTimeout); forgetErr != nil {
		return nil, controlPlaneError("forget this machine", forgetErr)
	}
	return connect.NewResponse(&machinesv1.ForgetResponse{ForgottenMachineId: req.Msg.GetMachineId()}), nil
}

// readFleet assembles the whole surface in one pass. An unreachable or
// unenrolled control plane is a stated outcome with a recovery action, never an
// empty list the client has to interpret.
func (h *machineRPC) readFleet(ctx context.Context) fleet {
	targets := h.server.remoteTargets()
	view := fleet{
		machines: make([]*machinesv1.Machine, 0, len(targets)+1),
		requests: []*machinesv1.JoinRequest{},
		presets:  []*machinesv1.PermissionPreset{},
	}
	client, base := bridgeNodeClient(ctx)
	view.controlURL = base.BaseURL
	view.consoleURL = h.controlPlaneConsoleURL(ctx)
	view.reachable = client != nil

	local := &machinesv1.Machine{
		Target: targetToProto(targetConnection{Target: localTerminalTarget()}),
		Grant: &machinesv1.Grant{
			Summary:       "Full control on this computer",
			Effects:       []string{"read", "write", "destructive"},
			CoversAllApps: true,
		},
		Manageable: false,
	}
	view.machines = append(view.machines, local)

	if client == nil {
		view.state = machinesv1.FleetState_FLEET_STATE_UNENROLLED
		view.message = base.Reason
		view.recovery = base.NextAction
		view.detail = base.Reason
		return view
	}

	enrollment, enrollErr := client.ListEnrollment(ctx, false, bridgeCallTimeout)
	if enrollErr == nil {
		view.presets = presetsToProto(enrollment.Presets)
		view.requests = joinRequestsToProto(enrollment.Requests)
	}

	linked := 0
	for _, target := range targets {
		// The synthetic placeholder the catalog returns when Bridge holds no
		// nodes carries no node identity; it is a state, not a machine.
		if strings.TrimSpace(target.NodeID) == "" {
			view.message = target.Reason
			view.recovery = target.NextAction
			view.detail = target.Reason
			continue
		}
		linked++
		view.machines = append(view.machines, machineFromTarget(target, view.presets))
	}

	switch {
	case linked > 0:
		view.state = machinesv1.FleetState_FLEET_STATE_READY
	default:
		view.state = machinesv1.FleetState_FLEET_STATE_EMPTY
	}
	if enrollErr != nil && view.message == "" {
		view.message = "The control plane did not report which machines are asking to join."
		view.recovery = "Check Bridge health, then refresh."
	}
	return view
}

// machineFromTarget projects one catalog target into the machines vocabulary.
// The readiness facts stay untouched: the launcher and this surface must never
// disagree about whether a machine answers.
func machineFromTarget(target targetConnection, presets []*machinesv1.PermissionPreset) *machinesv1.Machine {
	return &machinesv1.Machine{
		Target:              targetToProto(target),
		Grant:               grantFromScopes(target.Scopes, presets),
		HeartbeatAgeSeconds: heartbeatAgeSeconds(target.LastSeenAt),
		Manageable:          true,
	}
}

// grantFromScopes says the same thing three ways so each reader gets the form
// they need: a sentence, a shape, and the list itself.
func grantFromScopes(scopes []string, presets []*machinesv1.PermissionPreset) *machinesv1.Grant {
	cleaned := append([]string(nil), scopes...)
	effects, apps, wildcard := classifyScopes(cleaned)
	return &machinesv1.Grant{
		Summary:       grantSummaryForScopes(cleaned),
		Effects:       effects,
		AppCount:      int32(apps),
		CoversAllApps: wildcard,
		Scopes:        cleaned,
		Preset:        matchPreset(cleaned, presets),
	}
}

// classifyScopes reduces a scope list to the two things an operator judges it
// by: which effects it confers, and how far it reaches. A wildcard namespace is
// reported separately from a count because it also reaches apps that do not
// exist yet, which no count can express.
func classifyScopes(scopes []string) (effects []string, appCount int, coversAllApps bool) {
	return scopecatalog.ClassifyScopes(scopes)
}

func (h *machineRPC) machineForNode(node *registryv1.Node, presets []*machinesv1.PermissionPreset) *machinesv1.Machine {
	if node == nil {
		return nil
	}
	_, base := bridgeNodeClient(context.Background())
	return machineFromTarget(targetFromRegistryNode(base, node), presets)
}

// machineByNodeID re-reads the fleet after a mutation so the client renders the
// control plane's post-state rather than an optimistic guess.
func (h *machineRPC) machineByNodeID(ctx context.Context, nodeID string) *machinesv1.Machine {
	if strings.TrimSpace(nodeID) == "" {
		return nil
	}
	for _, machine := range h.readFleet(ctx).machines {
		if machine.GetTarget().GetNodeId() == nodeID {
			return machine
		}
	}
	return nil
}

// scopesForPreset asks the control plane what a preset grants. Resolving it
// here rather than in the browser is what keeps a single authorization
// vocabulary: the console names a posture, the control plane owns its meaning.
func (h *machineRPC) scopesForPreset(ctx context.Context, client *nodeclient.Client, preset string) ([]string, error) {
	enrollment, err := client.ListEnrollment(ctx, false, bridgeCallTimeout)
	if err != nil {
		return nil, controlPlaneError("read the permission presets", err)
	}
	for _, candidate := range enrollment.Presets {
		if strings.EqualFold(candidate.GetName(), preset) {
			return append([]string(nil), candidate.GetScopes()...), nil
		}
	}
	return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("the control plane does not offer a %q permission preset", preset))
}

func (h *machineRPC) presetsOnly(ctx context.Context, client *nodeclient.Client) []*machinesv1.PermissionPreset {
	enrollment, err := client.ListEnrollment(ctx, false, bridgeCallTimeout)
	if err != nil {
		return nil
	}
	return presetsToProto(enrollment.Presets)
}

func (h *machineRPC) client(ctx context.Context) (*nodeclient.Client, targetConnection, error) {
	client, base := bridgeNodeClient(ctx)
	if client == nil {
		return nil, base, connect.NewError(connect.CodeFailedPrecondition, errors.New("this computer is not enrolled with a fleet control plane"))
	}
	return client, base, nil
}

func presetsToProto(presets []*pairingv1.PermissionPreset) []*machinesv1.PermissionPreset {
	out := make([]*machinesv1.PermissionPreset, 0, len(presets))
	for _, preset := range presets {
		if preset == nil {
			continue
		}
		scopes := append([]string(nil), preset.GetScopes()...)
		effects, apps, _ := classifyScopes(scopes)
		out = append(out, &machinesv1.PermissionPreset{
			Name:        preset.GetName(),
			Title:       presetTitle(preset.GetName()),
			Description: preset.GetDescription(),
			Scopes:      scopes,
			Withholds:   append([]string(nil), preset.GetWithholds()...),
			Summary:     grantSummaryForScopes(scopes),
			Effects:     effects,
			AppCount:    int32(apps),
		})
	}
	return out
}

func joinRequestsToProto(requests []*pairingv1.PairingRequest) []*machinesv1.JoinRequest {
	out := make([]*machinesv1.JoinRequest, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		var requestedAt *timestamppb.Timestamp
		var age int64
		if created := request.GetCreatedAt(); created != nil {
			requestedAt = created
			age = int64(time.Since(created.AsTime()).Seconds())
			if age < 0 {
				age = 0
			}
		}
		out = append(out, &machinesv1.JoinRequest{
			Id:                  request.GetId(),
			Name:                request.GetName(),
			Os:                  request.GetOs(),
			Arch:                request.GetArch(),
			Endpoint:            request.GetEndpoint(),
			ConfirmationWords:   append([]string(nil), request.GetConfirmationWords()...),
			KeyFingerprint:      request.GetKeyFingerprint(),
			RequestedAt:         requestedAt,
			RequestedAgeSeconds: age,
		})
	}
	return out
}

// matchPreset names the posture a machine's grant corresponds to, or nothing
// when the grant was customized. Reporting a near-match as an exact one would
// hide a scope the operator never chose, so the comparison is exact.
func matchPreset(scopes []string, presets []*machinesv1.PermissionPreset) string {
	if len(scopes) == 0 {
		return ""
	}
	want := normalizedScopes(scopes)
	for _, preset := range presets {
		if want == normalizedScopes(preset.GetScopes()) {
			return preset.GetName()
		}
	}
	return ""
}

// presetTitle turns a control-plane identifier into the words a person reads.
// The identifier stays the value that travels on the wire; only the label
// changes, so the two never diverge into separate vocabularies.
func presetTitle(name string) string {
	words := strings.FieldsFunc(strings.TrimSpace(name), func(r rune) bool { return r == '-' || r == '_' })
	for i, word := range words {
		if i == 0 && word != "" {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
			continue
		}
		words[i] = strings.ToLower(word)
	}
	return strings.Join(words, " ")
}

func normalizedScopes(scopes []string) string {
	cleaned := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if trimmed := strings.ToLower(strings.TrimSpace(scope)); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	sort.Strings(cleaned)
	return strings.Join(cleaned, "\n")
}

func heartbeatAgeSeconds(lastSeen time.Time) int64 {
	if lastSeen.IsZero() {
		return 0
	}
	age := int64(time.Since(lastSeen).Seconds())
	if age < 0 {
		return 0
	}
	return age
}

// nodeIDForMachine accepts the catalog's machine id and returns the control
// plane's node id. The prefix exists so a machine id is never mistaken for a
// node id in a log or a URL.
func nodeIDForMachine(machineID string) (string, error) {
	trimmed := strings.TrimSpace(machineID)
	nodeID := strings.TrimPrefix(trimmed, "bridge-node:")
	if nodeID == "" || nodeID == trimmed {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%q is not a linked machine", machineID))
	}
	return nodeID, nil
}

// controlPlaneConsoleURL resolves a link a browser can actually open to the
// control plane's own interface.
//
// This is deliberately not view.controlURL. That value is the Bridge API base
// this process dials — resolved through the API port, against loopback, on the
// server. Handing it to a browser produces a 404 from a Connect endpoint, and
// on any access path other than a browser on this same machine it names the
// wrong computer entirely. Origin belongs to the caller, so the resolution
// takes the host the request arrived on and reads the UI port.
//
// An empty result is a valid answer: it means the interface could not be
// located, and the client hides the link rather than offering a broken one.
func (h *machineRPC) controlPlaneConsoleURL(ctx context.Context) string {
	if h.server == nil || h.server.resolveConsoleURL == nil {
		return ""
	}
	url, err := h.server.resolveConsoleURL(ctx, "vrooli-bridge", discovery.ExternalHostFromContext(ctx))
	if err != nil {
		return ""
	}
	return url
}

// controlPlaneError keeps a refusal legible. A rejected confirmation is a
// safety outcome the operator must see as such; everything else is reported as
// the control plane being unable to complete the action.
func controlPlaneError(action string, err error) error {
	if nodeclient.IsKind(err, nodeclient.ErrInvalidRequest) {
		// The words are the one field the joining machine could not choose for
		// itself, so a mismatch is the single refusal an operator must be able
		// to act on. It gets a sentence rather than the wire message.
		if strings.Contains(err.Error(), "confirmation_words") {
			return connect.NewError(connect.CodeInvalidArgument, errors.New(
				"those words do not match what this machine is showing; if they never match, deny the request — something else is asking to join"))
		}
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if nodeclient.IsKind(err, nodeclient.ErrNodeNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("could not %s: %w", action, err))
}
