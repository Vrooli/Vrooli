# Project-Level Bash → Go Migration Plan

**Status:** Weeks 0 through 6 are implemented and validated in the default Go path. The project-level command surface is native and validated (`status`, `doctor`, `info`, `clean`, `build`, `deploy`, `stop`, `orphans`, `locks`, `diagnose-port`, and top-level `backup`/`restore` dispatch). The `VROOLI_FORCE_BASH` escape hatch is gone, `path:scripts/manage.sh` is deleted, the dead top-level shell command handlers under `path:cli/commands/` and `path:cli/lib/` are gone, and the root `cli/` bootstrap directory is deleted. Remaining work is no longer blocked on project-level Go migration; it is concentrated in shared shell cleanup under `path:scripts/lib/` that is still consumed by resources and scenarios, plus legacy secrets/resource compatibility surfaces that belong to the adjacent migration tracks.
**Owner:** Matthew Halloran
**Scope:** Project-level orchestration only. Scenarios are explicitly out of scope.
**Target:** Zero project-level bash in ~6 weeks, on a path to cross-platform support.

---

## 0. For agents picking this up later

If you are an agent orienting to this plan, read this section first.

- **What this plan covers:** Porting `/cli/`, `/scripts/lib/`, and `/scripts/manage.sh` from Bash to Go. This is the *orchestrator* (the `vrooli` CLI and its supporting libraries), not the scenarios it manages.
- **What this plan does NOT cover:** Anything under `path:scenarios/*/`. Scenarios are already on their own cross-platform path via `path:packages/cli-core` and `path:packages/api-core`. Do not touch scenario internals as part of this work.
- **How to find current progress:** Check the "Progress Tracker" section at the bottom, then run `make validate` to verify the retained project-level validation suite end-to-end. Do not rely on `git log --grep="bash-to-go"`; that naming convention was not applied consistently, so the code and validation target are more reliable than commit subjects.
- **How to resume work:** Start from the remaining unchecked items in the Progress Tracker. At this point those items are cleanup and boundary-hardening work, not missing project-level CLI migrations.
- **Escape hatch:** Historical note only: `VROOLI_FORCE_BASH=1` used to route subcommands back to Bash during migration. That path is now removed from the Go CLI; any remaining mentions should be treated as cleanup debt or historical context.
- **If you find stale information in this plan:** Update it. The plan is a living document; the repo's current state is always authoritative over what's written here.

---

## 1. Motivation

### The problem
Vrooli's project-level orchestration is currently ~33,335 LOC of Bash:
- `cli/` — 11,278 LOC (the `vrooli` user-facing CLI and subcommand handlers)
- `path:scripts/lib/` — 21,779 LOC (lifecycle executor, port management, runtime setup, networking, secrets, process tracking)
- `path:scripts/manage.sh` — 278 LOC (setup/develop/build/deploy router)

Bash has three properties that are biting us:

1. **Not portable.** Windows has no native Bash. WSL exists but is an extra install and a compatibility layer, not a first-class experience. macOS has Bash but many lifecycle routines depend on Linux-specific tools (`apt`, `systemctl`, `/proc`, `iptables`, `bwrap`, `ip netns`). Roughly 30% of current bash is genuinely Linux-kernel-specific and unportable regardless of language choice; the remaining 70% is "thin glue" that *could* port cleanly to a cross-platform language.
2. **Unreliable at scale.** Bash error handling is hard to get right, tests are awkward (bats), and refactoring is risky because there's no type system catching drift.
3. **Hard to test.** There's no unit-test culture for ~22k LOC of library code. Most bugs get caught by end-to-end scenario runs, which is slow feedback.

Newer scenarios in this repo have already proven that Go is a workable replacement — `path:packages/cli-core` and `path:packages/api-core` are shared Go libraries consumed by 27+ scenarios, and the pattern works. The project level should follow the same trajectory.

### The goal
Port `/cli/`, `/scripts/lib/`, and `/scripts/manage.sh` to Go, organized as a single top-level Go module with multiple commands and shared `internal/` packages. Outcome:

- `vrooli` becomes a single statically-linked Go binary
- Cross-platform compilation works for Linux, macOS, and (with platform-gated features) Windows
- Lifecycle code is unit-testable
- "Running stale code" becomes impossible via build-time fingerprinting + startup stale check
- Expected size reduction: ~33k LOC Bash → ~10–12k LOC Go (2–3× compression is typical for this kind of port)

### Why Go, not Rust or C
- **Go:** Team already writes it. `path:packages/cli-core` and `path:packages/api-core` prove the pattern. Stdlib covers JSON/HTTP/process/signals/cross-compile. Single static binary per platform. `CGO_ENABLED=0` gives effortless cross-compilation.
- **Rust:** Would work technically, but the team doesn't use it. Switching cost for glue code outweighs the marginal safety benefit.
- **C:** No JSON/HTTP/process stdlib worth mentioning, poor memory safety for orchestration code, wrong tool.

### Why not "just wrap bash in WSL/Docker for Windows"
That's the compatibility-layer path. It works but it's fragile, adds install friction, and doesn't solve the testability/reliability problems. A real Go port solves all three concerns (portability, reliability, testability) at once.

---

## 2. Current state (as of 2026-04-10)

### Project-level Bash inventory
| Path | LOC | Purpose |
|---|---|---|
| `path:cli/vrooli` | 474 | Top-level CLI dispatcher |
| `path:cli/commands/*.sh` | 10,348 | Command handlers (scenario, resource, status, doctor, etc.) |
| `path:cli/lib/*.sh` | 580 | CLI-specific utilities |
| `path:scripts/manage.sh` | 278 | Lifecycle router |
| `path:scripts/lib/utils/` | 5,718 | Arg parsing, JSON, logging, config, retry logic |
| `path:scripts/lib/utils/lifecycle.sh` | 1,049 | **Lifecycle executor — core of the system** |
| `path:scripts/lib/network/` | 2,473 | Port allocation, firewall, SSH, diagnostics |
| `path:scripts/lib/network/ports.sh` | 898 | **Port allocation — flock-based registry** |
| `path:scripts/lib/system/` | 2,558 | Kernel config, system checks, clock sync, trash |
| `path:scripts/lib/service/` | 1,603 | Secrets, cloudflare, repos |
| `path:scripts/lib/resources/` | 720 | Resource orchestration, validation |
| `path:scripts/lib/scenario/runner.sh` | 668 | Scenario discovery + phase execution |
| `path:scripts/lib/process-manager.sh` | 154 | PID tracking |
| **Total** | **~33,335** | |

### Existing project-level Go (the foothold)
- **`/api/main.go`** — 2,026 LOC, module `vrooli.com/api`, gorilla/mux, compiles to `/api/vrooli-api`
  - Purpose: project-level HTTP orchestration server invoked from `vrooli develop`
  - Exposes: `/scenarios/start-all`, `/scenarios/start/<name>`, scenario listing, health checks
  - Currently delegates most real work back down to bash — **this is the seed crystal for the migration**
- **`/scripts/main/`** — Removed obsolete bash-only directory; use the `vrooli` CLI and repo-backed Go entrypoints instead.
- No top-level `cmd/`, `internal/`, or `pkg/` directories yet.

### The `vrooli scenario restart` call path (3,000 LOC of bash, concrete example)
When a user runs `vrooli scenario restart foo`:
1. `path:cli/vrooli` (474 LOC) parses top-level args, routes to scenario handler
2. `path:cli/commands/scenario/scenario-commands.sh` (167 LOC) dispatches to `restart` subcommand
3. `path:cli/commands/scenario/modules/lifecycle.sh` (294 LOC, restart is ~24 of them) calls stop then start
4. `path:scripts/lib/utils/lifecycle.sh` (1,049 LOC) handles process termination, port release
5. `path:scripts/lib/scenario/runner.sh` (668 LOC) re-sources the scenario's `.vrooli/service.json` and re-runs its phases

This is the canonical "hard case" — when you test the Go port, test this path first.

### Scenario discovery contract
There is **no registry file**. Discovery is pure directory scan:
- `path:scripts/lib/scenario/runner.sh::scenario::list()` iterates `$VROOLI_ROOT/scenarios/*/`
- Each candidate must contain `.vrooli/service.json`
- Process tracking lives in `~/.vrooli/processes/scenarios/<name>/*.json` and `*.pid`
- Sandbox-aware: if `VROOLI_SANDBOX_SCOPE` is set, paths resolve through the sandbox's overlay `merged/` directory (lines 47–98 of `runner.sh`). The Go port must preserve this behavior.

### What already has a good pattern worth copying
- **`path:packages/cli-core/buildinfo` + `cliutil.StaleChecker`** — scenario CLIs embed a source fingerprint via `-ldflags` at build time, and re-exec through `cli-installer` when the fingerprint drifts. This exact pattern should land at the project level for `vrooli`. See Section 4.
- **`path:packages/api-core/server`** — clean HTTP server lifecycle with signal handling and graceful shutdown. `path:cmd/vrooli-api/main.go` can consume it directly.

---

## 3. Target architecture

### Single top-level Go module
```
/go.mod                          # module vrooli.com (or similar)
/cmd/
  vrooli/                        # the CLI binary (replaces cli/vrooli + cli/commands/)
  vrooli-api/                    # moved from /api/main.go
/internal/
  buildinfo/                     # fingerprint + ldflags + stale check + rebuild+re-exec
  scenario/                      # discovery, service.json reader, sandbox-aware path resolution
  lifecycle/                     # phase executor (setup/develop/build/test/stop), condition eval, health-check polling
  ports/                         # allocation, registry with flock, net.Listen probing
  process/                       # PID files, signal-based stop, orphan detection
  config/                        # .vrooli/ path resolution, env var handling
  logx/                          # structured slog setup
  cliout/                        # human-vs-JSON output, table rendering, color (NO_COLOR + tty detection)
  runtime/                       # Go/Node/Python/Docker/Helm detection — PLATFORM-GATED via build tags
  network/                       # diagnostics, firewall — Linux-only, stubbed elsewhere
  secrets/                       # encrypted config read/write
  setup/                         # `vrooli setup` phase runner (replaces scripts/manage.sh)
```

### Platform gating via build tags
Linux-only functionality lives in files with `//go:build linux` build tags; other platforms get stubs that return `ErrUnsupportedPlatform`. Example:

```
internal/runtime/sysctl_linux.go     // real implementation
internal/runtime/sysctl_other.go     // //go:build !linux — returns ErrUnsupportedPlatform
```

This way, the compiler enforces platform support at build time. Running `GOOS=windows go build ./cmd/vrooli` produces a working (or honestly-failing) binary with no runtime `runtime.GOOS == ...` sniffing scattered through the code.

### Why single module, not multiple
Two separate modules (`path:cmd/vrooli` and `path:cmd/vrooli-api`) would re-create the duplication problem we already have in bash. Single module with shared `internal/` means:
- Both binaries can't drift
- One `go build ./...` builds everything
- Type-level refactors work across both CLI and API at once
- Shared lifecycle code has exactly one home

### What does NOT go in the top-level module
- Anything scenario-facing. Scenarios consume `path:packages/cli-core` and `path:packages/api-core` — those are their ABI and must stay separate. Conflating them with project orchestration code would couple scenario stability to internal orchestrator changes.

---

## 4. Compile best practices

The fundamental shift from Bash to Go is: Bash reads source on every invocation, Go compiles once and runs the binary. The risk is running stale binaries after source changed. Here's how to eliminate that risk entirely.

### For the `vrooli` CLI binary (short-lived, invoked on demand)

1. **Self-healing stale check on every invocation.** Same pattern the scenario CLIs already use, hoisted to the project level. On startup:
   - Compute a SHA256 fingerprint of `path:cmd/vrooli/` + `internal/` Go sources
   - Compare to the fingerprint embedded via `-ldflags` at build time
   - If they differ: rebuild, re-exec with the same argv, transparent to the user
   - One-time ~200ms cost when stale, zero cost when fresh

   **This is the single most important practice.** It eliminates "I edited code, forgot to rebuild, ran the old one" entirely. It should land in Week 1.

2. **Install location + PATH discipline.**
   - Install to `~/.vrooli/bin/vrooli` (user-owned, no sudo)
   - Setup script puts `~/.vrooli/bin` on PATH
   - **Never** install to `/usr/local/bin` — that path requires sudo and creates stale-root-owned-binary problems that are painful to debug

3. **Standardized build flags:**
   - `-trimpath` — deterministic fingerprints across machines (absolute paths get stripped)
   - `-ldflags "-s -w -X main.buildFingerprint=... -X main.gitCommit=... -X main.buildTime=..."` — small binary, embedded identity
   - `CGO_ENABLED=0` — pure Go, painless cross-compile
   - `-buildvcs=true` — Go records VCS info automatically

4. **Binary output convention:**
   - `~/.vrooli/bin/` for installed binaries
   - `.vrooli/build/` (gitignored) for dev builds
   - **Nothing in the source tree.** Keeps `git status` clean.

### For the `vrooli-api` server (long-running)

5. **Fail-fast on stale, don't self-rebuild.** An HTTP server can't safely re-exec mid-request. On boot:
   - Log the fingerprint
   - If `VROOLI_STRICT_FINGERPRINT=1` is set, compare to source and exit 1 with a clear message when stale
   - Let `vrooli develop` (the supervisor) be the thing that rebuilds and restarts the server in dev

6. **Hot-reload in dev mode only.** `vrooli develop` can watch `path:cmd/vrooli-api/**/*.go` and `path:internal/**/*.go`, rebuild, bounce the server. Prod/install path never watches.

### Cross-cutting

7. **Cross-compile matrix from Week 5.** `make release` or a CI workflow builds `platform:linux/amd64`, `platform:linux/arm64`, `platform:darwin/arm64`, `platform:darwin/amd64`, `platform:windows/amd64`. Trivial in Go once `CGO_ENABLED=0` is established. Catches platform drift before users do.

8. **Makefile targets** (baseline — add to `/Makefile` in Week 0):
   ```makefile
   LDFLAGS := -s -w \
       -X github.com/vrooli/vrooli/internal/buildinfo.GitCommit=$(shell git rev-parse HEAD) \
       -X github.com/vrooli/vrooli/internal/buildinfo.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

   build:
       VROOLI_CLI_FINGERPRINT=$$(go run ./cmd/vrooli-buildmeta --root . cmd/vrooli internal); \
           CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS) -X github.com/vrooli/vrooli/internal/buildinfo.Fingerprint=$$VROOLI_CLI_FINGERPRINT" -o .vrooli/build/vrooli ./cmd/vrooli
       VROOLI_API_FINGERPRINT=$$(go run ./cmd/vrooli-buildmeta --root . cmd/vrooli-api internal); \
           CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS) -X github.com/vrooli/vrooli/internal/buildinfo.Fingerprint=$$VROOLI_API_FINGERPRINT" -o .vrooli/build/vrooli-api ./cmd/vrooli-api

   install: build
       install -m 0755 .vrooli/build/vrooli ~/.vrooli/bin/vrooli
       install -m 0755 .vrooli/build/vrooli-api ~/.vrooli/bin/vrooli-api
   ```

---

## 5. Week-by-week migration plan

Strangler pattern. Each week delivered value standalone while the Bash fallback existed. That fallback is now retired from the Go CLI, so the remaining Week 6 work is deleting and collapsing the leftover compatibility surface.

### Week 0 — Foundation (implemented)

**Goal:** Get the scaffolding in place so every subsequent week is additive, not restructuring.

- [x] Create `/go.mod` at repo root with module path `github.com/vrooli/vrooli`
- [x] Move `path:api/main.go` → `path:cmd/vrooli-api/main.go`; verify `vrooli develop` still launches it correctly
- [x] Stand up `path:internal/buildinfo`:
  - `Fingerprint`, `GitCommit`, `BuildTime` vars populated by ldflags
  - `ComputeSourceFingerprint(rootDir string) (string, error)` — SHA256 over `.go` files under given root, sorted by path
  - `IsStale() bool` — compares embedded fingerprint to computed
  - `RebuildAndReexec(argv []string) error` — invokes `go build` + `syscall.Exec`
- [x] Stand up `path:internal/logx` — thin wrapper around `package:log/slog`, level from `VROOLI_LOG_LEVEL` env
- [x] Stand up `path:internal/cliout` — human-vs-JSON output helpers (respects `--json`, default human per feedback memory), table rendering, color detection
- [x] Add `/Makefile` with `build`, `install`, `test`, `clean` targets using the ldflags pattern from Section 4
- [x] Add `.vrooli/build/` to `.gitignore`

**Deliverable:** Repo structure ready. Running `make build && make install` produces `~/.vrooli/bin/vrooli-api` identical in behavior to the old `path:api/vrooli-api`. Nothing user-visible changed.

**Acceptance:** `vrooli develop` still works end-to-end.

---

### Week 1 — The dispatcher shim (implemented)

**Goal:** Every `vrooli` invocation goes through Go, even though Go still shells to Bash for 100% of subcommands. The stale-check concern gets solved before a single line of real bash is ported.

- [x] Build `path:cmd/vrooli/` as a pure pass-through dispatcher:
  - Parses top-level args (global flags like `--verbose`, `--json`, `--no-color`)
  - For each subcommand, routes to `exec.Command("bash", "cli/commands/<appropriate-handler>.sh", args...)`
  - Inherits stdin/stdout/stderr, propagates exit code
- [x] Wire in the `path:internal/buildinfo` stale-check on startup: if source fingerprint ≠ embedded, call `RebuildAndReexec`
- [x] Support `VROOLI_FORCE_BASH=1` — when set, skip Go-level logic and exec the old `path:cli/vrooli` directly. This was the temporary migration escape hatch and is now retired in Week 6.
- [x] Update the setup script to install `~/.vrooli/bin/vrooli` (the Go binary) as the entry point, instead of symlinking the old Bash `path:cli/vrooli`
- [x] Keep `path:cli/vrooli` in place temporarily as a compatibility shim while direct callers are migrated
- [x] Integration test: run `vrooli scenario list` through the new Go dispatcher; verify identical output to the old Bash path

**Deliverable:** As of this week, "I edited code and forgot to rebuild" is impossible. Every `vrooli` invocation verifies freshness. The #1 concern from this conversation is solved.

**Acceptance:** Modify a `.go` file in `internal/`, run `vrooli scenario list` → binary rebuilds automatically, runs fresh code.

---

### Week 2 — Read-only scenario commands (prove the pattern)

**Goal:** Migrate the easiest, safest subcommands end-to-end to validate the shared packages and the per-subcommand cut-over pattern.

- [x] `path:internal/scenario` package:
  - Typed `service.json` v2.0 reader (struct-based, not `map[string]any`)
  - Directory-scan discovery of `path:scenarios/*/.vrooli/service.json`
  - **Sandbox-aware path resolution** — preserve the `VROOLI_SANDBOX_SCOPE` logic from `path:scripts/lib/scenario/runner.sh` lines 47–98. This is the subtlest bit of Week 2; get it right up front.
- [x] `path:internal/process` package:
  - Read PID files under `~/.vrooli/processes/scenarios/<name>/`
  - Check liveness via `os.FindProcess` + signal 0
  - Enumerate running scenarios
- [x] Migrate subcommands (one PR each):
  - `vrooli scenario list`
  - `vrooli scenario info <name>`
  - `vrooli scenario status [name]`
- [x] Route migrated subcommands through Go while the remaining legacy commands still lived in Bash during the migration
- [x] Unit tests for `path:internal/scenario` and `path:internal/process`

**Deliverable:** ~3 subcommands in Go. Pattern is proven. First shared packages exist.

**Acceptance:** `vrooli scenario list` output is intentionally improved compared to the old Bash version: it now filters strictly to `path:scenarios/*/.vrooli/service.json`, correctly reports running scenarios from PID metadata, and `vrooli scenario status` is also CLI-native now without requiring the project API to be reachable.

**Note on the first week of migration:** Week 2 often takes longer than it looks — not because the code is hard, but because you're designing the shared `internal/` APIs for the first time. Accept that. Do not cargo-cult the first design into later packages; expect to refactor the API shape in Week 3.

---

### Week 3 — Lifecycle executor + stop/start/restart (the hardest week)

**Goal:** The 3k-LOC `vrooli scenario restart` call path moves to Go. This is the biggest single bash chunk and the most load-bearing.

- [x] `path:internal/lifecycle` package:
  - Phase runner: `setup`, `develop`, `build`, `test`, `stop`
  - Condition evaluator: `file_exists`, `directory_exists`, `resource_enabled`, etc.
  - Step executor: runs the `run` bash field of each step (yes, we still exec bash for user-defined scenario steps — scenarios are out of scope)
  - Health-check polling with timeout + backoff
  - Port from `path:scripts/lib/utils/lifecycle.sh` (1,049 LOC)
- [x] `path:internal/ports` package:
  - Registry file read/write under the existing `~/.vrooli/state/scenarios/.port_*.lock` contract with safe concurrent claims
  - `net.Listen` probing to detect occupied ports (cross-platform replacement for `lsof`)
  - Port allocation with sticky assignments per scenario
  - Port from `path:scripts/lib/network/ports.sh` (898 LOC)
- [x] Migrate in this order (each its own PR):
  1. `vrooli scenario stop` — simpler: signal + port release
  2. `vrooli scenario start` — complex: discovery + phase execution + health checks
  3. `vrooli scenario restart` — composition of stop + start
- [x] Repeatable validation: temp-fixture lifecycle smokes for `start`/`restart`/`stop`, plus real-manifest contract coverage for `test-genie` and `simple-test`
- [ ] Remove the now-unused Week 3 Bash implementations only after the remaining legacy wrappers and non-migrated shell consumers stop sourcing them

**Deliverable:** The call path from the original conversation is entirely Go. The biggest single bash chunk is gone.

**Acceptance:** `make validate-week3` is green and the migrated Go path preserves the existing process-state, lifecycle-log, and port-lock contracts under `~/.vrooli/processes/`, `~/.vrooli/logs/`, and `~/.vrooli/state/scenarios/`. A full live `test-genie` smoke remains a manual validation step because its setup path installs scenario dependencies.

**Risk note:** Budget 1.5× time for this week. The bash has accumulated edge cases over years (sandbox resolution, health-check timing, port lease renewal, stale PID detection). If Week 3 slips, Weeks 4–6 compress because they reuse the same packages — the real schedule risk is front-loaded here.

---

### Week 4 — Remaining `vrooli scenario *` commands + utilities (implemented)

**Goal:** Finish the scenario command family and remove the default Go CLI dependency on `path:cli/commands/scenario/`.

- [x] Migrate remaining scenario subcommands:
  - `vrooli scenario test <name>`
  - `vrooli scenario logs <name>`
  - `vrooli scenario run`, `start-all`, `setup`, `stop-all`
  - `vrooli scenario port`, `open`
  - `vrooli scenario requirements`, `ui-smoke`
  - `vrooli scenario template`, `generate`
  - `vrooli scenario completeness`, `heal-from-sandbox`
- [x] `path:internal/config` package:
  - `.vrooli/` path resolution (repo-local, home-level)
  - Env var handling with defaults
- [x] Delete `path:cli/commands/scenario/` entirely once the remaining legacy top-level wrappers and direct shell consumers are retired

**Deliverable:** `vrooli scenario --help` is entirely backed by Go handlers in the default path.

**Acceptance:** `vrooli scenario --help` lists the full scenario command surface, `make validate-week4` is green, and the repeatable suite covers native setup/test/logs/open/port/template/requirements/heal flows plus the thin CLI wrappers.

---

### Week 5 — Setup + runtime (platform-gated, implemented)

**Goal:** Port the setup story. This is where cross-platform support starts to actually matter.

- [x] `path:internal/runtime` package:
  - Go, Node, Python, Docker, Helm detection + install paths
  - Platform-gated implementations:
    - `runtime_linux.go` — apt, sysctl, systemd
    - `runtime_darwin.go` — brew, stub sysctl
    - `runtime_windows.go` — mostly stubs with clear "unsupported" errors
- [x] `path:internal/setup` package:
  - Port `path:scripts/manage.sh setup` logic
  - Delegates platform-specific bits to `path:internal/runtime`
- [x] Migrate subcommands:
  - `vrooli setup`
  - `vrooli develop` (the outer wrapper; the API server itself is already Go)
- [x] Delete `path:scripts/manage.sh` and the Linux-specific bits of `path:scripts/lib/system/` that were only project-level entrypoint debt
- [x] **Try the cross-compile.** Run `GOOS=darwin go build ./cmd/vrooli` and `GOOS=windows go build ./cmd/vrooli`. The compiler will honestly report everything still Linux-coupled — that becomes the punch list for Week 6 and beyond.

**Deliverable:** `path:scripts/manage.sh` deleted. First concrete cross-platform capability: you have a real punch list of what's blocking macOS/Windows.

**Acceptance:** `GOOS=linux go build ./cmd/vrooli` succeeds. `GOOS=darwin go build ./cmd/vrooli` either succeeds or fails with a list of platform-specific issues you understand.

---

### Week 6 — Long tail + cleanup (project-level implementation complete; shared cleanup remains)

**Goal:** Retire the remaining project-level Bash entrypoints and shrink the legacy boundary to the genuinely unmigrated platform-specific long tail.

- [x] `path:internal/secrets` package:
  - Encrypted read/write of `~/.vrooli/` secret files
  - AES-GCM or age (pick one; age is simpler and more auditable)
  - Port from `path:scripts/lib/service/` (~1,055 LOC)
- [x] `path:internal/network` package:
  - Start with the maintenance-facing slice first: lock parsing/listing plus listener inspection used by `locks` and `diagnose-port`
  - Keep it Linux-gated where process/listener inspection is platform-specific; use stubs elsewhere
  - Do not port the entire `path:scripts/lib/network/` tree unless a live project-level command still depends on it
- [x] Remaining top-level `vrooli` work:
  - Harden validation for the already-native commands: `vrooli status`, `doctor`, `info`, `clean`, `stop`, `backup`, `restore`, `orphans`, `locks`, `diagnose-port`
  - Remove their residual dependence on the Bash escape hatch and legacy shims
- [ ] Delete `path:scripts/lib/` entirely once the remaining shared resource/scenario consumers are removed or relocated
- [x] Delete `cli/` entirely
- [x] Delete `path:cli/vrooli` once the remaining direct callers stop execing it and the compatibility shim is no longer needed
- [x] Remove the `VROOLI_FORCE_BASH` escape hatch from `path:cmd/vrooli/main.go`

**Deliverable:** No project-level Bash remains. `wc -l cli/**/*.sh scripts/**/*.sh 2>/dev/null` returns zero (or only scenario-internal bash, which is out of scope).

**Acceptance:** A fresh checkout can run `make install && vrooli setup && vrooli develop` on Linux without touching any `.sh` file in `cli/` or `path:scripts/lib/`.

---

## 6. Cross-cutting principles

Apply these throughout all six weeks:

1. **One PR per subcommand migration.** Each PR:
   - Adds the Go handler
   - Wires it into the dispatcher
   - Deletes the Bash handler
   - Adds tests
   - Is independently reviewable and revertable

2. **Delete the Bash in the same PR as the Go replacement when the deleted file is no longer a shared dependency of non-migrated commands.** The temporary `VROOLI_FORCE_BASH` safety net is gone now. Do NOT keep parallel code paths "just in case" in the tree — but also do not delete shared Bash helpers that still have live consumers outside the project CLI.

3. **Integration-test each migrated command against a real scenario.** Use `test-genie` (small, fast) as the acceptance bar. Run the command through old Bash and new Go paths; compare outputs and state.

4. **Fingerprint check stays green.** After every merge, a fresh `vrooli <subcmd>` should trigger exactly one rebuild+re-exec and then run clean. If it loops, the fingerprint algorithm has a bug.

5. **No cross-platform work until Week 5.** Building for Linux only until then keeps scope manageable. Windows/macOS becomes a punch list the compiler generates for you, not speculative design upfront.

6. **Don't try to get to zero bash by a hard deadline.** If Week 6 leaves 2k LOC of obscure bash (exotic network diagnostics, rarely-used backup routines), that's fine. Port it opportunistically afterward. 95% of the value is in the scenario commands + setup + lifecycle, which land by Week 5.

7. **Greenfield over compat shims.** Per existing project feedback guidelines: don't add backwards-compatibility layers. When you port, port cleanly. The old `VROOLI_FORCE_BASH` escape hatch was the only major compat concession, and it is now gone.

8. **Do not touch scenarios.** Scenarios are out of scope. If you find yourself editing anything under `path:scenarios/*/`, stop — that's not this plan.

---

## 7. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Week 3 lifecycle port has subtle edge cases (sandbox path resolution, health-check timing, port lease renewal) | Schedule slip, incorrect behavior | Budget 1.5× time. Integration-test against `test-genie` and at least one scenario with resource dependencies. Preserve the legacy wrappers until the native validation slices are broad enough to replace parity confidence. |
| First Go package API (`path:internal/scenario` in Week 2) gets designed poorly | Downstream packages cargo-cult the bad shape | Expect to refactor in Week 3. Do not freeze the API after Week 2. |
| Cross-platform ambitions creep into early weeks | Schedule slip, scope explosion | Platform work is gated to Week 5. Earlier weeks target Linux only. |
| Stale-check fingerprint algorithm has a bug causing infinite rebuild loops | User experience broken, hard to diagnose | Unit test the fingerprint algorithm in Week 0. Include a `--no-stale-check` flag for debugging. |
| Bash handlers deleted prematurely, Go handler has a regression | User workflow broken | Gate deletion on repeatable integration coverage and live native smokes. Retain only narrow compatibility shims that still have known callers. |
| "A few weeks" is optimistic | Plan stretches to 10+ weeks | Acceptable — the incremental delivery means value lands each week regardless. Don't compress by skipping tests. |

---

## 8. Open questions

These don't block starting the work, but should be resolved as they come up:

- ~~**Module path.**~~ **DECIDED:** Use `github.com/vrooli/vrooli` (all lowercase, matching the convention of the existing scenario shared packages like `github.com/vrooli/cli-core`). Rationale: it's the standard Go convention (module paths should be fetchable VCS URLs), avoids the operational burden of serving vanity-import meta tags, and stays consistent with the existing `path:packages/cli-core` and `path:packages/api-core` module paths. The existing `path:api/go.mod` using `vrooli.com/api` is the outlier and gets fixed during the Week 0 move from `api/` → `path:cmd/vrooli-api/`.
- **`path:packages/api-core` reuse for `path:cmd/vrooli-api`.** Does the project-level API server benefit from importing the scenario-facing `path:packages/api-core`, or should it use its own HTTP lifecycle from `internal/`? Recommendation: use `path:packages/api-core/server` since it already handles graceful shutdown and signals correctly, and it's a one-way dependency (project uses scenario lib, not vice versa).
- **Secrets encryption scheme.** age vs AES-GCM vs reuse whatever Bash uses today. Decide in Week 6 when you start on `path:internal/secrets`.
- **Windows story.** Is Windows a first-class target, or a "best effort" target where certain commands are unavailable? This affects how much effort goes into `path:internal/runtime/runtime_windows.go`. Recommendation: second-class for now — `vrooli develop` against containerized resources should work, but `vrooli setup` can be "Linux only, use WSL on Windows for setup" initially.

---

## 9. Progress tracker

Update this section as work lands. Check boxes for completed items, note the PR or commit reference and date when available.

### Week 0 — Foundation
- [x] `/go.mod` created
- [x] `path:api/main.go` → `path:cmd/vrooli-api/main.go` moved
- [x] `path:internal/buildinfo` package shipped
- [x] `path:internal/logx` package shipped
- [x] `path:internal/cliout` package shipped
- [x] `/Makefile` with standardized build targets
- [x] `.vrooli/build/` in `.gitignore`

Validation note: As of 2026-04-10, the repeatable acceptance target `make validate-week0-week1` is green from a fresh `agi` worktree based on `master`. For Week 0 specifically, that target now actually covers `make clean`, `make build`, `make install`, `make test`, a direct `path:api/start.sh` health smoke on `VROOLI_API_PORT=18092`, and a fresh `VROOLI_API_PORT=18093 vrooli develop` smoke that reached a healthy API and completed the `vrooli-orchestrator` startup flow. The `vrooli develop` validation now performs its cleanup inside the target via `cd scenarios/vrooli-orchestrator && make stop`, with `VROOLI_ROOT` and `VROOLI_SOURCE_ROOT` pinned to the active checkout so stale-check rebuilds resolve against the correct worktree. Reaching a green `vrooli develop` still depends on the unrelated `vrooli-orchestrator` fixes discovered during the original validation: correcting the postgres initialization type to `seed`, making the schema/seed SQL idempotent, aligning the UI lifecycle steps with `pnpm`, and removing a runtime profile-creation step that depended on the orchestrator being up before setup had finished.

### Week 1 — Dispatcher shim
- [x] `path:cmd/vrooli/` pass-through dispatcher
- [x] Stale-check + rebuild+re-exec wired in
- [x] `VROOLI_FORCE_BASH` escape hatch (historical milestone; retired in Week 6)
- [x] Setup script updated to install Go binary
- [x] Integration test: `vrooli scenario list` via Go dispatcher

Validation note: As of 2026-04-11, the repeatable acceptance target `make validate-week0-week1` is green. For Week 1 specifically, that target covers `~/.vrooli/bin/vrooli --version`, a temp-`HOME` install smoke via `make install`, and a stale-check smoke that appends a temporary comment to `path:cmd/vrooli/main.go`, verifies the installed binary checksum changes on the next invocation, and then verifies the checksum stabilizes on the following invocation. The installed-binary smokes pin `VROOLI_ROOT` and `VROOLI_SOURCE_ROOT` to the active checkout so the stale-check works from fresh worktrees. The checksum-based stale smoke replaced the older `stat` mtime check because filesystem timestamp granularity can hide a real rebuild when both runs happen within the same second.

### Week 2 — Read-only scenario commands
- [x] `path:internal/scenario` package (with sandbox awareness)
- [x] `path:internal/process` package
- [x] `vrooli scenario list` migrated
- [x] `vrooli scenario info` migrated
- [x] `vrooli scenario status` migrated
- [x] Unit tests for `path:internal/scenario`, `path:internal/process`

Validation note: As of 2026-04-11, the repeatable acceptance targets `make validate-week2` and `make validate-week0-week2` are green. Week 2 validation now provisions a temp fixture repo plus a temp `HOME`, starts disposable HTTP servers with process metadata under `~/.vrooli/processes/scenarios/alpha/`, verifies `scenario list --json` only discovers directories with `.vrooli/service.json`, verifies `scenario list --json --include-ports` reports live port metadata from process files, verifies `scenario info alpha --json` honors `VROOLI_SANDBOX_MERGED` + `VROOLI_SANDBOX_SCOPE=scenarios/alpha`, and verifies `scenario status alpha --json` stays healthy with `VROOLI_API_PORT=1` (proving it no longer depends on the project API).

### Week 3 — Lifecycle executor + stop/start/restart
- [x] `path:internal/lifecycle` package
- [x] `path:internal/ports` package
- [x] `vrooli scenario stop` migrated
- [x] `vrooli scenario start` migrated
- [x] `vrooli scenario restart` migrated
- [x] Repeatable lifecycle validation landed (`make validate-week3`, `make validate-week0-week3`, plus real-manifest contract coverage for `test-genie` and `simple-test`)
- [ ] Week 3 Bash helpers retired once the remaining legacy wrappers, runtime shims, and shell consumers stop sourcing them

Validation note: As of 2026-04-11, the repeatable acceptance targets `make validate-week3` and `make validate-week0-week3` are green. Week 3 validation builds, installs, and tests the Go binaries; provisions a temp fixture repo plus a temp `HOME`; verifies native `scenario start alpha --json`, `scenario restart alpha --json`, and `scenario stop alpha --json`; confirms health polling, process metadata under `~/.vrooli/processes/scenarios/alpha/`, lifecycle logs under `~/.vrooli/logs/alpha.log`, rotated per-step logs under `~/.vrooli/logs/scenarios/alpha/`, and port locks under `~/.vrooli/state/scenarios/.port_*.lock`; and exercises the current dependency manifest schemas plus real-manifest environment/port contracts for `test-genie`, `simple-test`, and legacy-array scenarios like `code-smell`. `make validate-week3-live` is now a native-only live smoke against `test-genie` and `reference-react-vite` rather than a Bash-vs-Go parity harness, and it should be treated as a separate live integration check because it is slower and more environment-sensitive than the repeatable fixture suite.

### Week 4 — Remaining scenario commands + config
- [x] `vrooli scenario test` migrated
- [x] `vrooli scenario logs` migrated
- [x] `vrooli scenario run`, `start-all`, `setup`, `stop-all` migrated
- [x] `vrooli scenario port`, `open` migrated
- [x] `vrooli scenario requirements`, `ui-smoke` migrated
- [x] `vrooli scenario template`, `generate` migrated
- [x] `vrooli scenario completeness`, `heal-from-sandbox` migrated
- [x] `path:internal/config` package
- [x] `path:cli/commands/scenario/` deleted after the remaining legacy top-level wrappers and direct shell consumers were removed

Validation note: As of 2026-04-10, the acceptance targets `make validate-week4` and `make validate-week0-week4` are green. Week 4 validation now builds, installs, and tests the Go binaries; reruns the full project-level Go suite; runs focused CLI coverage for native `scenario setup`, `test`, `port`, `open`, `logs --clean`, `cli:template/generate`, `requirements snapshot`, and `heal-from-sandbox`; and confirms the installed binary exposes the expanded native command surface in `scenario --help` (`start-all`, `stop-all`, `template`, `generate`, `heal-from-sandbox`, and the rest of the remaining scenario utilities). Thin wrapper commands such as `ui-smoke`, `requirements report`, and `completeness` are now exercised through the Go dispatcher and covered by unit tests that assert the translated subprocess invocations. The remaining cleanup is deleting the legacy shell command tree once the last direct consumers are removed; the Go CLI no longer retains a runtime fallback path.

### Week 5 — Setup + runtime
- [x] `path:internal/runtime` package (platform-gated)
- [x] `path:internal/setup` package
- [x] `vrooli setup` migrated
- [x] `vrooli develop` (outer wrapper) migrated
- [x] `path:scripts/manage.sh` deleted
- [x] Cross-compile attempt for darwin/windows; punch list captured here:
  - `GOOS=linux GOARCH=amd64`, `GOOS=darwin GOARCH=amd64`, `GOOS=darwin GOARCH=arm64`, and `GOOS=windows GOARCH=amd64` now all compile for `./cmd/vrooli`
  - Compiler punch list for `path:cmd/vrooli` is now empty; the remaining cross-platform gap is runtime support, not buildability
  - `path:internal/runtime` still reports `vrooli setup`/`vrooli develop` unsupported on darwin/windows because resource orchestration and scenario lifecycle support remain Linux-oriented even though the repo-root project wrapper is now native

Validation note: As of 2026-04-12, the week-5-specific acceptance slices are green: `make validate-week5-cross` passes, and `make validate-week5` now scopes its package test run to the week-5-owned Go surfaces (`path:cmd/vrooli`, `path:cmd/vrooli-api`, `path:internal/runtime`, and `path:internal/setup`) before running the focused native CLI/setup assertions. Week 5 validation now builds, installs, and tests the project-level Go binaries; runs focused native CLI coverage for top-level `vrooli setup`, `vrooli develop`, `vrooli backup`, and `vrooli restore`; verifies the default bootstrap path by installing into a temp `HOME` and running `make setup SETUP_ARGS=--help` through the Go CLI; reuses the live repo-root `vrooli develop` smoke through the installed binary so the actual API/orchestrator startup path still reaches a healthy API and cleans up afterward; and verifies `./cmd/vrooli` cross-compiles for Linux, Darwin, and Windows. This narrower package scope intentionally avoids unrelated failures in other migration tracks from masking week 5 regressions. The native `path:internal/setup` path no longer shells through the repo-root `setup`/`develop` lifecycle definitions in `.vrooli/service.json`; it performs project setup, runtime probing, API launch, health checks, and orchestrator startup directly in Go, and the later greenfield cleanup removed the temporary shell-era environment export bridge that had been kept only for migration compatibility. `make setup` now bootstraps by running `make install` and then invokes the installed Go CLI. The historical `path:scripts/manage.sh` compatibility shim has now been removed.

### Week 6 — Long tail + cleanup
- [x] `path:internal/secrets` package
  Current scope: encrypted read/write, legacy plaintext fallback, migration helpers, and resource-environment loading are native and validated; shell-only template substitution, config migration helpers, and vault-specific behavior still live under `path:scripts/lib/service/secrets.sh` for resource consumers.
- [x] `path:internal/network` package
  Current scope: maintenance-facing lock parsing/listing plus listener inspection used by `locks` and `diagnose-port` are now extracted into `path:internal/network`; broader firewall/diagnostics parity remains future work only if a live project command still needs it.
- [x] Native project-level command validation broadened for `status`, `doctor`, `info`, `clean`, `build`, `deploy`, `stop`, `backup`, `restore`, `orphans`, `locks`, and `diagnose-port`
  Current coverage: `make validate-week6-slice` exercises those native commands through the installed binary against a temp fixture project and validates native lock cleanup plus build/clean/deploy/backup/restore dispatch.
- [x] `path:scripts/manage.sh` collapsed to a pure Go forwarding shim, then deleted
- [x] Dead top-level shell handlers under `path:cli/commands/` and `path:cli/lib/` deleted
- [ ] `path:scripts/lib/` deleted
  Deferred: this is now primarily shared resource/runtime cleanup rather than missing project-level Go orchestration work.
- [x] `cli/` deleted
- [x] `path:cli/vrooli` compatibility shim deleted after direct callers such as `path:scenarios/task-planner/api/task_parser.go` and `path:scenarios/elo-swipe/api/smart_pairing.go` stopped execing `/vrooli/cli/vrooli`
- [x] `VROOLI_FORCE_BASH` escape hatch removed

Validation note: As of 2026-04-12, both `make validate-week6-slice` and `make validate-week6-secrets` are green. The week-6-specific validation now builds and installs the project-level Go binaries, verifies the root bootstrap path no longer depends on `path:cli/install.sh` or `path:scripts/manage.sh`, exercises the native maintenance command slice (`status`, `doctor`, `info`, `clean`, `build`, `deploy`, `stop`, `backup`, `restore`, `orphans`, `locks`, and `diagnose-port`) through the installed binary, and keeps the encrypted project-secrets plus resource-environment loading path covered while broader shell-side secrets/template parity remains future work for resource consumers. The acceptance target no longer sources `path:scripts/lib/scenario/runner.sh`; stale-lock cleanup is validated through the native CLI/test surface instead. The `path:internal/project` status aggregation fixture is now hermetic against ambient installed `resource-*` binaries on `PATH`, so this slice no longer depends on workstation-specific resource installs. The top-level lifecycle commands now honor `--help` without accidentally executing their phases.

### Post-migration
- [ ] Zero `.sh` files remaining under `cli/` and `path:scripts/lib/` (confirmed via `find`)
- [ ] Cross-platform CI matrix green for linux/amd64, linux/arm64, darwin/arm64
- [ ] Documentation updated (CLAUDE.md, README, deployment docs)
