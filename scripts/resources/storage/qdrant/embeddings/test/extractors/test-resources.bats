#!/usr/bin/env bats
# Test resources extractor independently

load "../helpers/setup.bash"

setup() {
    # Set up test environment  
    TEST_DIR="$BATS_TEST_TMPDIR/resources_test"
    mkdir -p "$TEST_DIR"
    
    # Source the extractor
    source "$EMBEDDING_ROOT/extractors/resources.sh"
    
    # Create test fixtures
    setup_resources_fixtures
}

teardown() {
    rm -rf "$TEST_DIR"
}

setup_resources_fixtures() {
    # Copy resource test fixtures
    mkdir -p "$TEST_DIR/scripts/resources/email-processing"
    cp "$FIXTURE_ROOT/resources/sample-resource-config.json" "$TEST_DIR/scripts/resources/email-processing/config.json"
}

@test "resources extractor finds resource directories" {
    cd "$TEST_DIR"
    
    run qdrant::extract::find_resources "."
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email-processing"* ]]
}

@test "resources extractor handles empty directory" {
    empty_dir="$TEST_DIR/empty"
    mkdir -p "$empty_dir"
    cd "$empty_dir"
    
    run qdrant::extract::find_resources "."
    
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

@test "resources extractor processes resource config correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email-processor-resource"* ]]
    [[ "$output" == *"Email processing and categorization"* ]]
    [[ "$output" == *"AI integration"* ]]
    [[ "$output" == *"capabilities"* ]]
}

@test "resources extractor handles missing config file" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/nonexistent/config.json"
    
    [ "$status" -ne 0 ]
}

@test "resources extractor extracts capabilities correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email_processing"* ]]
    [[ "$output" == *"ai_integration"* ]]
    [[ "$output" == *"provider_integration"* ]]
    [[ "$output" == *"500 emails/second"* ]]
}

@test "resources extractor extracts CLI commands correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"process"* ]]
    [[ "$output" == *"categorize"* ]]
    [[ "$output" == *"analytics"* ]]
    [[ "$output" == *"--batch"* ]]
    [[ "$output" == *"--model"* ]]
}

@test "resources extractor extracts API endpoints correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"/api/emails"* ]]
    [[ "$output" == *"GET"* ]]
    [[ "$output" == *"POST"* ]]
    [[ "$output" == *"categorize"* ]]
    [[ "$output" == *"batch/process"* ]]
}

@test "resources extractor extracts dependencies correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"node >= 18.0.0"* ]]
    [[ "$output" == *"postgresql"* ]]
    [[ "$output" == *"express"* ]]
    [[ "$output" == *"Gmail API"* ]]
}

@test "resources extractor extracts Docker configuration" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email-processor:1.2.0"* ]]
    [[ "$output" == *"8080:8080"* ]]
    [[ "$output" == *"email_data"* ]]
}

@test "resources extractor processes batch correctly" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/resources_output.txt"
    
    run qdrant::extract::resources_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
    
    # Check output contains expected content
    grep -q "email-processor-resource" "$output_file"
    grep -q "capabilities" "$output_file"
    grep -q "cli_commands" "$output_file"
    grep -q "---SEPARATOR---" "$output_file"
}

@test "resources extractor handles invalid JSON gracefully" {
    cd "$TEST_DIR"
    
    # Create invalid JSON file
    mkdir -p "scripts/resources/broken-resource"
    echo "{ invalid json" > "scripts/resources/broken-resource/config.json"
    
    run qdrant::extract::resource_config "scripts/resources/broken-resource/config.json"
    
    [ "$status" -ne 0 ]
    [[ "$output" == *"Error"* ]] || [[ "$stderr" == *"Error"* ]]
}

@test "resources extractor extracts integrations correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"n8n"* ]]
    [[ "$output" == *"zapier"* ]]
    [[ "$output" == *"email-processor-categorize"* ]]
    [[ "$output" == *"webhook"* ]]
}

@test "resources extractor extracts monitoring configuration" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "scripts/resources/email-processing/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"health_check"* ]]
    [[ "$output" == *"/health"* ]]
    [[ "$output" == *"emails_processed_total"* ]]
    [[ "$output" == *"High Error Rate"* ]]
}

@test "resources extractor handles resource with minimal config" {
    cd "$TEST_DIR"
    
    # Create minimal resource config
    mkdir -p "scripts/resources/minimal-resource"
    cat > "scripts/resources/minimal-resource/config.json" << 'EOF'
{
  "name": "minimal-resource",
  "version": "1.0.0",
  "description": "A minimal resource for testing"
}
EOF
    
    run qdrant::extract::resource_config "scripts/resources/minimal-resource/config.json"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"minimal-resource"* ]]
    [[ "$output" == *"minimal resource for testing"* ]]
}

@test "resources extractor finds CLI scripts" {
    cd "$TEST_DIR"
    
    # Create CLI script
    mkdir -p "scripts/resources/email-processing/cli"
    cat > "scripts/resources/email-processing/cli/email-processor.sh" << 'EOF'
#!/usr/bin/env bash
# Email processor CLI tool

email_processor::process() {
    echo "Processing emails..."
}

email_processor::categorize() {
    echo "Categorizing email..."
}
EOF
    
    run qdrant::extract::resource_cli "scripts/resources/email-processing"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"email_processor::process"* ]]
    [[ "$output" == *"email_processor::categorize"* ]]
}

@test "resources extractor output format is consistent" {
    cd "$TEST_DIR"
    
    output_file="$TEST_DIR/format_test.txt"
    run qdrant::extract::resources_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    
    # Check format consistency
    local separator_count=$(grep -c "---SEPARATOR---" "$output_file")
    local resource_count=$(find . -name "config.json" -path "*/scripts/resources/*" | wc -l)
    
    # Should have appropriate separators
    [ "$separator_count" -eq "$resource_count" ] || [ "$separator_count" -eq $((resource_count - 1)) ]
}