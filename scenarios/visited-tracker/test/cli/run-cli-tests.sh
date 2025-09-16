#!/bin/bash
# CLI test runner - orchestrates all BATS tests without mutating the repository
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI_DIR="$SCENARIO_DIR/cli"
API_PORT="${API_PORT:-17695}"

log::info "🔧 Running CLI BATS tests..."

if ! command -v bats >/dev/null 2>&1; then
    log::error "❌ BATS is not installed"
    echo "   Install with: npm install -g bats (or use your package manager)"
    exit 1
fi

if [ ! -d "$CLI_DIR" ]; then
    log::error "❌ CLI directory not found: $CLI_DIR"
    exit 1
fi

log::info "🔍 Looking for BATS files in: $CLI_DIR"
mapfile -t bats_files < <(find "$CLI_DIR" -maxdepth 1 -name "*.bats" -type f | sort)

if [ ${#bats_files[@]} -eq 0 ]; then
    log::error "❌ No CLI BATS test files found in $CLI_DIR"
    echo "   Add at least one .bats file to cover the CLI surface before enabling this phase"
    exit 1
fi

test_count=0
failed_count=0

echo ""
echo "🧪 Running ${#bats_files[@]} BATS test file(s)..."

for bats_file in "${bats_files[@]}"; do
    echo ""
    log::info "📋 Running $(basename "$bats_file")..."
    if API_PORT="$API_PORT" bats "$bats_file" --tap; then
        log::success "✅ $(basename "$bats_file") passed"
    else
        log::error "❌ $(basename "$bats_file") failed"
        ((failed_count+=1))
    fi
    ((test_count+=1))
done

echo ""
echo "📊 CLI Test Summary:"
echo "   Test files run: $test_count"
echo "   Test files passed: $((test_count - failed_count))"
echo "   Test files failed: $failed_count"

if [ $failed_count -eq 0 ]; then
    log::success "✅ All $test_count CLI test suites passed"
    exit 0
else
    log::error "❌ $failed_count of $test_count CLI test suites failed"
    exit 1
fi
