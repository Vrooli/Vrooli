//go:build windows

package hostfs

import (
	"context"
	"errors"
)

type processLiveness struct{}

func NewProcessLiveness() *processLiveness { return &processLiveness{} }

func (*processLiveness) IsRunning(context.Context, string) (bool, error) {
	return false, errors.New("process liveness is unavailable on windows")
}
