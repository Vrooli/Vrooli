# Plan: Add Detailed Rule Documentation and Per-Rule Execution to Standards Page

## Required Reading

```bash
prompt-manager skill read react-coherence ux api-steer seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
```

## 0. Greenfield Declaration

This is a **greenfield enhancement** to the brand-manager scenario. The Standards page and audit endpoint exist but have not been shipped to external users; per-rule evaluation and rule metadata are net-new surface area. No backwards-compatibility shims, deprecation paths, or feature flags are required. The existing audit `POST /api/v1/audit/evaluate/{scenario}` call shape MUST remain working when no `?rule=` query param is supplied so that the ScannerPage continues to function unchanged — that is the only compatibility constraint.

## 1. Purpose

Transform the Standards page from a static rule list into an interactive audit tool where each rule is fully documented (what it checks, target files, examples, fix instructions) and individually executable against any scenario.

## 2. Problem Statement

The Standards page currently shows 5 branding rules with only name, description, and severity badge. Users cannot understand *what* a rule actually validates, *which files* are involved, or *how to fix* violations. There is no way to run a single rule from this page — the existing audit endpoint (`POST /api/v1/audit/evaluate/{scenario}`) evaluates all rules at once and is only accessible from the Scanner page.

## 3. Scope

**In scope:**
- Enrich `BrandingRule` struct with metadata fields (target files, examples, fix instructions, severity rationale, detailed description)
- Expand `GET /api/v1/standards` response to include metadata
- Add `?rule=` query parameter to existing audit evaluate endpoint for single-rule filtering
- Build expandable rule card UI on StandardsPage (multiple cards may be expanded simultaneously)
- Add per-rule "Check Scenario" inline evaluation
- Add batch "Scan All Rules" with fraction + severity-aware color summary
- Port prompt-manager's `MarkdownRenderer` + `CodeBlock` components into brand-manager UI to render rule metadata markdown
- Add `@tailwindcss/typography` to brand-manager UI and scope `prose` to the markdown wrapper

**Out of scope:**
- Persisting audit results to database
- Adding new rules beyond the existing 5
- Modifying ScannerPage (other than verifying no regression)
- Authentication/authorization changes
- Porting prompt-manager's `MarkedParser.ts` (prompt-manager-specific preprocessing not needed here)

## 4. Current Technical Context

### Backend
- **`scenarios/brand-manager/api/handlers/standards.go`**: Defines `BrandingRule` struct (ID, Name, Description, Severity, Category) and `standardRules` slice with 5 hardcoded rules. `GetStandards` handler returns `{rules, count}`.
- **`scenarios/brand-manager/api/handlers/audit_provider.go`**: `EvaluateScenario` handler evaluates ALL rules against a scenario's brand. Returns `{scenario, results: [{rule_id, pass, severity, message}]}`. Uses `ruleEvaluators` array pairing rules with validation functions.
- **`scenarios/brand-manager/api/handlers/brands.go`**: `RegisterRoutes` registers all routes under `/api/v1`.

### Frontend
- **`scenarios/brand-manager/ui/src/pages/StandardsPage.tsx`**: Fetches standards, renders flat list with severity badges.
- **`scenarios/brand-manager/ui/src/pages/ScannerPage.tsx`**: Reference for audit UX — calls `evaluateScenario()`, displays per-rule pass/fail with colored dots.
- **`scenarios/brand-manager/ui/src/lib/api.ts`**: `fetchStandards()`, `evaluateScenario()`, `fetchAuditRules()` functions. `StandardsResult` interface has `rules: {id, name, description, severity}[]`.
- **`scenarios/brand-manager/ui/package.json`**: No markdown dependencies today.

### Domain
- **`scenarios/brand-manager/api/domain/types.go`**: `Brand` struct with `Identity`, `Colors`, `Typography`, `Voice` fields.

### Reference (prompt-manager markdown stack to port)
- `scenarios/prompt-manager/ui/src/components/markdown/MarkdownRenderer.tsx` — ReactMarkdown wrapper with custom component overrides
- `scenarios/prompt-manager/ui/src/components/CodeBlock.tsx` — Shiki-powered code blocks with copy button + language label
- Versions confirmed in prompt-manager: `react-markdown@^9.1.0`, `remark-gfm@^4.0.1`, `marked@^15.0.0`, `shiki@^1.29.2`, plus `@tailwindcss/typography`

## 5. Target End State

- Each rule on Standards page expands (independently, multiple may be open at once) to show: detailed description, target file globs, passing example, failing example, fix instructions, severity rationale
- Each expanded rule has a "Check Scenario" button that evaluates just that rule using the shared scenario input at the page top
- Top-level "Scan All Rules" button evaluates all rules with fraction summary (e.g., `4/5 rules passing`) colored green/amber/red based on severity of any failures
- API returns full metadata on standards endpoint; audit endpoint accepts optional `?rule=` query param for single-rule filtering
- Markdown content (examples, fix instructions) renders via the ported `MarkdownRenderer` component using `@tailwindcss/typography` `prose` styling scoped to the markdown wrapper
- ScannerPage continues to work unchanged

## 6. Implementation Strategy

### Phase 1: Backend — Enrich Rule Metadata
1. Add metadata fields to `BrandingRule` struct in `standards.go`:
   - `TargetFiles []string` — glob patterns
   - `DetailedDescription string` — markdown explanation of validation logic
   - `PassingExample string` — markdown
   - `FailingExample string` — markdown
   - `FixInstructions string` — markdown
   - `SeverityRationale string` — markdown
2. Populate metadata as **hardcoded struct literals** for all 5 rules in `standardRules` (per d1: keeps content alongside code, minimal change).
3. Update `GetStandards` handler JSON response to include new fields. JSON tag names use snake_case: `target_files`, `detailed_description`, `passing_example`, `failing_example`, `fix_instructions`, `severity_rationale`.
4. Add `?rule=` query parameter support to `EvaluateScenario` (per d2: backward-compatible filter):
   - If `rule` param present, look it up in `ruleEvaluators` and evaluate only that one
   - If `rule` param does not match any known rule ID → return 400 with descriptive error
   - If `rule` param absent → existing behavior (evaluate all rules) preserved unchanged

### Phase 2: Frontend — Markdown Stack
1. Add deps to `scenarios/brand-manager/ui/package.json`: `react-markdown@^9.1.0`, `remark-gfm@^4.0.1`, `marked@^15.0.0`, `shiki@^1.29.2`, `@tailwindcss/typography` (latest compatible).
2. Update `scenarios/brand-manager/ui/tailwind.config.*` to include the typography plugin.
3. Port `MarkdownRenderer.tsx` and `CodeBlock.tsx` from `scenarios/prompt-manager/ui/src/components/` into a parallel folder under brand-manager's UI. Adjust imports and any prompt-manager-specific paths. Do NOT port `MarkedParser.ts`.
4. Confirm `prose` class is scoped to the renderer's wrapper element (NOT applied globally) to avoid affecting other brand-manager pages.

### Phase 3: Frontend — API Types & Client
1. Update `StandardsResult` rule type in `lib/api.ts` to include the 6 new fields.
2. Add `evaluateRule(scenario, ruleId)` helper that calls `POST /api/v1/audit/evaluate/{scenario}?rule={ruleId}` and returns the same shape as `evaluateScenario` (single result in `results`).

### Phase 4: Frontend — Expandable Rule Cards
1. Add expand/collapse state in StandardsPage as `Record<ruleId, boolean>` (per d6: multiple cards may be expanded simultaneously).
2. Build expanded view rendering each metadata field; markdown fields go through `MarkdownRenderer`.
3. Add chevron/toggle indicator on each card header.

### Phase 5: Frontend — Per-Rule & Batch Execution
1. Add a single shared scenario name input near the page top (per d4). Persist input value in component state.
2. "Check Scenario" button inside each expanded card uses the shared input value, calls `evaluateRule()`, displays inline pass/fail using the same colored-dot pattern used in ScannerPage.
3. "Scan All Rules" button at page top uses the shared input, calls `evaluateScenario()`, fans results out to each rule card AND renders a top-level summary.
4. Summary format (per d7): `X / N rules passing`, colored:
   - green when all pass
   - red when any failure has severity `error`
   - amber otherwise (e.g., warning- or info-level failures only)

## 7. Contract Decisions

### API: `GET /api/v1/standards` — enhanced response
```json
{
  "rules": [
    {
      "id": "has-logo",
      "name": "Logo Present",
      "description": "...",
      "severity": "warning",
      "category": "branding",
      "target_files": ["ui/public/logo.png", "ui/public/manifest.json"],
      "detailed_description": "Validates that the brand identity defines a non-empty logo path...",
      "passing_example": "```json\n{\"identity\": {\"logo_path\": \"/public/logo.png\"}}\n```",
      "failing_example": "```json\n{\"identity\": {\"logo_path\": \"\"}}\n```",
      "fix_instructions": "1. Add a logo asset to `ui/public/`...\n2. Set `Identity.LogoPath`...",
      "severity_rationale": "Warning rather than error because a missing logo degrades brand presentation but does not block scenario function."
    }
  ],
  "count": 5
}
```

### API: `POST /api/v1/audit/evaluate/{scenario}?rule={ruleId}` — single-rule filter
- Optional `rule` query param.
- When present and valid: response shape unchanged; `results` array contains exactly the matched rule.
- When present but unknown: HTTP 400 with `{"error": "unknown rule: <ruleId>"}`.
- When absent: HTTP 200 with all rules evaluated (existing behavior).

## 8. Testing Plan

### Backend (`scenarios/brand-manager/api/...`)
- `TestGetStandards_ReturnsAllMetadataFields` — every rule in response has non-empty `target_files`, `detailed_description`, `passing_example`, `failing_example`, `fix_instructions`, `severity_rationale`.
- `TestEvaluateScenario_SingleRule_ReturnsOneResult` — `?rule=has-logo` produces a `results` array of length 1 whose `rule_id` is `has-logo`.
- `TestEvaluateScenario_UnknownRule_Returns400` — `?rule=does-not-exist` returns HTTP 400 with descriptive error.
- `TestEvaluateScenario_NoRuleParam_ReturnsAllResults` — backward-compat regression guard; existing ScannerPage flow keeps producing all 5 results.

### Frontend (`scenarios/brand-manager/ui/...`)
- StandardsPage: expand/collapse interaction (multiple cards open simultaneously).
- StandardsPage: shared scenario input populates per-rule and batch evaluations.
- StandardsPage: per-rule "Check Scenario" calls `evaluateRule` and renders inline pass/fail.
- StandardsPage: "Scan All Rules" produces summary with correct color per severity rule (all pass → green, any error-severity fail → red, only warnings/info fail → amber).
- ScannerPage regression: ensure ScannerPage's existing evaluate call (no `rule` param) still renders correctly.
- MarkdownRenderer: code block renders via Shiki with copy button; `prose` class is scoped, does not leak to outer page.

## 9. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Shiki bundle size noticeably increases UI payload | Medium | Low | Shiki is already a peer in prompt-manager; brand-manager UI is internal tooling, payload growth is acceptable. Confirm via build size report. |
| Adding 5 npm deps violates implicit dep-add policy | Low | Medium | All 5 deps are already used in sibling scenario prompt-manager; ports are intentional reuse. Document in PR description. |
| ScannerPage regresses because audit endpoint changes | Low | High | The `?rule=` param is purely additive. Backend test `TestEvaluateScenario_NoRuleParam_ReturnsAllResults` is the regression guard; manual ScannerPage smoke check during rollout. |
| Markdown content embeds raw HTML, opening XSS surface | Low | High | `react-markdown` does not render raw HTML by default — do not enable `rehype-raw`. Rule metadata is author-controlled (hardcoded in Go), not user input, further limiting risk. |
| `@tailwindcss/typography` `prose` class bleeds into other pages | Medium | Low | Scope `prose` only to the `MarkdownRenderer` wrapper element, not globally. Add a frontend test asserting outer page elements are unaffected. |
| Hardcoded markdown content drifts as rules evolve | Low | Low | Acceptable for 5 static rules; revisit if rule count grows substantially. |
| Sibling backlog items (`brand-manager-scenario-picker`, `brand-manager-discovery-import-ui`) may also touch StandardsPage or `lib/api.ts` | Low | Medium | These items are queued separately under the same `brand-manager-readiness` initiative. Coordinate merge order with orchestrator; flag any shared file touches in the PR. |

## 10. Non-goals / Prohibited Patterns
- No new database tables, migrations, or persisted audit history
- No changes to ScannerPage UI/UX
- No new rule definitions
- No backwards-compat shims (greenfield)
- No raw HTML in markdown rendering (no `rehype-raw`)
- No global application of `prose` class

## 11. Definition of Done
- [ ] `BrandingRule` struct has the 6 new metadata fields, populated for all 5 rules
- [ ] `GET /api/v1/standards` returns enriched rule objects
- [ ] `POST /api/v1/audit/evaluate/{scenario}?rule=X` filters to a single rule; unknown rule → 400; no `rule` param → all rules
- [ ] Backend tests in §8 pass
- [ ] `MarkdownRenderer` and `CodeBlock` ported to brand-manager UI
- [ ] `@tailwindcss/typography` installed; `prose` scoped to renderer
- [ ] StandardsPage shows expandable cards with full documentation rendered as markdown
- [ ] Per-rule "Check Scenario" works end-to-end
- [ ] "Scan All Rules" shows fraction-with-color summary and per-rule results
- [ ] Frontend tests in §8 pass
- [ ] No regressions on ScannerPage audit integration

## 12. Rollout / Validation Checklist

1. **Build & type-check** brand-manager UI and API: `vrooli scenario build brand-manager` (or equivalent), confirm zero TS/Go errors.
2. **Run backend tests** for the new endpoint behavior and the no-`rule`-param regression guard.
3. **Run frontend tests** including the ScannerPage regression assertion.
4. **Operator action — restart the active scenario** (NOT performed by the implementation agent): the operator runs `vrooli scenario restart brand-manager` to load the new UI bundle and API binary, then manually:
   - Opens the Standards page, expands each of the 5 rules, and confirms metadata renders.
   - Enters a known scenario name in the shared input, clicks "Check Scenario" on one rule, confirms inline pass/fail.
   - Clicks "Scan All Rules", confirms summary fraction + correct color, and per-rule results populate.
   - Opens the Scanner page and confirms the existing scan flow still works (regression check).
5. **Sibling-item coordination check** — before merging, confirm with the orchestrator that `brand-manager-scenario-picker` and `brand-manager-discovery-import-ui` have not introduced overlapping edits to `StandardsPage.tsx` or `lib/api.ts`. If they have, coordinate rebase order.
