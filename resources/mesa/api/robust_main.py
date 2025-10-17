#!/usr/bin/env python3
"""
Mesa Agent-Based Simulation Framework API
Robust implementation with fallback for environments without Mesa
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

# Configuration
PORT = int(os.getenv("MESA_PORT", 9512))
EXPORT_PATH = Path(os.getenv("MESA_EXPORT_PATH", "/tmp/mesa_exports"))
METRICS_PATH = Path(os.getenv("MESA_METRICS_PATH", "/tmp/mesa_metrics"))
REDIS_HOST = os.getenv("REDIS_HOST", "localhost")
REDIS_PORT = int(os.getenv("REDIS_PORT", 6379))
REDIS_DB = int(os.getenv("REDIS_DB", 0))

# Ensure directories exist
EXPORT_PATH.mkdir(parents=True, exist_ok=True)
METRICS_PATH.mkdir(parents=True, exist_ok=True)

# FastAPI app
app = FastAPI(
    title="Mesa Agent-Based Simulation API",
    description="Run and manage agent-based simulations",
    version="1.0.0"
)

# Check if Mesa is available
MESA_AVAILABLE = False
try:
    import mesa
    from mesa import Agent, Model
    from mesa.time import RandomActivation
    from mesa.space import MultiGrid
    from mesa.datacollection import DataCollector
    MESA_AVAILABLE = True
    print(f"Mesa {mesa.__version__} loaded successfully")
except ImportError:
    print("Warning: Mesa not installed. Running in simulation mode.")

# Check if Redis is available
REDIS_AVAILABLE = False
redis_client = None
try:
    import redis
    redis_client = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, db=REDIS_DB, decode_responses=True)
    redis_client.ping()
    REDIS_AVAILABLE = True
    print(f"Redis connected at {REDIS_HOST}:{REDIS_PORT}")
except Exception as e:
    print(f"Redis not available: {e}. Results will be stored in memory only.")

# Store results in memory
simulation_results = {}
batch_jobs = {}

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


# Simple built-in models for demonstration
class SimpleAgent(Agent if MESA_AVAILABLE else object):
    """Simple agent for demonstration"""
    def __init__(self, unique_id, model):
        if MESA_AVAILABLE:
            super().__init__(unique_id, model)
        else:
            self.unique_id = unique_id
            self.model = model
        self.happy = random.random() > 0.5
    
    def step(self):
        # Simple behavior: randomly change happiness
        if random.random() < 0.1:
            self.happy = not self.happy


class SimpleModel(Model if MESA_AVAILABLE else object):
    """Simple model for demonstration"""
    def __init__(self, n_agents=10, width=10, height=10):
        if MESA_AVAILABLE:
            super().__init__()
            self.schedule = RandomActivation(self)
            self.grid = MultiGrid(width, height, True)
        else:
            self.schedule = type('Schedule', (), {'agents': [], 'step': lambda: None})()
            
        self.n_agents = n_agents
        
        # Create agents
        for i in range(n_agents):
            agent = SimpleAgent(i, self)
            if MESA_AVAILABLE:
                self.schedule.add(agent)
            else:
                self.schedule.agents.append(agent)
    
    def step(self):
        """Advance model by one step"""
        if MESA_AVAILABLE:
            self.schedule.step()
        else:
            for agent in self.schedule.agents:
                agent.step()


# Available models
def get_available_models():
    """Get available models based on what's installed"""
    models = {
        "simple": {
            "name": "Simple Demonstration Model",
            "params": {
                "n_agents": 10,
                "width": 10,
                "height": 10
            }
        }
    }
    
    if MESA_AVAILABLE:
        # Add real Mesa models if available
        try:
            from mesa.examples import schelling, virus_on_network
            models.update({
                "schelling": {
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
                    "name": "Virus on Network",
                    "params": {
                        "num_nodes": 50,
                        "avg_node_degree": 3,
                        "initial_infection": 0.1,
                        "virus_spread_chance": 0.4,
                        "recovery_chance": 0.3
                    }
                }
            })
        except ImportError:
            print("Mesa examples not available")
    
    return models


# Endpoints
@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "Mesa Agent-Based Simulation Framework",
        "version": "1.0.0",
        "status": "running",
        "mesa_available": MESA_AVAILABLE
    }


@app.get("/health")
async def health():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "mesa_installed": MESA_AVAILABLE,
        "redis_connected": REDIS_AVAILABLE
    }


@app.get("/models")
async def list_models():
    """List available models"""
    models = get_available_models()
    return {
        "models": [
            {
                "id": key,
                "name": value["name"],
                "parameters": list(value["params"].keys())
            }
            for key, value in models.items()
        ]
    }


@app.post("/simulate")
async def simulate(request: SimulateRequest):
    """Run a simulation"""
    
    models = get_available_models()
    
    # Validate model
    if request.model not in models:
        raise HTTPException(400, f"Unknown model: {request.model}")
    
    # Set random seed if provided
    if request.seed is not None:
        random.seed(request.seed)
    
    # Get model configuration
    model_config = models[request.model]
    params = model_config["params"].copy()
    
    # Override with custom parameters
    if request.parameters:
        params.update(request.parameters)
    
    # Create and run model
    try:
        if request.model == "simple" or not MESA_AVAILABLE:
            # Use simple model
            model = SimpleModel(**{k: v for k, v in params.items() if k in ['n_agents', 'width', 'height']})
        else:
            # Try to load real Mesa model
            if request.model == "schelling":
                from mesa.examples.schelling import Schelling
                model = Schelling(**params)
            elif request.model == "virus":
                from mesa.examples.virus_on_network import VirusOnNetwork
                model = VirusOnNetwork(**params)
            else:
                raise ValueError(f"Model {request.model} not implemented")
        
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
                "agents": len(model.schedule.agents) if hasattr(model.schedule, 'agents') else 0
            }
            
            # Model-specific metrics
            if request.model == "simple" or (request.model == "schelling" and not MESA_AVAILABLE):
                happy = sum(1 for agent in model.schedule.agents if hasattr(agent, 'happy') and agent.happy)
                step_data["happy_agents"] = happy
                step_data["percent_happy"] = happy / len(model.schedule.agents) if model.schedule.agents else 0
            
            results["data"].append(step_data)
            
            # Export snapshot if requested
            if request.export_snapshots and step % 10 == 0:
                snapshot_path = EXPORT_PATH / f"{request.model}_{request.seed}_{step}.json"
                with open(snapshot_path, 'w') as f:
                    json.dump(step_data, f)
        
        # Store results
        result_id = str(uuid.uuid4())
        simulation_results[result_id] = results
        
        # Store in Redis if available
        if REDIS_AVAILABLE and redis_client:
            try:
                # Store result with expiry (24 hours)
                redis_client.setex(
                    f"mesa:result:{result_id}",
                    86400,  # 24 hour expiry
                    json.dumps(results)
                )
                # Add to results list
                redis_client.lpush("mesa:results", result_id)
                redis_client.ltrim("mesa:results", 0, 999)  # Keep last 1000
                print(f"Stored result {result_id} in Redis")
            except Exception as e:
                print(f"Failed to store in Redis: {e}")
        
        # Export metrics
        metrics_file = METRICS_PATH / f"{request.model}_{result_id}.json"
        with open(metrics_file, 'w') as f:
            json.dump(results, f, indent=2)
        
        return {
            "id": result_id,
            "results": results,
            "metrics_file": str(metrics_file),
            "snapshots": request.export_snapshots,
            "redis_stored": REDIS_AVAILABLE
        }
        
    except Exception as e:
        raise HTTPException(500, f"Simulation failed: {str(e)}")


@app.post("/batch")
async def batch_simulate(request: BatchRequest, background_tasks: BackgroundTasks):
    """Run batch simulations with parameter sweeps"""
    
    models = get_available_models()
    
    if request.model not in models:
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
    
    # Try memory first
    if result_id in simulation_results:
        return simulation_results[result_id]
    
    # Try Redis if available
    if REDIS_AVAILABLE and redis_client:
        try:
            result_json = redis_client.get(f"mesa:result:{result_id}")
            if result_json:
                return json.loads(result_json)
        except Exception as e:
            print(f"Redis fetch error: {e}")
    
    raise HTTPException(404, "Result not found")


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