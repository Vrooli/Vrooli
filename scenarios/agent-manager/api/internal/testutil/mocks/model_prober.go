package mocks

import (
	"context"
	"sync"
)

// FakeModelProber records model probe calls and returns ProbeErr for each call.
type FakeModelProber struct {
	mu sync.Mutex

	ProbeErr error
	models   []string
}

func NewFakeModelProber() *FakeModelProber {
	return &FakeModelProber{}
}

func NewFailingModelProber(err error) *FakeModelProber {
	return &FakeModelProber{ProbeErr: err}
}

func (p *FakeModelProber) ProbeModel(_ context.Context, model string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.models = append(p.models, model)
	return p.ProbeErr
}

func (p *FakeModelProber) Models() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.models...)
}
