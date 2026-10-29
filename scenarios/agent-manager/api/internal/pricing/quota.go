package pricing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/domain"
)

// QuotaStanding is the provider's observed standing for one pool and window.
// It intentionally does not describe the amount of token or dollar usage.
type QuotaStanding string

const (
	QuotaStandingUnknown   QuotaStanding = "unknown"
	QuotaStandingAvailable QuotaStanding = "available"
	QuotaStandingLimited   QuotaStanding = "limited"
	QuotaStandingExhausted QuotaStanding = "exhausted"
)

// ObservationFreshness is evaluated against the caller's clock. Missing and
// unknown are distinct: missing means no observation exists, while unknown
// means an observation exists but does not establish a provider standing.
type ObservationFreshness string

const (
	ObservationFresh   ObservationFreshness = "fresh"
	ObservationStale   ObservationFreshness = "stale"
	ObservationMissing ObservationFreshness = "missing"
	ObservationUnknown ObservationFreshness = "unknown"
)

// QuotaObservation is an immutable provider fact. Used, Limit and Remaining
// are optional because many native CLIs only report that a limit was reached.
// No value is derived from token usage, pricing, or subscription dollars.
type QuotaObservation struct {
	ID            string               `json:"id"`
	Provider      string               `json:"provider"`
	Pool          string               `json:"pool"`
	Window        string               `json:"window"`
	ObservedAt    time.Time            `json:"observedAt"`
	Standing      QuotaStanding        `json:"standing"`
	Used          *int64               `json:"used,omitempty"`
	Limit         *int64               `json:"limit,omitempty"`
	Remaining     *int64               `json:"remaining,omitempty"`
	ResetAt       *time.Time           `json:"resetAt,omitempty"`
	Provenance    string               `json:"provenance"`
	Uncertainty   string               `json:"uncertainty,omitempty"`
	EvidenceRef   string               `json:"evidenceRef,omitempty"`
	SourceRunID   string               `json:"sourceRunId,omitempty"`
	Freshness     ObservationFreshness `json:"freshness,omitempty"`
	UsedPercent   *float64             `json:"usedPercent,omitempty"`
	WindowMinutes *int64               `json:"windowMinutes,omitempty"`
}

// QuotaObservationRepository is kept separate from pricing.Repository so
// existing pricing doubles do not silently become quota stores. Production
// repositories may implement this optional owner surface.
type QuotaObservationRepository interface {
	RecordQuotaObservation(context.Context, *QuotaObservation) error
	ListQuotaObservations(context.Context, string, string, string, int) ([]QuotaObservation, error)
}

// QuotaHeartbeat bounds how long an unchanged reading goes unrecorded. The
// store keeps a reading when it changes the window state (scope, reset time,
// used percent, standing) and otherwise at most once per heartbeat: Codex
// repeats the same reading on every token_count frame (SPEC A3 "deduped").
const QuotaHeartbeat = 5 * time.Minute

// QuotaCompaction reports one pass that removed repeated readings.
type QuotaCompaction struct {
	Scanned int64
	Deleted int64
}

// QuotaObservationCompactor is the optional owner surface that removes
// readings stored before write-time deduplication, preserving every reading
// span (first and last time of each distinct reading).
type QuotaObservationCompactor interface {
	CompactQuotaObservations(context.Context) (QuotaCompaction, error)
}

// QuotaReadingSpan summarizes repeated identical provider readings: one scope,
// reset minute, percent and standing, with the first and last time it was
// observed. Forecasts read spans instead of raw frames, so the read is bounded
// by distinct readings rather than by frame volume (Codex repeats the same
// reading on every token_count frame).
type QuotaReadingSpan struct {
	Provider      string
	Pool          string
	Window        string
	WindowMinutes *int64
	// ResetAt is truncated to the minute; providers jitter it by seconds.
	ResetAt     *time.Time
	UsedPercent *float64
	Standing    QuotaStanding
	FirstAt     time.Time
	LastAt      time.Time
	Samples     int64
}

// QuotaReadingSource reads reading spans observed at or after since, oldest
// first. An empty provider reads every provider.
type QuotaReadingSource interface {
	QuotaReadingSpans(ctx context.Context, provider string, since time.Time) ([]QuotaReadingSpan, error)
}

func (o *QuotaObservation) Validate() error {
	if o == nil {
		return errors.New("quota observation is required")
	}
	if strings.TrimSpace(o.Provider) == "" || strings.TrimSpace(o.Pool) == "" || strings.TrimSpace(o.Window) == "" {
		return errors.New("quota provider, pool, and window are required")
	}
	if o.ObservedAt.IsZero() {
		return errors.New("quota observation time is required")
	}
	if strings.TrimSpace(o.Provenance) == "" {
		return errors.New("quota observation provenance is required")
	}
	switch o.Standing {
	case QuotaStandingUnknown, QuotaStandingAvailable, QuotaStandingLimited, QuotaStandingExhausted:
	default:
		return fmt.Errorf("invalid quota standing %q", o.Standing)
	}
	switch o.Freshness {
	case "", ObservationFresh, ObservationStale, ObservationMissing, ObservationUnknown:
	default:
		return fmt.Errorf("invalid quota freshness %q", o.Freshness)
	}
	for name, value := range map[string]*int64{"used": o.Used, "limit": o.Limit, "remaining": o.Remaining} {
		if value != nil && *value < 0 {
			return fmt.Errorf("quota %s cannot be negative", name)
		}
	}
	if o.UsedPercent != nil && (*o.UsedPercent < 0 || *o.UsedPercent > 100) {
		return errors.New("quota used percentage must be between 0 and 100")
	}
	if o.WindowMinutes != nil && *o.WindowMinutes <= 0 {
		return errors.New("quota window minutes must be positive")
	}
	return nil
}

// Freshness returns the state of an observation without modifying it. A
// future timestamp is unknown because local clock skew cannot establish that
// the provider fact is current.
func Freshness(o *QuotaObservation, now time.Time, maxAge time.Duration) ObservationFreshness {
	if o == nil {
		return ObservationMissing
	}
	if o.ObservedAt.IsZero() || now.IsZero() || maxAge <= 0 || o.ObservedAt.After(now) {
		return ObservationUnknown
	}
	if now.Sub(o.ObservedAt) > maxAge {
		return ObservationStale
	}
	return ObservationFresh
}

// QuotaFreshnessMaxAge bounds how long a provider reading establishes the
// current standing. Older readings are stale (SPEC T-4: unknown after 6h).
const QuotaFreshnessMaxAge = 6 * time.Hour

// DefaultQuotaEventProvenance labels a quota event whose runner did not name
// its own provenance.
const DefaultQuotaEventProvenance = "agent-manager:run-event.quota"

// ObservationFromEvent is the one translation from a persisted quota event to
// a quota observation. Every capture path (live, interactive, recovery and
// import sinks, and the run_events back-fill) calls it, so one provider frame
// reached through two paths has one identity. It returns nil, nil for an
// event that is not an attributable quota observation.
func ObservationFromEvent(evt *domain.RunEvent, now time.Time) (*QuotaObservation, error) {
	if evt == nil || evt.EventType != domain.EventTypeQuota {
		return nil, nil
	}
	data, ok := evt.Data.(*domain.RateLimitEventData)
	if !ok || strings.TrimSpace(data.Provider) == "" || strings.TrimSpace(data.Pool) == "" {
		return nil, nil
	}
	provenance := strings.TrimSpace(data.Provenance)
	if provenance == "" {
		provenance = DefaultQuotaEventProvenance
	}
	observation, err := FromRateLimitEvent(data.Provider, data.Pool, provenance, evt.Timestamp.UTC(), data)
	if err != nil {
		return nil, err
	}
	observation.SourceRunID = evt.RunID.String()
	observation.EvidenceRef = "agent-manager://runs/" + evt.RunID.String() + "/events/" + evt.ID.String()
	observation.Freshness = Freshness(observation, now, QuotaFreshnessMaxAge)
	return observation, nil
}

// FromRateLimitEvent translates only fields actually reported by a runner.
// Provider, pool, and provenance are supplied by the owner that knows the
// authenticated resource binding; they are never guessed from model pricing.
func FromRateLimitEvent(provider, pool, provenance string, eventAt time.Time, event *domain.RateLimitEventData) (*QuotaObservation, error) {
	if event == nil {
		return nil, errors.New("rate limit event is required")
	}
	window := strings.TrimSpace(event.LimitType)
	if event.WindowMinutes > 0 {
		window = fmt.Sprintf("%dm", event.WindowMinutes)
	}
	o := &QuotaObservation{
		Provider:    strings.TrimSpace(provider),
		Pool:        strings.TrimSpace(pool),
		Window:      window,
		ObservedAt:  eventAt,
		Standing:    standingOf(event),
		Provenance:  strings.TrimSpace(provenance),
		Uncertainty: "account scope unavailable; do not combine across credentials; consumed amount and ceiling are unavailable unless explicitly reported",
	}
	if event.CurrentUsed > 0 {
		v := int64(event.CurrentUsed)
		o.Used = &v
	}
	if event.Limit > 0 {
		v := int64(event.Limit)
		o.Limit = &v
	}
	if event.UsedPercent != nil {
		v := *event.UsedPercent
		o.UsedPercent = &v
	}
	if event.WindowMinutes > 0 {
		v := int64(event.WindowMinutes)
		o.WindowMinutes = &v
	}
	if event.ResetTime != nil {
		v := event.ResetTime.UTC()
		o.ResetAt = &v
	}
	o.ID = QuotaObservationID(o)
	return o, o.Validate()
}

// standingOf prefers the provider's verbatim standing, then its percentage.
// A quota event with neither is a rate-limit hit, so it is exhausted.
func standingOf(event *domain.RateLimitEventData) QuotaStanding {
	switch strings.ToLower(strings.TrimSpace(event.Status)) {
	case "rejected", "exhausted":
		return QuotaStandingExhausted
	case "allowed_warning", "limited":
		return QuotaStandingLimited
	case "allowed", "available":
		return QuotaStandingAvailable
	}
	if event.UsedPercent != nil && *event.UsedPercent < 100 {
		return QuotaStandingAvailable
	}
	return QuotaStandingExhausted
}

// QuotaObservationID identifies one provider reading: its scope, time,
// provenance and the reported values. It excludes the run and event that
// carried the reading, so the same rollout frame reached through a live tail
// and an import (or a replay) is one observation, while a later frame is new.
func QuotaObservationID(observation *QuotaObservation) string {
	if observation == nil {
		return ""
	}
	percent, reset := "", ""
	if observation.UsedPercent != nil {
		percent = strconv.FormatFloat(*observation.UsedPercent, 'g', -1, 64)
	}
	if observation.ResetAt != nil {
		reset = observation.ResetAt.UTC().Format(time.RFC3339)
	}
	key := strings.Join([]string{observation.Provider, observation.Pool, observation.Window, observation.ObservedAt.UTC().Format(time.RFC3339Nano), observation.Provenance, string(observation.Standing), percent, reset}, "\x00")
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
