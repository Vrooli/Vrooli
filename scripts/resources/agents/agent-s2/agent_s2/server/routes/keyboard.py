"""Keyboard control routes for Agent S2 API"""

import logging
import time
from fastapi import APIRouter, HTTPException

from ..models.requests import KeyboardTypeRequest, KeyboardPressRequest
from ..models.responses import KeyboardActionResponse
from ..services.automation import AutomationService, AutomationError

logger = logging.getLogger(__name__)
router = APIRouter()

# Initialize service
automation_service = AutomationService()


@router.post("/type", response_model=KeyboardActionResponse)
async def type_text(request: KeyboardTypeRequest):
    """Type text using keyboard with optional target application focus
    
    Args:
        request: Text typing parameters with target awareness
        
    Returns:
        Action result with target information
    """
    start_time = time.time()
    
    try:
        # Use targeted typing method
        success, focused_window, focus_time = automation_service.type_text_targeted(
            text=request.text,
            target_app=request.target_app,
            interval=request.interval,
            window_criteria=request.window_criteria.dict() if request.window_criteria else None
        )
        
        execution_time = time.time() - start_time
        
        return KeyboardActionResponse(
            success=True,
            action="type",
            target_app=request.target_app,
            focused_window=focused_window.title if focused_window else None,
            window_id=focused_window.window_id if focused_window else None,
            focus_time=focus_time,
            execution_time=execution_time,
            message=f"Typed {len(request.text)} characters",
            text=request.text,
            interval=request.interval
        )
        
    except AutomationError as e:
        execution_time = time.time() - start_time
        logger.error(f"Targeted text typing failed: {e}")
        return KeyboardActionResponse(
            success=False,
            action="type",
            target_app=request.target_app,
            execution_time=execution_time,
            message="Failed to type text",
            error=str(e),
            text=request.text
        )
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Text typing failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/press", response_model=KeyboardActionResponse)
async def press_key(request: KeyboardPressRequest):
    """Press key(s) with optional target application focus
    
    Args:
        request: Key press parameters with target awareness
        
    Returns:
        Action result with target information
    """
    start_time = time.time()
    
    try:
        # Handle hotkey combinations vs single key
        if request.modifiers:
            keys = request.modifiers + [request.key]
            key_desc = "+".join(keys)
            
            # Use targeted hotkey method
            success, focused_window, focus_time = automation_service.hotkey_targeted(
                *keys,
                target_app=request.target_app,
                window_criteria=request.window_criteria.dict() if request.window_criteria else None
            )
        else:
            key_desc = request.key
            
            # Use targeted key press method
            success, focused_window, focus_time = automation_service.press_key_targeted(
                request.key,
                target_app=request.target_app,
                window_criteria=request.window_criteria.dict() if request.window_criteria else None
            )
        
        execution_time = time.time() - start_time
        
        return KeyboardActionResponse(
            success=True,
            action="press",
            target_app=request.target_app,
            focused_window=focused_window.title if focused_window else None,
            window_id=focused_window.window_id if focused_window else None,
            focus_time=focus_time,
            execution_time=execution_time,
            message=f"Pressed {key_desc}",
            key=key_desc
        )
        
    except AutomationError as e:
        execution_time = time.time() - start_time
        logger.error(f"Targeted key press failed: {e}")
        return KeyboardActionResponse(
            success=False,
            action="press",
            target_app=request.target_app,
            execution_time=execution_time,
            message="Failed to press key",
            error=str(e),
            key=request.key
        )
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Key press failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/hotkey", response_model=KeyboardActionResponse)
async def press_hotkey(keys: list[str]):
    """Press hotkey combination (legacy route - use /press with modifiers for target awareness)
    
    Args:
        keys: List of keys to press together
        
    Returns:
        Action result
    """
    start_time = time.time()
    
    try:
        if not keys:
            raise ValueError("No keys provided")
            
        automation_service.hotkey(*keys)
        key_desc = "+".join(keys)
        execution_time = time.time() - start_time
        
        return KeyboardActionResponse(
            success=True,
            action="hotkey",
            execution_time=execution_time,
            message=f"Pressed hotkey {key_desc}",
            key=key_desc
        )
    except ValueError as e:
        execution_time = time.time() - start_time
        logger.error(f"Invalid hotkey: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Hotkey press failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/key-down", response_model=KeyboardActionResponse)
async def key_down(key: str):
    """Hold key down (legacy route - no target awareness)
    
    Args:
        key: Key to hold down
        
    Returns:
        Action result
    """
    start_time = time.time()
    
    try:
        automation_service.key_down(key)
        execution_time = time.time() - start_time
        
        return KeyboardActionResponse(
            success=True,
            action="key_down",
            execution_time=execution_time,
            message=f"Holding {key} down",
            key=key
        )
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Key down failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/key-up", response_model=KeyboardActionResponse)
async def key_up(key: str):
    """Release key (legacy route - no target awareness)
    
    Args:
        key: Key to release
        
    Returns:
        Action result
    """
    start_time = time.time()
    
    try:
        automation_service.key_up(key)
        execution_time = time.time() - start_time
        
        return KeyboardActionResponse(
            success=True,
            action="key_up",
            execution_time=execution_time,
            message=f"Released {key}",
            key=key
        )
    except Exception as e:
        execution_time = time.time() - start_time
        logger.error(f"Key up failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))