package focus

import (
	"context"
	"log"

	"connectrpc.com/connect"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus"
	"personal-planner/internal/focus"
)

type (
	Deps struct {
		Service focus.Service
		Logger  *log.Logger
	}
	connectHandler struct{ deps Deps }
)

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) GetCurrentSession(ctx context.Context, _ *connect.Request[focusv1.GetCurrentSessionRequest]) (*connect.Response[focusv1.GetCurrentSessionResponse], error) {
	session, ok, err := h.deps.Service.Current(ctx)
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	return connect.NewResponse(&focusv1.GetCurrentSessionResponse{HasSession: ok, Session: optionalSession(ok, session)}), nil
}

func (h *connectHandler) StartFocus(ctx context.Context, req *connect.Request[focusv1.StartFocusRequest]) (*connect.Response[focusv1.FocusSessionResponse], error) {
	session, err := h.deps.Service.Start(ctx, focus.StartInput{WorkItemID: req.Msg.WorkItemId, Title: req.Msg.Title, Mode: req.Msg.Mode})
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	return connect.NewResponse(&focusv1.FocusSessionResponse{Session: domainToProto(session)}), nil
}

func (h *connectHandler) PauseFocus(ctx context.Context, req *connect.Request[focusv1.TransitionFocusRequest]) (*connect.Response[focusv1.FocusSessionResponse], error) {
	return h.transition(ctx, req.Msg.SessionId, req.Msg.ExpectedRevision, false)
}

func (h *connectHandler) ResumeFocus(ctx context.Context, req *connect.Request[focusv1.TransitionFocusRequest]) (*connect.Response[focusv1.FocusSessionResponse], error) {
	return h.transition(ctx, req.Msg.SessionId, req.Msg.ExpectedRevision, true)
}

func (h *connectHandler) transition(ctx context.Context, id string, revision int64, resume bool) (*connect.Response[focusv1.FocusSessionResponse], error) {
	var session focus.Session
	var err error
	if resume {
		session, err = h.deps.Service.Resume(ctx, id, revision)
	} else {
		session, err = h.deps.Service.Pause(ctx, id, revision)
	}
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	return connect.NewResponse(&focusv1.FocusSessionResponse{Session: domainToProto(session)}), nil
}

func (h *connectHandler) EndFocus(ctx context.Context, req *connect.Request[focusv1.EndFocusRequest]) (*connect.Response[focusv1.FocusSessionResponse], error) {
	session, err := h.deps.Service.End(ctx, req.Msg.SessionId, req.Msg.ExpectedRevision)
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	return connect.NewResponse(&focusv1.FocusSessionResponse{Session: domainToProto(session)}), nil
}

func (h *connectHandler) RecordManualActual(ctx context.Context, req *connect.Request[focusv1.RecordManualActualRequest]) (*connect.Response[focusv1.ActualResponse], error) {
	actual, err := h.deps.Service.RecordActual(ctx, focus.RecordActualInput{WorkItemID: req.Msg.WorkItemId, Title: req.Msg.Title, LocalDate: req.Msg.LocalDate, ReportedMinutes: req.Msg.ReportedMinutes, Certainty: req.Msg.Certainty, Note: req.Msg.Note})
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	return connect.NewResponse(&focusv1.ActualResponse{Actual: actualToProto(actual)}), nil
}

func (h *connectHandler) ListActuals(ctx context.Context, req *connect.Request[focusv1.ListActualsRequest]) (*connect.Response[focusv1.ListActualsResponse], error) {
	actuals, err := h.deps.Service.ListActuals(ctx, req.Msg.LocalDate)
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	response := &focusv1.ListActualsResponse{}
	for _, actual := range actuals {
		response.Actuals = append(response.Actuals, actualToProto(actual))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) ListActualCorrections(ctx context.Context, req *connect.Request[focusv1.ListActualCorrectionsRequest]) (*connect.Response[focusv1.ListActualCorrectionsResponse], error) {
	corrections, err := h.deps.Service.ListCorrections(ctx, req.Msg.ActualId, req.Msg.LocalDate, int(req.Msg.Limit))
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	response := &focusv1.ListActualCorrectionsResponse{}
	for _, correction := range corrections {
		response.Corrections = append(response.Corrections, correctionToProto(correction))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) CorrectActual(ctx context.Context, req *connect.Request[focusv1.CorrectActualRequest]) (*connect.Response[focusv1.ActualResponse], error) {
	actual, err := h.deps.Service.CorrectActual(ctx, focus.CorrectActualInput{ID: req.Msg.Id, ExpectedRevision: req.Msg.ExpectedRevision, ReportedMinutes: req.Msg.ReportedMinutes, Certainty: req.Msg.Certainty, Note: req.Msg.Note})
	if err != nil {
		return nil, focus.ToConnectError(err)
	}
	return connect.NewResponse(&focusv1.ActualResponse{Actual: actualToProto(actual)}), nil
}

func correctionToProto(correction focus.Correction) *focusv1.ActualCorrection {
	return &focusv1.ActualCorrection{Id: correction.ID, ActualId: correction.ActualID, PreviousMinutes: correction.PreviousMinutes, NewMinutes: correction.NewMinutes, PreviousCertainty: correction.PreviousCertainty, NewCertainty: correction.NewCertainty, Reason: correction.Reason, CreatedAtUnixSeconds: correction.CreatedAt.Unix()}
}

func optionalSession(ok bool, session focus.Session) *focusv1.FocusSession {
	if !ok {
		return nil
	}
	return domainToProto(session)
}
