"""Health check routes for Agent S2 API"""

import os
import logging
from fastapi import APIRouter, Request, HTTPException
import pyautogui

from ..models.responses import HealthResponse, CapabilitiesResponse
from ...config import Config

logger = logging.getLogger(__name__)
router = APIRouter()


@router.get("/health", response_model=HealthResponse)
async def health_check(request: Request):
    """Check service health and display status"""
    try:
        # Get app state
        app_state = request.app.state.app_state
        
        # Get screen information
        screen_width, screen_height = pyautogui.size()
        
        # AI status information
        ai_handler = app_state.get("ai_handler")
        ai_status = {
            "available": ai_handler is not None,
            "enabled": Config.AI_ENABLED,
            "initialized": ai_handler.initialized if ai_handler else False,
            "provider": ai_handler.provider if ai_handler and ai_handler.initialized else None,
            "model": ai_handler.model if ai_handler and ai_handler.initialized else None,
            "search_enabled": False  # TODO: Implement search support
        }
        
        return HealthResponse(
            status="healthy",
            display=os.environ.get("DISPLAY", ":99"),
            screen_size={"width": screen_width, "height": screen_height},
            startup_time=app_state["startup_time"],
            tasks_processed=app_state["task_counter"],
            ai_status=ai_status
        )
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"Service unhealthy: {str(e)}")


@router.get("/capabilities", response_model=CapabilitiesResponse)
async def get_capabilities(request: Request):
    """Get Agent S2 capabilities"""
    app_state = request.app.state.app_state
    ai_handler = app_state.get("ai_handler")
    ai_initialized = ai_handler and ai_handler.initialized
    
    capabilities = {
        "screenshot": True,
        "gui_automation": True,
        "mouse_control": True,
        "keyboard_control": True,
        "window_management": True,
        "planning": ai_initialized,
        "multi_step_tasks": True,
        "ai_reasoning": ai_initialized,
        "natural_language": ai_initialized,
        "screen_understanding": ai_initialized
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
    if ai_initialized:
        supported_tasks.extend([
            "ai_command",
            "ai_plan",
            "ai_analyze",
            "ai_sequence"
        ])
    
    ai_status = {
        "available": ai_handler is not None,
        "enabled": Config.AI_ENABLED,
        "initialized": ai_initialized,
        "provider": ai_handler.provider if ai_initialized else None,
        "model": ai_handler.model if ai_initialized else None
    }
    
    display_info = {
        "display": os.environ.get("DISPLAY", ":99"),
        "resolution": pyautogui.size()
    }
    
    return CapabilitiesResponse(
        capabilities=capabilities,
        supported_tasks=supported_tasks,
        ai_status=ai_status,
        display_info=display_info
    )