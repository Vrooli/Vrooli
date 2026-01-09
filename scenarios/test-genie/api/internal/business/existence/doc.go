// Package existence validates that required filesystem entries exist for business validation.
//
// This includes:
//   - requirements/ directory must exist
//   - requirements/index.json must exist (configurable)
//
// The package provides interfaces for testing seams and supports configuration
// via .vrooli/testing.json to enable/disable index file requirements.
package existence
