#!/usr/bin/env bats
# Tests for Browserless usage.sh functions

# Expensive setup operations run once per file
setup_file() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Load dependencies once per file
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    BROWSERLESS_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and messages once
    source "${BROWSERLESS_DIR}/config/defaults.sh"
    source "${BROWSERLESS_DIR}/config/messages.sh"
    
    # Load usage functions once
    source "${SCRIPT_DIR}/usage.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment (lightweight per-test)
    export BROWSERLESS_CUSTOM_PORT="9999"
    export BROWSERLESS_BASE_URL="http://localhost:9999"
    export USAGE_TYPE="help"
    export URL="https://example.com"
    export OUTPUT="/tmp/browserless-test-output.png"
    
    # Export config functions (lightweight)
    browserless::export_config
    browserless::export_messages
    
    # Mock API test functions (lightweight)
    browserless::test_screenshot() {
        echo "MOCK: Screenshot test with $1 $2"
        return 0
    }
    
    browserless::test_pdf() {
        echo "MOCK: PDF test with $1 $2"
        return 0
    }
    
    browserless::test_scrape() {
        echo "MOCK: Scrape test with $1 $2"
        return 0
    }
    
    browserless::test_pressure() {
        echo "MOCK: Pressure test"
        return 0
    }
    
    browserless::test_function() {
        echo "MOCK: Function test with $1"
        return 0
    }
    
    browserless::test_all_apis() {
        echo "MOCK: All APIs test with $1"
        return 0
    }
    
    # Export all mock functions
    export -f browserless::test_screenshot browserless::test_pdf browserless::test_scrape
    export -f browserless::test_pressure browserless::test_function browserless::test_all_apis
}

# Test usage help display
@test "browserless::show_usage_help displays comprehensive help information" {
    run browserless::show_usage_help
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless Usage Examples"* ]]
    [[ "$output" == *"Available usage examples:"* ]]
    [[ "$output" == *"--usage-type screenshot"* ]]
    [[ "$output" == *"--usage-type pdf"* ]]
    [[ "$output" == *"--usage-type scrape"* ]]
    [[ "$output" == *"--usage-type function"* ]]
    [[ "$output" == *"--usage-type pressure"* ]]
    [[ "$output" == *"--usage-type all"* ]]
    [[ "$output" == *"Advanced Examples:"* ]]
    [[ "$output" == *"Screenshot with mobile viewport"* ]]
    [[ "$output" == *"PDF with custom margins"* ]]
    [[ "$output" == *"Advanced scraping with selectors"* ]]
    [[ "$output" == *"https://www.browserless.io/docs/"* ]]
}

# Test usage help contains proper base URL
@test "browserless::show_usage_help uses correct base URL in examples" {
    run browserless::show_usage_help
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"$BROWSERLESS_BASE_URL/chrome/screenshot"* ]]
    [[ "$output" == *"$BROWSERLESS_BASE_URL/chrome/pdf"* ]]
    [[ "$output" == *"$BROWSERLESS_BASE_URL/chrome/scrape"* ]]
}

# Test usage example routing - screenshot
@test "browserless::run_usage_example routes screenshot correctly" {
    run browserless::run_usage_example "screenshot"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"MOCK: Screenshot test with $URL $OUTPUT"* ]]
}

# Test usage example routing - pdf
@test "browserless::run_usage_example routes pdf correctly" {
    run browserless::run_usage_example "pdf"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"MOCK: PDF test with $URL $OUTPUT"* ]]
}

# Test usage example routing - scrape
@test "browserless::run_usage_example routes scrape correctly" {
    run browserless::run_usage_example "scrape"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"MOCK: Scrape test with $URL $OUTPUT"* ]]
}

# Test usage example routing - pressure
@test "browserless::run_usage_example routes pressure correctly" {
    run browserless::run_usage_example "pressure"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"MOCK: Pressure test"* ]]
}

# Test usage example routing - function
@test "browserless::run_usage_example routes function correctly" {
    run browserless::run_usage_example "function"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"MOCK: Function test with $URL"* ]]
}

# Test usage example routing - all
@test "browserless::run_usage_example routes all correctly" {
    run browserless::run_usage_example "all"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"MOCK: All APIs test with $URL"* ]]
}

# Test usage example routing - help (default)
@test "browserless::run_usage_example defaults to help for unknown types" {
    run browserless::run_usage_example "unknown"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless Usage Examples"* ]]
}

# Test usage example routing with USAGE_TYPE variable
@test "browserless::run_usage_example uses USAGE_TYPE variable when no argument" {
    export USAGE_TYPE="pressure"
    
    run browserless::run_usage_example
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"MOCK: Pressure test"* ]]
}

# Test API reference display
@test "browserless::show_api_reference displays complete API documentation" {
    run browserless::show_api_reference
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless API Quick Reference"* ]]
    [[ "$output" == *"Base URL: $BROWSERLESS_BASE_URL"* ]]
    [[ "$output" == *"Core Endpoints:"* ]]
    [[ "$output" == *"POST /chrome/screenshot"* ]]
    [[ "$output" == *"POST /chrome/pdf"* ]]
    [[ "$output" == *"POST /chrome/content"* ]]
    [[ "$output" == *"POST /chrome/function"* ]]
    [[ "$output" == *"POST /chrome/scrape"* ]]
    [[ "$output" == *"GET  /pressure"* ]]
    [[ "$output" == *"GET  /metrics"* ]]
    [[ "$output" == *"Screenshot Options:"* ]]
    [[ "$output" == *"PDF Options:"* ]]
    [[ "$output" == *"Content Options:"* ]]
    [[ "$output" == *"Browser Pool Status"* ]]
    [[ "$output" == *"Error Responses:"* ]]
    [[ "$output" == *"400: Bad Request"* ]]
    [[ "$output" == *"408: Request Timeout"* ]]
    [[ "$output" == *"429: Too Many Requests"* ]]
    [[ "$output" == *"500: Internal Server Error"* ]]
}

# Test cURL examples display
@test "browserless::show_curl_examples displays working curl commands" {
    run browserless::show_curl_examples
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless cURL Examples"* ]]
    [[ "$output" == *"# Basic screenshot"* ]]
    [[ "$output" == *"curl -X POST $BROWSERLESS_BASE_URL/chrome/screenshot"* ]]
    [[ "$output" == *"# Full page screenshot"* ]]
    [[ "$output" == *"\"fullPage\": true"* ]]
    [[ "$output" == *"# Generate PDF"* ]]
    [[ "$output" == *"curl -X POST $BROWSERLESS_BASE_URL/chrome/pdf"* ]]
    [[ "$output" == *"# Get page content"* ]]
    [[ "$output" == *"curl -X POST $BROWSERLESS_BASE_URL/chrome/content"* ]]
    [[ "$output" == *"# Check service status"* ]]
    [[ "$output" == *"curl $BROWSERLESS_BASE_URL/pressure"* ]]
    [[ "$output" == *"# Execute custom function"* ]]
    [[ "$output" == *"curl -X POST $BROWSERLESS_BASE_URL/chrome/function"* ]]
    [[ "$output" == *"# Scrape with selectors"* ]]
    [[ "$output" == *"curl -X POST $BROWSERLESS_BASE_URL/chrome/scrape"* ]]
}

# Test that curl examples use correct base URL
@test "browserless::show_curl_examples uses configured base URL" {
    # Change base URL for this test
    export BROWSERLESS_BASE_URL="http://localhost:8888"
    
    run browserless::show_curl_examples
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"http://localhost:8888/chrome/screenshot"* ]]
    [[ "$output" == *"http://localhost:8888/chrome/pdf"* ]]
    [[ "$output" == *"http://localhost:8888/pressure"* ]]
}

# Test routing with environment variables
@test "browserless::run_usage_example uses environment variables for parameters" {
    export URL="https://custom-test.com"
    export OUTPUT="custom-output.png"
    
    run browserless::run_usage_example "screenshot"
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"https://custom-test.com"* ]]
    [[ "$output" == *"custom-output.png"* ]]
}

# Test API reference includes essential information
@test "browserless::show_api_reference includes all essential API details" {
    run browserless::show_api_reference
    
    [ "$status" -eq 0 ]
    # Check for essential endpoint information
    [[ "$output" == *"Capture page screenshots"* ]]
    [[ "$output" == *"Generate PDF documents"* ]]
    [[ "$output" == *"Extract page content"* ]]
    [[ "$output" == *"Execute custom Puppeteer code"* ]]
    [[ "$output" == *"Extract structured data"* ]]
    [[ "$output" == *"Check browser pool status"* ]]
    [[ "$output" == *"Prometheus metrics"* ]]
    # Check for option details
    [[ "$output" == *"fullPage: true/false"* ]]
    [[ "$output" == *"format: \"A4\""* ]]
    [[ "$output" == *"waitUntil:"* ]]
    # Check for status information
    [[ "$output" == *"running: current active sessions"* ]]
    [[ "$output" == *"maxConcurrent: maximum sessions"* ]]
}

# Test help routing when no usage type specified
@test "browserless::run_usage_example shows help when USAGE_TYPE is help" {
    export USAGE_TYPE="help"
    
    run browserless::run_usage_example
    
    [ "$status" -eq 0 ]
    [[ "$output" == *"Browserless Usage Examples"* ]]
    [[ "$output" == *"Available usage examples:"* ]]
}