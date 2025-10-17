#!/bin/bash

# OpenRouter Custom Routing Rules
# Provides dynamic model selection based on user-defined rules

set -euo pipefail

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source core dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "${RESOURCE_DIR}/config/defaults.sh" 2>/dev/null || true
source "${SCRIPT_DIR}/core.sh" 2>/dev/null || true

# Global variables - use defaults if not set
OPENROUTER_DATA_DIR="${OPENROUTER_DATA_DIR:-${var_ROOT_DIR:-/tmp}/data/openrouter}"
ROUTING_RULES_FILE="${OPENROUTER_DATA_DIR}/routing-rules.json"
ROUTING_HISTORY_FILE="${OPENROUTER_DATA_DIR}/routing-history.json"

# Initialize routing system
openrouter::routing::init() {
    mkdir -p "${OPENROUTER_DATA_DIR}"
    
    # Create default rules file if not exists
    if [[ ! -f "${ROUTING_RULES_FILE}" ]]; then
        cat > "${ROUTING_RULES_FILE}" << 'EOF'
{
  "rules": [
    {
      "name": "cost-optimizer",
      "description": "Select cheapest model that meets requirements",
      "priority": 100,
      "conditions": {
        "max_cost_per_million": 5.0
      },
      "action": {
        "type": "select_cheapest",
        "fallback": "openai/gpt-3.5-turbo"
      },
      "enabled": true
    },
    {
      "name": "code-specialist",
      "description": "Use specialized models for code tasks",
      "priority": 90,
      "conditions": {
        "prompt_contains": ["code", "function", "programming", "debug", "implement"]
      },
      "action": {
        "type": "select_model",
        "model": "anthropic/claude-3-opus",
        "fallback": "openai/gpt-4-turbo"
      },
      "enabled": false
    },
    {
      "name": "fast-response",
      "description": "Use fast models for simple queries",
      "priority": 80,
      "conditions": {
        "prompt_length_less_than": 100,
        "response_time_target": 1000
      },
      "action": {
        "type": "select_fastest",
        "candidates": ["mistralai/mistral-7b", "openai/gpt-3.5-turbo"],
        "fallback": "openai/gpt-3.5-turbo"
      },
      "enabled": false
    }
  ],
  "default_model": "openai/gpt-3.5-turbo",
  "routing_enabled": true
}
EOF
    fi
    
    # Initialize history file
    if [[ ! -f "${ROUTING_HISTORY_FILE}" ]]; then
        echo '{"history": []}' > "${ROUTING_HISTORY_FILE}"
    fi
}

# Add or update a routing rule
openrouter::routing::add_rule() {
    local name="${1:-}"
    local rule_json="${2:-}"
    
    if [[ -z "${name}" ]] || [[ -z "${rule_json}" ]]; then
        echo "[ERROR]   Rule name and JSON definition required"
        return 1
    fi
    
    # Validate JSON
    if ! echo "${rule_json}" | jq . >/dev/null 2>&1; then
        echo "[ERROR]   Invalid JSON format"
        return 1
    fi
    
    # Add or update rule
    local temp_file=$(mktemp)
    jq --arg name "${name}" --argjson rule "${rule_json}" '
        .rules = (.rules | map(select(.name != $name))) + 
        [$rule + {name: $name}] | 
        .rules |= sort_by(.priority | tonumber * -1)
    ' "${ROUTING_RULES_FILE}" > "${temp_file}"
    
    mv "${temp_file}" "${ROUTING_RULES_FILE}"
    echo "[SUCCESS] Routing rule '${name}' added/updated"
}

# List all routing rules
openrouter::routing::list_rules() {
    local format="${1:-table}"
    
    if [[ ! -f "${ROUTING_RULES_FILE}" ]]; then
        echo "[WARNING] No routing rules configured"
        return 0
    fi
    
    if [[ "${format}" == "json" ]]; then
        cat "${ROUTING_RULES_FILE}"
    else
        echo "Custom Routing Rules:"
        echo "===================="
        jq -r '.rules[] | 
            "Name: \(.name)\n" +
            "Priority: \(.priority)\n" +
            "Enabled: \(.enabled)\n" +
            "Description: \(.description)\n" +
            "Conditions: \(.conditions | tostring)\n" +
            "Action: \(.action.type) -> \(.action.model // .action.fallback)\n"
        ' "${ROUTING_RULES_FILE}"
    fi
}

# Remove a routing rule
openrouter::routing::remove_rule() {
    local name="${1:-}"
    
    if [[ -z "${name}" ]]; then
        echo "[ERROR]   Rule name required"
        return 1
    fi
    
    local temp_file=$(mktemp)
    jq --arg name "${name}" '.rules |= map(select(.name != $name))' \
        "${ROUTING_RULES_FILE}" > "${temp_file}"
    
    mv "${temp_file}" "${ROUTING_RULES_FILE}"
    echo "[SUCCESS] Routing rule '${name}' removed"
}

# Enable/disable a routing rule
openrouter::routing::toggle_rule() {
    local name="${1:-}"
    local enabled="${2:-}"
    
    if [[ -z "${name}" ]]; then
        echo "[ERROR]   Rule name required"
        return 1
    fi
    
    if [[ "${enabled}" != "true" ]] && [[ "${enabled}" != "false" ]]; then
        echo "[ERROR]   Enabled must be 'true' or 'false'"
        return 1
    fi
    
    local temp_file=$(mktemp)
    jq --arg name "${name}" --argjson enabled "${enabled}" '
        .rules |= map(if .name == $name then .enabled = $enabled else . end)
    ' "${ROUTING_RULES_FILE}" > "${temp_file}"
    
    mv "${temp_file}" "${ROUTING_RULES_FILE}"
    echo "[SUCCESS] Routing rule '${name}' ${enabled}"
}

# Evaluate routing rules for a prompt
openrouter::routing::evaluate() {
    local prompt="${1:-}"
    local context="${2:-{}}"
    
    if [[ -z "${prompt}" ]]; then
        echo "[ERROR]   Prompt required for routing evaluation"
        return 1
    fi
    
    # Check if routing is enabled
    local routing_enabled=$(jq -r '.routing_enabled // true' "${ROUTING_RULES_FILE}")
    if [[ "${routing_enabled}" != "true" ]]; then
        # Return default model
        jq -r '.default_model // "openai/gpt-3.5-turbo"' "${ROUTING_RULES_FILE}"
        return 0
    fi
    
    # Evaluate each enabled rule in priority order
    local selected_model=""
    local matched_rule=""
    
    while IFS= read -r rule; do
        local name=$(echo "${rule}" | jq -r '.name')
        local enabled=$(echo "${rule}" | jq -r '.enabled')
        
        if [[ "${enabled}" != "true" ]]; then
            continue
        fi
        
        # Check conditions
        if openrouter::routing::check_conditions "${prompt}" "${rule}" "${context}"; then
            # Apply action
            selected_model=$(openrouter::routing::apply_action "${rule}" "${context}")
            matched_rule="${name}"
            break
        fi
    done < <(jq -c '.rules[]' "${ROUTING_RULES_FILE}" | sort -t: -k2 -rn)
    
    # Record routing decision
    openrouter::routing::record_decision "${prompt}" "${matched_rule}" "${selected_model}"
    
    # Return selected model or default
    if [[ -n "${selected_model}" ]]; then
        echo "${selected_model}"
    else
        jq -r '.default_model // "openai/gpt-3.5-turbo"' "${ROUTING_RULES_FILE}"
    fi
}

# Check if conditions are met
openrouter::routing::check_conditions() {
    local prompt="${1}"
    local rule="${2}"
    local context="${3}"
    
    local conditions=$(echo "${rule}" | jq -c '.conditions')
    
    # Check prompt contains
    local prompt_contains=$(echo "${conditions}" | jq -r '.prompt_contains[]?' 2>/dev/null)
    if [[ -n "${prompt_contains}" ]]; then
        local matched=false
        while IFS= read -r keyword; do
            if [[ "${prompt,,}" == *"${keyword,,}"* ]]; then
                matched=true
                break
            fi
        done <<< "${prompt_contains}"
        
        if [[ "${matched}" != "true" ]]; then
            return 1
        fi
    fi
    
    # Check prompt length
    local max_length=$(echo "${conditions}" | jq -r '.prompt_length_less_than // 0')
    if [[ "${max_length}" -gt 0 ]] && [[ ${#prompt} -ge ${max_length} ]]; then
        return 1
    fi
    
    # Check max cost
    local max_cost=$(echo "${conditions}" | jq -r '.max_cost_per_million // 0')
    if [[ "${max_cost}" != "0" ]]; then
        # This would check against model pricing
        # For now, we'll pass this check
        true
    fi
    
    # Check response time target
    local response_time=$(echo "${conditions}" | jq -r '.response_time_target // 0')
    if [[ "${response_time}" != "0" ]]; then
        # This would check against model benchmarks
        # For now, we'll pass this check
        true
    fi
    
    return 0
}

# Apply routing action
openrouter::routing::apply_action() {
    local rule="${1}"
    local context="${2}"
    
    local action=$(echo "${rule}" | jq -c '.action')
    local action_type=$(echo "${action}" | jq -r '.type')
    
    case "${action_type}" in
        "select_model")
            # Return specified model
            echo "${action}" | jq -r '.model // .fallback'
            ;;
            
        "select_cheapest")
            # Select cheapest model meeting requirements
            # For demo, return fallback
            echo "${action}" | jq -r '.fallback // "openai/gpt-3.5-turbo"'
            ;;
            
        "select_fastest")
            # Select fastest model from candidates
            local candidates=$(echo "${action}" | jq -r '.candidates[]?' 2>/dev/null)
            if [[ -n "${candidates}" ]]; then
                # For demo, return first candidate
                echo "${candidates}" | head -n1
            else
                echo "${action}" | jq -r '.fallback // "openai/gpt-3.5-turbo"'
            fi
            ;;
            
        *)
            # Unknown action type, return fallback
            echo "${action}" | jq -r '.fallback // "openai/gpt-3.5-turbo"'
            ;;
    esac
}

# Record routing decision for analytics
openrouter::routing::record_decision() {
    local prompt="${1}"
    local rule="${2}"
    local model="${3}"
    
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local prompt_preview="${prompt:0:50}..."
    
    local temp_file=$(mktemp)
    jq --arg ts "${timestamp}" \
       --arg rule "${rule:-default}" \
       --arg model "${model}" \
       --arg prompt "${prompt_preview}" \
       '.history += [{
           timestamp: $ts,
           rule: $rule,
           model: $model,
           prompt_preview: $prompt
       }] | .history = (.history | .[-100:])' \
       "${ROUTING_HISTORY_FILE}" > "${temp_file}"
    
    mv "${temp_file}" "${ROUTING_HISTORY_FILE}"
}

# Show routing history
openrouter::routing::history() {
    local limit="${1:-10}"
    
    if [[ ! -f "${ROUTING_HISTORY_FILE}" ]]; then
        echo "[WARNING] No routing history available"
        return 0
    fi
    
    echo "Recent Routing Decisions:"
    echo "========================"
    jq -r --argjson limit "${limit}" '
        .history[-$limit:] | reverse | .[] |
        "\(.timestamp) | Rule: \(.rule) | Model: \(.model) | Prompt: \(.prompt_preview)"
    ' "${ROUTING_HISTORY_FILE}"
}

# Test routing rules with a sample prompt
openrouter::routing::test() {
    local prompt="${1:-Test prompt for routing}"
    
    echo "[INFO]    Testing routing rules with prompt: \"${prompt}\""
    echo ""
    
    # Show which rules would match
    echo "Rule Evaluation:"
    echo "==============="
    
    local matched=false
    while IFS= read -r rule; do
        local name=$(echo "${rule}" | jq -r '.name')
        local enabled=$(echo "${rule}" | jq -r '.enabled')
        local priority=$(echo "${rule}" | jq -r '.priority')
        
        if [[ "${enabled}" != "true" ]]; then
            echo "  [SKIP] ${name} (disabled)"
            continue
        fi
        
        if openrouter::routing::check_conditions "${prompt}" "${rule}" "{}"; then
            local model=$(openrouter::routing::apply_action "${rule}" "{}")
            echo "  [MATCH] ${name} (priority: ${priority}) -> ${model}"
            matched=true
            break
        else
            echo "  [MISS] ${name} (conditions not met)"
        fi
    done < <(jq -c '.rules[]' "${ROUTING_RULES_FILE}" | sort -t: -k2 -rn)
    
    if [[ "${matched}" != "true" ]]; then
        local default_model=$(jq -r '.default_model // "openai/gpt-3.5-turbo"' "${ROUTING_RULES_FILE}")
        echo "  [DEFAULT] No rules matched -> ${default_model}"
    fi
    
    echo ""
    echo "Selected Model: $(openrouter::routing::evaluate "${prompt}")"
}

# Export routing rule template
openrouter::routing::template() {
    cat << 'EOF'
{
  "description": "Example routing rule template",
  "priority": 50,
  "conditions": {
    "prompt_contains": ["keyword1", "keyword2"],
    "prompt_length_less_than": 500,
    "max_cost_per_million": 10.0,
    "response_time_target": 2000
  },
  "action": {
    "type": "select_model",
    "model": "openai/gpt-4-turbo",
    "fallback": "openai/gpt-3.5-turbo"
  },
  "enabled": true
}

Available action types:
- select_model: Use a specific model
- select_cheapest: Choose cheapest model meeting requirements
- select_fastest: Choose fastest model from candidates list

Available conditions:
- prompt_contains: Array of keywords to match (case-insensitive)
- prompt_length_less_than: Maximum prompt length in characters
- max_cost_per_million: Maximum cost per million tokens
- response_time_target: Target response time in milliseconds
EOF
}

# CLI interface
openrouter::routing::cli() {
    local action="${1:-list}"
    shift || true
    
    case "${action}" in
        "add")
            local name="${1:-}"
            local rule_file="${2:-}"
            if [[ -z "${name}" ]] || [[ -z "${rule_file}" ]]; then
                echo "Usage: routing add <name> <rule-file.json>"
                return 1
            fi
            if [[ ! -f "${rule_file}" ]]; then
                echo "[ERROR]   Rule file not found: ${rule_file}"
                return 1
            fi
            local rule_json=$(cat "${rule_file}")
            openrouter::routing::add_rule "${name}" "${rule_json}"
            ;;
            
        "list")
            openrouter::routing::list_rules "${1:-table}"
            ;;
            
        "remove")
            openrouter::routing::remove_rule "${1}"
            ;;
            
        "enable")
            openrouter::routing::toggle_rule "${1}" "true"
            ;;
            
        "disable")
            openrouter::routing::toggle_rule "${1}" "false"
            ;;
            
        "test")
            openrouter::routing::test "${1:-Test prompt}"
            ;;
            
        "history")
            openrouter::routing::history "${1:-10}"
            ;;
            
        "template")
            openrouter::routing::template
            ;;
            
        "evaluate")
            local prompt="${1:-}"
            if [[ -z "${prompt}" ]]; then
                echo "Usage: routing evaluate <prompt>"
                return 1
            fi
            openrouter::routing::evaluate "${prompt}"
            ;;
            
        *)
            echo "Usage: routing [add|list|remove|enable|disable|test|history|template|evaluate]"
            echo ""
            echo "Commands:"
            echo "  add <name> <file>    Add/update routing rule from JSON file"
            echo "  list [json]          List all routing rules"
            echo "  remove <name>        Remove a routing rule"
            echo "  enable <name>        Enable a routing rule"
            echo "  disable <name>       Disable a routing rule"
            echo "  test [prompt]        Test routing with sample prompt"
            echo "  history [limit]      Show routing history"
            echo "  template             Show rule template"
            echo "  evaluate <prompt>    Evaluate prompt and return selected model"
            ;;
    esac
}

# Initialize on source
openrouter::routing::init