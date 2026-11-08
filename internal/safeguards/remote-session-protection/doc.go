// Package remotesessionprotection owns the internal safeguards remote-session-protection boundary in Vrooli's control plane. It does not own host remediation or behavior outside this boundary; callers use its exported contracts and the owning service for those concerns.
//
// # Ownership boundary with agent_session_containment
//
// This safeguard writes the SYSTEM manager's units with privilege: the
// ancestor reservation under user.slice.d and desktop reservation under
// user-<uid>.slice.d (MemoryMin, MemoryLow,
// ManagedOOMPreference=omit) and workload.slice, the cgroup parent for Docker
// and batch jobs (MemoryHigh, MemoryMax, oomd kill). It never touches the
// user manager's units.
//
// agent_session_containment (internal/safeguards/agent-session-containment)
// owns the USER manager's vrooli-agents.slice, written without privilege under
// ~/.config/systemd/user, and the coding-agent launcher starts every session in
// a scope under it. That slice is a child of the user's delegated subtree, so
// the desktop aggregate reservation is backed by its non-root ancestor.
// This is not a per-core-service budget or immunity from OOM/oomd kills. Neither
// safeguard writes the other's paths.
//
// Inspect requires both recognized persistent definitions and actual kernel
// memory.min/memory.low floors on user.slice and user-1000.slice. Unknown,
// missing or unreadable observations cannot prove applied. Effective definitions
// are qualified from the SYSTEM manager's UnitPath, fragment/drop-in metadata
// and bounded on-disk generic, dashed-prefix and exact-name layers, including
// definitions not loaded yet. Persistent startup floors are checked separately
// from runtime overrides; later resets and ambiguous/unsupported memory intent
// refuse convergence instead of shadowing another owner's policy. The existing UID
// policy is 1000; this owner does not infer a new login identity.
//
// Reservation-only repair writes the two owned definitions as needed and uses
// systemd set-property --runtime for just MemoryMin/MemoryLow. It verifies exact
// persistent bytes and live floors, preserving stronger known definitions and
// effective values, including stronger other-owner definitions and manager
// properties. Unknown owned-path content or parent aliases are refused before
// effects. The reusable hostreqkit path qualifier checks every ancestor; these
// endpoint checks do not promise a transaction against arbitrary root writers.
// Lost command replies are reconciled from postconditions; failed or unreadable
// postconditions remain failures. This path performs no swap, daemon-reload,
// Docker/GPU restart, login-policy change, app restart or session termination.
// Persistent files describe future startup intent; physical boot and sustained
// host-pressure operation require separate qualification.
package remotesessionprotection
