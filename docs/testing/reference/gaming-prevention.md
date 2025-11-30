# Gaming Prevention System Reference

**Status**: Active
**Last Updated**: 2025-11-22
**Module**: `scenarios/scenario-completeness-scoring/api/pkg/validators`
**Audience**: System developers, maintainers

---

> **Note**: The Go packages listed above now own the gaming prevention logic, which was originally implemented in the legacy `scripts/scenarios/lib` directory. That directory has been removed after the migration; use the Go files below for current behavior and the snippets in this document only for historical reference.

## Overview

The gaming prevention system detects and prevents common patterns where requirement validation metrics are artificially inflated without genuine test coverage. This document provides technical details on how the system works and how to interpret its warnings.

**Design Goal**: Ensure completeness scores reflect true implementation quality, not superficial checkmarks.

---

## Architecture

### Module Structure

```
scenarios/scenario-completeness-scoring/api/pkg/
├── validators/              # Focused detection modules (Go equivalents of the old JS validators)
│   ├── component_detector.go    # Detects API/UI components (records available layers)
│   ├── layer_detector.go        # Multi-layer validation detection and criticality inference
│   ├── test_quality.go          # Test file quality heuristics (monolithic/empty tests)
│   ├── duplicate_detector.go    # Duplicate test ref detection
│   └── target_grouping.go       # Operational target grouping enforcement
├── analyzers/                # Quality analysis helpers (historical analog of requirement/target/ui analyzers)
├── scoring/                  # Completeness score calculator / classification helpers
├── collectors/               # Data loaders (requirements, tests, config, UI metrics)
├── handlers/                 # HTTP handlers that expose scores, config and analysis APIs
└── api/pkg/handlers/scores.go # Orchestrates the validators and scoring logic for completeness outputs
```

### Design Principles

1. **Separation of Concerns**: Each validator has one responsibility
2. **Reusability**: Validators used by multiple systems (completeness, linting)
3. **Testability**: Each module independently unit-testable
4. **No Duplication**: Shared logic in validators, not scattered
5. **Orchestration**: `gaming-detection.js` wires validators together

---

## Gaming Pattern Detection

The system detects **7 gaming patterns** with varying severity levels:

| Pattern | Severity | Description | Fix |
|---------|----------|-------------|-----|
| 1. 1:1 test-to-requirement ratio | Medium | Exactly 1 test per requirement (suspicious) | Add diverse test types per requirement |
| 2. Unsupported test/ directories | High | Using test/cli/, test/phases/ as validation refs | Move refs to actual test sources (API, UI, playbooks) |
| 3. Duplicate test refs | High | One test file validates ≥4 requirements | Break monolithic tests into focused files |
| 4. Excessive 1:1 target mapping | High | 100% of OTs have 1:1 requirement mapping | Group requirements under shared OT-XXX targets |
| 5. Missing validation diversity | High | P0/P1 requirements lack ≥2 automated layers | Add tests across API, UI, e2e layers |
| 6. Low-quality test files | Medium | Tests <20 LOC, no assertions, no test functions | Add meaningful test logic with edge cases |
| 7. Excessive manual validations | Medium/High | >10% manual OR ≥5 complete with only manual | Replace manual validations with automated tests |

---

## Pattern 1: 1:1 Test-to-Requirement Ratio

### Detection Logic

**File**: `gaming-detection.js:26-37`

```javascript
const testReqRatio = metrics.tests.total / Math.max(metrics.requirements.total, 1);
if (Math.abs(testReqRatio - 1.0) < 0.1) {  // Within 10% of 1:1
  patterns.one_to_one_test_ratio = {
    severity: 'medium',
    detected: true,
    ratio: testReqRatio,
    message: `Suspicious 1:1 test-to-requirement ratio (${metrics.tests.total} tests for ${metrics.requirements.total} requirements). Expected: 1.5-2.0x ratio with diverse test sources.`
  };
}
```

### Rationale

Proper test coverage requires multiple test types per requirement:
- API unit tests
- UI unit tests
- E2E automation

A perfect 1:1 ratio suggests single-layer validation (typically only CLI or only API tests).

### Example

❌ **Gaming**: 37 requirements, 37 tests (ratio: 1.0)
- All 37 tests are BATS CLI tests
- No API unit tests, no UI unit tests, no e2e automation

✅ **Proper**: 37 requirements, 90 tests (ratio: 2.43)
- 35 API Go unit tests
- 42 UI Vitest tests
- 13 BAS e2e playbooks

### Threshold

**Trigger**: Test-to-requirement ratio between 0.9 and 1.1 (within 10% of 1:1)

**Recommended ratio**: 1.5-2.0x (with diverse sources)

---

## Pattern 2: Unsupported Test Directory Usage

### Detection Logic

**File**: `gaming-detection.js:39-73`

```javascript
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

  patterns.unsupported_test_directory = {
    severity: ratio > 0.5 ? 'high' : 'medium',
    detected: true,
    count: unsupportedRefCount,
    message: `${unsupportedRefCount}/${metrics.requirements.total} requirements reference unsupported test/ directories.`
  };
}
```

### Rationale

**Rejected directories**:
- `test/phases/` - Orchestration scripts that RUN tests, not test sources
- `test/cli/` - CLI wrapper tests, not business logic validation
- `test/unit/`, `test/integration/` - Test runner infrastructure

**Allowed directory**:
- `test/playbooks/` - E2E automation workflows (actual test sources)

### Example

❌ **Gaming**:
```json
{
  "id": "DM-P0-001",
  "validation": [
    {"type": "test", "ref": "test/cli/dependency-analysis.bats"}  // ← Invalid
  ]
}
```

✅ **Proper**:
```json
{
  "id": "DM-P0-001",
  "validation": [
    {"type": "test", "ref": "api/analyzer_test.go"},  // ← Valid API test
    {"type": "automation", "ref": "test/playbooks/analyze.json"}  // ← Valid E2E
  ]
}
```

### Threshold

**Severity**:
- High: >50% of requirements use unsupported refs
- Medium: ≤50% of requirements use unsupported refs

### Component-Aware Recommendations

The warning includes valid test sources based on detected components:

```javascript
// If scenario has API component
validSources.push('api/**/*_test.go (API unit tests)');

// If scenario has UI component
validSources.push('ui/src/**/*.test.tsx (UI unit tests)');

// Always applicable
validSources.push('test/playbooks/**/*.{json,yaml} (e2e automation)');
```

---

## Pattern 3: Duplicate Test Refs

### Detection Logic

**File**: `scenarios/scenario-completeness-scoring/api/pkg/validators/duplicate_detector.go` (legacy JS logic shown below)

```javascript
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
    if (reqIds.length >= 4) {
      violations.push({
        test_ref: ref,
        shared_by: reqIds,
        count: reqIds.length,
        severity: reqIds.length >= 6 ? 'high' : 'medium'
      });
    }
  });

  return { violations, ... };
}
```

### Rationale

One test file shouldn't validate many requirements because:
1. Lack of test specificity (one monolithic test doing everything)
2. Low traceability (which test validates which requirement?)
3. Encourages "one big BATS file" anti-pattern

### Example

❌ **Gaming**:
- `test/cli/dependency-analysis.bats` validates 6 requirements
- `test/cli/profile-operations.bats` validates 8 requirements
- Total: 6 violations

✅ **Proper**:
- Each requirement has dedicated test files
- Some overlap is OK (shared helper functions), but not excessive

### Threshold

**Violation**: 1 test file validates ≥4 requirements

**Severity**:
- High: ≥6 requirements per file
- Medium: 4-5 requirements per file

**Penalty**: -1 point per 10% of requirements using duplicate refs (max -5 points)

---

## Pattern 4: Excessive 1:1 Target Mapping

### Detection Logic

**File**: `scenarios/scenario-completeness-scoring/api/pkg/validators/target_grouping.go` (legacy JS logic shown below)

```javascript
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
      current_ratio: oneToOneRatio,
      acceptable_ratio: acceptableRatio,
      message: `${Math.round(oneToOneRatio * 100)}% of targets have 1:1 requirement mapping (max ${Math.round(acceptableRatio * 100)}% recommended)`
    });
  }

  return { violations, ... };
}
```

### Rationale

Operational targets represent PRD goals, not individual requirements. Proper grouping:
- 1 OT target → 3-10 requirements

A 1:1 mapping inflates target completion metrics.

### Example

❌ **Gaming**:
```json
{"id": "REQ-001", "prd_ref": "OT-P0-001"},  // ← 37 unique targets
{"id": "REQ-002", "prd_ref": "OT-P0-002"},
// ... 35 more, each with unique OT-XXX
```
- 37 requirements → 37 targets (100% 1:1)
- All targets "complete" when requirements complete
- Inflated target pass rate

✅ **Proper**:
```json
{"id": "PROFILE-CREATE", "prd_ref": "OT-P0-001"},  // Profile Management
{"id": "PROFILE-UPDATE", "prd_ref": "OT-P0-001"},  // Same OT
{"id": "PROFILE-DELETE", "prd_ref": "OT-P0-001"},  // Same OT
{"id": "DEPLOY-VALIDATE", "prd_ref": "OT-P0-002"}, // Deployment Flow
{"id": "DEPLOY-EXECUTE", "prd_ref": "OT-P0-002"},  // Same OT
```
- 37 requirements → 7 targets (proper grouping)
- Target passes when ≥50% of linked requirements complete

### Threshold (Dynamic)

```
acceptableRatio = min(0.2, 5 / total_targets)

Examples:
- 5 targets  → max 1 allowed (20%)
- 10 targets → max 2 allowed (20%)
- 37 targets → max 7 allowed (20%)
```

**Severity**:
- High: >50% 1:1 mapping
- Medium: 20-50% 1:1 mapping

---

## Pattern 5: Missing Validation Diversity

### Detection Logic

**File**: `gaming-detection.js:104-138`

```javascript
const scenarioComponents = componentDetector.detectScenarioComponents(scenarioRoot);
const applicableLayers = componentDetector.getApplicableLayers(scenarioComponents);

const diversityIssues = requirements.filter(req => {
  const criticality = layerDetector.deriveRequirementCriticality(req);
  const layerAnalysis = layerDetector.detectValidationLayers(req, scenarioRoot);

  // Filter to only applicable AUTOMATED layers
  const applicable = new Set();
  if (scenarioComponents.has('API') && layerAnalysis.automated.has('API')) {
    applicable.add('API');
  }
  if (scenarioComponents.has('UI') && layerAnalysis.automated.has('UI')) {
    applicable.add('UI');
  }
  if (layerAnalysis.automated.has('E2E')) {
    applicable.add('E2E');
  }

  const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;
  return applicable.size < minLayers;
});

if (diversityIssues.length > 0) {
  patterns.missing_validation_diversity = {
    severity: 'high',
    count: diversityIssues.length,
    message: `${diversityIssues.length} critical requirements (P0/P1) lack multi-layer AUTOMATED validation`
  };
}
```

### Component-Aware Detection

**1. Detect scenario components** (`scenarios/scenario-completeness-scoring/api/pkg/validators/component_detector.go` - legacy JS excerpt shown below):
```javascript
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

**2. Determine applicable layers**:
```javascript
function getApplicableLayers(components) {
  const layers = new Set(['E2E']);  // E2E always applicable

  if (components.has('API')) layers.add('API');
  if (components.has('UI')) layers.add('UI');

  return layers;
}
```

**3. Detect validation layers per requirement** (`scenarios/scenario-completeness-scoring/api/pkg/validators/layer_detector.go` - legacy JS excerpt shown below):
```javascript
function detectValidationLayers(requirement, scenarioRoot) {
  const automatedLayers = new Set();
  const hasManual = (requirement.validation || []).some(v => v.type === 'manual');

  (requirement.validation || []).forEach(v => {
    // Skip manual validations
    if (v.type === 'manual') return;

    // Quality check test files
    if (v.type === 'test' && v.ref) {
      const quality = testQualityAnalyzer.analyzeTestFileQuality(v.ref, scenarioRoot);
      if (!quality.is_meaningful) return;  // Exclude low-quality tests
    }

    // Check layer patterns
    if (VALIDATION_LAYERS.API.patterns.some(p => p.test(v.ref))) {
      automatedLayers.add('API');
    }
    // ... similar for UI, E2E
  });

  return { automated: automatedLayers, has_manual: hasManual };
}
```

### Rationale

Critical requirements (P0/P1) need robust validation across multiple test layers:
- **Full-stack**: API + UI OR API + E2E OR UI + E2E (need ≥2)
- **API-only**: API + E2E (need ≥2, UI tests ignored)
- **UI-only**: UI + E2E (need ≥2, API tests ignored)

**Manual validations excluded**: They don't provide continuous automated validation.

### Example

❌ **Insufficient diversity**:
```json
{
  "id": "WORKFLOW-CRUD",
  "prd_ref": "OT-P0-001",  // ← P0 needs ≥2 automated layers
  "validation": [
    {"type": "test", "ref": "api/workflow_test.go"},  // ← Only 1 automated layer
    {"type": "manual", "notes": "Manually tested"}    // ← Manual doesn't count
  ]
}
// ❌ Only 1 automated layer (API), needs 2
```

✅ **Sufficient diversity**:
```json
{
  "id": "WORKFLOW-CRUD",
  "prd_ref": "OT-P0-001",
  "validation": [
    {"type": "test", "ref": "api/workflow_test.go"},  // ← API layer
    {"type": "test", "ref": "ui/src/stores/workflowStore.test.ts"},  // ← UI layer
    {"type": "automation", "ref": "test/playbooks/workflows/crud.json"}  // ← E2E layer (bonus)
  ]
}
// ✅ 3 automated layers (API + UI + E2E)
```

### Threshold

**Requirement diversity requirement**:
- P0/P1: ≥2 AUTOMATED layers (from applicable layers)
- P2: ≥1 AUTOMATED layer

**Severity**: High (always)

---

## Pattern 6: Low-Quality Test Files

### Detection Logic

**File**: `scenarios/scenario-completeness-scoring/api/pkg/validators/test_quality.go` (legacy JS logic shown below)

```javascript
function analyzeTestFileQuality(testFilePath, scenarioRoot) {
  const fullPath = path.join(scenarioRoot, testFilePath);

  if (!fs.existsSync(fullPath)) {
    return { exists: false, is_meaningful: false, reason: 'file_not_found' };
  }

  const content = fs.readFileSync(fullPath, 'utf8');
  const lines = content.split('\n');
  const nonEmptyLines = lines.filter(l => l.trim().length > 0);

  // Remove comment-only lines
  const nonCommentLines = nonEmptyLines.filter(l => {
    const t = l.trim();
    return !t.startsWith('//') && !t.startsWith('/*') &&
           !t.startsWith('*') && !t.startsWith('#');
  });

  // Count test functions
  const testFunctionMatches = content.match(/func Test|@test|test\(|it\(|describe\(|def test_/gi);
  const testFunctionCount = testFunctionMatches ? testFunctionMatches.length : 0;

  // Count assertions
  const assertionMatches = content.match(/assert|expect|require|Should|Equal|Contains|Error|True|False|toBe|toHaveBeenCalled/gi);
  const assertionCount = assertionMatches ? assertionMatches.length : 0;
  const assertionDensity = nonCommentLines.length > 0 ? assertionCount / nonCommentLines.length : 0;

  // Quality heuristics
  const hasMinimumCode = nonCommentLines.length >= 20;
  const hasAssertions = assertionCount > 0;
  const hasMultipleTestFunctions = testFunctionCount >= 3;
  const hasGoodAssertionDensity = assertionDensity >= 0.1;
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
    test_function_count: testFunctionCount,
    assertion_count: assertionCount,
    is_meaningful: qualityScore >= 4,  // Need 4+ indicators
    quality_score: qualityScore
  };
}
```

### Quality Indicators

| Indicator | Threshold | Score |
|-----------|-----------|-------|
| Lines of Code | ≥20 LOC (non-comment) | 1 |
| Assertions | ≥1 assertion | 1 |
| Test Functions | ≥1 test case | 1 |
| Multiple Test Cases | ≥3 test functions | 1 |
| Assertion Density | ≥0.1 (1 per 10 LOC) | 1 |

**Passing score**: ≥4/5

### Rationale

Prevents superficial test files created just to satisfy requirements:
```go
// api/placeholder_test.go (5 lines, quality score: 1/5)
package api
import "testing"
// Empty
```

**Impact**: Low-quality tests excluded from layer diversity counting.

### Playbook Quality

Similar checks for e2e playbooks (`analyzePlaybookQuality`):
- Must have ≥1 node/step with actions
- File size ≥100 bytes
- Valid JSON structure

### Threshold

**Detection**: Test file with quality score <4/5

**Severity**: Medium

**Impact**: Excluded from automated layer counting (doesn't contribute to diversity)

---

## Pattern 7: Excessive Manual Validations

### Detection Logic

**File**: `gaming-detection.js:170-205`

```javascript
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
    ratio: manualRatio,
    count: manualValidations,
    complete_with_manual: completeWithManual.length,
    message: manualRatio > 0.10
      ? `${Math.round(manualRatio * 100)}% of validations are manual (max 10% recommended)`
      : `${completeWithManual.length} requirements marked complete with ONLY manual validations`
  };
}
```

### Rationale

Manual validations are acceptable as temporary measures, but problematic when:
1. **>10% of all validations** are manual (automation deficit)
2. **≥5 complete requirements** have ONLY manual validation (no automated tests)

**Why manual validations don't count toward diversity**:
- They don't provide continuous validation (run once, forget)
- They can be gamed ("manually verified" without real testing)
- Goal is **automated** multi-layer coverage for CI/CD

### Example

❌ **Excessive manual usage**:
```json
// 15% of validations are manual
{
  "id": "REQ-001",
  "status": "complete",  // ← Complete but only manual
  "validation": [
    {"type": "manual", "notes": "Manually tested"}
  ]
},
// ... 9 more complete requirements with only manual validation
```

✅ **Acceptable manual usage**:
```json
// Manual as temporary bridge
{
  "id": "REQ-001",
  "status": "in_progress",  // ← Not complete yet
  "validation": [
    {"type": "manual", "notes": "Awaiting BAS automation setup"}
  ]
},
// Manual as supplementary evidence
{
  "id": "REQ-002",
  "status": "complete",
  "validation": [
    {"type": "test", "ref": "api/test.go"},  // ← Has automated test
    {"type": "manual", "notes": "Additional exploratory testing"}  // ← Supplementary
  ]
}
```

### Threshold

**Trigger conditions (either)**:
- Manual validation ratio >10%
- ≥5 complete requirements with ONLY manual validation

**Severity**:
- High: ≥10 complete requirements with only manual
- Medium: <10 complete requirements with only manual

---

## Completeness Score Impact

### Quality Score Penalties

| Pattern | Penalty | Max Impact |
|---------|---------|------------|
| Duplicate test refs | -1 point per 10% duplicate ratio | -5 points |
| Low-quality tests | Excluded from layer counting | Indirect (diversity fails) |
| Manual validations | Excluded from diversity | Indirect (diversity fails) |

### Requirement Pass Rate Impact

**Before gaming prevention**:
- Requirements passed with single-layer validation
- Manual validations counted toward diversity
- Low-quality tests counted as valid evidence

**After gaming prevention**:
- P0/P1 requirements need ≥2 AUTOMATED layers (from applicable components)
- Manual validations excluded from diversity
- Low-quality tests excluded from layer counting

**Impact**:
- deployment-manager: 100% pass rate → 0% (all requirements failed diversity)
- browser-automation-studio: 52% pass rate → 37% (stricter diversity enforcement)

---

## Migration Guide

### For Scenarios with Gaming Warnings

#### Warning: "Requirements lack multi-layer AUTOMATED validation"

**Action**:
1. Check requirement's `prd_ref` to determine criticality (OT-P0-XXX → P0)
2. Check scenario components (API, UI, both)
3. Add AUTOMATED validations for missing layers:
   - Full-stack: Need 2 from [API, UI, E2E]
   - API-only: Need API + E2E
   - UI-only: Need UI + E2E
4. Ensure manual validations are replaced with automation

#### Warning: "Requirements reference unsupported test/ directories"

**Action**:
1. Identify unsupported refs:
   - test/cli/*.bats → Replace with API tests + e2e
   - test/phases/*.sh → Replace with actual test files
2. Update requirement validation entries to point to proper sources

#### Warning: "Test files validate ≥4 requirements each"

**Action**:
1. Identify monolithic test files
2. Break into focused test files (one per requirement or logical group)
3. Update validation refs to point to specific files

#### Warning: "Test files appear superficial"

**Action**:
1. Add meaningful test logic (≥20 LOC, assertions, edge cases)
2. Ensure quality score ≥4/5:
   - Multiple test cases (≥3)
   - Good assertion density (≥0.1)
   - Comprehensive coverage (happy path + errors)

#### Warning: "100% 1:1 operational target mapping"

**Action**:
1. Review PRD operational targets
2. Group related requirements under shared OT-XXX refs
3. Aim for 5-10 targets grouping all requirements

#### Warning: "Excessive manual validations"

**Action**:
1. Replace manual validations with automated tests
2. Use BAS playbooks for UI/e2e coverage
3. Keep manual validations only for pending/in_progress requirements

---

## Testing the Gaming Prevention System

### Unit Tests

Each validator is covered by Go tests in `scenarios/scenario-completeness-scoring/api/pkg/validators`:

```bash
# Run all validator tests
go test ./scenarios/scenario-completeness-scoring/api/pkg/validators
```

### Integration Tests

Test on real scenarios:

```bash
# Test on gaming example
vrooli scenario completeness deployment-manager

# Expected: 5 high-severity warnings
# - Unsupported test/ directories
# - Duplicate test refs
# - 1:1 target mapping
# - Missing diversity
# - 1:1 test ratio

# Test on proper example
vrooli scenario completeness browser-automation-studio

# Expected: 4 medium-severity warnings (fewer issues)
# - Some unsupported refs (7/63)
# - Some duplicate refs (4 files)
# - Some diversity issues (14/63 P0/P1)
# - Some low-quality tests (9 files)
```

---

## See Also

- [Validation Best Practices](../guides/validation-best-practices.md) - User-facing guide
- [Requirement Schema](./requirement-schema.md) - Schema reference with component-aware examples
- [Completeness Scoring Plan](../../plans/completeness-gaming-prevention.md) - Implementation plan
