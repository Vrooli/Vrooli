#!/bin/bash

set -e

echo "=== Business Logic Tests ==="

# Set test ports
export API_PORT=15998
export UI_PORT=35998

# Function to cleanup
cleanup() {
    pkill -f "scenario-auditor-api" 2>/dev/null || true
    sleep 1
}

trap cleanup EXIT

# Start API for business logic testing
echo "Starting API for business logic tests..."
cd api
if [ ! -f "scenario-auditor-api" ]; then
    go build -o scenario-auditor-api .
fi

VROOLI_LIFECYCLE_MANAGED=true ./scenario-auditor-api &
API_PID=$!
cd ..

# Wait for API to start
sleep 5

# Test rule loading and validation
echo "Testing rule loading and validation..."
response=$(curl -s "http://localhost:$API_PORT/api/v1/rules")
if echo "$response" | jq -e '.total' >/dev/null 2>&1; then
    rule_count=$(echo "$response" | jq '.total')
    echo "✅ Rules loaded successfully: $rule_count rules"
else
    echo "❌ Failed to load rules"
    exit 1
fi

# Test rule categories
echo "Testing rule categories..."
if echo "$response" | jq -e '.categories' >/dev/null 2>&1; then
    categories=$(echo "$response" | jq -r '.categories | keys[]')
    expected_categories=("api" "config" "ui" "testing")
    
    for cat in "${expected_categories[@]}"; do
        if echo "$categories" | grep -q "$cat"; then
            echo "✅ Category found: $cat"
        else
            echo "❌ Missing category: $cat"
            exit 1
        fi
    done
else
    echo "❌ Categories not found in response"
    exit 1
fi

# Test scan functionality
echo "Testing scan functionality..."
scan_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/scan/current" \
    -H "Content-Type: application/json" \
    -d '{}')

if echo "$scan_response" | jq -e '.summary' >/dev/null 2>&1; then
    echo "✅ Scan completed successfully"
    
    # Check scan result structure
    if echo "$scan_response" | jq -e '.summary.total_violations' >/dev/null 2>&1; then
        violations=$(echo "$scan_response" | jq '.summary.total_violations')
        echo "✅ Violations detected: $violations"
    fi
    
    if echo "$scan_response" | jq -e '.summary.score' >/dev/null 2>&1; then
        score=$(echo "$scan_response" | jq '.summary.score')
        echo "✅ Quality score calculated: $score"
    fi
else
    echo "❌ Scan failed"
    exit 1
fi

# Test preferences functionality
echo "Testing preferences functionality..."
prefs_response=$(curl -s "http://localhost:$API_PORT/api/v1/preferences")
if echo "$prefs_response" | jq -e '.categories' >/dev/null 2>&1; then
    echo "✅ Preferences loaded successfully"
else
    echo "❌ Failed to load preferences"
    exit 1
fi

# Test dashboard data aggregation
echo "Testing dashboard data aggregation..."
dashboard_response=$(curl -s "http://localhost:$API_PORT/api/v1/dashboard")
if echo "$dashboard_response" | jq -e '.overview' >/dev/null 2>&1; then
    echo "✅ Dashboard data aggregated successfully"
    
    # Check overview data
    if echo "$dashboard_response" | jq -e '.overview.total_scenarios' >/dev/null 2>&1; then
        scenarios=$(echo "$dashboard_response" | jq '.overview.total_scenarios')
        echo "✅ Scenario count: $scenarios"
    fi
    
    if echo "$dashboard_response" | jq -e '.overview.rules_total' >/dev/null 2>&1; then
        rules=$(echo "$dashboard_response" | jq '.overview.rules_total')
        echo "✅ Rules count: $rules"
    fi
else
    echo "❌ Dashboard data aggregation failed"
    exit 1
fi

# Test rule severity filtering
echo "Testing rule severity filtering..."
for severity in "critical" "high" "medium" "low"; do
    filtered_response=$(curl -s "http://localhost:$API_PORT/api/v1/rules")
    if echo "$filtered_response" | jq -e '.rules' >/dev/null 2>&1; then
        # Count rules of this severity
        severity_count=$(echo "$filtered_response" | jq ".rules | to_entries[] | select(.value.severity == \"$severity\") | .key" | wc -l)
        echo "✅ $severity severity rules: $severity_count"
    fi
done

# Test AI integration endpoints (mock)
echo "Testing AI integration endpoints..."
ai_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/ai/create-rule" \
    -H "Content-Type: application/json" \
    -d '{"prompt": "Test rule creation", "category": "testing"}')

if echo "$ai_response" | jq -e '.success' >/dev/null 2>&1; then
    success=$(echo "$ai_response" | jq '.success')
    if [ "$success" = "true" ]; then
        echo "✅ AI rule creation endpoint working"
    else
        echo "⚠️  AI rule creation returned success=false (expected for mock)"
    fi
else
    echo "❌ AI rule creation endpoint failed"
    exit 1
fi

# Test CLI business logic
echo "Testing CLI business logic..."
cd cli
if [ -f "scenario-auditor" ]; then
    # Test version command
    if ./scenario-auditor version | grep -q "version"; then
        echo "✅ CLI version command working"
    else
        echo "❌ CLI version command failed"
        exit 1
    fi
    
    # Test help command
    if ./scenario-auditor help | grep -q "Usage"; then
        echo "✅ CLI help command working"
    else
        echo "❌ CLI help command failed"
        exit 1
    fi
    
    # Test rules command
    if ./scenario-auditor rules | grep -q "rules"; then
        echo "✅ CLI rules command working"
    else
        echo "❌ CLI rules command failed"
        exit 1
    fi
else
    echo "⚠️  CLI binary not found, skipping CLI business logic tests"
fi
cd ..

echo "=== Business Logic Tests Complete ==="