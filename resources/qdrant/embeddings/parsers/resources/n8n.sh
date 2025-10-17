#!/usr/bin/env bash
# N8n Workflow Parser for Qdrant Embeddings
# Extracts semantic information from n8n workflow JSON files
#
# Handles:
# - Workflow metadata and configuration
# - Node types and connections
# - Webhook endpoints and triggers
# - Integration detection
# - Workflow purpose analysis

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract workflow metadata
# 
# Gets basic workflow information
#
# Arguments:
#   $1 - Path to n8n workflow JSON file
# Returns: JSON with workflow metadata
#######################################
extractor::lib::n8n::extract_metadata() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Validate JSON format
    if ! jq empty "$file" 2>/dev/null; then
        log::debug "Invalid JSON format in n8n workflow: $file" >&2
        return 1
    fi
    
    # Extract basic metadata
    local name=$(jq -r '.name // "Unnamed Workflow"' "$file" 2>/dev/null)
    local active=$(jq -r '.active // false' "$file" 2>/dev/null)
    local node_count=$(jq '.nodes | length' "$file" 2>/dev/null || echo "0")
    local settings=$(jq -c '.settings // {}' "$file" 2>/dev/null || echo "{}")
    
    # Extract workflow ID if present
    local workflow_id=$(jq -r '.id // ""' "$file" 2>/dev/null)
    
    # Check for tags
    local tags=$(jq -c '.tags // []' "$file" 2>/dev/null || echo "[]")
    
    jq -n \
        --arg name "$name" \
        --arg active "$active" \
        --arg node_count "$node_count" \
        --arg workflow_id "$workflow_id" \
        --argjson settings "$settings" \
        --argjson tags "$tags" \
        '{
            name: $name,
            active: ($active == "true"),
            node_count: ($node_count | tonumber),
            workflow_id: $workflow_id,
            settings: $settings,
            tags: $tags
        }'
}

#######################################
# Extract node information
# 
# Analyzes workflow nodes and their types
#
# Arguments:
#   $1 - Path to n8n workflow JSON file
# Returns: JSON with node analysis
#######################################
extractor::lib::n8n::extract_nodes() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Count node types
    local node_types=$(jq -r '.nodes[].type' "$file" 2>/dev/null | sort | uniq -c | \
        awk '{print "{\"type\": \"" $2 "\", \"count\": " $1 "}"}' | jq -s '.')
    
    # Extract webhook nodes
    local webhooks=$(jq -r '.nodes[] | select(.type == "n8n-nodes-base.webhook") | .name' "$file" 2>/dev/null | jq -R . | jq -s '.')
    
    # Extract trigger nodes
    local triggers=$(jq -r '.nodes[] | select(.type | contains("trigger")) | .type' "$file" 2>/dev/null | sort -u | jq -R . | jq -s '.')
    
    # Find HTTP/API nodes
    local http_nodes=$(jq -r '.nodes[] | select(.type | contains("http") or contains("api")) | .type' "$file" 2>/dev/null | sort -u | jq -R . | jq -s '.')
    
    # Find database nodes
    local db_nodes=$(jq -r '.nodes[] | select(.type | contains("postgres") or contains("mysql") or contains("mongo") or contains("redis")) | .type' "$file" 2>/dev/null | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson node_types "$node_types" \
        --argjson webhooks "$webhooks" \
        --argjson triggers "$triggers" \
        --argjson http_nodes "$http_nodes" \
        --argjson db_nodes "$db_nodes" \
        '{
            node_types: $node_types,
            webhooks: $webhooks,
            triggers: $triggers,
            http_nodes: $http_nodes,
            database_nodes: $db_nodes
        }'
}

#######################################
# Detect integrations
# 
# Identifies external services and APIs
#
# Arguments:
#   $1 - Path to n8n workflow JSON file
# Returns: JSON with integration information
#######################################
extractor::lib::n8n::detect_integrations() {
    local file="$1"
    local integrations=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"integrations": []}'
        return
    fi
    
    # Check for common integrations
    local node_types=$(jq -r '.nodes[].type' "$file" 2>/dev/null | sort -u)
    
    while IFS= read -r node_type; do
        case "$node_type" in
            *slack*) integrations+=("slack") ;;
            *github*) integrations+=("github") ;;
            *google*) integrations+=("google") ;;
            *aws*) integrations+=("aws") ;;
            *stripe*) integrations+=("stripe") ;;
            *sendgrid*) integrations+=("sendgrid") ;;
            *twilio*) integrations+=("twilio") ;;
            *discord*) integrations+=("discord") ;;
            *telegram*) integrations+=("telegram") ;;
            *jira*) integrations+=("jira") ;;
            *notion*) integrations+=("notion") ;;
            *airtable*) integrations+=("airtable") ;;
            *salesforce*) integrations+=("salesforce") ;;
            *hubspot*) integrations+=("hubspot") ;;
            *mailchimp*) integrations+=("mailchimp") ;;
            *trello*) integrations+=("trello") ;;
            *asana*) integrations+=("asana") ;;
            *openai*) integrations+=("openai") ;;
            *anthropic*) integrations+=("anthropic") ;;
        esac
    done <<< "$node_types"
    
    # Remove duplicates and create JSON array
    local unique_integrations=$(printf '%s\n' "${integrations[@]}" | sort -u | jq -R . | jq -s '.')
    
    jq -n \
        --argjson integrations "$unique_integrations" \
        '{integrations: $integrations}'
}

#######################################
# Analyze workflow purpose
# 
# Determines workflow functionality based on nodes
#
# Arguments:
#   $1 - Path to n8n workflow JSON file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::n8n::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local node_types=$(jq -r '.nodes[].type' "$file" 2>/dev/null)
    local workflow_name=$(jq -r '.name // ""' "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    
    # Check for automation patterns
    if echo "$node_types" | grep -q "schedule"; then
        purposes+=("scheduled_automation")
    fi
    
    if echo "$node_types" | grep -q "webhook"; then
        purposes+=("webhook_handler")
    fi
    
    if echo "$node_types" | grep -q "email"; then
        purposes+=("email_automation")
    fi
    
    # Check for data processing
    if echo "$node_types" | grep -qE "(split|merge|aggregate|transform)"; then
        purposes+=("data_processing")
    fi
    
    # Check for ETL patterns
    if echo "$node_types" | grep -qE "(postgres|mysql|mongo)" && echo "$node_types" | grep -qE "(http|api)"; then
        purposes+=("etl_pipeline")
    fi
    
    # Check for notification patterns
    if echo "$node_types" | grep -qE "(slack|discord|telegram|email|sms)"; then
        purposes+=("notification_system")
    fi
    
    # Check for AI/ML patterns
    if echo "$node_types" | grep -qE "(openai|anthropic|huggingface|gpt)"; then
        purposes+=("ai_workflow")
    fi
    
    # Check workflow name for hints
    if [[ "$workflow_name" == *"backup"* ]]; then
        purposes+=("backup_automation")
    elif [[ "$workflow_name" == *"sync"* ]]; then
        purposes+=("data_synchronization")
    elif [[ "$workflow_name" == *"monitor"* ]]; then
        purposes+=("monitoring")
    elif [[ "$workflow_name" == *"deploy"* ]]; then
        purposes+=("deployment_automation")
    fi
    
    # Determine primary purpose
    local primary_purpose="general_automation"
    if [[ ${#purposes[@]} -gt 0 ]]; then
        primary_purpose="${purposes[0]}"
    fi
    
    local purposes_json=$(printf '%s\n' "${purposes[@]}" | jq -R . | jq -s '.')
    
    jq -n \
        --argjson purposes "$purposes_json" \
        --arg primary "$primary_purpose" \
        '{
            purposes: $purposes,
            primary_purpose: $primary
        }'
}

#######################################
# Extract connections and flow
# 
# Analyzes workflow connections and data flow
#
# Arguments:
#   $1 - Path to n8n workflow JSON file
# Returns: JSON with connection analysis
#######################################
extractor::lib::n8n::extract_connections() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"connection_count": 0, "has_branches": false, "has_loops": false}'
        return
    fi
    
    # Count connections
    local connection_count=$(jq '.connections | to_entries | map(.value | to_entries | map(.value | length)) | flatten | add' "$file" 2>/dev/null || echo "0")
    
    # Check for branching (IF nodes or multiple outputs)
    local has_branches="false"
    if jq -e '.nodes[] | select(.type | contains("if") or contains("switch"))' "$file" >/dev/null 2>/dev/null; then
        has_branches="true"
    fi
    
    # Check for loops (connections going backwards)
    local has_loops="false"
    # Simple heuristic: check if any node connects to a previous node
    # This is a simplified check - proper loop detection would need graph analysis
    
    # Check for error handling
    local has_error_handling="false"
    if jq -e '.nodes[] | select(.type | contains("error"))' "$file" >/dev/null 2>/dev/null; then
        has_error_handling="true"
    fi
    
    jq -n \
        --arg connections "$connection_count" \
        --arg branches "$has_branches" \
        --arg loops "$has_loops" \
        --arg error_handling "$has_error_handling" \
        '{
            connection_count: ($connections | tonumber),
            has_branches: ($branches == "true"),
            has_loops: ($loops == "true"),
            has_error_handling: ($error_handling == "true")
        }'
}

#######################################
# Extract all n8n workflow information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - N8n workflow file path or directory
#   $2 - Component type (workflow, automation, etc.)
#   $3 - Resource name
# Returns: JSON lines with all workflow information
#######################################
extractor::lib::n8n::extract_all() {
    local path="$1"
    local component_type="${2:-workflow}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        
        # Validate it's a JSON file
        if [[ ! "$filename" == *.json ]]; then
            return 1
        fi
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Extract all components
        local metadata=$(extractor::lib::n8n::extract_metadata "$file")
        local nodes=$(extractor::lib::n8n::extract_nodes "$file")
        local integrations=$(extractor::lib::n8n::detect_integrations "$file")
        local purpose=$(extractor::lib::n8n::analyze_purpose "$file")
        local connections=$(extractor::lib::n8n::extract_connections "$file")
        
        # Get key metrics
        local workflow_name=$(echo "$metadata" | jq -r '.name')
        local node_count=$(echo "$metadata" | jq -r '.node_count')
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        local integration_list=$(echo "$integrations" | jq -r '.integrations | join(", ")')
        
        # Build content summary
        local content="N8n Workflow: $workflow_name | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose | Nodes: $node_count"
        [[ -n "$integration_list" && "$integration_list" != "" ]] && content="$content | Integrations: $integration_list"
        
        # Check for special features
        local has_webhooks=$(echo "$nodes" | jq '.webhooks | length')
        [[ $has_webhooks -gt 0 ]] && content="$content | Has Webhooks"
        
        local has_branches=$(echo "$connections" | jq -r '.has_branches')
        [[ "$has_branches" == "true" ]] && content="$content | Has Conditional Logic"
        
        # Output comprehensive workflow analysis
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
                    workflow_type: "n8n",
                    file_size: ($file_size | tonumber),
                    workflow_metadata: $metadata,
                    nodes: $nodes,
                    integrations: $integrations,
                    purpose: $purpose,
                    connections: $connections,
                    content_type: "n8n_workflow",
                    extraction_method: "n8n_parser"
                }
            }' | jq -c
            
        # Output entries for specific integrations (for better searchability)
        echo "$integrations" | jq -r '.integrations[]' 2>/dev/null | while read -r integration; do
            [[ -z "$integration" ]] && continue
            
            local int_content="N8n Integration: $integration in $workflow_name | Resource: $resource_name"
            
            jq -n \
                --arg content "$int_content" \
                --arg resource "$resource_name" \
                --arg source_file "$file" \
                --arg workflow_name "$workflow_name" \
                --arg integration "$integration" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        workflow_name: $workflow_name,
                        integration: $integration,
                        component_type: $component_type,
                        content_type: "n8n_integration",
                        extraction_method: "n8n_parser"
                    }
                }' | jq -c
        done
        
    elif [[ -d "$path" ]]; then
        # Directory - find all workflow JSON files
        local workflow_files=()
        while IFS= read -r file; do
            workflow_files+=("$file")
        done < <(find "$path" -type f -name "*.json" 2>/dev/null)
        
        if [[ ${#workflow_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${workflow_files[@]}"; do
            extractor::lib::n8n::extract_all "$file" "$component_type" "$resource_name"
        done
    fi
}

#######################################
# Check if file is an n8n workflow
# 
# Validates if JSON file is an n8n workflow
#
# Arguments:
#   $1 - File path
# Returns: 0 if n8n workflow, 1 otherwise
#######################################
extractor::lib::n8n::is_workflow() {
    local file="$1"
    
    if [[ ! -f "$file" ]] || [[ ! "$file" == *.json ]]; then
        return 1
    fi
    
    # Check for n8n workflow structure
    if jq -e '.nodes and .connections' "$file" >/dev/null 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::n8n::extract_metadata
export -f extractor::lib::n8n::extract_nodes
export -f extractor::lib::n8n::detect_integrations
export -f extractor::lib::n8n::analyze_purpose
export -f extractor::lib::n8n::extract_connections
export -f extractor::lib::n8n::extract_all
export -f extractor::lib::n8n::is_workflow