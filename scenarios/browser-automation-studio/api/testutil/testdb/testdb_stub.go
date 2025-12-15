//go:build !testcontainers

package testdb

import (
	"context"
	"errors"
)

var ErrTestcontainersDisabled = errors.New("testcontainers support disabled (build with -tags=testcontainers)")

type Handle struct {
	DSN        string
	externalDS bool
}

func Start(ctx context.Context) (*Handle, error) {
	_ = ctx
	return nil, ErrTestcontainersDisabled
}

func (h *Handle) Terminate(ctx context.Context) {
	_ = ctx
}
