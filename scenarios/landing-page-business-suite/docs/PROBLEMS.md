# Known Problems & Follow-up Tasks

This file tracks known issues and technical debt that need attention.

---

## Security Issues

### SQL-002: SQL Injection in download_hosting.go

**Severity:** ~~Critical~~ False Positive
**Location:** `api/download_hosting.go:882`
**Status:** ✅ Analyzed - False Positive
**Reported:** 2026-01-16 (scenario-auditor)
**Analyzed:** 2026-01-16 by Claude (failure-topography)

The scenario auditor flagged a potential SQL injection via format string, but **this is a false positive**.

**Analysis:**
The code in question uses `fmt.Sprintf` to build a WHERE clause, but:
1. The `whereClause` only contains **static SQL structure** with positional parameter placeholders (`$1`, `$2`, `$3`, etc.)
2. All **user inputs** (`bundleKey`, `platform`, `query`) flow through the `args...` parameters
3. The `fmt.Sprintf` **only interpolates parameter position numbers**, not user data
4. The actual values are bound via parameterized query execution

**Example of what the code does:**
```go
// User input goes into args (safe - parameterized)
args := []interface{}{bundleKey}  // User input bound to $1

// whereClause contains ONLY static SQL with placeholders
whereClause := "bundle_key = $1"

// Query uses parameterized execution
db.QueryRowContext(ctx,
    fmt.Sprintf(`SELECT COUNT(*) FROM download_artifacts WHERE %s`, whereClause),
    args...)  // User data bound here, not interpolated
```

**Conclusion:** No remediation required. The code correctly uses parameterized queries. The auditor's pattern matching detected `fmt.Sprintf` in SQL context but didn't trace the data flow to confirm safety.

**Recommendation for scenario-auditor:** Consider enhancing the SQL injection detection to trace whether user input actually flows into the format string arguments vs being bound as query parameters.

---

## Failure Handling Gaps

### Remaining `alert()` Usage

**Severity:** Low
**Status:** ✅ Resolved
**Reported:** 2026-01-16
**Resolved:** 2026-01-16 by Claude (signal-and-feedback-surface-design)

The Customization page has been updated to use InlineAlert, and other admin pages now use Toast notifications:

- [x] VariantEditor.tsx - ✅ Uses Toast notifications for success/error feedback
- [x] SectionEditor.tsx - ✅ Replaced alert() with Toast notifications
- [x] Customization.tsx - ✅ Updated to use InlineAlert + Toast for success

---

## Test Organization

### Monolithic Test Files

**Severity:** Medium
**Status:** Open
**Reported:** 2026-01-16 (scenario-completeness-scoring)

The completeness scoring tool reports 5 test files validating 4+ requirements each, which hurts the score by -10 points:
- `coverage/manual-validations/log.jsonl` (validates 14 requirements)
- Plus 4 more test files

**Action Required:** Break monolithic test files into focused tests for each requirement.

### Manual Validation Percentage

**Severity:** Low
**Status:** Open
**Reported:** 2026-01-16

20% of validations are manual (recommended max: 10%). Manual validations should be converted to automated tests.

---

## Standards Compliance

### PRD Template Sections

**Severity:** Low
**Status:** Open
**Reported:** 2026-01-16 (scenario-auditor)

The PRD has unexpected sections not in the template:
- Design Decisions
- Success Metrics (Post-Launch)
- Evolution Path
- External References
- Internal References

These may be intentional extensions but are flagged by the standards check.

---

## Performance

No performance issues currently tracked.

---

## UI/UX

### Missing Success Notifications

**Severity:** Low
**Status:** ✅ Resolved
**Reported:** 2026-01-16
**Resolved:** 2026-01-16 by Claude (signal-and-feedback-surface-design)

~~When variant operations succeed (archive, delete, weight update), there's no visual confirmation. Consider adding toast notifications for success messages.~~

**Resolution:** Added Toast notification system (`ui/src/shared/ui/Toast.tsx`) with success feedback:
- Customization page: archive, delete, weight update show success toasts
- VariantEditor page: create, update, JSON save show success toasts
- SectionEditor page: save, reorder show success toasts

---

## Last Updated

2026-01-16 by Claude (failure-topography-and-graceful-degradation)
