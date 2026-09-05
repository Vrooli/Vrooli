# Problems — Business Health

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

_None yet._

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

### 2026-07-02 — docs phase carries a permanent `misplaced_doc` false positive on root `manifest.json`

**Symptom:** The docs phase reports `misplaced_doc` (severity 3) for `manifest.json`, suggesting it be moved to `docs/manifest.json`.

**Root cause:** knowledge-observatory's `misplacedCandidate` matches root-level files by basename against manifest-declared docs. The react-vite template intentionally ships `manifest.json` (scenario-architecture zones contract, consumed by architecture-cartographer) at scenario root; it collides by basename with `docs/manifest.json`.

**Status:** Filed to scenario-qa as `bug-inbox/code-defect/dochealth-misplaced-doc-root-architecture-manifest`. Do NOT move the file — it would break the architecture contract. Treat the finding as explained until the docs-health exemption lands.

### 2026-07-02 — `TestProtoConnectParity` red until the first Connect domain lands

**Symptom:** `api/internal/modules/registry_test.go` fails: `AllProtoFiles() returned no entries`.

**Root cause:** Post-detemplate the scaffold has no Connect-mounted domain module; the parity test asserts the desired end state (≥1 registered domain). Phase 2 of `docs/plans/business-health-provider-plan.md` mounts the validation/report/wizard services and resolves this without any test change.

**Status:** Expected transient during the documentation-first window; do not weaken the test.

### 2026-07-02 — three suite phases explained-red on template/tooling issues (not this scenario's code)

**Symptom:** `standards`, `docs`, and `tidiness` phases fail; the other suite phases pass (run 20260702-061719-506c2fc4).

**Root cause:** All three severity-3 drivers predate this scenario's domain code: (1) docs → the `misplaced_doc` false positive on the root architecture `manifest.json` (filed: `dochealth-misplaced-doc-root-architecture-manifest`); (2) standards → `OWASP Security Headers` findings from the deprecated scenario-auditor three-hop chain against the template-fresh scaffold — the chain `docs/plans/business-health-provider-plan.md` is dismantling; (3) tidiness → the template ships `no_prod_import_test.go` twice by design (api and cli are separate Go modules that cannot share the meta-test without a new shared package).

**Status:** Tracked here so later phases don't re-diagnose. Re-check after knowledge-observatory fixes the misplaced heuristic and after the standards chain retires (plan phase 11).

### 2026-07-02 — sync re-fabricates requirement completion on every comprehensive run

**Symptom:** After each comprehensive suite run, requirement statuses flip to `complete` (and validations to `implemented`) for domains that do not exist yet (e.g. the wizard — no `[REQ:BH-WIZ-...]` tag exists anywhere in the tree), and PRD checkboxes get checked.

**Root cause:** test-genie's requirements syncer marks test-typed validations implemented without binding them to any `[REQ:ID]`-tagged evidence, and ignores `requirements/index.json::_metadata.auto_sync_enabled: false`. Filed to scenario-qa as `bug-inbox/code-defect/requirements-sync-refless-validations-fake-complete`.

**Status:** Statuses are reverted to `planned` when noticed, but every suite run re-poisons them until the upstream fix lands. Before committing or judging this scenario's honesty, re-check `grep -c '"complete"' requirements/*/module.json` against what actually shipped. The evidence-traceability domain here cannot self-detect this shape yet (it would need `[REQ:ID]` tag scanning, a future ratchet).

### 2026-07-02 — playbooks: `allow_empty_test_pool` acknowledged (pure UI-smoke BAS cases)

**Symptom:** The playbooks phase failed with "routed e2e ran without exercising the test pool: 0 test-mode requests reached RoutedDB" while all 4 BAS workflows passed.

**Root cause:** The current BAS cases are observer-mode UI smokes (single navigate step + render assertions); their in-browser API calls don't round-trip the X-Vrooli-Test-Mode header, so the routed-lease isolation proof sees zero test-pool traffic — the phase's hard-fail default for suites that never touch the API.

**Status:** Acknowledged via the documented opt-out (`playbooks.allow_empty_test_pool: true` in `.vrooli/testing.json`), which downgrades the invariant to a warning observation. Revisit when a mutating BAS flow (wizard apply / manual-log) lands — that flow SHOULD prove routed isolation and the opt-out should then be removed.
