package fakes

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"sync"
	"time"

	"device-control/strategy"
)

// Strategy is a deterministic adapter fake used by conformance and flow
// tests. It records the exact calls while returning a valid PNG frame.
type Strategy struct {
	Declaration strategy.Declaration
	ObserveErr  error
	ActuateErr  error
	mu          sync.Mutex
	Actuations  []strategy.Actuation
}

func New(id string, status string, capabilities ...string) *Strategy {
	caps := make(map[string]strategy.Capability, len(capabilities))
	for _, name := range capabilities {
		caps[name] = strategy.Capability{Name: name, Status: status}
	}
	return &Strategy{Declaration: strategy.Declaration{StrategyID: id, Description: "deterministic test strategy", Status: status, Capabilities: caps, EvidenceClass: "release-grade", MinimumUsefulFPS: 5, Promotable: true}}
}
func (s *Strategy) ID() string                                             { return s.Declaration.StrategyID }
func (s *Strategy) Describe(context.Context) (strategy.Declaration, error) { return s.Declaration, nil }
func (s *Strategy) Observe(context.Context) (strategy.Frame, error) {
	if s.ObserveErr != nil {
		return strategy.Frame{}, s.ObserveErr
	}
	var out bytes.Buffer
	_ = png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	return strategy.Frame{Width: 2, Height: 2, Scale: 1, Timestamp: time.Now().UTC(), MediaType: "image/png", Bytes: out.Bytes()}, nil
}
func (s *Strategy) Actuate(_ context.Context, a strategy.Actuation) error {
	s.mu.Lock()
	s.Actuations = append(s.Actuations, a)
	s.mu.Unlock()
	return s.ActuateErr
}
func (s *Strategy) Calls() []strategy.Actuation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]strategy.Actuation(nil), s.Actuations...)
}
