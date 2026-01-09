// Package runtime detects required language runtimes for a scenario.
//
// Detection is based on file presence in the scenario directory:
//   - Go: api/go.mod, cli/go.mod, or *.go files
//   - Node.js: package.json in root or ui/ directory
//   - Python: requirements.txt or pyproject.toml
//
// The package provides an interface for testing seams and supports dependency
// injection for isolated unit tests.
package runtime
