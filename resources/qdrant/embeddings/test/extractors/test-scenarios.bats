#!/usr/bin/env bats
# Test scenarios extractor independently

load "../helpers/setup.bash"

setup() {
    # Set up test environment
    TEST_DIR="$BATS_TEST_TMPDIR/scenarios_test"
    mkdir -p "$TEST_DIR"
    
    # Source the extractor
    source "$EMBEDDING_ROOT/extractors/scenarios.sh"
    
    # Create test fixtures
    setup_scenario_fixtures
}

teardown() {
    rm -rf "$TEST_DIR"
}

setup_scenario_fixtures() {
    # Copy scenario test fixtures
    mkdir -p "$TEST_DIR/scenarios/test-scenario-app"
    cp "$FIXTURE_ROOT/scenarios/test-scenario-app/PRD.md" "$TEST_DIR/scenarios/test-scenario-app/"
    cp "$FIXTURE_ROOT/scenarios/test-scenario-app/.scenario.yaml" "$TEST_DIR/scenarios/test-scenario-app/"
}

@test "scenarios extractor finds PRD files" {
    cd "$TEST_DIR"
    
    run qdrant::extract::find_scenarios "."
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"PRD.md"* ]]
}

@test "scenarios extractor finds scenario config files" {
    cd "$TEST_DIR"
    
    run qdrant::extract::scenarios_batch "." "$TEST_DIR/scenarios_output.txt"
    
    [ "$status" -eq 0 ]
    [ -f "$TEST_DIR/scenarios_output.txt" ]
}

@test "scenarios extractor handles empty directory" {
    empty_dir="$TEST_DIR/empty"
    mkdir -p "$empty_dir"
    cd "$empty_dir"
    
    run qdrant::extract::find_scenarios "."
    
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "scenarios extractor processes PRD correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::prd "scenarios/test-scenario-app/PRD.md"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"PRD.md"* ]]
}

@test "scenarios extractor processes scenario config correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::scenario_config "scenarios/test-scenario-app/.scenario.yaml"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"test-email-assistant"* ]]
    [[ "$output" == *"productivity"* ]]
    [[ "$output" == *"Smart Email Categorization"* ]]
    [[ "$output" == *"React"* ]]
    [[ "$output" == *"freemium"* ]]
}

@test "scenarios extractor handles missing PRD file" {
    cd "$TEST_DIR"
    
    run qdrant::extract::prd "nonexistent/PRD.md"
    
    [ "$status" -ne 0 ]
}

@test "scenarios extractor handles invalid YAML" {
    cd "$TEST_DIR"
    
    # Create invalid YAML file
    mkdir -p "scenarios/invalid-app"
    echo "invalid: yaml: content: [" > "scenarios/invalid-app/.scenario.yaml"
    
    run qdrant::extract::scenario_config "scenarios/invalid-app/.scenario.yaml"
    
    [ "$status" -ne 0 ]
}

@test "scenarios extractor processes batch correctly" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/scenario_output.txt"
    
    run qdrant::extract::scenarios_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Check output contains expected content
    grep -q "Test Email Assistant" "$output_file"
    grep -q "test-email-assistant" "$output_file"
    grep -q -- "---SEPARATOR---" "$output_file"
}

@test "scenarios extractor extracts executive summary" {
    cd "$TEST_DIR"
    
    run qdrant::extract::prd "scenarios/test-scenario-app/PRD.md"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"lightweight email management assistant"* ]]
    [[ "$output" == *"AI-powered categorization"* ]]
}

@test "scenarios extractor extracts technical stack" {
    cd "$TEST_DIR"
    
    run qdrant::extract::scenario_config "scenarios/test-scenario-app/.scenario.yaml"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"React"* ]]
    [[ "$output" == *"TypeScript"* ]]
    [[ "$output" == *"Node.js"* ]]
    [[ "$output" == *"PostgreSQL"* ]]
}

@test "scenarios extractor extracts business model" {
    cd "$TEST_DIR"
    
    run qdrant::extract::prd "scenarios/test-scenario-app/PRD.md"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Revenue Streams"* ]]
    [[ "$output" == *"Freemium"* ]]
    [[ "$output" == *"\$12/month"* ]]
}

@test "scenarios extractor extracts success metrics" {
    cd "$TEST_DIR"
    
    run qdrant::extract::scenario_config "scenarios/test-scenario-app/.scenario.yaml"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"70%"* ]]  # daily_active_users
    [[ "$output" == *"50K"* ]]  # arr_target
    [[ "$output" == *"99.9%"* ]] # uptime
}

@test "scenarios extractor handles mixed content correctly" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/mixed_output.txt"
    run qdrant::extract::scenarios_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    
    # Should contain both PRD and config content
    grep -q "Product Requirements Document" "$output_file"
    grep -q "technical_stack" "$output_file"
}

@test "scenarios extractor output format is consistent" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/format_test.txt"
    run qdrant::extract::scenarios_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    
    # Check format consistency
    local separator_count=$(grep -c -- "---SEPARATOR---" "$output_file")
    local prd_count=$(find . -name "PRD.md" -path "*/scenarios/*" | wc -l)
    local config_count=$(find . -name ".scenario.yaml" -path "*/scenarios/*" | wc -l)
    local total_expected=$((prd_count + config_count))
    
    # Should have separators between items
    [ "$separator_count" -eq "$total_expected" ] || [ "$separator_count" -eq $((total_expected - 1)) ]
}