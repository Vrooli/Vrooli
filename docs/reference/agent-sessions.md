# Agent sessions: leases, owners, the list, reaping, claims, freezing and thawing

Every coding-agent session launched through the control plane is born inside
a ceiling, records what it belongs to, and stays visible for as long as it
lives. This page names the pieces and the commands that read and act on them.

## The editor lease

`vrooli agent launch` (and the `vrooli-agent-launcher` binary the shims call)
records an editor lease in the runtime registry
(`runtime_editor_leases`) at launch: session id, harness, agent, pid, host
boot id, working directory, containment scope, and the claimed paths. The
same lease is what `vrooli agent list` and the `vrooli status` summary read.

The launcher chooses one host session ID and uses it for attachment, containment,
lease registration and cleanup. The Agent Manager run ID and native conversation
ID remain distinct. `vrooli agent identity-binding --json` returns the shared
versioned proof with both the attached `harness_session_id` and actual
`lease_session_id`. Agent Manager also supplies its persisted `--run-id` and
`--run-pid`. A lease recorded under that exact run ID can be correlated when the
canonical host key is absent; the expected process, harness, kernel incarnation,
owner, scope and reread checks still apply. A conflicting canonical lease or a
registry error refuses correlation. This read does not rewrite leases or mint
identity credentials; the broker independently verifies its loaded run and peer.

A lease expires only on proof of death: a boot id from another boot, a
missing pid, or a pid this host proves dead. A session that is slow to
heartbeat is building, not gone, and is never evicted on elapsed time alone.
The stale-lock sweep (`vrooli maintenance clean-stale-locks`, and every
`--clean-stale` start) retires dead sessions with the reason recorded.
The runtime supervisor also checks editor leases during its normal tick. It
uses the same deadline and positive-death proof as the cleanup command, and
only retires the lease record; it does not stop the agent process. The separate
orphan-session reap remains opt-in and may stop a contained session only after
its owner is confirmed gone.

| Launch branch | Heartbeat | Why |
| --- | --- | --- |
| spawn-and-wait (the launcher keeps a child) | every 20 s, deadline 60 s | the launcher is alive to renew |
| exec-replace (the launcher becomes the agent) | none | the launcher stops existing; the lease's pid is the agent's, and the registry expires it on proof of death |

A registry that cannot be opened never blocks a launch: the session runs
unrecorded and the launcher says so (`LeaseRecorded: false`).

## The owner

Every lease records what the session belongs to (`owner_kind`, `owner_ref`,
`owner_evidence_json`, `owner_source` on `runtime_editor_leases`, schema 14).
The launcher's lease recorder reads it from its own process at launch; a lease
recorded before owners existed is backfilled the first time `vrooli agent list`
or `vrooli agent reap` sees it, from the session's own environment and process
tree. The owner is never read from a scope name.

| Kind | Evidence (first match wins) | Ref | Owner is gone when |
| --- | --- | --- | --- |
| `web-console-session` | `WC_WEB_CONSOLE_SESSION_ID` (plus `WC_SESSION_STATE_ROOT`, the tmux socket) | web-console session id | the host web-console answers `not_found` for the id (archived, dismissed or absent) |
| `desktop-app` | `VROOLI_DESKTOP_MODE=true` or `BUNDLE_ROOT`+`APP_DATA_DIR` | the app's `VROOLI_DATA` | the app runtime's pid is gone or reused (pid + start time recorded at launch); for a backfilled owner, no process outside the agent slice carries the same `VROOLI_DATA` |
| `agent-manager-run` | run id from the `VROOLI_AGENT_IDENTITY_TOKEN` claims or an `agent-manager/runs/<uuid>` `CODEX_HOME`/`GROK_HOME`; a `<RUNNER>_AGENT_TAG` alone is evidence only | run id | Agent Manager reports the run `COMPLETE`, `FAILED` or `CANCELLED`, or answers `NOT_FOUND_RUN` |
| `terminal` | a controlling tty, a non-web-console tmux, `SSH_TTY` or `STY` | tty or socket | never: no authority to ask |
| `unknown` | none of the above | — | never |

A web-console session id wins over the desktop markers because an operator
terminal inside a desktop build of web-console carries both, and persistent
terminals outlive the app. Owner liveness is read-only: `GET
/api/v1/runs/{id}` on Agent Manager, the Connect `SessionsService/Get` on
web-console, `/proc` for processes. Only the host web-console instance is
asked (its state root under the runtime state directory, not a variant's
`variants/<name>` root and not a desktop app's own root); a session of another
instance stays `unknown`. A probe that cannot answer leaves the owner
`unknown`, and an unknown owner is never orphaned. While a web-console session
is live its reported origin and owner are added to the recorded evidence,
because web-console stops answering for a session once it is archived.

### The desktop-helper leak this guards

On 2026-10-07, 42 `opencode serve` sessions started by the bundled
web-console desktop app (tree `/tmp/appimage_extracted_*/resources/bundle`,
parent pid 1, about 21 days old) were still running in their own agent scopes
after the app exited, because the scope outlives the process that started it.
A launch whose environment shows a bundled desktop app now records the app's
runtime (the topmost ancestor carrying the same `VROOLI_DATA`) as the owner,
so its helpers become `orphaned` the moment the app exits. The deeper fix,
bundled apps not routing their helpers through the host agent path at all,
belongs to the desktop-packaging goal and plan `bde99dd3`
(`trustworthy-bundled-desktop-apps-private-runtime-isolation`).

## The list

```bash
vrooli agent list [--json] [--idle-after 168h]
vrooli status              # "Agent sessions: N live (M in this tree)"
```

The table shows session, agent, state, owner and owner state, tree, scope,
pid, age, claims and frozen. `--json` adds, per row, `owner {kind, ref, alive,
liveness, detail, checked_at, source, evidence}`, `kernel_state` and
`last_activity`, the resource and reap fields below, and a `summary`. `alive`
is `true` or `false` when the owner's authority answered and `null` when it
could not be asked or the kind has no authority.

| Row field | Meaning |
| --- | --- |
| `memory_bytes`, `swap_bytes` | sum of `VmRSS` / `VmSwap` over the session's processes (shared pages count once per process); `null` when unknown, never 0 |
| `memory_status`, `swap_status` | `measured`; `partial` (some processes unreadable, the figure is a lower bound); `unsupported` (no per-process reader on this platform: macOS, Windows); `unavailable` (scope not held, unreadable, or no member readable) |
| `usage_detail` | why a status is not `measured` |
| `reapable`, `reap_reason` | the reap policy's verdict with the default `--older-than` (1h), e.g. `false` / `web-console session origin ui: an operator terminal is never auto-reaped` |

These figures and verdicts come from the same code `vrooli agent reap` uses
for its Memory/Swap columns, audit rows and plan (`agentsession.MeasureUsage`,
`agentsession.ReapDecision`). `summary` holds `by_owner_kind` (state counts per
owner kind), `sessions`, `memory_bytes`, `swap_bytes` (summed over rows whose
figure is known), `memory_unknown`, `swap_unknown` (rows left out of those
sums), `reapable`, `reapable_memory_bytes` and `reapable_swap_bytes`.

| State | Meaning |
| --- | --- |
| `live` | the session's process runs in a scope the kernel holds, and its owner is present or cannot be judged |
| `idle` | live, with no heartbeat and no terminal input or output for `--idle-after` (default 7 days, `VROOLI_TUNING_AGENT_SESSION_IDLE_AFTER`); reported, never reaped |
| `orphaned` | the owner kind is known and its authority confirmed it gone, while the scope still holds processes |
| `agentless` | a scope whose coding agent exited and left children (named `orphan` before owners existed) |
| `unrecorded` | a scope no lease names, or a lease naming a scope the kernel does not hold |
| `ghost` | a lease whose process is gone, awaiting the sweep |
| `undetermined` | the kernel could not be read; never shown as healthy |

The tree is the session's working directory; the scope is the containment
scope the session was born in (`vrooli-agents.slice/vrooli-agent-<id>.scope`
on Linux); frozen reflects the scope's `cgroup.freeze`.

## Reaping orphaned sessions

```bash
vrooli agent reap --orphaned --dry-run [--json]       # what would stop, with owner evidence and memory
vrooli agent reap --orphaned [--older-than 1h] [--json]
```

Reap considers only `orphaned` sessions; `live`, `idle`, `agentless`,
terminal and unknown-owner sessions are never candidates. An orphaned session
is still kept when its owner is a web-console session not known to be
programmatic (an observed origin of `ui`, `remote` or `unspecified`, or an
unobserved origin without Agent Manager evidence: an operator terminal is
never auto-reaped), when it is younger than `--older-than` (default 1h), when
its scope is not an agent session scope unit, or when its scope is frozen.
Immediately before each stop the owner is asked again; an owner that is no
longer confirmed gone wins. The stop goes through platform-go's `StopScope`:
`systemctl --user stop` of the scope on Linux (`cgroup.kill` for a hand-made
cgroup), guarded to `vrooli-agents.slice`; macOS and Windows report
`unsupported` until a process-group or Job Object adapter is host-verified.
Every stop, failure or unsupported outcome writes one row to
`runtime_agent_session_reaps` with the owner evidence, the liveness answer and
the memory, swap and tasks the scope held (`-1` for a figure that was
unknown). `--dry-run` never calls the
platform adapter and writes no audit row.

The runtime supervisor can run the same reap unattended. It is **off** by
default: set `VROOLI_TUNING_AGENT_ORPHAN_REAP_INTERVAL` (for example `1h`) in
the supervisor's environment to enable it; it uses the 1h age floor and
records `runtime-supervisor` as the initiator.

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
| owner model, liveness, state, reap plan | `internal/agentsession` |
| list and reap commands | `internal/cli/vroolicli/agent_list.go`, `internal/cli/vroolicli/agent_reap.go` |
| periodic reap (off by default) | `internal/runtimesupervisor/service.go` (`reapOrphanedAgentSessions`) |
| launcher seam and recorder | `packages/cli-core/cliutil/agentlease.go`, `internal/cli/vroolicli/sessionlease` |
| containment and stop | `packages/platform-go/containment*.go` (`StopScope`), `internal/safeguards/agent-session-containment` |
| overlap rule | `internal/scenarioruntime/pathoverlap.go` |
| storm authority | `scenarios/vrooli-autoheal` (`system-emergency-watchdog-report`, `contain-storm`) |
