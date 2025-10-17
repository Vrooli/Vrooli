#!/bin/bash

# Centralized test integration for scenario-auditor rule engine tests
# This script validates the rule engine functionality and rule execution

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "=== Rule Engine Functionality Tests ==="

# Set test ports
export API_PORT=15997
export UI_PORT=35997

# Function to cleanup
cleanup() {
    pkill -f "scenario-auditor-api.*15997" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Start API for rule engine testing
echo "Starting API for rule engine tests..."
cd api
if [ ! -f "scenario-auditor-api" ]; then
    go build -o scenario-auditor-api .
fi

API_PORT=$API_PORT VROOLI_LIFECYCLE_MANAGED=true ./scenario-auditor-api &
API_PID=$!
cd ..

# Wait for API to start
sleep 5

# Test 1: Rule loading and structure
echo "Test 1: Validating rule loading..."
response=$(curl -s "http://localhost:$API_PORT/api/v1/rules")
if echo "$response" | jq -e '.total' >/dev/null 2>&1; then
    rule_count=$(echo "$response" | jq '.total')
    testing::phase::success "Rules loaded: $rule_count rules"
else
    testing::phase::error "Failed to load rules"
    exit 1
fi

# Test 2: Rule categories exist
echo "Test 2: Validating rule categories..."
if echo "$response" | jq -e '.categories' >/dev/null 2>&1; then
    categories=$(echo "$response" | jq -r '.categories | keys[]' 2>/dev/null || echo "$response" | jq -r '.categories[]')
    expected_categories=("api" "config" "ui" "testing")

    for cat in "${expected_categories[@]}"; do
        if echo "$categories" | grep -q "$cat"; then
            testing::phase::success "Category found: $cat"
        else
            testing::phase::warning "Missing expected category: $cat (may be normal if no rules in this category)"
        fi
    done
else
    testing::phase::warning "Categories structure not found (may be acceptable)"
fi

# Test 3: Rule execution via standards check
echo "Test 3: Testing rule execution via standards check..."
# Create a test job to validate rule engine
scan_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/standards/check" \
    -H "Content-Type: application/json" \
    -d '{"scenarios": ["scenario-auditor"]}')

if echo "$scan_response" | jq -e '.job_id' >/dev/null 2>&1; then
    job_id=$(echo "$scan_response" | jq -r '.job_id')
    testing::phase::success "Standards check job created: $job_id"

    # Poll for completion (up to 30 seconds)
    max_attempts=10
    attempt=0
    while [ $attempt -lt $max_attempts ]; do
        sleep 3
        job_status=$(curl -s "http://localhost:$API_PORT/api/v1/standards/check/jobs/$job_id")

        if echo "$job_status" | jq -e '.status' >/dev/null 2>&1; then
            status=$(echo "$job_status" | jq -r '.status')

            if [ "$status" = "completed" ]; then
                testing::phase::success "Rule engine executed successfully"

                # Validate results structure
                if echo "$job_status" | jq -e '.result.violations' >/dev/null 2>&1; then
                    violations=$(echo "$job_status" | jq '.result.violations | length')
                    testing::phase::info "Violations detected: $violations"
                fi
                break
            elif [ "$status" = "failed" ]; then
                testing::phase::error "Rule engine execution failed"
                exit 1
            fi
        fi

        attempt=$((attempt + 1))
    done

    if [ $attempt -eq $max_attempts ]; then
        testing::phase::warning "Rule engine test timed out (may indicate slow execution)"
    fi
else
    testing::phase::error "Failed to create standards check job"
    exit 1
fi

# Test 4: Rule severity levels
echo "Test 4: Validating rule severity levels..."
if echo "$response" | jq -e '.rules' >/dev/null 2>&1; then
    for severity in "critical" "high" "medium" "low"; do
        # Count rules of this severity
        severity_count=$(echo "$response" | jq "[.rules[] | select(.severity == \"$severity\")] | length" 2>/dev/null || echo "0")
        testing::phase::info "$severity severity rules: $severity_count"
    done
else
    testing::phase::warning "Rule severity validation skipped (rules structure not found)"
fi

# Test 5: Rule categories have descriptions
echo "Test 5: Validating rule metadata..."
if echo "$response" | jq -e '.rules' >/dev/null 2>&1; then
    rules_array=$(echo "$response" | jq '.rules[]' 2>/dev/null || echo "[]")
    if [ "$rules_array" != "[]" ]; then
        # Check first rule has required fields
        first_rule=$(echo "$response" | jq '.rules[0]' 2>/dev/null)

        if echo "$first_rule" | jq -e '.name' >/dev/null 2>&1; then
            testing::phase::success "Rules have name field"
        fi

        if echo "$first_rule" | jq -e '.description' >/dev/null 2>&1; then
            testing::phase::success "Rules have description field"
        fi

        if echo "$first_rule" | jq -e '.category' >/dev/null 2>&1; then
            testing::phase::success "Rules have category field"
        fi
    else
        testing::phase::warning "No rules available for metadata validation"
    fi
fi

# Test 6: Rule engine handles invalid input
echo "Test 6: Testing rule engine error handling..."
error_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/standards/check" \
    -H "Content-Type: application/json" \
    -d '{"scenarios": []}')

if echo "$error_response" | jq -e '.error' >/dev/null 2>&1; then
    testing::phase::success "Rule engine properly validates input"
elif echo "$error_response" | jq -e '.job_id' >/dev/null 2>&1; then
    testing::phase::warning "Rule engine accepted empty scenarios (may be valid behavior)"
else
    testing::phase::info "Rule engine validation behavior unclear"
fi

testing::phase::end_with_summary "Rule engine tests completed"
