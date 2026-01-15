## Steer focus: Code Cleanup

Prioritize **systematic removal of dead code, deprecated implementations, and backwards-compatibility cruft** across this scenario.
Do **not** modify code in `packages/*` (shared code may have external consumers).
Do **not** remove code without verification; all removals must be provably safe.

Focus on **reducing code surface area** by eliminating artifacts that AI agents leave behind during iterative development.

---

### **1. The Problem: AI Agent Accumulation Patterns**

AI coding agents leave behind old implementations "for safety" when making changes. Over time, this accumulates into:

* **Duplicate implementations** - old + new code running in parallel
* **Deprecated markers** - code marked `// deprecated` but never removed
* **Forwarding shims** - functions that just call the new implementation
* **Stale TODO comments** - `// TODO: remove after X` that lingers indefinitely
* **Re-exports for compatibility** - exports no one actually imports

This cruft obscures the system's true structure, increases cognitive load, and creates maintenance burden. This phase systematically identifies and removes provably dead code.

---

### **2. What to Target**

Look for these cleanup candidates:
* Deprecated code markers (`// deprecated`, `// legacy`, `// old`)
* Duplicate implementations (functions with `Legacy`, `Old`, `V1` suffixes)
* Forwarding/shim functions that only delegate to new implementations
* Dead exports and unreachable code paths
* Stale TODO/FIXME/HACK comments that reference completed work
* Re-exports kept "for backward compatibility" with no actual consumers

**Do NOT remove:**
* Code in `packages/*` - shared packages may have external consumers you cannot see
* Active feature flags or gradual rollout code
* Code with explicit "keep until X" where X has not occurred
* Cross-scenario dependencies (verify before removing)
* Test fixtures, mocks, and intentional stubs

---

### **3. Detection Patterns**

Search for these markers to identify cleanup candidates:

**Explicit deprecation markers:**
```bash
# Find deprecated code markers
ast-grep --lang go --pattern '// deprecated$_'
ast-grep --lang go --pattern '// Deprecated$_'
ast-grep --lang typescript --pattern '// deprecated$_'
rg -n "// deprecated|// legacy|// old implementation|// TODO: remove" --type go --type ts
```

**Duplicate implementations:**
```bash
# Find functions with legacy/old suffixes
rg -n "func.*Legacy|func.*Old|func.*V1\(" --type go
rg -n "function.*Legacy|function.*Old|const.*Legacy" --type ts
```

**Forwarding shims:**
```bash
# Find thin wrapper functions (single-line delegation)
ast-grep --lang go --pattern 'func ($NAME) $METHOD($ARGS) { return $_.new$METHOD($ARGS) }'
```

**Stale TODO markers:**
```bash
# Find TODO/FIXME/HACK comments for review
rg -n "TODO:|FIXME:|HACK:" --type go --type ts
rg -n "// TODO: remove|// TODO: delete|// TODO: clean" --type go --type ts
```

**Unused exports:**
```bash
# Use golangci-lint for Go unused detection
golangci-lint run --enable=unused,deadcode

# For TypeScript, check import counts
rg -l "export.*function\s+\w+Legacy" --type ts
```

---

### **4. Verification Before Removal**

**CRITICAL: Never remove code without verification.**

For each candidate, perform these checks:

**Step 1: Search for all usages**
```bash
# Search entire codebase for references
rg -n "functionName" scenarios/
rg -n "functionName" packages/  # Check if shared code uses it

# For Go, check interface implementations
ast-grep --lang go --pattern '$_ = $_.(type{ $$$ functionName $$$ })'
```

**Step 2: Check test dependencies**
```bash
# Ensure no tests depend on the code
rg -n "functionName" --glob "*_test.go" --glob "*.test.ts"
```

**Step 3: Check cross-scenario imports**
```bash
# Verify no other scenarios import this
rg -n "import.*from.*{{TARGET}}" scenarios/ --glob "!scenarios/{{TARGET}}/**"
```

**Step 4: Run tests after removal**
```bash
# Always verify tests still pass
make test  # or: vrooli scenario test {{TARGET}}
```

**When verification fails:**
* If code has active consumers, do NOT remove it
* If code has unclear consumers, add a note and skip
* If tests fail after removal, revert and investigate

---

### **5. Removal Categories & Strategies**

#### **5.1 Deprecated Code with Replacements**

**Pattern:** Code marked deprecated with a newer replacement available.

```go
// Example: tasks/lifecycle.go
Manual        bool // deprecated: use IntentManual instead
ForceOverride bool // deprecated: use IntentReconcile when override is needed
```

**Strategy:**
1. Verify the replacement is fully adopted (search for old field usage)
2. Remove the deprecated field/function
3. Update any remaining callers to use the replacement
4. Run tests to confirm

#### **5.2 Duplicate Implementations**

**Pattern:** Old and new implementations existing in parallel.

```go
// Example: queue/execution.go
func (qp *Processor) executeTask() { ... }      // New path
func (qp *Processor) executeTaskLegacy() { ... } // Old path - 300+ lines
```

**Strategy:**
1. Verify the new implementation handles all cases
2. Search for any code paths still using the legacy version
3. Remove the legacy function entirely
4. Run tests to confirm

#### **5.3 Forwarding Shims**

**Pattern:** Functions that only delegate to new implementations.

```go
// Before (shim)
func (m *Manager) OldMethod(x int) error {
    return m.NewMethod(x)  // Just forwards
}

// After (removed)
// OldMethod deleted, callers updated to use NewMethod directly
```

**Strategy:**
1. Identify all callers of the shim
2. Update callers to use the target function directly
3. Remove the shim
4. Run tests to confirm

#### **5.4 Dead Exports and Unreachable Code**

**Pattern:** Exported symbols with no external consumers.

```typescript
// Re-exports that no one imports
export { legacyHelper } from './old-utils';  // 0 imports found
```

**Strategy:**
1. Search for imports of the symbol across all scenarios
2. If zero imports found, remove the export
3. If the underlying code is also unused, remove it too
4. Run tests to confirm

#### **5.5 Stale TODO/FIXME Comments**

**Pattern:** Comments referencing completed work or passed deadlines.

```go
// TODO: remove after migration to v2 (completed 6 months ago)
// FIXME: temporary workaround for bug #123 (bug was fixed)
// HACK: revert after release 2.0 (we're on 3.5)
```

**Strategy:**
1. Investigate the TODO context - is the condition met?
2. If the condition is met, remove the TODO and its associated code
3. If the condition is NOT met, leave the TODO and note the investigation
4. Run tests to confirm

#### **5.6 Backwards Compatibility Re-exports**

**Pattern:** Re-exports or type aliases kept for compatibility.

```go
// Legacy filesystem functions kept for backward compatibility
func GetResourcePath() string { return getResourcePathNew() }
```

**Strategy:**
1. Search for imports of the old name
2. If imports exist, update them to use the new name
3. Remove the re-export
4. Run tests to confirm

---

### **6. Aggressiveness Guidelines**

**DO remove:**
* Code with zero references after thorough search
* Code explicitly marked deprecated with available replacements
* Forwarding shims where all callers can be updated
* TODO comments referencing completed conditions
* Test fixtures for removed code

**DO NOT remove:**
* Code in `packages/*` (external consumers may exist)
* Code with unclear ownership or purpose (investigate first)
* Feature flags that may still be in gradual rollout
* Code with "keep until X" where X has not happened
* Cross-scenario shared code without full consumer analysis

**When uncertain:**
* Add a note: `// CLEANUP-CANDIDATE: reason why this looks dead, needs verification`
* Skip the removal and document for future review
* Err on the side of caution - false positives waste time, false negatives cause outages

---

### **7. Memory Management with Tidiness Manager**

Use `tidiness-manager` to get intelligent cleanup recommendations combining code metrics with visit tracking.

**At the start of each iteration:**
```bash
# Get files with highest cleanup potential
# Priority combines: staleness + tech debt markers + complexity
tidiness-manager recommend-refactors {{TARGET}} \
  --limit 5 \
  --sort-by priority \
  --min-lines 50

# Alternative: Focus on files with most TODO/deprecated markers
tidiness-manager issues {{TARGET}} --category lint --limit 10
```

**After cleaning each file:**
```bash
# Record cleanup work with specific notes
tidiness-manager visit <file-path> \
  --scenario {{TARGET}} \
  --note "<what was removed, what remains to investigate>"
```

**When a file has no cleanup candidates:**
```bash
# Exclude from future cleanup queries
tidiness-manager exclude <file-path> \
  --scenario {{TARGET}} \
  --reason "No dead code found - clean implementation"
```

**When a file is in packages/ or out of scope:**
```bash
# Exclude shared code from cleanup scope
tidiness-manager exclude <file-path> \
  --scenario {{TARGET}} \
  --reason "Shared package - out of scope for scenario cleanup"
```

**Before ending your session:**
```bash
# Add campaign note for handoff context
tidiness-manager campaign-note \
  --scenario {{TARGET}} \
  --note "<lines removed, patterns found, areas needing deeper investigation>"
```

**Interpreting the response:**
- **High TODO/FIXME counts** - files likely to have cleanup candidates
- **High staleness_score** - files not recently reviewed
- **Large line counts** - more surface area for dead code
- Focus on files with **tech debt markers** (deprecated, legacy, TODO)

**Note format guidelines:**
- **File notes**: Be specific about what you removed and what needs investigation
  - Good: "Removed executeTaskLegacy (300 lines), deprecated Manual field. TODO on line 45 needs owner verification before removal."
  - Bad: "Cleaned up some dead code"
- **Campaign notes**: Provide metrics and strategic context
  - Good: "Removed 847 lines across 12 files. Major patterns: legacy execution paths, deprecated struct fields. Blocked: insights.go fallback code needs manager confirmation."
  - Bad: "Made progress on cleanup"

---

### **8. Output Expectations**

You may remove:
* Deprecated functions, methods, and types with available replacements
* Duplicate implementations where the new path is fully adopted
* Forwarding shims and thin wrappers that only delegate
* Dead exports with zero consumers
* Stale TODO/FIXME/HACK comments for completed work
* Re-exports kept for backwards compatibility with no actual users

You **must**:
* Verify zero usage before every removal
* Run tests after every removal
* Keep changes atomic (one logical removal per commit when possible)
* Document uncertain cases rather than guessing
* Never touch `packages/*` without explicit permission
* Leave the codebase **smaller, cleaner, and easier to understand**

You **must not**:
* Remove code that "looks" unused without verification
* Weaken tests to make removals pass
* Remove active feature flags or gradual rollout code
* Touch cross-scenario dependencies without full analysis
* Rush removals - thoroughness prevents outages

---

### **9. Verification Checklist**

Before marking any file as "cleanup complete," verify:

- [ ] All deprecated markers investigated
- [ ] All TODO/FIXME comments reviewed for staleness
- [ ] All `Legacy`/`Old`/`V1` suffixed code checked for removal
- [ ] All forwarding shims evaluated for consolidation
- [ ] Zero external consumers confirmed for removed code
- [ ] Tests pass after all removals
- [ ] No new warnings or errors introduced

---

Focus this loop on **systematic, verified removal** of code that no longer serves a purpose. Every line removed is cognitive load eliminated, maintenance burden reduced, and clarity gained. But every line removed without verification is a potential outage - so be thorough, be methodical, and when in doubt, leave it and document.
