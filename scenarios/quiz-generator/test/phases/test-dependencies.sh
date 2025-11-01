#!/bin/bash
# Dependency validation for quiz-generator scenario
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

check_resource() {
  local label="$1"
  local status_cmd="$2"
  local required="$3"

  if eval "$status_cmd" >/dev/null 2>&1; then
    log::success "✅ ${label} available"
    testing::phase::add_test passed
    return 0
  fi

  if [ "$required" = "true" ]; then
    testing::phase::add_error "${label} unavailable"
    testing::phase::add_test failed
  else
    testing::phase::add_warning "${label} unavailable (optional)"
    testing::phase::add_test skipped
  fi
  return 1
}

if command -v resource-postgres >/dev/null 2>&1; then
  check_resource "PostgreSQL resource" "resource-postgres status" true
else
  testing::phase::add_warning "resource-postgres helper not found; skipping direct status check"
  testing::phase::add_test skipped
fi

if command -v resource-ollama >/dev/null 2>&1; then
  check_resource "Ollama resource" "resource-ollama status" true
else
  testing::phase::add_warning "resource-ollama helper not found; skipping direct status check"
  testing::phase::add_test skipped
fi

if command -v resource-redis >/dev/null 2>&1; then
  check_resource "Redis cache" "resource-redis status" false
fi

if command -v resource-qdrant >/dev/null 2>&1; then
  check_resource "Qdrant vector store" "resource-qdrant status" false
fi

if command -v go >/dev/null 2>&1; then
  if testing::phase::check "Go module graph" bash -c 'cd api && go list ./... >/dev/null'; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "Go toolchain not installed; skipping module check"
  testing::phase::add_test skipped
fi

if command -v npm >/dev/null 2>&1; then
  if testing::phase::check "UI dependency install --dry-run" bash -c 'cd ui && npm install --dry-run >/dev/null'; then
    testing::phase::add_test passed
  else
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "npm not installed; skipping UI dependency check"
  testing::phase::add_test skipped
fi

if command -v resource-postgres >/dev/null 2>&1; then
  SCHEMA_RAW=$(resource-postgres query "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='quiz_generator' AND table_name IN ('quizzes','questions','quiz_results');" 2>/dev/null || echo "0")
  SCHEMA_COUNT=$(printf '%s' "$SCHEMA_RAW" | tr -cd '0-9' || echo "0")
  if [ "$SCHEMA_COUNT" = "3" ]; then
    log::success "✅ Quiz schema tables present"
    testing::phase::add_test passed
  else
    testing::phase::add_error "Quiz schema tables missing (found ${SCHEMA_COUNT:-0})"
    testing::phase::add_test failed
  fi
else
  testing::phase::add_warning "resource-postgres helper not available; skipping schema validation"
  testing::phase::add_test skipped
fi

testing::phase::end_with_summary "Dependency validation completed"
