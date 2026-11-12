# Problems — Research Desk

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

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

### 2026-11-11 — generated scaffold lacked committed CLI evidence

**Symptom:** the first unit phase failed `TestPrimitiveEvidenceArtifactCurrent` because
`.vrooli/generated/cli-primitive-evidence.json` did not exist.

**Root cause:** `template-manager generate` does not write the CLI primitive evidence file the
CLI tests require (template defect, outside this scenario).

**Workaround:** regenerated with `UPDATE_CLI_EVIDENCE=1 go test -run
TestPrimitiveEvidenceArtifactCurrent .` in `cli/`.

**Real fix:** the react-vite template generates the evidence at scaffold time.

**Owner:** template-manager.

**Refs:** run 20261111-212023-3b570a18.

### 2026-11-11 — dependencies phase exceeds its 15-minute deadline

**Symptom:** `vrooli scenario test research-desk --phases dependencies` failed with "phase
dependencies did not complete before its 15m0s deadline" (run 20261111-212302-cd9f5c02); it
also reported two approved-dependency governance warnings on the first run.

**Root cause:** unknown; not diagnosed (shared test infrastructure, outside this segment).

**Workaround:** validate with `structure` and `unit` phases.

**Real fix:** diagnose the dependencies phase duration with Test Genie's owner.

**Owner:** unassigned.

**Refs:** runs 20261111-212023-3b570a18, 20261111-212302-cd9f5c02.

### 2026-11-11 — Codex goal that never completes hot-loops

**Symptom:** in the design probe, a goal whose work was already done produced about 90 task
starts in two minutes.

**Root cause:** Codex re-drives an active goal until it is marked complete.

**Workaround:** the launch proof must use a kernel that completes; real kernels already
end the turn after recording a review.

**Real fix:** none needed in the desk; the launch proof uses a completing kernel
(docs/concepts/SYSTEM.md §12, G2).

**Owner:** research-desk.

**Refs:** probe session `4a72d7a3-955c-491c-9c69-7b33867868c3`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

### 2026-11-12 — remaining-quota lens (ruling B3) not built

**Symptom:** the launch screen shows no remaining usage for the supervisor model; a launch can
start into a quota that is nearly spent.

**Root cause:** deferred by the lead's ruling; no quota read is wired.

**Workaround:** `limited` is derived only from the goal row's status (`usage_limited`,
`budget_limited` or `blocked`) while the TUI is present (`state.go` in the API package `internal/home`), so a
limit shows once Codex has hit it, not before.

**Real fix:** a quota lens read from the owner of usage data, shown in preflight.

**Owner:** unassigned.

**Refs:** `docs/concepts/SYSTEM.md` §5.
