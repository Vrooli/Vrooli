#!/bin/bash

# Centralized test integration for scenario-auditor unit tests  
# This script integrates with Vrooli's centralized testing infrastructure

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Scenario-auditor has rule tests with build tag 'ruletests'
# We need to run tests with this tag to get proper coverage
echo "🧪 Running all unit tests..."
echo ""

cd api

echo "🐹 Running Go unit tests..."
echo "📦 Downloading Go module dependencies..."
if ! go mod download; then
    echo "❌ Failed to download Go dependencies"
    exit 1
fi

echo "🧪 Running Go tests with ruletests tag..."
if ! DEV_MODE=true go test -tags=ruletests -cover -coverprofile=coverage.out -timeout 60s ./...; then
    echo "❌ Go unit tests failed"
    exit 1
fi

# Filter coverage to only include business logic packages (exclude main infrastructure code)
# Note: email-triage has all code in main package, so we use overall coverage
go tool cover -func=coverage.out | grep -E '^email-triage/(handlers|services|models)' > coverage_filtered.txt 2>/dev/null || true
total_filtered=$(go tool cover -func=coverage.out | grep -E '^email-triage/(handlers|services|models)' 2>/dev/null | awk '{s+=$3; n++} END {if(n>0) printf "%.1f", s/n; else print ""}' || echo "")

echo "✅ Go unit tests completed successfully"
echo ""
echo "📊 Go Test Coverage Summary:"
coverage_line=$(go tool cover -func=coverage.out | tail -1)
echo "$coverage_line"

# Extract coverage percentage (overall)
coverage_percent=$(echo "$coverage_line" | grep -o '[0-9]*\.[0-9]*%' | sed 's/%//' | head -1)

# Use filtered coverage for rules packages (the testable business logic) if available
if [ -n "$total_filtered" ] && [ "$total_filtered" != "0.0" ]; then
    coverage_num=$(echo "$total_filtered" | cut -d. -f1)
    echo ""
    echo "ℹ️  Overall coverage: $coverage_percent% (includes untested infrastructure code)"
    echo "ℹ️  Business logic coverage: ${total_filtered}% (handlers/services/models being tested)"
    echo ""

    if [ "$coverage_num" -lt 25 ]; then
        echo "❌ ERROR: Business logic test coverage (${total_filtered}%) is below error threshold (25%)"
        echo "   This indicates insufficient test coverage. Please add more comprehensive tests."
        exit 1
    elif [ "$coverage_num" -lt 50 ]; then
        echo "⚠️  WARNING: Business logic test coverage (${total_filtered}%) is below warning threshold (50%)"
        echo "   Consider adding more tests to improve code coverage."
    else
        echo "✅ Business logic test coverage (${total_filtered}%) meets quality thresholds"
    fi
else
    # Fallback to overall coverage (standard for email-triage which uses main package)
    if [ -n "$coverage_percent" ]; then
        coverage_num=$(echo "$coverage_percent" | cut -d. -f1)
        echo ""
        echo "ℹ️  Note: email-triage uses main package structure; using overall coverage"
        if [ "$coverage_num" -lt 25 ]; then
            echo "❌ ERROR: Go test coverage ($coverage_percent%) is below error threshold (25%)"
            echo "   This indicates insufficient test coverage. Please add more comprehensive tests."
            exit 1
        elif [ "$coverage_num" -lt 50 ]; then
            echo "⚠️  WARNING: Go test coverage ($coverage_percent%) is below warning threshold (50%)"
            echo "   Consider adding more tests to improve code coverage."
        else
            echo "✅ Go test coverage ($coverage_percent%) meets quality thresholds"
        fi
    fi
fi

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Unit tests completed"
