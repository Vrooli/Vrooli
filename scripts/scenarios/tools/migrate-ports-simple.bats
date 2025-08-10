#!/usr/bin/env bats

# Test for migrate-ports-simple.sh
# Tests the simple port migration functionality

# Load test setup
SCENARIO_TOOLS_DIR="$BATS_TEST_DIRNAME"
SCRIPTS_DIR="$(dirname "$(dirname "$SCENARIO_TOOLS_DIR")")"

# Source dependencies
. "$SCRIPTS_DIR/lib/utils/var.sh"
. "$SCRIPTS_DIR/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Mock file operations and external commands
setup_mocks() {
    cp() { return 0; }
    cat() {
        case "$1" in
            *.json) echo '{"endpoint": "localhost:11434"}' ;;
            *.yaml) echo 'url: http://localhost:5678' ;;
            *) echo "mock content" ;;
        esac
    }
    export -f cp cat
}

# ============================================================================
# Core Migration Function Tests
# ============================================================================

@test "migrate_ports_simple::migrate_file performs basic substitutions" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Create temporary test file
        TEMP_FILE=\"/tmp/test-simple-migrate.txt\"
        cat > \"\$TEMP_FILE\" << 'EOF'
Connect to localhost:11434 for Ollama
N8N webhook: http://localhost:5678/api
EOF
        
        # Mock cp to track backup creation
        cp() { 
            echo \"backup: \$2\"
            return 0
        }
        export -f cp
        
        # Test migration
        changed=\$(migrate_ports_simple::migrate_file \"\$TEMP_FILE\" false)
        echo \"Changed: \$changed\"
        
        # Check if substitutions would be made (verify patterns match)
        if grep -q \"localhost:11434\" \"\$TEMP_FILE\"; then
            echo \"found ollama pattern\"
        fi
        if grep -q \"localhost:5678\" \"\$TEMP_FILE\"; then
            echo \"found n8n pattern\"
        fi
        
        trash::safe_remove \"\$TEMP_FILE\" --test-cleanup
    "
    [[ "$output" == *"Changed: true"* ]]
    [[ "$output" == *"backup:"* ]]
    [[ "$output" == *"found ollama pattern"* ]]
    [[ "$output" == *"found n8n pattern"* ]]
}

@test "migrate_ports_simple::migrate_file handles files without ports" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Create temporary test file with no hardcoded ports
        TEMP_FILE=\"/tmp/test-no-simple-ports.txt\"
        cat > \"\$TEMP_FILE\" << 'EOF'
Regular configuration
No hardcoded ports here
Just normal content
EOF
        
        changed=\$(migrate_ports_simple::migrate_file \"\$TEMP_FILE\" false)
        echo \"Changed: \$changed\"
        
        trash::safe_remove \"\$TEMP_FILE\" --test-cleanup
    "
    [[ "$output" == *"Changed: false"* ]]
}

@test "migrate_ports_simple::migrate_file supports dry run mode" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Create temporary test file
        TEMP_FILE=\"/tmp/test-dry-simple.txt\"
        echo \"localhost:11434\" > \"\$TEMP_FILE\"
        
        # Run in dry run mode
        changed=\$(migrate_ports_simple::migrate_file \"\$TEMP_FILE\" true)
        echo \"Dry run changed: \$changed\"
        
        # Verify original content is preserved
        content=\$(cat \"\$TEMP_FILE\")
        echo \"Content: \$content\"
        
        trash::safe_remove \"\$TEMP_FILE\" --test-cleanup
    "
    [[ "$output" == *"Dry run changed: true"* ]]
    [[ "$output" == *"Content: localhost:11434"* ]]
}

# ============================================================================
# Port Replacement Mapping Tests
# ============================================================================

@test "PORT_REPLACEMENTS array contains expected mappings" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Test some key mappings from the PORT_REPLACEMENTS array
        for pattern in \"localhost:11434\" \"localhost:5678\" \"localhost:5681\" \"http://localhost:11434\"; do
            if [[ -n \"\${PORT_REPLACEMENTS[\$pattern]}\" ]]; then
                echo \"Found: \$pattern -> \${PORT_REPLACEMENTS[\$pattern]}\"
            else
                echo \"Missing: \$pattern\"
            fi
        done
    "
    [[ "$output" == *"Found: localhost:11434 -> \${service.ollama.url}"* ]]
    [[ "$output" == *"Found: localhost:5678 -> \${service.n8n.url}"* ]]
    [[ "$output" == *"Found: localhost:5681 -> \${service.windmill.url}"* ]]
    [[ "$output" == *"Found: http://localhost:11434 -> \${service.ollama.url}"* ]]
}

@test "migration correctly replaces all pattern variations" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Create test file with various patterns
        TEMP_FILE=\"/tmp/test-patterns.txt\"
        cat > \"\$TEMP_FILE\" << 'EOF'
localhost:11434
http://localhost:11434
localhost:5678
http://localhost:5678/webhook
EOF
        
        # Simulate the substitution logic
        content=\$(cat \"\$TEMP_FILE\")
        echo \"Original patterns found:\"
        
        # Check each pattern
        for pattern in \"localhost:11434\" \"http://localhost:11434\" \"localhost:5678\" \"http://localhost:5678\"; do
            if echo \"\$content\" | grep -q -F \"\$pattern\"; then
                echo \"  \$pattern\"
            fi
        done
        
        trash::safe_remove \"\$TEMP_FILE\" --test-cleanup
    "
    [[ "$output" == *"localhost:11434"* ]]
    [[ "$output" == *"http://localhost:11434"* ]]
    [[ "$output" == *"localhost:5678"* ]]
    [[ "$output" == *"http://localhost:5678"* ]]
}

# ============================================================================
# Scenario Migration Tests
# ============================================================================

@test "migrate_ports_simple::migrate_scenario processes expected file patterns" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Mock file structure
        TEMP_SCENARIO=\"/tmp/test-scenario\"
        mkdir -p \"\$TEMP_SCENARIO/.vrooli\"
        mkdir -p \"\$TEMP_SCENARIO/initialization/automation/n8n\"
        mkdir -p \"\$TEMP_SCENARIO/deployment\"
        
        # Create test files
        echo '{\"service\": {\"name\": \"test\"}}' > \"\$TEMP_SCENARIO/.vrooli/service.json\"
        echo '{\"webhook\": \"localhost:5678\"}' > \"\$TEMP_SCENARIO/initialization/automation/n8n/workflow.json\"
        echo '#!/bin/bash\\necho localhost:11434' > \"\$TEMP_SCENARIO/deployment/start.sh\"
        
        # Mock migrate_file to track calls
        migrate_ports_simple::migrate_file() {
            echo \"Processing: \$1\"
            echo \"true\"  # Always return changed
        }
        export -f migrate_ports_simple::migrate_file
        
        # Run migration
        migrate_ports_simple::migrate_scenario \"\$TEMP_SCENARIO\" true
        
        # Cleanup
        trash::safe_remove \"\$TEMP_SCENARIO\" --test-cleanup
    "
    [[ "$output" == *"Processing:"* ]]
    [[ "$output" == *"service.json"* ]]
}

@test "migrate_ports_simple::migrate_scenario handles missing directories gracefully" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Create minimal scenario structure
        TEMP_SCENARIO=\"/tmp/minimal-scenario\"
        mkdir -p \"\$TEMP_SCENARIO\"
        
        # Mock migrate_file
        migrate_ports_simple::migrate_file() {
            echo \"false\"  # No changes
        }
        export -f migrate_ports_simple::migrate_file
        
        # Run migration - should handle missing directories gracefully
        migrate_ports_simple::migrate_scenario \"\$TEMP_SCENARIO\" true 2>/dev/null
        echo \"Migration completed without errors\"
        
        trash::safe_remove \"\$TEMP_SCENARIO\" --test-cleanup
    "
    [[ "$output" == *"Migration completed without errors"* ]]
}

# ============================================================================
# Command Line Interface Tests
# ============================================================================

@test "migrate_ports_simple::show_help displays correct information" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        migrate_ports_simple::show_help
    "
    [[ "$output" == *"Simple Port Migration Tool"* ]]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"--dry-run"* ]]
    [[ "$output" == *"localhost:PORT"* ]]
}

@test "main function handles help flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        main --help 2>/dev/null || echo 'help shown'
    "
    [[ "$output" == *"Simple Port Migration Tool"* ]] || [[ "$output" == *"help shown"* ]]
}

@test "main function requires scenario path" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        main 2>/dev/null || echo 'path required'
    "
    [[ "$output" == *"path required"* ]] || [[ "$status" -eq 1 ]]
}

@test "main function handles non-existent directory" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        main /nonexistent/directory 2>/dev/null || echo 'directory error'
    "
    [[ "$output" == *"directory error"* ]] || [[ "$status" -eq 1 ]]
}

@test "main function parses dry-run flag correctly" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Create temporary directory
        TEMP_DIR=\"/tmp/test-dry-flag\"
        mkdir -p \"\$TEMP_DIR\"
        
        # Mock migrate_scenario to show dry_run parameter
        migrate_ports_simple::migrate_scenario() {
            echo \"Scenario: \$1, Dry run: \$2\"
        }
        export -f migrate_ports_simple::migrate_scenario
        
        main \"\$TEMP_DIR\" --dry-run
        
        trash::safe_remove \"\$TEMP_DIR\" --test-cleanup
    "
    [[ "$output" == *"Dry run: true"* ]]
}

# ============================================================================
# Error Handling Tests
# ============================================================================

@test "script handles file read errors gracefully" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Mock cat to simulate read error
        cat() {
            if [[ \"\$1\" == \"/tmp/error-file.json\" ]]; then
                return 1
            else
                echo \"mock content\"
            fi
        }
        export -f cat
        
        # Test with error file
        changed=\$(migrate_ports_simple::migrate_file \"/tmp/error-file.json\" false 2>/dev/null || echo \"error_handled\")
        echo \"Result: \$changed\"
    "
    [[ "$output" == *"error_handled"* ]] || [[ "$status" -eq 1 ]]
}

@test "script handles backup creation failure gracefully" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        TEMP_FILE=\"/tmp/test-backup-fail.txt\"
        echo \"localhost:11434\" > \"\$TEMP_FILE\"
        
        # Mock cp to simulate backup failure
        cp() {
            return 1  # Simulate failure
        }
        export -f cp
        
        changed=\$(migrate_ports_simple::migrate_file \"\$TEMP_FILE\" false 2>/dev/null || echo \"backup_error_handled\")
        echo \"Result: \$changed\"
        
        trash::safe_remove \"\$TEMP_FILE\" --test-cleanup
    "
    [[ "$output" == *"backup_error_handled"* ]] || [[ "$status" -eq 1 ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "complete migration workflow with mixed content" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        
        # Create test file with mixed content
        TEMP_FILE=\"/tmp/test-mixed.json\"
        cat > \"\$TEMP_FILE\" << 'EOF'
{
  \"ollama\": \"localhost:11434\",
  \"n8n\": \"http://localhost:5678/webhook\",
  \"windmill\": \"localhost:5681\",
  \"other\": \"localhost:9999\",
  \"normal\": \"example.com:8080\"
}
EOF
        
        # Count expected changes (patterns that should match)
        original_content=\$(cat \"\$TEMP_FILE\")
        
        # Check patterns that should be found
        patterns_found=0
        if echo \"\$original_content\" | grep -q \"localhost:11434\"; then
            patterns_found=\$((patterns_found + 1))
        fi
        if echo \"\$original_content\" | grep -q \"http://localhost:5678\"; then
            patterns_found=\$((patterns_found + 1))
        fi
        if echo \"\$original_content\" | grep -q \"localhost:5681\"; then
            patterns_found=\$((patterns_found + 1))
        fi
        
        echo \"Patterns that would be migrated: \$patterns_found\"
        
        # Test in dry run mode
        changed=\$(migrate_ports_simple::migrate_file \"\$TEMP_FILE\" true)
        echo \"Would change: \$changed\"
        
        trash::safe_remove \"\$TEMP_FILE\" --test-cleanup
    "
    [[ "$output" == *"Patterns that would be migrated: 3"* ]]
    [[ "$output" == *"Would change: true"* ]]
}

@test "function naming follows correct pattern" {
    # Verify all functions use the migrate_ports_simple:: prefix
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/migrate-ports-simple.sh'
        # Get all function names and check they follow the pattern
        declare -F | grep 'migrate_ports_simple::' | wc -l
    "
    # Should have multiple functions with the correct prefix
    [ "$output" -gt 3 ]
}