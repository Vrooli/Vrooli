#!/usr/bin/env bash

# Unstructured.io Resource Configuration Defaults
# This file contains all configuration constants and defaults for the Unstructured.io resource

# Get script directory for relative path resolution
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
CONFIG_DIR="${APP_ROOT}/resources/unstructured-io/config"

# shellcheck disable=SC1091
source "${CONFIG_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Service configuration
# Check if variables are already set to avoid readonly conflicts in tests
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_PORT="; then
    readonly UNSTRUCTURED_IO_PORT="${UNSTRUCTURED_IO_CUSTOM_PORT:-$(resources::get_default_port "unstructured-io")}"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_BASE_URL="; then
    readonly UNSTRUCTURED_IO_BASE_URL="http://localhost:${UNSTRUCTURED_IO_PORT}"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_SERVICE_NAME="; then
    readonly UNSTRUCTURED_IO_SERVICE_NAME="unstructured-io"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_CONTAINER_NAME="; then
    readonly UNSTRUCTURED_IO_CONTAINER_NAME="vrooli-unstructured-io"
fi

# Docker configuration
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_IMAGE="; then
    readonly UNSTRUCTURED_IO_IMAGE="downloads.unstructured.io/unstructured-io/unstructured-api:0.0.78"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_API_PORT="; then
    readonly UNSTRUCTURED_IO_API_PORT=8000  # Internal container port
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_MEMORY_LIMIT="; then
    readonly UNSTRUCTURED_IO_MEMORY_LIMIT="4g"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_CPU_LIMIT="; then
    readonly UNSTRUCTURED_IO_CPU_LIMIT="2.0"
fi

# Processing configuration
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_DEFAULT_STRATEGY="; then
    readonly UNSTRUCTURED_IO_DEFAULT_STRATEGY="hi_res"  # hi_res, fast, or auto
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_DEFAULT_LANGUAGES="; then
    readonly UNSTRUCTURED_IO_DEFAULT_LANGUAGES="eng"    # OCR languages
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_PARTITION_BY_API="; then
    readonly UNSTRUCTURED_IO_PARTITION_BY_API="true"   # Use API partitioning
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS="; then
    readonly UNSTRUCTURED_IO_INCLUDE_PAGE_BREAKS="true"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_ENCODING="; then
    readonly UNSTRUCTURED_IO_ENCODING="utf-8"
fi

# Resource limits
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_MAX_FILE_SIZE="; then
    readonly UNSTRUCTURED_IO_MAX_FILE_SIZE="50MB"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES="; then
    readonly UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES=$((50 * 1024 * 1024))  # 50MB in bytes
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_TIMEOUT_SECONDS="; then
    readonly UNSTRUCTURED_IO_TIMEOUT_SECONDS=300
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS="; then
    readonly UNSTRUCTURED_IO_MAX_CONCURRENT_REQUESTS=5
fi

# Health check configuration
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_HEALTH_ENDPOINT="; then
    readonly UNSTRUCTURED_IO_HEALTH_ENDPOINT="/healthcheck"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_HEALTH_INTERVAL="; then
    readonly UNSTRUCTURED_IO_HEALTH_INTERVAL=60
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_HEALTH_TIMEOUT="; then
    readonly UNSTRUCTURED_IO_HEALTH_TIMEOUT=5
fi

# API endpoints
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_PROCESS_ENDPOINT="; then
    readonly UNSTRUCTURED_IO_PROCESS_ENDPOINT="/general/v0/general"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_BATCH_ENDPOINT="; then
    readonly UNSTRUCTURED_IO_BATCH_ENDPOINT="/general/v0/general/batch"
fi
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_METRICS_ENDPOINT="; then
    readonly UNSTRUCTURED_IO_METRICS_ENDPOINT=""  # Metrics endpoint not available in this version
fi

# Supported file formats
if ! readonly -p | grep -q "^declare -[a-z]*r[a-z]* UNSTRUCTURED_IO_SUPPORTED_FORMATS="; then
    readonly UNSTRUCTURED_IO_SUPPORTED_FORMATS=(
    "pdf"     # Portable Document Format
    "docx"    # Microsoft Word
    "doc"     # Legacy Word
    "pptx"    # Microsoft PowerPoint
    "ppt"     # Legacy PowerPoint
    "xlsx"    # Microsoft Excel
    "xls"     # Legacy Excel
    "csv"     # Comma-separated values
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
fi

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
    # The formats array is already readonly, just export the reference
}

# Export the supported formats array
export UNSTRUCTURED_IO_SUPPORTED_FORMATS

# Export function for subshell availability
export -f unstructured_io::export_config