// Package playbooks validates the structure of test/playbooks/ directories.
//
// Playbooks validation is currently informational (warnings) rather than
// blocking. This allows scenarios to adopt conventions gradually.
//
// Validated conventions:
//   - registry.json must exist and be valid JSON
//   - capabilities/ and journeys/ subdirectories should use two-digit prefixes
//   - __subflows/ fixtures must declare fixture_id in metadata
//   - __seeds/ should contain apply.sh if it exists
//
// See docs/phases/playbooks/directory-structure.md for the canonical layout.
package playbooks
