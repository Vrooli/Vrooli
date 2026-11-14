//go:build windows

package resources

import (
	"context"
	platform "github.com/vrooli/platform-go"
)

func normalizeOpenRGBNativeEnvironment(context.Context, platform.ServiceRef) error { return nil }
