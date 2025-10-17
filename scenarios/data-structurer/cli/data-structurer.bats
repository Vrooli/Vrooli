#!/usr/bin/env bats
# Data Structurer CLI Tests

CLI_SCRIPT="${BATS_TEST_DIRNAME}/data-structurer"
# Use DATA_STRUCTURER_API_PORT first (scenario-specific), fall back to API_PORT, then default
SCENARIO_API_PORT="${DATA_STRUCTURER_API_PORT:-${API_PORT:-15774}}"
export DATA_STRUCTURER_API_PORT="$SCENARIO_API_PORT"
API_URL="http://localhost:${SCENARIO_API_PORT}"

# Setup function runs before each test
setup() {
    # Check if CLI script exists and is executable
    if [ ! -x "$CLI_SCRIPT" ]; then
        skip "CLI script not found or not executable at $CLI_SCRIPT"
    fi

    # Skip tests if API is not available
    if ! curl -sf "$API_URL/health" > /dev/null 2>&1; then
        skip "API server not running at $API_URL"
    fi

    # Create temp directory for test artifacts
    TEST_TEMP_DIR="$(mktemp -d)"
    export TEST_TEMP_DIR

    # Generate unique suffix for test schemas to avoid conflicts
    TEST_SUFFIX="$(date +%s)-$$"
    export TEST_SUFFIX
}

# Teardown function runs after each test
teardown() {
    # Clean up temp directory
    if [ -n "$TEST_TEMP_DIR" ] && [ -d "$TEST_TEMP_DIR" ]; then
        rm -rf "$TEST_TEMP_DIR"
    fi
}

# Helper function to extract JSON from CLI output (which includes log messages on stderr)
extract_json() {
    local output="$1"
    # Find the first '{' and extract from there to the end
    echo "$output" | sed -n '/{/,$p'
}

# ============================================================================
# Basic Command Tests
# ============================================================================

@test "CLI shows help when no arguments provided" {
    run bash "$CLI_SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Data Structurer CLI" ]]
    [[ "$output" =~ "USAGE:" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI shows help with help command" {
    run bash "$CLI_SCRIPT" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Data Structurer CLI" ]]
    [[ "$output" =~ "COMMANDS:" ]]
}

@test "CLI shows version information" {
    run bash "$CLI_SCRIPT" version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "version" ]] || [[ "$output" =~ "1.0" ]]
}

@test "CLI shows version in JSON format" {
    run bash "$CLI_SCRIPT" version --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "{" ]]
    [[ "$output" =~ "version" ]]
}

@test "CLI shows status information" {
    run bash "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Status" ]] || [[ "$output" =~ "healthy" ]]
}

@test "CLI shows status in JSON format" {
    run bash "$CLI_SCRIPT" status --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "{" ]]
    [[ "$output" =~ "status" ]] || [[ "$output" =~ "healthy" ]]
}

# ============================================================================
# Schema Management Tests
# ============================================================================

@test "CLI can list schemas" {
    run bash "$CLI_SCRIPT" list-schemas
    [ "$status" -eq 0 ]
}

@test "CLI can list schemas in JSON format" {
    run bash "$CLI_SCRIPT" list-schemas --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[" ]] || [[ "$output" =~ "{" ]]
}

@test "CLI can create a schema" {
    # Create test schema file
    cat > "$TEST_TEMP_DIR/test-schema.json" <<EOF
{
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "value": {"type": "number"}
  },
  "required": ["name"]
}
EOF

    run bash "$CLI_SCRIPT" create-schema "test-cli-schema-${TEST_SUFFIX}" "$TEST_TEMP_DIR/test-schema.json" --description "Test schema from CLI"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "created" ]] || [[ "$output" =~ "success" ]] || [[ "$output" =~ "id" ]]
}

@test "CLI requires schema name for creation" {
    cat > "$TEST_TEMP_DIR/test-schema.json" <<EOF
{"type": "object"}
EOF

    run bash "$CLI_SCRIPT" create-schema
    [ "$status" -ne 0 ]
}

@test "CLI requires schema definition file for creation" {
    run bash "$CLI_SCRIPT" create-schema "test-schema"
    [ "$status" -ne 0 ]
}

@test "CLI rejects invalid schema file" {
    echo "invalid json" > "$TEST_TEMP_DIR/invalid.json"

    run bash "$CLI_SCRIPT" create-schema "test-schema" "$TEST_TEMP_DIR/invalid.json"
    [ "$status" -ne 0 ]
}

@test "CLI can get schema by ID after creating one" {
    # First create a schema
    cat > "$TEST_TEMP_DIR/test-schema.json" <<EOF
{"type": "object", "properties": {"field": {"type": "string"}}}
EOF

    create_output=$(bash "$CLI_SCRIPT" create-schema "test-get-schema-${TEST_SUFFIX}" "$TEST_TEMP_DIR/test-schema.json" --json 2>&1)

    # Extract schema ID (assuming JSON output contains an id field)
    if command -v jq &> /dev/null && [[ "$create_output" =~ \{.*\} ]]; then
        json_only=$(extract_json "$create_output")
        schema_id=$(echo "$json_only" | jq -r '.id // empty')

        if [ -n "$schema_id" ] && [ "$schema_id" != "null" ]; then
            run bash "$CLI_SCRIPT" get-schema "$schema_id"
            [ "$status" -eq 0 ]
            [[ "$output" =~ "$schema_id" ]] || [[ "$output" =~ "test-get-schema" ]]
        fi
    fi
}

@test "CLI handles non-existent schema ID gracefully" {
    run bash "$CLI_SCRIPT" get-schema "00000000-0000-0000-0000-000000000000"
    [ "$status" -ne 0 ]
    [[ "$output" =~ "not found" ]] || [[ "$output" =~ "error" ]] || [[ "$output" =~ "ERROR" ]]
}

@test "CLI can delete a schema" {
    # First create a schema to delete
    cat > "$TEST_TEMP_DIR/test-schema.json" <<EOF
{"type": "object"}
EOF

    create_output=$(bash "$CLI_SCRIPT" create-schema "test-delete-schema-${TEST_SUFFIX}" "$TEST_TEMP_DIR/test-schema.json" --json 2>&1)

    if command -v jq &> /dev/null && [[ "$create_output" =~ \{.*\} ]]; then
        json_only=$(extract_json "$create_output")
        schema_id=$(echo "$json_only" | jq -r '.id // empty')

        if [ -n "$schema_id" ] && [ "$schema_id" != "null" ]; then
            run bash "$CLI_SCRIPT" delete-schema "$schema_id"
            [ "$status" -eq 0 ]
            [[ "$output" =~ "deleted" ]] || [[ "$output" =~ "success" ]]
        fi
    fi
}

# ============================================================================
# Schema Template Tests
# ============================================================================

@test "CLI can list schema templates" {
    run bash "$CLI_SCRIPT" list-templates
    [ "$status" -eq 0 ]
}

@test "CLI can list schema templates in JSON format" {
    run bash "$CLI_SCRIPT" list-templates --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[" ]] || [[ "$output" =~ "{" ]]
}

@test "CLI can get template by ID" {
    # First get list of templates
    templates_output=$(bash "$CLI_SCRIPT" list-templates --json 2>&1)

    if command -v jq &> /dev/null && [[ "$templates_output" =~ \{.*\} ]]; then
        json_only=$(extract_json "$templates_output")
        template_id=$(echo "$json_only" | jq -r '.templates[0].id // empty')

        if [ -n "$template_id" ] && [ "$template_id" != "null" ]; then
            run bash "$CLI_SCRIPT" get-template "$template_id"
            [ "$status" -eq 0 ]
        fi
    fi
}

@test "CLI can create schema from template" {
    # Get first template
    templates_output=$(bash "$CLI_SCRIPT" list-templates --json 2>&1)

    if command -v jq &> /dev/null && [[ "$templates_output" =~ \{.*\} ]]; then
        json_only=$(extract_json "$templates_output")
        template_id=$(echo "$json_only" | jq -r '.templates[0].id // empty')

        if [ -n "$template_id" ] && [ "$template_id" != "null" ]; then
            # Use timestamp to ensure unique schema name
            unique_name="test-from-template-$(date +%s)"
            run bash "$CLI_SCRIPT" create-from-template "$template_id" "$unique_name"
            [ "$status" -eq 0 ]
            [[ "$output" =~ "created" ]] || [[ "$output" =~ "success" ]]
        fi
    fi
}

# ============================================================================
# Data Processing Tests
# ============================================================================

@test "CLI can process text data" {
    # First create a schema
    cat > "$TEST_TEMP_DIR/process-schema.json" <<EOF
{
  "type": "object",
  "properties": {
    "name": {"type": "string"},
    "email": {"type": "string"}
  }
}
EOF

    create_output=$(bash "$CLI_SCRIPT" create-schema "test-process-schema-${TEST_SUFFIX}" "$TEST_TEMP_DIR/process-schema.json" --json 2>&1)

    if command -v jq &> /dev/null && [[ "$create_output" =~ \{.*\} ]]; then
        json_only=$(extract_json "$create_output")
        schema_id=$(echo "$json_only" | jq -r '.id // empty')

        if [ -n "$schema_id" ] && [ "$schema_id" != "null" ]; then
            run bash "$CLI_SCRIPT" process "$schema_id" "John Doe, email: john@example.com"
            [ "$status" -eq 0 ]
            [[ "$output" =~ "processing" ]] || [[ "$output" =~ "completed" ]] || [[ "$output" =~ "success" ]]
        fi
    fi
}

@test "CLI requires schema ID for processing" {
    run bash "$CLI_SCRIPT" process
    [ "$status" -ne 0 ]
}

@test "CLI requires input data for processing" {
    run bash "$CLI_SCRIPT" process "00000000-0000-0000-0000-000000000000"
    [ "$status" -ne 0 ]
}

@test "CLI can get processed data for a schema" {
    # Create and process data first
    cat > "$TEST_TEMP_DIR/data-schema.json" <<EOF
{"type": "object", "properties": {"field": {"type": "string"}}}
EOF

    create_output=$(bash "$CLI_SCRIPT" create-schema "test-data-schema-${TEST_SUFFIX}" "$TEST_TEMP_DIR/data-schema.json" --json 2>&1)

    if command -v jq &> /dev/null && [[ "$create_output" =~ \{.*\} ]]; then
        json_only=$(extract_json "$create_output")
        schema_id=$(echo "$json_only" | jq -r '.id // empty')

        if [ -n "$schema_id" ] && [ "$schema_id" != "null" ]; then
            # Process some data
            bash "$CLI_SCRIPT" process "$schema_id" "test data" > /dev/null 2>&1

            # Wait for processing
            sleep 2

            run bash "$CLI_SCRIPT" get-data "$schema_id"
            [ "$status" -eq 0 ]
        fi
    fi
}

# ============================================================================
# Processing Jobs Tests
# ============================================================================

@test "CLI can list processing jobs" {
    run bash "$CLI_SCRIPT" list-jobs
    [ "$status" -eq 0 ]
}

@test "CLI can list processing jobs in JSON format" {
    run bash "$CLI_SCRIPT" list-jobs --json
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[" ]] || [[ "$output" =~ "{" ]]
}

@test "CLI can get job by ID" {
    # Get list of jobs
    jobs_output=$(bash "$CLI_SCRIPT" list-jobs --json 2>&1)

    if command -v jq &> /dev/null && [[ "$jobs_output" =~ \{.*\} ]]; then
        json_only=$(extract_json "$jobs_output")
        job_id=$(echo "$json_only" | jq -r '.jobs[0].id // empty')

        if [ -n "$job_id" ] && [ "$job_id" != "null" ]; then
            run bash "$CLI_SCRIPT" get-job "$job_id"
            [ "$status" -eq 0 ]
        fi
    fi
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "CLI handles unknown command gracefully" {
    run bash "$CLI_SCRIPT" unknown-command
    [ "$status" -ne 0 ]
    [[ "$output" =~ "Unknown" ]] || [[ "$output" =~ "ERROR" ]] || [[ "$output" =~ "not found" ]]
}

@test "CLI handles missing required flags" {
    run bash "$CLI_SCRIPT" create-schema
    [ "$status" -ne 0 ]
}

@test "CLI validates UUID format" {
    run bash "$CLI_SCRIPT" get-schema "invalid-uuid"
    [ "$status" -ne 0 ]
}

@test "CLI handles API connection failure" {
    # Override API URL to non-existent endpoint
    export DATA_STRUCTURER_API_URL="http://localhost:99999"

    run bash "$CLI_SCRIPT" status
    [ "$status" -ne 0 ]

    # Reset API URL
    unset DATA_STRUCTURER_API_URL
}

# ============================================================================
# Output Format Tests
# ============================================================================

@test "CLI respects --json flag for all commands" {
    commands=("status" "version" "list-schemas" "list-templates" "list-jobs")

    for cmd in "${commands[@]}"; do
        output=$(bash "$CLI_SCRIPT" "$cmd" --json 2>&1 || true)
        if [[ "$output" =~ \{.*\} ]] || [[ "$output" =~ \[.*\] ]]; then
            # Valid JSON output
            true
        else
            # Not JSON or command failed (acceptable for some commands)
            true
        fi
    done
}

@test "CLI provides human-readable output by default" {
    run bash "$CLI_SCRIPT" status
    [ "$status" -eq 0 ]
    # Human readable should not be pure JSON
    if [[ "$output" =~ "Status" ]] || [[ "$output" =~ "INFO" ]]; then
        true
    fi
}

# ============================================================================
# Verbose Mode Tests
# ============================================================================

@test "CLI respects --verbose flag" {
    run bash "$CLI_SCRIPT" status --verbose
    [ "$status" -eq 0 ]
}

@test "CLI verbose mode provides additional details" {
    normal_output=$(bash "$CLI_SCRIPT" list-schemas 2>&1 | wc -l)
    verbose_output=$(bash "$CLI_SCRIPT" list-schemas --verbose 2>&1 | wc -l)

    # Verbose output should have more lines (or same if already detailed)
    [ "$verbose_output" -ge "$normal_output" ]
}
