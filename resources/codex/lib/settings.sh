#!/usr/bin/env bash
################################################################################
# Codex Settings Management
# 
# Handles viewing and updating Codex resource settings
# Modeled after claude-code's settings management system
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Configuration paths
CODEX_CONFIG_DIR="${CODEX_CONFIG_DIR:-${CODEX_HOME:-${HOME}/.codex}}"
CODEX_SETTINGS_FILE="${CODEX_CONFIG_DIR}/settings.json"
CODEX_PROJECT_SETTINGS="$(pwd)/.codex/settings.json"
CODEX_AUDIT_FILE="${CODEX_CONFIG_DIR}/audit.log"

################################################################################
# Settings Management Interface
################################################################################

#######################################
# View or update Codex settings
# Returns:
#   0 on success, 1 on failure
#######################################
codex_settings::show() {
    log::header "⚙️  Codex Settings"
    
    # Check for settings files
    if [[ -f "$CODEX_PROJECT_SETTINGS" ]]; then
        log::info "Project settings found: $CODEX_PROJECT_SETTINGS"
        echo
        log::info "Current project settings:"
        if type -t jq &>/dev/null; then
            cat "$CODEX_PROJECT_SETTINGS" | jq . 2>/dev/null || cat "$CODEX_PROJECT_SETTINGS"
        else
            cat "$CODEX_PROJECT_SETTINGS"
        fi
        echo
    fi
    
    if [[ -f "$CODEX_SETTINGS_FILE" ]]; then
        log::info "Global settings found: $CODEX_SETTINGS_FILE"
        echo
        log::info "Current global settings:"
        if type -t jq &>/dev/null; then
            cat "$CODEX_SETTINGS_FILE" | jq . 2>/dev/null || cat "$CODEX_SETTINGS_FILE"
        else
            cat "$CODEX_SETTINGS_FILE"
        fi
    else
        log::warn "No global settings file found"
        log::info "Settings will be created when you first run Codex with custom options"
    fi
    
    codex_settings::show_tips
}

#######################################
# Display settings tips and recommendations
#######################################
codex_settings::show_tips() {
    echo
    log::info "Tips:"
    log::info "  • Project settings (.codex/settings.json) override global settings"
    log::info "  • Use permission profiles: safe, development, admin"
    log::info "  • Configure allowed tools for security: --allowed-tools \"read_file,write_file\""
    log::info "  • Set conversation limits: --max-turns 10"
    log::info "  • Configure timeouts: --timeout 1800 (seconds)"
    log::info "  • Enable audit logging for compliance"
}

#######################################
# Get a specific setting value
# Arguments:
#   $1 - setting key (dot notation supported)
#   $2 - scope (project|global|auto)
# Returns:
#   Setting value or empty string
#######################################
codex_settings::get() {
    local key="$1"
    local scope="${2:-auto}"
    local settings_file=""
    
    # Determine which settings file to use
    if [[ "$scope" == "project" ]]; then
        settings_file="$CODEX_PROJECT_SETTINGS"
    elif [[ "$scope" == "global" ]]; then
        settings_file="$CODEX_SETTINGS_FILE"
    else
        # Auto: check project first, then global
        if [[ -f "$CODEX_PROJECT_SETTINGS" ]]; then
            settings_file="$CODEX_PROJECT_SETTINGS"
        elif [[ -f "$CODEX_SETTINGS_FILE" ]]; then
            settings_file="$CODEX_SETTINGS_FILE"
        fi
    fi
    
    if [[ -z "$settings_file" || ! -f "$settings_file" ]]; then
        echo ""
        return 1
    fi
    
    # Extract value using jq if available
    if type -t jq &>/dev/null; then
        jq -r ".$key // empty" "$settings_file" 2>/dev/null || echo ""
    else
        log::warn "jq not available - cannot parse settings"
        echo ""
    fi
}

#######################################
# Set a specific setting value
# Arguments:
#   $1 - setting key (dot notation supported)
#   $2 - setting value (JSON formatted)
#   $3 - scope (project|global)
# Returns:
#   0 on success, 1 on failure
#######################################
codex_settings::set() {
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
        settings_file="$CODEX_PROJECT_SETTINGS"
        mkdir -p "${settings_file%/*}"
    elif [[ "$scope" == "global" ]]; then
        settings_file="$CODEX_SETTINGS_FILE"
        mkdir -p "${settings_file%/*}"
    else
        log::error "Invalid scope: $scope (use 'project' or 'global')"
        return 1
    fi
    
    # Require jq for setting values
    if ! type -t jq &>/dev/null; then
        log::error "jq is required for setting values"
        log::info "Install jq: https://stedolan.github.io/jq/download/"
        return 1
    fi
    
    # Create or update settings file
    if [[ -f "$settings_file" ]]; then
        # Update existing file
        local temp_file="/tmp/codex-settings-$$.json"
        jq ".$key = $value" "$settings_file" > "$temp_file" 2>/dev/null
        if [[ $? -eq 0 ]]; then
            mv "$temp_file" "$settings_file"
            log::success "✓ Setting updated: $key = $value"
        else
            rm -f "$temp_file"
            log::error "Failed to update setting (invalid JSON value?)"
            return 1
        fi
    else
        # Create new file with defaults
        codex_settings::create_default_config "$settings_file"
        # Then update with the new value
        local temp_file="/tmp/codex-settings-$$.json"
        jq ".$key = $value" "$settings_file" > "$temp_file" 2>/dev/null
        if [[ $? -eq 0 ]]; then
            mv "$temp_file" "$settings_file"
            log::success "✓ Settings file created with: $key = $value"
        else
            rm -f "$temp_file"
            log::error "Failed to create settings file"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Create default configuration file
# Arguments:
#   $1 - Config file path
# Returns:
#   0 on success, 1 on failure
#######################################
codex_settings::create_default_config() {
    local config_file="$1"
    
    cat > "$config_file" << 'EOF'
{
  "execution": {
    "max_turns": 10,
    "timeout": 1800,
    "default_context": "sandbox",
    "safety_checks": true,
    "require_confirmation": true
  },
  "tools": {
    "allowed": ["read_file", "write_file", "list_files"],
    "default_profile": "safe"
  },
  "security": {
    "audit_logging": true,
    "workspace_isolation": true,
    "permission_violations_log": true
  },
  "profiles": {
    "safe": {
      "allowed_tools": ["read_file", "list_files", "analyze_code"],
      "max_turns": 5,
      "timeout": 300,
      "require_confirmation": false
    },
    "development": {
      "allowed_tools": ["read_file", "write_file", "execute_command(git *)", "execute_command(npm *)"],
      "max_turns": 20,
      "timeout": 1800,
      "require_confirmation": ["execute_command"]
    },
    "admin": {
      "allowed_tools": ["*"],
      "max_turns": 50,
      "timeout": 3600,
      "require_confirmation": ["execute_command(rm *)", "execute_command(sudo *)"]
    }
  }
}
EOF
    
    if [[ $? -eq 0 ]]; then
        log::success "✓ Default configuration created: $config_file"
        return 0
    else
        log::error "Failed to create default configuration"
        return 1
    fi
}

#######################################
# Initialize Codex settings if they don't exist
# Returns:
#   0 on success, 1 on failure
#######################################
codex_settings::init() {
    if type -t codex::ensure_home &>/dev/null; then
        local codex_home_path
        codex_home_path=$(codex::ensure_home | tail -n1)
        CODEX_CONFIG_DIR="${codex_home_path}"
        CODEX_SETTINGS_FILE="${CODEX_CONFIG_DIR}/settings.json"
        CODEX_AUDIT_FILE="${CODEX_CONFIG_DIR}/audit.log"
    fi

    # Create config directory
    mkdir -p "$CODEX_CONFIG_DIR"
    
    # Create global settings if they don't exist
    if [[ ! -f "$CODEX_SETTINGS_FILE" ]]; then
        codex_settings::create_default_config "$CODEX_SETTINGS_FILE"
    fi
    
    # Create audit log file
    touch "$CODEX_AUDIT_FILE"
    
    return 0
}

#######################################
# Reset settings to defaults
# Arguments:
#   $1 - scope (project|global|all)
# Returns:
#   0 on success, 1 on failure
#######################################
codex_settings::reset() {
    local scope="${1:-project}"
    
    case "$scope" in
        "project")
            if [[ -f "$CODEX_PROJECT_SETTINGS" ]]; then
                if [[ "${CODEX_SKIP_CONFIRMATIONS:-}" == "true" ]] || read -p "Reset project settings to defaults? (y/N): " -n 1 -r && [[ $REPLY =~ ^[Yy]$ ]]; then
                    echo
                    rm -f "$CODEX_PROJECT_SETTINGS"
                    log::success "✓ Project settings reset"
                else
                    echo
                    log::info "Reset cancelled"
                fi
            else
                log::info "No project settings to reset"
            fi
            ;;
        "global")
            if [[ -f "$CODEX_SETTINGS_FILE" ]]; then
                if [[ "${CODEX_SKIP_CONFIRMATIONS:-}" == "true" ]] || read -p "Reset global settings to defaults? (y/N): " -n 1 -r && [[ $REPLY =~ ^[Yy]$ ]]; then
                    echo
                    codex_settings::create_default_config "$CODEX_SETTINGS_FILE"
                    log::success "✓ Global settings reset"
                else
                    echo
                    log::info "Reset cancelled"
                fi
            else
                log::info "No global settings to reset"
            fi
            ;;
        "all")
            codex_settings::reset "project"
            codex_settings::reset "global"
            ;;
        *)
            log::error "Invalid scope: $scope (use 'project', 'global', or 'all')"
            return 1
            ;;
    esac
    
    return 0
}

################################################################################
# Profile Management
################################################################################

#######################################
# Get available permission profiles
# Returns:
#   JSON array of profile names
#######################################
codex_settings::get_profiles() {
    local settings=$(codex_settings::get "profiles")
    if [[ -n "$settings" && "$settings" != "null" ]]; then
        echo "$settings" | jq -r 'keys[]' 2>/dev/null || echo ""
    else
        echo "safe development admin"
    fi
}

#######################################
# Get profile configuration
# Arguments:
#   $1 - Profile name
# Returns:
#   Profile configuration JSON
#######################################
codex_settings::get_profile() {
    local profile_name="$1"
    
    if [[ -z "$profile_name" ]]; then
        log::error "Profile name is required"
        return 1
    fi
    
    local profile_config
    profile_config=$(codex_settings::get "profiles.$profile_name")
    
    if [[ -n "$profile_config" && "$profile_config" != "null" ]]; then
        echo "$profile_config"
    else
        log::error "Profile not found: $profile_name"
        return 1
    fi
}

#######################################
# Apply profile settings to current execution
# Arguments:
#   $1 - Profile name
# Returns:
#   0 on success, 1 on failure
#######################################
codex_settings::apply_profile() {
    local profile_name="$1"
    
    local profile_config
    profile_config=$(codex_settings::get_profile "$profile_name")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Export profile settings as environment variables
    export CODEX_PROFILE="$profile_name"
    export CODEX_MAX_TURNS=$(echo "$profile_config" | jq -r '.max_turns // 10')
    export CODEX_TIMEOUT=$(echo "$profile_config" | jq -r '.timeout // 1800')
    export CODEX_ALLOWED_TOOLS=$(echo "$profile_config" | jq -r '.allowed_tools // [] | join(",")')
    export CODEX_REQUIRE_CONFIRMATION=$(echo "$profile_config" | jq -r '.require_confirmation // [] | join(",")')
    
    log::info "✓ Applied profile: $profile_name"
    return 0
}

# Export functions
export -f codex_settings::show
export -f codex_settings::get
export -f codex_settings::set
export -f codex_settings::init
export -f codex_settings::reset
export -f codex_settings::get_profiles
export -f codex_settings::get_profile
export -f codex_settings::apply_profile
