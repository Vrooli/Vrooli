#!/usr/bin/env bats
# Enhanced Unstructured-IO Tests with Document Fixtures  
# Demonstrates document processing validation with real fixture files

# Load test infrastructure
load "$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/setup.bash"
load "$(dirname "${BATS_TEST_FILENAME}")/../../../__test/fixtures/fixture-loader.bash"

setup() {
    # Setup standard mocks and environment
    vrooli_auto_setup
    
    # Set test environment
    export UNSTRUCTURED_IO_CONTAINER_NAME="unstructured-io-test"
    export UNSTRUCTURED_IO_PORT="8000"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:8000"
    export UNSTRUCTURED_IO_API_TIMEOUT="30"
    
    # Source the unstructured-io functions to test
    source "$(dirname "${BATS_TEST_FILENAME}")/../common.sh"
    
    # Mock system functions
    system::is_port_in_use() { return 1; }
    docker() { echo "docker $*"; }
    curl() {
        echo "${MOCK_CURL_RESPONSE:-{\"elements\":[{\"type\":\"title\",\"text\":\"Sample Title\"}]}}"
    }
    
    export -f system::is_port_in_use docker curl
}

#######################################
# PDF Processing Tests with Fixtures
#######################################

@test "unstructured_io::process_pdf handles simple PDF fixture" {
    local pdf_file
    pdf_file=$(fixture_get_path "documents" "pdf/simple_text.pdf")
    
    # Mock structured document response
    fixture_mock_response "document_parse" "$pdf_file" '{
        "elements": [
            {"type": "title", "text": "Simple Document"},
            {"type": "paragraph", "text": "This is a simple PDF for testing."}
        ]
    }'
    
    run unstructured_io::process_document "$pdf_file"
    
    assert_success
    assert_output_contains "Simple Document"
    assert_output_contains "simple PDF for testing"
    
    # Verify correct API call
    fixture_assert_processed "$pdf_file" "-F files="
}

@test "unstructured_io::process_pdf handles complex PDF with tables" {
    local pdf_file
    pdf_file=$(fixture_get_path "documents" "pdf/table_document.pdf")
    
    fixture_mock_response "document_parse" "$pdf_file" '{
        "elements": [
            {"type": "title", "text": "Financial Report"},
            {"type": "table", "text": "Q1,100\nQ2,150\nQ3,200"},
            {"type": "paragraph", "text": "Table shows quarterly results."}
        ]
    }'
    
    run unstructured_io::process_document "$pdf_file"
    
    assert_success
    assert_output_contains "Financial Report"
    assert_output_contains "Q1,100"
    assert_output_contains "quarterly results"
}

@test "unstructured_io::process_pdf handles large PDF fixture" {
    local pdf_file
    pdf_file=$(fixture_get_path "documents" "pdf/large_document.pdf")
    
    fixture_mock_response "document_parse" "$pdf_file"
    
    run unstructured_io::process_document "$pdf_file"
    
    assert_success
    # Should handle large documents without timeout
}

#######################################
# Office Document Processing Tests
#######################################

@test "unstructured_io::process_document handles Word documents" {
    local word_file
    word_file=$(fixture_get_path "documents" "office/word/educational/ceu_sample.doc")
    
    fixture_mock_response "document_parse" "$word_file" '{
        "elements": [
            {"type": "title", "text": "Educational Document"},
            {"type": "paragraph", "text": "Sample educational content from CEU."}
        ]
    }'
    
    run unstructured_io::process_document "$word_file"
    
    assert_success
    assert_output_contains "Educational Document"
    assert_output_contains "educational content"
}

@test "unstructured_io::process_document handles Excel spreadsheets" {
    local excel_file
    excel_file=$(fixture_get_path "documents" "office/excel/educational/ou_sample_data.xlsx")
    
    fixture_mock_response "document_parse" "$excel_file" '{
        "elements": [
            {"type": "table", "text": "Name,Score\nJohn,95\nJane,87"},
            {"type": "paragraph", "text": "Student grade data"}
        ]
    }'
    
    run unstructured_io::process_document "$excel_file"
    
    assert_success
    assert_output_contains "Name,Score"
    assert_output_contains "John,95"
}

@test "unstructured_io::process_document handles PowerPoint presentations" {
    local ppt_file
    ppt_file=$(fixture_get_path "documents" "office/powerpoint/educational/noaa_bioluminescent_creatures_educational.pptx")
    
    fixture_mock_response "document_parse" "$ppt_file" '{
        "elements": [
            {"type": "title", "text": "Bioluminescent Creatures"},
            {"type": "paragraph", "text": "Educational presentation about marine life."}
        ]
    }'
    
    run unstructured_io::process_document "$ppt_file"
    
    assert_success
    assert_output_contains "Bioluminescent Creatures"
    assert_output_contains "marine life"
}

#######################################
# Structured Data Processing Tests
#######################################

@test "unstructured_io::process_document handles JSON fixture" {
    local json_file
    json_file=$(fixture_get_path "documents" "structured/database_export.json")
    
    fixture_mock_response "document_parse" "$json_file" '{
        "elements": [
            {"type": "data", "text": "Database export with 100 records"}
        ]
    }'
    
    run unstructured_io::process_document "$json_file"
    
    assert_success
    assert_output_contains "Database export"
}

@test "unstructured_io::process_document handles CSV fixture" {
    local csv_file
    csv_file=$(fixture_get_path "documents" "structured/customers.csv")
    
    fixture_mock_response "document_parse" "$csv_file" '{
        "elements": [
            {"type": "table", "text": "ID,Name,Email\n1,John Doe,john@example.com"}
        ]
    }'
    
    run unstructured_io::process_document "$csv_file"
    
    assert_success
    assert_output_contains "ID,Name,Email"
    assert_output_contains "john@example.com"
}

#######################################
# Error Handling Tests with Edge Cases
#######################################

@test "unstructured_io::process_document handles corrupted JSON fixture" {
    local corrupted_file
    corrupted_file=$(fixture_get_path "documents" "edge_cases/corrupted.json")
    
    fixture_mock_response "document_parse" "$corrupted_file" '{
        "error": "parse_error",
        "message": "Invalid JSON format"
    }'
    
    run unstructured_io::process_document "$corrupted_file"
    
    assert_failure
    assert_output_contains "parse_error"
    assert_output_contains "Invalid JSON format"
}

@test "unstructured_io::process_document handles empty PDF fixture" {
    local empty_file
    empty_file=$(fixture_get_path "documents" "edge_cases/empty.pdf")
    
    fixture_mock_response "document_parse" "$empty_file" '{
        "elements": [],
        "warning": "No extractable content found"
    }'
    
    run unstructured_io::process_document "$empty_file"
    
    assert_success
    assert_output_contains "No extractable content"
}

#######################################
# International Document Tests
#######################################

@test "unstructured_io::process_document handles international text fixtures" {
    # Test Arabic document
    local arabic_file
    arabic_file=$(fixture_get_path "documents" "international/arabic_document.txt")
    
    fixture_mock_response "document_parse" "$arabic_file" '{
        "elements": [{"type": "paragraph", "text": "Arabic text content"}],
        "language": "ar"
    }'
    
    run unstructured_io::process_document "$arabic_file"
    
    assert_success
    assert_output_contains "Arabic text"
    
    # Test Chinese document
    local chinese_file
    chinese_file=$(fixture_get_path "documents" "international/chinese_technical.txt")
    
    fixture_mock_response "document_parse" "$chinese_file" '{
        "elements": [{"type": "paragraph", "text": "Chinese technical content"}],
        "language": "zh"
    }'
    
    run unstructured_io::process_document "$chinese_file"
    
    assert_success
    assert_output_contains "Chinese technical"
}

#######################################
# Batch Processing Tests
#######################################

@test "unstructured_io::process_batch handles multiple document fixtures" {
    # Get sample documents from different categories
    local -a doc_files=(
        "$(fixture_get_path "documents" "pdf/simple_text.pdf")"
        "$(fixture_get_path "documents" "structured/customers.csv")"
        "$(fixture_get_path "documents" "samples/test-readme.md")"
    )
    
    # Mock batch processing response
    fixture_mock_response "document_parse" "batch" '{
        "results": [
            {"file": "simple_text.pdf", "status": "success"},
            {"file": "customers.csv", "status": "success"},
            {"file": "test-readme.md", "status": "success"}
        ]
    }'
    
    run unstructured_io::process_batch "${doc_files[@]}"
    
    assert_success
    assert_output_contains "simple_text.pdf"
    assert_output_contains "customers.csv" 
    assert_output_contains "test-readme.md"
    
    # Verify all files were processed
    for doc_file in "${doc_files[@]}"; do
        fixture_assert_processed "$doc_file" "-F files="
    done
}

#######################################
# Format Detection Tests
#######################################

@test "unstructured_io::detect_format correctly identifies fixture formats" {
    # Test various document formats
    local pdf_file
    pdf_file=$(fixture_get_path "documents" "pdf/simple_text.pdf")
    
    run unstructured_io::detect_format "$pdf_file"
    assert_success
    assert_output_contains "pdf"
    
    local docx_file
    docx_file=$(fixture_get_path "documents" "office/word/educational/iup_test_document.docx")
    
    run unstructured_io::detect_format "$docx_file"
    assert_success
    assert_output_contains "docx"
    
    local csv_file
    csv_file=$(fixture_get_path "documents" "structured/customers.csv")
    
    run unstructured_io::detect_format "$csv_file"  
    assert_success
    assert_output_contains "csv"
}