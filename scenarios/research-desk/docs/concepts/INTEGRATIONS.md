# Integrations — Research Desk

## Purpose Of This Document

Pointer. Every dependency is optional and is an entry in the capability registry; what
each one is used for and what the desk shows when it is down are in
[`SYSTEM.md`](SYSTEM.md) §1, §5–§7 and §11. No resource is needed. `.vrooli/service.json`
declares web-console, vrooli-events and notification-hub as scenario dependencies with
`required: false` and `startup_policy: ignore`, so the desk starts and reads files without
them; the reasons are in [`../DECISIONS.md`](../DECISIONS.md) D6, D9, D20 and D21.

## Cross-References

- [`SYSTEM.md`](SYSTEM.md)
- [`../DECISIONS.md`](../DECISIONS.md)
