#!/usr/bin/env bash
# Claude Code Settings Management Functions
# Handles viewing and updating Claude settings

# Source var.sh for directory variables if not already sourced
# shellcheck disable=SC1091
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

#######################################
# View or update Claude settings
#######################################
claude_code::settings() {
    log::header "⚙️  Claude Code Settings"
    
    if ! claude_code::is_installed; then
        log::error "Claude Code is not installed. Run: $0 --action install"
        return 1
    fi
    
    # Check for settings files
    if [[ -f "$CLAUDE_PROJECT_SETTINGS" ]]; then
        log::info "Project settings found: $CLAUDE_PROJECT_SETTINGS"
        echo
        log::info "Current project settings:"
        if system::is_command jq; then
            cat "$CLAUDE_PROJECT_SETTINGS" | jq . 2>/dev/null || cat "$CLAUDE_PROJECT_SETTINGS"
        else
            cat "$CLAUDE_PROJECT_SETTINGS"
        fi
        echo
    fi
    
    if [[ -f "$CLAUDE_SETTINGS_FILE" ]]; then
        log::info "Global settings found: $CLAUDE_SETTINGS_FILE"
        echo
        log::info "Current global settings:"
        if system::is_command jq; then
            cat "$CLAUDE_SETTINGS_FILE" | jq . 2>/dev/null || cat "$CLAUDE_SETTINGS_FILE"
        else
            cat "$CLAUDE_SETTINGS_FILE"
        fi
    else
        log::warn "No global settings file found"
        log::info "Settings will be created when you first run Claude"
    fi
    
    claude_code::settings_tips
}

#######################################
# Display settings tips and recommendations
#######################################
claude_code::settings_tips() {
    echo
    log::info "Tips:"
    log::info "  • Project settings override global settings"
    log::info "  • Use .claude/settings.json for project-specific configuration"
    log::info "  • Set environment variables in settings for tool timeouts"
    log::info "  • Configure allowed tools for security"
}

#######################################
# Get a specific setting value
# Arguments:
#   $1 - setting key (dot notation supported)
#   $2 - scope (project|global|auto)
# Outputs: setting value or empty string
#######################################
claude_code::settings_get() {
    local key="$1"
    local scope="${2:-auto}"
    local settings_file=""
    
    # Determine which settings file to use
    if [[ "$scope" == "project" ]]; then
        settings_file="$CLAUDE_PROJECT_SETTINGS"
    elif [[ "$scope" == "global" ]]; then
        settings_file="$CLAUDE_SETTINGS_FILE"
    else
        # Auto: check project first, then global
        if [[ -f "$CLAUDE_PROJECT_SETTINGS" ]]; then
            settings_file="$CLAUDE_PROJECT_SETTINGS"
        elif [[ -f "$CLAUDE_SETTINGS_FILE" ]]; then
            settings_file="$CLAUDE_SETTINGS_FILE"
        fi
    fi
    
    if [[ -z "$settings_file" || ! -f "$settings_file" ]]; then
        echo ""
        return 1
    fi
    
    # Extract value using jq if available
    if system::is_command jq; then
        jq -r ".$key // empty" "$settings_file" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

#######################################
# Set a specific setting value
# Arguments:
#   $1 - setting key (dot notation supported)
#   $2 - setting value
#   $3 - scope (project|global)
# Returns: 0 on success, 1 on failure
#######################################
claude_code::settings_set() {
    local key="$1"
    local value="$2"
    local scope="${3:-project}"
    local settings_file=""
    
    if [[ -z "$key" || -z "$value" ]]; then
        log::error "Both key and value are required"
        return 1
    fi
    
    # Determine which settings file to use
    if [[ "$scope" == "project" ]]; then
        settings_file="$CLAUDE_PROJECT_SETTINGS"
        mkdir -p "${settings_file%/*}"
    elif [[ "$scope" == "global" ]]; then
        settings_file="$CLAUDE_SETTINGS_FILE"
        mkdir -p "${settings_file%/*}"
    else
        log::error "Invalid scope: $scope (use 'project' or 'global')"
        return 1
    fi
    
    # Require jq for setting values
    if ! system::is_command jq; then
        log::error "jq is required for setting values"
        log::info "Install jq: https://stedolan.github.io/jq/download/"
        return 1
    fi
    
    # Create or update settings file
    if [[ -f "$settings_file" ]]; then
        # Update existing file
        local temp_file="/tmp/claude-settings-$$.json"
        jq ".$key = $value" "$settings_file" > "$temp_file" 2>/dev/null
        if [[ $? -eq 0 ]]; then
            mv "$temp_file" "$settings_file"
            log::success "✓ Setting updated: $key = $value"
        else
            trash::safe_remove "$temp_file" --temp
            log::error "Failed to update setting (invalid JSON value?)"
            return 1
        fi
    else
        # Create new file
        echo "{\"$key\": $value}" | jq . > "$settings_file" 2>/dev/null
        if [[ $? -eq 0 ]]; then
            log::success "✓ Settings file created with: $key = $value"
        else
            log::error "Failed to create settings file (invalid JSON value?)"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Reset Claude settings to defaults
# Arguments:
#   $1 - scope (project|global|all)
# Returns: 0 on success, 1 on failure
#######################################
claude_code::settings_reset() {
    local scope="${1:-project}"
    
    case "$scope" in
        "project")
            if [[ -f "$CLAUDE_PROJECT_SETTINGS" ]]; then
                if confirm "Reset project settings to defaults?"; then
                    trash::safe_remove "$CLAUDE_PROJECT_SETTINGS" --temp
                    log::success "✓ Project settings reset"
                else
                    log::info "Reset cancelled"
                fi
            else
                log::info "No project settings to reset"
            fi
            ;;
        "global")
            if [[ -f "$CLAUDE_SETTINGS_FILE" ]]; then
                if confirm "Reset global settings to defaults?"; then
                    trash::safe_remove "$CLAUDE_SETTINGS_FILE" --temp
                    log::success "✓ Global settings reset"
                else
                    log::info "Reset cancelled"
                fi
            else
                log::info "No global settings to reset"
            fi
            ;;
        "all")
            claude_code::settings_reset "project"
            claude_code::settings_reset "global"
            ;;
        *)
            log::error "Invalid scope: $scope (use 'project', 'global', or 'all')"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Set subscription tier and update rate limits
# Arguments:
#   $1 - tier (free|pro|max|teams|enterprise)
# Returns: 0 on success, 1 on failure  
#######################################
claude_code::set_subscription_tier() {
    local tier="$1"
    
    if [[ -z "$tier" ]]; then
        log::error "Subscription tier is required"
        return 1
    fi
    
    # Validate tier
    case "$tier" in
        free|pro|max|teams|enterprise)
            # Valid tier
            ;;
        *)
            log::error "Invalid subscription tier: $tier"
            log::info "Valid tiers: free, pro, max, teams, enterprise"
            return 1
            ;;
    esac
    
    # Source common functions for usage file management
    # shellcheck disable=SC1091
    source "${APP_ROOT}/resources/claude-code/lib/common.sh"
    
    # Initialize usage file if it doesn't exist
    claude_code::init_usage_tracking
    
    # Update subscription tier in usage tracking file
    local temp_file
    temp_file=$(mktemp)
    
    jq --arg tier "$tier" '.subscription_tier = $tier' "$CLAUDE_USAGE_FILE" > "$temp_file"
    if [[ $? -eq 0 ]]; then
        mv "$temp_file" "$CLAUDE_USAGE_FILE"
        log::success "✓ Subscription tier updated to: $tier"
        
        # Show updated limits
        log::info "Updated rate limits based on $tier tier:"
        local limits_key
        case "$tier" in
            max) limits_key="max_100" ;;  # Default to max_100, user can specify max_200 if needed
            *) limits_key="$tier" ;;
        esac
        
        local limits
        limits=$(jq -r ".estimated_limits.${limits_key}" "$CLAUDE_USAGE_FILE" 2>/dev/null)
        if [[ "$limits" != "null" && -n "$limits" ]]; then
            echo "$limits" | jq -r '"  5-hour: " + (.["5_hour"] | tostring) + " requests"'
            echo "$limits" | jq -r '"  Daily: " + (.daily | tostring) + " requests"' 
            echo "$limits" | jq -r '"  Weekly: " + (.weekly | tostring) + " requests"'
        fi
        
        return 0
    else
        rm -f "$temp_file"
        log::error "Failed to update subscription tier"
        return 1
    fi
}