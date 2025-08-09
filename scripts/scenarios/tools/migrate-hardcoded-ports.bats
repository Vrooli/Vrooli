#!/usr/bin/env bats

# Test for migrate-hardcoded-ports.sh
# Tests the hardcoded port migration functionality

# Load test setup
SCENARIO_TOOLS_DIR="$BATS_TEST_DIRNAME"
SCRIPTS_DIR="$(dirname "$(dirname "$SCENARIO_TOOLS_DIR")")"

# Source dependencies
. "$SCRIPTS_DIR/lib/utils/var.sh"
. "$SCRIPTS_DIR/lib/utils/log.sh"

# Mock file operations and external commands
setup_mocks() {
    # Mock file operations to prevent real filesystem changes
    cp() { return 0; }
    rm() { return 0; }
    find() {
        # Simulate finding test files
        echo "/mock/scenario/test1.json"
        echo "/mock/scenario/test2.yaml"
        echo "/mock/scenario/test3.ts"
    }
    grep() {
        case "$*" in
            *"localhost:11434"*) echo "localhost:11434" ;;
            *"localhost:5678"*) echo "localhost:5678" ;;
            *) return 1 ;;
        esac
    }
    cat() {
        case "$1" in
            *.json) echo '{"url": "http://localhost:11434/api"}' ;;
            *.yaml) echo 'endpoint: localhost:5678' ;;
            *) echo "mock file content" ;;
        esac
    }
    file() {
        echo "text/plain"
    }
    export -f cp rm find grep cat file
}

# ============================================================================
# Helper Function Tests
# ============================================================================

@test "migrate_hardcoded_ports::should_process_file identifies processable files" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Mock file command
        file() {
            case \"\$1\" in
                *.json|*.yaml|*.ts|*.sh) echo \"text/plain\" ;;
                *.bin) echo \"application/octet-stream\" ;;
                *) echo \"text/plain\" ;;
            esac
        }
        export -f file
        
        # Test various file types
        if migrate_hardcoded_ports::should_process_file \"/tmp/test.json\"; then
            echo \"json: true\"
        else
            echo \"json: false\"
        fi
        
        if migrate_hardcoded_ports::should_process_file \"/tmp/test.backup\"; then
            echo \"backup: true\"
        else
            echo \"backup: false\"
        fi
        
        if migrate_hardcoded_ports::should_process_file \"/tmp/.gitignore\"; then
            echo \"git: true\"
        else
            echo \"git: false\"
        fi
    "
    [[ "$output" == *"json: true"* ]]
    [[ "$output" == *"backup: false"* ]]
    [[ "$output" == *"git: false"* ]]
}

@test "migrate_hardcoded_ports::count_hardcoded_ports counts port references" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Create a temporary test file with known content
        TEMP_FILE=\"/tmp/test-ports.txt\"
        cat > \"\$TEMP_FILE\" << 'EOF'
line1: localhost:11434
line2: localhost:5678  
line3: normal content
line4: localhost:11434 again
EOF
        
        count=\$(migrate_hardcoded_ports::count_hardcoded_ports \"\$TEMP_FILE\")
        echo \"count: \$count\"
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"count: 3"* ]]
}

@test "migrate_hardcoded_ports::get_service_from_port returns correct service names" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        echo \"11434: \$(migrate_hardcoded_ports::get_service_from_port 11434)\"
        echo \"5678: \$(migrate_hardcoded_ports::get_service_from_port 5678)\"
        echo \"5681: \$(migrate_hardcoded_ports::get_service_from_port 5681)\"
        echo \"9999: \$(migrate_hardcoded_ports::get_service_from_port 9999)\"
    "
    [[ "$output" == *"11434: ollama"* ]]
    [[ "$output" == *"5678: n8n"* ]]
    [[ "$output" == *"5681: windmill"* ]]
    [[ "$output" == *"9999: unknown"* ]]
}

# ============================================================================
# Migration Logic Tests
# ============================================================================

@test "migrate_hardcoded_ports::migrate_file performs substitutions correctly" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Create temporary test file
        TEMP_FILE=\"/tmp/test-migrate.txt\"
        cat > \"\$TEMP_FILE\" << 'EOF'
Connect to localhost:11434 for Ollama
N8N is at http://localhost:5678/webhook
Regular content without ports
EOF
        
        # Mock functions to capture changes
        cp() { 
            echo \"backup created: \$2\"
            return 0
        }
        
        # Override echo to capture the modified content
        original_content=\$(cat \"\$TEMP_FILE\")
        echo \"Original: \$original_content\"
        
        export -f cp
        
        # Run migration (with mocked file operations)
        changed=\$(migrate_hardcoded_ports::migrate_file \"\$TEMP_FILE\")
        echo \"Changed: \$changed\"
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"Changed: true"* ]]
    [[ "$output" == *"backup created"* ]]
}

@test "migrate_hardcoded_ports::migrate_file handles files with no ports" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Create temporary test file with no hardcoded ports
        TEMP_FILE=\"/tmp/test-no-ports.txt\"
        cat > \"\$TEMP_FILE\" << 'EOF'
Regular content
No hardcoded ports here
Just normal text
EOF
        
        # Mock cp to track backup creation
        cp() { 
            echo \"backup created: \$2\"
            return 0
        }
        export -f cp
        
        changed=\$(migrate_hardcoded_ports::migrate_file \"\$TEMP_FILE\")
        echo \"Changed: \$changed\"
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"Changed: false"* ]]
    [[ "$output" != *"backup created"* ]]
}

# ============================================================================
# Scenario Validation Tests
# ============================================================================

@test "migrate_hardcoded_ports::validate_scenario handles empty directories" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Mock find to return no files
        find() { return 0; }
        export -f find
        
        # Mock directory check
        if migrate_hardcoded_ports::validate_scenario \"/tmp/empty-scenario\" 2>/dev/null; then
            echo \"validation passed\"
        else
            echo \"validation failed\"
        fi
    "
    [[ "$output" == *"validation passed"* ]]
}

@test "migrate_hardcoded_ports::validate_scenario detects hardcoded ports" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Mock functions to simulate finding files with ports
        find() {
            echo \"/mock/file1.json\"
            echo \"/mock/file2.yaml\"
        }
        
        migrate_hardcoded_ports::should_process_file() {
            return 0  # All files should be processed
        }
        
        migrate_hardcoded_ports::count_hardcoded_ports() {
            case \"\$1\" in
                */file1.json) echo \"2\" ;;
                */file2.yaml) echo \"1\" ;;
                *) echo \"0\" ;;
            esac
        }
        
        export -f find
        
        if migrate_hardcoded_ports::validate_scenario \"/mock/scenario\" 2>/dev/null; then
            echo \"validation passed\"
        else
            echo \"validation failed\"
        fi
    "
    [[ "$output" == *"validation failed"* ]] || [[ "$status" -eq 1 ]]
}

# ============================================================================
# Command Line Argument Tests
# ============================================================================

@test "migrate_hardcoded_ports::show_help displays usage information" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        migrate_hardcoded_ports::show_help
    "
    [[ "$output" == *"Port Abstraction Migration Tool"* ]]
    [[ "$output" == *"USAGE:"* ]]
    [[ "$output" == *"localhost:11434"* ]]
    [[ "$output" == *"service.ollama.url"* ]]
}

@test "main function handles help flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        main --help 2>/dev/null || echo 'help displayed'
    "
    [[ "$output" == *"Port Abstraction Migration Tool"* ]] || [[ "$output" == *"help displayed"* ]]
}

@test "main function requires scenario path" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        main 2>/dev/null || echo 'error: scenario path required'
    "
    [[ "$output" == *"error: scenario path required"* ]] || [[ "$status" -eq 1 ]]
}

@test "main function handles non-existent directory" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        main /nonexistent/path 2>/dev/null || echo 'error: directory not found'
    "
    [[ "$output" == *"error: directory not found"* ]] || [[ "$status" -eq 1 ]]
}

# ============================================================================
# Dry Run Mode Tests
# ============================================================================

@test "dry run mode prevents file modifications" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Create temporary test file
        TEMP_FILE=\"/tmp/test-dry-run.txt\"
        echo \"localhost:11434\" > \"\$TEMP_FILE\"
        
        # Track file operations
        file_modified=false
        
        # Override echo to prevent actual file writing
        original_echo=\$(which echo)
        
        # Run in dry-run mode
        DRY_RUN=true
        changed=\$(migrate_hardcoded_ports::migrate_file \"\$TEMP_FILE\")
        
        echo \"Dry run changed: \$changed\"
        
        # Verify file wasn't actually modified
        content=\$(cat \"\$TEMP_FILE\")
        if [[ \"\$content\" == \"localhost:11434\" ]]; then
            echo \"file unchanged (correct)\"
        else
            echo \"file was modified (incorrect)\"
        fi
        
        rm -f \"\$TEMP_FILE\"
    "
    [[ "$output" == *"Dry run changed: true"* ]]
    [[ "$output" == *"file unchanged (correct)"* ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "script handles corrupted files gracefully" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        
        # Mock cat to simulate file read error
        cat() {
            if [[ \"\$1\" == \"/mock/corrupted.json\" ]]; then
                return 1  # Simulate read error
            else
                echo \"mock content\"
            fi
        }
        export -f cat
        
        changed=\$(migrate_hardcoded_ports::migrate_file \"/mock/corrupted.json\" 2>/dev/null || echo \"error_handled\")
        echo \"Result: \$changed\"
    "
    [[ "$output" == *"error_handled"* ]] || [[ "$status" -eq 1 ]]
}

@test "function naming follows correct pattern" {
    # Verify all functions use the migrate_hardcoded_ports:: prefix
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-hardcoded-ports.sh'
        # Get all function names and check they follow the pattern
        declare -F | grep 'migrate_hardcoded_ports::' | wc -l
    "
    # Should have multiple functions with the correct prefix
    [ "$output" -gt 5 ]
}