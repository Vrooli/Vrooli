// Package workflow handles resolution and preparation of playbook workflow definitions.
//
// This includes:
//   - Resolving workflow files from disk or via the Python resolver script
//   - Substituting placeholders like ${BASE_URL} and {{UI_PORT}}
//   - Cleaning workflow definitions for BAS API consumption
package workflow
