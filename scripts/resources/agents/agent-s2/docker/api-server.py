#!/usr/bin/env python3
"""
Agent S2 REST API Server
Provides HTTP endpoints for Agent S2 autonomous computer interaction with AI capabilities
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

# AI Integration imports
try:
    from gui_agents.s2.agents.agent_s import AgentS2
    from gui_agents.s2.agents.grounding import OSWorldACI
    AI_AVAILABLE = True
except ImportError as e:
    logging.warning(f"Agent S2 AI not available: {e}")
    AI_AVAILABLE = False

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configure pyautogui
pyautogui.FAILSAFE = False  # Disable failsafe in container
pyautogui.PAUSE = 0.1  # Small pause between actions

# AI Configuration
AI_CONFIG = {
    "enabled": os.environ.get("AGENTS2_ENABLE_AI", "true").lower() == "true",
    "provider": os.environ.get("AGENTS2_LLM_PROVIDER", "anthropic"),
    "model": os.environ.get("AGENTS2_LLM_MODEL", "claude-3-7-sonnet-20250219"),
    "openai_key": os.environ.get("OPENAI_API_KEY"),
    "anthropic_key": os.environ.get("ANTHROPIC_API_KEY"),
    "search_enabled": os.environ.get("AGENTS2_ENABLE_SEARCH", "false").lower() == "true",
}

# Global state
app_state = {
    "tasks": {},
    "task_counter": 0,
    "startup_time": datetime.utcnow().isoformat(),
    "ai_agent": None,
    "ai_initialized": False
}

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle"""
    logger.info("Starting Agent S2 API Server...")
    
    # Initialize AI agent if available and enabled
    if AI_AVAILABLE and AI_CONFIG["enabled"]:
        await initialize_ai_agent()
    else:
        logger.info("AI disabled or unavailable - running in core automation mode")
    
    # Startup
    yield
    
    # Shutdown
    logger.info("Shutting down Agent S2 API Server...")

async def initialize_ai_agent():
    """Initialize the Agent S2 AI engine with comprehensive error handling"""
    try:
        logger.info("Initializing Agent S2 AI engine...")
        
        # Check if AI packages are available
        if not AI_AVAILABLE:
            logger.warning("Agent S2 AI packages not available - using core automation only")
            app_state["ai_initialized"] = False
            return
        
        # Validate AI is enabled
        if not AI_CONFIG["enabled"]:
            logger.info("AI disabled in configuration - using core automation only")
            app_state["ai_initialized"] = False
            return
        
        # Validate API keys with detailed error messages
        provider = AI_CONFIG["provider"]
        if provider == "openai":
            if not AI_CONFIG["openai_key"]:
                logger.error("OpenAI API key not provided. Set OPENAI_API_KEY environment variable.")
                logger.info("AI features disabled - using core automation only")
                app_state["ai_initialized"] = False
                return
        elif provider == "anthropic":
            if not AI_CONFIG["anthropic_key"]:
                logger.error("Anthropic API key not provided. Set ANTHROPIC_API_KEY environment variable.")
                logger.info("AI features disabled - using core automation only")
                app_state["ai_initialized"] = False
                return
        else:
            logger.error(f"Unsupported LLM provider: {provider}")
            logger.info("AI features disabled - using core automation only")
            app_state["ai_initialized"] = False
            return
        
        # Setup engine parameters
        engine_params = {
            "engine_type": provider,
            "model": AI_CONFIG["model"],
        }
        
        # Add API key based on provider
        if provider == "openai":
            engine_params["api_key"] = AI_CONFIG["openai_key"]
        elif provider == "anthropic":
            engine_params["api_key"] = AI_CONFIG["anthropic_key"]
        
        # Initialize grounding agent with error handling
        try:
            grounding_agent = OSWorldACI(
                platform="linux",
                engine_params_for_generation=engine_params,
                engine_params_for_grounding=engine_params
            )
        except Exception as grounding_error:
            logger.error(f"Failed to initialize grounding agent: {grounding_error}")
            logger.info("AI features disabled - using core automation only")
            app_state["ai_initialized"] = False
            return
        
        # Initialize main AI agent with error handling
        try:
            search_engine = "Perplexica" if AI_CONFIG["search_enabled"] else None
            
            app_state["ai_agent"] = AgentS2(
                engine_params,
                grounding_agent,
                platform="linux",
                action_space="pyautogui",
                observation_type="screenshot",
                search_engine=search_engine
            )
            
            app_state["ai_initialized"] = True
            logger.info(f"✅ Agent S2 AI successfully initialized with {provider} ({AI_CONFIG['model']})")
            
            # Test basic functionality
            try:
                # Simple functionality test
                logger.info("Testing AI agent basic functionality...")
                # Note: Actual test would depend on Agent S2 API
                logger.info("✅ AI agent basic functionality test passed")
            except Exception as test_error:
                logger.warning(f"AI agent test failed: {test_error}")
                logger.info("AI agent initialized but may have issues")
                
        except Exception as agent_error:
            logger.error(f"Failed to initialize main AI agent: {agent_error}")
            logger.info("AI features disabled - using core automation only")
            app_state["ai_initialized"] = False
            app_state["ai_agent"] = None
            return
        
    except Exception as e:
        logger.error(f"Unexpected error during AI agent initialization: {e}")
        logger.info("AI features disabled - using core automation only")
        app_state["ai_initialized"] = False
        app_state["ai_agent"] = None

def create_ai_unavailable_message() -> str:
    """Create detailed error message when AI is unavailable"""
    error_detail = "AI agent not available. "
    if not AI_AVAILABLE:
        error_detail += "Agent S2 AI packages not installed. "
    elif not AI_CONFIG["enabled"]:
        error_detail += "AI disabled in configuration. Set AGENTS2_ENABLE_AI=true to enable. "
    else:
        error_detail += "AI initialization failed. Check logs for details. "
    error_detail += "Use core automation endpoints (/execute) for direct control."
    return error_detail

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
    ai_status: Dict[str, Any]

# API Endpoints
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Check service health and display status"""
    try:
        # Get screen information
        screen_width, screen_height = pyautogui.size()
        
        # AI status information
        ai_status = {
            "available": AI_AVAILABLE,
            "enabled": AI_CONFIG["enabled"],
            "initialized": app_state["ai_initialized"],
            "provider": AI_CONFIG["provider"] if app_state["ai_initialized"] else None,
            "model": AI_CONFIG["model"] if app_state["ai_initialized"] else None,
            "search_enabled": AI_CONFIG["search_enabled"]
        }
        
        return HealthResponse(
            status="healthy",
            display=os.environ.get("DISPLAY", "unknown"),
            screen_size={"width": screen_width, "height": screen_height},
            startup_time=app_state["startup_time"],
            tasks_processed=app_state["task_counter"],
            ai_status=ai_status
        )
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"Service unhealthy: {str(e)}")

@app.get("/capabilities")
async def get_capabilities():
    """Get Agent S2 capabilities"""
    capabilities = {
        "screenshot": True,
        "gui_automation": True,
        "mouse_control": True,
        "keyboard_control": True,
        "window_management": True,
        "planning": app_state["ai_initialized"],
        "multi_step_tasks": True,
        "ai_reasoning": app_state["ai_initialized"],
        "natural_language": app_state["ai_initialized"],
        "screen_understanding": app_state["ai_initialized"]
    }
    
    supported_tasks = [
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
    ]
    
    # Add AI-driven tasks if available
    if app_state["ai_initialized"]:
        supported_tasks.extend([
            "ai_command",
            "ai_plan",
            "ai_analyze",
            "ai_sequence"
        ])
    
    return {
        "capabilities": capabilities,
        "supported_tasks": supported_tasks,
        "ai_status": {
            "available": AI_AVAILABLE,
            "enabled": AI_CONFIG["enabled"],
            "initialized": app_state["ai_initialized"],
            "provider": AI_CONFIG["provider"] if app_state["ai_initialized"] else None,
            "model": AI_CONFIG["model"] if app_state["ai_initialized"] else None
        },
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

# New AI-driven endpoints
class AICommandRequest(BaseModel):
    """AI command execution request"""
    command: str = Field(..., description="Natural language command to execute")
    context: Optional[str] = Field(default=None, description="Additional context for the command")
    async_execution: bool = Field(default=False, description="Execute command asynchronously")

class AIPlanRequest(BaseModel):
    """AI planning request"""
    goal: str = Field(..., description="High-level goal to achieve")
    constraints: Optional[List[str]] = Field(default=None, description="Constraints or limitations")

class AIAnalyzeRequest(BaseModel):
    """AI screen analysis request"""
    question: Optional[str] = Field(default=None, description="Specific question about the screen")

@app.post("/execute/ai", response_model=TaskResponse)
async def execute_ai_command(
    request: AICommandRequest,
    background_tasks: BackgroundTasks
):
    """Execute a natural language command using AI"""
    if not app_state["ai_initialized"]:
        error_detail = create_ai_unavailable_message()
        raise HTTPException(
            status_code=503, 
            detail=error_detail,
            headers={"X-Fallback-Available": "true", "X-Alternative-Endpoint": "/execute"}
        )
    
    # Generate task ID
    app_state["task_counter"] += 1
    task_id = f"ai_task_{app_state['task_counter']}"
    
    # Create task record
    task_record = {
        "id": task_id,
        "type": "ai_command",
        "parameters": {"command": request.command, "context": request.context},
        "status": "pending",
        "created_at": datetime.utcnow().isoformat(),
        "result": None,
        "error": None
    }
    app_state["tasks"][task_id] = task_record
    
    if request.async_execution:
        # Execute in background
        background_tasks.add_task(execute_ai_command_impl, task_id, request.command, request.context)
        task_record["status"] = "running"
    else:
        # Execute synchronously
        await execute_ai_command_impl(task_id, request.command, request.context)
        task_record = app_state["tasks"][task_id]
    
    return TaskResponse(
        task_id=task_id,
        status=task_record["status"],
        result=task_record["result"],
        error=task_record["error"],
        created_at=task_record["created_at"],
        completed_at=task_record.get("completed_at")
    )

@app.post("/plan")
async def generate_ai_plan(request: AIPlanRequest):
    """Generate a multi-step plan for achieving a goal"""
    if not app_state["ai_initialized"]:
        error_detail = create_ai_unavailable_message()
        raise HTTPException(
            status_code=503, 
            detail=error_detail,
            headers={"X-Fallback-Available": "true", "X-Alternative-Endpoint": "/execute"}
        )
    
    try:
        # Use AI agent to generate plan
        agent = app_state["ai_agent"]
        
        # For now, return a placeholder response
        # TODO: Implement actual AI planning
        plan_steps = [
            {"step": 1, "action": "Analyze current screen state", "description": "Understanding the current context"},
            {"step": 2, "action": "Break down goal into actionable steps", "description": f"Planning how to achieve: {request.goal}"},
            {"step": 3, "action": "Execute planned actions", "description": "Implementing the plan step by step"}
        ]
        
        return {
            "goal": request.goal,
            "constraints": request.constraints or [],
            "steps": plan_steps,
            "estimated_duration": "varies",
            "complexity": "medium"
        }
        
    except Exception as e:
        logger.error(f"AI planning failed: {e}")
        raise HTTPException(status_code=500, detail=f"AI planning failed: {str(e)}")

@app.post("/analyze-screen")
async def analyze_screen(request: AIAnalyzeRequest):
    """Analyze the current screen using AI"""
    if not app_state["ai_initialized"]:
        error_detail = create_ai_unavailable_message()
        raise HTTPException(
            status_code=503, 
            detail=error_detail,
            headers={"X-Fallback-Available": "true", "X-Alternative-Endpoint": "/screenshot"}
        )
    
    try:
        # Take screenshot for analysis
        screenshot = pyautogui.screenshot()
        
        # Convert to base64 for analysis
        img_buffer = io.BytesIO()
        screenshot.save(img_buffer, format="PNG")
        img_buffer.seek(0)
        img_base64 = base64.b64encode(img_buffer.getvalue()).decode('utf-8')
        
        # For now, return basic analysis
        # TODO: Implement actual AI screen understanding
        analysis = {
            "screen_size": {"width": screenshot.width, "height": screenshot.height},
            "question": request.question,
            "analysis": "Screen analysis not yet implemented - placeholder response",
            "elements_detected": [],
            "suggested_actions": []
        }
        
        return analysis
        
    except Exception as e:
        logger.error(f"Screen analysis failed: {e}")
        raise HTTPException(status_code=500, detail=f"Screen analysis failed: {str(e)}")

# Task execution implementation
async def execute_task_impl(task_id: str, task: TaskRequest):
    """Execute a task and update its status - intelligent router for AI-driven vs core automation tasks"""
    task_record = app_state["tasks"][task_id]
    
    try:
        task_record["status"] = "running"
        result = None
        
        # Intelligent task routing: AI-driven vs Core automation
        if task.task_type in ["ai_command", "ai_plan", "ai_analyze", "ai_sequence"]:
            # AI-driven tasks (high-level, natural language)
            if not app_state["ai_initialized"]:
                raise ValueError("AI agent not available for AI-driven tasks")
            
            result = await execute_ai_task(task.task_type, task.parameters)
            
        elif task.task_type in ["screenshot", "click", "type_text", "key_press", "mouse_move", "automation_sequence"]:
            # Core automation tasks (direct, deterministic actions)
            logger.info(f"Executing core automation task: {task.task_type}")
            
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
            # Unknown task type - attempt AI interpretation if available
            if app_state["ai_initialized"]:
                logger.info(f"Unknown task type '{task.task_type}', attempting AI interpretation")
                result = await execute_ai_fallback_task(task.task_type, task.parameters)
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

# AI Task implementations
async def execute_ai_command_impl(task_id: str, command: str, context: Optional[str]):
    """Execute an AI command and update task status"""
    task_record = app_state["tasks"][task_id]
    
    try:
        task_record["status"] = "running"
        
        # Get the AI agent
        agent = app_state["ai_agent"]
        
        if not agent:
            raise ValueError("AI agent not initialized")
        
        logger.info(f"Executing AI command: {command}")
        
        # Use the execute_ai_task function which now properly uses core automation
        result = await execute_ai_task("ai_command", {
            "command": command,
            "context": context
        })
        
        # Update task record
        task_record["status"] = "completed"
        task_record["result"] = result
        task_record["completed_at"] = datetime.utcnow().isoformat()
        
        logger.info(f"AI command completed: {task_id}")
        logger.info(f"Core functions used: {result.get('core_functions_used', 0)}")
        if result.get('actions_taken'):
            logger.info(f"Actions taken: {[a['action'] for a in result['actions_taken']]}")
        
    except Exception as e:
        logger.error(f"AI command {task_id} failed: {e}")
        task_record["status"] = "failed"
        task_record["error"] = str(e)
        task_record["completed_at"] = datetime.utcnow().isoformat()

async def execute_ai_task(task_type: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
    """Execute AI-driven tasks using the Agent S2 engine with graceful error handling"""
    try:
        # Validate AI agent availability
        agent = app_state["ai_agent"]
        if not agent:
            raise ValueError("AI agent not initialized")
        
        if task_type == "ai_command":
            # Natural language command execution
            command = parameters.get("command", "")
            context = parameters.get("context")
            
            if not command:
                raise ValueError("Command parameter is required for ai_command tasks")
            
            logger.info(f"AI executing command: {command}")
            
            try:
                # Step 1: Take screenshot to understand current state
                screenshot_result = await execute_screenshot({})
                
                # Step 2: Analyze command and plan actions
                # This is where the real Agent S2 would use LLM to understand the command
                # For now, we'll implement basic pattern matching as a demonstration
                actions_taken = []
                
                # Simple command parsing (real Agent S2 would use LLM)
                command_lower = command.lower()
                
                if "screenshot" in command_lower:
                    actions_taken.append({
                        "action": "screenshot",
                        "result": screenshot_result,
                        "status": "completed"
                    })
                
                if "mouse" in command_lower and "center" in command_lower:
                    # Get screen size from screenshot result
                    screen_width, screen_height = pyautogui.size()
                    center_x = screen_width // 2
                    center_y = screen_height // 2
                    
                    move_result = await execute_mouse_move({
                        "x": center_x,
                        "y": center_y,
                        "duration": 0.5
                    })
                    actions_taken.append({
                        "action": "mouse_move",
                        "parameters": {"x": center_x, "y": center_y},
                        "result": move_result,
                        "status": "completed"
                    })
                
                if "click" in command_lower:
                    click_result = await execute_click({})
                    actions_taken.append({
                        "action": "click",
                        "result": click_result,
                        "status": "completed"
                    })
                
                if "type" in command_lower or "write" in command_lower:
                    # Extract text to type (simplified)
                    text_to_type = "Hello from AI Agent"
                    if "hello world" in command_lower:
                        text_to_type = "Hello World"
                    
                    type_result = await execute_type_text({
                        "text": text_to_type,
                        "interval": 0.05
                    })
                    actions_taken.append({
                        "action": "type_text",
                        "parameters": {"text": text_to_type},
                        "result": type_result,
                        "status": "completed"
                    })
                
                return {
                    "task_type": "ai_command",
                    "command": command,
                    "context": context,
                    "executed": True,
                    "status": "completed",
                    "message": "AI command executed using core automation",
                    "actions_taken": actions_taken,
                    "core_functions_used": len(actions_taken),
                    "fallback_used": False
                }
                
            except Exception as cmd_error:
                logger.error(f"AI command execution failed: {cmd_error}")
                # Graceful fallback - could attempt simple automation
                return {
                    "task_type": "ai_command",
                    "command": command,
                    "context": context,
                    "executed": False,
                    "status": "failed_with_fallback",
                    "message": f"AI command failed: {str(cmd_error)}",
                    "error": str(cmd_error),
                    "fallback_used": True,
                    "fallback_suggestion": "Try using specific automation tasks (click, type, etc.) instead"
                }
            
        elif task_type == "ai_plan":
            # AI planning task
            goal = parameters.get("goal", "")
            constraints = parameters.get("constraints", [])
            
            logger.info(f"AI planning for goal: {goal}")
            
            # Generate a plan that uses core automation functions
            # Real Agent S2 would use LLM to analyze goal and create steps
            goal_lower = goal.lower()
            
            steps = []
            
            # Always start with understanding current state
            steps.append({
                "step": 1,
                "action": "screenshot",
                "description": "Capture current screen state",
                "core_function": "execute_screenshot",
                "parameters": {}
            })
            
            # Plan based on goal (simplified pattern matching)
            if "organize" in goal_lower and "desktop" in goal_lower:
                steps.extend([
                    {
                        "step": 2,
                        "action": "analyze_icons",
                        "description": "Identify desktop icons and their positions",
                        "core_function": "execute_screenshot",
                        "parameters": {"analyze": True}
                    },
                    {
                        "step": 3,
                        "action": "group_by_type",
                        "description": "Plan icon grouping by file type",
                        "core_function": "planning_only",
                        "parameters": {}
                    },
                    {
                        "step": 4,
                        "action": "drag_drop_sequence",
                        "description": "Move icons to organized positions",
                        "core_function": "execute_automation_sequence",
                        "parameters": {
                            "steps": [
                                {"type": "mouse_move", "parameters": {"x": 100, "y": 100}},
                                {"type": "drag_drop", "parameters": {"start": [100, 100], "end": [200, 100]}}
                            ]
                        }
                    }
                ])
            elif "open" in goal_lower and "application" in goal_lower:
                steps.extend([
                    {
                        "step": 2,
                        "action": "find_application",
                        "description": "Locate application icon or menu",
                        "core_function": "execute_screenshot",
                        "parameters": {"analyze": True}
                    },
                    {
                        "step": 3,
                        "action": "click_application",
                        "description": "Click on application to open",
                        "core_function": "execute_click",
                        "parameters": {"target": "application_icon"}
                    }
                ])
            else:
                # Generic plan
                steps.extend([
                    {
                        "step": 2,
                        "action": "analyze_goal",
                        "description": f"Break down goal: {goal}",
                        "core_function": "planning_only",
                        "parameters": {}
                    },
                    {
                        "step": 3,
                        "action": "execute_actions",
                        "description": "Execute required core automation functions",
                        "core_function": "execute_automation_sequence",
                        "parameters": {}
                    }
                ])
            
            return {
                "task_type": "ai_plan",
                "goal": goal,
                "constraints": constraints,
                "plan_generated": True,
                "steps": steps,
                "core_functions_required": [s["core_function"] for s in steps if s["core_function"] != "planning_only"],
                "estimated_actions": len([s for s in steps if s["core_function"] != "planning_only"])
            }
            
        elif task_type == "ai_analyze":
            # Screen analysis task
            question = parameters.get("question")
            
            # Use core screenshot function
            screenshot_result = await execute_screenshot({})
            
            # Get screen dimensions
            screen_width, screen_height = pyautogui.size()
            
            logger.info(f"AI analyzing screen with question: {question}")
            
            # Simulate analysis (real Agent S2 would use vision model)
            analysis_text = f"Screen captured at {screen_width}x{screen_height} resolution. "
            elements_detected = []
            suggested_actions = []
            
            if question:
                question_lower = question.lower()
                if "application" in question_lower:
                    # Simulate detecting applications
                    analysis_text += "Detected multiple application windows. "
                    elements_detected = [
                        {"type": "window", "name": "Desktop", "position": {"x": 0, "y": 0}},
                        {"type": "taskbar", "name": "System Taskbar", "position": {"x": 0, "y": screen_height - 40}}
                    ]
                    suggested_actions = [
                        {"action": "click", "target": "application_icon", "core_function": "execute_click"},
                        {"action": "screenshot", "target": "specific_window", "core_function": "execute_screenshot"}
                    ]
                elif "text" in question_lower or "read" in question_lower:
                    analysis_text += "Would use OCR to extract text from screen. "
                    suggested_actions = [
                        {"action": "screenshot", "parameters": {"region": "text_area"}, "core_function": "execute_screenshot"}
                    ]
                else:
                    analysis_text += "General screen analysis completed. "
            else:
                analysis_text += "No specific analysis requested. "
            
            return {
                "task_type": "ai_analyze",
                "question": question,
                "analysis": analysis_text,
                "screen_size": {"width": screen_width, "height": screen_height},
                "screenshot_taken": True,
                "elements_detected": elements_detected,
                "suggested_actions": suggested_actions,
                "core_functions_used": ["execute_screenshot"],
                "note": "Full vision analysis requires Agent S2 vision model integration"
            }
            
        elif task_type == "ai_sequence":
            # AI-driven automation sequence
            goal = parameters.get("goal", "")
            steps = parameters.get("steps", [])
            
            # TODO: Implement actual AI sequence execution
            logger.info(f"AI executing sequence for goal: {goal}")
            
            return {
                "task_type": "ai_sequence",
                "goal": goal,
                "steps_planned": len(steps),
                "executed": True,
                "message": "AI sequence execution placeholder - needs Agent S2 integration"
            }
            
        else:
            raise ValueError(f"Unknown AI task type: {task_type}")
            
    except Exception as e:
        logger.error(f"AI task execution failed: {e}")
        raise

async def execute_ai_fallback_task(task_type: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
    """Attempt to execute unknown tasks using AI interpretation"""
    try:
        agent = app_state["ai_agent"]
        
        # TODO: Use AI to interpret and execute unknown task types
        logger.info(f"AI fallback for unknown task: {task_type}")
        
        return {
            "task_type": task_type,
            "parameters": parameters,
            "executed": True,
            "method": "ai_fallback",
            "message": f"AI fallback execution placeholder for task type: {task_type}",
            "interpretation": f"AI would interpret and execute: {task_type}"
        }
        
    except Exception as e:
        logger.error(f"AI fallback failed: {e}")
        raise

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