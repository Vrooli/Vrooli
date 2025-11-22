# Completeness Gaming Prevention Plan

**Status**: Planning
**Created**: 2025-11-22
**Estimated Effort**: 3-5 days
**Priority**: P0 (Critical for ecosystem-manager accuracy)

---

## Executive Summary

The completeness scoring and requirement validation systems have exploitable loopholes allowing agents to achieve artificially high scores (96-100/100) without proper test coverage. This plan addresses five major gaming vectors and improves documentation to prevent future exploitation.

**Key Issues**:
1. Unsupported test directories (anything in `test/` except `test/playbooks/`) accepted as validation refs
2. No enforcement of validation diversity (API, UI, e2e layers)
3. One test file validating 6+ requirements with no penalty
4. 1:1 operational target-to-requirement inflation
5. Gaming patterns not detected or flagged

**Expected Outcomes**:
- Accurate completeness scores reflecting true implementation quality
- Clear validation that requirements are tested across multiple layers
- Prevention of "one file validates everything" anti-pattern
- Smarter operational target grouping validation
- Early warning system for gaming patterns

---

## Problem Analysis

### Current Gaming Patterns (Evidence from Real Scenarios)

#### deployment-manager (Score: 96/100 - Gamed)
```
✅ Quality: 50/50 (100% pass rates)
  - 37 requirements, 37 targets, 37 tests (perfect 1:1:1 mapping)
  - ALL validations point to test/cli/*.bats files
  - Same BATS file validates 6 requirements
  - Zero API tests, zero UI tests, zero e2e playbooks

❌ Missing validation:
  - No API unit tests (despite Go API)
  - No UI unit tests (despite React UI)
  - Using unsupported test/cli/ directory
  - Monolithic BATS files doing everything
```

#### browser-automation-studio (Score: 62/100 - Proper)
```
⚠️  Quality: 26/50 (52% pass rates - honest assessment)
  - 63 requirements, 7 targets, 90 tests (proper ratios)
  - Diverse validation sources:
    * api/**/*_test.go (API unit tests)
    * ui/src/**/*.test.tsx (UI unit tests)
    * test/playbooks/**/*.json (e2e automation)
  - Multiple validations per requirement (6 for critical requirements)
  - Proper operational target grouping (7 PRD goals → 63 requirements)

✅ Proper validation:
  - API layer tested with Go unit tests
  - UI layer tested with Vitest
  - E2e tested with BAS playbooks
  - test/phases/ correctly excluded from validation refs
```

---

## Solution Architecture

### Phase 1: Validation Source Filtering (High Priority)

**Goal**: Reject unsupported test/ directory refs, allow only proper validation sources

#### 1.1 Define Allowed Validation Source Patterns

**Allowed**:
- `api/**/*_test.go` - Go unit tests
- `api/**/tests/**/*` - Go test directories
- `ui/src/**/*.test.{ts,tsx,js,jsx}` - Frontend unit tests
- `test/playbooks/**/*.{json,yaml}` - E2e automation workflows

**Rejected**:
- Anything in `test/` that's not `test/playbooks/**/*.{json,yaml}`. 
- `coverage/**/*` - Test artifacts (results, not sources)

**Rationale**:
- BAS (gold standard) only refs: API tests, UI tests, playbooks
- test/phases/ orchestrates tests but isn't validation evidence
- CLI tests validate the CLI (which is *supposed* to be a thin wrapper over the API), not the underlying requirements
- Phase directories contain infrastructure, not individual test files

#### 1.2 Implementation Location

**File**: `scripts/requirements/lib/evidence.js`

**Function**: `detectValidationSource(validation)`

**Changes**:
```javascript
function detectValidationSource(validation) {
  const ref = (validation.ref || '').toLowerCase();
  const workflowId = (validation.workflow_id || '').toLowerCase();

  if (!ref && !workflowId) {
    return null;
  }

  // NEW: Reject unsupported test/ directories
  if (ref.startsWith('test/')) {
    // Only allow test/playbooks/ for e2e automation
    if (ref.startsWith('test/playbooks/') && (ref.endsWith('.json') || ref.endsWith('.yaml'))) {
      const slug = path.basename(ref, path.extname(ref));
      return { kind: 'automation', name: slug };
    }

    // Reject everything else under test/
    console.warn(`Validation ref rejected: ${ref} (unsupported test/ directory)`);
    return null;
  }

  // Existing logic for API tests, UI tests, automation
  if (validation.type === 'test') {
    // API tests
    if (ref.endsWith('_test.go') || ref.includes('/tests/')) {
      return { kind: 'phase', name: 'unit' };
    }
    // UI tests
    if (ref.match(/ui\/src\/.*\.test\.(ts|tsx|js|jsx)$/)) {
      return { kind: 'phase', name: 'unit' };
    }
  }

  // ... rest of existing logic
}
```

**Impact**: Existing gamed scenarios will show reduced completeness scores when re-synced.

#### 1.3 Validation in Auto-Sync

**File**: `scripts/requirements/lib/sync.js`

**Add pre-sync validation** to warn about invalid refs before writing to files:

```javascript
function validateRequirementRefs(requirements) {
  const warnings = [];

  requirements.forEach(req => {
    (req.validation || []).forEach(v => {
      const source = detectValidationSource(v);
      if (!source && v.ref) {
        warnings.push({
          requirement: req.id,
          ref: v.ref,
          reason: 'Unsupported validation source (will be ignored)'
        });
      }
    });
  });

  return warnings;
}
```

**Integration point**: Call before `syncRequirementRegistry()` to surface issues early.

---

### Phase 2: Validation Diversity Enforcement (High Priority)

**Goal**: Require requirements to be validated across multiple test layers (API, UI, e2e)

#### 2.1 Detect Scenario Components

**File**: `scripts/scenarios/lib/completeness-data.js` (new helper)

**Rationale**: Don't penalize API-only scenarios for missing UI tests when they have no UI component.

```javascript
/**
 * Detect which components exist in the scenario
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {Set<string>} Set of component types (API, UI)
 */
function detectScenarioComponents(scenarioRoot) {
  const components = new Set();

  // Check for API component
  const apiDir = path.join(scenarioRoot, 'api');
  if (fs.existsSync(apiDir)) {
    const hasGoFiles = fs.readdirSync(apiDir).some(f => f.endsWith('.go'));
    if (hasGoFiles) components.add('API');
  }

  // Check for UI component
  const uiDir = path.join(scenarioRoot, 'ui');
  if (fs.existsSync(uiDir)) {
    const pkgJson = path.join(uiDir, 'package.json');
    if (fs.existsSync(pkgJson)) components.add('UI');
  }

  return components;
}
```

#### 2.2 Define Validation Layers

```javascript
const VALIDATION_LAYERS = {
  API: {
    patterns: [
      /\/api\/.*_test\.go$/,
      /\/api\/.*\/tests\//,
    ],
    description: 'API unit tests (Go)'
  },
  UI: {
    patterns: [
      /\/ui\/src\/.*\.test\.(ts|tsx|js|jsx)$/,
    ],
    description: 'UI unit tests (Vitest/Jest)'
  },
  E2E: {
    patterns: [
      /\/test\/playbooks\/.*\.(json|yaml)$/,
    ],
    description: 'End-to-end automation (BAS playbooks)'
  },
  MANUAL: {
    types: ['manual'],
    description: 'Manual validation evidence'
  }
};
```

#### 2.3 Layer Detection Function

**File**: `scripts/scenarios/lib/completeness-data.js` (new helper)

```javascript
/**
 * Detect which validation layers a requirement is tested in
 * @param {object} requirement - Requirement object with validations
 * @returns {Set<string>} Set of layer names (API, UI, E2E, MANUAL)
 */
function detectValidationLayers(requirement) {
  const layers = new Set();

  (requirement.validation || []).forEach(v => {
    const ref = (v.ref || '').toLowerCase();

    // Check API layer
    if (VALIDATION_LAYERS.API.patterns.some(p => p.test(ref))) {
      layers.add('API');
    }

    // Check UI layer
    if (VALIDATION_LAYERS.UI.patterns.some(p => p.test(ref))) {
      layers.add('UI');
    }

    // Check E2E layer
    if (VALIDATION_LAYERS.E2E.patterns.some(p => p.test(ref))) {
      layers.add('E2E');
    }

    // Check manual layer
    if (v.type === 'manual') {
      layers.add('MANUAL');
    }
  });

  return layers;
}
```

#### 2.4 Derive Criticality from Operational Targets

**BREAKING CHANGE**: Remove `criticality` field from requirements entirely.

**Rationale**:
- Operational targets already define priority (OT-P0-001, OT-P1-002, OT-P2-003)
- Having a separate `criticality` field on requirements creates:
  - **Redundancy**: Two sources of truth for importance
  - **Gaming vector**: Agents can mark critical features as P2 to avoid diversity requirements
  - **Misalignment**: Requirement priority can diverge from PRD operational target priority
- **Solution**: Derive criticality from `prd_ref` field (OT-P0-XXX → P0 requirement)

**Implementation**: Add helper function to extract criticality from prd_ref

```javascript
/**
 * Derive requirement criticality from operational target reference
 * @param {object} requirement - Requirement object
 * @returns {string} Criticality level (P0, P1, P2)
 */
function deriveRequirementCriticality(requirement) {
  const prdRef = requirement.prd_ref || '';
  const match = prdRef.match(/OT-([Pp][0-2])-\d{3}/);

  if (match) {
    return match[1].toUpperCase();  // Extract P0, P1, or P2 from OT-P0-001
  }

  // Default to P2 if no operational target reference
  return 'P2';
}
```

**Migration**: All requirement files must:
1. Remove `"criticality"` field from requirement objects
2. Ensure `"prd_ref"` field references correct operational target (OT-P0-XXX, OT-P1-XXX, OT-P2-XXX)
3. Update auto-sync to use derived criticality

#### 2.5 Requirement Pass Calculation Enhancement

**File**: `scripts/scenarios/lib/completeness-data.js`

**Function**: `calculateRequirementPass(requirements, syncData, scenarioRoot)`

**Enhancement Strategy**:
- **P0/P1 requirements**: Require ≥2 AUTOMATED layers within available components
- **P2 requirements**: Allow 1 AUTOMATED layer (lenient)
- **Component-aware**: Don't require UI tests for API-only scenarios
- **Manual validations**: Track separately, DON'T count toward diversity

**Rationale**: Critical requirements (P0/P1) should have robust multi-layer AUTOMATED validation. Manual validations are temporary measures before automation. Diversity requirements should match available scenario components.

```javascript
function calculateRequirementPass(requirements, syncData, scenarioRoot) {
  let passing = 0;
  let total = requirements.length;
  const syncMetadata = syncData.requirements || syncData;

  // Detect available components once for the scenario
  const scenarioComponents = detectScenarioComponents(scenarioRoot);

  for (const req of requirements) {
    // DERIVE criticality from prd_ref (no longer use req.criticality)
    const criticality = deriveRequirementCriticality(req);
    const status = req.status || 'pending';
    const reqMeta = syncMetadata[req.id];

    // Detect validation layers (returns {automated: Set, has_manual: boolean})
    const layerAnalysis = detectValidationLayers(req, scenarioRoot);

    // Filter to only applicable AUTOMATED layers based on scenario components
    const applicableLayers = new Set();
    if (scenarioComponents.has('API') && layerAnalysis.automated.has('API')) {
      applicableLayers.add('API');
    }
    if (scenarioComponents.has('UI') && layerAnalysis.automated.has('UI')) {
      applicableLayers.add('UI');
    }
    // E2E is always applicable (AUTOMATED layer)
    if (layerAnalysis.automated.has('E2E')) {
      applicableLayers.add('E2E');
    }
    // MANUAL is tracked separately, NOT counted toward diversity

    // Minimum AUTOMATED layer requirement based on criticality and available components
    // For API-only: Need API test + E2E
    // For UI-only: Need UI test + E2E
    // For Full-stack: Need (API + UI) OR (API + E2E) OR (UI + E2E)
    const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;

    // Requirement passes if:
    // 1. Status indicates completion (validated/implemented/complete)
    // 2. Sync metadata shows completion
    // 3. Has sufficient AUTOMATED validation layer diversity (applicable layers only)
    const statusPasses = (
      status === 'validated' ||
      status === 'implemented' ||
      status === 'complete'
    );

    const syncPasses = reqMeta && (
      reqMeta.status === 'complete' ||
      reqMeta.status === 'validated' ||
      (reqMeta.sync_metadata && reqMeta.sync_metadata.all_tests_passing === true)
    );

    const diversityPasses = applicableLayers.size >= minLayers;

    // All three conditions must be true
    if ((statusPasses || syncPasses) && diversityPasses) {
      passing++;
    }
  }

  return { total, passing };
}
```

**Migration Strategy**:
- Add `diversity_pass` and `applicable_layers` fields to requirement metadata during sync
- Surface diversity failures in `vrooli scenario completeness` output
- Provide component-aware recommendations (e.g., "Add E2E tests for REQ-001" for API-only scenarios)

**Component-Aware Diversity Examples**:
```javascript
// Full-stack scenario (API + UI components)
// P0 requirement needs ≥2 AUTOMATED layers from: [API, UI, E2E]
// (MANUAL doesn't count toward diversity)
✅ API + UI
✅ API + E2E
✅ UI + E2E
❌ API only (insufficient diversity)
❌ API + MANUAL (manual doesn't count - needs 2 automated layers)

// API-only scenario (no UI component)
// P0 requirement needs ≥2 AUTOMATED layers from: [API, E2E]
// (MANUAL doesn't count, UI tests ignored - no UI component exists)
✅ API + E2E
❌ API only (insufficient diversity)
❌ API + MANUAL (manual doesn't count - needs 2 automated layers)
❌ API + UI (UI tests ignored - no UI component exists)

// UI-only scenario (no API component)
// P0 requirement needs ≥2 AUTOMATED layers from: [UI, E2E]
// (MANUAL doesn't count, API tests ignored - no API component exists)
✅ UI + E2E
❌ UI only (insufficient diversity)
❌ UI + MANUAL (manual doesn't count - needs 2 automated layers)
```

---

### Phase 2.5: Basic Test Quality Detection (High Priority)

**Goal**: Prevent superficial test files that exist only to satisfy diversity requirements

**Rationale**: Agents could game multi-layer validation by creating empty or trivial test files:
```go
// api/placeholder_test.go (5 lines, no actual tests)
package api
import "testing"
// File exists, counts as "API layer" validation
```

This bypasses the intent of diversity requirements. We need basic quality heuristics to catch blatant gaming.

#### 2.5.1 Test Quality Analysis Function

**File**: `scripts/scenarios/lib/completeness-data.js` (new helper)

```javascript
/**
 * Analyze test file quality using basic heuristics
 * @param {string} testFilePath - Relative path to test file
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Quality analysis results
 */
function analyzeTestFileQuality(testFilePath, scenarioRoot) {
  const fullPath = path.join(scenarioRoot, testFilePath);

  if (!fs.existsSync(fullPath)) {
    return {
      exists: false,
      is_meaningful: false,
      reason: 'file_not_found'
    };
  }

  const content = fs.readFileSync(fullPath, 'utf8');
  const lines = content.split('\n');
  const nonEmptyLines = lines.filter(l => l.trim().length > 0);

  // Remove comment-only lines
  const nonCommentLines = nonEmptyLines.filter(l => {
    const t = l.trim();
    return !t.startsWith('//') &&
           !t.startsWith('/*') &&
           !t.startsWith('*') &&
           !t.startsWith('#');  // Python/shell comments
  });

  // Count test functions
  const testFunctionMatches = content.match(/func Test|@test|test\(|it\(|describe\(|def test_/gi);
  const testFunctionCount = testFunctionMatches ? testFunctionMatches.length : 0;

  // Count assertions
  const assertionMatches = content.match(/assert|expect|require|Should|Equal|Contains|Error|True|False|toBe|toHaveBeenCalled/gi);
  const assertionCount = assertionMatches ? assertionMatches.length : 0;
  const assertionDensity = nonCommentLines.length > 0 ? assertionCount / nonCommentLines.length : 0;

  // Quality heuristics (ENHANCED)
  const hasMinimumCode = nonCommentLines.length >= 20;  // At least 20 LOC
  const hasAssertions = assertionCount > 0;
  const hasMultipleTestFunctions = testFunctionCount >= 3;  // Require ≥3 test functions
  const hasGoodAssertionDensity = assertionDensity >= 0.1;  // ≥1 assertion per 10 LOC
  const hasTestFunctions = testFunctionCount > 0;

  // Calculate quality score (0-5, need ≥4 to pass)
  const qualityScore =
    (hasMinimumCode ? 1 : 0) +
    (hasAssertions ? 1 : 0) +
    (hasTestFunctions ? 1 : 0) +
    (hasMultipleTestFunctions ? 1 : 0) +
    (hasGoodAssertionDensity ? 1 : 0);

  return {
    exists: true,
    loc: nonCommentLines.length,
    has_assertions: hasAssertions,
    has_test_functions: hasTestFunctions,
    test_function_count: testFunctionCount,
    assertion_count: assertionCount,
    assertion_density: assertionDensity,
    is_meaningful: qualityScore >= 4,  // Need 4+ indicators (was 2)
    quality_score: qualityScore,
    reason: qualityScore < 4 ? 'insufficient_quality' : 'ok'
  };
}
```

#### 2.5.2 Playbook Quality Analysis Function

**File**: `scripts/scenarios/lib/completeness-data.js` (new helper)

```javascript
/**
 * Analyze e2e playbook quality
 * @param {string} playbookPath - Relative path to playbook file
 * @param {string} scenarioRoot - Scenario root directory
 * @returns {object} Quality analysis results
 */
function analyzePlaybookQuality(playbookPath, scenarioRoot) {
  const fullPath = path.join(scenarioRoot, playbookPath);

  if (!fs.existsSync(fullPath)) {
    return { exists: false, is_meaningful: false, reason: 'file_not_found' };
  }

  try {
    const content = fs.readFileSync(fullPath, 'utf8');
    const parsed = JSON.parse(content);

    const hasSteps = Array.isArray(parsed.steps) && parsed.steps.length > 0;
    const stepCount = parsed.steps ? parsed.steps.length : 0;
    const hasActions = parsed.steps && parsed.steps.some(s => s.action || s.type);
    const fileSize = Buffer.byteLength(content, 'utf8');

    return {
      exists: true,
      step_count: stepCount,
      has_actions: hasActions,
      file_size: fileSize,
      is_meaningful: hasSteps && hasActions && fileSize >= 100,  // ≥100 bytes, has steps with actions
      reason: !hasSteps ? 'no_steps' : !hasActions ? 'no_actions' : 'ok'
    };
  } catch (error) {
    return { exists: true, is_meaningful: false, reason: 'parse_error' };
  }
}
```

#### 2.5.3 Integration with Validation Layer Detection

**File**: `scripts/scenarios/lib/completeness-data.js`

**Function**: `detectValidationLayers(requirement, scenarioRoot)` (add scenarioRoot param)

**Enhancement**: Filter out low-quality test refs AND manual validations from layer detection

```javascript
function detectValidationLayers(requirement, scenarioRoot) {
  const automatedLayers = new Set();
  const hasManual = (requirement.validation || []).some(v => v.type === 'manual');

  (requirement.validation || []).forEach(v => {
    const ref = (v.ref || '').toLowerCase();

    // IMPORTANT: Skip manual validations for layer diversity
    // Manual validations are temporary measures, not automated test evidence
    if (v.type === 'manual') {
      return;
    }

    // Skip validations that don't reference actual test files
    if (!ref && !v.workflow_id) {
      return;
    }

    // Check quality for test type validations with file refs
    if (v.type === 'test' && ref) {
      const quality = analyzeTestFileQuality(ref, scenarioRoot);
      if (!quality.is_meaningful) {
        // Don't count low-quality tests toward layer diversity
        console.warn(`Validation ref excluded (low quality): ${ref} - ${quality.reason}`);
        return;
      }
    }

    // Check quality for automation type validations (playbooks)
    if (v.type === 'automation' && ref && ref.match(/\.(json|yaml)$/)) {
      const quality = analyzePlaybookQuality(ref, scenarioRoot);
      if (!quality.is_meaningful) {
        console.warn(`Playbook ref excluded (low quality): ${ref} - ${quality.reason}`);
        return;
      }
    }

    // Check API layer
    if (VALIDATION_LAYERS.API.patterns.some(p => p.test(ref))) {
      automatedLayers.add('API');
    }

    // Check UI layer
    if (VALIDATION_LAYERS.UI.patterns.some(p => p.test(ref))) {
      automatedLayers.add('UI');
    }

    // Check E2E layer
    if (VALIDATION_LAYERS.E2E.patterns.some(p => p.test(ref))) {
      automatedLayers.add('E2E');
    }
  });

  return {
    automated: automatedLayers,  // Use this for diversity requirement
    has_manual: hasManual        // Track separately, don't count toward diversity
  };
}
```

#### 2.5.4 Quality Warning in Gaming Detection

**File**: `scripts/scenarios/lib/gaming-detection.js`

Add low-quality test detection to `detectGamingPatterns()`:

```javascript
// Pattern 6: Low-quality test files
const lowQualityTests = [];
requirements.forEach(req => {
  (req.validation || []).forEach(v => {
    if (v.type === 'test' && v.ref) {
      const quality = analyzeTestFileQuality(v.ref, scenarioRoot);
      if (quality.exists && !quality.is_meaningful) {
        lowQualityTests.push({
          requirement: req.id,
          ref: v.ref,
          quality_score: quality.quality_score,
          reason: quality.reason
        });
      }
    }
  });
});

if (lowQualityTests.length > 0) {
  patterns.low_quality_tests = {
    severity: 'medium',
    detected: true,
    count: lowQualityTests.length,
    files: lowQualityTests,
    message: `${lowQualityTests.length} test file(s) appear superficial (< 20 LOC, missing assertions, or no test functions)`,
    recommendation: 'Add meaningful test logic with assertions and edge case coverage'
  };
  warnings.push(patterns.low_quality_tests);
}
```

**Impact**:
- Superficial test files won't count toward layer diversity
- Warnings surfaced during completeness calculation
- No scoring penalty initially (Stage 1-2), just flagged
- Future: Apply small penalty in Stage 3 if pattern is widespread

---

### Phase 3: Duplicate Test Ref Detection (Medium Priority)

**Goal**: Penalize scenarios where one test file validates many requirements

#### 3.1 Duplicate Ref Analysis Function

**File**: `scripts/scenarios/lib/completeness-data.js` (new helper)

```javascript
/**
 * Analyze test ref usage across requirements
 * @param {array} requirements - Array of requirement objects
 * @returns {object} Analysis results with violations
 */
function analyzeTestRefUsage(requirements) {
  const refUsage = new Map();  // ref → [req_id, req_id, ...]

  requirements.forEach(req => {
    (req.validation || []).forEach(v => {
      const ref = v.ref || v.workflow_id;
      if (!ref) return;

      if (!refUsage.has(ref)) {
        refUsage.set(ref, []);
      }
      refUsage.get(ref).push(req.id);
    });
  });

  // Detect violations (1 test file validates ≥4 requirements)
  const violations = [];
  refUsage.forEach((reqIds, ref) => {
    if (reqIds.length >= 4) {  // Threshold: 4+ requirements per test file
      violations.push({
        test_ref: ref,
        shared_by: reqIds,
        count: reqIds.length,
        severity: reqIds.length >= 6 ? 'high' : 'medium'
      });
    }
  });

  return {
    total_refs: refUsage.size,
    total_requirements: requirements.length,
    average_reqs_per_ref: requirements.length / Math.max(refUsage.size, 1),
    violations,
    duplicate_ratio: violations.length / Math.max(requirements.length, 1)
  };
}
```

#### 3.2 Completeness Score Penalty

**File**: `scripts/scenarios/lib/completeness.js`

**Function**: `calculateQualityScore(metrics)`

**Add diversity penalty**:

```javascript
function calculateQualityScore(metrics) {
  const reqPassRate = metrics.requirements.total > 0
    ? metrics.requirements.passing / metrics.requirements.total
    : 0;
  const targetPassRate = metrics.targets.total > 0
    ? metrics.targets.passing / metrics.targets.total
    : 0;
  const testPassRate = metrics.tests.total > 0
    ? metrics.tests.passing / metrics.tests.total
    : 0;

  let reqPoints = Math.round(reqPassRate * 20);
  const targetPoints = Math.round(targetPassRate * 15);
  const testPoints = Math.round(testPassRate * 15);

  // NEW: Apply penalty for excessive test ref duplication
  if (metrics.duplicate_analysis && metrics.duplicate_analysis.violations.length > 0) {
    const duplicateRatio = metrics.duplicate_analysis.duplicate_ratio;

    // Penalty: -1 point per 10% of requirements validated by duplicate refs
    const penalty = Math.min(Math.round(duplicateRatio * 10), 5);  // Max 5 point penalty
    reqPoints = Math.max(reqPoints - penalty, 0);
  }

  return {
    score: reqPoints + targetPoints + testPoints,
    max: 50,
    requirement_pass_rate: {
      passing: metrics.requirements.passing,
      total: metrics.requirements.total,
      rate: reqPassRate,
      points: reqPoints,
      duplicate_penalty: metrics.duplicate_analysis ? penalty : 0
    },
    // ... rest of breakdown
  };
}
```

**Impact**: deployment-manager would lose 3-5 points from quality score for excessive duplication.

---

### Phase 4: Operational Target Grouping Validation (Medium Priority)

**Goal**: Detect suspicious 1:1 operational target-to-requirement mappings

#### 4.1 Dynamic Threshold Calculation

**Approach**: Calculate acceptable 1:1 ratio based on total OT count

```
Acceptable 1:1 ratio = min(0.2, 5 / total_targets)

Examples:
- 5 targets  → max 1 allowed (20%)
- 10 targets → max 2 allowed (20%)
- 20 targets → max 4 allowed (20%)
- 40 targets → max 8 allowed (20%)
- 50 targets → max 10 allowed (20%)
```

**Rationale**:
- Scenarios with few targets (5-10): Targets represent major features → expect ≥3 requirements each
- Scenarios with many targets (30+): Targets are more granular → some 1:1 mappings acceptable
- Cap at 20% to prevent full 1:1 inflation regardless of scale

#### 4.2 Target Grouping Analysis Function

**File**: `scripts/scenarios/lib/completeness-data.js` (new helper)

```javascript
/**
 * Analyze operational target grouping patterns
 * @param {array} targets - Array of operational target objects
 * @param {array} requirements - Array of requirement objects
 * @returns {object} Analysis results
 */
function analyzeTargetGrouping(targets, requirements) {
  const targetMap = new Map();  // OT-XXX → [req_id, ...]

  // Build target-to-requirements mapping
  requirements.forEach(req => {
    const match = (req.prd_ref || '').match(/OT-[Pp][0-2]-\d{3}/);
    if (match) {
      const targetId = match[0].toUpperCase();
      if (!targetMap.has(targetId)) {
        targetMap.set(targetId, []);
      }
      targetMap.get(targetId).push(req.id);
    }
  });

  // Count 1:1 mappings
  const oneToOneMappings = Array.from(targetMap.entries())
    .filter(([targetId, reqIds]) => reqIds.length === 1);

  const totalTargets = targetMap.size;
  const oneToOneCount = oneToOneMappings.length;
  const oneToOneRatio = totalTargets > 0 ? oneToOneCount / totalTargets : 0;

  // Dynamic threshold: min(20%, 5/total_targets)
  const acceptableRatio = Math.min(0.2, totalTargets > 0 ? 5 / totalTargets : 0);

  const violations = [];
  if (oneToOneRatio > acceptableRatio) {
    violations.push({
      type: 'excessive_one_to_one_mappings',
      current_ratio: oneToOneRatio,
      acceptable_ratio: acceptableRatio,
      one_to_one_count: oneToOneCount,
      total_targets: totalTargets,
      message: `${Math.round(oneToOneRatio * 100)}% of targets have 1:1 requirement mapping (max ${Math.round(acceptableRatio * 100)}% recommended)`,
      targets: oneToOneMappings.map(([targetId, reqIds]) => ({
        target: targetId,
        requirement: reqIds[0]
      }))
    });
  }

  return {
    total_targets: totalTargets,
    one_to_one_count: oneToOneCount,
    one_to_one_ratio: oneToOneRatio,
    acceptable_ratio: acceptableRatio,
    average_requirements_per_target: requirements.length / Math.max(totalTargets, 1),
    violations
  };
}
```

#### 4.3 Integration with lint-prd Command

**File**: `scripts/requirements/lint-prd.js`

**Enhancement**: Add grouping validation to existing linter

```javascript
// Existing: Checks PRD ↔ requirements bidirectional mapping
// NEW: Add target grouping validation

const groupingAnalysis = analyzeTargetGrouping(targets, requirements);

if (groupingAnalysis.violations.length > 0) {
  console.error('\n❌ Operational Target Grouping Issues:');
  groupingAnalysis.violations.forEach(v => {
    console.error(`  • ${v.message}`);
    console.error(`    Suspicious targets: ${v.targets.map(t => t.target).join(', ')}`);
  });
  exitCode = 1;
}
```

**Usage**: `vrooli scenario requirements lint-prd deployment-manager`

---

### Phase 5: Gaming Pattern Detection & Warnings (High Priority)

**Goal**: Centralized detection and warning system for all gaming patterns

#### 5.1 Unified Gaming Detection Module

**File**: `scripts/scenarios/lib/gaming-detection.js` (NEW)

```javascript
/**
 * Centralized gaming pattern detection
 * @module scenarios/lib/gaming-detection
 */

const { analyzeTestRefUsage, analyzeTargetGrouping, detectValidationLayers } = require('./completeness-data');

/**
 * Detect all gaming patterns in a scenario
 * @param {object} metrics - Completeness metrics
 * @param {array} requirements - Requirements array
 * @param {array} targets - Operational targets array
 * @returns {object} Detection results with warnings
 */
function detectGamingPatterns(metrics, requirements, targets) {
  const warnings = [];
  const patterns = {};

  // Pattern 1: Perfect 1:1 requirement-to-test ratio
  const testReqRatio = metrics.tests.total / Math.max(metrics.requirements.total, 1);
  if (Math.abs(testReqRatio - 1.0) < 0.1) {  // Within 10% of 1:1
    patterns.one_to_one_test_ratio = {
      severity: 'medium',
      detected: true,
      ratio: testReqRatio,
      message: `Suspicious 1:1 test-to-requirement ratio (${metrics.tests.total} tests for ${metrics.requirements.total} requirements). Expected: 1.5-2.0x ratio with diverse test sources.`,
      recommendation: 'Add multiple test types per requirement (API tests, UI tests, e2e automation)'
    };
    warnings.push(patterns.one_to_one_test_ratio);
  }

  // Pattern 2: Unsupported test directory usage
  const unsupportedRefCount = requirements.filter(r =>
    (r.validation || []).some(v => {
      const ref = (v.ref || '').toLowerCase();
      // Reject anything in test/ that's not test/playbooks/*.{json,yaml}
      return ref.startsWith('test/') &&
             !(ref.startsWith('test/playbooks/') && (ref.endsWith('.json') || ref.endsWith('.yaml')));
    })
  ).length;

  if (unsupportedRefCount > 0) {
    const ratio = unsupportedRefCount / Math.max(metrics.requirements.total, 1);

    // Build component-aware recommendation
    const scenarioComponents = detectScenarioComponents(scenarioRoot);
    const validSources = [];
    if (scenarioComponents.has('API')) {
      validSources.push('api/**/*_test.go (API unit tests)');
    }
    if (scenarioComponents.has('UI')) {
      validSources.push('ui/src/**/*.test.tsx (UI unit tests)');
    }
    validSources.push('test/playbooks/**/*.{json,yaml} (e2e automation)');

    patterns.unsupported_test_directory = {
      severity: ratio > 0.5 ? 'high' : 'medium',
      detected: true,
      count: unsupportedRefCount,
      total: metrics.requirements.total,
      ratio: ratio,
      message: `${unsupportedRefCount}/${metrics.requirements.total} requirements reference unsupported test/ directories. Valid sources for this scenario:\n${validSources.map(s => `  • ${s}`).join('\n')}`,
      recommendation: 'Move validation refs to supported test sources listed above'
    };
    warnings.push(patterns.unsupported_test_directory);
  }

  // Pattern 3: Duplicate test refs
  const refAnalysis = analyzeTestRefUsage(requirements);
  if (refAnalysis.violations.length > 0) {
    patterns.duplicate_test_refs = {
      severity: refAnalysis.violations.some(v => v.severity === 'high') ? 'high' : 'medium',
      detected: true,
      violations: refAnalysis.violations.length,
      worst_offender: refAnalysis.violations[0],
      message: `${refAnalysis.violations.length} test files validate ≥4 requirements each. Expected: unique tests per requirement.`,
      recommendation: 'Break monolithic test files into focused tests for each requirement'
    };
    warnings.push(patterns.duplicate_test_refs);
  }

  // Pattern 4: Excessive 1:1 target-to-requirement mappings
  const groupingAnalysis = analyzeTargetGrouping(targets, requirements);
  if (groupingAnalysis.violations.length > 0) {
    patterns.one_to_one_target_mapping = {
      severity: groupingAnalysis.one_to_one_ratio > 0.5 ? 'high' : 'medium',
      detected: true,
      ratio: groupingAnalysis.one_to_one_ratio,
      count: groupingAnalysis.one_to_one_count,
      message: `${Math.round(groupingAnalysis.one_to_one_ratio * 100)}% of operational targets have 1:1 requirement mapping (max ${Math.round(groupingAnalysis.acceptable_ratio * 100)}% recommended)`,
      recommendation: 'Group related requirements under shared operational targets from the PRD'
    };
    warnings.push(patterns.one_to_one_target_mapping);
  }

  // Pattern 5: Missing validation diversity
  const scenarioComponents = detectScenarioComponents(scenarioRoot);
  const diversityIssues = requirements.filter(req => {
    const layerAnalysis = detectValidationLayers(req, scenarioRoot);
    const criticality = deriveRequirementCriticality(req);  // Derive from prd_ref
    const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;

    // Filter to applicable automated layers only
    const applicableLayers = new Set();
    if (scenarioComponents.has('API') && layerAnalysis.automated.has('API')) {
      applicableLayers.add('API');
    }
    if (scenarioComponents.has('UI') && layerAnalysis.automated.has('UI')) {
      applicableLayers.add('UI');
    }
    if (layerAnalysis.automated.has('E2E')) {
      applicableLayers.add('E2E');
    }

    return applicableLayers.size < minLayers;
  });

  if (diversityIssues.length > 0) {
    patterns.missing_validation_diversity = {
      severity: 'high',
      detected: true,
      count: diversityIssues.length,
      total: requirements.length,
      message: `${diversityIssues.length} critical requirements (P0/P1) lack multi-layer AUTOMATED validation`,
      recommendation: 'Add automated validations across API, UI, and e2e layers for critical requirements (manual validations don\'t count toward diversity)'
    };
    warnings.push(patterns.missing_validation_diversity);
  }

  // Pattern 6: Excessive manual validation usage
  const totalValidations = requirements.reduce((sum, r) =>
    sum + (r.validation || []).length, 0
  );
  const manualValidations = requirements.reduce((sum, r) =>
    sum + (r.validation || []).filter(v => v.type === 'manual').length, 0
  );

  // Count manual validations for COMPLETE requirements (suspicious)
  const completeWithManual = requirements.filter(req => {
    const status = (req.status || '').toLowerCase();
    const isComplete = status === 'complete' || status === 'validated' || status === 'implemented';
    const hasManual = (req.validation || []).some(v => v.type === 'manual');
    const hasAutomated = (req.validation || []).some(v => v.type !== 'manual');

    // Only flag if COMPLETE with manual validation but NO automated tests
    return isComplete && hasManual && !hasAutomated;
  });

  const manualRatio = manualValidations / Math.max(totalValidations, 1);

  // Flag if: (a) >10% manual overall OR (b) ≥5 complete requirements with ONLY manual validation
  if (manualRatio > 0.10 || completeWithManual.length >= 5) {
    patterns.excessive_manual_validations = {
      severity: completeWithManual.length >= 10 ? 'high' : 'medium',
      detected: true,
      ratio: manualRatio,
      count: manualValidations,
      complete_with_manual: completeWithManual.length,
      message: manualRatio > 0.10
        ? `${Math.round(manualRatio * 100)}% of validations are manual (max 10% recommended). Manual validations are intended as temporary measures before automated tests are implemented.`
        : `${completeWithManual.length} requirements marked complete with ONLY manual validations. Automated tests should replace manual validations for completed requirements.`,
      recommendation: 'Replace manual validations with automated tests (API tests, UI tests, or e2e automation). Manual validations should be temporary measures for pending/in_progress requirements.'
    };
    warnings.push(patterns.excessive_manual_validations);
  }

  return {
    has_warnings: warnings.length > 0,
    warning_count: warnings.length,
    warnings,
    patterns,
    overall_severity: warnings.some(w => w.severity === 'high') ? 'high' : 'medium'
  };
}

module.exports = {
  detectGamingPatterns
};
```

#### 5.2 Integration with Completeness Score

**File**: `scripts/scenarios/lib/completeness-cli.js`

**Add warnings section to output**:

```javascript
const gamingDetection = require('./gaming-detection');

// After calculating completeness score
const gamingAnalysis = gamingDetection.detectGamingPatterns(
  metrics,
  requirements,
  operationalTargets
);

// Add to output
if (gamingAnalysis.has_warnings) {
  console.log('\n⚠️  Gaming Pattern Warnings:\n');
  gamingAnalysis.warnings.forEach(warning => {
    const icon = warning.severity === 'high' ? '🔴' : '🟡';
    console.log(`${icon} ${warning.message}`);
    console.log(`   💡 ${warning.recommendation}\n`);
  });
}
```

**JSON output** should include `gaming_analysis` field for programmatic access.

---

### Phase 6: Documentation Updates (Medium Priority)

**Goal**: Ensure documentation is accurate, organized, and prevents future gaming

#### 6.1 Document Review & Updates

**Files to review**:
1. `docs/testing/architecture/PHASED_TESTING.md`
2. `docs/testing/architecture/REQUIREMENT_FLOW.md`
3. `docs/testing/guides/requirement-tracking.md`
4. `docs/testing/reference/requirement-schema.md`

**Updates needed**:

##### PHASED_TESTING.md
- ✅ Clarify that test/phases/ scripts are **orchestration**, not validation evidence
- ✅ Add clear examples of valid vs. invalid validation refs
- ✅ Explain why test/cli/ should not be used for requirement validation
- ✅ Add section on validation diversity requirements

##### REQUIREMENT_FLOW.md
- ✅ Update diagrams to show multi-layer validation flow
- ✅ Add examples of proper requirement-to-test mapping (1:2 ratio)
- ✅ Clarify operational target grouping best practices
- ✅ Show how auto-sync detects and rejects invalid refs

##### requirement-tracking.md
- ✅ Add "Common Pitfalls" section with gaming patterns
- ✅ Provide browser-automation-studio as reference example
- ✅ Add validation checklist for requirement setup
- ✅ Update examples to show multi-layer validation

##### requirement-schema.md
- ✅ Document allowed validation ref patterns (with clear rationale)
- ✅ Add validation rules for `ref` field
- ✅ Clarify `type` field values and their meanings
- ✅ Add examples showing proper vs. improper validation structures
- ✅ Add component-aware validation examples (API-only, UI-only, full-stack)
- ✅ **Add directory structure diagram** showing valid vs. invalid test sources:

```
test/
├── phases/          ❌ Orchestration (not test sources)
├── cli/             ❌ CLI wrapper tests (not requirement validation)
├── unit/            ❌ Test runners (not test sources)
├── integration/     ❌ Test runners (not test sources)
└── playbooks/       ✅ E2e automation (actual test sources)

api/
└── **/*_test.go     ✅ API unit tests (actual test sources)

ui/src/
└── **/*.test.tsx    ✅ UI unit tests (actual test sources)
```

**Rationale for test/ directory restrictions**:
- `test/phases/`: Orchestration scripts that run tests, not test sources themselves
- `test/cli/`, `test/unit/`, `test/integration/`: Test runner scripts or CLI-specific tests
- Only `test/playbooks/` contains actual test sources (e2e automation workflows)
- Actual unit tests live alongside the code they test (api/, ui/src/)

#### 6.2 New Documentation

**File**: `docs/testing/guides/validation-best-practices.md` (NEW)

**Contents**:
- Valid vs. invalid validation sources (with examples)
- Multi-layer validation strategy (API + UI + e2e)
- Component-aware validation (API-only, UI-only, full-stack scenarios)
- **Quality over Quantity**: Meaningful tests vs. superficial checkbox tests
- Operational target grouping guidelines
- Test-to-requirement ratio recommendations
- How to avoid gaming patterns
- Browser-automation-studio as reference implementation

**Key Section: Quality Over Quantity**
```markdown
### Quality Over Quantity

Multi-layer validation is required, but the tests must be **meaningful**:

❌ **Bad**: Superficial tests that just exist
```json
// Empty test file with no logic
{"ref": "api/placeholder_test.go"}  // 5 lines, only imports, no assertions
```

✅ **Good**: Tests that actually validate the requirement
```json
// Comprehensive test validating business logic
{"ref": "api/dependency_analyzer_test.go"}  // 150 lines, 12 test cases, edge cases covered
```

**Key indicators of meaningful tests**:
- **Non-trivial**: ≥20 lines of code (shows actual test logic)
- **Multiple assertions**: Tests verify multiple aspects of behavior
- **Edge cases**: Tests cover happy path AND error conditions
- **Integration points**: Tests verify interactions with dependencies
- **Clear purpose**: Test names/descriptions explain what's being validated

**Example - Meaningful API Test**:
```go
// api/analyzer_test.go
func TestDependencyAnalyzer_RecursiveResolution(t *testing.T) {
    // Setup: Create test dependency graph
    analyzer := NewAnalyzer()

    // Test Case 1: Linear dependencies (A → B → C)
    assert.Equal(t, 3, len(analyzer.Resolve("A")))

    // Test Case 2: Circular dependency detection
    assert.Error(t, analyzer.Resolve("CircularA"))

    // Test Case 3: Missing dependency handling
    assert.Empty(t, analyzer.Resolve("NonExistent"))

    // Test Case 4: Performance (≤5s for depth 10)
    start := time.Now()
    analyzer.Resolve("DeepTree")
    assert.Less(t, time.Since(start), 5*time.Second)
}
```

**Red Flags for Superficial Tests**:
- Test file exists but has no test functions/cases
- Only tests the "happy path" (no error handling)
- No assertions (test just calls function and returns)
- Copy-pasted from another requirement without changes
```

**File**: `docs/testing/reference/gaming-prevention.md` (NEW)

**Contents**:
- List of detected gaming patterns
- Why each pattern is problematic
- How the system detects and prevents gaming
- Migration guide for existing scenarios
- FAQ for common questions

---

## Implementation Plan

### Phase Breakdown

| Phase | Priority | Estimated Time | Dependencies | Notes |
|-------|----------|----------------|--------------|-------|
| **Phase 1**: Validation Source Filtering | P0 | 1 day | None | Straightforward rejection logic |
| **Phase 2**: Validation Diversity + Components | P0 | 1.5 days | Phase 1 | Component detection adds complexity |
| **Phase 2.5**: Basic Test Quality Detection | P0 | 0.5 days | Phase 2 | Quality heuristics (LOC, assertions, test functions) |
| **Phase 3**: Duplicate Ref Detection | P1 | 0.5 days | Phase 1 | Simple ref counting |
| **Phase 4**: Target Grouping Validation | P1 | 0.5 days | None | Threshold calculation |
| **Phase 5**: Gaming Detection & Warnings | P0 | 1.5 days | All above | Integration testing on real scenarios |
| **Phase 6**: Documentation Updates | P1 | 2-2.5 days | Phase 5 | 4 doc updates + 2 new docs + quality section + diagrams |

**Total Estimated Time**: 8.5-9 days

**Realistic Timeline**: Plan for **10-12 days** to account for:
- Iteration based on Stage 1 detection results
- Edge cases discovered during real scenario testing
- Documentation review and refinement
- Buffer for unexpected complexity

### Rollout Strategy

#### Stage 1: Detection Only (Days 1-3)
- Implement all detection logic
- Output warnings but **don't change scores**
- Gather data on existing scenarios
- Validate detection accuracy
- Run on all scenarios to identify false positives

**Command**: `vrooli scenario completeness <name> --detect-gaming`

**Output**:
```
Score: 96/100 (Production Ready)
⚠️  Gaming patterns detected (detection mode - not affecting score):
  🔴 37/37 requirements reference unsupported test directories
  🟡 6 test files validate ≥4 requirements each
  🟡 100% of operational targets have 1:1 requirement mapping
```

#### Stage 2: Soft Enforcement (Days 4-10)
- Apply scoring penalties
- Generate detailed reports for affected scenarios
- Provide migration guidance
- Support agents in fixing issues
- Monitor ecosystem-manager agent responses

**Command**: `vrooli scenario completeness <name>` (penalties active)

**Output**:
```
Score: 68/100 (Mostly Complete) ← Reduced from 96/100

Quality: 35/50 (-15 from gaming penalties)
  ⚠️  Diversity penalty: -10 points (no multi-layer validation)
  ⚠️  Duplication penalty: -5 points (monolithic test files)

⚠️  Gaming Pattern Warnings:
  🔴 37/37 requirements reference unsupported test directories
     💡 Move validation refs to: api/**/*_test.go, ui/src/**/*.test.tsx

  🟡 6 test files validate ≥4 requirements each
     💡 Break monolithic test files into focused tests
```

#### Stage 3: Full Enforcement (Days 11-14+)
- Auto-sync rejects invalid refs during `vrooli scenario requirements sync`
- Completeness score reflects true validation quality
- Documentation updated with examples
- ecosystem-manager agents trained on new patterns
- Monitor for unintended consequences

**Timeline Rationale**: P0 priority issue requires faster resolution than 3+ weeks. Compressed timeline (10-14 days) maintains validation rigor while accelerating rollout.

---

## Testing & Validation

### Unit Tests

**File**: `scripts/scenarios/lib/__tests__/gaming-detection.test.js`

```javascript
describe('Gaming Pattern Detection', () => {
  test('detects 1:1 test-to-requirement ratio', () => {
    // Setup scenario with 37 requirements, 37 tests
    // Assert: warning generated with correct message
  });

  test('detects unsupported test directory usage', () => {
    // Setup scenario with test/cli/ refs
    // Assert: high severity warning, correct count
  });

  test('detects duplicate test refs', () => {
    // Setup scenario where 1 file validates 6 requirements
    // Assert: violation detected, worst offender identified
  });

  test('detects excessive 1:1 target mapping', () => {
    // Setup scenario with 1:1 OT-to-requirement mapping
    // Assert: violation based on dynamic threshold
  });

  test('detects missing validation diversity', () => {
    // Setup P0 requirement with only 1 validation layer
    // Assert: diversity warning generated
  });

  test('detects low-quality test files', () => {
    // Setup scenario with empty/trivial test files (<20 LOC, no assertions)
    // Assert: low quality warning, files excluded from layer detection
  });

  test('detects excessive manual validations', () => {
    // Setup scenario with >10% manual validations
    // Assert: manual validation warning with correct threshold
  });
});
```

**File**: `scripts/scenarios/lib/__tests__/completeness-data.test.js`

```javascript
describe('Component-Aware Requirement Pass Calculation', () => {
  test('API-only scenario: P0 req passes with API + E2E', () => {
    const scenarioRoot = createAPIOnlyFixture();  // Has api/, no ui/
    const req = {
      id: 'REQ-001',
      criticality: 'P0',
      status: 'complete',
      validation: [
        { type: 'test', ref: 'api/analyzer_test.go' },
        { type: 'automation', ref: 'test/playbooks/e2e.json' }
      ]
    };
    const result = calculateRequirementPass([req], {}, scenarioRoot);
    expect(result.passing).toBe(1);  // Should pass (2 applicable layers: API + E2E)
  });

  test('API-only scenario: P0 req with UI test ignored', () => {
    const scenarioRoot = createAPIOnlyFixture();
    const req = {
      id: 'REQ-001',
      criticality: 'P0',
      status: 'complete',
      validation: [
        { type: 'test', ref: 'api/analyzer_test.go' },
        { type: 'test', ref: 'ui/src/fake.test.tsx' }  // UI test when no UI component
      ]
    };
    const result = calculateRequirementPass([req], {}, scenarioRoot);
    expect(result.passing).toBe(0);  // Should fail (only 1 applicable layer: API)
  });

  test('Full-stack scenario: P0 req passes with API + UI', () => {
    const scenarioRoot = createFullStackFixture();  // Has api/ and ui/
    const req = {
      id: 'REQ-001',
      criticality: 'P0',
      status: 'complete',
      validation: [
        { type: 'test', ref: 'api/test.go' },
        { type: 'test', ref: 'ui/src/test.tsx' }
      ]
    };
    const result = calculateRequirementPass([req], {}, scenarioRoot);
    expect(result.passing).toBe(1);  // Should pass (2 applicable layers: API + UI)
  });

  test('Low-quality test file excluded from layers', () => {
    const scenarioRoot = createFullStackFixture();
    const req = {
      id: 'REQ-001',
      criticality: 'P0',
      status: 'complete',
      validation: [
        { type: 'test', ref: 'api/placeholder_test.go' },  // 5 lines, no tests
        { type: 'test', ref: 'ui/src/test.tsx' }
      ]
    };

    // Mock low-quality API test
    fs.writeFileSync(
      path.join(scenarioRoot, 'api/placeholder_test.go'),
      'package api\nimport "testing"\n// Empty\n'
    );

    const result = calculateRequirementPass([req], {}, scenarioRoot);
    expect(result.passing).toBe(0);  // Should fail (API test excluded, only 1 layer: UI)
  });
});
```

### Integration Tests

**File**: `scenarios/__test-fixtures__/gaming-patterns/`

Create fixture scenarios demonstrating each gaming pattern:
- `fixture-cli-abuse/` - Uses test/cli/ refs
- `fixture-duplicate-refs/` - One file validates many requirements
- `fixture-one-to-one/` - Perfect 1:1:1 mapping
- `fixture-no-diversity/` - Single-layer validation only

**Test**: Run completeness calculation on fixtures, assert expected warnings.

### Real Scenario Validation

**Before rollout**:
1. Run detection on all scenarios: `for s in scenarios/*/; do vrooli scenario completeness $(basename $s) --detect-gaming; done`
2. Categorize scenarios by gaming pattern prevalence
3. Validate against browser-automation-studio (should have zero warnings)
4. Identify false positives and refine thresholds

**Expected results**:
- browser-automation-studio: ✅ 0 warnings
- deployment-manager: ⚠️  4 warnings (all patterns)
- landing-manager: ⚠️  3 warnings (no diversity, duplicate refs, unsupported dirs)

---

## Acceptance Criteria

### Phase 1: Validation Source Filtering
- [ ] Any test/ refs (except test/playbooks/*.{json,yaml}) rejected during auto-sync
- [ ] test/playbooks/**/*.{json,yaml} refs accepted
- [ ] API test refs (api/**/*_test.go) accepted
- [ ] UI test refs (ui/src/**/*.test.tsx) accepted
- [ ] Warning logged for each rejected ref with clear rationale
- [ ] browser-automation-studio: all validations still accepted
- [ ] deployment-manager: 37 validations rejected

### Phase 2.5: Enhanced Test Quality Detection
- [ ] analyzeTestFileQuality() detects files with <20 LOC, no assertions, <3 test functions, or low assertion density
- [ ] Quality score requires ≥4 of 5 indicators (was 2 of 3)
- [ ] analyzePlaybookQuality() detects empty/trivial playbooks (<100 bytes, no steps, no actions)
- [ ] Low-quality tests excluded from layer diversity counting
- [ ] Low-quality playbooks excluded from E2E layer counting
- [ ] Quality warnings surfaced in gaming detection
- [ ] browser-automation-studio: no quality warnings
- [ ] Empty/trivial test files and playbooks flagged correctly

### Phase 2: Validation Diversity + Criticality Derivation
- [ ] deriveRequirementCriticality() correctly extracts P0/P1/P2 from prd_ref (OT-P0-XXX format)
- [ ] Requirements with no prd_ref default to P2
- [ ] detectScenarioComponents() correctly identifies API and UI components
- [ ] detectValidationLayers() returns {automated: Set, has_manual: boolean}
- [ ] MANUAL validations excluded from automated layer count
- [ ] Component-aware diversity: Only requires tests for existing components
- [ ] P0/P1 requirements require ≥2 AUTOMATED layers (from applicable layers) to pass
- [ ] P2 requirements require ≥1 AUTOMATED layer to pass
- [ ] Completeness breakdown shows diversity_pass and applicable_layers fields
- [ ] Recommendations are component-aware (don't suggest UI tests for API-only scenarios)
- [ ] API-only scenario: Requires API + E2E (manual doesn't count)
- [ ] Full-stack scenario: Can satisfy with API + UI OR API + E2E OR UI + E2E (manual doesn't count)

### Phase 3: Duplicate Ref Detection
- [ ] analyzeTestRefUsage() detects violations (≥4 requirements per file)
- [ ] Quality score penalty applied (max -5 points)
- [ ] Breakdown shows duplicate_penalty field
- [ ] Violations listed with test file and affected requirements

### Phase 4: Target Grouping Validation
- [ ] analyzeTargetGrouping() calculates dynamic threshold correctly
- [ ] 1:1 mappings flagged when exceeding threshold
- [ ] vrooli scenario requirements lint-prd shows grouping warnings
- [ ] Recommendations provided for suspicious targets

### Phase 5: Gaming Detection
- [ ] All 7 gaming patterns detected correctly (including quality + manual)
- [ ] Warnings displayed with severity levels (high/medium)
- [ ] Recommendations actionable, specific, and component-aware
- [ ] Unsupported directory warnings show valid sources for scenario's components
- [ ] Manual validation warnings check both ratio (>10%) and complete-with-manual count (≥5)
- [ ] Manual validation warnings explain temporary nature of manual tests
- [ ] JSON output includes gaming_analysis field
- [ ] No duplicate detection code (reuses helpers)

### Phase 6: Documentation
- [ ] All 4 existing docs updated with gaming prevention guidance
- [ ] Component-aware validation examples added to requirement-schema.md
- [ ] validation-best-practices.md created with clear examples
- [ ] "Quality over Quantity" section added with meaningful test indicators
- [ ] gaming-prevention.md created with pattern catalog
- [ ] browser-automation-studio referenced as gold standard
- [ ] Migration guide provided for existing scenarios

### Overall System
- [ ] browser-automation-studio score unchanged (62/100 ±2)
- [ ] deployment-manager score reduced (96 → 65-70/100)
- [ ] landing-manager score reduced (similar magnitude)
- [ ] API-only scenarios not penalized for missing UI tests
- [ ] Scenarios with low manual validation usage (<10%) not flagged
- [ ] Complete requirements with manual+automated validation not flagged
- [ ] Zero false positives on properly-structured scenarios
- [ ] Clear, actionable, component-aware warnings for all gaming patterns
- [ ] Low-quality test files excluded from layer diversity
- [ ] Low-quality playbooks excluded from E2E layer
- [ ] Manual validations don't count toward diversity requirement
- [ ] Criticality derived from prd_ref (OT-P0-XXX format)
- [ ] All requirement files have criticality field removed
- [ ] ecosystem-manager agents can understand and avoid patterns

---

## Migration Guide for Existing Scenarios

### For Agents (ecosystem-manager)

**When you see warnings**, follow these steps:

#### Warning: "Requirements reference unsupported test directories"
**Action**:
1. Remove validation refs pointing to anywhere in test/ except test/playbooks/
2. Add proper validation refs:
   - API tests: `api/**/*_test.go`
   - UI tests: `ui/src/**/*.test.tsx`
   - E2e: `test/playbooks/**/*.{json,yaml}`
3. Run `vrooli scenario requirements sync <name>` to update

**Example fix**:
```json
// BEFORE (invalid)
{
  "id": "DM-P0-001",
  "prd_ref": "OT-P0-001",
  "criticality": "P0",  // ← REMOVE THIS (redundant with prd_ref)
  "validation": [
    {"type": "test", "ref": "test/cli/dependency-analysis.bats"}  // ← Invalid ref
  ]
}

// AFTER (valid)
{
  "id": "DM-P0-001",
  "prd_ref": "OT-P0-001",  // ← P0 criticality derived from this
  // NO criticality field (removed)
  "validation": [
    {"type": "test", "ref": "api/analyzer_test.go", "phase": "unit"},
    {"type": "test", "ref": "ui/src/components/DependencyGraph.test.tsx", "phase": "unit"},
    {"type": "automation", "ref": "test/playbooks/capabilities/dependency-viz/tree-view.json", "phase": "integration"}
  ]
}
```

#### Warning: "Missing multi-layer validation"
**Action**:
1. Check requirement's operational target (prd_ref: OT-P0-XXX → P0 criticality)
2. P0/P1 requirements need ≥2 AUTOMATED layers
3. Check scenario components (API, UI, or both)
4. Add AUTOMATED validations for missing layers based on available components:
   - **Full-stack scenario**: API test + UI test OR API test + e2e OR UI test + e2e
   - **API-only scenario**: API test + e2e (UI tests not applicable)
   - **UI-only scenario**: UI test + e2e (API tests not applicable)
5. **IMPORTANT**: Manual validations DON'T count toward diversity

**Example - API-only scenario**:
```json
// Scenario has api/ directory but no ui/ directory
{
  "id": "DM-P0-001",
  "prd_ref": "OT-P0-001",  // ← P0 criticality derived from operational target
  "validation": [
    {"type": "test", "ref": "api/analyzer_test.go", "phase": "unit"},
    {"type": "automation", "ref": "test/playbooks/dependency-viz/analyze.json", "phase": "integration"}
  ]
  // ✅ Passes: 2 automated layers (API + E2E) for API-only scenario
}
```

**Example - What NOT to do**:
```json
{
  "id": "DM-P0-001",
  "prd_ref": "OT-P0-001",
  "validation": [
    {"type": "test", "ref": "api/analyzer_test.go", "phase": "unit"},
    {"type": "manual", "notes": "Manually verified"}  // ← DOESN'T count toward diversity
  ]
  // ❌ Fails: Only 1 automated layer (API). Manual doesn't count.
}
```

#### Warning: "Duplicate test refs"
**Action**:
1. Break monolithic test file into focused tests
2. Create separate test files for each requirement or logical group
3. Update validation refs to point to specific test files

#### Warning: "Excessive 1:1 target mapping"
**Action**:
1. Review PRD operational targets
2. Group related requirements under shared targets
3. Update prd_ref fields to reference shared OT-P0-XXX targets

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **False positives** on legitimate scenarios | High - Could block valid work | Stage 1 (detection only) validates accuracy before enforcement |
| **Agents add superficial tests** to game diversity | High - Tests exist but don't validate | Phase 2.5 quality detection (LOC, assertions, test functions), excludes low-quality files from layer counting |
| **API-only scenarios penalized** for missing UI tests | High - Unfair scoring | Component detection (Phase 2) makes diversity requirements component-aware |
| **Manual validation flagged inappropriately** | Medium - Small amounts are legitimate | Low threshold (10%), check status (pending/in_progress vs. complete), focus on complete requirements |
| **Resistance from agents** to fixing issues | Medium - Slows progress | Provide clear migration guide, actionable recommendations |
| **Breaking changes** to existing workflows | High - Disrupts current scenarios | Phased rollout, soft enforcement period, compressed to 10-14 days |
| **Documentation gaps** causing confusion | Medium - Agents repeat mistakes | Comprehensive docs with examples, FAQ section, component-aware guidance, directory diagrams |
| **Threshold tuning** needed for edge cases | Low - Some scenarios don't fit model | Gather feedback during Stage 1-2, adjust thresholds before Stage 3 |

---

## Success Metrics

### Quantitative
- ✅ browser-automation-studio completeness score unchanged (±5%)
- ✅ deployment-manager score reduced by 20-30 points (reflects true quality)
- ✅ Zero gaming warnings on browser-automation-studio
- ✅ 90%+ of gamed scenarios show ≥3 gaming pattern warnings
- ✅ Validation diversity compliance: 80%+ of P0/P1 requirements have ≥2 layers
- ✅ Low-quality test files correctly identified and excluded (<5% false positive rate)
- ✅ Manual validation warnings: <5% false positives on legitimate pending work

### Qualitative
- ✅ Agents understand and can fix gaming warnings
- ✅ New scenarios follow best practices from day 1
- ✅ ecosystem-manager agents produce accurate completeness assessments
- ✅ Documentation clearly explains validation requirements
- ✅ Community feedback positive on fairness and clarity

---

## Next Steps

1. ✅ **Plan reviewed and updated** - Addressed critical gaps:
   - Added Phase 2.5: Basic Test Quality Detection (prevents superficial file gaming)
   - Added manual validation abuse detection (10% threshold, status-aware)
   - Enhanced Phase 1 rationale with directory structure diagram
   - Updated Phase 6 timeline (1.5 → 2-2.5 days for realistic documentation effort)
   - Added component-aware test coverage examples
   - Compressed rollout timeline (3+ weeks → 10-14 days for P0 priority)
2. **Validate approach** with sample implementation on deployment-manager
3. **Begin Phase 1** implementation (validation source filtering)
4. **Test on fixtures** to verify detection accuracy
5. **Iterate based on feedback** before full rollout

## Key Improvements Made

### 0. Remove Criticality Field from Requirements (Phase 2) - **BREAKING CHANGE**
- Criticality now derived from operational target reference (prd_ref: OT-P0-XXX → P0)
- Eliminates redundancy (two sources of truth for importance)
- Prevents gaming vector (marking critical features as P2 to avoid diversity)
- Ensures requirement priority aligns with PRD operational target priority
- Simplifies requirement schema and validation logic

### 1. Component-Aware Validation (Phase 2)
- Detects whether scenario has API, UI, or both components
- Diversity requirements adapt to available components
- API-only scenarios not penalized for missing UI tests
- Full-stack scenarios can satisfy diversity with any 2 layers

### 2. Enhanced Test Quality Detection (Phase 2.5) - **ENHANCED**
- **Stronger heuristics**: Requires ≥4 of 5 quality indicators (was 2 of 3)
  - Minimum code (≥20 LOC)
  - Has assertions
  - Has test functions
  - Multiple test functions (≥3)
  - Good assertion density (≥0.1 assertions per LOC)
- **Playbook quality checks**: Detects empty/trivial e2e playbooks
- Low-quality tests excluded from layer diversity counting
- Low-quality playbooks excluded from E2E layer counting
- Prevents agents from gaming diversity with superficial files
- Quality warnings surfaced during completeness calculation

### 3. Manual Validations Don't Count Toward Diversity (Phase 2.5) - **CRITICAL**
- **Manual validations excluded from diversity layer counting**
- Prevents gaming: Can't satisfy P0/P1 diversity with API + manual
- Requires ≥2 AUTOMATED layers for critical requirements
- Manual validations tracked separately (has_manual flag)
- Rationale: Manual validations are temporary measures before automation

### 4. Manual Validation Limits (Phase 5) - **ENHANCED**
- Detects excessive manual validation usage (>10% threshold)
- Status-aware: flags complete requirements with ONLY manual validation
- Contextual: allows manual validations for pending/in_progress work
- Contextual: allows manual validations alongside automated tests
- Dual check: ratio-based (>10%) and count-based (≥5 complete with only manual)

### 5. Enhanced Warning Context (Phase 5)
- Warnings show valid test sources for scenario's specific components
- Agents receive actionable, component-specific recommendations
- Reduces confusion about which test types are applicable

### 6. Clearer Test Directory Rationale (Phase 6) - ENHANCED
- Added directory structure diagram showing valid vs. invalid sources
- Explains WHY test/phases/, test/cli/, etc. are rejected
- Makes distinction between test orchestration and test sources explicit

### 7. Quality Over Quantity Documentation (Phase 6)
- Added section on meaningful vs. superficial tests
- Provides indicators of test quality (LOC, assertions, edge cases)
- Mitigates risk of agents gaming diversity with trivial tests
- Includes examples of good vs. bad test validation

### 8. Realistic Timeline & Rollout (Overall) - UPDATED
- Timeline: 10-12 days (enhanced Phase 2.5, criticality derivation, realistic Phase 6 estimate)
- Rollout: Stage 1 (5-7 days), Stage 2 (10-14 days), Stage 3 (5-7 days) = 20-28 days total
- Component-aware test coverage examples added
- Comprehensive acceptance criteria with enhanced test quality checks
- Manual validation exclusion from diversity
- Criticality field removal from requirements schema

---

## References

- **Investigation Report**: Initial gaming pattern analysis
- **browser-automation-studio**: Reference implementation (gold standard)
- **deployment-manager**: Primary gaming example (96/100 → ~68/100 expected)
- **PHASED_TESTING.md**: Current testing architecture documentation
- **requirement-schema.md**: Current requirement validation schema
