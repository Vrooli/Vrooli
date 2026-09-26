package monitoring

import (
	"errors"
	"time"
)

const TimeFormat = time.RFC3339Nano

var ErrNotFound = errors.New("monitoring record not found")

type Schedule struct {
	ID                   string
	Name                 string
	Profile              string
	BaselineSnapshotID   string
	IntervalMinutes      int
	Enabled              bool
	LatencyThresholdMS   int
	UnavailableThreshold int
	Effects              []string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Alert struct {
	ID         string
	ScheduleID string
	SnapshotID string
	Severity   string
	Status     string
	Summary    string
	Evidence   []string
	CreatedAt  time.Time
}

type Run struct {
	ID                 string
	ScheduleID         string
	SnapshotID         string
	Status             string
	Summary            string
	RegressionDetected bool
	Alerts             []Alert
	Effects            []string
	CreatedAt          time.Time
}
