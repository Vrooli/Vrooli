# Infra Health Evidence

> **Retired 2026-08-20.** Both documents this folder held were hand-maintained
> ancestors of a computed surface. Both were deleted on 2026-08-20, once the links
> that pointed at them were repointed at the surfaces that compute the same thing.

This folder is **empty**.

- `CROSS_PLATFORM_LEDGER.md` — superseded by the `portability` domain of
  `scenarios/infrastructure-manager`, read with `vrooli capability ledger` and
  `vrooli capability fleet`, which delegate to it. It held zero entries for
  nearly four months while the computed surface answered the same question.
- `INSTRUMENTATION_ROADMAP.md` — superseded by the computed open-loop set
  (`infrastructure-manager coverage open-loop`), every cell of which carries the
  date its gap opened and how long it has been open.

Retiring the folder itself is the remaining step, tracked in the plan of record's
Future-PoR-work list.

Do not add documents to this folder. Live alerts, raw logs, and immediate incident
response stay outside this PoR folder, and anything that can be computed belongs to
the surface that computes it.

## Instrumentation gaps are computed, not listed

The open-loop set — every dimension with no sensor at all — is derived by
`infrastructure-manager` from owner-authored `MISSING` cells, each carrying the date its
gap opened and how long it has been open:

```bash
infrastructure-manager coverage open-loop --json
```

A gap is opened by marking a cell `MISSING` in the owning layer's
`docs/spaces/<projection>-space.md`, and closed by shipping its sensor and moving the
cell to `NOW`. It is never opened by adding a row here: a hand-maintained list can
disagree with the grid beside it, and this one did.
