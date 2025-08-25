#!/usr/bin/env bats

# KiCad Integration Tests

setup() {
    # Get the directory containing this test file
    TEST_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}" && builtin pwd)"
    CLI_PATH="${TEST_DIR}/../cli.sh"
    
    # Source test fixtures path
    FIXTURES_DIR="${TEST_DIR}/../../../../__test/fixtures/data"
    
    # Create test data directory
    TEST_DATA_DIR="/tmp/kicad_test_$$"
    mkdir -p "$TEST_DATA_DIR"
    
    # Export for use in tests
    export TEST_DATA_DIR
    export KICAD_PROJECTS_DIR="$TEST_DATA_DIR/projects"
    export KICAD_LIBRARIES_DIR="$TEST_DATA_DIR/libraries"
    export KICAD_DATA_DIR="$TEST_DATA_DIR"
}

teardown() {
    # Clean up test data
    if [[ -d "$TEST_DATA_DIR" ]]; then
        rm -rf "$TEST_DATA_DIR"
    fi
}

@test "KiCad: CLI exists and is executable" {
    [ -f "$CLI_PATH" ]
    [ -x "$CLI_PATH" ]
}

@test "KiCad: Help command works" {
    run "$CLI_PATH" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "KiCad CLI" ]]
    [[ "$output" =~ "Electronic Design Automation" ]]
}

@test "KiCad: Status command returns valid response" {
    run "$CLI_PATH" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "KiCad Status Report" ]]
}

@test "KiCad: Status command returns valid JSON" {
    run "$CLI_PATH" status --format json
    [ "$status" -eq 0 ]
    # Validate JSON structure
    echo "$output" | jq -e '.name == "kicad"' >/dev/null
    echo "$output" | jq -e '.category == "execution"' >/dev/null
}

@test "KiCad: Can inject PCB file" {
    # Create a mock PCB file
    local test_pcb="$TEST_DATA_DIR/test.kicad_pcb"
    echo "(kicad_pcb (version 20211014) (generator pcbnew))" > "$test_pcb"
    
    run "$CLI_PATH" inject "$test_pcb"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Project injected" ]]
    
    # Verify file was copied
    [ -f "$KICAD_PROJECTS_DIR/test/test.kicad_pcb" ]
}

@test "KiCad: Can inject schematic file" {
    # Create a mock schematic file
    local test_sch="$TEST_DATA_DIR/test.kicad_sch"
    echo "(kicad_sch (version 20211123) (generator eeschema))" > "$test_sch"
    
    run "$CLI_PATH" inject "$test_sch"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Project injected" ]]
    
    # Verify file was copied
    [ -f "$KICAD_PROJECTS_DIR/test/test.kicad_sch" ]
}

@test "KiCad: Can inject symbol library" {
    # Create a mock symbol library file
    local test_sym="$TEST_DATA_DIR/custom.kicad_sym"
    echo "(kicad_symbol_lib (version 20211014) (generator kicad_symbol_editor))" > "$test_sym"
    
    run "$CLI_PATH" inject "$test_sym" library
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Library injected" ]]
    
    # Verify file was copied
    [ -f "$KICAD_LIBRARIES_DIR/custom.kicad_sym" ]
}

@test "KiCad: Can list projects" {
    # Create test projects
    mkdir -p "$KICAD_PROJECTS_DIR/project1"
    mkdir -p "$KICAD_PROJECTS_DIR/project2"
    touch "$KICAD_PROJECTS_DIR/project1/project1.kicad_pro"
    touch "$KICAD_PROJECTS_DIR/project2/project2.kicad_pro"
    
    run "$CLI_PATH" list-projects
    [ "$status" -eq 0 ]
    [[ "$output" =~ "project1" ]]
    [[ "$output" =~ "project2" ]]
}

@test "KiCad: Can list libraries" {
    # Create test libraries
    mkdir -p "$KICAD_LIBRARIES_DIR"
    touch "$KICAD_LIBRARIES_DIR/resistors.kicad_sym"
    touch "$KICAD_LIBRARIES_DIR/capacitors.kicad_sym"
    
    run "$CLI_PATH" list-libraries
    [ "$status" -eq 0 ]
    [[ "$output" =~ "resistors.kicad_sym" ]]
    [[ "$output" =~ "capacitors.kicad_sym" ]]
}

@test "KiCad: Examples command works" {
    run "$CLI_PATH" examples
    [ "$status" -eq 0 ]
    [[ "$output" =~ "KiCad Examples" ]]
    [[ "$output" =~ "Project Management" ]]
    [[ "$output" =~ "Library Management" ]]
}

@test "KiCad: Start command provides appropriate message" {
    run "$CLI_PATH" start
    [ "$status" -eq 0 ]
    [[ "$output" =~ "desktop application" ]]
}

@test "KiCad: Stop command provides appropriate message" {
    run "$CLI_PATH" stop
    [ "$status" -eq 0 ]
    [[ "$output" =~ "desktop application" ]]
}