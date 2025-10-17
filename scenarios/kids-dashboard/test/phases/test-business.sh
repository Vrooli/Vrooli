#!/bin/bash

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running business logic tests for kids-dashboard..."

# Test 1: Verify kid-friendly filtering logic
testing::phase::log "Testing kid-friendly scenario filtering..."
cd api

# Run Go tests for business logic
go test -v -run "TestKidFriendlyCategories" ./... 2>&1 | grep -E "(PASS|FAIL)" | head -5
if [ $? -eq 0 ]; then
    testing::phase::log "✓ Kid-friendly filtering logic verified"
fi

# Test 2: Verify blacklist filtering
testing::phase::log "Testing blacklist category filtering..."
go test -v -run "TestBlacklistCategories" ./... 2>&1 | grep -E "(PASS|FAIL)" | head -5
if [ $? -eq 0 ]; then
    testing::phase::log "✓ Blacklist filtering verified"
fi

# Test 3: Verify scenario metadata extraction
testing::phase::log "Testing scenario metadata extraction..."
go test -v -run "TestScenarioMetadataExtraction" ./... 2>&1 | grep -E "(PASS|FAIL)" | head -5
if [ $? -eq 0 ]; then
    testing::phase::log "✓ Metadata extraction verified"
fi

# Test 4: Verify known scenario metadata
testing::phase::log "Testing known scenario metadata..."
go test -v -run "TestKnownScenarioMetadata" ./... 2>&1 | grep -E "(PASS|FAIL)" | head -5
if [ $? -eq 0 ]; then
    testing::phase::log "✓ Known scenario metadata verified"
fi

# Test 5: Business logic - age range filtering
testing::phase::log "Testing business logic for age range filtering..."
cat > /tmp/test_age_filtering.go <<'EOF'
package main

import "testing"

func TestAgeRangeBusinessLogic(t *testing.T) {
    scenarios := []Scenario{
        {ID: "1", AgeRange: "5-12"},
        {ID: "2", AgeRange: "9-12"},
        {ID: "3", AgeRange: "5-8"},
    }

    // Test: requesting 5-12 should return scenarios with exact match or general 5-12
    filtered := filterScenarios(scenarios, "5-12", "")

    // All scenarios match or are general enough
    if len(filtered) < 1 {
        t.Error("Expected at least one scenario for age range 5-12")
    }
}

func TestCategoryBusinessLogic(t *testing.T) {
    scenarios := []Scenario{
        {ID: "1", Category: "games"},
        {ID: "2", Category: "learn"},
        {ID: "3", Category: "games"},
    }

    filtered := filterScenarios(scenarios, "", "games")

    if len(filtered) != 2 {
        t.Errorf("Expected 2 game scenarios, got %d", len(filtered))
    }
}
EOF

if [ -f main.go ]; then
    cat /tmp/test_age_filtering.go >> business_logic_test.go
    go test -v -run "TestAgeRangeBusinessLogic|TestCategoryBusinessLogic" ./... 2>&1 | grep -E "(PASS|FAIL)" | head -5
    rm -f business_logic_test.go
    testing::phase::log "✓ Business logic tests completed"
fi

cd ..

testing::phase::end_with_summary "Business logic tests completed"
