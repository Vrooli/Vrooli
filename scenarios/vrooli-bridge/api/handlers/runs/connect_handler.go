package runs

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/runs"

	"connectrpc.com/connect"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

// Deps wires the seams the Connect runs handler needs. Verifier enforces
// per-node mutual auth on the node-facing ReportRunEvent (nil disables it, the
// pre-pairing stub). The operator verbs are owner-gated via auth.RequireOwner.
type Deps struct {
	Service  runs.Service
	Verifier *nodeauth.Verifier
	Logger   *log.Logger
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

func (h *connectHandler) GetRun(ctx context.Context, req *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	run, events, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.mapErr("GetRun", req.Msg.Id, err)
	}
	resp := &runsv1.GetRunResponse{
		Run:    domainRunToProto(run),
		Events: make([]*sharedv1.RunEvent, 0, len(events)),
	}
	for _, ev := range events {
		resp.Events = append(resp.Events, domainEventToProto(ev))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListRuns(ctx context.Context, req *connect.Request[runsv1.ListRunsRequest]) (*connect.Response[runsv1.ListRunsResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	list, err := h.deps.Service.List(ctx, runs.ListFilter{
		NodeID: req.Msg.NodeId,
		Limit:  int(req.Msg.Limit),
	})
	if err != nil {
		h.deps.Logger.Printf("runs.ListRuns: %v", err)
		return nil, runs.ToConnectError(err)
	}
	resp := &runsv1.ListRunsResponse{Runs: make([]*runsv1.Run, 0, len(list))}
	for _, r := range list {
		resp.Runs = append(resp.Runs, domainRunToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) WaitRun(ctx context.Context, req *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	timeout := time.Duration(req.Msg.TimeoutSeconds) * time.Second
	run, timedOut, err := h.deps.Service.Wait(ctx, req.Msg.Id, timeout)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, connect.NewError(connect.CodeCanceled, err)
		}
		return nil, h.mapErr("WaitRun", req.Msg.Id, err)
	}
	return connect.NewResponse(&runsv1.WaitRunResponse{
		Run:      domainRunToProto(run),
		TimedOut: timedOut,
	}), nil
}

func (h *connectHandler) AbortRun(ctx context.Context, req *connect.Request[runsv1.AbortRunRequest]) (*connect.Response[runsv1.AbortRunResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	run, err := h.deps.Service.Abort(ctx, req.Msg.Id, req.Msg.Reason)
	if err != nil {
		return nil, h.mapErr("AbortRun", req.Msg.Id, err)
	}
	return connect.NewResponse(&runsv1.AbortRunResponse{Run: domainRunToProto(run)}), nil
}

// StreamRunEvents subscribes to the run's live events first, replays the
// persisted history (de-duplicated by sequence), then tails live events until
// the run is terminal or the client disconnects. This is the human "follow"
// verb; agents use WaitRun (block-once). Owner-gated.
func (h *connectHandler) StreamRunEvents(ctx context.Context, req *connect.Request[runsv1.StreamRunEventsRequest], stream *connect.ServerStream[runsv1.RunEventMessage]) error {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return auth.ToConnectError(err)
	}
	id := req.Msg.Id

	// Subscribe BEFORE reading the persisted history so no event is lost in the
	// gap between the read and the subscription.
	live, cancel := h.deps.Service.Subscribe(id)
	defer cancel()

	run, events, err := h.deps.Service.Get(ctx, id)
	if err != nil {
		return h.mapErr("StreamRunEvents", id, err)
	}

	var maxSeq uint64
	for _, ev := range events {
		if ev.Sequence > maxSeq {
			maxSeq = ev.Sequence
		}
		if err := stream.Send(&runsv1.RunEventMessage{Event: domainEventToProto(ev)}); err != nil {
			return err
		}
	}
	if run.Status.Terminal() {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-live:
			if !ok {
				return nil
			}
			if ev.Sequence != 0 && ev.Sequence <= maxSeq {
				continue // already replayed from the persisted history
			}
			if ev.Sequence > maxSeq {
				maxSeq = ev.Sequence
			}
			if err := stream.Send(&runsv1.RunEventMessage{Event: domainEventToProto(ev)}); err != nil {
				return err
			}
			if isTerminalEvent(ev) {
				return nil
			}
		}
	}
}

// isTerminalEvent reports whether an event ends the run from the stream's view:
// a terminal EXIT, or an abort status synthesised by Abort.
func isTerminalEvent(ev runs.RunEvent) bool {
	if ev.Kind == runs.EventExit {
		return true
	}
	return ev.Kind == runs.EventStatus && strings.HasPrefix(strings.ToLower(ev.Status), "aborted")
}

// ReportRunEvent ingests one RunEvent streamed back from the node-agent. It is
// NODE-facing: the agent signs the call with its per-node Ed25519 credential,
// and the node may only report against its OWN runs (a cross-node forge is
// rejected). A terminal EXIT flips the run to its terminal status and wakes
// block-once waiters.
func (h *connectHandler) ReportRunEvent(ctx context.Context, req *connect.Request[runsv1.ReportRunEventRequest]) (*connect.Response[runsv1.ReportRunEventResponse], error) {
	ev := req.Msg.GetEvent()
	if ev == nil || strings.TrimSpace(ev.GetRunId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a run event with a run_id is required"))
	}

	var nodeID string
	if h.deps.Verifier != nil {
		proof, err := nodeauth.ParseHeaders(
			req.Header().Get(nodeauth.HeaderNode),
			req.Header().Get(nodeauth.HeaderTS),
			req.Header().Get(nodeauth.HeaderSig),
		)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err := h.deps.Verifier.VerifyProof(ctx, proof); err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		nodeID = proof.NodeID

		// A node may only report against its own runs. Look the run up and reject
		// a cross-node report. An unknown run is acknowledged (accepted=false) so
		// a confused node stops re-sending.
		run, _, err := h.deps.Service.Get(ctx, ev.GetRunId())
		if err != nil {
			var notFound runs.ErrRunNotFound
			if errors.As(err, &notFound) {
				return connect.NewResponse(&runsv1.ReportRunEventResponse{Accepted: false}), nil
			}
			h.deps.Logger.Printf("runs.ReportRunEvent: lookup run %q: %v", ev.GetRunId(), err)
			return nil, runs.ToConnectError(err)
		}
		if run.NodeID != nodeID {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("a node may only report events for its own runs"))
		}
	}

	accepted, err := h.deps.Service.AppendEvent(ctx, protoEventToDomain(ev))
	if err != nil {
		var invalid runs.ErrInvalidRun
		if errors.As(err, &invalid) {
			return nil, connect.NewError(connect.CodeInvalidArgument, invalid)
		}
		h.deps.Logger.Printf("runs.ReportRunEvent(run %q): %v", ev.GetRunId(), err)
		return nil, runs.ToConnectError(err)
	}
	return connect.NewResponse(&runsv1.ReportRunEventResponse{Accepted: accepted}), nil
}

// mapErr logs internal errors and returns the Connect translation.
func (h *connectHandler) mapErr(op, id string, err error) error {
	connectErr := runs.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("runs.%s(%q): %v", op, id, err)
	}
	return connectErr
}
