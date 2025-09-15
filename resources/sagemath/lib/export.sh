#!/usr/bin/env bash
################################################################################
# SageMath Export Operations
# 
# Export mathematical expressions to LaTeX, MathML, and image formats
################################################################################

# Export expression to LaTeX format
sagemath::export::latex() {
    local expression="${1:-}"
    local output_file="${2:-}"
    
    if [[ -z "$expression" ]]; then
        echo "Error: No expression provided" >&2
        echo "Usage: resource-sagemath export latex \"expression\" [output_file]" >&2
        echo "Example: resource-sagemath export latex \"solve(x^2 - 4 == 0, x)\"" >&2
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    # Set default output file if not provided
    if [[ -z "$output_file" ]]; then
        output_file="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs/latex_${timestamp}.tex"
    fi
    
    echo "🔄 Exporting to LaTeX format..."
    
    # First calculate the expression and get its LaTeX representation
    # Create a calculation that declares variables and returns LaTeX
    local calc_code="var('x y z t r theta'); result = $expression; latex(result)"
    local latex_result=$(sagemath::content::calculate "$calc_code")
    if [[ $? -ne 0 ]]; then
        echo "❌ Failed to generate LaTeX" >&2
        return 1
    fi
    
    # Extract just the LaTeX output (remove the calculation header)
    local latex_output=$(echo "$latex_result" | grep -A 100 "Result:" | sed 's/.*Result: //')
    
    # Save to file
    mkdir -p "$(dirname "$output_file")"
    echo "$latex_output" > "$output_file"
    
    echo "✅ LaTeX export complete"
    echo "📄 LaTeX: $latex_output"
    echo "💾 Saved to: $output_file"
}

# Export expression to MathML format
sagemath::export::mathml() {
    local expression="${1:-}"
    local output_file="${2:-}"
    
    if [[ -z "$expression" ]]; then
        echo "Error: No expression provided" >&2
        echo "Usage: resource-sagemath export mathml \"expression\" [output_file]" >&2
        echo "Example: resource-sagemath export mathml \"x^2 + y^2\"" >&2
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    # Set default output file if not provided
    if [[ -z "$output_file" ]]; then
        output_file="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs/mathml_${timestamp}.xml"
    fi
    
    echo "🔄 Exporting to MathML format..."
    
    # Calculate the expression first
    local calc_result=$(sagemath::content::calculate "var('x y z t r theta'); $expression")
    if [[ $? -ne 0 ]]; then
        echo "❌ Failed to evaluate expression" >&2
        return 1
    fi
    
    # Get LaTeX representation (MathML is not directly supported, so we'll create basic MathML from LaTeX)
    local calc_code="var('x y z t r theta'); result = $expression; latex(result)"
    local latex_result=$(sagemath::content::calculate "$calc_code")
    local latex_output=$(echo "$latex_result" | grep -A 100 "Result:" | sed 's/.*Result: //')
    
    # Create basic MathML wrapper
    local mathml_output="<math xmlns=\"http://www.w3.org/1998/Math/MathML\"><mtext>$latex_output</mtext></math>"
    
    # Save to file
    mkdir -p "$(dirname "$output_file")"
    echo "$mathml_output" > "$output_file"
    
    echo "✅ MathML export complete"
    echo "📄 MathML saved to: $output_file"
    echo "🔍 Preview: ${mathml_output:0:200}..."
}

# Render equation to PNG image
sagemath::export::image() {
    local expression="${1:-}"
    local output_file="${2:-}"
    local dpi="${3:-150}"
    local fontsize="${4:-16}"
    
    if [[ -z "$expression" ]]; then
        echo "Error: No expression provided" >&2
        echo "Usage: resource-sagemath export image \"expression\" [output_file] [dpi] [fontsize]" >&2
        echo "Example: resource-sagemath export image \"x^2 + y^2\" output.png 300 20" >&2
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    # Set default output file if not provided
    if [[ -z "$output_file" ]]; then
        output_file="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs/equation_${timestamp}.png"
    fi
    
    echo "🔄 Rendering equation to image..."
    
    # Use text() function to create a text plot with the LaTeX representation
    local plot_cmd="var('x y z t r theta'); expr = $expression; p = text('\$' + latex(expr) + '\$', (0.5, 0.5), fontsize=$fontsize, color='black'); p.save('$output_file', dpi=$dpi, figsize=(8, 2))"
    
    local result=$(sagemath::content::calculate "$plot_cmd")
    if [[ $? -eq 0 ]]; then
        echo "✅ Equation image created"
        echo "🖼️ Image saved to: $output_file"
        
        # Also show the LaTeX representation
        local calc_code="var('x y z t r theta'); result = $expression; latex(result)"
        local latex_result=$(sagemath::content::calculate "$calc_code")
        local latex_output=$(echo "$latex_result" | grep -A 100 "Result:" | sed 's/.*Result: //')
        echo "📄 LaTeX: $latex_output"
    else
        echo "❌ Image rendering failed" >&2
        return 1
    fi
}

# Export to multiple formats at once
sagemath::export::all() {
    local expression="${1:-}"
    local base_name="${2:-export}"
    
    if [[ -z "$expression" ]]; then
        echo "Error: No expression provided" >&2
        echo "Usage: resource-sagemath export all \"expression\" [base_name]" >&2
        echo "Example: resource-sagemath export all \"solve(x^2 - 4 == 0, x)\" solution" >&2
        return 1
    fi
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local output_dir="/home/matthalloran8/Vrooli/data/resources/sagemath/outputs"
    
    echo "🔄 Exporting to all formats..."
    
    # Export to LaTeX
    local latex_file="${output_dir}/${base_name}_${timestamp}.tex"
    sagemath::export::latex "$expression" "$latex_file"
    
    # Export to MathML
    local mathml_file="${output_dir}/${base_name}_${timestamp}.xml"
    sagemath::export::mathml "$expression" "$mathml_file"
    
    # Export to image
    local image_file="${output_dir}/${base_name}_${timestamp}.png"
    sagemath::export::image "$expression" "$image_file"
    
    echo ""
    echo "✅ All exports complete:"
    echo "  📄 LaTeX: $latex_file"
    echo "  📄 MathML: $mathml_file"
    echo "  🖼️  Image: $image_file"
}

# List available export formats
sagemath::export::formats() {
    echo "Available export formats:"
    echo "  latex  - LaTeX mathematical notation"
    echo "  mathml - MathML for web display"
    echo "  image  - PNG image of rendered equation"
    echo "  all    - Export to all formats at once"
    echo ""
    echo "Examples:"
    echo "  resource-sagemath export latex \"x^2 + y^2 = r^2\""
    echo "  resource-sagemath export image \"\\sum_{i=1}^{n} i = \\frac{n(n+1)}{2}\""
    echo "  resource-sagemath export all \"solve(x^2 - 4 == 0, x)\" solution"
}