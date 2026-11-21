//go:build !linux

package procsampler

import (
	"context"
	"runtime"
)

func InvestigateReadActivity(_ context.Context) (ReadActivity, error) {
	return ReadActivity{Status: "unsupported", Platform: runtime.GOOS, Reason: "per-process read activity has no native backend on this platform", Semantics: "No read rates are inferred when the platform backend is unavailable."}, nil
}
