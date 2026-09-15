// Package integrationstatus owns one truthful projection of external
// dependencies and the workflow-start preflight derived from it.
package integrationstatus

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/stringsx"
	"swarm-manager/internal/transitions"
)

type Availability string

const (
	Available    Availability = "available"
	Unavailable  Availability = "unavailable"
	Stale        Availability = "stale"
	Unconfigured Availability = "unconfigured"
)

type Status struct {
	ID                  string       `json:"id"`
	Required            bool         `json:"required"`
	Configured          bool         `json:"configured"`
	Availability        Availability `json:"availability"`
	CheckedAt           time.Time    `json:"checkedAt"`
	FreshUntil          time.Time    `json:"freshUntil"`
	DegradedBehavior    string       `json:"degradedBehavior"`
	Diagnostic          string       `json:"diagnostic,omitempty"`
	AffectedTransitions []string     `json:"affectedTransitions"`
}

func (s Status) ReadyAt(now time.Time) bool {
	return s.Configured && s.Availability == Available && (s.FreshUntil.IsZero() || now.Before(s.FreshUntil) || now.Equal(s.FreshUntil))
}

// Checker owns the integration-specific transport call. The provider owns
// normalization and never lets callers independently infer health.
type Checker interface {
	Check(context.Context) (Status, error)
}

type Provider struct {
	checkers            map[string]Checker
	affectedTransitions map[string][]string
	now                 func() time.Time
}

func New(checkers map[string]Checker) *Provider {
	return NewWithClock(checkers, time.Now)
}

func NewWithClock(checkers map[string]Checker, now func() time.Time) *Provider {
	copy := make(map[string]Checker, len(checkers))
	for id, checker := range checkers {
		copy[strings.TrimSpace(id)] = checker
	}
	if now == nil {
		now = time.Now
	}
	return &Provider{checkers: copy, affectedTransitions: make(map[string][]string), now: now}
}

// SetTransitionRegistry supplies the read-only dependency-to-transition
// projection. The registry remains the source of truth; callers never keep a
// second list of affected workflows beside it.
func (p *Provider) SetTransitionRegistry(registry transitions.Registry) {
	affected := make(map[string][]string, len(p.checkers))
	for _, definition := range registry.Definitions() {
		for _, requirement := range definition.Requires {
			affected[requirement] = append(affected[requirement], definition.Key)
		}
	}
	for integration, keys := range affected {
		sort.Strings(keys)
		affected[integration] = keys
	}
	p.affectedTransitions = affected
}

func (p *Provider) Statuses(ctx context.Context) ([]Status, error) {
	ids := make([]string, 0, len(p.checkers))
	for id := range p.checkers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	// Each check is an independent probe of a different integration, so the
	// whole projection costs one probe rather than the sum of all of them.
	// Results stay in sorted id order regardless of which probe answers first.
	statuses := make([]Status, len(ids))
	errs := make([]error, len(ids))
	var wg sync.WaitGroup
	for index, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := p.checkers[id].Check(ctx)
			if err != nil {
				errs[index] = fmt.Errorf("check integration %q: %w", id, err)
				return
			}
			status.ID = id
			status.AffectedTransitions = append([]string(nil), p.affectedTransitions[id]...)
			if strings.TrimSpace(status.DegradedBehavior) == "" {
				errs[index] = fmt.Errorf("check integration %q: degraded behavior is required", id)
				return
			}
			statuses[index] = status
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return statuses, nil
}

// Preflight is the sole policy decision used before an affected transition
// starts. An unknown, stale, unavailable, or unconfigured requirement fails
// closed and names the integration that prevented the start.
func (p *Provider) Preflight(ctx context.Context, definition transitions.Definition) error {
	for _, requirement := range definition.Requires {
		checker, exists := p.checkers[requirement]
		if !exists {
			return fmt.Errorf("transition %q requires unknown integration %q", definition.Key, requirement)
		}
		status, err := checker.Check(ctx)
		if err != nil {
			return fmt.Errorf("transition %q preflight %q: %w", definition.Key, requirement, err)
		}
		if !status.ReadyAt(p.now()) {
			return fmt.Errorf("transition %q blocked by %q: %s", definition.Key, requirement, stringsx.FirstNonEmpty(status.Diagnostic, string(status.Availability), "unavailable"))
		}
	}
	return nil
}
