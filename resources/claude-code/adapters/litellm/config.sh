#!/usr/bin/env bash
# LiteLLM Adapter Configuration Manager
# Manages configuration settings for LiteLLM integration

set -euo pipefail

# Get script directory
ADAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source required libraries
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/../../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${ADAPTER_DIR}/state.sh"

#######################################
# Display current configuration
#######################################
litellm::config_show() {
    litellm::init_state
    
    log::header "⚙️  LiteLLM Adapter Configuration"
    echo
    
    local config_json
    config_json=$(cat "${CLAUDE_CONFIG_DIR:-$HOME/.claude}/litellm_config.json" 2>/dev/null)
    
    if [[ -z "$config_json" ]]; then
        log::error "Configuration file not found"
        return 1
    fi
    
    log::info "🔧 Auto-Fallback Settings:"
    log::info "   Enabled: $(echo "$config_json" | jq -r '.auto_fallback_enabled')"
    log::info "   On Rate Limit: $(echo "$config_json" | jq -r '.auto_fallback_on_rate_limit')"
    log::info "   On Error: $(echo "$config_json" | jq -r '.auto_fallback_on_error')"
    log::info "   Auto-disconnect After: $(echo "$config_json" | jq -r '.auto_disconnect_after_hours') hours"
    echo
    
    log::info "🤖 Model Configuration:"
    log::info "   Preferred Model: $(echo "$config_json" | jq -r '.preferred_model')"
    log::info "   Model Mappings:"
    echo "$config_json" | jq -r '.model_mappings | to_entries[] | "     \(.key) → \(.value)"'
    echo
    
    log::info "💰 Cost Tracking:"
    log::info "   Enabled: $(echo "$config_json" | jq -r '.cost_tracking.enabled')"
    log::info "   Native Claude Cost: \$$(echo "$config_json" | jq -r '.cost_tracking.native_claude_cost')"
    log::info "   LiteLLM Cost: \$$(echo "$config_json" | jq -r '.cost_tracking.litellm_cost')"
    log::info "   Last Reset: $(echo "$config_json" | jq -r '.cost_tracking.last_reset')"
    echo
    
    log::info "📊 Performance Metrics:"
    log::info "   Enabled: $(echo "$config_json" | jq -r '.performance_metrics.enabled')"
    log::info "   Track Latency: $(echo "$config_json" | jq -r '.performance_metrics.track_latency')"
    log::info "   Track Quality: $(echo "$config_json" | jq -r '.performance_metrics.track_quality')"
    
    # Check for endpoint configuration
    local endpoint
    endpoint=$(echo "$config_json" | jq -r '.litellm_endpoint // null')
    if [[ "$endpoint" != "null" && -n "$endpoint" ]]; then
        echo
        log::info "🌐 Custom Endpoint: $endpoint"
    fi
}

#######################################
# Set a configuration value
# Arguments:
#   $1 - Setting name
#   $2 - Value
#######################################
litellm::config_set() {
    local setting="$1"
    local value="$2"
    
    case "$setting" in
        auto-fallback)
            if [[ "$value" == "on" || "$value" == "true" ]]; then
                litellm::set_config "auto_fallback_enabled" "true"
                log::success "✅ Auto-fallback enabled"
            elif [[ "$value" == "off" || "$value" == "false" ]]; then
                litellm::set_config "auto_fallback_enabled" "false"
                log::success "✅ Auto-fallback disabled"
            else
                log::error "Invalid value. Use 'on' or 'off'"
                return 1
            fi
            ;;
            
        auto-fallback-on-error)
            if [[ "$value" == "on" || "$value" == "true" ]]; then
                litellm::set_config "auto_fallback_on_error" "true"
                log::success "✅ Auto-fallback on error enabled"
            elif [[ "$value" == "off" || "$value" == "false" ]]; then
                litellm::set_config "auto_fallback_on_error" "false"
                log::success "✅ Auto-fallback on error disabled"
            else
                log::error "Invalid value. Use 'on' or 'off'"
                return 1
            fi
            ;;
            
        auto-fallback-on-rate-limit)
            if [[ "$value" == "on" || "$value" == "true" ]]; then
                litellm::set_config "auto_fallback_on_rate_limit" "true"
                log::success "✅ Auto-fallback on rate limit enabled"
            elif [[ "$value" == "off" || "$value" == "false" ]]; then
                litellm::set_config "auto_fallback_on_rate_limit" "false"
                log::success "✅ Auto-fallback on rate limit disabled"
            else
                log::error "Invalid value. Use 'on' or 'off'"
                return 1
            fi
            ;;
            
        model|set-model)
            litellm::set_config "preferred_model" "$value"
            log::success "✅ Preferred model set to: $value"
            ;;
            
        endpoint|set-endpoint)
            litellm::set_config "litellm_endpoint" "$value"
            log::success "✅ LiteLLM endpoint set to: $value"
            ;;
            
        auto-disconnect-hours)
            if [[ "$value" =~ ^[0-9]+$ ]]; then
                litellm::set_config "auto_disconnect_after_hours" "$value"
                log::success "✅ Auto-disconnect set to $value hours"
            else
                log::error "Invalid value. Must be a number"
                return 1
            fi
            ;;
            
        cost-tracking)
            if [[ "$value" == "on" || "$value" == "true" ]]; then
                litellm::set_config "cost_tracking.enabled" "true"
                log::success "✅ Cost tracking enabled"
            elif [[ "$value" == "off" || "$value" == "false" ]]; then
                litellm::set_config "cost_tracking.enabled" "false"
                log::success "✅ Cost tracking disabled"
            else
                log::error "Invalid value. Use 'on' or 'off'"
                return 1
            fi
            ;;
            
        reset-costs)
            litellm::set_config "cost_tracking.native_claude_cost" "0.0"
            litellm::set_config "cost_tracking.litellm_cost" "0.0"
            litellm::set_config "cost_tracking.last_reset" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
            log::success "✅ Cost tracking reset"
            ;;
            
        add-model-mapping)
            # Format: claude-model:litellm-model
            if [[ "$value" =~ ^([^:]+):(.+)$ ]]; then
                local claude_model="${BASH_REMATCH[1]}"
                local litellm_model="${BASH_REMATCH[2]}"
                
                # Read current config, add mapping, write back
                local config_file="${CLAUDE_CONFIG_DIR:-$HOME/.claude}/litellm_config.json"
                local temp_file
                temp_file=$(mktemp)
                
                jq --arg cm "$claude_model" \
                   --arg lm "$litellm_model" \
                   '.model_mappings[$cm] = $lm' \
                   "$config_file" > "$temp_file" && mv "$temp_file" "$config_file"
                
                log::success "✅ Added model mapping: $claude_model → $litellm_model"
            else
                log::error "Invalid format. Use: claude-model:litellm-model"
                return 1
            fi
            ;;
            
        *)
            log::error "Unknown setting: $setting"
            log::info "Available settings:"
            log::info "  auto-fallback <on|off>"
            log::info "  auto-fallback-on-error <on|off>"
            log::info "  auto-fallback-on-rate-limit <on|off>"
            log::info "  model <model-name>"
            log::info "  endpoint <url>"
            log::info "  auto-disconnect-hours <hours>"
            log::info "  cost-tracking <on|off>"
            log::info "  reset-costs"
            log::info "  add-model-mapping <claude-model:litellm-model>"
            return 1
            ;;
    esac
}

#######################################
# Get a configuration value
# Arguments:
#   $1 - Setting name
#######################################
litellm::config_get() {
    local setting="$1"
    
    local value
    case "$setting" in
        auto-fallback)
            value=$(litellm::get_config "auto_fallback_enabled")
            ;;
        auto-fallback-on-rate-limit)
            value=$(litellm::get_config "auto_fallback_on_rate_limit")
            ;;
        model)
            value=$(litellm::get_config "preferred_model")
            ;;
        endpoint)
            value=$(litellm::get_config "litellm_endpoint")
            ;;
        auto-disconnect-hours)
            value=$(litellm::get_config "auto_disconnect_after_hours")
            ;;
        cost-tracking)
            value=$(litellm::get_config "cost_tracking.enabled")
            ;;
        *)
            log::error "Unknown setting: $setting"
            return 1
            ;;
    esac
    
    # Output the value
    echo "$value"
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    ACTION="${1:-show}"
    shift || true
    
    case "$ACTION" in
        show)
            litellm::config_show
            ;;
        set|set-*)
            if [[ $# -lt 2 ]]; then
                log::error "Usage: config set <setting> <value>"
                exit 1
            fi
            # Remove 'set-' prefix if present
            SETTING="${1#set-}"
            litellm::config_set "$SETTING" "$2"
            ;;
        get)
            if [[ $# -lt 1 ]]; then
                log::error "Usage: config get <setting>"
                exit 1
            fi
            litellm::config_get "$1"
            ;;
        *)
            log::error "Unknown action: $ACTION"
            log::info "Available actions: show, set, get"
            exit 1
            ;;
    esac
fi

# Export functions
export -f litellm::config_show
export -f litellm::config_set
export -f litellm::config_get