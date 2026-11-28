package forecasts

import "context"

const (
	AlgorithmVersion   = "deterministic-capacity-v1"
	FreshnessCurrent   = "current"
	StateNoKnownWork   = "no_known_work"
	StateFeasible      = "feasible_in_scenario"
	StateBeyondHorizon = "beyond_horizon"
	RiskUnknown        = "unknown"
	RiskOnTrack        = "on_track"
	RiskElevated       = "elevated"
)

type Forecast struct {
	GeneratedAt         string
	Freshness           string
	InputFingerprint    string
	HorizonStart        string
	HorizonEnd          string
	AlgorithmVersion    string
	CentralFinish       string
	CautiousFinish      string
	ResultState         string
	RiskState           string
	Explanation         string
	KnownWorkMinutes    int64
	AvailableMinutes    int64
	ReserveMinutes      int64
	UnresolvedWorkCount int64
	CommitmentOutlooks  []CommitmentOutlook
	SnapshotID          string
	PreviousSnapshotID  string
	ChangeExplanation   string
}

type CommitmentInput struct {
	ID               string
	Result           string
	PromisedBoundary string
}

type CommitmentOutlook struct {
	ID               string
	Result           string
	PromisedBoundary string
	ForecastFinish   string
	RiskState        string
	Explanation      string
}

type SnapshotSummary struct {
	ID                 string
	GeneratedAt        string
	InputFingerprint   string
	HorizonStart       string
	HorizonEnd         string
	CentralFinish      string
	CautiousFinish     string
	ResultState        string
	RiskState          string
	Explanation        string
	PreviousSnapshotID string
	ChangeExplanation  string
}

type Snapshot struct {
	StartDate       string
	Timezone        string
	HorizonDays     int
	KnownWork       int64
	UnresolvedWork  int64
	OccupiedMinutes int64
	Commitments     []CommitmentInput
}

type Service interface {
	Get(context.Context, string, string, int) (Forecast, error)
	List(context.Context, int) ([]SnapshotSummary, error)
}
