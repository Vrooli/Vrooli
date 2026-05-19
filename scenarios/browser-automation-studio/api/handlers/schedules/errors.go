package schedules

import "errors"

var (
	errInvalidScheduleID = errors.New("invalid schedule id")
	errInvalidWorkflowID = errors.New("invalid workflow id")
	errScheduleNotFound  = errors.New("schedule not found")
	errWorkflowNotFound  = errors.New("workflow not found")
	errNameRequired      = errors.New("name is required")
	errCronRequired      = errors.New("cron_expression is required")
	errInvalidCron       = errors.New("invalid cron expression")
	errInvalidTimezone   = errors.New("invalid timezone")
	errNameTooLong       = errors.New("name exceeds maximum length")
	errCronTooLong       = errors.New("cron_expression exceeds maximum length")
	errTimezoneTooLong   = errors.New("timezone exceeds maximum length")
	errRangeRequired     = errors.New("start and end are required")
	errRangeInverted     = errors.New("end must be after start")
	errRangeTooLarge     = errors.New("date range cannot exceed 1 year")
)
