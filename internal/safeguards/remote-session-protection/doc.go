// Package remotesessionprotection owns the internal safeguards remote-session-protection boundary in Vrooli's control plane. It does not own host remediation or behavior outside this boundary; callers use its exported contracts and the owning service for those concerns.
//
// # Ownership boundary with agent_session_containment
//
// This safeguard writes the SYSTEM manager's units with privilege: the
// desktop reservation drop-in under user-<uid>.slice.d (MemoryMin, MemoryLow,
// ManagedOOMPreference=omit) and workload.slice, the cgroup parent for Docker
// and batch jobs (MemoryHigh, MemoryMax, oomd kill). It never touches the
// user manager's units.
//
// agent_session_containment (internal/safeguards/agent-session-containment)
// owns the USER manager's vrooli-agents.slice, written without privilege under
// ~/.config/systemd/user, and the coding-agent launcher starts every session in
// a scope under it. That slice is a child of the user's delegated subtree, so
// the desktop reservation written here still bounds it from above. Neither
// safeguard writes the other's paths.
package remotesessionprotection
