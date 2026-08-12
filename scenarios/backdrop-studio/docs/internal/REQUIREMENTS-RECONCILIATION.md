# Requirements reconciliation — 2026-08-11

The generated requirement modules remain the canonical IDs. This ledger records
the implementation state after the end-to-end treatment work so planned entries
are not silently mistaken for delivered behavior.

Implemented: `CAT-001..007`, `SCAF-001..004`, `CMP-001..008`, `RND-001..006`,
`LEG-001..006`, `REL-001`, `REL-003..005`, `UIX-001..007`, `SUR-001..004`.

Implemented as explicit capability seams: `REL-002` (model-backed release
requires and records an Asset Studio publisher reference) and `UIX-008..009`
(product/store chrome selection is represented in the workbench and is tested
as a surface-kind decision).

Unbuilt and intentionally not marked implemented:

- `REL-006`: automatic sized-variant derivation preserving regions is not in
  this plan's shipping path; surface geometry and device composition are
  implemented, but variant generation remains a separate capability.
- The treatment taxonomy's Tier-3 values (`glitch`, `kaleidoscope`,
  `slit_scan`, `fluted_glass`, `photomosaic`, `resample`) remain explicitly
  unbuilt as required by the plan.

The upstream Asset Studio publisher is an injectable contract because the
current Asset Studio API exposes render/release metadata operations but no
backdrop byte-ingress operation. A model-backed release therefore fails closed
until that capability is configured; procedural releases remain independent.
