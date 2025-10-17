#!/usr/bin/env bash
# Node-RED Flow Parser for Qdrant Embeddings
# Extracts semantic information from Node-RED flow JSON files
#
# Handles:
# - Flow definitions and node configurations
# - Node types and connections
# - Subflow definitions and usage
# - Integration patterns and APIs
# - Input/output nodes and triggers

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract flow metadata
# 
# Gets basic flow information
#
# Arguments:
#   $1 - Path to Node-RED flow JSON file
# Returns: JSON with flow metadata
#######################################
extractor::lib::node_red::extract_metadata() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Validate JSON format
    if ! jq empty "$file" 2>/dev/null; then
        log::debug "Invalid JSON format in Node-RED flow: $file" >&2
        return 1
    fi
    
    # Extract flow-level metadata
    local flow_name=""
    local flow_id=""
    local version=""
    local disabled="false"
    
    # Node-RED flows can be arrays of flows or single flow objects
    local flow_count=1
    if jq -e 'type == "array"' "$file" >/dev/null 2>/dev/null; then
        flow_count=$(jq 'length' "$file" 2>/dev/null || echo "1")
        # Get first flow's metadata
        flow_name=$(jq -r '.[0].label // .[0].name // ""' "$file" 2>/dev/null)
        flow_id=$(jq -r '.[0].id // ""' "$file" 2>/dev/null)
        disabled=$(jq -r '.[0].disabled // false' "$file" 2>/dev/null)
    else
        flow_name=$(jq -r '.label // .name // ""' "$file" 2>/dev/null)
        flow_id=$(jq -r '.id // ""' "$file" 2>/dev/null)
        disabled=$(jq -r '.disabled // false' "$file" 2>/dev/null)
    fi
    
    # Count total nodes across all flows
    local total_nodes=0
    if jq -e 'type == "array"' "$file" >/dev/null 2>/dev/null; then
        total_nodes=$(jq '[.[] | select(.type != "tab") | select(.type != "subflow")] | length' "$file" 2>/dev/null || echo "0")
    else
        # Single flow format - count nodes
        total_nodes=1
    fi
    
    # Count subflows
    local subflow_count=0
    if jq -e 'type == "array"' "$file" >/dev/null 2>/dev/null; then
        subflow_count=$(jq '[.[] | select(.type == "subflow")] | length' "$file" 2>/dev/null || echo "0")
    fi
    
    # Extract Node-RED version if available
    version=$(jq -r '.version // ""' "$file" 2>/dev/null)
    
    jq -n \
        --arg name "$flow_name" \
        --arg id "$flow_id" \
        --arg version "$version" \
        --arg flow_count "$flow_count" \
        --arg node_count "$total_nodes" \
        --arg subflow_count "$subflow_count" \
        --arg disabled "$disabled" \
        '{
            flow_name: $name,
            flow_id: $id,
            version: $version,
            flow_count: ($flow_count | tonumber),
            total_nodes: ($node_count | tonumber),
            subflow_count: ($subflow_count | tonumber),
            disabled: ($disabled == "true")
        }'
}

#######################################
# Extract node information
# 
# Analyzes flow nodes and their types
#
# Arguments:
#   $1 - Path to Node-RED flow JSON file
# Returns: JSON with node analysis
#######################################
extractor::lib::node_red::extract_nodes() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local node_types=()
    local input_nodes=()
    local output_nodes=()
    local function_nodes=0
    local mqtt_nodes=0
    local http_nodes=0
    local database_nodes=0
    
    # Extract all nodes from flows
    local nodes_data=""
    if jq -e 'type == "array"' "$file" >/dev/null 2>/dev/null; then
        nodes_data=$(jq -c '[.[] | select(.type != "tab")]' "$file")
    else
        nodes_data=$(jq -c '[.]' "$file")
    fi
    
    # Process each node
    echo "$nodes_data" | jq -c '.[]' | while IFS= read -r node; do
        [[ -z "$node" ]] && continue
        
        local node_type=$(echo "$node" | jq -r '.type // ""')
        [[ -z "$node_type" ]] && continue
        
        node_types+=("$node_type")
        
        # Categorize nodes
        case "$node_type" in
            "inject"|"timestamp"|"trigger"|"catch"|"status")
                input_nodes+=("$node_type")
                ;;
            "debug"|"complete"|"link out"|"comment")
                output_nodes+=("$node_type")
                ;;
            "function"|"template"|"change"|"switch"|"range")
                ((function_nodes++))
                ;;
            *"mqtt"*)
                ((mqtt_nodes++))
                ;;
            *"http"*|*"websocket"*)
                ((http_nodes++))
                ;;
            *"mysql"*|*"postgres"*|*"mongodb"*|*"influx"*)
                ((database_nodes++))
                ;;
        esac
    done
    
    # Get unique node types and counts
    local unique_types=($(printf '%s\n' "${node_types[@]}" | sort -u))
    local node_type_counts=()
    
    for type in "${unique_types[@]}"; do
        local count=$(printf '%s\n' "${node_types[@]}" | grep -c "^$type$" || echo "0")
        node_type_counts+=("{\"type\": \"$type\", \"count\": $count}")
    done
    
    local node_types_json="[]"
    if [[ ${#node_type_counts[@]} -gt 0 ]]; then
        node_types_json="[$(IFS=','; echo "${node_type_counts[*]}")]"
    fi
    
    local input_types_json="[]"
    local output_types_json="[]"
    [[ ${#input_nodes[@]} -gt 0 ]] && input_types_json=$(printf '%s\n' "${input_nodes[@]}" | sort -u | jq -R . | jq -s '.')
    [[ ${#output_nodes[@]} -gt 0 ]] && output_types_json=$(printf '%s\n' "${output_nodes[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson node_types "$node_types_json" \
        --argjson input_types "$input_types_json" \
        --argjson output_types "$output_types_json" \
        --arg function_count "$function_nodes" \
        --arg mqtt_count "$mqtt_nodes" \
        --arg http_count "$http_nodes" \
        --arg db_count "$database_nodes" \
        '{
            node_types: $node_types,
            input_node_types: $input_types,
            output_node_types: $output_types,
            function_nodes: ($function_count | tonumber),
            mqtt_nodes: ($mqtt_count | tonumber),
            http_nodes: ($http_count | tonumber),
            database_nodes: ($db_count | tonumber)
        }'
}

#######################################
# Extract integrations
# 
# Identifies external services and protocols
#
# Arguments:
#   $1 - Path to Node-RED flow JSON file
# Returns: JSON with integration information
#######################################
extractor::lib::node_red::extract_integrations() {
    local file="$1"
    local integrations=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"integrations": []}'
        return
    fi
    
    # Extract node types and analyze for integrations
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Check for common integrations by node type
    if echo "$content" | grep -q '"type"[[:space:]]*:[[:space:]]*"mqtt'; then
        integrations+=("mqtt")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*"http|websocket'; then
        integrations+=("http_api")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*sql|mongodb|influx'; then
        integrations+=("database")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*email'; then
        integrations+=("email")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*slack'; then
        integrations+=("slack")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*telegram'; then
        integrations+=("telegram")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*twitter'; then
        integrations+=("twitter")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*aws|amazon'; then
        integrations+=("aws")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*azure'; then
        integrations+=("azure")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*google'; then
        integrations+=("google")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*gpio|raspberry'; then
        integrations+=("hardware")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*serial|modbus'; then
        integrations+=("industrial_protocols")
    fi
    
    # Remove duplicates and create JSON array
    local unique_integrations=($(printf '%s\n' "${integrations[@]}" | sort -u))
    local integrations_json="[]"
    [[ ${#unique_integrations[@]} -gt 0 ]] && integrations_json=$(printf '%s\n' "${unique_integrations[@]}" | jq -R . | jq -s '.')
    
    jq -n --argjson integrations "$integrations_json" '{integrations: $integrations}'
}

#######################################
# Analyze flow purpose
# 
# Determines flow functionality based on nodes and patterns
#
# Arguments:
#   $1 - Path to Node-RED flow JSON file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::node_red::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local filename=$(basename "$file" | tr '[:upper:]' '[:lower:]')
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Analyze filename for hints
    if [[ "$filename" == *"dashboard"* ]]; then
        purposes+=("dashboard")
    elif [[ "$filename" == *"automation"* ]]; then
        purposes+=("automation")
    elif [[ "$filename" == *"monitoring"* ]]; then
        purposes+=("monitoring")
    elif [[ "$filename" == *"api"* ]]; then
        purposes+=("api_integration")
    elif [[ "$filename" == *"iot"* ]]; then
        purposes+=("iot_control")
    elif [[ "$filename" == *"alert"* ]] || [[ "$filename" == *"notification"* ]]; then
        purposes+=("alerting")
    elif [[ "$filename" == *"data"* ]]; then
        purposes+=("data_processing")
    fi
    
    # Analyze content patterns
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*"inject.*cron|schedule'; then
        purposes+=("scheduled_automation")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*"http.*in'; then
        purposes+=("webhook_handler")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*"mqtt'; then
        purposes+=("mqtt_broker_integration")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*sql|mongodb'; then
        purposes+=("database_integration")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*"debug'; then
        purposes+=("development_testing")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*"ui_|dashboard'; then
        purposes+=("user_interface")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*email|slack|telegram'; then
        purposes+=("notification_system")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*"function'; then
        purposes+=("data_transformation")
    fi
    
    if echo "$content" | grep -qE '"type"[[:space:]]*:[[:space:]]*".*gpio|serial'; then
        purposes+=("hardware_control")
    fi
    
    # Determine primary purpose
    local primary_purpose="flow_automation"
    if [[ ${#purposes[@]} -gt 0 ]]; then
        primary_purpose="${purposes[0]}"
    fi
    
    local purposes_json=$(printf '%s\n' "${purposes[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson purposes "$purposes_json" \
        --arg primary "$primary_purpose" \
        '{
            purposes: $purposes,
            primary_purpose: $primary
        }'
}

#######################################
# Extract connection patterns
# 
# Analyzes flow structure and message routing
#
# Arguments:
#   $1 - Path to Node-RED flow JSON file
# Returns: JSON with connection analysis
#######################################
extractor::lib::node_red::extract_connections() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"has_complex_routing": false, "has_loops": false, "has_error_handling": false}'
        return
    fi
    
    # Count wires/connections
    local total_connections=0
    if jq -e 'type == "array"' "$file" >/dev/null 2>/dev/null; then
        total_connections=$(jq '[.[] | .wires? // [] | length] | add // 0' "$file" 2>/dev/null || echo "0")
    fi
    
    # Check for complex routing patterns
    local has_switch_nodes="false"
    if grep -qE '"type"[[:space:]]*:[[:space:]]*"switch"' "$file" 2>/dev/null; then
        has_switch_nodes="true"
    fi
    
    # Check for error handling
    local has_catch_nodes="false"
    if grep -qE '"type"[[:space:]]*:[[:space:]]*"catch"' "$file" 2>/dev/null; then
        has_catch_nodes="true"
    fi
    
    # Check for status monitoring
    local has_status_nodes="false"
    if grep -qE '"type"[[:space:]]*:[[:space:]]*"status"' "$file" 2>/dev/null; then
        has_status_nodes="true"
    fi
    
    # Check for link nodes (flow organization)
    local has_link_nodes="false"
    if grep -qE '"type"[[:space:]]*:[[:space:]]*"link' "$file" 2>/dev/null; then
        has_link_nodes="true"
    fi
    
    jq -n \
        --arg connections "$total_connections" \
        --arg switch_nodes "$has_switch_nodes" \
        --arg catch_nodes "$has_catch_nodes" \
        --arg status_nodes "$has_status_nodes" \
        --arg link_nodes "$has_link_nodes" \
        '{
            total_connections: ($connections | tonumber),
            has_conditional_routing: ($switch_nodes == "true"),
            has_error_handling: ($catch_nodes == "true"),
            has_status_monitoring: ($status_nodes == "true"),
            has_link_organization: ($link_nodes == "true")
        }'
}

#######################################
# Extract all Node-RED flow information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - Node-RED flow file path or directory
#   $2 - Component type (flow, automation, etc.)
#   $3 - Resource name
# Returns: JSON lines with all flow information
#######################################
extractor::lib::node_red::extract_all() {
    local path="$1"
    local component_type="${2:-flow}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a JSON file
        case "$file_ext" in
            json)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Extract all components
        local metadata=$(extractor::lib::node_red::extract_metadata "$file")
        local nodes=$(extractor::lib::node_red::extract_nodes "$file")
        local integrations=$(extractor::lib::node_red::extract_integrations "$file")
        local purpose=$(extractor::lib::node_red::analyze_purpose "$file")
        local connections=$(extractor::lib::node_red::extract_connections "$file")
        
        # Get key metrics
        local flow_name=$(echo "$metadata" | jq -r '.flow_name')
        [[ "$flow_name" == "" || "$flow_name" == "null" ]] && flow_name="$filename"
        
        local node_count=$(echo "$metadata" | jq -r '.total_nodes')
        local flow_count=$(echo "$metadata" | jq -r '.flow_count')
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        local integration_list=$(echo "$integrations" | jq -r '.integrations | join(", ")')
        
        # Build content summary
        local content="Node-RED Flow: $flow_name | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose | Nodes: $node_count"
        [[ $flow_count -gt 1 ]] && content="$content | Flows: $flow_count"
        [[ -n "$integration_list" && "$integration_list" != "" ]] && content="$content | Integrations: $integration_list"
        
        # Check for special features
        local has_error_handling=$(echo "$connections" | jq -r '.has_error_handling')
        [[ "$has_error_handling" == "true" ]] && content="$content | Has Error Handling"
        
        local has_subflows=$(echo "$metadata" | jq -r '.subflow_count')
        [[ $has_subflows -gt 0 ]] && content="$content | Subflows: $has_subflows"
        
        # Output comprehensive flow analysis
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg file_size "$file_size" \
            --argjson metadata "$metadata" \
            --argjson nodes "$nodes" \
            --argjson integrations "$integrations" \
            --argjson purpose "$purpose" \
            --argjson connections "$connections" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    flow_type: "node_red",
                    file_size: ($file_size | tonumber),
                    flow_metadata: $metadata,
                    nodes: $nodes,
                    integrations: $integrations,
                    purpose: $purpose,
                    connections: $connections,
                    content_type: "node_red_flow",
                    extraction_method: "node_red_parser"
                }
            }' | jq -c
            
        # Output entries for each integration (for better searchability)
        echo "$integrations" | jq -r '.integrations[]?' 2>/dev/null | while read -r integration; do
            [[ -z "$integration" || "$integration" == "null" ]] && continue
            
            local int_content="Node-RED Integration: $integration in $flow_name | Resource: $resource_name"
            
            jq -n \
                --arg content "$int_content" \
                --arg resource "$resource_name" \
                --arg source_file "$file" \
                --arg flow_name "$flow_name" \
                --arg integration "$integration" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        flow_name: $flow_name,
                        integration: $integration,
                        component_type: $component_type,
                        content_type: "node_red_integration",
                        extraction_method: "node_red_parser"
                    }
                }' | jq -c
        done
        
    elif [[ -d "$path" ]]; then
        # Directory - find all Node-RED flow files
        local flow_files=()
        while IFS= read -r file; do
            flow_files+=("$file")
        done < <(find "$path" -type f -name "*.json" 2>/dev/null)
        
        if [[ ${#flow_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${flow_files[@]}"; do
            # Check if it's likely a Node-RED flow
            if jq -e 'type == "array" or .type' "$file" >/dev/null 2>/dev/null; then
                extractor::lib::node_red::extract_all "$file" "$component_type" "$resource_name"
            fi
        done
    fi
}

#######################################
# Check if file is a Node-RED flow
# 
# Validates if JSON file is a Node-RED flow definition
#
# Arguments:
#   $1 - File path
# Returns: 0 if Node-RED flow, 1 otherwise
#######################################
extractor::lib::node_red::is_flow() {
    local file="$1"
    
    if [[ ! -f "$file" ]] || [[ ! "$file" == *.json ]]; then
        return 1
    fi
    
    # Check for Node-RED flow structure
    # Node-RED flows are either arrays of flow objects or single flow objects with specific properties
    if jq -e 'type == "array" or (.type and .id)' "$file" >/dev/null 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::node_red::extract_metadata
export -f extractor::lib::node_red::extract_nodes
export -f extractor::lib::node_red::extract_integrations
export -f extractor::lib::node_red::analyze_purpose
export -f extractor::lib::node_red::extract_connections
export -f extractor::lib::node_red::extract_all
export -f extractor::lib::node_red::is_flow