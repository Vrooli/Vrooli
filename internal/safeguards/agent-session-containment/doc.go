// Package agentsessioncontainment is the host safeguard that gives every
// coding-agent session a ceiling: it writes and verifies vrooli-agents.slice
// under the invoking user's systemd manager, and the launcher starts every
// session in a scope under that slice.
//
// # The failure it prevents
//
// On 2026-09-02 three agent sessions built the repository at once on a
// 32-core host. Go starts one compile per core, so three sessions became about
// a hundred linkers; memory filled, swap absorbed the overflow, the 15-minute
// load reached 1,499 and the host was unusable for twenty minutes. Nothing
// could stop it: the sessions ran in user-1000.slice with MemoryMax=infinity,
// TasksMax=164514 and ManagedOOMPreference=omit, so systemd-oomd protected the
// desktop and never touched the storm.
//
// # What it changes
//
// A user unit at ~/.config/systemd/user/vrooli-agents.slice, rendered from
// platform-go's slice definition with four typed settings: CPUWeight (default
// 50, half the neutral share), MemoryHigh (50% of physical memory, throttling),
// MemoryMax (60%, the kernel kills inside the slice before the host swaps) and
// TasksMax (4096, a fork storm stops at the ceiling). ManagedOOMMemoryPressure=
// kill lets systemd-oomd reclaim in the slice first. Inspect renders the unit,
// compares the file, runs it through systemd-analyze, then reads the LIVE slice
// (ActiveState, ControlGroup and the cgroup's memory.max, pids.max, cpu.weight):
// a file that is written but not loaded, or loaded with other values, is
// not-applied. A probe that cannot run is undetermined, never ok.
//
// # Ownership boundary with remote_session_protection
//
// remote_session_protection owns the SYSTEM manager's units: the desktop
// reservation in user-<uid>.slice.d and workload.slice (Docker's cgroup
// parent), both written with privilege by `sudo vrooli setup`. This safeguard
// owns the USER manager's vrooli-agents.slice and needs no privilege. Neither
// writes the other's paths; the agent slice is a child of the user's own
// delegated subtree, so the desktop reservation still bounds it from above.
//
// # Other platforms
//
// macOS and Windows have no slice. The launcher applies the same ceilings
// per session there (an rlimit shim on macOS, a Job Object with quotas on
// Windows); this safeguard reports those defaults and is not applicable as a
// host mutation. Those tiers are fixture-verified, not host-verified.
package agentsessioncontainment
