// Package discovery provides a validation wrapper around requirements/discovery.
//
// This package adapts the generic requirements discovery functionality into
// the business validation pattern, providing:
//   - A Validator interface for dependency injection
//   - Result types consistent with other business validators
//   - Proper observation and error handling
package discovery
