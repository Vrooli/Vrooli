# Workspace Sandbox: Reliability Follow-Ups (Post Round-4)

## 1. Purpose

Close the residual reliability gaps in `scenarios/workspace-sandbox` that
remain after Rounds 1–4 of refactoring (driver collapse, reconciler
abstraction, home-overlay decision unification, service carving,
handler thinning, durability tracker, property tests, testutil package,
clock seam, live-HTTP harness, driver-contract expansion, SSE writer
seam, audit helper, mounter/process-starter seams, control-surface
consolidation, schema/profile registry hardening).

The user is experiencing a "whack-a-mole" feel from sandbox flakiness
even though most structural seams are now in place. This plan attacks
the **remaining true gaps** — not new architecture — focusing on:

1. Profile-side home-overlay requirement is binary; can't model
   "optional" → silent or wrong refusals.
2. `ChangeTracker` interface is unified, but the **walk implementations**
   for overlay vs copy are still two ~100-line near-duplicates with
   independent bug surface.
3. Mount/daemon lifecycle is reaped *eventually* by `daemon_reaper.go`;
   teardown does not have a deterministic per-sandbox kill+wait that's
   asserted in tests.
4. Whack-a-mole symptoms persist because boundary **invariants are not
   captured in writing or tests**: each round documents *what was
   fixed*, not *what must always hold*.

Out of scope: anything already shipped in Rounds 1–4 (see §5 for the
"already in place" inventory) and any redesign that breaks the
green-field cutover constraints from prior plans.

## 2. Required Reading

Run before executing this plan:

```bash
prompt-manager skill read implementation-plan-authoring boundary-of-responsibility-enforcement seam-discovery-and-enforcement test unit-testing-architecture-steer assumption-mapping-and-hardening
prompt-manager skill read cli-steer api-steer utils-unification
```

Also read in repo:

- `scenarios/workspace-sandbox/docs/SEAMS.md` — current seam inventory.
- `scenarios/workspace-sandbox/docs/PROBLEMS.md` — known-issues log.
- `scenarios/workspace-sandbox/api/internal/policy/home_overlay.go` —
  current home-overlay decision (binary input).
- `scenarios/workspace-sandbox/api/internal/driver/driver.go` — the
  `Driver`, `MountDriver`, `ChangeTracker`, `MountVerifier` interfaces.
- `scenarios/workspace-sandbox/api/internal/driver/helpers.go:244` —
  `getOverlayChangedFiles` (overlay walk implementation).
- `scenarios/workspace-sandbox/api/internal/driver/copy.go:168` —
  `getCopyChangedFiles` (copy walk implementation).
- `scenarios/workspace-sandbox/api/internal/sandbox/daemon_reaper.go` —
  fuse-overlayfs orphan reaper.
- `scenarios/workspace-sandbox/api/internal/types/types.go:216` —
  `HomeOverlayState` runtime enum.
- `scenarios/workspace-sandbox/api/internal/config/profiles.go:47` —
  profile `RequiresHomeOverlay bool`.

## 3. Problem Statement

Despite Rounds 1–4 introducing strong seams (sse package, home-overlay
policy, daemon reaper, ChangeTracker, Repository, Mounter, ProcessStarter,
clock seam, live-HTTP harness), bug fixes still feel reactive. The
remaining root causes:

**P1 — Profile contract is too coarse.**
`config.IsolationProfile.RequiresHomeOverlay` is `bool`. There is no
way to declare "this profile uses $HOME if available, but works without
it" — every consumer is forced into "required" or "off". Profiles that
*can* fall back are mis-modeled; refusal is binary.

**P2 — `ChangeTracker` interface is a thin shell.**
`overlay` and `copy` drivers each ship a private ~100-line
`getXxxChangedFiles` walk. Whiteout handling, hidden-file rules,
size/mtime semantics, error wrapping — three independent code paths
(overlay-userns + overlay-root share helpers; fuse-overlayfs shares the
same helpers; copy is its own world). When a diff bug surfaces, it is
fixed in one path. The interface gives no shared building blocks.

**P3 — Daemon teardown is reactive, not deterministic.**
`daemon_reaper.go` (added 2026-04-29) reaps fuse-overlayfs daemons
whose sandbox UUID is no longer registered, on a periodic loop. The
delete-path itself does not own a per-sandbox kill+wait sequence with
test coverage. Result: between Delete and the next reaper tick,
remount at the same sandbox dir can race a still-live daemon.

**P4 — Invariants are not captured in writing or in contract tests.**
Each round of refactoring fixed a class of bug. None left behind a
`docs/internal/INVARIANTS.md` file or a corresponding contract-test
matrix asserting the invariant. So the *next* class of bug arrives by
the same path: an implicit assumption (timeout shape, frame ordering,
mount-then-cleanup ordering) gets violated by a benign-looking change.
This is the meta-cause of the whack-a-mole feeling.

## 4. Scope

**In scope (this plan):**

- Replace `RequiresHomeOverlay bool` with a tri-state
  `HomeOverlayRequirement` enum. Update profile registry, policy,
  contract tests, and agent-manager adapter consumer.
- Extract a `driver/changedetect/` package containing one **walker**
  used by both overlay and copy `ChangeTracker` implementations. Keep
  driver-specific differences (whiteout semantics, lower-vs-upper
  comparison) behind small strategy interfaces.
- Make sandbox Delete own a deterministic teardown sequence:
  unmount → kill registered daemon (if any) → wait → remove dir →
  emit audit. Add an integration test that verifies "Delete returns
  → Create same-uuid succeeds without reaper involvement".
- Establish `scenarios/workspace-sandbox/docs/internal/INVARIANTS.md`
  and an `assumptions_test.go`-style contract suite that names each
  invariant and asserts it at the smallest meaningful seam.

**Out of scope:**

- Any new sandbox driver (e.g., zfs, btrfs).
- agent-manager-side runner consolidation (tracked separately as
  Phase E of agent-manager work).
- UI changes to surface new home-overlay requirement values.
- PostgreSQL anything (sqlite cutover already shipped).
- Brownfield/migration shims (see Greenfield Constraint, §6).
- Cross-platform support beyond what's already declared per driver.

## 5. Current Technical Context

What is **already in place** and must not be rebuilt:

| Concern | Where | Status |
|---|---|---|
| SSE wire-format contract + Flusher seam | `internal/sse/sse.go` | ✅ shipped (Round 4 Phase 5) |
| Home-overlay decision (single source of truth) | `internal/policy/home_overlay.go` | ✅ shipped (input still binary — that's P1) |
| Mount/Unmount seam | `internal/fsmount/mount.go` | ✅ shipped (Round 4 Phase 7) |
| Process starter seam | `internal/process/starter.go` | ✅ shipped (Round 4 Phase 7) |
| Clock seam (deterministic time in tests) | `internal/clock/` (used by tests) | ✅ shipped (Round 4 Phase 2) |
| Driver `ChangeTracker` interface | `internal/driver/driver.go` | ✅ shipped — but two separate impls (P2) |
| Repository interface | `internal/repository/sandbox_repo.go` | ✅ shipped |
| FUSE daemon reaper | `internal/sandbox/daemon_reaper.go` | ✅ shipped 2026-04-29 — but only as background safety net (P3) |
| Driver contract tests | `internal/driver/contract_test.go`, `contract_failure_test.go` | ✅ shipped (Round 4 Phase 4) |
| Live-HTTP test harness | `internal/handlers/process_sse_test.go` etc. | ✅ shipped (Round 4 Phase 3) |
| Schema version + profile registry hardening | (Round 4 Phase 9) | ✅ shipped |

What is **still gap-fillable** (the four phases below):

- Profile-side requirement remains a `bool` (P1).
- Walk logic in `helpers.go:244` and `copy.go:168` is duplicated (P2).
- Sandbox `Delete` does not have a deterministic daemon teardown
  sequence with a test pinning the contract (P3).
- No `INVARIANTS.md` or invariant contract suite (P4).

## 6. Greenfield Constraint (Hard Rule)

Per repo convention and existing user feedback (`feedback_planning_guidelines`):

- **No backwards-compatibility shims.** Replacing `RequiresHomeOverlay
  bool` with the enum means deleting the bool field, not aliasing it.
  Every reader updates in one PR.
- **No deprecation comments.** Removed code is removed.
- **No "if old format" branching.** Profile JSON files are rewritten in
  the same change.
- The plan is rejected if any of the following remain in the merged tree:
  - `RequiresHomeOverlay` field on `IsolationProfile`.
  - Two separate `getXxxChangedFiles` walk loops in `driver/`.
  - A code path that calls `repo.DeleteSandbox` for a fuse-backed
    sandbox without first invoking the daemon-kill seam.
  - An `INVARIANTS.md` containing `TODO`, `TBD`, or "see code" without
    the corresponding test reference.

## 7. Target End State

When the plan is done:

1. A profile declares `HomeOverlayRequirement` ∈ {`required`, `optional`,
   `not_needed`}. `policy.DecideHomeOverlay` returns `Allowed:true,
   Reason:"profile permits absent overlay"` for the `optional` +
   `Absent` case (and a structured warning code, not an error).
2. `internal/driver/changedetect/` exists with one walker. Overlay and
   copy `ChangeTracker.GetChangedFiles` implementations each consist of
   <40 lines of glue calling into the shared walker via a strategy
   interface. The two ~100-line walks at `helpers.go:244` and
   `copy.go:168` are gone.
3. `Delete` (or its sandbox-service caller) deterministically:
   unmounts → kills its own daemon if registered → waits with bounded
   timeout → removes dir → marks repo deleted. Daemon-reaper continues
   to exist as a *safety net* covering API-crash paths only. A test
   exercises Delete-then-immediate-Create-with-same-uuid without the
   reaper running.
4. `docs/internal/INVARIANTS.md` lists every behavioural invariant the
   system relies on, with one test reference per row. A new
   `internal/sandbox/invariants_test.go` (or per-package equivalent)
   asserts each.

## 8. Implementation Strategy (Phased)

Each phase is independently mergeable. Phases 1–3 fix the recurring
bugs; Phase 4 captures the invariants so this stops happening.

### Phase 1 — Tri-state `HomeOverlayRequirement`

Goal: replace `bool RequiresHomeOverlay` with a 3-valued enum so
profiles can express "uses $HOME if present, falls back if not".

Steps:

1. In `internal/types/types.go`, add:
   ```go
   type HomeOverlayRequirement string
   const (
       HomeOverlayNotNeeded HomeOverlayRequirement = "not_needed"
       HomeOverlayOptional  HomeOverlayRequirement = "optional"
       HomeOverlayRequired  HomeOverlayRequirement = "required"
   )
   ```
2. In `internal/config/profiles.go`:
   - Replace field `RequiresHomeOverlay bool` with
     `HomeOverlayRequirement HomeOverlayRequirement`.
   - Update each built-in profile literal. Default = `HomeOverlayNotNeeded`.
   - Validate at registry-load time that the value is one of the three.
3. In `internal/policy/home_overlay.go`:
   - Take `profile.HomeOverlayRequirement` instead of `bool`.
   - For `HomeOverlayOptional` + `Absent`/`Unsupported`: return
     `Allowed:true` with a non-empty `Code = "HOME_OVERLAY_FALLBACK"`
     so callers can log/record the soft-degradation.
   - For `HomeOverlayRequired` + `Absent` or `Unsupported`: keep
     existing refusal codes.
   - For `HomeOverlayNotNeeded`: unconditionally allowed (no code).
4. Update `internal/policy/home_overlay_test.go` to a table-driven
   test with 9 cells (3 requirements × 3 effective states).
5. Update agent-manager adapter
   (`scenarios/agent-manager/api/internal/adapters/sandbox/`) to read
   `Code` and translate `HOME_OVERLAY_FALLBACK` to a structured run-event
   "soft fallback" rather than a refusal.
6. Update profile JSON in `initialization/` (if any) to the new field
   name. Greenfield: no aliasing.
7. Build + tests + scenario restart.

Done when:
- `rg "RequiresHomeOverlay" scenarios/workspace-sandbox` returns 0
  matches.
- `rg "RequiresHomeOverlay" scenarios/agent-manager` returns 0 matches.
- New table-driven test enumerates all 9 cells.

### Phase 2 — Unify change-detection walk

Goal: collapse the duplicate ~100-line walks behind one helper.

Steps:

1. Create `internal/driver/changedetect/` with:
   - `walker.go`: a `Walk(ctx, root, strategy ChangeStrategy)
     ([]*types.FileChange, error)` function that owns directory
     traversal, hidden-file filtering, error wrapping, and ctx
     cancellation.
   - `strategy.go`: the `ChangeStrategy` interface — methods like
     `IsWhiteout(name string, info fs.FileInfo) bool`, `Compare(rel
     string, upper, lower fs.FileInfo) (ChangeType, bool)`,
     `LowerLookup(rel string) (fs.FileInfo, error)`.
   - `overlay_strategy.go`: implements strategy with overlayfs
     whiteout semantics. Replaces the body of `helpers.go:244
     getOverlayChangedFiles`.
   - `copy_strategy.go`: implements strategy with double-walk semantics.
     Replaces the body of `copy.go:168 getCopyChangedFiles`.
2. Refactor `helpers.go` and `copy.go`:
   - Each driver's `GetChangedFiles` becomes a one-line call:
     `return changedetect.Walk(ctx, sandboxUpper, &overlayStrategy{...})`.
3. Move existing change-detection assertions out of `helpers_test.go`
   and `copy_test.go` into one **shared contract test**
   `internal/driver/changedetect/walker_contract_test.go` that runs the
   walker against a fixture filesystem with both strategies.
4. Add edge cases that previously were tested in only one path:
   - File replaced by directory of same name.
   - Symlink whose target moved.
   - Permission-denied subtree.
   - Empty upper (no changes).
   - Unicode/emoji filenames.
5. Build + tests + scenario restart.

Done when:
- The two `getXxxChangedFiles` private functions no longer exist in
  `helpers.go` and `copy.go`.
- Coverage of `changedetect/` is ≥85% by `go test -cover`.
- `walker_contract_test.go` exercises ≥10 fixture cases per strategy.

### Phase 3 — Deterministic daemon teardown on Delete

Goal: the delete path must own daemon termination instead of relying
on the reaper's eventual sweep.

Steps:

1. In the fuse-overlayfs driver, ensure each `Mount()` registers the
   spawned daemon PID (if not already) into a per-driver
   `daemonRegistry` keyed by sandbox UUID.
2. Add `MountDriver.UnmountAndKillDaemon(ctx, sandboxID)` to the driver
   surface (or as a method on `MountVerifier`/a new
   `DaemonOwner` interface — pick the smallest seam).
3. In `internal/sandbox/service_lifecycle.go` `Delete` path:
   1. Call existing unmount.
   2. Call `UnmountAndKillDaemon`. Log + audit if the daemon was still
      running.
   3. `wait` (bounded — clock-seam-aware, default 5s) for daemon exit.
   4. Remove sandbox dir.
   5. Mark repo deleted.
4. Move `daemon_reaper.go` from "primary cleaner" to "safety net":
   add a metric counter `daemon_reaped_total{cause="api_crash"}`
   that increments only when the reaper finds a daemon whose sandbox
   was already marked deleted in the repo. Non-zero = API crashed
   between unmount and dir-remove. Alert on it.
5. Add an integration test
   `internal/sandbox/delete_daemon_lifecycle_test.go` that:
   - Mounts a fuse-backed sandbox.
   - Calls `Delete`.
   - Asserts daemon PID is gone within 5s **without** running the reaper.
   - Re-creates a sandbox at the same UUID; verifies mount succeeds.
6. Build + tests + scenario restart.

Done when:
- `Delete` for fuse-overlayfs sandboxes is deterministic in test (no
  flaky time-based waits beyond the bounded one).
- Reaper metric is wired and surfaced in `/metrics`.
- New integration test passes 100/100 in `go test -run
  Delete_Daemon_Lifecycle -count=100`.

### Phase 4 — Invariant capture & contract suite

Goal: stop the whack-a-mole. Each round has been a one-time fix; this
phase encodes invariants so future regressions fail at write-time.

Steps:

1. Create `scenarios/workspace-sandbox/docs/internal/INVARIANTS.md`.
   For each row, record: invariant name, where enforced (file:line or
   package), where tested (test file:test name).
2. Seed it with at minimum:
   - **I-SSE-1**: every SSE stream emits `event: end` exactly once,
     after `event: exit` if any. Enforced by `internal/sse`. Tested by
     `internal/handlers/process_sse_test.go`.
   - **I-SSE-2**: SSE writers refuse construction without an
     `http.Flusher`. Enforced/tested in `internal/sse`.
   - **I-HOME-1**: `policy.DecideHomeOverlay` is pure (no I/O). Tested.
   - **I-HOME-2**: `HomeOverlayOptional` + `Absent` ⇒ allowed +
     fallback code (Phase 1).
   - **I-MOUNT-1**: `Delete` returns ⇒ no fuse-overlayfs daemon
     remains for that sandbox UUID (Phase 3).
   - **I-MOUNT-2**: Mount is idempotent: calling `Mount` on an already-mounted
     sandbox is a no-op or returns the same paths.
   - **I-CHANGE-1**: `ChangeTracker.GetChangedFiles` is deterministic
     for a given filesystem state (no random ordering, stable sort).
   - **I-DRIVER-1**: A sandbox's `DriverID` is immutable after Create
     (driver swap only affects new sandboxes).
   - **I-AUDIT-1**: every state transition emits exactly one
     audit-log entry.
3. For each invariant, ensure a test asserts it. If a test is missing,
   add it in the package owning the invariant. Where multiple
   invariants live in one package, use a single
   `invariants_test.go` listing them as subtests with names matching
   the invariant ID (e.g., `t.Run("I-MOUNT-1", ...)`).
4. Add a CI grep gate (or `assumptions_test.go`-style scan) that fails
   if any invariant ID in `INVARIANTS.md` lacks a matching test name.
5. Build + tests + scenario restart.

Done when:
- `INVARIANTS.md` exists with ≥9 invariants, each tied to a file:line
  enforcement and a test name.
- A test scan asserts every invariant ID from the doc is present in a
  `t.Run` somewhere in the codebase.
- Running `make test` (or `vrooli scenario test workspace-sandbox`)
  reports the invariant suite passing.

## 9. Contract Decisions

These define the wire/API behaviours that downstream consumers
(agent-manager, UI, CLI) depend on. Phase numbers in parentheses.

### C1 — Profile field rename (Phase 1)

- `IsolationProfile.RequiresHomeOverlay bool` → `HomeOverlayRequirement
  HomeOverlayRequirement` (string-typed enum).
- JSON schema: `"requiresHomeOverlay": true` → `"homeOverlayRequirement":
  "required"`. No alias, no migration code.
- Default for un-set field at decode time: `not_needed`.
- Validator rejects unknown values at profile-load time.

### C2 — `policy.DecideHomeOverlay` extended return shape (Phase 1)

- Add code value `HOME_OVERLAY_FALLBACK` (success-with-warning).
- `Decision.Allowed = true` when fallback applies.
- `Decision.Code = "HOME_OVERLAY_FALLBACK"`, `Reason` populated.
- Existing refusal codes (`HOME_OVERLAY_REQUIRED`,
  `HOME_OVERLAY_UNSUPPORTED_DRIVER`) retained unchanged.

### C3 — `ChangeTracker.GetChangedFiles` ordering (Phase 2)

- Output is sorted by `RelativePath` ascending (stable).
- Implementation must use `sort.SliceStable` after walk.
- Documented on the interface; tested by I-CHANGE-1.

### C4 — Sandbox Delete sequence (Phase 3)

- `Delete(ctx, id)` for a fuse-backed sandbox MUST:
  1. Unmount overlay.
  2. Kill registered daemon (if any), bounded wait.
  3. Remove sandbox dir.
  4. Mark repo deleted.
- Order is part of the contract. Test pins it.
- For non-fuse drivers, daemon-kill step is a no-op.

### C5 — Reaper role (Phase 3)

- Reaper is a *safety net* covering only API-crash paths.
- Metric `daemon_reaped_total{cause="api_crash"}` MUST be exposed via
  `/metrics`.
- Non-zero counter is an operational signal, not a normal occurrence.

### C6 — Invariant test naming (Phase 4)

- Every invariant ID in `INVARIANTS.md` MUST appear as a `t.Run`
  subtest name somewhere in the test tree.
- Scan tool: simple `rg`-based test fails CI if an ID is missing.

## 10. Testing Plan

Automated. Per `feedback_testing_over_manual`, no manual steps.

| Phase | New / changed tests | Command |
|---|---|---|
| 1 | `internal/policy/home_overlay_test.go` table-driven 3×3; `internal/config/profiles_test.go` rejects unknown enum values; agent-manager adapter test for `HOME_OVERLAY_FALLBACK` translation | `go test ./...` in workspace-sandbox; `go test ./internal/adapters/sandbox/...` in agent-manager |
| 2 | `internal/driver/changedetect/walker_contract_test.go` (≥10 fixtures × 2 strategies); coverage ≥85% in `changedetect/` | `go test -cover ./internal/driver/changedetect/` |
| 3 | `internal/sandbox/delete_daemon_lifecycle_test.go` × 100 iterations | `go test -run Delete_Daemon_Lifecycle -count=100 ./internal/sandbox/...` |
| 4 | `invariants_test.go` per package; CI-level scan asserting every `INVARIANTS.md` ID has a matching `t.Run` | `make test` plus a small `scripts/check-invariants.sh` (bash) that diffs IDs in doc vs grep over tests |

Quality requirements from the `test` and `unit-testing-architecture-steer`
skills:

- Arrange-Act-Assert structure for every new test.
- Table-driven where the input domain is enumerable (Phases 1, 2).
- `t.Helper()` in every test helper added.
- No `time.Now()` in test code — use `internal/clock`.
- Existing live-HTTP harness used unchanged.

Verification gates:

- `go build ./...` from `scenarios/workspace-sandbox/api/`.
- `go test ./... -timeout 300s` from same.
- `go test ./... -timeout 300s` from `scenarios/agent-manager/api/`
  (Phase 1 cross-cuts).
- `gofumpt -d` clean; `golangci-lint run` clean.
- `vrooli scenario restart workspace-sandbox` followed by health check.

## 11. Rollout / Validation Checklist

Per phase:

- [ ] Code change compiles in workspace-sandbox.
- [ ] Code change compiles in agent-manager (Phase 1 only).
- [ ] All package tests pass.
- [ ] `gofumpt -w` + `golangci-lint run` clean.
- [ ] `vrooli scenario restart workspace-sandbox` succeeds.
- [ ] `vrooli scenario test workspace-sandbox` passes.
- [ ] Health endpoint reports OK.
- [ ] `docs/SEAMS.md` updated to reflect the new shape.
- [ ] `docs/PROBLEMS.md` updated — close the relevant entry, do not just delete.
- [ ] (Phase 4) `docs/internal/INVARIANTS.md` reviewed for completeness.

## 12. Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Phase 1 breaks agent-manager runs that currently rely on hard refusal | Medium | High | Cross-cut both repos in one PR; agent-manager test for FALLBACK code added in same change |
| Phase 2 strategy interface design under-fits one driver | Medium | Medium | Prototype overlay strategy first against the 100-line existing walk; only generalize once the second strategy lands cleanly |
| Phase 3 daemon-kill races on slow systems | Low | Medium | Bounded wait via clock seam; integration test runs 100× in CI to surface flakes |
| Phase 4 invariant doc rots | High over time | Medium | CI scan ensures every doc ID has a test; failing scan blocks merge |
| Whole plan creates merge conflicts with in-flight Round 4 leftovers | Low | Low | All Round 4 phases marked complete in task tracker; no overlap in files except potentially `policy/home_overlay.go` |
| Phase 2 changes diff ordering, breaking external consumers | Low | High | C3 names ordering as a contract; if any consumer relies on prior order, that's a separate fix in the consumer, not a compat shim here |

## 13. Non-Goals & Prohibited Patterns

- ❌ No `RequiresHomeOverlay` alias kept for compatibility.
- ❌ No `if profile.RequiresHomeOverlay or profile.HomeOverlayRequirement
  == "required"` dual-read.
- ❌ No new sandbox driver.
- ❌ No tweaks to `internal/sse/` (it is already correct).
- ❌ No deletion of `daemon_reaper.go` — it stays as the safety net.
- ❌ No per-phase feature flag.
- ❌ No "TODO migrate later" markers in the merged tree.
- ❌ No manual test checklists.
- ❌ No comments explaining what was removed ("was previously…", "old
  field used to be…").

## 14. Definition of Done

The plan is done when **all** of these hold simultaneously:

1. `rg "RequiresHomeOverlay" scenarios/` returns zero matches.
2. `rg "getOverlayChangedFiles\|getCopyChangedFiles"
   scenarios/workspace-sandbox/api/internal/driver/` returns zero
   matches outside `changedetect/` (and even there only as private
   strategy methods, not as a duplicate walk).
3. `internal/sandbox/delete_daemon_lifecycle_test.go` exists and passes
   100/100 in CI.
4. `docs/internal/INVARIANTS.md` exists with ≥9 entries; every entry
   has a matching `t.Run` name found by the CI scan.
5. `vrooli scenario restart workspace-sandbox` succeeds and the health
   endpoint reports OK.
6. `vrooli scenario test workspace-sandbox` passes.
7. `go test ./... -count=1` passes in both workspace-sandbox and
   agent-manager.
8. `docs/SEAMS.md`, `docs/PROBLEMS.md`, and `docs/internal/INVARIANTS.md`
   reflect the new state.
9. No new file outside the in-scope list was added; no compatibility
   shim, alias, or migration helper is present in the merged tree.
