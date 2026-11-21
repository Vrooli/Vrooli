// Package surfaces aggregates owner-provided views without owning inventory or
// granting control. Session admission remains with each destination owner.
package surfaces

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/targetmodel"
)

type Provider interface {
	List(context.Context) (ProviderResult, error)
}

type bearerContextKey struct{}

// WithBearerToken carries the already-authenticated operator token across a
// server-side provider call. It is never serialized into a surface descriptor.
func WithBearerToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, bearerContextKey{}, token)
}

func bearerToken(ctx context.Context) string {
	token, _ := ctx.Value(bearerContextKey{}).(string)
	return token
}

type ProviderResult struct {
	Surfaces []targetmodel.SurfaceDescriptor
	Partial  bool
}
type Source struct {
	Owner    string
	Provider Provider
}
type SourceStatus struct {
	Owner      string
	State      string
	ReasonCode string
}
type Snapshot struct {
	Surfaces   []targetmodel.SurfaceDescriptor
	Sources    []SourceStatus
	ObservedAt time.Time
}
type Catalog struct {
	sources []Source
	timeout time.Duration
	now     func() time.Time
	slots   chan struct{}
}

func NewCatalog(sources []Source, timeout time.Duration, now func() time.Time) (*Catalog, error) {
	if timeout <= 0 || timeout > 10*time.Second || now == nil || len(sources) > 16 {
		return nil, fmt.Errorf("catalog requires a clock, positive deadline of at most 10s, and at most 16 sources")
	}
	seen := map[string]bool{}
	for _, source := range sources {
		if source.Provider == nil || (targetmodel.TargetRef{OwnerScenario: source.Owner, ResourceID: "probe"}).Validate() != nil || seen[source.Owner] {
			return nil, fmt.Errorf("invalid or duplicate source owner")
		}
		seen[source.Owner] = true
	}
	return &Catalog{sources: append([]Source{}, sources...), timeout: timeout, now: now, slots: make(chan struct{}, 16)}, nil
}

type sourceResult struct {
	index int
	value ProviderResult
	err   error
}

func (c *Catalog) List(ctx context.Context) Snapshot {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	snapshot := Snapshot{ObservedAt: c.now(), Surfaces: []targetmodel.SurfaceDescriptor{}, Sources: make([]SourceStatus, len(c.sources))}
	results := make(chan sourceResult, len(c.sources))
	pending := map[int]bool{}
	for i, source := range c.sources {
		snapshot.Sources[i] = SourceStatus{Owner: source.Owner, State: "unavailable", ReasonCode: "source_busy"}
		select {
		case c.slots <- struct{}{}:
			pending[i] = true
			go func(index int, p Provider) {
				defer func() { <-c.slots }()
				value, err := p.List(ctx)
				results <- sourceResult{index: index, value: value, err: err}
			}(i, source.Provider)
		default: // Concurrent callers cannot create an unbounded number of workers.
		}
	}
	for len(pending) > 0 {
		select {
		case result := <-results:
			delete(pending, result.index)
			source := c.sources[result.index]
			status := SourceStatus{Owner: source.Owner, State: "ready"}
			switch {
			case result.err != nil:
				// Provider errors may contain credential-bearing URLs. Only bounded reason
				// codes cross the projection boundary.
				status.State, status.ReasonCode = "unavailable", "source_unreachable"
			case validateSource(source.Owner, result.value.Surfaces) != nil:
				status.State, status.ReasonCode = "invalid", "invalid_source_descriptor"
			default:
				if result.value.Partial {
					status.State, status.ReasonCode = "partial", "incomplete_inventory"
				}
				for _, surface := range result.value.Surfaces {
					// Copy through the shared converter to avoid mutating provider caches.
					wire, _ := surface.Proto()
					copy, _ := targetmodel.SurfaceDescriptorFromProto(wire)

					snapshot.Surfaces = append(snapshot.Surfaces, copy)
				}
			}
			snapshot.Sources[result.index] = status
		case <-ctx.Done():
			for index := range pending {
				snapshot.Sources[index] = SourceStatus{Owner: c.sources[index].Owner, State: "unavailable", ReasonCode: "source_deadline"}
			}
			pending = nil
		}
	}
	snapshot.ObservedAt = c.now()
	for i := range snapshot.Surfaces {
		for j, fact := range snapshot.Surfaces[i].Capabilities {
			if fact.EffectiveState(snapshot.ObservedAt) == targetmodel.CapabilityUnknown && fact.State != targetmodel.CapabilityUnknown {
				snapshot.Surfaces[i].Capabilities[j].State = targetmodel.CapabilityUnknown
				snapshot.Surfaces[i].Capabilities[j].ReasonCode = "observation_stale"
			}
		}
	}
	sort.Slice(snapshot.Surfaces, func(i, j int) bool {
		return surfaceKey(snapshot.Surfaces[i].Ref) < surfaceKey(snapshot.Surfaces[j].Ref)
	})
	return snapshot
}

func surfaceKey(ref targetmodel.SurfaceRef) string {
	return ref.Target.OwnerScenario + "/" + ref.Target.ResourceID + "/" + ref.OwnerScenario + "/" + ref.SurfaceID
}

func validateSource(owner string, surfaces []targetmodel.SurfaceDescriptor) error {
	if len(surfaces) > 256 {
		return fmt.Errorf("source inventory exceeds bound")
	}
	seen := map[string]bool{}
	for _, surface := range surfaces {
		if surface.Ref.OwnerScenario != owner {
			return fmt.Errorf("surface owner mismatch")
		}
		if _, err := surface.Proto(); err != nil {
			return err
		}
		key := surfaceKey(surface.Ref)
		if seen[key] {
			return fmt.Errorf("duplicate surface identity")
		}
		seen[key] = true
	}
	return nil
}

// Resolve never picks the first of several equal names. Even a unique match is
// just a reference: opening or actuating it requires owner admission.
func (s Snapshot) Resolve(label string, exact *targetmodel.SurfaceRef) ([]targetmodel.SurfaceDescriptor, error) {
	label = strings.TrimSpace(label)
	if exact != nil {
		if err := exact.Validate(); err != nil {
			return nil, err
		}
	} else if label == "" || len(label) > 256 {
		return nil, fmt.Errorf("a bounded label or exact reference is required")
	}
	matches := []targetmodel.SurfaceDescriptor{}
	for _, surface := range s.Surfaces {
		if exact != nil {
			if surface.Ref == *exact {
				matches = append(matches, surface)
			}
		} else if strings.EqualFold(surface.DisplayLabel, label) {
			matches = append(matches, surface)
		}
	}
	return matches, nil
}
