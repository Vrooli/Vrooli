// Package playbooks provides Vrooli Ascension (BAS) workflow execution
// for end-to-end UI testing. It is organized into sub-packages:
//
//   - registry: Loads and parses playbook registry files
//   - workflow: Resolves workflow definitions and substitutes placeholders
//   - execution: BAS API client for workflow execution and status polling
//   - seeds: Manages seed script application and cleanup
//   - artifacts: Writes timeline dumps and phase results
//
// The package exposes a Runner that orchestrates all playbook operations and
// returns structured results suitable for integration with the test-genie phase system.
//
// This design follows "screaming architecture" principles where the package
// structure itself communicates the domain concepts rather than technical concerns.
package playbooks
