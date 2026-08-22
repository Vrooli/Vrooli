# Problems — Infrastructure Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### P-001 — Setpoint confidence is `SKETCH` because no obligation list exists

**Status:** open · blocks honest coverage reporting · **owner:** the `infra-health` team, not this scenario

The 14 authored target kinds are at *operation* granularity. The **obligation**
list they should derive from — what the team must be able to do, from which the
targets follow — has never been written. `TARGET_MODEL.md` § 4 is direct that
conflating the two is the most common modelling error and that a denominator
built this way *"can never rise above `sketch` confidence."*

The 2026-08-19 projection model narrows this substantially without closing it.
The ten projections and their cell grids **are** the structural half of the
missing obligation list: they name what the platform must be able to demonstrate,
per layer, rather than which command happens to emit a number. What remains is
the team's judgment call that the ten projections are the right ten — and each
owner authoring the cells inside its own space doc. Until both exist, every
denominator reports `SKETCH`, which is stated on every ratio rather than rounded
away.

**Revisit trigger:** the team confirms the projection set against `I2`, and each
control layer authors its `docs/spaces/<projection>-space.md`.

### P-002 — Every current-state reading is stale; the team loop is paused

**Status:** open · **owner:** the operator

The `infra-health` loop has been `paused-manual` since 2026-07-24. Every
current-state value in the plan of record — the 1,058/24h alarm flood, the
~55-running/~25-supervised diff — reflects the pause, not the platform. Building
an observer against them would bake a stale reading into the scenario's first
evidence.

**Revisit trigger:** resume the loop and run the resume-protocol scan before the
first vertical slice reports anything as `measured`.

### P-003 — Two of four trust rules cannot be computed until roadmap Gap 10 ships

**Status:** closed 2026-08-20 · **owner:** `vrooli-autoheal`

Ghost and shelved verdicts need a two-direction `check reconcile` and a
shelve-with-expiry verb on `vrooli-autoheal`, neither of which exists. The
scenario computes saturated and unit-mismatch, and marks the rest `UNTRUSTED`
rather than assuming `VALID`, so the gap degrades conservatively — but declared
blindness is wider than it will be, and the shelving record is a hand-maintained
artifact in the meantime.

**Resolution:** the typed autoheal checks surface now exposes both-direction
reconcile, saturation, and mandatory-expiry shelves; the condition source
consumes those reads and assigns the closed trust vocabulary conservatively.

**Evidence:** `scenarios/vrooli-autoheal/api/internal/handlers/typed_connect.go`
and the autoheal checks/reconcile tests.

### P-004 — Generated from a template whose registry status is `quarantined`

**Status:** open · escalated upstream · **owner:** `template-manager`

`template-manager registry list --kind scenario --json` was re-run on 2026-08-20
and reports `react-vite` 1.6.5 as `quarantined` (and also reports the other
scenario template as quarantined). Template Manager's own `PROGRESS.md` records
react-vite 1.6.5 passing deep validation on 2026-07-12, so the registry state is
still inconsistent with the last known validation evidence. Generation succeeded
and the scaffold validates, but this run cannot independently establish that the
quarantine is stale or safe to override.

The finding is escalated rather than closed: the plan's provenance gate remains
open until Template Manager records a definitive resolution. Product work may
continue only against the already-generated scenario while this provenance
question is tracked.

**Revisit trigger:** Template Manager publishes whether the flag is stale or
genuine. If genuine, re-evaluate the template choice before claiming the
scenario's provenance is healthy.

### P-005 — Three extension rules are prose with no mechanical enforcement

**Status:** open · **owner:** this scenario

Extension rules 3 (never persist a verdict), 4 (never cache the setpoint or a
derived set) and 6 (no actuation path) are the mitigations for the three
highest-impact risks in `SECURITY.md`, and all three are currently enforced only
by review. Each is the kind of invariant a well-meaning caching or convenience
change erodes silently.

**Revisit trigger:** first vertical slice. Add architecture tests asserting that
`coverage` owns no schema, that no setpoint write path exists, and that no dependency client exposes
a mutating verb.

### P-006 — `secrets-manager` exposes no typed read surface

**Status:** open · blocks a target row · **owner:** `secrets-manager`

Secrets availability is load-bearing for scenario start and has no target row
anywhere in the team's plan of record. The scenario ships a `cli/` directory but
**no `cli/manifest.json`**, so it is unbindable and exposes no typed read this
board could consume.

**Revisit trigger:** `secrets-manager` ships a manifest and a health/status verb.
Do not compensate for the gap here with log scraping — that would breach the
typed-read boundary that keeps credentials out of the reading store.

### P-007 — `vrooli-autoheal` has no typed read surface at all

**Status:** closed 2026-08-20 · **owner:** this scenario's plan

`vrooli-autoheal` backs three of ten projections (`supervision`, `availability`,
`recovery`), owns the check registry, and answers two of four trust rules.

The typed surface now preserves the existing checks, actions, incidents,
healing, and Measures domains rather than introducing a minimal shim.

**Evidence:** `packages/proto/schemas/vrooli-autoheal/v1/`,
`scenarios/vrooli-autoheal/api/internal/handlers/typed_connect.go`, and
`scenarios/vrooli-autoheal/cli/manifest.json`.

**Refs:** `scenarios/vrooli-autoheal/api/main.go`, `api/internal/{checks,healing,incidents}/`

### P-008 — No control layer has authored a reliability space doc

**Status:** substantially resolved 2026-08-20 · **owner:** each control layer

The coverage denominator is designed to be read from each owner's
`docs/spaces/<projection>-space.md` through its `space --projection <p> --json`
verb — the same contract `search-hub`, `test-genie`, `prompt-manager` and
`program-runtime` already implement for `meta-optimization-manager`. The six
scenario owners and the control-plane exception now author the required spaces.

The denominator still reports `SKETCH` until the obligation list and operator
confidence are approved; that is an intentional confidence state, not a
missing transport. The scenario must not assert the cell sets itself: that
would be a roster in all but name and would go stale on owner change.

**Evidence:** owner `docs/spaces/` documents and the shared flat `space` CLI
projection contract.

### P-009 — `vrooli capacity` cannot be read typed, by construction

**Status:** open · accepted, fenced · **owner:** the control plane

The capacity projection's sensor is `vrooli capacity reconcile|recommend`, and
`vrooli capacity` lives in the repo-root `internal/capacity` package. Go's
`internal/` visibility rule means a scenario — a separate module — cannot import
it, and `api-core/discovery` resolves scenario ports, not control-plane
subcommands. So this one source is read as a bounded CLI subprocess.

This is accepted rather than fixed, but it is **fenced to one named source**. A
second CLI read is a design smell and should be challenged in review, because
the failure mode it reintroduces — a hung owner stalling the board — is the one
`meta-optimization-manager` migrated away from CLI reads to escape.

**Real fix:** the control plane exposes a typed local surface for capacity
reads. Until then, the subprocess carries the same 10s deadline and
`UNAVAILABLE` degradation as every typed source.

**Refs:** `internal/capacity/`, `packages/api-core/discovery/resolve.go`

### P-010 — Capability resolution is `OR` for a population that is partly `AND`

**Status:** open · **owner:** the control plane (`internal/deployability`)

**Symptom:** Nine capability×OS cells in `portability grid` report
`IMPLEMENTED` while a safeguard declaring that same capability is unsupported
on that OS. `credential-storage` reads green on Windows and macOS while both
credential safeguards are Linux-only; `remote-desktop` reads green on all three
while `remote_session_protection` — the hardening half — is Linux-only;
`developer-utility` reads green on Windows while `vrooli_launcher`,
`onboarding_apply_privileges` and `path_hygiene` are not.

**Root cause:** `ResolveCapability` collects every declarer that supports the
target OS, sorts by qualification rank, and returns `candidates[0]`. The
declarers that do *not* support that OS are collected into `unwired` and
`ineligible` and then discarded, because the function returns from the winner
branch before reading them. This is correct for **tools**, which are
substitutable — `winget` genuinely does stand in for `apt-get`. It is wrong for
**safeguards**, which are independently required controls: five declare
`crash-forensics` and they are five jobs, not five ways of doing one job.

**Workaround:** read `vrooli host safeguard <name>` per safeguard, or read the
`platforms` array in each `safeguard.json` directly. Do not treat a green
capability row as evidence that every control in that category is present.

**Real fix:** add `control` to the `capability_role` vocabulary, resolve
`control` declarers conjunctively, and add an absent-declarer set to
`CapabilityResolution` that is populated in **every** branch including the
winner branch. Reporting the absent set is worth doing for providers too:
*"resolves via winget; apt-get, dnf, pacman, rpm are Linux-only"* is strictly
better than a bare green lamp. Model documented in
[`../concepts/PORTABILITY-MODEL.md`](../concepts/PORTABILITY-MODEL.md)
§ Provider Roles And Control Roles.

**Refs:** `internal/deployability/capability.go`,
`.vrooli/schemas/safeguard.schema.json`, `api/internal/portability/grid.go`

### P-011 — The capability vocabulary is maintained in three files and has drifted

**Status:** open · **owner:** the control plane

**Symptom:** `.vrooli/capability-vocabulary.json` carries 41 capability names.
The `capability` enum in `.vrooli/schemas/safeguard.schema.json` carries 27, and
`.vrooli/schemas/tool.schema.json` carries another 27. The 14
scenario-contributed names (`system-monitor-*`, `autoheal-*`) exist in the
vocabulary and in neither schema.

**Root cause:** three hand-maintained copies with no drift gate.
`.vrooli/repo-contract.json` allowlists the vocabulary file's existence but
never compares its contents to the schemas.

**Workaround:** treat `.vrooli/capability-vocabulary.json` as authoritative — it
is what `portability` and `vrooli capability ledger` actually read. The schema
enums only constrain what a tool or safeguard may author.

**Real fix:** generate both schema enums from the vocabulary and add a
repo-contract drift check. If tools and safeguards are deliberately barred from
claiming scenario-contributed capabilities, that rule should be *expressed*
rather than left as an accident of two lists nobody synchronises.

**Refs:** `.vrooli/capability-vocabulary.json`, `.vrooli/schemas/{tool,safeguard}.schema.json`,
`.vrooli/repo-contract.json`

### P-012 — Nothing verifies that a platform declaration is true

**Status:** open · **owner:** the control plane · scope note below

**Symptom:** A manifest can declare `platforms: ["linux","macos","windows"]`
with a handler that only compiles on Linux, and every surface in this
scenario — grid, ladder, fleet — will report it as covered. The declaration is
the only evidence anything reads.

**Root cause:** no validator anywhere compares a declared platform claim against
what the code can build for. No `.vrooli/repo-contract.json` rule mentions
platform or portability; nothing inspects build tags or `GOOS` against a
manifest. A 2026-08-21 sweep cross-compiled all 313 Go modules for
`windows/amd64` and `darwin/arm64` by hand to establish this, and no gate
preserves that result.

**Scope note — this is not a cell and must not become one.** Portability is
measurement and has a denominator; conformance is pass/fail and does not.
Forcing "is this declaration true" into a coverage ratio is the mistake the
production-ledger archetype exists to prevent, and it is why
`platform-code-audit` stays deferred in
[`../concepts/DOMAINS.md`](../concepts/DOMAINS.md#deferred-domains).

**Workaround:** none. Treat `qualified` on a non-local OS as an unverified
claim until the gate exists.

**Real fix:** a declared-versus-actual gate in the control plane beside the
resolver, failing loudly in CI, feeding the `platform-code-auditor` lane as
evidence rather than becoming a ratio here.

**Refs:** `internal/deployability/`, `.vrooli/repo-contract.json`,
`docs/infra-health/operating/OPERATING_MODEL.md` loop 2

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

### 2026-08-20 — Four sensor-channel defects fixed; two model corrections

**Symptom:** The instrument ran green while reporting almost nothing. Trust
returned an empty distribution, 62–69% of readings were `UNTRUSTED`
non-deterministically, 24 live scenarios were classified as ghosts, and the
ranked surface could only ever emit coverage gaps.

**Root causes, all now fixed:**

1. **N+1 saturation reads.** The autoheal reader issued one `GetSaturation` per
   check inside a 10s per-source deadline. Whatever did not fit was marked
   `UNTRUSTED` — a deadline reported as a verdict about the plant. Replaced by
   `ChecksService.ListSaturation`, one tally for the whole registry.
2. **Sequential qualifiers.** `GetReconcile` derives the core set through a
   subprocess taking ~4s, and ran ahead of the cheap reads, pushing them past
   the same deadline. The three qualifier reads are now concurrent, and a
   failure is scoped to the readings that qualifier actually governs —
   reconcile only classifies `scenario-*` checks, so an unreadable reconcile no
   longer blanks the substrate readings.
3. **Ghost meant the wrong thing.** `reconcile.Compare` flagged a check as a
   ghost when its target was absent from the *core set*, not when the target
   was *gone*. Ghost readings are excluded from every aggregate, so 24 running
   scenarios were silently dropped from uptime. Existence and
   should-be-supervised are now separate questions with separate inputs, and
   `out_of_scope` is its own answer.
4. **`plainNameSet` over-stripped.** Normalizing the name side with the same
   `scenario-` TrimPrefix used on the id side turned `scenario-authenticator`
   into `authenticator`, so scenarios genuinely named `scenario-*` were
   reported as ghosts *and* as unsupervised plant at once.

**Model corrections:**

- **Saturation is not quietness.** It was derived from `!transitioned`, which is
  also true of every check steadily at OK — 61 of 79 here. Since saturated
  readings are excluded from aggregates, that emptied the uptime figure of
  exactly the checks that were fine. Saturation now requires a non-normal
  current status as well as no transition.
- **Ungraded is not one state.** 20 of 21 bars carried prose-only deadbands, so
  every reading fell through to `NOT_EVALUATED` with no stated reason.
  `NOT_GRADEABLE` now distinguishes "the bar authors no threshold" from "the
  reading could not be trusted", and `LoadSetpoint` rejects a bar that is
  neither gradeable nor explains why.

**Refs:** `scenarios/vrooli-autoheal/api/internal/reconcile/reconcile.go`,
`scenarios/vrooli-autoheal/api/internal/handlers/typed_connect.go`,
`api/internal/sources/autoheal.go`, `api/internal/focus/condition_source.go`,
`api/internal/coverage/service.go`, `api/internal/condition/banding.go`.

### 2026-08-20 — `substrate/SB6`: no core-dump sensor exists anywhere

**Symptom:** A userspace process crash loop is invisible to every reliability
surface unless it also fails a liveness check.

**Root cause:** Nothing in the platform reads `core_pattern`,
`systemd-coredump` or `coredumpctl`. A repo-wide search for those terms returns
zero hits. `system-panic-evidence` and `system-pstore-evidence` cover *kernel*
panics via kdump and pstore; neither sees a userspace core dump.

**Workaround:** None. The gap is declared as `substrate/SB6` (`MISSING`,
`gap_opened_on: 2026-08-20`) so it is counted and aged by
`coverage open-loop` rather than being silently absent.

**Real fix:** A crash-accounting sensor owned by `vrooli-autoheal`, reading the
coredump journal and attributing each crash to an owning scenario or run. Then
`SB6` moves to `NOW` and gains a bar.

**Owner:** unassigned.
