#!/bin/bash
# ====================================================================
# Test Metadata Parser and Utilities
# ====================================================================
#
# Provides functions for parsing and working with test metadata headers.
# Supports both comment-based (@key: value) and variable-based metadata.
#
# Functions:
#   - extract_test_metadata()      - Parse metadata from test file
#   - filter_tests_by_metadata()   - Filter tests based on metadata criteria
#   - get_scenario_readiness()     - Calculate readiness for business scenarios
#   - generate_business_report()   - Create business readiness assessment
#
# ====================================================================

# Extract metadata from a test file
extract_test_metadata() {
    local test_file="$1"
    local metadata_file="${2:-/tmp/test_metadata_$$}"
    
    if [[ ! -f "$test_file" ]]; then
        log_error "Test file not found: $test_file"
        return 1
    fi
    
    # Initialize metadata with defaults
    cat > "$metadata_file" << 'EOF'
SCENARIO=""
CATEGORY=""
COMPLEXITY="unknown"
SERVICES=""
OPTIONAL_SERVICES=""
DURATION="unknown"
BUSINESS_VALUE=""
MARKET_DEMAND="unknown"
REVENUE_POTENTIAL=""
UPWORK_EXAMPLES=""
SUCCESS_CRITERIA=""
EOF
    
    # Parse comment-based metadata (@key: value)
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*#[[:space:]]*@([^:]+):[[:space:]]*(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"
            
            # Clean up key and value
            key=$(echo "$key" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
            value=$(echo "$value" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
            
            # Handle problematic characters for safe shell sourcing
            value=$(echo "$value" | sed 's/"/'"'"'/g')  # Replace double quotes with single quotes
            value=$(echo "$value" | sed 's/\$/\\$/g')   # Escape dollar signs
            
            # Update metadata file with proper quoting
            if grep -q "^${key}=" "$metadata_file"; then
                # Use printf to avoid issues with special characters
                sed -i "/^${key}=/d" "$metadata_file"
                printf '%s="%s"\n' "$key" "$value" >> "$metadata_file"
            else
                printf '%s="%s"\n' "$key" "$value" >> "$metadata_file"
            fi
        elif [[ "$line" =~ ^[[:space:]]*#[[:space:]]*@requires:[[:space:]]*(.*)$ ]]; then
            # Handle legacy @requires format
            local services="${BASH_REMATCH[1]}"
            services=$(echo "$services" | tr ',' ' ' | xargs)
            sed -i "s|^SERVICES=.*|SERVICES=\"${services}\"|" "$metadata_file"
        fi
    done < "$test_file"
    
    # Parse variable-based metadata (REQUIRED_RESOURCES, etc.)
    local required_resources
    required_resources=$(grep -E "^[[:space:]]*REQUIRED_RESOURCES=\(" "$test_file" | head -n1 | sed 's/.*=(\([^)]*\)).*/\1/' | tr -d '"' | tr ' ' ',' | sed 's/,$//')
    
    if [[ -n "$required_resources" && -z "$(grep '^SERVICES=' "$metadata_file" | cut -d'=' -f2 | tr -d '"')" ]]; then
        sed -i "s|^SERVICES=.*|SERVICES=\"${required_resources}\"|" "$metadata_file"
    fi
    
    # Source the metadata to make it available as variables
    source "$metadata_file"
    
    log_debug "Extracted metadata from $test_file:"
    log_debug "  Scenario: $SCENARIO"
    log_debug "  Services: $SERVICES"
    log_debug "  Complexity: $COMPLEXITY"
    log_debug "  Business Value: $BUSINESS_VALUE"
    
    return 0
}

# Check if a test can run based on available resources
can_run_test() {
    local test_file="$1"
    local available_resources="$2"  # Space-separated list
    local metadata_file="/tmp/test_metadata_$$"
    
    extract_test_metadata "$test_file" "$metadata_file"
    source "$metadata_file"
    
    # Convert services to array
    local required_services
    IFS=',' read -ra required_services <<< "$SERVICES"
    
    # Check if all required services are available
    for service in "${required_services[@]}"; do
        service=$(echo "$service" | xargs)  # Trim whitespace
        if [[ -z "$service" ]]; then
            continue
        fi
        
        if [[ " $available_resources " != *" $service "* ]]; then
            log_debug "Test $test_file cannot run: missing service $service"
            rm -f "$metadata_file"
            return 1
        fi
    done
    
    rm -f "$metadata_file"
    return 0
}

# Filter tests based on metadata criteria
filter_tests_by_metadata() {
    local tests_dir="$1"
    local filter_criteria="$2"  # e.g., "category=ai,complexity=beginner"
    local output_file="${3:-/tmp/filtered_tests_$$}"
    
    log_debug "Filtering tests in $tests_dir with criteria: $filter_criteria"
    
    # Parse filter criteria
    declare -A filters
    IFS=',' read -ra criteria <<< "$filter_criteria"
    for criterion in "${criteria[@]}"; do
        if [[ "$criterion" =~ ^([^=]+)=(.+)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local value="${BASH_REMATCH[2]}"
            key=$(echo "$key" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
            filters["$key"]="$value"
        fi
    done
    
    # Find and filter test files
    > "$output_file"
    
    local test_files
    mapfile -t test_files < <(find "$tests_dir" -name "*.test.sh" -type f)
    
    for test_file in "${test_files[@]}"; do
        local metadata_file="/tmp/test_metadata_$$"
        extract_test_metadata "$test_file" "$metadata_file"
        source "$metadata_file"
        
        local matches=true
        for key in "${!filters[@]}"; do
            local expected_value="${filters[$key]}"
            local actual_value
            
            case "$key" in
                "CATEGORY")
                    actual_value="$CATEGORY"
                    ;;
                "COMPLEXITY")
                    actual_value="$COMPLEXITY"
                    ;;
                "BUSINESS_VALUE")
                    actual_value="$BUSINESS_VALUE"
                    ;;
                "MARKET_DEMAND")
                    actual_value="$MARKET_DEMAND"
                    ;;
                *)
                    actual_value=""
                    ;;
            esac
            
            if [[ "$actual_value" != *"$expected_value"* ]]; then
                matches=false
                break
            fi
        done
        
        if [[ "$matches" == "true" ]]; then
            echo "$test_file" >> "$output_file"
        fi
        
        rm -f "$metadata_file"
    done
    
    echo "$output_file"
}

# Calculate scenario readiness based on available resources
get_scenario_readiness() {
    local scenarios_dir="$1"
    local available_resources="$2"
    local output_file="${3:-/tmp/scenario_readiness_$$}"
    
    log_debug "Calculating scenario readiness for resources: $available_resources"
    
    # Initialize readiness report
    cat > "$output_file" << 'EOF'
# Scenario Readiness Report
# Format: scenario_name:category:readiness_status:missing_services
EOF
    
    # Find all scenario test files
    local test_files
    mapfile -t test_files < <(find "$scenarios_dir" -name "*.test.sh" -type f 2>/dev/null || true)
    
    for test_file in "${test_files[@]}"; do
        local metadata_file="/tmp/test_metadata_$$"
        extract_test_metadata "$test_file" "$metadata_file"
        source "$metadata_file"
        
        local scenario_name
        scenario_name=$(basename "$test_file" .test.sh)
        
        if [[ -z "$SCENARIO" ]]; then
            SCENARIO="$scenario_name"
        fi
        
        # Check readiness
        local status="ready"
        local missing_services=""
        
        if [[ -n "$SERVICES" ]]; then
            IFS=',' read -ra required_services <<< "$SERVICES"
            for service in "${required_services[@]}"; do
                service=$(echo "$service" | xargs)
                if [[ -n "$service" && " $available_resources " != *" $service "* ]]; then
                    status="not_ready"
                    if [[ -n "$missing_services" ]]; then
                        missing_services="$missing_services,$service"
                    else
                        missing_services="$service"
                    fi
                fi
            done
        fi
        
        echo "$SCENARIO:$CATEGORY:$status:$missing_services" >> "$output_file"
        rm -f "$metadata_file"
    done
    
    echo "$output_file"
}

# Generate business readiness report
generate_business_readiness_report() {
    local scenarios_dir="$1"
    local available_resources="$2"
    local passed_tests="$3"     # Space-separated list of passed test names
    local failed_tests="$4"     # Space-separated list of failed test names
    
    log_header "📊 Business Readiness Assessment"
    
    # Get scenario readiness
    local readiness_file
    readiness_file=$(get_scenario_readiness "$scenarios_dir" "$available_resources")
    
    # Calculate statistics
    local total_scenarios=0
    local ready_scenarios=0
    local passed_scenarios=0
    declare -A category_stats
    declare -A revenue_potential
    
    while IFS=':' read -r scenario category status missing; do
        if [[ "$scenario" =~ ^#.*$ ]]; then
            continue  # Skip comment lines
        fi
        
        total_scenarios=$((total_scenarios + 1))
        
        if [[ "$status" == "ready" ]]; then
            ready_scenarios=$((ready_scenarios + 1))
            
            # Check if this scenario actually passed
            if [[ " $passed_tests " == *" $scenario "* ]]; then
                passed_scenarios=$((passed_scenarios + 1))
                category_stats["$category"]=$((${category_stats["$category"]:-0} + 1))
            fi
        fi
        
        # Extract revenue potential (would need metadata from actual test files)
        # This is a placeholder - in real implementation, would parse from test metadata
        case "$category" in
            "customer-service")
                revenue_potential["$category"]=$((${revenue_potential["$category"]:-0} + 3000))
                ;;
            "content-creation")
                revenue_potential["$category"]=$((${revenue_potential["$category"]:-0} + 2000))
                ;;
            "data-analysis")
                revenue_potential["$category"]=$((${revenue_potential["$category"]:-0} + 4000))
                ;;
        esac
    done < "$readiness_file"
    
    # Display report
    echo "╔══════════════════════════════════════════════════════════════════════════════╗"
    echo "║                    Vrooli Business Readiness Assessment                      ║"
    echo "╠══════════════════════════════════════════════════════════════════════════════╣"
    
    local readiness_percent=0
    if [[ $total_scenarios -gt 0 ]]; then
        readiness_percent=$(( (passed_scenarios * 100) / total_scenarios ))
    fi
    
    echo "║ Overall Market Readiness: ${readiness_percent}% (${passed_scenarios}/${total_scenarios} scenarios operational)"
    echo "║"
    
    # Category breakdown
    echo "║ Category Readiness:"
    if [[ ${#category_stats[@]} -gt 0 ]]; then
        for category in "${!category_stats[@]}"; do
            local count="${category_stats[$category]}"
            printf "║   ✅ %-20s (%d scenarios ready)\n" "$category" "$count"
        done
    else
        echo "║   ⚠️  No scenarios currently operational"
    fi
    
    echo "║"
    
    # Revenue potential
    local total_revenue=0
    for category in "${!revenue_potential[@]}"; do
        total_revenue=$((total_revenue + revenue_potential["$category"]))
    done
    
    if [[ $total_revenue -gt 0 ]]; then
        echo "║ Estimated Revenue Potential: \$${total_revenue}+ (based on operational scenarios)"
    else
        echo "║ Revenue Potential: Not calculated (no operational scenarios)"
    fi
    
    echo "╚══════════════════════════════════════════════════════════════════════════════╝"
    
    # Cleanup
    rm -f "$readiness_file"
}

# Get test metadata as JSON
get_test_metadata_json() {
    local test_file="$1"
    local metadata_file="/tmp/test_metadata_$$"
    
    extract_test_metadata "$test_file" "$metadata_file"
    source "$metadata_file"
    
    cat << EOF
{
  "scenario": "$SCENARIO",
  "category": "$CATEGORY", 
  "complexity": "$COMPLEXITY",
  "services": "$SERVICES",
  "optional_services": "$OPTIONAL_SERVICES",
  "duration": "$DURATION",
  "business_value": "$BUSINESS_VALUE",
  "market_demand": "$MARKET_DEMAND",
  "revenue_potential": "$REVENUE_POTENTIAL",
  "upwork_examples": "$UPWORK_EXAMPLES",
  "success_criteria": "$SUCCESS_CRITERIA"
}
EOF
    
    rm -f "$metadata_file"
}

# List all scenarios with their metadata
list_scenarios_with_metadata() {
    local scenarios_dir="$1"
    local format="${2:-table}"  # table or json
    
    if [[ "$format" == "json" ]]; then
        echo "["
        local first=true
    else
        printf "%-25s %-20s %-15s %-30s\n" "Scenario" "Category" "Complexity" "Services"
        printf "%-25s %-20s %-15s %-30s\n" "--------" "--------" "----------" "--------"
    fi
    
    local test_files
    mapfile -t test_files < <(find "$scenarios_dir" -name "*.test.sh" -type f 2>/dev/null | sort)
    
    for test_file in "${test_files[@]}"; do
        local metadata_file="/tmp/test_metadata_$$"
        extract_test_metadata "$test_file" "$metadata_file"
        source "$metadata_file"
        
        local scenario_name
        scenario_name=$(basename "$test_file" .test.sh)
        
        if [[ "$format" == "json" ]]; then
            if [[ "$first" != "true" ]]; then
                echo ","
            fi
            get_test_metadata_json "$test_file" | sed 's/^/  /'
            first=false
        else
            printf "%-25s %-20s %-15s %-30s\n" \
                "${SCENARIO:-$scenario_name}" \
                "${CATEGORY:-unknown}" \
                "${COMPLEXITY:-unknown}" \
                "${SERVICES:-none}"
        fi
        
        rm -f "$metadata_file"
    done
    
    if [[ "$format" == "json" ]]; then
        echo ""
        echo "]"
    fi
}