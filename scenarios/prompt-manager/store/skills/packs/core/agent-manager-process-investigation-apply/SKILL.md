## Meta focus: Apply Investigation Recommendations

Implement fixes from an investigation report, organized by category, with per-change verification. Produce a structured change report documenting what was applied, verified, and deferred.

Required reading:
- `prompt-manager skill read skill-principles`

---

### 1. Scope Boundaries

**In scope:**
- Implementing recommendations from the investigation report
- Fixing configs, paths, tools, and service settings (Environment/Tooling)
- Modifying prompts, instructions, guardrails, and context (Agent Setup)
- Verifying each change individually and cross-category

**Out of scope:**
- Re-investigating the original failure (already done)
- Adding improvements not listed in the investigation recommendations
- Completing the original failed task
- Architecture redesign or feature work

---

### 2. Inputs

Context attachments provided automatically:
- **Investigation run summary**: the investigation run's metadata and results
- **Investigation run events**: the investigation agent's actions and findings
- **Original run data**: summaries, events, and diffs from the failed run(s)
- **Custom context**: user-provided additional information

For additional data, use the agent-manager CLI:
```bash
agent-manager run get <run-id>      # Full run details
agent-manager run events <run-id>   # All events from investigation
agent-manager run diff <run-id>     # Any code changes investigation made
```

---

### 3. Apply Workflow

#### Step 1: Parse and Prioritize

1. Read the investigation report from the run events (look for the structured report in the investigation agent's final output).
2. Extract all recommendations, noting their category (Environment/Tooling or Agent Setup), severity, and priority.
3. Order recommendations: Environment/Tooling fixes first, then Agent Setup fixes, each ordered by priority within category.

#### Step 2: Apply Environment/Tooling Fixes

For each Environment/Tooling recommendation, in priority order:
1. **Read** the target file/config to understand current state
2. **Plan** the minimal change needed
3. **Implement** the fix
4. **Verify**: confirm the change is correct using the appropriate verification method (see Verification Decision Table)

Common Environment/Tooling fixes:
- Fix config file syntax or values
- Correct file paths
- Create missing files or directories
- Fix tool settings or permissions
- Add missing dependencies to config

#### Step 3: Apply Agent Setup Fixes

For each Agent Setup recommendation, in priority order:
1. **Read** the target prompt/instruction file
2. **Plan** the minimal change that addresses the finding
3. **Implement** the fix
4. **Verify**: check consistency with other instructions (see Verification Decision Table)

Common Agent Setup fixes:
- Add explicit guardrails to prevent the failure pattern
- Clarify ambiguous instructions
- Resolve contradictions between instruction sources
- Add examples of correct behavior
- Add scope boundaries or constraints

#### Step 4: Cross-Category Verification

After all individual fixes are applied:
1. Check that Environment/Tooling fixes don't conflict with Agent Setup changes
2. Verify that prompt changes reference tools/configs that now exist correctly
3. Confirm no existing safety checks were removed

#### Step 5: Produce Change Report

Document all changes using the output format below.

---

### 4. Convergence Patterns

#### Fix Order Decision

| Situation | Action |
|---|---|
| Both categories have recommendations | Apply Environment/Tooling first, then Agent Setup |
| Only Environment/Tooling recommendations | Apply and verify each in priority order |
| Only Agent Setup recommendations | Apply and verify each in priority order |
| A fix depends on another fix | Apply the dependency first regardless of category |

#### Verification Decision Table

| Change Type | Verification Method |
|---|---|
| Config file change | Parse/validate the file format (YAML, JSON, TOML) |
| Path correction | Verify the path exists and is accessible |
| Missing file creation | Verify file exists with expected content |
| Tool setting change | Run a safe diagnostic command to confirm |
| Prompt wording change | Read the full prompt and check for internal consistency |
| Guardrail addition | Verify it doesn't conflict with existing instructions |
| Instruction clarification | Check against other instruction sources for contradictions |

---

### 5. Safety Guardrails

- **Only implement listed recommendations**: do not add improvements, refactors, or fixes not in the investigation report
- **Verify each change**: never skip verification
- **Don't remove existing safety checks**: only add to them unless the investigation explicitly recommends removal with justification
- **Git-revertible changes only**: all changes must be revertible via `git checkout` or `git revert`
- **Conservative interpretation**: if a recommendation is ambiguous, implement the narrower interpretation
- **Stop on cascading problems**: if a fix causes a new problem, stop, document it, and do not attempt to "fix the fix"
- **Scope to referenced files**: only modify files mentioned in or clearly implied by the recommendations

---

### 6. Output Expectations

Produce this report:

```markdown
# Apply Investigation Report

## Summary
- **Recommendations received**: [count]
- **Applied successfully**: [count]
- **Not applied**: [count]
- **Verification failures**: [count]

## Environment/Tooling Changes
| ID | Recommendation | File | Change | Verification | Status |
|---|---|---|---|---|---|
| E1 | ... | ... | ... | ... | ✅ Applied |

## Agent Setup Changes
| ID | Recommendation | File | Change | Verification | Status |
|---|---|---|---|---|---|
| A1 | ... | ... | ... | ... | ✅ Applied |

## Not Applied
| ID | Recommendation | Reason |
|---|---|---|
| ... | ... | [Ambiguous | Out of scope | Requires manual review | Blocked by ...] |

## Cross-Category Verification
- **Conflicts found**: [Yes/No]
- **Details**: ...

## Follow-Up Actions
- ...
```

If a category has no changes, include the section header with "No changes in this category."

---

### 7. Troubleshooting & Edge Cases

- **Investigation report not found in events**: If the structured report cannot be located in the investigation run events, check the run's final output or diff. If still not found, document this and apply only recommendations that can be clearly identified from available data.
- **Recommendation references nonexistent file**: If a recommendation targets a file that doesn't exist, check whether the investigation intended a different path. If unclear, list it under "Not Applied" with reason.
- **Conflicting recommendations**: If two recommendations contradict each other, apply neither and document the conflict under "Not Applied."
- **Partial verification failure**: If a change applies correctly but verification reveals an unexpected side effect, revert the change and document it under "Not Applied" with details.
