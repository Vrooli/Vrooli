# vrooli-autoheal-loop

The boot-recovery loop is the process a native scheduler unit (systemd user
unit, launchd agent, Windows scheduled task) keeps alive so that the
`vrooli-autoheal` scenario is started at boot and restarted when it dies. It
is the one component that must keep working when the scenario it supervises
cannot build, which is why it is a nested Go module with a deliberately small
dependency graph.

## Files

| File | Owns |
|------|------|
| `main.go` | Entry: flags, root and CLI resolution, the signal context, `--self-test`, exit codes 2 and 4 |
| `config.go` | `Config`, flag parsing, exit-code constants, the single `setBaseURL`, `sleepCtx`, endpoint validation |
| `ports.go` | `detectAPIPort` (six strategies, every one identity-checked) and `adoptPort`, the one adoption path |
| `lifecycle.go` | `invokeVrooli`, `ensureAPIRunning`, `heal`, `runLifecycleWithRecovery`, `waitForAPIHealthy`, `runTick`, `healError` |
| `preflight.go` | `Preflight` and its six named checks |
| `state.go` | The status file (`loopStatus`, atomic `statusWriter`) |
| `loop.go` | The state machine |
| `selfheal.go` | The recovery floor (dependency-drift repair through `langrecover`) with its persisted circuit breaker |
| `identity.go` | `isAutohealAPI`: a port is adopted only when `/health` names the autoheal service |
| `install.go`, `atomic_install_*.go` | `--install-self`: atomic replacement of the installed binary |

## State machine

```
Preflight ──ok──▶ Detect ──verified──▶ Verify ──tick ok──▶ Healthy ◀──┐
   │                 │                    │                    │        │
   │ failed          │ nothing/pending    │ tick failed        │ tick ok│
   ▼                 ▼                    ▼                    ▼        │
Degraded ◀────────  Heal ◀──────────── (alive? Degraded : Heal)   Degraded
   │                 │
   │ 3 non-healable  │ 3 non-healable
   ▼                 ▼
 Exit(3)           Exit(3)
```

- **Preflight** runs the six checks below. Failure is non-healable.
- **Detect** runs `detectAPIPort`. A verified port goes to Verify; nothing, or
  a registry port nobody answers on, goes to Heal with a reason.
- **Heal** waits out the backoff, then calls `ensureAPIRunning`: if the API
  came up on its own in the meantime it is adopted without a lifecycle
  command; otherwise `heal` runs `vrooli scenario start|restart <name>
  --best-effort` through the recovery floor and waits for the API to identify
  itself. Failures are classified by `cliinvoke.Result.Class`:
  - `usage`, `binary-missing`: non-healable. Counted; retried after one tick
    interval (a chance for the host to change, not a backoff); the third
    consecutive one exits 3. The recovery floor is never consulted, so no
    breaker slot is spent.
  - `lifecycle`: the recovery floor inspects the output and the lifecycle log
    for a dependency-drift signature, repairs once, retries once. Then backoff.
  - `timeout`, `refusal`: retried with backoff.
  - Backoff: 1 minute, doubling to a 15-minute cap, reset by the first healthy
    tick.
- **Verify** runs one tick against the freshly adopted API.
- **Healthy** ticks every `--interval` seconds (default 60) and re-runs the
  preflight every 60 ticks. After `--max-failures` consecutive tick failures it
  asks whether autoheal still answers on the adopted port: a live process whose
  ticks fail is busy, not dead (Degraded); a silent one is what Heal is for.
- **Degraded** keeps the tick cadence, re-runs the preflight on every interval,
  and returns to Verify as soon as the preflight passes. A preflight that keeps
  failing counts toward exit 3.
- **Exit** writes the status file with the exit code, then the process ends.

One `context.Context` from `signal.NotifyContext(SIGINT, SIGTERM)` reaches
every sleep, every wait, every HTTP probe and every `cliinvoke.Run`. A child
`vrooli` in flight is cancelled and the loop returns within
`cliinvoke.DefaultWaitDelay` (5 s) of the signal, well inside the unit's
`TimeoutStopSec`.

## Port detection and adoption

Only `Config.adoptPort` sets `APIPort`, `LastKnownPort`, `HealthEndpoint` and
`TickEndpoint`, and it refuses any port whose `/health` does not name the
autoheal service. `detectAPIPort` tries, in order: `API_PORT`, the process
registry's `port` file, `vrooli scenario status --json` then `vrooli scenario
port`, the registry's `metadata.json`, the last known port, and a list of
historical allocations. Every strategy verifies identity before returning a
port. The registry port file alone may return an unverified port, and only as
`Pending`: `waitForAPIHealthy` polls it first, but never adopts it unverified.

## Preflight checks

| Name | Passes when | Skipped when |
|------|-------------|--------------|
| `cli-resolves` | `cliinvoke.Resolve` found a binary (explicit `--vrooli-bin`, `VROOLI_BIN`, the runtime home's bin entry, PATH) | never |
| `cli-answers` | `vrooli version --json` exits 0 and parses | no binary |
| `cli-contract` | `vrooli scenario status vrooli-autoheal --json` parses through the typed shape and names the scenario | no binary |
| `state-writable` | the state directory can be created and a temp file written in it | never |
| `toolchain` | `go` is found on PATH or in the per-OS table `langrecover.DefaultPathEntries` | `--no-manage-api` (the recovery floor never runs) |
| `root-resolves` | the repository root resolved | never |

Each check yields `ok`, `failed` or `skipped` with a reason. A check that ran
the CLI and failed also carries the invoker's `class`, which the status file
reports as `last_failure_class` (a stage name is not a class).

`--self-test` runs the preflight against the resolved CLI with read-only
probes, prints the result as JSON, and exits 0 when every check passed or 3
otherwise. Phase 10's freshness safeguard calls it.

## Status file

`~/.vrooli/state/vrooli-autoheal/loop-status.json`, resolved through
`repocontract.RuntimeHomeEntryPath(home, HomeKeyState)`, written atomically
(temp file plus rename) on every tick and every state change.

```json
{
  "started_at": "2026-09-02T13:50:44Z",
  "last_tick_at": "2026-09-02T13:51:44Z",
  "last_tick_status": "ok",
  "state": "healthy",
  "consecutive_failures": 0,
  "last_failure_class": "",
  "degraded_reason": "",
  "preflight": { "at": "...", "ok": true, "checks": [ { "name": "cli-resolves", "status": "ok", "reason": "..." } ] },
  "binary_sha256": "...",
  "pid": 12345,
  "exit_code": 3,
  "updated_at": "2026-09-02T13:51:44Z"
}
```

| Field | Meaning |
|-------|---------|
| `started_at`, `updated_at` | Process start; last write |
| `last_tick_at` | `null` until the first tick |
| `last_tick_status` | `pending`, `ok`, `failed` |
| `state` | `preflight`, `detect`, `heal`, `verify`, `healthy`, `degraded`, `exit` |
| `consecutive_failures` | Consecutive non-healable failures; three exits the loop. Reset by a healthy tick, not by a passing preflight |
| `last_failure_class` | `usage`, `binary-missing`, `timeout`, `refusal`, `lifecycle`, `preflight`, `tick` |
| `degraded_reason` | Why the loop is not healthy, or why it exited |
| `preflight` | The last preflight result |
| `binary_sha256` | Hash of the running executable |
| `exit_code` | Present only once the loop has decided to exit |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | SIGINT or SIGTERM honoured |
| 2 | Repository root could not be resolved, or the flags did not parse |
| 3 | Three consecutive non-healable failures (status file written first); also `--self-test` with a failed check |
| 4 | State directory cannot be created |

The unit uses `Restart=on-failure` with a start-limit burst and an
`OnFailure=` target, so escalation past exit 3 is the scheduler's.

## Flags

`--interval`, `--max-failures`, `--vrooli-bin` and `--install-self` are part
of the unit and safeguard contract and must not change. `--api-url` fixes the
API location and skips detection. `--no-manage-api` disables lifecycle
commands (Heal goes to Degraded). `--self-test` is described above.

## Dependency posture

`go.mod` depends on `repo-contract-go` (which carries the `cliinvoke`
invoker), `envkit-go`, and `langrecover`. `langrecover` is stdlib-only. Neither
`packages/proto` nor `api-core` may appear in `go mod graph`: a shared-package
change that breaks the scenario's API must not be able to break the loop that
repairs it. `langrecover/toolchain.go` copies the per-OS PATH table from
`platform-go` rather than importing it for the same reason; the equality test
between the two tables lives in `packages/platform-go`.

## Building and testing

```
make -C scenarios/vrooli-autoheal loop-test loop-lint   # module tests and lint
make -C scenarios/vrooli-autoheal loop-build            # explicit developer build to cli/vrooli-autoheal-loop
```

`vrooli setup` builds and installs the binary the unit runs; the Makefile
target is the developer's path, not the lifecycle's. To exercise a build
without touching the host, run it with `--vrooli-bin` pointing at a fake and
`HOME` pointing at an empty directory, as the phase-8 evidence under
`~/.vrooli/plan-artifacts/boot-recovery-readiness/evidence/p08/` does.
