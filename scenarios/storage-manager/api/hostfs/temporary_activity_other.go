//go:build !linux

package hostfs

import (
	"context"
	"fmt"
	"storage-manager/internal/cleanup"
)

type temporaryActivityObserver struct{}

func NewTemporaryActivityObserver() cleanup.TemporaryActivityObserver {
	return temporaryActivityObserver{}
}
func (temporaryActivityObserver) ObserveTemporary(context.Context, string) (map[string]struct{}, error) {
	return nil, fmt.Errorf("privileged temporary activity observation requires Linux")
}
