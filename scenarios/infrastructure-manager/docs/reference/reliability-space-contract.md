# Reliability Space Contract

This contract is the shared seam between the reliability instrument and the
control layers that own its denominator. Owners define which cells exist;
`infrastructure-manager` reads those definitions and joins them to the
operator-owned setpoint. The verb is read-only and must never mutate an
owner's registry, measurement, or policy.

## Space document

Every owner keeps one document at `docs/spaces/<projection>-space.md` with the
following shape:

```markdown
## This Space

| | |
|---|---|
| Projection | supervision |
| Owner | `vrooli-autoheal` |
| Denominator confidence | `PARTIAL` — registry is complete for configured checks; host surfaces are not swept |
| Leg unit | check |

## Coverage Grid

| ID | Cell | Owner | Unit | Status | Sensor | Gap opened on |
|---|---|---|---|---|---|---|
| SUP-001 | Registry coverage | vrooli-autoheal | check | NOW | checks list | |
| SUP-002 | Shelf hygiene | vrooli-autoheal | check | MISSING | | 2026-08-20 |
```

The metadata table is mandatory. `Projection`, `Owner`, `Denominator
confidence` and `Leg unit` must each be present, and those are the only four
rows the parser reads — a fifth row is documentation, not data.

Two shapes are load-bearing and are easy to get wrong:

- **`Owner` is read as its first whitespace-delimited token.** `control plane
  (`vrooli capacity`)` parses to the owner `control`. Write a single token —
  `vrooli-autoheal`, `storage-manager`, `control-plane` — and put any
  qualification after it in the same cell, where it stays readable to a human
  and is dropped from the typed response.
- **Level and rationale share the `Denominator confidence` cell.** The parser
  takes the level from the first recognised keyword and the rationale from the
  whole cell, so the convention is `` `PARTIAL` — <why> ``. There is no separate
  rationale row.

The grid must contain a stable cell ID, a question, and one of the three status
values. A `MISSING` cell must carry `gap_opened_on`; it is never silently
inferred from a failed read. An unstated confidence defaults to `SKETCH`,
because the least confident reading is the honest one to assume.

`AUTHORITATIVE`, `PARTIAL`, and `SKETCH` describe the completeness of the
denominator, not the health of any reading. Health and trust are separate
condition-axis values.

## Typed response

The equivalent CLI command is:

```text
<owner> space --projection <projection> --json
```

It emits `space-definition/v1` — the `spacedoc.SpaceDefinition` encoding, whose
schema is `.vrooli/schemas/space-definition.schema.json`. Keys are snake_case,
status and confidence values are lower-cased on the wire, and `source` carries
the repo-relative document the denominator was read from so a reader can always
find the authority behind a number:

```json
{
  "schema_version": "v1",
  "projection": "supervision",
  "owner": "vrooli-autoheal",
  "denominator_confidence": "partial",
  "confidence_rationale": "`PARTIAL` — registry is complete for configured checks; host surfaces are not swept",
  "leg_unit": "check",
  "source": "scenarios/vrooli-autoheal/docs/spaces/supervision-space.md",
  "cells": [
    {
      "id": "SUP-001",
      "question": "Registry coverage",
      "owner": "vrooli-autoheal",
      "status": "now",
      "notes": ["checks list"]
    },
    {
      "id": "SUP-002",
      "question": "Shelf hygiene",
      "owner": "vrooli-autoheal",
      "status": "missing",
      "gap_opened_on": "2026-08-20"
    }
  ]
}
```

Omitting `--json` renders a counted text summary instead; it is for humans and
carries no cell detail.

A source that cannot be read fails loudly rather than emitting a partial
document. Consumers must preserve authored cell status when an owner is
unreachable: an unavailable space is not an empty space and never converts
cells to `MISSING`.

## Projection vocabulary

The eleven projection identifiers are `supervision`, `availability`,
`recovery`, `substrate`, `capacity`, `headroom`, `durability`, `attribution`,
`validation-cost`, `agent-throughput`, and `commissioning`. Nine are
owner-authored by the control layers named in
[`../concepts/COVERAGE-MODEL.md`](../concepts/COVERAGE-MODEL.md) § The
Projections. `capacity` and `commissioning` are held by
`infrastructure-manager` itself, because the control plane is not a scenario
and has nowhere to put a space document; both carry a dated revisit trigger and
are the only two projections this scenario's own `space` verb will serve.

Registration is per owner, in that scenario's `cli/domains/domains.go`, via
`spacecli.CommandGroup`. An owner that authors a space document but does not
register its projection publishes a denominator no typed caller can read — the
document is then reachable only by file path, which is the coupling this
contract exists to remove.

## Conformance rules

- The verb is read-only. No `create`, `update`, `delete`, `reconcile-and-fix`,
  restart, shelve, or policy mutation is part of this contract.
- A caller may request one projection at a time. Unknown projections and
  malformed documents fail loudly.
- Cell status and denominator confidence are explicit fields; omitted values do
  not mean healthy or authoritative.
- Source availability is reported beside the space and never collapsed into a
  zero numerator.
- A new cell is a document change, not an `infrastructure-manager` code change.
# Storage headroom reader

The reliability instrument reads the typed Storage Manager feed through the
storage source adapter. The feed is observational: it contains device census,
growth, declared ceilings, recovery efficacy, budget truth, and hot-writer
readings. The instrument assigns a trust verdict to each reading and does not
recompute deletion decisions.

The storage projection maps these cells:

- H1: device census and available bytes.
- H2: measured growth slope.
- H3: declared storage ceilings.
- H4: recovery efficacy and recent terminal runs.
- H5: budget truth and exceeded budgets.
- H6: hot writers and their observed rates.

An unavailable or stale storage feed is reported as unavailable, not as a
healthy zero. Operators can compare the instrument with
`storage-manager storage infra-health --json`.
