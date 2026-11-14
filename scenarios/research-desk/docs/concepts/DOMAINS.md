# Domains — Research Desk

## Purpose Of This Document

The domain map. RPCs and CLI verbs per domain are listed once, in [`SYSTEM.md`](SYSTEM.md)
§2; the rules each domain applies are in the sections named below.

## Domain Inventory

| Domain | Responsibility | Primary Archetype | Secondary Traits | Source Paths |
|---|---|---|---|---|
| activity | Read-only projection: commissions, states, topics, changes since last look (SYSTEM §3, §4, §8, §9) | query | aggregation | api/handlers/activityapi (over api/internal/home), cli/domains/activity |
| launch | Preflight, packet build through `packet.py`, launch and control of Web Console sessions (SYSTEM §5, §6) | orchestration | validation | api/handlers/launchapi, api/internal/launch (sessions through api/internal/launch/launcher.go), cli/domains/launch |
| operator | `OPERATOR.md` entries, needs-you courier, reviews and verdicts (SYSTEM §7, §9; courier rules FILE-CONTRACT §5) | mutation | orchestration | api/handlers/operatorapi, api/internal/courier, api/internal/writer, cli/domains/operator |

Not a domain: `api/internal/home`, the one reader every domain uses, and
`api/internal/writer`, the one writer (SYSTEM §1). Out of
scope: scheduling owners and writing files the contract gives to others (SYSTEM end).

## Cross-References

- [`SYSTEM.md`](SYSTEM.md)
- [`../reference/FILE-CONTRACT.md`](../reference/FILE-CONTRACT.md)
