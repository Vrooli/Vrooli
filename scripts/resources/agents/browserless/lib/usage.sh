#!/usr/bin/env bash
# Browserless Usage Examples and Help
# User guidance and example coordination

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Source API functions for test functions
BROWSERLESS_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/api.sh" 2>/dev/null || true

#######################################
# Show usage examples help
#######################################
browserless::show_usage_help() {
    log::header "🎭 Browserless Usage Examples"
    echo
    echo "Available usage examples:"
    echo "  $0 --action usage --usage-type screenshot [url] [output]  # Test screenshot API"
    echo "  $0 --action usage --usage-type pdf [url] [output]         # Test PDF generation"
    echo "  $0 --action usage --usage-type scrape [url]               # Test web scraping"
    echo "  $0 --action usage --usage-type function [url]             # Test function execution"
    echo "  $0 --action usage --usage-type pressure                   # Check pool status"
    echo "  $0 --action usage --usage-type all [url]                  # Run all examples"
    echo
    echo "Examples:"
    echo "  $0 --action usage --usage-type screenshot --url https://github.com --output github.png"
    echo "  $0 --action usage --usage-type pdf --url https://wikipedia.org --output wiki.pdf"
    echo "  $0 --action usage --usage-type scrape --url https://news.ycombinator.com"
    echo "  $0 --action usage --usage-type all --url https://example.com"
    echo
    echo "Advanced Examples:"
    echo
    echo "  # Screenshot with mobile viewport"
    echo "  curl -X POST $BROWSERLESS_BASE_URL/chrome/screenshot \\"
    echo "    -H \"Content-Type: application/json\" \\"
    echo "    -d '{"
    echo "      \"url\": \"https://example.com\","
    echo "      \"viewport\": {"
    echo "        \"width\": 375,"
    echo "        \"height\": 812,"
    echo "        \"deviceScaleFactor\": 3,"
    echo "        \"isMobile\": true"
    echo "      }"
    echo "    }' --output mobile.png"
    echo
    echo "  # PDF with custom margins"
    echo "  curl -X POST $BROWSERLESS_BASE_URL/chrome/pdf \\"
    echo "    -H \"Content-Type: application/json\" \\"
    echo "    -d '{"
    echo "      \"url\": \"https://example.com\","
    echo "      \"options\": {"
    echo "        \"format\": \"A4\","
    echo "        \"displayHeaderFooter\": true,"
    echo "        \"headerTemplate\": \"<div>Header</div>\","
    echo "        \"footerTemplate\": \"<div>Page <span class=\\\"pageNumber\\\"></span></div>\""
    echo "      }"
    echo "    }' --output report.pdf"
    echo
    echo "  # Advanced scraping with selectors"
    echo "  curl -X POST $BROWSERLESS_BASE_URL/chrome/scrape \\"
    echo "    -H \"Content-Type: application/json\" \\"
    echo "    -d '{"
    echo "      \"url\": \"https://example.com\","
    echo "      \"elements\": [{"
    echo "        \"selector\": \"h1\","
    echo "        \"property\": \"textContent\""
    echo "      }]"
    echo "    }'"
    echo
    echo "For comprehensive API documentation, visit:"
    echo "https://www.browserless.io/docs/"
}

#######################################
# Route usage example requests to appropriate functions
# Arguments:
#   $1 - Usage type (screenshot, pdf, scrape, pressure, function, all, help)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::run_usage_example() {
    local usage_type="${1:-${USAGE_TYPE:-help}}"
    
    # Track files to clean up (only default output files, not user-specified ones)
    local cleanup_files=()
    local should_cleanup=false
    
    # If OUTPUT is not set by user, we'll clean up the default files
    if [[ -z "${OUTPUT:-}" ]]; then
        should_cleanup=true
        # Set up cleanup trap
        trap 'browserless::cleanup_usage_files "${cleanup_files[@]}"' EXIT
    fi
    
    case "$usage_type" in
        "screenshot")
            if [[ "$should_cleanup" == "true" ]]; then
                cleanup_files+=("${BROWSERLESS_TEST_OUTPUT_DIR:-/tmp/browserless-test-outputs}/screenshot_test.png")
            fi
            browserless::test_screenshot "$URL" "$OUTPUT"
            ;;
        "pdf")
            if [[ "$should_cleanup" == "true" ]]; then
                cleanup_files+=("${BROWSERLESS_TEST_OUTPUT_DIR:-/tmp/browserless-test-outputs}/document_test.pdf")
            fi
            browserless::test_pdf "$URL" "$OUTPUT"
            ;;
        "scrape")
            if [[ "$should_cleanup" == "true" ]]; then
                cleanup_files+=("${BROWSERLESS_TEST_OUTPUT_DIR:-/tmp/browserless-test-outputs}/scrape_test.html")
            fi
            browserless::test_scrape "$URL" "$OUTPUT"
            ;;
        "pressure")
            browserless::test_pressure
            ;;
        "function")
            browserless::test_function "$URL"
            ;;
        "all")
            if [[ "$should_cleanup" == "true" ]]; then
                cleanup_files+=("${BROWSERLESS_TEST_OUTPUT_DIR:-/tmp/browserless-test-outputs}/screenshot_test.png")
                cleanup_files+=("${BROWSERLESS_TEST_OUTPUT_DIR:-/tmp/browserless-test-outputs}/document_test.pdf") 
                cleanup_files+=("${BROWSERLESS_TEST_OUTPUT_DIR:-/tmp/browserless-test-outputs}/scrape_test.html")
            fi
            browserless::test_all_apis "$URL"
            ;;
        "help"|*)
            browserless::show_usage_help
            ;;
    esac
}

#######################################
# Clean up usage example files
# Arguments:
#   $@ - List of files to clean up
#######################################
browserless::cleanup_usage_files() {
    local files=("$@")
    
    if [[ ${#files[@]} -eq 0 ]]; then
        return 0
    fi
    
    log::debug "Cleaning up browserless usage example files..."
    
    for file in "${files[@]}"; do
        if [[ -f "$file" ]]; then
            if trash::safe_remove "$file" --temp 2>/dev/null; then
                log::debug "✓ Cleaned up: $file"
            else
                log::debug "⚠ Failed to clean up: $file"
            fi
        fi
    done
}

#######################################
# Show quick API reference
#######################################
browserless::show_api_reference() {
    cat << EOF
=== Browserless API Quick Reference ===

Base URL: $BROWSERLESS_BASE_URL

Core Endpoints:
┌─────────────────────────────────────────────────────────────────┐
│ POST /chrome/screenshot  - Capture page screenshots             │
│ POST /chrome/pdf         - Generate PDF documents               │
│ POST /chrome/content     - Extract page content                 │
│ POST /chrome/function    - Execute custom Puppeteer code        │
│ POST /chrome/scrape      - Extract structured data              │
│ GET  /pressure           - Check browser pool status            │
│ GET  /metrics            - Prometheus metrics                   │
└─────────────────────────────────────────────────────────────────┘

Common Request Format:
{
  "url": "https://example.com",
  "options": {
    // Endpoint-specific options
  }
}

Screenshot Options:
- fullPage: true/false
- type: "png" | "jpeg"
- quality: 0-100 (for jpeg)
- clip: { x, y, width, height }

PDF Options:
- format: "A4" | "A3" | "Letter" | etc.
- landscape: true/false
- printBackground: true/false
- margin: { top, bottom, left, right }

Content Options:
- waitUntil: "load" | "domcontentloaded" | "networkidle0" | "networkidle2"
- timeout: milliseconds

Browser Pool Status (/pressure):
- running: current active sessions
- queued: pending requests
- maxConcurrent: maximum sessions
- isAvailable: accepting requests
- cpu/memory: resource usage (0-1)

Error Responses:
- 400: Bad Request (invalid parameters)
- 408: Request Timeout
- 429: Too Many Requests (pool full)
- 500: Internal Server Error

For detailed documentation: https://www.browserless.io/docs/
EOF
}

#######################################
# Show example curl commands for common tasks
#######################################
browserless::show_curl_examples() {
    cat << EOF
=== Browserless cURL Examples ===

# Basic screenshot
curl -X POST $BROWSERLESS_BASE_URL/chrome/screenshot \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}' \\
  --output screenshot.png

# Full page screenshot
curl -X POST $BROWSERLESS_BASE_URL/chrome/screenshot \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com", "options": {"fullPage": true}}' \\
  --output fullpage.png

# Generate PDF
curl -X POST $BROWSERLESS_BASE_URL/chrome/pdf \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}' \\
  --output document.pdf

# Get page content
curl -X POST $BROWSERLESS_BASE_URL/chrome/content \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}' | jq .

# Check service status
curl $BROWSERLESS_BASE_URL/pressure | jq .

# Execute custom function
curl -X POST $BROWSERLESS_BASE_URL/chrome/function \\
  -H "Content-Type: application/json" \\
  -d '{
    "code": "async ({ page }) => {
      await page.goto(\"https://example.com\");
      return await page.title();
    }"
  }'

# Scrape with selectors
curl -X POST $BROWSERLESS_BASE_URL/chrome/scrape \\
  -H "Content-Type: application/json" \\
  -d '{
    "url": "https://example.com",
    "elements": [{
      "selector": "h1",
      "property": "textContent"
    }]
  }'
EOF
}

#######################################
# Test Function Implementations
# These functions provide actual API testing functionality
#######################################

#######################################
# Screenshot helper function
#######################################
browserless::screenshot() {
    local url="${1:-http://httpbin.org/html}"
    local output="${2:-/tmp/screenshot.png}"
    
    curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/screenshot" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$url\"}" \
        -o "$output"
}

#######################################
# PDF helper function
#######################################
browserless::pdf() {
    local url="${1:-http://httpbin.org/html}"
    local output="${2:-/tmp/document.pdf}"
    
    curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/chrome/pdf" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$url\"}" \
        -o "$output"
}

#######################################
# Scrape helper function
#######################################
browserless::scrape() {
    local url="${1:-http://httpbin.org/html}"
    local selector="${2:-}"
    
    local data="{\"url\":\"$url\""
    if [[ -n "$selector" ]]; then
        data+=",\"selector\":\"$selector\""
    fi
    data+="}"
    
    curl -s -X POST \
        "http://localhost:$BROWSERLESS_PORT/chrome/content" \
        -H "Content-Type: application/json" \
        -d "$data"
}

#######################################
# Pressure test helper function
#######################################
browserless::pressure_test() {
    local pressure_data
    pressure_data=$(curl -s "http://localhost:$BROWSERLESS_PORT/pressure")
    
    if [[ -n "$pressure_data" ]]; then
        echo "Browser Pressure Status:"
        echo "$pressure_data" | jq '.' 2>/dev/null || echo "$pressure_data"
        return 0
    else
        return 1
    fi
}

# Note: Test functions browserless::test_screenshot, browserless::test_pdf, 
# browserless::test_scrape, browserless::test_pressure, and browserless::test_function
# are now provided by api.sh which is sourced above

#######################################
# Test all APIs
#######################################
browserless::test_all_apis() {
    local url="${1:-${URL:-http://httpbin.org/html}}"
    
    log::header "🧪 Testing all Browserless APIs"
    
    local failed_tests=0
    
    # Test screenshot
    if ! browserless::test_screenshot "$url"; then
        ((failed_tests++))
    fi
    
    echo
    
    # Test PDF
    if ! browserless::test_pdf "$url"; then
        ((failed_tests++))
    fi
    
    echo
    
    # Test scraping
    if ! browserless::test_scrape "$url"; then
        ((failed_tests++))
    fi
    
    echo
    
    # Test pressure
    if ! browserless::test_pressure; then
        ((failed_tests++))
    fi
    
    echo
    
    # Test function
    if ! browserless::test_function "$url"; then
        ((failed_tests++))
    fi
    
    echo
    log::header "🏁 Test Summary"
    
    if [[ $failed_tests -eq 0 ]]; then
        log::success "✅ All API tests passed!"
    else
        log::error "❌ $failed_tests test(s) failed"
        return 1
    fi
}

# Export functions for subshell availability
export -f browserless::show_usage_help
export -f browserless::run_usage_example
export -f browserless::show_api_reference
export -f browserless::show_curl_examples
# Test functions are exported by api.sh
export -f browserless::test_all_apis