# Performance

## Scan Performance

Light and tidiness scans should complete within the PRD targets for typical scenarios or return clear timeout/degradation information.

## Expensive Work

Smart scans and AI-backed recommendations must use explicit file lists, batching, campaign limits, and force-rescan controls to avoid repeated expensive analysis.

## UI Performance

The UI should stay responsive while scans or campaigns are pending. Large tables should remain filterable and readable.

## Operational Validation

Use Test Genie performance and Lighthouse phases for release-level checks. Use API and UI focused tests for local regression checks.
