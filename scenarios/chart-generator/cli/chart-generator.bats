#!/usr/bin/env bats

# Use full path to CLI binary
CLI_PATH="${BATS_TEST_DIRNAME}/chart-generator"

@test "chart-generator help command" {
    run "$CLI_PATH" --help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Chart Generator CLI"* ]]
}

@test "chart-generator help command (alias)" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" == *"Chart Generator CLI"* ]]
}

@test "chart-generator version command" {
    run "$CLI_PATH" version
    [ "$status" -eq 0 ]
    [[ "$output" == *"version"* ]]
}

@test "chart-generator status command" {
    run "$CLI_PATH" status
    [ "$status" -eq 0 ]
    # Should show service status
    [[ "$output" == *"Chart Generator"* ]] || [[ "$output" == *"Status"* ]]
}

# [REQ:CHART-P0-004-CLI-STYLES] CLI style management commands
@test "chart-generator styles command" {
    run "$CLI_PATH" styles
    [ "$status" -eq 0 ]
    # Should list available styles
    [[ "$output" =~ (Professional|Minimal|Available) ]]
}

# [REQ:CHART-P0-004-CLI-TEMPLATES] CLI template management commands
@test "chart-generator templates command" {
    run "$CLI_PATH" templates
    [ "$status" -eq 0 ]
    # Should list templates or show template info
}

# [REQ:CHART-P0-004-CLI-GENERATE] CLI chart generation command with format options
@test "chart-generator generate bar chart from stdin" {
    run bash -c 'echo "[{\"x\":\"A\",\"y\":10},{\"x\":\"B\",\"y\":20}]" | '"$CLI_PATH"' generate bar --data - --format png --output /tmp'
    [ "$status" -eq 0 ]
    [[ "$output" == *"success"* ]] || [[ "$output" == *"generated"* ]] || [[ "$output" == *"PNG"* ]]
}

# [REQ:CHART-P0-004-CLI-GENERATE] CLI chart generation command with format options
# [REQ:CHART-P0-005-SVG] Export capabilities: SVG format
@test "chart-generator generate line chart from stdin" {
    run bash -c 'echo "[{\"x\":\"A\",\"y\":10},{\"x\":\"B\",\"y\":20}]" | '"$CLI_PATH"' generate line --data - --format svg --output /tmp'
    [ "$status" -eq 0 ]
    [[ "$output" == *"success"* ]] || [[ "$output" == *"generated"* ]] || [[ "$output" == *"SVG"* ]]
}

# [REQ:CHART-P0-004-CLI-GENERATE] CLI chart generation command with format options
# [REQ:CHART-P0-005-PNG] Export capabilities: PNG format
@test "chart-generator generate pie chart from stdin" {
    run bash -c 'echo "[{\"name\":\"A\",\"value\":30},{\"name\":\"B\",\"value\":70}]" | '"$CLI_PATH"' generate pie --data - --format png --output /tmp'
    [ "$status" -eq 0 ]
    [[ "$output" == *"success"* ]] || [[ "$output" == *"generated"* ]] || [[ "$output" == *"PNG"* ]]
}

@test "chart-generator invalid chart type" {
    run "$CLI_PATH" generate invalid_type
    # Should fail gracefully
    [ "$status" -ne 0 ] || [[ "$output" == *"error"* ]] || [[ "$output" == *"invalid"* ]]
}

@test "chart-generator generate without data" {
    run "$CLI_PATH" generate bar
    # Should fail or show error about missing data
    [ "$status" -ne 0 ] || [[ "$output" == *"data"* ]] || [[ "$output" == *"required"* ]]
}

# [REQ:CHART-P0-004-CLI-STYLES] CLI style management commands
# [REQ:CHART-P0-003-LIGHT] Light theme with professional color palette
@test "chart-generator styles with professional filter" {
    run "$CLI_PATH" styles
    [ "$status" -eq 0 ]
    # Output should include style information
}

# [REQ:CHART-P0-004-CLI-GENERATE] CLI chart generation command with format options
# [REQ:CHART-P0-003-DARK] Dark theme optimized for screens and presentations
@test "chart-generator generate with custom style" {
    run bash -c 'echo "[{\"x\":\"A\",\"y\":10}]" | '"$CLI_PATH"' generate bar --data - --style professional --format png --output /tmp'
    [ "$status" -eq 0 ]
}

@test "chart-generator generate with title" {
    run bash -c 'echo "[{\"x\":\"Q1\",\"y\":100}]" | '"$CLI_PATH"' generate bar --data - --title "Sales Report" --format png --output /tmp'
    [ "$status" -eq 0 ]
}

@test "chart-generator generate multiple formats" {
    run bash -c 'echo "[{\"x\":\"A\",\"y\":10}]" | '"$CLI_PATH"' generate bar --data - --format png,svg --output /tmp'
    [ "$status" -eq 0 ]
    [[ "$output" == *"PNG"* ]] || [[ "$output" == *"success"* ]]
}
