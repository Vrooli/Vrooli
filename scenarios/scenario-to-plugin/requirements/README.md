# Requirements Registry

This registry is the measurement layer for `PRD.md`. The PRD says what Scenario to Plugin
promises; these modules say how each promise will be proven. Modules are organized by
bounded context, not by priority tier, because the build order follows the domain chain.

## Registry contract

- **Operational target linkage.** Every requirement carries a `prd_ref` naming a real
  `OT-` line in `PRD.md`. A `prd_ref` that matches no operational target is a dangling
  claim; an operational target with no requirement pointing at it is an orphan. Both are
  validation errors, so the two files stay in step by construction.
- **Validation entries.** Every requirement carries at least one `validation[]` entry
  describing how the claim is proven. Code-typed refs (`test`, `automation`) are checked
  against the filesystem and must point at a file that exists. `manual` refs are not
  path-checked and describe an attended procedure instead.
- **Auto-sync.** Tests tag `[REQ:ID]`; test-genie's requirements sync updates each
  requirement's status from the run. Statuses are earned from evidence and are never
  hand-set — a requirement is `complete` only when its validations pass.

## Current state — pre-implementation

No code exists yet. Every requirement is `planned`, and every validation is a single
`manual` entry pointing at [`../docs/internal/TESTING.md`](../docs/internal/TESTING.md)
whose `notes` field names the automated layers that will replace it.

This is deliberate. Naming an `api/…_test.go` path before the file exists would be a
broken proof path, and the registry would assert coverage that nothing backs. As each
domain lands, replace that manual entry with the real code-typed refs — at least two
automated layers for every P0 and P1 requirement.

## Module map

| Module | Domain | Operational targets |
|---|---|---|
| `01-declaration/` | Plugin declaration and publish readiness | OT-P0-001, OT-P1-005 |
| `02-composition/` | Agent Plugins artifact composition | OT-P0-001, OT-P1-002, OT-P2-002 |
| `03-conformance/` | Skill spec, CLI drift, install safety | OT-P0-002, OT-P0-003, OT-P0-004, OT-P2-003 |
| `04-attestation/` | Scan, sign, provenance, SBOM | OT-P0-005 |
| `05-rehearsal/` | Clean-room install and command exercise | OT-P0-006, OT-P1-004 |
| `06-distribution/` | Release gate, channels, revocation | OT-P0-007, OT-P0-008, OT-P1-001, OT-P1-003, OT-P2-001, OT-P2-004 |

Modules are numbered in dependency order: composition reads the declaration, conformance
reads the composed package, attestation runs only after conformance passes, rehearsal
installs the attested artifact, and distribution publishes only what rehearsal proved.

## Contributor notes

- `delivery_scope: roadmap` marks a linked capability that is intentionally outside the
  current delivery commitment. It keeps the claim visible without diluting completeness.
  Do not use it to defer a P0.
- Write titles as EARS statements with RFC 2119 keywords matched to the tier: `shall` for
  P0, `should` for P1, `may` for P2. Prohibitions are unwanted-behaviour claims about an
  observable response, never bare absences.
- Never add compatibility shims — duplicate folders or alias imports — during a
  migration. Let things fail temporarily instead of adding debt.
- Schema details: `scenarios/test-genie/docs/reference/requirement-schema.md`.
  Sync behavior: `scenarios/test-genie/docs/phases/business/requirements-sync.md`.
