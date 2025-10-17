# Testing Safety Guidelines

This document outlines critical safety measures to prevent dangerous behaviors in test scripts, particularly around file operations that could cause data loss.

## 🚨 Critical Safety Rules

### 1. BATS Teardown Safety

**NEVER** use unguarded wildcard patterns in teardown functions:

```bash
# ❌ DANGEROUS - Can delete everything if TEST_FILE_PREFIX is empty
teardown() {
    rm -f "${TEST_FILE_PREFIX}"*
}

# ✅ SAFE - Always guard with proper checks
teardown() {
    if [ -n "${TEST_FILE_PREFIX:-}" ] && [ "${TEST_FILE_PREFIX}" != "/" ]; then
        case "${TEST_FILE_PREFIX}" in
            /tmp/*)
                rm -f "${TEST_FILE_PREFIX}"* 2>/dev/null || true
                ;;
            *)
                echo "WARNING: Unsafe TEST_FILE_PREFIX '${TEST_FILE_PREFIX}', skipping cleanup" >&2
                ;;
        esac
    fi
}
```

**Why this happens:** BATS teardown functions run even when tests are skipped. If setup() calls `skip` before setting variables, teardown() runs with empty variables.

### 2. Setup Function Order

**ALWAYS** set critical variables before any skip conditions:

```bash
# ❌ DANGEROUS - Variables not set if CLI missing
setup() {
    if ! command -v my-cli >/dev/null 2>&1; then
        skip "CLI not installed"
    fi
    export TEST_FILE_PREFIX="/tmp/my-test"  # Never reached if CLI missing
}

# ✅ SAFE - Set variables first
setup() {
    export TEST_FILE_PREFIX="/tmp/my-test"  # Always set
    if ! command -v my-cli >/dev/null 2>&1; then
        skip "CLI not installed"
    fi
}
```

### 3. Path Validation

**ALWAYS** validate paths before destructive operations:

```bash
# ❌ DANGEROUS - No path validation
rm -rf "$SOME_DIR"

# ✅ SAFE - Validate path structure
if [ -n "${SOME_DIR:-}" ] && [ "${SOME_DIR}" != "/" ]; then
    case "${SOME_DIR}" in
        /tmp/*)
            rm -rf "$SOME_DIR"
            ;;
        *)
            echo "ERROR: Unsafe path '$SOME_DIR', refusing to delete" >&2
            return 1
            ;;
    esac
fi
```

### 4. Test File Isolation

**ALWAYS** use isolated locations for test files:

```bash
# ❌ DANGEROUS - Test files in working directory
TEST_DIR="./test-data"

# ✅ SAFE - Test files isolated under /tmp
TEST_DIR="/tmp/my-scenario-test-$$"  # Include PID for uniqueness
```

## 🔧 Implementation Patterns

### BATS Template Pattern

Use our safe BATS template:

```bash
# Copy the safe template
cp scripts/scenarios/testing/templates/bats/cli-test.bats.template \
   scenarios/my-scenario/cli/my-cli.bats

# Customize for your CLI
sed -i 's/REPLACE_WITH_YOUR_CLI_NAME/my-cli/g' \
   scenarios/my-scenario/cli/my-cli.bats
```

### Variable Safety Checks

Create defensive variable handling:

```bash
# Safe variable expansion
SAFE_VAR="${UNSAFE_VAR:-/tmp/default}"

# Path validation function
validate_test_path() {
    local path="$1"
    [ -n "$path" ] || return 1
    [ "$path" != "/" ] || return 1
    case "$path" in
        /tmp/*) return 0 ;;
        *) return 1 ;;
    esac
}
```

### Error Handling

Always include error handling in cleanup:

```bash
cleanup_test_files() {
    local test_prefix="$1"
    
    if ! validate_test_path "$test_prefix"; then
        echo "ERROR: Invalid test path '$test_prefix'" >&2
        return 1
    fi
    
    # Use individual commands with error handling
    rm -f "${test_prefix}"* 2>/dev/null || true
    rm -rf "${test_prefix}-dir" 2>/dev/null || true
    
    # Verify cleanup
    if ls "${test_prefix}"* >/dev/null 2>&1; then
        echo "WARNING: Some test files remain: ${test_prefix}*" >&2
    fi
}
```

## 📋 Pre-commit Checklist

Before committing test scripts, verify:

- [ ] All `rm` commands are guarded with path validation
- [ ] BATS setup() sets variables before skip conditions  
- [ ] BATS teardown() validates variables before cleanup
- [ ] Test files are created under `/tmp` or other safe location
- [ ] Wildcard patterns (`*`) are never used with empty variables
- [ ] Error handling prevents cascading failures

## 🔍 Common Vulnerability Patterns

### Pattern 1: Unguarded Wildcards

```bash
# VULNERABLE
rm -f "$PREFIX"*

# SAFE  
[ -n "$PREFIX" ] && rm -f "$PREFIX"*
```

### Pattern 2: Missing Path Validation

```bash
# VULNERABLE
rm -rf "$BUILD_DIR"

# SAFE
if [ -d "$BUILD_DIR" ] && [[ "$BUILD_DIR" =~ ^/tmp/ ]]; then
    rm -rf "$BUILD_DIR"
fi
```

### Pattern 3: Variable Expansion in Loops

```bash
# VULNERABLE - Empty arrays can cause issues
for file in "${FILES[@]}"; do
    rm "$file"
done

# SAFE
if [ ${#FILES[@]} -gt 0 ]; then
    for file in "${FILES[@]}"; do
        [ -f "$file" ] && rm "$file"
    done
fi
```

## 🔧 Resource Cleanup Safety

### Safe Resource Cleanup Patterns

When tests interact with resources (databases, containers, files), ensure safe cleanup:

#### 1. Track Resource State

```bash
# Keep track of what you created
CREATED_RESOURCES=()

create_test_resource() {
    local resource_id="$1"
    # Create resource
    docker run -d --name "$resource_id" nginx:alpine
    # Track it
    CREATED_RESOURCES+=("$resource_id")
}

cleanup_resources() {
    # Only clean up what we created
    for resource in "${CREATED_RESOURCES[@]}"; do
        docker rm -f "$resource" 2>/dev/null || true
    done
    CREATED_RESOURCES=()
}

# Use trap for guaranteed cleanup
trap cleanup_resources EXIT
```

#### 2. Namespace Test Resources

```bash
# Use unique namespaces to avoid conflicts
TEST_NAMESPACE="test-$$-$(date +%s)"

# Database: Create test-specific schema
psql -c "CREATE SCHEMA IF NOT EXISTS ${TEST_NAMESPACE}"

# Redis: Use test-specific key prefix
redis-cli SET "${TEST_NAMESPACE}:key" "value"

# Cleanup: Only affects test namespace
psql -c "DROP SCHEMA ${TEST_NAMESPACE} CASCADE"
redis-cli --scan --pattern "${TEST_NAMESPACE}:*" | xargs redis-cli DEL
```

#### 3. Validate Before Cleanup

```bash
cleanup_database() {
    local schema="$1"
    
    # Validate schema name format (test-only)
    if [[ ! "$schema" =~ ^test-[0-9]+-[0-9]+$ ]]; then
        echo "ERROR: Refusing to drop non-test schema: $schema"
        return 1
    fi
    
    # Additional safety: Check for test marker
    local marker=$(psql -t -c "SELECT 1 FROM information_schema.schemata 
                               WHERE schema_name='$schema' 
                               AND schema_name LIKE 'test-%'")
    
    if [ "$marker" = "1" ]; then
        psql -c "DROP SCHEMA $schema CASCADE"
    fi
}
```

#### 4. Resource Cleanup Anti-Patterns

```bash
# ❌ DANGEROUS - No validation
cleanup() {
    docker rm -f $(docker ps -q)  # Kills ALL containers!
    psql -c "DROP SCHEMA public CASCADE"  # Drops production schema!
}

# ✅ SAFE - Targeted cleanup
cleanup() {
    # Only remove test containers
    docker ps --filter "label=test-run=$TEST_ID" -q | xargs -r docker rm -f
    
    # Only drop test schemas
    psql -c "DROP SCHEMA IF EXISTS test_${TEST_ID} CASCADE"
}
```

## 🔄 Concurrent Test Execution Safety

### Preventing Test Interference

When multiple test suites run simultaneously, they must not interfere:

#### 1. Unique Test Identifiers

```bash
# Generate unique test ID
TEST_ID="${SCENARIO_NAME}-$$-$(date +%s)-${RANDOM}"

# Use in all test artifacts
TEST_DIR="/tmp/test-${TEST_ID}"
TEST_PORT=$((30000 + RANDOM % 10000))
TEST_DB="testdb_${TEST_ID}"
```

#### 2. Port Allocation Safety

```bash
# Find available port safely
find_free_port() {
    local port
    for port in $(seq 30000 40000); do
        if ! nc -z localhost "$port" 2>/dev/null; then
            # Double-check with bind attempt
            if nc -l -p "$port" -w 1 2>/dev/null; then
                echo "$port"
                return 0
            fi
        fi
    done
    return 1
}

# Use dynamic ports
API_PORT=$(find_free_port) || exit 1
export API_PORT
```

#### 3. File Locking for Shared Resources

```bash
# Use flock for exclusive access
run_exclusive_test() {
    local lockfile="/tmp/test-resource.lock"
    local lock_fd=200
    
    # Acquire lock
    exec {lock_fd}>>"$lockfile"
    flock -x "$lock_fd" || {
        echo "Failed to acquire lock"
        return 1
    }
    
    # Run test with exclusive access
    run_test_that_needs_exclusive_resource
    
    # Lock automatically released when script exits
}
```

#### 4. Process Isolation

```bash
# Use process groups for cleanup
run_isolated_test() {
    # Start new process group
    set -m
    
    # Run test in background
    (
        # All child processes in same group
        trap 'kill -TERM -$$' EXIT
        
        # Start test services
        ./start-test-services.sh &
        
        # Run tests
        ./run-tests.sh
    ) &
    
    TEST_PGID=$!
    wait $TEST_PGID
    
    # Cleanup entire process group if needed
    kill -TERM -$TEST_PGID 2>/dev/null || true
}
```

#### 5. Database Isolation

```bash
# Each test gets its own database
setup_test_database() {
    local test_db="testdb_${TEST_ID}"
    
    # Create isolated database
    createdb "$test_db"
    
    # Run migrations
    psql -d "$test_db" -f schema.sql
    
    # Export for test use
    export DATABASE_URL="postgresql://localhost/$test_db"
}

# Cleanup only test database
cleanup_test_database() {
    dropdb --if-exists "testdb_${TEST_ID}"
}
```

#### 6. Concurrent Safety Checklist

- [ ] Each test run has unique identifier
- [ ] No hardcoded ports - all dynamically allocated
- [ ] Test artifacts use unique namespaces
- [ ] Cleanup targets only test-created resources
- [ ] File operations restricted to test directories
- [ ] Database operations on test schemas only
- [ ] Process groups for reliable cleanup
- [ ] Lock files for exclusive resource access

## 🧪 Testing Safety Measures

### Test Your Safety Measures

Create tests that verify safety measures work:

```bash
# Test empty variable handling
TEST_FILE_PREFIX="" teardown_function
# Should not delete any files

# Test invalid path handling  
TEST_FILE_PREFIX="/etc" teardown_function
# Should refuse to delete

# Test with skipped setup
# Run teardown without setup - should be safe
```

### Validate Template Usage

When using templates:

1. **Customize thoroughly** - Don't leave placeholder values
2. **Test with missing dependencies** - Ensure graceful degradation  
3. **Verify cleanup** - Check that test files are properly removed
4. **Test failure modes** - Ensure safety when tests fail

## 📚 Related Resources

- [BATS Testing Framework](https://bats-core.readthedocs.io/)
- [Shell Script Safety](https://mywiki.wooledge.org/BashPitfalls)
- [Testing Templates](/scripts/scenarios/testing/templates/README.md)

## 🐛 Reporting Safety Issues

If you discover unsafe patterns in existing test scripts:

1. **Immediate**: Stop using the script if data loss is possible
2. **Report**: Create an issue with "SAFETY" label
3. **Fix**: Apply safety measures following this guide
4. **Test**: Verify fix works in all scenarios
5. **Document**: Update this guide if needed

Remember: **Data safety is more important than test convenience.**