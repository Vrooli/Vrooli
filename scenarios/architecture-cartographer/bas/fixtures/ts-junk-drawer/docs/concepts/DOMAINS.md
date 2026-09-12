# Domains - ts-junk-drawer fixture

Synthetic fixture used by the conflicts integration test. Shared substrate
imports product-domain code so the manifest-backed zone classifier and layering
detector classify a junk-drawer dependency.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| billing | Owns product billing code. | Provide a product domain that substrate must not import. | none | service | validation | Invoice | `api/internal/billing/` |

## Non-Domains

- `api/internal/utils/` - shared helpers and substrate.
