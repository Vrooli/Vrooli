# ⚠️ Testing Safety: CRITICAL INFORMATION

> **THIS DIRECTORY CONTAINS CRITICAL SAFETY INFORMATION TO PREVENT DATA LOSS**

## 🚨 The Problem

Test scripts can accidentally delete important files. This has happened in production:
- A BATS teardown function with `rm -f "${TEST_FILE_PREFIX}"*` deleted Makefiles, READMEs, and PRDs
- The bug occurred when the CLI wasn't installed, causing TEST_FILE_PREFIX to be empty
- This turned the command into `rm -f *`, deleting everything in the current directory

## 🛡️ Required Reading

1. **[Safety Guidelines](GUIDELINES.md)** - Comprehensive safety rules and patterns
2. **[BATS Teardown Bug](BATS_TEARDOWN_BUG.md)** - Detailed analysis of the actual incident

## ✅ Quick Safety Checklist

Before committing ANY test script:

```bash
# 1. Run the safety linter
scripts/scenarios/testing/lint-tests.sh test/

# 2. Verify these patterns are NOT present:
❌ rm -f "${VAR}"*                    # Unguarded wildcards
❌ rm -rf "$DIR"                      # No path validation
❌ TEST_PREFIX set after skip         # BATS variable order wrong
❌ teardown() without guards          # Missing safety checks

# 3. Ensure these patterns ARE present:
✅ if [ -n "${VAR:-}" ]; then        # Variable validation
✅ case "$PATH" in /tmp/*)           # Path restrictions
✅ Variables set BEFORE skip          # Correct BATS order
✅ rm operations with 2>/dev/null     # Error handling
```

## 🔒 Safe BATS Teardown Template

```bash
#!/usr/bin/env bats

# ALWAYS set critical variables FIRST
setup() {
    # Set variables BEFORE any skip conditions
    export TEST_FILE_PREFIX="/tmp/test-$$"
    export TEST_DIR="/tmp/test-dir-$$"
    
    # Skip conditions come AFTER variable setup
    if ! command -v my-cli >/dev/null 2>&1; then
        skip "CLI not installed"
    fi
    
    # Create test files
    mkdir -p "$TEST_DIR"
}

# SAFE teardown with multiple guards
teardown() {
    # Guard 1: Check variable is set
    if [ -n "${TEST_FILE_PREFIX:-}" ]; then
        # Guard 2: Verify it's in /tmp
        case "${TEST_FILE_PREFIX}" in
            /tmp/*)
                # Safe to clean up
                rm -f "${TEST_FILE_PREFIX}"* 2>/dev/null || true
                ;;
            *)
                echo "WARNING: Unsafe path, skipping cleanup" >&2
                ;;
        esac
    fi
    
    # Same pattern for directories
    if [ -n "${TEST_DIR:-}" ] && [ -d "$TEST_DIR" ]; then
        case "${TEST_DIR}" in
            /tmp/*)
                rm -rf "$TEST_DIR" 2>/dev/null || true
                ;;
        esac
    fi
}
```

## 🚫 What NOT to Do

```bash
# ❌ NEVER: Unguarded cleanup
teardown() {
    rm -f "${TEST_PREFIX}"*  # If TEST_PREFIX is empty, deletes everything!
}

# ❌ NEVER: Variables after skip
setup() {
    [ -f /some/file ] || skip "Missing file"
    TEST_PREFIX="/tmp/test"  # Never reached if skipped!
}

# ❌ NEVER: Cleanup outside /tmp
TEST_DIR="./test-data"  # Could delete production files!

# ❌ NEVER: No error handling
rm -rf "$DIR"  # Fails loudly if DIR doesn't exist
```

## 🛠️ Use Safe Templates

```bash
# Copy the safe BATS template
cp scripts/scenarios/testing/templates/bats/cli-test.bats.template \
   test/cli/my-test.bats

# Customize for your needs
sed -i 's/REPLACE_WITH_YOUR_CLI_NAME/my-cli/g' test/cli/my-test.bats
```

## 📊 The Cost of Getting It Wrong

- **Actual incident**: Deleted critical project files (Makefile, README.md, PRD.md)
- **Recovery time**: Hours of git restoration and debugging
- **Trust impact**: Developers lose confidence in test suite
- **Productivity loss**: Fear of running tests slows development

## 🎯 Golden Rules

1. **Validate before you devastate** - Check variables before using them
2. **Restrict to /tmp** - Never clean up files outside temporary directories  
3. **Order matters in BATS** - Set variables BEFORE skip conditions
4. **Lint before commit** - Always run the safety linter
5. **When in doubt, don't delete** - Better to leave temp files than delete production

## 🆘 Getting Help

- Found a dangerous pattern? Report immediately with "SAFETY" label
- Need help making tests safe? Ask in #testing channel
- Want to improve safety? Submit PRs to safety guidelines

---

**Remember: One bad `rm` command can ruin everyone's day. Take safety seriously.**