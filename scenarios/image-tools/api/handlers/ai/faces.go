package ai

import (
	"context"
	"errors"
	"io"
	"sync"
)

// maxFaceCheckBytes bounds what the people check reads from one stored input.
const maxFaceCheckBytes = 64 << 20

// inputFaces counts the people in one request's input images, at most once.
//
// The consent rule for a low-weight operation asks for an affirmation only when
// the image shows people, and two places need that answer: the engine, while it
// weighs a cloud route, and the gate, once the route is known. Both read this one
// memoised result, so the detector runs once per request and only when a route
// that leaves this machine is actually being considered.
type inputFaces struct {
	gate Gate
	load func(ctx context.Context) ([][]byte, error)

	once  sync.Once
	count int
	err   error
	ran   bool
}

// newInputFaces reads the stored inputs identified by keys when asked.
func (h *Deps) newInputFaces(keys ...string) *inputFaces {
	if h.Gate == nil {
		return nil
	}
	return &inputFaces{gate: h.Gate, load: func(ctx context.Context) ([][]byte, error) {
		var images [][]byte
		for _, key := range keys {
			if key == "" {
				continue
			}
			reader, _, err := h.Store.Get(ctx, key)
			if err != nil {
				return nil, err
			}
			data, readErr := io.ReadAll(io.LimitReader(reader, maxFaceCheckBytes+1))
			closeErr := reader.Close()
			if err := errors.Join(readErr, closeErr); err != nil {
				return nil, err
			}
			if len(data) > maxFaceCheckBytes {
				return nil, errors.New("input is too large for the people check")
			}
			images = append(images, data)
		}
		return images, nil
	}}
}

// Count returns the number of faces across the inputs. A nil receiver means
// there is no gate, so nobody can look.
func (f *inputFaces) Count(ctx context.Context) (int, error) {
	if f == nil {
		return 0, errors.New("no consent gate to count faces")
	}
	f.once.Do(func() {
		f.ran = true
		images, err := f.load(ctx)
		if err != nil {
			f.err = err
			return
		}
		f.count, f.err = f.gate.CountFaces(ctx, images)
	})
	return f.count, f.err
}

// Clear reports that the check ran and found nobody: the only outcome that lets a
// low-weight request skip consent on a route that leaves this machine.
func (f *inputFaces) Clear() bool {
	return f != nil && f.ran && f.err == nil && f.count == 0
}

// Func is the people check in the shape the engine and the gate take; nil when
// there is no gate.
func (f *inputFaces) Func() func(context.Context) (int, error) {
	if f == nil {
		return nil
	}
	return f.Count
}
