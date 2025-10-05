#!/bin/bash
# Phase 2: Dependencies validation - <30 seconds
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Phase 2: Dependencies Validation ==="
start_time=$(date +%s)
error_count=0
test_count=0

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

# Check Go dependencies
((test_count++))
if command -v go >/dev/null 2>&1; then
  go_version=$(go version | awk '{print $3}')
  log::success "✅ Go available: $go_version"

  # Check if go.mod exists and verify dependencies
  if [ -f "api/go.mod" ]; then
    cd api
    if go mod verify >/dev/null 2>&1; then
      log::success "✅ Go dependencies verified"
    else
      log::warning "⚠️  Go dependencies may need updating (run: go mod download)"
    fi
    cd ..
  fi
else
  log::error "❌ Go not found"
  ((error_count++))
fi

# Check Node.js (for UI)
((test_count++))
if command -v node >/dev/null 2>&1; then
  node_version=$(node --version)
  log::success "✅ Node.js available: $node_version"

  # Check UI dependencies if package.json exists
  if [ -f "ui/package.json" ]; then
    if [ -d "ui/node_modules" ]; then
      log::success "✅ UI dependencies installed"
    else
      log::warning "⚠️  UI dependencies not installed (run: cd ui && npm install)"
    fi
  fi
else
  log::warning "⚠️  Node.js not found (needed for UI)"
fi

# Check BATS (for CLI testing)
((test_count++))
if command -v bats >/dev/null 2>&1; then
  bats_version=$(bats --version | head -1)
  log::success "✅ BATS available: $bats_version"
else
  log::warning "⚠️  BATS not found (needed for CLI tests)"
fi

# Check jq
((test_count++))
if command -v jq >/dev/null 2>&1; then
  jq_version=$(jq --version)
  log::success "✅ jq available: $jq_version"
else
  log::error "❌ jq not found (required for JSON processing)"
  ((error_count++))
fi

# Check curl
((test_count++))
if command -v curl >/dev/null 2>&1; then
  log::success "✅ curl available"
else
  log::error "❌ curl not found (required for API testing)"
  ((error_count++))
fi

# Check resource availability (non-critical)
log::info "Checking resource availability..."
required_resources=("postgres" "qdrant" "redis" "ollama" "n8n")
for resource in "${required_resources[@]}"; do
  ((test_count++))
  if vrooli resource status "$resource" --json 2>/dev/null | jq -e '.status == "running"' >/dev/null 2>&1; then
    log::success "✅ Resource available: $resource"
  else
    log::warning "⚠️  Resource not running: $resource (will be started if needed)"
  fi
done

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))

if [ $error_count -eq 0 ]; then
  log::success "✅ Dependencies validation completed in ${duration}s"
  log::success "   Tests run: $test_count, Errors: $error_count"
  exit 0
else
  log::error "❌ Dependencies validation failed with $error_count errors in ${duration}s"
  exit 1
fi
