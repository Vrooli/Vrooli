#!/usr/bin/env python3
"""
Splink API Service - Probabilistic Record Linkage at Scale
"""

import os
import json
import asyncio
import logging
from datetime import datetime
from typing import Dict, List, Optional, Any
from uuid import uuid4
import time

from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
import uvicorn

# Import Splink engine
from api.splink_engine import get_splink_engine

# Configure logging
logging.basicConfig(
    level=getattr(logging, os.getenv("LOG_LEVEL", "INFO")),
    format='{"time":"%(asctime)s","level":"%(levelname)s","message":"%(message)s"}' 
    if os.getenv("LOG_FORMAT") == "json" else '%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Splink Record Linkage API",
    description="Probabilistic record linkage and deduplication at scale",
    version="1.0.0"
)

# CORS configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=os.getenv("API_CORS_ORIGINS", "*").split(","),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configuration
SPLINK_PORT = int(os.getenv("SPLINK_PORT", "8096"))
SPLINK_BACKEND = os.getenv("SPLINK_BACKEND", "duckdb")
DATA_DIR = os.getenv("DATA_DIR", "/data")

# In-memory job storage (would use Redis in production)
jobs: Dict[str, Dict] = {}

# Request/Response Models
class LinkageSettings(BaseModel):
    """Settings for linkage operations"""
    threshold: float = Field(default=0.9, ge=0, le=1)
    link_type: str = Field(default="one_to_one", pattern="^(one_to_one|one_to_many|many_to_many)$")
    blocking_rules: List[str] = Field(default_factory=list)
    comparison_columns: List[str] = Field(default_factory=list)

class DeduplicationRequest(BaseModel):
    """Request for deduplication operation"""
    dataset_id: str
    settings: LinkageSettings = Field(default_factory=LinkageSettings)

class LinkRequest(BaseModel):
    """Request for linking two datasets"""
    dataset1_id: str
    dataset2_id: str
    settings: LinkageSettings = Field(default_factory=LinkageSettings)

class EstimationRequest(BaseModel):
    """Request for parameter estimation"""
    dataset_id: str
    settings: Optional[Dict[str, Any]] = Field(default_factory=dict)

class JobResponse(BaseModel):
    """Response for job submission"""
    job_id: str
    status: str
    message: str
    created_at: str

class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    backend: str
    version: str
    timestamp: str

# Health check endpoint
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Check service health"""
    return HealthResponse(
        status="healthy",
        backend=SPLINK_BACKEND,
        version="3.9.14",
        timestamp=datetime.utcnow().isoformat()
    )

# Root endpoint
@app.get("/")
async def root():
    """API root endpoint"""
    return {
        "service": "Splink Record Linkage API",
        "version": "1.0.0",
        "backend": SPLINK_BACKEND,
        "documentation": "/docs"
    }

# Deduplication endpoint
@app.post("/linkage/deduplicate", response_model=JobResponse)
async def deduplicate(request: DeduplicationRequest, background_tasks: BackgroundTasks):
    """
    Start a deduplication job for a dataset
    """
    job_id = str(uuid4())
    
    # Create job record
    job = {
        "id": job_id,
        "type": "deduplication",
        "dataset_id": request.dataset_id,
        "settings": request.settings.dict(),
        "status": "pending",
        "created_at": datetime.utcnow().isoformat(),
        "updated_at": datetime.utcnow().isoformat(),
        "progress": 0,
        "result": None,
        "error": None
    }
    
    jobs[job_id] = job
    
    # Queue background processing
    background_tasks.add_task(process_deduplication, job_id, request)
    
    logger.info(f"Created deduplication job: {job_id}")
    
    return JobResponse(
        job_id=job_id,
        status="pending",
        message=f"Deduplication job created for dataset: {request.dataset_id}",
        created_at=job["created_at"]
    )

# Link datasets endpoint
@app.post("/linkage/link", response_model=JobResponse)
async def link_datasets(request: LinkRequest, background_tasks: BackgroundTasks):
    """
    Start a linkage job between two datasets
    """
    job_id = str(uuid4())
    
    # Create job record
    job = {
        "id": job_id,
        "type": "linkage",
        "dataset1_id": request.dataset1_id,
        "dataset2_id": request.dataset2_id,
        "settings": request.settings.dict(),
        "status": "pending",
        "created_at": datetime.utcnow().isoformat(),
        "updated_at": datetime.utcnow().isoformat(),
        "progress": 0,
        "result": None,
        "error": None
    }
    
    jobs[job_id] = job
    
    # Queue background processing
    background_tasks.add_task(process_linkage, job_id, request)
    
    logger.info(f"Created linkage job: {job_id}")
    
    return JobResponse(
        job_id=job_id,
        status="pending",
        message=f"Linkage job created for datasets: {request.dataset1_id} and {request.dataset2_id}",
        created_at=job["created_at"]
    )

# Parameter estimation endpoint
@app.post("/linkage/estimate", response_model=JobResponse)
async def estimate_parameters(request: EstimationRequest, background_tasks: BackgroundTasks):
    """
    Estimate linkage parameters using EM algorithm
    """
    job_id = str(uuid4())
    
    # Create job record
    job = {
        "id": job_id,
        "type": "estimation",
        "dataset_id": request.dataset_id,
        "settings": request.settings,
        "status": "pending",
        "created_at": datetime.utcnow().isoformat(),
        "updated_at": datetime.utcnow().isoformat(),
        "progress": 0,
        "result": None,
        "error": None
    }
    
    jobs[job_id] = job
    
    # Queue background processing
    background_tasks.add_task(process_estimation, job_id, request)
    
    logger.info(f"Created estimation job: {job_id}")
    
    return JobResponse(
        job_id=job_id,
        status="pending",
        message=f"Parameter estimation job created for dataset: {request.dataset_id}",
        created_at=job["created_at"]
    )

# Get job status
@app.get("/linkage/job/{job_id}")
async def get_job_status(job_id: str):
    """Get the status of a linkage job"""
    if job_id not in jobs:
        raise HTTPException(status_code=404, detail=f"Job {job_id} not found")
    
    return jobs[job_id]

# Get job results
@app.get("/linkage/results/{job_id}")
async def get_job_results(job_id: str):
    """Get the results of a completed linkage job"""
    if job_id not in jobs:
        raise HTTPException(status_code=404, detail=f"Job {job_id} not found")
    
    job = jobs[job_id]
    
    if job["status"] != "completed":
        raise HTTPException(
            status_code=400, 
            detail=f"Job {job_id} is not completed. Current status: {job['status']}"
        )
    
    return {
        "job_id": job_id,
        "type": job["type"],
        "status": job["status"],
        "result": job["result"],
        "completed_at": job["updated_at"]
    }

# List all jobs
@app.get("/linkage/jobs")
async def list_jobs(limit: int = 100, offset: int = 0):
    """List all linkage jobs"""
    job_list = list(jobs.values())
    
    # Sort by creation time (newest first)
    job_list.sort(key=lambda x: x["created_at"], reverse=True)
    
    # Apply pagination
    paginated = job_list[offset:offset + limit]
    
    return {
        "total": len(jobs),
        "limit": limit,
        "offset": offset,
        "jobs": paginated
    }

# Delete job
@app.delete("/linkage/jobs/{job_id}")
async def delete_job(job_id: str):
    """Delete a linkage job and its results"""
    if job_id not in jobs:
        raise HTTPException(status_code=404, detail=f"Job {job_id} not found")
    
    del jobs[job_id]
    
    return {"message": f"Job {job_id} deleted successfully"}

# Background processing functions using actual Splink engine

async def process_deduplication(job_id: str, request: DeduplicationRequest):
    """Process deduplication job using Splink engine"""
    try:
        jobs[job_id]["status"] = "processing"
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        
        # Get Splink engine
        engine = get_splink_engine(SPLINK_BACKEND, DATA_DIR)
        
        # Define progress callback
        def update_progress(progress: int, message: str):
            jobs[job_id]["progress"] = progress
            jobs[job_id]["message"] = message
            jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
            logger.debug(f"Job {job_id}: {progress}% - {message}")
        
        # Run deduplication
        result = await asyncio.to_thread(
            engine.deduplicate,
            request.dataset_id,
            request.settings.dict(),
            update_progress
        )
        
        if result.get("success"):
            jobs[job_id]["status"] = "completed"
            jobs[job_id]["result"] = result
            jobs[job_id]["progress"] = 100
            logger.info(f"Completed deduplication job: {job_id}")
        else:
            jobs[job_id]["status"] = "failed"
            jobs[job_id]["error"] = result.get("error", "Unknown error")
            logger.error(f"Failed deduplication job {job_id}: {result.get('error')}")
        
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        
    except Exception as e:
        jobs[job_id]["status"] = "failed"
        jobs[job_id]["error"] = str(e)
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        logger.error(f"Failed deduplication job {job_id}: {e}")

async def process_linkage(job_id: str, request: LinkRequest):
    """Process linkage job using Splink engine"""
    try:
        jobs[job_id]["status"] = "processing"
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        
        # Get Splink engine
        engine = get_splink_engine(SPLINK_BACKEND, DATA_DIR)
        
        # Define progress callback
        def update_progress(progress: int, message: str):
            jobs[job_id]["progress"] = progress
            jobs[job_id]["message"] = message
            jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
            logger.debug(f"Job {job_id}: {progress}% - {message}")
        
        # Run linkage
        result = await asyncio.to_thread(
            engine.link_datasets,
            request.dataset1_id,
            request.dataset2_id,
            request.settings.dict(),
            update_progress
        )
        
        if result.get("success"):
            jobs[job_id]["status"] = "completed"
            jobs[job_id]["result"] = result
            jobs[job_id]["progress"] = 100
            logger.info(f"Completed linkage job: {job_id}")
        else:
            jobs[job_id]["status"] = "failed"
            jobs[job_id]["error"] = result.get("error", "Unknown error")
            logger.error(f"Failed linkage job {job_id}: {result.get('error')}")
        
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        
    except Exception as e:
        jobs[job_id]["status"] = "failed"
        jobs[job_id]["error"] = str(e)
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        logger.error(f"Failed linkage job {job_id}: {e}")

async def process_estimation(job_id: str, request: EstimationRequest):
    """Process parameter estimation job using Splink engine"""
    try:
        jobs[job_id]["status"] = "processing"
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        
        # Get Splink engine
        engine = get_splink_engine(SPLINK_BACKEND, DATA_DIR)
        
        # Define progress callback
        def update_progress(progress: int, message: str):
            jobs[job_id]["progress"] = progress
            jobs[job_id]["message"] = message
            jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
            logger.debug(f"Job {job_id}: {progress}% - {message}")
        
        # Run parameter estimation
        result = await asyncio.to_thread(
            engine.estimate_parameters,
            request.dataset_id,
            request.settings,
            update_progress
        )
        
        if result.get("success"):
            jobs[job_id]["status"] = "completed"
            jobs[job_id]["result"] = result
            jobs[job_id]["progress"] = 100
            logger.info(f"Completed estimation job: {job_id}")
        else:
            jobs[job_id]["status"] = "failed"
            jobs[job_id]["error"] = result.get("error", "Unknown error")
            logger.error(f"Failed estimation job {job_id}: {result.get('error')}")
        
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        
    except Exception as e:
        jobs[job_id]["status"] = "failed"
        jobs[job_id]["error"] = str(e)
        jobs[job_id]["updated_at"] = datetime.utcnow().isoformat()
        logger.error(f"Failed estimation job {job_id}: {e}")

# Main entry point
if __name__ == "__main__":
    logger.info(f"Starting Splink API on port {SPLINK_PORT} with backend {SPLINK_BACKEND}")
    uvicorn.run(app, host="0.0.0.0", port=SPLINK_PORT)