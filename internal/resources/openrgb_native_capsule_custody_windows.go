//go:build windows

package resources

import (
	"context"
	platform "github.com/vrooli/platform-go"
)

func prepareOpenRGBNativeWrapperCustody(ctx context.Context, intent platform.ServiceIntent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return platform.PrepareWindowsOwnedServiceSubmission(ctx, platform.OperationIntent(intent))
}
