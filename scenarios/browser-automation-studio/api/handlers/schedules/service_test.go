package schedules

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
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

// fakeRepo implements the Repo seam for handler tests.
type fakeRepo struct {
	mu      sync.Mutex
	items   map[uuid.UUID]*database.ScheduleIndex
	listErr error
	getErr  error
	putErr  error
	delErr  error
	lastRun map[uuid.UUID]time.Time
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: make(map[uuid.UUID]*database.ScheduleIndex), lastRun: make(map[uuid.UUID]time.Time)}
}

func (f *fakeRepo) CreateSchedule(_ context.Context, s *database.ScheduleIndex) error {
	if f.putErr != nil {
		return f.putErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()
	f.items[s.ID] = s
	return nil
}

func (f *fakeRepo) GetSchedule(_ context.Context, id uuid.UUID) (*database.ScheduleIndex, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.items[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	return s, nil
}

func (f *fakeRepo) UpdateSchedule(_ context.Context, s *database.ScheduleIndex) error {
	if f.putErr != nil {
		return f.putErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s.UpdatedAt = time.Now()
	f.items[s.ID] = s
	return nil
}

func (f *fakeRepo) DeleteSchedule(_ context.Context, id uuid.UUID) error {
	if f.delErr != nil {
		return f.delErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[id]; !ok {
		return database.ErrNotFound
	}
	delete(f.items, id)
	return nil
}

func (f *fakeRepo) ListSchedules(_ context.Context, workflowID *uuid.UUID, activeOnly bool, _, _ int) ([]*database.ScheduleIndex, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*database.ScheduleIndex
	for _, s := range f.items {
		if workflowID != nil && s.WorkflowID != *workflowID {
			continue
		}
		if activeOnly && !s.IsActive {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func (f *fakeRepo) UpdateScheduleLastRun(_ context.Context, id uuid.UUID, lastRun time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastRun[id] = lastRun
	if s, ok := f.items[id]; ok {
		s.LastRunAt = &lastRun
	}
	return nil
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

// fakeExecutor implements Executor.
type fakeExecutor struct {
	lastWorkflowID uuid.UUID
	lastParams     map[string]any
	out            *database.ExecutionIndex
	err            error
}

func (e *fakeExecutor) ExecuteWorkflow(_ context.Context, wfID uuid.UUID, params map[string]any) (*database.ExecutionIndex, error) {
	e.lastWorkflowID = wfID
	e.lastParams = params
	if e.err != nil {
		return nil, e.err
	}
	if e.out != nil {
		return e.out, nil
	}
	return &database.ExecutionIndex{ID: uuid.New(), WorkflowID: wfID}, nil
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

type harness struct {
	client   schedulesconnect.SchedulesServiceClient
	repo     *fakeRepo
	catalog  *fakeCatalog
	executor *fakeExecutor
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	repo := newFakeRepo()
	catalog := &fakeCatalog{workflows: make(map[uuid.UUID]*basapi.WorkflowSummary)}
	executor := &fakeExecutor{}
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	mount := Module(Deps{Repo: repo, Catalog: catalog, Executor: executor, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
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
	require.Equal(t, wfID, h.executor.lastWorkflowID)
	require.Equal(t, "bar", h.executor.lastParams["foo"])
	require.Equal(t, "manual_schedule_trigger", h.executor.lastParams["_trigger_type"])
	require.Equal(t, s.ID.String(), h.executor.lastParams["_schedule_id"])

	_, ok := h.repo.lastRun[s.ID]
	require.True(t, ok)
}

func TestTriggerExecutorFailure(t *testing.T) {
	h := newHarness(t)
	wfID := uuid.New()
	h.catalog.workflows[wfID] = &basapi.WorkflowSummary{Name: "wf"}
	s := &database.ScheduleIndex{WorkflowID: wfID, Name: "n", CronExpression: "0 * * * *", Timezone: "UTC", IsActive: true}
	require.NoError(t, h.repo.CreateSchedule(context.Background(), s))
	h.executor.err = errors.New("boom")

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
