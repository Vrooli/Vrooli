# Problems — Persona

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This register contains inherited template history and current deferred
issues. The implementation plan owns the active delivery sequence; append
new defects here only when they are real, reproducible, and intentionally
deferred.

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

### 2026-08-18 — RESOLVED: `make setup` failed at `build-ui` on duplicate `@tanstack/query-core`

**Symptom (was):** `make setup` failed at `build-ui` with TS2322 on a
freshly generated scenario, because `@tanstack/query-core` resolved at
both 5.59.0 and 5.100.9 and the two `QueryClient` types were not
mutually assignable.

**Root cause:** `packages/api-base` hard-pinned `@tanstack/react-query`
at exact `5.59.0` while the `react-vite` template declared `^5.59.0`
(resolving to 5.100.9). `api-base` is consumed via `file:` and is
provisioned with its **own private `node_modules`**
(`pnpm install --frozen-lockfile --ignore-workspace`), so TypeScript
resolving `api-base/dist/testing/*.d.ts` found its private copy while
the scenario used its own. Because the two installs are independent,
neither range alignment, nor `pnpm.overrides`, nor `peerDependencies`
deduplicates them — each was tried and failed. Only exact agreement
works.

**Fix (applied):** `templates/scenarios/react-vite/ui/package.json` now
pins `"@tanstack/react-query": "5.59.0"`, matching `api-base`'s existing
exact-pin convention for every other React-ecosystem package. The
template's `pnpm-lock.yaml` was regenerated to match — mandatory,
because the template lock is copied verbatim into generated scenarios
and installed with `--frozen-lockfile`, so a specifier mismatch
hard-fails `install-ui-deps`. `packages/api-base` was left unchanged.

**Verification:** fresh generation → `make setup` exit 0, exactly one
`@tanstack/react-query` and one `@tanstack/query-core` (5.59.0) in both
the scenario and `api-base`, `ui/dist` built. Confirmed again on this
scenario after wiping `ui/node_modules`. Run
`20260818-204515-39ac879b` executes 22 phases with no `tanstack`,
`query-core`, or `TS2322` finding.

**Correction to an earlier entry in this file:** a previous revision
claimed this defect also caused `vrooli scenario start persona` to time
out and therefore blocked every suite phase. **That was wrong.** The
start timeout has a separate cause — see the next entry. The accurate
impact of this defect was: `build-ui` fails, no `ui/dist` is produced,
and orientation Gate 0 cannot pass.

**Owner:** resolved; no further action.

**Refs:** scenario-qa bug `knw-1787083684482980043` (filed at severity
`major`; the investigator may close it against this fix).

### 2026-08-18 — Inherited template debt: shared React-ecosystem versions drift between two manifests

**Symptom:** Generated scenarios install two copies of packages that must
be singletons. Originally surfaced as a TS2322 build failure on
`@tanstack/query-core`; investigation found nine misaligned packages.

**Root cause:** `packages/api-base` exact-pins its React-ecosystem
dependencies while the `react-vite` template declared caret ranges for
the same packages. `api-base` is consumed via `file:` and provisioned
with its own private `node_modules` (`--ignore-workspace`), so the two
installs resolve independently and only *exact agreement* deduplicates.

**Fix (applied):** the template now mirrors `api-base`'s exact pins for
all nine — `@tanstack/react-query`, `@testing-library/react`,
`@types/react`, `@types/react-dom`, `axe-core`, `i18next`, `react`,
`react-dom`, `react-i18next` — plus `react-router-dom`, which was
aligned **upward** (see below). Both lockfiles regenerated. Verified:
fresh generation builds with zero React-ecosystem duplicates.

**Important lesson — alignment has a direction.** The first attempt
pinned the template *down* to `api-base`'s versions, which reintroduced
patched vulnerabilities in `react-router-dom` (`GHSA-2j2x-hqr9-3h42`,
vulnerable `<6.30.4`; `GHSA-9jcx-v3wj-wh4m`, patched `>=6.30.2`). The
caret had been floating to the *fixed* 6.30.4. `api-base`'s pins are
stale by definition — that staleness is why the caret drifted — so
aligning down adopts known-vulnerable versions by construction. Both
manifests were bumped to `6.30.4` instead, which is single-copy *and*
patched. The regression was caught by the `security` phase, not by
review.

**Owner:** resolved. The durable fix remains open: two manifests
hand-synchronised on ten versions will drift again. A shared catalog
both consume would make the invariant structural.

**Refs:** runs `20260818-210259-01c4d4d6` (security failing, regression
present) and `20260818-211300-b34d85d2` (security passing, 14/22);
`packages/api-base/package.json`;
`templates/scenarios/react-vite/ui/package.json`.

### 2026-08-18 — Pre-existing npm advisories inherited from the template

**Symptom:** `pnpm audit` in a generated scenario reports advisories
that no scenario-level change can clear.

**Root cause:** The `react-vite` template's own dependency majors are
behind their patched releases. None of these were introduced by this
scenario or by the pin-alignment work above; they are inherited.

| Package | Advisories | Patched in | Template declares |
|---|---|---|---|
| `react-router` | `GHSA-337j-9hxr-rhxg`, `GHSA-wrjc-x8rr-h8h6` | `>=7.18.0` (major) | `6.30.4` |
| `react-router-dom` | `GHSA-jjmj-jmhj-qwj2` | no patch available | `6.30.4` |
| `vite` | `GHSA-4w7w-66w2-5vf9`, `GHSA-v6wh-96g9-6wx3`, `GHSA-fx2h-pf6j-xcff` (high) | `>=6.4.3` (major) | `^5.4.6` |
| `vitest` | `GHSA-5xrq-8626-4rwp` (**critical**) | `>=3.2.6` (major) | `^2.1.4` |
| `esbuild` | `GHSA-67mh-4wv8-2f99` | `>=0.25.0` | transitive |

**Workaround:** None at the scenario level. These must be fixed in the
template, and each requires a major-version migration.

**Real fix:** Template-owned major upgrades — react-router v7, vite 6,
vitest 3 — each its own piece of work with real migration cost.

**Owner:** unassigned — `react-vite` template owner. Worth filing to
scenario-qa separately from the resolved dedupe defect.

**Refs:** `pnpm audit` in `scenarios/persona/ui`.

### 2026-08-18 — Required scenario dependencies are declared ahead of the code

**Symptom:** Two related effects. First, a cold test run fails with
`start target scenario persona: lifecycle start timed out after 2m0s`,
because test-genie must boot `agent-manager`, `document-manager`, and
`secrets-manager` before persona, and that chain exceeds the 2m budget
from cold. Started directly with those dependencies already warm,
persona reports healthy in about 16 seconds. Second, the `dependencies`
phase fails with `dependency.declared-without-import-evidence`.

**Root cause:** Both are the same thing and both are correct. Those
three scenarios are declared `required` in `.vrooli/service.json`
because the PRD's P0 targets genuinely need them, but no code imports
them yet, so there is no import evidence and no reason for the boot cost
to have been paid.

**Workaround:** Ensure the three dependencies are running before
invoking the suite; the run then proceeds and executes all 22 phases.

**Real fix:** Land the first vertical slice. The import evidence appears
with the code that uses the dependencies, and the finding clears on its
own. Do **not** downgrade the declarations to optional to silence the
check — they are required by design, and the honest failing check is
worth more than a green one that lies.

**Owner:** unassigned.

**Refs:** runs `20260818-203956-8d500a61` (cold, start timeout) and
`20260818-204515-39ac879b` (warm, 22 phases executed);
`.vrooli/service.json`.

### 2026-08-18 — Documentation is authored ahead of any implementation

**Symptom:** Every operational target is unchecked, every requirement is
`planned`, and `business-health` reports `business_evidence_stale`
("suite runs exist but no requirements-sync snapshot was written").
Experience page specs carry `status: draft` and no machine-tier claims.

**Root cause:** Deliberate sequencing, not an oversight. The hard part of
this scenario is its contracts — the delegation chain, the fail-closed
rule, the release-into-handoff constraint, the OTP adapter parity rule —
and settling those before code is the point. See the final row of
[`DECISIONS.md`](DECISIONS.md).

**Workaround:** Read the docs as intent rather than as description. No
claim in this tree is evidence-backed yet, and nothing should be cited
as implemented.

**Real fix:** Build the first real vertical slice (orientation gate 6),
then complete the first real vertical slice (gates 6 and 7). Requirement
statuses become earned rather than asserted once tagged tests run and
requirements-sync writes a snapshot; experience pages flip from `draft`
to `active` as each route lands.

**Owner:** unassigned.

**Refs:** `requirements/`, `experience/index.json`,
`docs/START-HERE.md` gates 6 and 7.

### 2026-08-18 — The `agent-manager` `persona_id` binding does not exist yet

**Symptom:** `PSN-P0-002` depends on `persona_id` being carried in
`Claims.Meta` and on a `persona.act-as:<id>` scope namespace. Neither
exists in `agent-manager` today, so the delegation chain currently stops
at the account subject.

**Root cause:** The change is small but lives in another scenario.
`docs/agent-system/RUNTIME_ATTRIBUTION.md` already documents an
equivalent pending workshop for carrying `team_id` and `member_id` in
the same `Meta` map with cross-verification; `persona_id` is the
identical shape and should land the same way.

**Workaround:** None. Until it lands, a persona cannot be bound into
signed claims, so act-as cannot be cryptographically attributed and
`PSN-P0-002` cannot be satisfied.

**Real fix:** Land the `Meta` key and scope namespace in `agent-manager`
as its own small change, sequenced before this scenario's `access`
domain — step 3 of the PRD launch sequencing.

**Owner:** unassigned — coordinate with the `agent-manager` owner.

**Refs:** `scenarios/agent-manager/api/internal/identity/claims.go`,
`attenuation.go`, `scopes.go`;
`docs/agent-system/RUNTIME_ATTRIBUTION.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| Product verticals | The template example surface has been removed and the seven persona domains now own their contracts. | No current orientation impact; preserve the domain boundary when extending the scenario. | Keep proto, service, handler, CLI, UI, and tests aligned per domain. |
| UI surfaces | The console routes are implemented, while the UI health tool still reports inherited component/layout debt. | Advisory only; runtime rendering and experience validation pass. | Migrate governed components incrementally and record any deliberate exception. |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
## Work ladder

- Rung: W0
- Evidence: `swarm-manager goals list --json` returned no goal whose name, title, or description names `persona`; the user-provided implementation plan and `PRD.md` are therefore the active contract evidence for this run. Its P0 targets explicitly name the seven product domains and the delegation/handoff/document boundaries now being implemented.
- Blocker: No named swarm-manager goal was available to perform the normal bidirectional goal-to-PRD comparison; proceed under the user-provided plan and preserve this finding for the next run.
- Measured: 2026-08-19
