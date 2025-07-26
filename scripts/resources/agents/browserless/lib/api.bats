#!/usr/bin/env bats
# Tests for Browserless api.sh functions

# Setup for each test
setup() {
    # Set test environment
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    export URL="https://example.com"
    export OUTPUT="test-output.png"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    browserless::export_config
    browserless::export_messages
    
    # Mock health check function
    browserless::is_healthy() {
        return 0  # Always healthy for API tests
    }
    
    # Mock logging functions
    log::header() { echo "HEADER: $*"; }
    log::info() { echo "INFO: $*"; }
    log::success() { echo "SUCCESS: $*"; }
    log::error() { echo "ERROR: $*"; }
    
    # Load API functions
    source "${SCRIPT_DIR}/api.sh"
}

# Test screenshot API function
@test "browserless::test_screenshot handles successful request" {
    # Mock successful curl
    curl() {
        if [[ "$*" == *"/chrome/screenshot"* ]]; then
            echo "mock screenshot data" > "$OUTPUT"
            return 0
        fi
        return 1
    }
    
    # Mock file operations
    du() {
        echo "15K	$OUTPUT"
    }
    
    run browserless::test_screenshot "https://test.com" "test.png"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Taking screenshot of: https://test.com"* ]]
    [[ "$output" == *"Screenshot saved to: test.png"* ]]
}

# Test screenshot API function with unhealthy service
@test "browserless::test_screenshot fails when service unhealthy" {
    # Override health check to fail
    browserless::is_healthy() {
        return 1
    }
    
    run browserless::test_screenshot
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Browserless is not running or healthy"* ]]
}

# Test PDF API function
@test "browserless::test_pdf handles successful request" {
    # Mock successful curl
    curl() {
        if [[ "$*" == *"/chrome/pdf"* ]]; then
            echo "mock pdf data" > "${OUTPUT:-document_test.pdf}"
            return 0
        fi
        return 1
    }
    
    # Mock file operations
    du() {
        echo "25K	${OUTPUT:-document_test.pdf}"
    }
    
    run browserless::test_pdf "https://test.com" "test.pdf"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Generating PDF from: https://test.com"* ]]
    [[ "$output" == *"PDF saved to: test.pdf"* ]]
}

# Test scraping API function
@test "browserless::test_scrape handles successful request" {
    # Mock successful curl
    curl() {
        if [[ "$*" == *"/chrome/content"* ]]; then
            echo "<html><body><h1>Test Content</h1></body></html>"
            return 0
        fi
        return 1
    }
    
    # Mock head command
    head() {
        if [[ "$1" == "-c" ]]; then
            echo "<html><body><h1>Test Content</h1></body></html>" | head -c "$2"
        fi
    }
    
    run browserless::test_scrape "https://test.com" "scrape.html"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Scraping content from: https://test.com"* ]]
    [[ "$output" == *"Content scraped successfully"* ]]
}

# Test pressure API function
@test "browserless::test_pressure handles successful request" {
    # Mock successful curl with JSON response
    curl() {
        if [[ "$*" == *"/pressure"* ]]; then
            echo '{"running":2,"queued":0,"maxConcurrent":5,"isAvailable":true,"cpu":0.15,"memory":0.3}'
            return 0
        fi
        return 1
    }
    
    # Mock jq command
    jq() {
        case "$1" in
            ".") echo '{"running":2,"queued":0,"maxConcurrent":5,"isAvailable":true,"cpu":0.15,"memory":0.3}' ;;
            "-r") 
                case "$2" in
                    ".running // 0") echo "2" ;;
                    ".queued // 0") echo "0" ;;
                    ".maxConcurrent // \"N/A\"") echo "5" ;;
                    ".isAvailable // false") echo "true" ;;
                    ".cpu // 0") echo "0.15" ;;
                    ".memory // 0") echo "0.3" ;;
                esac
                ;;
        esac
    }
    
    # Mock command check
    command() {
        if [[ "$1" == "-v" && "$2" == "jq" ]]; then
            return 0
        fi
        return 1
    }
    
    # Mock bc command
    bc() {
        echo "15"
    }
    
    run browserless::test_pressure
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Pool status retrieved"* ]]
    [[ "$output" == *"Running browsers: 2"* ]]
    [[ "$output" == *"Max concurrent: 5"* ]]
}

# Test function execution API
@test "browserless::test_function handles successful request" {
    # Mock successful curl
    curl() {
        if [[ "$*" == *"/chrome/function"* ]]; then
            echo '{"title":"Test Page","url":"https://test.com","viewport":{"width":1280,"height":720}}'
            return 0
        fi
        return 1
    }
    
    # Mock jq command
    jq() {
        echo '{"title":"Test Page","url":"https://test.com","viewport":{"width":1280,"height":720}}'
    }
    
    run browserless::test_function "https://test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Executing custom function on: https://test.com"* ]]
    [[ "$output" == *"Function executed successfully"* ]]
}

# Test function execution failure
@test "browserless::test_function handles curl failure" {
    # Mock failed curl
    curl() {
        return 1
    }
    
    run browserless::test_function "https://test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Failed to execute function"* ]]
}

# Test all APIs function
@test "browserless::test_all_apis coordinates multiple tests" {
    # Mock mkdir and cd
    mkdir() { return 0; }
    cd() { return 0; }
    pwd() { echo "/tmp/browserless_test_20231201_120000"; }
    
    # Mock individual test functions
    browserless::test_pressure() { 
        echo "Pressure test passed"
        return 0
    }
    browserless::test_screenshot() {
        echo "Screenshot test passed"
        return 0
    }
    browserless::test_pdf() {
        echo "PDF test passed"
        return 0
    }
    browserless::test_scrape() {
        echo "Scrape test passed"
        return 0
    }
    browserless::test_function() {
        echo "Function test passed"
        return 0
    }
    
    # Mock sleep
    sleep() { return 0; }
    
    run browserless::test_all_apis "https://test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All tests completed successfully"* ]]
    [[ "$output" == *"Pressure test passed"* ]]
    [[ "$output" == *"Screenshot test passed"* ]]
}

# Test all APIs with some failures
@test "browserless::test_all_apis handles partial failures" {
    # Mock mkdir and cd
    mkdir() { return 0; }
    cd() { return 0; }
    pwd() { echo "/tmp/browserless_test_20231201_120000"; }
    
    # Mock individual test functions with some failures
    browserless::test_pressure() { return 0; }
    browserless::test_screenshot() { return 1; }  # Fail
    browserless::test_pdf() { return 0; }
    browserless::test_scrape() { return 1; }      # Fail
    browserless::test_function() { return 0; }
    
    # Mock sleep and log
    sleep() { return 0; }
    log::warn() { echo "WARN: $*"; }
    
    run browserless::test_all_apis "https://test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"2 test(s) failed"* ]]
}