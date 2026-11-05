package mocks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestFakeRepository_Smoke(t *testing.T) {
	f := &FakeRepository{Rows: []store.UsageRow{{OperationID: "x"}}}
	rows, err := f.ListRecent(context.Background(), time.Unix(0, 0), 10, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int64(1), f.ListCalls.Load())

	sum, err := f.Summary(context.Background(), time.Unix(0, 0), "")
	require.NoError(t, err)
	require.Equal(t, store.UsageSummary{}, sum)

	f.ListErr = errors.New("e")
	f.SumErr = errors.New("e")
	_, err = f.ListRecent(context.Background(), time.Unix(0, 0), 10, "", "")
	require.Error(t, err)
	_, err = f.Summary(context.Background(), time.Unix(0, 0), "")
	require.Error(t, err)
}
