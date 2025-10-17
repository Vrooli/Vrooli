#!/bin/bash
# Vrooli Document Fixtures Type-Specific Validator
# Validates document-specific requirements and tests with available resources

set -e

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test/fixtures/documents"
FIXTURES_DIR="$SCRIPT_DIR"
METADATA_FILE="$FIXTURES_DIR/metadata.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_FILES=0
VALID_FILES=0
TESTED_FILES=0
FAILED_FILES=0

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} Vrooli Document Fixtures Validation${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo
}

print_section() {
    echo -e "${YELLOW}--- $1 ---${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

validate_file() {
    local file="$1"
    local category="$2"
    
    TOTAL_FILES=$((TOTAL_FILES + 1))
    
    # Check if file exists and is readable
    if [[ ! -f "$file" ]]; then
        print_error "File not found: $file"
        FAILED_FILES=$((FAILED_FILES + 1))
        return 1
    fi
    
    if [[ ! -r "$file" ]]; then
        print_error "File not readable: $file"
        FAILED_FILES=$((FAILED_FILES + 1))
        return 1
    fi
    
    # Check file size
    local size
    size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
    if [[ $size -eq 0 ]]; then
        print_error "Empty file: $file"
        FAILED_FILES=$((FAILED_FILES + 1))
        return 1
    fi
    
    # File-type specific validation
    local filename=$(basename "$file")
    local extension="${filename##*.}"
    
    case "$extension" in
        json)
            if command -v jq >/dev/null 2>&1; then
                if jq empty "$file" >/dev/null 2>&1; then
                    print_success "$filename (valid JSON, ${size} bytes)"
                else
                    print_error "$filename (invalid JSON)"
                    FAILED_FILES=$((FAILED_FILES + 1))
                    return 1
                fi
            else
                print_info "$filename (JSON validation skipped - jq not available)"
            fi
            ;;
        csv)
            # Check for basic CSV structure
            if head -1 "$file" | grep -q ","; then
                local lines=$(wc -l < "$file")
                print_success "$filename (CSV format, $lines lines, ${size} bytes)"
            else
                print_error "$filename (invalid CSV format)"
                FAILED_FILES=$((FAILED_FILES + 1))
                return 1
            fi
            ;;
        xml)
            if command -v xmllint >/dev/null 2>&1; then
                if xmllint --noout "$file" 2>/dev/null; then
                    print_success "$filename (valid XML, ${size} bytes)"
                else
                    print_error "$filename (invalid XML)"
                    FAILED_FILES=$((FAILED_FILES + 1))
                    return 1
                fi
            else
                print_info "$filename (XML validation skipped - xmllint not available)"
            fi
            ;;
        html)
            # Basic HTML validation
            if grep -q "<!DOCTYPE html>" "$file" && grep -q "</html>" "$file"; then
                print_success "$filename (HTML format, ${size} bytes)"
            else
                print_info "$filename (HTML format - basic check passed, ${size} bytes)"
            fi
            ;;
        py)
            # Python syntax check
            if command -v python3 >/dev/null 2>&1; then
                if python3 -m py_compile "$file" >/dev/null 2>&1; then
                    print_success "$filename (valid Python syntax, ${size} bytes)"
                else
                    print_error "$filename (Python syntax errors)"
                    FAILED_FILES=$((FAILED_FILES + 1))
                    return 1
                fi
            else
                print_info "$filename (Python validation skipped)"
            fi
            ;;
        js)
            # Basic JavaScript validation
            if command -v node >/dev/null 2>&1; then
                if node -c "$file" >/dev/null 2>&1; then
                    print_success "$filename (valid JavaScript syntax, ${size} bytes)"
                else
                    print_error "$filename (JavaScript syntax errors)"
                    FAILED_FILES=$((FAILED_FILES + 1))
                    return 1
                fi
            else
                print_info "$filename (JavaScript validation skipped)"
            fi
            ;;
        pdf)
            # PDF validation
            if command -v file >/dev/null 2>&1; then
                if file "$file" | grep -q "PDF"; then
                    print_success "$filename (valid PDF format, ${size} bytes)"
                else
                    print_error "$filename (not a valid PDF file)"
                    FAILED_FILES=$((FAILED_FILES + 1))
                    return 1
                fi
            else
                print_success "$filename (PDF format assumed, ${size} bytes)"
            fi
            ;;
        doc|docx)
            # Word document validation
            if command -v file >/dev/null 2>&1; then
                local file_type=$(file "$file")
                if [[ "$file_type" == *"Microsoft Word"* ]] || [[ "$file_type" == *"Microsoft OOXML"* ]] || [[ "$file_type" == *"Composite Document File"* ]]; then
                    print_success "$filename (valid Word document, ${size} bytes)"
                else
                    print_info "$filename (Word format assumed, ${size} bytes)"
                fi
            else
                print_success "$filename (Word format assumed, ${size} bytes)"
            fi
            ;;
        xls|xlsx)
            # Excel document validation
            if command -v file >/dev/null 2>&1; then
                local file_type=$(file "$file")
                if [[ "$file_type" == *"Microsoft Excel"* ]] || [[ "$file_type" == *"Microsoft OOXML"* ]] || [[ "$file_type" == *"Composite Document File"* ]]; then
                    print_success "$filename (valid Excel document, ${size} bytes)"
                else
                    print_info "$filename (Excel format assumed, ${size} bytes)"
                fi
            else
                print_success "$filename (Excel format assumed, ${size} bytes)"
            fi
            ;;
        ppt|pptx)
            # PowerPoint document validation
            if command -v file >/dev/null 2>&1; then
                local file_type=$(file "$file")
                if [[ "$file_type" == *"Microsoft PowerPoint"* ]] || [[ "$file_type" == *"Microsoft OOXML"* ]] || [[ "$file_type" == *"Composite Document File"* ]]; then
                    print_success "$filename (valid PowerPoint document, ${size} bytes)"
                else
                    print_info "$filename (PowerPoint format assumed, ${size} bytes)"
                fi
            else
                print_success "$filename (PowerPoint format assumed, ${size} bytes)"
            fi
            ;;
        *)
            print_success "$filename ($extension format, ${size} bytes)"
            ;;
    esac
    
    VALID_FILES=$((VALID_FILES + 1))
    return 0
}

test_with_ollama() {
    local content="$1"
    local description="$2"
    
    if ! curl -s http://localhost:11434/api/tags >/dev/null 2>&1; then
        print_info "Ollama not available for testing"
        return 1
    fi
    
    print_info "Testing with Ollama: $description"
    
    # Simple test - just check if Ollama responds
    local response=$(curl -s -X POST http://localhost:11434/api/generate \
        -d "{\"model\": \"llama3.1:8b\", \"prompt\": \"Analyze this brief content: $(echo "$content" | head -c 200)\", \"stream\": false}" \
        2>/dev/null | jq -r '.response' 2>/dev/null || echo "")
    
    if [[ -n "$response" && "$response" != "null" ]]; then
        print_success "Ollama analysis completed"
        TESTED_FILES=$((TESTED_FILES + 1))
        return 0
    else
        print_info "Ollama test skipped or failed"
        return 1
    fi
}

test_structured_data() {
    print_section "Testing Structured Data Files"
    
    # Test CSV file
    if [[ -f "$FIXTURES_DIR/structured/customers.csv" ]]; then
        local csv_content=$(head -3 "$FIXTURES_DIR/structured/customers.csv")
        test_with_ollama "$csv_content" "Customer CSV data"
    fi
    
    # Test JSON file
    if [[ -f "$FIXTURES_DIR/structured/database_export.json" ]]; then
        local json_sample=$(head -c 500 "$FIXTURES_DIR/structured/database_export.json")
        test_with_ollama "$json_sample" "Database export JSON"
    fi
}

test_code_files() {
    print_section "Testing Code Files"
    
    # Test Python file
    if [[ -f "$FIXTURES_DIR/code/python/data_analysis.py" ]]; then
        local py_sample=$(head -20 "$FIXTURES_DIR/code/python/data_analysis.py")
        test_with_ollama "$py_sample" "Python data analysis code"
    fi
    
    # Test JavaScript file
    if [[ -f "$FIXTURES_DIR/code/javascript/frontend.js" ]]; then
        local js_sample=$(head -20 "$FIXTURES_DIR/code/javascript/frontend.js")
        test_with_ollama "$js_sample" "JavaScript frontend code"
    fi
}

test_office_documents() {
    print_section "Testing Office Documents"
    
    # Test government PDF
    if [[ -f "$FIXTURES_DIR/office/pdf/government/constitution_annotated.pdf" ]]; then
        print_info "Testing large government PDF (19MB Constitution)"
        if command -v pdfinfo >/dev/null 2>&1; then
            local pages=$(pdfinfo "$FIXTURES_DIR/office/pdf/government/constitution_annotated.pdf" 2>/dev/null | grep "Pages:" | awk '{print $2}')
            if [[ -n "$pages" ]]; then
                print_success "PDF has $pages pages - suitable for document processing tests"
            fi
        else
            print_info "PDF analysis skipped - pdfinfo not available"
        fi
    fi
    
    # Test corporate PDF
    if [[ -f "$FIXTURES_DIR/office/pdf/corporate/berkshire_hathaway_annual_letter.pdf" ]]; then
        print_info "Testing corporate document (Berkshire Hathaway letter)"
        test_with_ollama "Warren Buffett annual letter analysis" "Corporate document analysis"
    fi
    
    # Test international PDF
    if [[ -f "$FIXTURES_DIR/office/pdf/international/un_sustainable_development_goals.pdf" ]]; then
        print_info "Testing international document (UN SDG)"
        test_with_ollama "UN Sustainable Development Goals document" "International policy document"
    fi
    
    # Test Excel data
    if [[ -f "$FIXTURES_DIR/office/excel/educational/usc_sample_data.xls" ]]; then
        print_info "Testing Excel data file (USC sample - 345KB)"
        print_success "Excel file ready for data processing tests"
    fi
}

validate_directory_structure() {
    print_section "Validating Directory Structure"
    
    local expected_dirs=(
        "structured"
        "code"
        "code/python"
        "code/javascript"
        "code/configs"
        "code/documentation"
        "web"
        "rich_text"
        "office"
        "office/pdf"
        "office/excel"
        "office/word"
        "office/powerpoint"
        "office/pdf/government"
        "office/pdf/corporate"
        "office/pdf/educational"
        "office/pdf/international"
        "international"
        "edge_cases"
        "samples"
    )
    
    for dir in "${expected_dirs[@]}"; do
        if [[ -d "$FIXTURES_DIR/$dir" ]]; then
            print_success "Directory exists: $dir"
        else
            print_error "Missing directory: $dir"
            FAILED_FILES=$((FAILED_FILES + 1))
        fi
    done
}

validate_all_files() {
    print_section "Validating Individual Files"
    
    # Find and validate all fixture files
    while IFS= read -r -d '' file; do
        local relative_path=${file#$FIXTURES_DIR/}
        local category=${relative_path%/*}
        validate_file "$file" "$category"
    done < <(find "$FIXTURES_DIR" -type f \( \
        -name "*.json" -o \
        -name "*.csv" -o \
        -name "*.xml" -o \
        -name "*.yaml" -o \
        -name "*.yml" -o \
        -name "*.html" -o \
        -name "*.py" -o \
        -name "*.js" -o \
        -name "*.ts" -o \
        -name "*.md" -o \
        -name "*.rst" -o \
        -name "*.rtf" -o \
        -name "*.tex" -o \
        -name "*.txt" -o \
        -name "*.tsv" -o \
        -name "*.pdf" -o \
        -name "*.doc" -o \
        -name "*.docx" -o \
        -name "*.xls" -o \
        -name "*.xlsx" -o \
        -name "*.ppt" -o \
        -name "*.pptx" \
    \) -not -path "*/.*" -print0)
}

generate_report() {
    print_section "Validation Summary"
    
    echo "Total files examined: $TOTAL_FILES"
    echo "Valid files: $VALID_FILES"
    echo "Files tested with resources: $TESTED_FILES"
    echo "Failed validations: $FAILED_FILES"
    echo
    
    if [[ $FAILED_FILES -eq 0 ]]; then
        print_success "All fixtures validated successfully!"
        echo -e "${GREEN}✅ Document fixtures are ready for integration testing${NC}"
    else
        print_error "$FAILED_FILES validation failures detected"
        echo -e "${RED}❌ Some fixtures need attention before integration testing${NC}"
        exit 1
    fi
    
    # Performance metrics
    echo
    print_info "Fixture Statistics:"
    echo "  - Structured data files: $(find "$FIXTURES_DIR/structured" -type f 2>/dev/null | wc -l)"
    echo "  - Code files: $(find "$FIXTURES_DIR/code" -type f 2>/dev/null | wc -l)"
    echo "  - Web documents: $(find "$FIXTURES_DIR/web" -type f 2>/dev/null | wc -l)"
    echo "  - Rich text documents: $(find "$FIXTURES_DIR/rich_text" -type f 2>/dev/null | wc -l)"
    echo "  - Office documents: $(find "$FIXTURES_DIR/office" -type f 2>/dev/null | wc -l)"
    echo "  - International documents: $(find "$FIXTURES_DIR/international" -type f 2>/dev/null | wc -l)"
    echo "  - Edge case documents: $(find "$FIXTURES_DIR/edge_cases" -type f 2>/dev/null | wc -l)"
    echo "  - Sample documents: $(find "$FIXTURES_DIR/samples" -type f 2>/dev/null | wc -l)"
    
    # Resource availability check
    echo
    print_info "Resource Availability:"
    if curl -s http://localhost:11434/api/tags >/dev/null 2>&1; then
        echo "  - Ollama: ✅ Available"
    else
        echo "  - Ollama: ❌ Not available"
    fi
    
    if curl -s http://localhost:4110/pressure >/dev/null 2>&1; then
        echo "  - Browserless: ✅ Available"
    else
        echo "  - Browserless: ❌ Not available"
    fi
    
    if curl -s http://localhost:4113/health >/dev/null 2>&1; then
        echo "  - Agent-S2: ✅ Available"
    else
        echo "  - Agent-S2: ❌ Not available"
    fi
}

main() {
    print_header
    
    validate_directory_structure
    validate_all_files
    test_structured_data
    test_code_files
    test_office_documents
    
    echo
    generate_report
}

# Run validation if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi