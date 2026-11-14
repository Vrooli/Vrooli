# Flows

## Scan Flow

1. A user or agent calls the CLI or API.
2. The request identifies a scenario or path and scan type.
3. The API validates the target and applies timeout/resource limits.
4. Scanners collect metrics and optional tool output.
5. Findings and metrics are returned and, where appropriate, persisted.

## Tidiness Phase Flow

Test Genie calls the Tidiness Manager tidiness scan endpoint. The response is mapped into Test Genie tidiness findings. Static-quality findings are intentionally excluded.

## Smart Scan Flow

Smart scans require explicit file input. They may use AI resources, campaign IDs, and force-rescan flags. Results can create issue candidates and campaign context.

## Campaign Flow

Campaigns start in an active state, run scan sessions under configured limits, and transition through pause, resume, completed, terminated, or error states. Visited-tracker can provide file visit context and handoff notes.

## Issue Resolution Flow

Issues are listed through filters, then resolved, ignored, or reopened. Resolution notes should explain why the issue is done or intentionally deferred.

## Failure Flow

Scanner failures should preserve partial context, return sanitized errors, and record enough server-side detail for debugging. Optional tool failures should degrade findings rather than stopping basic maintainability scans.
