package sessions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"google.golang.org/protobuf/proto"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/sessionstore"
)

// Deps wires the seams the Connect sessions handler needs. The human-owner
// delegation mint is no longer invoked here: it is owned by the Adapter, which
// runs it through a create-scoped pre-spawn hook so the delegation is stored
// BEFORE the session's PTY environment is built. The handler only forwards the
// operator's candidate credentials down via CreateInput.HumanOwnerCredentials.
type Deps struct {
	Service          Service
	Logger           *log.Logger
	VerifyOwnSession func(context.Context) (string, error)
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
	if d.VerifyOwnSession == nil {
		d.VerifyOwnSession = provenance.VerifyOwnWebConsoleSession
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

	// ErrRemoteUnavailable means the remote target or federation transport
	// cannot satisfy the requested operation. It is mapped to an actionable
	// failed-precondition/unavailable Connect error instead of a false create.
	ErrRemoteUnavailable   = errors.New("remote federation unavailable")
	ErrTargetUnavailable   = errors.New("remote target unavailable")
	ErrTargetNotFound      = errors.New("remote target not found")
	ErrIdempotencyConflict = errors.New("idempotency key reused for a different session create")
)

const idempotencyHeader = "X-Idempotency-Key"

// humanOwnerSubjectHeader carries the verified human a trusted service caller
// (agent-manager) is acting for when it opens or lists sessions on the
// operator's behalf (a tunnel-UI spawn). It is a creator-identity LABEL only:
// web-console attributes the session's stable-create owner to it but derives NO
// authority from it — the caller's own verified identity still gates the call.
// agent-manager sets it ONLY from a verified-independent-human spawn context.
// Must match agent-manager's webconsole.HumanOwnerSubjectHeader.
const humanOwnerSubjectHeader = "X-Human-Owner-Subject"

func humanOwnerCredentialCandidates(headers http.Header) []string {
	var credentials []string
	if authz := strings.TrimSpace(headers.Get("Authorization")); authz != "" {
		credentials = append(credentials, authz)
	}
	if cfAccess := strings.TrimSpace(headers.Get("Cf-Access-Jwt-Assertion")); cfAccess != "" {
		credentials = append(credentials, cfAccess)
	}
	return credentials
}

func (h *connectHandler) Create(ctx context.Context, req *connect.Request[sessionsv1.CreateRequest]) (*connect.Response[sessionsv1.CreateResponse], error) {
	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	digest := sha256.Sum256(raw)
	in := CreateInput{
		RequestedSessionID:    req.Msg.GetRequestedSessionId(),
		RequestFingerprint:    hex.EncodeToString(digest[:]),
		Shell:                 req.Msg.GetShell(),
		Cols:                  int(req.Msg.GetCols()),
		Rows:                  int(req.Msg.GetRows()),
		Backend:               req.Msg.GetBackend(),
		LaunchCommand:         req.Msg.GetLaunchCommand(),
		ExecuteLaunchCommand:  req.Msg.GetExecuteLaunchCommand(),
		AgentType:             req.Msg.GetAgentType(),
		Origin:                originToString(req.Msg.GetOrigin()),
		Owner:                 req.Msg.GetOwner(),
		DisplayLabel:          req.Msg.GetDisplayLabel(),
		TargetID:              req.Msg.GetTargetId(),
		WorkingDir:            req.Msg.GetWorkingDir(),
		TmuxMouseMode:         req.Msg.GetTmuxMouseMode(),
		LaunchMode:            req.Msg.GetLaunchMode(),
		LaunchDescriptorJSON:  req.Msg.GetLaunchDescriptorJson(),
		IdempotencyKey:        req.Header().Get(idempotencyHeader),
		HumanOwnerSubject:     strings.TrimSpace(req.Header().Get(humanOwnerSubjectHeader)),
		AgentRunIdentityToken: req.Msg.GetAgentRunIdentityToken(),
	}
	// Carry the operator's candidate human credentials down into the create flow
	// so a session-scoped human-owner delegation can be minted BEFORE the PTY
	// environment is built (see Adapter.Create). The raw credentials authenticate
	// the mint at scenario-authenticator only; they are never persisted or placed
	// in the agent env. We pass both the Authorization bearer and the Cf-Access
	// assertion because for the tunnel operator the Cf-Access one is the human
	// proof while the bearer may be a non-human service token; the authenticator
	// refuses any non-human credential, so trying both is safe.
	in.HumanOwnerCredentials = humanOwnerCredentialCandidates(req.Header())
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

func (h *connectHandler) List(ctx context.Context, req *connect.Request[sessionsv1.ListRequest]) (*connect.Response[sessionsv1.ListResponse], error) {
	out, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, h.classify(err, "sessions.List")
	}
	rs := h.deps.Service.RecoveryStatus(ctx)
	stable := false
	creatorOwner := ""
	if principal, ok := identity.PrincipalFromContext(ctx); ok {
		creatorOwner = principal.OwnerSubject()
		// A trusted, VERIFIED service caller acting for a verified human (a
		// tunnel-UI spawn) carries that human on the header; the stable-create
		// owner is then the human — the operator the spawned session is attributed
		// to — not the service caller. This only RELABELS a request the caller's
		// own verified identity already authorized (creatorOwner was non-empty): it
		// is a creator-identity LABEL, never authority. It is deliberately inside
		// the verified-principal branch so an anonymous caller that sets the header
		// gets nothing — the header can never conjure identity. Empty header leaves
		// the caller-derived owner unchanged (the prior non-human-spawn behaviour).
		if creatorOwner != "" {
			if human := strings.TrimSpace(req.Header().Get(humanOwnerSubjectHeader)); human != "" {
				creatorOwner = human
			}
		}
	}
	if owner, ok := h.deps.Service.(interface{ SupportsStableCreateIdentity() bool }); ok {
		stable = owner.SupportsStableCreateIdentity()
	}
	return connect.NewResponse(&sessionsv1.ListResponse{
		Sessions:                     sessionsToProto(out),
		SupportsStableCreateIdentity: stable,
		StableCreateOwner:            creatorOwner,
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

func (h *connectHandler) ListArchived(ctx context.Context, _ *connect.Request[sessionsv1.ListArchivedRequest]) (*connect.Response[sessionsv1.ListArchivedResponse], error) {
	rows, err := h.deps.Service.ListArchived(ctx)
	if err != nil {
		return nil, h.classify(err, "sessions.ListArchived")
	}
	out := make([]*sessionsv1.ArchivedSession, 0, len(rows))
	for _, row := range rows {
		out = append(out, archivedToProto(row))
	}
	return connect.NewResponse(&sessionsv1.ListArchivedResponse{Sessions: out, Total: int32(len(out))}), nil
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
	if got, want := strings.TrimSpace(req.Msg.GetConfirmation()), "DELETE:"+id; got != want {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("permanent deletion requires confirmation %q", want))
	}
	ctx = WithOperationID(ctx, req.Header().Get(idempotencyHeader))
	if err := h.deps.Service.Delete(ctx, id); err != nil {
		return nil, h.classify(err, "sessions.Delete")
	}
	return connect.NewResponse(&sessionsv1.DeleteResponse{}), nil
}

func (h *connectHandler) Archive(ctx context.Context, req *connect.Request[sessionsv1.ArchiveRequest]) (*connect.Response[sessionsv1.ArchiveResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	ctx = WithOperationID(ctx, req.Header().Get(idempotencyHeader))
	if err := h.deps.Service.Archive(ctx, id); err != nil {
		return nil, h.classify(err, "sessions.Archive")
	}
	return connect.NewResponse(&sessionsv1.ArchiveResponse{Id: id}), nil
}

func (h *connectHandler) Unarchive(ctx context.Context, req *connect.Request[sessionsv1.UnarchiveRequest]) (*connect.Response[sessionsv1.UnarchiveResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	ctx = WithOperationID(ctx, req.Header().Get(idempotencyHeader))
	if err := h.deps.Service.Unarchive(ctx, id); err != nil {
		return nil, h.classify(err, "sessions.Unarchive")
	}
	return connect.NewResponse(&sessionsv1.UnarchiveResponse{Id: id}), nil
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
	ctx = WithOperationID(ctx, req.Header().Get(idempotencyHeader))
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
	ctx = WithOperationID(ctx, req.Header().Get(idempotencyHeader))
	res, err := h.deps.Service.Recover(ctx, RecoverInput{
		ID:                    id,
		IdempotencyKey:        req.Header().Get(idempotencyHeader),
		HumanOwnerCredentials: humanOwnerCredentialCandidates(req.Header()),
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

func (h *connectHandler) Reopen(ctx context.Context, req *connect.Request[sessionsv1.ReopenRequest]) (*connect.Response[sessionsv1.ReopenResponse], error) {
	id := strings.TrimSpace(req.Msg.GetId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	ctx = WithOperationID(ctx, req.Header().Get(idempotencyHeader))
	res, err := h.deps.Service.Recover(ctx, RecoverInput{
		ID:                    id,
		IdempotencyKey:        req.Header().Get(idempotencyHeader),
		HumanOwnerCredentials: humanOwnerCredentialCandidates(req.Header()),
	})
	if err != nil {
		return nil, h.classify(err, "sessions.Reopen")
	}
	return connect.NewResponse(&sessionsv1.ReopenResponse{
		OldSessionId:    res.OldSessionID,
		NewSessionId:    res.NewSessionID,
		AgentType:       res.AgentType,
		CommandSent:     res.CommandSent,
		CodexHomeCopied: res.CodexHomeCopied,
	}), nil
}

func (h *connectHandler) GetArchiveRetention(ctx context.Context, _ *connect.Request[sessionsv1.GetArchiveRetentionRequest]) (*connect.Response[sessionsv1.GetArchiveRetentionResponse], error) {
	snapshot, err := h.deps.Service.GetArchiveRetention(ctx)
	if err != nil {
		return nil, h.classify(err, "sessions.GetArchiveRetention")
	}
	return connect.NewResponse(&sessionsv1.GetArchiveRetentionResponse{
		Policy: &sessionsv1.ArchiveRetentionPolicy{
			MessageLessAgeDays: int32(snapshot.Policy.MessageLessAge / (24 * time.Hour)),
			AgentHomeAgeDays:   int32(snapshot.Policy.AgentHomeAge / (24 * time.Hour)),
			MaxBytes:           snapshot.Policy.MaxBytes,
		},
		Stats: archiveRetentionStatsToProto(snapshot.Stats),
	}), nil
}

func (h *connectHandler) PruneArchive(ctx context.Context, req *connect.Request[sessionsv1.PruneArchiveRequest]) (*connect.Response[sessionsv1.PruneArchiveResponse], error) {
	result, err := h.deps.Service.PruneArchive(ctx, req.Msg.GetApply())
	if err != nil {
		return nil, h.classify(err, "sessions.PruneArchive")
	}
	actions := make([]*sessionsv1.ArchivePruneAction, 0, len(result.Actions))
	for _, action := range result.Actions {
		actions = append(actions, &sessionsv1.ArchivePruneAction{
			SessionId: action.SessionID,
			Kind:      string(action.Kind),
			Bytes:     action.Bytes,
			Applied:   action.Applied,
		})
	}
	return connect.NewResponse(&sessionsv1.PruneArchiveResponse{
		DryRun:         result.DryRun,
		Actions:        actions,
		ReclaimedBytes: result.ReclaimedBytes,
		Before:         archiveRetentionStatsToProto(result.Before),
		After:          archiveRetentionStatsToProto(result.After),
	}), nil
}

func archiveRetentionStatsToProto(stats ArchiveRetentionStats) *sessionsv1.ArchiveRetentionStats {
	return &sessionsv1.ArchiveRetentionStats{
		EntryCount:      stats.EntryCount,
		MessageCount:    stats.MessageCount,
		TranscriptBytes: stats.TranscriptBytes,
		AgentHomeBytes:  stats.AgentHomeBytes,
		TotalBytes:      stats.TotalBytes,
	}
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
	case errors.Is(err, ErrTargetUnavailable):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, ErrTargetNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrIdempotencyConflict):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, ErrRemoteUnavailable):
		return connect.NewError(connect.CodeUnavailable, err)
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
		Id:                      s.ID,
		Shell:                   s.Shell,
		CreatedAt:               s.CreatedAt,
		Cols:                    int32(s.Cols),
		Rows:                    int32(s.Rows),
		Backend:                 s.Backend,
		SurvivesRestart:         s.SurvivesRestart,
		Policy:                  policyToProto(s.Policy),
		Recovered:               s.Recovered,
		Origin:                  originToEnum(s.Origin),
		Owner:                   s.Owner,
		DisplayLabel:            s.DisplayLabel,
		CreateFingerprint:       s.CreateFingerprint,
		TrackingDegraded:        s.TrackingDegraded,
		Target:                  s.Target,
		Activity:                s.Activity,
		LaunchMode:              s.LaunchMode,
		ControlMode:             s.ControlMode,
		NativeOwner:             s.NativeOwner,
		NativeTransport:         s.NativeTransport,
		ProviderVersion:         s.ProviderVersion,
		NativeThreadId:          s.NativeThreadID,
		LastVerifiedTurnId:      s.LastVerifiedTurnID,
		ForkedFromNativeSession: s.ForkedFromNativeSession,
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

func archivedToProto(row ArchivedSession) *sessionsv1.ArchivedSession {
	state := sessionsv1.ArchiveRestoreState_ARCHIVE_RESTORE_STATE_UNSPECIFIED
	switch row.RestoreState {
	case RestoreStateReopenable:
		state = sessionsv1.ArchiveRestoreState_ARCHIVE_RESTORE_STATE_REOPENABLE
	case RestoreStateReadOnly:
		state = sessionsv1.ArchiveRestoreState_ARCHIVE_RESTORE_STATE_READ_ONLY
	case RestoreStateNothingToRestore:
		state = sessionsv1.ArchiveRestoreState_ARCHIVE_RESTORE_STATE_NOTHING_TO_RESTORE
	}
	return &sessionsv1.ArchivedSession{
		Id:                 row.ID,
		ArchivedAt:         row.ArchivedAt,
		CreatedAt:          row.CreatedAt,
		AgentType:          row.AgentType,
		AgentSessionId:     row.AgentSessionID,
		Cwd:                row.CWD,
		PaneName:           row.PaneName,
		HeaderColor:        row.HeaderColor,
		GroupName:          row.GroupName,
		MessageCount:       row.MessageCount,
		RestoreState:       state,
		RestoreStateReason: row.RestoreStateReason,
		AwaitingRecovery:   row.AwaitingRecovery,
	}
}

func recoverableToProto(r RecoverableSession) *sessionsv1.RecoverableSession {
	return &sessionsv1.RecoverableSession{
		Id:                      r.ID,
		Backend:                 r.Backend,
		Shell:                   r.Shell,
		Cols:                    int32(r.Cols),
		Rows:                    int32(r.Rows),
		CreatedAt:               r.CreatedAt,
		OrphanedAt:              r.OrphanedAt,
		LastActivityAt:          r.LastActivityAt,
		AgentType:               r.AgentType,
		AgentSessionId:          r.AgentSessionID,
		LaunchCommand:           r.LaunchCommand,
		Cwd:                     r.CWD,
		LastRolloutPath:         r.LastRolloutPath,
		Recoverable:             r.Recoverable,
		NotRecoverableReason:    r.NotRecoverable,
		PaneName:                r.PaneName,
		HeaderColor:             r.HeaderColor,
		GroupName:               r.GroupName,
		LaunchMode:              r.LaunchMode,
		ControlMode:             r.ControlMode,
		NativeOwner:             r.NativeOwner,
		NativeTransport:         r.NativeTransport,
		ProviderVersion:         r.ProviderVersion,
		NativeThreadId:          r.NativeThreadID,
		LastVerifiedTurnId:      r.LastVerifiedTurnID,
		ForkedFromNativeSession: r.ForkedFromNativeSession,
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

func (h *connectHandler) ReadNativeUsage(ctx context.Context, req *connect.Request[sessionsv1.ReadNativeUsageRequest]) (*connect.Response[sessionsv1.ReadNativeUsageResponse], error) {
	if err := h.authorizeNativeRead(ctx, req.Msg.GetSessionId()); err != nil {
		return nil, err
	}
	reader, ok := h.deps.Service.(interface {
		ReadNativeUsage(context.Context, *sessionsv1.ReadNativeUsageRequest) (*sessionsv1.ReadNativeUsageResponse, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
	}
	result, err := reader.ReadNativeUsage(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}

func (h *connectHandler) authorizeNativeRead(ctx context.Context, requestedSession string) error {
	if _, ok := identity.PrincipalFromContext(ctx); !ok {
		return connect.NewError(connect.CodeUnauthenticated, errors.New("verified identity required"))
	}
	if _, err := authn.RequireCapability(ctx, "web-console:read"); err != nil {
		principal, _ := identity.PrincipalFromContext(ctx)
		p := provenance.FromContext(ctx)
		sessionID, verifyErr := "", error(nil)
		if principal.Kind == identity.ActorAgent && p.IsVerifiedAgent() && h.deps.VerifyOwnSession != nil {
			sessionID, verifyErr = h.deps.VerifyOwnSession(ctx)
		}
		if verifyErr != nil || sessionID == "" || sessionID != requestedSession {
			return connect.NewError(connect.CodePermissionDenied, errors.New("read capability or verified own session required"))
		}
	}
	return nil
}

func (h *connectHandler) ReadNativeThreadState(ctx context.Context, req *connect.Request[sessionsv1.ReadNativeThreadStateRequest]) (*connect.Response[sessionsv1.ReadNativeThreadStateResponse], error) {
	if err := h.authorizeNativeRead(ctx, req.Msg.GetSessionId()); err != nil {
		return nil, err
	}
	reader, ok := h.deps.Service.(interface {
		ReadNativeThreadState(context.Context, *sessionsv1.ReadNativeThreadStateRequest) (*sessionsv1.ReadNativeThreadStateResponse, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, ErrUnavailable)
	}
	result, err := reader.ReadNativeThreadState(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(result), nil
}
