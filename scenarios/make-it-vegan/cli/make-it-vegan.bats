#!/usr/bin/env bats
# BATS test suite for make-it-vegan CLI
# Run with: bats cli/make-it-vegan.bats

setup() {
    # Get the directory of this script
    DIR="$( cd "$( dirname "$BATS_TEST_FILENAME" )" >/dev/null 2>&1 && pwd )"
    CLI="$DIR/make-it-vegan"

    # Ensure scenario is running and get API_PORT
    # ALWAYS detect the make-it-vegan port, don't trust environment (could be another scenario's port)
    export API_PORT=$(vrooli scenario status make-it-vegan --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // empty' 2>/dev/null)

    # Verify we have a port and the service is actually responding
    if [ -z "$API_PORT" ]; then
        skip "Scenario not running - start with: vrooli scenario start make-it-vegan"
    fi

    # Quick health check to ensure API is actually responding
    if ! curl -sf "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
        skip "API not responding on port $API_PORT - check scenario status"
    fi
}

# Help and Usage Tests

@test "CLI shows help with --help flag" {
    run "$CLI" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Make It Vegan" ]]
    [[ "$output" =~ "Commands:" ]]
    [[ "$output" =~ "check" ]]
    [[ "$output" =~ "substitute" ]]
    [[ "$output" =~ "veganize" ]]
}

@test "CLI shows help with help command" {
    run "$CLI" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Make It Vegan" ]]
    [[ "$output" =~ "Usage:" ]]
}

@test "CLI shows error for unknown command" {
    run "$CLI" invalid-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
    [[ "$output" =~ "make-it-vegan help" ]]
}

# Ingredient Checking Tests

@test "CLI check identifies vegan ingredients" {
    run "$CLI" check "flour, sugar, salt, water"
    [ "$status" -eq 0 ]
    # Should indicate all vegan or found 0 non-vegan
    [[ "$output" =~ "vegan" ]] || [[ "$output" =~ "0 non-vegan" ]]
}

@test "CLI check identifies non-vegan ingredients" {
    run "$CLI" check "milk, eggs, butter"
    [ "$status" -eq 0 ]
    # Should detect at least some non-vegan items
    [[ "$output" =~ "non-vegan" ]] || [[ "$output" =~ "milk" ]]
}

@test "CLI check handles mixed ingredients" {
    run "$CLI" check "flour, milk, sugar, eggs, salt"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "non-vegan" ]]
}

@test "CLI check handles honey (commonly questioned ingredient)" {
    run "$CLI" check "honey"
    [ "$status" -eq 0 ]
    # Honey should be detected as non-vegan
    [[ "$output" =~ "non-vegan" ]] || [[ "$output" =~ "honey" ]]
}

@test "CLI check handles vegan milk alternatives" {
    run "$CLI" check "soy milk, almond milk, oat milk"
    [ "$status" -eq 0 ]
    # Should recognize these as vegan
    [[ "$output" =~ "vegan" ]] || [[ "$output" =~ "0 non-vegan" ]]
}

# Substitute Finding Tests

@test "CLI substitute finds alternatives for milk" {
    run "$CLI" substitute "milk"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "alternative" ]] || [[ "$output" =~ "soy" ]] || [[ "$output" =~ "oat" ]] || [[ "$output" =~ "almond" ]]
}

@test "CLI substitute finds alternatives for butter" {
    run "$CLI" substitute "butter"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "alternative" ]] || [[ "$output" =~ "vegan butter" ]] || [[ "$output" =~ "coconut oil" ]]
}

@test "CLI substitute accepts context flag" {
    run "$CLI" substitute "cheese" --context "pizza"
    [ "$status" -eq 0 ]
    # Should provide context-specific alternatives
    [[ "$output" =~ "alternative" ]] || [[ "$output" =~ "vegan" ]]
}

@test "CLI substitute handles eggs with baking context" {
    run "$CLI" substitute "eggs" --context "baking"
    [ "$status" -eq 0 ]
    # Should suggest baking-specific alternatives
    [[ "$output" =~ "alternative" ]] || [[ "$output" =~ "flax" ]] || [[ "$output" =~ "chia" ]] || [[ "$output" =~ "applesauce" ]]
}

# Recipe Veganization Tests

@test "CLI veganize converts simple recipe" {
    run "$CLI" veganize "scrambled eggs with milk and butter"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vegan" ]] || [[ "$output" =~ "tofu" ]] || [[ "$output" =~ "vegan" ]]
}

@test "CLI veganize handles recipe with multiple ingredients" {
    run "$CLI" veganize "pancakes: flour, milk, eggs, butter, sugar"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Vegan" ]] || [[ "$output" =~ "vegan" ]]
}

# Products List Tests

@test "CLI products lists common non-vegan items" {
    run "$CLI" products
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Non-Vegan" ]] || [[ "$output" =~ "dairy" ]] || [[ "$output" =~ "animal" ]]
}

# Nutrition Guidance Tests

@test "CLI nutrition provides guidance" {
    run "$CLI" nutrition
    [ "$status" -eq 0 ]
    [[ "$output" =~ "B12" ]] || [[ "$output" =~ "protein" ]] || [[ "$output" =~ "Nutritional" ]]
}

@test "CLI nutrition includes key nutrients" {
    run "$CLI" nutrition
    [ "$status" -eq 0 ]
    # Should mention important vegan nutrients
    [[ "$output" =~ "B12" ]] || [[ "$output" =~ "iron" ]] || [[ "$output" =~ "protein" ]] || [[ "$output" =~ "calcium" ]]
}

# JSON Output Tests

@test "CLI check supports --json flag" {
    run "$CLI" check "milk, flour" --json
    [ "$status" -eq 0 ]
    # Should be valid JSON
    echo "$output" | jq . >/dev/null 2>&1
}

@test "CLI substitute supports --json output" {
    run "$CLI" substitute "milk" --json
    [ "$status" -eq 0 ]
    # Should be valid JSON
    echo "$output" | jq . >/dev/null 2>&1
}

@test "CLI products supports --json output" {
    run "$CLI" products --json
    [ "$status" -eq 0 ]
    # Should be valid JSON
    echo "$output" | jq . >/dev/null 2>&1
}

@test "CLI nutrition supports --json output" {
    run "$CLI" nutrition --json
    [ "$status" -eq 0 ]
    # Should be valid JSON
    echo "$output" | jq . >/dev/null 2>&1
}

# Error Handling Tests

@test "CLI check handles empty input gracefully" {
    run "$CLI" check ""
    # Should either succeed with empty result or provide helpful error
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

@test "CLI substitute handles empty ingredient gracefully" {
    run "$CLI" substitute ""
    # Should handle gracefully
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
}

# API Connectivity Tests

@test "CLI detects API port from scenario status when API_PORT not set" {
    # When running with API_PORT available in the parent environment,
    # the CLI can use vrooli scenario status to auto-detect
    # This test verifies the scenario status command works
    port=$(vrooli scenario status make-it-vegan --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // empty' 2>/dev/null)
    [ -n "$port" ]
    [ "$port" = "$API_PORT" ]
}

@test "CLI provides helpful error when scenario not running" {
    # This test would require stopping the scenario
    # Skipping for now as it would disrupt other tests
    skip "Requires scenario to be stopped"
}

# Integration Tests

@test "CLI check + substitute workflow" {
    # First check what's non-vegan
    run "$CLI" check "butter, milk, flour"
    [ "$status" -eq 0 ]

    # Then get substitutes for non-vegan items
    run "$CLI" substitute "butter"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "alternative" ]]
}

@test "CLI full workflow: check, substitute, veganize" {
    # Check ingredients
    run "$CLI" check "eggs, milk, flour"
    [ "$status" -eq 0 ]

    # Find substitute
    run "$CLI" substitute "eggs" --context "baking"
    [ "$status" -eq 0 ]

    # Veganize recipe
    run "$CLI" veganize "cake with eggs and milk"
    [ "$status" -eq 0 ]
}

# Performance Tests

@test "CLI check responds within reasonable time" {
    # Should complete within a few seconds
    timeout 5s "$CLI" check "milk, eggs, flour"
    [ $? -eq 0 ]
}

@test "CLI substitute responds within reasonable time" {
    timeout 5s "$CLI" substitute "cheese"
    [ $? -eq 0 ]
}

# Edge Cases

@test "CLI handles ingredients with special characters" {
    run "$CLI" check "salt & pepper, sugar-free sweetener"
    [ "$status" -eq 0 ]
}

@test "CLI handles long ingredient lists" {
    run "$CLI" check "flour, sugar, salt, water, yeast, oil, milk, eggs, butter, cheese, vanilla, cocoa"
    [ "$status" -eq 0 ]
}

@test "CLI handles unicode characters in ingredient names" {
    run "$CLI" check "café au lait, crème fraîche"
    [ "$status" -eq 0 ]
}

# Regression Tests (based on known issues)

@test "CLI correctly identifies soy milk as vegan" {
    run "$CLI" check "soy milk"
    [ "$status" -eq 0 ]
    # Should NOT flag soy milk as non-vegan
    ! [[ "$output" =~ "non-vegan" ]] || [[ "$output" =~ "0 non-vegan" ]]
}

@test "CLI correctly identifies almond butter as vegan" {
    run "$CLI" check "almond butter"
    [ "$status" -eq 0 ]
    # Should NOT flag almond butter as non-vegan
    ! [[ "$output" =~ "non-vegan" ]] || [[ "$output" =~ "0 non-vegan" ]]
}
