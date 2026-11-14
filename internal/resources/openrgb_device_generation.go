package resources

import (
	"context"
	"errors"

	"github.com/vrooli/vrooli/internal/lightingruntime"
)

var ErrOpenRGBDeviceGenerationChanged = errors.New("selected OpenRGB device generation changed; refresh the exact native backend")

func refreshOpenRGBDeviceGeneration(ctx context.Context, cause error, quiesce, retire func(context.Context) error, start func(context.Context) (lightingruntime.Backend, error)) (lightingruntime.Backend, error) {
	if !errors.Is(cause, ErrOpenRGBDeviceGenerationChanged) {
		return lightingruntime.Backend{}, cause
	}
	if err := quiesce(ctx); err != nil {
		return lightingruntime.Backend{}, errors.Join(cause, err)
	}
	if err := retire(ctx); err != nil {
		return lightingruntime.Backend{}, errors.Join(cause, err)
	}
	// Exactly one replacement per Ensure. A failed or uncertain replacement
	// remains fenced/reconcilable; it never becomes an internal restart loop.
	return start(ctx)
}
