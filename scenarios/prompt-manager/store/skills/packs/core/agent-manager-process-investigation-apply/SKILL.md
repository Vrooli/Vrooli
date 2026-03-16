## Apply Investigation Fixes

Implement the recommendations from the attached investigation report. Produce a change report documenting what was applied, verified, and deferred.

---

### What you have

Context attachments contain the investigation report (in the investigation run's events/summary), the original failed run data, and any user-provided guidance.

### What to do

1. Find the investigation report in the attached investigation run events (look for the structured markdown report in the final assistant message)
2. Extract all recommendations, noting their category and priority
3. Apply **Environment/Tooling fixes first**, then Agent Setup fixes (each in priority order)
4. For each fix:
   - Read the target file to understand its current state
   - Make the minimal change needed
   - Verify: configs parse, paths exist, prompts are internally consistent
5. After all fixes: check that Environment/Tooling changes don't conflict with Agent Setup changes

### Rules

- Only implement recommendations from the investigation report — no extras
- All changes must be git-revertible
- Don't remove existing safety checks unless the investigation explicitly recommends it with justification
- If a recommendation is ambiguous, implement the narrower interpretation
- If a fix causes a new problem, stop and document it — don't "fix the fix"

### Output format

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
| E1 | ... | ... | ... | ... | Applied |

## Agent Setup Changes
| ID | Recommendation | File | Change | Verification | Status |
|---|---|---|---|---|---|
| A1 | ... | ... | ... | ... | Applied |

## Not Applied
| ID | Recommendation | Reason |
|---|---|---|
| ... | ... | ... |

## Cross-Category Verification
- **Conflicts found**: [Yes/No]
- **Details**: ...

## Follow-Up Actions
- ...
```

If a category has no changes, include the header with "No changes in this category."
