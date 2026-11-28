package catalog

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type warningLog struct {
	message string
	args    []any
}

func (l *warningLog) Warn(message string, args ...any) {
	l.message, l.args = message, args
}

func TestMatrixCancellationWarningUsesInjectedLogger(t *testing.T) {
	logger := &warningLog{}
	wantErr := context.Canceled
	h := &handler{logger: logger}
	h.warnMatrixCanceled(wantErr)
	require.Equal(t, "catalog matrix canceled", logger.message)
	require.Equal(t, []any{"error", wantErr}, logger.args)
}

func TestMatrixCancellationWithoutLoggerIsSafe(t *testing.T) {
	require.NotPanics(t, func() { (&handler{}).warnMatrixCanceled(errors.New("cancelled")) })
}
