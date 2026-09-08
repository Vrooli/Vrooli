package channel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/api-core/targetmodel"
	sessioncore "github.com/vrooli/vrooli/packages/session-core"
	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/session"

	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
)

func sessionBindingFromQuery(r *http.Request, nodeID, ownerID string) (session.Binding, error) {
	q := r.URL.Query()
	values := []string{q.Get("surface_id"), q.Get("transport"), q.Get("lease_id"), q.Get("policy_revision"), q.Get("lease_epoch")}
	hasBinding := false
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			hasBinding = true
			break
		}
	}
	if !hasBinding {
		return session.Binding{}, nil
	}
	epoch, err := strconv.ParseUint(strings.TrimSpace(q.Get("lease_epoch")), 10, 64)
	if err != nil || epoch == 0 {
		return session.Binding{}, errors.New("remote session lease_epoch must be a positive integer")
	}
	ref := targetmodel.SurfaceRef{
		Target:        targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: nodeID, HostNodeID: nodeID},
		OwnerScenario: strings.TrimSpace(q.Get("surface_owner")),
		SurfaceID:     strings.TrimSpace(q.Get("surface_id")),
	}
	if ref.OwnerScenario == "" {
		ref.OwnerScenario = "device-control"
	}
	return session.Binding{Surface: ref, Transport: strings.TrimSpace(q.Get("transport")), OwnerID: ownerID, LeaseID: strings.TrimSpace(q.Get("lease_id")), LeaseEpoch: epoch, PolicyRevision: strings.TrimSpace(q.Get("policy_revision"))}, nil
}

func sessionOpenProto(state session.State, shell, workingDir string) *sessionv1.Open {
	open := &sessionv1.Open{SessionId: state.ID, NodeId: state.NodeID, ReceiveWindow: state.Window, IdleTimeoutSeconds: uint32(state.Idle / time.Second), MaxLifetimeSeconds: uint32(state.MaxLifetime / time.Second), Shell: shell, WorkingDir: workingDir, MaxFrameBytes: state.MaxFrame, MaxFramesPerSecond: state.MaxRate}
	if state.Binding.Surface.SurfaceID != "" {
		open.Binding = &sessionv1.Binding{Surface: state.Binding.Surface.Proto(), Transport: state.Binding.Transport, OwnerId: state.Binding.OwnerID, LeaseId: state.Binding.LeaseID, LeaseEpoch: state.Binding.LeaseEpoch, PolicyRevision: state.Binding.PolicyRevision}
	}
	return open
}

type sessionWSHandler struct {
	manager  *session.Manager
	auth     auth.Validator
	registry registry.Service
	audit    audit.Sink
	push     func(context.Context, string, string, *sessionv1.Frame) error
}

func (h *sessionWSHandler) handle(w http.ResponseWriter, r *http.Request) {
	owner, err := auth.RequireOwner(r.Context())
	if err != nil {
		http.Error(w, "owner authentication required", http.StatusUnauthorized)
		return
	}
	// The ambient bearer token is intentionally insufficient. The caller must
	// present a second, short-lived re-authentication proof for shell access.
	// An enrolled local session is already a short-lived credential minted from
	// the operator's locally held signing key, so it is the proof itself and
	// does not need a second authenticator JWT.
	reauth := strings.TrimSpace(r.Header.Get("X-Bridge-Owner-Reauth"))
	if reauth == "" && owner.AuthMethod == auth.AuthMethodEnrolled {
		// The middleware has already verified the enrolled local session and
		// populated the owner identity from that proof.
	} else if reauth == "" || h.auth == nil {
		h.reject(w, r, owner.OwnerID, "owner re-authentication required")
		return
	} else {
		verified, err := h.auth.Validate(r.Context(), reauth)
		if err != nil || verified.OwnerID != owner.OwnerID {
			h.reject(w, r, owner.OwnerID, "owner re-authentication rejected")
			return
		}
	}

	nodeID := strings.TrimSpace(r.URL.Query().Get("node"))
	if nodeID == "" {
		http.Error(w, "node is required", http.StatusBadRequest)
		return
	}
	scopes := []string{}
	if h.registry != nil {
		node, getErr := h.registry.Get(r.Context(), nodeID)
		if getErr != nil {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}
		scopes = node.Scopes
	} else {
		scopes = strings.Split(r.URL.Query().Get("scopes"), ",")
	}
	if !scopecatalog.Resolve(owner.Scopes, session.TransportScope) {
		h.reject(w, r, owner.OwnerID, "owner authorization ceiling does not include "+session.TransportScope)
		return
	}

	id := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if id == "" {
		id = time.Now().UTC().Format("20060102T150405.000000000Z07:00")
	}
	binding, err := sessionBindingFromQuery(r, nodeID, owner.OwnerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	state, err := h.manager.Open(r.Context(), session.OpenRequest{ID: id, NodeID: nodeID, OwnerID: owner.OwnerID, Scopes: scopes, Reauth: true, Binding: binding})
	if err != nil {
		h.reject(w, r, owner.OwnerID, err.Error())
		return
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize: 32 * 1024, WriteBufferSize: 32 * 1024,
		CheckOrigin: func(req *http.Request) bool {
			return req.Header.Get("Origin") == "" || sessioncore.SameOrigin(req.Header.Get("Origin"), req.Host)
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = h.manager.Close(context.Background(), id, "websocket upgrade failed")
		return
	}
	defer conn.Close()
	nodeCloseSent := false
	sessionTerminal := false
	defer func() {
		// A browser/network disconnect is a transport detach, not a terminal
		// session close. The authenticated owner can reconnect with the same
		// session id while the node PTY remains alive and the manager replays
		// bounded scrollback. Only an explicit close, node close, kill, or
		// policy timeout makes the session terminal.
		if h.push != nil && sessionTerminal && !nodeCloseSent {
			_ = h.push(context.Background(), nodeID, id, &sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Code: "transport_closed", Reason: "browser disconnected"}}})
		}
		if sessionTerminal {
			_ = h.manager.Close(context.Background(), id, "websocket closed")
		}
	}()
	output, unsubscribe, outputErr := h.manager.SubscribeOutput(id)
	if outputErr != nil {
		h.reject(w, r, owner.OwnerID, outputErr.Error())
		return
	}
	defer unsubscribe()
	if h.push != nil {
		if err := h.push(r.Context(), nodeID, id, &sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: sessionOpenProto(state, r.URL.Query().Get("shell"), r.URL.Query().Get("working_dir"))}}); err != nil {
			_ = h.manager.Close(context.Background(), id, "node session open failed")
			h.reject(w, r, owner.OwnerID, "node session open failed")
			return
		}
	}
	var writeMu sync.Mutex
	write := func(frame *sessionv1.Frame) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return h.write(conn, frame)
	}
	done, doneErr := h.manager.Done(id)
	if doneErr == nil {
		go func() {
			<-done
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "session closed"), time.Now().Add(time.Second))
			_ = conn.Close()
		}()
	}
	if err := write(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: sessionOpenProto(state, "", "")}}); err != nil {
		return
	}
	go func() {
		for {
			select {
			case result := <-output:
				if err := write(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: result.Sequence, Data: result.Data}}}); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()
	deadline := time.Now().Add(minDuration(state.Idle, state.MaxLifetime))
	_ = conn.SetReadDeadline(deadline)
	for {
		kind, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			return
		}
		if kind != websocket.BinaryMessage {
			continue
		}
		var frame sessionv1.Frame
		if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &frame); err != nil {
			_ = write(rejectFrame("malformed_frame", err.Error()))
			continue
		}
		switch p := frame.Payload.(type) {
		case *sessionv1.Frame_Data:
			result, dataErr := h.manager.AcceptData(r.Context(), id, p.Data.Sequence, p.Data.Data)
			if dataErr != nil {
				_ = write(rejectFrame("data_rejected", dataErr.Error()))
				continue
			}
			h.recordBytes(r.Context(), state, audit.ActionSessionDataIn, result.Data)
			if h.push != nil {
				if pushErr := h.push(r.Context(), nodeID, id, &sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: result.Sequence, Data: result.Data}}}); pushErr != nil {
					_ = write(rejectFrame("node_unavailable", pushErr.Error()))
					continue
				}
			}
			if err := write(&sessionv1.Frame{Payload: &sessionv1.Frame_Ack{Ack: &sessionv1.Ack{Accepted: true, Sequence: result.Sequence, WindowAvailable: state.Window, CommandId: p.Data.GetCommandId()}}}); err != nil {
				return
			}
			if h.push == nil {
				// Focused protocol tests use the echo seam when no node transport is wired.
				h.recordBytes(r.Context(), state, audit.ActionSessionDataOut, result.Data)
				if err := write(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: result.Sequence, Data: result.Data}}}); err != nil {
					return
				}
			}
			if err := h.manager.Acknowledge(id, 1); err != nil {
				return
			}
		case *sessionv1.Frame_Resize:
			if resizeErr := h.manager.Resize(r.Context(), id, session.ResizeRequest{Columns: p.Resize.Columns, Rows: p.Resize.Rows}); resizeErr != nil {
				_ = h.write(conn, rejectFrame("resize_rejected", resizeErr.Error()))
				continue
			}
			if h.push != nil {
				if pushErr := h.push(r.Context(), nodeID, id, &sessionv1.Frame{Payload: &sessionv1.Frame_Resize{Resize: p.Resize}}); pushErr != nil {
					_ = write(rejectFrame("node_unavailable", pushErr.Error()))
					continue
				}
			}
			if err := write(&sessionv1.Frame{Payload: &sessionv1.Frame_Ack{Ack: &sessionv1.Ack{Accepted: true, WindowAvailable: state.Window}}}); err != nil {
				return
			}
		case *sessionv1.Frame_Close:
			sessionTerminal = true
			if h.push != nil {
				_ = h.push(r.Context(), nodeID, id, &sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: p.Close}})
				nodeCloseSent = true
			}
			_ = h.manager.Close(r.Context(), id, p.Close.Reason)
			return
		case *sessionv1.Frame_Evidence:
			if evidenceErr := recordSessionEvidence(r.Context(), h.audit, state, p.Evidence); evidenceErr != nil {
				_ = write(rejectFrame("evidence_rejected", evidenceErr.Error()))
			}
		case *sessionv1.Frame_Revoke:
			if p.Revoke == nil || p.Revoke.GetLeaseId() == "" || p.Revoke.GetLeaseEpoch() == 0 {
				_ = write(rejectFrame("revoke_rejected", "lease identity is required"))
				continue
			}
			if revokeErr := h.manager.Revoke(r.Context(), id, p.Revoke.GetLeaseId(), p.Revoke.GetLeaseEpoch(), p.Revoke.GetReason()); revokeErr != nil {
				_ = write(rejectFrame("revoke_rejected", revokeErr.Error()))
				continue
			}
			sessionTerminal = true
			return
		default:
			_ = write(rejectFrame("unsupported_frame", "open is server-owned"))
		}
		if current, getErr := h.manager.Get(id); getErr == nil {
			deadline = time.Now().Add(minDuration(current.Idle, current.MaxLifetime))
			_ = conn.SetReadDeadline(deadline)
		}
	}
}

// kill closes a live session by id. Owner authentication is required before
// looking up the session so the endpoint cannot be used as an oracle.
func (h *sessionWSHandler) kill(w http.ResponseWriter, r *http.Request) {
	owner, err := auth.RequireOwner(r.Context())
	if err != nil {
		http.Error(w, "owner authentication required", http.StatusUnauthorized)
		return
	}
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		http.Error(w, "session id is required", http.StatusBadRequest)
		return
	}
	state, getErr := h.manager.Get(id)
	if getErr != nil || state.OwnerID != owner.OwnerID {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if err := h.manager.Kill(r.Context(), id); err != nil {
		http.Error(w, "session could not be terminated", http.StatusConflict)
		return
	}
	if h.push != nil {
		_ = h.push(r.Context(), state.NodeID, id, &sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Code: "revoked", Reason: "owner terminated session"}}})
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *sessionWSHandler) recordBytes(ctx context.Context, state session.State, action audit.Action, data []byte) {
	if h.audit == nil || len(data) == 0 {
		return
	}
	const maxAuditBytes = 64 * 1024
	originalSize := len(data)
	truncated := len(data) > maxAuditBytes
	digest := sha256.Sum256(data)
	detail := hex.EncodeToString(digest[:])
	if truncated {
		detail += ":truncated"
	}
	_, _ = h.audit.Append(ctx, audit.Record{Action: action, Outcome: audit.OutcomeCompleted, Actor: state.OwnerID, NodeID: state.NodeID, RunID: state.ID, Detail: fmt.Sprintf("%s:sha256=%s:bytes=%d", direction(action), detail, originalSize)})
}

func recordSessionEvidence(ctx context.Context, sink audit.Sink, state session.State, evidence *sessionv1.Evidence) error {
	if evidence == nil || strings.TrimSpace(evidence.GetEvidenceId()) == "" || strings.TrimSpace(evidence.GetKind()) == "" || strings.TrimSpace(evidence.GetDigest()) == "" {
		return errors.New("evidence requires evidence_id, kind, and digest")
	}
	if len(evidence.GetEvidenceId()) > 255 || len(evidence.GetKind()) > 128 || len(evidence.GetDigest()) > 128 || len(evidence.GetCommandId()) > 255 || len(evidence.GetRemoteCommandId()) > 255 {
		return errors.New("evidence metadata exceeds its bound")
	}
	if sink == nil {
		return nil
	}
	detail := fmt.Sprintf("evidence kind=%s evidence_id=%s digest=%s command_id=%s remote_command_id=%s", evidence.GetKind(), evidence.GetEvidenceId(), evidence.GetDigest(), evidence.GetCommandId(), evidence.GetRemoteCommandId())
	_, err := sink.Append(ctx, audit.Record{Action: audit.ActionSessionDataOut, Outcome: audit.OutcomeCompleted, Actor: "node:" + state.NodeID, NodeID: state.NodeID, RunID: state.ID, Detail: detail})
	return err
}

func direction(action audit.Action) string {
	if action == audit.ActionSessionDataOut {
		return "out"
	}
	return "in"
}

func (h *sessionWSHandler) reject(w http.ResponseWriter, r *http.Request, owner, reason string) {
	if h.audit != nil {
		_, _ = h.audit.Append(r.Context(), audit.Record{Action: audit.ActionSessionOpen, Outcome: audit.OutcomeRejected, Actor: owner, NodeID: r.URL.Query().Get("node"), Detail: reason})
	}
	http.Error(w, reason, http.StatusForbidden)
}

func (h *sessionWSHandler) write(conn *websocket.Conn, frame *sessionv1.Frame) error {
	b, err := proto.Marshal(frame)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, b)
}

func rejectFrame(code, reason string) *sessionv1.Frame {
	return &sessionv1.Frame{Payload: &sessionv1.Frame_Ack{Ack: &sessionv1.Ack{Code: code, Reason: reason}}}
}

func minDuration(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}
