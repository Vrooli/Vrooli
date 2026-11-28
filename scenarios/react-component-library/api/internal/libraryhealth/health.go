// Package libraryhealth is a read composition, never a source of truth.
//
// Overview composes counts, Quality reads coverage evidence, Activity reads the
// event log, and Services reads the capability registry through an injected
// provider. Every section carries its own availability: an unavailable source
// becomes StateUnavailable with a reason, not a zero that looks measured.
package libraryhealth

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const timestampLayout = "2006-01-02T15:04:05.000000000Z"

const (
	defaultActivityDays = 30
	maxActivityDays     = 365
)

// SectionState distinguishes a measured zero from unavailable data.
type SectionState string

const (
	StateAvailable   SectionState = "available"
	StateEmpty       SectionState = "empty"
	StatePartial     SectionState = "partial"
	StateUnavailable SectionState = "unavailable"
)

// Metric is one labelled measure. Total is the denominator when meaningful
// (for example pass/fail against coverage rows); zero means no denominator.
type Metric struct {
	Key    string
	Label  string
	Value  int
	Total  int
	Detail string
}

// Section is one independently available part of the health snapshot.
type Section struct {
	State       SectionState
	Reason      string
	Metrics     []Metric
	GeneratedAt string
}

// OverviewCounts are the summary numbers Stats shows above the sections. The
// fields are owned by their source domains; this struct only composes them.
type OverviewCounts struct {
	Assets       int
	Published    int
	Workspaces   int
	FindingsOpen int
	CoverageRows int
}

// Providers are the injected reads behind each section. A nil function means
// the caller did not configure that source; the service reports it as
// unavailable rather than pretending the value is zero.
type Providers struct {
	Overview func(ctx context.Context) (OverviewCounts, error)
	Quality  func(ctx context.Context) ([]Metric, error)
	Activity func(ctx context.Context, days int) ([]Metric, error)
	Services func(ctx context.Context) ([]Metric, error)
}

// Health is the complete Stats payload.
type Health struct {
	Overview    Section
	Quality     Section
	Activity    Section
	Services    Section
	Counts      OverviewCounts
	GeneratedAt string
}

// PartialError means a provider read some sources but not all. The returned
// metrics remain honest for the sources that succeeded; Unavailable names the
// keys that must not be rendered as zero.
type PartialError struct {
	Reason      string
	Unavailable []string
}

func (e *PartialError) Error() string {
	if e == nil {
		return ""
	}
	if e.Reason != "" {
		return e.Reason
	}
	return "some sources are unavailable"
}

func asPartial(err error) (*PartialError, bool) {
	var partial *PartialError
	if errors.As(err, &partial) {
		return partial, true
	}
	return nil, false
}

// Service composes the sections. It holds no cache and writes nothing.
type Service struct {
	providers Providers
	clock     func() time.Time
}

// NewService wires provider functions.
func NewService(providers Providers) *Service {
	return &Service{providers: providers, clock: time.Now}
}

// WithClock replaces the time source used for section timestamps.
func (s *Service) WithClock(clock func() time.Time) *Service {
	if clock != nil {
		s.clock = clock
	}
	return s
}

// Get returns every section. Provider failures stay inside their section; the
// caller always receives a renderable snapshot.
func (s *Service) Get(ctx context.Context, activityDays int) Health {
	now := s.now()
	generatedAt := now.Format(timestampLayout)
	days := clampActivityDays(activityDays)

	overview, counts := s.overview(ctx, generatedAt)
	health := Health{
		Overview:    overview,
		Quality:     s.metricsSection(ctx, generatedAt, "quality", s.providers.Quality),
		Activity:    s.activitySection(ctx, days, generatedAt),
		Services:    s.metricsSection(ctx, generatedAt, "services", s.providers.Services),
		Counts:      counts,
		GeneratedAt: generatedAt,
	}
	return health
}

func (s *Service) overview(ctx context.Context, generatedAt string) (Section, OverviewCounts) {
	if s.providers.Overview == nil {
		return Section{
			State:       StateUnavailable,
			Reason:      "overview provider is not configured",
			GeneratedAt: generatedAt,
		}, OverviewCounts{}
	}
	counts, err := s.providers.Overview(ctx)
	metrics := overviewMetrics(counts)
	if err == nil {
		return Section{State: stateForMetrics(metrics), Metrics: metrics, GeneratedAt: generatedAt}, counts
	}
	if partial, ok := asPartial(err); ok {
		return Section{
			State:       StatePartial,
			Reason:      partial.Error(),
			Metrics:     filterUnavailable(metrics, partial.Unavailable),
			GeneratedAt: generatedAt,
		}, counts
	}
	// A non-partial error invalidates every count in this section. Do not leak
	// the provider's zero values as measured data.
	return Section{
		State:       StateUnavailable,
		Reason:      errorReason(err, "overview"),
		GeneratedAt: generatedAt,
	}, OverviewCounts{}
}

func (s *Service) metricsSection(ctx context.Context, generatedAt, name string, provider func(context.Context) ([]Metric, error)) Section {
	if provider == nil {
		return Section{
			State:       StateUnavailable,
			Reason:      name + " provider is not configured",
			GeneratedAt: generatedAt,
		}
	}
	metrics, err := provider(ctx)
	return buildSection(generatedAt, metrics, err, name)
}

func (s *Service) activitySection(ctx context.Context, days int, generatedAt string) Section {
	if s.providers.Activity == nil {
		return Section{
			State:       StateUnavailable,
			Reason:      "activity provider is not configured",
			GeneratedAt: generatedAt,
		}
	}
	metrics, err := s.providers.Activity(ctx, days)
	return buildSection(generatedAt, metrics, err, "activity")
}

func buildSection(generatedAt string, metrics []Metric, err error, name string) Section {
	if err == nil {
		return Section{State: stateForMetrics(metrics), Metrics: metrics, GeneratedAt: generatedAt}
	}
	if partial, ok := asPartial(err); ok {
		return Section{
			State:       StatePartial,
			Reason:      partial.Error(),
			Metrics:     filterUnavailable(metrics, partial.Unavailable),
			GeneratedAt: generatedAt,
		}
	}
	return Section{
		State:       StateUnavailable,
		Reason:      errorReason(err, name),
		GeneratedAt: generatedAt,
	}
}

func stateForMetrics(metrics []Metric) SectionState {
	if len(metrics) == 0 {
		return StateEmpty
	}
	for _, metric := range metrics {
		if metric.Value != 0 {
			return StateAvailable
		}
	}
	return StateEmpty
}

func filterUnavailable(metrics []Metric, unavailable []string) []Metric {
	if len(unavailable) == 0 {
		return metrics
	}
	skip := make(map[string]bool, len(unavailable))
	for _, key := range unavailable {
		skip[key] = true
	}
	out := make([]Metric, 0, len(metrics))
	for _, metric := range metrics {
		if !skip[metric.Key] {
			out = append(out, metric)
		}
	}
	return out
}

func overviewMetrics(counts OverviewCounts) []Metric {
	return []Metric{
		{Key: "assets", Label: "Assets", Value: counts.Assets},
		{Key: "published", Label: "Published", Value: counts.Published},
		{Key: "workspaces", Label: "Workspaces", Value: counts.Workspaces},
		{Key: "findings_open", Label: "Open findings", Value: counts.FindingsOpen},
		{Key: "coverage_rows", Label: "Coverage rows", Value: counts.CoverageRows},
	}
}

func errorReason(err error, name string) string {
	if err == nil {
		return ""
	}
	if message := err.Error(); message != "" {
		return message
	}
	return fmt.Sprintf("%s source is unavailable", name)
}

func (s *Service) now() time.Time {
	if s.clock != nil {
		return s.clock().UTC()
	}
	return time.Now().UTC()
}

func clampActivityDays(days int) int {
	if days <= 0 {
		return defaultActivityDays
	}
	if days > maxActivityDays {
		return maxActivityDays
	}
	return days
}
