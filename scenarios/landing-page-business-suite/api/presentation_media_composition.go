package main

import (
	"context"
	"errors"

	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/presentation"
)

func combinePresentationPublicationVerifiers(first, second experimentation.PresentationPublicationVerifier) experimentation.PresentationPublicationVerifier {
	return func(ctx context.Context, document presentation.Document, active *presentation.Document) error {
		if first == nil || second == nil {
			return errors.New("presentation publication verifier is unavailable")
		}
		if err := first(ctx, document, active); err != nil {
			return err
		}
		return second(ctx, document, active)
	}
}
