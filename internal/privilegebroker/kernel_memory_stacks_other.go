//go:build !linux

package privilegebroker

import (
	"context"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

func sampleKernelMemoryStacks(context.Context, Executor, string) (hostinventory.KernelMemoryStackReport, error) {
	return hostinventory.KernelMemoryStackReport{}, platformgo.ErrUnsupported
}
