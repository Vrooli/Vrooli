package main

import "landing-page-business-suite-api/internal/intelligence"

// AIGateway is the interface for the AI gateway service.
// This seam enables handler testing without the real service implementation.
type AIGateway = intelligence.Gateway
