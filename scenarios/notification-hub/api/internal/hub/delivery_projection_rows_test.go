package hub

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type iterationFailureRows struct {
	nexts  int
	closed bool
	err    error
}

func (r *iterationFailureRows) Next() bool {
	r.nexts++
	return r.nexts == 1
}

func (*iterationFailureRows) Scan(...any) error { return nil }
func (r *iterationFailureRows) Err() error      { return r.err }
func (r *iterationFailureRows) Close() error {
	r.closed = true
	return nil
}

func TestEachDeliveryProjectionRowRejectsPartialIteration(t *testing.T) {
	iterationErr := errors.New("injected row iteration failure")
	rows := &iterationFailureRows{err: iterationErr}
	visited := 0

	err := eachDeliveryProjectionRow(rows, func() error {
		visited++
		return nil
	})

	require.ErrorIs(t, err, iterationErr)
	require.Equal(t, 1, visited, "the helper must reject a stream that fails after returning a partial row")
	require.True(t, rows.closed, "failed row streams must be closed")
}
