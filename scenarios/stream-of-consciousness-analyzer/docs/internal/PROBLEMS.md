# Known Problems & Risks

This is the scenario's current issue ledger. Historical scores, test counts,
and audit results are observations of earlier revisions, not current readiness.

## Confirmed implementation gaps

- **Suggestion generation is a stub.** `GenerateSuggestions` selects a provider
  and returns an empty slice; it does not call an LLM. Implement graph-to-prompt
  construction, provider invocation, and response parsing behind the existing
  service seam. [CODE: api/suggestion_service.go]
- **Database startup resilience.** The API exits when database connection or
  schema initialization fails. Retry or degraded startup remains a product
  decision. [CODE: api/main.go]

## Unresolved design and operational checks

- **Request rate limiting:** the prior ledger reported no API rate limiting.
  Recheck the request boundary before exposing it to high traffic.
- **Clock skew:** verify server-assigned timestamps govern last-write-wins sync;
  client timestamps should serve display, not conflict authority.
- **Ollama contention:** verify the proposed throttling (30-second interval,
  one pending generation) under concurrent scenario load. The mitigation was
  designed, but its load-test evidence remains unverified.
- **IndexedDB capacity:** research browser-specific limits for voice blobs;
  verify size accounting, approaching-limit warnings, and sync prioritization.
- **Production lifecycle:** confirm the deployment target's lifecycle declaration
  before deployment. Earlier notes reported a missing production phase.
- **Smoke-test readiness:** earlier 502s were attributed to smoke starting before
  API initialization. Wait for lifecycle health before validation; recheck any
  recurrence against the current startup contract.

## Historical validation claims requiring fresh evidence

The two previous ledgers disagreed: one recorded a score near 39, untracked
requirements, and three standards findings; the other later reported 96 and
zero findings. Neither snapshot proves current status. The earlier proposed
causes (API-client detection and Go requirement-annotation parsing) remain
historical hypotheses, not verified current defects.

Use Test Genie for a scoped current assessment:

```text
vrooli scenario test stream-of-consciousness-analyzer --phases docs
```

Select additional phases for the issue under investigation using the project
[testing guide](../../../../docs/TESTING.md). A passing docs phase does not
establish product readiness or requirement coverage.

## Historical resolutions

Earlier work recorded canvas error feedback, configurable provider polling,
an application error boundary, a connection-health indicator, accessible
controls, safer TypeScript assertions, and test/configuration repairs. Keep
these as regression expectations; archived test counts and accessibility
scores do not replace current runs.

## Preserved ledger sources

Both pre-merge ledgers are preserved under the protected runtime-home entry
`plan-artifacts/docs-cleanup-20260907-final-txumdx73/scenarios/stream-of-consciousness-analyzer/docs/`:
`PROBLEMS.md` and `internal/PROBLEMS.md`. Original hashes and recovery
instructions are indexed by the project [cleanup record](../../../../docs/internal/PROGRESS.md#documentation-cleanup-completion--2026-09-07).
