#!/usr/bin/env python3
"""
Mesa Agent-Based Simulation Framework API
Simplified implementation for scaffolding
"""

import os
import json
import uuid
import random
from typing import Dict, Any, Optional
from datetime import datetime
from pathlib import Path

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

# Configuration
PORT = int(os.getenv("MESA_PORT", 9512))
EXPORT_PATH = Path(os.getenv("MESA_EXPORT_PATH", "/tmp/mesa_exports"))
METRICS_PATH = Path(os.getenv("MESA_METRICS_PATH", "/tmp/mesa_metrics"))

# Ensure directories exist
EXPORT_PATH.mkdir(parents=True, exist_ok=True)
METRICS_PATH.mkdir(parents=True, exist_ok=True)

# FastAPI app
app = FastAPI(
    title="Mesa Agent-Based Simulation API",
    description="Run and manage agent-based simulations",
    version="1.0.0"
)

# Store results in memory
simulation_results = {}


# Request models
class SimulateRequest(BaseModel):
    model: str = Field(..., description="Model to simulate")
    steps: int = Field(100, description="Number of steps")
    parameters: Optional[Dict[str, Any]] = Field(None, description="Model parameters")
    seed: Optional[int] = Field(None, description="Random seed")
    export_snapshots: bool = Field(False, description="Export state snapshots")


class BatchRequest(BaseModel):
    model: str = Field(..., description="Model to simulate")
    parameters: Dict[str, Any] = Field(..., description="Parameter values")
    steps: int = Field(100, description="Steps per run")
    runs: int = Field(1, description="Number of runs")
    seed: Optional[int] = Field(None, description="Base random seed")


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "Mesa Agent-Based Simulation Framework",
        "version": "1.0.0",
        "status": "running"
    }


@app.get("/health")
async def health():
    """Health check endpoint"""
    return {"status": "healthy", "timestamp": datetime.utcnow().isoformat()}


@app.get("/models")
async def list_models():
    """List available models"""
    return {
        "models": [
            {
                "id": "schelling",
                "name": "Schelling Segregation",
                "description": "Agent preference model showing emergence of segregation"
            },
            {
                "id": "virus",
                "name": "Virus on Network", 
                "description": "Epidemiology model of virus spread through network"
            },
            {
                "id": "forest_fire",
                "name": "Forest Fire",
                "description": "Fire spread simulation through forest"
            },
            {
                "id": "wealth",
                "name": "Wealth Distribution",
                "description": "Economic model of wealth distribution"
            }
        ]
    }


@app.post("/simulate")
async def simulate(request: SimulateRequest):
    """Run a simulation - simplified for scaffolding"""
    
    # Set random seed if provided
    if request.seed is not None:
        random.seed(request.seed)
    
    # Generate deterministic mock data based on seed
    results = {
        "model": request.model,
        "parameters": request.parameters or {},
        "steps": request.steps,
        "seed": request.seed,
        "data": []
    }
    
    # Generate deterministic results
    base_value = (request.seed or 42) % 100
    
    for step in range(request.steps):
        step_data = {
            "step": step,
            "agents": 100 - (step % 10)
        }
        
        if request.model == "schelling":
            happy = base_value + (step * 2) % 30
            step_data["happy_agents"] = happy
            step_data["percent_happy"] = happy / 100
        elif request.model == "virus":
            infected = max(1, (base_value - step) % 50)
            step_data["infected"] = infected
            step_data["resistant"] = step % 20
            step_data["susceptible"] = 100 - infected - (step % 20)
        
        results["data"].append(step_data)
        
        # Export snapshot if requested
        if request.export_snapshots and step % 10 == 0:
            snapshot_path = EXPORT_PATH / f"{request.model}_{request.seed}_{step}.json"
            with open(snapshot_path, 'w') as f:
                json.dump(step_data, f)
    
    # Store results
    result_id = str(uuid.uuid4())
    simulation_results[result_id] = results
    
    # Export metrics
    metrics_file = METRICS_PATH / f"{request.model}_{result_id}.json"
    with open(metrics_file, 'w') as f:
        json.dump(results, f, indent=2)
    
    return {
        "id": result_id,
        "results": results,
        "metrics_file": str(metrics_file),
        "snapshots": request.export_snapshots
    }


@app.post("/batch")
async def batch_simulate(request: BatchRequest):
    """Run batch simulations"""
    
    batch_id = str(uuid.uuid4())
    
    # Simplified batch - just return mock response
    return {
        "batch_id": batch_id,
        "status": "completed",
        "runs": request.runs,
        "message": f"Batch of {request.runs} simulations completed"
    }


@app.get("/results")
async def list_results():
    """List all simulation results"""
    return {
        "results": [
            {
                "id": rid,
                "model": data["model"],
                "steps": data["steps"],
                "seed": data["seed"]
            }
            for rid, data in simulation_results.items()
        ]
    }


@app.get("/metrics/latest")
async def get_latest_metrics():
    """Get latest metrics"""
    
    # Find most recent metrics file
    metrics_files = list(METRICS_PATH.glob("*.json"))
    
    if not metrics_files:
        # Return mock metrics for scaffolding
        return {
            "model": "schelling",
            "timestamp": datetime.utcnow().isoformat(),
            "metrics": {
                "total_steps": 100,
                "final_happiness": 0.85,
                "convergence_step": 42
            }
        }
    
    latest = max(metrics_files, key=lambda p: p.stat().st_mtime)
    
    with open(latest) as f:
        return json.load(f)


# Run the application
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=PORT)