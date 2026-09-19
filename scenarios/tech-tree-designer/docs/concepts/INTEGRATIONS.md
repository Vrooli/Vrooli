# Integrations — Tech Tree Designer

## Purpose Of This Document

Assign ownership before implementing adapters. Listed target integrations are not claims of installed or qualified dependencies.

## Scenario Dependencies

| Owner | Contract and boundary | State |
|---|---|---|
| Scenario Dependency Analyzer | Supplies observed interface graph; owns scanning and dependency governance | Existing GraphSource implementation |
| Tech Tree Designer | Owns intended ontology, proposal identities/revisions and design review | Ontology/proto planning exist; general proposals targeted |
| Plan Manager | Owns plans and immutable proposal references; not copies of every draft artifact | General proposal integration requires qualification |
| Swarm Manager | Owns approval, development mandate, budget and acceptance | Consume authoritative grants/decisions; do not duplicate |
| Agent Manager | Owns execution and supported harness goal/fallback behavior | TTD exposes design targets, not another agent loop |
| Workspace Sandbox | Qualified workspace/diff/apply operations | Discover actual capabilities and qualify before reuse |
| Artifact owners | PRD generation, requirements validation, experience validation, proto generation, skill/program publication | Dispatch by artifact kind; no bypass writes to generated/live catalogs |
| Test Genie and evidence owners | Own validation runs and durable evidence | Link exact revisions; unavailable evidence is not a pass |
| Control plane | Owns runtime identity, ports, databases and effect isolation | No scenario-private host repair or runtime launcher |

SDA failure must be explicit. Retain usable cached/planned context with provenance and staleness; do not report unavailable data as a complete empty ecosystem. An unavailable apply owner leaves an actionable held entry rather than silently using direct filesystem writes.

## Resource Dependencies

Embedded SQLite is the current metadata store. Do not add Postgres, Redis, Qdrant or model resources merely because ecosystem scale is large. New dependencies flow through SDA governance and require measured need.

## Future Integrations

First qualify general artifact owner capabilities, immutable retention, revision references, conflict handling, receipts and recovery. Optional model-assisted strategic analysis follows deterministic workflows and retains provenance and human disposition. Runtime experiments require explicit scope and qualified isolation, not a general-purpose IDE.

## Cross-References

- [Shared development method](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md)
- [Architecture](ARCHITECTURE.md)
- [Security](../internal/SECURITY.md)
- [Known qualification gaps](../internal/PROBLEMS.md)
