#!/usr/bin/env python3
"""
QwenCoder API Server
Provides OpenAI-compatible API for code generation using QwenCoder models
"""

import os
import sys
import json
import time
import argparse
import logging
from typing import List, Dict, Any, Optional, Union
from datetime import datetime
from pathlib import Path

import uvicorn
from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

# Add parent directory to path for imports
sys.path.append(str(Path(__file__).parent.parent))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="QwenCoder API",
    description="Code generation API using QwenCoder models",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global model storage
MODEL_INSTANCE = None
TOKENIZER_INSTANCE = None
MODEL_NAME = "qwencoder-1.5b"
DEVICE = "cpu"

# Pydantic models for API
class CompletionRequest(BaseModel):
    model: str = Field(default="qwencoder-1.5b")
    prompt: str
    max_tokens: int = Field(default=128, ge=1, le=4096)
    temperature: float = Field(default=0.7, ge=0.0, le=2.0)
    top_p: float = Field(default=0.9, ge=0.0, le=1.0)
    n: int = Field(default=1, ge=1, le=5)
    stream: bool = False
    language: Optional[str] = None
    stop: Optional[Union[str, List[str]]] = None

class ChatMessage(BaseModel):
    role: str
    content: str

class ChatCompletionRequest(BaseModel):
    model: str = Field(default="qwencoder-1.5b")
    messages: List[ChatMessage]
    max_tokens: int = Field(default=128, ge=1, le=4096)
    temperature: float = Field(default=0.7, ge=0.0, le=2.0)
    top_p: float = Field(default=0.9, ge=0.0, le=1.0)
    n: int = Field(default=1, ge=1, le=5)
    stream: bool = False
    functions: Optional[List[Dict[str, Any]]] = None
    function_call: Optional[Union[str, Dict[str, str]]] = None

class CompletionChoice(BaseModel):
    text: str
    index: int
    finish_reason: str

class CompletionResponse(BaseModel):
    id: str
    object: str = "text_completion"
    created: int
    model: str
    choices: List[CompletionChoice]
    usage: Dict[str, int]

class ChatChoice(BaseModel):
    index: int
    message: ChatMessage
    finish_reason: str

class ChatCompletionResponse(BaseModel):
    id: str
    object: str = "chat.completion"
    created: int
    model: str
    choices: List[ChatChoice]
    usage: Dict[str, int]

def load_model(model_name: str, device: str):
    """Load QwenCoder model and tokenizer"""
    global MODEL_INSTANCE, TOKENIZER_INSTANCE
    
    try:
        # For now, mock the model loading
        # In production, this would load actual QwenCoder models
        logger.info(f"Loading model {model_name} on device {device}")
        
        # Mock successful load
        MODEL_INSTANCE = {"name": model_name, "device": device, "loaded": True}
        TOKENIZER_INSTANCE = {"loaded": True}
        
        logger.info(f"Model {model_name} loaded successfully")
        return True
    except Exception as e:
        logger.error(f"Failed to load model: {e}")
        return False

def generate_completion(prompt: str, max_tokens: int = 128, temperature: float = 0.7) -> str:
    """Generate code completion"""
    # Mock implementation for now
    # In production, this would use actual model inference
    
    # Simple mock responses based on prompt patterns
    if "def " in prompt or "function" in prompt:
        return "\n    return result"
    elif "print(" in prompt:
        return '"Hello, World!")'
    elif "class " in prompt:
        return "\n    def __init__(self):\n        pass"
    else:
        return "# Generated code completion"

@app.on_event("startup")
async def startup_event():
    """Initialize model on startup"""
    logger.info("Starting QwenCoder API server...")
    if not load_model(MODEL_NAME, DEVICE):
        logger.error("Failed to load model, server will run in mock mode")

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "name": "QwenCoder API",
        "version": "1.0.0",
        "model": MODEL_NAME,
        "status": "running"
    }

@app.get("/health")
async def health():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "model": MODEL_NAME,
        "model_loaded": MODEL_INSTANCE is not None
    }

@app.get("/models")
async def list_models():
    """List available models"""
    return {
        "object": "list",
        "data": [
            {
                "id": "qwencoder-0.5b",
                "object": "model",
                "owned_by": "alibaba",
                "permission": []
            },
            {
                "id": "qwencoder-1.5b",
                "object": "model",
                "owned_by": "alibaba",
                "permission": []
            },
            {
                "id": "qwencoder-7b",
                "object": "model",
                "owned_by": "alibaba",
                "permission": []
            },
            {
                "id": "qwencoder-32b",
                "object": "model",
                "owned_by": "alibaba",
                "permission": []
            }
        ]
    }

@app.post("/v1/completions")
async def create_completion(request: CompletionRequest):
    """Create completion endpoint (OpenAI compatible)"""
    try:
        # Generate completions
        choices = []
        total_tokens = 0
        
        for i in range(request.n):
            completion_text = generate_completion(
                request.prompt,
                request.max_tokens,
                request.temperature
            )
            
            choices.append(CompletionChoice(
                text=completion_text,
                index=i,
                finish_reason="stop"
            ))
            
            # Mock token counting
            prompt_tokens = len(request.prompt.split())
            completion_tokens = len(completion_text.split())
            total_tokens = prompt_tokens + completion_tokens
        
        return CompletionResponse(
            id=f"cmpl-{int(time.time())}",
            created=int(time.time()),
            model=request.model,
            choices=choices,
            usage={
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": total_tokens
            }
        )
    except Exception as e:
        logger.error(f"Completion error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/v1/chat/completions")
async def create_chat_completion(request: ChatCompletionRequest):
    """Create chat completion endpoint (OpenAI compatible)"""
    try:
        # Convert chat messages to prompt
        prompt = ""
        for msg in request.messages:
            if msg.role == "system":
                prompt += f"System: {msg.content}\n"
            elif msg.role == "user":
                prompt += f"User: {msg.content}\n"
            elif msg.role == "assistant":
                prompt += f"Assistant: {msg.content}\n"
        
        prompt += "Assistant: "
        
        # Generate response
        response_text = generate_completion(
            prompt,
            request.max_tokens,
            request.temperature
        )
        
        choices = []
        for i in range(request.n):
            choices.append(ChatChoice(
                index=i,
                message=ChatMessage(role="assistant", content=response_text),
                finish_reason="stop"
            ))
        
        # Mock token counting
        prompt_tokens = len(prompt.split())
        completion_tokens = len(response_text.split())
        
        return ChatCompletionResponse(
            id=f"chatcmpl-{int(time.time())}",
            created=int(time.time()),
            model=request.model,
            choices=choices,
            usage={
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens
            }
        )
    except Exception as e:
        logger.error(f"Chat completion error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/v1/code/complete")
async def code_complete(request: dict):
    """Fill-in-the-middle code completion"""
    try:
        prefix = request.get("prefix", "")
        suffix = request.get("suffix", "")
        max_tokens = request.get("max_tokens", 128)
        
        # Mock FIM completion
        completion = generate_completion(prefix, max_tokens)
        
        return {
            "id": f"fim-{int(time.time())}",
            "completion": completion,
            "model": MODEL_NAME
        }
    except Exception as e:
        logger.error(f"FIM completion error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description="QwenCoder API Server")
    parser.add_argument("--port", type=int, default=26501, help="Port to run server on")
    parser.add_argument("--host", type=str, default="0.0.0.0", help="Host to bind to")
    parser.add_argument("--model", type=str, default="qwencoder-1.5b", help="Model to load")
    parser.add_argument("--device", type=str, default="cpu", help="Device to use (cpu/cuda)")
    
    args = parser.parse_args()
    
    # Update globals
    global MODEL_NAME, DEVICE
    MODEL_NAME = args.model
    DEVICE = args.device
    
    # Run server
    uvicorn.run(
        app,
        host=args.host,
        port=args.port,
        log_level="info"
    )

if __name__ == "__main__":
    main()