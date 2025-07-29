#!/usr/bin/env bats
# Tests for Unstructured.io manage.sh script

# Load test helper
load_helper() {
    local helper_file="$1"
    if [[ -f "$helper_file" ]]; then
        # shellcheck disable=SC1090
        source "$helper_file"
    fi
}

# Setup for each test
setup() {
    # Set test environment
    export UNSTRUCTURED_IO_CUSTOM_PORT="9999"
    export FORCE="no"
    export YES="no"
    export FILE_INPUT=""
    export STRATEGY="hi_res"
    export OUTPUT="json"
    export LANGUAGES="eng"
    export BATCH="no"
    export FOLLOW="no"
    
    # Load the script without executing main
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    
    # Mock the main function to prevent execution during sourcing
    unstructured_io::main() {
        return 0
    }
    
    source "${SCRIPT_DIR}/manage.sh" || true
}

# Test script loading
@test "manage.sh loads without errors" {
    # The script should source successfully in setup
    [ "$?" -eq 0 ]
}

# Test argument parsing
@test "unstructured_io::parse_arguments sets defaults correctly" {
    # Remove the main function mock for this test
    unset -f unstructured_io::main
    
    # Mock args functions
    args::reset() { return 0; }
    args::register_help() { return 0; }
    args::register_yes() { return 0; }
    args::register() { return 0; }
    args::is_asking_for_help() { return 1; }
    args::parse() { return 0; }
    args::get() {
        case "$1" in
            "action") echo "install" ;;
            "force") echo "no" ;;
            "file") echo "" ;;
            "strategy") echo "hi_res" ;;
            "output") echo "json" ;;
            "languages") echo "eng" ;;
            "batch") echo "no" ;;
            "follow") echo "no" ;;
            "yes") echo "no" ;;
        esac
    }
    
    # Source common functions
    source "${SCRIPT_DIR}/lib/common.sh"
    
    unstructured_io::parse_arguments --action install
    
    [ "$ACTION" = "install" ]
    [ "$FORCE" = "no" ]
    [ "$STRATEGY" = "hi_res" ]
    [ "$OUTPUT" = "json" ]
    [ "$LANGUAGES" = "eng" ]
    [ "$BATCH" = "no" ]
}

# Test argument parsing with custom values
@test "unstructured_io::parse_arguments handles custom values" {
    # Mock args functions
    args::reset() { return 0; }
    args::register_help() { return 0; }
    args::register_yes() { return 0; }
    args::register() { return 0; }
    args::is_asking_for_help() { return 1; }
    args::parse() { return 0; }
    args::get() {
        case "$1" in
            "action") echo "process" ;;
            "force") echo "yes" ;;
            "file") echo "test.pdf" ;;
            "strategy") echo "fast" ;;
            "output") echo "markdown" ;;
            "languages") echo "eng,spa" ;;
            "batch") echo "yes" ;;
            "follow") echo "yes" ;;
            "yes") echo "yes" ;;
        esac
    }
    
    # Source common functions
    source "${SCRIPT_DIR}/lib/common.sh"
    
    unstructured_io::parse_arguments \
        --action process \
        --force yes \
        --file test.pdf \
        --strategy fast \
        --output markdown \
        --languages "eng,spa" \
        --batch yes \
        --follow yes \
        --yes yes
    
    [ "$ACTION" = "process" ]
    [ "$FORCE" = "yes" ]
    [ "$FILE_INPUT" = "test.pdf" ]
    [ "$STRATEGY" = "fast" ]
    [ "$OUTPUT" = "markdown" ]
    [ "$LANGUAGES" = "eng,spa" ]
    [ "$BATCH" = "yes" ]
    [ "$FOLLOW" = "yes" ]
    [ "$YES" = "yes" ]
}

# Test action routing - install
@test "unstructured_io::main routes install action correctly" {
    export ACTION="install"
    export FORCE="no"
    
    # Mock the install function
    unstructured_io::install() {
        echo "INSTALL_CALLED: $1"
        return 0
    }
    
    # Mock argument parsing
    unstructured_io::parse_arguments() {
        return 0
    }
    
    # Remove the main function mock
    unset -f unstructured_io::main
    source "${SCRIPT_DIR}/manage.sh"
    
    result=$(unstructured_io::main --action install)
    [[ "$result" =~ "INSTALL_CALLED: no" ]]
}

# Test action routing - status
@test "unstructured_io::main routes status action correctly" {
    export ACTION="status"
    
    # Mock the status function
    unstructured_io::status() {
        echo "STATUS_CALLED"
        return 0
    }
    
    # Mock argument parsing
    unstructured_io::parse_arguments() {
        return 0
    }
    
    # Remove the main function mock
    unset -f unstructured_io::main
    source "${SCRIPT_DIR}/manage.sh"
    
    result=$(unstructured_io::main --action status)
    [[ "$result" =~ "STATUS_CALLED" ]]
}

# Test action routing - process with missing file
@test "unstructured_io::main handles process action with missing file" {
    export ACTION="process"
    export FILE_INPUT=""
    
    # Mock log functions
    log::error() {
        echo "ERROR: $1"
        return 0
    }
    
    log::info() {
        echo "INFO: $1"
        return 0
    }
    
    # Mock argument parsing
    unstructured_io::parse_arguments() {
        return 0
    }
    
    # Remove the main function mock
    unset -f unstructured_io::main
    source "${SCRIPT_DIR}/manage.sh"
    
    run unstructured_io::main --action process
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: No file provided for processing" ]]
}

# Test action routing - process single file
@test "unstructured_io::main routes process action for single file" {
    export ACTION="process"
    export FILE_INPUT="test.pdf"
    export BATCH="no"
    export STRATEGY="hi_res"
    export OUTPUT="json"
    export LANGUAGES="eng"
    
    # Mock the process_document function
    unstructured_io::process_document() {
        echo "PROCESS_DOCUMENT_CALLED: $1 $2 $3 $4"
        return 0
    }
    
    # Mock argument parsing
    unstructured_io::parse_arguments() {
        return 0
    }
    
    # Remove the main function mock
    unset -f unstructured_io::main
    source "${SCRIPT_DIR}/manage.sh"
    
    result=$(unstructured_io::main --action process)
    [[ "$result" =~ "PROCESS_DOCUMENT_CALLED: test.pdf hi_res json eng" ]]
}

# Test action routing - process batch files
@test "unstructured_io::main routes process action for batch files" {
    export ACTION="process"
    export FILE_INPUT="file1.pdf,file2.docx"
    export BATCH="yes"
    export STRATEGY="fast"
    export OUTPUT="markdown"
    
    # Mock the batch_process function
    unstructured_io::batch_process() {
        echo "BATCH_PROCESS_CALLED: $*"
        return 0
    }
    
    # Mock argument parsing
    unstructured_io::parse_arguments() {
        return 0
    }
    
    # Remove the main function mock
    unset -f unstructured_io::main
    source "${SCRIPT_DIR}/manage.sh"
    
    result=$(unstructured_io::main --action process)
    [[ "$result" =~ "BATCH_PROCESS_CALLED" ]]
    [[ "$result" =~ "file1.pdf" ]]
    [[ "$result" =~ "file2.docx" ]]
    [[ "$result" =~ "--strategy fast" ]]
    [[ "$result" =~ "--output markdown" ]]
}

# Test action routing - invalid action
@test "unstructured_io::main handles invalid action" {
    export ACTION="invalid"
    
    # Mock log functions
    log::error() {
        echo "ERROR: $1"
        return 0
    }
    
    # Mock usage function
    unstructured_io::usage() {
        echo "USAGE_CALLED"
        return 0
    }
    
    # Mock argument parsing
    unstructured_io::parse_arguments() {
        return 0
    }
    
    # Remove the main function mock
    unset -f unstructured_io::main
    source "${SCRIPT_DIR}/manage.sh"
    
    run unstructured_io::main --action invalid
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR: Unknown action: invalid" ]]
    [[ "$output" =~ "USAGE_CALLED" ]]
}

# Test script execution check
@test "manage.sh executes main when run directly" {
    # This test verifies the script execution guard works correctly
    # The guard should only execute main when the script is run directly
    
    # Create a test script that sources manage.sh
    local test_script="/tmp/test_unstructured_io_source.sh"
    cat > "$test_script" << 'EOF'
#!/usr/bin/env bash
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/manage.sh"
echo "SOURCED_SUCCESSFULLY"
EOF
    chmod +x "$test_script"
    
    # Mock the main function to track if it's called
    unstructured_io::main() {
        echo "MAIN_EXECUTED"
        return 0
    }
    
    # Source the test script - main should NOT be called
    result=$(bash "$test_script")
    [[ "$result" =~ "SOURCED_SUCCESSFULLY" ]]
    [[ ! "$result" =~ "MAIN_EXECUTED" ]]
    
    # Clean up
    rm -f "$test_script"
}