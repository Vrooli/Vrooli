# Project-Level Bash → Go Migration Plan

**Status:** Not yet started
**Owner:** Matthew Halloran
**Scope:** Project-level orchestration only. Scenarios are explicitly out of scope.
**Target:** Zero project-level bash in ~6 weeks, on a path to cross-platform support.

---

## 0. For agents picking this up later

If you are an agent orienting to this plan, read this section first.

- **What this plan covers:** Porting `/cli/`, `/scripts/lib/`, and `/scripts/manage.sh` from Bash to Go. This is the *orchestrator* (the `vrooli` CLI and its supporting libraries), not the scenarios it manages.
- **What this plan does NOT cover:** Anything under `scenarios/*/`. Scenarios are already on their own cross-platform path via `packages/cli-core` and `packages/api-core`. Do not touch scenario internals as part of this work.
- **How to find current progress:** Check the "Progress Tracker" section at the bottom. Each week has a checklist. Also run `git log --oneline --grep="bash-to-go"` to see landed commits.
- **How to resume work:** Find the first unchecked item in the Progress Tracker, re-read its Week section for context and deliverables, and start there. Each week's PRs are designed to be independently reviewable and revertable.
- **Escape hatch:** Throughout the migration, the env var `VROOLI_FORCE_BASH=1` should route any subcommand back to its original bash handler if the Go path misbehaves. Remove this only in Week 6.
- **If you find stale information in this plan:** Update it. The plan is a living document; the repo's current state is always authoritative over what's written here.

---

## 1. Motivation

### The problem
Vrooli's project-level orchestration is currently ~33,335 LOC of Bash:
- `cli/` — 11,278 LOC (the `vrooli` user-facing CLI and subcommand handlers)
- `scripts/lib/` — 21,779 LOC (lifecycle executor, port management, runtime setup, networking, secrets, process tracking)
- `scripts/manage.sh` — 278 LOC (setup/develop/build/deploy router)

Bash has three properties that are biting us:

1. **Not portable.** Windows has no native Bash. WSL exists but is an extra install and a compatibility layer, not a first-class experience. macOS has Bash but many lifecycle routines depend on Linux-specific tools (`apt`, `systemctl`, `/proc`, `iptables`, `bwrap`, `ip netns`). Roughly 30% of current bash is genuinely Linux-kernel-specific and unportable regardless of language choice; the remaining 70% is "thin glue" that *could* port cleanly to a cross-platform language.
2. **Unreliable at scale.** Bash error handling is hard to get right, tests are awkward (bats), and refactoring is risky because there's no type system catching drift.
3. **Hard to test.** There's no unit-test culture for ~22k LOC of library code. Most bugs get caught by end-to-end scenario runs, which is slow feedback.

Newer scenarios in this repo have already proven that Go is a workable replacement — `packages/cli-core` and `packages/api-core` are shared Go libraries consumed by 27+ scenarios, and the pattern works. The project level should follow the same trajectory.

### The goal
Port `/cli/`, `/scripts/lib/`, and `/scripts/manage.sh` to Go, organized as a single top-level Go module with multiple commands and shared `internal/` packages. Outcome:

- `vrooli` becomes a single statically-linked Go binary
- Cross-platform compilation works for Linux, macOS, and (with platform-gated features) Windows
- Lifecycle code is unit-testable
- "Running stale code" becomes impossible via build-time fingerprinting + startup stale check
- Expected size reduction: ~33k LOC Bash → ~10–12k LOC Go (2–3× compression is typical for this kind of port)

### Why Go, not Rust or C
- **Go:** Team already writes it. `packages/cli-core` and `packages/api-core` prove the pattern. Stdlib covers JSON/HTTP/process/signals/cross-compile. Single static binary per platform. `CGO_ENABLED=0` gives effortless cross-compilation.
- **Rust:** Would work technically, but the team doesn't use it. Switching cost for glue code outweighs the marginal safety benefit.
- **C:** No JSON/HTTP/process stdlib worth mentioning, poor memory safety for orchestration code, wrong tool.

### Why not "just wrap bash in WSL/Docker for Windows"
That's the compatibility-layer path. It works but it's fragile, adds install friction, and doesn't solve the testability/reliability problems. A real Go port solves all three concerns (portability, reliability, testability) at once.

---

## 2. Current state (as of 2026-04-10)

### Project-level Bash inventory
| Path | LOC | Purpose |
|---|---|---|
| `cli/vrooli` | 474 | Top-level CLI dispatcher |
| `cli/commands/*.sh` | 10,348 | Command handlers (scenario, resource, status, doctor, etc.) |
| `cli/lib/*.sh` | 580 | CLI-specific utilities |
| `scripts/manage.sh` | 278 | Lifecycle router |
| `scripts/lib/utils/` | 5,718 | Arg parsing, JSON, logging, config, retry logic |
| `scripts/lib/utils/lifecycle.sh` | 1,049 | **Lifecycle executor — core of the system** |
| `scripts/lib/network/` | 2,473 | Port allocation, firewall, SSH, diagnostics |
| `scripts/lib/network/ports.sh` | 898 | **Port allocation — flock-based registry** |
| `scripts/lib/system/` | 2,558 | Kernel config, system checks, clock sync, trash |
| `scripts/lib/service/` | 1,603 | Secrets, cloudflare, repos |
| `scripts/lib/resources/` | 720 | Resource orchestration, validation |
| `scripts/lib/scenario/runner.sh` | 668 | Scenario discovery + phase execution |
| `scripts/lib/process-manager.sh` | 154 | PID tracking |
| **Total** | **~33,335** | |

### Existing project-level Go (the foothold)
- **`/api/main.go`** — 2,026 LOC, module `vrooli.com/api`, gorilla/mux, compiles to `/api/vrooli-api`
  - Purpose: project-level HTTP orchestration server invoked from `vrooli develop`
  - Exposes: `/scenarios/start-all`, `/scenarios/start/<name>`, scenario listing, health checks
  - Currently delegates most real work back down to bash — **this is the seed crystal for the migration**
- **`/scripts/main/`** — NOT Go. Contains `check-cli-binaries.sh` (92 LOC bash linter) and a stray compiled scenario binary. Ignore; the name is misleading.
- No top-level `cmd/`, `internal/`, or `pkg/` directories yet.

### The `vrooli scenario restart` call path (3,000 LOC of bash, concrete example)
When a user runs `vrooli scenario restart foo`:
1. `cli/vrooli` (474 LOC) parses top-level args, routes to scenario handler
2. `cli/commands/scenario/scenario-commands.sh` (167 LOC) dispatches to `restart` subcommand
3. `cli/commands/scenario/modules/lifecycle.sh` (294 LOC, restart is ~24 of them) calls stop then start
4. `scripts/lib/utils/lifecycle.sh` (1,049 LOC) handles process termination, port release
5. `scripts/lib/scenario/runner.sh` (668 LOC) re-sources the scenario's `.vrooli/service.json` and re-runs its phases

This is the canonical "hard case" — when you test the Go port, test this path first.

### Scenario discovery contract
There is **no registry file**. Discovery is pure directory scan:
- `scripts/lib/scenario/runner.sh::scenario::list()` iterates `$VROOLI_ROOT/scenarios/*/`
- Each candidate must contain `.vrooli/service.json`
- Process tracking lives in `~/.vrooli/processes/scenarios/<name>/*.json` and `*.pid`
- Sandbox-aware: if `VROOLI_SANDBOX_SCOPE` is set, paths resolve through the sandbox's overlay `merged/` directory (lines 47–98 of `runner.sh`). The Go port must preserve this behavior.

### What already has a good pattern worth copying
- **`packages/cli-core/buildinfo` + `cliutil.StaleChecker`** — scenario CLIs embed a source fingerprint via `-ldflags` at build time, and re-exec through `cli-installer` when the fingerprint drifts. This exact pattern should land at the project level for `vrooli`. See Section 4.
- **`packages/api-core/server`** — clean HTTP server lifecycle with signal handling and graceful shutdown. `cmd/vrooli-api/main.go` can consume it directly.

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
Two separate modules (`cmd/vrooli` and `cmd/vrooli-api`) would re-create the duplication problem we already have in bash. Single module with shared `internal/` means:
- Both binaries can't drift
- One `go build ./...` builds everything
- Type-level refactors work across both CLI and API at once
- Shared lifecycle code has exactly one home

### What does NOT go in the top-level module
- Anything scenario-facing. Scenarios consume `packages/cli-core` and `packages/api-core` — those are their ABI and must stay separate. Conflating them with project orchestration code would couple scenario stability to internal orchestrator changes.

---

## 4. Compile best practices

The fundamental shift from Bash to Go is: Bash reads source on every invocation, Go compiles once and runs the binary. The risk is running stale binaries after source changed. Here's how to eliminate that risk entirely.

### For the `vrooli` CLI binary (short-lived, invoked on demand)

1. **Self-healing stale check on every invocation.** Same pattern the scenario CLIs already use, hoisted to the project level. On startup:
   - Compute a SHA256 fingerprint of `cmd/vrooli/` + `internal/` Go sources
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

6. **Hot-reload in dev mode only.** `vrooli develop` can watch `cmd/vrooli-api/**/*.go` and `internal/**/*.go`, rebuild, bounce the server. Prod/install path never watches.

### Cross-cutting

7. **Cross-compile matrix from Week 5.** `make release` or a CI workflow builds `linux/amd64`, `linux/arm64`, `darwin/arm64`, `darwin/amd64`, `windows/amd64`. Trivial in Go once `CGO_ENABLED=0` is established. Catches platform drift before users do.

8. **Makefile targets** (baseline — add to `/Makefile` in Week 0):
   ```makefile
   LDFLAGS := -s -w \
       -X vrooli.com/internal/buildinfo.Fingerprint=$(shell ./hack/fingerprint.sh) \
       -X vrooli.com/internal/buildinfo.GitCommit=$(shell git rev-parse HEAD) \
       -X vrooli.com/internal/buildinfo.BuildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

   build:
       CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o .vrooli/build/vrooli ./cmd/vrooli
       CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o .vrooli/build/vrooli-api ./cmd/vrooli-api

   install: build
       install -m 0755 .vrooli/build/vrooli ~/.vrooli/bin/vrooli
       install -m 0755 .vrooli/build/vrooli-api ~/.vrooli/bin/vrooli-api
   ```

---

## 5. Week-by-week migration plan

Strangler pattern. Each week delivers value standalone. Bash stays working until the matching Go handler is green-light. Escape hatch: `VROOLI_FORCE_BASH=1` routes any subcommand back to its bash handler.

### Week 0 — Foundation (no user-visible change)

**Goal:** Get the scaffolding in place so every subsequent week is additive, not restructuring.

- [ ] Create `/go.mod` at repo root with module path `github.com/Vrooli/vrooli`
- [ ] Move `api/main.go` → `cmd/vrooli-api/main.go`; verify `vrooli develop` still launches it correctly
- [ ] Stand up `internal/buildinfo`:
  - `Fingerprint`, `GitCommit`, `BuildTime` vars populated by ldflags
  - `ComputeSourceFingerprint(rootDir string) (string, error)` — SHA256 over `.go` files under given root, sorted by path
  - `IsStale() bool` — compares embedded fingerprint to computed
  - `RebuildAndReexec(argv []string) error` — invokes `go build` + `syscall.Exec`
- [ ] Stand up `internal/logx` — thin wrapper around `log/slog`, level from `VROOLI_LOG_LEVEL` env
- [ ] Stand up `internal/cliout` — human-vs-JSON output helpers (respects `--json`, default human per feedback memory), table rendering, color detection
- [ ] Add `/Makefile` with `build`, `install`, `test`, `clean` targets using the ldflags pattern from Section 4
- [ ] Add `.vrooli/build/` to `.gitignore`

**Deliverable:** Repo structure ready. Running `make build && make install` produces `~/.vrooli/bin/vrooli-api` identical in behavior to the old `api/vrooli-api`. Nothing user-visible changed.

**Acceptance:** `vrooli develop` still works end-to-end.

---

### Week 1 — The dispatcher shim (unlocks stale-check immediately)

**Goal:** Every `vrooli` invocation goes through Go, even though Go still shells to Bash for 100% of subcommands. The stale-check concern gets solved before a single line of real bash is ported.

- [ ] Build `cmd/vrooli/` as a pure pass-through dispatcher:
  - Parses top-level args (global flags like `--verbose`, `--json`, `--no-color`)
  - For each subcommand, routes to `exec.Command("bash", "cli/commands/<appropriate-handler>.sh", args...)`
  - Inherits stdin/stdout/stderr, propagates exit code
- [ ] Wire in the `internal/buildinfo` stale-check on startup: if source fingerprint ≠ embedded, call `RebuildAndReexec`
- [ ] Support `VROOLI_FORCE_BASH=1` — when set, skip Go-level logic and exec the old `cli/vrooli` directly. This is the escape hatch.
- [ ] Update the setup script to install `~/.vrooli/bin/vrooli` (the Go binary) as the entry point, instead of symlinking the old Bash `cli/vrooli`
- [ ] Keep the old Bash `cli/vrooli` in place for fallback
- [ ] Integration test: run `vrooli scenario list` through the new Go dispatcher; verify identical output to the old Bash path

**Deliverable:** As of this week, "I edited code and forgot to rebuild" is impossible. Every `vrooli` invocation verifies freshness. The #1 concern from this conversation is solved.

**Acceptance:** Modify a `.go` file in `internal/`, run `vrooli scenario list` → binary rebuilds automatically, runs fresh code.

---

### Week 2 — Read-only scenario commands (prove the pattern)

**Goal:** Migrate the easiest, safest subcommands end-to-end to validate the shared packages and the per-subcommand cut-over pattern.

- [ ] `internal/scenario` package:
  - Typed `service.json` v2.0 reader (struct-based, not `map[string]any`)
  - Directory-scan discovery of `scenarios/*/.vrooli/service.json`
  - **Sandbox-aware path resolution** — preserve the `VROOLI_SANDBOX_SCOPE` logic from `scripts/lib/scenario/runner.sh` lines 47–98. This is the subtlest bit of Week 2; get it right up front.
- [ ] `internal/process` package:
  - Read PID files under `~/.vrooli/processes/scenarios/<name>/`
  - Check liveness via `os.FindProcess` + signal 0
  - Enumerate running scenarios
- [ ] Migrate subcommands (one PR each):
  - `vrooli scenario list`
  - `vrooli scenario info <name>`
  - `vrooli scenario status [name]`
- [ ] Delete corresponding Bash handlers in the same PR — no dual maintenance
- [ ] Unit tests for `internal/scenario` and `internal/process`

**Deliverable:** ~3 subcommands in Go. Pattern is proven. First shared packages exist.

**Acceptance:** `vrooli scenario list` output is byte-identical (or intentionally improved) compared to the old Bash version. Running with `VROOLI_FORCE_BASH=1` still works via the fallback path.

**Note on the first week of migration:** Week 2 often takes longer than it looks — not because the code is hard, but because you're designing the shared `internal/` APIs for the first time. Accept that. Do not cargo-cult the first design into later packages; expect to refactor the API shape in Week 3.

---

### Week 3 — Lifecycle executor + stop/start/restart (the hardest week)

**Goal:** The 3k-LOC `vrooli scenario restart` call path moves to Go. This is the biggest single bash chunk and the most load-bearing.

- [ ] `internal/lifecycle` package:
  - Phase runner: `setup`, `develop`, `build`, `test`, `stop`
  - Condition evaluator: `file_exists`, `directory_exists`, `resource_enabled`, etc.
  - Step executor: runs the `run` bash field of each step (yes, we still exec bash for user-defined scenario steps — scenarios are out of scope)
  - Health-check polling with timeout + backoff
  - Port from `scripts/lib/utils/lifecycle.sh` (1,049 LOC)
- [ ] `internal/ports` package:
  - Registry file read/write under `~/.vrooli/ports/` with `flock` for concurrency
  - `net.Listen` probing to detect occupied ports (cross-platform replacement for `lsof`)
  - Port allocation with sticky assignments per scenario
  - Port from `scripts/lib/network/ports.sh` (898 LOC)
- [ ] Migrate in this order (each its own PR):
  1. `vrooli scenario stop` — simpler: signal + port release
  2. `vrooli scenario start` — complex: discovery + phase execution + health checks
  3. `vrooli scenario restart` — composition of stop + start
- [ ] Integration tests: run `test-genie` scenario through old Bash and new Go paths; compare process state + port state + log contents
- [ ] Delete `scripts/lib/scenario/runner.sh`, `scripts/lib/utils/lifecycle.sh`, `scripts/lib/network/ports.sh`, and the corresponding `cli/commands/scenario/modules/*.sh` files

**Deliverable:** The call path from the original conversation is entirely Go. The biggest single bash chunk is gone.

**Acceptance:** `vrooli scenario restart test-genie` works end-to-end with `VROOLI_FORCE_BASH` unset. Process state in `~/.vrooli/processes/` is identical between old and new paths.

**Risk note:** Budget 1.5× time for this week. The bash has accumulated edge cases over years (sandbox resolution, health-check timing, port lease renewal, stale PID detection). If Week 3 slips, Weeks 4–6 compress because they reuse the same packages — the real schedule risk is front-loaded here.

---

### Week 4 — Remaining `vrooli scenario *` commands + utilities

**Goal:** Finish the scenario command family, clear out `cli/commands/scenario/` entirely.

- [ ] Migrate remaining scenario subcommands:
  - `vrooli scenario test <name>`
  - `vrooli scenario logs <name>`
  - `vrooli scenario clean <name>`
  - `vrooli scenario build <name>`
  - Anything else under `cli/commands/scenario/modules/`
- [ ] `internal/config` package:
  - `.vrooli/` path resolution (repo-local, home-level)
  - Env var handling with defaults
- [ ] Delete `cli/commands/scenario/` entirely

**Deliverable:** `cli/commands/scenario/` no longer exists. All scenario commands run through Go.

**Acceptance:** `vrooli scenario --help` lists all commands, all backed by Go handlers.

---

### Week 5 — Setup + runtime (platform-gated)

**Goal:** Port the setup story. This is where cross-platform support starts to actually matter.

- [ ] `internal/runtime` package:
  - Go, Node, Python, Docker, Helm detection + install paths
  - Platform-gated implementations:
    - `runtime_linux.go` — apt, sysctl, systemd
    - `runtime_darwin.go` — brew, stub sysctl
    - `runtime_windows.go` — mostly stubs with clear "unsupported" errors
- [ ] `internal/setup` package:
  - Port `scripts/manage.sh setup` logic
  - Delegates platform-specific bits to `internal/runtime`
- [ ] Migrate subcommands:
  - `vrooli setup`
  - `vrooli develop` (the outer wrapper; the API server itself is already Go)
- [ ] Delete `scripts/manage.sh` and the Linux-specific bits of `scripts/lib/system/`
- [ ] **Try the cross-compile.** Run `GOOS=darwin go build ./cmd/vrooli` and `GOOS=windows go build ./cmd/vrooli`. The compiler will honestly report everything still Linux-coupled — that becomes the punch list for Week 6 and beyond.

**Deliverable:** `scripts/manage.sh` deleted. First concrete cross-platform capability: you have a real punch list of what's blocking macOS/Windows.

**Acceptance:** `GOOS=linux go build ./cmd/vrooli` succeeds. `GOOS=darwin go build ./cmd/vrooli` either succeeds or fails with a list of platform-specific issues you understand.

---

### Week 6 — Long tail + cleanup

**Goal:** Mop up. Delete the remaining bash.

- [ ] `internal/secrets` package:
  - Encrypted read/write of `~/.vrooli/` secret files
  - AES-GCM or age (pick one; age is simpler and more auditable)
  - Port from `scripts/lib/service/` (~1,055 LOC)
- [ ] `internal/network` package:
  - Diagnostics, firewall wrappers
  - Linux-only via build tags; stubs return clear errors on other platforms
  - Port from `scripts/lib/network/` remainder (~1,575 LOC)
- [ ] Remaining top-level `vrooli` subcommands:
  - `vrooli status`
  - `vrooli doctor`
  - `vrooli info`
  - `vrooli clean`
  - `vrooli stop`
  - `vrooli backup`
  - `vrooli restore`
  - `vrooli orphans`
  - `vrooli locks`
- [ ] Delete `scripts/lib/` entirely
- [ ] Delete `cli/` entirely
- [ ] Delete `cli/vrooli` (the old Bash entry point — by now it's orphaned)
- [ ] Remove the `VROOLI_FORCE_BASH` escape hatch from `cmd/vrooli/main.go`

**Deliverable:** No project-level Bash remains. `wc -l cli/**/*.sh scripts/**/*.sh 2>/dev/null` returns zero (or only scenario-internal bash, which is out of scope).

**Acceptance:** A fresh checkout can run `make install && vrooli setup && vrooli develop` on Linux without touching any `.sh` file in `cli/` or `scripts/lib/`.

---

## 6. Cross-cutting principles

Apply these throughout all six weeks:

1. **One PR per subcommand migration.** Each PR:
   - Adds the Go handler
   - Wires it into the dispatcher
   - Deletes the Bash handler
   - Adds tests
   - Is independently reviewable and revertable

2. **Delete the Bash in the same PR as the Go replacement.** The `VROOLI_FORCE_BASH` env var is the only safety net. Do NOT keep parallel code paths "just in case" in the tree — that creates dual-maintenance burden and drift.

3. **Integration-test each migrated command against a real scenario.** Use `test-genie` (small, fast) as the acceptance bar. Run the command through old Bash and new Go paths; compare outputs and state.

4. **Fingerprint check stays green.** After every merge, a fresh `vrooli <subcmd>` should trigger exactly one rebuild+re-exec and then run clean. If it loops, the fingerprint algorithm has a bug.

5. **No cross-platform work until Week 5.** Building for Linux only until then keeps scope manageable. Windows/macOS becomes a punch list the compiler generates for you, not speculative design upfront.

6. **Don't try to get to zero bash by a hard deadline.** If Week 6 leaves 2k LOC of obscure bash (exotic network diagnostics, rarely-used backup routines), that's fine. Port it opportunistically afterward. 95% of the value is in the scenario commands + setup + lifecycle, which land by Week 5.

7. **Greenfield over compat shims.** Per existing project feedback guidelines: don't add backwards-compatibility layers. When you port, port cleanly. The `VROOLI_FORCE_BASH` escape hatch is the ONLY compat concession, and it goes away in Week 6.

8. **Do not touch scenarios.** Scenarios are out of scope. If you find yourself editing anything under `scenarios/*/`, stop — that's not this plan.

---

## 7. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Week 3 lifecycle port has subtle edge cases (sandbox path resolution, health-check timing, port lease renewal) | Schedule slip, incorrect behavior | Budget 1.5× time. Integration-test against `test-genie` and at least one scenario with resource dependencies. Ship with `VROOLI_FORCE_BASH` still available for one extra week after Week 3. |
| First Go package API (`internal/scenario` in Week 2) gets designed poorly | Downstream packages cargo-cult the bad shape | Expect to refactor in Week 3. Do not freeze the API after Week 2. |
| Cross-platform ambitions creep into early weeks | Schedule slip, scope explosion | Platform work is gated to Week 5. Earlier weeks target Linux only. |
| Stale-check fingerprint algorithm has a bug causing infinite rebuild loops | User experience broken, hard to diagnose | Unit test the fingerprint algorithm in Week 0. Include a `--no-stale-check` flag for debugging. |
| Bash handlers deleted prematurely, Go handler has a regression | User workflow broken | Keep `VROOLI_FORCE_BASH` functional for one full week after each handler migrates. Integration tests gate the merge. |
| "A few weeks" is optimistic | Plan stretches to 10+ weeks | Acceptable — the incremental delivery means value lands each week regardless. Don't compress by skipping tests. |

---

## 8. Open questions

These don't block starting the work, but should be resolved as they come up:

- ~~**Module path.**~~ **DECIDED:** Use `github.com/Vrooli/vrooli` (lowercase, matching the convention of the existing scenario shared packages like `github.com/vrooli/cli-core`). Rationale: it's the standard Go convention (module paths should be fetchable VCS URLs), avoids the operational burden of serving vanity-import meta tags, and stays consistent with the existing `packages/cli-core` and `packages/api-core` module paths. The existing `api/go.mod` using `vrooli.com/api` is the outlier and gets fixed during the Week 0 move from `api/` → `cmd/vrooli-api/`.
- **`packages/api-core` reuse for `cmd/vrooli-api`.** Does the project-level API server benefit from importing the scenario-facing `packages/api-core`, or should it use its own HTTP lifecycle from `internal/`? Recommendation: use `packages/api-core/server` since it already handles graceful shutdown and signals correctly, and it's a one-way dependency (project uses scenario lib, not vice versa).
- **Secrets encryption scheme.** age vs AES-GCM vs reuse whatever Bash uses today. Decide in Week 6 when you start on `internal/secrets`.
- **Windows story.** Is Windows a first-class target, or a "best effort" target where certain commands are unavailable? This affects how much effort goes into `internal/runtime/runtime_windows.go`. Recommendation: second-class for now — `vrooli develop` against containerized resources should work, but `vrooli setup` can be "Linux only, use WSL on Windows for setup" initially.

---

## 9. Progress tracker

Update this section as work lands. Check boxes for completed items, note the PR number and date.

### Week 0 — Foundation
- [ ] `/go.mod` created
- [ ] `api/main.go` → `cmd/vrooli-api/main.go` moved
- [ ] `internal/buildinfo` package shipped
- [ ] `internal/logx` package shipped
- [ ] `internal/cliout` package shipped
- [ ] `/Makefile` with standardized build targets
- [ ] `.vrooli/build/` in `.gitignore`

### Week 1 — Dispatcher shim
- [ ] `cmd/vrooli/` pass-through dispatcher
- [ ] Stale-check + rebuild+re-exec wired in
- [ ] `VROOLI_FORCE_BASH` escape hatch
- [ ] Setup script updated to install Go binary
- [ ] Integration test: `vrooli scenario list` via Go dispatcher

### Week 2 — Read-only scenario commands
- [ ] `internal/scenario` package (with sandbox awareness)
- [ ] `internal/process` package
- [ ] `vrooli scenario list` migrated
- [ ] `vrooli scenario info` migrated
- [ ] `vrooli scenario status` migrated
- [ ] Unit tests for `internal/scenario`, `internal/process`

### Week 3 — Lifecycle executor + stop/start/restart
- [ ] `internal/lifecycle` package
- [ ] `internal/ports` package
- [ ] `vrooli scenario stop` migrated
- [ ] `vrooli scenario start` migrated
- [ ] `vrooli scenario restart` migrated
- [ ] Integration tests against `test-genie`
- [ ] `scripts/lib/scenario/runner.sh`, `scripts/lib/utils/lifecycle.sh`, `scripts/lib/network/ports.sh` deleted

### Week 4 — Remaining scenario commands + config
- [ ] `vrooli scenario test` migrated
- [ ] `vrooli scenario logs` migrated
- [ ] `vrooli scenario clean` migrated
- [ ] `vrooli scenario build` migrated
- [ ] `internal/config` package
- [ ] `cli/commands/scenario/` deleted

### Week 5 — Setup + runtime
- [ ] `internal/runtime` package (platform-gated)
- [ ] `internal/setup` package
- [ ] `vrooli setup` migrated
- [ ] `vrooli develop` (outer wrapper) migrated
- [ ] `scripts/manage.sh` deleted
- [ ] Cross-compile attempt for darwin/windows; punch list captured here:
  - (fill in as discovered)

### Week 6 — Long tail + cleanup
- [ ] `internal/secrets` package
- [ ] `internal/network` package
- [ ] `vrooli status`, `doctor`, `info`, `clean`, `stop`, `backup`, `restore`, `orphans`, `locks` migrated
- [ ] `scripts/lib/` deleted
- [ ] `cli/` deleted
- [ ] `VROOLI_FORCE_BASH` escape hatch removed

### Post-migration
- [ ] Zero `.sh` files remaining under `cli/` and `scripts/lib/` (confirmed via `find`)
- [ ] Cross-platform CI matrix green for linux/amd64, linux/arm64, darwin/arm64
- [ ] Documentation updated (CLAUDE.md, README, deployment docs)
