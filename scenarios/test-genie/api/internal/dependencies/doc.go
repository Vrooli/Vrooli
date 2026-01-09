// Package dependencies validates that required runtime dependencies are available
// for a scenario. It is organized into sub-packages:
//
//   - commands: Validates that required commands (bash, curl, jq) are in PATH
//   - runtime: Detects required language runtimes (Go, Node, Python)
//   - packages: Detects required package managers (pnpm, npm, yarn)
//   - resources: Validates resource expectations and health from service.json
//
// The package exposes a Runner that orchestrates all validations and returns
// structured results suitable for integration with the test-genie phase system.
//
// This design follows "screaming architecture" principles where the package
// structure itself communicates the domain concepts (commands vs runtime vs
// packages vs resources validation) rather than technical concerns.
package dependencies
