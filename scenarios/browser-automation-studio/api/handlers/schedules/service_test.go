package schedules

import (
	"context"
	"errors"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/internal/testutil/executormocks"
	"github.com/vrooli/browser-automation-studio/internal/testutil/schedulemocks"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	schedulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schedules"
	schedulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schedules/schedulesconnect"
)

func newFakeRepo() *schedulemocks.Repository {
	return schedulemocks.NewRepository()
}

// fakeCatalog implements the Catalog seam.
type fakeCatalog struct {
	workflows map[uuid.UUID]*basapi.WorkflowSummary
	err       error
}

func (c *fakeCatalog) GetWorkflow(_ context.Context, id uuid.UUID) (*basapi.WorkflowSummary, error) {
	if c.err != nil {
		return nil, c.err
	}
	w, ok := c.workflows[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	return w, nil
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

type harness struct {
	client   schedulesconnect.SchedulesServiceClient
	repo     *schedulemocks.Repository
	catalog  *fakeCatalog
	executor *executormocks.Executor
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	repo := newFakeRepo()
	catalog := &fakeCatalog{workflows: make(map[uuid.UUID]*basapi.WorkflowSummary)}
	executor := &executormocks.Executor{}
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount := Module(Deps{Repo: repo, Catalog: catalog, Executor: executor, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := testutil.StartHTTPServer(t, mux)
	t.Cleanup(srv.Close)
	return &harness{
		client:   schedulesconnect.NewSchedulesServiceClient(srv.Client(), srv.URL),
		repo:     repo,
		catalog:  catalog,
		executor: executor,
	}
}

func TestModulePanicsOnMissingDeps(t *testing.T) {
	require.Panics(t, func() { Module(Deps{}) })
	require.Panics(t, func() { Module(Deps{Logger: logrus.New()}) })
	require.Panics(t, func() { Module(Deps{Logger: logrus.New(), Repo: newFakeRepo()}) })
	require.Panics(t, func() {
		Module(Deps{Logger: logrus.New(), Repo: newFakeRepo(), Catalog: &fakeCatalog{}})
	})
}

func TestCreateHappyPath(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "My Workflow"}
	res, err := h.client.Create(context.Background(), connect.NewRequest(&schedulesv1.CreateScheduleRequest{
		WorkflowId:     wfID.String(),
		Name:           "Nightly",
		CronExpression: "0 0 * * *",
		Timezone:       "UTC",
	}))
	require.NoError(t, err)
	require.Equal(t, "created", res.Msg.GetStatus())
	require.NotEmpty(t, res.Msg.GetScheduleId())
	require.True(t, res.Msg.GetSchedule().GetIsActive())
	require.Equal(t, "My Workflow", res.Msg.GetSchedule().GetWorkflowName())
	require.Equal(t, "never", res.Msg.GetSchedule().GetLastRunStatus())
}

func TestCreateInvalidWorkflowID(t *testing.T) {
	h := newHarness(t)
	_, err := h.client.Create(context.Background(), connect.NewRequest(&schedulesv1.CreateScheduleRequest{
		WorkflowId:     "not-a-uuid",
		Name:           "x",
		CronExpression: "0 0 * * *",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateRejectsEmptyName(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	_, err := h.client.Create(context.Background(), connect.NewRequest(&schedulesv1.CreateScheduleRequest{
		WorkflowId:     wfID.String(),
		Name:           "   ",
		CronExpression: "0 0 * * *",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateRejectsInvalidCron(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	_, err := h.client.Create(context.Background(), connect.NewRequest(&schedulesv1.CreateScheduleRequest{
		WorkflowId:     wfID.String(),
		Name:           "n",
		CronExpression: "definitely not cron",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateRejectsInvalidTimezone(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	_, err := h.client.Create(context.Background(), connect.NewRequest(&schedulesv1.CreateScheduleRequest{
		WorkflowId:     wfID.String(),
		Name:           "n",
		CronExpression: "0 0 * * *",
		Timezone:       "Mars/Olympus",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateUnknownWorkflow(t *testing.T) {
	h := newHarness(t)
	_, err := h.client.Create(context.Background(), connect.NewRequest(&schedulesv1.CreateScheduleRequest{
		WorkflowId:     uuid.New().String(),
		Name:           "n",
		CronExpression: "0 0 * * *",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestListAndListByWorkflow(t *testing.T) {
	h := newHarness(t)
	wfA := uuid.New()
	wfB := uuid.New()
	h.catalog.workflows[wfA] = &basapi.WorkflowSummary{Name: "A"}
	h.catalog.workflows[wfB] = &basapi.WorkflowSummary{Name: "B"}
	// Seed two schedules — one active, one inactive — for each workflow.
	for _, wf := range []uuid.UUID{wfA, wfB} {
		for _, active := range []bool{true, false} {
			require.NoError(t, h.repo.CreateSchedule(context.Background(), &database.ScheduleIndex{
				WorkflowID: wf, Name: "s", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: active,
			}))
		}
	}

	all, err := h.client.List(context.Background(), connect.NewRequest(&schedulesv1.ListSchedulesRequest{}))
	require.NoError(t, err)
	require.Equal(t, int32(4), all.Msg.GetTotal())

	activeOnly, err := h.client.List(context.Background(), connect.NewRequest(&schedulesv1.ListSchedulesRequest{ActiveOnly: true}))
	require.NoError(t, err)
	require.Equal(t, int32(2), activeOnly.Msg.GetTotal())

	byWF, err := h.client.ListByWorkflow(context.Background(), connect.NewRequest(&schedulesv1.ListByWorkflowRequest{WorkflowId: wfA.String()}))
	require.NoError(t, err)
	require.Equal(t, int32(2), byWF.Msg.GetTotal())
	for _, s := range byWF.Msg.GetSchedules() {
		require.Equal(t, "A", s.GetWorkflowName())
	}
}

func TestGetNotFound(t *testing.T) {
	h := newHarness(t)
	_, err := h.client.Get(context.Background(), connect.NewRequest(&schedulesv1.GetScheduleRequest{ScheduleId: uuid.New().String()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestUpdateAppliesPartialFields(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "old", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: true}
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))

	newName := "new"
	inactive := false
	res, err := h.client.Update(context.Background(), connect.NewRequest(&schedulesv1.UpdateScheduleRequest{
		ScheduleId: s.ID.String(),
		Name:       &newName,
		IsActive:   &inactive,
	}))
	require.NoError(t, err)
	require.Equal(t, "updated", res.Msg.GetStatus())
	require.Equal(t, "new", res.Msg.GetSchedule().GetName())
	require.False(t, res.Msg.GetSchedule().GetIsActive())
}

func TestUpdateRejectsBadCron(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "n", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: true}
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))

	bad := "garbage"
	_, err := h.client.Update(context.Background(), connect.NewRequest(&schedulesv1.UpdateScheduleRequest{
		ScheduleId:     s.ID.String(),
		CronExpression: &bad,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestDelete(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "n", CronExpression: "0 * * * *", Timezone: "UTC"}
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))

	res, err := h.client.Delete(context.Background(), connect.NewRequest(&schedulesv1.DeleteScheduleRequest{ScheduleId: s.ID.String()}))
	require.NoError(t, err)
	require.Equal(t, s.ID.String(), res.Msg.GetScheduleId())

	_, err = h.client.Get(context.Background(), connect.NewRequest(&schedulesv1.GetScheduleRequest{ScheduleId: s.ID.String()}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestToggleFlipsActiveAndRecomputesNextRun(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "n", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: false}
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))

	res, err := h.client.Toggle(context.Background(), connect.NewRequest(&schedulesv1.ToggleScheduleRequest{ScheduleId: s.ID.String()}))
	require.NoError(t, err)
	require.True(t, res.Msg.GetIsActive())
	require.NotNil(t, res.Msg.GetSchedule().GetNextRunAt())
}

func TestTriggerExecutesAndUpdatesLastRun(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "Nightly", CronExpression: "0 0 * * *", Timezone: "UTC", IsActive: true}
	require.NoError(t, s.SetParameters(map[string]any{"foo": "bar"}))
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))

	res, err := h.client.Trigger(context.Background(), connect.NewRequest(&schedulesv1.TriggerScheduleRequest{ScheduleId: s.ID.String()}))
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetExecutionId())
	require.Equal(t, wfID.String(), res.Msg.GetWorkflowId())
	calls := h.executor.Calls()
	require.Len(t, calls, 1)
	require.Equal(t, wfID, calls[0].WorkflowID)
	require.Equal(t, "bar", calls[0].Parameters["foo"])
	require.Equal(t, "manual_schedule_trigger", calls[0].Parameters["_trigger_type"])
	require.Equal(t, s.ID.String(), calls[0].Parameters["_schedule_id"])

	_, ok := h.repo.LastRun(s.ID)
	require.True(t, ok)
}

func TestTriggerExecutorFailure(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "n", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: true}
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))
	h.executor.Err = errors.New("boom")

	_, err := h.client.Trigger(context.Background(), connect.NewRequest(&schedulesv1.TriggerScheduleRequest{ScheduleId: s.ID.String()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

func TestOccurrencesValidatesRange(t *testing.T) {
	h := newHarness(t)
	now := time.Now()

	// missing both endpoints
	_, err := h.client.Occurrences(context.Background(), connect.NewRequest(&schedulesv1.OccurrencesRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	// inverted
	_, err = h.client.Occurrences(context.Background(), connect.NewRequest(&schedulesv1.OccurrencesRequest{
		Start: timestamppb.New(now.Add(time.Hour)),
		End:   timestamppb.New(now),
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	// too large
	_, err = h.client.Occurrences(context.Background(), connect.NewRequest(&schedulesv1.OccurrencesRequest{
		Start: timestamppb.New(now),
		End:   timestamppb.New(now.AddDate(2, 0, 0)),
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestOccurrencesProjectsActiveSchedules(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	require.NoError(t, h.repo.CreateSchedule(context.Background(), &database.ScheduleIndex{
		WorkflowID: wfID, Name: "Hourly", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: true,
	}))
	// Inactive schedule must be skipped.
	require.NoError(t, h.repo.CreateSchedule(context.Background(), &database.ScheduleIndex{
		WorkflowID: wfID, Name: "Off", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: false,
	}))

	now := time.Now().UTC()
	res, err := h.client.Occurrences(context.Background(), connect.NewRequest(&schedulesv1.OccurrencesRequest{
		Start:          timestamppb.New(now),
		End:            timestamppb.New(now.Add(3 * time.Hour)),
		MaxPerSchedule: 10,
	}))
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(res.Msg.GetOccurrences()), 2)
	for _, occ := range res.Msg.GetOccurrences() {
		require.Equal(t, "Hourly", occ.GetScheduleName())
	}
}

func TestUpdateReplacesParameters(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "n", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: true}
	require.NoError(t, s.SetParameters(map[string]any{"old": true}))
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))

	newParams, err := structpb.NewStruct(map[string]any{"new": "value"})
	require.NoError(t, err)
	res, err := h.client.Update(context.Background(), connect.NewRequest(&schedulesv1.UpdateScheduleRequest{
		ScheduleId: s.ID.String(),
		Parameters: newParams,
	}))
	require.NoError(t, err)
	require.Equal(t, "value", res.Msg.GetSchedule().GetParameters().GetFields()["new"].GetStringValue())
}
