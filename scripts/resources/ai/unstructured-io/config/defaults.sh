#!/usr/bin/env bash

# Unstructured.io Resource Configuration Defaults
# This file contains all configuration constants and defaults for the Unstructured.io resource

# Service configuration
readonly UNSTRUCTURED_IO_PORT="${UNSTRUCTURED_IO_CUSTOM_PORT:-$(resources::get_default_port "unstructured-io")}"
readonly UNSTRUCTURED_IO_BASE_URL="http://localhost:${UNSTRUCTURED_IO_PORT}"
readonly UNSTRUCTURED_IO_SERVICE_NAME="unstructured-io"
readonly UNSTRUCTURED_IO_CONTAINER_NAME="vrooli-unstructured-io"

# Docker configuration
readonly UNSTRUCTURED_IO_IMAGE="downloads.unstructured.io/unstructured-io/unstructured-api:latest"
readonly UNSTRUCTURED_IO_API_PORT=8000  # Internal container port
readonly UNSTRUCTURED_IO_MEMORY_LIMIT="4g"
readonly UNSTRUCTURED_IO_CPU_LIMIT="2.0"

# Processing configuration
readonly UNSTRUCTURED_IO_DEFAULT_STRATEGY="hi_res"  # hi_res, fast, or auto
readonly UNSTRUCTURED_IO_DEFAULT_LANGUAGES="eng"    # OCR languages
readonly UNSTRUCTURED_IO_PARTITION_BY_API="true"   # Use API partitioning
readonly UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS="true"
readonly UNSTRUCTURED_IO_ENCODING="utf-8"

# Resource limits
readonly UNSTRUCTURED_IO_MAX_FILE_SIZE="50MB"
readonly UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES=$((50 * 1024 * 1024))  # 50MB in bytes
readonly UNSTRUCTURED_IO_TIMEOUT_SECONDS=300
readonly UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS=5

# Health check configuration
readonly UNSTRUCTURED_IO_HEALTH_ENDPOINT="/healthcheck"
readonly UNSTRUCTURED_IO_HEALTH_INTERVAL=60
readonly UNSTRUCTURED_IO_HEALTH_TIMEOUT=5

# API endpoints
readonly UNSTRUCTURED_IO_PROCESS_ENDPOINT="/general/v0/general"
readonly UNSTRUCTURED_IO_BATCH_ENDPOINT="/general/v0/general/batch"
readonly UNSTRUCTURED_IO_METRICS_ENDPOINT="/metrics"

# Supported file formats
readonly UNSTRUCTURED_IO_SUPPORTED_FORMATS=(
    "pdf"     # Portable Document Format
    "docx"    # Microsoft Word
    "doc"     # Legacy Word
    "pptx"    # Microsoft PowerPoint
    "ppt"     # Legacy PowerPoint
    "xlsx"    # Microsoft Excel
    "xls"     # Legacy Excel
    "txt"     # Plain text
    "rtf"     # Rich Text Format
    "html"    # Web pages
    "xml"     # XML documents
    "md"      # Markdown
    "rst"     # reStructuredText
    "odt"     # OpenDocument Text
    "epub"    # E-book format
    "msg"     # Outlook messages
    "eml"     # Email files
    "png"     # Images with text
    "jpg"     # Images with text
    "jpeg"    # Images with text
    "tiff"    # Images with text
    "bmp"     # Images with text
    "heic"    # Apple images
)

# Processing strategies
declare -A PROCESSING_STRATEGIES=(
    ["fast"]="Quick processing with basic extraction"
    ["hi_res"]="High resolution processing with advanced features"
    ["auto"]="Automatically select best strategy based on document"
)

# Output formats
declare -A OUTPUT_FORMATS=(
    ["json"]="Structured JSON with elements and metadata"
    ["markdown"]="Markdown formatted text for LLM consumption"
    ["text"]="Plain text extraction"
    ["elements"]="Detailed element-by-element breakdown"
)

#######################################
# Export configuration variables
#######################################
unstructured_io::export_config() {
    export UNSTRUCTURED_IO_PORT UNSTRUCTURED_IO_BASE_URL UNSTRUCTURED_IO_SERVICE_NAME
    export UNSTRUCTURED_IO_CONTAINER_NAME UNSTRUCTURED_IO_IMAGE UNSTRUCTURED_IO_API_PORT
    export UNSTRUCTURED_IO_MEMORY_LIMIT UNSTRUCTURED_IO_CPU_LIMIT
    export UNSTRUCTURED_IO_DEFAULT_STRATEGY UNSTRUCTURED_IO_DEFAULT_LANGUAGES
    export UNSTRUCTURED_IO_MAX_FILE_SIZE UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES
    export UNSTRUCTURED_IO_TIMEOUT_SECONDS UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS
    export UNSTRUCTURED_IO_HEALTH_ENDPOINT UNSTRUCTURED_IO_HEALTH_INTERVAL
    export UNSTRUCTURED_IO_PROCESS_ENDPOINT UNSTRUCTURED_IO_BATCH_ENDPOINT
}