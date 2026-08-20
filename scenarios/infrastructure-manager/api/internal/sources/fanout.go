// Package sources is the transport-only fan-out boundary for peer reliability
// readers. It carries observations and availability without deciding trust,
// band, ranking, or actuation semantics; those decisions remain in domains.
package sources

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Observation struct {
	ID         string
	CellRef    string
	Value      float64
	Unit       string
	Source     string
	ObservedAt time.Time
	TrustHints TrustHints
}

// TrustHints carries raw source facts across the transport boundary. The
// condition domain, not a source client, turns these facts into a verdict.
type TrustHints struct {
	Ghost       bool
	Saturated   bool
	Shelved     bool
	UnitMatches bool
	Untrusted   bool
	// UntrustedReason names the signal that could not be qualified. A bare
	// UNTRUSTED verdict is not actionable, and an untrusted reading routes to
	// the instrument's owner rather than becoming plant work.
	UntrustedReason string
	// OutOfScope marks a reading whose target exists but sits outside the
	// derived should-be-supervised set. It is carried for supervision
	// reporting and never lowers trust.
	OutOfScope bool
}

type Reader interface {
	Read(context.Context) ([]Observation, error)
}

type ReaderFunc func(context.Context) ([]Observation, error)

func (f ReaderFunc) Read(ctx context.Context) ([]Observation, error) { return f(ctx) }

type Endpoint struct {
	ID     string
	Reader Reader
}

type Result struct {
	ID           string
	Observations []Observation
	Available    bool
	Reason       string
	CheckedAt    time.Time
}

// DefaultTimeout is the per-source read deadline. It is the trust model's
// documented `readDeadline = 10s`: a slower source is an honest UNAVAILABLE,
// not a hang, and a deadline tight enough to make healthy sources
// intermittently vanish manufactures exactly the false blindness the trust
// axis exists to prevent.
const DefaultTimeout = 10 * time.Second

// TypedResult carries one peer's typed payload with the same availability
// semantics as Result.
type TypedResult[T any] struct {
	ID        string
	Value     T
	Available bool
	Reason    string
	CheckedAt time.Time
}

// ReadTyped queries one peer for a typed payload under the same per-source
// deadline as Read, and reports unavailability the same way.
//
// It exists beside Read rather than replacing it because the device graph and
// the capability grid are structured payloads, not float observations. Forcing
// a rung ladder or a resolution status through Observation.Value would mean
// grading a rung against a numeric bar — the exact unit merge the substrate
// space document names as the reason the SB9-SB13 join is not free.
func ReadTyped[T any](ctx context.Context, id string, read func(context.Context) (T, error), timeout time.Duration) TypedResult[T] {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	result := TypedResult[T]{ID: id, CheckedAt: time.Now().UTC()}
	if read == nil {
		result.Reason = "source reader is not configured"
		return result
	}
	peerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	value, err := read(peerCtx)
	if err != nil {
		if peerCtx.Err() != nil {
			result.Reason = fmt.Sprintf("source deadline exceeded: %v", peerCtx.Err())
		} else {
			result.Reason = err.Error()
		}
		return result
	}
	result.Available = true
	result.Value = value
	return result
}

// Read concurrently queries every configured peer with an independent
// deadline. A timeout or malformed peer affects only that peer's result.
func Read(ctx context.Context, endpoints []Endpoint, timeout time.Duration) []Result {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	results := make([]Result, len(endpoints))
	var wait sync.WaitGroup
	for index, endpoint := range endpoints {
		wait.Add(1)
		go func(index int, endpoint Endpoint) {
			defer wait.Done()
			checkedAt := time.Now().UTC()
			result := Result{ID: endpoint.ID, CheckedAt: checkedAt}
			if endpoint.Reader == nil {
				result.Reason = "source reader is not configured"
				results[index] = result
				return
			}
			peerCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			observations, err := endpoint.Reader.Read(peerCtx)
			if err != nil {
				if peerCtx.Err() != nil {
					result.Reason = fmt.Sprintf("source deadline exceeded: %v", peerCtx.Err())
				} else {
					result.Reason = err.Error()
				}
				results[index] = result
				return
			}
			result.Available = true
			result.Observations = observations
			results[index] = result
		}(index, endpoint)
	}
	wait.Wait()
	return results
}
