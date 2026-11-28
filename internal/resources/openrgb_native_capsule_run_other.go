//go:build !darwin && !windows

package resources

import (
	"context"
	platform "github.com/vrooli/platform-go"
)

func RunOpenRGBNativeCapsule(context.Context, string) error { return platform.ErrUnsupported }
