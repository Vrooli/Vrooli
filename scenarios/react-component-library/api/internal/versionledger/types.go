package versionledger

import "time"

type VersionLedger struct {
	LibraryID       string
	Version         string
	CreatedAt       time.Time
	ReleasedAt      time.Time
	RetiredAt       time.Time
	LifecycleState  string
	GatePassCount   int
	GateFailCount   int
	TestRuns        int
	TestPassRate    float64
	AdoptionCurrent int
	AdoptionPeak    int
	FileCount       int
	LinesOfCode     int
	DependencyCount int
}
