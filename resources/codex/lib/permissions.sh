#!/usr/bin/env bash
################################################################################
# Codex Permission Management System
# 
# Handles tool permissions, risk assessment, and security validation
# Provides claude-code compatible permission patterns
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source settings management
source "${APP_ROOT}/resources/codex/lib/settings.sh"

################################################################################
# Core Permission Validation
################################################################################

#######################################
# Check if a tool is allowed to execute
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   0 if allowed, 1 if blocked
#######################################
permissions::is_tool_allowed() {
    local tool_name="$1"
    local arguments="$2"
    
    # Get current allowed tools
    local allowed_tools="${CODEX_ALLOWED_TOOLS:-$(codex_settings::get "tools.allowed" | jq -r '. | join(",")' 2>/dev/null)}"
    
    # If no restrictions are set, default to safe profile
    if [[ -z "$allowed_tools" || "$allowed_tools" == "null" ]]; then
        allowed_tools="read_file,list_files,analyze_code"
        log::debug "No tool restrictions found, defaulting to safe profile"
    fi
    
    # Check for wildcard permission (admin access)
    if [[ "$allowed_tools" =~ \*$ ]] || [[ "$allowed_tools" == "*" ]]; then
        log::debug "Wildcard permission granted for $tool_name"
        return 0
    fi
    
    # Check if tool is explicitly allowed
    if permissions::matches_tool_pattern "$tool_name" "$arguments" "$allowed_tools"; then
        log::debug "Tool $tool_name is allowed by current permissions"
        return 0
    fi
    
    # Tool is not allowed
    log::debug "Tool $tool_name is blocked by current permissions"
    permissions::log_violation "$tool_name" "$arguments"
    return 1
}

#######################################
# Check if tool matches permission patterns
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
#   $3 - Allowed tools pattern (comma-separated)
# Returns:
#   0 if matches, 1 if not
#######################################
permissions::matches_tool_pattern() {
    local tool_name="$1"
    local arguments="$2"
    local allowed_patterns="$3"
    
    # Convert comma-separated list to array
    IFS=',' read -ra patterns <<< "$allowed_patterns"
    
    for pattern in "${patterns[@]}"; do
        pattern=$(echo "$pattern" | xargs)  # Trim whitespace
        
        # Check exact tool name match
        if [[ "$pattern" == "$tool_name" ]]; then
            return 0
        fi
        
        # Check pattern with arguments (e.g., execute_command(git *))
        if [[ "$pattern" == *"("* ]] && [[ "$pattern" == *")" ]]; then
            local pattern_tool="${pattern%%(*}"
            local pattern_args="${pattern#*\(}"
            pattern_args="${pattern_args%\)}"
            
            if [[ "$pattern_tool" == "$tool_name" ]]; then
                if permissions::matches_argument_pattern "$arguments" "$pattern_args"; then
                    return 0
                fi
            fi
        fi
        
        # Check wildcard patterns
        if [[ "$pattern" =~ \*$ ]] && [[ "$tool_name" == "${pattern%\*}"* ]]; then
            return 0
        fi
    done
    
    return 1
}

#######################################
# Check if tool arguments match permission pattern
# Arguments:
#   $1 - Tool arguments (JSON)
#   $2 - Pattern to match against
# Returns:
#   0 if matches, 1 if not
#######################################
permissions::matches_argument_pattern() {
    local arguments="$1"
    local pattern="$2"
    
    # Handle wildcard pattern
    if [[ "$pattern" == "*" ]]; then
        return 0
    fi
    
    # Extract command from arguments for execute_command tool
    local command
    if echo "$arguments" | jq -e '.command' &>/dev/null; then
        command=$(echo "$arguments" | jq -r '.command // ""')
    else
        # For other tools, use path or other relevant fields
        command=$(echo "$arguments" | jq -r '.path // .url // .content // ""' 2>/dev/null || echo "")
    fi
    
    # Pattern matching logic
    case "$pattern" in
        "git *")
            [[ "$command" =~ ^git[[:space:]] ]]
            ;;
        "npm *")
            [[ "$command" =~ ^npm[[:space:]] ]]
            ;;
        "rm *")
            [[ "$command" =~ ^rm[[:space:]] ]]
            ;;
        "sudo *")
            [[ "$command" =~ ^sudo[[:space:]] ]]
            ;;
        *)
            # Exact match or shell pattern matching
            [[ "$command" == $pattern ]]
            ;;
    esac
}

################################################################################
# Risk Assessment System
################################################################################

#######################################
# Assess risk level for a tool operation
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   Risk level: LOW, MEDIUM, HIGH, CRITICAL
#######################################
permissions::assess_risk_level() {
    local tool_name="$1"
    local arguments="$2"
    
    case "$tool_name" in
        "read_file"|"list_files"|"analyze_code")
            echo "LOW"
            ;;
        "write_file")
            echo "MEDIUM"
            ;;
        "execute_command")
            permissions::assess_command_risk "$arguments"
            ;;
        *)
            echo "MEDIUM"
            ;;
    esac
}

#######################################
# Assess risk level for command execution
# Arguments:
#   $1 - Command arguments (JSON)
# Returns:
#   Risk level: LOW, MEDIUM, HIGH, CRITICAL
#######################################
permissions::assess_command_risk() {
    local arguments="$1"
    local command=$(echo "$arguments" | jq -r '.command // ""')
    
    # CRITICAL risk patterns
    local critical_patterns=(
        'rm.*-rf'
        'sudo.*rm'
        'chmod.*777'
        'chown'
        'passwd'
        'mkfs'
        'fdisk'
        'dd.*if=/dev'
        'reboot'
        'shutdown'
        'halt'
        'poweroff'
        'kill.*-9.*1'
    )
    
    for pattern in "${critical_patterns[@]}"; do
        if [[ "$command" =~ $pattern ]]; then
            echo "CRITICAL"
            return
        fi
    done
    
    # HIGH risk patterns  
    local high_patterns=(
        'git.*push'
        'npm.*publish'
        'rm'
        'mv.*/'
        'cp.*-r'
        'wget.*\|'
        'curl.*\|'
        'chmod'
    )
    
    for pattern in "${high_patterns[@]}"; do
        if [[ "$command" =~ $pattern ]]; then
            echo "HIGH"
            return
        fi
    done
    
    # MEDIUM risk patterns
    local medium_patterns=(
        'git.*commit'
        'npm.*install'
        'pip.*install'
        'cargo.*install'
        'make'
        'mvn'
        'gradle'
    )
    
    for pattern in "${medium_patterns[@]}"; do
        if [[ "$command" =~ $pattern ]]; then
            echo "MEDIUM"
            return
        fi
    done
    
    # Default to LOW for safe commands
    echo "LOW"
}

################################################################################
# Confirmation System
################################################################################

#######################################
# Check if tool operation requires confirmation
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   0 if confirmation required, 1 if not
#######################################
permissions::requires_confirmation() {
    local tool_name="$1"
    local arguments="$2"
    
    # Skip confirmations in non-interactive environments
    if [[ "${CODEX_SKIP_CONFIRMATIONS:-}" == "true" ]] || [[ ! -t 0 ]]; then
        return 1
    fi
    
    # Get confirmation requirements
    local require_confirmation="${CODEX_REQUIRE_CONFIRMATION:-$(codex_settings::get "execution.require_confirmation")}"
    
    # Check if tool requires confirmation
    if [[ "$require_confirmation" =~ $tool_name ]] || [[ "$require_confirmation" == "true" ]]; then
        return 0
    fi
    
    # Check risk-based confirmation
    local risk_level=$(permissions::assess_risk_level "$tool_name" "$arguments")
    case "$risk_level" in
        "HIGH"|"CRITICAL")
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

#######################################
# Prompt user for confirmation
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
#   $3 - Risk level
# Returns:
#   0 if confirmed, 1 if denied
#######################################
permissions::prompt_confirmation() {
    local tool_name="$1"
    local arguments="$2"  
    local risk_level="$3"
    
    # Format operation description
    local operation_desc
    case "$tool_name" in
        "execute_command")
            local command=$(echo "$arguments" | jq -r '.command // ""')
            operation_desc="Execute command: $command"
            ;;
        "write_file")
            local path=$(echo "$arguments" | jq -r '.path // ""')
            operation_desc="Write file: $path"
            ;;
        *)
            operation_desc="Execute tool: $tool_name"
            ;;
    esac
    
    # Color-code by risk level
    local color_code=""
    case "$risk_level" in
        "LOW") color_code="\033[0;32m" ;;      # Green
        "MEDIUM") color_code="\033[0;33m" ;;   # Yellow
        "HIGH") color_code="\033[0;31m" ;;     # Red
        "CRITICAL") color_code="\033[1;31m" ;; # Bold Red
    esac
    
    echo
    echo -e "${color_code}[${risk_level}] About to perform operation:\033[0m"
    echo "  $operation_desc"
    echo
    
    # Get user confirmation
    local prompt="Continue? (y/N): "
    if [[ "$risk_level" == "CRITICAL" ]]; then
        prompt="This is a CRITICAL operation. Are you absolutely sure? (type 'yes'): "
        read -p "$prompt" -r
        if [[ "$REPLY" == "yes" ]]; then
            return 0
        else
            return 1
        fi
    else
        read -p "$prompt" -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            return 0
        else
            return 1
        fi
    fi
}

################################################################################
# Audit and Logging
################################################################################

#######################################
# Log tool execution for audit trail
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
#   $3 - Risk level  
#   $4 - Status (ALLOWED, BLOCKED, CONFIRMED, DENIED)
# Returns:
#   0 always
#######################################
permissions::log_execution() {
    local tool_name="$1"
    local arguments="$2"
    local risk_level="$3"
    local status="$4"
    
    # Skip logging if disabled
    local audit_logging=$(codex_settings::get "security.audit_logging" 2>/dev/null)
    if [[ "$audit_logging" == "false" ]]; then
        return 0
    fi
    
    # Create log entry
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local user="${USER:-unknown}"
    local working_dir="$(pwd)"
    
    # Extract key argument for logging (command, path, etc.)
    local key_arg=""
    if echo "$arguments" | jq -e '.command' &>/dev/null; then
        key_arg=$(echo "$arguments" | jq -r '.command // ""')
    elif echo "$arguments" | jq -e '.path' &>/dev/null; then
        key_arg=$(echo "$arguments" | jq -r '.path // ""')
    fi
    
    # Create log entry
    local log_entry="$timestamp [$risk_level] $tool_name: $key_arg [$status] (user: $user, cwd: $working_dir)"
    
    # Append to audit log
    local audit_file="${CODEX_AUDIT_FILE:-${HOME}/.codex/audit.log}"
    mkdir -p "$(dirname "$audit_file")"
    echo "$log_entry" >> "$audit_file"
    
    # Rotate log if it gets too large (>10MB)
    if [[ -f "$audit_file" ]] && [[ $(wc -c < "$audit_file") -gt 10485760 ]]; then
        permissions::rotate_audit_log "$audit_file"
    fi
    
    return 0
}

#######################################
# Log permission violation
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   0 always
#######################################
permissions::log_violation() {
    local tool_name="$1"
    local arguments="$2"
    
    local risk_level=$(permissions::assess_risk_level "$tool_name" "$arguments")
    permissions::log_execution "$tool_name" "$arguments" "$risk_level" "BLOCKED"
    
    # Also log to stderr for immediate visibility
    log::warn "Permission denied: $tool_name (not in allowed tools list)"
    
    return 0
}

#######################################
# Rotate audit log file
# Arguments:
#   $1 - Audit log file path
# Returns:
#   0 on success, 1 on failure
#######################################
permissions::rotate_audit_log() {
    local audit_file="$1"
    local backup_file="${audit_file}.$(date +%Y%m%d-%H%M%S)"
    
    if mv "$audit_file" "$backup_file"; then
        log::info "Audit log rotated to: $backup_file"
        
        # Keep only the last 5 rotated logs
        ls -t "${audit_file}".* | tail -n +6 | xargs -r rm
        
        return 0
    else
        log::error "Failed to rotate audit log"
        return 1
    fi
}

################################################################################
# Permission Enforcement Integration
################################################################################

#######################################
# Main permission check function
# Called by tool registry before execution
# Arguments:
#   $1 - Tool name
#   $2 - Tool arguments (JSON)
# Returns:
#   0 if allowed to proceed, 1 if blocked
#######################################
permissions::check_and_confirm() {
    local tool_name="$1"
    local arguments="$2"
    
    # Step 1: Check basic permissions
    if ! permissions::is_tool_allowed "$tool_name" "$arguments"; then
        return 1
    fi
    
    # Step 2: Assess risk level
    local risk_level=$(permissions::assess_risk_level "$tool_name" "$arguments")
    
    # Step 3: Check if confirmation is required
    if permissions::requires_confirmation "$tool_name" "$arguments"; then
        if permissions::prompt_confirmation "$tool_name" "$arguments" "$risk_level"; then
            permissions::log_execution "$tool_name" "$arguments" "$risk_level" "CONFIRMED"
            return 0
        else
            permissions::log_execution "$tool_name" "$arguments" "$risk_level" "DENIED"
            return 1
        fi
    else
        permissions::log_execution "$tool_name" "$arguments" "$risk_level" "ALLOWED"
        return 0
    fi
}

################################################################################
# Utility Functions
################################################################################

#######################################
# Show current permission status
# Returns:
#   Permission summary
#######################################
permissions::show_status() {
    log::header "🔐 Codex Permission Status"
    
    echo "Current Profile: ${CODEX_PROFILE:-default}"
    echo "Allowed Tools: ${CODEX_ALLOWED_TOOLS:-not set}"
    echo "Max Turns: ${CODEX_MAX_TURNS:-not set}"
    echo "Timeout: ${CODEX_TIMEOUT:-not set}"
    echo "Confirmations: ${CODEX_REQUIRE_CONFIRMATION:-not set}"
    echo
    
    local audit_logging=$(codex_settings::get "security.audit_logging" 2>/dev/null)
    if [[ "$audit_logging" != "false" ]]; then
        local audit_file="${CODEX_AUDIT_FILE:-${HOME}/.codex/audit.log}"
        if [[ -f "$audit_file" ]]; then
            local log_size=$(wc -l < "$audit_file")
            echo "Audit Log: $audit_file ($log_size entries)"
        else
            echo "Audit Log: Not initialized"
        fi
    else
        echo "Audit Log: Disabled"
    fi
}

# Export functions
export -f permissions::is_tool_allowed
export -f permissions::check_and_confirm
export -f permissions::assess_risk_level
export -f permissions::show_status
export -f permissions::log_execution