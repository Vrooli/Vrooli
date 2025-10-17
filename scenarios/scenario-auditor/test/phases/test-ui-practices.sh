#!/bin/bash

# Centralized test integration for scenario-auditor UI practices validation
# This script validates UI testing best practices enforcement

set -e

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "=== UI Testing Best Practices Validation ==="

# Test 1: UI component structure
echo "Test 1: Validating UI component structure..."
if [ -d "ui/src" ]; then
    echo "✅" "UI source directory exists"

    # Check for components directory
    if [ -d "ui/src/components" ]; then
        echo "✅" "Components directory exists"
    else
        echo "⚠️ " "No components directory found"
    fi

    # Check for pages directory
    if [ -d "ui/src/pages" ]; then
        echo "✅" "Pages directory exists"
    else
        echo "⚠️ " "No pages directory found"
    fi
else
    echo "❌" "UI source directory not found"
    exit 1
fi

# Test 2: UI testing best practices in code
echo "Test 2: Checking for UI testing best practices..."

# Check for data-testid attributes (best practice for testing)
if grep -r "data-testid" ui/src/ --include="*.tsx" --include="*.ts" >/dev/null 2>&1; then
    testid_count=$(grep -r "data-testid" ui/src/ --include="*.tsx" --include="*.ts" | wc -l)
    echo "✅" "Found $testid_count data-testid attributes (good for testing)"
else
    echo "⚠️ " "No data-testid attributes found (consider adding for better test automation)"
fi

# Check for semantic HTML
if grep -r "aria-" ui/src/ --include="*.tsx" --include="*.ts" >/dev/null 2>&1; then
    aria_count=$(grep -r "aria-" ui/src/ --include="*.tsx" --include="*.ts" | wc -l)
    echo "✅" "Found $aria_count ARIA attributes (good for accessibility)"
else
    echo "⚠️ " "No ARIA attributes found (consider adding for accessibility)"
fi

# Test 3: UI package.json configuration
echo "Test 3: Validating UI package configuration..."
if [ -f "ui/package.json" ]; then
    echo "✅" "UI package.json exists"

    # Check for test scripts
    if grep -q "\"test\":" ui/package.json; then
        echo "✅" "Test script defined in package.json"
    else
        echo "⚠️ " "No test script in package.json (consider adding UI tests)"
    fi

    # Check for dev dependencies
    if grep -q "\"vite\":" ui/package.json; then
        echo "✅" "Vite build tool configured"
    fi

    if grep -q "\"react\":" ui/package.json; then
        echo "✅" "React framework configured"
    fi

    if grep -q "\"typescript\":" ui/package.json; then
        echo "✅" "TypeScript configured"
    fi
else
    echo "❌" "UI package.json not found"
    exit 1
fi

# Test 4: UI health endpoint
echo "Test 4: Validating UI health endpoint..."
if [ -f "ui/src/main.tsx" ] || [ -f "ui/src/App.tsx" ]; then
    echo "✅" "UI entry point exists"
else
    echo "⚠️ " "UI entry point not in standard location"
fi

# Check for health endpoint in UI server config
if grep -r "/health" ui/ --include="*.ts" --include="*.tsx" >/dev/null 2>&1; then
    echo "✅" "Health endpoint configured in UI"
else
    echo "⚠️ " "No health endpoint found (required for lifecycle monitoring)"
fi

# Test 5: UI loading states
echo "Test 5: Checking for loading state management..."
if grep -ri "loading\|isLoading\|skeleton" ui/src/ --include="*.tsx" --include="*.ts" >/dev/null 2>&1; then
    loading_count=$(grep -ri "loading\|isLoading\|skeleton" ui/src/ --include="*.tsx" --include="*.ts" | wc -l)
    echo "✅" "Found $loading_count loading state indicators"
else
    echo "⚠️ " "No loading state indicators found (consider adding for better UX)"
fi

# Test 6: Error handling in UI
echo "Test 6: Checking for error handling..."
if grep -ri "error\|catch\|Error" ui/src/ --include="*.tsx" --include="*.ts" >/dev/null 2>&1; then
    error_count=$(grep -ri "error\|catch\|Error" ui/src/ --include="*.tsx" --include="*.ts" | wc -l)
    echo "✅" "Found $error_count error handling references"
else
    echo "⚠️ " "No error handling found (consider adding error boundaries)"
fi

# Test 7: UI build configuration
echo "Test 7: Validating UI build configuration..."
if [ -f "ui/vite.config.ts" ] || [ -f "ui/vite.config.js" ]; then
    echo "✅" "Vite config exists"

    # Check for health endpoint proxy config
    if grep -q "proxy" ui/vite.config.* 2>/dev/null; then
        echo "✅" "Proxy configuration found"
    fi
else
    echo "⚠️ " "No Vite config file found"
fi

# Test 8: UI TypeScript configuration
echo "Test 8: Validating TypeScript configuration..."
if [ -f "ui/tsconfig.json" ]; then
    echo "✅" "TypeScript config exists"

    # Check for strict mode
    if grep -q "\"strict\":" ui/tsconfig.json; then
        echo "✅" "TypeScript strict mode configured"
    else
        echo "⚠️ " "Consider enabling TypeScript strict mode"
    fi
else
    echo "⚠️ " "No TypeScript config found"
fi

# Test 9: UI API integration
echo "Test 9: Checking UI-API integration..."
if grep -r "fetch\|axios\|api" ui/src/ --include="*.tsx" --include="*.ts" | grep -v "node_modules" >/dev/null 2>&1; then
    api_calls=$(grep -r "fetch\|axios" ui/src/ --include="*.tsx" --include="*.ts" | wc -l)
    echo "✅" "Found $api_calls API integration points"
else
    echo "⚠️ " "No API integration found in UI"
fi

# Test 10: UI routing
echo "Test 10: Validating UI routing..."
if grep -r "react-router\|Route\|BrowserRouter" ui/src/ --include="*.tsx" --include="*.ts" >/dev/null 2>&1; then
    echo "✅" "React Router configured"
else
    echo "⚠️ " "No routing library detected (may be single-page app)"
fi

# Test 11: UI environment configuration
echo "Test 11: Checking environment configuration..."
if [ -f "ui/.env.example" ] || [ -f "ui/.env" ]; then
    echo "✅" "Environment configuration present"
else
    echo "ℹ️ " "No environment file (may use defaults)"
fi

# Test 12: Component organization
echo "Test 12: Validating component organization..."
if [ -d "ui/src/components" ]; then
    component_count=$(find ui/src/components -name "*.tsx" -o -name "*.ts" | wc -l)
    echo "ℹ️ " "Found $component_count component files"

    if [ $component_count -gt 0 ]; then
        echo "✅" "Components organized in dedicated directory"
    fi
fi

testing::phase::end_with_summary "UI practices validation completed"
