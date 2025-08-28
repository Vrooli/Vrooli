#!/usr/bin/env bash
# R Language Parser for Qdrant Embeddings  
# Extracts functions, data operations, visualizations, and statistical analyses from R files
#
# Handles:
# - Function definitions and documentation
# - Data manipulation (dplyr, data.table)
# - Statistical analyses and models
# - Visualizations (ggplot2, base R plots)
# - R Markdown documents (.Rmd)
# - Package dependencies

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract functions from R file
# 
# Finds function definitions and their documentation
#
# Arguments:
#   $1 - Path to R file
# Returns: JSON with function information
#######################################
extractor::lib::r::extract_functions() {
    local file="$1"
    local functions=()
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Find function definitions (name <- function(...) or name = function(...))
    while IFS= read -r line; do
        # Match function assignments
        if [[ "$line" =~ ^[[:space:]]*([a-zA-Z_][a-zA-Z0-9_.]*)[[:space:]]*(\<-|=)[[:space:]]*function[[:space:]]*\( ]]; then
            local func_name="${BASH_REMATCH[1]}"
            functions+=("$func_name")
        fi
    done < "$file"
    
    # Also find S3/S4 method definitions
    local s3_methods=$(grep -oE "^[a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*[[:space:]]*<-[[:space:]]*function" "$file" 2>/dev/null | \
        sed 's/[[:space:]]*<-.*//' || echo "")
    
    if [[ -n "$s3_methods" ]]; then
        while IFS= read -r method; do
            [[ -n "$method" ]] && functions+=("$method")
        done <<< "$s3_methods"
    fi
    
    if [[ ${#functions[@]} -gt 0 ]]; then
        printf '%s\n' "${functions[@]}" | sort -u | jq -R . | jq -s '{functions: .}'
    else
        echo '{"functions": []}'
    fi
}

#######################################
# Extract data operations
# 
# Identifies data manipulation patterns
#
# Arguments:
#   $1 - Path to R file
# Returns: JSON with data operation information
#######################################
extractor::lib::r::extract_data_operations() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"has_data_operations": false}'
        return
    fi
    
    # Check for dplyr operations
    local has_dplyr="false"
    if grep -qE "\b(mutate|filter|select|summarise|summarize|group_by|arrange|join)\s*\(" "$file" 2>/dev/null; then
        has_dplyr="true"
    fi
    
    # Check for data.table operations
    local has_datatable="false"
    if grep -qE "\[DT|\[data\.table|:=|\.SD|\.N" "$file" 2>/dev/null; then
        has_datatable="true"
    fi
    
    # Check for base R data operations
    local has_base_ops="false"
    if grep -qE "\b(read\.csv|read\.table|write\.csv|merge|aggregate|subset)\s*\(" "$file" 2>/dev/null; then
        has_base_ops="true"
    fi
    
    # Check for tidyr operations
    local has_tidyr="false"
    if grep -qE "\b(pivot_longer|pivot_wider|gather|spread|separate|unite)\s*\(" "$file" 2>/dev/null; then
        has_tidyr="true"
    fi
    
    # Count pipe operators
    local pipe_count=$(grep -c "%>%" "$file" 2>/dev/null || echo "0")
    local native_pipe_count=$(grep -c "|>" "$file" 2>/dev/null || echo "0")
    
    jq -n \
        --arg dplyr "$has_dplyr" \
        --arg datatable "$has_datatable" \
        --arg base "$has_base_ops" \
        --arg tidyr "$has_tidyr" \
        --arg pipes "$pipe_count" \
        --arg native_pipes "$native_pipe_count" \
        '{
            has_dplyr: ($dplyr == "true"),
            has_datatable: ($datatable == "true"),
            has_base_operations: ($base == "true"),
            has_tidyr: ($tidyr == "true"),
            pipe_operators: ($pipes | tonumber),
            native_pipe_operators: ($native_pipes | tonumber)
        }'
}

#######################################
# Extract visualization information
# 
# Identifies plotting and visualization code
#
# Arguments:
#   $1 - Path to R file
# Returns: JSON with visualization information
#######################################
extractor::lib::r::extract_visualizations() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"has_visualizations": false}'
        return
    fi
    
    # Check for ggplot2
    local has_ggplot="false"
    if grep -qE "\bggplot\s*\(|\bgeom_|aes\s*\(" "$file" 2>/dev/null; then
        has_ggplot="true"
    fi
    
    # Check for base R plots
    local has_base_plots="false"
    if grep -qE "\b(plot|hist|boxplot|barplot|scatter|pairs)\s*\(" "$file" 2>/dev/null; then
        has_base_plots="true"
    fi
    
    # Check for plotly
    local has_plotly="false"
    if grep -qE "\bplot_ly\s*\(|ggplotly\s*\(" "$file" 2>/dev/null; then
        has_plotly="true"
    fi
    
    # Check for lattice
    local has_lattice="false"
    if grep -qE "\b(xyplot|bwplot|dotplot|splom)\s*\(" "$file" 2>/dev/null; then
        has_lattice="true"
    fi
    
    # Count different geom types for ggplot
    local geom_types=()
    for geom in "point" "line" "bar" "histogram" "boxplot" "density" "smooth" "text" "tile" "violin"; do
        if grep -qE "geom_$geom\s*\(" "$file" 2>/dev/null; then
            geom_types+=("geom_$geom")
        fi
    done
    
    local geom_json="[]"
    if [[ ${#geom_types[@]} -gt 0 ]]; then
        geom_json=$(printf '%s\n' "${geom_types[@]}" | jq -R . | jq -s .)
    fi
    
    jq -n \
        --arg ggplot "$has_ggplot" \
        --arg base "$has_base_plots" \
        --arg plotly "$has_plotly" \
        --arg lattice "$has_lattice" \
        --argjson geoms "$geom_json" \
        '{
            has_ggplot: ($ggplot == "true"),
            has_base_plots: ($base == "true"),
            has_plotly: ($plotly == "true"),
            has_lattice: ($lattice == "true"),
            geom_types: $geoms
        }'
}

#######################################
# Extract statistical analyses
# 
# Identifies statistical models and tests
#
# Arguments:
#   $1 - Path to R file
# Returns: JSON with statistical information
#######################################
extractor::lib::r::extract_statistics() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"has_statistics": false}'
        return
    fi
    
    local stat_functions=()
    
    # Linear models
    grep -qE "\b(lm|glm|gam|lmer|glmer)\s*\(" "$file" 2>/dev/null && stat_functions+=("linear_models")
    
    # Statistical tests
    grep -qE "\b(t\.test|wilcox\.test|chisq\.test|anova|aov|cor\.test)\s*\(" "$file" 2>/dev/null && stat_functions+=("statistical_tests")
    
    # Time series
    grep -qE "\b(ts|arima|forecast|acf|pacf|stl)\s*\(" "$file" 2>/dev/null && stat_functions+=("time_series")
    
    # Machine learning
    grep -qE "\b(randomForest|svm|kmeans|hclust|prcomp|caret::train)\s*\(" "$file" 2>/dev/null && stat_functions+=("machine_learning")
    
    # Descriptive statistics
    grep -qE "\b(mean|median|sd|var|summary|describe)\s*\(" "$file" 2>/dev/null && stat_functions+=("descriptive")
    
    local stat_json="[]"
    if [[ ${#stat_functions[@]} -gt 0 ]]; then
        stat_json=$(printf '%s\n' "${stat_functions[@]}" | jq -R . | jq -s .)
    fi
    
    jq -n \
        --arg has_stats "$([[ ${#stat_functions[@]} -gt 0 ]] && echo "true" || echo "false")" \
        --argjson types "$stat_json" \
        '{
            has_statistics: ($has_stats == "true"),
            analysis_types: $types
        }'
}

#######################################
# Extract package dependencies
# 
# Identifies required R packages
#
# Arguments:
#   $1 - Path to R file
# Returns: JSON with package information
#######################################
extractor::lib::r::extract_packages() {
    local file="$1"
    local packages=()
    
    if [[ ! -f "$file" ]]; then
        echo '{"packages": []}'
        return
    fi
    
    # Find library() calls
    local lib_packages=$(grep -oE "library\([\"']?[a-zA-Z][a-zA-Z0-9.]*[\"']?\)" "$file" 2>/dev/null | \
        sed -E "s/library\([\"']?([a-zA-Z][a-zA-Z0-9.]*)[\"']?\)/\1/" || echo "")
    
    # Find require() calls
    local req_packages=$(grep -oE "require\([\"']?[a-zA-Z][a-zA-Z0-9.]*[\"']?\)" "$file" 2>/dev/null | \
        sed -E "s/require\([\"']?([a-zA-Z][a-zA-Z0-9.]*)[\"']?\)/\1/" || echo "")
    
    # Find package::function calls
    local ns_packages=$(grep -oE "[a-zA-Z][a-zA-Z0-9.]*::[a-zA-Z]" "$file" 2>/dev/null | \
        sed 's/::.*//' | sort -u || echo "")
    
    # Combine all packages
    if [[ -n "$lib_packages" ]]; then
        while IFS= read -r pkg; do
            [[ -n "$pkg" ]] && packages+=("$pkg")
        done <<< "$lib_packages"
    fi
    
    if [[ -n "$req_packages" ]]; then
        while IFS= read -r pkg; do
            [[ -n "$pkg" ]] && packages+=("$pkg")
        done <<< "$req_packages"
    fi
    
    if [[ -n "$ns_packages" ]]; then
        while IFS= read -r pkg; do
            [[ -n "$pkg" ]] && packages+=("$pkg")
        done <<< "$ns_packages"
    fi
    
    if [[ ${#packages[@]} -gt 0 ]]; then
        printf '%s\n' "${packages[@]}" | sort -u | jq -R . | jq -s '{packages: .}'
    else
        echo '{"packages": []}'
    fi
}

#######################################
# Extract R Markdown information
# 
# Processes .Rmd files to extract chunks and metadata
#
# Arguments:
#   $1 - Path to .Rmd file
# Returns: JSON with R Markdown information
#######################################
extractor::lib::r::extract_rmarkdown() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo '{"is_rmarkdown": false}'
        return
    fi
    
    # Count code chunks
    local chunk_count=$(grep -c "^```{r" "$file" 2>/dev/null || echo "0")
    
    # Check for YAML header
    local has_yaml="false"
    if head -1 "$file" | grep -q "^---$"; then
        has_yaml="true"
    fi
    
    # Extract output format from YAML if present
    local output_format=""
    if [[ "$has_yaml" == "true" ]]; then
        output_format=$(awk '/^---$/,/^---$/' "$file" | grep "output:" | sed 's/.*output:[[:space:]]*//' || echo "")
    fi
    
    # Count different chunk options
    local echo_false=$(grep -c "echo.*=.*FALSE" "$file" 2>/dev/null || echo "0")
    local eval_false=$(grep -c "eval.*=.*FALSE" "$file" 2>/dev/null || echo "0")
    local include_false=$(grep -c "include.*=.*FALSE" "$file" 2>/dev/null || echo "0")
    
    jq -n \
        --arg chunks "$chunk_count" \
        --arg yaml "$has_yaml" \
        --arg output "$output_format" \
        --arg echo_f "$echo_false" \
        --arg eval_f "$eval_false" \
        --arg include_f "$include_false" \
        '{
            is_rmarkdown: true,
            code_chunks: ($chunks | tonumber),
            has_yaml_header: ($yaml == "true"),
            output_format: $output,
            chunks_echo_false: ($echo_f | tonumber),
            chunks_eval_false: ($eval_f | tonumber),
            chunks_include_false: ($include_f | tonumber)
        }'
}

#######################################
# Extract all R information from a file
# 
# Main extraction function that combines all R extractions
#
# Arguments:
#   $1 - R file path or directory
#   $2 - Component type (analysis, visualization, modeling, etc.)
#   $3 - Scenario/resource name
# Returns: JSON lines with all R information
#######################################
extractor::lib::r::extract_all() {
    local path="$1"
    local component_type="${2:-analysis}"
    local scenario_name="${3:-unknown}"
    
    if [[ -f "$path" ]]; then
        # Single file
        local file="$path"
        local filename=$(basename "$file")
        local file_ext="${filename##*.}"
        
        # Check if it's an R file
        case "$file_ext" in
            R|r|Rmd|rmd)
                ;;
            *)
                return 1
                ;;
        esac
        
        # Get file statistics
        local line_count=$(wc -l < "$file" 2>/dev/null || echo "0")
        local file_size=$(wc -c < "$file" 2>/dev/null || echo "0")
        
        # Determine file type
        local file_type="script"
        local rmd_info="{}"
        
        if [[ "$file_ext" == "Rmd" ]] || [[ "$file_ext" == "rmd" ]]; then
            file_type="rmarkdown"
            rmd_info=$(extractor::lib::r::extract_rmarkdown "$file")
        fi
        
        # Extract components
        local functions=$(extractor::lib::r::extract_functions "$file")
        local data_ops=$(extractor::lib::r::extract_data_operations "$file")
        local visualizations=$(extractor::lib::r::extract_visualizations "$file")
        local statistics=$(extractor::lib::r::extract_statistics "$file")
        local packages=$(extractor::lib::r::extract_packages "$file")
        
        # Get counts
        local function_count=$(echo "$functions" | jq '.functions | length')
        local package_count=$(echo "$packages" | jq '.packages | length')
        
        # Build content summary
        local content="R: $filename | Type: $file_type | Component: $component_type"
        [[ $function_count -gt 0 ]] && content="$content | Functions: $function_count"
        [[ $package_count -gt 0 ]] && content="$content | Packages: $package_count"
        
        local has_viz=$(echo "$visualizations" | jq -r '.has_ggplot or .has_base_plots')
        [[ "$has_viz" == "true" ]] && content="$content | Has Visualizations"
        
        local has_stats=$(echo "$statistics" | jq -r '.has_statistics')
        [[ "$has_stats" == "true" ]] && content="$content | Has Statistical Analysis"
        
        # Output main file overview
        jq -n \
            --arg content "$content" \
            --arg scenario "$scenario_name" \
            --arg source_file "$file" \
            --arg filename "$filename" \
            --arg component_type "$component_type" \
            --arg file_type "$file_type" \
            --arg line_count "$line_count" \
            --arg file_size "$file_size" \
            --argjson functions "$functions" \
            --argjson data_ops "$data_ops" \
            --argjson visualizations "$visualizations" \
            --argjson statistics "$statistics" \
            --argjson packages "$packages" \
            --argjson rmd_info "$rmd_info" \
            '{
                content: $content,
                metadata: {
                    scenario: $scenario,
                    source_file: $source_file,
                    filename: $filename,
                    component_type: $component_type,
                    language: "r",
                    file_type: $file_type,
                    line_count: ($line_count | tonumber),
                    file_size: ($file_size | tonumber),
                    functions: $functions,
                    data_operations: $data_ops,
                    visualizations: $visualizations,
                    statistics: $statistics,
                    packages: $packages,
                    rmarkdown: $rmd_info,
                    content_type: "r_code",
                    extraction_method: "r_parser"
                }
            }' | jq -c
            
        # Output individual function entries
        echo "$functions" | jq -c '.functions[]' 2>/dev/null | while read -r func_name; do
            func_name=$(echo "$func_name" | tr -d '"')
            local func_content="R Function: $func_name | File: $filename"
            
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
                        language: "r",
                        function_name: $function_name,
                        content_type: "r_function",
                        extraction_method: "r_parser"
                    }
                }' | jq -c
        done
        
        # Output package dependency summary if significant
        if [[ $package_count -gt 3 ]]; then
            local pkg_list=$(echo "$packages" | jq -r '.packages | join(", ")')
            local pkg_content="R Dependencies: $pkg_list | File: $filename"
            
            jq -n \
                --arg content "$pkg_content" \
                --arg scenario "$scenario_name" \
                --arg source_file "$file" \
                --arg component_type "$component_type" \
                --argjson packages "$packages" \
                '{
                    content: $content,
                    metadata: {
                        scenario: $scenario,
                        source_file: $source_file,
                        component_type: $component_type,
                        language: "r",
                        packages: $packages,
                        content_type: "r_dependencies",
                        extraction_method: "r_parser"
                    }
                }' | jq -c
        fi
    elif [[ -d "$path" ]]; then
        # Directory - find all R files
        local r_files=()
        while IFS= read -r file; do
            r_files+=("$file")
        done < <(find "$path" -type f \( -name "*.R" -o -name "*.r" -o -name "*.Rmd" -o -name "*.rmd" \) 2>/dev/null)
        
        if [[ ${#r_files[@]} -eq 0 ]]; then
            return 1
        fi
        
        for file in "${r_files[@]}"; do
            extractor::lib::r::extract_all "$file" "$component_type" "$scenario_name"
        done
    fi
}

#######################################
# Check if directory contains R files
# 
# Helper function to detect R presence
#
# Arguments:
#   $1 - Directory path
# Returns: 0 if R files found, 1 otherwise
#######################################
extractor::lib::r::has_r_files() {
    local dir="$1"
    
    if find "$dir" -type f \( -name "*.R" -o -name "*.r" -o -name "*.Rmd" -o -name "*.rmd" \) 2>/dev/null | grep -q .; then
        return 0
    else
        return 1
    fi
}

# Export all functions
export -f extractor::lib::r::extract_functions
export -f extractor::lib::r::extract_data_operations
export -f extractor::lib::r::extract_visualizations
export -f extractor::lib::r::extract_statistics
export -f extractor::lib::r::extract_packages
export -f extractor::lib::r::extract_rmarkdown
export -f extractor::lib::r::extract_all
export -f extractor::lib::r::has_r_files