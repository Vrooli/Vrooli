#!/usr/bin/env bats
# Tests for Browserless api.sh functions

# Expensive setup operations run once per file
setup_file() {
    # Load dependencies once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    
    # Load API functions once
    source "${SCRIPT_DIR}/api.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_BROWSERLESS_DIR="$BROWSERLESS_DIR"
}

# Lightweight per-test setup
setup() {
    # Use paths from setup_file
    # Mock resources functions to avoid hang
    declare -A DEFAULT_PORTS=(
        ["ollama"]="11434"
        ["agent-s2"]="4113"
        ["browserless"]="3000"
        ["unstructured-io"]="8000"
        ["n8n"]="5678"
        ["node-red"]="1880"
        ["huginn"]="3000"
        ["windmill"]="8000"
        ["judge0"]="2358"
        ["searxng"]="8080"
        ["qdrant"]="6333"
        ["questdb"]="9000"
        ["vault"]="8200"
    )
    resources::get_default_port() { echo "${DEFAULT_PORTS[$1]:-8080}"; }
    export -f resources::get_default_port
    
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    BROWSERLESS_DIR="${SETUP_FILE_BROWSERLESS_DIR}"
    
    # Set test environment (lightweight per-test)
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_CONTAINER_NAME="browserless-test"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    export URL="https://example.com"
    export OUTPUT="test-output.png"
    
    # Basic mock functions (lightweight)
    mock::network::set_online() { return 0; }
    setup_standard_mocks() { 
        export FORCE="${FORCE:-no}"
        export YES="${YES:-no}"
        export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
        export QUIET="${QUIET:-no}"
        mock::network::set_online
    }
    
    # Setup mocks
    setup_standard_mocks
    
    # Re-source config to ensure export functions are available
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    
    # Re-source only essential API functions
    source "${SCRIPT_DIR}/api.sh"
    
    # Export config functions  
    browserless::export_config
    browserless::export_messages
    
    # Mock health check function
    browserless::is_healthy() {
        return 0  # Always healthy for API tests
    }
    export -f browserless::is_healthy
}

# Test screenshot API function
@test "browserless::test_screenshot handles successful request" {
    # Mock successful curl with proper status code handling
    curl() {
        if [[ "$*" == *"/chrome/screenshot"* ]]; then
            # Handle --write-out for HTTP status
            if [[ "$*" == *"--write-out"* ]]; then
                # Create a mock PNG file (at least 1KB)
                printf '\x89PNG\r\n\x1a\n' > "/tmp/browserless_screenshot_$$"
                dd if=/dev/zero bs=1024 count=2 >> "/tmp/browserless_screenshot_$$" 2>/dev/null
                echo "200"  # Return HTTP 200 status
                return 0
            fi
            return 0
        fi
        return 1
    }
    
    # Mock file operations
    du() {
        echo "2K	$1"
    }
    
    # Mock file command for MIME type detection
    file() {
        if [[ "$*" == *"--mime-type"* ]]; then
            echo "image/png"
            return 0
        fi
        return 0
    }
    
    # Mock stat for file size check
    stat() {
        echo "2048"  # 2KB
        return 0
    }
    
    # Mock mv command
    mv() {
        touch "$2"  # Create the target file
        return 0
    }
    
    run browserless::test_screenshot "https://test.com" "test.png"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Taking screenshot of: https://test.com"* ]]
    [[ "$output" == *"Screenshot saved to: test.png"* ]]
    [[ "$output" == *"Validated as:"* ]]
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

# Test screenshot validation - HTTP 500 error
@test "browserless::test_screenshot handles HTTP 500 error" {
    curl() {
        if [[ "$*" == *"--write-out"* ]]; then
            echo "Internal Server Error" > "/tmp/browserless_screenshot_$$"
            echo "500"  # Return HTTP 500 status
            return 0
        fi
        return 0
    }
    
    run browserless::test_screenshot "https://test.com" "test.png"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Screenshot request failed with HTTP status: 500"* ]]
}

# Test screenshot validation - small file (error text)
@test "browserless::test_screenshot rejects small files" {
    curl() {
        if [[ "$*" == *"--write-out"* ]]; then
            echo "Not Found" > "/tmp/browserless_screenshot_$$"
            echo "200"
            return 0
        fi
        return 0
    }
    
    stat() {
        echo "9"  # 9 bytes - too small
    }
    
    file() {
        if [[ "$*" == *"--mime-type"* ]]; then
            echo "text/plain"  # Will be rejected as non-image
            return 0
        fi
        return 0
    }
    
    run browserless::test_screenshot "https://test.com" "test.png"
    
    [ "$status" -eq 1 ]
    # Could fail on either check - file type or size
    [[ "$output" == *"Response is not an image"* ]] || [[ "$output" == *"Screenshot file too small"* ]]
}

# Test screenshot validation - non-image MIME type
@test "browserless::test_screenshot rejects non-image files" {
    curl() {
        if [[ "$*" == *"--write-out"* ]]; then
            echo "<!DOCTYPE html><html><body>Error</body></html>" > "/tmp/browserless_screenshot_$$"
            echo "200"
            return 0
        fi
        return 0
    }
    
    stat() {
        echo "2048"  # Large enough
    }
    
    file() {
        if [[ "$*" == *"--mime-type"* ]]; then
            echo "text/html"  # Not an image
            return 0
        fi
        return 0
    }
    
    run browserless::test_screenshot "https://test.com" "test.png"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"Response is not an image"* ]]
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
    
    # Mock head command - simplified to avoid hanging
    head() {
        echo "<html><body><h1>Test Conte"
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
    
    # Override individual test functions to avoid conflicts
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
    
    # Mock sleep and date
    sleep() { return 0; }
    date() { echo "20231201_120000"; }
    
    run browserless::test_all_apis "https://test.com"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"All tests completed successfully"* ]]
}

# Test all APIs with some failures
@test "browserless::test_all_apis handles partial failures" {
    # Mock mkdir and cd
    mkdir() { return 0; }
    cd() { return 0; }
    pwd() { echo "/tmp/browserless_test_20231201_120000"; }
    
    # Override individual test functions with some failures
    browserless::test_pressure() { return 0; }
    browserless::test_screenshot() { return 1; }  # Fail
    browserless::test_pdf() { return 0; }
    browserless::test_scrape() { return 1; }      # Fail
    browserless::test_function() { return 0; }
    
    # Mock sleep, date and log
    sleep() { return 0; }
    date() { echo "20231201_120000"; }
    log::warn() { echo "WARN: $*"; }
    
    run browserless::test_all_apis "https://test.com"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"2 test(s) failed"* ]]
}

# Test safe screenshot function
@test "browserless::safe_screenshot validates output before allowing read" {
    # Mock browserless::test_screenshot to succeed
    browserless::test_screenshot() {
        touch "$2"
        printf '\x89PNG\r\n\x1a\n' > "$2"
        dd if=/dev/zero bs=1024 count=2 >> "$2" 2>/dev/null
        return 0
    }
    
    # Mock file command
    file() {
        if [[ "$*" == *"--mime-type"* ]]; then
            echo "image/png"
            return 0
        fi
        return 0
    }
    
    # Mock stat
    stat() {
        echo "2048"
        return 0
    }
    
    run browserless::safe_screenshot "https://test.com" "/tmp/safe_test.png"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Screenshot validated and safe to read"* ]]
}

# Test safe screenshot rejects invalid files
@test "browserless::safe_screenshot rejects invalid files" {
    # Mock browserless::test_screenshot to create small text file
    browserless::test_screenshot() {
        echo "Not Found" > "$2"
        return 0  # Pretend it succeeded
    }
    
    # Mock stat for small file
    stat() {
        echo "9"
        return 0
    }
    
    run browserless::safe_screenshot "https://test.com" "/tmp/unsafe_test.png"
    
    [ "$status" -eq 1 ]
    [[ "$output" == *"suspiciously small"* ]]
}
