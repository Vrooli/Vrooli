package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Availability string

const (
	Available   Availability = "available"
	Unavailable Availability = "unavailable"
	Unknown     Availability = "unknown"
)

type (
	HostFacts map[string]bool
	Listing   struct {
		Descriptor   Descriptor   `json:"descriptor"`
		Availability Availability `json:"availability"`
		Reason       string       `json:"reason,omitempty"`
		Implemented  bool         `json:"implemented"`
	}
)

type Registry struct {
	descriptors map[string]Descriptor
	adapters    map[string]Adapter
}

type HTTPAdapter interface {
	HTTPPath() string
	Handler() http.Handler
	BindReceive(func(Envelope) error)
}

func Load(dir string, adapters ...Adapter) (*Registry, error) {
	r := &Registry{descriptors: map[string]Descriptor{}, adapters: map[string]Adapter{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read channel descriptors: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var d Descriptor
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("%s: invalid JSON: %w", path, err)
		}
		if err := d.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if _, exists := r.descriptors[d.ID]; exists {
			return nil, fmt.Errorf("%s: duplicate field id %q", path, d.ID)
		}
		r.descriptors[d.ID] = d
	}
	for _, a := range adapters {
		if a != nil {
			r.adapters[a.ID()] = a
		}
	}
	return r, nil
}

func (r *Registry) Get(id string) (Descriptor, bool)  { d, ok := r.descriptors[id]; return d, ok }
func (r *Registry) Adapter(id string) (Adapter, bool) { a, ok := r.adapters[id]; return a, ok }
func (r *Registry) HTTPAdapters() []HTTPAdapter {
	out := make([]HTTPAdapter, 0)
	for _, adapter := range r.adapters {
		if httpAdapter, ok := adapter.(HTTPAdapter); ok {
			out = append(out, httpAdapter)
		}
	}
	return out
}

// ThreadStarters returns every adapter that can open a conversation on demand.
func (r *Registry) ThreadStarters() []ThreadStarter {
	out := make([]ThreadStarter, 0)
	for _, adapter := range r.adapters {
		if starter, ok := adapter.(ThreadStarter); ok {
			out = append(out, starter)
		}
	}
	return out
}

// Start connects all configured, non-HTTP adapters whose descriptor says they
// are available on this host. The returned function cancels and joins all
// adapter loops. HTTP adapters are already bound by the HTTP module and must
// not be connected a second time.
func (r *Registry) Start(ctx context.Context, facts HostFacts, receive func(Envelope) error, logf func(string, ...any)) func() {
	if ctx == nil {
		ctx = context.Background()
	}
	if receive == nil {
		receive = func(Envelope) error { return nil }
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	runCtx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup

	listings := r.List(runCtx, facts)
	for _, listing := range listings {
		if listing.Availability != Available || !listing.Implemented {
			continue
		}
		adapter, ok := r.adapters[listing.Descriptor.ID]
		if !ok {
			continue
		}
		if _, ok := adapter.(HTTPAdapter); ok {
			continue
		}
		wg.Add(1)
		go func(id string, adapter Adapter) {
			defer wg.Done()
			probeCtx, probeCancel := context.WithTimeout(runCtx, 5*time.Second)
			probe := adapter.Probe(probeCtx)
			probeCancel()
			if !probe.Available {
				if probe.Reason != "" {
					logf("channel adapter %s not started: %s", id, probe.Reason)
				}
				return
			}
			if err := adapter.Connect(runCtx, receive); err != nil && runCtx.Err() == nil {
				logf("channel adapter %s stopped: %v", id, err)
			}
		}(listing.Descriptor.ID, adapter)
	}

	return func() {
		cancel()
		wg.Wait()
	}
}

func (r *Registry) List(_ context.Context, facts HostFacts) []Listing {
	out := make([]Listing, 0, len(r.descriptors))
	for _, d := range r.descriptors {
		state, reason := Evaluate(d, facts)
		_, implemented := r.adapters[d.ID]
		if !implemented {
			reason = "listed but unimplemented"
		}
		out = append(out, Listing{Descriptor: d, Availability: state, Reason: reason, Implemented: implemented})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Descriptor.Setup.Friction < out[j].Descriptor.Setup.Friction })
	return out
}

func Evaluate(d Descriptor, facts HostFacts) (Availability, string) {
	for _, req := range d.Requires {
		value, ok := facts[req.Key]
		if !ok {
			return Unknown, fmt.Sprintf("host fact %q is unknown", req.Key)
		}
		if !value {
			return Unavailable, req.Description
		}
	}
	return Available, ""
}
