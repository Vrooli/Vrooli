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

#### 2.1 Define Validation Layers

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

#### 2.2 Layer Detection Function

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

#### 2.3 Requirement Pass Calculation Enhancement

**File**: `scripts/scenarios/lib/completeness-data.js`

**Function**: `calculateRequirementPass(requirements, syncData)`

**Enhancement Strategy**:
- P0 requirements: Require ≥2 layers (strict)
- P1 requirements: Require ≥2 layers (strict)
- P2 requirements: Allow 1 layer (lenient)

**Rationale**: Critical requirements (P0/P1) should have robust multi-layer validation. Nice-to-have features (P2) can have simpler validation.

```javascript
function calculateRequirementPass(requirements, syncData) {
  let passing = 0;
  let total = requirements.length;
  const syncMetadata = syncData.requirements || syncData;

  for (const req of requirements) {
    const criticality = (req.criticality || 'P2').toUpperCase();
    const status = req.status || 'pending';
    const reqMeta = syncMetadata[req.id];

    // Detect validation layers
    const layers = detectValidationLayers(req);

    // Minimum layer requirement based on criticality
    const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;

    // Requirement passes if:
    // 1. Status indicates completion (validated/implemented/complete)
    // 2. Sync metadata shows completion
    // 3. Has sufficient validation layer diversity
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

    const diversityPasses = layers.size >= minLayers;

    // All three conditions must be true
    if ((statusPasses || syncPasses) && diversityPasses) {
      passing++;
    }
  }

  return { total, passing };
}
```

**Migration Strategy**:
- Add `diversity_pass` field to requirement metadata during sync
- Surface diversity failures in `vrooli scenario completeness` output
- Provide actionable recommendations (e.g., "Add UI tests for REQ-001")

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
Acceptable 1:1 ratio = min(0.5, 5 / total_targets)

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
    patterns.unsupported_test_directory = {
      severity: ratio > 0.5 ? 'high' : 'medium',
      detected: true,
      count: unsupportedRefCount,
      total: metrics.requirements.total,
      ratio: ratio,
      message: `${unsupportedRefCount}/${metrics.requirements.total} requirements reference unsupported test/ directories. Only test/playbooks/*.{json,yaml} is valid.`,
      recommendation: 'Move validation refs to: api/**/*_test.go, ui/src/**/*.test.tsx, or test/playbooks/**/*.{json,yaml}'
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
  const diversityIssues = requirements.filter(req => {
    const layers = detectValidationLayers(req);
    const criticality = (req.criticality || 'P2').toUpperCase();
    const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;
    return layers.size < minLayers;
  });

  if (diversityIssues.length > 0) {
    patterns.missing_validation_diversity = {
      severity: 'high',
      detected: true,
      count: diversityIssues.length,
      total: requirements.length,
      message: `${diversityIssues.length} critical requirements (P0/P1) lack multi-layer validation`,
      recommendation: 'Add validations across API, UI, and e2e layers for critical requirements'
    };
    warnings.push(patterns.missing_validation_diversity);
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
- ✅ Document allowed validation ref patterns
- ✅ Add validation rules for `ref` field
- ✅ Clarify `type` field values and their meanings
- ✅ Add examples showing proper vs. improper validation structures

#### 6.2 New Documentation

**File**: `docs/testing/guides/validation-best-practices.md` (NEW)

**Contents**:
- Valid vs. invalid validation sources (with examples)
- Multi-layer validation strategy (API + UI + e2e)
- Operational target grouping guidelines
- Test-to-requirement ratio recommendations
- How to avoid gaming patterns
- Browser-automation-studio as reference implementation

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

| Phase | Priority | Estimated Time | Dependencies |
|-------|----------|----------------|--------------|
| **Phase 1**: Validation Source Filtering | P0 | 1 day | None |
| **Phase 2**: Validation Diversity | P0 | 1 day | Phase 1 |
| **Phase 3**: Duplicate Ref Detection | P1 | 0.5 days | Phase 1 |
| **Phase 4**: Target Grouping Validation | P1 | 0.5 days | None |
| **Phase 5**: Gaming Detection & Warnings | P0 | 1 day | All above |
| **Phase 6**: Documentation Updates | P1 | 1 day | Phase 5 (for examples) |

**Total Estimated Time**: 5 days

### Rollout Strategy

#### Stage 1: Detection Only (Week 1)
- Implement all detection logic
- Output warnings but **don't change scores**
- Gather data on existing scenarios
- Validate detection accuracy

**Command**: `vrooli scenario completeness <name> --detect-gaming`

**Output**:
```
Score: 96/100 (Production Ready)
⚠️  Gaming patterns detected (detection mode - not affecting score):
  🔴 37/37 requirements reference unsupported test directories
  🟡 6 test files validate ≥4 requirements each
  🟡 100% of operational targets have 1:1 requirement mapping
```

#### Stage 2: Soft Enforcement (Week 2)
- Apply scoring penalties
- Generate detailed reports for affected scenarios
- Provide migration guidance
- Support agents in fixing issues

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

#### Stage 3: Full Enforcement (Week 3+)
- Auto-sync rejects invalid refs during `vrooli scenario requirements sync`
- Completeness score reflects true validation quality
- Documentation updated with examples
- ecosystem-manager agents trained on new patterns

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
- [ ] Warning logged for each rejected ref
- [ ] browser-automation-studio: all validations still accepted
- [ ] deployment-manager: 37 validations rejected

### Phase 2: Validation Diversity
- [ ] detectValidationLayers() correctly identifies API, UI, E2E, MANUAL
- [ ] P0/P1 requirements require ≥2 layers to pass
- [ ] P2 requirements require ≥1 layer to pass
- [ ] Completeness breakdown shows diversity_pass field
- [ ] Recommendations generated for diversity failures

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
- [ ] All 5 gaming patterns detected correctly
- [ ] Warnings displayed with severity levels (high/medium)
- [ ] Recommendations actionable and specific
- [ ] JSON output includes gaming_analysis field
- [ ] No duplicate detection code (reuses helpers)

### Phase 6: Documentation
- [ ] All 4 existing docs updated with gaming prevention guidance
- [ ] validation-best-practices.md created with clear examples
- [ ] gaming-prevention.md created with pattern catalog
- [ ] browser-automation-studio referenced as gold standard
- [ ] Migration guide provided for existing scenarios

### Overall System
- [ ] browser-automation-studio score unchanged (62/100)
- [ ] deployment-manager score reduced (96 → 65-70/100)
- [ ] landing-manager score reduced (similar magnitude)
- [ ] Zero false positives on properly-structured scenarios
- [ ] Clear, actionable warnings for all gaming patterns
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
  "validation": [
    {"type": "test", "ref": "test/cli/dependency-analysis.bats"}
  ]
}

// AFTER (valid)
{
  "id": "DM-P0-001",
  "validation": [
    {"type": "test", "ref": "api/analyzer_test.go", "phase": "unit"},
    {"type": "test", "ref": "ui/src/components/DependencyGraph.test.tsx", "phase": "unit"},
    {"type": "automation", "ref": "test/playbooks/capabilities/dependency-viz/tree-view.json", "phase": "integration"}
  ]
}
```

#### Warning: "Missing multi-layer validation"
**Action**:
1. Identify requirement criticality (P0/P1 needs ≥2 layers)
2. Add validations for missing layers
3. Ensure at least: API test + UI test OR API test + e2e playbook

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
| **Resistance from agents** to fixing issues | Medium - Slows progress | Provide clear migration guide, actionable recommendations |
| **Breaking changes** to existing workflows | High - Disrupts current scenarios | Phased rollout, soft enforcement period |
| **Documentation gaps** causing confusion | Medium - Agents repeat mistakes | Comprehensive docs with examples, FAQ section |
| **Threshold tuning** needed for edge cases | Low - Some scenarios don't fit model | Make thresholds configurable, gather feedback |

---

## Success Metrics

### Quantitative
- ✅ browser-automation-studio completeness score unchanged (±5%)
- ✅ deployment-manager score reduced by 20-30 points (reflects true quality)
- ✅ Zero gaming warnings on browser-automation-studio
- ✅ 90%+ of gamed scenarios show ≥3 gaming pattern warnings
- ✅ Validation diversity compliance: 80%+ of P0/P1 requirements have ≥2 layers

### Qualitative
- ✅ Agents understand and can fix gaming warnings
- ✅ New scenarios follow best practices from day 1
- ✅ ecosystem-manager agents produce accurate completeness assessments
- ✅ Documentation clearly explains validation requirements
- ✅ Community feedback positive on fairness and clarity

---

## Next Steps

1. **Review this plan** with stakeholders
2. **Validate approach** with sample implementation on deployment-manager
3. **Begin Phase 1** implementation (validation source filtering)
4. **Test on fixtures** to verify detection accuracy
5. **Iterate based on feedback** before full rollout

---

## References

- **Investigation Report**: Initial gaming pattern analysis
- **browser-automation-studio**: Reference implementation (gold standard)
- **deployment-manager**: Primary gaming example (96/100 → ~68/100 expected)
- **PHASED_TESTING.md**: Current testing architecture documentation
- **requirement-schema.md**: Current requirement validation schema
