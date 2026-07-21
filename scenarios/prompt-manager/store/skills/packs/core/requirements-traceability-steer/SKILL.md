## Steer focus: Requirements Traceability

`scenarios/{{TARGET}}/`'s requirements registry is a **claim** about what the scenario does and how you'd know; when behavior changes and the registry doesn't, the claim is a lie. Your job is to **make the registry true, not green** — every requirement tied to real evidence, every PRD operational target tied to requirements, every status earned rather than declared.

The drift signal you are answering comes from the test-genie `business` phase's typed findings (source `BUSINESS`): starter-template registries, requirements with no validation, dangling validation refs, prd_refs that match no PRD operational target. Move `{{TARGET}}` **up the Traceability Maturity Ladder** (§2) until the latest business-phase run is findings-clean — do not invent new requirements to look thorough.

Required reading:
- `path:scenarios/test-genie/docs/requirements/STATUS_MODEL.md` — how declared status, live evidence, and the sync snapshot compose (statuses are *earned* from `[REQ:ID]`-tagged test results, not asserted).
- `path:scenarios/test-genie/docs/requirements/IMPROVING_COVERAGE.md` — the `[REQ:ID]` tagging mechanics per language and the validation-ref formats (`path/file.go::TestName`).
- `path:scenarios/business-health/docs/reference/canonical-prd-template.md` — the PRD shape; operational targets (`OT-P0-001 | Title | …`) are the IDs `prd_ref` must match.

Read first when present (prior findings — continue, don't restart):
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — deferred requirements work or a known-stale module.

> Universal authoring/quality bars (intent statement, convergence patterns, anti-gaming framing, the agent memory loop) are canon in `path:docs/agent-system/SKILL_AUTHORING.md` and are not restated here.

---

### 1. Scope Boundaries

**In scope** (anchored to `scenarios/{{TARGET}}/`):
- truing `requirements/*.json` to actual behavior: descriptions, statuses, criticality, `validation[]` refs.
- `prd_ref` ↔ PRD.md operational-target linkage (both directions: dangling refs and uncovered targets).
- tagging existing tests with `[REQ:ID]` and pointing `validation[].ref` at them.
- flipping PRD operational-target checkboxes **only** when the linked requirements are complete with passing evidence.
- recording deferred registry work in `docs/internal/PROBLEMS.md`.

**Out of scope** (hand off):
- authoring a brand-new PRD or renegotiating product scope → the business-health wizard flow (`business-health wizard`) (`vrooli scenario requirements validate` tells you *what's unlinked*, not *what to build*).
- test quality / writing new test suites → the `test` skill; here you only **link** evidence that exists (write a test only when a P0 requirement has none at all).
- the requirements **sync** machinery itself (test-genie internals) → never hand-edit sync snapshots or force sync to make statuses move.
- CLI/API/manifest contract conformance → `cli-steer` / `api-steer`.

---

### 2. Traceability Maturity Ladder

Grade `{{TARGET}}` against this ladder, then climb only as far as the scenario's reality justifies. Every level is gated by a **runnable command**, so two agents grading the same scenario land on the same rung.

The findings command throughout (the same producer EM scores the `business` dimension with):

```bash
test-genie execute {{TARGET}} business --json   # .phases[] | select(.name=="business") | .findings[]
```

| Level | Name | What exists (verifiable artifact) | When to stop here |
|---|---|---|---|
| **0** | Starter template | `grep -rl template-starter scenarios/{{TARGET}}/requirements/` is non-empty (finding `business_starter_template`). The registry describes the scaffold, not the scenario. | Never. A starter registry is a placeholder, not a claim. |
| **1** | Honest skeleton | Registry parses and is structurally sound: `vrooli scenario requirements validate {{TARGET}}` exits 0; no error-severity business findings (`business_duplicate_req_id`, `business_import_cycle`, `business_orphaned_ref`). Requirements describe *this scenario's* intended behavior. | Only mid-build, when behavior is still moving daily. |
| **2** | Evidence-linked | Every requirement has at least one real validation ref: no `business_req_no_validation` / `business_validation_ref_missing` findings; refs resolve to files that exist. | A scenario with no automated tests yet — but then a P0 with no validation is an ERROR finding you must not waive away; write the missing test. |
| **3** | Live-traced | Tests carry `[REQ:ID]` tags and the sync snapshot shows live evidence: `vrooli scenario requirements report {{TARGET}}` shows `live_passed > 0` and a `critical_gap` of 0; `vrooli scenario requirements snapshot {{TARGET}}` is from a recent full run. | Most scenarios. Statuses now *earn* themselves via sync on comprehensive runs. |
| **4** | Findings-clean | Latest business run emits **zero** BUSINESS findings; `vrooli scenario requirements lint-prd {{TARGET}}` shows every P0 operational target linked; PRD checkboxes match requirement completion. | **The target.** Re-grade after every behavior change (§3). |

The rung is not a vanity score — write the level reached (and what still blocks the next one) into `PROBLEMS.md` so the next agent continues rather than re-discovers.

---

### 3. You changed X → check Y

Walk this table **after any change to `{{TARGET}}`**; it is the per-change reflex that keeps the registry true. Each row is deterministic — no judgment call about *whether* to check, only *what you find*.

| You changed… | Check… |
|---|---|
| Added a new capability / endpoint / command | Does an operational target + requirement for it exist? If not, draft the requirement now and link `prd_ref` (add the OT line to PRD.md, or extend it via the business-health wizard, if the PRD predates the capability). |
| Changed existing behavior | Which requirement described the old behavior? Its description, validation refs, or status is now wrong — fix whichever side is the lie (§4 ban #2 decides how). |
| Removed behavior | The requirement that claimed it: mark `not_implemented` or delete it (and its PRD checkbox), don't leave a passing-looking ghost. |
| Added a test | Tag it `[REQ:ID]` and add/confirm the `validation[]` ref pointing at it — an untagged test is invisible to sync. |
| Renamed/moved a test file | Every `validation[].ref` that pointed at the old path is now a `business_validation_ref_missing` finding — update the refs. |
| Edited PRD.md operational targets | `vrooli scenario requirements lint-prd {{TARGET}}` — every `prd_ref` must still match; every new P0 target needs a requirement. |

---

### 4. Anti-gaming bans (the registry must be true, not green)

The EM `gameguard` zeroes credit for suppression-shaped fixes; these are the suppression shapes for this dimension:

1. **Never flip a status to `complete` without a passing validation ref.** Status is downstream of evidence; `vrooli scenario requirements sync` earns it on full runs. Hand-flipping is lying to the ladder.
2. **Never rewrite a requirement's description to match drifted code without deciding which side is wrong.** Code-follows-spec or spec-follows-code is a *product decision*: cite the operational target it serves. If the OT still wants the old behavior, the code is the bug — surface it (`report-bug`) instead of papering the registry over it.
3. **Never bulk-add boilerplate requirements** (one per file/endpoint, copy-pasted descriptions) to fatten linkage counts. One requirement = one falsifiable behavioral claim someone could test.
4. **Never delete or waive a P0 requirement to silence a `business_req_no_validation` ERROR.** Write the missing test, or downgrade the criticality *with a written reason* if it was never truly P0.
5. **Never point `validation[].ref` at a file that doesn't actually validate the claim** (e.g. an unrelated test that merely exists). The ref's job is to let a human jump from claim to proof.

---

### 5. Requirement wording standard (EARS + RFC 2119)

When you author or true a requirement's `title`/`description`, write it as an
**EARS** (Easy Approach to Requirements Syntax) statement — the falsifiable-
behavioral-claim rule (§4 ban #3) made structural. Pick the template that fits:

| Pattern | Template |
|---|---|
| Ubiquitous | The `<system>` shall `<response>`. |
| Event-driven | When `<trigger>`, the `<system>` shall `<response>`. |
| State-driven | While `<state>`, the `<system>` shall `<response>`. |
| Unwanted behaviour | If `<undesired trigger>`, then the `<system>` shall `<response>`. |
| Optional feature | Where `<feature is present>`, the `<system>` shall `<response>`. |

Use **RFC 2119** keywords with their defined meanings: `shall`/`must` for
P0-linked requirements, `should` for P1, `may` for P2. Do not use those words
loosely elsewhere in the description.

Two consequences worth internalizing:

- A "the system must not X" obligation is an **unwanted-behaviour** claim about
  an observable response ("If an unauthenticated request arrives, then the API
  shall return 401"), never a bare absence ("X no longer exists"). Absence
  claims are untestable and unenumerable.
- This is an on-touch standard, not a migration: rewrite a requirement into
  EARS form when you are already editing it. Do not bulk-rewrite a registry
  just to change wording — that is churn, not truth.

---

### 6. Verification gate

Before claiming the `business` dimension closed for `{{TARGET}}`:

```bash
vrooli scenario requirements validate {{TARGET}}        # structural: exit 0
vrooli scenario requirements report {{TARGET}}          # coverage: no critical_gap
test-genie execute {{TARGET}} business --json           # producer: zero BUSINESS findings
```

All three clean = L4. Anything less, record the rung + blockers in `PROBLEMS.md`.

---

### 7. Output expectations

You **may**: edit `scenarios/{{TARGET}}/requirements/*.json` (descriptions, refs, statuses *with evidence*, criticality with reasons); tag tests with `[REQ:ID]`; fix `prd_ref` values; flip PRD checkboxes whose requirements are complete with passing evidence; update `PROBLEMS.md`.

You **must**: keep every requirement a falsifiable behavioral claim; keep validation refs resolvable; run the §6 gate before claiming done.

You **must NOT**: hand-edit sync snapshots or `coverage/` artifacts; force requirements sync (`TESTING_REQUIREMENTS_SYNC_FORCE`) to move statuses; restructure the PRD outside the operational-targets section (business-health owns the document shape — `canonical-prd-template.md`); create standalone `*_AUDIT.md` reports — findings go in durable docs.

---

### 8. Troubleshooting & Edge Cases

- **`validate` passes but the business phase still emits findings.** `validate` is structural only; the producer also checks registry drift (starter tags, empty `validation[]`, unmatched `prd_ref`). The findings list *is* the work queue.
- **`business_prd_ref_unmatched` but the target looks right.** The producer matches literal `OT-…` tokens in PRD.md; a reformatted or renamed target breaks the match. Fix the ref or the PRD line — exact ID match, no fuzz.
- **Statuses won't move after fixing refs.** Sync only runs on full (comprehensive) suite runs with no skipped required phases — `test-genie execute {{TARGET}}` (no phase filter) and check `vrooli scenario requirements snapshot {{TARGET}}` afterward. Quick/smoke runs validate but never write.
- **A requirement is real but genuinely can't have an automated validation** (e.g. a manual ops procedure). Use a `manual` validation type with `vrooli scenario requirements manual-log` evidence instead of leaving `validation[]` empty.
- **No PRD.md at all.** The prd_ref check skips silently; the scenario needs a PRD first — drive `business-health wizard` — record that in `PROBLEMS.md` and stop at L2.
