package main

import (
	"time"

	"github.com/google/uuid"
)

// Models for time-tools API

type TimezoneConversionRequest struct {
	Time         string `json:"time"`
	FromTimezone string `json:"from_timezone"`
	ToTimezone   string `json:"to_timezone"`
}

type TimezoneConversionResponse struct {
	OriginalTime   string `json:"original_time"`
	ConvertedTime  string `json:"converted_time"`
	FromTimezone   string `json:"from_timezone"`
	ToTimezone     string `json:"to_timezone"`
	OffsetMinutes  int    `json:"offset_minutes"`
	IsDST          bool   `json:"is_dst"`
}

type DurationCalculationRequest struct {
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	Timezone         string `json:"timezone,omitempty"`
	ExcludeWeekends  bool   `json:"exclude_weekends"`
	ExcludeHolidays  bool   `json:"exclude_holidays"`
	BusinessHoursOnly bool  `json:"business_hours_only"`
	EntityID         string `json:"entity_id,omitempty"`
}

type DurationCalculationResponse struct {
	StartTime        string  `json:"start_time"`
	EndTime          string  `json:"end_time"`
	TotalMinutes     int     `json:"total_minutes"`
	TotalHours       float64 `json:"total_hours"`
	TotalDays        float64 `json:"total_days"`
	BusinessMinutes  int     `json:"business_minutes,omitempty"`
	BusinessHours    float64 `json:"business_hours,omitempty"`
	BusinessDays     int     `json:"business_days,omitempty"`
	CalendarDays     int     `json:"calendar_days"`
}

type TimeArithmeticRequest struct {
	Time     string `json:"time"`
	Duration string `json:"duration"`
	Unit     string `json:"unit,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type TimeArithmeticResponse struct {
	OriginalTime string `json:"original_time"`
	Duration     string `json:"duration"`
	ResultTime   string `json:"result_time"`
	Operation    string `json:"operation"`
}

type TimeParseRequest struct {
	Input    string `json:"input"`
	Timezone string `json:"timezone,omitempty"`
	Format   string `json:"format,omitempty"`
}

type TimeParseResponse struct {
	ParsedTime   string `json:"parsed_time"`
	RFC3339      string `json:"rfc3339"`
	Unix         int64  `json:"unix"`
	Timezone     string `json:"timezone"`
	IsAmbiguous  bool   `json:"is_ambiguous"`
	Confidence   string `json:"confidence"`
}

type ScheduleOptimizationRequest struct {
	Participants     []string  `json:"participants"`
	DurationMinutes  int       `json:"duration_minutes"`
	PreferredTimes   []string  `json:"preferred_times,omitempty"`
	Timezone         string    `json:"timezone"`
	EarliestDate     string    `json:"earliest_date"`
	LatestDate       string    `json:"latest_date"`
	BusinessHoursOnly bool     `json:"business_hours_only"`
}

type OptimalSlot struct {
	StartTime       string   `json:"start_time"`
	EndTime         string   `json:"end_time"`
	Score           float64  `json:"score"`
	ConflictCount   int      `json:"conflict_count"`
	ParticipantsFree []string `json:"participants_free"`
}

type ScheduledEvent struct {
	ID              uuid.UUID              `json:"id"`
	Title           string                 `json:"title"`
	Description     *string                `json:"description,omitempty"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	DurationMinutes int                    `json:"duration_minutes"`
	Timezone        string                 `json:"timezone"`
	AllDay          bool                   `json:"all_day"`
	EventType       string                 `json:"event_type,omitempty"`
	Status          string                 `json:"status"`
	Priority        string                 `json:"priority"`
	OrganizerID     string                 `json:"organizer_id,omitempty"`
	Participants    []map[string]interface{} `json:"participants"`
	Location        *string                `json:"location,omitempty"`
	LocationType    *string                `json:"location_type,omitempty"`
	Tags            []string               `json:"tags"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type ConflictDetectionRequest struct {
	EventID      string `json:"event_id,omitempty"`
	OrganizerID  string `json:"organizer_id"`
	StartTime    string `json:"start_time"`
	EndTime      string `json:"end_time"`
	Participants []string `json:"participants,omitempty"`
}

type ConflictInfo struct {
	EventID        uuid.UUID `json:"event_id"`
	EventTitle     string    `json:"event_title"`
	ConflictType   string    `json:"conflict_type"`
	Severity       string    `json:"severity"`
	OverlapMinutes int       `json:"overlap_minutes"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
}