# Mock System Documentation

## Overview

The Vrooli test infrastructure uses a **3-tier mock architecture** to balance simplicity, functionality, and maintainability. This document describes how to use and extend the mock system.

## Architecture

### Tier 1: Simple Mocks (Stateless)
- **Purpose**: Basic command simulation, version checks, simple responses
- **Characteristics**: <100 lines, no state, minimal logic
- **Use Cases**: Tools that just need to exist (curl, git, docker)
- **Location**: `__test/mocks/tier1/`

### Tier 2: Balanced Mocks (Stateful) ⭐ RECOMMENDED
- **Purpose**: Core service testing with essential features
- **Characteristics**: 400-500 lines, in-memory state, 80% functionality
- **Use Cases**: Critical services (Redis, PostgreSQL, N8n)
- **Location**: `__test/mocks/tier2/`

### Tier 3: Full Mocks (Complex)
- **Purpose**: Complete service simulation
- **Characteristics**: 1000+ lines, full API coverage
- **Use Cases**: When integration testing requires exact behavior
- **Location**: `__test/mocks/tier3/`

## Using Tier 2 Mocks

### Redis Mock

```bash
# Source the mock
source __test/mocks/tier2/redis.sh

# Basic usage
redis-cli SET key value
result=$(redis-cli GET key)
echo "$result"  # outputs: value

# TTL support
redis-cli SET session:123 data EX 3600

# List operations
redis-cli LPUSH queue item1
redis-cli LPUSH queue item2
item=$(redis-cli RPOP queue)
echo "$item"  # outputs: item1

# Transactions
redis-cli MULTI
redis-cli SET key1 value1
redis-cli SET key2 value2
redis-cli EXEC

# Error injection for testing
redis_mock_set_error "connection_failed"
redis-cli PING  # Returns connection error
redis_mock_set_error ""  # Clear error

# Convention-based tests
test_redis_connection  # Test connectivity
test_redis_health      # Test health status
test_redis_basic       # Test basic operations
```

### PostgreSQL Mock

```bash
# Source the mock
source __test/mocks/tier2/postgres.sh

# Create table
psql -c "CREATE TABLE users (id INT, name TEXT, email TEXT)"

# Insert data
psql -c "INSERT INTO users VALUES (1, 'Alice', 'alice@test.com')"

# Query data
result=$(psql -c "SELECT COUNT(*) FROM users")
echo "$result"  # Shows formatted table with count

# Transactions
psql -c "BEGIN"
psql -c "INSERT INTO users VALUES (2, 'Bob', 'bob@test.com')"
psql -c "ROLLBACK"  # Undo the insert

# Error injection
postgres_mock_set_error "auth_failed"
psql -c "SELECT 1"  # Returns auth error
postgres_mock_set_error ""  # Clear error

# Convention-based tests
test_postgres_connection  # Test connectivity
test_postgres_health      # Test health status  
test_postgres_basic       # Test CRUD operations
```

### N8n Mock

```bash
# Source the mock
source __test/mocks/tier2/n8n.sh

# Import workflow
cat > workflow.json << 'EOF'
{
    "name": "My Workflow",
    "nodes": [{"type": "start"}, {"type": "webhook"}]
}
EOF
n8n import:workflow --input=workflow.json
# Output: Imported workflow as ID: wf_1

# Activate workflow
n8n update:workflow --id=wf_1 --active=true

# Execute workflow
n8n execute --id=wf_1
# Output: Execution ID: exec_1, Status: success

# List workflows
n8n list:workflows

# Export workflow
n8n export:workflow --id=wf_1 --output=exported.json

# Webhook simulation
exec_id=$(n8n_mock_simulate_webhook "webhook_123" '{"data":"test"}')
echo "Webhook triggered execution: $exec_id"

# Convention-based tests
test_n8n_connection  # Test connectivity
test_n8n_health      # Test health status
test_n8n_basic       # Test workflow operations
```

## Creating New Mocks

### Step 1: Choose the Right Tier

- **Tier 1**: If you only need version checks or simple responses
- **Tier 2**: If you need state management and core functionality (RECOMMENDED)
- **Tier 3**: If you need complete API compatibility

### Step 2: Use the Template

For Tier 2 mocks, start with the template:

```bash
cp __test/mocks/TEMPLATE_TIER2.sh __test/mocks/tier2/myservice.sh
```

### Step 3: Implement Core Functions

1. **Main command function**: Handle CLI arguments
2. **Convention functions**: `test_<service>_connection()`, `test_<service>_health()`, `test_<service>_basic()`
3. **State management**: `<service>_mock_reset()`, `<service>_mock_dump_state()`
4. **Error injection**: `<service>_mock_set_error()`

### Step 4: Export Functions

```bash
export -f myservice
export -f test_myservice_connection
export -f test_myservice_health
export -f test_myservice_basic
export -f myservice_mock_reset
export -f myservice_mock_set_error
```

## BATS Integration

### Basic BATS Test Setup

```bash
#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Load test infrastructure (automatically loads mocks)
source "${BATS_TEST_DIRNAME}/../fixtures/setup.bash"

# Load BATS helpers
load "../helpers/bats-support/load"
load "../helpers/bats-assert/load"

setup() {
    # This loads all required mocks automatically
    vrooli_setup_unit_test
}

teardown() {
    vrooli_cleanup_test
}

@test "Redis mock functionality" {
    run redis-cli ping
    assert_success
    assert_output "PONG"
}

@test "PostgreSQL mock functionality" {
    run psql -c "SELECT 1"
    assert_success
}
```

### Alternative: Direct Shell Integration

Test multiple mocks together without BATS:

```bash
#!/usr/bin/env bash
# Source all required mocks
source __test/mocks/tier2/redis.sh
source __test/mocks/tier2/postgres.sh
source __test/mocks/tier2/n8n.sh

# Reset state
redis_mock_reset
postgres_mock_reset
n8n_mock_reset

# Run integrated test scenario
psql -c "CREATE TABLE events (id INT, data TEXT)"
redis-cli SET "event:latest" "pending"
n8n execute --id=wf_1
psql -c "INSERT INTO events VALUES (1, 'processed')"
redis-cli SET "event:latest" "complete"

# Verify results
db_result=$(psql -c "SELECT COUNT(*) FROM events")
cache_result=$(redis-cli GET "event:latest")

echo "Database records: $db_result"
echo "Cache status: $cache_result"
```

## Best Practices

### DO:
- ✅ Use Tier 2 mocks for most testing (best balance)
- ✅ Reset mock state before each test
- ✅ Use error injection to test failure scenarios
- ✅ Implement convention-based test functions
- ✅ Keep mocks focused on essential features
- ✅ Use debug mode for troubleshooting: `REDIS_DEBUG=1`

### DON'T:
- ❌ Don't implement features that aren't tested
- ❌ Don't persist state between test runs
- ❌ Don't exceed 500 lines for Tier 2 mocks
- ❌ Don't mix tiers in the same directory
- ❌ Don't forget to export functions

## Mock Statistics

| Service | Tier | Lines | Reduction | Coverage |
|---------|------|-------|-----------|----------|
| Redis | 2 | 489 | 65% | 80% |
| PostgreSQL | 2 | 525 | 50% | 80% |
| N8n | 2 | 575 | N/A | 80% |
| **Total** | - | **1,589** | **~50%** | **80%** |

*Legacy mocks: ~3,000+ lines with 100% coverage but high complexity*

## Debugging

Enable debug output for any mock:

```bash
# Enable debug for specific service
REDIS_DEBUG=1 redis-cli SET key value
POSTGRES_DEBUG=1 psql -c "SELECT 1"
N8N_DEBUG=1 n8n execute --id=wf_1

# Dump current state
redis_mock_dump_state
postgres_mock_dump_state
n8n_mock_dump_state
```

## Migration Status

**🎉 MIGRATION COMPLETED!** (August 2025)

- ✅ **All 28 services migrated** to Tier 2 architecture
- ✅ **50% code reduction** achieved (~12,000+ lines saved)
- ✅ **Legacy system removed** and cleaned up
- ✅ **Production-ready** with full integration testing
- ✅ **Zero downtime migration** completed successfully

For migration history and technical details, see [MIGRATION_HISTORY.md](../MIGRATION_HISTORY.md).

## Contributing

When adding new mocks:

1. Check if existing mocks can be reused
2. Choose appropriate tier based on needs
3. Use the template for consistency
4. Include convention-based test functions
5. Document usage in this README
6. Add integration tests if the service interacts with others

For questions or improvements, consult this README or the project documentation.