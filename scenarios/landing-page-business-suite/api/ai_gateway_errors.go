package main

import (
	"errors"

	"landing-page-business-suite-api/internal/intelligence"
)

// Compatibility aliases preserve the existing API transport contract while
// intelligence owns AI-provider and gateway error identities. Insufficient
// credits remains composition-owned until the usage service's error contract
// is moved into commerce, because it is shared policy rather than AI logic.
var (
	ErrInsufficientCredits   = errors.New("insufficient credits for this operation")
	ErrNoAPIKeyConfigured    = intelligence.ErrNoAPIKeyConfigured
	ErrModelNotAllowed       = intelligence.ErrModelNotAllowed
	ErrAIGatewayUnavailable  = intelligence.ErrAIGatewayUnavailable
	ErrStreamingNotSupported = intelligence.ErrStreamingNotSupported
)
