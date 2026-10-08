# Problems — Scenario to Plugin

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear.

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

### 2026-08-19 — Published versions are never re-verified against a changed CLI

**Symptom:** A skill published today documents commands that exist today. If
the wrapped scenario later renames or removes one of those commands, the
published package silently becomes wrong. Users discover it; we do not.

**Root cause:** The drift gate (`PLG-CONF-DRIFT`) runs at composition time
only. Nothing re-runs it against already-published versions when the wrapped
CLI moves.

**Workaround:** None. Drift in a published version is currently discovered by
report. Revoke, fix the skill, republish.

**Real fix:** A scheduled job that re-resolves each published version's
documented commands against the current `cli-manifest`, using the recorded
`manifest_pins` revision as the baseline for the comparison. This is the
reason `PLG-CONF-DRIFT-PIN` records the revision — the data will be there.

**Owner:** unassigned.

**Refs:** `docs/internal/SECURITY.md` (Security Gaps), `OT-P0-002`,
`PLG-CONF-DRIFT-PIN`, `docs/operations/OBSERVABILITY.md` (Telemetry Gaps).

### 2026-08-19 — Retention rules are documented but not enforced

**Symptom:** Artifact trees, rehearsal logs, and superseded declaration
snapshots accumulate without bound.

**Root cause:** `docs/concepts/DATA.md` defines retention windows; no prune
job implements them.

**Workaround:** Manual capture-store cleanup. Do not prune anything referenced
by a `publications` row.

**Real fix:** A prune job honoring the DATA.md table, with `publications`,
`revocations`, and their referenced attestations explicitly excluded — losing
publication history loses the revocation fan-out.

**Owner:** unassigned.

**Refs:** `docs/concepts/DATA.md` (Retention And Deletion).

### 2026-08-19 — The `skill-registries` channel has no machine-evaluable trigger

**Symptom:** Offer Desk carries `skill-registries` as `CANDIDATE`, but
`offers gates-evaluate` returns no evaluation for it. The channel cannot be
promoted from evidence — only by argument.

**Root cause:** The channel's activation trigger exists as prose in
`docs/monetization/catalogs/channels/skill-registries.md` and was never
registered with `offer-desk offers gates-trigger`.

**Workaround:** None needed yet; the channel's prerequisites are unmet
regardless.

**Real fix:** Register the trigger. It is cheap and does not depend on this
scenario shipping. Not this scenario's code, but it is this scenario's
downstream consumer, so it is tracked here until someone owns it.

**Owner:** unassigned (monetization).

**Refs:** Offer Desk channel node `153e6270-47db-442b-ae22-bf34a95b8090`,
`docs/business/MONETIZATION.md` (Current Status).

### 2026-08-19 — The business-health wizard could not be used to author this PRD

**Symptom:** Every `WizardService` RPC — `StartSession`, `SubmitAnswers` —
returns an empty reply after exactly 51 seconds. The API logs
`status=200 duration=51.04s` but the client receives nothing.

**Root cause:** The wizard's state-render path runs a capability-dedup search
whose reranker calls Ollama. Ollama reports healthy but `generate` returns
`signal: killed` on every attempt, so the search retries for ~51s and the
response never lands. It is an environmental Ollama wedge plus a missing
degradation path in the dedup search — the search should degrade to no hints
rather than blocking the RPC.

**Workaround:** Author `PRD.md` directly against
`scenarios/business-health/docs/reference/canonical-prd-template.md`, then
validate with `vrooli scenario requirements validate`. The validator, not the
wizard, is the contract; the wizard is a conformant-by-construction
convenience. This PRD validates clean with zero findings.

**Real fix:** Belongs to business-health, not here: bound the dedup search and
degrade to zero hints on reranker failure. Filed as an observation rather than
a fix because the defect is in another scenario.

**Owner:** unassigned (business-health).

**Refs:** `business-health` API log, `WizardService/StartSession`;
`docs/internal/PROGRESS.md` 2026-08-19.

### 2026-08-19 — Every control in this scenario is designed, not proven

**Symptom:** The threat model, gate ordering, and fail-closed claims are
documented in detail. No code implements any of them.

**Root cause:** Expected. The scenario is pre-implementation by intent — the
contract was authored first.

**Workaround:** Do not cite any control here as active. Do not describe the
scenario as having security properties it has not demonstrated.

**Real fix:** The release checklist in `docs/operations/DEPLOYMENT.md` step 7
requires a gate self-test against deliberately broken fixtures before any real
publication: a skill documenting a removed command must fail conformance, an
unpinned install must fail, and a package carrying a credential literal must
fail before any network call. Until that self-test passes, this scenario has
proven nothing.

**Owner:** unassigned.

**Refs:** `docs/operations/DEPLOYMENT.md` (Release Checklist),
`docs/internal/SECURITY.md` (Security Gaps).

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| `docs/internal/SEAMS.md` | Still describes only the template's example seams. The six real pipeline domains register no seams yet. | Tests that depend on domain seams cannot be written until the registry is real. | Register each domain's seams as that domain lands, before writing its tests. |
| `docs/reference/api-endpoints.md`, `cli-commands.md` | Describe the planned surface, not an implemented one. Marked `draft` in `docs/manifest.json` to say so. | Reference docs that read as current truth would misreport maturity. | Regenerate from the proto and `cli/manifest.json` once the first domain ships. |
| `experience/` | Page specs are L0/L1 intent for routes that do not exist yet. | Experience maturity cannot exceed intent until routes are real. | Raise to L2+ as each route lands with stable roles and testids. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
