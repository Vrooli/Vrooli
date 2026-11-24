// Package harness describes clients that can operate through an already-owned
// BAS browser session. Harness adapters are deliberately not session owners.
package harness

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// BorrowedSession is the entire authority granted to a harness adapter. It
// cannot create, attach, or close sessions, nor can it obtain a driver client.
type BorrowedSession interface {
	Observe(context.Context, ObserveRequest) (Observation, error)
	Act(context.Context, ActRequest) (ActionResult, error)
}

type ObserveRequest struct {
	Mode    string          `json:"mode"`
	Query   string          `json:"query,omitempty"`
	Options json.RawMessage `json:"options,omitempty"`
}

type Observation struct {
	Mode    string          `json:"mode"`
	Data    json.RawMessage `json:"data,omitempty"`
	TraceID string          `json:"trace_id,omitempty"`
}

type ActRequest struct {
	Action string          `json:"action"`
	Target string          `json:"target,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
}

type ActionResult struct {
	Accepted bool            `json:"accepted"`
	Data     json.RawMessage `json:"data,omitempty"`
	TraceID  string          `json:"trace_id,omitempty"`
}

type Descriptor struct {
	ID               string   `json:"id"`
	Version          string   `json:"version"`
	Summary          string   `json:"summary"`
	ObservationModes []string `json:"observation_modes"`
	ActionVocabulary []string `json:"action_vocabulary"`
	EffectClasses    []string `json:"effect_classes"`
	ReplayFidelity   string   `json:"replay_fidelity"`
	InferenceRoles   []string `json:"inference_roles"`
	Upstream         string   `json:"upstream,omitempty"`
	PinnedVersion    string   `json:"pinned_version,omitempty"`
}

type Outcome string

const (
	OutcomeQualified   Outcome = "qualified"
	OutcomeUnsupported Outcome = "unsupported"
	OutcomeFailed      Outcome = "failed"
)

type Qualification struct {
	AdapterID string   `json:"adapter_id"`
	Outcome   Outcome  `json:"outcome"`
	CheckedAt string   `json:"checked_at,omitempty"`
	Evidence  []string `json:"evidence,omitempty"`
	Limits    []string `json:"limits,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// Probe receives only a borrowed session. A nil probe means the adapter is
// catalogued but not integrated in this build; it must report unsupported.
type Probe func(context.Context, BorrowedSession) Qualification

type Entry struct {
	Descriptor Descriptor
	Probe      Probe
}

type Registry struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

func NewRegistry(entries ...Entry) (*Registry, error) {
	r := &Registry{entries: make(map[string]Entry, len(entries))}
	for _, entry := range entries {
		if err := r.Register(entry); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Register(entry Entry) error {
	if r == nil {
		return errors.New("harness registry is nil")
	}
	d := entry.Descriptor
	d.ID = strings.TrimSpace(d.ID)
	if d.ID == "" || strings.TrimSpace(d.Version) == "" || strings.TrimSpace(d.ReplayFidelity) == "" {
		return errors.New("harness descriptor requires id, version, and replay fidelity")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[d.ID]; exists {
		return fmt.Errorf("harness %q already registered", d.ID)
	}
	d.ObservationModes = clean(d.ObservationModes)
	d.ActionVocabulary = clean(d.ActionVocabulary)
	d.EffectClasses = clean(d.EffectClasses)
	d.InferenceRoles = clean(d.InferenceRoles)
	if d.Upstream != "" && d.PinnedVersion == "" {
		return fmt.Errorf("harness %q upstream must be pinned", d.ID)
	}
	entry.Descriptor = d
	r.entries[d.ID] = entry
	return nil
}

func (r *Registry) List() []Descriptor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Descriptor, 0, len(r.entries))
	for _, entry := range r.entries {
		out = append(out, cloneDescriptor(entry.Descriptor))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (r *Registry) Qualify(ctx context.Context, id string, session BorrowedSession) Qualification {
	if r == nil {
		return Qualification{AdapterID: id, Outcome: OutcomeFailed, Error: "harness registry is nil"}
	}
	r.mu.RLock()
	entry, ok := r.entries[id]
	r.mu.RUnlock()
	if !ok {
		return Qualification{AdapterID: id, Outcome: OutcomeUnsupported, Limits: []string{"adapter is not registered"}}
	}
	if session == nil {
		return Qualification{AdapterID: id, Outcome: OutcomeUnsupported, Limits: []string{"no existing borrowed session was supplied"}}
	}
	if entry.Probe == nil {
		return Qualification{AdapterID: id, Outcome: OutcomeUnsupported, Limits: []string{"adapter is catalogued but has no integration probe"}}
	}
	result := entry.Probe(ctx, session)
	result.AdapterID = id
	if result.Outcome == "" {
		result.Outcome = OutcomeFailed
		result.Error = "probe returned no outcome"
	}
	return result
}

func clean(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func cloneDescriptor(d Descriptor) Descriptor {
	d.ObservationModes = append([]string(nil), d.ObservationModes...)
	d.ActionVocabulary = append([]string(nil), d.ActionVocabulary...)
	d.EffectClasses = append([]string(nil), d.EffectClasses...)
	d.InferenceRoles = append([]string(nil), d.InferenceRoles...)
	return d
}
