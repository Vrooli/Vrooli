#!/bin/bash
# KiCad Status Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_STATUS_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source dependencies
source "${KICAD_STATUS_LIB_DIR}/common.sh"
source "${APP_ROOT}/scripts/lib/status-args.sh"

# Collect KiCad status data
kicad::status::collect_data() {
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
    
    # Initialize directories
    kicad::init_dirs
    
    # Check installation
    local installed="false"
    local healthy="false"
    local health_message="KiCad not installed"
    local version="not_installed"
    local python_api="unavailable"
    
    if kicad::is_installed; then
        installed="true"
        version=$(kicad::get_version)
        healthy="true"
        health_message="KiCad is installed and ready"
        
        # Check Python API
        if kicad::check_python_api; then
            python_api="available"
        fi
    fi
    
    # Count projects (skip in fast mode)
    local project_count=0
    if [[ "$fast_mode" == "false" && -d "$KICAD_PROJECTS_DIR" ]]; then
        project_count=$(find "$KICAD_PROJECTS_DIR" -name "*.kicad_pro" 2>/dev/null | wc -l)
    fi
    
    # Count libraries
    local library_count=0
    if [[ -d "$KICAD_LIBRARIES_DIR" ]]; then
        library_count=$(find "$KICAD_LIBRARIES_DIR" -name "*.kicad_sym" -o -name "*.kicad_mod" 2>/dev/null | wc -l)
    fi
    
    # Check for test results
    local test_status="not_run"
    local test_timestamp=""
    local test_results_file="${KICAD_DATA_DIR}/test_results.json"
    
    if [[ "$fast_mode" == "false" && -f "$test_results_file" ]]; then
        if [[ -r "$test_results_file" ]]; then
            test_timestamp=$(jq -r '.timestamp // ""' "$test_results_file" 2>/dev/null)
            local test_passed=$(jq -r '.passed // "unknown"' "$test_results_file" 2>/dev/null)
            local test_total=$(jq -r '.total // "unknown"' "$test_results_file" 2>/dev/null)
            
            if [[ -n "$test_timestamp" && "$test_passed" != "unknown" ]]; then
                if [[ "$test_passed" == "$test_total" ]]; then
                    test_status="passed"
                else
                    test_status="failed"
                fi
            fi
        fi
    fi
    
    # Build status data
    status_data+=("name" "kicad")
    status_data+=("category" "execution")
    status_data+=("description" "Electronic design automation for PCB design")
    status_data+=("installed" "$installed")
    status_data+=("running" "$installed")  # KiCad is a desktop app, "running" when installed
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("version" "$version")
    status_data+=("python_api" "$python_api")
    status_data+=("data_dir" "$KICAD_DATA_DIR")
    status_data+=("projects_dir" "$KICAD_PROJECTS_DIR")
    status_data+=("libraries_dir" "$KICAD_LIBRARIES_DIR")
    status_data+=("templates_dir" "$KICAD_TEMPLATES_DIR")
    status_data+=("outputs_dir" "$KICAD_OUTPUTS_DIR")
    status_data+=("project_count" "$project_count")
    status_data+=("library_count" "$library_count")
    status_data+=("export_formats" "$KICAD_EXPORT_FORMATS")
    status_data+=("test_status" "$test_status")
    status_data+=("test_timestamp" "$test_timestamp")
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

# Display KiCad status in text format
kicad::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        if [[ $value_idx -le $# ]]; then
            local value="${!value_idx}"
            data["$key"]="$value"
        fi
    done
    
    # Header
    echo "KiCad Status Report"
    echo "==================="
    echo
    echo "Description: ${data[description]:-unknown}"
    echo "Category: ${data[category]:-unknown}"
    echo
    
    # Basic status
    echo "Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        echo "  ✅ Installed: Yes"
    else
        echo "  ❌ Installed: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "  ✅ Health: Healthy"
    else
        echo "  ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Configuration
    echo "Configuration:"
    echo "  📊 Version: ${data[version]:-unknown}"
    echo "  🐍 Python API: ${data[python_api]:-unknown}"
    echo "  📁 Data Directory: ${data[data_dir]:-unknown}"
    echo "  📂 Projects: ${data[project_count]:-0} projects"
    echo "  📚 Libraries: ${data[library_count]:-0} components"
    echo "  📤 Export Formats: ${data[export_formats]:-unknown}"
    echo
    
    # Test results
    if [[ -n "${data[test_timestamp]:-}" ]]; then
        echo "Test Results:"
        if [[ "${data[test_status]:-}" == "passed" ]]; then
            echo "  ✅ Status: All tests passed"
        elif [[ "${data[test_status]:-}" == "failed" ]]; then
            echo "  ❌ Status: Some tests failed"
        else
            echo "  ⚠️  Status: ${data[test_status]:-unknown}"
        fi
        echo "  🕐 Last Run: ${data[test_timestamp]:-}"
        echo
    fi
    
    # Status message
    echo "Status Message:"
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "  ✅ ${data[health_message]:-unknown}"
    else
        echo "  ⚠️  ${data[health_message]:-unknown}"
    fi
}

# Get KiCad status
kicad_status() {
    status::run_standard "kicad" "kicad::status::collect_data" "kicad::status::display_text" "$@"
}

# Export function
export -f kicad_status