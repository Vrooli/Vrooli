# Problems — Security Health

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

### 2026-06-01 — UI is still the react-vite placeholder (Phase F not built)

**Symptom:** The backend (Phases C–E) is complete, but the UI is still the
generated placeholder shell + `notes` example. The Posture / Dependencies /
Secrets pages and the embeddable security-posture badge widget (source plan
Phase F) are not built, and the `notes` example domain has not been removed
(START-HERE Gate 7).

**Root cause:** Scoping — the producer loop (the plan's primary purpose) and the
justifying dependency-intelligence feature were prioritized. The placeholder UI
boots green, so the scenario self-test stays green in the meantime.

**Workaround:** Use the CLI (`security-health validate scenario <name> --json`,
`security-health deps search --vulnerable-only`) and the Connect API directly.

**Real fix:** Build Phase F — replace the AppShell + home with Posture (validation
findings, severity-grouped, remediation, re-scan), Dependencies (the SBOM search
with ecosystem + vulnerable-only filters), Secrets (gitleaks findings, redacted),
and the `@vrooliWidget` posture badge; then remove the `notes` example (Gate 7).

**Owner:** unassigned.

**Refs:** source plan Phase F; `ui/src/`; `docs/START-HERE.md` Gates 6–7.

### 2026-06-01 — AI/Qdrant semantic search deferred (dependency index is TEXT-only)

**Symptom:** `DependencyService.Search` serves TEXT + structured-filter queries
over the SQLite corpus. A `MODE_AI` request degrades to TEXT (`mode_used=TEXT`),
and `Status.qdrant`/`ollama` report availability but nothing is embedded.

**Root cause:** Proper semantic search needs a pre-embedded vector index
(Qdrant); embedding the whole corpus on every query (the only no-Qdrant option)
does not scale. The faithful cli-health vectorstore clone (~600 lines) was
deferred in favor of shipping the always-available TEXT + structured core, which
already answers the headline query ("which scenarios are exposed to CVE-X?")
deterministically via `--vulnerable-only`/`--name-glob`.

**Workaround:** TEXT + structured filters cover the high-value queries today.
The `qdrant`+`ollama` deps stay declared `required:false`/`try_start` (their
documented `degraded_behavior` is exactly this TEXT fallback).

**Real fix:** Clone `scenarios/cli-health/api/internal/aisearch/{embedder,
vectorstore,reconciler}.go`, retarget points to `DependencyRecord` in the
`security-health-deps` Qdrant collection, embed on reconcile, and have
`Service.Search` ANN-rank in MODE_AI. The `Service.aiProbe` seam + `ModeUsed`
plumbing are already in place for it.

**Owner:** unassigned.

**Refs:** `internal/dependencies/service.go` (aiProbe seam), source plan Phase E.

### 2026-06-01 — Template ships a critical vitest dev-dependency CVE (fleet-wide)

**Symptom:** Every react-vite scenario's `ui/pnpm-lock.yaml` pins `vitest <4.1.0`
(GHSA-5xrq-8626-4rwp, critical). security-health's own dev-dependency audit
flags it.

**Root cause:** The react-vite template's pinned vitest predates the fix.

**Workaround:** It is a **dev-only** dependency (test runner, not in the shipped
artifact), so security-health's pnpm-audit scanner downgrades it to WARNING via
the prod/dev split — it does not gate R1. security-health validates itself clean
(errors=0).

**Real fix:** Bump the react-vite template (and existing scenarios) to
vitest ≥ 4.1.0. Cross-cutting — file via `report-bug` against the template, not
fixed here.

**Owner:** unassigned (template-wide).

**Refs:** `internal/validation/scan_pnpm_audit.go` (prod/dev split).

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
