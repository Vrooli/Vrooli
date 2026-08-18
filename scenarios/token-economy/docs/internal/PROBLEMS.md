# Problems — Token Economy

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear. The first entry is deliberately the most
important one: **this scenario is documented and not implemented**, and a reader
should hit that before drawing any conclusion from the rest of the docs.

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

### 2026-08-18 — The scenario is fully documented and entirely unimplemented

**Symptom:** `PRD.md`, `requirements/`, `docs/` and `experience/` describe seven
domains, 31 requirements and 12 pages. None of it exists in code. A reader
skimming the documentation could reasonably believe the scenario works.

**Root cause:** Deliberate. The initialization pass covered orientation gates
0–5b (documentation) and stopped before gate 6 (first real vertical slice) and
gate 7 (example-domain removal), which are implementation work.

**Workaround:** Trust `make orient` (currently 7/9) and this file over any
narrative reading of the docs. Every requirement is `status: planned`, every
new experience page is `status: draft`, and every validation entry names a
*planned* test file rather than an existing one.

**Real fix:** Build the first real domain per gate 6, then remove the example
domain per gate 7. Suggested order follows the PRD's launch sequencing:
`mints` → `journal` → `grants` → `holders` → `earning` → `catalog` →
`redemption` → console.

**Owner:** unassigned.

**Refs:** `docs/START-HERE.md` gates 6–7, `docs/internal/PROGRESS.md`.

---

### 2026-08-18 — Requirement validation refs point at files that do not exist

**Symptom:** Every `validation` entry in `requirements/` carries its intended
test path inside `notes` (as ``Planned test file: `…` ``) rather than in the
schema's `ref` field.

**Root cause:** `vrooli scenario requirements validate` treats a `ref` pointing
at a non-existent path as an **error** (`intent.ref_missing`), so a planned test
cannot be declared through `ref` before it is written. The intended paths were
moved into `notes` to keep the registry green while preserving the strategy.

**Workaround:** Read `notes` for the intended path. The `[REQ:TKE-…]` tag named
in each note is what auto-sync will match once the test exists.

**Real fix:** When each test lands, move its path from `notes` back into `ref`
and flip the validation `status` from `planned` to `implemented`. Auto-sync then
updates requirement status from tagged results.

**Owner:** whoever implements each domain.

**Refs:** `requirements/01-must-ship/module.json`,
`scenarios/test-genie/docs/reference/requirement-schema.md`.

---

### 2026-08-18 — The consumer market hypothesis is unvalidated

**Symptom:** `docs/business/MONETIZATION.md` and `GO-TO-MARKET.md` state a
customer, a pain and a competitive gap with real confidence. None of it has been
checked with a household.

**Root cause:** The positioning is reasoned from a 2026-08 competitive scan
(Greenlight, BusyKid, FamZoo, Modak, Acorns Early), not from user contact.

**Workaround:** Treat the P1 tier as blocked. The PRD sequences it explicitly
behind this signal, and `MONETIZATION.md` records the threshold: sustained use
across four consecutive weeks by one household, with the *child* opening the
holder view unprompted.

**Real fix:** Run the P0 loop in one real household. Allow the hypothesis to
fail — the scenario's internal role as `treasury`'s rehearsal surface justifies
building it either way.

**Owner:** operator.

**Refs:** `docs/business/MONETIZATION.md` § Validation Plan.

---

### 2026-08-18 — Earning-submission dedup window is undefined

**Symptom:** `DATA.md` says earning submissions are "retained for the dedup
window, then prunable" without naming the window.

**Root cause:** The window is a product decision that depends on how adapters
actually retry, which is unknown until `TKE-P1-009` wires a real one.

**Workaround:** None needed yet; nothing is implemented.

**Real fix:** Fix the window before `TKE-P0-007` ships. Too short and a slow
adapter retry double-grants; too long and the table grows without bound.

**Owner:** whoever implements `earning`.

**Refs:** `docs/concepts/DATA.md` § Retention And Deletion.

---

### 2026-08-18 — Hard deletion for a departing holder is undesigned

**Symptom:** Holder removal tombstones the record so their events stay
attributable. There is no erasure path.

**Root cause:** Correct for audit and for the append-only contract; wrong for a
right-to-erasure request. The tension is real and was resolved in favor of audit
because the deployment is self-hosted and single-household.

**Workaround:** Acceptable while self-hosted. The operator holds the only copy
and can destroy the database.

**Real fix:** If this is ever offered as a hosted product, design an erasure
path that preserves aggregate integrity **before** launch, not as a retrofit.
This stores behavioral data about minors; see `DATA.md` § Privacy Notes.

**Owner:** unassigned; blocking for any non-self-hosted deployment.

**Refs:** `docs/concepts/DATA.md` § Privacy Notes, `docs/internal/SECURITY.md`.

---

### 2026-08-18 — Inherited: the `notes` example violates the 44px tap-target floor

**Symptom:** `experience-manager spec validate token-economy` fails with
`experience.floor_tap_target_size` — *"observed 36.00px below required
44.00px"* — located at `experience/pages/notes.json`.

**Root cause:** Inherited from the `react-vite` template, not authored here.
`DESIGN.md` declares `touch: 44px` / `touchTarget: "44px"` as a **binding**
floor, and the template's own example UI violates it:
`ui/src/features/notes/AttachmentUpload.tsx` styles its file input with
`file:py-1.5`, producing roughly a 36px target. The failing claim is an implicit
design floor applied by experience-manager, not a claim authored in this
scenario's specs. It only began to surface once `make setup` completed and a
live accessibility capture became possible — before that the same page reported
`capture_unavailable` and the floor could not be evaluated.

**Workaround:** None applied, deliberately. The two dishonest fixes were
rejected: weakening the claim tier would hide a real accessibility violation,
and editing the example UI would be work that `template-manager detemplate`
deletes. This scenario's own pages all target the floor or exceed it — the
holder view commits to the *upper* end of the guidance because it may be used
by a child (`DESIGN.md` § Scenario adaptation).

**Real fix:** Two independent paths, and both should happen. In this scenario,
the finding disappears at gate 7 when `template-manager detemplate token-economy`
removes the `notes` example. Upstream, the template itself should be fixed so
every future generated scenario does not inherit a failing accessibility floor
on day one — that is a `template-manager` defect and belongs in scenario-qa,
not here.

**Owner:** unassigned for this scenario (resolves at gate 7); template-manager
for the upstream fix.

**Refs:** `ui/src/features/notes/AttachmentUpload.tsx:69`, `DESIGN.md:47`,
`DESIGN.md:276`, `experience/pages/notes.json`.

---

### 2026-08-18 — `business_evidence_stale` after a suite run

**Symptom:** `vrooli scenario requirements validate token-economy` reports a
warning: *"snapshot generated … predates the latest suite run …"*, pointing at
`coverage/requirements-sync/latest.json`.

**Root cause:** Ordinary staleness, not a defect. The coverage snapshot is
refreshed by requirement auto-sync *during* a comprehensive suite run; validating
between runs leaves the snapshot older than the most recent run.

**Workaround:** Harmless at documentation stage — every requirement is
`status: planned` with no test evidence to sync. Validation still reports
`PASSED`.

**Real fix:** Resolves on its own once tests carry `[REQ:TKE-…]` tags and
auto-sync has real evidence to write.

**Owner:** unassigned.

**Refs:** `coverage/requirements-sync/latest.json`.

---

### 2026-08-18 — Inherited: a freshly generated scenario does not pass its own suite

**Symptom:** `test-genie` verdict **FAIL** — 7 of 22 phases failing on run
`20260818-220435-6e34f3d0`: `contracts`, `ui-health`, `dependencies`, `unit`,
`workflow`, `experience`, `measures`.

**Root cause:** Inherited from `react-vite` v1.6.5, established by evidence
rather than assumption: the **first run taken immediately after generation, and
before any authoring in this scenario**, already returned FAIL with
`unit / TEST_EXECUTION_FAILURE` as a `foundation_blocker`, plus `contracts`,
`ui-health`, `dependencies`, `quality`, `docs`, `storage`, `workflow` and
`experience` findings.

The `unit` blocker is 20 vitest failures across 7 files, all in the removable
`notes` example, all i18n resolution — tests expect resolved English but receive
the raw key (`notes.measure.thisWeek`, `notes.attachmentsLabel`). The keys exist
in `ui/src/i18n/locales/en.json` with correct values and `pnpm strings:gen`
reports *already up to date*, so the resource file is fine and the
test-environment i18n runtime is what is not resolving. **Go API tests all
pass** (`go test ./...` from `api/` is green).

**What this scenario's authoring actually changed**, for the record:

| Finding | Direction | Cause |
|---|---|---|
| `intent.ot_orphan` × 4 | **resolved** | P1/P2 targets gained linked requirements |
| `experience.state_missing` | **resolved** | DESIGN.md-required UX states declared on every page |
| `dependency.declared-without-import-evidence` | **introduced, expected** | Three scenario dependencies declared in `service.json` that no code imports yet |
| `measures.uncovered-domain` | **introduced, expected** | Seven domains declared in `DOMAINS.md` that do not exist in code yet |
| `experience.floor_tap_target_size` | **newly visible** | Pre-existing; was masked by `capture_unavailable` until `make setup` enabled a live capture |

The two introduced findings are the direct, intended consequence of documenting
ahead of code and resolve as each domain lands. Neither was papered over.

**Workaround:** None. Do not treat the suite as a regression signal for this
scenario until the inherited baseline is fixed upstream; compare against the
post-generation baseline instead.

**Real fix:** Upstream in the template. Filed to scenario-qa as
**`knw-1787091113421895908`**.

**Owner:** template-manager.

**Refs:** run `20260818-220435-6e34f3d0`; baseline run `20260818-213745-ae2f73b2`.

---

### 2026-08-18 — Inherited: `make orient` reports scaffold-health green on submission, not verdict

**Symptom:** `make orient` reported `scaffold-health` complete while the
underlying Test Genie verdict was FAIL.

**Root cause:** The gate check in `.vrooli/orientation.json` is
`{"kind": "command", "run": "make test", "timeout": "15m"}`. `make test`
**submits** a server-owned run and exits 0 immediately, returning a run id and a
duration estimate — not a verdict. Only
`test-genie runs wait --json <scenario> <run-id>` establishes the outcome, per
`docs/TESTING.md`. The gate therefore passes on submission.

**Consequence, and why it is worth recording here:** this is the gate an agent
trusts *before* starting product work, so a false green on gate 0 makes the
inherited suite failure above invisible. It misled this session — an earlier
handoff reported the baseline as healthy on the strength of `make test` exiting
0. Corrected once the verdict returned.

**Workaround:** Never trust `make orient`'s scaffold-health alone. Block once on
`test-genie runs wait --json token-economy <run-id>` and read the verdict.

**Real fix:** Upstream. Filed to scenario-qa as **`knw-1787091113421895908`**
together with the baseline failure, since the two compound.

**Owner:** template-manager.

**Refs:** `.vrooli/orientation.json` step `scaffold-health`; `docs/TESTING.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Docs vs. code | Total, by construction. `DOMAINS.md` declares seven domains; `api/internal/` contains only the `notes` example and infrastructure. This is not drift in the usual sense — nothing has diverged, because nothing was built — but a drift audit run today would report it as such. | `measures.uncovered-domain` blocks the `measures` phase. Expected and correct while documentation leads implementation. | Build the domains in the PRD's launch order. Each landed domain removes one row of this gap. |
| Declared dependencies vs. imports | `.vrooli/service.json` declares `scenario-authenticator`, `notification-hub` and `agent-manager`; no code imports any of them yet. | `dependency.declared-without-import-evidence` (INFO) blocks the `dependencies` phase. | Resolves when `holders` binds to the authenticator and `journal` resolves provenance. Reverting the declarations was rejected — it would put `service.json` in disagreement with `INTEGRATIONS.md`, trading an honest INFO finding for a silent inconsistency. |
| Experience specs vs. built UI | 12 pages declared `draft`; only the scaffold's dashboard, notes and settings routes exist. | None — `draft` status makes reconciliation advisory by design, which is the correct use of the status. | Flip each page to `active` as its route is built and stable selectors land, then raise its claims from `aspirational` to `machine`. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
