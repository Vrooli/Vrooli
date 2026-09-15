package desktophelper

import (
	"context"
	"errors"
	"path/filepath"
	"time"
)

// MaintainGrantStatus publishes owner lease liveness to the helper. A dead or
// stalled publisher loses authority within one second. The helper's maintenance
// tick then releases held input even when no client sends another request.
// snapshot must recheck live owner state and cap expiry at its earliest lease.
func MaintainGrantStatus(ctx context.Context, path string, snapshot func(context.Context) (GrantStatus, error)) (result error) {
	return MaintainGrantStatusWithRefresh(ctx, path, snapshot, nil)
}

// Refresh requests carry buffered acknowledgement channels. The single writer
// publishes before acknowledging, allowing Open to avoid racing a periodic tick.
func MaintainGrantStatusWithRefresh(ctx context.Context, path string, snapshot func(context.Context) (GrantStatus, error), refresh <-chan chan error) (result error) {
	if snapshot == nil {
		return ErrBootstrap
	}
	if !filepath.IsAbs(path) {
		return ErrBootstrap
	}
	if err := privateDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	unlock, err := lockPrivateFile(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	defer func() {
		now := nowUTC()
		result = errors.Join(result, writePrivateJSON(path, GrantStatus{Active: []string{}, ObservedAt: now, ExpiresAt: now.Add(GrantStatusLifetime)}))
	}()
	publish := func() error {
		state, err := snapshot(ctx)
		if err != nil {
			return err
		}
		if !validGrantStatus(state, nowUTC()) {
			return ErrBootstrap
		}
		return writePrivateJSON(path, state)
	}
	if ctx.Err() != nil {
		return nil
	}
	if err := publish(); err != nil {
		return err
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case reply := <-refresh:
			err := publish()
			reply <- err
			if err != nil {
				return err
			}
		case <-ticker.C:
			if err := publish(); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}
