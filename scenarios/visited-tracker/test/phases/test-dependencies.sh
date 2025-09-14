#!/bin/bash
# Simplified Dependencies validation phase - <30 seconds
# Validates required resources using resource CLI commands
set -uo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

echo "=== Dependencies Phase (Target: <30s) ==="
start_time=$(date +%s)

error_count=0
warning_count=0

# Check PostgreSQL (required) using resource CLI
echo "🔍 Checking PostgreSQL (required)..."
if resource-postgres test smoke >/dev/null 2>&1; then
    log::success "✅ PostgreSQL is running and ready"
else
    log::error "❌ PostgreSQL smoke test failed"
    echo "   Start with: vrooli resource start postgres"
    ((error_count++))
fi

# Check Redis (optional) using resource CLI
echo "🔍 Checking Redis (optional)..."
if resource-redis test smoke >/dev/null 2>&1; then
    log::success "✅ Redis is running and responsive"
else
    log::warning "⚠️  Redis smoke test failed (optional dependency)"
    echo "   Start with: vrooli resource start redis"
    ((warning_count++))
fi

# Check Go environment
echo "🔍 Checking Go environment..."
if go version >/dev/null 2>&1; then
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)
    log::success "✅ Go is available: $go_version"
else
    log::error "❌ Go is not installed"
    ((error_count++))
fi

# Check Node.js environment
echo "🔍 Checking Node.js environment..."
if node --version >/dev/null 2>&1; then
    node_version=$(node --version)
    log::success "✅ Node.js is available: $node_version"
else
    log::error "❌ Node.js is not installed"
    ((error_count++))
fi

# Check essential tools
echo "🔍 Checking essential tools..."
essential_tools=("jq" "curl")

for tool in "${essential_tools[@]}"; do
    if "$tool" --version >/dev/null 2>&1; then
        log::success "✅ $tool is available"
    else
        log::error "❌ $tool is not available"
        ((error_count++))
    fi
done

# Performance check
end_time=$(date +%s)
duration=$((end_time - start_time))
echo ""

# Results
if [ $error_count -eq 0 ]; then
    if [ $warning_count -eq 0 ]; then
        log::success "✅ Dependencies validation completed successfully in ${duration}s"
    else
        log::success "✅ Dependencies validation completed with $warning_count warnings in ${duration}s"
    fi
else
    log::error "❌ Dependencies validation failed with $error_count errors and $warning_count warnings in ${duration}s"
fi

if [ $duration -gt 30 ]; then
    log::warning "⚠️  Dependencies phase exceeded 30s target"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi