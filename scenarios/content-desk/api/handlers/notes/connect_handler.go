package notes

import (
	"context"
	"log"

	"content-desk/internal/clock"
	"content-desk/internal/notes"

	"connectrpc.com/connect"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/notes"
)

// Deps wires the seams the Connect notes handler needs.
type Deps struct {
	Service notes.Service
	// Clock anchors the CountNotes measure's relative time-window resolution
	// (e.g. "this_week"). Explicit so tests resolve windows deterministically.
	Clock  clock.Clock
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Clock == nil {
		d.Clock = clock.System{}
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListNotes(ctx context.Context, req *connect.Request[notesv1.ListNotesRequest]) (*connect.Response[notesv1.ListNotesResponse], error) {
	results, err := h.deps.Service.List(ctx, 0)
	if err != nil {
		h.deps.Logger.Printf("notes.ListNotes: %v", err)
		return nil, notes.ToConnectError(err)
	}

	resp := &notesv1.ListNotesResponse{
		Notes: make([]*notesv1.Note, 0, len(results)),
	}
	for _, n := range results {
		resp.Notes = append(resp.Notes, domainToProto(n))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CreateNote(ctx context.Context, req *connect.Request[notesv1.CreateNoteRequest]) (*connect.Response[notesv1.CreateNoteResponse], error) {
	created, err := h.deps.Service.Create(ctx, notes.CreateInput{
		Title: req.Msg.Title,
		Body:  req.Msg.Body,
	})
	if err != nil {
		connectErr := notes.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("notes.CreateNote: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&notesv1.CreateNoteResponse{Note: domainToProto(created)}), nil
}

func (h *connectHandler) GetNote(ctx context.Context, req *connect.Request[notesv1.GetNoteRequest]) (*connect.Response[notesv1.GetNoteResponse], error) {
	got, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		connectErr := notes.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("notes.GetNote(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&notesv1.GetNoteResponse{Note: domainToProto(got)}), nil
}

// CountNotes answers the `notes count` measure: it resolves the request's
// canonical TimeWindow to a concrete [from, to) range (defaulting to this_week
// when unset) and returns the count of notes created in it. The same service
// method backs the measures-go serve registry in measures.go, so the RPC and
// the measure can never report different numbers.
func (h *connectHandler) CountNotes(ctx context.Context, req *connect.Request[notesv1.CountNotesRequest]) (*connect.Response[notesv1.CountNotesResponse], error) {
	rng, err := resolveCountWindow(req.Msg.GetWindow(), h.deps.Clock.Now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	n, err := h.deps.Service.CountInWindow(ctx, rng.From, rng.To)
	if err != nil {
		h.deps.Logger.Printf("notes.CountNotes: %v", err)
		return nil, notes.ToConnectError(err)
	}
	return connect.NewResponse(&notesv1.CountNotesResponse{Count: int64(n)}), nil
}
