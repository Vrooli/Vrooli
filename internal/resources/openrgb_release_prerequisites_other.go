//go:build !windows && !linux && !darwin

package resources

import (
	"context"
)

func verifyOpenRGBReleaseRuntimePrerequisites(ctx context.Context, _ string) error { return ctx.Err() }
