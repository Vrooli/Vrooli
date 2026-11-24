package activity_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity"
	activityhandler "react-component-library/handlers/activity"
	domain "react-component-library/internal/activity"
)

type stubService struct {
	listFilter  domain.Filter
	listLimit   int
	events      []domain.Event
	listErr     error
	summary     domain.Summary
	summaryDays int
	summaryErr  error
}

func (s *stubService) List(_ context.Context, filter domain.Filter, limit int) ([]domain.Event, error) {
	s.listFilter, s.listLimit = filter, limit
	return s.events, s.listErr
}

func (s *stubService) Summary(_ context.Context, days int) (domain.Summary, error) {
	s.summaryDays = days
	return s.summary, s.summaryErr
}

func TestListActivityMapsFiltersAndEventFields(t *testing.T) {
	service := &stubService{events: []domain.Event{{
		ID: "evt-1", Actor: "ada", Action: "asset.publish", SubjectKind: "asset", SubjectID: "button",
		Summary: "Published button", BeforeJSON: `{"state":"draft"}`, AfterJSON: `{"state":"published"}`,
		At: "2026-11-18T09:00:00Z", TraceID: "trace-1",
	}}}
	resp, err := activityhandler.NewHandler(service).ListActivity(context.Background(), connect.NewRequest(&activityv1.ListActivityRequest{
		SubjectKind: " asset ", SubjectId: " button ", Actor: " ada ", Since: " 2026-11-18T00:00:00Z ", Limit: 17,
	}))
	require.NoError(t, err)
	require.Equal(t, domain.Filter{SubjectKind: "asset", SubjectID: "button", Actor: "ada", Since: "2026-11-18T00:00:00Z"}, service.listFilter)
	require.Equal(t, 17, service.listLimit)
	require.Equal(t, &activityv1.ActivityEvent{
		Id: "evt-1", Actor: "ada", Action: "asset.publish", SubjectKind: "asset", SubjectId: "button",
		Summary: "Published button", BeforeJson: `{"state":"draft"}`, AfterJson: `{"state":"published"}`,
		At: "2026-11-18T09:00:00Z", TraceId: "trace-1",
	}, resp.Msg.Events[0])
}

func TestActivityHandlerMapsFailuresAndRejectsMissingService(t *testing.T) {
	t.Run("invalid input", func(t *testing.T) {
		service := &stubService{listErr: errors.Join(domain.ErrInvalid, errors.New("since must be RFC3339"))}
		_, err := activityhandler.NewHandler(service).ListActivity(context.Background(), connect.NewRequest(&activityv1.ListActivityRequest{}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Contains(t, err.Error(), "since must be RFC3339")
	})
	t.Run("storage error is sanitized", func(t *testing.T) {
		service := &stubService{summaryErr: errors.New("private sqlite details")}
		_, err := activityhandler.NewHandler(service).GetActivitySummary(context.Background(), connect.NewRequest(&activityv1.GetActivitySummaryRequest{Days: 9}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		require.NotContains(t, err.Error(), "private sqlite details")
		require.Equal(t, 9, service.summaryDays)
	})
	t.Run("nil handler", func(t *testing.T) {
		var handler *activityhandler.Handler
		_, err := handler.ListActivity(context.Background(), connect.NewRequest(&activityv1.ListActivityRequest{}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})
	t.Run("nil service", func(t *testing.T) {
		_, err := activityhandler.NewHandler(nil).GetActivitySummary(context.Background(), connect.NewRequest(&activityv1.GetActivitySummaryRequest{}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})
}

func TestGetActivitySummaryMapsEveryProjection(t *testing.T) {
	service := &stubService{summary: domain.Summary{
		Total: 6, Empty: false,
		ByDay:    []domain.DayCount{{Date: "2026-11-18", Count: 6}},
		ByActor:  []domain.ActorCount{{Actor: "ada", Count: 4}, {Actor: "lin", Count: 2}},
		ByAction: []domain.ActionCount{{Action: "asset.publish", Count: 6}},
	}}
	resp, err := activityhandler.NewHandler(service).GetActivitySummary(context.Background(), connect.NewRequest(&activityv1.GetActivitySummaryRequest{Days: 30}))
	require.NoError(t, err)
	require.Equal(t, 30, service.summaryDays)
	require.Equal(t, int32(6), resp.Msg.Total)
	require.Equal(t, &activityv1.ActivityDay{Date: "2026-11-18", Count: 6}, resp.Msg.ByDay[0])
	require.Equal(t, []*activityv1.ActivityActor{{Actor: "ada", Count: 4}, {Actor: "lin", Count: 2}}, resp.Msg.ByActor)
	require.Equal(t, &activityv1.ActivityAction{Action: "asset.publish", Count: 6}, resp.Msg.ByAction[0])
	require.False(t, resp.Msg.Empty)
}

func TestModuleDescriptorAndSchema(t *testing.T) {
	mod := activityhandler.Module(nil)
	require.Equal(t, "activity", mod.Name)
	require.Len(t, mod.Endpoints, 2)
	require.NotEmpty(t, activityhandler.Schema())
	router := mux.NewRouter()
	require.NotPanics(t, func() { mod.Mount(router) })
}
