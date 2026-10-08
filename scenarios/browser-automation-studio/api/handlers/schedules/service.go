package schedules

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/scheduler"
	schedulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schedules"
)

const (
	defaultMaxPerSchedule = 100
	maxPerScheduleCap     = 1000
	maxOccurrenceRange    = 365 * 24 * time.Hour

	maxNameLength = 255
	maxCronLength = 100
	maxTZLength   = 50
)

type service struct {
	deps Deps
}

// =============================================================================
// Create
// =============================================================================

func (s *service) Create(
	ctx context.Context,
	req *connect.Request[schedulesv1.CreateScheduleRequest],
) (*connect.Response[schedulesv1.ScheduleMutationResponse], error) {
	msg := req.Msg
	workflowID, err := uuid.Parse(msg.GetWorkflowId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowID)
	}

	name := strings.TrimSpace(msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNameRequired)
	}
	if len(name) > maxNameLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNameTooLong)
	}

	cronExpr := strings.TrimSpace(msg.GetCronExpression())
	if cronExpr == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errCronRequired)
	}
	if len(cronExpr) > maxCronLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, errCronTooLong)
	}
	if err := scheduler.ValidateCronExpression(cronExpr); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidCron)
	}

	tz := strings.TrimSpace(msg.GetTimezone())
	if tz == "" {
		tz = "UTC"
	} else {
		if len(tz) > maxTZLength {
			return nil, connect.NewError(connect.CodeInvalidArgument, errTimezoneTooLong)
		}
		if _, tzErr := time.LoadLocation(tz); tzErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidTimezone)
		}
	}

	workflow, err := s.deps.Catalog.GetWorkflow(ctx, workflowID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errWorkflowNotFound)
		}
		s.deps.Logger.WithError(err).WithField("workflow_id", workflowID).Error("Failed to verify workflow")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	isActive := true
	if msg.IsActive != nil {
		isActive = *msg.IsActive
	}

	schedule := &database.ScheduleIndex{
		WorkflowID:     workflowID,
		Name:           name,
		CronExpression: cronExpr,
		Timezone:       tz,
		IsActive:       isActive,
	}
	if params := structToMap(msg.GetParameters()); params != nil {
		_ = schedule.SetParameters(params)
	}
	if next, err := scheduler.CalculateNextRun(cronExpr, tz); err == nil && !next.IsZero() {
		schedule.NextRunAt = &next
	}

	if err := s.deps.Repo.CreateSchedule(ctx, schedule); err != nil {
		s.deps.Logger.WithError(err).Error("Failed to create schedule")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := scheduleToProto(schedule, workflowName(workflow))
	resp.LastRunStatus = "never"
	return connect.NewResponse(&schedulesv1.ScheduleMutationResponse{
		ScheduleId: schedule.ID.String(),
		Status:     "created",
		Schedule:   resp,
	}), nil
}

// =============================================================================
// List / Get
// =============================================================================

func (s *service) ListByWorkflow(
	ctx context.Context,
	req *connect.Request[schedulesv1.ListByWorkflowRequest],
) (*connect.Response[schedulesv1.ListSchedulesResponse], error) {
	workflowID, err := uuid.Parse(req.Msg.GetWorkflowId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowID)
	}

	schedules, err := s.deps.Repo.ListSchedules(ctx, &workflowID, req.Msg.GetActiveOnly(), int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		s.deps.Logger.WithError(err).WithField("workflow_id", workflowID).Error("Failed to list workflow schedules")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wfName := ""
	if wf, wfErr := s.deps.Catalog.GetWorkflow(ctx, workflowID); wfErr == nil {
		wfName = workflowName(wf)
	}

	out := make([]*schedulesv1.WorkflowSchedule, len(schedules))
	for i, sch := range schedules {
		out[i] = scheduleToProto(sch, wfName)
	}
	return connect.NewResponse(&schedulesv1.ListSchedulesResponse{
		Schedules: out,
		Total:     int32(len(out)),
	}), nil
}

func (s *service) List(
	ctx context.Context,
	req *connect.Request[schedulesv1.ListSchedulesRequest],
) (*connect.Response[schedulesv1.ListSchedulesResponse], error) {
	var workflowFilter *uuid.UUID
	if raw := strings.TrimSpace(req.Msg.GetWorkflowId()); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowID)
		}
		workflowFilter = &id
	}

	schedules, err := s.deps.Repo.ListSchedules(ctx, workflowFilter, req.Msg.GetActiveOnly(), int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		s.deps.Logger.WithError(err).Error("Failed to list schedules")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wfNames := make(map[uuid.UUID]string)
	out := make([]*schedulesv1.WorkflowSchedule, len(schedules))
	for i, sch := range schedules {
		name, ok := wfNames[sch.WorkflowID]
		if !ok {
			if wf, wfErr := s.deps.Catalog.GetWorkflow(ctx, sch.WorkflowID); wfErr == nil {
				name = workflowName(wf)
				wfNames[sch.WorkflowID] = name
			}
		}
		out[i] = scheduleToProto(sch, name)
	}
	return connect.NewResponse(&schedulesv1.ListSchedulesResponse{
		Schedules: out,
		Total:     int32(len(out)),
	}), nil
}

func (s *service) Get(
	ctx context.Context,
	req *connect.Request[schedulesv1.GetScheduleRequest],
) (*connect.Response[schedulesv1.ScheduleResponse], error) {
	scheduleID, err := uuid.Parse(req.Msg.GetScheduleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidScheduleID)
	}

	schedule, err := s.deps.Repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errScheduleNotFound)
		}
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wfName := ""
	if wf, wfErr := s.deps.Catalog.GetWorkflow(ctx, schedule.WorkflowID); wfErr == nil {
		wfName = workflowName(wf)
	}
	return connect.NewResponse(&schedulesv1.ScheduleResponse{
		Schedule: scheduleToProto(schedule, wfName),
	}), nil
}

// =============================================================================
// Update / Delete / Toggle / Trigger
// =============================================================================

func (s *service) Update(
	ctx context.Context,
	req *connect.Request[schedulesv1.UpdateScheduleRequest],
) (*connect.Response[schedulesv1.ScheduleMutationResponse], error) {
	msg := req.Msg
	scheduleID, err := uuid.Parse(msg.GetScheduleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidScheduleID)
	}

	schedule, err := s.deps.Repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errScheduleNotFound)
		}
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule for update")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if msg.Name != nil {
		name := strings.TrimSpace(*msg.Name)
		if name != "" {
			if len(name) > maxNameLength {
				return nil, connect.NewError(connect.CodeInvalidArgument, errNameTooLong)
			}
			schedule.Name = name
		}
	}
	if msg.CronExpression != nil {
		cronExpr := strings.TrimSpace(*msg.CronExpression)
		if len(cronExpr) > maxCronLength {
			return nil, connect.NewError(connect.CodeInvalidArgument, errCronTooLong)
		}
		if err := scheduler.ValidateCronExpression(cronExpr); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidCron)
		}
		schedule.CronExpression = cronExpr
	}
	if msg.Timezone != nil {
		tz := strings.TrimSpace(*msg.Timezone)
		if len(tz) > maxTZLength {
			return nil, connect.NewError(connect.CodeInvalidArgument, errTimezoneTooLong)
		}
		if _, tzErr := time.LoadLocation(tz); tzErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidTimezone)
		}
		schedule.Timezone = tz
	}
	// Recompute next_run_at when either cron or timezone changed.
	if msg.CronExpression != nil || msg.Timezone != nil {
		if next, err := scheduler.CalculateNextRun(schedule.CronExpression, schedule.Timezone); err == nil && !next.IsZero() {
			schedule.NextRunAt = &next
		}
	}
	if msg.GetParameters() != nil {
		_ = schedule.SetParameters(structToMap(msg.GetParameters()))
	}
	if msg.IsActive != nil {
		schedule.IsActive = *msg.IsActive
	}

	if err := s.deps.Repo.UpdateSchedule(ctx, schedule); err != nil {
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to update schedule")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wfName := ""
	if wf, wfErr := s.deps.Catalog.GetWorkflow(ctx, schedule.WorkflowID); wfErr == nil {
		wfName = workflowName(wf)
	}
	return connect.NewResponse(&schedulesv1.ScheduleMutationResponse{
		ScheduleId: schedule.ID.String(),
		Status:     "updated",
		Schedule:   scheduleToProto(schedule, wfName),
	}), nil
}

func (s *service) Delete(
	ctx context.Context,
	req *connect.Request[schedulesv1.DeleteScheduleRequest],
) (*connect.Response[schedulesv1.DeleteScheduleResponse], error) {
	scheduleID, err := uuid.Parse(req.Msg.GetScheduleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidScheduleID)
	}
	if err := s.deps.Repo.DeleteSchedule(ctx, scheduleID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errScheduleNotFound)
		}
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to delete schedule")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&schedulesv1.DeleteScheduleResponse{ScheduleId: scheduleID.String()}), nil
}

func (s *service) Toggle(
	ctx context.Context,
	req *connect.Request[schedulesv1.ToggleScheduleRequest],
) (*connect.Response[schedulesv1.ToggleScheduleResponse], error) {
	scheduleID, err := uuid.Parse(req.Msg.GetScheduleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidScheduleID)
	}

	schedule, err := s.deps.Repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errScheduleNotFound)
		}
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule for toggle")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	schedule.IsActive = !schedule.IsActive
	if schedule.IsActive {
		if next, err := scheduler.CalculateNextRun(schedule.CronExpression, schedule.Timezone); err == nil && !next.IsZero() {
			schedule.NextRunAt = &next
		}
	}
	if err := s.deps.Repo.UpdateSchedule(ctx, schedule); err != nil {
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to toggle schedule")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wfName := ""
	if wf, wfErr := s.deps.Catalog.GetWorkflow(ctx, schedule.WorkflowID); wfErr == nil {
		wfName = workflowName(wf)
	}
	return connect.NewResponse(&schedulesv1.ToggleScheduleResponse{
		ScheduleId: schedule.ID.String(),
		IsActive:   schedule.IsActive,
		Schedule:   scheduleToProto(schedule, wfName),
	}), nil
}

func (s *service) Trigger(
	ctx context.Context,
	req *connect.Request[schedulesv1.TriggerScheduleRequest],
) (*connect.Response[schedulesv1.TriggerScheduleResponse], error) {
	scheduleID, err := uuid.Parse(req.Msg.GetScheduleId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidScheduleID)
	}

	schedule, err := s.deps.Repo.GetSchedule(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errScheduleNotFound)
		}
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to get schedule for trigger")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	params := make(map[string]any)
	if parsed, perr := schedule.GetParameters(); perr == nil {
		for k, v := range parsed {
			params[k] = v
		}
	}
	params["_trigger_type"] = "manual_schedule_trigger"
	params["_schedule_id"] = scheduleID.String()
	params["_schedule_name"] = schedule.Name

	execution, err := s.deps.Executor.ExecuteWorkflow(ctx, schedule.WorkflowID, params)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("schedule_id", scheduleID).Error("Failed to trigger scheduled workflow")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	now := time.Now()
	if updateErr := s.deps.Repo.UpdateScheduleLastRun(ctx, scheduleID, now); updateErr != nil {
		s.deps.Logger.WithError(updateErr).WithField("schedule_id", scheduleID).Warn("Failed to update schedule last_run_at after trigger")
	}

	return connect.NewResponse(&schedulesv1.TriggerScheduleResponse{
		ScheduleId:  scheduleID.String(),
		ExecutionId: execution.ID.String(),
		WorkflowId:  schedule.WorkflowID.String(),
		TriggeredAt: timestamppb.New(now),
	}), nil
}

// =============================================================================
// Occurrences (calendar projection)
// =============================================================================

func (s *service) Occurrences(
	ctx context.Context,
	req *connect.Request[schedulesv1.OccurrencesRequest],
) (*connect.Response[schedulesv1.OccurrencesResponse], error) {
	startPB := req.Msg.GetStart()
	endPB := req.Msg.GetEnd()
	if startPB == nil || endPB == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRangeRequired)
	}
	start := startPB.AsTime()
	end := endPB.AsTime()
	if end.Before(start) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRangeInverted)
	}
	if end.Sub(start) > maxOccurrenceRange {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRangeTooLarge)
	}

	maxPer := int(req.Msg.GetMaxPerSchedule())
	if maxPer <= 0 {
		maxPer = defaultMaxPerSchedule
	}
	if maxPer > maxPerScheduleCap {
		maxPer = maxPerScheduleCap
	}

	var workflowFilter *uuid.UUID
	if raw := strings.TrimSpace(req.Msg.GetWorkflowId()); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errInvalidWorkflowID)
		}
		workflowFilter = &id
	}

	schedules, err := s.deps.Repo.ListSchedules(ctx, workflowFilter, true, 0, 0)
	if err != nil {
		s.deps.Logger.WithError(err).Error("Failed to list schedules for occurrences")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wfNames := make(map[uuid.UUID]string)
	for _, sch := range schedules {
		if _, ok := wfNames[sch.WorkflowID]; !ok {
			if wf, wfErr := s.deps.Catalog.GetWorkflow(ctx, sch.WorkflowID); wfErr == nil {
				wfNames[sch.WorkflowID] = workflowName(wf)
			}
		}
	}

	occurrences := make([]*schedulesv1.ScheduleOccurrence, 0)
	aggregates := make(map[string]*schedulesv1.ScheduleAggregate)

	for _, sch := range schedules {
		runs, calcErr := calculateOccurrencesInRange(sch.CronExpression, sch.Timezone, start, end, maxPer+1)
		if calcErr != nil {
			s.deps.Logger.WithError(calcErr).WithField("schedule_id", sch.ID).Warn("Failed to calculate occurrences for schedule")
			continue
		}

		if len(runs) > maxPer {
			total := estimateTotalRuns(sch.CronExpression, start, end)
			aggregates[sch.ID.String()] = &schedulesv1.ScheduleAggregate{
				ScheduleId:     sch.ID.String(),
				ScheduleName:   sch.Name,
				TotalRuns:      int32(total),
				Truncated:      true,
				CronExpression: sch.CronExpression,
			}
			if len(runs) > maxPer {
				runs = runs[:maxPer]
			}
		}

		for _, runAt := range runs {
			occurrences = append(occurrences, &schedulesv1.ScheduleOccurrence{
				ScheduleId:     sch.ID.String(),
				ScheduleName:   sch.Name,
				WorkflowId:     sch.WorkflowID.String(),
				WorkflowName:   wfNames[sch.WorkflowID],
				RunAt:          timestamppb.New(runAt),
				IsActive:       sch.IsActive,
				CronExpression: sch.CronExpression,
				Timezone:       sch.Timezone,
			})
		}
	}

	return connect.NewResponse(&schedulesv1.OccurrencesResponse{
		Occurrences: occurrences,
		Aggregates:  aggregates,
		Total:       int32(len(occurrences)),
		RangeStart:  timestamppb.New(start),
		RangeEnd:    timestamppb.New(end),
	}), nil
}

// =============================================================================
// Helpers
// =============================================================================

func scheduleToProto(s *database.ScheduleIndex, workflowName string) *schedulesv1.WorkflowSchedule {
	if s == nil {
		return nil
	}
	out := &schedulesv1.WorkflowSchedule{
		Id:             s.ID.String(),
		WorkflowId:     s.WorkflowID.String(),
		Name:           s.Name,
		CronExpression: s.CronExpression,
		Timezone:       s.Timezone,
		IsActive:       s.IsActive,
		CreatedAt:      timestamppb.New(s.CreatedAt),
		UpdatedAt:      timestamppb.New(s.UpdatedAt),
		WorkflowName:   workflowName,
		NextRunHuman:   formatRelativeTime(s.NextRunAt),
		LastRunStatus:  lastRunStatus(s.LastRunAt),
	}
	if s.NextRunAt != nil {
		out.NextRunAt = timestamppb.New(*s.NextRunAt)
	}
	if s.LastRunAt != nil {
		out.LastRunAt = timestamppb.New(*s.LastRunAt)
	}
	if params, err := s.GetParameters(); err == nil && len(params) > 0 {
		if pb, err := structpb.NewStruct(params); err == nil {
			out.Parameters = pb
		}
	}
	return out
}

func structToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func workflowName(wf interface{ GetName() string }) string {
	if wf == nil {
		return ""
	}
	return wf.GetName()
}

func formatRelativeTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	now := time.Now()
	diff := t.Sub(now)
	past := diff < 0
	if past {
		diff = -diff
	}

	var phrase string
	switch {
	case diff < time.Minute:
		if past {
			return "just now"
		}
		return "in less than a minute"
	case diff < time.Hour:
		phrase = pluralize(int(diff.Minutes()), "minute")
	case diff < 24*time.Hour:
		phrase = pluralize(int(diff.Hours()), "hour")
	default:
		phrase = pluralize(int(diff.Hours()/24), "day")
	}
	if past {
		return phrase + " ago"
	}
	return "in " + phrase
}

func pluralize(count int, unit string) string {
	if count == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(count) + " " + unit + "s"
}

func lastRunStatus(lastRun *time.Time) string {
	if lastRun == nil {
		return "never"
	}
	return "success"
}

func calculateOccurrencesInRange(cronExpr, timezone string, start, end time.Time, maxRuns int) ([]time.Time, error) {
	cronSchedule, err := scheduler.ParseCronExpression(cronExpr)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	var runs []time.Time
	current := start.In(loc)
	for len(runs) < maxRuns {
		next := cronSchedule.Next(current)
		if next.After(end) {
			break
		}
		runs = append(runs, next.UTC())
		current = next
	}
	return runs, nil
}

func estimateTotalRuns(cronExpr string, start, end time.Time) int {
	duration := end.Sub(start)
	fields := strings.Fields(strings.TrimSpace(cronExpr))
	if len(fields) >= 5 {
		minute := fields[0]
		hour := fields[1]
		if minute == "*" && hour == "*" {
			return int(duration.Minutes())
		}
		if strings.HasPrefix(minute, "*/") && hour == "*" {
			if n, err := strconv.Atoi(strings.TrimPrefix(minute, "*/")); err == nil && n > 0 {
				return int(duration.Minutes()) / n
			}
		}
		if minute == "0" && hour == "*" {
			return int(duration.Hours())
		}
		if minute == "0" && strings.HasPrefix(hour, "*/") {
			if n, err := strconv.Atoi(strings.TrimPrefix(hour, "*/")); err == nil && n > 0 {
				return int(duration.Hours()) / n
			}
		}
	}
	return int(duration.Hours() / 24)
}
