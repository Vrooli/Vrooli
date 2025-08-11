#!/usr/bin/env bash
# Browserless API Functions
# API testing, examples, and usage demonstrations

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test output directory configuration
BROWSERLESS_TEST_OUTPUT_DIR="${BROWSERLESS_TEST_OUTPUT_DIR:-/tmp/browserless-test-outputs}"

# Ensure test output directory exists
browserless::ensure_test_output_dir() {
    if [[ ! -d "$BROWSERLESS_TEST_OUTPUT_DIR" ]]; then
        mkdir -p "$BROWSERLESS_TEST_OUTPUT_DIR"
        log::debug "Created test output directory: $BROWSERLESS_TEST_OUTPUT_DIR"
    fi
}

#######################################
# Test screenshot API endpoint
# Arguments:
#   $1 - URL to screenshot (optional, uses URL var)
#   $2 - Output filename (optional, uses OUTPUT var)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::test_screenshot() {
    local test_url="${1:-${URL:-https://example.com}}"
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    local output_file="${2:-${OUTPUT:-$BROWSERLESS_TEST_OUTPUT_DIR/screenshot_test.png}}"
    local temp_file="/tmp/browserless_screenshot_$$"
    
    log::header "${MSG_USAGE_SCREENSHOT}"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    log::info "Taking screenshot of: $test_url"
    log::info "Output file: $output_file"
    
    # First save to temp file and capture HTTP status
    local http_status
    http_status=$(curl -X POST "$BROWSERLESS_BASE_URL/chrome/screenshot" \
        -H "Content-Type: application/json" \
        -d "{\"url\": \"$test_url\", \"options\": {\"fullPage\": true}}" \
        --output "$temp_file" \
        --write-out "%{http_code}" \
        --silent \
        --show-error \
        --max-time 60)
    
    local curl_exit_code=$?
    
    # Check curl succeeded
    if [[ $curl_exit_code -ne 0 ]]; then
        log::error "Failed to connect to browserless service (exit code: $curl_exit_code)"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # Check HTTP status
    if [[ "$http_status" != "200" ]]; then
        log::error "Screenshot request failed with HTTP status: $http_status"
        if [[ -f "$temp_file" ]]; then
            log::debug "Error response: $(head -c 200 "$temp_file" 2>/dev/null || echo 'No response body')"
            trash::safe_remove "$temp_file" --temp
        fi
        return 1
    fi
    
    # Validate file exists and has content
    if [[ ! -f "$temp_file" ]] || [[ ! -s "$temp_file" ]]; then
        log::error "No screenshot data received"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # Check if it's actually an image using file command
    if command -v file &> /dev/null; then
        local file_type
        file_type=$(file -b --mime-type "$temp_file" 2>/dev/null || echo "unknown")
        if [[ "$file_type" != image/* ]]; then
            log::error "Response is not an image (detected: $file_type)"
            log::debug "Response preview: $(head -c 100 "$temp_file" 2>/dev/null || echo 'Cannot read file')"
            trash::safe_remove "$temp_file" --temp
            return 1
        fi
    fi
    
    # Check minimum file size (real screenshots should be at least a few KB)
    local file_size
    file_size=$(stat -f%z "$temp_file" 2>/dev/null || stat -c%s "$temp_file" 2>/dev/null || echo "0")
    if [[ "$file_size" -lt 1024 ]]; then
        log::error "Screenshot file too small ($file_size bytes) - likely an error message"
        log::debug "Content: $(cat "$temp_file" 2>/dev/null || echo 'Cannot read file')"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # All checks passed, move to final location
    if mv "$temp_file" "$output_file"; then
        log::success "✓ Screenshot saved to: $output_file"
        log::info "File size: $(du -h "$output_file" | cut -f1)"
        log::info "Validated as: $(file -b --mime-type "$output_file" 2>/dev/null || echo 'image file')"
        return 0
    else
        log::error "Failed to save screenshot to: $output_file"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
}

#######################################
# Safe screenshot capture for AI analysis
# Ensures output is valid before allowing reads
# Arguments:
#   $1 - URL to screenshot
#   $2 - Output filename
# Returns: 0 if successful and safe to read, 1 if failed
#######################################
browserless::safe_screenshot() {
    local url="${1:?URL required}"
    local output="${2:?Output filename required}"
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    # Use the validated screenshot function
    if ! browserless::test_screenshot "$url" "$output"; then
        log::error "Screenshot capture failed for: $url"
        return 1
    fi
    
    # Double-check the file is safe to read
    if [[ ! -f "$output" ]]; then
        log::error "Screenshot file not created: $output"
        return 1
    fi
    
    local file_size
    file_size=$(stat -f%z "$output" 2>/dev/null || stat -c%s "$output" 2>/dev/null || echo "0")
    if [[ "$file_size" -lt 1024 ]]; then
        log::error "Screenshot file suspiciously small - removing for safety"
        trash::safe_remove "$output" --temp
        return 1
    fi
    
    # Final validation - ensure it's an image
    if command -v file &> /dev/null; then
        local mime_type
        mime_type=$(file -b --mime-type "$output" 2>/dev/null || echo "unknown")
        if [[ "$mime_type" != image/* ]]; then
            log::error "File is not an image (mime: $mime_type) - removing for safety"
            trash::safe_remove "$output" --temp
            return 1
        fi
    fi
    
    log::success "Screenshot validated and safe to read: $output"
    return 0
}

#######################################
# Test PDF generation API endpoint
# Arguments:
#   $1 - URL to convert (optional, uses URL var)
#   $2 - Output filename (optional, uses OUTPUT var)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::test_pdf() {
    local test_url="${1:-${URL:-https://example.com}}"
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    local output_file="${2:-${OUTPUT:-$BROWSERLESS_TEST_OUTPUT_DIR/document_test.pdf}}"
    
    log::header "${MSG_USAGE_PDF}"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    log::info "Generating PDF from: $test_url"
    log::info "Output file: $output_file"
    
    if curl -X POST "$BROWSERLESS_BASE_URL/chrome/pdf" \
        -H "Content-Type: application/json" \
        -d "{
            \"url\": \"$test_url\",
            \"options\": {
                \"format\": \"A4\",
                \"printBackground\": true,
                \"margin\": {
                    \"top\": \"1cm\",
                    \"bottom\": \"1cm\",
                    \"left\": \"1cm\",
                    \"right\": \"1cm\"
                }
            }
        }" \
        --output "$output_file" \
        --silent \
        --show-error \
        --max-time 60; then
        
        log::success "✓ PDF saved to: $output_file"
        if [[ -f "$output_file" ]]; then
            log::info "File size: $(du -h "$output_file" | cut -f1)"
        fi
        return 0
    else
        log::error "Failed to generate PDF"
        return 1
    fi
}

#######################################
# Test web scraping API endpoint
# Arguments:
#   $1 - URL to scrape (optional, uses URL var)
#   $2 - Output filename (optional, uses OUTPUT var)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::test_scrape() {
    local test_url="${1:-${URL:-https://example.com}}"
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    local output_file="${2:-${OUTPUT:-$BROWSERLESS_TEST_OUTPUT_DIR/scrape_test.html}}"
    
    log::header "${MSG_USAGE_SCRAPE}"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    log::info "Scraping content from: $test_url"
    
    local response
    response=$(curl -X POST "$BROWSERLESS_BASE_URL/chrome/content" \
        -H "Content-Type: application/json" \
        -d "{
            \"url\": \"$test_url\"
        }" \
        --silent \
        --show-error \
        --max-time 60 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "✓ Content scraped successfully"
        echo
        log::info "Response preview (first 500 chars):"
        echo "$response" | head -c 500
        echo
        echo "..."
        
        # Save full response to file
        echo "$response" > "$output_file"
        log::info "Full response saved to: $output_file"
        return 0
    else
        log::error "Failed to scrape content: $response"
        return 1
    fi
}

#######################################
# Check browser pool status via pressure endpoint
# Returns: 0 if successful, 1 if failed
#######################################
browserless::test_pressure() {
    log::header "${MSG_USAGE_PRESSURE}"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    local response
    response=$(curl -X GET "$BROWSERLESS_BASE_URL/pressure" \
        --silent \
        --show-error \
        --max-time 30 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "✓ Pool status retrieved"
        echo
        log::info "Current pressure metrics:"
        echo "$response" | jq . 2>/dev/null || echo "$response"
        
        # Parse and display key metrics
        if command -v jq &> /dev/null; then
            local running=$(echo "$response" | jq -r '.running // 0')
            local queued=$(echo "$response" | jq -r '.queued // 0')
            local maxConcurrent=$(echo "$response" | jq -r '.maxConcurrent // "N/A"')
            local isAvailable=$(echo "$response" | jq -r '.isAvailable // false')
            local cpu=$(echo "$response" | jq -r '.cpu // 0')
            local memory=$(echo "$response" | jq -r '.memory // 0')
            
            echo
            log::info "Summary:"
            log::info "  Running browsers: $running"
            log::info "  Queued requests: $queued"
            log::info "  Max concurrent: $maxConcurrent"
            log::info "  Available: $isAvailable"
            log::info "  CPU usage: $(echo "$cpu * 100" | bc 2>/dev/null || echo "N/A")%"
            log::info "  Memory usage: $(echo "$memory * 100" | bc 2>/dev/null || echo "N/A")%"
        fi
        return 0
    else
        log::error "Failed to get pool status: $response"
        return 1
    fi
}

#######################################
# Test custom function execution
# Arguments:
#   $1 - URL to execute function on (optional)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::test_function() {
    local test_url="${1:-${URL:-https://example.com}}"
    
    log::header "🔧 Testing Browserless Function Execution"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    log::info "Executing custom function on: $test_url"
    
    local response
    response=$(curl -X POST "$BROWSERLESS_BASE_URL/chrome/function" \
        -H "Content-Type: application/json" \
        -d "{
            \"code\": \"async ({ page }) => {
                await page.goto('$test_url');
                const title = await page.title();
                const url = page.url();
                const viewport = page.viewport();
                return { title, url, viewport };
            }\"
        }" \
        --silent \
        --show-error \
        --max-time 60 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "✓ Function executed successfully"
        echo
        log::info "Function result:"
        echo "$response" | jq . 2>/dev/null || echo "$response"
        return 0
    else
        log::error "Failed to execute function: $response"
        return 1
    fi
}

#######################################
# Run all API usage examples
# Arguments:
#   $1 - URL to use for all tests (optional, uses URL var)
# Returns: 0 if all successful, 1 if any failed
#######################################
browserless::test_all_apis() {
    local test_url="${1:-${URL:-https://example.com}}"
    
    log::header "${MSG_USAGE_ALL}"
    
    if ! browserless::is_healthy; then
        log::error "Browserless is not healthy. Please check the service."
        return 1
    fi
    
    # Create test output directory
    local test_dir="browserless_test_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$test_dir"
    cd "$test_dir" || return 1
    
    log::info "Test outputs will be saved in: $(pwd)"
    echo
    
    local failed_tests=0
    
    # Run each test
    browserless::test_pressure || ((failed_tests++))
    echo
    sleep 2
    
    URL="$test_url" browserless::test_screenshot || ((failed_tests++))
    echo
    sleep 2
    
    URL="$test_url" browserless::test_pdf || ((failed_tests++))
    echo
    sleep 2
    
    URL="$test_url" browserless::test_scrape || ((failed_tests++))
    echo
    sleep 2
    
    URL="$test_url" browserless::test_function || ((failed_tests++))
    echo
    
    cd ..
    
    if [[ $failed_tests -eq 0 ]]; then
        log::success "✅ All tests completed successfully. Results saved in: $(pwd)/$test_dir"
        return 0
    else
        log::warn "⚠️  $failed_tests test(s) failed. Results saved in: $(pwd)/$test_dir"
        return 1
    fi
}

# Export functions for subshell availability
export -f browserless::ensure_test_output_dir
export -f browserless::test_screenshot
export -f browserless::safe_screenshot
export -f browserless::test_pdf
export -f browserless::test_scrape
export -f browserless::test_pressure
export -f browserless::test_function
export -f browserless::test_all_apis