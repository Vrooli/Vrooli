package workspace

import (
	"fmt"
	"time"
)

const profileID = "default"

type Profile struct {
	ID                   string
	Timezone             string
	WeekStart            string
	DailyCapacityMinutes int
	ReserveMinutes       int
	FocusSessionMinutes  int
	Revision             int64
	UpdatedAt            time.Time
}

type UpdateInput struct {
	Timezone             string
	WeekStart            string
	DailyCapacityMinutes int
	ReserveMinutes       int
	FocusSessionMinutes  int
	ExpectedRevision     int64
}

type ErrInvalidProfile struct{ Field, Reason string }

func (e ErrInvalidProfile) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrRevisionConflict struct{}

func (ErrRevisionConflict) Error() string { return "planning profile changed; reload before updating" }

type AvailabilityWindow struct {
	ID, Timezone, EffectiveStartDate, EffectiveEndDate string
	Weekday, StartMinute, EndMinute, Priority          int
	Revision                                           int64
}

type AvailabilityException struct {
	ID, Date, Kind, Reason string
	StartMinute, EndMinute int
	Revision               int64
}

type Availability struct {
	Windows    []AvailabilityWindow
	Exceptions []AvailabilityException
	Revision   int64
}

type AvailabilityInput struct {
	Windows          []AvailabilityWindow
	Exceptions       []AvailabilityException
	ExpectedRevision int64
}

type ErrInvalidAvailability struct{ Field, Reason string }

func (e ErrInvalidAvailability) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }
