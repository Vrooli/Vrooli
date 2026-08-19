package journal

import (
	"context"

	"connectrpc.com/connect"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/journal"
	"google.golang.org/protobuf/types/known/timestamppb"
	domain "persona/internal/journal"
)

type connectHandler struct{ service domain.Service }

func NewConnectHandler(service domain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) Append(ctx context.Context, req *connect.Request[journalv1.AppendRequest]) (*connect.Response[journalv1.AppendResponse], error) {
	entry := req.Msg.GetEntry()
	if entry == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, domain.ErrMissingVerb)
	}
	got, err := h.service.Append(ctx, domain.Entry{PersonaID: entry.GetPersonaId(), Actor: entry.GetActor(), Verb: entry.GetVerb(), RunID: entry.GetRunId(), AuthorisingHuman: entry.GetAuthorisingHuman(), Outcome: entry.GetOutcome(), Constraint: entry.GetConstraint(), Details: entry.GetDetails()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&journalv1.AppendResponse{Entry: toProto(got)}), nil
}

func (h *connectHandler) List(ctx context.Context, req *connect.Request[journalv1.ListRequest]) (*connect.Response[journalv1.ListResponse], error) {
	entries, err := h.service.List(ctx, req.Msg.GetPersonaId(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &journalv1.ListResponse{Entries: make([]*journalv1.JournalEntry, 0, len(entries))}
	for _, entry := range entries {
		out.Entries = append(out.Entries, toProto(entry))
	}
	return connect.NewResponse(out), nil
}

func toProto(e domain.Entry) *journalv1.JournalEntry {
	return &journalv1.JournalEntry{Id: e.ID, PersonaId: e.PersonaID, Actor: e.Actor, Verb: e.Verb, RunId: e.RunID, AuthorisingHuman: e.AuthorisingHuman, At: timestamppb.New(e.At), Outcome: e.Outcome, Constraint: e.Constraint, Details: e.Details}
}
