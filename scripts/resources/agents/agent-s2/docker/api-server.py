#!/usr/bin/env python3
"""
Agent S2 REST API Server
Provides HTTP endpoints for Agent S2 autonomous computer interaction
"""

import os
import sys
import json
import base64
import asyncio
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
import uvicorn
import pyautogui
from PIL import Image
import io

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configure pyautogui
pyautogui.FAILSAFE = False  # Disable failsafe in container
pyautogui.PAUSE = 0.1  # Small pause between actions

# Global state
app_state = {
    "tasks": {},
    "task_counter": 0,
    "startup_time": datetime.utcnow().isoformat()
}

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle"""
    logger.info("Starting Agent S2 API Server...")
    # Startup
    yield
    # Shutdown
    logger.info("Shutting down Agent S2 API Server...")

# Create FastAPI app
app = FastAPI(
    title="Agent S2 API",
    description="Autonomous computer interaction service",
    version="1.0.0",
    lifespan=lifespan
)

# Request/Response models
class TaskRequest(BaseModel):
    """Task execution request"""
    task_type: str = Field(..., description="Type of task: screenshot, click, type, automation")
    parameters: Dict[str, Any] = Field(default_factory=dict, description="Task parameters")
    async_execution: bool = Field(default=False, description="Execute task asynchronously")

class TaskResponse(BaseModel):
    """Task execution response"""
    task_id: str
    status: str
    result: Optional[Dict[str, Any]] = None
    error: Optional[str] = None
    created_at: str
    completed_at: Optional[str] = None

class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    display: str
    screen_size: Dict[str, int]
    startup_time: str
    tasks_processed: int

# API Endpoints
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Check service health and display status"""
    try:
        # Get screen information
        screen_width, screen_height = pyautogui.size()
        
        return HealthResponse(
            status="healthy",
            display=os.environ.get("DISPLAY", "unknown"),
            screen_size={"width": screen_width, "height": screen_height},
            startup_time=app_state["startup_time"],
            tasks_processed=app_state["task_counter"]
        )
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"Service unhealthy: {str(e)}")

@app.get("/capabilities")
async def get_capabilities():
    """Get Agent S2 capabilities"""
    return {
        "capabilities": {
            "screenshot": True,
            "gui_automation": True,
            "mouse_control": True,
            "keyboard_control": True,
            "window_management": True,
            "planning": True,
            "multi_step_tasks": True
        },
        "supported_tasks": [
            "screenshot",
            "click",
            "double_click",
            "right_click",
            "type_text",
            "key_press",
            "mouse_move",
            "drag_drop",
            "scroll",
            "automation_sequence"
        ],
        "display_info": {
            "display": os.environ.get("DISPLAY", ":99"),
            "resolution": pyautogui.size()
        }
    }

@app.post("/screenshot")
async def take_screenshot(
    format: str = "png",
    quality: int = 95,
    region: Optional[List[int]] = None
):
    """Take a screenshot of the current display"""
    try:
        # Take screenshot
        if region and len(region) == 4:
            screenshot = pyautogui.screenshot(region=tuple(region))
        else:
            screenshot = pyautogui.screenshot()
        
        # Convert to bytes
        img_buffer = io.BytesIO()
        if format.lower() == "jpeg":
            screenshot = screenshot.convert('RGB')
        screenshot.save(img_buffer, format=format.upper(), quality=quality)
        img_buffer.seek(0)
        
        # Encode to base64
        img_base64 = base64.b64encode(img_buffer.getvalue()).decode('utf-8')
        
        return {
            "success": True,
            "format": format,
            "size": {"width": screenshot.width, "height": screenshot.height},
            "data": f"data:image/{format};base64,{img_base64}"
        }
    except Exception as e:
        logger.error(f"Screenshot failed: {e}")
        raise HTTPException(status_code=500, detail=f"Screenshot failed: {str(e)}")

@app.post("/execute", response_model=TaskResponse)
async def execute_task(
    task: TaskRequest,
    background_tasks: BackgroundTasks
):
    """Execute an automation task"""
    # Generate task ID
    app_state["task_counter"] += 1
    task_id = f"task_{app_state['task_counter']}"
    
    # Create task record
    task_record = {
        "id": task_id,
        "type": task.task_type,
        "parameters": task.parameters,
        "status": "pending",
        "created_at": datetime.utcnow().isoformat(),
        "result": None,
        "error": None
    }
    app_state["tasks"][task_id] = task_record
    
    if task.async_execution:
        # Execute in background
        background_tasks.add_task(execute_task_impl, task_id, task)
        task_record["status"] = "running"
    else:
        # Execute synchronously
        await execute_task_impl(task_id, task)
        task_record = app_state["tasks"][task_id]
    
    return TaskResponse(
        task_id=task_id,
        status=task_record["status"],
        result=task_record["result"],
        error=task_record["error"],
        created_at=task_record["created_at"],
        completed_at=task_record.get("completed_at")
    )

@app.get("/tasks/{task_id}", response_model=TaskResponse)
async def get_task_status(task_id: str):
    """Get status of a specific task"""
    if task_id not in app_state["tasks"]:
        raise HTTPException(status_code=404, detail="Task not found")
    
    task = app_state["tasks"][task_id]
    return TaskResponse(
        task_id=task_id,
        status=task["status"],
        result=task["result"],
        error=task["error"],
        created_at=task["created_at"],
        completed_at=task.get("completed_at")
    )

@app.get("/mouse/position")
async def get_mouse_position():
    """Get current mouse position"""
    try:
        x, y = pyautogui.position()
        return {"x": x, "y": y}
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to get mouse position: {str(e)}")

# Task execution implementation
async def execute_task_impl(task_id: str, task: TaskRequest):
    """Execute a task and update its status"""
    task_record = app_state["tasks"][task_id]
    
    try:
        task_record["status"] = "running"
        result = None
        
        # Execute based on task type
        if task.task_type == "screenshot":
            result = await execute_screenshot(task.parameters)
        elif task.task_type == "click":
            result = await execute_click(task.parameters)
        elif task.task_type == "type_text":
            result = await execute_type_text(task.parameters)
        elif task.task_type == "key_press":
            result = await execute_key_press(task.parameters)
        elif task.task_type == "mouse_move":
            result = await execute_mouse_move(task.parameters)
        elif task.task_type == "automation_sequence":
            result = await execute_automation_sequence(task.parameters)
        else:
            raise ValueError(f"Unknown task type: {task.task_type}")
        
        # Update task record
        task_record["status"] = "completed"
        task_record["result"] = result
        task_record["completed_at"] = datetime.utcnow().isoformat()
        
    except Exception as e:
        logger.error(f"Task {task_id} failed: {e}")
        task_record["status"] = "failed"
        task_record["error"] = str(e)
        task_record["completed_at"] = datetime.utcnow().isoformat()

# Task implementations
async def execute_screenshot(params: Dict[str, Any]) -> Dict[str, Any]:
    """Execute screenshot task"""
    region = params.get("region")
    if region:
        screenshot = pyautogui.screenshot(region=tuple(region))
    else:
        screenshot = pyautogui.screenshot()
    
    # Don't return the actual image data in task result
    return {
        "width": screenshot.width,
        "height": screenshot.height,
        "message": "Screenshot captured successfully"
    }

async def execute_click(params: Dict[str, Any]) -> Dict[str, Any]:
    """Execute mouse click"""
    x = params.get("x", 0)
    y = params.get("y", 0)
    button = params.get("button", "left")
    clicks = params.get("clicks", 1)
    
    pyautogui.click(x=x, y=y, button=button, clicks=clicks)
    return {"clicked": True, "position": {"x": x, "y": y}}

async def execute_type_text(params: Dict[str, Any]) -> Dict[str, Any]:
    """Type text"""
    text = params.get("text", "")
    interval = params.get("interval", 0.0)
    
    pyautogui.typewrite(text, interval=interval)
    return {"typed": True, "text_length": len(text)}

async def execute_key_press(params: Dict[str, Any]) -> Dict[str, Any]:
    """Press key(s)"""
    keys = params.get("keys", [])
    if isinstance(keys, str):
        keys = [keys]
    
    for key in keys:
        pyautogui.press(key)
    
    return {"pressed": True, "keys": keys}

async def execute_mouse_move(params: Dict[str, Any]) -> Dict[str, Any]:
    """Move mouse"""
    x = params.get("x", 0)
    y = params.get("y", 0)
    duration = params.get("duration", 0.0)
    relative = params.get("relative", False)
    
    if relative:
        pyautogui.moveRel(x, y, duration=duration)
    else:
        pyautogui.moveTo(x, y, duration=duration)
    
    new_x, new_y = pyautogui.position()
    return {"moved": True, "position": {"x": new_x, "y": new_y}}

async def execute_automation_sequence(params: Dict[str, Any]) -> Dict[str, Any]:
    """Execute a sequence of automation steps"""
    steps = params.get("steps", [])
    results = []
    
    for i, step in enumerate(steps):
        step_type = step.get("type")
        step_params = step.get("parameters", {})
        
        try:
            if step_type == "click":
                result = await execute_click(step_params)
            elif step_type == "type_text":
                result = await execute_type_text(step_params)
            elif step_type == "key_press":
                result = await execute_key_press(step_params)
            elif step_type == "wait":
                await asyncio.sleep(step_params.get("seconds", 1))
                result = {"waited": True}
            else:
                result = {"error": f"Unknown step type: {step_type}"}
            
            results.append({"step": i, "result": result})
        except Exception as e:
            results.append({"step": i, "error": str(e)})
            break
    
    return {"steps_executed": len(results), "results": results}

# Error handlers
@app.exception_handler(Exception)
async def general_exception_handler(request, exc):
    logger.error(f"Unhandled exception: {exc}")
    return JSONResponse(
        status_code=500,
        content={"detail": "Internal server error", "error": str(exc)}
    )

# Main entry point
if __name__ == "__main__":
    host = os.environ.get("AGENTS2_API_HOST", "0.0.0.0")
    port = int(os.environ.get("AGENTS2_API_PORT", "4113"))
    
    logger.info(f"Starting Agent S2 API server on {host}:{port}")
    uvicorn.run(app, host=host, port=port, log_level="info")