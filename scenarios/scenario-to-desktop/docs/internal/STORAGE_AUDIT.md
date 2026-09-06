# Scenario to Desktop storage audit

Updated 2026-09-05. Scope: packaging staging retention and canonical path routing.

## Ownership and enforcement

The pipeline domain owns staging eligibility; api-core owns filesystem measurement, oldest-first age/capacity selection and contained deletion. `.vrooli/service.json` declares the custom 20GiB/7d staging budget. The host root is `safe_with_owner`; generic retention defers custom entries. No SQL schema or migration changes were required.

Generation and pipeline staging resolve through `storagepaths.Locator`. Production locator construction uses `storage.ScenarioNamespace`; generation accepts an injected resolver to keep test files in temporary storage. Active pipelines, generation/build operations, smoke tests and live desktop sessions are protected. The owner checks eligibility again before removing complete build directories at depth two. Latest-build protection and unknown-build grace can leave storage above budget and are reported as incomplete.

## Evidence and limits

Focused regression tests cover capacity overflow before expiry, active-work protection, grouped build deletion, unknown-build grace, unavailable owner state, manifest defaults, positive overrides and storage-resolution failure. See `api/pipeline/staging_retention_test.go` and `api/generation/service_test.go`. Full storage-validator and test-genie results are recorded in PROBLEMS.md when terminal.

This scoped audit does not certify unrelated storage domains. Staging limits are periodic retention targets, not synchronous admission quotas; concurrent writers can temporarily exceed them. Configuration and operator details are in `docs/OVERVIEW.md#packaging-staging-retention`.
