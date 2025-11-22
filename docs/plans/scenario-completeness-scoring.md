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
**Replace AI-based classification with objective metrics** from the existing, validated requirement/operational target sync system.

**Old approach** (current):
- Ollama analyzes final agent response → generates classification + note
- Classification drives recycler behavior (finalization, requeue)
- Prone to hallucination and false positives

**New approach**:
- **Metrics system** calculates completeness score → generates classification + note prefix
- **Ollama** summarizes agent output → generates note content only (no classification)
- **Metrics classification** drives recycler behavior exclusively

### Why This is Better
```
OLD: AI hallucinates "significant_progress" → misleading note → agent wastes time polishing
NEW: Metrics show 45% complete → clear gaps identified → agent focuses on missing features
```

This eliminates false positives entirely by grounding classification in test results, requirement status, and operational target completion.

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

### Threshold Configuration

**Location**: `scripts/scenarios/lib/completeness-thresholds.json`

This separate config file makes threshold tuning easy without touching core logic.

```json
{
  "version": "1.0.0",
  "default_category": "utility",
  "categories": {
    "utility": {
      "description": "Simple tools (picker-wheel, math-tools)",
      "requirements": { "ok": 10, "good": 15, "excellent": 25 },
      "targets": { "ok": 8, "good": 12, "excellent": 20 },
      "tests": { "ok": 15, "good": 25, "excellent": 40 }
    },
    "business-application": {
      "description": "SaaS apps (chore-tracking, invoice-generator)",
      "requirements": { "ok": 25, "good": 40, "excellent": 60 },
      "targets": { "ok": 15, "good": 25, "excellent": 35 },
      "tests": { "ok": 40, "good": 60, "excellent": 100 }
    },
    "automation": {
      "description": "Orchestration (deployment-manager, tidiness-manager)",
      "requirements": { "ok": 15, "good": 25, "excellent": 40 },
      "targets": { "ok": 12, "good": 18, "excellent": 28 },
      "tests": { "ok": 25, "good": 40, "excellent": 70 }
    },
    "platform": {
      "description": "Infrastructure (token-economy, notification-hub)",
      "requirements": { "ok": 30, "good": 50, "excellent": 80 },
      "targets": { "ok": 20, "good": 30, "excellent": 45 },
      "tests": { "ok": 50, "good": 80, "excellent": 120 }
    },
    "developer_tools": {
      "description": "Dev tooling (scenario-auditor, browser-automation-studio)",
      "requirements": { "ok": 20, "good": 30, "excellent": 50 },
      "targets": { "ok": 12, "good": 20, "excellent": 30 },
      "tests": { "ok": 30, "good": 50, "excellent": 80 }
    }
  }
}
```

### Threshold Summary Table

| Category | Type | Req (Ok/Good/Excellent) | Targets (Ok/Good/Excellent) | Tests (Ok/Good/Excellent) |
|----------|------|------------------------|----------------------------|---------------------------|
| **utility** | Simple tools | 10/15/25 | 8/12/20 | 15/25/40 |
| **business-application** | SaaS apps | 25/40/60 | 15/25/35 | 40/60/100 |
| **automation** | Orchestration | 15/25/40 | 12/18/28 | 25/40/70 |
| **platform** | Infrastructure | 30/50/80 | 20/30/45 | 50/80/120 |
| **developer_tools** | Dev tooling | 20/30/50 | 12/20/30 | 30/50/80 |

**Default**: Use `utility` thresholds for unknown categories

### Adjusting Thresholds

To recalibrate:
1. Edit `scripts/scenarios/lib/completeness-thresholds.json`
2. No code changes needed - values are loaded at runtime
3. Test with `vrooli scenario completeness <name>` to validate
4. Commit threshold changes separately from logic changes

---

## 📈 Classification Mapping

Map completeness scores to classification levels that **replace** the AI-based classifications:

| Score Range | Classification | Recycler Note Prefix | Recycler Behavior | Meaning |
|-------------|----------------|---------------------|-------------------|---------|
| **0-20** | `early_stage` | "**Score: {score}/100** - Just starting, needs significant development" | Reset completion claims to 0 | Scaffolding only, minimal tests |
| **21-40** | `foundation_laid` | "**Score: {score}/100** - Foundation laid, core features in progress" | Reset completion claims to 0 | Some features working, many gaps |
| **41-60** | `functional_incomplete` | "**Score: {score}/100** - Functional but incomplete, needs more features/tests" | Reset completion claims to 0 | Core working, missing P1/P2 features |
| **61-80** | `mostly_complete` | "**Score: {score}/100** - Mostly complete, needs refinement and validation" | Reset completion claims to 0 | Most features done, test gaps remain |
| **81-95** | `nearly_ready` | "**Score: {score}/100** - Nearly ready, final polish and edge cases" | **+0.5 to completion claims** (6 consecutive → finalize) | High quality, minor gaps |
| **96-100** | `production_ready` | "**Score: {score}/100** - Production ready, excellent validation coverage" | **+1 to completion claims** (3 consecutive → finalize) | Exceptional quality |

### Recycler Logic Changes

**OLD** (broken binary logic):
```go
switch classification {
case "full_complete":
    task.ConsecutiveCompletionClaims++
default:
    task.ConsecutiveCompletionClaims = 0
}
```

**NEW** (graduated progress with fractional increments):
```go
switch classification {
case "production_ready":
    task.ConsecutiveCompletionClaims++       // 96-100: full credit (3x → finalize)
case "nearly_ready":
    task.ConsecutiveCompletionClaims += 0.5  // 81-95: half credit (6x → finalize)
default:
    task.ConsecutiveCompletionClaims = 0     // <81: no progress toward finalization
}
```

**Why fractional increments?**
- Scenarios at 85% complete are genuinely close to done - should get partial credit
- Prevents frustration of "reset to 0" when making real progress
- Still requires sustained high quality (6 consecutive 81-95 scores = finalized)
- Incentivizes pushing to 96+ for faster finalization (3x vs 6x)

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

### Phase 2: Update Ollama Prompt to Remove Classification

**Location**: `scenarios/ecosystem-manager/api/pkg/summarizer/prompt_template.txt`

**Changes**: Remove classification requirement from prompt. New prompt should only extract structured notes without making completion judgments.

**Location**: `scenarios/ecosystem-manager/api/pkg/summarizer/summarizer.go`

Update `parseResult()` and `decorateNote()` to extract note sections without classification. The note should be raw facts only - completeness scoring will add the classification prefix.

### Phase 3: Integration with ecosystem-manager Recycler

**Location**: `scenarios/ecosystem-manager/api/pkg/recycler/recycler.go`

**Changes**: Replace AI classification with metrics classification as the source of truth:

```go
// In processCompletedTask()
func (r *Recycler) processCompletedTask(task *tasks.TaskItem, cfg settings.RecyclerSettings) error {
    output := extractOutput(task.Results)
    now := timeutil.NowRFC3339()

    // Get AI summary (notes only, NO classification)
    var aiNote string
    if strings.TrimSpace(output) != "" {
        result, err := summarizer.GenerateNote(context.Background(), summarizer.Config{
            Provider: cfg.ModelProvider,
            Model:    cfg.ModelName,
        }, summarizer.Input{Output: output})
        if err != nil {
            log.Printf("Recycler summarizer error for task %s: %v", task.ID, err)
            aiNote = "Unable to generate summary"
        } else {
            aiNote = result.Note
        }
    }

    // Get metrics classification (NEW - this is the source of truth)
    metricsResult, err := getCompletenessClassification(task.Name)
    if err != nil {
        log.Printf("Completeness check failed for %s: %v", task.Name, err)
        // Fallback: treat as early stage if metrics unavailable
        metricsResult = CompletenessResult{
            Classification: "early_stage",
            Score: 0,
            Breakdown: "⚠️  Completeness check unavailable - metrics could not be calculated",
            Recommendations: []string{"Check that requirements are defined", "Verify test results are not stale"},
        }
    }

    // Build composite note with metrics prefix
    compositeNote := buildCompositeNote(metricsResult, aiNote)

    // Store in task results
    taskResults := tasks.FromMap(task.Results)
    taskResults.SetRecyclerInfo(metricsResult.Classification, now)  // Store new classification
    taskResults.SetCompletenessInfo(metricsResult)
    task.Results = taskResults.ToMap()

    // NEW: Use metrics classification directly (no legacy mapping)
    classification := strings.ToLower(metricsResult.Classification)
    switch classification {
    case "production_ready":
        task.ConsecutiveCompletionClaims++       // 96-100: full increment
    case "nearly_ready":
        task.ConsecutiveCompletionClaims += 0.5  // 81-95: partial increment
    default:
        task.ConsecutiveCompletionClaims = 0     // <81: reset
    }
    task.ConsecutiveFailures = 0

    task.Notes = compositeNote
    task.UpdatedAt = now

    // Check finalization threshold (unchanged)
    if shouldFinalize(task.ConsecutiveCompletionClaims, cfg.CompletionThreshold) {
        // ... finalize task
    }

    // Otherwise requeue (unchanged)
    // ... requeue logic
}
```

**Helper functions**:

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

func buildCompositeNote(metricsResult CompletenessResult, aiNote string) string {
    var builder strings.Builder

    // Metrics-driven classification prefix (replaces old AI prefix)
    builder.WriteString(getClassificationPrefix(metricsResult.Classification, metricsResult.Score))
    builder.WriteString("\n\n")

    // Completeness breakdown
    builder.WriteString(metricsResult.Breakdown)
    builder.WriteString("\n\n")

    // Recommendations (actionable next steps from metrics)
    if len(metricsResult.Recommendations) > 0 {
        builder.WriteString("**Priority Actions:**\n")
        for _, rec := range metricsResult.Recommendations {
            builder.WriteString("- ")
            builder.WriteString(rec)
            builder.WriteString("\n")
        }
        builder.WriteString("\n")
    }

    // AI notes from last run (factual summary only, no completion judgment)
    builder.WriteString("**Notes from Last Run:**\n")
    builder.WriteString(aiNote)

    return builder.String()
}

func getClassificationPrefix(classification string, score int) string {
    switch classification {
    case "production_ready":
        return fmt.Sprintf("**Score: %d/100** - Production ready, excellent validation coverage", score)
    case "nearly_ready":
        return fmt.Sprintf("**Score: %d/100** - Nearly ready, final polish and edge cases", score)
    case "mostly_complete":
        return fmt.Sprintf("**Score: %d/100** - Mostly complete, needs refinement and validation", score)
    case "functional_incomplete":
        return fmt.Sprintf("**Score: %d/100** - Functional but incomplete, needs more features/tests", score)
    case "foundation_laid":
        return fmt.Sprintf("**Score: %d/100** - Foundation laid, core features in progress", score)
    case "early_stage":
        return fmt.Sprintf("**Score: %d/100** - Just starting, needs significant development", score)
    default:
        return fmt.Sprintf("**Score: %d/100** - Status unclear", score)
    }
}
```

### Phase 4: Data Structures

**New types** in `scenarios/ecosystem-manager/api/pkg/recycler/completeness.go`:

```go
type CompletenessResult struct {
    Scenario         string              `json:"scenario"`
    Category         string              `json:"category"`
    Score            int                 `json:"score"`
    Classification   string              `json:"classification"`        // production_ready, nearly_ready, mostly_complete, etc.
    Breakdown        string              `json:"breakdown"`             // Human-readable summary
    BreakdownDetails CompletenessDetails `json:"breakdown_details"`     // Structured data
    Warnings         []Warning           `json:"warnings"`
    Recommendations  []string            `json:"recommendations"`
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

## 📝 Implementation Decisions

### 1. Scenario Complexity Estimation

**Question**: How do we estimate expected scope for a scenario to calibrate thresholds?

**Decision**: Use **operational target count** from PRD.md as the primary complexity signal.

**Rationale**:
- Operational targets (OT-P0-001, OT-P1-002, etc.) represent concrete deliverables
- More targets = more complex scenario
- PRD structure is standardized, making parsing reliable
- Already exists for all scenarios (required by standards)

**Alternative considered**: README analysis (word count, section depth)
- **Rejected**: README content varies too much in quality and completeness
- Early-stage scenarios may have minimal READMEs
- PRD operational targets are more reliable

**Usage**:
- This is for **future Phase 5** (adaptive thresholds)
- Current implementation uses static category-based thresholds
- When ready to make thresholds adaptive, parse `## 🎯 Operational Targets` section from PRD.md

### 2. Scenarios Without UIs

**Question**: How do we handle CLI-only tools, libraries, APIs without UIs?

**Decision**: Category thresholds already account for this.

**Mapping**:
- CLI utilities → `utility` category (lower thresholds)
- Developer libraries → `developer_tools` category
- APIs/microservices → `automation` or `platform` (depending on scope)

No special handling needed.

### 3. Cross-Scenario Validations

**Question**: How to count BAS workflows that test other scenarios (e.g., BAS testing deployment-manager)?

**Decision**: Include in test counts via `scenario` field in requirement validation entries.

**Current schema support**:
```json
{
  "type": "automation",
  "scenario": "browser-automation-studio",
  "workflow_id": "test-deployment-manager",
  "status": "implemented"
}
```

These are already parsed and counted as validations. No changes needed.

### 4. Stale Test Result Penalty

**Question**: Should we reduce the score if test results are stale (>48h old)?

**Decision**: **No penalty**. Warning only.

**Rationale**:
- Agents are already instructed to run tests regularly
- Penalizing would create false negatives during legitimate long-running work
- Warning is sufficient to surface the issue
- If truly stale, the agent will see the warning and run tests

---

## ✅ Implementation Checklist

### Phase 1: CLI Command Foundation
- [ ] Create `scripts/scenarios/lib/completeness-thresholds.json` config file
- [ ] Create `scripts/scenarios/lib/completeness.js` with scoring algorithm
  - [ ] Implement `loadThresholds()` to read config file
  - [ ] Implement `calculateQualityScore()` (requirement/target/test pass rates)
  - [ ] Implement `calculateCoverageScore()` (test coverage ratio + depth)
  - [ ] Implement `calculateQuantityScore()` (threshold-based scaling)
  - [ ] Implement `calculateCompletenessScore()` (combines all 3)
  - [ ] Implement `classifyScore()` (maps score → classification)
  - [ ] Implement `checkStaleness()` (reuse drift-check logic)
  - [ ] Implement `generateRecommendations()` (actionable next steps)
- [ ] Implement `vrooli scenario completeness` CLI command
  - [ ] Add to `scripts/scenarios/index.sh` or equivalent
  - [ ] Support `--format json|human` flag
  - [ ] Human output: color-coded, readable, includes breakdown
  - [ ] JSON output: structured, machine-parseable
- [ ] Add unit tests for scoring logic (`scripts/scenarios/lib/completeness.test.js`)
  - [ ] Test score calculation accuracy for all 6 metrics
  - [ ] Test category threshold lookup
  - [ ] Test classification mapping (score → classification)
  - [ ] Test edge cases (empty scenario, 100% pass rate, etc.)
- [ ] Add integration test (`test/cli/completeness.bats`)
  - [ ] Test on a real scenario (deployment-manager)
  - [ ] Verify JSON output structure
  - [ ] Verify human output readability

### Phase 2: Simplify Ollama Summarizer
- [ ] Update `scenarios/ecosystem-manager/api/pkg/summarizer/prompt_template.txt`
  - [ ] Remove classification requirement from prompt
  - [ ] Focus on extracting factual summaries only (accomplished, status, issues, next actions, evidence)
  - [ ] Remove completion judgment language
- [ ] Update `scenarios/ecosystem-manager/api/pkg/summarizer/summarizer.go`
  - [ ] Modify `parseResult()` to parse structured sections instead of classification
  - [ ] Remove `decorateNote()` function (metrics will handle prefixes)
  - [ ] Update `Result` struct to remove `Classification` field
  - [ ] Update `DefaultResult()` to return notes-only result

### Phase 3: ecosystem-manager Recycler Integration
- [ ] Create `scenarios/ecosystem-manager/api/pkg/recycler/completeness.go`
  - [ ] Define `CompletenessResult` struct (no legacy field needed)
  - [ ] Implement `getCompletenessClassification(scenarioName)` (calls CLI)
  - [ ] Implement `buildCompositeNote(metricsResult, aiNote)`
  - [ ] Implement `getClassificationPrefix(classification, score)`
- [ ] Update `scenarios/ecosystem-manager/api/pkg/recycler/recycler.go`
  - [ ] Import completeness package
  - [ ] Modify `processCompletedTask()` to:
    - [ ] Get AI note (no classification)
    - [ ] Get metrics classification (source of truth)
    - [ ] Build composite note with metrics prefix
    - [ ] **Replace switch statement** to use new classifications directly
    - [ ] **Add fractional increment logic** for `nearly_ready` (+0.5)
  - [ ] Store completeness result in task results
  - [ ] Update `SetRecyclerInfo()` call to store new classification (not legacy)
- [ ] Update `scenarios/ecosystem-manager/api/pkg/tasks/results.go`
  - [ ] Add `SetCompletenessInfo()` method
  - [ ] Add completeness fields to results map

### Phase 4: Testing & Validation
- [ ] Test completeness CLI on 5 diverse scenarios:
  - [ ] 1 utility (math-tools or picker-wheel)
  - [ ] 1 business-application (chore-tracking or invoice-generator)
  - [ ] 1 automation (deployment-manager or tidiness-manager)
  - [ ] 1 platform (token-economy or notification-hub)
  - [ ] 1 developer_tools (scenario-auditor or browser-automation-studio)
- [ ] Validate scores correlate with human assessment (≥0.8 correlation)
- [ ] Test ecosystem-manager integration end-to-end
  - [ ] Create test task, run through recycler
  - [ ] Verify completeness check runs successfully
  - [ ] Verify metrics classification drives recycler behavior directly
  - [ ] Verify composite note format (score prefix + breakdown + recommendations + AI notes)
  - [ ] Verify fractional increments work for `nearly_ready` (0.5 per iteration)
  - [ ] Verify finalization occurs after 3x `production_ready` or 6x `nearly_ready`
- [ ] Monitor agent behavior changes (do agents focus on right priorities?)

### Phase 5: Documentation & Deployment
- [ ] Document completeness scoring in `scripts/scenarios/README.md`
- [ ] Document CLI command in `vrooli help` output
- [ ] Document recycler changes in `scenarios/ecosystem-manager/README.md`
- [ ] Add example output to documentation
- [ ] Deploy to ecosystem-manager production
- [ ] Monitor first 3-5 tasks for unexpected behavior

### Future Phases (Documented but Not Implemented Yet)
- [ ] **Phase 5**: Adaptive thresholds based on PRD operational target count
- [ ] **Phase 6**: Progress velocity tracking (score deltas over time)
- [ ] **Phase 7**: Requirement quality scoring (detect trivial requirements)

---

## 🚀 Key Architecture Changes

### **What Changed from Original Plan**

**Before** (dual-classifier approach):
- AI classifier generates classification + note
- Metrics classifier generates classification + breakdown
- Combine both using pessimistic selection
- Both influence final behavior

**After** (metrics-only classification):
- **Metrics system** generates classification + breakdown (source of truth)
- **AI summarizer** generates note content only (no classification)
- Classification driven **entirely by objective metrics**
- Cleaner separation of concerns

### **Why This Is Better**

1. **Simpler Logic**: No need to combine classifications or map between AI/metrics formats
2. **Eliminates Confusion**: Only one source of truth for classification
3. **Clearer Responsibilities**:
   - Metrics: "What is the objective completeness score?"
   - AI: "What happened in the last run?"
4. **Easier to Tune**: Change threshold config without touching AI prompts
5. **More Reliable**: Can't have AI hallucinate "significant_progress" when tests are failing

### **Breaking Changes to Recycler**

This implementation **completely replaces** the existing classification system:

**Removed**:
- AI-based classifications (`full_complete`, `significant_progress`, `some_progress`, `uncertain`)
- Binary completion logic (only `full_complete` counted)
- Misleading note decoration ("Already pretty good..." prefix)

**Added**:
- 6 metrics-based classifications tied to objective scores
- Graduated completion logic (full credit at 96+, partial at 81-95)
- Actionable note format (score + breakdown + recommendations + AI summary)

**Impact**:
- Tasks in queue will get re-classified on next recycler pass
- Existing `ConsecutiveCompletionClaims` counters remain valid (may need adjustment based on new classifications)
- No migration needed - new system takes over immediately

## 🚀 Ready to Implement

The plan is complete and addresses all requirements:

✅ **Weighting**: 70% quality, 20% coverage, 10% quantity
✅ **Category thresholds**: Separate JSON config for easy tuning
✅ **Complexity metric**: Operational target count (for future adaptive thresholds)
✅ **Staleness**: Warning only, no penalty
✅ **Single source of truth**: Metrics classification drives all behavior
✅ **AI role simplified**: Factual summarization only, no completion judgment

**Next step**: Begin Phase 1 implementation starting with threshold config file and core scoring algorithm.
