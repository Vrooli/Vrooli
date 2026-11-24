package main

import (
	"context"

	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	delivery "landing-page-business-suite-api/internal/delivery"
)

func deliveryAppDependencies(plans interface{ BundleKey() string }, hosting ...interface {
	GetArtifact(context.Context, string, int64) (*delivery.Artifact, error)
}) downloadhttp.AppDependencies {
	deps := downloadhttp.AppDependencies{
		BundleKey: plans.BundleKey, PathParam: getPathParam, DecodeJSON: decodeJSONBody,
		WriteError: writeJSONError, WriteData: writeJSONSuccessData, WriteSuccess: writeJSONSuccessSimple,
	}
	if len(hosting) > 0 && hosting[0] != nil {
		deps.GetArtifact = hosting[0].GetArtifact
	}
	return deps
}
