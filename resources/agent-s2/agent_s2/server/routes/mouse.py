"""Mouse control routes for Agent S2 API"""

import logging
import time
from typing import Optional
from fastapi import APIRouter, HTTPException

from ..models.requests import MouseMoveRequest, MouseClickRequest
from ..models.responses import MousePositionResponse, MouseActionResponse
from ..services.automation import AutomationService, AutomationError

logger = logging.getLogger(__name__)
router = APIRouter()

# Initialize service
automation_service = AutomationService()


@router.get("/position", response_model=MousePositionResponse)
async def get_mouse_position():
    """Get current mouse position"""
    try:
        x, y = automation_service.get_mouse_position()
        return MousePositionResponse(x=x, y=y)
    except Exception as e:
        logger.error(f"Failed to get mouse position: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/move", response_model=MouseActionResponse)
async def move_mouse(request: MouseMoveRequest):
    """Move mouse to specified position with optional target application focus
    
    Args:
        request: Mouse movement parameters with target awareness
        
    Returns:
        Action result with target information
    """
    start_time = time.time()
    focused_window = None
    focus_time = 0.0
    
    try:
        # Handle target focusing if specified
        if request.target_app:
            success, focused_window, focus_time = automation_service._ensure_target_focus(
                request.target_app, 
                request.window_criteria.dict() if request.window_criteria else None
            )
            if not success and request.ensure_focus:
                raise AutomationError(f"Could not focus target application: {request.target_app}")
        
        # Proceed with mouse movement
        automation_service.move_mouse(
            x=request.x,
            y=request.y,
            duration=request.duration
        )
        
        execution_time = time.time() - start_time
        
        return MouseActionResponse(
            success=True,
            action="move",
            target_app=request.target_app,
            focused_window=focused_window.title if focused_window else None,
            window_id=focused_window.window_id if focused_window else None,
            focus_time=focus_time,
            execution_time=execution_time,
            message=f"Moved to ({request.x}, {request.y})",
            position={"x": request.x, "y": request.y}
        )
        
    except AutomationError as e:
        execution_time = time.time() - start_time
        logger.error(f"Targeted mouse move failed: {e}")
        return MouseActionResponse(
            success=False,
            action="move",
            target_app=request.target_app,
            execution_time=execution_time,
            message="Failed to move mouse",
            error=str(e),
            position={"x": request.x, "y": request.y}
        )
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Mouse move failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/click", response_model=MouseActionResponse)
async def click_mouse(request: MouseClickRequest):
    """Click mouse button with optional target application focus
    
    Args:
        request: Mouse click parameters with target awareness
        
    Returns:
        Action result with target information
    """
    start_time = time.time()
    
    try:
        # Get current position if not specified
        if request.x is None or request.y is None:
            current_x, current_y = automation_service.get_mouse_position()
            x = request.x if request.x is not None else current_x
            y = request.y if request.y is not None else current_y
        else:
            x, y = request.x, request.y
        
        # Use targeted click method
        success, focused_window, focus_time = automation_service.click_targeted(
            x=x,
            y=y,
            target_app=request.target_app,
            button=request.button,
            clicks=request.clicks,
            window_criteria=request.window_criteria.dict() if request.window_criteria else None
        )
        
        execution_time = time.time() - start_time
        action = "click" if request.clicks == 1 else f"click({request.clicks}x)"
        
        return MouseActionResponse(
            success=True,
            action=action,
            target_app=request.target_app,
            focused_window=focused_window.title if focused_window else None,
            window_id=focused_window.window_id if focused_window else None,
            focus_time=focus_time,
            execution_time=execution_time,
            message=f"{request.button.capitalize()} {action} at ({x}, {y})",
            position={"x": x, "y": y},
            button=request.button,
            clicks=request.clicks
        )
        
    except AutomationError as e:
        execution_time = time.time() - start_time
        logger.error(f"Targeted mouse click failed: {e}")
        return MouseActionResponse(
            success=False,
            action="click",
            target_app=request.target_app,
            execution_time=execution_time,
            message="Failed to click mouse",
            error=str(e),
            position={"x": x, "y": y} if 'x' in locals() else None,
            button=request.button
        )
    except ValueError as e:
        execution_time = time.time() - start_time
        logger.error(f"Invalid click parameters: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Mouse click failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/drag", response_model=MouseActionResponse)
async def drag_mouse(
    start_x: int,
    start_y: int,
    end_x: int,
    end_y: int,
    duration: float = 1.0,
    button: str = "left"
):
    """Drag mouse from one position to another (legacy route - no target awareness)
    
    Args:
        start_x: Starting X coordinate
        start_y: Starting Y coordinate
        end_x: Ending X coordinate
        end_y: Ending Y coordinate
        duration: Drag duration in seconds
        button: Mouse button to hold
        
    Returns:
        Action result
    """
    start_time = time.time()
    
    try:
        automation_service.drag(
            start_x=start_x,
            start_y=start_y,
            end_x=end_x,
            end_y=end_y,
            duration=duration,
            button=button
        )
        
        execution_time = time.time() - start_time
        
        return MouseActionResponse(
            success=True,
            action="drag",
            execution_time=execution_time,
            message=f"Dragged from ({start_x}, {start_y}) to ({end_x}, {end_y})",
            position={"start_x": start_x, "start_y": start_y, "end_x": end_x, "end_y": end_y},
            button=button
        )
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Mouse drag failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/scroll", response_model=MouseActionResponse)
async def scroll_mouse(
    clicks: int,
    x: Optional[int] = None,
    y: Optional[int] = None
):
    """Scroll mouse wheel (legacy route - no target awareness)
    
    Args:
        clicks: Number of scroll clicks (positive=up, negative=down)
        x: Optional X coordinate (uses current if None)
        y: Optional Y coordinate (uses current if None)
        
    Returns:
        Action result
    """
    start_time = time.time()
    
    try:
        automation_service.scroll(clicks=clicks, x=x, y=y)
        
        execution_time = time.time() - start_time
        direction = "up" if clicks > 0 else "down"
        
        return MouseActionResponse(
            success=True,
            action="scroll",
            execution_time=execution_time,
            message=f"Scrolled {abs(clicks)} clicks {direction}",
            position={"x": x, "y": y} if x is not None and y is not None else None,
            clicks=abs(clicks)
        )
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Mouse scroll failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))