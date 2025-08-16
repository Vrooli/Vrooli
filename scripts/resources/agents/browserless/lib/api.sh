#!/usr/bin/env bash
# Browserless API Functions
# API testing, examples, and usage demonstrations

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Source configuration messages
BROWSERLESS_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${BROWSERLESS_LIB_DIR}/../config/messages.sh" 2>/dev/null || true
# Export message variables
browserless::export_messages 2>/dev/null || true

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
    local test_url="${1:-${URL:-http://httpbin.org/html}}"
    
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
    local test_url="${1:-${URL:-http://httpbin.org/html}}"
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    local output_file="${2:-${OUTPUT:-$BROWSERLESS_TEST_OUTPUT_DIR/document_test.pdf}}"
    local temp_file="/tmp/browserless_pdf_$$"
    
    log::header "${MSG_USAGE_PDF}"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    log::info "Generating PDF from: $test_url"
    log::info "Output file: $output_file"
    
    # First save to temp file and capture HTTP status
    local http_status
    http_status=$(curl -X POST "$BROWSERLESS_BASE_URL/chrome/pdf" \
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
        log::error "PDF request failed with HTTP status: $http_status"
        if [[ -f "$temp_file" ]]; then
            log::debug "Error response: $(head -c 200 "$temp_file" 2>/dev/null || echo 'No response body')"
            trash::safe_remove "$temp_file" --temp
        fi
        return 1
    fi
    
    # Validate file exists and has content
    if [[ ! -f "$temp_file" ]] || [[ ! -s "$temp_file" ]]; then
        log::error "No PDF data received"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # Check if it's actually a PDF using file command
    if command -v file &> /dev/null; then
        local file_type
        file_type=$(file -b --mime-type "$temp_file" 2>/dev/null || echo "unknown")
        if [[ "$file_type" != application/pdf ]]; then
            log::error "Response is not a PDF (detected: $file_type)"
            log::debug "Response preview: $(head -c 100 "$temp_file" 2>/dev/null || echo 'Cannot read file')"
            trash::safe_remove "$temp_file" --temp
            return 1
        fi
    fi
    
    # Check minimum file size (real PDFs should be at least a few KB)
    local file_size
    file_size=$(stat -f%z "$temp_file" 2>/dev/null || stat -c%s "$temp_file" 2>/dev/null || echo "0")
    if [[ "$file_size" -lt 1024 ]]; then
        log::error "PDF file too small ($file_size bytes) - likely an error message"
        log::debug "Content: $(cat "$temp_file" 2>/dev/null || echo 'Cannot read file')"
        trash::safe_remove "$temp_file" --temp
        return 1
    fi
    
    # All checks passed, move to final location
    if mv "$temp_file" "$output_file"; then
        log::success "✓ PDF saved to: $output_file"
        log::info "File size: $(du -h "$output_file" | cut -f1)"
        log::info "Validated as: $(file -b --mime-type "$output_file" 2>/dev/null || echo 'PDF file')"
        return 0
    else
        log::error "Failed to save PDF to: $output_file"
        trash::safe_remove "$temp_file" --temp
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
    local test_url="${1:-${URL:-http://httpbin.org/html}}"
    
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
    local test_url="${1:-${URL:-http://httpbin.org/html}}"
    
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
    local test_url="${1:-${URL:-http://httpbin.org/html}}"
    
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

#######################################
# Execute n8n workflow via browser automation
# Arguments:
#   $1 - Workflow ID (required)
#   $2 - N8N base URL (optional, default: http://localhost:5678)
#   $3 - Timeout in milliseconds (optional, default: 60000)
#   $4 - Input data (optional, JSON string or @file or env var)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::execute_n8n_workflow() {
    local workflow_id="${1:?Workflow ID required}"
    local n8n_url="${2:-http://localhost:5678}"
    local timeout="${3:-60000}"
    local input_data="${4:-}"
    
    log::header "🚀 Executing N8N Workflow via Browser Automation"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    # Build workflow URL
    local workflow_url
    if [[ "$workflow_id" == http* ]]; then
        workflow_url="$workflow_id"
    else
        workflow_url="${n8n_url}/workflow/${workflow_id}"
    fi
    
    # Process input data
    local processed_input=""
    local input_description="None"
    
    if [[ -n "$input_data" ]]; then
        # Check if input_data starts with @ (file reference)
        if [[ "$input_data" == @* ]]; then
            local input_file="${input_data#@}"
            if [[ -f "$input_file" ]]; then
                processed_input=$(cat "$input_file")
                input_description="From file: $input_file"
            else
                log::error "Input file not found: $input_file"
                return 1
            fi
        else
            # Direct JSON input or environment variable
            if [[ "$input_data" == "\$"* ]]; then
                # Environment variable reference
                local env_var="${input_data#\$}"
                processed_input="${!env_var}"
                input_description="From environment variable: $env_var"
            else
                # Direct JSON input
                processed_input="$input_data"
                input_description="Direct JSON input"
            fi
        fi
        
        # Check for WORKFLOW_INPUT environment variable as fallback
        if [[ -z "$processed_input" && -n "${WORKFLOW_INPUT:-}" ]]; then
            processed_input="$WORKFLOW_INPUT"
            input_description="From WORKFLOW_INPUT environment variable"
        fi
        
        # Validate JSON if input provided
        if [[ -n "$processed_input" ]]; then
            if ! echo "$processed_input" | jq . >/dev/null 2>&1; then
                log::error "Invalid JSON input data"
                log::debug "Input: $processed_input"
                return 1
            fi
        fi
    fi
    
    log::info "Workflow ID: $workflow_id"
    log::info "Workflow URL: $workflow_url"
    log::info "Timeout: ${timeout}ms"
    log::info "Input Data: $input_description"
    if [[ -n "$processed_input" ]]; then
        log::debug "Input JSON: $(echo "$processed_input" | jq -c . 2>/dev/null || echo "$processed_input")"
    fi
    echo
    
    # Generate function code for workflow execution
    local function_code
    read -r -d '' function_code << 'EOF' || true
export default async ({ page }) => {
  const executionData = {
    workflowId: '%WORKFLOW_ID%',
    workflowUrl: '%WORKFLOW_URL%',
    consoleLogs: [],
    pageErrors: [],
    networkErrors: [],
    executionStatus: {},
    startTime: new Date().toISOString(),
    success: false
  };
  
  try {
    // Set up console log capture
    page.on('console', msg => {
      executionData.consoleLogs.push({
        type: msg.type(),
        text: msg.text(),
        timestamp: new Date().toISOString()
      });
    });
    
    // Set up page error capture
    page.on('pageerror', err => {
      executionData.pageErrors.push({
        message: err.message,
        stack: err.stack,
        timestamp: new Date().toISOString()
      });
    });
    
    // Set up network error capture
    page.on('requestfailed', req => {
      executionData.networkErrors.push({
        url: req.url(),
        method: req.method(),
        failure: req.failure()?.errorText || 'Unknown error',
        timestamp: new Date().toISOString()
      });
    });
    
    console.log('Navigating to workflow:', '%WORKFLOW_URL%');
    
    // Navigate to workflow
    await page.goto('%WORKFLOW_URL%', {
      waitUntil: 'networkidle2',
      timeout: %TIMEOUT%
    });
    
    // Check if we landed on a signin page
    const currentUrl = page.url();
    console.log('Current URL after navigation:', currentUrl);
    console.log('URL includes signin:', currentUrl.includes('/signin'));
    console.log('URL includes login:', currentUrl.includes('/login'));
    
    if (currentUrl.includes('/signin') || currentUrl.includes('/login')) {
      console.log('Authentication required - attempting login');
      
      try {
        // Wait for login form
        await page.waitForSelector('input[type=email], input[name=email], input[id=email]', { timeout: 5000 });
        
        // Fill in credentials using environment variables passed from shell
        const email = '%N8N_EMAIL%';
        const password = '%N8N_PASSWORD%';
        
        console.log('Credential check - email:', email);
        console.log('Credential check - password length:', password ? password.length : 0);
        console.log('Email placeholder check:', email !== '%N8N_EMAIL%');
        console.log('Password placeholder check:', password !== '%N8N_PASSWORD%');
        
        if (email && email !== '%N8N_EMAIL%' && password && password !== '%N8N_PASSWORD%') {
          console.log('Using provided credentials for login');
          
          await page.type('input[type=email], input[name=email], input[id=email]', email);
          await page.type('input[type=password], input[name=password], input[id=password]', password);
          
          // Click submit button
          await page.click('button[type=submit], input[type=submit], button:contains("Sign in"), button:contains("Login")');
          
          // Wait for redirect
          await page.waitForNavigation({ waitUntil: 'networkidle2', timeout: 10000 });
          
          console.log('After login, URL:', page.url());
          
          // Navigate to workflow again if needed
          if (!page.url().includes('/workflow/%WORKFLOW_ID%')) {
            console.log('Navigating to workflow after authentication');
            await page.goto('%WORKFLOW_URL%', { 
              waitUntil: 'networkidle2',
              timeout: %TIMEOUT%
            });
          }
        } else {
          console.log('No valid credentials provided, cannot authenticate');
          throw new Error('Authentication required but no credentials provided');
        }
      } catch (authError) {
        console.log('Authentication failed:', authError.message);
        throw new Error('Failed to authenticate: ' + authError.message);
      }
    }
    
    // Wait for page to load after potential authentication
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    console.log('Looking for execute button...');
    
    // Wait for execute button to appear
    await page.waitForSelector('[data-test-id="execute-workflow-button"]', {
      timeout: 10000
    });
    
    console.log('Clicking execute button...');
    
    // Click the execute button
    await page.click('[data-test-id="execute-workflow-button"]');
    
    // If input data is provided, try to fill it in
    const inputData = '%INPUT_DATA%';
    if (inputData && inputData !== '%INPUT_DATA%') {
      console.log('Input data provided, looking for input form...');
      
      try {
        // Wait a moment for any input dialog to appear
        await new Promise(resolve => setTimeout(resolve, 1000));
        
        // Look for common input field selectors in n8n
        const inputSelectors = [
          'textarea[data-test-id="workflow-input-data"]',
          'textarea[placeholder*="JSON"]',
          'textarea[placeholder*="input"]',
          '.cm-editor textarea',
          '.monaco-editor textarea',
          'textarea:not([readonly])',
          'input[type="text"]:not([readonly])'
        ];
        
        let inputFound = false;
        for (const selector of inputSelectors) {
          try {
            await page.waitForSelector(selector, { timeout: 2000 });
            console.log(`Found input field with selector: ${selector}`);
            
            // Clear existing content and input new data
            await page.evaluate((sel) => {
              const element = document.querySelector(sel);
              if (element) {
                element.value = '';
                element.focus();
              }
            }, selector);
            
            await page.type(selector, inputData, { delay: 50 });
            console.log('Input data entered successfully');
            inputFound = true;
            break;
          } catch (e) {
            // Continue to next selector
          }
        }
        
        if (!inputFound) {
          console.log('No input field found, proceeding without input data');
        }
        
        // Look for and click any "Execute" or "Run" button that might appear
        try {
          const executeSelectors = [
            'button[data-test-id="execute-button"]',
            'button:contains("Execute")',
            'button:contains("Run")',
            '.execute-button'
          ];
          
          for (const selector of executeSelectors) {
            try {
              await page.waitForSelector(selector, { timeout: 1000 });
              await page.click(selector);
              console.log(`Clicked additional execute button: ${selector}`);
              break;
            } catch (e) {
              // Continue to next selector
            }
          }
        } catch (e) {
          // No additional execute button found
        }
        
      } catch (error) {
        console.log('Input handling failed, continuing with execution:', error.message);
      }
    }
    
    console.log('Workflow execution triggered, monitoring...');
    
    // Wait and monitor for execution completion
    let pollCount = 0;
    const maxPolls = Math.floor(%TIMEOUT% / 2000); // Poll every 2 seconds
    
    while (pollCount < maxPolls) {
      await new Promise(resolve => setTimeout(resolve, 2000));
      pollCount++;
      
      // Check for execution completion indicators in console logs
      const recentLogs = executionData.consoleLogs.slice(-5);
      let executionComplete = false;
      
      for (const log of recentLogs) {
        const text = log.text.toLowerCase();
        if (text.includes('workflow execution finished') ||
            text.includes('execution completed') ||
            text.includes('execution successful') ||
            text.includes('finished executing')) {
          executionData.executionStatus.completed = true;
          executionComplete = true;
          break;
        } else if (text.includes('workflow execution failed') ||
                   text.includes('execution error') ||
                   text.includes('execution failed')) {
          executionData.executionStatus.failed = true;
          executionComplete = true;
          break;
        }
      }
      
      if (executionComplete) {
        console.log(`Execution detected complete after ${pollCount} polls`);
        break;
      }
      
      console.log(`Monitoring poll ${pollCount}/${maxPolls}...`);
    }
    
    executionData.endTime = new Date().toISOString();
    executionData.pollsCompleted = pollCount;
    executionData.success = true;
    
    return {
      data: executionData,
      type: 'application/json'
    };
    
  } catch (err) {
    executionData.error = {
      message: err.message,
      stack: err.stack,
      timestamp: new Date().toISOString()
    };
    executionData.endTime = new Date().toISOString();
    executionData.success = false;
    
    return {
      data: executionData,
      type: 'application/json'
    };
  }
}
EOF
    
    # Replace placeholders in function code  
    function_code="${function_code//%WORKFLOW_ID%/$workflow_id}"
    function_code="${function_code//%WORKFLOW_URL%/$workflow_url}"
    function_code="${function_code//%TIMEOUT%/$timeout}"
    
    # Inject N8N credentials if available
    local n8n_email="${N8N_EMAIL:-}"
    local n8n_password="${N8N_PASSWORD:-}"
    
    # Log credential status  
    if [[ -n "$n8n_email" ]]; then
        log::info "🔐 Found N8N email credential: $n8n_email"
    else
        log::info "⚠️  No N8N_EMAIL environment variable found"
    fi
    
    if [[ -n "$n8n_password" ]]; then
        log::info "🔐 Found N8N password credential: [${#n8n_password} characters]"
    else
        log::info "⚠️  No N8N_PASSWORD environment variable found"
    fi
    
    function_code="${function_code//%N8N_EMAIL%/$n8n_email}"
    function_code="${function_code//%N8N_PASSWORD%/$n8n_password}"
    
    # Escape input data for JavaScript insertion
    local escaped_input=""
    if [[ -n "$processed_input" ]]; then
        # Escape quotes and newlines for safe JavaScript insertion
        escaped_input=$(echo "$processed_input" | sed 's/\\/\\\\/g; s/"/\\"/g; s/$/\\n/' | tr -d '\n' | sed 's/\\n$//')
    fi
    function_code="${function_code//%INPUT_DATA%/$escaped_input}"
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    local temp_file="/tmp/browserless_workflow_exec_$$"
    # Sanitize workflow_id for filename use (replace special characters with underscores)
    local safe_workflow_id
    safe_workflow_id=$(echo "$workflow_id" | sed 's/[^a-zA-Z0-9._-]/_/g')
    local output_file="${BROWSERLESS_TEST_OUTPUT_DIR}/workflow_execution_${safe_workflow_id}_$(date +%Y%m%d_%H%M%S).json"
    
    log::info "Executing workflow automation..."
    
    # Execute the function via browserless
    local response
    response=$(curl -X POST "$BROWSERLESS_BASE_URL/chrome/function" \
        -H "Content-Type: application/javascript" \
        -d "$function_code" \
        --silent \
        --show-error \
        --max-time $((timeout / 1000 + 30)) \
        2>&1)
    
    local curl_exit_code=$?
    
    if [[ $curl_exit_code -ne 0 ]]; then
        echo
        log::error "❌ Failed to execute workflow automation (curl exit code: $curl_exit_code)"
        
        # Provide specific curl error information
        case $curl_exit_code in
            6)  log::info "• Error: Could not resolve host - check if browserless is running on $BROWSERLESS_BASE_URL" ;;
            7)  log::info "• Error: Failed to connect to host - check if browserless is accessible" ;;
            28) log::info "• Error: Operation timeout - try increasing timeout (currently ${timeout}ms)" ;;
            52) log::info "• Error: Empty response from server - browserless may be overloaded" ;;
            56) log::info "• Error: Network receive failure - connection interrupted" ;;
            *)  log::info "• Curl error code $curl_exit_code - check network connectivity" ;;
        esac
        
        if [[ -n "$response" ]]; then
            echo
            log::info "🔍 Curl Response/Error:"
            echo "----------------------------------------"
            echo "$response"
            echo "----------------------------------------"
        fi
        
        log::info "💡 Troubleshooting steps:"
        log::info "  1. Check browserless status: resource-browserless status"
        log::info "  2. Test browserless health: curl -s $BROWSERLESS_BASE_URL/pressure"
        log::info "  3. Verify n8n is accessible: curl -s $n8n_url/healthz"
        
        return 1
    fi
    
    # Save and parse response
    echo "$response" > "$temp_file"
    
    # Try to parse the response
    if command -v jq &> /dev/null && jq . "$temp_file" &> /dev/null; then
        # Valid JSON response
        local success=$(jq -r '.success // false' "$temp_file")
        local workflow_success=$(jq -r '.executionStatus.completed // false' "$temp_file")
        local workflow_failed=$(jq -r '.executionStatus.failed // false' "$temp_file")
        local console_logs_count=$(jq -r '.consoleLogs | length' "$temp_file" 2>/dev/null || echo "0")
        local errors_count=$(jq -r '.pageErrors | length' "$temp_file" 2>/dev/null || echo "0")
        local network_errors_count=$(jq -r '.networkErrors | length' "$temp_file" 2>/dev/null || echo "0")
        local polls_completed=$(jq -r '.pollsCompleted // 0' "$temp_file")
        
        # Move to final output location
        mv "$temp_file" "$output_file"
        
        echo
        log::success "✓ Workflow automation completed"
        log::info "Results saved to: $output_file"
        echo
        
        # Display summary
        log::info "📊 Execution Summary:"
        log::info "  Automation Success: $success"
        log::info "  Workflow Completed: $workflow_success"
        log::info "  Workflow Failed: $workflow_failed"
        log::info "  Console Logs: $console_logs_count"
        log::info "  Page Errors: $errors_count"
        log::info "  Network Errors: $network_errors_count"
        log::info "  Monitoring Polls: $polls_completed"
        log::info "  Input Data Used: $input_description"
        
        # Show recent console logs
        if [[ "$console_logs_count" -gt 0 ]]; then
            echo
            log::info "📋 Recent Console Logs:"
            jq -r '.consoleLogs[-5:][] | "  [\(.type | ascii_upcase)] \(.text)"' "$output_file" 2>/dev/null || true
            
            # Check for error patterns in console logs
            local error_log_count
            error_log_count=$(jq -r '.consoleLogs[] | select(.type == "error") | .text' "$output_file" 2>/dev/null | wc -l || echo "0")
            local warn_log_count
            warn_log_count=$(jq -r '.consoleLogs[] | select(.type == "warning") | .text' "$output_file" 2>/dev/null | wc -l || echo "0")
            
            if [[ "$error_log_count" -gt 0 ]] || [[ "$warn_log_count" -gt 0 ]]; then
                echo
                log::warn "⚠️  Detected $error_log_count errors and $warn_log_count warnings in console logs"
                log::info "Run the following to see detailed logs:"
                log::info "  jq '.consoleLogs[] | select(.type == \"error\" or .type == \"warning\")' '$output_file'"
            fi
        fi
        
        # Show errors if any
        if [[ "$errors_count" -gt 0 ]]; then
            echo
            log::warn "⚠️  Page Errors:"
            jq -r '.pageErrors[] | "  \(.message)"' "$output_file" 2>/dev/null || true
        fi
        
        # Determine overall success
        if [[ "$success" == "true" ]]; then
            if [[ "$workflow_success" == "true" ]]; then
                log::success "🎉 Workflow execution completed successfully!"
                return 0
            elif [[ "$workflow_failed" == "true" ]]; then
                log::warn "⚠️  Workflow execution failed"
                return 1
            else
                log::warn "⚠️  Workflow execution status unclear (may have timed out)"
                return 1
            fi
        else
            log::error "❌ Browser automation failed"
            return 1
        fi
    else
        # Invalid JSON or non-JSON response - provide detailed error information
        mv "$temp_file" "$output_file"
        echo
        log::error "❌ Invalid response from workflow automation"
        log::info "Full response saved to: $output_file"
        
        # Always show response preview for debugging
        echo
        log::info "🔍 Response Preview (first 1000 characters):"
        echo "----------------------------------------"
        head -c 1000 "$output_file" 2>/dev/null || echo "Cannot read response file"
        echo
        echo "----------------------------------------"
        
        # Check if it's an HTTP error response
        if head -c 50 "$output_file" 2>/dev/null | grep -qi "error\|exception\|failed\|timeout"; then
            echo
            log::warn "⚠️  This appears to be an error response from browserless"
            log::info "Common causes:"
            log::info "  • N8N authentication required (workflow redirects to login)"
            log::info "  • Workflow ID '$workflow_id' not found"  
            log::info "  • N8N service not accessible at $n8n_url"
            log::info "  • Browser timeout during execution"
            log::info "  • Workflow execution errors"
        fi
        
        # Check if response looks like HTML (authentication redirect)
        if head -c 50 "$output_file" 2>/dev/null | grep -qi "<html\|<!doctype"; then
            echo
            log::warn "🔐 Response appears to be HTML - likely an authentication redirect"
            log::info "Try using the enhanced authentication workflow:"
            log::info "  resource-n8n list-workflows | grep auth"
        fi
        
        return 1
    fi
}

#######################################
# Capture console logs from any URL
# Arguments:
#   $1 - URL to capture console logs from (required)
#   $2 - Output filename (optional, default: console-capture.json)
#   $3 - Wait time in milliseconds (optional, default: 3000)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::capture_console_logs() {
    local url="${1:?URL required}"
    local output="${2:-console-capture.json}"
    local wait_time="${3:-3000}"
    
    log::header "📋 Capturing Console Logs"
    
    if ! browserless::is_healthy; then
        log::error "${MSG_NOT_HEALTHY}"
        return 1
    fi
    
    log::info "Target URL: $url"
    log::info "Wait time: ${wait_time}ms"
    echo
    
    # Ensure test output directory exists
    browserless::ensure_test_output_dir
    
    # If output is relative path, put it in test output directory
    if [[ "$output" != /* ]]; then
        output="${BROWSERLESS_TEST_OUTPUT_DIR}/$output"
    fi
    
    # Generate function code for console capture
    local function_code
    read -r -d '' function_code << 'EOF' || true
export default async ({ page }) => {
  const captureData = {
    url: '%TARGET_URL%',
    consoleLogs: [],
    pageErrors: [],
    networkErrors: [],
    dialogs: [],
    performanceMetrics: {},
    pageInfo: {},
    startTime: new Date().toISOString(),
    success: false
  };
  
  try {
    // Set up console log capture
    page.on('console', msg => {
      captureData.consoleLogs.push({
        type: msg.type(),
        text: msg.text(),
        args: msg.args().length,
        location: msg.location(),
        timestamp: new Date().toISOString()
      });
    });
    
    // Set up page error capture
    page.on('pageerror', err => {
      captureData.pageErrors.push({
        message: err.message,
        stack: err.stack,
        name: err.name,
        timestamp: new Date().toISOString()
      });
    });
    
    // Set up network error capture
    page.on('requestfailed', req => {
      captureData.networkErrors.push({
        url: req.url(),
        method: req.method(),
        failure: req.failure()?.errorText || 'Unknown error',
        timestamp: new Date().toISOString()
      });
    });
    
    // Set up dialog capture
    page.on('dialog', dialog => {
      captureData.dialogs.push({
        type: dialog.type(),
        message: dialog.message(),
        defaultValue: dialog.defaultValue(),
        timestamp: new Date().toISOString()
      });
      // Auto-dismiss dialogs to prevent hanging
      dialog.dismiss().catch(() => {});
    });
    
    console.log('Navigating to:', '%TARGET_URL%');
    
    // Navigate to the URL
    const navigationStart = Date.now();
    await page.goto('%TARGET_URL%', {
      waitUntil: 'networkidle2',
      timeout: 30000
    });
    const navigationTime = Date.now() - navigationStart;
    
    console.log('Page loaded, waiting %WAIT_TIME%ms for content...');
    
    // Wait for content to load
    await new Promise(resolve => setTimeout(resolve, %WAIT_TIME%));
    
    // Capture basic page information
    captureData.pageInfo = {
      title: await page.title(),
      url: page.url(),
      navigationTime: navigationTime,
      timestamp: new Date().toISOString()
    };
    
    // Capture performance metrics
    try {
      const performanceData = await page.evaluate(() => {
        const perfTiming = performance.timing;
        const navigation = performance.getEntriesByType('navigation')[0];
        
        return {
          domContentLoaded: perfTiming.domContentLoadedEventEnd - perfTiming.navigationStart,
          loadComplete: perfTiming.loadEventEnd - perfTiming.navigationStart,
          firstPaint: performance.getEntriesByType('paint').find(entry => entry.name === 'first-paint')?.startTime || null,
          firstContentfulPaint: performance.getEntriesByType('paint').find(entry => entry.name === 'first-contentful-paint')?.startTime || null,
          resourceLoadTime: navigation?.loadEventEnd - navigation?.loadEventStart || null,
          memoryUsage: performance.memory ? {
            usedJSHeapSize: performance.memory.usedJSHeapSize,
            totalJSHeapSize: performance.memory.totalJSHeapSize,
            jsHeapSizeLimit: performance.memory.jsHeapSizeLimit
          } : null
        };
      });
      
      captureData.performanceMetrics = performanceData;
    } catch (err) {
      captureData.performanceMetrics = { error: 'Failed to capture performance metrics: ' + err.message };
    }
    
    captureData.endTime = new Date().toISOString();
    captureData.success = true;
    
    console.log(`Console capture completed. Logs: ${captureData.consoleLogs.length}, Errors: ${captureData.pageErrors.length}`);
    
    return {
      data: captureData,
      type: 'application/json'
    };
    
  } catch (err) {
    captureData.success = false;
    captureData.error = {
      message: err.message,
      stack: err.stack,
      timestamp: new Date().toISOString()
    };
    captureData.endTime = new Date().toISOString();
    
    return {
      data: captureData,
      type: 'application/json'
    };
  }
}
EOF
    
    # Replace placeholders in function code
    function_code="${function_code//%TARGET_URL%/$url}"
    function_code="${function_code//%WAIT_TIME%/$wait_time}"
    
    local temp_file="/tmp/browserless_console_capture_$$"
    
    log::info "Capturing console logs..."
    
    # Execute the function via browserless
    local response
    response=$(curl -X POST "$BROWSERLESS_BASE_URL/chrome/function" \
        -H "Content-Type: application/javascript" \
        -d "$function_code" \
        --silent \
        --show-error \
        --max-time 60 \
        2>&1)
    
    local curl_exit_code=$?
    
    if [[ $curl_exit_code -ne 0 ]]; then
        echo
        log::error "❌ Failed to capture console logs (curl exit code: $curl_exit_code)"
        
        # Provide specific curl error information
        case $curl_exit_code in
            6)  log::info "• Error: Could not resolve host - check if browserless is running on $BROWSERLESS_BASE_URL" ;;
            7)  log::info "• Error: Failed to connect to host - check if browserless is accessible" ;;
            28) log::info "• Error: Operation timeout - try increasing wait time (currently ${wait_time}ms)" ;;
            52) log::info "• Error: Empty response from server - browserless may be overloaded" ;;
            56) log::info "• Error: Network receive failure - connection interrupted" ;;
            *)  log::info "• Curl error code $curl_exit_code - check network connectivity" ;;
        esac
        
        if [[ -n "$response" ]]; then
            echo
            log::info "🔍 Curl Response/Error:"
            echo "----------------------------------------"
            echo "$response"
            echo "----------------------------------------"
        fi
        
        log::info "💡 Troubleshooting steps:"
        log::info "  1. Check browserless status: resource-browserless status"
        log::info "  2. Test browserless health: curl -s $BROWSERLESS_BASE_URL/pressure"
        log::info "  3. Verify target URL is accessible: curl -s '$url'"
        
        return 1
    fi
    
    # Save and parse response
    echo "$response" > "$temp_file"
    
    # Try to parse the response
    if command -v jq &> /dev/null && jq . "$temp_file" &> /dev/null; then
        # Valid JSON response
        local success=$(jq -r '.success // false' "$temp_file")
        local console_logs_count=$(jq -r '.consoleLogs | length' "$temp_file" 2>/dev/null || echo "0")
        local page_errors_count=$(jq -r '.pageErrors | length' "$temp_file" 2>/dev/null || echo "0")
        local network_errors_count=$(jq -r '.networkErrors | length' "$temp_file" 2>/dev/null || echo "0")
        local page_title=$(jq -r '.pageInfo.title // "Unknown"' "$temp_file" 2>/dev/null)
        
        # Move to final output location
        mv "$temp_file" "$output"
        
        echo
        log::success "✓ Console capture completed"
        log::info "Results saved to: $output"
        echo
        
        # Display summary
        log::info "📊 Capture Summary:"
        log::info "  Success: $success"
        log::info "  Page Title: $page_title"
        log::info "  Console Logs: $console_logs_count"
        log::info "  Page Errors: $page_errors_count"
        log::info "  Network Errors: $network_errors_count"
        
        # Show console log breakdown by type
        if [[ "$console_logs_count" -gt 0 ]]; then
            echo
            log::info "📋 Console Log Types:"
            jq -r '.consoleLogs | group_by(.type) | .[] | "\(.length) \(.[0].type) messages"' "$output" 2>/dev/null || true
        fi
        
        # Show recent console logs
        if [[ "$console_logs_count" -gt 0 ]]; then
            echo
            log::info "📋 Recent Console Logs (last 5):"
            jq -r '.consoleLogs[-5:][] | "  [\(.type | ascii_upcase)] \(.text)"' "$output" 2>/dev/null || true
        fi
        
        # Show errors if any
        if [[ "$page_errors_count" -gt 0 ]]; then
            echo
            log::warn "⚠️  Page Errors:"
            jq -r '.pageErrors[] | "  \(.message)"' "$output" 2>/dev/null || true
        fi
        
        if [[ "$success" == "true" ]]; then
            log::success "🎉 Console capture completed successfully!"
            return 0
        else
            log::error "❌ Console capture failed"
            return 1
        fi
    else
        # Invalid JSON or non-JSON response - provide detailed error information
        mv "$temp_file" "$output"
        echo
        log::error "❌ Invalid response from console capture"
        log::info "Full response saved to: $output"
        
        # Always show response preview for debugging
        echo
        log::info "🔍 Response Preview (first 1000 characters):"
        echo "----------------------------------------"
        head -c 1000 "$output" 2>/dev/null || echo "Cannot read response file"
        echo
        echo "----------------------------------------"
        
        # Check if it's an HTTP error response
        if head -c 50 "$output" 2>/dev/null | grep -qi "error\|exception\|failed\|timeout"; then
            echo
            log::warn "⚠️  This appears to be an error response from browserless"
            log::info "Common causes:"
            log::info "  • Target URL '$url' not accessible"
            log::info "  • Browserless service overloaded or timing out"
            log::info "  • Network connectivity issues"
            log::info "  • JavaScript execution errors on the page"
        fi
        
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
export -f browserless::execute_n8n_workflow
export -f browserless::capture_console_logs