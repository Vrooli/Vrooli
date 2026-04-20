package server

import (
	"context"

	"deployment-manager/deployments"
	"deployment-manager/releases"
)

// releasesVerifierAdapter adapts the deployments package's LPBSReleaseClient
// (which carries extra readiness methods) onto the verify-only seam that the
// releases handler needs. Decoupling the two avoids an import cycle and keeps
// the releases package independent of deployments.
type releasesVerifierAdapter struct {
	inner deployments.LPBSReleaseClient
}

// Verify satisfies releases.LPBSVerifier.
func (a releasesVerifierAdapter) Verify(ctx context.Context, req *releases.VerifyCall) (*releases.VerifyOutcome, error) {
	if a.inner == nil {
		return &releases.VerifyOutcome{}, nil
	}
	out, err := a.inner.Verify(ctx, &deployments.LPBSVerifyRequest{
		AppKey:          req.AppKey,
		Channel:         req.Channel,
		Platform:        req.Platform,
		ExpectedVersion: req.ExpectedVersion,
		Deep:            req.Deep,
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		return &releases.VerifyOutcome{}, nil
	}
	return &releases.VerifyOutcome{
		AppKey:          out.AppKey,
		Channel:         out.Channel,
		Platform:        out.Platform,
		ExpectedVersion: out.ExpectedVersion,
		ObservedVersion: out.ObservedVersion,
		SHA512Match:     out.SHA512Match,
		Match:           out.Match,
		Error:           out.Error,
	}, nil
}
