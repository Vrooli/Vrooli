#!/bin/bash
# Unit tests for social-media-scheduler scenario

# Initialize testing phase with centralized infrastructure
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

# Source centralized test runners
source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

# Navigate to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Set test database configuration
export TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgres://postgres:postgres@localhost:5432/vrooli_social_media_scheduler_test?sslmode=disable}"
export TEST_REDIS_URL="${TEST_REDIS_URL:-redis://localhost:6379/15}"
export TEST_MINIO_URL="${TEST_MINIO_URL:-http://localhost:9000}"
export TEST_OLLAMA_URL="${TEST_OLLAMA_URL:-http://localhost:11434}"

# Create test database if it doesn't exist
echo "🔧 Setting up test database..."
psql -h localhost -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'vrooli_social_media_scheduler_test'" | grep -q 1 || \
    psql -h localhost -U postgres -c "CREATE DATABASE vrooli_social_media_scheduler_test"

# Initialize test database schema
if [ -f "initialization/storage/schema.sql" ]; then
    echo "📊 Initializing test database schema..."
    psql -h localhost -U postgres -d vrooli_social_media_scheduler_test -f initialization/storage/schema.sql &> /dev/null || true
fi

# Run all unit tests using centralized test runner
testing::unit::run_all_tests \
    --go-dir "api" \
    --skip-node \
    --skip-python \
    --coverage-warn 80 \
    --coverage-error 50 \
    --build-tags "testing"

# Clean up test database
echo "🧹 Cleaning up test data..."
psql -h localhost -U postgres -d vrooli_social_media_scheduler_test -c "
    DELETE FROM scheduled_posts WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM social_accounts WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM campaigns WHERE created_at < NOW() - INTERVAL '1 hour';
    DELETE FROM users WHERE email LIKE 'test%';
" &> /dev/null || true

# End testing phase with summary
testing::phase::end_with_summary "Unit tests completed"
