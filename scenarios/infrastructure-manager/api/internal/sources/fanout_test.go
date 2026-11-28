package sources

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReadFansOutAndKeepsPartialFailureNamed(t *testing.T) {
	results := Read(context.Background(), []Endpoint{
		{ID: "healthy", Reader: ReaderFunc(func(context.Context) ([]Observation, error) { return []Observation{{ID: "A1"}}, nil })},
		{ID: "broken", Reader: ReaderFunc(func(context.Context) ([]Observation, error) { return nil, errors.New("peer unavailable") })},
	}, time.Second)
	require.Len(t, results, 2)
	require.True(t, results[0].Available)
	require.False(t, results[1].Available)
	require.Equal(t, "peer unavailable", results[1].Reason)
}

func TestReadNamesTimeoutPerSource(t *testing.T) {
	results := Read(context.Background(), []Endpoint{{ID: "slow", Reader: ReaderFunc(func(ctx context.Context) ([]Observation, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})}}, time.Millisecond)
	require.Len(t, results, 1)
	require.False(t, results[0].Available)
	require.Contains(t, results[0].Reason, "deadline")
}
