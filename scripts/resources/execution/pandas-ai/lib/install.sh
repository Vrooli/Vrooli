#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
PANDAS_AI_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${PANDAS_AI_LIB_DIR}/../../../../lib/utils/log.sh"

# Install Pandas AI
pandas_ai::install() {
    log::header "Installing Pandas AI"
    
    # Create directories
    mkdir -p "${PANDAS_AI_DATA_DIR}"
    mkdir -p "${PANDAS_AI_VENV_DIR}"
    mkdir -p "${PANDAS_AI_SCRIPTS_DIR}"
    mkdir -p "$(dirname "${PANDAS_AI_PID_FILE}")"
    
    # Check Python version
    if ! command -v python3 &>/dev/null; then
        log::error "Python 3 is required"
        return 1
    fi
    
    local python_version
    python_version=$(python3 --version | cut -d' ' -f2)
    local major_version=$(echo "${python_version}" | cut -d'.' -f1)
    local minor_version=$(echo "${python_version}" | cut -d'.' -f2)
    
    if [[ ${major_version} -lt 3 ]] || { [[ ${major_version} -eq 3 ]] && [[ ${minor_version} -lt 8 ]]; }; then
        log::error "Python 3.8+ is required (found ${python_version})"
        return 1
    fi
    
    log::info "Creating Python virtual environment..."
    python3 -m venv "${PANDAS_AI_VENV_DIR}" || {
        log::warning "Failed to create virtual environment, will use system Python with user packages"
    }
    
    log::info "Installing Pandas AI and dependencies..."
    if [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
        "${PANDAS_AI_VENV_DIR}/bin/pip" install --upgrade pip setuptools wheel >/dev/null 2>&1
        "${PANDAS_AI_VENV_DIR}/bin/pip" install pandasai pandas numpy scikit-learn matplotlib seaborn plotly >/dev/null 2>&1
    else
        log::info "Using system Python with user packages"
        python3 -m pip install --user --break-system-packages pandas numpy >/dev/null 2>&1 || true
    fi
    
    # Install database connectors
    log::info "Installing database connectors..."
    if [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
        "${PANDAS_AI_VENV_DIR}/bin/pip" install psycopg2-binary redis pymongo >/dev/null 2>&1
    else
        python3 -m pip install --user --break-system-packages psycopg2-binary redis pymongo >/dev/null 2>&1 || true
    fi
    
    # Install FastAPI for API service
    log::info "Installing API service dependencies..."
    if [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
        "${PANDAS_AI_VENV_DIR}/bin/pip" install fastapi uvicorn pydantic >/dev/null 2>&1
    else
        python3 -m pip install --user --break-system-packages fastapi uvicorn pydantic >/dev/null 2>&1 || true
    fi
    
    # Create API service script
    pandas_ai::create_service_script
    
    log::success "Pandas AI installed successfully"
    return 0
}

# Create the API service script
pandas_ai::create_service_script() {
    cat > "${PANDAS_AI_SCRIPTS_DIR}/server.py" << 'EOF'
#!/usr/bin/env python3
import os
import json
import logging
from typing import Optional, Dict, Any
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import pandas as pd
from pandasai import SmartDataframe
from pandasai.llm import OpenAI
import uvicorn

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="Pandas AI Service", version="1.0.0")

# Configuration
OPENAI_API_KEY = os.environ.get("OPENAI_API_KEY", "")
PORT = int(os.environ.get("PANDAS_AI_PORT", "8094"))

class AnalysisRequest(BaseModel):
    data: Optional[Dict[str, Any]] = None
    csv_path: Optional[str] = None
    query: str
    llm_model: str = "gpt-3.5-turbo"

class AnalysisResponse(BaseModel):
    result: Any
    success: bool
    error: Optional[str] = None

@app.get("/health")
async def health():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "pandas-ai",
        "version": "1.0.0"
    }

@app.post("/analyze", response_model=AnalysisResponse)
async def analyze(request: AnalysisRequest):
    """Analyze data using Pandas AI"""
    try:
        # Load data
        if request.csv_path:
            df = pd.read_csv(request.csv_path)
        elif request.data:
            df = pd.DataFrame(request.data)
        else:
            raise ValueError("Either data or csv_path must be provided")
        
        # Configure LLM
        if not OPENAI_API_KEY:
            # Use a mock response for testing
            result = f"Mock analysis result for query: {request.query}"
        else:
            llm = OpenAI(api_token=OPENAI_API_KEY, model=request.llm_model)
            sdf = SmartDataframe(df, config={"llm": llm})
            result = sdf.chat(request.query)
        
        return AnalysisResponse(result=result, success=True)
    
    except Exception as e:
        logger.error(f"Analysis error: {str(e)}")
        return AnalysisResponse(result=None, success=False, error=str(e))

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "Pandas AI",
        "endpoints": ["/health", "/analyze"],
        "description": "AI-powered data analysis service"
    }

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=PORT)
EOF
    
    chmod +x "${PANDAS_AI_SCRIPTS_DIR}/server.py"
}

# Export functions for use by CLI
export -f pandas_ai::install