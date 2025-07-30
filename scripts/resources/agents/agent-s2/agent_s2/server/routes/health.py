"""Health check routes for Agent S2 API"""

import os
import logging
from fastapi import APIRouter, Request, HTTPException
import pyautogui

from ..models.responses import HealthResponse, CapabilitiesResponse
from ...config import Config, AgentMode
from ...environment import ModeContext
from ..services.proxy_service import ProxyManager

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
        
        # Get mode information
        try:
            context = ModeContext()
            mode_info = {
                "current_mode": Config.get_mode_name(),
                "host_mode_enabled": Config.HOST_MODE_ENABLED,
                "security_level": context.get_context_summary()["security_level"],
                "applications_available": context.get_context_summary()["applications_count"]
            }
        except Exception as e:
            logger.warning(f"Failed to get mode info for health check: {e}")
            mode_info = {
                "current_mode": Config.get_mode_name(),
                "host_mode_enabled": Config.HOST_MODE_ENABLED,
                "security_level": "unknown",
                "applications_available": 0
            }
        
        # Get proxy status
        proxy_status = {
            "enabled": False,
            "running": False,
            "transparent_mode": False,
            "stats": None
        }
        
        try:
            proxy_service = app_state.get("proxy_service")
            if proxy_service:
                proxy_status["enabled"] = True
                proxy_status["running"] = proxy_service.running
                proxy_status["transparent_mode"] = proxy_service.is_proxy_configured()
                
                # Get proxy stats if available
                if proxy_service.running and hasattr(proxy_service, 'master') and proxy_service.master:
                    addon = next((a for a in proxy_service.master.addons if hasattr(a, 'get_stats')), None)
                    if addon:
                        proxy_status["stats"] = addon.get_stats()
        except Exception as e:
            logger.debug(f"Failed to get proxy status: {e}")

        return HealthResponse(
            status="healthy",
            display=os.environ.get("DISPLAY", ":99"),
            screen_size={"width": screen_width, "height": screen_height},
            startup_time=app_state["startup_time"],
            tasks_processed=app_state["task_counter"],
            ai_status=ai_status,
            mode_info=mode_info,
            proxy_status=proxy_status
        )
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"Service unhealthy: {str(e)}")


@router.get("/capabilities", response_model=CapabilitiesResponse)
async def get_capabilities(request: Request):
    """Get Agent S2 capabilities (mode-aware)"""
    app_state = request.app.state.app_state
    ai_handler = app_state.get("ai_handler")
    ai_initialized = ai_handler and ai_handler.initialized
    
    # Get mode-aware capabilities
    try:
        context = ModeContext()
        mode_capabilities = context.discovery.capabilities.get("capabilities", [])
        
        # Base capabilities (always available)
        capabilities = {
            "screenshot": True,
            "gui_automation": True,
            "mouse_control": True,
            "keyboard_control": True,
            "window_management": True,
            "multi_step_tasks": True,
        }
        
        # AI capabilities
        capabilities.update({
            "planning": ai_initialized,
            "ai_reasoning": ai_initialized,
            "natural_language": ai_initialized,
            "screen_understanding": ai_initialized
        })
        
        # Mode-specific capabilities
        if Config.is_host_mode():
            capabilities.update({
                "host_applications": True,
                "file_system_access": True,
                "native_desktop_integration": True,
                "system_automation": True
            })
        
        # Get mode-aware supported tasks
        supported_tasks = context.context_data["available_actions"]
        
        # Add AI tasks if available
        if ai_initialized:
            supported_tasks.extend([
                "ai_command",
                "ai_plan", 
                "ai_analyze",
                "ai_sequence"
            ])
        
    except Exception as e:
        logger.warning(f"Failed to get mode-aware capabilities: {e}")
        # Fallback to basic capabilities
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
        supported_tasks = ["screenshot", "click", "type_text", "key_press", "mouse_move"]
    
    ai_status = {
        "available": ai_handler is not None,
        "enabled": Config.AI_ENABLED,
        "initialized": ai_initialized,
        "provider": ai_handler.provider if ai_initialized else None,
        "model": ai_handler.model if ai_initialized else None
    }
    
    display_info = {
        "display": os.environ.get("DISPLAY", ":99"),
        "resolution": pyautogui.size(),
        "mode": Config.get_mode_name()
    }
    
    return CapabilitiesResponse(
        capabilities=capabilities,
        supported_tasks=supported_tasks,
        ai_status=ai_status,
        display_info=display_info,
        mode_info={
            "current_mode": Config.get_mode_name(),
            "host_mode_enabled": Config.HOST_MODE_ENABLED,
            "applications_available": len(context.context_data.get("applications", {})) if 'context' in locals() else 0
        }
    )