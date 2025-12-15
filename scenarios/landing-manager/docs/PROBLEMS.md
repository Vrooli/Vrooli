---
title: "Known Issues"
description: "Tracked problems and their status"
category: "operational"
order: 3
audience: ["developers"]
---

# Problems & Known Issues

> **Last Updated**: 2025-11-27 (Scenario Improver Agent Phase 1 Iteration 10)
> **Status**: requirements 65% (13/20 passing), completeness 59/100 (functional_incomplete), test suite 5/6 phases passing

## Open Issues

### 🟡 BAS Integration Test Failures - Workflow Assertion Mismatches (MEDIUM PRIORITY, 2025-11-27T23:22)
- **What**: Integration phase failing with 3/3 BAS workflows executing but failing on UI element assertions.
- **Status**: ✅ Workflow validation issues FIXED (iteration 10). Workflows now execute successfully. Structure phase passing ✅. Playbook metadata correct, selectors properly defined.
- **Progress**:
  - **Fixed** (2025-11-27): Workflow schema validation errors (waitUntil: "networkidle0" → "networkidle", assertMode: "contains_text" → "text_contains", expectedText → expectedValue, waitForMs → durationMs for time waits)
  - **Current Issue**: Workflows execute but assertions fail because UI elements/content don't match expected values
  - dry-run-generation: 9 steps, 1/3 assertions passed (fails at assert-result-message, assert-status)
  - lifecycle-management: 4 steps, 0/1 assertions passed (fails at assert-lifecycle-controls-exist)
  - scenario-promotion: 4 steps, 0/1 assertions passed (fails at assert-promote-button-exists)
- **Root Cause**: UI implementation doesn't fully match the expected test automation flow. Either:
  1. UI elements use different selectors than workflows expect
  2. UI flow/behavior differs from workflow expectations
  3. UI features partially implemented but not complete end-to-end
- **Impact**: Integration tests fail but **core functionality exists and is proven via API/unit tests**. Features like dry-run, lifecycle management, and promotion work via API/CLI. Requirements TMPL-DRY-RUN, TMPL-LIFECYCLE, TMPL-PROMOTION marked "in_progress" pending E2E validation.
- **Boundary**: Within landing-manager scope - UI implementation needs alignment with test expectations.
- **Evidence**:
  - Workflow artifacts: `/home/matthalloran8/Vrooli/scenarios/landing-manager/coverage/automation/bas/cases/01-template-management/ui/`
  - API tests: All lifecycle/dry-run/promotion API tests passing in api/integration_test.go
  - Unit tests: UI component tests passing (111/111 React tests)
- **Next**: Either (1) update UI to match workflow expectations, or (2) update workflows to match actual UI implementation, or (3) complete unimplemented UI features.

### 🟡 Factory validation coverage gaps (PARTIALLY RESOLVED)
- **What**: Factory UI/API flows weren't validated end-to-end in automation.
- **Resolution**: Structure phase now passing ✅. BAS playbooks exist for TMPL-AVAILABILITY and TMPL-DRY-RUN with correct metadata structure.
- **Remaining**: Integration phase failing (2/2 workflows fail at execution - see above). Manual validations at 24% (6/26 validations).
- **Impact**: Medium - core functionality validated via unit/CLI tests, but missing working browser automation for end-to-end flows.
- **Next**: Debug BAS workflow failures (see issue above). Run `vrooli scenario ui-smoke landing-manager` to verify production bundle (currently passing ✅).

### 🟠 Agent customization dependencies
- **What**: `customize` endpoint files issues to app-issue-tracker and is exposed in the factory UI; end-to-end agent flow (issue creation + investigation) hasn’t been validated recently.
- **Impact**: Users may believe an agent is working when downstream automation is unhealthy; silent failure risk.
- **Next**: Keep app-issue-tracker healthy, run a full customize request against a generated scenario, and capture the resulting issue/run IDs for verification. No on-box model dependency.

### 🟡 Template validation drift
- **What**: Template PRD and requirements were out of sync on P1 design/branding status; checkboxes now updated, but remaining targets (multi-template, advanced analytics, etc.) still lack validation.
- **Impact**: Risk of over-reporting readiness if future edits don't land in the template payload.
- **Next**: Keep runtime changes inside `scripts/scenarios/templates/landing-page-react-vite`; ensure requirement updates remain paired with PRD status changes.

### 🟢 Lighthouse NO_LCP Infrastructure Issue (RESOLVED)
- **What**: Lighthouse performance phase was failing with 0% score due to NO_LCP error. This was a Lighthouse detection issue, not an actual performance problem.
- **Resolution**: Performance phase now passing (90% performance score, 97% accessibility, 96% best practices, 92% SEO) as of 2025-11-24. The timing issues appear to have been resolved through scenario restart and config adjustments.
- **Status**: Closed - test suite now reports 5/6 phases passing with performance phase ✅.

### 🟢 UI Test Assertion Issues (RESOLVED)
- **What**: Created 6 focused UI test files for multi-layer validation showing 17/81 failures due to strict assertions.
- **Resolution**: All 81 UI tests now passing ✅ (as of 2025-11-25). Fixed by:
  1. Replacing strict `getByText` with flexible `getAllByText` for duplicate content
  2. Adding missing API mocks (`getTemplate`, `listGeneratedScenarios`)
  3. Adding defensive null-safety in FactoryHome.tsx (templates?.find, templates?.[0] ?? null)
- **Status**: Closed - robust test foundation established with 100% pass rate.

### 🟢 UI Smoke Test Issues (FULLY RESOLVED)
- **What (Iteration 9)**: UI smoke test was reporting network failures for lifecycle status endpoints (e.g., HTTP 404 → `/api/v1/lifecycle/test-dry/status`)
- **Root Cause (Iteration 9)**: Test artifacts (test-dry, test-landing) left in `generated/` folder were causing UI to attempt fetching status for non-running scenarios, resulting in expected 404s that smoke tests treated as errors.
- **Resolution (Iteration 9)**: (1) Cleaned up test artifacts from `generated/` folder (test-dry, test-landing removed), (2) Fixed preview links API to use `vrooli scenario port` command instead of reading service.json (resolves underlying issue where GetPreviewLinks failed for staging scenarios). Preview functionality now works correctly - users can start generated scenarios from staging area and preview links appear immediately.
- **What (Iteration 10)**: After cleanup, UI smoke test failed with "Cannot read properties of null (reading 'length')" error.
- **Root Cause (Iteration 10)**: API endpoint `/api/v1/generated` returned `null` instead of `[]` when generated folder is empty. In Go, `var scenarios []GeneratedScenario` creates nil slice which JSON-marshals to `null`. UI code calls `.length` on this, causing crash.
- **Resolution (Iteration 10)**: Changed template_service.go:775 from `var scenarios []GeneratedScenario` to `scenarios := make([]GeneratedScenario, 0)` to ensure empty array instead of null.
- **Status**: FULLY RESOLVED ✅ (as of 2025-11-28). UI smoke tests pass (1255ms, handshake: 3ms). Empty generated/ folder now safe.
- **User-visible improvement**: Preview/open functionality for generated landing pages works end-to-end. Users can generate → start → click "Public Landing" or "Admin Dashboard" links directly from factory UI without needing terminal or manual URL construction. UI now handles empty state gracefully.

### 🟡 Requirements Structure Misalignment (KNOWN, PARTIALLY ADDRESSED)
- **What**: Completeness penalty (-13pts) due to test quality issues:
  1. Monolithic test files: 2 test files validate ≥4 requirements each (api/template_service_test.go, ui FactoryHome.test.tsx)
  2. 3 requirements reference unsupported test/ directories (CLI tests in test/cli/)
  3. 3 critical requirements (P0/P1) lack multi-layer AUTOMATED validation
- **Impact**: Completeness score 58/100 (functional_incomplete) vs potential ~71/100 (functional_complete) without penalty
- **Root Cause**: Test quality issues prevent gaming-prevention system from accepting validation
- **Progress**: Requirements completion improved from 67% to 80% (10/15 → 12/15). Unit test coverage at 68.7% (target 90%). All tests passing with zero flaky tests.
- **Next**: Split monolithic test files, update unsupported test/ refs to valid locations, add missing test layers to critical requirements

## Security Audit False Positive - Path Traversal (template_service.go:499-500)

**Status**: Known false positive  
**Severity**: None (secure by design)  
**Date**: 2025-11-26

scenario-auditor flags lines 499-500 in `api/template_service.go` as HIGH severity path traversal vulnerabilities:

```go
if strings.HasPrefix(strValue, "file:../../../packages/") {
    packageName := strings.TrimPrefix(strValue, "file:../../../packages/")
```

**Why this is a false positive:**

1. **Constant prefix validation**: Code only processes strings matching exact prefix `"file:../../../packages/"`, not arbitrary user input
2. **Path cleaning**: `filepath.Clean(packageName)` normalizes the extracted path (line 502)
3. **Escape validation**: Explicitly checks `strings.Contains(packageName, "..")` and skips if true (lines 504-506)
4. **Directory boundary validation**: Verifies `filepath.Clean(absolutePath)` stays within `filepath.Clean(packagesDir)` (lines 509-511)
5. **Context**: This code runs during scenario generation to fix workspace dependencies by converting relative paths to absolute paths

**Pattern matcher limitation**: The static analysis tool flags the string prefix check itself, not understanding the multi-layer validation that follows.

**Recommendation**: Accept as false positive. The code implements defense-in-depth path validation and is secure by design. No changes needed unless more sophisticated pattern matching becomes available.
