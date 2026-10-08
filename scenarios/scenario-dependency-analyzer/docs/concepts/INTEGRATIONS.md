# Integrations

## Purpose Of This Document

Document resources, scenario dependencies, and integration contracts.

## Dependency Inventory

SDA depends on SQLite, Ollama, Qdrant, `proto-health`, `code-facts`, and Security Health's dependency-index evidence.

## Vrooli Resources

SQLite stores local metadata. Ollama and Qdrant support semantic analysis and optimization surfaces.

## Scenario Dependencies

`proto-health` provides batch proto surface facts. `code-facts` provides fleet import facts. Security Health owns vulnerability scanning and dependency-index evidence. SDA dependency health reads Security Health index status only; SDA governance commands consume vulnerability evidence for approval, denial, and remediation decisions. `test-genie` consumes SDA dependency health during dependency validation and consumes Security Health separately during the security phase.

## Third-Party Services

No external SaaS dependency is required for core local operation.

## Failure Modes

If fact services are unavailable, actual graph responses include upstream errors and drift may be incomplete. If Security Health is unavailable, dependency health reports a degraded `security-index` context but does not fail dependency readiness or run vulnerability scanners.

## Cross-References

- `../reference/configuration.md`
- `ARCHITECTURE.md`
- `../internal/SEAMS.md`
