#!/usr/bin/env bats

# Test for validate-port-abstraction.sh
# Tests the port abstraction validation functionality

# Load test setup
SCENARIO_TOOLS_DIR="$BATS_TEST_DIRNAME"
SCRIPTS_DIR="$(dirname "$(dirname "$SCENARIO_TOOLS_DIR")")"

# Source dependencies
. "$SCRIPTS_DIR/lib/utils/var.sh"
. "$SCRIPTS_DIR/lib/utils/log.sh"
# Source trash module for safe test cleanup
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# ============================================================================
# File Validation Tests
# ============================================================================

@test "validate_port_abstraction::should_validate_file identifies correct file types" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Test various file types
        test_files=(
            \"/tmp/test.json\"
            \"/tmp/config.yaml\"
            \"/tmp/script.ts\"
            \"/tmp/data.js\"
            \"/tmp/script.sh\"
            \"/tmp/readme.md\"
            \"/tmp/backup.json.backup\"
            \"/tmp/.gitignore\"
            \"/tmp/binary.bin\"
        )
        
        for file in \"\${test_files[@]}\"; do
            if validate_port_abstraction::should_validate_file \"\$file\"; then
                echo \"validate: \$(basename \"\$file\")\"
            else
                echo \"skip: \$(basename \"\$file\")\"
            fi
        done
    "
    [[ "$output" == *"validate: test.json"* ]]
    [[ "$output" == *"validate: config.yaml"* ]]
    [[ "$output" == *"validate: script.ts"* ]]
    [[ "$output" == *"validate: script.sh"* ]]
    [[ "$output" == *"skip: backup.json.backup"* ]]
    [[ "$output" == *"skip: .gitignore"* ]]
}

@test "validate_port_abstraction::find_hardcoded_ports detects port references" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create temporary test files
        TEMP_FILE1=\"/tmp/test-ports1.json\"
        cat > \"\$TEMP_FILE1\" << 'EOF'
{
  \"ollama\": \"localhost:11434\",
  \"n8n\": \"localhost:5678\",
  \"other\": \"example.com:8080\"
}
EOF
        
        TEMP_FILE2=\"/tmp/test-ports2.yaml\"
        cat > \"\$TEMP_FILE2\" << 'EOF'
endpoints:
  - url: localhost:11434/api
  - url: localhost:11434/health
  - url: localhost:6333/collections
EOF
        
        TEMP_FILE3=\"/tmp/test-clean.json\"
        cat > \"\$TEMP_FILE3\" << 'EOF'
{
  \"service\": \"\${service.ollama.url}\",
  \"other\": \"example.com:8080\"
}
EOF
        
        # Test each file
        count1=\$(validate_port_abstraction::find_hardcoded_ports \"\$TEMP_FILE1\" false)
        count2=\$(validate_port_abstraction::find_hardcoded_ports \"\$TEMP_FILE2\" false)
        count3=\$(validate_port_abstraction::find_hardcoded_ports \"\$TEMP_FILE3\" false)
        
        echo \"File1 violations: \$count1\"
        echo \"File2 violations: \$count2\"
        echo \"File3 violations: \$count3\"
        
        # Clean up temp files
        for f in \"\$TEMP_FILE1\" \"\$TEMP_FILE2\" \"\$TEMP_FILE3\"; do
            [[ -f \"\$f\" ]] && rm -f \"\$f\"
        done
    "
    [[ "$output" == *"File1 violations: 2"* ]]
    [[ "$output" == *"File2 violations: 3"* ]]
    [[ "$output" == *"File3 violations: 0"* ]]
}

@test "validate_port_abstraction::find_hardcoded_ports shows verbose output" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create test file
        TEMP_FILE=\"/tmp/test-verbose.json\"
        cat > \"\$TEMP_FILE\" << 'EOF'
line 1: {\"url\": \"localhost:11434\"}
line 2: normal content
line 3: {\"endpoint\": \"localhost:5678/webhook\"}
EOF
        
        # Test with verbose output
        count=\$(validate_port_abstraction::find_hardcoded_ports \"\$TEMP_FILE\" true)
        echo \"Total violations: \$count\"
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"Total violations: 2"* ]]
    [[ "$output" == *"Line"* ]]
    [[ "$output" == *"service.RESOURCE.url"* ]]
}

# ============================================================================
# Service Suggestion Tests
# ============================================================================

@test "validate_port_abstraction::get_service_suggestion returns correct services" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Test known ports
        ports=(\"11434\" \"5678\" \"5681\" \"9200\" \"6333\" \"9000\" \"9999\")
        
        for port in \"\${ports[@]}\"; do
            service=\$(validate_port_abstraction::get_service_suggestion \"\$port\")
            echo \"Port \$port: \$service\"
        done
    "
    [[ "$output" == *"Port 11434: ollama"* ]]
    [[ "$output" == *"Port 5678: n8n"* ]]
    [[ "$output" == *"Port 5681: windmill"* ]]
    [[ "$output" == *"Port 9200: searxng"* ]]
    [[ "$output" == *"Port 6333: qdrant"* ]]
    [[ "$output" == *"Port 9000: minio"* ]]
    [[ "$output" == *"Port 9999: unknown"* ]]
}

# ============================================================================
# File Validation Tests
# ============================================================================

@test "validate_port_abstraction::validate_file passes clean files" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create clean test file
        TEMP_FILE=\"/tmp/test-clean-validation.json\"
        cat > \"\$TEMP_FILE\" << 'EOF'
{
  \"ollama\": \"\${service.ollama.url}\",
  \"n8n\": \"\${service.n8n.url}\",
  \"external\": \"external-service.com:8080\"
}
EOF
        
        if validate_port_abstraction::validate_file \"\$TEMP_FILE\" false false; then
            echo \"validation: passed\"
        else
            echo \"validation: failed\"
        fi
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"validation: passed"* ]]
}

@test "validate_port_abstraction::validate_file fails files with hardcoded ports" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create file with hardcoded ports
        TEMP_FILE=\"/tmp/test-hardcoded-validation.json\"
        cat > \"\$TEMP_FILE\" << 'EOF'
{
  \"ollama\": \"localhost:11434\",
  \"n8n\": \"localhost:5678\"
}
EOF
        
        if validate_port_abstraction::validate_file \"\$TEMP_FILE\" false false; then
            echo \"validation: passed\"
        else
            echo \"validation: failed\"
        fi
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"validation: failed"* ]]
}

@test "validate_port_abstraction::validate_file shows fix suggestions" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create file with hardcoded ports
        TEMP_FILE=\"/tmp/test-suggestions.json\"
        cat > \"\$TEMP_FILE\" << 'EOF'
{\"url\": \"localhost:11434\"}
EOF
        
        # Capture output with suggestions enabled
        validate_port_abstraction::validate_file \"\$TEMP_FILE\" false true 2>&1 || true
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"Fix suggestions"* ]]
    [[ "$output" == *"localhost:11434"* ]]
    [[ "$output" == *"service.ollama.url"* ]]
}

# ============================================================================
# Path Validation Tests
# ============================================================================

@test "validate_port_abstraction::validate_path handles single files" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create test file
        TEMP_FILE=\"/tmp/test-single.json\"
        echo '{\"service\": \"\${service.ollama.url}\"}' > \"\$TEMP_FILE\"
        
        if validate_port_abstraction::validate_path \"\$TEMP_FILE\" false false false 2>/dev/null; then
            echo \"single file: passed\"
        else
            echo \"single file: failed\"
        fi
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"single file: passed"* ]]
}

@test "validate_port_abstraction::validate_path handles directories" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create test directory structure
        TEMP_DIR=\"/tmp/test-validation-dir\"
        mkdir -p \"\$TEMP_DIR\"
        
        echo '{\"url\": \"\${service.ollama.url}\"}' > \"\$TEMP_DIR/clean.json\"
        echo '{\"url\": \"localhost:11434\"}' > \"\$TEMP_DIR/dirty.json\"
        
        # Mock find to avoid complex directory traversal in test
        find() {
            echo \"\$TEMP_DIR/clean.json\"
            echo \"\$TEMP_DIR/dirty.json\"
        }
        export -f find
        
        if validate_port_abstraction::validate_path \"\$TEMP_DIR\" false false false 2>/dev/null; then
            echo \"directory: passed\"
        else
            echo \"directory: failed\"
        fi
        
        # Clean up temp directory
        [[ -d \"\$TEMP_DIR\" ]] && rm -rf \"\$TEMP_DIR\"
    "
    [[ "$output" == *"directory: failed"* ]] || [[ "$status" -eq 1 ]]
}

@test "validate_port_abstraction::validate_path handles non-existent paths" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        validate_port_abstraction::validate_path \"/nonexistent/path\" false false false 2>/dev/null || echo \"error handled\"
    "
    [[ "$output" == *"error handled"* ]] || [[ "$status" -eq 2 ]]
}

# ============================================================================
# Summary and Statistics Tests
# ============================================================================

@test "validate_path produces correct summary statistics" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create test directory with mixed content
        TEMP_DIR=\"/tmp/test-stats\"
        mkdir -p \"\$TEMP_DIR\"
        
        # Create files with known violations
        echo '{\"url\": \"localhost:11434\"}' > \"\$TEMP_DIR/file1.json\"  # 1 violation
        echo 'endpoint: localhost:5678' > \"\$TEMP_DIR/file2.yaml\"       # 1 violation
        echo '{\"service\": \"\${service.ollama.url}\"}' > \"\$TEMP_DIR/file3.json\"  # 0 violations
        
        # Mock find to return our test files
        find() {
            echo \"\$TEMP_DIR/file1.json\"
            echo \"\$TEMP_DIR/file2.yaml\"
            echo \"\$TEMP_DIR/file3.json\"
        }
        export -f find
        
        validate_port_abstraction::validate_path \"\$TEMP_DIR\" false false true 2>&1 || true
        
        # Clean up temp directory
        [[ -d \"\$TEMP_DIR\" ]] && rm -rf \"\$TEMP_DIR\"
    "
    [[ "$output" == *"Files checked:"* ]]
    [[ "$output" == *"Files with violations:"* ]]
    [[ "$output" == *"Total hardcoded port references:"* ]]
}

# ============================================================================
# Command Line Interface Tests
# ============================================================================

@test "validate_port_abstraction::show_help displays usage information" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        validate_port_abstraction::show_help
    "
    [[ "$output" == *"Port Abstraction Validation Tool"* ]]
    [[ "$output" == *"USAGE:"* ]]
    [[ "$output" == *"--verbose"* ]]
    [[ "$output" == *"--fix-suggestions"* ]]
    [[ "$output" == *"EXIT CODES:"* ]]
}

@test "main function handles help flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        main --help 2>/dev/null || echo 'help shown'
    "
    [[ "$output" == *"Port Abstraction Validation Tool"* ]] || [[ "$output" == *"help shown"* ]]
}

@test "main function uses current directory as default path" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Mock validate_path to show the path being used
        validate_port_abstraction::validate_path() {
            echo \"Validating path: \$1\"
            return 0
        }
        export -f validate_port_abstraction::validate_path
        
        # Run without path argument
        main 2>/dev/null
    "
    # Should validate current working directory
    [[ "$output" == *"Validating path:"* ]]
}

@test "main function handles verbose flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create temporary test file
        TEMP_FILE=\"/tmp/test-verbose-flag.json\"
        echo '{\"url\": \"\${service.ollama.url}\"}' > \"\$TEMP_FILE\"
        
        # Mock validate_path to show parameters
        validate_port_abstraction::validate_path() {
            echo \"Path: \$1, Verbose: \$2, Suggestions: \$3, Summary: \$4\"
            return 0
        }
        export -f validate_port_abstraction::validate_path
        
        main \"\$TEMP_FILE\" --verbose
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"Verbose: true"* ]]
}

@test "main function handles fix-suggestions flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Mock validate_path to show parameters
        validate_port_abstraction::validate_path() {
            echo \"Path: \$1, Verbose: \$2, Suggestions: \$3, Summary: \$4\"
            return 0
        }
        export -f validate_port_abstraction::validate_path
        
        main --fix-suggestions
    "
    [[ "$output" == *"Suggestions: true"* ]]
}

@test "main function handles summary-only flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Mock validate_path to show parameters
        validate_port_abstraction::validate_path() {
            echo \"Path: \$1, Verbose: \$2, Suggestions: \$3, Summary: \$4\"
            return 0
        }
        export -f validate_port_abstraction::validate_path
        
        main --summary-only
    "
    [[ "$output" == *"Summary: true"* ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "complete validation workflow detects mixed scenarios" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        
        # Create comprehensive test scenario
        TEMP_DIR=\"/tmp/test-complete-validation\"
        mkdir -p \"\$TEMP_DIR/initialization\"
        mkdir -p \"\$TEMP_DIR/deployment\"
        
        # Create files with various patterns
        cat > \"\$TEMP_DIR/initialization/config.json\" << 'EOF'
{
  \"services\": {
    \"ollama\": \"localhost:11434\",
    \"n8n\": \"\${service.n8n.url}\",
    \"external\": \"api.example.com:443\"
  }
}
EOF
        
        cat > \"\$TEMP_DIR/deployment/script.sh\" << 'EOF'
#!/bin/bash
curl localhost:5678/webhook
curl \${service.minio.url}/health
EOF
        
        # Mock find to return our files
        find() {
            echo \"\$TEMP_DIR/initialization/config.json\"
            echo \"\$TEMP_DIR/deployment/script.sh\"
        }
        export -f find
        
        # Test validation
        validate_port_abstraction::validate_path \"\$TEMP_DIR\" false true false 2>&1 || echo \"validation completed\"
        
        # Clean up temp directory
        [[ -d \"\$TEMP_DIR\" ]] && rm -rf \"\$TEMP_DIR\"
    "
    [[ "$output" == *"hardcoded port"* ]]
    [[ "$output" == *"localhost:11434"* ]]
    [[ "$output" == *"localhost:5678"* ]]
    [[ "$output" == *"Fix suggestions"* ]]
}

@test "function naming follows correct pattern" {
    # Verify all functions use the validate_port_abstraction:: prefix
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/validate-port-abstraction.sh'
        # Get all function names and check they follow the pattern
        declare -F | grep 'validate_port_abstraction::' | wc -l
    "
    # Should have multiple functions with the correct prefix
    [ "$output" -gt 5 ]
}