# Plan-of-Record Structure

Plan-of-record folders are stable canon. They should let an agent understand what a team or system owns, how work enters and leaves, which classification systems apply, and where durable extensions belong without reading runtime logs first.

This document defines the target structure. The machine-readable base contract lives in [`team-plan-of-record.manifest.json`](team-plan-of-record.manifest.json).

## Standard Shape

Use this shape for team-owned plan-of-record folders:

```text
docs/<team>/
  README.md
  manifest.json

  operating/
    OPERATING_MODEL.md

  interfaces/
    inputs.md
    outputs.md
    consumers.md

  taxonomies/
    <taxonomy-id>/
      README.md
      taxonomy.json
      schemas/

  methods/
    <method-family>/
      README.md
      <method-slug>.md

  catalogs/
    <catalog-family>/
      README.md
      <entry-slug>.md

  strategy/
    README.md
    <strategy-doc>.md

  evidence/
    README.md
    <evidence-doc>.md

  governance/
    editing.md
    adoption-validation.md
    changelog.md

  extensions/
    <extension-id>/
      manifest.json
      README.md
```

Not every folder is required for every plan of record. `README.md`, `manifest.json`, `operating/OPERATING_MODEL.md`, and `governance/` are the default required core. Optional modules become required when declared in the local manifest.

## Folder Roles

| Folder | Role |
|---|---|
| `operating/` | Team contract: mission, scope, operating graph, topics, decisions, inputs, outputs, feedback loops, implementation gaps, and validation commands. |
| `interfaces/` | Stable descriptions of how signal enters, leaves, and is consumed downstream when that material outgrows the operating model. |
| `taxonomies/` | Classification and routing systems with a human-readable README and machine-readable `taxonomy.json`. |
| `methods/` | Reusable techniques, checks, audit lenses, routing methods, or review patterns. These often pair with skills. |
| `catalogs/` | Inventories of durable entities: post types, revenue lines, channels, products, scenario classes, registries. |
| `strategy/` | Directional canon: positioning, pricing, roadmaps, principles, policy, or prioritization logic. |
| `evidence/` | Benchmarks, research summaries, financial models, telemetry reviews, and other supporting evidence. |
| `governance/` | Editing authority, adoption and validation commands, changelog, and migration notes. |
| `extensions/` | Declared custom packages that do not honestly fit a standard folder. |

## Extension Rules

1. Put custom durable canon in the most specific standard folder that honestly fits.
2. Use `taxonomies/` only for classification or routing systems with machine-readable sidecars.
3. Use `methods/` for repeatable judgment or procedure patterns that agents apply.
4. Use `catalogs/` for entity inventories where one-entry-per-file keeps the canon maintainable.
5. Use `strategy/` for directional truth and `evidence/` for supporting proof.
6. Use `extensions/` only when the standard folders would misrepresent the intent. Every extension must declare a local `manifest.json`, `README.md`, owner, purpose, allowed entry types, and graduation rule.
7. Working notebooks are not authoritative plan-of-record canon. If a team keeps a notebook near its PoR, the local manifest must mark it as `working-notebook` and exclude it from PoR authority.

## Validation Expectations

The first validation slice checks the local manifest, required files, required headings, taxonomy packages, method registries, governance docs, and unregistered durable Markdown files. Deeper document-type validators still own semantic checks. For example, `operating/OPERATING_MODEL.md` is structurally required by the PoR manifest and semantically validated by the operating-model validator.

