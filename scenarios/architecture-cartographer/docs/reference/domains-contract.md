# DOMAINS.md Machine Contract

The architecture cartographer **derives** a scenario's intended domain map
from sources that already exist on disk — there is no per-scenario
architecture manifest. The richest of those sources is the structured
**Domain Inventory** table in [`docs/concepts/DOMAINS.md`](../concepts/DOMAINS.md).
This document defines the strict, parseable shape the `domains` domain's
`DomainsDocExtractor` reads.

## Purpose

`docs/concepts/DOMAINS.md` is dual-purpose: it is the canonical
human-readable domain map **and** a machine contract. The extractor reads
the Domain Inventory table and the Non-Domains section; everything else in
the document is prose for humans and is ignored by the parser. The contract
is deliberately permissive about extra columns and surrounding prose so the
document stays readable.

## The extraction ladder

The derived domain map is resolved from a trust ladder. The highest
available rung defines the *expected* domain set; lower rungs contribute
provenance and (in the `conflicts` domain) cross-surface convergence
findings. Default trust order, highest first:

1. **API manifest** — reserved for a future `APIManifestExtractor` that
   ships with api-health. No extractor emits this rung yet.
2. **`docs/concepts/DOMAINS.md`** — the structured Domain Inventory (this
   contract).
3. **`api/internal/<domain>/` folders** — de-facto domains.
4. **`cli/manifest.json` groups** — command groups.

UI feature folders (`ui/src/features/<feature>/`) are advisory coverage
only and never define the authoritative set.

A scenario that ships no DOMAINS.md still derives a domain map from its
folders — the contract below is what *upgrades* that derivation from
"folder names" to "named domains with archetypes, owned paths, and
vocabulary".

## Domain Inventory table

The parser is **header-driven**: it locates columns by header text, not by
position, so human-facing columns may be added or reordered freely. Header
matching is case-insensitive.

Required columns:

| Header | Meaning | Rule |
|---|---|---|
| `Domain` | Domain name | Required, non-empty, unique per row. |
| `Source Paths` (prefix match, e.g. `Source Paths (planned)`) | Repo-relative path globs/prefixes the domain owns | Required, at least one. Comma-separated; each token may be wrapped in backticks. Accepts directory prefixes (`api/internal/graph/`), recursive globs (`api/internal/graph/**`), single-level globs (`.../*`), and exact files (`api/main.go`). |

Recommended columns:

| Header | Meaning | Rule |
|---|---|---|
| `Responsibility` | Semantic anchor for the capability | Required by the template contract; one concise sentence. |
| `Purpose` | Why the capability exists | Optional human-authored context. |
| `Owns Data` | Data ownership claim | Optional; use explicit `None` when the domain is stateless. |
| `Primary Archetype` (or `Archetype`) | Declared archetype | Required by the template contract. The canonical archetype vocabulary is the fixed fleet enum — `reporting`, `service`, `mutation`, `classification`, `orchestration`, `scoring`, `query` (proto `architecture-cartographer.v1.domains.Archetype`); declared labels are normalized onto it and a non-canonical label is preserved verbatim and reported as drift rather than silently coerced. Compound cells such as `service / orchestration` are preserved as multiple declared archetypes rather than truncating to the first token. (Distinct from the zone-layering *coordinating roles* vocabulary, which is a deliberate superset used only to decide which packages may coordinate siblings.) |
| `Secondary Traits` | Additional declared archetype traits | Optional list normalized onto the same canonical archetype vocabulary. |
| `Glossary` | Canonical vocabulary for the symbol-glossary signal | Optional. Comma-separated type/function names; backticks tolerated. Empty cell = no glossary. |

A row whose `Domain` cell is empty is skipped. A present row with no source
paths is a hard parse error — fix the table rather than leaving the cell
blank.

## Non-Domains section

The `## Non-Domains` section lists cross-cutting infrastructure that must
never become a product domain. The parser reads the backtick-wrapped path
from each bulleted line:

```
- `api/internal/server/` — HTTP composition substrate.
- `api/main.go` — composition root.
```

These paths become the derived map's **shared substrate** (paths owned by
no product domain), and each path's last segment becomes a non-domain name.

## What the contract deliberately does NOT express

- **Allowed dependencies / forbidden edges.** Declaring "A may import B" is
  high-maintenance and rarely enforced. Architectural health is recovered
  instead from the import graph via cycle detection and coupling heuristics
  (instability, fan-out, stable-kernel). See
  [`configuration.md`](configuration.md).
- **Per-scenario thresholds or signal weights.** Those are
  cartographer-global control-surface levers, not scenario declarations.
- **Transitional / suppression state.** Durable sanctioned deviations live
  as in-repo `// arch:allow` markers next to the code they excuse, not in
  this document.

## Cross-references

- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — the canonical domain map this contract governs
- [`../../../../docs/reference/intent-alignment.md`](../../../../docs/reference/intent-alignment.md) — the PRD ↔ requirements ↔ domains ↔ code doctrine that consumes the derived domain map for vertical intent checks
- [`configuration.md`](configuration.md) — cartographer-global tunable levers (ladder trust order, heuristic thresholds)
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — the `DomainSourceExtractor` seam registry
