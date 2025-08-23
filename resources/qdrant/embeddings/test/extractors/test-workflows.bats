#!/usr/bin/env bats
# Test workflows extractor independently

load "../helpers/setup.bash"

setup() {
    # Set up test environment
    TEST_DIR="$BATS_TEST_TMPDIR/workflows_test"
    mkdir -p "$TEST_DIR"
    
    # Source the extractor
    source "$EMBEDDING_ROOT/extractors/workflows.sh"
    
    # Create test fixtures
    setup_workflow_fixtures
}

teardown() {
    rm -rf "$TEST_DIR"
}

setup_workflow_fixtures() {
    # Copy workflow test fixtures
    mkdir -p "$TEST_DIR/initialization/automation/n8n"
    cp "$FIXTURE_ROOT/workflows/email-notification.json" "$TEST_DIR/initialization/automation/n8n/"
    cp "$FIXTURE_ROOT/workflows/data-processing-pipeline.json" "$TEST_DIR/initialization/automation/n8n/"
}

@test "workflows extractor finds workflow files" {
    # Test workflow discovery
    cd "$TEST_DIR"
    
    run qdrant::extract::find_workflows "."
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email-notification.json"* ]]
    [[ "$output" == *"data-processing-pipeline.json"* ]]
}

@test "workflows extractor handles empty directory" {
    # Test with no workflows
    empty_dir="$TEST_DIR/empty"
    mkdir -p "$empty_dir"
    cd "$empty_dir"
    
    run qdrant::extract::find_workflows "."
    
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "workflows extractor processes single workflow correctly" {
    cd "$TEST_DIR"
    
    # Test single workflow processing
    run qdrant::extract::workflow "initialization/automation/n8n/email-notification.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Email Notification Workflow"* ]]
    [[ "$output" == *"webhook"* ]]
    [[ "$output" == *"gmail"* ]]
}

@test "workflows extractor handles malformed JSON gracefully" {
    cd "$TEST_DIR"
    
    # Create malformed JSON file
    echo "{ invalid json" > "initialization/automation/n8n/broken.json"
    
    run qdrant::extract::workflow "initialization/automation/n8n/broken.json"
    
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]] || [[ "$stderr" == *"Error"* ]]
}

@test "workflows extractor processes batch correctly" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/workflow_output.txt"
    
    # Test batch processing
    run qdrant::extract::workflows_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Check output contains expected content
    grep -q "Email Notification Workflow" "$output_file"
    grep -q "Data Processing Pipeline" "$output_file"
    grep -q "---SEPARATOR---" "$output_file"
}

@test "workflows extractor extracts node types correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::workflow "initialization/automation/n8n/data-processing-pipeline.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"cron"* ]]
    [[ "$output" == *"httpRequest"* ]]
    [[ "$output" == *"postgres"* ]]
    [[ "$output" == *"if"* ]]
}

@test "workflows extractor finds sticky notes" {
    cd "$TEST_DIR"
    
    run qdrant::extract::workflow "initialization/automation/n8n/data-processing-pipeline.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Processing completed successfully"* ]]
}

@test "workflows extractor handles missing workflow file" {
    cd "$TEST_DIR"
    
    run qdrant::extract::workflow "nonexistent.json"
    
    [ "$status" -ne 0 ]
}

@test "workflows extractor filters by file pattern correctly" {
    cd "$TEST_DIR"
    
    # Create non-workflow JSON in wrong location
    mkdir -p "$TEST_DIR/config"
    echo '{"not": "a workflow"}' > "$TEST_DIR/config/settings.json"
    
    run qdrant::extract::find_workflows "."
    
    [ "$status" -eq 0 ]
    [[ "$output" != *"settings.json"* ]]  # Should not find config files
}

@test "workflows extractor processes webhook paths correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::workflow "initialization/automation/n8n/email-notification.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"/webhook/email-trigger"* ]]
}

@test "workflows extractor handles empty workflow file" {
    cd "$TEST_DIR"
    
    # Create empty JSON file
    echo '{}' > "initialization/automation/n8n/empty.json"
    
    run qdrant::extract::workflow "initialization/automation/n8n/empty.json"
    
    [ "$status" -eq 0 ]  # Should handle gracefully
    [ -n "$output" ]     # Should produce some output
}

@test "workflows extractor output format is consistent" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/format_test.txt"
    run qdrant::extract::workflows_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    
    # Check format consistency
    local separator_count=$(grep -c "---SEPARATOR---" "$output_file")
    local workflow_count=$(find . -name "*.json" -path "*/initialization/*" | wc -l)
    
    # Should have separators between workflows
    [ "$separator_count" -eq "$workflow_count" ] || [ "$separator_count" -eq $((workflow_count - 1)) ]
}