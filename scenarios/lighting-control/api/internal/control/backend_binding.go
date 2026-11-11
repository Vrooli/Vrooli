package control

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

// BackendBinding is the ephemeral scenario-side half of the headless owner
// contract. It accepts only the exact loopback endpoint handed off by the
// control-plane owner and never stores it on disk.
type BackendBinding struct {
	mu          sync.Mutex
	backend     Backend
	incarnation string
	active      int
	drained     chan struct{}
}

func NewBackendBinding() *BackendBinding { return &BackendBinding{} }

func (b *BackendBinding) Incarnation() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.incarnation
}

func (b *BackendBinding) Bind(ctx context.Context, address, incarnation string) error {
	address = strings.TrimSpace(address)
	incarnation = strings.TrimSpace(incarnation)
	if incarnation == "" {
		return errors.New("verified OpenRGB backend incarnation is required")
	}
	if _, err := validateLoopbackAddress(address); err != nil {
		return err
	}
	if err := b.Quiesce(ctx); err != nil {
		return err
	}
	b.mu.Lock()
	b.backend = SDKClient{Address: address, Timeout: 3 * time.Second}
	b.incarnation = incarnation
	b.mu.Unlock()
	return nil
}

// Quiesce removes the callable endpoint before waiting, so new work fails
// closed while already-started SDK operations drain.
func (b *BackendBinding) Quiesce(ctx context.Context) error {
	b.mu.Lock()
	b.backend = nil
	b.incarnation = ""
	if b.active == 0 {
		b.drained = nil
		b.mu.Unlock()
		return nil
	}
	if b.drained == nil {
		b.drained = make(chan struct{})
	}
	drained := b.drained
	b.mu.Unlock()
	select {
	case <-drained:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for lighting SDK operations to drain: %w", ctx.Err())
	}
}

func (b *BackendBinding) acquire() (Backend, func(), error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.backend == nil {
		return nil, nil, ErrBackendUnavailable
	}
	b.active++
	return b.backend, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.active--
		if b.active == 0 && b.drained != nil {
			close(b.drained)
			b.drained = nil
		}
	}, nil
}

func (b *BackendBinding) ProtocolVersion(ctx context.Context) (uint32, error) {
	backend, done, err := b.acquire()
	if err != nil {
		return 0, err
	}
	defer done()
	return backend.ProtocolVersion(ctx)
}

func (b *BackendBinding) Discover(ctx context.Context) ([]SDKController, error) {
	backend, done, err := b.acquire()
	if err != nil {
		return nil, err
	}
	defer done()
	return backend.Discover(ctx)
}

func (b *BackendBinding) SendZoneColors(ctx context.Context, controllerID, zoneIndex uint32, colors []RGB) (SDKCommandResult, error) {
	backend, done, err := b.acquire()
	if err != nil {
		return SDKCommandResult{}, err
	}
	defer done()
	return backend.SendZoneColors(ctx, controllerID, zoneIndex, colors)
}

func (b *BackendBinding) SendMode(ctx context.Context, controllerID uint32, target SDKModeTarget, mode SDKMode) (SDKCommandResult, error) {
	backend, done, err := b.acquire()
	if err != nil {
		return SDKCommandResult{}, err
	}
	defer done()
	modes, ok := backend.(modeBackend)
	if !ok {
		return SDKCommandResult{}, ErrUnsupported
	}
	return modes.SendMode(ctx, controllerID, target, mode)
}

func (b *BackendBinding) ApplyStableZone(ctx context.Context, controllerID, fingerprint string, desired *controlv1.ZoneState) (SDKCommandResult, error) {
	backend, done, err := b.acquire()
	if err != nil {
		return SDKCommandResult{}, err
	}
	defer done()
	stable, ok := backend.(interface {
		ApplyStableZone(context.Context, string, string, *controlv1.ZoneState) (SDKCommandResult, error)
	})
	if !ok {
		return SDKCommandResult{}, ErrUnsupported
	}
	return stable.ApplyStableZone(ctx, controllerID, fingerprint, desired)
}
