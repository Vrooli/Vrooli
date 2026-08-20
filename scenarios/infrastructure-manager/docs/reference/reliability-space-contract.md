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
| Owner | vrooli-autoheal |
| Leg unit | check |
| Denominator confidence | PARTIAL |
| Confidence rationale | Registry is complete for configured checks; host surfaces are not swept. |
| Legend | NOW · IN-REACH · MISSING |

## Coverage Grid

| ID | Cell | Owner | Unit | Status | Sensor | Gap opened on |
|---|---|---|---|---|---|---|
| SUP-001 | Registry coverage | vrooli-autoheal | check | NOW | checks list | |
| SUP-002 | Shelf hygiene | vrooli-autoheal | check | MISSING | | 2026-08-20 |
```

The metadata table is mandatory. `Projection`, `Owner`, `Leg unit`,
`Denominator confidence`, and `Confidence rationale` must each be present.
The grid must contain a stable cell ID, a question, a leg unit, and one of the
three status values. A `MISSING` cell must carry `gap_opened_on`; it is never
silently inferred from a failed read.

`AUTHORITATIVE`, `PARTIAL`, and `SKETCH` describe the completeness of the
denominator, not the health of any reading. Health and trust are separate
condition-axis values.

## Typed response

The equivalent CLI command is:

```text
<owner> space --projection <projection> --json
```

Its JSON shape is the proto-JSON encoding of the following logical object:

```json
{
  "projection": "supervision",
  "owner": "vrooli-autoheal",
  "legUnit": "check",
  "confidence": {
    "level": "PARTIAL",
    "rationale": "Registry is complete for configured checks; host surfaces are not swept."
  },
  "cells": [
    {
      "id": "SUP-001",
      "question": "Registry coverage",
      "owner": "vrooli-autoheal",
      "legUnit": "check",
      "status": "NOW",
      "sensorRef": "checks list"
    },
    {
      "id": "SUP-002",
      "question": "Shelf hygiene",
      "owner": "vrooli-autoheal",
      "legUnit": "check",
      "status": "MISSING",
      "gapOpenedOn": "2026-08-20"
    }
  ]
}
```

The service may return `available=false` and a verbatim `unavailableReason`
when its source cannot be read. Consumers must preserve authored cell status
in that case: an unavailable space is not an empty space and never converts
cells to `MISSING`.

## Projection vocabulary

The ten projection identifiers are `supervision`, `availability`, `recovery`,
`capacity`, `headroom`, `durability`, `attribution`, `validation-cost`,
`agent-throughput`, and `commissioning`. The first eight are owner-authored by
the control layers named in the plan; `capacity` and `commissioning` are
temporarily instrument-held control-plane spaces with a dated revisit trigger.

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
