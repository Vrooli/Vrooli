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
// TasksMax (16384, a fork storm stops at the ceiling). ManagedOOMMemoryPressure=
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

// The safeguard converges a second slice, vrooli-services.slice, because a
// ceiling without somewhere for services to go is not a ceiling for long. A
// background step started by the lifecycle engine is a detached daemon; until
// 2026-09-04 it inherited the cgroup of whatever started the phase, so a
// scenario started from inside a coding-agent session kept that session's
// scope alive after the agent exited and stayed charged to the agent slice
// for as long as it ran. Seven such scopes held 1,410 tasks and 20 GB, the
// slice reached its task ceiling with every host-level bar green, and the
// kernel began refusing the fork of every new session — which surfaced to the
// operator as an unrelated component reporting its database corrupt.
//
// The services slice carries the opposite policy to the agent slice on
// purpose: CPUWeight above agents, MemoryHigh to throttle, and deliberately no
// MemoryMax and no ManagedOOMMemoryPressure=kill. An agent session is worth
// killing to save the host; a scenario or resource service is the thing being
// saved.
//
// Inspect reports occupancy as well as configuration. A ceiling is a safety
// property only while there is room under it, and a full slice is a correctly
// configured slice, so occupancy is reported as a note and never as drift:
// applying the safeguard is not the fix for a slice that is simply full.

// The ceilings themselves are still provisional. Decision D3 of the
// polite-host plan proposed CPUWeight=50, MemoryHigh=50%, MemoryMax=60% and
// TasksMax=4096 and said in terms that "the operator has not yet confirmed
// these numbers"; the plan then shipped without that confirmation. TasksMax
// was sized to stop a fork storm, which is a burst, and never against steady
// state, which is what actually filled it: roughly forty concurrent sessions
// on 2026-09-04, several of them holding scenario services that had no other
// slice to live in.
//
// 16384 is a headroom figure, not a derived one. Deriving it needs a time
// series rather than a spot reading, and until 2026-09-05 nothing recorded
// one. The autoheal check system-containment-occupancy now samples both
// slices every 60 seconds against setpoint bar substrate/SB21, so the
// derivation becomes possible once a week of representative operation has
// been recorded — representative meaning after two distortions were removed
// on 2026-09-05: a probe loop that minted roughly 29,000 session scopes a day
// by asking every agent `--version` through the shims, and services being
// charged to the agent slice instead of their own.
//
// First clean baseline, 2026-09-05, 35 live sessions with the fleet already
// moved out: vrooli-agents.slice 3,445 tasks and 30 GB; vrooli-services.slice
// 883 tasks and 12 GB. Both slices sit at 16384 tasks. Set the confirmed
// values from the recorded peak, not from this single reading.
