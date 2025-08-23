#!/bin/bash

# Status functions for SageMath

sagemath_status() {
    # Source format utility for consistent JSON output
    # SCRIPT_DIR is the lib directory, so we need to go up 3 levels to get to scripts
    source "$(cd "$SCRIPT_DIR" && cd ../../../lib/utils && pwd)/format.sh"
    
    # Handle --json flag
    local format="text"
    local verbose=""
    
    for arg in "$@"; do
        case "$arg" in
            --json|json)
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
    local jupyter_url=""
    local scripts_count=0
    local notebooks_count=0
    
    # Check installation
    if sagemath_container_exists; then
        installed="true"
        
        # Check if running
        if sagemath_container_running; then
            running="true"
            
            # Get version
            version=$(docker exec "$SAGEMATH_CONTAINER_NAME" sage --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+' | head -1 || echo "unknown")
            
            # Check health
            if curl -s "http://localhost:${SAGEMATH_PORT_JUPYTER}" >/dev/null 2>&1; then
                health="healthy"
                jupyter_url="http://localhost:${SAGEMATH_PORT_JUPYTER}"
            else
                health="degraded"
            fi
        else
            health="stopped"
        fi
    fi
    
    # Count files
    scripts_count=$(find "$SAGEMATH_SCRIPTS_DIR" -name "*.sage" 2>/dev/null | wc -l)
    notebooks_count=$(find "$SAGEMATH_NOTEBOOKS_DIR" -name "*.ipynb" 2>/dev/null | wc -l)
    
    # Test results
    local test_status="not_run"
    local test_timestamp=""
    if [ -f "$SAGEMATH_OUTPUTS_DIR/.last_test" ]; then
        test_timestamp=$(stat -c %Y "$SAGEMATH_OUTPUTS_DIR/.last_test" 2>/dev/null || echo "")
        test_status="passed"
    fi
    
    # Determine healthy boolean
    local healthy="false"
    [[ "$health" == "healthy" ]] && healthy="true"
    
    # Output in the requested format
    if [[ "$format" == "json" ]]; then
        format::key_value json \
            name "$SAGEMATH_RESOURCE_NAME" \
            description "$SAGEMATH_RESOURCE_DESCRIPTION" \
            category "$SAGEMATH_RESOURCE_CATEGORY" \
            status "$health" \
            installed "$installed" \
            running "$running" \
            health "$health" \
            healthy "$healthy" \
            version "$version" \
            container_name "$SAGEMATH_CONTAINER_NAME" \
            container_image "$SAGEMATH_IMAGE" \
            jupyter_port "$SAGEMATH_PORT_JUPYTER" \
            api_port "$SAGEMATH_PORT_API" \
            jupyter_url "$jupyter_url" \
            scripts_count "$scripts_count" \
            notebooks_count "$notebooks_count" \
            scripts_dir "$SAGEMATH_SCRIPTS_DIR" \
            notebooks_dir "$SAGEMATH_NOTEBOOKS_DIR" \
            outputs_dir "$SAGEMATH_OUTPUTS_DIR" \
            test_status "$test_status" \
            test_timestamp "$test_timestamp"
    else
        # Text format
        echo "[HEADER]  SageMath Status Report"
        echo ""
        echo "Description: $SAGEMATH_RESOURCE_DESCRIPTION"
        echo "Category: $SAGEMATH_RESOURCE_CATEGORY"
        echo ""
        echo "Basic Status:"
        echo "  $([ "$installed" == "true" ] && echo "✅" || echo "❌") Installed: $([ "$installed" == "true" ] && echo "Yes" || echo "No")"
        echo "  $([ "$running" == "true" ] && echo "✅" || echo "❌") Running: $([ "$running" == "true" ] && echo "Yes" || echo "No")"
        echo "  $([ "$health" == "healthy" ] && echo "✅" || echo "⚠️") Health: $health"
        echo ""
        echo "Container Info:"
        echo "  📦 Name: $SAGEMATH_CONTAINER_NAME"
        echo "  🖼️  Image: $SAGEMATH_IMAGE"
        echo "  📊 Version: $version"
        echo ""
        if [ -n "$jupyter_url" ]; then
            echo "Jupyter Interface:"
            echo "  🌐 URL: $jupyter_url"
        fi
        echo ""
        echo "Metrics:"
        echo "  📝 Scripts: $scripts_count"
        echo "  📓 Notebooks: $notebooks_count"
        echo ""
        echo "Directories:"
        echo "  📁 Scripts: $SAGEMATH_SCRIPTS_DIR"
        echo "  📁 Notebooks: $SAGEMATH_NOTEBOOKS_DIR"
        echo "  📁 Outputs: $SAGEMATH_OUTPUTS_DIR"
        echo ""
        echo "Test Status:"
        echo "  🧪 Status: $test_status"
        [ -n "$test_timestamp" ] && echo "  ⏰ Last run: $(date -d @$test_timestamp 2>/dev/null || echo 'N/A')"
        echo ""
        echo "Status Message:"
        echo "  $([ "$health" == "healthy" ] && echo "✅ SageMath is healthy and ready" || echo "⚠️  SageMath needs attention")"
    fi
}