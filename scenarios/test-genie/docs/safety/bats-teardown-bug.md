# The BATS Teardown Bug: A Case Study

## Executive Summary

A BATS test teardown function accidentally deleted critical project files (Makefile, README.md, PRD.md) when the visited-tracker CLI wasn't installed. This document analyzes the incident to prevent recurrence.

## The Incident

### What Happened
On September 15, 2024, running `make test` in the visited-tracker scenario deleted important project files instead of just cleaning up test artifacts.

### Root Cause
```bash
# The problematic code
teardown() {
    rm -f "${TEST_FILE_PREFIX}"*
}
```

When the CLI wasn't installed:
1. `setup()` would call `skip "visited-tracker CLI not installed"`
2. BATS still runs `teardown()` even for skipped tests
3. `TEST_FILE_PREFIX` was never set (empty string)
4. The command became `rm -f *`
5. All files in the current directory were deleted

## Technical Deep Dive

### BATS Lifecycle Behavior

BATS (Bash Automated Testing System) has a specific lifecycle that created this vulnerability:

```
Test Execution Flow:
├─ setup()       # Runs before each test
│  ├─ Check CLI availability
│  ├─ skip if not found  <- STOPS HERE
│  └─ Set TEST_FILE_PREFIX (never reached)
├─ @test         # Skipped
└─ teardown()    # STILL RUNS! <- DANGER
   └─ rm -f "${TEST_FILE_PREFIX}"*
      └─ Expands to: rm -f *
```

### The Perfect Storm

Three conditions combined to create the bug:

1. **Variable expansion with wildcards**
   ```bash
   "${EMPTY_VAR}"* -> * -> matches everything
   ```

2. **BATS teardown always runs**
   - Even when tests are skipped
   - Even when setup() exits early

3. **Variable set after skip**
   - Skip condition checked first
   - Variable assignment never reached

## The Fix

### Immediate Fix Applied
```bash
# Before (DANGEROUS)
setup() {
    if ! command -v visited-tracker >/dev/null 2>&1; then
        skip "visited-tracker CLI not installed"
    fi
    export TEST_FILE_PREFIX="/tmp/visited-tracker-cli-test"
}

teardown() {
    rm -f "${TEST_FILE_PREFIX}"*
}

# After (SAFE)
setup() {
    # Set variable FIRST
    export TEST_FILE_PREFIX="/tmp/visited-tracker-cli-test"

    # Then check skip conditions
    if ! command -v visited-tracker >/dev/null 2>&1; then
        skip "visited-tracker CLI not installed"
    fi
}

teardown() {
    # Validate before deletion
    if [ -n "${TEST_FILE_PREFIX:-}" ]; then
        rm -f "${TEST_FILE_PREFIX}"* 2>/dev/null || true
    fi
}
```

### Complete Safety Pattern
```bash
teardown() {
    # Multiple layers of protection
    if [ -n "${TEST_FILE_PREFIX:-}" ] && [ "${TEST_FILE_PREFIX}" != "/" ]; then
        case "${TEST_FILE_PREFIX}" in
            /tmp/*)
                # Safe to delete
                rm -f "${TEST_FILE_PREFIX}"* 2>/dev/null || true
                ;;
            *)
                echo "WARNING: Unsafe path '${TEST_FILE_PREFIX}', skipping cleanup" >&2
                ;;
        esac
    fi
}
```

## Prevention Measures Implemented

### 1. Safety Linter
Created `scripts/scenarios/testing/lint-tests.sh` to detect:
- Unguarded rm with wildcards
- Variables set after skip conditions
- Missing path validation
- Dangerous cleanup patterns

### 2. Safe BATS Template
Created `scripts/scenarios/testing/templates/bats/cli-test.bats.template` with:
- Correct variable ordering
- Multiple safety guards
- Path restriction to /tmp
- Comprehensive documentation

### 3. Documentation Updates
- Added safety guidelines
- Updated testing architecture docs
- Created this case study
- Added prominent warnings

## Lessons Learned

### 1. Never Trust Empty Variables
```bash
# Always use default values
"${VAR:-default}"

# Always check before use
if [ -n "${VAR:-}" ]; then
```

### 2. Order Matters in Test Setup
```bash
setup() {
    # 1. Set ALL variables first
    export VAR1="value1"
    export VAR2="value2"

    # 2. Then check conditions
    [ -f /some/file ] || skip "Missing file"

    # 3. Then do setup work
    mkdir -p "$TEST_DIR"
}
```

### 3. Defense in Depth
Don't rely on a single safety measure:
- Variable validation
- Path restrictions
- Error handling
- Linting
- Code review

### 4. Test the Failure Paths
```bash
# Test with missing CLI
CLI_NOT_INSTALLED=1 bats test.bats

# Test with empty variables
TEST_FILE_PREFIX="" ./teardown_function

# Verify no production files deleted
```

## Impact Analysis

### What Could Have Been Lost
- Source code (if not in git)
- Configuration files
- Documentation
- Build artifacts
- User data

### What Was Actually Lost
- Makefile (recovered from git)
- README.md (recovered from git)
- PRD.md (recovered from git)
- Developer time (~2 hours)
- Confidence in test suite

## Recommendations

### For BATS Users
1. Always set variables before skip conditions
2. Always validate variables in teardown
3. Restrict cleanup to /tmp directories
4. Use the safe template as starting point
5. Run safety linter before commit

### For Framework Maintainers
1. Make safety linter part of CI/CD
2. Require code review for test changes
3. Document BATS gotchas prominently
4. Provide safe templates by default
5. Consider safer test frameworks

## Detection Script

Run this to find vulnerable patterns in your codebase:

```bash
#!/bin/bash
# Find potentially dangerous BATS patterns

echo "Searching for dangerous patterns..."

# Unguarded rm in teardown
echo "Checking for unguarded rm in teardown..."
grep -r "teardown()" -A 10 --include="*.bats" | \
    grep "rm.*\$" | \
    grep -v "if \[ -n"

# Variables after skip
echo "Checking for variables set after skip..."
for file in $(find . -name "*.bats"); do
    if grep -q "skip" "$file"; then
        skip_line=$(grep -n "skip" "$file" | head -1 | cut -d: -f1)
        var_line=$(grep -n "TEST.*=" "$file" | head -1 | cut -d: -f1)
        if [ -n "$var_line" ] && [ -n "$skip_line" ] && [ "$skip_line" -lt "$var_line" ]; then
            echo "WARNING: $file may have variables after skip"
        fi
    fi
done
```

## Conclusion

This bug demonstrates how seemingly innocent test code can cause significant damage. The combination of BATS' teardown behavior, shell variable expansion, and incorrect setup ordering created a perfect storm for data loss.

The incident led to comprehensive safety improvements:
- Immediate fix to affected file
- Creation of safety guidelines and linter
- Development of safe templates
- Documentation updates
- Cultural shift toward safety-first testing

**Key Takeaway**: In testing, as in production, the principle of "do no harm" must come first. A test that might delete production files is worse than no test at all.

## See Also

- [Safety Guidelines](GUIDELINES.md) - Complete safety reference
- [CLI Testing Guide](../guides/cli-testing.md) - BATS testing patterns
- [BATS Documentation](https://bats-core.readthedocs.io/) - Official BATS docs
