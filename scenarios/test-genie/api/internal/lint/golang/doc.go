// Package golang provides Go-specific linting using golangci-lint or go vet.
//
// Tool priority:
//  1. golangci-lint (comprehensive, respects .golangci.yml)
//  2. go vet (fallback, built-in)
//
// If golangci-lint is not available, a warning is logged and go vet is used.
// If neither is available, the check is skipped with an info message.
package golang
