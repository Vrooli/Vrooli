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

// Read concurrently queries every configured peer with an independent
// deadline. A timeout or malformed peer affects only that peer's result.
func Read(ctx context.Context, endpoints []Endpoint, timeout time.Duration) []Result {
	if timeout <= 0 {
		timeout = 10 * time.Second
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
