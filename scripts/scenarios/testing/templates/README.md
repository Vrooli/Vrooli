# Phase Test Templates

This directory contains template files for creating phased test scripts that use the new `phase-helpers.sh` boilerplate elimination system.

## 🎯 Benefits

Using `phase-helpers.sh` reduces test phase boilerplate by **80%** (from 40+ lines to ~5 lines) while providing:
- Automatic scenario/directory detection
- Standardized timing and reporting
- Consistent error/warning/test counting
- Cleanup function management
- Built-in helper functions

## 📁 Available Templates

### `phase-minimal.sh.template`
**Use for:** Creating new custom test phases
- Shows absolute minimum boilerplate required
- Includes all available helper functions
- Good starting point for any test phase

### `phase-unit.sh.template`
**Use for:** Unit test phases
- Runs language-specific unit tests with coverage
- Uses centralized testing library directly (no fallback)
- Configurable coverage thresholds
- Supports Go, Node.js, and Python tests

### `phase-structure.sh.template` 
**Use for:** Structure validation phases
- Validates files, directories, and configuration
- Includes service.json validation
- Language-specific checks (Go, Node.js, etc.)

### `phase-integration.sh.template`
**Use for:** Integration test phases
- API/UI connectivity testing
- Resource integration checks
- End-to-end workflow testing
- Cleanup function registration

## 🚀 Quick Start

1. **Copy a template to your scenario:**
```bash
cp scripts/scenarios/testing/templates/phase-minimal.sh.template \
   scenarios/my-scenario/test/phases/test-mynewphase.sh
```

2. **Customize the TODO sections**

3. **Run the test:**
```bash
./test/phases/test-mynewphase.sh
```

## 📋 Before/After Comparison

### Before (40+ lines of boilerplate):
```bash
#!/bin/bash
set -euo pipefail
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/core.sh"
echo "=== My Test Phase ==="
start_time=$(date +%s)
error_count=0
test_count=0

# ... actual test logic ...

end_time=$(date +%s)
duration=$((end_time - start_time))
if [ $error_count -eq 0 ]; then
    log::success "✅ Tests passed in ${duration}s"
    exit 0
else
    log::error "❌ Tests failed in ${duration}s"
    exit 1
fi
```

### After (5 lines of boilerplate):
```bash
#!/bin/bash
source "$(dirname "${BASH_SOURCE[0]}")/../../../../scripts/scenarios/testing/shell/phase-helpers.sh"
testing::phase::init --target-time "30s"

# ... actual test logic ...

testing::phase::end_with_summary
```

## 🛠️ Helper Functions Reference

### Core Functions
- `testing::phase::init [options]` - Initialize phase environment
- `testing::phase::end_with_summary [message]` - End with standardized summary

### Test Counting
- `testing::phase::add_test [passed|failed|skipped]` - Count test result
- `testing::phase::add_error [message]` - Add error with optional message
- `testing::phase::add_warning [message]` - Add warning (doesn't fail tests)

### Convenience Helpers
- `testing::phase::check "description" command` - Run check and auto-count
- `testing::phase::check_files file1 file2...` - Verify files exist
- `testing::phase::check_directories dir1 dir2...` - Verify directories exist
- `testing::phase::timed_exec "description" command` - Execute with timing

### Cleanup Management
- `testing::phase::register_cleanup function_name` - Register cleanup function

## 📊 Global Variables

These are automatically set by `testing::phase::init`:
- `$TESTING_PHASE_SCENARIO_NAME` - Auto-detected scenario name
- `$TESTING_PHASE_SCENARIO_DIR` - Path to scenario directory  
- `$TESTING_PHASE_APP_ROOT` - Path to Vrooli root
- `$TESTING_PHASE_ERROR_COUNT` - Current error count
- `$TESTING_PHASE_TEST_COUNT` - Current test count
- `$TESTING_PHASE_WARNING_COUNT` - Current warning count
- `$TESTING_PHASE_SKIPPED_COUNT` - Current skipped count

## ✅ Best Practices

1. **Always source phase-helpers.sh first** - Must be the first non-comment line after shebang
2. **Use helper functions** - Prefer `testing::phase::check` over manual counting
3. **Register cleanup early** - Call `testing::phase::register_cleanup` before creating test data
4. **Set realistic target times** - Helps identify performance regressions
5. **End with summary** - Always call `testing::phase::end_with_summary` at the end

## 🔄 Migration Guide

To migrate existing test phases:

1. Replace boilerplate with phase-helpers source and init
2. Replace manual error counting with helper functions
3. Replace exit logic with `testing::phase::end_with_summary`
4. Test to ensure behavior is identical

See visited-tracker scenario for complete examples of migrated phases.