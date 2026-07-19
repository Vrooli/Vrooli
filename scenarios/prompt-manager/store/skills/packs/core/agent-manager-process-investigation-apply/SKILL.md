## Apply Investigation Fixes

Implement the operator-approved recommendations from a completed investigation. Produce a change report documenting what was applied, verified, and deferred.

---

### What you have

The structured investigation findings and the operator's approval decision are provided inline below — you do NOT need to re-open the investigation run's transcript.

**Investigation findings** (schema-validated categorized recommendations):

{{.findings}}

**Operator approval** (`decision`, and the `selected` recommendation texts the operator approved to apply):

{{.approval}}

**Original run context** (what the investigation was diagnosing):

{{.context}}

### What to do

1. Read the findings and the operator's `selected` list. Apply **only** the recommendations the operator selected (if `selected` is empty, apply every recommendation in the approved findings).
2. Apply **Environment/Tooling fixes first**, then Agent Setup fixes, each in severity order (Critical → Major → Gap → Minor).
3. For each fix:
   - Read the target file to understand its current state
   - Make the minimal change needed
   - Verify: configs parse, paths exist, prompts are internally consistent
4. After all fixes: check that Environment/Tooling changes don't conflict with Agent Setup changes.

### Rules

- Only implement recommendations that appear in the approved findings — no extras
- All changes must be git-revertible
- Don't remove existing safety checks unless a recommendation explicitly requires it with justification
- If a recommendation is ambiguous, implement the narrower interpretation
- If a fix causes a new problem, stop and document it — don't "fix the fix"

### Output format

```markdown
# Apply Investigation Report

## Summary
- **Recommendations approved**: [count]
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
