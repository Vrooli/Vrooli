package backends

import (
	"context"
	"testing"
)

type providerStub struct {
	name                         string
	standalone, cloud, available bool
}

func (p providerStub) Name() string                                     { return p.name }
func (p providerStub) Operations() []string                             { return []string{"compose"} }
func (p providerStub) Standalone() bool                                 { return p.standalone }
func (p providerStub) IsCloud() bool                                    { return p.cloud }
func (p providerStub) Available(_ context.Context) bool                 { return p.available }
func (p providerStub) Execute(context.Context, Request) (Result, error) { return Result{}, nil }
func TestRegistryRequiresHeadlessProvider(t *testing.T) {
	r := New()
	if err := r.Register(providerStub{name: "cloud", cloud: true}); err != nil {
		t.Fatal(err)
	}
	if err := r.Validate(); err == nil {
		t.Fatal("restricted registry accepted cloud-only operation")
	}
}
func TestRegistrySelectsLocal(t *testing.T) {
	r := New()
	_ = r.Register(providerStub{name: "local", standalone: true, available: true})
	p, err := r.Select(context.Background(), "compose")
	if err != nil || p.Name() != "local" {
		t.Fatalf("provider=%v err=%v", p, err)
	}
}
