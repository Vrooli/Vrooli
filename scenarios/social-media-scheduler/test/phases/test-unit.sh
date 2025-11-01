#!/bin/bash
# Unit tests for social-media-scheduler scenario

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

source "${APP_ROOT}/scripts/scenarios/testing/unit/run-all.sh"

cd "$TESTING_PHASE_SCENARIO_DIR"

if ! command -v go >/dev/null 2>&1; then
  testing::phase::add_warning "Go toolchain not available; skipping unit tests"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Unit tests skipped"
fi

if ! command -v psql >/dev/null 2>&1; then
  testing::phase::add_warning "psql client not available; skipping database-assisted unit tests"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Unit tests skipped"
fi

export TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgres://postgres:postgres@localhost:5432/vrooli_social_media_scheduler_test?sslmode=disable}"
export TEST_REDIS_URL="${TEST_REDIS_URL:-redis://localhost:6379/15}"
export TEST_MINIO_URL="${TEST_MINIO_URL:-http://localhost:9000}"
export TEST_OLLAMA_URL="${TEST_OLLAMA_URL:-http://localhost:11434}"

echo "🔧 Setting up test database..."
psql -h localhost -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'vrooli_social_media_scheduler_test'" | grep -q 1 || \
  psql -h localhost -U postgres -c "CREATE DATABASE vrooli_social_media_scheduler_test"

if [ -f "initialization/storage/schema.sql" ]; then
  echo "📊 Applying schema to test database..."
  psql -h localhost -U postgres -d vrooli_social_media_scheduler_test -f initialization/storage/schema.sql >/dev/null 2>&1 || true
fi

if testing::unit::run_all_tests \
  --go-dir "api" \
  --skip-node \
  --skip-python \
  --coverage-warn 80 \
  --coverage-error 50; then
  testing::phase::add_test passed
else
  testing::phase::add_error "Unit test runner reported failures"
  testing::phase::add_test failed
fi

if command -v psql >/dev/null 2>&1; then
  echo "🧹 Cleaning up test data..."
  psql -h localhost -U postgres -d vrooli_social_media_scheduler_test -c "TRUNCATE TABLE scheduled_posts, social_accounts, campaigns, users RESTART IDENTITY" >/dev/null 2>&1 || true
fi

testing::phase::end_with_summary "Unit tests completed"
