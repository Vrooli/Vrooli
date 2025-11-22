# Scenario Completeness Scoring System

> **Status**: Planning
> **Created**: 2025-11-21
> **Target**: ecosystem-manager recycler enhancement
> **Related**: requirement-sync-metadata-separation.md

---

## 🎯 Problem Statement

### Current Issue
The ecosystem-manager recycler uses an AI classifier (Ollama) to determine scenario progress from the final agent response. When the AI classifies work as "significant_progress", the recycler adds this note:

> "Already pretty good, but could use some additional validation/tidying"

**This creates a false sense of security** that slows down actual progress. The AI classifier can be overly optimistic or hallucinate completion when critical features are still missing or tests are failing.

### Impact
- Agents waste time on minor polish instead of implementing missing features
- Scenarios stall at 60-70% completeness thinking they're "almost done"
- Progress velocity drops significantly as agents focus on the wrong priorities

---

## 🎯 Proposed Solution

### Core Concept
**Ground the AI classifier with objective metrics** from the existing, validated requirement/operational target sync system. Use a **dual-classifier approach** where the final classification is the **more pessimistic** of:

1. **AI Classifier** (existing): Ollama analyzes final agent response
2. **Metrics Classifier** (new): Calculate completeness score from test results, requirement states, and operational target states

### Conservative Approach
```
final_classification = worse_of(ai_classification, metrics_classification)
```

This prevents false positives from either method while allowing either to catch real issues.

---

## 📊 Completeness Score Algorithm

### Score Calculation (0-100 scale)

#### Core Quality Metrics (70% weight)
These measure **actual validation**, not just existence:

| Metric | Weight | Calculation | Rationale |
|--------|--------|-------------|-----------|
| **Requirement Pass Rate** | 30% | `(passing_reqs / total_reqs) * 30` | Primary indicator of feature completeness |
| **Operational Target Pass Rate** | 20% | `(passing_targets / total_targets) * 20` | Validates business/functional goals |
| **Test Pass Rate** | 20% | `(passing_tests / total_tests) * 20` | Code quality and regression prevention |

#### Coverage Metrics (20% weight)
These ensure **depth of validation**:

| Metric | Weight | Calculation | Rationale |
|--------|--------|-------------|-----------|
| **Test Coverage Ratio** | 10% | `min(test_count / req_count, 2.0) * 5` | Ensures multiple tests per requirement (cap at 2x) |
| **Requirement Depth** | 10% | `min(avg_nesting_depth / 3.0, 1.0) * 10` | Deeper trees indicate thorough specification (cap at 3 levels) |

#### Quantity Metrics (10% weight)
These ensure **sufficient scope**:

| Metric | Weight | Calculation | Formula |
|--------|--------|-------------|---------|
| **Absolute Counts** | 10% | Scaled by scenario category thresholds | See Category Thresholds below |

### Quantity Scoring Formula

```javascript
function calculateQuantityScore(counts, category) {
  const thresholds = CATEGORY_THRESHOLDS[category] || CATEGORY_THRESHOLDS['utility'];

  const reqScore = min(counts.requirements / thresholds.requirements_good, 1.0) * 4;
  const targetScore = min(counts.targets / thresholds.targets_good, 1.0) * 3;
  const testScore = min(counts.tests / thresholds.tests_good, 1.0) * 3;

  return reqScore + targetScore + testScore; // Max 10 points
}
```

---

## 🏷️ Category-Specific Thresholds

Scenarios have different completion expectations based on their `category` field in `.vrooli/service.json`.

### Threshold Definitions

| Category | Type | Req (Ok/Good/Excellent) | Targets (Ok/Good/Excellent) | Tests (Ok/Good/Excellent) |
|----------|------|------------------------|----------------------------|---------------------------|
| **utility** | Simple tools | 10/15/25 | 8/12/20 | 15/25/40 |
| **business-application** | SaaS apps | 25/40/60 | 15/25/35 | 40/60/100 |
| **automation** | Orchestration | 15/25/40 | 12/18/28 | 25/40/70 |
| **platform** | Infrastructure | 30/50/80 | 20/30/45 | 50/80/120 |
| **developer_tools** | Dev tooling | 20/30/50 | 12/20/30 | 30/50/80 |

**Default**: Use `utility` thresholds for unknown categories

### Rationale
- **Utility tools** (picker-wheel, math-tools): Small, focused scope = fewer requirements
- **Business applications** (chore-tracking, invoice-generator): Complex UIs and business logic = more requirements
- **Automation** (deployment-manager): Orchestration logic = moderate requirements but high test needs
- **Platform** (token-economy, notification-hub): Core infrastructure = highest standards
- **Developer tools** (scenario-auditor, tidiness-manager): Specialized tooling = moderate complexity

---

## 📈 Classification Mapping

Map completeness scores to classification levels that align with AI classifier output:

| Score Range | Classification | Recycler Note Prefix | Meaning |
|-------------|----------------|---------------------|---------|
| **0-20** | `early_stage` | "Just starting - needs significant development" | Scaffolding only, minimal tests |
| **21-40** | `foundation_laid` | "Foundation laid - core features in progress" | Some features working, many gaps |
| **41-60** | `functional_incomplete` | "Functional but incomplete - needs more features/tests" | Core working, missing P1/P2 features |
| **61-80** | `mostly_complete` | "Mostly complete - needs refinement and validation" | Most features done, test gaps remain |
| **81-95** | `nearly_ready` | "Nearly ready - final polish and edge cases" | High quality, minor gaps |
| **96-100** | `production_ready` | "Production ready - excellent validation coverage" | Exceptional quality |

### Mapping to AI Classifications

When combining with AI classifier, use this equivalence:

| AI Classification | Equivalent Metric Classification | Pessimism Level |
|-------------------|----------------------------------|-----------------|
| `full_complete` | `production_ready` (96-100) | Most optimistic |
| `significant_progress` | `mostly_complete` (61-80) | Moderately optimistic |
| `some_progress` | `functional_incomplete` (41-60) | Neutral |
| `uncertain` | `early_stage` (0-20) | Most pessimistic |

**Conservative logic**:
```javascript
function combineClassifications(aiClass, metricsClass) {
  const pessimismOrder = [
    'early_stage',           // Most pessimistic
    'foundation_laid',
    'functional_incomplete',
    'mostly_complete',
    'nearly_ready',
    'production_ready'       // Most optimistic
  ];

  const aiIndex = pessimismOrder.indexOf(mapAItoMetrics(aiClass));
  const metricsIndex = pessimismOrder.indexOf(metricsClass);

  // Return whichever is more pessimistic (lower index)
  return pessimismOrder[Math.min(aiIndex, metricsIndex)];
}
```

---

## 🚨 Special Considerations

### 1. Test Staleness Detection

**Issue**: If tests haven't been run recently, the data is unreliable.

**Solution**: Add staleness warning without penalizing score:

```javascript
function checkStaleness(lastTestRun) {
  const hoursSinceTest = (Date.now() - Date.parse(lastTestRun)) / (1000 * 60 * 60);

  if (hoursSinceTest > 48) {
    return {
      warning: true,
      message: "⚠️  Test results are stale (>48h old). Run `make test` to refresh.",
      hoursStale: Math.floor(hoursSinceTest)
    };
  }

  return { warning: false };
}
```

**Current implementation**: The `drift-check.js` script already detects staleness by comparing `coverage/sync/*.json` timestamps against test phase results. Reuse this logic.

### 2. Early-Stage Scenarios (Bootstrap Mode)

**Issue**: A brand-new scenario with scaffolding but no requirements scores 0%, which is technically accurate but harsh.

**Decision**: **No special handling**. Zero is the correct score. This creates pressure to define requirements early, which is desirable behavior.

**Rationale**:
- Forces agents to think about requirements before writing code (good practice)
- Prevents "building in the dark" without validation criteria
- Aligns with test-driven development principles

### 3. Different Requirement Depths

**Issue**: 5 deeply-nested requirement trees might represent more work than 30 shallow ones.

**Solution**: Include **Requirement Depth Score** (10% weight):

```javascript
function calculateDepthScore(requirements) {
  const depths = requirements.map(req => getMaxDepth(req));
  const avgDepth = depths.reduce((a, b) => a + b, 0) / depths.length;

  // Reward depth up to 3 levels (perfect score at 3+)
  // 1 level = 3.3 pts, 2 levels = 6.6 pts, 3+ levels = 10 pts
  return Math.min(avgDepth / 3.0, 1.0) * 10;
}

function getMaxDepth(requirement) {
  if (!requirement.children || requirement.children.length === 0) {
    return 1;
  }
  return 1 + Math.max(...requirement.children.map(getMaxDepth));
}
```

### 4. Gaming Prevention

**Risk**: Agents might create trivial requirements to boost scores.

**Mitigations**:
1. **Quality >> Quantity** (70% weight on pass rates vs 10% on counts)
2. **Test Coverage Ratio** penalizes req inflation without corresponding tests
3. **Depth Score** rewards thoughtful requirement hierarchies
4. **Human review** of ecosystem-manager outputs catches obvious gaming

---

## 🛠️ Implementation Plan

### Phase 1: CLI Command (`vrooli scenario completeness`)

**Location**: `scripts/scenarios/lib/completeness.js`

**Command**:
```bash
vrooli scenario completeness <name> [--format json|human]
```

**Output (human format)**:
```
Scenario: deployment-manager
Category: automation
Completeness Score: 67/100 (Mostly complete)

Quality Metrics (47/70):
  ✅ Requirements: 42 total, 28 passing (67%) → 20/30 pts
  ✅ Op Targets: 18 total, 14 passing (78%) → 16/20 pts
  ⚠️  Tests: 85 total, 71 passing (84%) → 11/20 pts  [Target: 90%+]

Coverage Metrics (17/20):
  ✅ Test Coverage: 2.0x (85 tests / 42 req) → 10/10 pts
  ✅ Depth Score: 3.2 avg levels → 7/10 pts

Quantity Metrics (8/10):
  ✅ Requirements: 42 (Good for automation) → 4/4 pts
  ⚠️  Targets: 18 (Ok for automation) → 2/3 pts  [Target: 25+]
  ✅ Tests: 85 (Good for automation) → 2/3 pts

Warnings:
  ⚠️  Test results stale (72h old). Run `make test` to refresh.

Classification: functional_incomplete
Next Focus: Increase test pass rate (84% → 90%+) and add 7 more operational targets
```

**Output (JSON format)**:
```json
{
  "scenario": "deployment-manager",
  "category": "automation",
  "score": 67,
  "classification": "functional_incomplete",
  "breakdown": {
    "quality": {
      "score": 47,
      "max": 70,
      "requirement_pass_rate": { "passing": 28, "total": 42, "rate": 0.67, "points": 20 },
      "target_pass_rate": { "passing": 14, "total": 18, "rate": 0.78, "points": 16 },
      "test_pass_rate": { "passing": 71, "total": 85, "rate": 0.84, "points": 11 }
    },
    "coverage": {
      "score": 17,
      "max": 20,
      "test_coverage_ratio": { "ratio": 2.0, "points": 10 },
      "depth_score": { "avg_depth": 3.2, "points": 7 }
    },
    "quantity": {
      "score": 8,
      "max": 10,
      "requirements": { "count": 42, "threshold": "good", "points": 4 },
      "targets": { "count": 18, "threshold": "ok", "points": 2 },
      "tests": { "count": 85, "threshold": "good", "points": 2 }
    }
  },
  "warnings": [
    {
      "type": "staleness",
      "message": "Test results stale (72h old)",
      "action": "Run `make test` to refresh"
    }
  ],
  "recommendations": [
    "Increase test pass rate from 84% to 90%+",
    "Add 7 more operational targets to reach 'good' threshold (25)"
  ]
}
```

### Phase 2: Integration with ecosystem-manager Recycler

**Location**: `scenarios/ecosystem-manager/api/pkg/recycler/recycler.go`

**Changes**:

1. **Add completeness check** alongside AI summarizer:

```go
// In processCompletedTask(), before calling summarizer
func (r *Recycler) processCompletedTask(task *tasks.TaskItem, cfg settings.RecyclerSettings) error {
    output := extractOutput(task.Results)

    // Get AI classification (existing)
    aiResult, err := summarizer.GenerateNote(context.Background(), summarizer.Config{
        Provider: cfg.ModelProvider,
        Model:    cfg.ModelName,
    }, summarizer.Input{Output: output})
    if err != nil {
        aiResult = summarizer.DefaultResult()
    }

    // Get metrics classification (NEW)
    metricsResult, err := getCompletenessClassification(task.Name)
    if err != nil {
        log.Printf("Completeness check failed for %s: %v", task.Name, err)
        metricsResult = CompletenessResult{
            Classification: "uncertain",
            Score: 0,
            Breakdown: "Completeness check unavailable",
        }
    }

    // Combine classifications (take more pessimistic)
    finalClassification := combineClassifications(
        aiResult.Classification,
        metricsResult.Classification,
    )

    // Build combined note
    combinedNote := buildCompositeNote(aiResult, metricsResult, finalClassification)

    // Store in task results
    taskResults := tasks.FromMap(task.Results)
    taskResults.SetRecyclerInfo(finalClassification, now)
    taskResults.SetCompletenessInfo(metricsResult) // NEW
    task.Results = taskResults.ToMap()
    task.Notes = combinedNote

    // ... rest of existing logic
}
```

2. **Add helper function**:

```go
func getCompletenessClassification(scenarioName string) (CompletenessResult, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "vrooli", "scenario", "completeness", scenarioName, "--format", "json")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return CompletenessResult{}, fmt.Errorf("completeness command failed: %w", err)
    }

    var result CompletenessResult
    if err := json.Unmarshal(output, &result); err != nil {
        return CompletenessResult{}, fmt.Errorf("parse completeness JSON: %w", err)
    }

    return result, nil
}

func combineClassifications(aiClass, metricsClass string) string {
    pessimismOrder := []string{
        "early_stage",
        "foundation_laid",
        "functional_incomplete",
        "mostly_complete",
        "nearly_ready",
        "production_ready",
    }

    aiMapped := mapAIClassification(aiClass)

    aiIndex := indexOf(pessimismOrder, aiMapped)
    metricsIndex := indexOf(pessimismOrder, metricsClass)

    if aiIndex < 0 { aiIndex = 0 }  // Default to most pessimistic
    if metricsIndex < 0 { metricsIndex = 0 }

    return pessimismOrder[min(aiIndex, metricsIndex)]
}

func mapAIClassification(aiClass string) string {
    switch strings.ToLower(aiClass) {
    case "full_complete":
        return "production_ready"
    case "significant_progress":
        return "mostly_complete"
    case "some_progress":
        return "functional_incomplete"
    default:
        return "early_stage"
    }
}

func buildCompositeNote(aiResult summarizer.Result, metricsResult CompletenessResult, finalClass string) string {
    var builder strings.Builder

    // Classification header
    builder.WriteString(getClassificationPrefix(finalClass))
    builder.WriteString("\n\n")

    // Metrics summary
    builder.WriteString(fmt.Sprintf("**Completeness Score**: %d/100\n", metricsResult.Score))
    builder.WriteString(metricsResult.Breakdown)
    builder.WriteString("\n\n")

    // AI notes
    builder.WriteString("**AI Analysis**:\n")
    builder.WriteString(aiResult.Note)

    return builder.String()
}
```

### Phase 3: Data Structures

**New types** in `scenarios/ecosystem-manager/api/pkg/recycler/completeness.go`:

```go
type CompletenessResult struct {
    Scenario       string              `json:"scenario"`
    Category       string              `json:"category"`
    Score          int                 `json:"score"`
    Classification string              `json:"classification"`
    Breakdown      CompletenessDetails `json:"breakdown"`
    Warnings       []Warning           `json:"warnings"`
    Recommendations []string           `json:"recommendations"`
}

type CompletenessDetails struct {
    Quality  QualityMetrics  `json:"quality"`
    Coverage CoverageMetrics `json:"coverage"`
    Quantity QuantityMetrics `json:"quantity"`
}

type QualityMetrics struct {
    Score              int         `json:"score"`
    Max                int         `json:"max"`
    RequirementPassRate PassRate   `json:"requirement_pass_rate"`
    TargetPassRate      PassRate   `json:"target_pass_rate"`
    TestPassRate        PassRate   `json:"test_pass_rate"`
}

type PassRate struct {
    Passing int     `json:"passing"`
    Total   int     `json:"total"`
    Rate    float64 `json:"rate"`
    Points  int     `json:"points"`
}

type Warning struct {
    Type    string `json:"type"`
    Message string `json:"message"`
    Action  string `json:"action,omitempty"`
}
```

---

## 🧪 Testing Strategy

### Unit Tests

**File**: `scripts/scenarios/lib/completeness.test.js`

Test cases:
1. **Score calculation accuracy**: Verify math for all 6 metrics
2. **Category threshold lookup**: Ensure correct thresholds applied per category
3. **Classification mapping**: Verify score → classification mapping
4. **Staleness detection**: Check timestamp comparison logic
5. **Depth calculation**: Test recursive depth scoring with various tree structures
6. **Edge cases**: Empty scenario, no tests, 100% pass rate, etc.

### Integration Tests

**File**: `scenarios/ecosystem-manager/test/cli/completeness-integration.bats`

Test cases:
1. Run `vrooli scenario completeness` on a real scenario (deployment-manager)
2. Verify JSON output structure matches schema
3. Verify human output is readable and includes all sections
4. Test classification combination logic with mock AI results
5. Test recycler integration end-to-end with a test scenario

### Manual Validation

Before deploying to ecosystem-manager:
1. Run completeness check on 5-10 scenarios across different categories
2. Compare scores with human assessment of actual completeness
3. Tune thresholds if scores feel consistently off
4. Validate that agents respond appropriately to new classifications

---

## 📊 Success Metrics

### Quantitative Goals
- **Accuracy**: Completeness score should correlate ≥0.8 with human assessment (test on 20 scenarios)
- **Performance**: Completeness check completes in <5s for scenarios with ≤100 requirements
- **Progress velocity**: Ecosystem-manager scenarios should reach 80%+ completeness 30% faster

### Qualitative Goals
- Agents stop getting stuck in "polish mode" when features are missing
- Clearer actionable feedback in recycler notes
- Ecosystem-manager becomes more reliable at building complete scenarios

### Monitoring
- Track completeness score progression over time for each scenario
- Log AI vs metrics classification disagreements for analysis
- Monitor agent behavior changes in response to new notes

---

## 🔄 Future Enhancements

### Phase 4: Progress Velocity Tracking
- Store completeness score history in database
- Calculate score delta between recycler runs
- Bonus classification boost if velocity is high (e.g., +10 pts in one iteration)

### Phase 5: Adaptive Thresholds
- After 10+ scenarios per category, train a model to predict expected requirement counts
- Auto-calibrate thresholds based on PRD complexity analysis (word count, section count)
- Scenario-specific overrides in `.vrooli/service.json`

### Phase 6: Requirement Quality Scoring
- Detect trivial requirements (e.g., "must have a UI")
- Weight requirements by depth, acceptance criteria count, linked tests
- Penalize low-quality requirement inflation

---

## 📝 Open Questions

1. **Should we factor in PRD complexity?** You mentioned README might be better than PRD sections since PRD structure is standardized. How should we measure PRD/README complexity?
   - **Proposed**: Count operational targets in PRD.md as a proxy for scope. More targets = more complex scenario.

2. **What about scenarios without UIs?** (e.g., CLI-only tools, libraries)
   - **Proposed**: Category thresholds already handle this. CLI tools likely fall under "utility" or "developer_tools" with lower thresholds.

3. **How to handle cross-scenario validations?** (e.g., BAS workflows testing deployment-manager)
   - **Proposed**: External validations already tracked in requirement schema via `scenario` field. Include in test counts.

4. **Should we penalize stale test results?** Currently we warn but don't reduce score.
   - **Decision**: No penalty. Warning is sufficient. Agents are already instructed to run tests regularly.

---

## ✅ Implementation Checklist

- [ ] Create `scripts/scenarios/lib/completeness.js` with scoring algorithm
- [ ] Add category threshold definitions to new config file
- [ ] Implement `vrooli scenario completeness` CLI command
- [ ] Add unit tests for scoring logic
- [ ] Add integration tests for CLI command
- [ ] Create `CompletenessResult` types in ecosystem-manager
- [ ] Implement `getCompletenessClassification()` helper
- [ ] Implement `combineClassifications()` logic
- [ ] Update `processCompletedTask()` to use dual-classifier approach
- [ ] Update recycler note formatting to include completeness breakdown
- [ ] Test on 5 real scenarios and validate output
- [ ] Deploy to ecosystem-manager and monitor behavior
- [ ] Document in ecosystem-manager README
- [ ] Add to `vrooli help` output

---

**Next Steps**: Review this plan, provide feedback on open questions, then proceed with Phase 1 implementation.
