#!/bin/bash
# PDF Export Module for Privacy & Terms Generator
# Version: 1.0.0
# Description: Export legal documents to PDF using Browserless

set -euo pipefail

# Source paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Source generator for db functions
source "${SCRIPT_DIR}/generator.sh"

# Function to convert markdown to HTML with styling
markdown_to_html() {
    local content="$1"
    local business_name="${2:-Company}"
    local doc_type="${3:-document}"
    
    # Generate styled HTML document
    cat <<EOF
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>${business_name} - ${doc_type^}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 40px 20px;
            background: white;
        }
        h1 {
            color: #2c3e50;
            border-bottom: 3px solid #3498db;
            padding-bottom: 10px;
            margin-bottom: 30px;
            font-size: 2em;
        }
        h2 {
            color: #34495e;
            margin-top: 30px;
            margin-bottom: 15px;
            font-size: 1.5em;
        }
        h3 {
            color: #7f8c8d;
            margin-top: 20px;
            margin-bottom: 10px;
            font-size: 1.2em;
        }
        p {
            margin: 10px 0;
            text-align: justify;
        }
        ul, ol {
            margin: 10px 0 10px 20px;
        }
        li {
            margin: 5px 0;
        }
        strong {
            color: #2c3e50;
        }
        .header {
            text-align: center;
            margin-bottom: 40px;
            padding-bottom: 20px;
            border-bottom: 1px solid #ecf0f1;
        }
        .footer {
            margin-top: 50px;
            padding-top: 20px;
            border-top: 1px solid #ecf0f1;
            font-size: 0.9em;
            color: #7f8c8d;
            text-align: center;
        }
        .section {
            margin: 30px 0;
        }
        .effective-date {
            font-style: italic;
            color: #7f8c8d;
            text-align: center;
            margin: 20px 0;
        }
        @media print {
            body {
                padding: 0;
                margin: 0;
            }
            .page-break {
                page-break-after: always;
            }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>${business_name}</h1>
        <h2 style="border: none; color: #7f8c8d; font-weight: normal;">${doc_type^}</h2>
    </div>
    <div class="content">
$(echo "$content" | sed 's/^# \(.*\)$/<h1>\1<\/h1>/g' | \
                   sed 's/^## \(.*\)$/<h2>\1<\/h2>/g' | \
                   sed 's/^### \(.*\)$/<h3>\1<\/h3>/g' | \
                   sed 's/^\* \(.*\)$/<li>\1<\/li>/g' | \
                   sed 's/^[0-9]\+\. \(.*\)$/<li>\1<\/li>/g' | \
                   sed 's/\*\*\([^*]*\)\*\*/<strong>\1<\/strong>/g' | \
                   sed 's/^$/<\/p><p>/g' | \
                   sed '1s/^/<p>/' | \
                   sed '$s/$/<\/p>/')
    </div>
    <div class="footer">
        <p>Generated on $(date '+%B %d, %Y') by Vrooli Privacy & Terms Generator</p>
        <p>This document is provided for informational purposes. Please consult with legal counsel for your specific needs.</p>
    </div>
</body>
</html>
EOF
}

# Function to generate PDF using Browserless
generate_pdf_with_browserless() {
    local html_content="$1"
    local output_file="${2:-document.pdf}"
    
    # Check if Browserless is available
    if ! command -v resource-browserless &>/dev/null; then
        echo "Error: Browserless resource not available" >&2
        return 1
    fi
    
    # Create temporary HTML file
    local temp_html="/tmp/legal_doc_$$.html"
    echo "$html_content" > "$temp_html"
    
    # Generate PDF using Browserless
    resource-browserless content pdf \
        --url "file://${temp_html}" \
        --output "${output_file}" \
        --format "A4" \
        --margin-top "20mm" \
        --margin-bottom "20mm" \
        --margin-left "15mm" \
        --margin-right "15mm" \
        --print-background
    
    local result=$?
    
    # Clean up temp file
    rm -f "$temp_html"
    
    if [ $result -eq 0 ]; then
        echo "PDF generated successfully: ${output_file}"
    else
        echo "Error: Failed to generate PDF" >&2
        return 1
    fi
    
    return $result
}

# Function to export document from database to PDF
export_document_to_pdf() {
    local doc_id="$1"
    local output_file="${2:-document.pdf}"
    
    # Fetch document from database
    local doc_data=$(db_query "SELECT d.content, d.document_type, b.name 
                               FROM generated_documents d
                               JOIN business_profiles b ON d.business_id = b.id
                               WHERE d.id = '${doc_id}'")
    
    if [ -z "$doc_data" ]; then
        echo "Error: Document not found (ID: ${doc_id})" >&2
        return 1
    fi
    
    # Parse document data
    local content=$(echo "$doc_data" | cut -f1)
    local doc_type=$(echo "$doc_data" | cut -f2)
    local business_name=$(echo "$doc_data" | cut -f3)
    
    # Convert to HTML
    local html_content=$(markdown_to_html "$content" "$business_name" "$doc_type")
    
    # Generate PDF
    generate_pdf_with_browserless "$html_content" "$output_file"
}

# Function to batch export multiple documents
batch_export_to_pdf() {
    local business_name="$1"
    local output_dir="${2:-.}"
    
    echo "Exporting all documents for ${business_name}..."
    
    # Get all documents for the business
    local doc_ids=$(db_query "SELECT d.id, d.document_type
                              FROM generated_documents d
                              JOIN business_profiles b ON d.business_id = b.id
                              WHERE b.name = '${business_name}'
                              AND d.status = 'active'")
    
    if [ -z "$doc_ids" ]; then
        echo "No documents found for ${business_name}" >&2
        return 1
    fi
    
    # Export each document
    echo "$doc_ids" | while read -r id doc_type; do
        local output_file="${output_dir}/${business_name// /_}_${doc_type}.pdf"
        echo "Exporting ${doc_type} to ${output_file}..."
        export_document_to_pdf "$id" "$output_file"
    done
    
    echo "Batch export completed"
}

# Function to generate PDF directly from markdown content
generate_pdf_from_markdown() {
    local markdown_content="$1"
    local output_file="$2"
    local business_name="${3:-Company}"
    local doc_type="${4:-document}"
    
    # Convert markdown to HTML
    local html_content=$(markdown_to_html "$markdown_content" "$business_name" "$doc_type")
    
    # Generate PDF
    generate_pdf_with_browserless "$html_content" "$output_file"
}

# Export functions
export -f markdown_to_html
export -f generate_pdf_with_browserless
export -f export_document_to_pdf
export -f batch_export_to_pdf
export -f generate_pdf_from_markdown

# Main execution
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    case "${1:-}" in
        export)
            shift
            export_document_to_pdf "$@"
            ;;
        batch)
            shift
            batch_export_to_pdf "$@"
            ;;
        convert)
            shift
            if [ $# -lt 2 ]; then
                echo "Usage: $0 convert <input.md> <output.pdf> [business_name] [doc_type]"
                exit 1
            fi
            content=$(cat "$1")
            generate_pdf_from_markdown "$content" "$2" "${3:-Company}" "${4:-document}"
            ;;
        *)
            echo "Usage: $0 {export|batch|convert} [options]"
            echo ""
            echo "Commands:"
            echo "  export <doc_id> <output.pdf>         - Export document to PDF"
            echo "  batch <business_name> [output_dir]   - Export all business documents"
            echo "  convert <input.md> <output.pdf>       - Convert markdown file to PDF"
            exit 1
            ;;
    esac
fi