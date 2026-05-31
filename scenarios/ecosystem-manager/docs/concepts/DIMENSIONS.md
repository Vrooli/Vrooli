# Dimensions — the controller's vocabulary

This document is the prose half of the **improvement-dimension vocabulary**.
The machine-readable single source of truth is
[`../../api/pkg/dimensions/dimensions.json`](../../api/pkg/dimensions/dimensions.json);
this file explains *why* the vocabulary exists and how the two mapping tables
are derived. If the two ever disagree, the JSON wins and this prose is stale.

## Why a shared vocabulary

[`CONTROL-MODEL.md`](CONTROL-MODEL.md) reframes Ecosystem Manager as a
closed-loop controller whose **state is the open `test-genie` findings set,
bucketed by dimension and severity**, and whose **selection** picks the skill
that targets the heaviest open dimension. That only works if *both* sides —
test-genie findings and skill declarations — speak the same vocabulary. The
dimension list is that vocabulary. Without it the "skill → dimension map" (the
linchpin of selection) is meaningless.

## The dimensions

The canonical list lives in `dimensions.json` (`dimensions[].id`). Each entry
carries a one-line description. Dimensions a skill can target but test-genie
does **not** produce findings for in v0 (`accessibility`, `coverage`,
`security`, `operational-targets`) are still first-class vocabulary members:
they are simply never the heaviest open cluster until a producer exists, and
`operational-targets` is measured through gap metrics rather than findings.

## Two mapping tables, two signal channels

test-genie surfaces two distinct signals, and the controller ingests both:

1. **Structured findings** — every `architecture.v1.ArchitectureFinding`
   carries a `Source` enum (one of `STRUCTURE, CLI, UI, DOCS, STANDARDS,
   ARCHITECTURE, TIDINESS`). `testgenie_source_map` keys those proto enum
   **names** to a dimension. `FINDING_SOURCE_UNSPECIFIED` is intentionally
   unmapped.
2. **Phase pass/fail** — phases that emit no structured findings (e.g. `unit`,
   `integration`, `lint`, `performance`) still report a status.
   `testgenie_phase_map` keys each test-genie catalog phase name to a
   dimension so a failing-but-findingless phase contributes a synthetic finding
   in its dimension.

A finding's dimension is therefore resolved by `Source` when one is present and
falls back to the producing phase otherwise. Both `ForSource` and `ForPhase`
return exactly one dimension by map construction.

### Phase → dimension (v0)

| test-genie phase | dimension |
|---|---|
| `structure` | `structure` |
| `contracts` | `contracts` |
| `ui-health` | `ui` |
| `standards`, `lint` | `standards` |
| `architecture` | `cycles` |
| `dependencies` | `dependencies` |
| `docs` | `docs` |
| `smoke`, `playbooks` | `visual` |
| `unit`, `integration` | `tests` |
| `business` | `business` |
| `performance` | `performance` |

### Source → dimension (v0)

| `FindingSource` | dimension |
|---|---|
| `STRUCTURE` | `structure` |
| `CLI` | `contracts` |
| `UI` | `ui` |
| `DOCS` | `docs` |
| `STANDARDS` | `standards` |
| `ARCHITECTURE` | `cycles` |
| `TIDINESS` | `tidiness` |

## Anti-drift

`dimensions_test.go` enforces the contract:

- every `FindingSource` (except `UNSPECIFIED`) maps to a valid dimension — adding
  or renaming a source in test-genie's proto fails the build until the SSOT is
  updated;
- a captured audit fixture (`testdata/testgenie_audit_fixture.json`, a recorded
  copy of test-genie's `--json` output) exercises every source, and every
  `plannedPhases` entry maps with no stale phase mappings left behind.

When test-genie adds a phase, re-capture the fixture; the phase-map test then
fails until `dimensions.json` is extended.
