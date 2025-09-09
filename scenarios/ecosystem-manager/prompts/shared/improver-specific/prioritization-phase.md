# Prioritization Phase for Improvers

## Purpose
After assessment reveals the true state, prioritization ensures you work on the most valuable improvements. Not all improvements are equal - some unlock massive value, others are cosmetic.

## Prioritization Allocation: 20% of Total Effort

### Prioritization Philosophy

**Maximum Value, Minimum Effort**
- High impact, low effort = DO FIRST
- High impact, high effort = PLAN CAREFULLY
- Low impact, low effort = BATCH TOGETHER
- Low impact, high effort = SKIP

## Multi-Factor Scoring System

### Factor 1: Business Value (Weight: 40%)

Score each improvement 1-10:
```markdown
10 - Enables new revenue stream
9  - Unblocks multiple customers
8  - Solves critical user pain point
7  - Significantly improves user experience
6  - Moderate user experience improvement
5  - Nice to have feature
4  - Minor enhancement
3  - Cosmetic improvement
2  - Internal only benefit
1  - Negligible value
```

### Factor 2: Technical Impact (Weight: 30%)

Score each improvement 1-10:
```markdown
10 - Unblocks multiple other improvements
9  - Fixes critical system issue
8  - Resolves security vulnerability
7  - Major performance improvement
6  - Improves reliability/stability
5  - Better error handling
4  - Code quality improvement
3  - Technical debt reduction
2  - Minor optimization
1  - Cosmetic code change
```

### Factor 3: Implementation Effort (Weight: 20%)

Score INVERSE (high score = low effort):
```markdown
10 - Under 30 minutes
9  - 30-60 minutes
8  - 1-2 hours
7  - 2-4 hours
6  - 4-8 hours (half day)
5  - 1 day
4  - 2 days
3  - 3-5 days
2  - 1 week
1  - Over 1 week
```

### Factor 4: Risk Level (Weight: 10%)

Score INVERSE (high score = low risk):
```markdown
10 - No risk, isolated change
9  - Minimal risk, well-tested area
8  - Low risk, clear rollback path
7  - Some risk, but manageable
6  - Moderate risk, needs careful testing
5  - Notable risk, affects core functionality
4  - High risk, affects multiple systems
3  - Very high risk, could break dependencies
2  - Critical risk, affects production
1  - Extreme risk, could cause data loss
```

### Calculate Priority Score
```python
priority_score = (
    (business_value * 0.4) +
    (technical_impact * 0.3) +
    (effort_score * 0.2) +
    (risk_score * 0.1)
)

# Example:
# Feature: Add authentication to API
# Business: 9 (unblocks customers) * 0.4 = 3.6
# Technical: 8 (security fix) * 0.3 = 2.4
# Effort: 6 (4-8 hours) * 0.2 = 1.2
# Risk: 7 (manageable) * 0.1 = 0.7
# Total: 7.9 (HIGH PRIORITY)
```

## PRD Requirement Prioritization

### P0 Requirements (Must Have)
These always get priority, but still score them:
```markdown
P0 requirements are launch blockers:
- Without these, the component doesn't work
- Users can't achieve core value
- System may be insecure or unstable

Even among P0s, prioritize by:
1. Security issues
2. Data integrity
3. Core functionality
4. User-facing features
```

### P1 Requirements (Should Have)
Score these normally:
```markdown
P1 requirements are important:
- Significantly improve user experience
- Enable important integrations
- Improve system reliability

Prioritize P1s that:
- Unblock other improvements
- Add measurable value
- Users actively request
```

### P2 Requirements (Nice to Have)
Only if time permits:
```markdown
P2 requirements are enhancements:
- Polish and refinement
- Advanced features
- Optimizations

Consider P2s only when:
- All P0s complete
- Critical P1s done
- High value/effort ratio
```

## Cross-Scenario Impact Analysis

### Force Multipliers
Prioritize improvements that benefit multiple scenarios:
```markdown
## Cross-Scenario Value
High Priority:
- Shared API improvements
- Common resource enhancements
- Reusable component creation
- Pattern standardization

Example:
Improving ollama resource health checks
- Benefits: 20+ scenarios
- Priority multiplier: 3x
```

### Dependency Chains
Identify improvements that unblock others:
```markdown
## Dependency Analysis
Improvement A enables → B, C, D
Therefore: A priority = A + (B+C+D) * 0.3

Example:
Fix authentication (A) enables:
- User management (B)
- Role permissions (C)  
- API security (D)
Adjusted priority: Higher
```

## Queue Selection Strategy

### The Selection Algorithm
```bash
# From assessment report, for each improvement:
1. Calculate priority score
2. Sort by score descending
3. Check dependencies
4. Consider current queue
5. Select top item that:
   - Has highest score
   - Dependencies are met
   - Not blocked by in-progress work
   - Fits in time budget
```

### Time Budget Consideration
```markdown
## Iteration Time Budget
Available time: 4 hours

Option A: One big improvement (3.5 hours, score 8.5)
Option B: Three small improvements (1.2 hours each, scores 7.8, 7.5, 7.2)

Choose B if:
- Combined value > single value
- Lower risk profile
- Better progress visibility

Choose A if:
- Critical blocker
- Enables many improvements
- Time-sensitive
```

## Prioritization Output

### Required: Prioritized Improvement List
```markdown
# Prioritized Improvements

## Selected for This Iteration
1. **Add API Authentication** [Score: 8.7]
   - P0 Requirement
   - Effort: 4 hours
   - Impact: Unblocks 5 features
   - Risk: Manageable

## Next Iteration Candidates
2. **Implement Search Pagination** [Score: 7.9]
   - P1 Requirement
   - Effort: 2 hours
   - Impact: Major UX improvement

3. **Add Input Validation** [Score: 7.5]
   - P0 Requirement
   - Effort: 3 hours
   - Impact: Security improvement

## Backlog (Lower Priority)
4. **Optimize Database Queries** [Score: 6.2]
5. **Add Logging Framework** [Score: 5.8]
6. **Update UI Colors** [Score: 3.1]

## Blocked/Deferred
- **Add Payment Processing** [Blocked by: Authentication]
- **Implement Webhooks** [Deferred: Needs architecture decision]
```

## Common Prioritization Mistakes

### Feature Bias
❌ **Bad**: Always choosing new features
✅ **Good**: Balancing features, fixes, and foundation

### Effort Bias
❌ **Bad**: Only doing easy things
✅ **Good**: Mix of quick wins and important work

### Recency Bias
❌ **Bad**: Working on latest request
✅ **Good**: Systematic scoring and selection

### Perfectionism
❌ **Bad**: Trying to fix everything
✅ **Good**: Focused, high-value improvements

## Prioritization Decision Tree

```
Is it P0 and broken?
├── Yes → DO FIRST
└── No → Continue
    │
    Is it blocking other work?
    ├── Yes → HIGH PRIORITY
    └── No → Continue
        │
        Does it affect multiple scenarios?
        ├── Yes → INCREASE PRIORITY 2x
        └── No → Continue
            │
            Score > 7?
            ├── Yes → ADD TO QUEUE
            └── No → BACKLOG
```

## Remember for Prioritization

**Value over volume** - One high-impact improvement beats ten minor ones

**Unblock first** - Remove bottlenecks to enable more improvements

**Quick wins matter** - Build momentum with fast victories

**Risk awareness** - Don't break what works

**Time box** - Respect iteration boundaries

Prioritization is about making smart choices. Every improvement has an opportunity cost - make sure you're choosing the ones that deliver maximum value to Vrooli and its users.