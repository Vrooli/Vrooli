// Package provider owns the provider-neutral capacity boundary. Provider
// adapters expose only operations that survive cloud billing: create, describe,
// list and destroy. In particular, there is deliberately no stop or pause
// operation.
package provider

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCreateResponseLost = errors.New("provider create response lost")
var ErrProviderUnavailable = errors.New("provider unavailable")

type BillingFacts struct {
	RoundingUnit        time.Duration
	MinimumBillable     time.Duration
	StoppedStillBills   bool
	InboundCountsToward bool
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
}

type Provider interface {
	Name() string
	Facts() BillingFacts
	Create(context.Context, Spec) (Instance, error)
	Describe(context.Context, string) (Instance, error)
	List(context.Context) ([]Instance, error)
	Destroy(context.Context, string) error
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
	instance := Instance{ID: "fake-" + formatID(f.NextID), Region: spec.Region, Size: spec.Size, Image: spec.Image, Address: "198.51.100." + formatID(f.NextID), CreatedAt: time.Now().UTC()}
	f.Instances[instance.ID] = instance
	f.CreateCalls++
	if f.LoseCreateResponse {
		return Instance{}, ErrCreateResponseLost
	}
	return instance, nil
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
