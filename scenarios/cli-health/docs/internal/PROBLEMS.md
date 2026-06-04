# Problems — CLI Health

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

### Pre-existing STANDARDS + security campaign blocks `vrooli scenario test`

**Symptom:** `vrooli scenario test cli-health` fails the `standards` phase
(`fail_on=high`, highest=critical) and reports security findings. As of
2026-06-03: 236 findings (4 blocker/error, 111 warn, 121 info), 36 standards
violations (2 critical, 33 medium, 1 low).

**Cause:** Template/scaffolding and repo-wide debt, **not** the
aisearch-adoption-hardening work:
- **critical** `api/internal/httpc/doer.go:34` "HTTP Client Without Timeout" —
  false-positive: the flagged line is a *comment* (`// callers pass
  &http.Client{...} directly`), not code.
- **critical** `Makefile` "Scenario Required Structure / lifecycle wrapper
  targets" — template structural gap.
- **medium** "Unstructured Logging in API" across `aisearch/service.go`,
  `command_index.go`, `main.go` — the whole package uses `log.Printf` by
  convention; converting to `slog` is a package-wide refactor (separate
  campaign). New log lines added by this work follow the existing convention.
- **security** `osv.GO-2026-5038 / GO-2026-5039` stdlib@1.25.0 (api + cli
  `go.mod`) — bump toolchain to ≥1.25.11; repo-wide, toolchain-level.

**Workaround:** The actual code gates are green — `gofumpt`, `go build`,
`golangci-lint`, `go test` (api + cli + packages/aisearch-go), and the UI
`eslint` / `tsc` / `vitest` all pass. Track the standards debt as a campaign.

**Real fix:** `architecture-cartographer campaign create cli-health
--from-audit <audit.json>`; fix the httpc false-positive in scenario-auditor's
`resource-management-v1` rule; add the Makefile lifecycle targets; bump the Go
toolchain repo-wide.

**Owner:** unassigned (campaign).

**Refs:** `docs/plans/aisearch-adoption-hardening-plan.md`; auditor run
`scenario-auditor standards scan cli-health`.

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
