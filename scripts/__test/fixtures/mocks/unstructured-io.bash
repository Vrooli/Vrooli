#!/usr/bin/env bash
# Unstructured-IO Resource Mock Implementation
# Provides realistic mock responses for Unstructured document processing service

# Prevent duplicate loading
if [[ "${UNSTRUCTURED_IO_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export UNSTRUCTURED_IO_MOCK_LOADED="true"

#######################################
# Setup Unstructured-IO mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::unstructured-io::setup() {
    local state="${1:-healthy}"
    
    # Configure Unstructured-IO-specific environment
    export UNSTRUCTURED_IO_PORT="${UNSTRUCTURED_IO_PORT:-8000}"
    export UNSTRUCTURED_IO_BASE_URL="http://localhost:${UNSTRUCTURED_IO_PORT}"
    export UNSTRUCTURED_IO_CONTAINER_NAME="${TEST_NAMESPACE}_unstructured-io"
    export UNSTRUCTURED_IO_API_KEY="${UNSTRUCTURED_IO_API_KEY:-test-api-key}"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$UNSTRUCTURED_IO_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::unstructured-io::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::unstructured-io::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::unstructured-io::setup_installing_endpoints
            ;;
        "stopped")
            mock::unstructured-io::setup_stopped_endpoints
            ;;
        *)
            echo "[UNSTRUCTURED_IO_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[UNSTRUCTURED_IO_MOCK] Unstructured-IO mock configured with state: $state"
}

#######################################
# Setup healthy Unstructured-IO endpoints
#######################################
mock::unstructured-io::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/health" \
        '{"status":"ok","version":"0.10.0"}'
    
    # General processing endpoint
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/general/v0/general" \
        '{
            "elements": [
                {
                    "type": "Title",
                    "text": "Document Title",
                    "metadata": {
                        "filename": "test.pdf",
                        "page_number": 1,
                        "coordinates": {
                            "points": [[0, 0], [100, 0], [100, 20], [0, 20]],
                            "system": "PixelSpace"
                        }
                    }
                },
                {
                    "type": "NarrativeText",
                    "text": "This is the main body text of the document containing important information.",
                    "metadata": {
                        "filename": "test.pdf",
                        "page_number": 1
                    }
                },
                {
                    "type": "Table",
                    "text": "Header1\tHeader2\nValue1\tValue2",
                    "metadata": {
                        "filename": "test.pdf",
                        "page_number": 2,
                        "text_as_html": "<table><tr><th>Header1</th><th>Header2</th></tr><tr><td>Value1</td><td>Value2</td></tr></table>"
                    }
                }
            ],
            "metadata": {
                "filename": "test.pdf",
                "filetype": "application/pdf",
                "page_count": 2,
                "languages": ["eng"]
            }
        }' \
        "POST"
    
    # Extract tables endpoint
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/tables/v0/extract" \
        '{
            "tables": [
                {
                    "text": "Header1\tHeader2\tHeader3\nRow1Col1\tRow1Col2\tRow1Col3\nRow2Col1\tRow2Col2\tRow2Col3",
                    "html": "<table><thead><tr><th>Header1</th><th>Header2</th><th>Header3</th></tr></thead><tbody><tr><td>Row1Col1</td><td>Row1Col2</td><td>Row1Col3</td></tr><tr><td>Row2Col1</td><td>Row2Col2</td><td>Row2Col3</td></tr></tbody></table>",
                    "metadata": {
                        "page_number": 1,
                        "table_number": 1
                    }
                }
            ]
        }' \
        "POST"
    
    # Partition endpoint
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/partition/v0/partition" \
        '{
            "elements": [
                {
                    "type": "Header",
                    "text": "Section 1",
                    "metadata": {"section": "1.0"}
                },
                {
                    "type": "Text",
                    "text": "Content of section 1",
                    "metadata": {"section": "1.0"}
                }
            ]
        }' \
        "POST"
}

#######################################
# Setup unhealthy Unstructured-IO endpoints
#######################################
mock::unstructured-io::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/health" \
        '{"status":"unhealthy","error":"Processing engine unavailable"}' \
        "GET" \
        "503"
    
    # Processing endpoint returns error
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/general/v0/general" \
        '{"error":"Service unavailable","details":"Document processing engine offline"}' \
        "POST" \
        "503"
}

#######################################
# Setup installing Unstructured-IO endpoints
#######################################
mock::unstructured-io::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/health" \
        '{"status":"installing","progress":70,"current_step":"Loading NLP models"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$UNSTRUCTURED_IO_BASE_URL/general/v0/general" \
        '{"error":"Service is still initializing","eta_seconds":180}' \
        "POST" \
        "503"
}

#######################################
# Setup stopped Unstructured-IO endpoints
#######################################
mock::unstructured-io::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$UNSTRUCTURED_IO_BASE_URL"
}

#######################################
# Mock Unstructured-IO-specific operations
#######################################

# Mock document processing for different file types
mock::unstructured-io::simulate_document_processing() {
    local file_type="$1"
    local complexity="${2:-simple}"
    
    case "$file_type" in
        "pdf")
            echo '{
                "elements": [
                    {"type":"Title","text":"PDF Document"},
                    {"type":"NarrativeText","text":"PDF content extracted successfully."}
                ],
                "metadata": {"filetype":"application/pdf","page_count":5}
            }'
            ;;
        "docx")
            echo '{
                "elements": [
                    {"type":"Title","text":"Word Document"},
                    {"type":"ListItem","text":"Item 1"},
                    {"type":"ListItem","text":"Item 2"}
                ],
                "metadata": {"filetype":"application/vnd.openxmlformats-officedocument.wordprocessingml.document"}
            }'
            ;;
        "html")
            echo '{
                "elements": [
                    {"type":"Title","text":"Web Page Title"},
                    {"type":"NarrativeText","text":"Web content extracted."}
                ],
                "metadata": {"filetype":"text/html","links_found":3}
            }'
            ;;
        *)
            echo '{
                "elements": [
                    {"type":"Text","text":"Generic content extracted."}
                ],
                "metadata": {"filetype":"text/plain"}
            }'
            ;;
    esac
}

# Mock batch processing
mock::unstructured-io::simulate_batch_processing() {
    local file_count="$1"
    local results=()
    
    for i in $(seq 1 "$file_count"); do
        results+=("{\"file\":\"document_$i.pdf\",\"status\":\"completed\",\"elements_count\":10}")
    done
    
    echo "{\"batch_id\":\"batch_$(date +%s)\",\"total_files\":$file_count,\"completed\":$file_count,\"results\":[${results[*]}]}"
}

# Mock OCR processing
mock::unstructured-io::simulate_ocr_processing() {
    echo '{
        "elements": [
            {
                "type": "Text",
                "text": "This text was extracted using OCR from the image.",
                "metadata": {
                    "detection_method": "ocr",
                    "confidence": 0.92,
                    "language": "eng"
                }
            }
        ],
        "ocr_languages": ["eng"],
        "ocr_confidence": 0.92
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::unstructured-io::setup
export -f mock::unstructured-io::setup_healthy_endpoints
export -f mock::unstructured-io::setup_unhealthy_endpoints
export -f mock::unstructured-io::setup_installing_endpoints
export -f mock::unstructured-io::setup_stopped_endpoints
export -f mock::unstructured-io::simulate_document_processing
export -f mock::unstructured-io::simulate_batch_processing
export -f mock::unstructured-io::simulate_ocr_processing

echo "[UNSTRUCTURED_IO_MOCK] Unstructured-IO mock implementation loaded"