// Package provider owns the provider-neutral capacity boundary. Provider
// adapters expose only operations that survive cloud billing: create, describe,
// list and destroy. In particular, there is deliberately no stop or pause
// operation.
package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrCreateResponseLost  = errors.New("provider create response lost")
	ErrProviderUnavailable = errors.New("provider unavailable")
)

type BillingFacts struct {
	RoundingUnit        time.Duration
	MinimumBillable     time.Duration
	StoppedStillBills   bool
	InboundCountsToward bool
}

// BillingStatement is an optional provider accounting observation. It is
// deliberately separate from Provider so lifecycle consumers still depend on
// exactly four operations.
type BillingStatement struct {
	Provider           string
	ProviderInstanceID string
	Minutes            int64
	From, To           time.Time
}

type BillingStatementSource interface {
	BillingStatements(context.Context, time.Time, time.Time) ([]BillingStatement, error)
}

type Spec struct {
	Region   string
	Size     string
	Image    string
	UserData string
	Tags     map[string]string
}

type Instance struct {
	ID        string
	Region    string
	Size      string
	Image     string
	Address   string
	CreatedAt time.Time
	Tags      map[string]string
}

type Provider interface {
	Name() string
	Facts() BillingFacts
	Create(context.Context, Spec) (Instance, error)
	Describe(context.Context, string) (Instance, error)
	List(context.Context) ([]Instance, error)
	Destroy(context.Context, string) error
}

// Registry is the provider selection boundary. Consumers select by the
// stable adapter name and never depend on a concrete implementation.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
	primary   Provider
}

func NewRegistry(providers ...Provider) *Registry {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, p := range providers {
		if p != nil {
			r.providers[p.Name()] = p
			if r.primary == nil {
				r.primary = p
			}
		}
	}
	return r
}

func (r *Registry) Register(p Provider) error {
	if r == nil || p == nil || p.Name() == "" {
		return fmt.Errorf("provider and provider name are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.providers == nil {
		r.providers = make(map[string]Provider)
	}
	if _, exists := r.providers[p.Name()]; exists {
		return fmt.Errorf("provider %q is already registered", p.Name())
	}
	r.providers[p.Name()] = p
	if r.primary == nil {
		r.primary = p
	}
	return nil
}

func (r *Registry) Primary() (Provider, error) {
	if r == nil {
		return nil, fmt.Errorf("provider registry is unavailable")
	}
	r.mu.RLock()
	p := r.primary
	r.mu.RUnlock()
	if p == nil {
		return nil, fmt.Errorf("provider registry has no primary")
	}
	return p, nil
}

func (r *Registry) Get(name string) (Provider, error) {
	if r == nil {
		return nil, fmt.Errorf("provider registry is unavailable")
	}
	r.mu.RLock()
	p, ok := r.providers[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("provider %q is not registered", name)
	}
	return p, nil
}

// Fake is deterministic provider infrastructure for tests. LoseCreateResponse
// persists the created instance while returning ErrCreateResponseLost, which
// exercises the intent/reconciliation path without cloud credentials.
type Fake struct {
	mu                 sync.Mutex
	Instances          map[string]Instance
	NextID             int
	LoseCreateResponse bool
	CreateCalls        int
	DestroyCalls       int
	Now                func() time.Time
}

func (*Fake) Name() string { return "fake" }
func (*Fake) Facts() BillingFacts {
	return BillingFacts{RoundingUnit: time.Hour, MinimumBillable: time.Hour, StoppedStillBills: true}
}

func (f *Fake) Create(_ context.Context, spec Spec) (Instance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Instances == nil {
		f.Instances = make(map[string]Instance)
	}
	f.NextID++
	now := time.Now
	if f.Now != nil {
		now = f.Now
	}
	instance := Instance{ID: "fake-" + formatID(f.NextID), Region: spec.Region, Size: spec.Size, Image: spec.Image, Address: "198.51.100." + formatID(f.NextID), CreatedAt: now().UTC(), Tags: cloneTags(spec.Tags)}
	f.Instances[instance.ID] = instance
	f.CreateCalls++
	if f.LoseCreateResponse {
		return Instance{}, ErrCreateResponseLost
	}
	return instance, nil
}

func cloneTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	copyOf := make(map[string]string, len(tags))
	for key, value := range tags {
		copyOf[key] = value
	}
	return copyOf
}

func (f *Fake) Describe(_ context.Context, id string) (Instance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	instance, ok := f.Instances[id]
	if !ok {
		return Instance{}, errors.New("provider instance not found")
	}
	return instance, nil
}

func (f *Fake) List(_ context.Context) ([]Instance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Instance, 0, len(f.Instances))
	for _, instance := range f.Instances {
		out = append(out, instance)
	}
	return out, nil
}

func (f *Fake) Destroy(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.Instances[id]; !ok {
		return errors.New("provider instance not found")
	}
	delete(f.Instances, id)
	f.DestroyCalls++
	return nil
}

func formatID(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	out := make([]byte, 0, 10)
	for n > 0 {
		out = append([]byte{digits[n%10]}, out...)
		n /= 10
	}
	return string(out)
}
