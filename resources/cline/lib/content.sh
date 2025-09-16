#!/usr/bin/env bash
# Cline Content Management Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CLINE_LIB_DIR="${APP_ROOT}/resources/cline/lib"

# Source required utilities
# shellcheck disable=SC1091
source "${CLINE_LIB_DIR}/common.sh"
# Agent management is now handled via unified system (see cli.sh)

#######################################
# Add content to Cline (templates, configs, etc.)
# Args: <type> <name> [file]
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::add() {
    local type="${1:-}"
    local name="${2:-}"
    local file="${3:-}"
    
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Usage: content add <type> <name> [file]"
        log::info "Types: template, config, prompt"
        return 1
    fi
    
    # Ensure directories exist
    mkdir -p "$CLINE_DATA_DIR/$type"
    
    case "$type" in
        template|templates)
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                cp "$file" "$CLINE_DATA_DIR/templates/$name"
                log::success "Template added: $name"
            else
                log::error "Template file not found: $file"
                return 1
            fi
            ;;
        config|configs)
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                if jq empty "$file" 2>/dev/null; then
                    cp "$file" "$CLINE_DATA_DIR/configs/$name.json"
                    log::success "Configuration added: $name"
                else
                    log::error "Invalid JSON configuration file"
                    return 1
                fi
            else
                log::error "Configuration file not found: $file"
                return 1
            fi
            ;;
        prompt|prompts)
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                cp "$file" "$CLINE_DATA_DIR/prompts/$name"
                log::success "Prompt added: $name"
            else
                log::error "Prompt file not found: $file"
                return 1
            fi
            ;;
        *)
            log::error "Unknown content type: $type"
            log::info "Available types: template, config, prompt"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# List content in Cline
# Args: [type]
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::list() {
    local type="${1:-all}"
    
    if [[ ! -d "$CLINE_DATA_DIR" ]]; then
        log::info "No content directory found. Use 'content add' to add content."
        return 0
    fi
    
    case "$type" in
        all)
            log::info "Cline Content:"
            for content_type in templates configs prompts; do
                if [[ -d "$CLINE_DATA_DIR/$content_type" ]]; then
                    local count=$(find "$CLINE_DATA_DIR/$content_type" -type f 2>/dev/null | wc -l)
                    if [[ $count -gt 0 ]]; then
                        echo "  $content_type/ ($count files)"
                        find "$CLINE_DATA_DIR/$content_type" -type f -printf "    %f\n" 2>/dev/null | sort
                    fi
                fi
            done
            ;;
        template|templates)
            if [[ -d "$CLINE_DATA_DIR/templates" ]]; then
                log::info "Templates:"
                find "$CLINE_DATA_DIR/templates" -type f -printf "  %f\n" 2>/dev/null | sort
            else
                log::info "No templates found"
            fi
            ;;
        config|configs)
            if [[ -d "$CLINE_DATA_DIR/configs" ]]; then
                log::info "Configurations:"
                find "$CLINE_DATA_DIR/configs" -type f -printf "  %f\n" 2>/dev/null | sort
            else
                log::info "No configurations found"
            fi
            ;;
        prompt|prompts)
            if [[ -d "$CLINE_DATA_DIR/prompts" ]]; then
                log::info "Prompts:"
                find "$CLINE_DATA_DIR/prompts" -type f -printf "  %f\n" 2>/dev/null | sort
            else
                log::info "No prompts found"
            fi
            ;;
        *)
            log::error "Unknown content type: $type"
            log::info "Available types: all, template, config, prompt"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Get content from Cline
# Args: <type> <name>
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::get() {
    local type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Usage: content get <type> <name>"
        log::info "Types: template, config, prompt"
        return 1
    fi
    
    local file_path
    case "$type" in
        template|templates)
            file_path="$CLINE_DATA_DIR/templates/$name"
            ;;
        config|configs)
            file_path="$CLINE_DATA_DIR/configs/$name.json"
            if [[ ! -f "$file_path" ]]; then
                file_path="$CLINE_DATA_DIR/configs/$name"
            fi
            ;;
        prompt|prompts)
            file_path="$CLINE_DATA_DIR/prompts/$name"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    if [[ -f "$file_path" ]]; then
        log::info "Content: $type/$name"
        echo "---"
        cat "$file_path"
    else
        log::error "Content not found: $type/$name"
        return 1
    fi
    
    return 0
}

#######################################
# Remove content from Cline
# Args: <type> <name>
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::remove() {
    local type="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$type" ]] || [[ -z "$name" ]]; then
        log::error "Usage: content remove <type> <name>"
        log::info "Types: template, config, prompt"
        return 1
    fi
    
    local file_path
    case "$type" in
        template|templates)
            file_path="$CLINE_DATA_DIR/templates/$name"
            ;;
        config|configs)
            file_path="$CLINE_DATA_DIR/configs/$name.json"
            if [[ ! -f "$file_path" ]]; then
                file_path="$CLINE_DATA_DIR/configs/$name"
            fi
            ;;
        prompt|prompts)
            file_path="$CLINE_DATA_DIR/prompts/$name"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    if [[ -f "$file_path" ]]; then
        rm "$file_path"
        log::success "Content removed: $type/$name"
    else
        log::error "Content not found: $type/$name"
        return 1
    fi
    
    return 0
}

#######################################
# Setup agent cleanup on signals
# Arguments:
#   $1 - Agent ID
#######################################
cline::setup_agent_cleanup() {
    local agent_id="$1"
    
    # Export the agent ID so trap can access it
    export CLINE_CURRENT_AGENT_ID="$agent_id"
    
    # Cleanup function that uses the exported variable
    cline::agent_cleanup() {
        if [[ -n "${CLINE_CURRENT_AGENT_ID:-}" ]] && type -t agents::unregister &>/dev/null; then
            agents::unregister "${CLINE_CURRENT_AGENT_ID}" >/dev/null 2>&1
        fi
        exit 0
    }
    
    # Register cleanup for common signals
    trap 'cline::agent_cleanup' EXIT SIGTERM SIGINT
}

#######################################
# Execute content using Cline (business functionality)
# Args: <action> [args...]
# Returns:
#   0 on success, 1 on failure
#######################################
cline::content::execute() {
    local action="${1:-}"
    shift || true
    
    # Register agent if agent management is available
    local agent_id=""
    if type -t agents::register &>/dev/null; then
        agent_id=$(agents::generate_id)
        local command_string="resource-cline content execute $action $*"
        if agents::register "$agent_id" $$ "$command_string"; then
            log::debug "Registered agent: $agent_id"
            
            # Set up signal handler for cleanup
            cline::setup_agent_cleanup "$agent_id"
        fi
    fi
    
    case "$action" in
        open|launch)
            # Open VS Code with Cline if available
            if cline::check_vscode && cline::is_installed; then
                log::info "Opening VS Code with Cline..."
                code --command "cline.openChat" 2>/dev/null || code
            else
                log::error "VS Code or Cline extension not available"
                return 1
            fi
            ;;
        configure)
            # Configure Cline with current settings
            cline::configure
            ;;
        chat)
            # Information about starting a chat session
            log::info "To start a Cline chat session:"
            log::info "1. Open VS Code"
            log::info "2. Press Cmd/Ctrl+Shift+P"
            log::info "3. Type 'Cline: Open Chat'"
            log::info "4. Select your provider and model"
            log::info "5. Start coding with AI assistance!"
            ;;
        prompt)
            # Send a prompt directly to the AI provider from terminal
            local prompt="$*"
            if [[ -z "$prompt" ]]; then
                log::error "Please provide a prompt"
                return 1
            fi
            
            local provider=$(cline::get_provider)
            log::info "Sending prompt to $provider..."
            
            # For now, echo the prompt - in future this would integrate with the API
            log::info "Prompt: $prompt"
            log::info "Note: Direct API integration coming soon. Use VS Code for now."
            ;;
        provider)
            # Switch or show provider
            local new_provider="${1:-}"
            if [[ -n "$new_provider" ]]; then
                cline::config_provider "$new_provider"
            else
                log::info "Current provider: $(cline::get_provider)"
                log::info "Available: ollama, openrouter, anthropic, openai, google"
            fi
            ;;
        models)
            # List available models for the current provider
            local provider=$(cline::get_provider)
            log::info "Models for $provider:"
            
            case "$provider" in
                ollama)
                    if timeout 5 curl -sf http://localhost:11434/api/tags 2>/dev/null | jq -r '.models[].name' 2>/dev/null; then
                        :
                    else
                        log::warn "Cannot fetch Ollama models (is Ollama running?)"
                    fi
                    ;;
                openrouter)
                    log::info "Popular OpenRouter models:"
                    log::info "  - anthropic/claude-3.5-sonnet"
                    log::info "  - openai/gpt-4-turbo"
                    log::info "  - meta-llama/llama-3.1-70b"
                    log::info "  - google/gemini-pro-1.5"
                    ;;
                *)
                    log::info "Model listing not implemented for $provider"
                    ;;
            esac
            ;;
        context)
            # Workspace context management
            local subaction="${1:-load}"
            shift || true
            case "$subaction" in
                load)
                    # Load workspace context
                    local workspace_path="${1:-$(pwd)}"
                    if [[ ! -d "$workspace_path" ]]; then
                        log::error "Invalid workspace path: $workspace_path"
                        return 1
                    fi
                    
                    log::info "Loading workspace context from: $workspace_path"
                    
                    # Create context file for the workspace
                    local context_file="$CLINE_DATA_DIR/contexts/$(basename "$workspace_path").json"
                    mkdir -p "$CLINE_DATA_DIR/contexts"
                    
                    # Analyze workspace structure
                    local file_count=$(find "$workspace_path" -type f -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.go" 2>/dev/null | wc -l)
                    local total_size=$(du -sh "$workspace_path" 2>/dev/null | cut -f1)
                    
                    # Create context JSON
                    cat > "$context_file" <<EOF
{
  "workspace": "$workspace_path",
  "loaded_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "statistics": {
    "file_count": $file_count,
    "total_size": "$total_size",
    "languages": $(find "$workspace_path" -type f \( -name "*.js" -o -name "*.ts" -o -name "*.py" -o -name "*.go" \) -exec basename {} \; 2>/dev/null | sed 's/.*\.//' | sort -u | jq -R . | jq -s .)
  },
  "important_files": [
    $(find "$workspace_path" -maxdepth 3 -type f \( -name "package.json" -o -name "README.md" -o -name "Makefile" -o -name "go.mod" -o -name "requirements.txt" \) 2>/dev/null | head -10 | jq -R . | jq -s 'join(",\n    ")')
  ],
  "active": true
}
EOF
                    log::success "Workspace context loaded: $(basename "$workspace_path")"
                    log::info "  Files: $file_count"
                    log::info "  Size: $total_size"
                    log::info "Context saved to: $context_file"
                    ;;
                
                show)
                    # Show current context
                    if [[ -d "$CLINE_DATA_DIR/contexts" ]]; then
                        local active_context=$(find "$CLINE_DATA_DIR/contexts" -name "*.json" -exec jq -r 'select(.active == true) | .workspace' {} \; 2>/dev/null | head -1)
                        if [[ -n "$active_context" ]]; then
                            log::info "Active workspace context: $active_context"
                            local context_file="$CLINE_DATA_DIR/contexts/$(basename "$active_context").json"
                            if [[ -f "$context_file" ]]; then
                                jq '.' "$context_file"
                            fi
                        else
                            log::info "No active workspace context"
                        fi
                    else
                        log::info "No workspace contexts loaded"
                    fi
                    ;;
                
                clear)
                    # Clear context
                    if [[ -d "$CLINE_DATA_DIR/contexts" ]]; then
                        find "$CLINE_DATA_DIR/contexts" -name "*.json" -exec jq '.active = false' {} \; -exec sponge {} \; 2>/dev/null || {
                            # Fallback if sponge is not available
                            for file in "$CLINE_DATA_DIR/contexts"/*.json; do
                                if [[ -f "$file" ]]; then
                                    local tmp_file=$(mktemp)
                                    jq '.active = false' "$file" > "$tmp_file" && mv "$tmp_file" "$file"
                                fi
                            done
                        }
                        log::success "Workspace context cleared"
                    else
                        log::info "No workspace contexts to clear"
                    fi
                    ;;
                
                list)
                    # List all contexts
                    if [[ -d "$CLINE_DATA_DIR/contexts" ]]; then
                        log::info "Available workspace contexts:"
                        for context_file in "$CLINE_DATA_DIR/contexts"/*.json; do
                            if [[ -f "$context_file" ]]; then
                                local workspace=$(jq -r '.workspace' "$context_file")
                                local loaded_at=$(jq -r '.loaded_at' "$context_file")
                                local active=$(jq -r '.active' "$context_file")
                                local marker=""
                                [[ "$active" == "true" ]] && marker=" [ACTIVE]"
                                echo "  - $(basename "$workspace"): $workspace (loaded: $loaded_at)$marker"
                            fi
                        done
                    else
                        log::info "No workspace contexts found"
                    fi
                    ;;
                
                *)
                    log::error "Unknown context action: $subaction"
                    log::info "Available: load <path>, show, clear, list"
                    return 1
                    ;;
            esac
            ;;
        
        instructions)
            # Custom instructions management
            local subaction="${1:-list}"
            shift || true
            
            case "$subaction" in
                add)
                    # Add custom instruction
                    local name="${1:-}"
                    local instruction="${2:-}"
                    
                    if [[ -z "$name" ]] || [[ -z "$instruction" ]]; then
                        log::error "Usage: instructions add <name> <instruction>"
                        return 1
                    fi
                    
                    mkdir -p "$CLINE_DATA_DIR/instructions"
                    echo "$instruction" > "$CLINE_DATA_DIR/instructions/$name.txt"
                    
                    # Also save metadata
                    cat > "$CLINE_DATA_DIR/instructions/$name.json" <<EOF
{
  "name": "$name",
  "instruction": "$instruction",
  "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "active": true
}
EOF
                    log::success "Custom instruction added: $name"
                    ;;
                
                list)
                    # List custom instructions
                    log::info "Custom Instructions:"
                    if [[ -d "$CLINE_DATA_DIR/instructions" ]]; then
                        for inst_file in "$CLINE_DATA_DIR/instructions"/*.json; do
                            if [[ -f "$inst_file" ]]; then
                                local name=$(jq -r '.name' "$inst_file")
                                local active=$(jq -r '.active' "$inst_file")
                                local marker=""
                                [[ "$active" == "true" ]] && marker=" [ACTIVE]"
                                echo "  - $name$marker"
                            fi
                        done 2>/dev/null || echo "  No custom instructions found"
                    else
                        echo "  No custom instructions found"
                    fi
                    ;;
                
                show)
                    # Show specific instruction
                    local name="${1:-}"
                    if [[ -z "$name" ]]; then
                        log::error "Usage: instructions show <name>"
                        return 1
                    fi
                    
                    if [[ -f "$CLINE_DATA_DIR/instructions/$name.txt" ]]; then
                        log::info "Instruction: $name"
                        echo "---"
                        cat "$CLINE_DATA_DIR/instructions/$name.txt"
                    else
                        log::error "Instruction not found: $name"
                        return 1
                    fi
                    ;;
                
                remove)
                    # Remove instruction
                    local name="${1:-}"
                    if [[ -z "$name" ]]; then
                        log::error "Usage: instructions remove <name>"
                        return 1
                    fi
                    
                    if [[ -f "$CLINE_DATA_DIR/instructions/$name.txt" ]]; then
                        rm -f "$CLINE_DATA_DIR/instructions/$name.txt"
                        rm -f "$CLINE_DATA_DIR/instructions/$name.json"
                        log::success "Instruction removed: $name"
                    else
                        log::error "Instruction not found: $name"
                        return 1
                    fi
                    ;;
                
                activate)
                    # Activate an instruction
                    local name="${1:-}"
                    if [[ -z "$name" ]]; then
                        log::error "Usage: instructions activate <name>"
                        return 1
                    fi
                    
                    if [[ -f "$CLINE_DATA_DIR/instructions/$name.json" ]]; then
                        local tmp_file=$(mktemp)
                        jq '.active = true' "$CLINE_DATA_DIR/instructions/$name.json" > "$tmp_file" && mv "$tmp_file" "$CLINE_DATA_DIR/instructions/$name.json"
                        log::success "Instruction activated: $name"
                    else
                        log::error "Instruction not found: $name"
                        return 1
                    fi
                    ;;
                
                *)
                    log::error "Unknown instructions action: $subaction"
                    log::info "Available: add <name> <instruction>, list, show <name>, remove <name>, activate <name>"
                    return 1
                    ;;
            esac
            ;;
        
        batch)
            # Batch operations on multiple files
            local operation="${1:-analyze}"
            shift || true
            
            case "$operation" in
                analyze)
                    # Analyze multiple files
                    local pattern="${1:-*.js}"
                    local search_dir="${2:-$(pwd)}"
                    
                    log::info "Analyzing files matching: $pattern in $search_dir"
                    
                    local batch_report="$CLINE_DATA_DIR/batch_reports/$(date +%Y%m%d_%H%M%S).json"
                    mkdir -p "$CLINE_DATA_DIR/batch_reports"
                    
                    # Find matching files
                    local files_found=0
                    local total_lines=0
                    local file_list=()
                    
                    while IFS= read -r file; do
                        ((files_found++))
                        file_list+=("$file")
                        local lines=$(wc -l < "$file" 2>/dev/null || echo 0)
                        ((total_lines+=lines))
                    done < <(find "$search_dir" -type f -name "$pattern" 2>/dev/null | head -100)
                    
                    # Create batch report
                    {
                        echo "{"
                        echo "  \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\","
                        echo "  \"pattern\": \"$pattern\","
                        echo "  \"directory\": \"$search_dir\","
                        echo "  \"files_found\": $files_found,"
                        echo "  \"total_lines\": $total_lines,"
                        echo "  \"files\": ["
                        
                        local first=true
                        for file in "${file_list[@]}"; do
                            [[ "$first" == "true" ]] && first=false || echo ","
                            echo -n "    \"$file\""
                        done
                        
                        echo ""
                        echo "  ]"
                        echo "}"
                    } > "$batch_report"
                    
                    log::success "Batch analysis complete:"
                    log::info "  Files found: $files_found"
                    log::info "  Total lines: $total_lines"
                    log::info "  Report saved: $batch_report"
                    ;;
                
                process)
                    # Process files with AI operations (stub for now)
                    local pattern="${1:-*.js}"
                    local operation_type="${2:-refactor}"
                    
                    log::info "Batch processing files: $pattern"
                    log::info "Operation: $operation_type"
                    
                    # Create batch job
                    local batch_id="batch_$(date +%Y%m%d_%H%M%S)"
                    local batch_dir="$CLINE_DATA_DIR/batch_jobs/$batch_id"
                    mkdir -p "$batch_dir"
                    
                    # Find files to process
                    local files_count=0
                    find "$(pwd)" -type f -name "$pattern" 2>/dev/null | head -20 | while read -r file; do
                        ((files_count++))
                        echo "$file" >> "$batch_dir/files.txt"
                    done
                    
                    # Create batch configuration
                    cat > "$batch_dir/config.json" <<EOF
{
  "batch_id": "$batch_id",
  "pattern": "$pattern",
  "operation": "$operation_type",
  "status": "queued",
  "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "files_count": $(wc -l < "$batch_dir/files.txt" 2>/dev/null || echo 0)
}
EOF
                    
                    log::success "Batch job created: $batch_id"
                    log::info "Files queued: $(wc -l < "$batch_dir/files.txt" 2>/dev/null || echo 0)"
                    log::info "Use 'content execute batch status $batch_id' to check progress"
                    ;;
                
                status)
                    # Check batch job status
                    local batch_id="${1:-}"
                    
                    if [[ -z "$batch_id" ]]; then
                        # List all batch jobs
                        log::info "Batch jobs:"
                        if [[ -d "$CLINE_DATA_DIR/batch_jobs" ]]; then
                            for job_dir in "$CLINE_DATA_DIR/batch_jobs"/batch_*; do
                                if [[ -f "$job_dir/config.json" ]]; then
                                    local id=$(basename "$job_dir")
                                    local status=$(jq -r '.status' "$job_dir/config.json")
                                    local files=$(jq -r '.files_count' "$job_dir/config.json")
                                    echo "  $id: $status ($files files)"
                                fi
                            done
                        else
                            echo "  No batch jobs found"
                        fi
                    else
                        # Show specific batch job
                        local batch_dir="$CLINE_DATA_DIR/batch_jobs/$batch_id"
                        if [[ -f "$batch_dir/config.json" ]]; then
                            log::info "Batch job: $batch_id"
                            jq '.' "$batch_dir/config.json"
                        else
                            log::error "Batch job not found: $batch_id"
                            return 1
                        fi
                    fi
                    ;;
                
                *)
                    log::error "Unknown batch operation: $operation"
                    log::info "Available: analyze <pattern>, process <pattern> <operation>, status [batch_id]"
                    return 1
                    ;;
            esac
            ;;
        
        analytics|usage)
            # Usage analytics tracking
            local subaction="${1:-show}"
            shift || true
            
            # Initialize analytics file if needed
            local analytics_file="$CLINE_DATA_DIR/analytics.json"
            if [[ ! -f "$analytics_file" ]]; then
                mkdir -p "$CLINE_DATA_DIR"
                echo '{"sessions": [], "total_tokens": 0, "total_cost": 0}' > "$analytics_file"
            fi
            
            case "$subaction" in
                show)
                    log::info "Cline Usage Analytics:"
                    if [[ -f "$analytics_file" ]]; then
                        local total_tokens=$(jq -r '.total_tokens // 0' "$analytics_file")
                        local total_cost=$(jq -r '.total_cost // 0' "$analytics_file")
                        local session_count=$(jq -r '.sessions | length' "$analytics_file")
                        
                        echo "  Total sessions: $session_count"
                        echo "  Total tokens: $total_tokens"
                        echo "  Estimated cost: \$$total_cost"
                        
                        # Show recent sessions
                        if [[ $session_count -gt 0 ]]; then
                            echo ""
                            echo "Recent sessions:"
                            jq -r '.sessions | sort_by(.timestamp) | reverse | .[0:5] | .[] | "  \(.timestamp): \(.provider) - \(.tokens) tokens (\(.model))"' "$analytics_file" 2>/dev/null || echo "  No session details available"
                        fi
                    else
                        log::info "No analytics data available yet"
                    fi
                    ;;
                
                track)
                    # Track a new session (would be called by Cline integration)
                    local provider="${1:-ollama}"
                    local model="${2:-unknown}"
                    local tokens="${3:-0}"
                    local cost="${4:-0}"
                    
                    local new_session=$(jq -n \
                        --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
                        --arg prov "$provider" \
                        --arg mod "$model" \
                        --argjson tok "$tokens" \
                        --argjson cst "$cost" \
                        '{timestamp: $ts, provider: $prov, model: $mod, tokens: $tok, cost: $cst}')
                    
                    jq --argjson session "$new_session" \
                        '.sessions += [$session] | .total_tokens += $session.tokens | .total_cost += $session.cost' \
                        "$analytics_file" > "${analytics_file}.tmp" && mv "${analytics_file}.tmp" "$analytics_file"
                    
                    log::success "Session tracked: $provider/$model - $tokens tokens"
                    ;;
                
                reset)
                    echo '{"sessions": [], "total_tokens": 0, "total_cost": 0}' > "$analytics_file"
                    log::success "Analytics data reset"
                    ;;
                
                *)
                    log::error "Unknown analytics action: $subaction"
                    log::info "Available: show, track <provider> <model> <tokens> <cost>, reset"
                    return 1
                    ;;
            esac
            ;;
        
        status)
            # Show detailed status for execution context
            cline::status
            ;;
        *)
            log::error "Unknown execute action: $action"
            log::info "Available actions: open, configure, chat, prompt, provider, models, context, analytics, batch, instructions, status"
            return 1
            ;;
    esac
    
    # Unregister agent on success
    if [[ -n "$agent_id" ]] && type -t agents::unregister &>/dev/null; then
        agents::unregister "$agent_id" >/dev/null 2>&1
    fi
    
    return 0
}

# Main entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-list}" in
        add)
            shift
            cline::content::add "$@"
            ;;
        list)
            shift
            cline::content::list "$@"
            ;;
        get)
            shift
            cline::content::get "$@"
            ;;
        remove)
            shift
            cline::content::remove "$@"
            ;;
        execute)
            shift
            cline::content::execute "$@"
            ;;
        *)
            echo "Usage: $0 [add|list|get|remove|execute] [args...]"
            exit 1
            ;;
    esac
fi