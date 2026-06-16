# Domains

## Purpose Of This Document

This document names Tidiness Manager's bounded contexts and points to the implementation surfaces that own them.

## Domain Inventory

| Domain | Purpose | Source Paths |
| --- | --- | --- |
| Scan orchestration | Light, tidiness, and smart scan request flow | `api/handlers.go`, `api/light_scanner.go`, `api/smart_scanner.go` |
| Metrics and detectors | File metrics, complexity, duplication, debt markers | `api/code_metrics.go`, `api/complexity_analyzer.go`, `api/duplication_detector.go` |
| Issues | Stored findings, status changes, filters | `api/agent_handlers.go`, `api/issue_generator.go`, `api/issue_store.go`, `api/persistence.go` |
| Campaigns | Cleanup campaign lifecycle and visited-tracker handoff | `api/auto_campaigns.go`, `api/campaign_manager.go`, `api/handlers_campaigns.go` |
| CLI | Agent-facing commands | `cli/domains/`, `cli/internal/` |
| UI | Human dashboard and issue workflows | `ui/src/` |

## Domain Details

Scan orchestration converts scenario names or paths into bounded scan work and returns normalized results. Metrics and detectors produce the raw maintainability signals. Issues turn those signals into trackable findings. Campaigns coordinate repeated cleanup sessions. CLI and UI expose the same product concepts for different users.

## Shared Concepts

Shared concepts include scenario identity, file path, severity, issue category, scan type, campaign state, and visit state.

## Deferred Domains

P2 integrations such as app-issue-tracker task creation, code-smell handoff, custom rules, trend reporting, and remediation automation are deferred product domains.

## Non-Domains

Lint policy, type policy, static-quality config enforcement, and standards auditing are not Tidiness Manager domains.

## Cross-References

- `ARCHITECTURE.md`
- `FLOWS.md`
- `DATA.md`
- `../internal/SEAMS.md`
