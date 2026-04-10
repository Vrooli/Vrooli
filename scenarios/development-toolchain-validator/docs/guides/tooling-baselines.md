# Tooling Baselines

## Overview

Tooling baselines validate that the development tools used in ecosystem-manager's scenario-improver loop produce correct results when run against a known-good reference scenario. This is a P1 feature.

## The Quick Validation Loop

The scenario-improver.md prompt used by ecosystem-manager instructs agents to run these tools in every iteration:

| Tool | Command | Purpose |
|------|---------|---------|
| `vrooli scenario status` | `vrooli scenario status {target}` | Lifecycle health check |
| `scenario-completeness-scoring` | `scenario-completeness-scoring score {target}` | Quality scoring (0-100) |
| `scenario-auditor` | `scenario-auditor audit {target} --timeout 240` | Standards violations |
| `test-genie` | `vrooli scenario test {target}` | 11-phase test suite |
| `vrooli scenario ui-smoke` | `vrooli scenario ui-smoke {target}` | UI validation |

If any of these tools produce incorrect results on a known-good reference, every scenario development loop that uses them makes suboptimal decisions.

## Baseline Configuration

### scenario-auditor Baseline

**Expected result**: Zero violations on a healthy reference.

```bash
development-toolchain-validator baselines configure auditor \
  --reference reference-react-vite \
  --expected-violations 0
```

What this validates:
- No false positive rules (rules that flag correct code)
- Rule categories covered: api, config, ui, testing, go, typescript, makefile
- External rule providers (stack-governor, test-genie, app-monitor, PRD) also pass

**If baseline fails**: The violation is a false positive. Investigate the specific rule:
1. Get the rule ID from the violation
2. Check the rule's test cases: `scenario-auditor rules test {rule_id}`
3. File an issue or fix the rule

### test-genie Baseline

**Expected result**: All 11 phases pass.

```bash
development-toolchain-validator baselines configure test-genie \
  --reference reference-react-vite \
  --expected-all-pass true
```

The 11 phases and what they check:

| Phase | Timeout | What it validates |
|-------|---------|-------------------|
| structure | 15s | File layout, manifests, JSON health |
| standards | 60s | scenario-auditor rules |
| dependencies | 30s | Runtime, tool, resource availability |
| lint | 30s | Static analysis, type checking |
| docs | 60s | Markdown, mermaid, links, portability |
| smoke | 90s | UI loads, iframe-bridge works |
| unit | 60s | Go unit tests, shell syntax |
| integration | 120s | CLI/Bats suite, API testing |
| playbooks | 120s | BAS workflow execution |
| business | 180s | Requirements mapping to targets |
| performance | 60s | Build benchmarks, Lighthouse |

**If baseline fails**: The phase logic is wrong for correct code. Investigate:
1. Get the phase name and failure details
2. Check the phase runner in test-genie's source
3. File an issue or fix the phase logic

### scenario-completeness-scoring Baseline

**Expected result**: Score >= 96 (Production Ready classification).

```bash
development-toolchain-validator baselines configure completeness \
  --reference reference-react-vite \
  --expected-min-score 96
```

Scoring dimensions:
- **Quality (50%)**: Requirement/target/test pass rates
- **Coverage (15%)**: Test depth and requirement complexity
- **Quantity (10%)**: Absolute counts with category-based thresholds
- **UI (25%)**: Template detection, component complexity, API integration

**If baseline fails**: The scoring model is miscalibrated. Investigate:
1. Get the score breakdown
2. Identify which dimension is scoring low
3. Check if the collector is miscounting or if thresholds need adjustment

## Running Baselines

```bash
# Run all baselines for a reference
development-toolchain-validator baselines run reference-react-vite

# Run specific baseline
development-toolchain-validator baselines run reference-react-vite --tool auditor
development-toolchain-validator baselines run reference-react-vite --tool test-genie
development-toolchain-validator baselines run reference-react-vite --tool completeness
```

## Baseline Results History [P1]

Results are stored in PostgreSQL for trend tracking:

```bash
# View baseline history
development-toolchain-validator baselines history reference-react-vite

# Detect regressions (tool that used to pass but now fails)
development-toolchain-validator baselines regressions reference-react-vite
```

## Integration with Meta Optimization Team

The meta optimization team in prompt-manager can use baseline results to:

1. **Detect tool regressions**: When a tool that passed last week now fails, something changed in the tool.
2. **Prioritize tool fixes**: Baseline failures directly impact every scenario development loop.
3. **Validate tool improvements**: After fixing a tool, re-run baselines to confirm the fix.

```bash
# Meta team would run this periodically
development-toolchain-validator baselines run reference-react-vite --json
```

The `--json` output can be parsed by agents to determine if action is needed.
