"""Mouse control routes for Agent S2 API"""

import logging
from typing import Optional
from fastapi import APIRouter, HTTPException

from ..models.requests import MouseMoveRequest, MouseClickRequest
from ..models.responses import MousePositionResponse, MouseActionResponse
from ..services.automation import AutomationService

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
    """Move mouse to specified position
    
    Args:
        request: Mouse movement parameters
        
    Returns:
        Action result
    """
    try:
        automation_service.move_mouse(
            x=request.x,
            y=request.y,
            duration=request.duration
        )
        
        return MouseActionResponse(
            success=True,
            action="move",
            position={"x": request.x, "y": request.y},
            message=f"Moved to ({request.x}, {request.y})"
        )
    except Exception as e:
        logger.error(f"Mouse move failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/click", response_model=MouseActionResponse)
async def click_mouse(request: MouseClickRequest):
    """Click mouse button
    
    Args:
        request: Mouse click parameters
        
    Returns:
        Action result
    """
    try:
        # Get current position if not specified
        if request.x is None or request.y is None:
            current_x, current_y = automation_service.get_mouse_position()
            x = request.x if request.x is not None else current_x
            y = request.y if request.y is not None else current_y
        else:
            x, y = request.x, request.y
            
        automation_service.click(
            x=x,
            y=y,
            button=request.button,
            clicks=request.clicks
        )
        
        action = "click" if request.clicks == 1 else f"click({request.clicks}x)"
        
        return MouseActionResponse(
            success=True,
            action=action,
            position={"x": x, "y": y},
            message=f"{request.button.capitalize()} {action} at ({x}, {y})"
        )
    except ValueError as e:
        logger.error(f"Invalid click parameters: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
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
    """Drag mouse from one position to another
    
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
    try:
        automation_service.drag(
            start_x=start_x,
            start_y=start_y,
            end_x=end_x,
            end_y=end_y,
            duration=duration,
            button=button
        )
        
        return MouseActionResponse(
            success=True,
            action="drag",
            position={"start_x": start_x, "start_y": start_y, "end_x": end_x, "end_y": end_y},
            message=f"Dragged from ({start_x}, {start_y}) to ({end_x}, {end_y})"
        )
    except Exception as e:
        logger.error(f"Mouse drag failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/scroll", response_model=MouseActionResponse)
async def scroll_mouse(
    clicks: int,
    x: Optional[int] = None,
    y: Optional[int] = None
):
    """Scroll mouse wheel
    
    Args:
        clicks: Number of scroll clicks (positive=up, negative=down)
        x: Optional X coordinate (uses current if None)
        y: Optional Y coordinate (uses current if None)
        
    Returns:
        Action result
    """
    try:
        automation_service.scroll(clicks=clicks, x=x, y=y)
        
        direction = "up" if clicks > 0 else "down"
        return MouseActionResponse(
            success=True,
            action="scroll",
            message=f"Scrolled {abs(clicks)} clicks {direction}"
        )
    except Exception as e:
        logger.error(f"Mouse scroll failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))