// Package bridge provides inactive finite owner adapters. It registers no route,
// opens no database, and supplies no authority, credentials or process boundary.
package bridge

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/encoding/protojson"
)

var ErrSelectedRead = errors.New("selected event read rejected")
var selectedID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// Boundary must share serialization with source authority changes and broker
// session invalidation. Within invokes fn once, synchronously, with cancellation
// and expiry preserved. A disconnected mutex is not a production boundary.
type Boundary interface {
	Within(context.Context, func(context.Context) error) error
}

// SelectedReader reads the existing Events owner's canonical store, never a
// generic anonymous HTTP endpoint. Construction performs no I/O. The returned
// value can satisfy Program Runtime's structurally typed CanonicalEventReader
// without making either module depend on the other's private implementation.
type SelectedReader struct {
	store    *store.SQLiteStore
	boundary Boundary
}

func NewSelectedReader(s *store.SQLiteStore, b Boundary) (*SelectedReader, error) {
	if s == nil || b == nil {
		return nil, ErrSelectedRead
	}
	return &SelectedReader{store: s, boundary: b}, nil
}

// ReadSelected accepts trusted verifier callbacks, not worker request fields.
// Recheck occurs before storage and after encoding under the same boundary.
// Validate receives identity from the canonical protobuf AND matching indexes.
// A missing canonical record returns nil, never invented event occurrence data.
func (r *SelectedReader) ReadSelected(ctx context.Context, id string, recheck func(context.Context) error, validate func(context.Context, string, string, string) error) ([]byte, error) {
	if r == nil || r.store == nil || r.boundary == nil || ctx == nil || ctx.Err() != nil || !selectedID.MatchString(id) || recheck == nil || validate == nil {
		return nil, ErrSelectedRead
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var result []byte
	called := false
	err := r.boundary.Within(ctx, func(inner context.Context) error {
		if called || inner == nil || ctx.Err() != nil || inner.Err() != nil {
			return ErrSelectedRead
		}
		called = true
		if err := recheck(inner); err != nil {
			return ErrSelectedRead
		}
		e, err := r.store.SelectedCanonical(inner, id)
		if err != nil {
			return ErrSelectedRead
		}
		if e != nil {
			env, err := convert.EventToEnvelope(*e)
			if err != nil || env.GetEventId() != id || env.GetEventId() != e.EventID || env.GetEventType() != e.EventType || env.GetSource().GetScenario() != e.SourceScenario {
				return ErrSelectedRead
			}
			if err = validate(inner, env.GetEventId(), env.GetEventType(), env.GetSource().GetScenario()); err != nil {
				return ErrSelectedRead
			}
			result, err = protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(env)
			if err != nil || len(result) > 1<<20 {
				return ErrSelectedRead
			}
		}
		if ctx.Err() != nil || inner.Err() != nil || recheck(inner) != nil {
			return ErrSelectedRead
		}
		return nil
	})
	if err != nil || !called || ctx.Err() != nil {
		return nil, ErrSelectedRead
	}
	return result, nil
}
