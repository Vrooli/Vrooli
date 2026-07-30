# Known Problems & Follow-up Tasks

This file tracks known issues and technical debt that need attention.

---

## Work ladder

## Landing configuration orchestration crosses sibling domains

**Severity:** Architecture blocker
**Status:** Resolved
**Resolved:** 2026-07-30

`internal/landing/LandingConfigService` now coordinates commerce, delivery,
and experimentation as the declared `landing` aggregation domain. Its
composition and Connect transport remain at the API root, and
`docs/concepts/DOMAINS.md` plus `ARCHITECTURE.md` document the permitted
sibling dependencies. Architecture Cartographer accepts this boundary with no
remaining `layering/domain-imports-sibling-domain` error.

---

- Rung: W3 / R2–R3 implementation hardening
- Evidence: active goal `landing-page-api-domain-subpackages` requires behavior-preserving domain extraction; the PRD's P0 monetization/download targets remain compatible, and `vrooli scenario requirements validate landing-page-business-suite` passed on 2026-07-28. The latest API suite is green, while Tidiness reports 27 long files and 315 duplication findings, including the 1,039-line Download Settings route.
- Blocker: none; continue domain-oriented API work and direct-UI decomposition without weakening requirements or validator rules.
- Measured: 2026-07-29

---

## Security Issues

### Resolved: persisted-domain measures coverage

**Status:** Resolved  
**Updated:** 2026-07-27

All num[sot]:26 persisted entities detected by Measures Health now have a typed,
time-windowed `MeasuresService` RPC and a shared registry aggregate over
authoritative Postgres state. The registry and Connect routes remain
admin-or-service protected. Measures Health's behavioral/indexing assessment
is clean; its central probe uses no authenticated request yet, so this access
boundary remains intentionally covered by route and handler tests.

### Resolved: reachable dependency and Go toolchain vulnerabilities

**Status:** Resolved
**Updated:** 2026-07-26

Reachability analysis found production paths through an outdated JWT parser and
AWS S3 EventStream decoder, plus Go 1.25.0 standard-library vulnerabilities.
The API now requires `github.com/golang-jwt/jwt/v5` 5.2.2, AWS S3 1.97.3, and
the scenario API and CLI modules require Go 1.25.12. The upgrades were applied
through Scenario Dependency Analyzer, then validated with the complete API and
CLI test suites and `GOWORK=off govulncheck ./...` (zero reachable
vulnerabilities).

### SQL-002: SQL Injection in delivery artifact listing

**Severity:** ~~Critical~~ False Positive
**Location:** `api/internal/delivery/service.go` (artifact-listing query construction)
**Status:** ✅ Analyzed - False Positive
**Reported:** 2026-01-16 (scenario-auditor)
**Analyzed:** 2026-01-16 by Claude (failure-topography)

The scenario auditor flagged a potential SQL injection via format string, but **this is a false positive**.

**Analysis:**
The code in question uses `fmt.Sprintf` to build a WHERE clause, but:
1. The `whereClause` only contains **static SQL structure** with positional parameter placeholders (`$1`, `$2`, `$3`, etc.)
2. All **user inputs** (`bundleKey`, `platform`, `query`) flow through the `args...` parameters
3. The `fmt.Sprintf` **only interpolates parameter position numbers**, not user data
4. The actual values are bound via parameterized query execution

**Example of what the code does:**
```go
// User input goes into args (safe - parameterized)
args := []interface{}{bundleKey}  // User input bound to $1

// whereClause contains ONLY static SQL with placeholders
whereClause := "bundle_key = $1"

// Query uses parameterized execution
db.QueryRowContext(ctx,
    fmt.Sprintf(`SELECT COUNT(*) FROM download_artifacts WHERE %s`, whereClause),
    args...)  // User data bound here, not interpolated
```

**Conclusion:** No remediation required. The code correctly uses parameterized queries. The auditor's pattern matching detected `fmt.Sprintf` in SQL context but didn't trace the data flow to confirm safety.

**Recommendation for scenario-auditor:** Consider enhancing the SQL injection detection to trace whether user input actually flows into the format string arguments vs being bound as query parameters.

---

## Failure Handling Gaps

### Remaining `alert()` Usage

**Severity:** Low
**Status:** ✅ Resolved
**Reported:** 2026-01-16
**Resolved:** 2026-01-16 by Claude (signal-and-feedback-surface-design)

The Customization page has been updated to use InlineAlert, and other admin pages now use Toast notifications:

- [x] VariantEditor.tsx - ✅ Uses Toast notifications for success/error feedback
- [x] SectionEditor.tsx - ✅ Replaced alert() with Toast notifications
- [x] Customization.tsx - ✅ Updated to use InlineAlert + Toast for success

---

## Stability Issues

### Missing ESLint config for UI

**Severity:** Medium  
**Status:** ✅ Resolved  
**Reported:** 2026-01-26  
**Resolved:** 2026-01-26  

The UI does not include an ESLint configuration, so React hook rules and unsafe-type checks are not enforced.  
Added `ui/eslint.config.js` with the safety rules; dependencies already exist in `ui/package.json`.

---

## Test Organization

### Resolved: Concurrent Stripe webhook tests used a stale signing fixture

**Status:** Resolved
**Updated:** 2026-07-26

The concurrency tests signed mock Stripe events with `whsec_test_default`, while
`ConfigureStripeService` injects `DefaultStripeTestConfig().WebhookSecret`
(`stripe-test-webhook`). Valid events were therefore rejected before their
subscription upsert, which made the tests report zero persisted subscriptions.
The tests now derive their HMAC secret from the injected `StripeTestConfig`.

**Validation:** Focused concurrent webhook, subscription, credit, and email
migration tests pass, as does `GOWORK=off go test ./... -count=1 -timeout 10m`
from `api/`.

**Follow-up update (2026-07-28):** The ConfigStore-backed SEO, content, and
variant handler tests no longer silently skip when fixture configuration is
absent or empty; they now fail explicitly, and the obsolete database-variant
helper that always skipped was removed. Focused coverage tests exercise the
affected paths successfully. Unit Health's remaining skip and assertion
findings are reproducible validator defects: three stress tests skip only when
the caller explicitly requests Go short mode, and `TestMain` is a lifecycle
entrypoint rather than an assertion-bearing test. The associated Scenario QA
reports also cover its fabricated missing `runtime/` surface and a UI literal
that its focused-test regex mistakes for `fit`.

### Monolithic Test Files

**Severity:** Medium
**Status:** Open
**Reported:** 2026-01-16 (scenario-completeness-scoring)

The completeness scoring tool reports num[sot]:5 test files validating num[sot]:4+ requirements each, which hurts the score by -10 points:
- `coverage/manual-validations/log.jsonl` (validates num[sot]:14 requirements)
- Plus num[sot]:4 more test files

**Action Required:** Break monolithic test files into focused tests for each requirement.

### Manual Validation Percentage

**Severity:** Low
**Status:** Open
**Reported:** 2026-01-16

20% of validations are manual (recommended max: 10%). Manual validations should be converted to automated tests.

---

## Standards Compliance

### PRD Template Sections

**Severity:** Low
**Status:** Open
**Reported:** 2026-01-16 (scenario-auditor)

The PRD has unexpected sections not in the template:
- Design Decisions
- Success Metrics (Post-Launch)
- Evolution Path
- External References
- Internal References

These may be intentional extensions but are flagged by the standards check.

---

## Performance

No performance issues currently tracked.

---

## Requirements Traceability

### Operational-target linkage and live evidence

**Status:** In progress
**Updated:** 2026-07-26

The requirements registry now links every operational target to the requirement
that describes that exact behavior. The structural validator passes. The
remaining business findings are limited to checked P0/P1 targets whose linked
requirements have not yet earned live completion from a comprehensive test run.

**Next steps:**

- Add focused `[REQ:<id>]` tags and precise test references for the affected
  metrics, billing, subscription, and design requirements.
- Run a comprehensive Test Genie suite so the requirements snapshot can earn
  live evidence; do not hand-edit requirement status or sync artifacts.
- Consolidate the legacy PRD extension headings under the canonical appendix
  through the business-health-owned PRD workflow.

---

## UI/UX

### UX Issues

#### Stripe import modal clarity

**Severity:** Medium  
**Status:** ✅ Resolved  
**Reported:** 2026-01-26  
**Resolved:** 2026-01-26 by Codex

The import modal used per-row action dropdowns (import/overwrite/skip), which made the overwrite behavior unclear and added unnecessary cognitive load.

**Resolution:**
- Replaced per-row action dropdowns with checkboxes (skip = unselected)
- Added explicit conflict warnings and overwrite messaging
- Added product-level selection and bulk actions for new/conflict prices

### Missing Success Notifications

**Severity:** Low
**Status:** ✅ Resolved
**Reported:** 2026-01-16
**Resolved:** 2026-01-16 by Claude (signal-and-feedback-surface-design)

~~When variant operations succeed (archive, delete, weight update), there's no visual confirmation. Consider adding toast notifications for success messages.~~

**Resolution:** Added Toast notification system (`ui/src/shared/ui/Toast.tsx`) with success feedback:
- Customization page: archive, delete, weight update show success toasts
- VariantEditor page: create, update, JSON save show success toasts
- SectionEditor page: save, reorder show success toasts

---

## Last Updated

2026-07-28 by Codex

---

## Structure-health hardcoded-value classification

**Status:** In progress
**Updated:** 2026-07-28

Structure Health reports 82 `PROFILE_HARDCODED_VALUES` warnings through a
syntax-level URL and port matcher. They are not one class of defect and must
not be remediated by turning protocol constants or user-facing help links into
runtime environment variables.

| Classification | Examples | Disposition |
|---|---|---|
| Runtime configuration candidates | Public auth-link origin, optional provider base URLs | Confirm that the value is operator-configurable and add a targeted setting only where the deployment can legitimately vary it. |
| Protocol/provider constants | Stripe, SendGrid, OpenAI, Anthropic endpoints; sitemap namespace | Keep as code-owned integration contracts. They are not deployment settings. |
| Derived values | Cloudflare R2 endpoint assembled from an operator-entered account ID | Keep derived; making it a separate setting permits inconsistent configuration. |
| User-facing content | Storage setup guides, provider comparison links, UI video links | Keep with the content that presents them; move only if product editing requirements require it. |
| Test fixtures and CSS | API mocks, UI tests, CSS data URLs | Keep test- or presentation-local. |
| Fallback content | Baked landing fallback payload | Treat `.vrooli/fallback/fallback.json` as the normal editable source; retain the baked payload only as an offline last resort. |

The provider currently has no auto-fix candidates and cannot distinguish these
classes. The remaining work is therefore to remove genuine configuration
defaults selectively and improve the provider's classification upstream; it is
not evidence that all 82 locations are operationally misconfigured.

---

## Proto Health Evidence Gap

### Measures domain source layout

**Severity:** Advisory  
**Status:** Documented  
**Updated:** 2026-07-27

`MeasuresService` now correctly lives at
`packages/proto/schemas/landing-page-business-suite/v1/measures/measures.proto`,
matching the canonical domain-folder style and `api/handlers/measures`. The
generated Go, TypeScript, Python, API, and CLI consumers were regenerated and
their Go test suites pass.

`proto-health` still reports `handler domain "measures" has no matching proto
domain` because its scenario matcher recognizes the legacy flat
`v1/measures.proto` filename but does not yet associate the canonical nested
path with this scenario. This is validator evidence debt, not an instruction to
restore a non-canonical flat schema. Revisit after proto-health's domain-path
matcher supports canonical nested source paths.

## Work ladder

- Rung: W3 / R2–R3 implementation hardening
- Evidence: the user's active objective explicitly requires implementation of
  `landing-page-business-suite-in-place-modernization.md`, whose Outcome and
  Phase 7 direct `api/internal/<domain>/` plus `api/handlers/<domain>/`. The
  active swarm goal's earlier `api/domain/<name>/` wording conflicts with that
  current instruction; the implementation and Architecture Cartographer now
  use the requested handler/internal layout (architecture L3 in Test Genie run
  `20260730-033431-64b6ffcd`).
- Blocker: none; do not create an `api/domain/` parallel tree. Continue the
  requested domain decomposition and transport migration in the existing
  `handlers` and `internal` boundaries.
- Measured: 2026-07-30
