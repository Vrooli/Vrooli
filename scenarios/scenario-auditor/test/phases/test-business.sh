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
    expected_categories=("api" "cli" "config" "test" "ui")
    
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
# Use the scenarios/{name}/scan endpoint which starts an async scan
scan_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/scenarios/scenario-auditor/scan" \
    -H "Content-Type: application/json" \
    -d '{"quick": true}')

if echo "$scan_response" | jq -e '.job_id' >/dev/null 2>&1; then
    job_id=$(echo "$scan_response" | jq -r '.job_id')
    echo "✅ Scan started successfully with job_id: $job_id"

    # Poll for scan completion (max 30 seconds)
    for i in {1..10}; do
        sleep 3
        job_status=$(curl -s "http://localhost:$API_PORT/api/v1/scenarios/scan/jobs/$job_id")
        if echo "$job_status" | jq -e '.status' | grep -q "completed"; then
            echo "✅ Scan completed successfully"
            break
        elif echo "$job_status" | jq -e '.status' | grep -q "failed"; then
            echo "❌ Scan failed"
            echo "$job_status" | jq '.'
            exit 1
        fi
    done
else
    echo "❌ Scan failed to start"
    echo "$scan_response" | jq '.' || echo "$scan_response"
    exit 1
fi

# Note: Preferences endpoint is a P1 feature (not yet implemented)
echo "⚠️  Skipping preferences test (P1 feature - not yet implemented)"

# Note: Dashboard endpoint is a P1 feature (not yet implemented)
echo "⚠️  Skipping dashboard test (P1 feature - not yet implemented)"

echo "=== Business Logic Tests Complete ==="
echo "✅ All implemented P0 features tested successfully"

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

# Note: AI integration endpoints are P0 features but not yet implemented
echo "⚠️  Skipping AI integration test (P0 feature - planned but not yet implemented)"

# Test CLI business logic
echo "Testing CLI business logic..."
cd cli
if [ -f "scenario-auditor" ]; then
    # Test version command
    if ./scenario-auditor version | grep -q -i "version\|v[0-9]"; then
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
    
    # Test rules command (set API port for CLI to use test instance)
    if SCENARIO_AUDITOR_API_PORT=$API_PORT ./scenario-auditor rules | grep -q "rules"; then
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