#!/usr/bin/env bats

# Setup and teardown
setup() {
    # Set up test environment - use lifecycle-allocated port
    export API_PORT="${API_PORT:-16917}"
    export PALETTE_GEN_API_URL="http://localhost:${API_PORT}"
    CLI_PATH="${BATS_TEST_DIRNAME}/../../cli/palette-gen"

    # Ensure CLI is executable
    chmod +x "$CLI_PATH"

    # Check if API is available
    if curl -sf "${PALETTE_GEN_API_URL}/health" > /dev/null 2>&1; then
        export API_AVAILABLE="true"
    else
        export API_AVAILABLE="false"
    fi
}

teardown() {
    # Clean up any temporary files
    rm -f /tmp/palette-gen-test-*
}

# Help command tests
@test "CLI shows help with --help flag" {
    run "$CLI_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Palette Generator CLI" ]]
    [[ "$output" =~ "Commands:" ]]
}

@test "CLI shows help with help command" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Palette Generator CLI" ]]
}

@test "CLI shows help for unknown command" {
    run "$CLI_PATH" unknown-command
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Unknown command" ]]
    [[ "$output" =~ "Palette Generator CLI" ]]
}

# Generate command tests
@test "CLI generate palette with default options" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" generate "ocean"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success" ]]
    [[ "$output" =~ "palette" ]]
}

@test "CLI generate palette with style option" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" generate "sunset" --style vibrant
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success" ]]
}

@test "CLI generate palette with colors option" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" generate "forest" --colors 7
    [ "$status" -eq 0 ]
    [[ "$output" =~ "palette" ]]
}

@test "CLI generate palette with base color" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" generate "tech" --base "#3B82F6"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success" ]]
}

# Suggest command tests
@test "CLI suggest palettes for use case" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" suggest "e-commerce"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "suggestions" ]]
}

# Export command tests
@test "CLI export palette to CSS" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" export css --palette "#FF6B6B,#4ECDC4,#45B7D1"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ":root" ]] || [[ "$output" =~ "--color" ]]
}

@test "CLI export palette to JSON" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" export json --palette "#FF6B6B,#4ECDC4,#45B7D1"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "#FF6B6B" ]]
}

@test "CLI export palette to SCSS" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" export scss --palette "#FF6B6B,#4ECDC4,#45B7D1"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "$color" ]]
}

@test "CLI export fails without palette" {
    run "$CLI_PATH" export css
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error" ]]
}

# Accessibility check tests
@test "CLI check accessibility with valid colors" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" check "#000000" "#FFFFFF"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "contrast_ratio" ]]
}

@test "CLI check accessibility fails with missing colors" {
    run "$CLI_PATH" check "#000000"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error" ]]
}

# Harmony check tests
@test "CLI check harmony with valid colors" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" harmony "#FF6B6B,#4ECDC4,#45B7D1"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "harmony" ]] || [[ "$output" =~ "score" ]]
}

@test "CLI check harmony fails without colors" {
    run "$CLI_PATH" harmony
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error" ]]
}

# Colorblind simulation tests
@test "CLI simulate protanopia colorblindness" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" colorblind "#FF6B6B,#4ECDC4" protanopia
    [ "$status" -eq 0 ]
    [[ "$output" =~ "simulated" ]]
}

@test "CLI simulate deuteranopia colorblindness" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" colorblind "#FF6B6B,#4ECDC4" deuteranopia
    [ "$status" -eq 0 ]
    [[ "$output" =~ "simulated" ]]
}

@test "CLI simulate tritanopia colorblindness" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" colorblind "#FF6B6B,#4ECDC4" tritanopia
    [ "$status" -eq 0 ]
    [[ "$output" =~ "simulated" ]]
}

@test "CLI simulate colorblindness with default type" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" colorblind "#FF6B6B,#4ECDC4"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "protanopia" ]] # Default type
}

@test "CLI colorblind fails without colors" {
    run "$CLI_PATH" colorblind
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Error" ]]
}

# History tests
@test "CLI view history" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" history
    [ "$status" -eq 0 ]
    [[ "$output" =~ "history" ]] || [[ "$output" =~ "success" ]]
}

# Integration tests (require running server)
@test "CLI full workflow: generate, export, check" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server - integration test"

    # Generate palette
    run "$CLI_PATH" generate "ocean" --style vibrant --colors 5
    [ "$status" -eq 0 ]

    # Export to CSS
    run "$CLI_PATH" export css --palette "#2272C3,#B32AED,#E46773,#C0D411,#3BDC6C"
    [ "$status" -eq 0 ]

    # Check accessibility
    run "$CLI_PATH" check "#2272C3" "#FFFFFF"
    [ "$status" -eq 0 ]
}

# Edge cases
@test "CLI handles empty theme gracefully" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" generate ""
    # Should either succeed with default or return error
    [[ "$status" -eq 0 ]] || [[ "$output" =~ "error" ]]
}

@test "CLI handles special characters in theme" {
    [[ "$API_AVAILABLE" != "true" ]] && skip "Requires running API server"
    run "$CLI_PATH" generate "ocean & sunset"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "success" ]]
}
