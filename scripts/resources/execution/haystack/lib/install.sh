#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
HAYSTACK_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${HAYSTACK_LIB_DIR}/common.sh"
source "${HAYSTACK_LIB_DIR}/../../../../lib/utils/log.sh"

# Install Haystack
haystack::install() {
    log::info "Installing Haystack"
    
    # Create directories
    mkdir -p "${HAYSTACK_DATA_DIR}"
    mkdir -p "${HAYSTACK_LOG_DIR}"
    mkdir -p "${HAYSTACK_SCRIPTS_DIR}"
    
    # Check for Python 3.8+
    if ! command -v python3 &>/dev/null; then
        log::error "Python 3 is required but not installed"
        return 1
    fi
    
    local python_version
    python_version=$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
    local major minor
    IFS='.' read -r major minor <<< "${python_version}"
    if [[ ${major} -lt 3 ]] || { [[ ${major} -eq 3 ]] && [[ ${minor} -lt 8 ]]; }; then
        log::error "Python 3.8+ is required, found ${python_version}"
        return 1
    fi
    
    # Create virtual environment
    log::info "Creating Python virtual environment..."
    # Try to use venv first, fall back to virtualenv if needed
    if python3 -m venv "${HAYSTACK_VENV_DIR}" 2>/dev/null; then
        log::info "Created virtual environment using venv"
    elif command -v virtualenv &>/dev/null; then
        log::info "Using virtualenv as fallback"
        virtualenv -p python3 "${HAYSTACK_VENV_DIR}"
    else
        # Install virtualenv using pipx or with break-system-packages flag
        log::info "Installing virtualenv..."
        if command -v pipx &>/dev/null; then
            pipx install virtualenv
        else
            python3 -m pip install --user --break-system-packages virtualenv
        fi
        export PATH="${HOME}/.local/bin:${PATH}"
        virtualenv -p python3 "${HAYSTACK_VENV_DIR}"
    fi
    
    # Upgrade pip
    "${HAYSTACK_VENV_DIR}/bin/pip" install --upgrade pip setuptools wheel
    
    # Install Haystack with common components (using newer haystack-ai)
    log::info "Installing Haystack framework..."
    "${HAYSTACK_VENV_DIR}/bin/pip" install \
        "haystack-ai>=2.0.0" \
        "fastapi>=0.100.0" \
        "uvicorn[standard]>=0.23.0" \
        "python-multipart>=0.0.6" \
        "pydantic>=2.0.0" \
        "sentence-transformers>=2.2.0"
    
    # Install additional useful components
    log::info "Installing additional components..."
    "${HAYSTACK_VENV_DIR}/bin/pip" install \
        "transformers>=4.30.0" \
        "torch>=2.0.0"
    
    # Create the server script
    cat > "${HAYSTACK_SCRIPTS_DIR}/server.py" << 'EOF'
#!/usr/bin/env python3
import os
import json
import logging
from typing import Optional, List, Dict, Any
from fastapi import FastAPI, HTTPException, File, UploadFile
from fastapi.responses import JSONResponse
from pydantic import BaseModel
import uvicorn
from haystack import Pipeline, Document
from haystack.components.preprocessors import DocumentCleaner, DocumentSplitter
from haystack.components.embedders import SentenceTransformersDocumentEmbedder, SentenceTransformersTextEmbedder
from haystack.components.writers import DocumentWriter
from haystack.components.retrievers import InMemoryEmbeddingRetriever
from haystack.document_stores.in_memory import InMemoryDocumentStore
from haystack.components.builders import PromptBuilder

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Haystack API", version="1.0.0")

# Initialize document store
document_store = InMemoryDocumentStore()

# Model name for embeddings
EMBEDDING_MODEL = "sentence-transformers/all-MiniLM-L6-v2"

# Request models
class IndexRequest(BaseModel):
    documents: List[Dict[str, Any]]
    metadata: Optional[Dict[str, Any]] = None

class QueryRequest(BaseModel):
    query: str
    top_k: Optional[int] = 5
    filters: Optional[Dict[str, Any]] = None

class AnswerRequest(BaseModel):
    question: str
    context: Optional[str] = None
    top_k: Optional[int] = 3

@app.get("/health")
async def health():
    return {"status": "healthy", "service": "haystack"}

@app.post("/index")
async def index_documents(request: IndexRequest):
    """Index documents into the document store"""
    try:
        # Convert to Haystack documents
        docs = []
        for doc in request.documents:
            content = doc.get("content", "")
            metadata = doc.get("metadata", {})
            if request.metadata:
                metadata.update(request.metadata)
            docs.append(Document(content=content, meta=metadata))
        
        # Create indexing pipeline with fresh components
        indexing_pipeline = Pipeline()
        indexing_pipeline.add_component("cleaner", DocumentCleaner())
        indexing_pipeline.add_component("splitter", DocumentSplitter(split_length=100, split_overlap=20))
        indexing_pipeline.add_component("embedder", SentenceTransformersDocumentEmbedder(model=EMBEDDING_MODEL))
        indexing_pipeline.add_component("writer", DocumentWriter(document_store=document_store))
        
        indexing_pipeline.connect("cleaner", "splitter")
        indexing_pipeline.connect("splitter", "embedder")
        indexing_pipeline.connect("embedder", "writer")
        
        # Run the pipeline
        result = indexing_pipeline.run({"cleaner": {"documents": docs}})
        
        return {"status": "success", "documents_indexed": len(docs)}
    except Exception as e:
        logger.error(f"Indexing error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/query")
async def query_documents(request: QueryRequest):
    """Query documents using semantic search"""
    try:
        # Create retrieval pipeline with fresh components
        retrieval_pipeline = Pipeline()
        retrieval_pipeline.add_component("embedder", SentenceTransformersTextEmbedder(model=EMBEDDING_MODEL))
        retrieval_pipeline.add_component("retriever", InMemoryEmbeddingRetriever(document_store, top_k=request.top_k))
        
        retrieval_pipeline.connect("embedder.embedding", "retriever.query_embedding")
        
        # Run the pipeline
        result = retrieval_pipeline.run({"embedder": {"text": request.query}})
        
        # Format results
        documents = result.get("retriever", {}).get("documents", [])
        formatted_docs = [
            {
                "content": doc.content,
                "score": doc.score,
                "metadata": doc.meta
            }
            for doc in documents
        ]
        
        return {
            "status": "success",
            "query": request.query,
            "results": formatted_docs
        }
    except Exception as e:
        logger.error(f"Query error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/upload")
async def upload_file(file: UploadFile = File(...)):
    """Upload and index a file"""
    try:
        content = await file.read()
        text = content.decode('utf-8')
        
        # Create document
        doc = Document(
            content=text,
            meta={"filename": file.filename, "content_type": file.content_type}
        )
        
        # Index the document
        docs = [doc]
        
        # Create and run indexing pipeline with fresh components
        indexing_pipeline = Pipeline()
        indexing_pipeline.add_component("cleaner", DocumentCleaner())
        indexing_pipeline.add_component("splitter", DocumentSplitter(split_length=100, split_overlap=20))
        indexing_pipeline.add_component("embedder", SentenceTransformersDocumentEmbedder(model=EMBEDDING_MODEL))
        indexing_pipeline.add_component("writer", DocumentWriter(document_store=document_store))
        
        indexing_pipeline.connect("cleaner", "splitter")
        indexing_pipeline.connect("splitter", "embedder")
        indexing_pipeline.connect("embedder", "writer")
        
        result = indexing_pipeline.run({"cleaner": {"documents": docs}})
        
        return {
            "status": "success",
            "filename": file.filename,
            "message": "File uploaded and indexed successfully"
        }
    except Exception as e:
        logger.error(f"Upload error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/stats")
async def get_stats():
    """Get document store statistics"""
    try:
        count = document_store.count_documents()
        return {
            "status": "success",
            "document_count": count,
            "store_type": "InMemory"
        }
    except Exception as e:
        logger.error(f"Stats error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.delete("/clear")
async def clear_store():
    """Clear all documents from the store"""
    try:
        # Reinitialize the document store
        global document_store
        document_store = InMemoryDocumentStore()
        return {"status": "success", "message": "Document store cleared"}
    except Exception as e:
        logger.error(f"Clear error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    port = int(os.environ.get("HAYSTACK_PORT", 8075))
    uvicorn.run(app, host="0.0.0.0", port=port)
EOF
    
    chmod +x "${HAYSTACK_SCRIPTS_DIR}/server.py"
    
    # Register the CLI
    local resource_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    "${HAYSTACK_LIB_DIR}/../../../../lib/resources/install-resource-cli.sh" "${resource_dir}" 2>/dev/null || true
    
    log::success "Haystack installed successfully"
    return 0
}

# Uninstall Haystack
haystack::uninstall() {
    log::info "Uninstalling Haystack"
    
    # Stop if running
    if haystack::is_running; then
        haystack::stop
    fi
    
    # Remove installation
    rm -rf "${HAYSTACK_DATA_DIR}"
    
    log::success "Haystack uninstalled successfully"
    return 0
}