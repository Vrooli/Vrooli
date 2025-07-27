"""AI-driven routes for Agent S2 API"""

import logging
from fastapi import APIRouter, Request, HTTPException, BackgroundTasks

from ..models.requests import AICommandRequest, AIPlanRequest, AIAnalyzeRequest, AIActionRequest
from ..models.responses import AICommandResponse, AIPlanResponse, AIAnalyzeResponse, AIActionResponse, TaskResponse

logger = logging.getLogger(__name__)
router = APIRouter()


def get_ai_handler(request: Request):
    """Get AI handler from app state"""
    ai_handler = request.app.state.app_state.get("ai_handler")
    if not ai_handler or not ai_handler.initialized:
        raise HTTPException(
            status_code=503,
            detail="AI service not available. Use core automation endpoints instead.",
            headers={
                "X-AI-Available": "false",
                "X-Alternative-Endpoint": "/execute"
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
        result = await ai_handler.execute_action(
            task=request.task,
            screenshot=request.screenshot,
            context=request.context
        )
        
        return AIActionResponse(
            success=result.get("success", False),
            task=request.task,
            summary=result.get("summary", "Action completed"),
            actions_taken=result.get("actions_taken", []),
            reasoning=result.get("reasoning"),
            error=result.get("error")
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


# Import datetime for task timestamps
from datetime import datetime