# Progress — Chain Forensics

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This log begins with the generated-scenario foundation; it records verified
milestones and deliberately leaves the full CF6 workstream open.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-11-09 | CF6 owner | partial | Qualified React-Vite through Template Manager and generated the scenario; authored CF6 PRD, requirements, domain/data/flow/provider/privacy docs, and case-workspace experience contract. Added an importable Go evidence package with exact Bitcoin normalization/outpoint joins, bounded rate-aware mempool.space reads, strict provenance checks, a package CLI, versioned private-packet serialization, and hash-verified offline 1HQ3 sources. Package tests pass; inspect and packet commands ran, and the packet was verified mode 0600 without overwrite. Experience and requirements validation pass; orientation is 6/9. Full scenario run 20261109-214830-675f059d passed 24/27; security findings remain. Scenario Connect/API, registered CLI, case UI, and CI2 import remain open. |
| 2026-11-09 | CF6 owner | partial | Follow-up full run 20261109-221404-04608913 found docs, storage, and security failures. Fixed the domain inventory contract and declared the generated deployment report plus storage-enforcement receipt as bounded, regenerable owner surfaces. Corrected broken marked-reference paths and stale starter CLI examples; Test Genie run 20261109-222747-04044ca7 verified storage PASS, and run 20261109-223037-4fdb1a03 verified docs PASS at L1 Readable. Security remains unresolved; no full suite rerun. Scenario Connect/API, registered CLI, case UI, CI2 import, and the other two showcase cases remain open. |
| 2026-11-14 | CF6 owner | partial | Added a version-pinned Pump `create_v2` discriminator/Borsh decoder to the existing bounded finalized Solana transaction reader. The real slot 451779205 RPC response yields one decoded launch instruction while a second unknown Pump discriminator stays unresolved. Exact response and pinned official IDL bytes plus hashes are retained under `packages/chain-forensics/evidence/cf6-pump-createv2-20261114`; package tests pass. Extended case/proto storage, Connect, SQLite merge, private packet, CLI and UI to preserve these source-linked records. A private Pump case was imported through the running API from the retained exact RPC response and reads back with the raw-envelope hash, one decoded instruction, and one unresolved discriminator. Parent's private Wormhole pair still reads with its protocol edge after a normal lifecycle restart. Focused API/CLI/UI tests and UI type-check pass. The published IDL is not proof of deployed binary layout. Bridges, exchange clusters, three full showcases, a real browser render, and real CI2 integration remain open. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
