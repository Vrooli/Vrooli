# Domains — go-cycles fixture

Synthetic fixture used by the conflicts integration test. The two domains
form a deliberate cross-domain import cycle so the cycle detector has a
known case to classify.

## Domain Inventory

| Domain | Primary Archetype | Source Paths |
|---|---|---|
| alpha | service | `pkg/alpha/` |
| beta | service | `pkg/beta/` |

## Non-Domains

(none)
