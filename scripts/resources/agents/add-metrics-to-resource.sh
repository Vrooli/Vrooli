#!/usr/bin/env bash
################################################################################
# Add Metrics Integration to Resource
# 
# Helper script to add proper metrics tracking to resource operation functions
# Usage: ./add-metrics-to-resource.sh <resource-name> <operation-function-name>
################################################################################

set -euo pipefail

if [[ $# -lt 2 ]]; then
    echo "Usage: $0 <resource-name> <operation-function-name>"
    echo ""
    echo "Example: $0 gemini gemini_generate"
    echo "         $0 openrouter openrouter_chat"
    exit 1
fi

RESOURCE_NAME="$1"
OPERATION_FUNCTION="$2"

APP_ROOT="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../" && pwd)}"
RESOURCE_CLI="${APP_ROOT}/resources/${RESOURCE_NAME}/cli.sh"

if [[ ! -f "$RESOURCE_CLI" ]]; then
    echo "Error: Resource CLI not found: $RESOURCE_CLI"
    exit 1
fi

echo "🔧 Adding metrics integration to $RESOURCE_NAME::$OPERATION_FUNCTION"

# Check if the function exists
if ! grep -q "${OPERATION_FUNCTION}()" "$RESOURCE_CLI"; then
    echo "Error: Function $OPERATION_FUNCTION not found in $RESOURCE_CLI"
    exit 1
fi

# Create backup
cp "$RESOURCE_CLI" "${RESOURCE_CLI}.backup.$(date +%s)"
echo "📁 Created backup: ${RESOURCE_CLI}.backup.$(date +%s)"

# Generate metrics integration code
METRICS_CODE="    # Execute operation with metrics tracking
    local result=0
    local start_time=\$(date +%s%3N)  # Milliseconds
    
    # Track operation start metrics
    if [[ -n \"\$agent_id\" ]] && type -t agents::metrics::increment &>/dev/null; then
        agents::metrics::increment \"\${REGISTRY_FILE:-\${APP_ROOT}/.vrooli/${RESOURCE_NAME}-agents.json}\" \"\$agent_id\" \"requests\" 1
    fi
    
    # REPLACE THIS LINE WITH ACTUAL OPERATION EXECUTION
    echo \"TODO: Replace this with actual operation execution\"
    result=\$?
    
    # Track operation completion metrics
    if [[ -n \"\$agent_id\" ]] && type -t agents::metrics::histogram &>/dev/null; then
        local end_time=\$(date +%s%3N)
        local duration=\$((end_time - start_time))
        agents::metrics::histogram \"\${REGISTRY_FILE:-\${APP_ROOT}/.vrooli/${RESOURCE_NAME}-agents.json}\" \"\$agent_id\" \"request_duration_ms\" \"\$duration\"
        
        # Track success/error
        if [[ \$result -eq 0 ]]; then
            log::debug \"${RESOURCE_NAME} operation completed successfully\"
        else
            type -t agents::metrics::increment &>/dev/null && \\
                agents::metrics::increment \"\${REGISTRY_FILE:-\${APP_ROOT}/.vrooli/${RESOURCE_NAME}-agents.json}\" \"\$agent_id\" \"errors\" 1
        fi
        
        # Update process metrics
        if type -t agents::metrics::gauge &>/dev/null; then
            # Get current process memory usage (in MB)
            local memory_kb=\$(ps -o rss= -p \$\$ 2>/dev/null | awk '{print \$1}' || echo \"0\")
            local memory_mb=\$((memory_kb / 1024))
            agents::metrics::gauge \"\${REGISTRY_FILE:-\${APP_ROOT}/.vrooli/${RESOURCE_NAME}-agents.json}\" \"\$agent_id\" \"memory_mb\" \"\$memory_mb\"
        fi
    fi"

echo "📝 Generated metrics integration code"
echo ""
echo "⚠️  Manual steps required:"
echo "1. Edit $RESOURCE_CLI"
echo "2. Find the $OPERATION_FUNCTION function"
echo "3. Add agent registration code at the beginning (see INTEGRATION-TEMPLATE.md)"
echo "4. Replace the actual operation execution with metrics tracking"
echo "5. Add agent cleanup at the end"
echo ""
echo "📋 Use the integration template: scripts/resources/agents/INTEGRATION-TEMPLATE.md"
echo "🔍 Reference implementations: ollama/cli.sh, claude-code/cli.sh"
echo ""
echo "Generated metrics code snippet:"
echo "────────────────────────────────────────────────────────────────"
echo "$METRICS_CODE"
echo "────────────────────────────────────────────────────────────────"