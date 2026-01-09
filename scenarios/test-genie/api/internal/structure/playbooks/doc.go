// Package playbooks validates the structure of bas/ playbook directories.
//
// Playbooks validation is currently informational (warnings) rather than
// blocking. This allows scenarios to adopt conventions gradually.
//
// Validated conventions:
//   - bas/registry.json must exist and be valid JSON
//   - bas/cases/ and bas/flows/ top-level folders should use two-digit prefixes
//   - bas/seeds/ should contain seed.go or seed.sh if it exists
//
// See docs/phases/playbooks/directory-structure.md for the canonical layout.
package playbooks
