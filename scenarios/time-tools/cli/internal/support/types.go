package support

// TimezoneConversionResponse mirrors the payload returned by
// POST /api/v1/time/convert.
type TimezoneConversionResponse struct {
	OriginalTime  string `json:"original_time"`
	ConvertedTime string `json:"converted_time"`
	FromTimezone  string `json:"from_timezone"`
	ToTimezone    string `json:"to_timezone"`
	OffsetMinutes int    `json:"offset_minutes"`
	IsDST         bool   `json:"is_dst"`
}

// DurationCalculationResponse mirrors the payload returned by
// POST /api/v1/time/duration.
type DurationCalculationResponse struct {
	StartTime       string  `json:"start_time"`
	EndTime         string  `json:"end_time"`
	TotalMinutes    int     `json:"total_minutes"`
	TotalHours      float64 `json:"total_hours"`
	TotalDays       float64 `json:"total_days"`
	BusinessMinutes int     `json:"business_minutes,omitempty"`
	BusinessHours   float64 `json:"business_hours,omitempty"`
	BusinessDays    int     `json:"business_days,omitempty"`
	CalendarDays    int     `json:"calendar_days"`
}

// TimeArithmeticResponse mirrors the payload returned by
// POST /api/v1/time/add and POST /api/v1/time/subtract.
type TimeArithmeticResponse struct {
	OriginalTime string `json:"original_time"`
	Duration     string `json:"duration"`
	ResultTime   string `json:"result_time"`
	Operation    string `json:"operation"`
}

// TimeFormatResponse mirrors the payload returned by POST /api/v1/time/format.
type TimeFormatResponse struct {
	Original  string `json:"original"`
	Formatted string `json:"formatted"`
	Format    string `json:"format"`
	Timezone  string `json:"timezone"`
}

// TimeParseResponse mirrors the payload returned by POST /api/v1/time/parse.
type TimeParseResponse struct {
	ParsedTime  string `json:"parsed_time"`
	RFC3339     string `json:"rfc3339"`
	Unix        int64  `json:"unix"`
	Timezone    string `json:"timezone"`
	IsAmbiguous bool   `json:"is_ambiguous"`
	Confidence  string `json:"confidence"`
}

// OptimalSlot is one entry in the optimal_slots array returned by
// POST /api/v1/schedule/optimal.
type OptimalSlot struct {
	StartTime        string   `json:"start_time"`
	EndTime          string   `json:"end_time"`
	Score            float64  `json:"score"`
	ConflictCount    int      `json:"conflict_count"`
	ParticipantsFree []string `json:"participants_free"`
}

// ScheduleOptimalResponse wraps the optimal_slots response.
type ScheduleOptimalResponse struct {
	OptimalSlots []OptimalSlot `json:"optimal_slots"`
}

// ConflictInfo is one entry in the conflicts array returned by
// POST /api/v1/schedule/conflicts.
type ConflictInfo struct {
	EventID        string `json:"event_id"`
	EventTitle     string `json:"event_title"`
	ConflictType   string `json:"conflict_type"`
	Severity       string `json:"severity"`
	OverlapMinutes int    `json:"overlap_minutes"`
	StartTime      string `json:"start_time"`
	EndTime        string `json:"end_time"`
}

// ConflictDetectionResponse mirrors the payload returned by
// POST /api/v1/schedule/conflicts.
type ConflictDetectionResponse struct {
	Conflicts     []ConflictInfo `json:"conflicts"`
	HasConflicts  bool           `json:"has_conflicts"`
	ConflictCount int            `json:"conflict_count"`
}

// ScheduledEvent mirrors the event shape returned by GET/POST /api/v1/events.
type ScheduledEvent struct {
	ID           string `json:"id,omitempty"`
	Title        string `json:"title,omitempty"`
	StartTime    string `json:"start_time,omitempty"`
	EndTime      string `json:"end_time,omitempty"`
	Timezone     string `json:"timezone,omitempty"`
	Status       string `json:"status,omitempty"`
	Priority     string `json:"priority,omitempty"`
	OrganizerID  string `json:"organizer_id,omitempty"`
	EventType    string `json:"event_type,omitempty"`
	Location     string `json:"location,omitempty"`
	LocationType string `json:"location_type,omitempty"`
}

// EventsListResponse wraps GET /api/v1/events.
type EventsListResponse struct {
	Events []ScheduledEvent `json:"events"`
	Count  int              `json:"count"`
}
