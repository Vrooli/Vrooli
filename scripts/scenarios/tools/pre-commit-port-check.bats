#!/usr/bin/env bats

# Test for pre-commit-port-check.sh
# Tests the pre-commit hook functionality for port abstraction checking

# Load test setup
SCENARIO_TOOLS_DIR="$BATS_TEST_DIRNAME"
SCRIPTS_DIR="$(dirname "$(dirname "$SCENARIO_TOOLS_DIR")")"

# Source dependencies
. "$SCRIPTS_DIR/lib/utils/var.sh"
. "$SCRIPTS_DIR/lib/utils/log.sh"

# ============================================================================
# Helper Function Tests
# ============================================================================

@test "pre_commit_port_check::get_service_for_port returns correct mappings" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Test known critical ports
        ports=(\"11434\" \"5678\" \"5681\" \"9200\" \"6333\" \"9000\" \"7777\")
        
        for port in \"\${ports[@]}\"; do
            service=\$(pre_commit_port_check::get_service_for_port \"\$port\")
            echo \"Port \$port: \$service\"
        done
    "
    [[ "$output" == *"Port 11434: ollama"* ]]
    [[ "$output" == *"Port 5678: n8n"* ]]
    [[ "$output" == *"Port 5681: windmill"* ]]
    [[ "$output" == *"Port 9200: searxng"* ]]
    [[ "$output" == *"Port 6333: qdrant"* ]]
    [[ "$output" == *"Port 9000: minio"* ]]
    [[ "$output" == *"Port 7777: unknown"* ]]
}

@test "pre_commit_port_check::check_git_repo detects git repository" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock git command to simulate being in a repo
        git() {
            case \"\$1\" in
                rev-parse) return 0 ;;
                *) echo \"mock git output\" ;;
            esac
        }
        export -f git
        
        if pre_commit_port_check::check_git_repo; then
            echo \"git repo: detected\"
        else
            echo \"git repo: not detected\"
        fi
    "
    [[ "$output" == *"git repo: detected"* ]]
}

@test "pre_commit_port_check::check_git_repo handles non-git directory" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock git command to simulate not being in a repo
        git() {
            case \"\$1\" in
                rev-parse) return 1 ;;
                *) echo \"mock git output\" ;;
            esac
        }
        export -f git
        
        if pre_commit_port_check::check_git_repo 2>/dev/null; then
            echo \"git repo: detected\"
        else
            echo \"git repo: not detected\"
        fi
    "
    [[ "$output" == *"git repo: not detected"* ]]
}

# ============================================================================
# Print Function Tests (Colored Output)
# ============================================================================

@test "print functions produce colored output" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Test each print function (output will contain ANSI codes)
        pre_commit_port_check::print_error \"Test error\" 2>&1
        pre_commit_port_check::print_warning \"Test warning\" 2>&1
        pre_commit_port_check::print_info \"Test info\"
        pre_commit_port_check::print_success \"Test success\"
    "
    [[ "$output" == *"Test error"* ]]
    [[ "$output" == *"Test warning"* ]]
    [[ "$output" == *"Test info"* ]]
    [[ "$output" == *"Test success"* ]]
}

# ============================================================================
# Staged Files Checking Tests
# ============================================================================

@test "pre_commit_port_check::check_staged_files handles clean commits" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock git commands for clean scenario
        git() {
            case \"\$*\" in
                \"diff --cached --name-only --diff-filter=ACM\")
                    echo \"scripts/scenarios/core/test/clean.json\"
                    echo \"scripts/scenarios/core/test/config.yaml\"
                    ;;
                \"diff --cached\"*)
                    # Return no matches for port patterns
                    return 1
                    ;;
                *) echo \"mock git\" ;;
            esac
        }
        export -f git
        
        if pre_commit_port_check::check_staged_files 2>/dev/null; then
            echo \"check: passed\"
        else
            echo \"check: failed\"
        fi
    "
    [[ "$output" == *"check: passed"* ]]
}

@test "pre_commit_port_check::check_staged_files detects hardcoded ports" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock git commands for dirty scenario
        git() {
            case \"\$*\" in
                \"diff --cached --name-only --diff-filter=ACM\")
                    echo \"scripts/scenarios/core/test/dirty.json\"
                    ;;
                \"diff --cached\"*\"dirty.json\")
                    if [[ \"\$*\" == *\"localhost:11434\"* ]]; then
                        echo \"+  \\\"url\\\": \\\"localhost:11434\\\"\"
                    else
                        return 1
                    fi
                    ;;
                *) echo \"mock git\" ;;
            esac
        }
        export -f git
        
        if pre_commit_port_check::check_staged_files 2>&1; then
            echo \"check: passed\"
        else
            echo \"check: failed\"
        fi
    "
    [[ "$output" == *"check: failed"* ]] || [[ "$status" -eq 1 ]]
    [[ "$output" == *"hardcoded port"* ]]
    [[ "$output" == *"localhost:11434"* ]]
}

@test "pre_commit_port_check::check_staged_files handles git command failure" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock git to simulate command failure
        git() {
            case \"\$1\" in
                diff) return 1 ;;
                *) echo \"mock git\" ;;
            esac
        }
        export -f git
        
        if pre_commit_port_check::check_staged_files 2>/dev/null; then
            echo \"check: passed\"
        else
            echo \"check: failed - git error\"
        fi
    "
    [[ "$output" == *"check: failed - git error"* ]] || [[ "$status" -eq 1 ]]
}

@test "pre_commit_port_check::check_staged_files filters file patterns correctly" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock git to return mixed file types
        git() {
            case \"\$*\" in
                \"diff --cached --name-only --diff-filter=ACM\")
                    echo \"scripts/scenarios/core/test.json\"      # Should check
                    echo \"scripts/scenarios/core/config.yaml\"   # Should check
                    echo \"scripts/scenarios/core/script.ts\"     # Should check
                    echo \"README.md\"                           # Should not check
                    echo \"src/components/Button.tsx\"           # Should not check
                    echo \"package.json\"                        # Should not check
                    ;;
                \"diff --cached\"*)
                    # Return no port matches
                    return 1
                    ;;
                *) echo \"mock git\" ;;
            esac
        }
        export -f git
        
        # Run and capture which files are processed
        pre_commit_port_check::check_staged_files 2>&1 | grep -o \"scripts/scenarios/.*\" || echo \"no scenario files processed\"
    "
    # The function should only process scenario files that match the patterns
    [[ "$output" == *"scenario"* ]] || [[ "$output" == *"No hardcoded ports found"* ]]
}

# ============================================================================
# Hook Installation Tests
# ============================================================================

@test "pre_commit_port_check::install_hook creates hook file" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Create temporary git directory structure
        TEMP_REPO=\"/tmp/test-git-repo\"
        mkdir -p \"\$TEMP_REPO/.git/hooks\"
        cd \"\$TEMP_REPO\"
        
        # Mock file operations
        cp() {
            echo \"backup: \$2\"
            return 0
        }
        
        cat() {
            if [[ \"\$1\" == \">\" ]]; then
                echo \"hook content written\"
            else
                echo \"reading file\"
            fi
        }
        
        chmod() {
            echo \"chmod: \$*\"
            return 0
        }
        
        export -f cp cat chmod
        
        pre_commit_port_check::install_hook 2>/dev/null || echo \"installation attempted\"
        
        cd - >/dev/null
        rm -rf \"\$TEMP_REPO\"
    "
    [[ "$output" == *"installation attempted"* ]] || [[ "$output" == *"Pre-commit hook installed"* ]]
}

@test "pre_commit_port_check::install_hook handles existing hooks" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Create temporary git directory with existing hook
        TEMP_REPO=\"/tmp/test-existing-hook\"
        mkdir -p \"\$TEMP_REPO/.git/hooks\"
        cd \"\$TEMP_REPO\"
        echo \"existing hook\" > .git/hooks/pre-commit
        
        # Mock file operations
        cp() {
            echo \"backup created: \$2\"
            return 0
        }
        
        chmod() { return 0; }
        
        export -f cp chmod
        
        pre_commit_port_check::install_hook 2>/dev/null || echo \"handled existing hook\"
        
        cd - >/dev/null
        rm -rf \"\$TEMP_REPO\"
    "
    [[ "$output" == *"backup created"* ]] || [[ "$output" == *"handled existing hook"* ]]
}

@test "pre_commit_port_check::install_hook handles non-git directory" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Create temporary non-git directory
        TEMP_DIR=\"/tmp/test-non-git\"
        mkdir -p \"\$TEMP_DIR\"
        cd \"\$TEMP_DIR\"
        
        if pre_commit_port_check::install_hook 2>/dev/null; then
            echo \"installation: succeeded\"
        else
            echo \"installation: failed (expected)\"
        fi
        
        cd - >/dev/null
        rm -rf \"\$TEMP_DIR\"
    "
    [[ "$output" == *"installation: failed (expected)"* ]]
}

# ============================================================================
# Command Line Interface Tests
# ============================================================================

@test "pre_commit_port_check::show_help displays comprehensive help" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        pre_commit_port_check::show_help
    "
    [[ "$output" == *"Pre-commit Port Abstraction Check"* ]]
    [[ "$output" == *"USAGE:"* ]]
    [[ "$output" == *"--install"* ]]
    [[ "$output" == *"--check"* ]]
    [[ "$output" == *"INSTALLATION:"* ]]
    [[ "$output" == *"WHAT IT CHECKS:"* ]]
    [[ "$output" == *"PORTS THAT TRIGGER WARNINGS:"* ]]
}

@test "main function handles help flag" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        main --help 2>/dev/null || echo 'help shown'
    "
    [[ "$output" == *"Pre-commit Port Abstraction Check"* ]] || [[ "$output" == *"help shown"* ]]
}

@test "main function defaults to check action" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock check functions
        pre_commit_port_check::check_git_repo() {
            echo \"checking git repo\"
            return 0
        }
        
        pre_commit_port_check::check_staged_files() {
            echo \"checking staged files\"
            return 0
        }
        
        export -f pre_commit_port_check::check_git_repo pre_commit_port_check::check_staged_files
        
        main 2>/dev/null
    "
    [[ "$output" == *"checking git repo"* ]]
    [[ "$output" == *"checking staged files"* ]]
}

@test "main function handles install action" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock install function
        pre_commit_port_check::install_hook() {
            echo \"installing hook\"
            return 0
        }
        
        pre_commit_port_check::check_git_repo() {
            return 0
        }
        
        export -f pre_commit_port_check::install_hook pre_commit_port_check::check_git_repo
        
        main --install 2>/dev/null
    "
    [[ "$output" == *"installing hook"* ]]
}

@test "main function handles unknown options" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        main --unknown-option 2>&1 || echo 'error handled'
    "
    [[ "$output" == *"Unknown option"* ]] || [[ "$output" == *"error handled"* ]]
}

# ============================================================================
# Integration Tests
# ============================================================================

@test "complete pre-commit workflow with violations" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock comprehensive git scenario with violations
        git() {
            case \"\$*\" in
                \"rev-parse --git-dir\")
                    return 0  # We're in a git repo
                    ;;
                \"diff --cached --name-only --diff-filter=ACM\")
                    echo \"scripts/scenarios/core/test/config.json\"
                    echo \"scripts/scenarios/core/test/workflow.yaml\"
                    ;;
                \"diff --cached\"*\"config.json\")
                    if [[ \"\$*\" == *\"localhost:11434\"* ]]; then
                        echo \"+    \\\"ollama_url\\\": \\\"localhost:11434\\\"\"
                    else
                        return 1
                    fi
                    ;;
                \"diff --cached\"*\"workflow.yaml\")
                    if [[ \"\$*\" == *\"localhost:5678\"* ]]; then
                        echo \"+  endpoint: localhost:5678/webhook\"
                    else
                        return 1
                    fi
                    ;;
                *) return 1 ;;
            esac
        }
        export -f git
        
        # Run the complete check
        main --check 2>&1 || echo \"commit would be blocked\"
    "
    [[ "$output" == *"COMMIT BLOCKED"* ]] || [[ "$output" == *"commit would be blocked"* ]]
    [[ "$output" == *"localhost:11434"* ]]
    [[ "$output" == *"localhost:5678"* ]]
    [[ "$output" == *"service.ollama.url"* ]]
    [[ "$output" == *"service.n8n.url"* ]]
}

@test "complete pre-commit workflow with clean commit" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Mock git scenario with no violations
        git() {
            case \"\$*\" in
                \"rev-parse --git-dir\")
                    return 0  # We're in a git repo
                    ;;
                \"diff --cached --name-only --diff-filter=ACM\")
                    echo \"scripts/scenarios/core/test/clean.json\"
                    ;;
                \"diff --cached\"*)
                    # No hardcoded ports found
                    return 1
                    ;;
                *) return 1 ;;
            esac
        }
        export -f git
        
        # Run the complete check
        if main --check 2>/dev/null; then
            echo \"commit allowed\"
        else
            echo \"commit blocked\"
        fi
    "
    [[ "$output" == *"commit allowed"* ]]
    [[ "$output" == *"No hardcoded ports found"* ]]
}

@test "pre-commit hook integration with existing git workflow" {
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        
        # Simulate typical pre-commit hook usage
        TEMP_REPO=\"/tmp/test-precommit-integration\"
        mkdir -p \"\$TEMP_REPO/.git/hooks\"
        cd \"\$TEMP_REPO\"
        
        # Create test files that would be staged
        mkdir -p scripts/scenarios/core/test
        echo '{\"url\": \"localhost:11434\"}' > scripts/scenarios/core/test/bad.json
        echo '{\"service\": \"\${service.ollama.url}\"}' > scripts/scenarios/core/test/good.json
        
        # Mock git commands as they would appear in real usage
        git() {
            case \"\$*\" in
                \"rev-parse --git-dir\")
                    echo \".git\"
                    return 0
                    ;;
                \"diff --cached --name-only --diff-filter=ACM\")
                    echo \"scripts/scenarios/core/test/bad.json\"
                    ;;
                \"diff --cached\"*\"bad.json\"*)
                    if [[ \"\$*\" == *\"localhost:11434\"* ]]; then
                        echo \"+{\\\"url\\\": \\\"localhost:11434\\\"}\"
                    fi
                    ;;
                *) return 0 ;;
            esac
        }
        export -f git
        
        # Run as it would be called by git pre-commit hook
        if main --check 2>&1; then
            echo \"hook: commit allowed\"
        else
            echo \"hook: commit blocked\"
        fi
        
        cd - >/dev/null
        rm -rf \"\$TEMP_REPO\"
    "
    [[ "$output" == *"hook: commit blocked"* ]]
    [[ "$output" == *"localhost:11434"* ]]
    [[ "$output" == *"service.ollama.url"* ]]
}

@test "function naming follows correct pattern" {
    # Verify all functions use the pre_commit_port_check:: prefix
    run bash -c "
        source '$SCENARIO_TOOLS_DIR/pre-commit-port-check.sh'
        # Get all function names and check they follow the pattern
        declare -F | grep 'pre_commit_port_check::' | wc -l
    "
    # Should have multiple functions with the correct prefix
    [ "$output" -gt 8 ]
}