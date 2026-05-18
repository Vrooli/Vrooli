package notes

import (
	"context"
	"log"

	"development-toolchain-validator/internal/notes"

	"connectrpc.com/connect"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/notes"
)

// Deps wires the seams the Connect notes handler needs.
type Deps struct {
	Service notes.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
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
