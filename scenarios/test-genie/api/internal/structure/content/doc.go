// Package content validates file contents within a scenario.
//
// This includes:
//   - JSON syntax validation for all .json files
//   - Manifest schema validation (service.json structure and semantics)
//
// The package is designed for extensibility - future validators can be added
// for YAML files, TypeScript configs, or other content types.
//
// Each validator implements a common interface for consistent testing and
// integration with the structure validation runner.
package content
