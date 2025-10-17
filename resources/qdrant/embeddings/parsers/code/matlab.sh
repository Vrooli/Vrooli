#!/usr/bin/env bash
# MATLAB Language Parser for Qdrant Embeddings
# Extracts functions, classes, scripts, and documentation from MATLAB/Octave files
#
# Handles:
# - Function definitions and signatures
# - Class definitions with properties and methods
# - Scripts and live scripts (.mlx)
# - MATLAB documentation comments
# - Simulink model references
# - MEX file interfaces

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from MATLAB file
# 
# Finds function definitions, nested functions, and anonymous functions
#
# Arguments:
#   $1 - Path to MATLAB file
# Returns: JSON with function information
#######################################
extractor::lib::matlab::extract_functions() {
    local file="$1"
    local functions=()
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find function definitions (main and nested)
    while IFS= read -r line; do
        # Match function definitions with or without output arguments
        if [[ "$line" =~ ^[[:space:]]*function[[:space:]]+(\[.*\]|[a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*=[[:space:]]*([a-zA-Z_][a-zA-Z0-9_]*) ]]; then
            local func_name="${BASH_REMATCH[2]}"
            functions+=("$func_name")
        elif [[ "$line" =~ ^[[:space:]]*function[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\( ]]; then
            local func_name="${BASH_REMATCH[1]}"
            functions+=("$func_name")
        elif [[ "$line" =~ ^[[:space:]]*function[[:space:]]+([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*$ ]]; then
            local func_name="${BASH_REMATCH[1]}"
            functions+=("$func_name")
        fi
    done < "$file"
    
    if [[ ${#functions[@]} -gt 0 ]]; then
        printf '%s\n' "${functions[@]}" | jq -R . | jq -s '{functions: .}'
    else
        echo '{"functions": []}'
    fi
}

#######################################
# Extract classes from MATLAB file
# 
# Finds classdef blocks with properties and methods
#
# Arguments:
#   $1 - Path to MATLAB file
# Returns: JSON with class information
#######################################
extractor::lib::matlab::extract_classes() {
    local file="$1"
    local classes=()
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find classdef definitions
    while IFS= read -r class_def; do
        # Extract class name
        local class_name=$(echo "$class_def" | sed -E 's/^[[:space:]]*classdef[[:space:]]+(\([^)]+\)[[:space:]]+)?([a-zA-Z_][a-zA-Z0-9_]*).*/\2/')
        
        if [[ -n "$class_name" ]]; then
            # Count properties and methods
            local property_count=$(grep -c "^[[:space:]]*properties" "$file" 2>/dev/null || echo "0")
            local method_count=$(grep -c "^[[:space:]]*methods" "$file" 2>/dev/null || echo "0")
            
            # Check for inheritance
            local inherits=""
            if [[ "$class_def" =~ classdef.*\<[[:space:]]*([a-zA-Z_][a-zA-Z0-9_.]*) ]]; then
                inherits="${BASH_REMATCH[1]}"
            fi
            
            classes+=("$(jq -n \
                --arg name "$class_name" \
                --arg props "$property_count" \
                --arg methods "$method_count" \
                --arg inherits "$inherits" \
                '{
                    name: $name,
                    property_blocks: ($props | tonumber),
                    method_blocks: ($methods | tonumber),
                    inherits: $inherits
                }')")
        fi
    done < <(grep -E "^[[:space:]]*classdef" "$file" 2>/dev/null)
    
    if [[ ${#classes[@]} -gt 0 ]]; then
        printf '%s\n' "${classes[@]}" | jq -s '{classes: .}'
    else
        echo '{"classes": []}'
    fi
}

#######################################
# Extract MATLAB documentation
# 
# Finds help text and documentation comments
#
# Arguments:
#   $1 - Path to MATLAB file
# Returns: JSON with documentation information
#######################################
extractor::lib::matlab::extract_documentation() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"has_help": false, "comment_lines": 0}'
        return
    fi
    
    # Check for help text (comments at the beginning of file or after function declaration)
    local has_help="false"
    local first_non_comment_line=$(grep -n "^[^%]" "$file" 2>/dev/null | head -1 | cut -d: -f1 || echo "0")
    
    if [[ $first_non_comment_line -gt 1 ]]; then
        has_help="true"
    fi
    
    # Count comment lines
    local comment_lines=$(grep -c "^[[:space:]]*%" "$file" 2>/dev/null || echo "0")
    
    # Check for section markers (%%)
    local section_count=$(grep -c "^%%" "$file" 2>/dev/null || echo "0")
    
    jq -n \
        --arg help "$has_help" \
        --arg comments "$comment_lines" \
        --arg sections "$section_count" \
        '{
            has_help: ($help == "true"),
            comment_lines: ($comments | tonumber),
            section_markers: ($sections | tonumber)
        }'
}

#######################################
# Extract script information
# 
# Determines if file is a script, function, or class file
#
# Arguments:
#   $1 - Path to MATLAB file
# Returns: File type and characteristics
#######################################
extractor::lib::matlab::extract_file_info() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"file_type": "unknown"}'
        return
    fi
    
    local file_type="script"
    local is_live_script="false"
    
    # Check file extension
    local ext="${file##*.}"
    if [[ "$ext" == "mlx" ]]; then
        is_live_script="true"
        file_type="live_script"
    fi
    
    # Check for function definition
    if grep -q "^[[:space:]]*function" "$file" 2>/dev/null; then
        file_type="function"
    fi
    
    # Check for class definition
    if grep -q "^[[:space:]]*classdef" "$file" 2>/dev/null; then
        file_type="class"
    fi
    
    # Count sections
    local section_count=$(grep -c "^%%" "$file" 2>/dev/null || echo "0")
    
    # Check for clear/clc/close commands (common in scripts)
    local has_clear="false"
    if grep -qE "^[[:space:]]*(clear|clc|close all)" "$file" 2>/dev/null; then
        has_clear="true"
    fi
    
    jq -n \
        --arg type "$file_type" \
        --arg live "$is_live_script" \
        --arg sections "$section_count" \
        --arg clear "$has_clear" \
        '{
            file_type: $type,
            is_live_script: ($live == "true"),
            section_count: ($sections | tonumber),
            has_clear_commands: ($clear == "true")
        }'
}

#######################################
# Extract plot and visualization commands
# 
# Identifies plotting and visualization functions
#
# Arguments:
#   $1 - Path to MATLAB file
# Returns: JSON with visualization information
#######################################
extractor::lib::matlab::extract_visualizations() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"has_plots": false}'
        return
    fi
    
    # Common plotting functions
    local plot_commands=("plot" "plot3" "scatter" "bar" "histogram" "surf" "mesh" "contour" "imagesc" "imshow")
    local has_plots="false"
    local plot_types=()
    
    for cmd in "${plot_commands[@]}"; do
        if grep -qE "\b$cmd\s*\(" "$file" 2>/dev/null; then
            has_plots="true"
            plot_types+=("$cmd")
        fi
    done
    
    # Check for figure creation
    local figure_count=$(grep -c "\bfigure\s*\(" "$file" 2>/dev/null || echo "0")
    
    # Check for subplots
    local has_subplots="false"
    if grep -qE "\bsubplot\s*\(" "$file" 2>/dev/null; then
        has_subplots="true"
    fi
    
    local plot_types_json="[]"
    if [[ ${#plot_types[@]} -gt 0 ]]; then
        plot_types_json=$(printf '%s\n' "${plot_types[@]}" | jq -R . | jq -s .)
    fi
    
    jq -n \
        --arg plots "$has_plots" \
        --arg figures "$figure_count" \
        --arg subplots "$has_subplots" \
        --argjson types "$plot_types_json" \
        '{
            has_plots: ($plots == "true"),
            figure_count: ($figures | tonumber),
            has_subplots: ($subplots == "true"),
            plot_types: $types
        }'
}

#######################################
# Extract toolbox dependencies
# 
# Identifies MATLAB toolbox usage
#
# Arguments:
#   $1 - Path to MATLAB file
# Returns: JSON with toolbox information
#######################################
extractor::lib::matlab::extract_toolboxes() {
    local file="$1"
    local toolboxes=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"toolboxes": []}'
        return
    fi
    
    # Check for common toolbox indicators
    grep -qE "\bfft\b|\bifft\b|\bfilter\b" "$file" 2>/dev/null && toolboxes+=("signal_processing")
    grep -qE "\bimread\b|\bimwrite\b|\bimadjust\b" "$file" 2>/dev/null && toolboxes+=("image_processing")
    grep -qE "\bsimulink\b|\bsim\b|\bset_param\b" "$file" 2>/dev/null && toolboxes+=("simulink")
    grep -qE "\boptimset\b|\bfmincon\b|\blinprog\b" "$file" 2>/dev/null && toolboxes+=("optimization")
    grep -qE "\bneuralNetwork\b|\btrainNetwork\b" "$file" 2>/dev/null && toolboxes+=("deep_learning")
    grep -qE "\bfitlm\b|\banova\b|\bttest\b" "$file" 2>/dev/null && toolboxes+=("statistics")
    grep -qE "\bpde\b|\bpdegplot\b" "$file" 2>/dev/null && toolboxes+=("pde")
    grep -qE "\bcontrol\b|\btf\b|\bss\b|\bpid\b" "$file" 2>/dev/null && toolboxes+=("control_systems")
    
    if [[ ${#toolboxes[@]} -gt 0 ]]; then
        printf '%s\n' "${toolboxes[@]}" | jq -R . | jq -s '{toolboxes: .}'
    else
        echo '{"toolboxes": []}'
    fi
}

#######################################
# Extract all MATLAB information from a file
# 
# Main extraction function that combines all MATLAB extractions
#
# Arguments:
#   $1 - MATLAB file path or directory
#   $2 - Component type (computation, simulation, analysis, etc.)
#   $3 - Scenario/resource name
# Returns: JSON lines with all MATLAB information
#######################################
extractor::lib::matlab::extract_all() {
    local path="$1"
    local component_type="${2:-computation}"
    local scenario_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's a MATLAB file
        case "$file_ext" in
            m|mlx|mat)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Extract components
        local file_info=$(extractor::lib::matlab::extract_file_info "$file")
        local functions=$(extractor::lib::matlab::extract_functions "$file")
        local classes=$(extractor::lib::matlab::extract_classes "$file")
        local documentation=$(extractor::lib::matlab::extract_documentation "$file")
        local visualizations=$(extractor::lib::matlab::extract_visualizations "$file")
        local toolboxes=$(extractor::lib::matlab::extract_toolboxes "$file")
        
        # Get counts
        local function_count=$(echo "$functions" | jq '.functions | length')
        local class_count=$(echo "$classes" | jq '.classes | length')
        local file_type=$(echo "$file_info" | jq -r '.file_type')
        
        # Build content summary
        local content="MATLAB: $filename | Type: $file_type | Component: $component_type"
        [[ $function_count -gt 0 ]] && content="$content | Functions: $function_count"
        [[ $class_count -gt 0 ]] && content="$content | Classes: $class_count"
        
        local has_plots=$(echo "$visualizations" | jq -r '.has_plots')
        [[ "$has_plots" == "true" ]] && content="$content | Has Visualizations"
        
        local toolbox_list=$(echo "$toolboxes" | jq -r '.toolboxes | join(", ")')
        [[ -n "$toolbox_list" && "$toolbox_list" != "" ]] && content="$content | Toolboxes: $toolbox_list"
        
        # Output main file overview
        jq -n \
            --arg content "$content" \
            --arg scenario "$scenario_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg line_count "$line_count" \
            --arg file_size "$file_size" \
            --argjson file_info "$file_info" \
            --argjson functions "$functions" \
            --argjson classes "$classes" \
            --argjson documentation "$documentation" \
            --argjson visualizations "$visualizations" \
            --argjson toolboxes "$toolboxes" \
            '{
                content: $content,
                metadata: {
                    scenario: $scenario,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    language: "matlab",
                    line_count: ($line_count | tonumber),
                    file_size: ($file_size | tonumber),
                    file_info: $file_info,
                    functions: $functions,
                    classes: $classes,
                    documentation: $documentation,
                    visualizations: $visualizations,
                    toolboxes: $toolboxes,
                    content_type: "matlab_code",
                    extraction_method: "matlab_parser"
                }
            }' | jq -c
            
        # Output individual function entries
        echo "$functions" | jq -c '.functions[]' 2>/dev/null | while read -r func_name; do
            func_name=$(echo "$func_name" | tr -d '"')
            local func_content="MATLAB Function: $func_name | File: $filename"
            
            jq -n \
                --arg content "$func_content" \
                --arg scenario "$scenario_name" \
                --arg source_file "$file" \
                --arg function_name "$func_name" \
                --arg component_type "$component_type" \
                '{
                    content: $content,
                    metadata: {
                        scenario: $scenario,
                        source_file: $source_file,
                        component_type: $component_type,
                        language: "matlab",
                        function_name: $function_name,
                        content_type: "matlab_function",
                        extraction_method: "matlab_parser"
                    }
                }' | jq -c
        done
        
        # Output individual class entries
        echo "$classes" | jq -c '.classes[]' 2>/dev/null | while read -r class_obj; do
            local class_name=$(echo "$class_obj" | jq -r '.name')
            local method_count=$(echo "$class_obj" | jq -r '.method_blocks')
            local class_content="MATLAB Class: $class_name | File: $filename | Methods: $method_count"
            
            jq -n \
                --arg content "$class_content" \
                --arg scenario "$scenario_name" \
                --arg source_file "$file" \
                --arg class_name "$class_name" \
                --arg component_type "$component_type" \
                --argjson class_info "$class_obj" \
                '{
                    content: $content,
                    metadata: {
                        scenario: $scenario,
                        source_file: $source_file,
                        component_type: $component_type,
                        language: "matlab",
                        class_name: $class_name,
                        class_info: $class_info,
                        content_type: "matlab_class",
                        extraction_method: "matlab_parser"
                    }
                }' | jq -c
        done
    elif [[ -d "$path" ]]; then
        # Directory - find all MATLAB files
        local matlab_files=()
        while IFS= read -r file; do
            matlab_files+=("$file")
        done < <(find "$path" -type f \( -name "*.m" -o -name "*.mlx" \) 2>/dev/null)
        
        if [[ ${#matlab_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${matlab_files[@]}"; do
            extractor::lib::matlab::extract_all "$file" "$component_type" "$scenario_name"
        done
    fi
}

#######################################
# Check if directory contains MATLAB files
# 
# Helper function to detect MATLAB presence
#
# Arguments:
#   $1 - Directory path
# Returns: 0 if MATLAB files found, 1 otherwise
#######################################
extractor::lib::matlab::has_matlab_files() {
    local dir="$1"
    
    if find "$dir" -type f \( -name "*.m" -o -name "*.mlx" \) 2>/dev/null | grep -q .; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::matlab::extract_functions
export -f extractor::lib::matlab::extract_classes
export -f extractor::lib::matlab::extract_documentation
export -f extractor::lib::matlab::extract_file_info
export -f extractor::lib::matlab::extract_visualizations
export -f extractor::lib::matlab::extract_toolboxes
export -f extractor::lib::matlab::extract_all
export -f extractor::lib::matlab::has_matlab_files