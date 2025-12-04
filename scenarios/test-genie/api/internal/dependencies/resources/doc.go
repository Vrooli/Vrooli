// Package resources validates resource expectations and health for a scenario.
//
// This includes:
//   - Loading required resources from .vrooli/service.json
//   - Checking resource health via scenario status telemetry
//   - Validating API dependencies connectivity
//
// The package provides interfaces for testing seams and supports dependency
// injection for isolated unit tests.
package resources
