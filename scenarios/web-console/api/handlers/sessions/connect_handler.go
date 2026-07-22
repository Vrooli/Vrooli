package sessions

import (
	"context"
	"errors"
	"log"
	"strings"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/sessionstore"
)

// Deps wires the seams the Connect sessions handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// SessionsServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Sentinel errors mapped to Connect codes via classify().
var (
	// ErrNotFound is returned for missing session ids (live or recovery rows).
	// Mapped to CodeNotFound.
	ErrNotFound = errors.New("session not found")

	// ErrInvalidArgument is returned for malformed/missing inputs that the
	// handler-level validation does not catch (policy validation, etc.).
	// Mapped to CodeInvalidArgument.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrResourceExhausted indicates the configured session-limit was hit.
	// Mapped to CodeResourceExhausted.
	ErrResourceExhausted = errors.New("resource exhausted")

	// ErrUnavailable indicates the requested backend is not currently
	// available (tmux missing, etc.). Mapped to CodeUnavailable.
	ErrUnavailable = errors.New("backend unavailable")

	// ErrInternal is returned for unclassified failures we still want to
	// surface as 500. Mapped to CodeInternal.
	ErrInternal = errors.New("internal error")

	// ErrFailedPrecondition is returned when an operation is rejected because
	// the session is not in the right state (e.g. recovering a session that
	// is not awaiting_recovery, or a claude session missing its agent id).
	// Mapped to CodeFailedPrecondition.
	ErrFailedPrecondition = errors.New("failed precondition")
)

const idempotencyHeader = "X-Idempotency-Key"

func (h *connectHandler) Create(ctx context.Context, req *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	in := CreateInput{
		Shell:                req.Msg.GetShell(),
		Cols:                 int(req.Msg.GetCols()),
		Rows:                 int(req.Msg.GetRows()),
		Backend:              req.Msg.GetBackend(),
		LaunchCommand:        req.Msg.GetLaunchCommand(),
		ExecuteLaunchCommand: req.Msg.GetExecuteLaunchCommand(),
		AgentType:            req.Msg.GetAgentType(),
		Origin:               originToString(req.Msg.GetOrigin()),
		Owner:                req.Msg.GetOwner(),
		DisplayLabel:         req.Msg.GetDisplayLabel(),
		IdempotencyKey:       req.Header().Get(idempotencyHeader),
	}
	if req.Msg.GetHasPolicy() && req.Msg.GetPolicy() != nil {
		in.HasPolicy = true
		in.Policy = Policy{
			Mode:     req.Msg.GetPolicy().GetMode(),
			Duration: req.Msg.GetPolicy().GetDuration(),
		}
	}
	sess, err := h.deps.Service.Create(ctx, in)
	if err != nil {
		return nil, h.classify(err, "sessions.Create")
	}
	return connect.NewResponse(&sessionsv1.CreateResponse{Session: sessionToProto(sess)}), nil
}

func (h *connectHandler) List(ctx context.Context, _ *connect.Request[sessionsv1.ListRequest]) (*connect.Response[sessionsv1.ListResponse], error) {
	out, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, h.classify(err, "sessions.List")
	}
	rs := h.deps.Service.RecoveryStatus(ctx)
	return connect.NewResponse(&sessionsv1.ListResponse{
		Sessions: sessionsToProto(out),
		Recovery: &sessionsv1.RecoveryStatus{
			InProgress:        rs.InProgress,
			Total:             int32(rs.Total),
			Recovered:         int32(rs.Recovered),
			AwaitingRecovery:  int32(rs.AwaitingRecovery),
			Adopted:           int32(rs.Adopted),
			StartedAtUnixMs:   rs.StartedAtUnixMs,
			CompletedAtUnixMs: rs.CompletedAtUnixMs,
		},
	}), nil
}

func (h *connectHandler) Get(ctx context.Context, req *connect.Request[sessionsv1.GetRequest]) (*connect.Response[sessionsv1.GetResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	sess, err := h.deps.Service.Get(ctx, id)
	if err != nil {
		return nil, h.classify(err, "sessions.Get")
	}
	return connect.NewResponse(&sessionsv1.GetResponse{Session: sessionToProto(sess)}), nil
}

func (h *connectHandler) Delete(ctx context.Context, req *connect.Request[sessionsv1.DeleteRequest]) (*connect.Response[sessionsv1.DeleteResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.deps.Service.Delete(ctx, id); err != nil {
		return nil, h.classify(err, "sessions.Delete")
	}
	return connect.NewResponse(&sessionsv1.DeleteResponse{}), nil
}

func (h *connectHandler) ListRecoverable(ctx context.Context, _ *connect.Request[sessionsv1.ListRecoverableRequest]) (*connect.Response[sessionsv1.ListRecoverableResponse], error) {
	rows, err := h.deps.Service.ListRecoverable(ctx)
	if err != nil {
		return nil, h.classify(err, "sessions.ListRecoverable")
	}
	out := make([]*sessionsv1.RecoverableSession, 0, len(rows))
	for _, r := range rows {
		out = append(out, recoverableToProto(r))
	}
	return connect.NewResponse(&sessionsv1.ListRecoverableResponse{Sessions: out}), nil
}

func (h *connectHandler) DismissRecoverable(ctx context.Context, req *connect.Request[sessionsv1.DismissRecoverableRequest]) (*connect.Response[sessionsv1.DismissRecoverableResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := h.deps.Service.DismissRecoverable(ctx, id); err != nil {
		return nil, h.classify(err, "sessions.DismissRecoverable")
	}
	return connect.NewResponse(&sessionsv1.DismissRecoverableResponse{Id: id}), nil
}

func (h *connectHandler) Recover(ctx context.Context, req *connect.Request[sessionsv1.RecoverRequest]) (*connect.Response[sessionsv1.RecoverResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	res, err := h.deps.Service.Recover(ctx, RecoverInput{
		ID:             id,
		IdempotencyKey: req.Header().Get(idempotencyHeader),
	})
	if err != nil {
		return nil, h.classify(err, "sessions.Recover")
	}
	return connect.NewResponse(&sessionsv1.RecoverResponse{
		OldSessionId:    res.OldSessionID,
		NewSessionId:    res.NewSessionID,
		AgentType:       res.AgentType,
		CommandSent:     res.CommandSent,
		CodexHomeCopied: res.CodexHomeCopied,
	}), nil
}

func (h *connectHandler) GetPolicy(ctx context.Context, req *connect.Request[sessionsv1.GetPolicyRequest]) (*connect.Response[sessionsv1.GetPolicyResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	view, err := h.deps.Service.GetPolicy(ctx, id)
	if err != nil {
		return nil, h.classify(err, "sessions.GetPolicy")
	}
	return connect.NewResponse(&sessionsv1.GetPolicyResponse{Policy: policyViewToProto(view)}), nil
}

func (h *connectHandler) UpdatePolicy(ctx context.Context, req *connect.Request[sessionsv1.UpdatePolicyRequest]) (*connect.Response[sessionsv1.UpdatePolicyResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if req.Msg.GetPolicy() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("policy is required"))
	}
	view, err := h.deps.Service.UpdatePolicy(ctx, id, Policy{
		Mode:     req.Msg.GetPolicy().GetMode(),
		Duration: req.Msg.GetPolicy().GetDuration(),
	})
	if err != nil {
		return nil, h.classify(err, "sessions.UpdatePolicy")
	}
	return connect.NewResponse(&sessionsv1.UpdatePolicyResponse{Policy: policyViewToProto(view)}), nil
}

func (h *connectHandler) classify(err error, op string) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrFailedPrecondition):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrResourceExhausted):
		return connect.NewError(connect.CodeResourceExhausted, err)
	case errors.Is(err, ErrUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
	case errors.Is(err, ErrInternal):
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	default:
		h.deps.Logger.Printf("%s: %v", op, err)
		return connect.NewError(connect.CodeInternal, err)
	}
}

// -----------------------------------------------------------------------------
// proto helpers
// -----------------------------------------------------------------------------

func policyToProto(p Policy) *sessionsv1.ExpirationPolicy {
	return &sessionsv1.ExpirationPolicy{Mode: p.Mode, Duration: p.Duration}
}

func sessionToProto(s Session) *sessionsv1.Session {
	return &sessionsv1.Session{
		Id:               s.ID,
		Shell:            s.Shell,
		CreatedAt:        s.CreatedAt,
		Cols:             int32(s.Cols),
		Rows:             int32(s.Rows),
		Backend:          s.Backend,
		SurvivesRestart:  s.SurvivesRestart,
		Policy:           policyToProto(s.Policy),
		Busy:             s.Busy,
		Recovered:        s.Recovered,
		Origin:           originToEnum(s.Origin),
		Owner:            s.Owner,
		DisplayLabel:     s.DisplayLabel,
		TrackingDegraded: s.TrackingDegraded,
	}
}

// originToString maps the wire enum to the closed-set vocabulary the service
// layer speaks. SESSION_ORIGIN_UNSPECIFIED becomes the empty string, which
// Create normalizes to "programmatic".
func originToString(o sessionsv1.SessionOrigin) string {
	switch o {
	case sessionsv1.SessionOrigin_SESSION_ORIGIN_UI:
		return string(sessionstore.OriginUI)
	case sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC:
		return string(sessionstore.OriginProgrammatic)
	case sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE:
		return string(sessionstore.OriginRemote)
	default:
		return ""
	}
}

// originToEnum maps the stored/service origin string back to the wire enum.
func originToEnum(s string) sessionsv1.SessionOrigin {
	switch sessionstore.Origin(s) {
	case sessionstore.OriginUI:
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_UI
	case sessionstore.OriginProgrammatic:
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_PROGRAMMATIC
	case sessionstore.OriginRemote:
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_REMOTE
	default:
		return sessionsv1.SessionOrigin_SESSION_ORIGIN_UNSPECIFIED
	}
}

func sessionsToProto(in []Session) []*sessionsv1.Session {
	out := make([]*sessionsv1.Session, 0, len(in))
	for _, s := range in {
		out = append(out, sessionToProto(s))
	}
	return out
}

func recoverableToProto(r RecoverableSession) *sessionsv1.RecoverableSession {
	return &sessionsv1.RecoverableSession{
		Id:                   r.ID,
		Backend:              r.Backend,
		Shell:                r.Shell,
		Cols:                 int32(r.Cols),
		Rows:                 int32(r.Rows),
		CreatedAt:            r.CreatedAt,
		OrphanedAt:           r.OrphanedAt,
		LastActivityAt:       r.LastActivityAt,
		AgentType:            r.AgentType,
		AgentSessionId:       r.AgentSessionID,
		LaunchCommand:        r.LaunchCommand,
		Cwd:                  r.CWD,
		LastRolloutPath:      r.LastRolloutPath,
		Recoverable:          r.Recoverable,
		NotRecoverableReason: r.NotRecoverable,
		PaneName:             r.PaneName,
		HeaderColor:          r.HeaderColor,
		GroupName:            r.GroupName,
	}
}

func policyViewToProto(v PolicyView) *sessionsv1.PolicyView {
	return &sessionsv1.PolicyView{
		SessionId:  v.SessionID,
		Policy:     policyToProto(v.Policy),
		ExpiresAt:  v.ExpiresAt,
		TtlSeconds: v.TTLSeconds,
		HasExpiry:  v.HasExpiry,
	}
}
