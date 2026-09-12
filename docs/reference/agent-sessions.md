# Agent sessions: leases, the list, claims, freezing and thawing

Every coding-agent session launched through the control plane is born inside
a ceiling and stays visible for as long as it lives. This page names the four
pieces and the commands that read them.

## The editor lease

`vrooli agent launch` (and the `vrooli-agent-launcher` binary the shims call)
records an editor lease in the runtime registry
(`runtime_editor_leases`) at launch: session id, harness, agent, pid, host
boot id, working directory, containment scope, and the claimed paths. The
same lease is what `vrooli agent list` and the `vrooli status` summary read.

A lease expires only on proof of death: a boot id from another boot, a
missing pid, or a pid this host proves dead. A session that is slow to
heartbeat is building, not gone, and is never evicted on elapsed time alone.
The stale-lock sweep (`vrooli maintenance clean-stale-locks`, and every
`--clean-stale` start) retires dead sessions with the reason recorded.

| Launch branch | Heartbeat | Why |
| --- | --- | --- |
| spawn-and-wait (the launcher keeps a child) | every 20 s, deadline 60 s | the launcher is alive to renew |
| exec-replace (the launcher becomes the agent) | none | the launcher stops existing; the lease's pid is the agent's, and the registry expires it on proof of death |

A registry that cannot be opened never blocks a launch: the session runs
unrecorded and the launcher says so (`LeaseRecorded: false`).

## The list

```bash
vrooli agent list          # session, harness, tree, scope, pid, age, claims, frozen
vrooli status              # "Agent sessions: N live (M in this tree)"
```

The tree is the session's working directory; the scope is the containment
scope the session was born in (`vrooli-agents.slice/vrooli-agent-<id>.scope`
on Linux); frozen reflects the scope's `cgroup.freeze`.

## Claims (advisory)

```bash
vrooli agent launch --runner claude --claim internal/setpoint --claim docs/reference
```

A claim is a path the session says it will edit. At launch the launcher
compares the claims against every live lease (the same path-overlap rule
workspace-sandbox uses for sandbox scopes) and prints each overlapping holder:
agent, session, pid, age and tree. The launch continues; in this release
claims are advisory by the operator's choice, and refusal is a follow-up flag
on the same comparison.

## Freezing and thawing

When the emergency watchdog attributes a sustained fork storm to a session
scope, vrooli-autoheal's `contain-storm` action freezes that scope
(`cgroup.freeze`), records a decision row, and opens an incident that names
the session and its tree. It never freezes a supervisor or a resource: the
target must be a `vrooli-agent-*` scope under `vrooli-agents.slice`, and the
action runs only through the runtime recovery gate.

```bash
vrooli agent thaw <session-or-scope>   # reverses the freeze and resolves the incident
vrooli-autoheal storm status           # frozen scopes and their decisions
```

A frozen session's terminal stops responding; the incident says so and names
the thaw command.

## Where the pieces live

| Piece | Code |
| --- | --- |
| lease store and sweep | `internal/scenarioruntime/editor_lease.go`, `internal/maintenance` |
| launcher seam and recorder | `packages/cli-core/cliutil/agentlease.go`, `internal/cli/vroolicli/sessionlease` |
| containment | `packages/platform-go/containment*.go`, `internal/safeguards/agent-session-containment` |
| overlap rule | `internal/scenarioruntime/pathoverlap.go` |
| storm authority | `scenarios/vrooli-autoheal` (`system-emergency-watchdog-report`, `contain-storm`) |
