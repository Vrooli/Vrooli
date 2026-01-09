// Package unit provides language-specific unit test runners for test-genie.
//
// This package follows the same architectural pattern as the structure package:
// a thin orchestrator (Runner) that delegates to specialized sub-packages for
// each supported language (golang, nodejs, python, shell).
//
// Each language runner implements the LanguageRunner interface, allowing for:
//   - Consistent behavior across languages
//   - Easy testing via dependency injection
//   - Clear separation of concerns
//
// The orchestrator detects which languages are present in a scenario and
// runs the appropriate test suites, collecting observations and coverage data.
package unit
