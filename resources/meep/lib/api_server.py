#!/usr/bin/env python3
"""
MEEP API Server - Provides REST interface for FDTD simulations
"""

import os
import json
import asyncio
import hashlib
from datetime import datetime
from pathlib import Path
from typing import Optional, Dict, Any, List
import logging

from fastapi import FastAPI, HTTPException, UploadFile, File, Body
from fastapi.responses import FileResponse, JSONResponse
from pydantic import BaseModel
import uvicorn
import numpy as np

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Try to import MEEP
try:
    import meep as mp
    MEEP_AVAILABLE = True
except ImportError:
    MEEP_AVAILABLE = False
    logger.warning("MEEP not available, running in mock mode")

# Try to import database libraries
DB_AVAILABLE = False
try:
    import psycopg2
    from psycopg2.extras import RealDictCursor
    DB_AVAILABLE = True
    logger.info("PostgreSQL support enabled")
except ImportError:
    logger.warning("PostgreSQL not available, using in-memory storage")

# Try to import MinIO libraries
MINIO_AVAILABLE = False
try:
    from minio import Minio
    from minio.error import S3Error
    MINIO_AVAILABLE = True
    logger.info("MinIO support enabled")
except ImportError:
    logger.warning("MinIO not available, using local filesystem only")

app = FastAPI(title="MEEP API Server", version="1.0.0")

# Configuration
API_PORT = int(os.environ.get("MEEP_PORT", 8193))
DATA_DIR = Path("/data")
RESULTS_DIR = DATA_DIR / "results"
TEMPLATES_DIR = DATA_DIR / "templates"

# Ensure directories exist
RESULTS_DIR.mkdir(parents=True, exist_ok=True)
TEMPLATES_DIR.mkdir(parents=True, exist_ok=True)

# Store active simulations
active_simulations = {}

# Database connection helper
def get_db_connection():
    """Get database connection if available"""
    if not DB_AVAILABLE:
        return None
    
    try:
        # Get PostgreSQL connection from environment or use defaults
        conn = psycopg2.connect(
            host=os.environ.get("POSTGRES_HOST", "localhost"),
            port=os.environ.get("POSTGRES_PORT", "5432"),
            database=os.environ.get("POSTGRES_DB", "meep"),
            user=os.environ.get("POSTGRES_USER", "postgres"),
            password=os.environ.get("POSTGRES_PASSWORD", "postgres")
        )
        return conn
    except Exception as e:
        logger.error(f"Database connection failed: {e}")
        return None

def get_minio_client():
    """Get MinIO client if available"""
    if not MINIO_AVAILABLE:
        return None
    
    try:
        client = Minio(
            os.environ.get("MINIO_ENDPOINT", "localhost:9000"),
            access_key=os.environ.get("MINIO_ACCESS_KEY", "minioadmin"),
            secret_key=os.environ.get("MINIO_SECRET_KEY", "minioadmin"),
            secure=False
        )
        
        # Ensure bucket exists
        bucket_name = "meep-results"
        if not client.bucket_exists(bucket_name):
            client.make_bucket(bucket_name)
        
        return client
    except Exception as e:
        logger.error(f"MinIO connection failed: {e}")
        return None

def save_result_to_minio(sim_id: str, file_path: Path, file_type: str = "hdf5"):
    """Save result file to MinIO"""
    if not MINIO_AVAILABLE:
        logger.info(f"MinIO not available, keeping {file_path} local")
        return None
    
    client = get_minio_client()
    if not client:
        return None
    
    try:
        bucket_name = "meep-results"
        object_name = f"{sim_id}/{file_path.name}"
        
        client.fput_object(
            bucket_name,
            object_name,
            str(file_path),
            content_type="application/octet-stream"
        )
        
        logger.info(f"Saved {object_name} to MinIO")
        return f"minio://{bucket_name}/{object_name}"
    except Exception as e:
        logger.error(f"Failed to save to MinIO: {e}")
        return None

def save_simulation_metadata(sim_id: str, metadata: dict):
    """Save simulation metadata to database"""
    if not DB_AVAILABLE:
        logger.info(f"Saving simulation {sim_id} to memory only")
        return
    
    conn = get_db_connection()
    if not conn:
        return
    
    try:
        with conn.cursor() as cur:
            # Create table if not exists
            cur.execute("""
                CREATE TABLE IF NOT EXISTS simulations (
                    id VARCHAR(255) PRIMARY KEY,
                    status VARCHAR(50),
                    created_at TIMESTAMP,
                    config JSONB,
                    results JSONB
                )
            """)
            
            # Insert or update simulation
            cur.execute("""
                INSERT INTO simulations (id, status, created_at, config)
                VALUES (%s, %s, %s, %s)
                ON CONFLICT (id) DO UPDATE
                SET status = EXCLUDED.status,
                    config = EXCLUDED.config
            """, (
                sim_id,
                metadata.get("status"),
                metadata.get("created_at"),
                json.dumps(metadata.get("config", {}))
            ))
            
            conn.commit()
            logger.info(f"Saved simulation {sim_id} to database")
    except Exception as e:
        logger.error(f"Failed to save simulation metadata: {e}")
    finally:
        conn.close()

class SimulationRequest(BaseModel):
    template: Optional[str] = None
    resolution: Optional[float] = 50
    runtime: Optional[float] = 100
    geometry: Optional[Dict[str, Any]] = None
    sources: Optional[List[Dict[str, Any]]] = None
    monitors: Optional[List[Dict[str, Any]]] = None

class SimulationResponse(BaseModel):
    simulation_id: str
    status: str
    created_at: str

class HealthResponse(BaseModel):
    status: str
    meep_available: bool
    version: str
    uptime_seconds: float

# Track server start time
server_start_time = datetime.now()

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    uptime = (datetime.now() - server_start_time).total_seconds()
    
    return HealthResponse(
        status="healthy",
        meep_available=MEEP_AVAILABLE,
        version="1.0.0",
        uptime_seconds=uptime
    )

@app.post("/simulation/create", response_model=SimulationResponse)
async def create_simulation(request: SimulationRequest):
    """Create a new simulation"""
    sim_id = hashlib.md5(
        json.dumps(request.dict(), sort_keys=True).encode()
    ).hexdigest()[:12]
    
    metadata = {
        "id": sim_id,
        "status": "created",
        "created_at": datetime.now().isoformat(),
        "config": request.dict()
    }
    
    active_simulations[sim_id] = metadata
    
    # Persist to database if available
    save_simulation_metadata(sim_id, metadata)
    
    return SimulationResponse(
        simulation_id=sim_id,
        status="created",
        created_at=metadata["created_at"]
    )

@app.post("/simulation/{sim_id}/run")
async def run_simulation(sim_id: str):
    """Execute a simulation"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    sim = active_simulations[sim_id]
    sim["status"] = "running"
    
    # In production, this would launch actual MEEP simulation
    # For scaffolding, we'll simulate completion
    if MEEP_AVAILABLE:
        # Run basic waveguide simulation
        asyncio.create_task(run_meep_simulation(sim_id))
    else:
        # Mock simulation
        await asyncio.sleep(1)
        sim["status"] = "completed"
    
    return {"message": f"Simulation {sim_id} started"}

@app.get("/simulation/{sim_id}/status")
async def get_simulation_status(sim_id: str):
    """Get simulation status"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    sim = active_simulations[sim_id]
    return {
        "simulation_id": sim_id,
        "status": sim["status"],
        "created_at": sim["created_at"]
    }

@app.get("/simulation/{sim_id}/fields")
async def get_field_data(sim_id: str):
    """Download field data in HDF5 format"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    # Check if results exist
    result_file = RESULTS_DIR / f"{sim_id}_fields.h5"
    
    if not result_file.exists():
        # Create mock HDF5 file for testing
        import h5py
        with h5py.File(result_file, 'w') as f:
            f.create_dataset('ez', data=np.random.rand(100, 100))
    
    return FileResponse(
        path=result_file,
        media_type="application/x-hdf5",
        filename=f"{sim_id}_fields.h5"
    )

@app.get("/simulation/{sim_id}/spectra")
async def get_spectra_data(sim_id: str):
    """Get transmission/reflection spectra"""
    if sim_id not in active_simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    # Mock spectral data
    frequencies = np.linspace(0.1, 0.5, 100)
    transmission = np.exp(-(frequencies - 0.3)**2 / 0.01)
    
    return {
        "frequencies": frequencies.tolist(),
        "transmission": transmission.tolist(),
        "reflection": (1 - transmission).tolist()
    }

@app.post("/batch/sweep")
async def parameter_sweep(request: dict = Body(...)):
    """Run parameter sweep"""
    template = request.get("template")
    parameter = request.get("parameter")
    values = request.get("values", [])
    
    sweep_id = hashlib.md5(
        f"{template}{parameter}{values}".encode()
    ).hexdigest()[:12]
    
    simulations = []
    for value in values:
        sim_id = f"{sweep_id}_{value}"
        active_simulations[sim_id] = {
            "id": sim_id,
            "status": "queued",
            "created_at": datetime.now().isoformat(),
            "sweep_id": sweep_id,
            "parameter": parameter,
            "value": value
        }
        simulations.append(sim_id)
    
    return {
        "sweep_id": sweep_id,
        "simulations": simulations,
        "parameter": parameter,
        "values": values
    }

@app.get("/templates")
async def list_templates():
    """List available simulation templates"""
    templates = []
    
    # Add built-in templates
    templates.extend([
        {"name": "waveguide", "description": "2D waveguide with bends"},
        {"name": "resonator", "description": "Ring resonator"},
        {"name": "photonic_crystal", "description": "2D photonic crystal"}
    ])
    
    # Add user templates
    for template_file in TEMPLATES_DIR.glob("*.py"):
        templates.append({
            "name": template_file.stem,
            "description": "User template"
        })
    
    return {"templates": templates}

async def run_meep_simulation(sim_id: str):
    """Run actual MEEP simulation (async)"""
    sim = active_simulations[sim_id]
    
    try:
        # Simple waveguide simulation
        cell_size = mp.Vector3(16, 8, 0)
        
        geometry = [
            mp.Block(
                mp.Vector3(mp.inf, 1, mp.inf),
                center=mp.Vector3(),
                material=mp.Medium(epsilon=12)
            )
        ]
        
        sources = [
            mp.Source(
                mp.ContinuousSource(frequency=0.15),
                component=mp.Ez,
                center=mp.Vector3(-7, 0)
            )
        ]
        
        pml_layers = [mp.PML(1.0)]
        
        sim_obj = mp.Simulation(
            cell_size=cell_size,
            boundary_layers=pml_layers,
            geometry=geometry,
            sources=sources,
            resolution=sim["config"].get("resolution", 50)
        )
        
        sim_obj.run(until=sim["config"].get("runtime", 100))
        
        # Save results
        result_file = RESULTS_DIR / f"{sim_id}_fields.h5"
        sim_obj.output_epsilon(output_file=str(result_file))
        
        sim["status"] = "completed"
    except Exception as e:
        sim["status"] = "failed"
        sim["error"] = str(e)

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=API_PORT)
