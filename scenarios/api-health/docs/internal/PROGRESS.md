# Progress — API Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| 2026-07-03 | Codex | done | Generated API Health from `react-vite` with `vrooli-default`, removed the template notes example domain, authored the provider PRD, requirements registry, maturity spec, and core concept/reference docs as the foundation for implementation. |
| 2026-07-03 | Codex | done | Added static lifecycle validation for API targets: service health metadata, api-core preflight ordering/name checks, direct ListenAndServe detection, native lifecycle evidence, and unit fixtures for the Phase 2 provider contract. |
| 2026-07-03 | Codex | done | Added execution-mode live `/health` probing with Vrooli API port resolution, bounded one-shot HTTP checks, api-core health schema validation, native probe evidence, CLI `--include-execution`, and httptest coverage for healthy, degraded, unhealthy, timeout, non-JSON, malformed, unreachable, and readiness-inconsistent responses. |
| 2026-07-03 | Codex | done | Added HTTP response semantics validation for production API Go files: route classification/versioning, endpoint metadata rest-exception exemptions, raw status literal detection, implicit stdlib JSON error-success detection, content-type evidence, native HTTP details, and fixture coverage for low-ambiguity defects and false-positive exclusions. |
| 2026-07-03 | Codex | done | Added API runtime hygiene validation for production API Go files: unbounded/default HTTP client usage, unclosed outbound response bodies, dropped request contexts, uncancellable long-lived goroutines, unstructured operational logging, native runtime evidence, and focused fixture coverage for compliant and noncompliant runtime paths. |
| 2026-07-03 | Codex | done | Added Phase 6 deterministic fix registry and shared Fix RPC support for API Health: dry-run/apply candidates, CLI `fix-preview` and `fix-apply`, implemented maturity fixability declarations for mechanical fixes, manual reasons for design-bearing repairs, and unit coverage for preview no-write behavior, apply, idempotency, RPC mapping, and no-candidate messaging. |
| 2026-07-03 | Codex | done | Added Phase 7 scenario-auditor API migration accounting: a test-backed ledger for every production legacy API rule file, explicit redesigned/delegated decisions, reference documentation, and requirement evidence for APIH-MIG-001. |
| 2026-07-03 | Codex | done | Added Phase 8 operator evidence UI: a provider-backed validation workbench for target summary, capability ladder, findings, live probe evidence, and dry-run fix previews; added typed validation API decoding, route/nav/selector/i18n wiring, RouterProvider future flags, unique nav landmark labels, and UI tests. |
| 2026-07-03 | Codex | done | Added Phase 9 Test Genie cutover: registered a dedicated `api` phase backed by api-health through ScenarioValidationService, included it in curated presets, preserved the legacy `standards` phase for remaining non-API scenario-auditor rules, updated phase docs/schema, and added guard tests for routing and preset membership. |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map
