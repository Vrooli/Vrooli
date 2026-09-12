// Package testing exposes the skill-testing transport boundary.
package testing

import domain "prompt-manager/internal/testing"

var (
	NewHandlers     = domain.NewHandlers
	NewOllamaClient = domain.NewOllamaClient
	NewRepository   = domain.NewRepository
)
