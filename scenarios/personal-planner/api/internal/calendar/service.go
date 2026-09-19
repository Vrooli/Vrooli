package calendar

import (
	"context"
	"strings"
	"time"
)

type Service interface {
	ListToday(context.Context, string) (Today, error)
	ListRange(context.Context, string, string) (DateRange, error)
	Create(context.Context, CreateInput) (Allocation, error)
	Preview(context.Context, PreviewInput) (PlacementProposal, error)
	ApplyProposal(context.Context, ApplyProposalInput) (Allocation, error)
	PreviewSchedule(context.Context, SchedulePreviewInput) (ScheduleProposal, error)
	ApplyScheduleProposal(context.Context, ApplyScheduleProposalInput) ([]Allocation, error)
	CarryForward(context.Context, CarryForwardInput) (Allocation, error)
	ListRoutines(context.Context) ([]Routine, error)
	CreateRoutine(context.Context, CreateRoutineInput) (Routine, error)
	ListRoutineOccurrences(context.Context, string, string) ([]RoutineOccurrence, error)
	SkipRoutineOccurrence(context.Context, string, string, int64) error
	RescheduleRoutineOccurrence(context.Context, string, string, int64, int64) error
}

func (s *service) ListRoutines(ctx context.Context) ([]Routine, error) {
	return s.repo.ListRoutines(ctx)
}

func (s *service) CreateRoutine(ctx context.Context, in CreateRoutineInput) (Routine, error) {
	if strings.TrimSpace(in.Title) == "" {
		return Routine{}, ErrInvalidAllocation{"title", "required"}
	}
	if in.Kind != "fixed" && in.Kind != "flexible" {
		return Routine{}, ErrInvalidAllocation{"kind", "must be fixed or flexible"}
	}
	if _, err := time.LoadLocation(in.Timezone); err != nil {
		return Routine{}, ErrInvalidAllocation{"timezone", "must be an IANA timezone"}
	}
	if _, err := time.Parse("2006-01-02", in.StartDate); err != nil {
		return Routine{}, ErrInvalidAllocation{"start_date", "must be YYYY-MM-DD"}
	}
	if in.EndDate != "" {
		end, err := time.Parse("2006-01-02", in.EndDate)
		if err != nil || in.EndDate < in.StartDate {
			return Routine{}, ErrInvalidAllocation{"end_date", "must be an ordered YYYY-MM-DD date"}
		}
		_ = end
	}
	if len(in.Weekdays) == 0 {
		return Routine{}, ErrInvalidAllocation{"weekdays", "choose at least one weekday"}
	}
	seen := map[int]bool{}
	for _, day := range in.Weekdays {
		if day < 1 || day > 7 || seen[day] {
			return Routine{}, ErrInvalidAllocation{"weekdays", "must contain unique values from 1 through 7"}
		}
		seen[day] = true
	}
	if in.StartMinute < 0 || in.StartMinute >= 1440 {
		return Routine{}, ErrInvalidAllocation{"start_minute", "must be within the day"}
	}
	if in.DurationMinutes <= 0 || in.StartMinute+in.DurationMinutes > 1440 {
		return Routine{}, ErrInvalidAllocation{"duration_minutes", "must fit within the day"}
	}
	if in.Kind == "flexible" && (in.FrequencyPerWeek < 1 || in.FrequencyPerWeek > len(in.Weekdays)) {
		return Routine{}, ErrInvalidAllocation{"frequency_per_week", "must fit the eligible weekday count"}
	}
	if in.Kind == "fixed" {
		in.FrequencyPerWeek = len(in.Weekdays)
	}
	return s.repo.CreateRoutine(ctx, in)
}

func (s *service) ListRoutineOccurrences(ctx context.Context, start, end string) ([]RoutineOccurrence, error) {
	if len(start) != 10 || len(end) != 10 || start > end {
		return nil, ErrInvalidAllocation{"date_range", "must be an ordered YYYY-MM-DD range"}
	}
	if _, err := time.Parse("2006-01-02", start); err != nil {
		return nil, ErrInvalidAllocation{"date_range", "must be YYYY-MM-DD"}
	}
	if _, err := time.Parse("2006-01-02", end); err != nil {
		return nil, ErrInvalidAllocation{"date_range", "must be YYYY-MM-DD"}
	}
	return s.repo.ListRoutineOccurrences(ctx, start, end)
}

func (s *service) SkipRoutineOccurrence(ctx context.Context, routineID, localDate string, expectedRevision int64) error {
	if strings.TrimSpace(routineID) == "" {
		return ErrInvalidAllocation{"routine_id", "required"}
	}
	if _, err := time.Parse("2006-01-02", localDate); err != nil {
		return ErrInvalidAllocation{"local_date", "must be YYYY-MM-DD"}
	}
	if expectedRevision <= 0 {
		return ErrInvalidAllocation{"expected_revision", "must be positive"}
	}
	return s.repo.SkipRoutineOccurrence(ctx, routineID, localDate, expectedRevision)
}

func (s *service) RescheduleRoutineOccurrence(ctx context.Context, routineID, localDate string, startMinute, expectedRevision int64) error {
	if strings.TrimSpace(routineID) == "" {
		return ErrInvalidAllocation{"routine_id", "required"}
	}
	if _, err := time.Parse("2006-01-02", localDate); err != nil {
		return ErrInvalidAllocation{"local_date", "must be YYYY-MM-DD"}
	}
	if startMinute < 0 || startMinute > 1439 {
		return ErrInvalidAllocation{"start_minute", "must be within the day"}
	}
	if expectedRevision <= 0 {
		return ErrInvalidAllocation{"expected_revision", "must be positive"}
	}
	return s.repo.RescheduleRoutineOccurrence(ctx, routineID, localDate, int(startMinute), expectedRevision)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }
func (s *service) ListToday(ctx context.Context, date string) (Today, error) {
	return s.repo.ListToday(ctx, date)
}

func (s *service) ListRange(ctx context.Context, start, end string) (DateRange, error) {
	if len(start) != 10 || len(end) != 10 || start > end {
		return DateRange{}, ErrInvalidAllocation{"date_range", "must be an ordered YYYY-MM-DD range"}
	}
	return s.repo.ListRange(ctx, start, end)
}

func (s *service) Create(ctx context.Context, in CreateInput) (Allocation, error) {
	if strings.TrimSpace(in.WorkItemID) == "" {
		return Allocation{}, ErrInvalidAllocation{"work_item_id", "required"}
	}
	if len(in.LocalDate) != 10 {
		return Allocation{}, ErrInvalidAllocation{"local_date", "must be YYYY-MM-DD"}
	}
	if in.StartMinutes < 0 || in.StartMinutes >= 1440 {
		return Allocation{}, ErrInvalidAllocation{"start_minutes", "must be within the day"}
	}
	if in.DurationMinutes <= 0 || in.DurationMinutes > 480 {
		return Allocation{}, ErrInvalidAllocation{"duration_minutes", "must be between 1 and 480"}
	}
	if in.StartMinutes+in.DurationMinutes > 1440 {
		return Allocation{}, ErrInvalidAllocation{"duration_minutes", "must fit within the day"}
	}
	a, err := s.repo.WorkItem(ctx, in.WorkItemID)
	if err != nil {
		return Allocation{}, err
	}
	a.LocalDate, a.StartMinutes, a.DurationMinutes = in.LocalDate, in.StartMinutes, in.DurationMinutes
	return s.repo.Create(ctx, a)
}

func (s *service) Preview(ctx context.Context, in PreviewInput) (PlacementProposal, error) {
	if strings.TrimSpace(in.WorkItemID) == "" {
		return PlacementProposal{}, ErrInvalidAllocation{"work_item_id", "required"}
	}
	if _, err := time.Parse("2006-01-02", in.LocalDate); err != nil {
		return PlacementProposal{}, ErrInvalidAllocation{"local_date", "must be YYYY-MM-DD"}
	}
	if in.StartMinutes < 0 || in.StartMinutes >= 1440 {
		return PlacementProposal{}, ErrInvalidAllocation{"start_minutes", "must be within the day"}
	}
	if in.DurationMinutes <= 0 || in.DurationMinutes > 480 || in.StartMinutes+in.DurationMinutes > 1440 {
		return PlacementProposal{}, ErrInvalidAllocation{"duration_minutes", "must be between 1 and 480 and fit within the day"}
	}
	if _, err := s.repo.WorkItem(ctx, in.WorkItemID); err != nil {
		return PlacementProposal{}, err
	}
	return s.repo.Preview(ctx, in)
}

func (s *service) ApplyProposal(ctx context.Context, in ApplyProposalInput) (Allocation, error) {
	if strings.TrimSpace(in.ProposalID) == "" {
		return Allocation{}, ErrInvalidAllocation{"proposal_id", "required"}
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return Allocation{}, ErrInvalidAllocation{"idempotency_key", "required"}
	}
	if in.ExpectedRevision <= 0 {
		return Allocation{}, ErrInvalidAllocation{"expected_revision", "must be positive"}
	}
	return s.repo.ApplyProposal(ctx, in)
}

func (s *service) PreviewSchedule(ctx context.Context, in SchedulePreviewInput) (ScheduleProposal, error) {
	if _, err := time.Parse("2006-01-02", in.LocalDate); err != nil {
		return ScheduleProposal{}, ErrInvalidAllocation{"local_date", "must be YYYY-MM-DD"}
	}
	if in.StartMinutes < 0 || in.StartMinutes >= 1440 {
		return ScheduleProposal{}, ErrInvalidAllocation{"start_minutes", "must be within the day"}
	}
	if len(in.WorkItemIDs) == 0 || len(in.WorkItemIDs) > 20 {
		return ScheduleProposal{}, ErrInvalidAllocation{"work_item_ids", "must contain between 1 and 20 items"}
	}
	seen := map[string]bool{}
	for _, id := range in.WorkItemIDs {
		if strings.TrimSpace(id) == "" || seen[id] {
			return ScheduleProposal{}, ErrInvalidAllocation{"work_item_ids", "must contain unique non-empty ids"}
		}
		seen[id] = true
	}
	return s.repo.PreviewSchedule(ctx, in)
}

func (s *service) ApplyScheduleProposal(ctx context.Context, in ApplyScheduleProposalInput) ([]Allocation, error) {
	if strings.TrimSpace(in.ProposalID) == "" {
		return nil, ErrInvalidAllocation{"proposal_id", "required"}
	}
	if strings.TrimSpace(in.IdempotencyKey) == "" {
		return nil, ErrInvalidAllocation{"idempotency_key", "required"}
	}
	if in.ExpectedRevision <= 0 {
		return nil, ErrInvalidAllocation{"expected_revision", "must be positive"}
	}
	return s.repo.ApplyScheduleProposal(ctx, in)
}

func (s *service) CarryForward(ctx context.Context, in CarryForwardInput) (Allocation, error) {
	if strings.TrimSpace(in.AllocationID) == "" {
		return Allocation{}, ErrInvalidAllocation{"allocation_id", "required"}
	}
	if _, err := time.Parse("2006-01-02", in.TargetLocalDate); err != nil {
		return Allocation{}, ErrInvalidAllocation{"target_local_date", "must be YYYY-MM-DD"}
	}
	if in.StartMinutes < 0 || in.StartMinutes >= 1440 {
		return Allocation{}, ErrInvalidAllocation{"start_minutes", "must be within the day"}
	}
	return s.repo.CarryForward(ctx, in)
}
