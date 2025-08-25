#\!/bin/bash

# Status functions for AutoGPT

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"

autogpt_status() {
    # Source format utility for consistent JSON output
    source "${APP_ROOT}/scripts/lib/utils/format.sh"
    
    # Handle --format json flag
    local format="text"
    local verbose=""
    
    for arg in "$@"; do
        case "$arg" in
            --format)
                shift
                format="$1"
                ;;
            json|--json)
                format="json"
                ;;
            --verbose)
                verbose="true"
                ;;
        esac
    done
    
    # Gather status information
    local installed="false"
    local running="false"
    local health="not_installed"
    local version="unknown"
    local api_url=""
    local ui_url=""
    local agents_count=0
    local tools_count=0
    local llm_provider="none"
    
    # Check installation
    if autogpt_container_exists; then
        installed="true"
        
        # Check if running
        if autogpt_container_running; then
            running="true"
            
            # Get version
            version=$(docker exec "$AUTOGPT_CONTAINER_NAME" python -c "import autogpt; print(autogpt.__version__)" 2>/dev/null || echo "unknown")
            
            # Check health
            if curl -s "http://localhost:${AUTOGPT_PORT_API}/health" >/dev/null 2>&1; then
                health="healthy"
                api_url="http://localhost:${AUTOGPT_PORT_API}"
                ui_url="http://localhost:${AUTOGPT_PORT_UI}"
            else
                health="degraded"
            fi
            
            # Get LLM provider
            llm_provider=$(autogpt_get_llm_config)
        else
            health="stopped"
        fi
    fi
    
    # Count agents and tools
    [ -d "$AUTOGPT_AGENTS_DIR" ] && agents_count=$(find "$AUTOGPT_AGENTS_DIR" -name "*.yaml" -o -name "*.yml" 2>/dev/null | wc -l)
    [ -d "$AUTOGPT_TOOLS_DIR" ] && tools_count=$(find "$AUTOGPT_TOOLS_DIR" -name "*.py" 2>/dev/null | wc -l)
    
    # Test results
    local test_status="not_run"
    local test_timestamp=""
    if [ -f "$AUTOGPT_LOGS_DIR/.last_test" ]; then
        test_timestamp=$(stat -c %Y "$AUTOGPT_LOGS_DIR/.last_test" 2>/dev/null || echo "")
        test_status="passed"
    fi
    
    # Determine healthy boolean
    local healthy="false"
    [[ "$health" == "healthy" ]] && healthy="true"
    
    # Output in the requested format
    if [[ "$format" == "json" ]]; then
        format::key_value json \
            name "$AUTOGPT_RESOURCE_NAME" \
            description "$AUTOGPT_RESOURCE_DESCRIPTION" \
            category "$AUTOGPT_RESOURCE_CATEGORY" \
            status "$health" \
            installed "$installed" \
            running "$running" \
            health "$health" \
            healthy "$healthy" \
            version "$version" \
            container_name "$AUTOGPT_CONTAINER_NAME" \
            container_image "$AUTOGPT_IMAGE" \
            api_port "$AUTOGPT_PORT_API" \
            ui_port "$AUTOGPT_PORT_UI" \
            api_url "$api_url" \
            ui_url "$ui_url" \
            llm_provider "$llm_provider" \
            agents_count "$agents_count" \
            tools_count "$tools_count" \
            agents_dir "$AUTOGPT_AGENTS_DIR" \
            tools_dir "$AUTOGPT_TOOLS_DIR" \
            workspace_dir "$AUTOGPT_WORKSPACE_DIR" \
            test_status "$test_status" \
            test_timestamp "$test_timestamp"
    else
        # Text format
        echo "[HEADER]  AutoGPT Status Report"
        echo ""
        echo "Description: $AUTOGPT_RESOURCE_DESCRIPTION"
        echo "Category: $AUTOGPT_RESOURCE_CATEGORY"
        echo ""
        echo "Basic Status:"
        echo "  $([ "$installed" == "true" ] && echo "✅" || echo "❌") Installed: $([ "$installed" == "true" ] && echo "Yes" || echo "No")"
        echo "  $([ "$running" == "true" ] && echo "✅" || echo "❌") Running: $([ "$running" == "true" ] && echo "Yes" || echo "No")"
        echo "  $([ "$health" == "healthy" ] && echo "✅" || echo "⚠️") Health: $health"
        echo ""
        echo "Container Info:"
        echo "  📦 Name: $AUTOGPT_CONTAINER_NAME"
        echo "  🖼️  Image: $AUTOGPT_IMAGE"
        echo "  📊 Version: $version"
        echo ""
        if [ -n "$api_url" ]; then
            echo "Service URLs:"
            echo "  🌐 API: $api_url"
            echo "  🖥️  UI: $ui_url"
            echo ""
        fi
        echo "Configuration:"
        echo "  🤖 LLM Provider: $llm_provider"
        echo ""
        echo "Metrics:"
        echo "  👤 Agents: $agents_count"
        echo "  🔧 Tools: $tools_count"
        echo ""
        echo "Directories:"
        echo "  📁 Agents: $AUTOGPT_AGENTS_DIR"
        echo "  📁 Tools: $AUTOGPT_TOOLS_DIR"
        echo "  📁 Workspace: $AUTOGPT_WORKSPACE_DIR"
        echo ""
        echo "Test Status:"
        echo "  🧪 Status: $test_status"
        [ -n "$test_timestamp" ] && echo "  ⏰ Last run: $(date -d @$test_timestamp 2>/dev/null || echo 'N/A')"
        echo ""
        echo "Status Message:"
        echo "  $([ "$health" == "healthy" ] && echo "✅ AutoGPT is healthy and ready" || echo "⚠️  AutoGPT needs attention")"
    fi
}
