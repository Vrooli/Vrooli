# Plan: Add Detailed Rule Documentation and Per-Rule Execution to Standards Page

## Required Reading

```bash
prompt-manager skill read react-coherence ux api-steer seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
```

## 1. Purpose

Transform the Standards page from a static rule list into an interactive audit tool where each rule is fully documented (what it checks, target files, examples, fix instructions) and individually executable against any scenario.

## 2. Problem Statement

The Standards page currently shows 5 branding rules with only name, description, and severity badge. Users cannot understand *what* a rule actually validates, *which files* are involved, or *how to fix* violations. There is no way to run a single rule from this page — the existing audit endpoint (`POST /api/v1/audit/evaluate/{scenario}`) evaluates all rules at once and is only accessible from the Scanner page.

## 3. Scope

**In scope:**
- Enrich `BrandingRule` struct with metadata fields (target files, examples, fix instructions)
- Expand `GET /api/v1/standards` response to include metadata
- Add `?rule=` query parameter to existing audit evaluate endpoint for single-rule filtering
- Build expandable rule card UI on StandardsPage
- Add per-rule "Check Scenario" inline evaluation
- Add batch "Scan All Rules" with summary

**Out of scope:**
- Persisting audit results to database
- Adding new rules beyond the existing 5
- Modifying ScannerPage
- Authentication/authorization changes

## 4. Current Technical Context

### Backend
- **`handlers/standards.go`**: Defines `BrandingRule` struct (ID, Name, Description, Severity, Category) and `standardRules` slice with 5 hardcoded rules. `GetStandards` handler returns `{rules, count}`.
- **`handlers/audit_provider.go`**: `EvaluateScenario` handler evaluates ALL rules against a scenario's brand. Returns `{scenario, results: [{rule_id, pass, severity, message}]}`. Uses `ruleEvaluators` array pairing rules with validation functions.
- **`handlers/brands.go`**: `RegisterRoutes` registers all routes under `/api/v1`.

### Frontend
- **`pages/StandardsPage.tsx`**: Fetches standards, renders flat list with severity badges.
- **`pages/ScannerPage.tsx`**: Has audit integration — calls `evaluateScenario()`, displays per-rule pass/fail with colored dots.
- **`lib/api.ts`**: `fetchStandards()`, `evaluateScenario()`, `fetchAuditRules()` functions. `StandardsResult` interface has `rules: {id, name, description, severity}[]`.

### Domain
- **`domain/types.go`**: `Brand` struct with `Identity`, `Colors`, `Typography`, `Voice` fields.

## 5. Target End State

- Each rule on Standards page expands to show: validation logic explanation, target file globs, passing/failing examples, fix instructions, severity rationale
- Each expanded rule has a "Check Scenario" button that evaluates just that rule
- Top-level "Scan All Rules" button evaluates all rules with summary score
- API returns full metadata on standards endpoint; audit endpoint supports single-rule filtering

## 6. Implementation Strategy

### Phase 1: Backend — Enrich Rule Metadata
1. Add metadata fields to `BrandingRule` struct:
   - `TargetFiles []string` (glob patterns)
   - `PassingExample string` (markdown)
   - `FailingExample string` (markdown)
   - `FixInstructions string` (markdown)
   - `SeverityRationale string`
   - `DetailedDescription string`
2. Populate metadata for all 5 rules in `standardRules`
3. Update `GetStandards` handler JSON response to include new fields
4. Add `?rule=` query parameter support to `EvaluateScenario`:
   - If `rule` param present, filter `ruleEvaluators` to only that rule
   - Return same response shape but with single result

### Phase 2: Frontend — API Types & Client
1. Update `StandardsResult` interface in `api.ts` with new fields
2. Add `evaluateRule(scenario, ruleId)` function that calls `POST /api/v1/audit/evaluate/{scenario}?rule={ruleId}`

### Phase 3: Frontend — Expandable Rule Cards
1. Add expand/collapse state per rule in StandardsPage
2. Build expanded view showing all metadata fields
3. Render markdown content for examples and fix instructions
4. Add chevron/toggle indicator

### Phase 4: Frontend — Per-Rule & Batch Execution
1. Add "Check Scenario" button in each expanded card
2. Inline scenario name input + Run button
3. Display pass/fail result inline
4. Add "Scan All Rules" button at page top
5. Show per-rule results with summary (X/5 passing, compliance score)

## 7. Contract Decisions

### API: `GET /api/v1/standards` enhanced response
```json
{
  "rules": [{
    "id": "has-logo",
    "name": "Logo Present",
    "description": "...",
    "severity": "warning",
    "category": "branding",
    "target_files": ["ui/public/logo.png", "ui/public/manifest.json"],
    "detailed_description": "Validates that the brand identity...",
    "passing_example": "```json\n{\"logo_path\": \"/public/logo.png\"}\n```",
    "failing_example": "```json\n{\"logo_path\": \"\"}\n```",
    "fix_instructions": "1. Add a logo file to...",
    "severity_rationale": "Warning because..."
  }],
  "count": 5
}
```

### API: `POST /api/v1/audit/evaluate/{scenario}?rule={ruleId}` — single-rule filter
- Same response shape as full evaluation
- `results` array contains only the matched rule
- 400 if `rule` param doesn't match any known rule ID

## 8. Testing Plan

### Backend Tests
- Test `GetStandards` returns all metadata fields populated
- Test `EvaluateScenario` with `?rule=has-logo` returns single result
- Test `EvaluateScenario` with `?rule=invalid` returns 400
- Test `EvaluateScenario` without `rule` param still returns all results (backward compat)

### Frontend Tests
- Expand/collapse interaction
- Scenario input + single-rule evaluation flow
- Batch scan renders all results with summary

## 9. Risks + Mitigations

| Risk | Mitigation |
|------|-----------|
| Large markdown content in rule metadata bloats standards response | Metadata is static for 5 rules — negligible payload |
| ScannerPage uses same audit endpoint | Adding optional `rule` query param is backward-compatible |
| Markdown rendering in UI needs a library | Check if existing UI already has markdown rendering; if not, use a lightweight renderer |

## 10. Non-goals / Prohibited Patterns
- No new database tables or migrations
- No audit result history/persistence
- No changes to ScannerPage
- No new rule definitions

## 11. Definition of Done
- [ ] `GET /api/v1/standards` returns enriched rule objects with all metadata
- [ ] `POST /api/v1/audit/evaluate/{scenario}?rule=X` filters to single rule
- [ ] Standards page shows expandable cards with full documentation
- [ ] Per-rule "Check Scenario" works end-to-end
- [ ] Batch "Scan All Rules" shows summary
- [ ] Backend tests pass for new endpoint behavior
- [ ] No regressions on ScannerPage audit integration
