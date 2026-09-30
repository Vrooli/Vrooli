# Monetization PoR Changelog

## Changelog

| Date | Change |
|---|---|
| 2026-05-10 | Migrated monetization plan-of-record into the shared PoR manifest shape: `operating/`, `taxonomies/`, `catalogs/`, `strategy/`, `evidence/`, and `governance/`. |
| 2026-08-17 | Repaired workflow drift after the Offer Desk cutover: lifecycle meaning and triggers now live on instrument records, durable opportunity context uses the `team:monetization` Source Ledger, and financial posture is read from Money Ledger rather than a parallel document or heartbeat snapshot. |
