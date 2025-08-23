"""AI-driven routes for Agent S2 API"""

import os
import logging
from datetime import datetime
from fastapi import APIRouter, Request, HTTPException, BackgroundTasks

from ..models.requests import AICommandRequest, AIPlanRequest, AIAnalyzeRequest, AIActionRequest
from ..models.responses import AICommandResponse, AIPlanResponse, AIAnalyzeResponse, AIActionResponse, TaskResponse

logger = logging.getLogger(__name__)
router = APIRouter()



def get_ai_handler(request: Request):
    """Get AI handler from app state"""
    app_state = request.app.state.app_state
    ai_handler = app_state.get("ai_handler")
    
    if not ai_handler or not ai_handler.initialized:
        # Get detailed error information
        init_error = app_state.get("ai_init_error", {})
        
        # Build detailed error response
        error_detail = {
            "error": "AI service not available",
            "reason": init_error.get("details", "Unknown initialization failure"),
            "category": init_error.get("category", "unknown"),
            "suggestions": init_error.get("suggestions", [
                "Check AI service configuration",
                "Verify required services are running"
            ]),
            "alternatives": {
                "endpoints": {
                    "screenshot": "/screenshot - Capture screen images",
                    "mouse": "/mouse/click, /mouse/move - Control mouse",
                    "keyboard": "/keyboard/type, /keyboard/press - Control keyboard",
                    "tasks": "/tasks - View task status"
                },
                "documentation": "See /docs for API documentation"
            },
            "diagnostics": "Use GET /ai/diagnostics for detailed troubleshooting"
        }
        
        # Include timestamp if available
        if "timestamp" in init_error:
            error_detail["error_timestamp"] = init_error["timestamp"]
        
        raise HTTPException(
            status_code=503,
            detail=error_detail,
            headers={
                "X-AI-Available": "false",
                "X-AI-Error-Category": init_error.get("category", "unknown"),
                "X-Alternative-Endpoints": "/screenshot,/mouse,/keyboard,/tasks"
            }
        )
    return ai_handler




@router.post("/action", response_model=AIActionResponse)
async def ai_action(request: AIActionRequest, req: Request):
    """Execute an AI-driven action
    
    Args:
        request: AI action parameters
        
    Returns:
        Action result with AI reasoning
    """
    ai_handler = get_ai_handler(req)
    
    try:
        # Include security config in context if provided
        context = request.context or {}
        if request.security_config:
            context["security_config"] = request.security_config.dict()
            
        result = await ai_handler.execute_action(
            task=request.task,
            screenshot=request.screenshot,
            context=context
        )
        
        
        # Format debug_info for management script compatibility
        debug_info = result.get("debug_info", {})
        formatted_debug_info = {}
        
        if "from_planner" in debug_info:
            planner_debug = debug_info["from_planner"]
            formatted_debug_info["from_handler"] = {
                "system_prompt": planner_debug.get("system_prompt"),
                "user_prompt": planner_debug.get("user_prompt"),
                "screenshot_included": planner_debug.get("screenshot_included"),
                "context": planner_debug.get("context")
            }
        
        # Include other debug info
        for key, value in debug_info.items():
            if key != "from_planner":
                formatted_debug_info[key] = value
        
        return AIActionResponse(
            success=result.get("success", False),
            task=request.task,
            summary=result.get("summary", "Action completed"),
            actions_taken=result.get("actions_taken", []),
            reasoning=result.get("reasoning"),
            error=result.get("error"),
            debug_info=formatted_debug_info
        )
    except Exception as e:
        logger.error(f"AI action failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/command", response_model=TaskResponse)
async def execute_ai_command(
    request: AICommandRequest,
    req: Request,
    background_tasks: BackgroundTasks
):
    """Execute a natural language command using AI
    
    Args:
        request: AI command parameters
        
    Returns:
        Task response with execution details
    """
    ai_handler = get_ai_handler(req)
    app_state = req.app.state.app_state
    
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
        background_tasks.add_task(
            ai_handler.execute_command_async,
            task_id,
            request.command,
            request.context,
            task_record
        )
        task_record["status"] = "running"
    else:
        # Execute synchronously
        try:
            result = await ai_handler.execute_command(
                command=request.command,
                context=request.context
            )
            task_record["status"] = "completed"
            task_record["result"] = result
            task_record["completed_at"] = datetime.utcnow().isoformat()
        except Exception as e:
            task_record["status"] = "failed"
            task_record["error"] = str(e)
            task_record["completed_at"] = datetime.utcnow().isoformat()
    
    return TaskResponse(
        task_id=task_id,
        status=task_record["status"],
        result=task_record["result"],
        error=task_record["error"],
        created_at=task_record["created_at"],
        completed_at=task_record.get("completed_at")
    )


@router.post("/plan", response_model=AIPlanResponse)
async def generate_ai_plan(request: AIPlanRequest, req: Request):
    """Generate a multi-step plan for achieving a goal
    
    Args:
        request: Planning parameters
        
    Returns:
        Generated plan with steps
    """
    ai_handler = get_ai_handler(req)
    
    try:
        plan = await ai_handler.generate_plan(
            goal=request.goal,
            constraints=request.constraints
        )
        
        return AIPlanResponse(
            goal=request.goal,
            constraints=request.constraints or [],
            steps=plan["steps"],
            estimated_duration=plan.get("estimated_duration", "varies"),
            complexity=plan.get("complexity", "medium")
        )
    except Exception as e:
        logger.error(f"AI planning failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/analyze", response_model=AIAnalyzeResponse)
async def analyze_screen(request: AIAnalyzeRequest, req: Request):
    """Analyze the current screen using AI
    
    Args:
        request: Analysis parameters
        
    Returns:
        Screen analysis results
    """
    ai_handler = get_ai_handler(req)
    
    try:
        analysis = await ai_handler.analyze_screen(
            question=request.question,
            screenshot=request.screenshot
        )
        
        return AIAnalyzeResponse(
            screen_size=analysis["screen_size"],
            question=request.question,
            analysis=analysis["analysis"],
            elements_detected=analysis.get("elements_detected", []),
            suggested_actions=analysis.get("suggested_actions", [])
        )
    except Exception as e:
        logger.error(f"Screen analysis failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/status")
async def get_ai_status(req: Request):
    """Get AI service status
    
    Returns:
        AI service status information
    """
    app_state = req.app.state.app_state
    ai_handler = app_state.get("ai_handler")
    
    if not ai_handler:
        return {
            "available": False,
            "initialized": False,
            "enabled": False,
            "message": "AI handler not configured"
        }
        
    return {
        "available": True,
        "initialized": ai_handler.initialized,
        "enabled": ai_handler.enabled,
        "provider": ai_handler.provider if ai_handler.initialized else None,
        "model": ai_handler.model if ai_handler.initialized else None,
        "capabilities": ai_handler.get_capabilities() if ai_handler.initialized else []
    }


@router.get("/diagnostics")
async def ai_diagnostics(req: Request):
    """Comprehensive AI service diagnostics for troubleshooting
    
    Returns:
        Detailed diagnostic information including configuration, 
        connectivity tests, and troubleshooting suggestions
    """
    from ...config import Config
    import requests
    
    try:
        app_state = req.app.state.app_state
        ai_handler = app_state.get("ai_handler")
        init_error = app_state.get("ai_init_error", {})
        
        # Basic status
        diagnostics = {
            "service_status": {
                "ai_enabled": Config.AI_ENABLED,
                "handler_created": ai_handler is not None,
                "handler_initialized": ai_handler.initialized if ai_handler else False,
                "initialization_attempted": ai_handler is not None or init_error.get("timestamp") is not None
            },
            "configuration": {
                "enable_ai": Config.AI_ENABLED,
                "api_url": Config.AI_API_URL,
                "model": Config.AI_MODEL,
                "timeout": Config.AI_TIMEOUT,
                "provider": ai_handler.provider if ai_handler and hasattr(ai_handler, 'provider') else "ollama"
            },
            "initialization_error": init_error if init_error else None
        }
    except Exception as e:
        # If we can't even get basic status, return minimal diagnostics
        return {
            "error": "Failed to gather basic diagnostics",
            "error_detail": str(e),
            "quick_fixes": [
                "Restart Agent-S2: docker restart agent-s2",
                "Check logs: docker logs agent-s2 --tail 50",
                "Verify Ollama is running: curl http://localhost:11434/api/tags"
            ]
        }
    
    # Connectivity tests
    connectivity_results = []
    
    # Test Ollama connectivity
    try:
        ollama_base_url = Config.AI_API_URL.split("/api/")[0] if "/api/" in Config.AI_API_URL else "http://localhost:11434"
        
        # Test 1: Basic connectivity
        try:
            response = requests.get(f"{ollama_base_url}/api/tags", timeout=5)
            connectivity_results.append({
                "test": "Ollama API connectivity",
                "endpoint": f"{ollama_base_url}/api/tags",
                "status": "success" if response.status_code == 200 else "failed",
                "status_code": response.status_code,
            "response_time_ms": int(response.elapsed.total_seconds() * 1000)
        })
        
            # If successful, check models
            if response.status_code == 200:
                models = response.json().get("models", [])
                model_names = [m["name"] for m in models]
                connectivity_results.append({
                    "test": "Available models",
                    "status": "success",
                    "models": model_names,
                    "count": len(models),
                    "has_vision_model": any("vision" in m or "llava" in m for m in model_names)
                })
        except Exception as e:
            connectivity_results.append({
                "test": "Ollama API connectivity", 
                "endpoint": f"{ollama_base_url}/api/tags",
                "status": "error",
                "error": str(e),
                "suggestion": "Check if Ollama is running: ollama serve"
            })
        
        # Test 2: Network reachability (different approaches)
        alternative_urls = [
            "http://localhost:11434",
            "http://ollama:11434",  # Docker service name
            "http://host.docker.internal:11434"  # Docker Desktop
        ]
        
        for url in alternative_urls:
            if url != ollama_base_url:
                try:
                    response = requests.get(f"{url}/api/tags", timeout=1)
                    if response.status_code == 200:
                        connectivity_results.append({
                            "test": f"Alternative endpoint",
                            "endpoint": url,
                            "status": "reachable",
                            "suggestion": f"Consider using AGENTS2_OLLAMA_BASE_URL={url}"
                        })
                except:
                    pass  # Silent fail for alternatives
    except Exception as e:
        connectivity_results.append({
            "test": "Connectivity tests",
            "status": "error",
            "error": str(e),
            "suggestion": "Check network connectivity and firewall settings"
        })
    
    diagnostics["connectivity_tests"] = connectivity_results
    
    # Generate recommendations
    recommendations = []
    
    try:
        if not Config.AI_ENABLED:
            recommendations.append({
                "issue": "AI is disabled",
                "solution": "Set environment variable: AGENTS2_ENABLE_AI=true",
                "priority": "high"
            })
        
        if init_error and init_error.get("category") == "connection_failed":
            recommendations.append({
                "issue": "Cannot connect to Ollama",
                "solution": "Install and start Ollama: curl -fsSL https://ollama.com/install.sh | sh && ollama serve",
                "priority": "high"
            })
            recommendations.append({
                "issue": "Docker network isolation",
                "solution": "If running in Docker, use --network host or set OLLAMA_HOST=0.0.0.0 on the Ollama container",
                "priority": "medium"
            })
        
        if init_error and init_error.get("category") == "model_not_found":
            recommendations.append({
                "issue": f"Model {Config.AI_MODEL} not found",
                "solution": f"Pull the model: ollama pull {Config.AI_MODEL}",
                "priority": "high"
        })
    
        # Check if any alternative endpoints were found
        alt_endpoints = [r for r in connectivity_results if r.get("test") == "Alternative endpoint" and r.get("status") == "reachable"]
        if alt_endpoints and not (ai_handler and ai_handler.initialized):
            recommendations.append({
                "issue": "Ollama found at different endpoint",
                "solution": alt_endpoints[0]["suggestion"],
                "priority": "high"
            })
    except Exception as e:
        recommendations.append({
            "issue": "Error generating recommendations",
            "solution": "Check diagnostics error details",
            "priority": "low",
            "error": str(e)
        })
    
    diagnostics["recommendations"] = recommendations
    
    # Quick start guide
    diagnostics["quick_start"] = {
        "install_ollama": "curl -fsSL https://ollama.com/install.sh | sh",
        "start_ollama": "ollama serve",
        "pull_vision_model": "ollama pull llama3.2-vision:11b",
        "configure_agent_s2": "export AGENTS2_ENABLE_AI=true && export AGENTS2_LLM_MODEL=llama3.2-vision:11b",
        "test_ai": "curl -X POST http://localhost:4113/ai/action -H 'Content-Type: application/json' -d '{\"task\": \"test AI\"}'",
        "verify_status": "curl http://localhost:4113/ai/status"
    }
    
    # Environment info for debugging
    diagnostics["environment"] = {
        "container_mode": os.environ.get("CONTAINER_MODE", "unknown"),
        "display": os.environ.get("DISPLAY", "not set"),
        "hostname": os.environ.get("HOSTNAME", "unknown")
    }
    
    return diagnostics


