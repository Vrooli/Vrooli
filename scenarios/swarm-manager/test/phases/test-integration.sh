#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running integration tests..."

# Check if required resources are available
POSTGRES_AVAILABLE=false
REDIS_AVAILABLE=false

if command -v psql &>/dev/null; then
    if psql -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-postgres}" -c "SELECT 1" &>/dev/null; then
        POSTGRES_AVAILABLE=true
        testing::phase::success "PostgreSQL available"
    fi
fi

if command -v redis-cli &>/dev/null; then
    if redis-cli -h "${REDIS_HOST:-localhost}" -p "${REDIS_PORT:-6379}" ping &>/dev/null; then
        REDIS_AVAILABLE=true
        testing::phase::success "Redis available"
    fi
fi

if [ "$POSTGRES_AVAILABLE" = false ]; then
    testing::phase::warn "PostgreSQL not available, skipping database integration tests"
fi

if [ "$REDIS_AVAILABLE" = false ]; then
    testing::phase::warn "Redis not available, skipping cache integration tests"
fi

# Run Go integration tests with database
if [ "$POSTGRES_AVAILABLE" = true ]; then
    testing::phase::log "Running Go integration tests with database..."

    cd api

    # Set test database environment
    export TEST_POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
    export TEST_POSTGRES_PORT="${POSTGRES_PORT:-5432}"
    export TEST_POSTGRES_USER="${POSTGRES_USER:-postgres}"
    export TEST_POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
    export TEST_POSTGRES_DB="swarm_manager_test"

    # Create test database if it doesn't exist
    psql -h "$TEST_POSTGRES_HOST" -p "$TEST_POSTGRES_PORT" -U "$TEST_POSTGRES_USER" -c "CREATE DATABASE $TEST_POSTGRES_DB;" 2>/dev/null || true

    # Run integration tests (not in short mode)
    if go test -tags=testing -v -run "TestCreate|TestGet|TestUpdate|TestDelete|TestAgent|TestProblem|TestMetrics|TestConfig|TestCalculatePriority" . 2>&1 | tee /tmp/swarm-integration-test.log; then
        testing::phase::success "Integration tests passed"
    else
        testing::phase::error "Integration tests failed"
        cat /tmp/swarm-integration-test.log
        cd ..
        testing::phase::end_with_summary "Integration tests failed" 1
    fi

    # Cleanup test database
    psql -h "$TEST_POSTGRES_HOST" -p "$TEST_POSTGRES_PORT" -U "$TEST_POSTGRES_USER" -c "DROP DATABASE IF EXISTS $TEST_POSTGRES_DB;" 2>/dev/null || true

    cd ..
else
    testing::phase::log "Skipping database integration tests (no PostgreSQL)"
fi

# Test API health endpoint
testing::phase::log "Testing API health check integration..."

# Check if API is running
API_PORT="${API_PORT:-8080}"
if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    testing::phase::success "API health check successful"
else
    testing::phase::warn "API not running on port $API_PORT, skipping live API tests"
fi

# Test file system operations
testing::phase::log "Testing file system integration..."

# Create temporary test directory
TEST_TASK_DIR=$(mktemp -d)
trap "rm -rf $TEST_TASK_DIR" EXIT

# Test task file operations
cd api
if go test -tags=testing -v -run "TestSaveTaskToFile|TestReadTasksFromFolder|TestFindTaskFile" . >/tmp/fs-test.log 2>&1; then
    testing::phase::success "File system operations tests passed"
else
    testing::phase::error "File system operations tests failed"
    cat /tmp/fs-test.log
    cd ..
    testing::phase::end_with_summary "File system tests failed" 1
fi
cd ..

# Test task workflow integration
testing::phase::log "Testing task workflow integration..."

# This would test: create -> analyze -> execute -> complete
# For now, we verify the structure is in place
if grep -q "func executeTask" api/main.go && grep -q "func analyzeTask" api/main.go; then
    testing::phase::success "Task workflow functions present"
else
    testing::phase::warn "Some task workflow functions may be missing"
fi

testing::phase::end_with_summary "Integration tests completed"
