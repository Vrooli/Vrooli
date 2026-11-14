// Package vrooli centralizes autoheal's integration with the core Vrooli CLI.
//
// The contract boundary lives here:
//   - checks and healers consume typed results
//   - command execution and JSON validation stay isolated
//   - orphan, lock, and port-diagnostic truth comes from core maintenance commands
package vrooli
