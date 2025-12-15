// Package existence validates that required filesystem entries exist for a scenario.
//
// This includes:
//   - Required directories (api, cli, docs, requirements, ui)
//   - Required files (README.md, PRD.md, Makefile, service.json, testing.json)
//   - CLI structure (legacy bash scripts or cross-platform Go binaries)
//
// The package provides interfaces for testing seams and supports configuration
// via .vrooli/testing.json to add or exclude paths from validation.
package existence
