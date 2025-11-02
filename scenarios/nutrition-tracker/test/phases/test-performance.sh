#!/bin/bash
# Run targeted performance benchmarks and concurrency tests

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

if [ ! -f "api/go.mod" ]; then
  testing::phase::add_warning "Go module missing; skipping performance checks"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance phase skipped"
fi

if ! command -v go >/dev/null 2>&1; then
  testing::phase::add_warning "Go toolchain not available; skipping performance tests"
  testing::phase::add_test skipped
  testing::phase::end_with_summary "Performance phase skipped"
fi

run_go_test() {
  local description="$1"
  shift
  if testing::phase::timed_exec "$description" "$@"; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
    testing::phase::add_error "$description failed"
  fi
}

run_go_test "Benchmark suite" bash -c 'cd api && go test -tags testing -run ^$ -bench . -benchmem'
run_go_test "Concurrent request test" bash -c 'cd api && go test -tags testing -run TestConcurrentRequests -timeout 120s -v'
run_go_test "Response time regression" bash -c 'cd api && go test -tags testing -run TestResponseTime -timeout 120s -v'
run_go_test "Database connection pool test" bash -c 'cd api && go test -tags testing -run TestDatabaseConnectionPool -timeout 120s -v'

testing::phase::end_with_summary "Performance benchmarks completed"
