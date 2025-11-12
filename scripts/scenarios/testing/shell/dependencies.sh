#!/usr/bin/env bash
# Unified dependency validation for all scenarios
# Leverages `vrooli scenario status --json` for comprehensive dependency checks
set -euo pipefail

# Source existing helpers
SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/core.sh"
source "$SHELL_DIR/phase-helpers.sh"

# === Main Dependency Validation Function ===

testing::dependencies::validate_all() {
    local summary="Dependency validation completed"
    testing::phase::auto_lifecycle_start \
        --phase-name "dependencies" \
        --default-target-time "60s" \
        --summary "$summary" \
        --config-phase-key "dependencies" \
        || true

    local scenario_name=""
    local enforce_resources=false
    local enforce_runtimes=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scenario)
                scenario_name="$2"
                shift 2
                ;;
            --enforce-resources)
                enforce_resources=true
                shift
                ;;
            --enforce-runtimes)
                enforce_runtimes=true
                shift
                ;;
            *)
                echo "Unknown option: $1" >&2
                testing::phase::auto_lifecycle_end "$summary"
                return 1
                ;;
        esac
    done

    # Auto-detect scenario if not provided
    if [ -z "$scenario_name" ]; then
        scenario_name=$(testing::core::detect_scenario)
    fi

    echo "🔍 Validating dependencies for $scenario_name..."
    echo ""

    # Step 1: Get comprehensive status from vrooli CLI
    local status_json
    if ! status_json=$(vrooli scenario status "$scenario_name" --json 2>/dev/null); then
        echo "⚠️  Could not fetch scenario status from API"
        echo "   Falling back to direct file detection..."
        status_json="{}"
    fi

    # Step 2: Validate language runtimes
    _validate_runtimes "$scenario_name" "$status_json" "$enforce_runtimes"

    # Step 3: Validate package managers
    _validate_package_managers "$scenario_name" "$status_json"

    # Step 4: Validate resources
    _validate_resources "$scenario_name" "$status_json" "$enforce_resources"

    # Step 5: Validate runtime connectivity (if scenario is running)
    _validate_runtime_connectivity "$scenario_name" "$status_json"

    echo ""
    local total_tests=$((${TESTING_RUNTIME_CHECKS_TOTAL:-0} + ${TESTING_PACKAGE_CHECKS_TOTAL:-0} + ${TESTING_RESOURCE_CHECKS_TOTAL:-0} + ${TESTING_CONNECTIVITY_CHECKS_TOTAL:-0}))
    local total_passed=$((${TESTING_RUNTIME_CHECKS_PASSED:-0} + ${TESTING_PACKAGE_CHECKS_PASSED:-0} + ${TESTING_RESOURCE_CHECKS_PASSED:-0} + ${TESTING_CONNECTIVITY_CHECKS_PASSED:-0}))
    local total_skipped=$((total_tests - total_passed))

    # Check if there are actual errors (not just warnings/skipped)
    local error_count=${TESTING_PHASE_ERROR_COUNT:-0}

    if [ $error_count -eq 0 ]; then
        if [ $total_skipped -eq 0 ]; then
            echo "✅ All dependency checks passed ($total_tests/$total_tests)"
        else
            echo "✅ Dependency checks completed with $total_skipped warning(s) ($total_passed/$total_tests passed)"
        fi
        testing::phase::auto_lifecycle_end "$summary"
        return 0
    else
        echo "❌ $error_count dependency check(s) failed ($total_passed/$total_tests passed, $total_skipped skipped)"
        testing::phase::auto_lifecycle_end "$summary"
        return 1
    fi
}

# === Private Helper Functions ===

_validate_runtimes() {
    local scenario_name="$1"
    local status_json="$2"
    local enforce="$3"

    echo "🔧 Checking language runtimes..."

    TESTING_RUNTIME_CHECKS_TOTAL=0
    TESTING_RUNTIME_CHECKS_PASSED=0

    # Get tech stack from insights
    local has_go_api=$(echo "$status_json" | jq -r '.insights.stack.api.language == "Go"' 2>/dev/null || echo "false")
    local has_node_ui=$(echo "$status_json" | jq -r '.insights.stack.ui != null and .insights.stack.ui != {}' 2>/dev/null || echo "false")
    local has_python=$(echo "$status_json" | jq -r '.insights.stack | has("python")' 2>/dev/null || echo "false")

    # Fallback to file-based detection if status doesn't have insights
    if [ "$has_go_api" = "false" ] && [ "$has_node_ui" = "false" ]; then
        echo "  ℹ️  Using file-based detection (API not available)"
        # Use existing core.sh function
        local languages
        languages=$(testing::core::detect_languages)

        for lang in $languages; do
            TESTING_RUNTIME_CHECKS_TOTAL=$((TESTING_RUNTIME_CHECKS_TOTAL + 1))
            _check_runtime "$lang" "$enforce"
        done
    else
        # Use insights data
        if [ "$has_go_api" = "true" ]; then
            TESTING_RUNTIME_CHECKS_TOTAL=$((TESTING_RUNTIME_CHECKS_TOTAL + 1))
            _check_runtime "go" "$enforce"
        fi

        if [ "$has_node_ui" = "true" ]; then
            TESTING_RUNTIME_CHECKS_TOTAL=$((TESTING_RUNTIME_CHECKS_TOTAL + 1))
            _check_runtime "node" "$enforce"
        fi

        if [ "$has_python" = "true" ]; then
            TESTING_RUNTIME_CHECKS_TOTAL=$((TESTING_RUNTIME_CHECKS_TOTAL + 1))
            _check_runtime "python" "$enforce"
        fi
    fi

    if [ $TESTING_RUNTIME_CHECKS_TOTAL -eq 0 ]; then
        echo "  ℹ️  No language runtimes detected"
    fi

    echo ""
}

_check_runtime() {
    local lang="$1"
    local enforce="$2"

    case "$lang" in
        go)
            if command -v go >/dev/null 2>&1; then
                local go_version=$(go version | awk '{print $3}')
                echo "  ✅ Go runtime available ($go_version)"
                TESTING_RUNTIME_CHECKS_PASSED=$((TESTING_RUNTIME_CHECKS_PASSED + 1))
                testing::phase::add_test passed
            else
                echo "  ❌ Go runtime not available"
                if [ "$enforce" = true ]; then
                    testing::phase::add_error "Go runtime required but not available"
                    testing::phase::add_test failed
                else
                    testing::phase::add_warning "Go runtime not available"
                    testing::phase::add_test skipped
                fi
            fi
            ;;

        node)
            if command -v node >/dev/null 2>&1; then
                local node_version=$(node --version)
                echo "  ✅ Node.js runtime available ($node_version)"
                TESTING_RUNTIME_CHECKS_PASSED=$((TESTING_RUNTIME_CHECKS_PASSED + 1))
                testing::phase::add_test passed
            else
                echo "  ❌ Node.js runtime not available"
                if [ "$enforce" = true ]; then
                    testing::phase::add_error "Node.js runtime required but not available"
                    testing::phase::add_test failed
                else
                    testing::phase::add_warning "Node.js runtime not available"
                    testing::phase::add_test skipped
                fi
            fi
            ;;

        python)
            if command -v python3 >/dev/null 2>&1; then
                local python_version=$(python3 --version)
                echo "  ✅ Python runtime available ($python_version)"
                TESTING_RUNTIME_CHECKS_PASSED=$((TESTING_RUNTIME_CHECKS_PASSED + 1))
                testing::phase::add_test passed
            else
                echo "  ❌ Python runtime not available"
                if [ "$enforce" = true ]; then
                    testing::phase::add_error "Python runtime required but not available"
                    testing::phase::add_test failed
                else
                    testing::phase::add_warning "Python runtime not available"
                    testing::phase::add_test skipped
                fi
            fi
            ;;
    esac
}

_validate_package_managers() {
    local scenario_name="$1"
    local status_json="$2"

    echo "📦 Checking package managers..."

    TESTING_PACKAGE_CHECKS_TOTAL=0
    TESTING_PACKAGE_CHECKS_PASSED=0

    # Check Go modules
    if [ -f "api/go.mod" ]; then
        TESTING_PACKAGE_CHECKS_TOTAL=$((TESTING_PACKAGE_CHECKS_TOTAL + 1))
        echo "  🐹 Validating Go module dependencies..."

        if command -v go >/dev/null 2>&1; then
            if cd api && go list ./... >/dev/null 2>&1; then
                echo "    ✅ Go modules resolve successfully"
                TESTING_PACKAGE_CHECKS_PASSED=$((TESTING_PACKAGE_CHECKS_PASSED + 1))
                testing::phase::add_test passed
                cd ..
            else
                echo "    ❌ Go module validation failed"
                testing::phase::add_error "Go modules do not resolve"
                testing::phase::add_test failed
                cd ..
            fi
        else
            echo "    ⚠️  Go runtime not available"
            testing::phase::add_test skipped
        fi
    fi

    # Check Node packages
    local ui_path=""
    for candidate in ui frontend web app; do
        if [ -f "$candidate/package.json" ]; then
            ui_path="$candidate"
            break
        fi
    done

    if [ -n "$ui_path" ]; then
        TESTING_PACKAGE_CHECKS_TOTAL=$((TESTING_PACKAGE_CHECKS_TOTAL + 1))
        echo "  📦 Validating Node.js dependencies..."

        # Detect package manager
        local package_manager="npm"
        if [ -f "$ui_path/pnpm-lock.yaml" ]; then
            package_manager="pnpm"
        elif [ -f "$ui_path/yarn.lock" ]; then
            package_manager="yarn"
        fi

        echo "    Detected package manager: $package_manager"

        # Check if node_modules exists and contains packages
        if [ -d "$ui_path/node_modules" ] && [ -n "$(ls -A "$ui_path/node_modules" 2>/dev/null)" ]; then
            echo "    ✅ Node.js dependencies installed"
            TESTING_PACKAGE_CHECKS_PASSED=$((TESTING_PACKAGE_CHECKS_PASSED + 1))
            testing::phase::add_test passed
        else
            echo "    ⚠️  node_modules not found or empty (run '$package_manager install')"
            testing::phase::add_warning "Node.js dependencies not installed"
            testing::phase::add_test skipped
        fi
    fi

    # Check Python packages
    if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ]; then
        TESTING_PACKAGE_CHECKS_TOTAL=$((TESTING_PACKAGE_CHECKS_TOTAL + 1))
        echo "  🐍 Python dependencies detected"
        # Could add pip freeze validation here
        echo "    ℹ️  Python dependency validation not implemented"
        testing::phase::add_test skipped
    fi

    if [ $TESTING_PACKAGE_CHECKS_TOTAL -eq 0 ]; then
        echo "  ℹ️  No package managers detected"
    fi

    echo ""
}

_validate_resources() {
    local scenario_name="$1"
    local status_json="$2"
    local enforce="$3"

    echo "🔗 Checking resource integrations..."

    TESTING_RESOURCE_CHECKS_TOTAL=0
    TESTING_RESOURCE_CHECKS_PASSED=0

    # Get resources from service.json
    local service_json=".vrooli/service.json"
    if [ ! -f "$service_json" ]; then
        echo "  ℹ️  No service.json found, skipping resource checks"
        echo ""
        return 0
    fi

    # Get enabled/required resources
    local resources
    local jq_resources="$var_JQ_RESOURCES_EXPR"
    [[ -z "$jq_resources" ]] && jq_resources='(.dependencies.resources // {})'
    resources=$(jq -r "$jq_resources | to_entries[] | select(.value.enabled == true or .value.required == true) | .key" "$service_json" 2>/dev/null || echo "")

    if [ -z "$resources" ]; then
        echo "  ℹ️  No resources configured"
        echo ""
        return 0
    fi

    # Try to get resource health from insights
    local has_health_data=false
    if echo "$status_json" | jq -e '.insights.resources.items' >/dev/null 2>&1; then
        has_health_data=true
    fi

    for resource in $resources; do
        TESTING_RESOURCE_CHECKS_TOTAL=$((TESTING_RESOURCE_CHECKS_TOTAL + 1))

        if [ "$has_health_data" = true ]; then
            # Use health data from insights
            local resource_status
            resource_status=$(echo "$status_json" | jq -r ".insights.resources.items[] | select(.name == \"$resource\") | .status" 2>/dev/null || echo "unknown")

            if [ "$resource_status" = "healthy" ] || [ "$resource_status" = "running" ]; then
                echo "  ✅ $resource: $resource_status"
                TESTING_RESOURCE_CHECKS_PASSED=$((TESTING_RESOURCE_CHECKS_PASSED + 1))
                testing::phase::add_test passed
            elif [ "$resource_status" = "unknown" ] || [ -z "$resource_status" ]; then
                echo "  ⚠️  $resource: health unknown"
                testing::phase::add_warning "$resource health cannot be verified"
                testing::phase::add_test skipped
            else
                echo "  ❌ $resource: $resource_status"
                if [ "$enforce" = true ]; then
                    testing::phase::add_error "$resource is not healthy"
                    testing::phase::add_test failed
                else
                    testing::phase::add_warning "$resource is not healthy"
                    testing::phase::add_test skipped
                fi
            fi
        else
            # No health data available from API
            echo "  ⚠️  $resource: health data not available from API"
            testing::phase::add_warning "$resource health cannot be verified (API not available)"
            testing::phase::add_test skipped
        fi
    done

    echo ""
}

_validate_runtime_connectivity() {
    local scenario_name="$1"
    local status_json="$2"

    echo "🔌 Checking runtime connectivity..."

    TESTING_CONNECTIVITY_CHECKS_TOTAL=0
    TESTING_CONNECTIVITY_CHECKS_PASSED=0

    # Check if scenario is running
    local is_running
    is_running=$(echo "$status_json" | jq -r '.scenario_data.status == "running"' 2>/dev/null || echo "false")

    if [ "$is_running" != "true" ]; then
        echo "  ℹ️  Scenario not running, skipping connectivity checks"
        echo ""
        return 0
    fi

    # Check API health
    TESTING_CONNECTIVITY_CHECKS_TOTAL=$((TESTING_CONNECTIVITY_CHECKS_TOTAL + 1))
    local api_health
    api_health=$(echo "$status_json" | jq -r '.diagnostics.health_checks.api.status // "unknown"' 2>/dev/null)

    if [ "$api_health" = "healthy" ]; then
        echo "  ✅ API endpoint: healthy"
        TESTING_CONNECTIVITY_CHECKS_PASSED=$((TESTING_CONNECTIVITY_CHECKS_PASSED + 1))
        testing::phase::add_test passed
    elif [ "$api_health" = "unknown" ]; then
        echo "  ℹ️  API endpoint: health status not available"
        testing::phase::add_test skipped
    else
        echo "  ⚠️  API endpoint: $api_health"
        testing::phase::add_warning "API health check did not return healthy status"
        testing::phase::add_test skipped
    fi

    # Check UI health
    TESTING_CONNECTIVITY_CHECKS_TOTAL=$((TESTING_CONNECTIVITY_CHECKS_TOTAL + 1))
    local ui_health
    ui_health=$(echo "$status_json" | jq -r '.diagnostics.health_checks.ui.status // "unknown"' 2>/dev/null)

    if [ "$ui_health" = "healthy" ]; then
        echo "  ✅ UI endpoint: healthy"
        TESTING_CONNECTIVITY_CHECKS_PASSED=$((TESTING_CONNECTIVITY_CHECKS_PASSED + 1))
        testing::phase::add_test passed
    elif [ "$ui_health" = "unknown" ]; then
        echo "  ℹ️  UI endpoint: health status not available"
        testing::phase::add_test skipped
    else
        echo "  ⚠️  UI endpoint: $ui_health"
        testing::phase::add_warning "UI health check did not return healthy status"
        testing::phase::add_test skipped
    fi

    echo ""
}

# Export functions
export -f testing::dependencies::validate_all
