// Package invocationreadmodel owns the durable analytical projection contract.
package invocationreadmodel

import (
	"context"
	"time"

	"agent-manager/internal/runsignal"
	"agent-manager/internal/tokenaccounting"
)

// Store persists the durable analytical projection. Replace is atomic: facts
// and their source watermark either move forward together or not at all.
type Store interface {
	Replace(context.Context, []Fact, Watermark) error
	Facts(context.Context, string) ([]Fact, error)
	Watermark(context.Context, string) (*Watermark, error)
	Aggregate(context.Context, Filter, string, int) ([]AggregateRow, error)
	Cohort(context.Context, Filter, int) (Cohort, error)
	Metrics(context.Context, Filter) (Metrics, error)
	RunMetrics(context.Context, Filter) (RunMetrics, error)
	RunDurationStatistics(context.Context, Filter) (RunDurationStatistics, error)
	RunStatusCounts(context.Context, Filter) ([]RunStatusCount, error)
	RunBreakdown(context.Context, Filter, string, int) ([]RunBreakdownRow, error)
	RunTimeSeries(context.Context, Filter, time.Duration) ([]RunTimeSeriesBucket, error)
	ToolUsage(context.Context, Filter, int) ([]ToolUsageRow, error)
	ErrorPatterns(context.Context, Filter, int) ([]ErrorPattern, error)
	FindingMetrics(context.Context, Filter) (FindingMetrics, error)
}

type CanaryRun struct {
	Role       string
	Arm        string
	Status     string
	DurationMS float64
	CostUSD    float64
}

type CanaryStore interface {
	// CanaryRuns returns terminal executed runs with their immutable arm and role snapshot.
	CanaryRuns(context.Context, Filter) ([]CanaryRun, error)
}

// CohortDefinitionStore persists named, classifier-bound filters. Members are
// resolved at read time so a definition remains reproducible without freezing
// a moving run population at creation.
type CohortDefinitionStore interface {
	DefineCohort(context.Context, CohortDefinition) error
	ListCohorts(context.Context) ([]CohortDefinition, error)
	GetCohortDefinition(context.Context, string) (*CohortDefinition, error)
	DeleteCohort(context.Context, string) error
}

type CohortDefinition struct {
	Name              string    `json:"name"`
	FilterJSON        string    `json:"filterJson"`
	ClassifierVersion string    `json:"classifierVersion"`
	CreatedAt         time.Time `json:"createdAt"`
	ChangeBinding     string    `json:"changeBinding,omitempty"`
}

// RunStore is implemented by projections that also retain one terminal
// throughput summary per run. It is deliberately additive to Store so the
// invocation-fact contract remains usable by focused test doubles and older
// projection adapters during rollout.
type RunStore interface {
	ReplaceRun(context.Context, RunFact) error
}

// WorkloadStore updates only the identity columns during historical
// backfills, preserving consumption already projected for runs without
// retained source events.
type WorkloadStore interface {
	BackfillWorkload(context.Context, string, string, string, string) error
}

// ProjectionStore atomically advances both analytical projections for a run.
// Consumers may fall back to Store during rollout, but the production adapter
// implements this contract so a crash cannot leave terminal throughput facts
// ahead of, or behind, invocation facts and their watermark.
type ProjectionStore interface {
	ReplaceProjection(context.Context, []Fact, []ErrorFact, []Episode, []SelfReportSpan, Watermark, RunFact) error
	Episodes(context.Context, Filter, int) ([]Episode, error)
	EpisodesForRun(context.Context, string) ([]Episode, error)
	SelfReportSpansForRun(context.Context, string) ([]SelfReportSpan, error)
	TimeAccountingForRun(context.Context, string) (runsignal.TimeAccounting, bool, error)
}

// Filter is the single analytical predicate shared by aggregates and cohort
// selection. A zero filter is the complete retained corpus.
type Filter struct {
	From         *time.Time `json:"from,omitempty"`
	To           *time.Time `json:"to,omitempty"`
	Ownership    string     `json:"ownership,omitempty"`
	Outcome      string     `json:"outcome,omitempty"`
	Executable   string     `json:"executable,omitempty"`
	Fingerprint  string     `json:"fingerprint,omitempty"`
	ProfileID    string     `json:"profileId,omitempty"`
	RunnerType   string     `json:"runnerType,omitempty"`
	Model        string     `json:"model,omitempty"`
	WorkloadKind string     `json:"workloadKind,omitempty"`
	// ExcludedWorkloadKinds is an internal cohort boundary used by measures
	// that must not mix imported/interactive evidence with normal runs.
	ExcludedWorkloadKinds []string `json:"-"`
	WorkloadKey           string   `json:"workloadKey,omitempty"`
	ErrorCode             string   `json:"errorCode,omitempty"`
	TagPrefix             string   `json:"tagPrefix,omitempty"`
	RunStatus             string   `json:"runStatus,omitempty"`
	ToolName              string   `json:"toolName,omitempty"`
	ClassifierVersion     string   `json:"classifierVersion,omitempty"`
	EpisodePattern        string   `json:"episodePattern,omitempty"`
	EpisodeCauseScope     string   `json:"episodeCauseScope,omitempty"`
	EpisodeFingerprint    string   `json:"episodeFingerprint,omitempty"`
	SelfReportRuleID      string   `json:"selfReportRuleId,omitempty"`
	SelfReportCauseScope  string   `json:"selfReportCauseScope,omitempty"`
	TargetScenario        string   `json:"targetScenario,omitempty"`
	Operation             string   `json:"operation,omitempty"`
}

type AggregateRow struct {
	Dimension string `json:"dimension"`
	Value     string `json:"value"`
	Count     int64  `json:"count"`
}

// CapabilityUsageRow is the receipt-backed efficacy population. A receipt is
// the only evidence that a project-owned capability returned an outcome.
type CapabilityUsageRow struct {
	TargetScenario      string  `json:"targetScenario"`
	Operation           string  `json:"operation"`
	CallCount           int64   `json:"callCount"`
	SuccessCount        int64   `json:"successCount"`
	FailedCount         int64   `json:"failedCount"`
	TotalDurationMS     uint64  `json:"totalDurationMs"`
	TotalTokens         int64   `json:"totalTokens"`
	EstimatedTokenShare float64 `json:"estimatedTokenShare"`
}

type CapabilityEfficacyRow struct {
	TargetScenario     string `json:"targetScenario"`
	Operation          string `json:"operation"`
	CallCount          int64  `json:"callCount"`
	SuccessCount       int64  `json:"successCount"`
	FallbackAfterCount int64  `json:"fallbackAfterCount"`
	AbandonedCount     int64  `json:"abandonedCount"`
}
type Cohort struct {
	RunIDs      []string    `json:"runIds"`
	Rows        []CohortRun `json:"rows,omitempty"`
	Truncated   bool        `json:"truncated"`
	MatchedRuns int         `json:"matchedRuns"`
	DroppedRuns int         `json:"droppedRuns"`
}

type CohortRun struct {
	RunID               string
	TaskTitle           string
	ProfileID           string
	ProfileName         string
	Status              string
	CreatedAt           time.Time
	Model               string
	RunnerType          string
	WorkloadKey         string
	TotalTokens         int64
	TotalChargeMicroUSD *int64
	ChargeBasis         *string
	ToolCallCount       *int64
}

// Metrics is the shared SQL result used by friction measure consumers. Counts
// are retained separately so callers never hide an unknown population behind a
// percentage.
type Metrics struct {
	TotalCalls               int64   `json:"totalCalls"`
	ResolvedCalls            int64   `json:"resolvedCalls"`
	ExternalCalls            int64   `json:"externalCalls"`
	UnknownCalls             int64   `json:"unknownCalls"`
	NotACommandCalls         int64   `json:"notACommandCalls"`
	CompoundUnresolvedCalls  int64   `json:"compoundUnresolvedCalls"`
	UnparseableCalls         int64   `json:"unparseableCalls"`
	UnclassifiedCalls        int64   `json:"unclassifiedCalls"`
	FailedCalls              int64   `json:"failedCalls"`
	RetryCalls               int64   `json:"retryCalls"`
	HelpRecoveries           int64   `json:"helpRecoveries"`
	RepeatedCalls            int64   `json:"repeatedCalls"`
	LargestFingerprintBucket int64   `json:"largestFingerprintBucket"`
	ClassifiedBase           int64   `json:"classifiedBase"`
	PairedCalls              int64   `json:"pairedCalls"`
	UnpairedCalls            int64   `json:"unpairedCalls"`
	UnclassifiedCount        int64   `json:"unclassifiedCount"`
	UnclassifiedShare        float64 `json:"unclassifiedShare"`
	ExternalToolShare        float64 `json:"externalToolShare"`
	FailureRate              float64 `json:"failureRate"`
	RetryRate                float64 `json:"retryRate"`
	HelpRecoveryRate         float64 `json:"helpRecoveryRate"`
	RepeatedWorkRate         float64 `json:"repeatedWorkRate"`
}

// RunMetrics is the shared SQL result for run-level throughput measures. It
// intentionally has separate denominators for terminal success and completed
// cycle-time populations, so a partial lifecycle is never presented as a
// precise rate or duration.
type RunMetrics struct {
	TotalRuns                          int64   `json:"totalRuns"`
	TerminalRuns                       int64   `json:"terminalRuns"`
	SuccessfulRuns                     int64   `json:"successfulRuns"`
	SuccessRate                        float64 `json:"successRate"`
	CompletedDurationRuns              int64   `json:"completedDurationRuns"`
	AverageDurationMS                  float64 `json:"averageDurationMs"`
	TotalCostUSD                       float64 `json:"totalCostUsd"`
	AverageCostUSD                     float64 `json:"averageCostUsd"`
	TotalTokens                        int64   `json:"totalTokens"`
	InputTokens                        int64   `json:"inputTokens"`
	OutputTokens                       int64   `json:"outputTokens"`
	CacheReadTokens                    int64   `json:"cacheReadTokens"`
	CacheCreationTokens                int64   `json:"cacheCreationTokens"`
	InputCostUSD                       float64 `json:"inputCostUsd"`
	OutputCostUSD                      float64 `json:"outputCostUsd"`
	CacheReadCostUSD                   float64 `json:"cacheReadCostUsd"`
	CacheCreationCostUSD               float64 `json:"cacheCreationCostUsd"`
	TotalChargeMicroUSD                int64   `json:"totalChargeMicroUsd"`
	MeteredChargeMicroUSD              int64   `json:"meteredChargeMicroUsd"`
	UnpricedTokenCount                 int64   `json:"unpricedTokenCount"`
	ConsumptionPerSuccessfulCompletion float64 `json:"consumptionPerSuccessfulCompletion"`
	ReadCalls                          int64   `json:"readCalls"`
	FileRereads                        int64   `json:"fileRereads"`
	FileRereadRate                     float64 `json:"fileRereadRate"`
}

type ChargeByBasis struct {
	Basis          string
	RunCount       int64
	ChargeMicroUSD int64
	TokenCount     int64
	ChargeReason   string
}

// RunDurationStatistics retains the complete duration summary used by the
// dashboard and CSV export. Percentile values follow the underlying durable
// store's documented semantics rather than reopening the event stream.
type RunDurationStatistics struct {
	AverageDurationMS float64
	P50DurationMS     float64
	P95DurationMS     float64
	P99DurationMS     float64
	MinDurationMS     int64
	MaxDurationMS     int64
	Count             int64
}

// RunStatusCount is one explicit terminal-run status bucket. It is a table
// rather than a synthetic scalar so consumers can preserve the current
// status-distribution product without reconstructing it from raw events.
type RunStatusCount struct {
	Status string
	Count  int64
}

// RunBreakdownRow is a durable run-summary aggregate for one dimension. The
// dimension is selected by the owning typed measure (runner, model, or
// profile), never by an untyped transport switch.
type RunBreakdownRow struct {
	Key                                string
	Value                              string
	RunCount                           int64
	SuccessCount                       int64
	FailedCount                        int64
	TotalCostUSD                       float64
	TotalTokens                        int64
	TotalChargeMicroUSD                int64
	InputTokens                        int64
	OutputTokens                       int64
	CacheReadTokens                    int64
	CacheCreationTokens                int64
	AvgDurationMS                      float64
	ConsumptionPerSuccessfulCompletion float64
	CompletionRate                     float64
}

// RunTimeSeriesBucket is a terminal-time bucket. It records completed,
// failed, and cancelled runs without pretending that a terminal projection
// can authoritatively count in-progress starts.
type RunTimeSeriesBucket struct {
	Bucket               time.Time
	TerminalRuns         int64
	CompletedRuns        int64
	FailedRuns           int64
	CancelledRuns        int64
	TotalCostUSD         float64
	InputCostUSD         float64
	OutputCostUSD        float64
	CacheReadCostUSD     float64
	CacheCreationCostUSD float64
	AvgDurationMS        float64
}

// ToolUsageRow retains one tool's invocation population and classified
// outcomes, so tool analytics do not need to reopen prunable event JSON.
type ToolUsageRow struct {
	ToolName            string
	CallCount           int64
	SuccessCount        int64
	FailedCount         int64
	TotalTokens         int64
	EstimatedTokenShare float64
}

type ToolCommandRow struct {
	Executable          string
	CommandPath         string
	CallCount           int64
	SuccessCount        int64
	FailedCount         int64
	RunCount            int64
	TotalTokens         int64
	EstimatedTokenShare float64
	P50FootprintTokens  int64
	P95FootprintTokens  int64
	MaxFootprintTokens  int64
	Truncated           bool
}

// TokenAttributionRow is the typed aggregate behind the three token views.
// The factors remain separate in storage; TotalTokens is selected at query
// time from footprint, residency, or incurred cost.
type TokenAttributionRow struct {
	GroupBy             string
	Value               string
	CallCount           int64
	TotalTokens         int64
	EstimatedTokens     int64
	EstimatedTokenShare float64
	P50FootprintTokens  int64
	P95FootprintTokens  int64
	MaxFootprintTokens  int64
}

// ErrorFact retains only safe analytical error vocabulary. Human messages,
// stack traces, and opaque provider details stay in the prunable event log.
type ErrorFact struct {
	RunID      string
	EventID    string
	OccurredAt time.Time
	TimeBasis  string
	ErrorCode  string
	ProfileID  string
	RunnerType string
	Model      string
	Tag        string
}

// Episode is a durable friction signal plus the run dimensions required for
// the shared analytical filter. It retains identifiers and bounded classifier
// output only; raw event bodies stay in the prunable event log.
type Episode struct {
	runsignal.FrictionEpisode
	RunID      string
	OccurredAt time.Time
	TimeBasis  string
	ProfileID  string
	RunnerType string
	Model      string
	Tag        string
	RunStatus  string
}

// SelfReportSpan is a durable, redacted self-reported-friction signal. The
// classifier's Text field is already bounded/redacted before it reaches this
// projection.
type SelfReportSpan struct {
	runsignal.SelfReportSpan
	RunID      string
	OccurredAt time.Time
	TimeBasis  string
	ProfileID  string
	RunnerType string
	Model      string
	Tag        string
	RunStatus  string
}

type ErrorPattern struct {
	ErrorCode   string
	Count       int64
	LastSeen    time.Time
	SampleRunID string
}

// FindingMetrics measures persisted investigation evidence by finding
// creation time. Recurrence is explicit: a row is recurring only when its
// fingerprint appears more than once in the same filtered population.
type FindingMetrics struct {
	TotalFindings         int64   `json:"totalFindings"`
	RecurringFindings     int64   `json:"recurringFindings"`
	RecurringFingerprints int64   `json:"recurringFingerprints"`
	RecurrenceRate        float64 `json:"recurrenceRate"`
}

// Fact is one persisted analytical observation plus the dimensions required by
// corpus filters. Unknown is represented as an explicit value, never NULL.
type Fact struct {
	runsignal.InvocationFact
	RunID      string
	OccurredAt time.Time
	TimeBasis  string
	ProfileID  string
	RunnerType string
	Model      string
	Tag        string
	RunStatus  string
}

// RunFact is the durable terminal summary used for run-level throughput and
// cost measurements. occurred_at is terminal time when available, otherwise
// creation time, with TimeBasis making that fallback visible to consumers.
type RunFact struct {
	RunID                  string
	GoalID                 string
	GoalStatus             string
	OccurredAt             time.Time
	CreatedAt              time.Time
	StartedAt              *time.Time
	EndedAt                *time.Time
	DurationMS             int64
	Status                 string
	ProfileID              string
	RunnerType             string
	Model                  string
	Tag                    string
	WorkloadKind           string
	WorkloadKey            string
	WorkloadInstance       string
	TotalCostUSD           float64
	InputCostUSD           float64
	OutputCostUSD          float64
	CacheReadCostUSD       float64
	CacheCreationCostUSD   float64
	TotalTokens            int64
	InputTokens            int64
	OutputTokens           int64
	CacheReadTokens        int64
	CacheCreationTokens    int64
	Turns                  int64
	ToolCalls              int64
	TotalChargeMicroUSD    int64
	MeteredChargeMicroUSD  int64
	UnpricedTokenCount     int64
	PreambleInjectedTokens int64
	PreambleFixedTokens    int64
	PreambleTokenBasis     tokenaccounting.Basis
	UnattributedTokens     int64
	UnattributedReason     string
	ReadCalls              int64
	FileRereads            int64
	TimeAccounting         runsignal.TimeAccounting
	CostTimeBasis          string
	// TimeBasis identifies whether lifecycle/event time came from the source
	// transcript or was assigned while Agent Manager ingested the run.
	TimeBasis   string
	ProjectedAt time.Time
}

type Efficiency struct {
	Consumption      int64
	SuccessfulRuns   int64
	PerSuccessfulRun float64
}

// ConsumptionPerSuccessfulCompletion deliberately includes failed attempts
// in the numerator; excluding them would make unreliable cheap models look
// efficient after retries.
func ConsumptionPerSuccessfulCompletion(facts []RunFact) Efficiency {
	var result Efficiency
	for _, fact := range facts {
		result.Consumption += fact.TotalTokens
		if fact.Status == "complete" || fact.Status == "completed" || fact.Status == "success" {
			result.SuccessfulRuns++
		}
	}
	if result.SuccessfulRuns > 0 {
		result.PerSuccessfulRun = float64(result.Consumption) / float64(result.SuccessfulRuns)
	}
	return result
}

// Watermark states the newest source event incorporated for a run.
type Watermark struct {
	RunID                       string
	LastEventID                 string
	LastEventAt                 time.Time
	ClassifierVersion           string
	EpisodeClassifierVersion    string
	SelfReportClassifierVersion string
	ProjectedAt                 time.Time
}
