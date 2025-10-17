#!/usr/bin/env bash
# Windmill Flow Parser for Qdrant Embeddings
# Extracts semantic information from Windmill workflow YAML/JSON files
#
# Handles:
# - Flow metadata and configuration
# - Script steps and language detection
# - Dependencies and resource usage
# - Scheduling and triggers
# - Flow purpose analysis

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract flow metadata
# 
# Gets basic flow information from Windmill file
#
# Arguments:
#   $1 - Path to Windmill flow file (YAML/JSON)
# Returns: JSON with flow metadata
#######################################
extractor::lib::windmill::extract_metadata() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    local metadata="{}"
    
    if [[ "$file_ext" == "yaml" ]] || [[ "$file_ext" == "yml" ]]; then
        # Convert YAML to JSON for processing
        if command -v yq &>/dev/null; then
            metadata=$(yq eval -o=json '.' "$file" 2>/dev/null || echo "{}")
        else
            # Fallback: extract key fields with grep/sed
            local name=$(grep "^summary:" "$file" 2>/dev/null | sed 's/summary:[[:space:]]*//' || echo "")
            local description=$(grep "^description:" "$file" 2>/dev/null | sed 's/description:[[:space:]]*//' || echo "")
            metadata=$(jq -n --arg name "$name" --arg desc "$description" '{summary: $name, description: $desc}')
        fi
    elif [[ "$file_ext" == "json" ]]; then
        metadata=$(jq '.' "$file" 2>/dev/null || echo "{}")
    else
        return 1
    fi
    
    # Extract key fields
    local summary=$(echo "$metadata" | jq -r '.summary // .name // "Unnamed Flow"')
    local description=$(echo "$metadata" | jq -r '.description // ""')
    local schema=$(echo "$metadata" | jq -r '.schema // ""')
    local ws_error_handler_muted=$(echo "$metadata" | jq -r '.ws_error_handler_muted // false')
    
    # Count flow modules/steps
    local module_count=$(echo "$metadata" | jq '.value.modules | length // 0' 2>/dev/null || echo "0")
    
    jq -n \
        --arg summary "$summary" \
        --arg description "$description" \
        --arg schema "$schema" \
        --arg error_muted "$ws_error_handler_muted" \
        --arg modules "$module_count" \
        '{
            summary: $summary,
            description: $description,
            schema: $schema,
            error_handler_muted: ($error_muted == "true"),
            module_count: ($modules | tonumber)
        }'
}

#######################################
# Extract script steps
# 
# Analyzes flow modules/steps and their languages
#
# Arguments:
#   $1 - Path to Windmill flow file
# Returns: JSON with step analysis
#######################################
extractor::lib::windmill::extract_steps() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    local flow_data="{}"
    
    # Convert to JSON if needed
    if [[ "$file_ext" == "yaml" ]] || [[ "$file_ext" == "yml" ]]; then
        if command -v yq &>/dev/null; then
            flow_data=$(yq eval -o=json '.' "$file" 2>/dev/null || echo "{}")
        else
            # Simple extraction without yq
            echo '{"steps": [], "languages": []}'
            return
        fi
    else
        flow_data=$(jq '.' "$file" 2>/dev/null || echo "{}")
    fi
    
    # Extract module information
    local modules=$(echo "$flow_data" | jq '.value.modules // []' 2>/dev/null || echo "[]")
    
    # Count step types
    local script_steps=0
    local flow_steps=0
    local approval_steps=0
    local branch_steps=0
    local loop_steps=0
    
    # Languages used
    local languages=()
    
    # Process each module
    echo "$modules" | jq -c '.[]' 2>/dev/null | while IFS= read -r module; do
        local module_type=$(echo "$module" | jq -r '.value | keys[0] // "unknown"')
        
        case "$module_type" in
            "rawscript")
                ((script_steps++))
                local lang=$(echo "$module" | jq -r '.value.rawscript.language // "unknown"')
                languages+=("$lang")
                ;;
            "script")
                ((script_steps++))
                ;;
            "flow")
                ((flow_steps++))
                ;;
            "approval")
                ((approval_steps++))
                ;;
            "branchone"|"branchall")
                ((branch_steps++))
                ;;
            "loop"|"forloop")
                ((loop_steps++))
                ;;
        esac
    done
    
    # Get unique languages
    local unique_langs=$(printf '%s\n' "${languages[@]}" | sort -u | jq -R . | jq -s '.')
    
    # Extract step summaries
    local step_summaries=$(echo "$modules" | jq '[.[] | .summary // .id // "unnamed"]' 2>/dev/null || echo "[]")
    
    jq -n \
        --arg script "$script_steps" \
        --arg flow "$flow_steps" \
        --arg approval "$approval_steps" \
        --arg branch "$branch_steps" \
        --arg loop "$loop_steps" \
        --argjson languages "$unique_langs" \
        --argjson summaries "$step_summaries" \
        '{
            script_steps: ($script | tonumber),
            flow_steps: ($flow | tonumber),
            approval_steps: ($approval | tonumber),
            branch_steps: ($branch | tonumber),
            loop_steps: ($loop | tonumber),
            languages: $languages,
            step_summaries: $summaries
        }'
}

#######################################
# Extract dependencies and resources
# 
# Identifies external dependencies and resource usage
#
# Arguments:
#   $1 - Path to Windmill flow file
# Returns: JSON with dependency information
#######################################
extractor::lib::windmill::extract_dependencies() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"resources": [], "dependencies": []}'
        return
    fi
    
    local file_ext="${file##*.}"
    local flow_data="{}"
    
    # Convert to JSON if needed
    if [[ "$file_ext" == "yaml" ]] || [[ "$file_ext" == "yml" ]]; then
        if command -v yq &>/dev/null; then
            flow_data=$(yq eval -o=json '.' "$file" 2>/dev/null || echo "{}")
        else
            echo '{"resources": [], "dependencies": []}'
            return
        fi
    else
        flow_data=$(jq '.' "$file" 2>/dev/null || echo "{}")
    fi
    
    # Extract resource references
    local resources=$(echo "$flow_data" | jq -r '.. | .resources? // empty | .[]' 2>/dev/null | sort -u | jq -R . | jq -s '.')
    
    # Extract Python dependencies (requirements)
    local python_deps=$(echo "$flow_data" | jq -r '.. | .requirements? // empty | .[]' 2>/dev/null | sort -u | jq -R . | jq -s '.')
    
    # Extract TypeScript/Deno dependencies
    local ts_deps=$(echo "$flow_data" | jq -r '.. | .lock? // empty' 2>/dev/null)
    local has_ts_deps="false"
    [[ -n "$ts_deps" && "$ts_deps" != "null" ]] && has_ts_deps="true"
    
    # Check for database operations
    local has_database="false"
    if echo "$flow_data" | grep -qE "(postgres|mysql|sqlite|mongodb)" 2>/dev/null; then
        has_database="true"
    fi
    
    # Check for API calls
    local has_api_calls="false"
    if echo "$flow_data" | grep -qE "(http|fetch|axios|request)" 2>/dev/null; then
        has_api_calls="true"
    fi
    
    jq -n \
        --argjson resources "$resources" \
        --argjson python_deps "$python_deps" \
        --arg has_ts_deps "$has_ts_deps" \
        --arg has_database "$has_database" \
        --arg has_api_calls "$has_api_calls" \
        '{
            resources: $resources,
            python_dependencies: $python_deps,
            has_typescript_deps: ($has_ts_deps == "true"),
            has_database_ops: ($has_database == "true"),
            has_api_calls: ($has_api_calls == "true")
        }'
}

#######################################
# Extract scheduling and triggers
# 
# Identifies flow scheduling and trigger configuration
#
# Arguments:
#   $1 - Path to Windmill flow file
# Returns: JSON with scheduling information
#######################################
extractor::lib::windmill::extract_scheduling() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"has_schedule": false, "is_webhook": false}'
        return
    fi
    
    local file_content=$(cat "$file" 2>/dev/null)
    
    # Check for schedule configuration
    local has_schedule="false"
    if echo "$file_content" | grep -qE "(schedule|cron|interval)" 2>/dev/null; then
        has_schedule="true"
    fi
    
    # Check for webhook trigger
    local is_webhook="false"
    if echo "$file_content" | grep -qE "(webhook|http_trigger)" 2>/dev/null; then
        is_webhook="true"
    fi
    
    # Check for event triggers
    local has_events="false"
    if echo "$file_content" | grep -qE "(on_success|on_failure|on_event)" 2>/dev/null; then
        has_events="true"
    fi
    
    # Check for manual trigger only
    local is_manual="false"
    if [[ "$has_schedule" == "false" && "$is_webhook" == "false" && "$has_events" == "false" ]]; then
        is_manual="true"
    fi
    
    jq -n \
        --arg schedule "$has_schedule" \
        --arg webhook "$is_webhook" \
        --arg events "$has_events" \
        --arg manual "$is_manual" \
        '{
            has_schedule: ($schedule == "true"),
            is_webhook: ($webhook == "true"),
            has_event_triggers: ($events == "true"),
            is_manual_only: ($manual == "true")
        }'
}

#######################################
# Analyze flow purpose
# 
# Determines flow functionality based on content
#
# Arguments:
#   $1 - Path to Windmill flow file
# Returns: JSON with purpose analysis
#######################################
extractor::lib::windmill::analyze_purpose() {
    local file="$1"
    local purposes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"purposes": [], "primary_purpose": "unknown"}'
        return
    fi
    
    local file_content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]')
    local filename=$(basename "$file" | tr '[:upper:]' '[:lower:]')
    
    # Check for ETL patterns
    if echo "$file_content" | grep -qE "(extract|transform|load|etl|pipeline)"; then
        purposes+=("etl_pipeline")
    fi
    
    # Check for data processing
    if echo "$file_content" | grep -qE "(process|analyze|aggregate|compute)"; then
        purposes+=("data_processing")
    fi
    
    # Check for automation
    if echo "$file_content" | grep -qE "(automate|scheduled|cron|trigger)"; then
        purposes+=("automation")
    fi
    
    # Check for API/webhook handling
    if echo "$file_content" | grep -qE "(webhook|api|endpoint|request|response)"; then
        purposes+=("api_handler")
    fi
    
    # Check for notification
    if echo "$file_content" | grep -qE "(notify|alert|email|slack|discord)"; then
        purposes+=("notification")
    fi
    
    # Check for deployment
    if echo "$file_content" | grep -qE "(deploy|release|rollout|ci/cd)"; then
        purposes+=("deployment")
    fi
    
    # Check for monitoring
    if echo "$file_content" | grep -qE "(monitor|check|health|status|metric)"; then
        purposes+=("monitoring")
    fi
    
    # Check for backup
    if echo "$file_content" | grep -qE "(backup|snapshot|archive|restore)"; then
        purposes+=("backup")
    fi
    
    # Check for ML/AI
    if echo "$file_content" | grep -qE "(train|predict|model|ai|ml|neural)"; then
        purposes+=("machine_learning")
    fi
    
    # Check filename for hints
    if [[ "$filename" == *"sync"* ]]; then
        purposes+=("synchronization")
    elif [[ "$filename" == *"report"* ]]; then
        purposes+=("reporting")
    elif [[ "$filename" == *"test"* ]]; then
        purposes+=("testing")
    fi
    
    # Determine primary purpose
    local primary_purpose="general_workflow"
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
# Extract all Windmill flow information
# 
# Main extraction function that combines all analyses
#
# Arguments:
#   $1 - Windmill flow file path or directory
#   $2 - Component type (flow, automation, etc.)
#   $3 - Resource name
# Returns: JSON lines with all flow information
#######################################
extractor::lib::windmill::extract_all() {
    local path="$1"
    local component_type="${2:-flow}"
    local resource_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a supported file type
        case "$file_ext" in
            yaml|yml|json)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
        
        # Extract all components
        local metadata=$(extractor::lib::windmill::extract_metadata "$file")
        local steps=$(extractor::lib::windmill::extract_steps "$file")
        local dependencies=$(extractor::lib::windmill::extract_dependencies "$file")
        local scheduling=$(extractor::lib::windmill::extract_scheduling "$file")
        local purpose=$(extractor::lib::windmill::analyze_purpose "$file")
        
        # Get key metrics
        local flow_name=$(echo "$metadata" | jq -r '.summary')
        local module_count=$(echo "$metadata" | jq -r '.module_count')
        local primary_purpose=$(echo "$purpose" | jq -r '.primary_purpose')
        local languages=$(echo "$steps" | jq -r '.languages | join(", ")')
        
        # Build content summary
        local content="Windmill Flow: $flow_name | Type: $component_type | Resource: $resource_name"
        content="$content | Purpose: $primary_purpose | Steps: $module_count"
        [[ -n "$languages" && "$languages" != "" ]] && content="$content | Languages: $languages"
        
        # Check for special features
        local has_branches=$(echo "$steps" | jq -r '.branch_steps')
        [[ $has_branches -gt 0 ]] && content="$content | Has Branching"
        
        local has_loops=$(echo "$steps" | jq -r '.loop_steps')
        [[ $has_loops -gt 0 ]] && content="$content | Has Loops"
        
        local has_schedule=$(echo "$scheduling" | jq -r '.has_schedule')
        [[ "$has_schedule" == "true" ]] && content="$content | Scheduled"
        
        # Output comprehensive flow analysis
        jq -n \
            --arg content "$content" \
            --arg resource "$resource_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg file_size "$file_size" \
            --arg line_count "$line_count" \
            --argjson metadata "$metadata" \
            --argjson steps "$steps" \
            --argjson dependencies "$dependencies" \
            --argjson scheduling "$scheduling" \
            --argjson purpose "$purpose" \
            '{
                content: $content,
                metadata: {
                    resource: $resource,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    workflow_type: "windmill",
                    file_size: ($file_size | tonumber),
                    line_count: ($line_count | tonumber),
                    flow_metadata: $metadata,
                    steps: $steps,
                    dependencies: $dependencies,
                    scheduling: $scheduling,
                    purpose: $purpose,
                    content_type: "windmill_flow",
                    extraction_method: "windmill_parser"
                }
            }' | jq -c
            
        # Output entries for each language used (for better searchability)
        echo "$steps" | jq -r '.languages[]' 2>/dev/null | while read -r lang; do
            [[ -z "$lang" || "$lang" == "null" ]] && continue
            
            local lang_content="Windmill $lang Script in $flow_name | Resource: $resource_name"
            
            jq -n \
                --arg content "$lang_content" \
                --arg resource "$resource_name" \
                --arg source_file "$file" \
                --arg flow_name "$flow_name" \
                --arg language "$lang" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        resource: $resource,
                        source_file: $source_file,
                        flow_name: $flow_name,
                        language: $language,
                        component_type: $component_type,
                        content_type: "windmill_script",
                        extraction_method: "windmill_parser"
                    }
                }' | jq -c
        done
        
    elif [[ -d "$path" ]]; then
        # Directory - find all Windmill flow files
        local flow_files=()
        while IFS= read -r file; do
            flow_files+=("$file")
        done < <(find "$path" -type f \( -name "*.yaml" -o -name "*.yml" -o -name "*.json" \) 2>/dev/null)
        
        if [[ ${#flow_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${flow_files[@]}"; do
            # Check if it's likely a Windmill flow (basic heuristic)
            if grep -qE "(summary:|value:|modules:|rawscript:)" "$file" 2>/dev/null; then
                extractor::lib::windmill::extract_all "$file" "$component_type" "$resource_name"
            fi
        done
    fi
}

#######################################
# Check if file is a Windmill flow
# 
# Validates if file is a Windmill flow definition
#
# Arguments:
#   $1 - File path
# Returns: 0 if Windmill flow, 1 otherwise
#######################################
extractor::lib::windmill::is_flow() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local file_ext="${file##*.}"
    
    case "$file_ext" in
        yaml|yml|json)
            # Check for Windmill flow markers
            if grep -qE "(summary:|value:|modules:|rawscript:)" "$file" 2>/dev/null; then
                return 0
            fi
            ;;
    esac
    
    return 1
}

# Export all functions
export -f extractor::lib::windmill::extract_metadata
export -f extractor::lib::windmill::extract_steps
export -f extractor::lib::windmill::extract_dependencies
export -f extractor::lib::windmill::extract_scheduling
export -f extractor::lib::windmill::analyze_purpose
export -f extractor::lib::windmill::extract_all
export -f extractor::lib::windmill::is_flow