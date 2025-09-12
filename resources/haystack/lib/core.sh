#!/usr/bin/env bash
################################################################################
# Haystack Core Library - v2.0 Universal Contract Compliant
# 
# Core functionality for Haystack resource
################################################################################

set -euo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HAYSTACK_DIR="${APP_ROOT}/resources/haystack"
HAYSTACK_LIB_DIR="${HAYSTACK_DIR}/lib"
HAYSTACK_CONFIG_DIR="${HAYSTACK_DIR}/config"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/resources/port-registry.sh"
source "${HAYSTACK_CONFIG_DIR}/defaults.sh"

# Paths
export HAYSTACK_DATA_DIR="${var_ROOT_DIR}/data/haystack"
export HAYSTACK_LOG_DIR="${var_ROOT_DIR}/logs/haystack"
export HAYSTACK_PID_FILE="${var_ROOT_DIR}/run/haystack.pid"
export HAYSTACK_LOG_FILE="${HAYSTACK_LOG_DIR}/haystack.log"
export HAYSTACK_VENV_DIR="${HAYSTACK_DATA_DIR}/venv"
export HAYSTACK_SCRIPTS_DIR="${HAYSTACK_DIR}/scripts"

# ============================================================================
# Core Functions
# ============================================================================

# Get configured port for Haystack
haystack::get_port() {
    resources::get_port "haystack" || echo "8075"
}

# Check if Haystack is installed
haystack::is_installed() {
    [[ -d "${HAYSTACK_VENV_DIR}" ]] && \
    [[ -f "${HAYSTACK_VENV_DIR}/bin/python" ]] && \
    [[ -f "${HAYSTACK_SCRIPTS_DIR}/server.py" ]]
}

# Check if Haystack is running
haystack::is_running() {
    local port
    port=$(haystack::get_port)
    
    # Check health endpoint
    if timeout 5 curl -sf "http://localhost:${port}/health" &>/dev/null; then
        return 0
    fi
    
    # Check PID file
    if [[ -f "${HAYSTACK_PID_FILE}" ]]; then
        local pid
        pid=$(cat "${HAYSTACK_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            return 0
        fi
    fi
    
    return 1
}

# Get Haystack status
haystack::status() {
    local status="unknown"
    local health="unknown"
    local port
    port=$(haystack::get_port)
    
    if haystack::is_running; then
        status="running"
        # Get health details
        local health_response
        health_response=$(timeout 5 curl -sf "http://localhost:${port}/health" 2>/dev/null || echo "{}")
        if [[ -n "${health_response}" ]]; then
            health="healthy"
        else
            health="unhealthy"
        fi
    elif haystack::is_installed; then
        status="stopped"
        health="not running"
    else
        status="not installed"
        health="not installed"
    fi
    
    cat <<EOF
{
  "status": "${status}",
  "health": "${health}",
  "port": ${port},
  "pid_file": "${HAYSTACK_PID_FILE}",
  "log_file": "${HAYSTACK_LOG_FILE}",
  "data_dir": "${HAYSTACK_DATA_DIR}"
}
EOF
}

# Initialize directories
haystack::init_directories() {
    mkdir -p "${HAYSTACK_DATA_DIR}"
    mkdir -p "${HAYSTACK_LOG_DIR}"
    mkdir -p "${var_ROOT_DIR}/run"
    mkdir -p "${HAYSTACK_SCRIPTS_DIR}"
}

# Validate environment
haystack::validate_environment() {
    # Check Python availability
    if ! command -v "${HAYSTACK_PYTHON}" &>/dev/null; then
        log::error "Python not found: ${HAYSTACK_PYTHON}"
        return 1
    fi
    
    # Check Python version (need 3.8+)
    local python_version
    python_version=$("${HAYSTACK_PYTHON}" -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")' 2>/dev/null)
    
    if [[ "${python_version}" < "3.8" ]]; then
        log::error "Python 3.8+ required, found: ${python_version}"
        return 1
    fi
    
    return 0
}

# Create server script if it doesn't exist
haystack::create_server_script() {
    if [[ -f "${HAYSTACK_SCRIPTS_DIR}/server.py" ]]; then
        return 0
    fi
    
    cat > "${HAYSTACK_SCRIPTS_DIR}/server.py" << 'EOF'
#!/usr/bin/env python3
"""
Haystack REST API Server
Provides document indexing and querying capabilities
"""

import os
import sys
import json
import logging
from typing import List, Dict, Any, Optional
from pathlib import Path

from fastapi import FastAPI, HTTPException, UploadFile, File
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import uvicorn

# Configure logging
logging.basicConfig(
    level=os.getenv("HAYSTACK_LOG_LEVEL", "INFO"),
    format='{"time":"%(asctime)s","level":"%(levelname)s","message":"%(message)s"}' if os.getenv("HAYSTACK_LOG_FORMAT") == "json" else "%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(title="Haystack API", version="2.0.0")

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=os.getenv("HAYSTACK_CORS_ORIGINS", "*").split(","),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Simple in-memory document store for now
document_store = []

# Request/Response models
class Document(BaseModel):
    content: str
    metadata: Optional[Dict[str, Any]] = {}

class IndexRequest(BaseModel):
    documents: List[Document]

class QueryRequest(BaseModel):
    query: str
    top_k: int = 10

class HealthResponse(BaseModel):
    status: str
    service: str
    version: str = "2.0.0"

# Endpoints
@app.get("/health", response_model=HealthResponse)
async def health():
    """Health check endpoint"""
    return HealthResponse(status="healthy", service="haystack")

@app.post("/index")
async def index_documents(request: IndexRequest):
    """Index documents into the store"""
    try:
        for doc in request.documents:
            document_store.append({
                "content": doc.content,
                "metadata": doc.metadata
            })
        return {"status": "success", "documents_indexed": len(request.documents)}
    except Exception as e:
        logger.error(f"Error indexing documents: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/query")
async def query_documents(request: QueryRequest):
    """Query indexed documents"""
    try:
        # Simple keyword search for now
        results = []
        query_lower = request.query.lower()
        
        for doc in document_store:
            if query_lower in doc["content"].lower():
                results.append(doc)
                if len(results) >= request.top_k:
                    break
        
        return {"query": request.query, "results": results, "count": len(results)}
    except Exception as e:
        logger.error(f"Error querying documents: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/upload")
async def upload_file(file: UploadFile = File(...)):
    """Upload and index a file"""
    try:
        content = await file.read()
        text = content.decode("utf-8")
        
        document_store.append({
            "content": text,
            "metadata": {"filename": file.filename}
        })
        
        return {"status": "success", "filename": file.filename, "size": len(text)}
    except Exception as e:
        logger.error(f"Error uploading file: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/stats")
async def get_stats():
    """Get statistics about the document store"""
    return {
        "total_documents": len(document_store),
        "total_characters": sum(len(doc["content"]) for doc in document_store)
    }

@app.delete("/clear")
async def clear_documents():
    """Clear all documents from the store"""
    document_store.clear()
    return {"status": "success", "message": "All documents cleared"}

if __name__ == "__main__":
    port = int(os.getenv("HAYSTACK_PORT", 8075))
    host = os.getenv("HAYSTACK_HOST", "0.0.0.0")
    workers = int(os.getenv("HAYSTACK_WORKERS", 2))
    
    logger.info(f"Starting Haystack server on {host}:{port}")
    
    uvicorn.run(
        "server:app",
        host=host,
        port=port,
        workers=1,  # Use 1 worker for shared memory store
        log_level=os.getenv("HAYSTACK_LOG_LEVEL", "info").lower()
    )
EOF
    
    chmod +x "${HAYSTACK_SCRIPTS_DIR}/server.py"
}

# Clean up resources
haystack::cleanup() {
    # Stop service if running
    if haystack::is_running; then
        haystack::stop
    fi
    
    # Remove PID file
    rm -f "${HAYSTACK_PID_FILE}"
    
    # Clear logs if requested
    if [[ "${1:-}" == "--clear-logs" ]]; then
        rm -rf "${HAYSTACK_LOG_DIR}"
    fi
}

# Display info
haystack::info() {
    local runtime_file="${HAYSTACK_CONFIG_DIR}/runtime.json"
    
    if [[ ! -f "${runtime_file}" ]]; then
        log::error "runtime.json not found"
        return 1
    fi
    
    # Parse and display runtime configuration
    if command -v jq &>/dev/null; then
        jq . "${runtime_file}"
    else
        cat "${runtime_file}"
    fi
}

# Execute content operations
haystack::content() {
    local action="${1:-list}"
    shift || true
    
    local port
    port=$(haystack::get_port)
    
    case "${action}" in
        list)
            log::info "Listing indexed documents..."
            curl -sf "http://localhost:${port}/stats" | jq . 2>/dev/null || echo "Service not running"
            ;;
        add)
            local file="${1:-}"
            if [[ -z "${file}" ]]; then
                log::error "File path required"
                return 1
            fi
            
            if [[ ! -f "${file}" ]]; then
                log::error "File not found: ${file}"
                return 1
            fi
            
            log::info "Indexing file: ${file}"
            
            # Check file extension
            if [[ "${file}" == *.json ]]; then
                # Index as JSON documents
                curl -X POST "http://localhost:${port}/index" \
                    -H "Content-Type: application/json" \
                    -d "@${file}"
            else
                # Upload as text file
                curl -X POST "http://localhost:${port}/upload" \
                    -F "file=@${file}"
            fi
            ;;
        query)
            local query="${1:-}"
            if [[ -z "${query}" ]]; then
                log::error "Query text required"
                return 1
            fi
            
            log::info "Querying: ${query}"
            curl -X POST "http://localhost:${port}/query" \
                -H "Content-Type: application/json" \
                -d "{\"query\":\"${query}\",\"top_k\":10}" | jq . 2>/dev/null
            ;;
        clear)
            log::info "Clearing all documents..."
            curl -X DELETE "http://localhost:${port}/clear"
            ;;
        *)
            log::error "Unknown content action: ${action}"
            echo "Usage: content [list|add <file>|query <text>|clear]"
            return 1
            ;;
    esac
}