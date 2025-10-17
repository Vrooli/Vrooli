#!/bin/bash

set -e

echo "=== Service JSON Validation Tests ==="

# Check service.json exists
echo "Checking service.json file..."
if [ ! -f ".vrooli/service.json" ]; then
    echo "❌ service.json not found"
    exit 1
fi
echo "✅ service.json exists"

# Validate JSON syntax
echo "Validating JSON syntax..."
if ! jq empty .vrooli/service.json 2>/dev/null; then
    echo "❌ Invalid JSON syntax"
    exit 1
fi
echo "✅ Valid JSON syntax"

# Check required top-level fields
echo "Checking required fields..."

required_fields=(
    "service.name"
    "service.displayName"
    "service.description"
    "service.version"
    "ports.api.env_var"
    "ports.ui.env_var"
    "lifecycle.version"
    "lifecycle.health"
    "lifecycle.setup"
    "lifecycle.develop"
    "lifecycle.test"
    "lifecycle.stop"
)

for field in "${required_fields[@]}"; do
    if jq -e ".$field" .vrooli/service.json >/dev/null 2>&1; then
        echo "✅ Field exists: $field"
    else
        echo "❌ Missing field: $field"
        exit 1
    fi
done

# Validate lifecycle structure
echo "Validating lifecycle configuration..."

# Check setup steps
if jq -e '.lifecycle.setup.steps | length > 0' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ Setup steps defined"
else
    echo "❌ No setup steps found"
    exit 1
fi

# Check develop steps
if jq -e '.lifecycle.develop.steps | length > 0' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ Develop steps defined"
else
    echo "❌ No develop steps found"
    exit 1
fi

# Check test steps
if jq -e '.lifecycle.test.steps | length > 0' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ Test steps defined"
else
    echo "❌ No test steps found"
    exit 1
fi

# Validate health check configuration
echo "Validating health check configuration..."

if jq -e '.lifecycle.health.endpoints.api' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ API health endpoint configured"
else
    echo "❌ Missing API health endpoint"
    exit 1
fi

if jq -e '.lifecycle.health.endpoints.ui' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ UI health endpoint configured"
else
    echo "❌ Missing UI health endpoint"
    exit 1
fi

# Validate port configuration
echo "Validating port configuration..."

api_port_range=$(jq -r '.ports.api.range' .vrooli/service.json)
if [[ "$api_port_range" == "15000-19999" ]]; then
    echo "✅ API port range correct: $api_port_range"
else
    echo "❌ Invalid API port range: $api_port_range"
    exit 1
fi

# Check for required resources
echo "Checking resource dependencies..."

if jq -e '.resources.postgres' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ PostgreSQL resource configured"
else
    echo "⚠️  PostgreSQL resource not configured"
fi

if jq -e '.resources["claude-code"]' .vrooli/service.json >/dev/null 2>&1; then
    echo "✅ Claude Code resource configured"
else
    echo "⚠️  Claude Code resource not configured"
fi

# Validate test steps reference existing scripts
echo "Validating test step scripts..."

test_scripts=$(jq -r '.lifecycle.test.steps[].run' .vrooli/service.json | grep -E '\.sh$' | sed 's/^cd test && \.\/phases\///' || true)

if [ -n "$test_scripts" ]; then
    for script in $test_scripts; do
        if [ -f "test/phases/$script" ]; then
            echo "✅ Test script exists: $script"
        else
            echo "⚠️  Test script not found: $script (non-blocking)"
        fi
    done
else
    echo "⚠️  No test scripts found in lifecycle.test.steps"
fi

echo "=== Service JSON Validation Complete ==="
