// Package structure provides screaming-architecture validation for scenario file
// structure and content. It is organized into sub-packages:
//
//   - existence: Validates that required directories, files, and CLI artifacts exist
//   - content: Validates file contents (JSON syntax, manifest schemas, etc.)
//
// The package exposes a Runner that orchestrates all validations and returns
// structured results suitable for integration with the test-genie phase system.
//
// This design follows "screaming architecture" principles where the package
// structure itself communicates the domain concepts (existence vs content
// validation) rather than technical concerns.
package structure
