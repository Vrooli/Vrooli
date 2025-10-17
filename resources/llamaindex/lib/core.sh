#!/bin/bash

# LlamaIndex Core Functions
# Provides RAG capabilities, document processing, and knowledge base management

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LLAMAINDEX_LIB_DIR="${APP_ROOT}/resources/llamaindex/lib"
LLAMAINDEX_DIR="${APP_ROOT}/resources/llamaindex"
LLAMAINDEX_ROOT_DIR="${APP_ROOT}/resources"
LLAMAINDEX_SCRIPTS_DIR="${APP_ROOT}"

# Source dependencies
source "$LLAMAINDEX_SCRIPTS_DIR/scripts/lib/utils/var.sh" || exit 1
source "$LLAMAINDEX_SCRIPTS_DIR/scripts/lib/utils/log.sh" || exit 1
source "$LLAMAINDEX_SCRIPTS_DIR/scripts/lib/utils/format.sh" || exit 1
source "$LLAMAINDEX_SCRIPTS_DIR/scripts/resources/lib/resource-functions.sh" || exit 1
source "$LLAMAINDEX_SCRIPTS_DIR/scripts/resources/port_registry.sh" || exit 1

# Configuration
LLAMAINDEX_PORT="$(ports::get_resource_port "llamaindex" 8091)"
LLAMAINDEX_VERSION="0.11.14"
LLAMAINDEX_PYTHON_VERSION="3.11"
LLAMAINDEX_VENV_DIR="${HOME}/.vrooli/llamaindex/venv"
LLAMAINDEX_DATA_DIR="${HOME}/.vrooli/llamaindex/data"
LLAMAINDEX_CONFIG_DIR="${HOME}/.vrooli/llamaindex/config"
LLAMAINDEX_SERVICE_FILE="/etc/systemd/system/llamaindex.service"

# Initialize LlamaIndex
llamaindex::init() {
    log::debug "Initializing LlamaIndex"
    
    # Create directories
    mkdir -p "$LLAMAINDEX_DATA_DIR"
    mkdir -p "$LLAMAINDEX_CONFIG_DIR"
    
    # Check for OpenAI/OpenRouter API keys for embeddings
    if [[ -f "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" ]]; then
        local api_key=$(jq -r '.data.apiKey // empty' "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" 2>/dev/null)
        if [[ -n "$api_key" && "$api_key" != "placeholder" ]]; then
            export OPENROUTER_API_KEY="$api_key"
            log::debug "Loaded OpenRouter API key for LlamaIndex"
        fi
    fi
    
    # Check if Ollama is available for local embeddings
    if command -v ollama &> /dev/null; then
        export LLAMAINDEX_USE_OLLAMA="true"
        export OLLAMA_HOST="http://localhost:$(ports::get_resource_port "ollama" 11434)"
        log::debug "Ollama available for local embeddings"
    fi
    
    return 0
}

# Check if installed
llamaindex::is_installed() {
    # Check if Docker image exists with proper formatting
    docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^llamaindex:latest$"
}

# Check if running
llamaindex::is_running() {
    # Check if Docker container is running
    if docker ps --format "{{.Names}}" | grep -q "^llamaindex$"; then
        # Also check if API is responding
        if timeout 2 curl -s "http://localhost:$LLAMAINDEX_PORT/health" &>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Get version
llamaindex::get_version() {
    if llamaindex::is_installed; then
        echo "$LLAMAINDEX_VERSION"
    else
        echo "not_installed"
    fi
}

# Get status
llamaindex::get_status() {
    local format="${1:-plain}"
    local installed=$(llamaindex::is_installed && echo "true" || echo "false")
    local running=$(llamaindex::is_running && echo "true" || echo "false")
    local version=$(llamaindex::get_version)
    local health="unhealthy"
    
    if [[ "$running" == "true" ]]; then
        if timeout 2 curl -s "http://localhost:$LLAMAINDEX_PORT/health" | grep -q "ok" 2>/dev/null; then
            health="healthy"
        fi
    fi
    
    # Count indexed documents
    local doc_count=0
    if [[ -d "$LLAMAINDEX_DATA_DIR/indices" ]]; then
        doc_count=$(find "$LLAMAINDEX_DATA_DIR/indices" -type f -name "*.json" 2>/dev/null | wc -l)
    fi
    
    # Use format::output consistently for all formats
    format::output "$format" "kv" \
        "installed" "$installed" \
        "running" "$running" \
        "version" "$version" \
        "health" "$health" \
        "port" "$LLAMAINDEX_PORT" \
        "documents" "$doc_count"
}

# Create API server script
llamaindex::create_server_script() {
    cat > "$LLAMAINDEX_CONFIG_DIR/server.py" <<'EOF'
#!/usr/bin/env python3
"""LlamaIndex API Server for Vrooli"""

import os
import json
from pathlib import Path
from typing import List, Dict, Any, Optional
from fastapi import FastAPI, HTTPException, File, UploadFile
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import uvicorn

# LlamaIndex imports
from llama_index.core import (
    Document,
    VectorStoreIndex,
    SimpleDirectoryReader,
    StorageContext,
    load_index_from_storage,
    Settings
)
from llama_index.core.node_parser import SentenceSplitter
from llama_index.core.response.pprint_utils import pprint_response

# Configure embeddings
if os.getenv("LLAMAINDEX_USE_OLLAMA") == "true":
    from llama_index.embeddings.ollama import OllamaEmbedding
    from llama_index.llms.ollama import Ollama
    
    Settings.embed_model = OllamaEmbedding(
        model_name="nomic-embed-text:latest",
        base_url=os.getenv("OLLAMA_HOST", "http://localhost:11434")
    )
    Settings.llm = Ollama(
        model="llama3.2:3b",
        base_url=os.getenv("OLLAMA_HOST", "http://localhost:11434")
    )
elif os.getenv("OPENROUTER_API_KEY"):
    from llama_index.embeddings.openai import OpenAIEmbedding
    from llama_index.llms.openai import OpenAI
    
    Settings.embed_model = OpenAIEmbedding(
        api_key=os.getenv("OPENROUTER_API_KEY"),
        api_base="https://openrouter.ai/api/v1"
    )
    Settings.llm = OpenAI(
        api_key=os.getenv("OPENROUTER_API_KEY"),
        api_base="https://openrouter.ai/api/v1",
        model="meta-llama/llama-3.2-3b-instruct:free"
    )

app = FastAPI(title="LlamaIndex Server", version="1.0.0")

# Storage paths
DATA_DIR = Path(os.getenv("LLAMAINDEX_DATA_DIR", "/var/lib/vrooli/llamaindex/data"))
INDEX_DIR = DATA_DIR / "indices"
DOCS_DIR = DATA_DIR / "documents"

# Ensure directories exist
INDEX_DIR.mkdir(parents=True, exist_ok=True)
DOCS_DIR.mkdir(parents=True, exist_ok=True)

# Index storage
indices: Dict[str, VectorStoreIndex] = {}

class QueryRequest(BaseModel):
    query: str
    index_name: str = "default"
    top_k: int = 5

class IndexRequest(BaseModel):
    name: str
    documents: List[Dict[str, Any]]

class DocumentRequest(BaseModel):
    content: str
    metadata: Optional[Dict[str, Any]] = None

@app.get("/health")
async def health():
    return {"status": "ok", "indices": list(indices.keys())}

@app.post("/index/create")
async def create_index(request: IndexRequest):
    """Create a new index with documents"""
    try:
        docs = []
        for doc_data in request.documents:
            doc = Document(
                text=doc_data.get("text", ""),
                metadata=doc_data.get("metadata", {})
            )
            docs.append(doc)
        
        index = VectorStoreIndex.from_documents(docs)
        
        # Persist index
        index_path = INDEX_DIR / request.name
        index.storage_context.persist(persist_dir=str(index_path))
        
        indices[request.name] = index
        
        return {"status": "success", "index": request.name, "documents": len(docs)}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/index/load")
async def load_index(name: str):
    """Load an existing index from storage"""
    try:
        index_path = INDEX_DIR / name
        if not index_path.exists():
            raise HTTPException(status_code=404, detail=f"Index {name} not found")
        
        storage_context = StorageContext.from_defaults(persist_dir=str(index_path))
        index = load_index_from_storage(storage_context)
        indices[name] = index
        
        return {"status": "success", "index": name}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/query")
async def query_index(request: QueryRequest):
    """Query an index"""
    try:
        if request.index_name not in indices:
            # Try to load it
            index_path = INDEX_DIR / request.index_name
            if index_path.exists():
                storage_context = StorageContext.from_defaults(persist_dir=str(index_path))
                indices[request.index_name] = load_index_from_storage(storage_context)
            else:
                raise HTTPException(status_code=404, detail=f"Index {request.index_name} not found")
        
        index = indices[request.index_name]
        query_engine = index.as_query_engine(similarity_top_k=request.top_k)
        response = query_engine.query(request.query)
        
        return {
            "response": str(response),
            "sources": [
                {
                    "text": node.node.text[:200],
                    "score": node.score,
                    "metadata": node.node.metadata
                }
                for node in response.source_nodes
            ]
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/documents/upload")
async def upload_documents(files: List[UploadFile] = File(...)):
    """Upload documents to be indexed"""
    try:
        saved_files = []
        for file in files:
            file_path = DOCS_DIR / file.filename
            content = await file.read()
            file_path.write_bytes(content)
            saved_files.append(str(file_path))
        
        return {"status": "success", "files": saved_files}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/documents/index")
async def index_directory(directory: str = None, index_name: str = "default"):
    """Index all documents in a directory"""
    try:
        if directory:
            docs_path = Path(directory)
        else:
            docs_path = DOCS_DIR
        
        if not docs_path.exists():
            raise HTTPException(status_code=404, detail=f"Directory {docs_path} not found")
        
        reader = SimpleDirectoryReader(str(docs_path))
        documents = reader.load_data()
        
        index = VectorStoreIndex.from_documents(documents)
        
        # Persist index
        index_path = INDEX_DIR / index_name
        index.storage_context.persist(persist_dir=str(index_path))
        
        indices[index_name] = index
        
        return {"status": "success", "index": index_name, "documents": len(documents)}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/indices")
async def list_indices():
    """List all available indices"""
    stored_indices = [d.name for d in INDEX_DIR.iterdir() if d.is_dir()]
    loaded_indices = list(indices.keys())
    
    return {
        "loaded": loaded_indices,
        "stored": stored_indices,
        "total": len(set(stored_indices + loaded_indices))
    }

if __name__ == "__main__":
    # Load existing indices on startup
    for index_dir in INDEX_DIR.iterdir():
        if index_dir.is_dir():
            try:
                storage_context = StorageContext.from_defaults(persist_dir=str(index_dir))
                indices[index_dir.name] = load_index_from_storage(storage_context)
                print(f"Loaded index: {index_dir.name}")
            except Exception as e:
                print(f"Failed to load index {index_dir.name}: {e}")
    
    uvicorn.run(app, host="0.0.0.0", port=int(os.getenv("LLAMAINDEX_PORT", "8090")))
EOF
    
    chmod +x "$LLAMAINDEX_CONFIG_DIR/server.py"
}

# Install LlamaIndex
llamaindex::install() {
    log::info "Installing LlamaIndex..."
    
    # Initialize first
    llamaindex::init
    
    # Create server script
    llamaindex::create_server_script
    
    # Build Docker image
    log::info "Building Docker image..."
    if docker build -t llamaindex:latest "$LLAMAINDEX_DIR"; then
        if llamaindex::is_installed; then
            log::success "LlamaIndex installed successfully"
            
            # Register CLI with Vrooli
            # shellcheck disable=SC1091
            "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${APP_ROOT}/resources/llamaindex" 2>/dev/null || true
            
            return 0
        else
            log::error "Docker image built but not found in images list"
            return 1
        fi
    else
        log::error "Failed to build Docker image"
        return 1
    fi
}

# Start LlamaIndex
llamaindex::start() {
    if llamaindex::is_running; then
        log::info "LlamaIndex is already running"
        return 0
    fi
    
    if ! llamaindex::is_installed; then
        log::error "LlamaIndex is not installed. Run 'install' first"
        return 1
    fi
    
    log::info "Starting LlamaIndex..."
    
    # Start Docker container
    docker run -d \
        --name llamaindex \
        --restart unless-stopped \
        --network host \
        -v "$LLAMAINDEX_CONFIG_DIR:/app/config:ro" \
        -v "$LLAMAINDEX_DATA_DIR:/app/data" \
        -e "LLAMAINDEX_PORT=$LLAMAINDEX_PORT" \
        -e "LLAMAINDEX_DATA_DIR=/app/data" \
        -e "LLAMAINDEX_USE_OLLAMA=true" \
        -e "OLLAMA_HOST=http://localhost:$(ports::get_resource_port "ollama" 11434)" \
        -e "OPENROUTER_API_KEY=${OPENROUTER_API_KEY:-}" \
        llamaindex:latest &>/dev/null
    
    sleep 5
    if llamaindex::is_running; then
        log::success "LlamaIndex started successfully"
        return 0
    else
        log::error "Failed to start LlamaIndex"
        docker logs llamaindex 2>&1 | tail -10
        return 1
    fi
}

# Stop LlamaIndex
llamaindex::stop() {
    if ! llamaindex::is_running; then
        log::info "LlamaIndex is not running"
        return 0
    fi
    
    log::info "Stopping LlamaIndex..."
    docker stop llamaindex &>/dev/null
    docker rm llamaindex &>/dev/null
    
    sleep 2
    if ! llamaindex::is_running; then
        log::success "LlamaIndex stopped"
        return 0
    else
        log::error "Failed to stop LlamaIndex"
        return 1
    fi
}

# Restart LlamaIndex
llamaindex::restart() {
    llamaindex::stop
    llamaindex::start
}

# Uninstall LlamaIndex
llamaindex::uninstall() {
    llamaindex::stop
    
    log::info "Uninstalling LlamaIndex..."
    
    # Remove Docker image
    docker rmi llamaindex:latest &>/dev/null
    
    # Remove data directory (optionally)
    # [[ -d "$LLAMAINDEX_DATA_DIR" ]] && rm -rf "$LLAMAINDEX_DATA_DIR"
    
    log::success "LlamaIndex uninstalled"
    return 0
}

# Index documents
llamaindex::index_documents() {
    local path="${1:-$LLAMAINDEX_DATA_DIR/docs}"
    local index_name="${2:-default}"
    
    if ! llamaindex::is_running; then
        log::error "LlamaIndex is not running"
        return 1
    fi
    
    log::info "Indexing documents from $path..."
    curl -X POST "http://localhost:$LLAMAINDEX_PORT/documents/index" \
        -H "Content-Type: application/json" \
        -d "{\"directory\": \"$path\", \"index_name\": \"$index_name\"}" \
        2>/dev/null | jq '.' 2>/dev/null || echo '{"error": "Failed to index documents"}'
}

# Query documents
llamaindex::query() {
    local query="${1:-}"
    local index_name="${2:-default}"
    
    if [[ -z "$query" ]]; then
        log::error "Query text required"
        return 1
    fi
    
    if ! llamaindex::is_running; then
        log::error "LlamaIndex is not running"
        return 1
    fi
    
    curl -X POST "http://localhost:$LLAMAINDEX_PORT/query" \
        -H "Content-Type: application/json" \
        -d "{\"query\": \"$query\", \"index_name\": \"$index_name\"}" \
        2>/dev/null | jq '.response' -r 2>/dev/null || echo "Failed to query"
}

# List indices
llamaindex::list_indices() {
    if ! llamaindex::is_running; then
        log::error "LlamaIndex is not running"
        return 1
    fi
    
    curl -s "http://localhost:$LLAMAINDEX_PORT/indices" | jq '.' 2>/dev/null || echo '{"error": "Failed to list indices"}'
}

# Show logs
llamaindex::logs() {
    local lines="${1:-50}"
    if docker ps --format "{{.Names}}" | grep -q "^llamaindex$"; then
        log::info "Showing last $lines lines of LlamaIndex logs:"
        docker logs --tail "$lines" llamaindex 2>&1
    else
        log::warning "LlamaIndex container is not running"
        return 1
    fi
}

# Export functions
export -f llamaindex::init
export -f llamaindex::is_installed
export -f llamaindex::is_running
export -f llamaindex::get_version
export -f llamaindex::get_status
export -f llamaindex::install
export -f llamaindex::start
export -f llamaindex::stop
export -f llamaindex::restart
export -f llamaindex::uninstall
export -f llamaindex::index_documents
export -f llamaindex::query
export -f llamaindex::list_indices
export -f llamaindex::create_server_script
export -f llamaindex::logs