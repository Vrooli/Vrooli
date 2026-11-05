package mocks

import (
	"context"
	"errors"
	"testing"
)

func TestFakePinger(t *testing.T) {
	p := NewFakePinger()
	if err := p.PingContext(context.Background()); err != nil {
		t.Errorf("default PingContext should be nil, got %v", err)
	}
	if p.Calls != 1 {
		t.Errorf("Calls = %d, want 1", p.Calls)
	}
	p.Err = errors.New("db down")
	if err := p.PingContext(context.Background()); err == nil {
		t.Error("PingContext should surface Err")
	}
	if p.Calls != 2 {
		t.Errorf("Calls = %d, want 2", p.Calls)
	}
}
