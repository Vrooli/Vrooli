# Gaming Prevention System Reference

**Status**: Active
**Last Updated**: 2025-12-02
**Module**: `scenarios/scenario-completeness-scoring/api/pkg/validators`
**Audience**: System developers, maintainers

---

## Overview

The gaming prevention system detects and prevents common patterns where requirement validation metrics are artificially inflated without genuine test coverage.

**Design Goal**: Ensure completeness scores reflect true implementation quality, not superficial checkmarks.

---

## Architecture

### Module Structure

```
scenarios/scenario-completeness-scoring/api/pkg/
├── validators/
│   ├── component_detector.go    # Detects API/UI components
│   ├── layer_detector.go        # Multi-layer validation detection
│   ├── test_quality.go          # Test file quality heuristics
│   ├── duplicate_detector.go    # Duplicate test ref detection
│   └── target_grouping.go       # Operational target grouping
├── analyzers/                   # Quality analysis helpers
├── scoring/                     # Completeness score calculator
└── collectors/                  # Data loaders
```

---

## Gaming Pattern Detection

The system detects **7 gaming patterns** with varying severity levels:

| Pattern | Severity | Description |
|---------|----------|-------------|
| 1:1 test-to-requirement ratio | Medium | Exactly 1 test per requirement |
| Unsupported test/ directories | High | Using test/cli/, test/phases/ as refs |
| Duplicate test refs | High | One test file validates >= 4 requirements |
| Excessive 1:1 target mapping | High | 100% of OTs have 1:1 requirement mapping |
| Missing validation diversity | High | P0/P1 requirements lack >= 2 automated layers |
| Low-quality test files | Medium | Tests < 20 LOC, no assertions |
| Excessive manual validations | Medium/High | > 10% manual OR >= 5 complete with only manual |

---

## Pattern 1: 1:1 Test-to-Requirement Ratio

### Detection Logic

```javascript
const testReqRatio = metrics.tests.total / Math.max(metrics.requirements.total, 1);
if (Math.abs(testReqRatio - 1.0) < 0.1) {  // Within 10% of 1:1
  // Flag as suspicious
}
```

### Rationale

Proper test coverage requires multiple test types per requirement:
- API unit tests
- UI unit tests
- E2E automation

A perfect 1:1 ratio suggests single-layer validation.

### Example

**Gaming**: 37 requirements, 37 tests (ratio: 1.0)

**Proper**: 37 requirements, 90 tests (ratio: 2.43)

### Threshold

**Trigger**: Test-to-requirement ratio between 0.9 and 1.1
**Recommended ratio**: 1.5-2.0x

---

## Pattern 2: Unsupported Test Directory Usage

### Rationale

**Rejected directories**:
- `test/phases/` - Orchestration scripts that RUN tests
- `test/cli/` - CLI wrapper tests, not business logic
- `test/unit/`, `test/integration/` - Test runner infrastructure

**Allowed directory**:
- `test/playbooks/` - E2E automation workflows

### Example

**Gaming**:
```json
{"type": "test", "ref": "test/cli/dependency-analysis.bats"}
```

**Proper**:
```json
{"type": "test", "ref": "api/analyzer_test.go"},
{"type": "automation", "ref": "test/playbooks/analyze.json"}
```

### Threshold

**Severity**:
- High: > 50% of requirements use unsupported refs
- Medium: <= 50% of requirements use unsupported refs

---

## Pattern 3: Duplicate Test Refs

### Detection Logic

Tracks how many requirements reference each test file:

```javascript
// Detect violations (1 test file validates >= 4 requirements)
if (reqIds.length >= 4) {
  violations.push({
    test_ref: ref,
    shared_by: reqIds,
    severity: reqIds.length >= 6 ? 'high' : 'medium'
  });
}
```

### Rationale

One test file shouldn't validate many requirements because:
1. Lack of test specificity
2. Low traceability
3. Encourages "one big BATS file" anti-pattern

### Threshold

**Violation**: 1 test file validates >= 4 requirements

**Severity**:
- High: >= 6 requirements per file
- Medium: 4-5 requirements per file

---

## Pattern 4: Excessive 1:1 Target Mapping

### Detection Logic

```javascript
const oneToOneRatio = oneToOneCount / totalTargets;
const acceptableRatio = Math.min(0.2, 5 / totalTargets);

if (oneToOneRatio > acceptableRatio) {
  // Flag violation
}
```

### Rationale

Operational targets represent PRD goals, not individual requirements. Proper grouping:
- 1 OT target → 3-10 requirements

A 1:1 mapping inflates target completion metrics.

### Example

**Gaming**:
```json
{"id": "REQ-001", "prd_ref": "OT-P0-001"},
{"id": "REQ-002", "prd_ref": "OT-P0-002"},
// 37 requirements → 37 targets (100% 1:1)
```

**Proper**:
```json
{"id": "PROFILE-CREATE", "prd_ref": "OT-P0-001"},
{"id": "PROFILE-UPDATE", "prd_ref": "OT-P0-001"},
{"id": "PROFILE-DELETE", "prd_ref": "OT-P0-001"},
// 37 requirements → 7 targets (proper grouping)
```

---

## Pattern 5: Missing Validation Diversity

### Component-Aware Detection

**1. Detect scenario components**:
```javascript
// Check for API component
const hasAPI = fs.existsSync(path.join(scenarioRoot, 'api'));

// Check for UI component
const hasUI = fs.existsSync(path.join(scenarioRoot, 'ui/package.json'));
```

**2. Determine applicable layers**:
```javascript
const layers = new Set(['E2E']);
if (components.has('API')) layers.add('API');
if (components.has('UI')) layers.add('UI');
```

**3. Check diversity per requirement**:
```javascript
const minLayers = (criticality === 'P0' || criticality === 'P1') ? 2 : 1;
return applicable.size < minLayers;
```

### Rationale

Critical requirements (P0/P1) need robust validation across multiple layers:
- **Full-stack**: API + UI OR API + E2E OR UI + E2E
- **API-only**: API + E2E
- **UI-only**: UI + E2E

**Manual validations excluded**: They don't provide continuous automated validation.

---

## Pattern 6: Low-Quality Test Files

### Quality Indicators

| Indicator | Threshold | Score |
|-----------|-----------|-------|
| Lines of Code | >= 20 LOC (non-comment) | 1 |
| Assertions | >= 1 assertion | 1 |
| Test Functions | >= 1 test case | 1 |
| Multiple Test Cases | >= 3 test functions | 1 |
| Assertion Density | >= 0.1 (1 per 10 LOC) | 1 |

**Passing score**: >= 4/5

### Rationale

Prevents superficial test files:
```go
// api/placeholder_test.go (5 lines, quality score: 1/5)
package api
import "testing"
// Empty
```

**Impact**: Low-quality tests excluded from layer diversity counting.

---

## Pattern 7: Excessive Manual Validations

### Detection Logic

```javascript
const manualRatio = manualValidations / totalValidations;

// Flag if: (a) > 10% manual overall OR (b) >= 5 complete with only manual
if (manualRatio > 0.10 || completeWithManual.length >= 5) {
  // Flag violation
}
```

### Rationale

Manual validations are acceptable as temporary measures, but problematic when:
1. **> 10% of all validations** are manual
2. **>= 5 complete requirements** have ONLY manual validation

---

## Completeness Score Impact

### Quality Score Penalties

| Pattern | Penalty | Max Impact |
|---------|---------|------------|
| Duplicate test refs | -1 point per 10% | -5 points |
| Low-quality tests | Excluded from counting | Indirect |
| Manual validations | Excluded from diversity | Indirect |

---

## Migration Guide

### Warning: "Requirements lack multi-layer validation"

**Action**:
1. Check requirement's `prd_ref` to determine criticality
2. Check scenario components (API, UI, both)
3. Add AUTOMATED validations for missing layers

### Warning: "Requirements reference unsupported directories"

**Action**:
1. Replace `test/cli/*.bats` with API tests + e2e
2. Replace `test/phases/*.sh` with actual test files

### Warning: "Test files validate >= 4 requirements each"

**Action**:
1. Identify monolithic test files
2. Break into focused test files
3. Update validation refs

### Warning: "Test files appear superficial"

**Action**:
1. Add meaningful test logic (>= 20 LOC)
2. Add assertions and edge cases
3. Ensure quality score >= 4/5

---

## Testing the System

### Integration Tests

```bash
# Test on gaming example
vrooli scenario completeness deployment-manager

# Test on proper example
vrooli scenario completeness browser-automation-studio
```

---

## See Also

- [Validation Best Practices](../guides/validation-best-practices.md) - User-facing guide
- [Requirement Schema](requirement-schema.md) - Schema reference
- [Examples](examples.md) - Gold standard implementations
