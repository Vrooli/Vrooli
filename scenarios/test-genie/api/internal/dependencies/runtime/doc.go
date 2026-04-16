// Package runtime detects required language runtimes for a scenario.
//
// Detection is based on file presence and manifest-declared CLI adapters:
//   - Go: api/go.mod, api/*.go, or a service.json CLI adapter with kind=go_module
//   - Node.js: package.json in root or ui/ directory
//   - Python: requirements.txt or pyproject.toml
//
// The package provides an interface for testing seams and supports dependency
// injection for isolated unit tests.
package runtime
