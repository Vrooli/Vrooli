#!/usr/bin/env python3
"""
Mesa Agent-Based Simulation Framework API
Provides REST endpoints for model execution and management
"""

import os
import json
import uuid
import random
import asyncio
from typing import Dict, Any, List, Optional
from datetime import datetime
from pathlib import Path

from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

# Import Mesa models
from mesa.examples.schelling import Schelling
from mesa.examples.virus_on_network import VirusOnNetwork
from mesa.examples.wolf_sheep import WolfSheep
from mesa.examples.forest_fire import ForestFire

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

# Store results in memory (in production, use Redis/database)
simulation_results = {}
batch_jobs = {}

# Available models
MODELS = {
    "schelling": {
        "class": Schelling,
        "name": "Schelling Segregation",
        "params": {
            "width": 20,
            "height": 20,
            "density": 0.8,
            "minority_fraction": 0.2,
            "homophily": 3
        }
    },
    "virus": {
        "class": VirusOnNetwork,
        "name": "Virus on Network",
        "params": {
            "num_nodes": 50,
            "avg_node_degree": 3,
            "initial_infection": 0.1,
            "virus_spread_chance": 0.4,
            "recovery_chance": 0.3,
            "gain_resistance_chance": 0.05
        }
    },
    "forest_fire": {
        "class": ForestFire,
        "name": "Forest Fire",
        "params": {
            "width": 50,
            "height": 50,
            "density": 0.65
        }
    },
    "wolf_sheep": {
        "class": WolfSheep,
        "name": "Wolf-Sheep Predation",
        "params": {
            "width": 20,
            "height": 20,
            "initial_sheep": 100,
            "initial_wolves": 50,
            "sheep_reproduce": 0.04,
            "wolf_reproduce": 0.05,
            "grass_regrowth_time": 30,
            "wolf_gain_from_food": 20,
            "sheep_gain_from_food": 4
        }
    }
}


# Request models
class SimulateRequest(BaseModel):
    model: str = Field(..., description="Model to simulate")
    steps: int = Field(100, description="Number of steps")
    parameters: Optional[Dict[str, Any]] = Field(None, description="Model parameters")
    seed: Optional[int] = Field(None, description="Random seed")
    export_snapshots: bool = Field(False, description="Export state snapshots")
    
class BatchRequest(BaseModel):
    model: str = Field(..., description="Model to simulate")
    parameters: Dict[str, List[Any]] = Field(..., description="Parameter sweep")
    steps: int = Field(100, description="Steps per run")
    runs: int = Field(1, description="Runs per parameter set")
    seed: Optional[int] = Field(None, description="Base random seed")


# Endpoints
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
                "id": key,
                "name": value["name"],
                "parameters": list(value["params"].keys())
            }
            for key, value in MODELS.items()
        ]
    }


@app.post("/simulate")
async def simulate(request: SimulateRequest):
    """Run a simulation"""
    
    # Validate model
    if request.model not in MODELS:
        raise HTTPException(400, f"Unknown model: {request.model}")
    
    # Set random seed if provided
    if request.seed is not None:
        random.seed(request.seed)
    
    # Get model configuration
    model_config = MODELS[request.model]
    params = model_config["params"].copy()
    
    # Override with custom parameters
    if request.parameters:
        params.update(request.parameters)
    
    # Create and run model
    try:
        model_class = model_config["class"]
        model = model_class(**params)
        
        # Collect data
        results = {
            "model": request.model,
            "parameters": params,
            "steps": request.steps,
            "seed": request.seed,
            "data": []
        }
        
        # Run simulation
        for step in range(request.steps):
            model.step()
            
            # Collect metrics
            step_data = {
                "step": step,
                "agents": model.schedule.get_agent_count() if hasattr(model.schedule, 'get_agent_count') else len(model.schedule.agents)
            }
            
            # Model-specific metrics
            if request.model == "schelling":
                happy = sum(1 for agent in model.schedule.agents if agent.happy)
                step_data["happy_agents"] = happy
                step_data["percent_happy"] = happy / len(model.schedule.agents) if model.schedule.agents else 0
            elif request.model == "virus":
                infected = sum(1 for agent in model.nodes if agent.state == 1)
                resistant = sum(1 for agent in model.nodes if agent.state == 2)
                step_data["infected"] = infected
                step_data["resistant"] = resistant
                step_data["susceptible"] = len(model.nodes) - infected - resistant
            
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
        
    except Exception as e:
        raise HTTPException(500, f"Simulation failed: {str(e)}")


@app.post("/batch")
async def batch_simulate(request: BatchRequest, background_tasks: BackgroundTasks):
    """Run batch simulations with parameter sweeps"""
    
    if request.model not in MODELS:
        raise HTTPException(400, f"Unknown model: {request.model}")
    
    batch_id = str(uuid.uuid4())
    batch_jobs[batch_id] = {
        "status": "running",
        "total": 0,
        "completed": 0,
        "results": []
    }
    
    # Run simulations in background
    background_tasks.add_task(run_batch, batch_id, request)
    
    return {
        "batch_id": batch_id,
        "status": "started",
        "message": "Batch simulation started in background"
    }


async def run_batch(batch_id: str, request: BatchRequest):
    """Background task to run batch simulations"""
    
    import itertools
    
    # Generate parameter combinations
    param_names = list(request.parameters.keys())
    param_values = [request.parameters[name] for name in param_names]
    combinations = list(itertools.product(*param_values))
    
    total_runs = len(combinations) * request.runs
    batch_jobs[batch_id]["total"] = total_runs
    
    results = []
    base_seed = request.seed or 42
    
    for combo_idx, combo in enumerate(combinations):
        # Create parameter dict
        params = dict(zip(param_names, combo))
        
        for run in range(request.runs):
            # Set seed for reproducibility
            seed = base_seed + combo_idx * 1000 + run
            
            # Run simulation
            sim_request = SimulateRequest(
                model=request.model,
                steps=request.steps,
                parameters=params,
                seed=seed
            )
            
            result = await simulate(sim_request)
            results.append({
                "parameters": params,
                "run": run,
                "seed": seed,
                "result_id": result["id"]
            })
            
            batch_jobs[batch_id]["completed"] += 1
    
    batch_jobs[batch_id]["status"] = "completed"
    batch_jobs[batch_id]["results"] = results


@app.get("/batch/{batch_id}")
async def get_batch_status(batch_id: str):
    """Get batch job status"""
    
    if batch_id not in batch_jobs:
        raise HTTPException(404, "Batch job not found")
    
    return batch_jobs[batch_id]


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


@app.get("/results/{result_id}")
async def get_result(result_id: str):
    """Get specific simulation result"""
    
    if result_id not in simulation_results:
        raise HTTPException(404, "Result not found")
    
    return simulation_results[result_id]


@app.get("/metrics/latest")
async def get_latest_metrics():
    """Get latest metrics"""
    
    # Find most recent metrics file
    metrics_files = list(METRICS_PATH.glob("*.json"))
    
    if not metrics_files:
        return {"message": "No metrics available"}
    
    latest = max(metrics_files, key=lambda p: p.stat().st_mtime)
    
    with open(latest) as f:
        return json.load(f)


# Run the application
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=PORT)