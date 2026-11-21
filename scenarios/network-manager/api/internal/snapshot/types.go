package snapshot

import (
	"context"
	"errors"
	"time"
)

const TimeFormat = time.RFC3339Nano

var ErrNotFound = errors.New("snapshot not found")

type Metric struct {
	Name   string
	Value  string
	Unit   string
	Status string
}

type Snapshot struct {
	ID        string
	Status    string
	Profile   string
	Summary   string
	Metrics   []Metric
	Findings  []string
	CreatedAt time.Time
}

type ProbeResult struct {
	Name    string
	Value   string
	Unit    string
	Status  string
	Finding string
}

type ProbeRunner interface {
	Run(ctx context.Context, profile string) ([]ProbeResult, error)
}
