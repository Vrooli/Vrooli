# Interpreting Reports

## Report Structure

A validation report for a reference scenario contains:

1. **Per-connection results**: Pass/fail for each skill's expectations
2. **Drift status**: Which skills have changed since connection
3. **Overlaps**: Where multiple skills target the same area
4. **Conflicts**: Where skills have mutually exclusive expectations
5. **Unconfigured skills**: Connected but without expectations
6. **Summary statistics**: Aggregate counts

## Reading Per-Connection Results

Each connection shows:

```
api-steer (v49, no drift)
  Structural: 5/5 pass, 0 fail, 1 skip
  CLI Tools:  3/3 pass, 0 fail
  ✅ All expectations satisfied
```

### Failure Examples

```
storage-steer (v12, DRIFT DETECTED)
  Structural: 3/4 pass, 1 fail, 0 skip
  CLI Tools:  2/2 pass
  ❌ FAIL: Folder "api/repositories/" not found
    Expected: api/repositories/ (required)
    Description: Repository pattern requires dedicated directory
```

**What to do with failures**:

1. **Is the reference wrong?** If the reference scenario doesn't follow the skill's guidance, update the reference.
2. **Is the expectation wrong?** If the expectation doesn't accurately reflect the skill's guidance, update the expectation.
3. **Is the skill wrong?** If the skill's guidance is unreasonable, flag it for the meta optimization team.

### Drift Warnings

```
cli-steer (v15, DRIFT DETECTED)
  ⚠️  Skill content changed since connection (was v12, now v15)
  Structural: 4/4 pass
  CLI Tools:  1/1 pass
  ⚠️  Expectations may be stale — review skill changes
```

**What to do with drift**:
1. Read the skill's current content (use `prompt-manager skill read cli-steer`)
2. Compare with what your expectations check
3. If expectations still match → refresh the version pin
4. If expectations are stale → update expectations, then refresh

## Reading Overlaps

Overlaps show where multiple skills care about the same area:

```
Overlaps Detected: 3

  api/handlers/
    Skills: api-steer, unit-testing-architecture-steer, screaming-architecture-audit
    Expectations:
      - api-steer: folder "api/handlers/{domain}/" (domain module organization)
      - unit-testing-architecture-steer: file "api/handlers/**/*_test.go" (co-located tests)
      - screaming-architecture-audit: folder "api/handlers/" (scream the architecture)
    Verdict: COMPATIBLE — all expectations can be satisfied simultaneously
```

**Healthy overlaps**: Multiple skills reinforcing the same structure. No action needed.

**Concerning overlaps**: Many skills with detailed expectations in the same area increases the chance of future conflicts. Consider whether some skills' expectations can be consolidated.

## Reading Conflicts

Conflicts are the most important finding — they indicate cross-steer incompatibility:

```
Conflicts Detected: 1

  api/error handling
    Skills: api-steer vs interoperability-steer
    api-steer expects:
      - snippet in api/middleware/errors.go: "ErrorResponse{Code:"
    interoperability-steer expects:
      - snippet in api/middleware/errors.go: "ProtoError{Status:"
    Verdict: CONFLICT — both skills expect different error shapes
    Recommendation: Resolve by aligning error formats or merging the expectations
```

**What to do with conflicts**:
1. Determine which skill's guidance is correct for this template
2. Update the less-correct skill's guidance in prompt-manager
3. Update the expectations in DTV to match
4. Flag the conflict in the meta optimization team's work queue

## Understanding Unconfigured Skills

```
Unconfigured Skills: 4
  - polish (connected, no expectations)
  - security (connected, no expectations)
  - performance (connected, no expectations)
  - cognitive-load-reduction (connected, no expectations)
```

Unconfigured skills mean:
- The skill is known to apply to this reference
- Nobody has defined what it expects structurally
- **This is the primary backlog for skill improvement**

Each unconfigured skill is a candidate for:
1. Adding structural expectations (if the skill's guidance is structured enough)
2. Adding CLI tool assertions (if the skill references validation tools)
3. Determining the skill needs more structure before it can be configured

## Understanding Maturity Scores [P1]

```
Skill Maturity Scores:
  api-steer:                    92/100  (structural: 5, cli: 3, all pass, no conflicts)
  unit-testing-architecture-steer: 88/100  (structural: 4, cli: 2, all pass, no conflicts)
  documentation-health:         75/100  (structural: 3, cli: 1, all pass, no conflicts)
  react-coherence:              45/100  (structural: 2, cli: 0, all pass, no conflicts)
  polish:                        0/100  (unconfigured)
  security:                      0/100  (unconfigured)
```

**Maturity factors**:
- Has structural expectations → base score
- Has CLI tool assertions → additional score
- All expectations pass → multiplier
- No conflicts with other skills → bonus
- Unconfigured → score 0

## Tooling Baseline Results [P1]

```
Tooling Baselines:
  scenario-auditor:  ✅ 0 violations (expected: 0)
  test-genie:        ✅ 11/11 phases pass (expected: all pass)
  completeness:      ✅ Score 97 (expected: >= 96)
```

When baselines fail:

```
Tooling Baselines:
  scenario-auditor:  ❌ 2 violations (expected: 0)
    - content_type_headers: api/handlers/tasks/list.go:42
    - graceful_shutdown: api/main.go:15
  test-genie:        ✅ 11/11 phases pass
  completeness:      ⚠️  Score 94 (expected: >= 96, shortfall: 2 points)
```

**Baseline failures mean the tool is wrong, not the reference** (assuming the reference is maintained correctly). Investigate the tool:
- scenario-auditor: Is the rule producing a false positive? File an issue.
- test-genie: Is a phase misconfigured? Check the phase logic.
- completeness-scoring: Is the scoring model miscalibrated? Check dimension weights.
