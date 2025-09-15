#!/usr/bin/env bash
################################################################################
# SageMath Content Operations
# 
# Business functionality for SageMath mathematical computations
################################################################################

# Universal Contract v2.0 handler for content::add
sagemath::content::add() {
    local file=""
    local name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                # Assume it's the file if no flag provided
                file="$1"
                shift
                ;;
        esac
    done
    
    if [ -z "$file" ]; then
        echo "Error: No file specified" >&2
        echo "Usage: resource-sagemath content add --file <file.sage|file.ipynb> [--name <custom_name>]" >&2
        return 1
    fi
    
    if [ ! -f "$file" ]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    local filename=$(basename "$file")
    local extension="${filename##*.}"
    
    # Use custom name if provided
    if [ -n "$name" ]; then
        filename="$name"
    fi
    
    echo "Adding SageMath content: $filename"
    
    case "$extension" in
        sage|py)
            cp "$file" "$SAGEMATH_SCRIPTS_DIR/$filename"
            echo "✅ Script added to: $SAGEMATH_SCRIPTS_DIR/$filename"
            ;;
        ipynb)
            cp "$file" "$SAGEMATH_NOTEBOOKS_DIR/$filename"
            echo "✅ Notebook added to: $SAGEMATH_NOTEBOOKS_DIR/$filename"
            ;;
        *)
            echo "❌ Unsupported file type: $extension" >&2
            echo "Supported types: .sage, .py, .ipynb" >&2
            return 1
            ;;
    esac
    
    return 0
}

# Universal Contract v2.0 handler for content::list
sagemath::content::list() {
    local format="${1:-text}"
    
    echo "SageMath Content Inventory:"
    echo ""
    
    # List scripts
    echo "📝 Scripts:"
    local scripts_count=0
    if [ -d "$SAGEMATH_SCRIPTS_DIR" ]; then
        for script in "$SAGEMATH_SCRIPTS_DIR"/*.sage "$SAGEMATH_SCRIPTS_DIR"/*.py; do
            if [ -f "$script" ]; then
                echo "   $(basename "$script")"
                ((scripts_count++)) || true
            fi
        done 2>/dev/null
    fi
    [ $scripts_count -eq 0 ] && echo "   (none)"
    echo ""
    
    # List notebooks
    echo "📓 Notebooks:"
    local notebooks_count=0
    if [ -d "$SAGEMATH_NOTEBOOKS_DIR" ]; then
        for notebook in "$SAGEMATH_NOTEBOOKS_DIR"/*.ipynb; do
            if [ -f "$notebook" ]; then
                echo "   $(basename "$notebook")"
                ((notebooks_count++)) || true
            fi
        done 2>/dev/null
    fi
    [ $notebooks_count -eq 0 ] && echo "   (none)"
    echo ""
    
    # Summary
    echo "📊 Summary: $scripts_count scripts, $notebooks_count notebooks"
    
    return 0
}

# Universal Contract v2.0 handler for content::get
sagemath::content::get() {
    local item="${1:-}"
    
    if [ -z "$item" ]; then
        echo "Error: No content item specified" >&2
        echo "Usage: resource-sagemath content get <filename>" >&2
        return 1
    fi
    
    # Search for the file
    local found_file=""
    
    # Check scripts directory
    if [ -f "$SAGEMATH_SCRIPTS_DIR/$item" ]; then
        found_file="$SAGEMATH_SCRIPTS_DIR/$item"
    elif [ -f "$SAGEMATH_NOTEBOOKS_DIR/$item" ]; then
        found_file="$SAGEMATH_NOTEBOOKS_DIR/$item"
    fi
    
    if [ -z "$found_file" ]; then
        echo "❌ Content not found: $item" >&2
        echo "Use 'resource-sagemath content list' to see available content"
        return 1
    fi
    
    echo "📄 Content: $item"
    echo "📁 Path: $found_file"
    echo ""
    echo "--- Content ---"
    cat "$found_file"
    
    return 0
}

# Universal Contract v2.0 handler for content::remove
sagemath::content::remove() {
    local item=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                item="$2"
                shift 2
                ;;
            *)
                item="$1"
                shift
                ;;
        esac
    done
    
    if [ -z "$item" ]; then
        echo "Error: No content item specified" >&2
        echo "Usage: resource-sagemath content remove <filename>" >&2
        return 1
    fi
    
    # Search for the file
    local found_file=""
    
    if [ -f "$SAGEMATH_SCRIPTS_DIR/$item" ]; then
        found_file="$SAGEMATH_SCRIPTS_DIR/$item"
    elif [ -f "$SAGEMATH_NOTEBOOKS_DIR/$item" ]; then
        found_file="$SAGEMATH_NOTEBOOKS_DIR/$item"
    fi
    
    if [ -z "$found_file" ]; then
        echo "❌ Content not found: $item" >&2
        return 1
    fi
    
    echo "🗑️  Removing content: $item"
    if rm "$found_file"; then
        echo "✅ Content removed successfully"
        return 0
    else
        echo "❌ Failed to remove content" >&2
        return 1
    fi
}

# Clean up old temporary calculation files
sagemath::content::cleanup() {
    local days="${1:-7}"  # Default to 7 days old
    local SAGEMATH_SCRIPTS_DIR="${SAGEMATH_DATA_DIR}/scripts"
    
    echo "🧹 Cleaning up temporary calculation files older than $days days..."
    
    # Count files before cleanup
    local count_before=$(find "$SAGEMATH_SCRIPTS_DIR" -name "temp_calc_*.sage*" -type f 2>/dev/null | wc -l)
    
    if [ "$count_before" -eq 0 ]; then
        echo "✅ No temporary files to clean up"
        return 0
    fi
    
    # Remove temporary calculation files older than specified days
    local deleted=0
    while IFS= read -r file; do
        if rm "$file" 2>/dev/null; then
            ((deleted++))
        fi
    done < <(find "$SAGEMATH_SCRIPTS_DIR" -name "temp_calc_*.sage*" -type f -mtime +"$days" 2>/dev/null)
    
    # Also clean up any temp files without proper dates (just remove all temp_calc files for now)
    while IFS= read -r file; do
        if rm "$file" 2>/dev/null; then
            ((deleted++))
        fi
    done < <(find "$SAGEMATH_SCRIPTS_DIR" -name "temp_calc_*.sage*" -type f 2>/dev/null)
    
    echo "✅ Cleaned up $deleted temporary files (was $count_before files)"
    return 0
}

# Universal Contract v2.0 handler for content::execute
sagemath::content::execute() {
    local expression_or_file=""
    local name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                expression_or_file="$1"
                shift
                ;;
        esac
    done
    
    # If name is provided, use it as the file
    if [ -n "$name" ]; then
        expression_or_file="$name"
    fi
    
    if [ -z "$expression_or_file" ]; then
        echo "Error: No expression or file specified" >&2
        echo "Usage: resource-sagemath content execute <expression|filename>" >&2
        echo "Examples:"
        echo "  resource-sagemath content execute 'factor(100)'"
        echo "  resource-sagemath content execute script.sage"
        return 1
    fi
    
    if ! sagemath_container_running; then
        echo "Starting SageMath..."
        sagemath::docker::start
    fi
    
    # Check if it's a file or expression
    if [ -f "$expression_or_file" ] || [ -f "$SAGEMATH_SCRIPTS_DIR/$expression_or_file" ]; then
        # It's a file - execute script
        local script_file="$expression_or_file"
        [ -f "$SAGEMATH_SCRIPTS_DIR/$expression_or_file" ] && script_file="$SAGEMATH_SCRIPTS_DIR/$expression_or_file"
        
        echo "🧮 Executing SageMath script: $(basename "$script_file")"
        
        local script_name=$(basename "$script_file")
        local internal_path="/home/sage/scripts/$script_name"
        
        # Only copy if it's not already in the scripts directory
        if [[ "$script_file" != "$SAGEMATH_SCRIPTS_DIR/$script_name" ]]; then
            cp "$script_file" "$SAGEMATH_SCRIPTS_DIR/"
        fi
        
        # Execute script
        local output_file="$SAGEMATH_OUTPUTS_DIR/${script_name%.*}_$(date +%Y%m%d_%H%M%S).out"
        
        if docker exec "$SAGEMATH_CONTAINER_NAME" sage "$internal_path" > "$output_file" 2>&1; then
            echo "✅ Script executed successfully"
            echo "📄 Output saved to: $output_file"
            echo ""
            cat "$output_file"
            return 0
        else
            echo "❌ Script execution failed" >&2
            cat "$output_file" >&2
            return 1
        fi
    else
        # It's an expression - calculate directly
        echo "🧮 Calculating: $expression_or_file"
        
        # Check if it's a plot command
        local is_plot=false
        if echo "$expression_or_file" | grep -qE "plot\(|plot3d\(|matrix_plot\(|list_plot\(|density_plot\(|contour_plot\("; then
            is_plot=true
        fi
        
        # Clean up old temp files first (older than 1 day)
        find "$SAGEMATH_SCRIPTS_DIR" -name "temp_calc_*.sage*" -type f -mtime +1 -delete 2>/dev/null || true
        
        # Create temporary script
        local temp_script="$SAGEMATH_SCRIPTS_DIR/temp_calc_$$.sage"
        local timestamp=$(date +%Y%m%d_%H%M%S)
        
        if [ "$is_plot" = true ]; then
            # For plot operations, save to file
            local plot_file="/home/sage/outputs/plot_${timestamp}.png"
            cat > "$temp_script" << EOF
try:
    # Execute the full expression
    exec(compile('''$expression_or_file''', '<string>', 'exec'))
    
    # Find the last plot object in locals
    plot_obj = None
    for var_name in reversed(list(locals().keys())):
        var = locals()[var_name]
        if hasattr(var, 'save') and hasattr(var, 'plot'):
            plot_obj = var
            break
    
    # If no plot object found, try to get the last expression result
    if plot_obj is None:
        import sys
        if '_' in dir():
            plot_obj = _
    
    # Check if the expression itself creates a plot
    if 'plot' in '''$expression_or_file''' or 'plot3d' in '''$expression_or_file''':
        plot_expr = '''$expression_or_file'''
        # Extract just the plot command if there are multiple statements
        if ';' in plot_expr:
            plot_expr = plot_expr.split(';')[-1].strip()
        plot_obj = eval(plot_expr)
    
    if plot_obj and hasattr(plot_obj, 'save'):
        plot_obj.save('$plot_file', dpi=150)
        print("Plot saved to: outputs/plot_${timestamp}.png")
        print("Result:", plot_obj)
    else:
        print("Result:", plot_obj if plot_obj else "Plot generation attempted")
except Exception as e:
    print("Error:", str(e))
EOF
        else
            # For non-plot operations
            cat > "$temp_script" << EOF
try:
    result = $expression_or_file
    print("Result:", result)
except Exception as e:
    print("Error:", str(e))
EOF
        fi
        
        # Execute calculation
        local output=$(docker exec "$SAGEMATH_CONTAINER_NAME" sage "/home/sage/scripts/$(basename "$temp_script")" 2>&1)
        
        # Clean up
        rm -f "$temp_script"
        
        # Display result
        echo "📊 $output"
        
        # If plot was generated, show the file path
        if [ "$is_plot" = true ] && [ -f "$SAGEMATH_OUTPUTS_DIR/plot_${timestamp}.png" ]; then
            echo "📈 Plot saved to: $SAGEMATH_OUTPUTS_DIR/plot_${timestamp}.png"
        fi
        
        return 0
    fi
}

# Add custom content subcommand: calculate (direct mathematical expressions)
sagemath::content::calculate() {
    local expression="${1:-}"
    
    if [ -z "$expression" ]; then
        echo "Error: No expression provided" >&2
        echo "Usage: resource-sagemath content calculate \"expression\"" >&2
        return 1
    fi
    
    sagemath::content::execute "$expression"
}

# Add custom content subcommand: notebook (open Jupyter interface)
sagemath::content::notebook() {
    if ! sagemath_container_running; then
        echo "Starting SageMath..."
        sagemath::docker::start
    fi
    
    # Get Jupyter token
    local token=$(docker logs "$SAGEMATH_CONTAINER_NAME" 2>&1 | grep -o 'token=[a-z0-9]*' | tail -1 | cut -d= -f2)
    local url="http://localhost:${SAGEMATH_PORT_JUPYTER}"
    
    if [ -n "$token" ]; then
        url="${url}/?token=${token}"
    fi
    
    echo "🚀 Opening SageMath Jupyter notebook interface..."
    echo "🌐 URL: $url"
    
    # Try to open in browser if available
    if command -v xdg-open >/dev/null 2>&1; then
        xdg-open "$url" 2>/dev/null &
        echo "✅ Browser opened automatically"
    elif command -v open >/dev/null 2>&1; then
        open "$url" 2>/dev/null &
        echo "✅ Browser opened automatically"
    else
        echo "📋 Please open this URL in your browser manually"
    fi
    
    return 0
}