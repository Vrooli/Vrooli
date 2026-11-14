//go:build !windows

package resources

import (
	"context"
	platform "github.com/vrooli/platform-go"
)

func prepareOpenRGBNativeWrapperCustody(ctx context.Context, _ platform.ServiceIntent) error {
	return ctx.Err()
}
