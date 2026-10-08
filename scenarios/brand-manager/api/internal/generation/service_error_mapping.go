package generation

import (
	"errors"

	"connectrpc.com/connect"
)

// ToConnectError translates domain sentinels into Connect's typed error model.
// Unknown errors map to internal so callers never depend on raw provider or
// storage details.
func ToConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid ErrInvalidGeneration
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	var notFound ErrBrandNotFound
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, notFound)
	}
	var srcNotFound ErrSourceAssetNotFound
	if errors.As(err, &srcNotFound) {
		return connect.NewError(connect.CodeNotFound, srcNotFound)
	}
	var unavailable ErrProvidersUnavailable
	if errors.As(err, &unavailable) {
		return connect.NewError(connect.CodeUnavailable, unavailable)
	}
	// Image-tools readiness failures, translated to brand-level codes.
	var imgUnavailable ErrImageBackendUnavailable
	if errors.As(err, &imgUnavailable) {
		return connect.NewError(connect.CodeUnavailable, imgUnavailable)
	}
	var imgNotReady ErrImageBackendNotReady
	if errors.As(err, &imgNotReady) {
		return connect.NewError(connect.CodeFailedPrecondition, imgNotReady)
	}
	var imgJobFailed ErrImageJobFailed
	if errors.As(err, &imgJobFailed) {
		return connect.NewError(connect.CodeInternal, imgJobFailed)
	}
	return connect.NewError(connect.CodeInternal, err)
}
