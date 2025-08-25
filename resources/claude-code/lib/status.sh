#!/usr/bin/env bash
# Claude Code Status Functions - Standardized Format
# Handles status checking, info display, and log viewing with JSON support

# Set CLAUDE_CODE_SCRIPT_DIR if not already set (for BATS test compatibility)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
CLAUDE_CODE_SCRIPT_DIR="${CLAUDE_CODE_SCRIPT_DIR:-${APP_ROOT}/resources/claude-code}"

# Source var.sh for directory variables if not already sourced
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/format.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CLAUDE_CODE_SCRIPT_DIR}/../../lib/status-args.sh"
# shellcheck disable=SC1091
source "${CLAUDE_CODE_SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CLAUDE_CODE_SCRIPT_DIR}/common.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v claude_code::export_config &>/dev/null; then
    claude_code::export_config 2>/dev/null || true
fi

#######################################
# Collect Claude Code status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
claude_code::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local health_message="Unknown"
    local version="unknown"
    local auth_status="unknown"
    local tty_compatible="unknown"
    
    # Check if Claude Code is installed
    if claude_code::is_installed; then
        installed="true"
        version=$(claude_code::get_version 2>/dev/null || echo "unknown")
        
        # Since Claude Code is a CLI tool, it's "running" if installed and available
        running="true"
        
        # Perform health check to determine if healthy (skip if fast mode)
        if [[ "$fast_mode" == "false" ]]; then
            local health_output
            if health_output=$(timeout 3s claude_code::health_check "basic" "json" 2>/dev/null); then
                local health_status
                health_status=$(echo "$health_output" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown")
                auth_status=$(echo "$health_output" | jq -r '.auth_status // "unknown"' 2>/dev/null || echo "unknown")
                tty_compatible=$(echo "$health_output" | jq -r '.tty_compatible // "unknown"' 2>/dev/null || echo "unknown")
                
                case "$health_status" in
                    "healthy")
                        healthy="true"
                        health_message="Healthy - Claude Code CLI is functional and authenticated"
                        ;;
                    "degraded")
                        healthy="false"
                        health_message="Degraded - Claude Code installed but has issues"
                        ;;
                    *)
                        healthy="false"
                        health_message="Unhealthy - Claude Code not functioning properly"
                        ;;
                esac
            else
                healthy="false"
                health_message="Health check failed - Unable to determine status"
            fi
        else
            # Fast mode - assume healthy if installed
            healthy="true"
            health_message="Fast mode - skipped health check"
            auth_status="N/A"
            tty_compatible="N/A"
        fi
    else
        health_message="Not installed - Claude Code CLI not found"
    fi
    
    # Basic resource information
    status_data+=("name" "claude-code")
    status_data+=("category" "agents")
    status_data+=("description" "Anthropic's official CLI for Claude AI assistant")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("version" "$version")
    
    # Service endpoints and configuration
    status_data+=("cli_command" "claude")
    status_data+=("package_name" "${CLAUDE_PACKAGE:-@anthropic-ai/claude-code}")
    status_data+=("min_node_version" "${MIN_NODE_VERSION:-18}")
    
    # Configuration details
    status_data+=("config_dir" "${CLAUDE_CONFIG_DIR:-$HOME/.claude}")
    status_data+=("sessions_dir" "${CLAUDE_SESSIONS_DIR:-$HOME/.claude/sessions}")
    
    # Runtime information (only if installed)
    if [[ "$installed" == "true" ]]; then
        # Node.js version check (skip if fast mode)
        local node_available="false"
        local node_version="unknown"
        if [[ "$fast_mode" == "false" ]]; then
            if claude_code::check_node_version; then
                node_available="true"
                node_version=$(node --version 2>/dev/null || echo "unknown")
            fi
        else
            node_available="N/A"
            node_version="N/A"
        fi
        status_data+=("node_available" "$node_available")
        status_data+=("node_version" "$node_version")
        
        # Authentication and TTY status
        status_data+=("auth_status" "$auth_status")
        status_data+=("tty_compatible" "$tty_compatible")
        
        # Configuration file existence
        local config_exists="false"
        local project_config_exists="false"
        if [[ -f "${CLAUDE_SETTINGS_FILE:-$HOME/.claude/settings.json}" ]]; then
            config_exists="true"
        fi
        if [[ -f "${CLAUDE_PROJECT_SETTINGS:-$(pwd)/.claude/settings.json}" ]]; then
            project_config_exists="true"
        fi
        status_data+=("config_exists" "$config_exists")
        status_data+=("project_config_exists" "$project_config_exists")
        
        # Log directory status (skip if fast mode)
        local log_dirs_found=0
        if [[ "$fast_mode" == "false" ]]; then
            for log_dir in "${CLAUDE_LOG_LOCATIONS[@]}"; do
                if [[ -d "$log_dir" ]]; then
                    log_dirs_found=$((log_dirs_found + 1))
                fi
            done
        else
            log_dirs_found="N/A"
        fi
        status_data+=("log_dirs_found" "$log_dirs_found")
        
        # Capabilities check
        local capabilities_count=13  # Based on known tools: Bash, Read, Edit, etc.
        status_data+=("capabilities_count" "$capabilities_count")
        
        # Usage tracking data (skip if fast mode)
        if [[ "$fast_mode" == "false" ]]; then
            local usage_info
            usage_info=$(claude_code::get_usage 2>/dev/null)
            
            if [[ -n "$usage_info" ]]; then
                local hourly_requests=$(echo "$usage_info" | jq -r '.current_hour_requests // "0"')
                local daily_requests=$(echo "$usage_info" | jq -r '.current_day_requests // "0"')
                local weekly_requests=$(echo "$usage_info" | jq -r '.current_week_requests // "0"')
                local last_5h_requests=$(echo "$usage_info" | jq -r '.last_5_hours // "0"')
                local subscription_tier=$(echo "$usage_info" | jq -r '.subscription_tier // "unknown"')
                
                # Get last rate limit info
                local last_rate_limit=$(echo "$usage_info" | jq -r '.last_rate_limit // null')
                local last_rate_limit_time="N/A"
                local last_rate_limit_type="N/A"
                if [[ "$last_rate_limit" != "null" ]]; then
                    last_rate_limit_time=$(echo "$last_rate_limit" | jq -r '.timestamp // "N/A"')
                    last_rate_limit_type=$(echo "$last_rate_limit" | jq -r '.limit_type // "N/A"')
                fi
                
                # Calculate usage percentages based on tier
                local tier_key="${subscription_tier:-free}"
                [[ "$tier_key" == "unknown" ]] && tier_key="free"
                
                local limit_5h=$(echo "$usage_info" | jq -r ".estimated_limits.${tier_key}.\"5_hour\" // 45")
                local limit_daily=$(echo "$usage_info" | jq -r ".estimated_limits.${tier_key}.daily // 50")
                local limit_weekly=$(echo "$usage_info" | jq -r ".estimated_limits.${tier_key}.weekly // 350")
                
                local pct_5h=0
                local pct_daily=0
                local pct_weekly=0
                
                # Calculate percentages safely
                if [[ $limit_5h -gt 0 ]]; then
                    pct_5h=$((last_5h_requests * 100 / limit_5h))
                fi
                if [[ $limit_daily -gt 0 ]]; then
                    pct_daily=$((daily_requests * 100 / limit_daily))
                fi
                if [[ $limit_weekly -gt 0 ]]; then
                    pct_weekly=$((weekly_requests * 100 / limit_weekly))
                fi
                
                # Add usage data to status
                status_data+=("usage_hourly" "$hourly_requests")
                status_data+=("usage_daily" "$daily_requests")
                status_data+=("usage_weekly" "$weekly_requests")
                status_data+=("usage_5hour" "$last_5h_requests")
                status_data+=("usage_5hour_limit" "$limit_5h")
                status_data+=("usage_5hour_percent" "$pct_5h")
                status_data+=("usage_daily_limit" "$limit_daily")
                status_data+=("usage_daily_percent" "$pct_daily")
                status_data+=("usage_weekly_limit" "$limit_weekly")
                status_data+=("usage_weekly_percent" "$pct_weekly")
                status_data+=("subscription_tier" "$subscription_tier")
                status_data+=("last_rate_limit_time" "$last_rate_limit_time")
                status_data+=("last_rate_limit_type" "$last_rate_limit_type")
                
                # Time until reset for different limit types
                local time_to_5h_reset=$(claude_code::time_until_reset "5_hour" 2>/dev/null || echo "N/A")
                local time_to_daily_reset=$(claude_code::time_until_reset "daily" 2>/dev/null || echo "N/A")
                local time_to_weekly_reset=$(claude_code::time_until_reset "weekly" 2>/dev/null || echo "N/A")
                
                status_data+=("time_to_5h_reset" "$time_to_5h_reset")
                status_data+=("time_to_daily_reset" "$time_to_daily_reset")
                status_data+=("time_to_weekly_reset" "$time_to_weekly_reset")
            else
                # Add N/A values if usage tracking is not available
                status_data+=("usage_hourly" "N/A")
                status_data+=("usage_daily" "N/A")
                status_data+=("usage_weekly" "N/A")
                status_data+=("usage_5hour" "N/A")
                status_data+=("subscription_tier" "N/A")
                status_data+=("last_rate_limit_time" "N/A")
            fi
        else
            # Fast mode - skip usage tracking
            status_data+=("usage_note" "Usage tracking skipped in fast mode")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Claude Code status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
claude_code::status::show() {
    local format="text"
    local verbose="false"
    local fast="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            --fast)
                fast="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data (pass fast flag if set)
    local data_string
    local collect_args=""
    if [[ "$fast" == "true" ]]; then
        collect_args="--fast"
    fi
    data_string=$(claude_code::status::collect_data $collect_args 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect Claude Code status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${data_array[@]}"
    else
        # Text format with standardized structure
        claude_code::status::display_text "${data_array[@]}"
    fi
    
    # Return appropriate exit code
    local healthy="false"
    local running="false"
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        case "${data_array[i]}" in
            "healthy") healthy="${data_array[i+1]}" ;;
            "running") running="${data_array[i+1]}" ;;
        esac
    done
    
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    else
        return 2
    fi
}

#######################################
# Display status in text format
#######################################
claude_code::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 Claude Code Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Claude Code, run: npm install -g @anthropic-ai/claude-code"
        log::info "   Or run: ./manage.sh --action install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Available: Yes"
    else
        log::warn "   ⚠️  Available: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # CLI Information
    log::info "💻 CLI Information:"
    log::info "   📦 Package: ${data[package_name]:-unknown}"
    log::info "   🔖 Version: ${data[version]:-unknown}"
    log::info "   💻 Command: ${data[cli_command]:-claude}"
    echo
    
    # Node.js Requirements
    log::info "📋 Node.js Requirements:"
    log::info "   📊 Minimum Version: ${data[min_node_version]:-18}"
    if [[ "${data[node_available]:-false}" == "true" ]]; then
        log::success "   ✅ Node.js Available: ${data[node_version]:-unknown}"
    else
        log::warn "   ⚠️  Node.js Available: No (required for installation)"
    fi
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📁 Config Directory: ${data[config_dir]:-unknown}"
    log::info "   📂 Sessions Directory: ${data[sessions_dir]:-unknown}"
    if [[ "${data[config_exists]:-false}" == "true" ]]; then
        log::success "   ✅ Global Config: Found"
    else
        log::warn "   ⚠️  Global Config: Not found"
    fi
    if [[ "${data[project_config_exists]:-false}" == "true" ]]; then
        log::success "   ✅ Project Config: Found"
    else
        log::info "   ℹ️  Project Config: Not found (optional)"
    fi
    echo
    
    # Runtime information (only if installed and healthy)
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        log::info "   🔐 Auth Status: ${data[auth_status]:-unknown}"
        if [[ "${data[tty_compatible]:-unknown}" == "true" ]]; then
            log::success "   ✅ TTY Compatible: Yes"
        elif [[ "${data[tty_compatible]:-unknown}" == "false" ]]; then
            log::warn "   ⚠️  TTY Compatible: No (limited functionality)"
        else
            log::info "   ❓ TTY Compatible: Unknown"
        fi
        log::info "   📄 Log Directories: ${data[log_dirs_found]:-0} found"
        log::info "   🛠️  Available Tools: ${data[capabilities_count]:-0}"
        echo
        
        # Quick access info
        if [[ "${data[healthy]:-false}" == "true" ]]; then
            log::info "🎯 Quick Actions:"
            log::info "   🚀 Run Claude: claude"
            log::info "   🩺 Health Check: ./manage.sh --action health-check"
            log::info "   📄 View Logs: ./manage.sh --action logs"
            log::info "   ⚙️  Settings: claude config"
        else
            log::info "🔧 Troubleshooting:"
            log::info "   🩺 Run Health Check: ./manage.sh --action health-check --check-type full"
            log::info "   🔐 Authentication: claude login"
            log::info "   📄 Check Logs: ./manage.sh --action logs"
        fi
    fi
}

#######################################
# Check Claude Code status (Legacy function - now calls standardized version)
#######################################
claude_code::status() {
    # Main status function for CLI registration - use standardized format
    status::run_standard "claude-code" "claude_code::status::collect_data" "claude_code::status::display_text" "$@"
}

# CLI framework expects hyphenated function name
claude-code::status() {
    status::run_standard "claude-code" "claude_code::status::collect_data" "claude_code::status::display_text" "$@"
}

#######################################
# Display detailed information
#######################################
claude_code::info() {
    log::header "ℹ️  Claude Code Information"
    
    # Display general information
    claude_code::display_info
    
    # Show current status
    echo
    claude_code::status
}

#######################################
# View Claude logs
#######################################
claude_code::logs() {
    log::header "📜 Claude Code Logs"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    local found_logs=false
    
    for log_dir in "${CLAUDE_LOG_LOCATIONS[@]}"; do
        if [[ -d "$log_dir" ]]; then
            found_logs=true
            log::info "Found logs in: $log_dir"
            
            # Show recent log files
            local recent_logs
            recent_logs=$(ls -t "$log_dir" 2>/dev/null | head -5)
            
            if [[ -n "$recent_logs" ]]; then
                echo "Recent log files:"
                echo "$recent_logs" | nl
                echo
                
                # Show tail of most recent log
                local latest_log
                latest_log=$(ls -t "$log_dir" | head -1)
                if [[ -n "$latest_log" ]]; then
                    log::info "Last 20 lines of $latest_log:"
                    tail -20 "$log_dir/$latest_log"
                fi
            fi
            echo
        fi
    done
    
    if ! $found_logs; then
        log::info "No log files found"
        log::info "Logs are created when Claude runs into issues"
    fi
    
    # Check for debug mode
    echo
    log::info "To enable debug logging:"
    log::info "  export CLAUDE_DEBUG=1"
    log::info "  export LOG_LEVEL=debug"
}

#######################################
# Start Claude Code (CLI tool - not applicable)
#######################################
claude_code::start() {
    log::info "Claude Code is a CLI tool and doesn't run as a persistent service"
    log::info "Use: $0 --action run --prompt \"your prompt\" to execute Claude"
    return 2  # Success but no action needed
}

#######################################
# Stop Claude Code (CLI tool - not applicable)
#######################################
claude_code::stop() {
    log::info "Claude Code is a CLI tool and doesn't run as a persistent service"
    log::info "Claude sessions terminate automatically when complete"
    return 2  # Success but no action needed
}

#######################################
# Restart Claude Code (CLI tool - not applicable)
#######################################
claude_code::restart() {
    log::info "Claude Code is a CLI tool and doesn't run as a persistent service"
    log::info "Each invocation is independent and doesn't require restart"
    return 2  # Success but no action needed
}

#######################################
# Test Claude Code functionality
#######################################
claude_code::test() {
    log::header "🧪 Testing Claude Code Functionality"
    
    # Use the existing test-safe implementation
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed"
        return 1
    fi
    
    log::info "This mode only verifies installation and configuration"
    log::info "No prompts will be executed to avoid file system changes"
    
    # Check installation
    local version
    version=$(claude_code::get_version 2>/dev/null || echo "unknown")
    log::success "✓ Claude Code is installed (version: $version)"
    
    # Check for config directory
    if [[ -d "$HOME/.claude-code" ]] || [[ -d "$HOME/.claude" ]]; then
        log::success "✓ Configuration directory exists"
    else
        log::warn "⚠️  Configuration directory not found"
    fi
    
    # Check Node.js requirements
    if claude_code::check_node_version; then
        local node_version
        node_version=$(node --version)
        log::success "✓ Node.js requirements met ($node_version)"
    else
        log::warn "⚠️  Node.js version requirement not met"
        return 1
    fi
    
    # Report sandbox availability
    if [[ -f "${CLAUDE_CODE_SCRIPT_DIR}/sandbox/claude-sandbox.sh" ]]; then
        log::success "✓ Sandbox is available for safe testing"
    else
        log::info "ℹ️  Sandbox not available"
    fi
    
    log::success "✅ Claude Code test completed successfully"
    return 0
}