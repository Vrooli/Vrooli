package pricing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
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

// FromRateLimitEvent translates only fields actually reported by a runner.
// Provider, pool, and provenance are supplied by the owner that knows the
// authenticated resource binding; they are never guessed from model pricing.
func FromRateLimitEvent(provider, pool, provenance string, eventAt time.Time, event *domain.RateLimitEventData) (*QuotaObservation, error) {
	if event == nil {
		return nil, errors.New("rate limit event is required")
	}
	standing := QuotaStandingExhausted
	if event.UsedPercent != nil && *event.UsedPercent < 100 {
		standing = QuotaStandingAvailable
	}
	window := strings.TrimSpace(event.LimitType)
	if event.WindowMinutes > 0 {
		window = fmt.Sprintf("%dm", event.WindowMinutes)
	}
	o := &QuotaObservation{
		ID:          uuid.NewString(),
		Provider:    strings.TrimSpace(provider),
		Pool:        strings.TrimSpace(pool),
		Window:      window,
		ObservedAt:  eventAt,
		Standing:    standing,
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
	return o, o.Validate()
}

// QuotaObservationID provides stable identity for event replays. It includes
// the source run, evidence identity, and immutable scope/time, so replaying
// one native frame is an idempotent insert while a later provider frame
// remains a new observation.
func QuotaObservationID(sourceRunID string, observation *QuotaObservation) string {
	if observation == nil {
		return ""
	}
	key := strings.Join([]string{sourceRunID, observation.Provider, observation.Pool, observation.Window, observation.ObservedAt.UTC().Format(time.RFC3339Nano), observation.Provenance, observation.EvidenceRef}, "\x00")
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
