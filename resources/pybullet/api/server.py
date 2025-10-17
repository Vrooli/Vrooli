"""PyBullet API Server"""

import os
import time
import asyncio
from typing import Dict, Any, Optional, List
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, validator, Field
import pybullet as p
import pybullet_data
import re

app = FastAPI(title="PyBullet Physics API", version="1.0.0")

# Global simulation instances and metadata
simulations: Dict[str, int] = {}
simulation_metadata: Dict[str, Dict] = {}

class SimulationCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=100, pattern="^[a-zA-Z0-9_-]+$")
    gravity: list = Field(default=[0, 0, -9.81], min_items=3, max_items=3)
    timestep: float = Field(default=1/240, gt=0, le=1.0)
    use_gui: bool = False
    
    @validator('gravity')
    def validate_gravity(cls, v):
        if not all(isinstance(x, (int, float)) and -100 <= x <= 100 for x in v):
            raise ValueError('Gravity values must be numbers between -100 and 100')
        return v

class SimulationStep(BaseModel):
    steps: int = Field(default=1, ge=1, le=10000)
    real_time: bool = False
    target_fps: float = Field(default=60.0, ge=1.0, le=240.0)

class SpawnObject(BaseModel):
    shape: str = Field(default="box", pattern="^(box|sphere|cylinder)$")
    position: list = Field(default=[0, 0, 1], min_items=3, max_items=3)
    size: list = Field(default=[1, 1, 1], min_items=3, max_items=3)
    mass: float = Field(default=1.0, gt=0, le=1000.0)
    color: Optional[List[float]] = Field(default=[0.5, 0.5, 0.5, 1.0], min_items=4, max_items=4)
    
    @validator('position', 'size')
    def validate_vectors(cls, v):
        if not all(isinstance(x, (int, float)) and -1000 <= x <= 1000 for x in v):
            raise ValueError('Vector values must be numbers between -1000 and 1000')
        return v
    
    @validator('color')
    def validate_color(cls, v):
        if v and not all(isinstance(x, (int, float)) and 0 <= x <= 1 for x in v):
            raise ValueError('Color values must be between 0 and 1')
        return v

class ApplyForce(BaseModel):
    body_id: int
    force: List[float]
    position: Optional[List[float]] = None
    
class SetJoint(BaseModel):
    body_id: int
    joint_index: int
    target_position: Optional[float] = None
    target_velocity: Optional[float] = None
    force: Optional[float] = None

@app.get("/health")
async def health():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "pybullet",
        "version": "2.4.1",
        "simulations": len(simulations),
        "timestamp": time.time()
    }

@app.post("/simulation/create")
async def create_simulation(config: SimulationCreate):
    """Create new simulation instance"""
    # Check simulation limits
    max_simulations = int(os.environ.get("PYBULLET_MAX_SIMULATIONS", "10"))
    if len(simulations) >= max_simulations:
        raise HTTPException(status_code=429, detail=f"Maximum number of simulations ({max_simulations}) reached")
    
    if config.name in simulations:
        raise HTTPException(status_code=400, detail="Simulation already exists")
    
    if config.use_gui:
        client_id = p.connect(p.GUI)
    else:
        client_id = p.connect(p.DIRECT)
    
    p.setGravity(*config.gravity, physicsClientId=client_id)
    p.setTimeStep(config.timestep, physicsClientId=client_id)
    p.setAdditionalSearchPath(pybullet_data.getDataPath())
    
    # Load ground plane
    p.loadURDF("plane.urdf", physicsClientId=client_id)
    
    simulations[config.name] = client_id
    simulation_metadata[config.name] = {
        "timestep": config.timestep,
        "created_at": time.time(),
        "last_step": time.time(),
        "total_steps": 0
    }
    
    return {
        "name": config.name,
        "client_id": client_id,
        "status": "created"
    }

@app.delete("/simulation/{name}")
async def destroy_simulation(name: str):
    """Destroy simulation instance"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    p.disconnect(simulations[name])
    del simulations[name]
    if name in simulation_metadata:
        del simulation_metadata[name]
    
    return {"status": "destroyed", "name": name}

@app.post("/simulation/{name}/step")
async def step_simulation(name: str, config: SimulationStep):
    """Advance simulation by N steps with optional real-time control"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    metadata = simulation_metadata[name]
    
    if config.real_time:
        # Real-time stepping with target FPS
        target_timestep = 1.0 / config.target_fps
        start_time = time.time()
        
        for i in range(config.steps):
            step_start = time.time()
            p.stepSimulation(physicsClientId=client_id)
            metadata["total_steps"] += 1
            
            # Sleep to maintain target FPS
            elapsed = time.time() - step_start
            if elapsed < target_timestep:
                await asyncio.sleep(target_timestep - elapsed)
        
        actual_time = time.time() - start_time
        actual_fps = config.steps / actual_time if actual_time > 0 else 0
        
        metadata["last_step"] = time.time()
        return {
            "status": "stepped", 
            "steps": config.steps,
            "real_time": True,
            "target_fps": config.target_fps,
            "actual_fps": actual_fps
        }
    else:
        # Fast stepping without real-time constraints
        for _ in range(config.steps):
            p.stepSimulation(physicsClientId=client_id)
            metadata["total_steps"] += 1
        
        metadata["last_step"] = time.time()
        return {"status": "stepped", "steps": config.steps, "real_time": False}

@app.get("/simulation/{name}/state")
async def get_state(name: str):
    """Get current simulation state"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    num_bodies = p.getNumBodies(physicsClientId=client_id)
    
    bodies = []
    for i in range(num_bodies):
        pos, orn = p.getBasePositionAndOrientation(i, physicsClientId=client_id)
        vel, ang_vel = p.getBaseVelocity(i, physicsClientId=client_id)
        bodies.append({
            "id": i,
            "position": pos,
            "orientation": orn,
            "velocity": vel,
            "angular_velocity": ang_vel
        })
    
    return {
        "name": name,
        "num_bodies": num_bodies,
        "bodies": bodies
    }

@app.post("/simulation/{name}/spawn")
async def spawn_object(name: str, config: SpawnObject):
    """Spawn object in simulation"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    
    # Create collision shape
    if config.shape == "box":
        shape_id = p.createCollisionShape(p.GEOM_BOX, halfExtents=[s/2 for s in config.size])
        visual_id = p.createVisualShape(p.GEOM_BOX, halfExtents=[s/2 for s in config.size], 
                                       rgbaColor=config.color)
    elif config.shape == "sphere":
        shape_id = p.createCollisionShape(p.GEOM_SPHERE, radius=config.size[0])
        visual_id = p.createVisualShape(p.GEOM_SPHERE, radius=config.size[0],
                                       rgbaColor=config.color)
    elif config.shape == "cylinder":
        shape_id = p.createCollisionShape(p.GEOM_CYLINDER, radius=config.size[0], height=config.size[1])
        visual_id = p.createVisualShape(p.GEOM_CYLINDER, radius=config.size[0], length=config.size[1],
                                       rgbaColor=config.color)
    else:
        raise HTTPException(status_code=400, detail=f"Unknown shape: {config.shape}")
    
    # Create body with visual shape
    body_id = p.createMultiBody(
        baseMass=config.mass,
        baseCollisionShapeIndex=shape_id,
        baseVisualShapeIndex=visual_id,
        basePosition=config.position,
        physicsClientId=client_id
    )
    
    return {"body_id": body_id, "shape": config.shape, "position": config.position}

@app.post("/simulation/{name}/apply_force")
async def apply_force(name: str, config: ApplyForce):
    """Apply force to a body in the simulation"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    
    if config.position:
        p.applyExternalForce(
            config.body_id,
            -1,  # Link index (-1 for base)
            config.force,
            config.position,
            p.WORLD_FRAME,
            physicsClientId=client_id
        )
    else:
        p.applyExternalForce(
            config.body_id,
            -1,
            config.force,
            [0, 0, 0],
            p.LINK_FRAME,
            physicsClientId=client_id
        )
    
    return {"status": "force_applied", "body_id": config.body_id, "force": config.force}

@app.post("/simulation/{name}/set_joint")
async def set_joint(name: str, config: SetJoint):
    """Control robot joint position/velocity"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    
    if config.target_position is not None:
        p.setJointMotorControl2(
            config.body_id,
            config.joint_index,
            p.POSITION_CONTROL,
            targetPosition=config.target_position,
            force=config.force if config.force else 1000,
            physicsClientId=client_id
        )
    elif config.target_velocity is not None:
        p.setJointMotorControl2(
            config.body_id,
            config.joint_index,
            p.VELOCITY_CONTROL,
            targetVelocity=config.target_velocity,
            force=config.force if config.force else 1000,
            physicsClientId=client_id
        )
    else:
        raise HTTPException(status_code=400, detail="Must specify target_position or target_velocity")
    
    return {
        "status": "joint_set",
        "body_id": config.body_id,
        "joint_index": config.joint_index
    }

@app.get("/simulation/{name}/sensors")
async def get_sensors(name: str, body_id: int):
    """Read sensor data from a body"""
    if name not in simulations:
        raise HTTPException(status_code=404, detail="Simulation not found")
    
    client_id = simulations[name]
    
    # Get joint states if the body has joints
    num_joints = p.getNumJoints(body_id, physicsClientId=client_id)
    joint_states = []
    for i in range(num_joints):
        state = p.getJointState(body_id, i, physicsClientId=client_id)
        joint_info = p.getJointInfo(body_id, i, physicsClientId=client_id)
        joint_states.append({
            "index": i,
            "name": joint_info[1].decode('utf-8') if joint_info[1] else f"joint_{i}",
            "position": state[0],
            "velocity": state[1],
            "reaction_forces": state[2],
            "applied_torque": state[3]
        })
    
    # Get base state
    pos, orn = p.getBasePositionAndOrientation(body_id, physicsClientId=client_id)
    vel, ang_vel = p.getBaseVelocity(body_id, physicsClientId=client_id)
    
    return {
        "body_id": body_id,
        "base": {
            "position": pos,
            "orientation": orn,
            "velocity": vel,
            "angular_velocity": ang_vel
        },
        "joints": joint_states
    }

if __name__ == "__main__":
    import uvicorn
    # Port must come from environment, no fallback
    port_str = os.environ.get("PYBULLET_PORT")
    if not port_str:
        raise ValueError("PYBULLET_PORT environment variable not set")
    port = int(port_str)
    uvicorn.run(app, host="0.0.0.0", port=port)
