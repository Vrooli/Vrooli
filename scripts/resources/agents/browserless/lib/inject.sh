#!/usr/bin/env bash
# Browserless Simple Test Data Injection
# Minimal test data injection for service validation

#######################################  
# Inject function or test data into Browserless
# Arguments:
#   $1 - Optional JSON file path for function injection
# If no file provided, creates test data for validation
#######################################
browserless::inject() {
    local file_path="${1:-}"
    
    # If file provided, handle JSON function injection
    if [[ -n "$file_path" ]]; then
        browserless::inject_function "$file_path"
        return $?
    fi
    
    # Otherwise, fallback to legacy test data injection
    log::info "Injecting test data into Browserless..."
    
    # Verify service is available
    if ! docker::is_running "$BROWSERLESS_CONTAINER_NAME"; then
        log::error "Browserless container not running - start it first"
        log::info "Run: ./manage.sh --action start"
        return 1
    fi
    
    # Create test workspace
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    mkdir -p "$test_dir"
    
    log::info "Creating test validation script..."
    
    # Create minimal test script
    cat > "$test_dir/validation.js" << 'EOF'
module.exports = async ({ page }) => {
    await page.goto('https://httpbin.org/html');
    const title = await page.title();
    return { 
        success: true, 
        title, 
        url: page.url(),
        timestamp: new Date().toISOString() 
    };
};
EOF
    
    # Create simple screenshot test
    cat > "$test_dir/screenshot-test.js" << 'EOF'
module.exports = async ({ page }) => {
    await page.goto('https://httpbin.org/html');
    await page.screenshot({ path: '/workspace/screenshots/validation-test.png' });
    return { success: true, message: 'Screenshot saved to validation-test.png' };
};
EOF
    
    log::success "Test scripts created in $test_dir"
    
    # Validate injection worked
    if browserless::validate_injection; then
        log::success "✅ Test data injected and validated successfully"
        browserless::show_injection_info
        return 0
    else
        log::error "❌ Injection validation failed"
        return 1
    fi
}

#######################################
# Validate injection is working  
#######################################
browserless::validate_injection() {
    # Check test directory exists
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    if [[ ! -d "$test_dir" ]]; then
        log::error "Test data directory not found: $test_dir"
        return 1
    fi
    
    # Check test files exist
    if [[ ! -f "$test_dir/validation.js" ]]; then
        log::error "Validation script not found"
        return 1
    fi
    
    if [[ ! -f "$test_dir/screenshot-test.js" ]]; then
        log::error "Screenshot test script not found"  
        return 1
    fi
    
    # Check service responds
    if ! http::check_endpoint "http://localhost:$BROWSERLESS_PORT/pressure"; then
        log::error "Browserless API not responding"
        return 1
    fi
    
    log::success "All injection validation checks passed"
    return 0
}

#######################################
# Show injection status and information
#######################################  
browserless::injection_status() {
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    
    log::header "📊 Browserless Injection Status"
    
    if [[ -d "$test_dir" ]]; then
        local file_count=$(find "$test_dir" -name "*.js" | wc -l)
        log::info "Test data directory: $test_dir"
        log::info "Test script files: $file_count"
        
        # List test files
        if [[ $file_count -gt 0 ]]; then
            echo "Test files:"
            find "$test_dir" -name "*.js" -exec basename {} \; | sed 's/^/  - /'
        fi
        
        echo
        if browserless::validate_injection 2>/dev/null; then
            log::success "✅ Injection is healthy and validated"
        else
            log::warn "⚠️  Injection exists but validation failed"  
            log::info "Try: ./manage.sh --action inject (to recreate)"
        fi
    else
        log::info "❌ No test data found"
        log::info "Run: ./manage.sh --action inject"
    fi
}

#######################################
# Show information about injected test data
#######################################
browserless::show_injection_info() {
    echo
    echo "Test Data Information:"
    echo "  Location: ${BROWSERLESS_DATA_DIR}/test-data/"
    echo "  Files created: validation.js, screenshot-test.js"
    echo
    echo "Next steps:"
    echo "  ./manage.sh --action usage --usage-type screenshot  # Test screenshot API"
    echo "  ./manage.sh --action usage --usage-type all         # Test all APIs"
    echo "  ./manage.sh --action injection-status              # Check injection status"
}

#######################################
# Inject custom function from JSON file
# Arguments:
#   $1 - JSON file path containing function definition
# Returns:
#   0 on success, 1 on failure
#######################################
browserless::inject_function() {
    local file_path="${1:?JSON file path required}"
    
    log::header "🚀 Injecting Custom Function from $file_path"
    
    # Validate file exists and is readable
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if [[ ! -r "$file_path" ]]; then
        log::error "File not readable: $file_path"
        return 1
    fi
    
    # Validate JSON format
    if ! jq . "$file_path" >/dev/null 2>&1; then
        log::error "Invalid JSON format in file: $file_path"
        return 1
    fi
    
    # Initialize function storage if needed
    browserless::init_function_storage
    
    # Extract function metadata
    local function_name
    function_name=$(jq -r '.metadata.name // empty' "$file_path")
    
    if [[ -z "$function_name" ]]; then
        log::error "Function name missing from metadata.name in JSON file"
        return 1
    fi
    
    # Validate function name format
    if ! [[ "$function_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log::error "Invalid function name: $function_name (only alphanumeric, dash, underscore allowed)"
        return 1
    fi
    
    log::info "📝 Function name: $function_name"
    
    # Create function directory
    local function_dir="${BROWSERLESS_DATA_DIR}/functions/${function_name}"
    mkdir -p "$function_dir/executions"
    
    # Extract and validate function code
    local function_code
    function_code=$(jq -r '.function.code // empty' "$file_path")
    
    if [[ -z "$function_code" ]]; then
        log::error "Function code missing from function.code in JSON file"
        return 1
    fi
    
    # Basic JavaScript syntax validation
    if ! echo "$function_code" | node -c 2>/dev/null; then
        log::warn "⚠️  JavaScript syntax validation failed (function may still work in browser context)"
    fi
    
    # Store function files
    cp "$file_path" "$function_dir/manifest.json"
    echo "$function_code" > "$function_dir/function.js"
    
    # Update registry
    browserless::update_function_registry "$function_name" "add"
    
    log::success "✅ Function '$function_name' injected successfully"
    log::info "📁 Stored in: $function_dir"
    log::info "🔧 Execute with: resource-browserless execute $function_name"
    
    return 0
}

#######################################
# Initialize function storage directories and registry
#######################################
browserless::init_function_storage() {
    local functions_dir="${BROWSERLESS_DATA_DIR}/functions"
    local registry_file="${functions_dir}/registry.json"
    
    # Create directories
    mkdir -p "$functions_dir"
    mkdir -p "$functions_dir/templates"
    mkdir -p "$functions_dir/examples"
    
    # Initialize registry if it doesn't exist
    if [[ ! -f "$registry_file" ]]; then
        cat > "$registry_file" << 'EOF'
{
  "version": "1.0.0",
  "functions": {},
  "metadata": {
    "created": "",
    "last_updated": "",
    "total_functions": 0
  }
}
EOF
        # Set creation timestamp
        local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        jq --arg timestamp "$timestamp" '.metadata.created = $timestamp | .metadata.last_updated = $timestamp' "$registry_file" > "${registry_file}.tmp" && mv "${registry_file}.tmp" "$registry_file"
        
        log::debug "Initialized function registry: $registry_file"
    fi
}

#######################################
# Update function registry
# Arguments:
#   $1 - Function name
#   $2 - Operation (add, remove, update)
#######################################
browserless::update_function_registry() {
    local function_name="${1:?Function name required}"
    local operation="${2:?Operation required}"
    local registry_file="${BROWSERLESS_DATA_DIR}/functions/registry.json"
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    case "$operation" in
        "add"|"update")
            # Add or update function entry
            jq --arg name "$function_name" \
               --arg timestamp "$timestamp" \
               '.functions[$name] = {
                   "path": ($name + "/"),
                   "version": "1.0.0",
                   "status": "active",
                   "last_used": null,
                   "execution_count": 0,
                   "created": $timestamp
               } | 
               .metadata.last_updated = $timestamp |
               .metadata.total_functions = (.functions | length)' \
               "$registry_file" > "${registry_file}.tmp" && mv "${registry_file}.tmp" "$registry_file"
            ;;
        "executed")
            # Update execution count and last used time
            jq --arg name "$function_name" \
               --arg timestamp "$timestamp" \
               '.functions[$name].last_used = $timestamp |
               .functions[$name].execution_count = (.functions[$name].execution_count + 1) |
               .metadata.last_updated = $timestamp' \
               "$registry_file" > "${registry_file}.tmp" && mv "${registry_file}.tmp" "$registry_file"
            ;;
        "remove")
            # Remove function entry
            jq --arg name "$function_name" \
               --arg timestamp "$timestamp" \
               'del(.functions[$name]) | 
               .metadata.last_updated = $timestamp |
               .metadata.total_functions = (.functions | length)' \
               "$registry_file" > "${registry_file}.tmp" && mv "${registry_file}.tmp" "$registry_file"
            ;;
        *)
            log::error "Unknown registry operation: $operation"
            return 1
            ;;
    esac
    
    log::debug "Registry updated: $operation $function_name"
    return 0
}

#######################################
# Clean up test data
#######################################
browserless::cleanup_injection() {
    local test_dir="${BROWSERLESS_DATA_DIR}/test-data"
    
    if [[ -d "$test_dir" ]]; then
        log::info "Removing test data directory..."
        rm -rf "$test_dir"
        log::success "✅ Test data cleaned up"
        
        # Also clean up any test screenshots
        local screenshots_dir="${BROWSERLESS_DATA_DIR}/screenshots"
        if [[ -f "$screenshots_dir/validation-test.png" ]]; then
            rm -f "$screenshots_dir/validation-test.png"
            log::info "Removed validation screenshot"
        fi
    else
        log::info "No test data to clean up"
    fi
}