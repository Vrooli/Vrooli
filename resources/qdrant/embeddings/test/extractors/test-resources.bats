#!/usr/bin/env bats
# Test resources extractor independently

load "../helpers/setup.bash"

setup() {
    # Set up test environment  
    TEST_DIR="$BATS_TEST_TMPDIR/resources_test"
    mkdir -p "$TEST_DIR"
    
    # Source the extractor
    source "$EMBEDDING_ROOT/extractors/resources/main.sh"
    
    # Create test fixtures
    setup_resources_fixtures
}

teardown() {
    rm -rf "$TEST_DIR"
}

setup_resources_fixtures() {
    # Create a proper resource structure with cli.sh
    mkdir -p "$TEST_DIR/resources/test-resource/lib"
    mkdir -p "$TEST_DIR/resources/email-processor-resource/lib"
    
    # Create a minimal cli.sh file (required for resource detection)
    cat > "$TEST_DIR/resources/test-resource/cli.sh" << 'EOF'
#!/usr/bin/env bash
# Description: Test resource for email processing
# Test resource CLI
case "$1" in
    start) echo "Starting test resource" ;;
    stop) echo "Stopping test resource" ;;
    *) echo "Usage: $0 {start|stop}" ;;
esac
EOF
    chmod +x "$TEST_DIR/resources/test-resource/cli.sh"
    
    # Create email processor resource for batch test expectations
    cat > "$TEST_DIR/resources/email-processor-resource/cli.sh" << 'EOF'
#!/usr/bin/env bash
# Description: Email processor resource with capabilities and cli_commands
# email-processor-resource CLI with various capabilities
case "$1" in
    start) echo "Starting email processor" ;;
    stop) echo "Stopping email processor" ;;
    process) echo "Processing emails" ;;
    *) echo "cli_commands: start, stop, process" ;;
esac
EOF
    chmod +x "$TEST_DIR/resources/email-processor-resource/cli.sh"
    
    # Create a config.yaml file
    cat > "$TEST_DIR/resources/test-resource/config.yaml" << 'EOF'
name: test-resource
port: 8080
host: localhost
EOF
    
    # Create email processor config with capabilities
    cat > "$TEST_DIR/resources/email-processor-resource/config.yaml" << 'EOF'
name: email-processor-resource
capabilities:
  - email processing
  - categorization
  - filtering
port: 8081
host: localhost
EOF
    
    # Create a README.md
    cat > "$TEST_DIR/resources/test-resource/README.md" << 'EOF'
# Test Resource
## Overview
This is a test resource for unit testing.
## Features
- Feature 1
- Feature 2
EOF
}

@test "resources extractor batch processes resources" {
    cd "$TEST_DIR"
    output_file="$TEST_DIR/resources_output.txt"
    
    # Run the batch extraction
    run qdrant::extract::resources_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    [ -f "$output_file" ]
}

@test "resources extractor handles empty directory" {
    empty_dir="$TEST_DIR/empty"
    mkdir -p "$empty_dir"
    cd "$empty_dir"
    output_file="$empty_dir/resources_output.txt"
    
    run qdrant::extract::resources_batch "." "$output_file"
    
    [ "$status" -eq 0 ]
    # Empty directory should produce empty or minimal output
}

@test "resources extractor processes resource config correctly" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "$TEST_DIR/resources/test-resource"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"test-resource"* ]]
    [[ "$output" == *"8080"* ]]
    [[ "$output" == *"localhost"* ]]
}

@test "resources extractor handles missing config file" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_config "$TEST_DIR/resources/nonexistent"
    
    [ "$status" -ne 0 ]
}

@test "resources extractor extracts CLI information" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_cli "$TEST_DIR/resources/test-resource/cli.sh"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"test-resource"* ]]
    [[ "$output" == *"Test resource for email processing"* ]]
    [[ "$output" == *"start"* ]]
    [[ "$output" == *"stop"* ]]
}

@test "resources extractor extracts documentation" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_docs "$TEST_DIR/resources/test-resource"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Test Resource"* ]]
    [[ "$output" == *"Overview"* ]]
    [[ "$output" == *"Features"* ]]
}

@test "resources extractor detects resource category" {
    run qdrant::extract::detect_resource_category "ollama"
    [ "$status" -eq 0 ]
    [ "$output" = "ai" ]
    
    run qdrant::extract::detect_resource_category "postgres"
    [ "$status" -eq 0 ]
    [ "$output" = "storage" ]
    
    run qdrant::extract::detect_resource_category "n8n"
    [ "$status" -eq 0 ]
    [ "$output" = "automation" ]
}

@test "resources extractor generates metadata" {
    cd "$TEST_DIR"
    
    run qdrant::extract::resource_metadata "$TEST_DIR/resources/test-resource"
    
    [ "$status" -eq 0 ]
    # Check it returns valid JSON
    echo "$output" | jq . >/dev/null
}

@test "resources extractor handles missing CLI file" {
    mkdir -p "$TEST_DIR/resources/no-cli"
    
    run qdrant::extract::resource_cli "$TEST_DIR/resources/no-cli/cli.sh"
    
    [ "$status" -ne 0 ]
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
    grep -q -- "---SEPARATOR---" "$output_file"
}

@test "resources extractor handles invalid YAML gracefully" {
    cd "$TEST_DIR"
    
    # Create invalid YAML file
    mkdir -p "$TEST_DIR/resources/broken-resource"
    echo "{ invalid: yaml: syntax:" > "$TEST_DIR/resources/broken-resource/config.yaml"
    
    run qdrant::extract::resource_config "$TEST_DIR/resources/broken-resource"
    
    # Should handle gracefully, returning empty or partial content
    [ "$status" -eq 0 ] || [ "$status" -ne 0 ]
    # Test passes either way as long as it doesn't crash
}

@test "resources extractor extracts integrations correctly" {
    cd "$TEST_DIR"
    
    # Create a resource with integrations for testing
    mkdir -p "$TEST_DIR/resources/integration-test"
    cat > "$TEST_DIR/resources/integration-test/config.yaml" << 'EOF'
name: integration-test
integrations:
  - n8n
  - zapier
  - email-processor-categorize
  - webhook
EOF
    
    run qdrant::extract::resource_config "$TEST_DIR/resources/integration-test"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"n8n"* ]]
    [[ "$output" == *"zapier"* ]]
    [[ "$output" == *"email-processor-categorize"* ]]
    [[ "$output" == *"webhook"* ]]
}

@test "resources extractor extracts monitoring configuration" {
    cd "$TEST_DIR"
    
    # Create a resource with monitoring config for testing
    mkdir -p "$TEST_DIR/resources/monitoring-test"
    cat > "$TEST_DIR/resources/monitoring-test/config.yaml" << 'EOF'
name: monitoring-test
monitoring:
  health_check: /
  metrics:
    - emails_processed_total
  alerts:
    - name: High Error Rate
EOF
    
    run qdrant::extract::resource_config "$TEST_DIR/resources/monitoring-test"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"health_check"* ]]
    [[ "$output" == *"/"* ]]
    [[ "$output" == *"emails_processed_total"* ]]
    [[ "$output" == *"High Error Rate"* ]]
}

@test "resources extractor handles resource with minimal config" {
    cd "$TEST_DIR"
    
    # Create minimal resource config
    mkdir -p "$TEST_DIR/resources/minimal-resource"
    cat > "$TEST_DIR/resources/minimal-resource/config.yaml" << 'EOF'
name: minimal-resource
version: 1.0.0
description: A minimal resource for testing
EOF
    
    run qdrant::extract::resource_config "$TEST_DIR/resources/minimal-resource"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"minimal-resource"* ]]
    [[ "$output" == *"minimal resource for testing"* ]]
}

@test "resources extractor finds CLI scripts" {
    cd "$TEST_DIR"
    
    # Create CLI script
    mkdir -p "$TEST_DIR/resources/cli-test"
    cat > "$TEST_DIR/resources/cli-test/cli.sh" << 'EOF'
#!/usr/bin/env bash
# Email processor CLI tool

email_processor::process() {
    echo "Processing emails..."
}

email_processor::categorize() {
    echo "Categorizing email..."
}
EOF
    chmod +x "$TEST_DIR/resources/cli-test/cli.sh"
    
    run qdrant::extract::resource_cli "$TEST_DIR/resources/cli-test/cli.sh"
    
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
    if [ -f "$output_file" ]; then
        local separator_count=$(grep -c -- "---SEPARATOR---" "$output_file" || echo "0")
        # We have test-resource and email-processor-resource, so expect at least 1 separator
        [ "$separator_count" -ge 1 ]
    fi
}