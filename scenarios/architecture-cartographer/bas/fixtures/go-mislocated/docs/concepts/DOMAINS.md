# Domains - go-mislocated fixture

Synthetic fixture used by the conflicts integration test. A symbol with
wrong-domain vocabulary intentionally lives under the right-domain path so the
glossary drift detector has a symbol-backed case to classify.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| right | Owns the correctly placed implementation path. | Provide the intended home for right-domain code. | none | service | validation | RightThing | `api/internal/right/` |
| wrong | Owns the vocabulary that should not appear in right-domain code. | Provide foreign glossary terms for drift detection. | none | service | validation | WrongThing | `api/internal/wrong/` |

## Non-Domains

(none)
