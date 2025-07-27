"""Keyboard control routes for Agent S2 API"""

import logging
from fastapi import APIRouter, HTTPException

from ..models.requests import KeyboardTypeRequest, KeyboardPressRequest
from ..models.responses import KeyboardActionResponse
from ..services.automation import AutomationService

logger = logging.getLogger(__name__)
router = APIRouter()

# Initialize service
automation_service = AutomationService()


@router.post("/type", response_model=KeyboardActionResponse)
async def type_text(request: KeyboardTypeRequest):
    """Type text using keyboard
    
    Args:
        request: Text typing parameters
        
    Returns:
        Action result
    """
    try:
        automation_service.type_text(
            text=request.text,
            interval=request.interval
        )
        
        return KeyboardActionResponse(
            success=True,
            action="type",
            text=request.text,
            message=f"Typed {len(request.text)} characters"
        )
    except Exception as e:
        logger.error(f"Text typing failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/press", response_model=KeyboardActionResponse)
async def press_key(request: KeyboardPressRequest):
    """Press key(s)
    
    Args:
        request: Key press parameters
        
    Returns:
        Action result
    """
    try:
        # Handle hotkey combinations
        if request.modifiers:
            keys = request.modifiers + [request.key]
            automation_service.hotkey(*keys)
            key_desc = "+".join(keys)
        else:
            automation_service.press_key(request.key)
            key_desc = request.key
            
        return KeyboardActionResponse(
            success=True,
            action="press",
            key=key_desc,
            message=f"Pressed {key_desc}"
        )
    except Exception as e:
        logger.error(f"Key press failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/hotkey", response_model=KeyboardActionResponse)
async def press_hotkey(keys: list[str]):
    """Press hotkey combination
    
    Args:
        keys: List of keys to press together
        
    Returns:
        Action result
    """
    try:
        if not keys:
            raise ValueError("No keys provided")
            
        automation_service.hotkey(*keys)
        key_desc = "+".join(keys)
        
        return KeyboardActionResponse(
            success=True,
            action="hotkey",
            key=key_desc,
            message=f"Pressed hotkey {key_desc}"
        )
    except ValueError as e:
        logger.error(f"Invalid hotkey: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Hotkey press failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/key-down", response_model=KeyboardActionResponse)
async def key_down(key: str):
    """Hold key down
    
    Args:
        key: Key to hold down
        
    Returns:
        Action result
    """
    try:
        automation_service.key_down(key)
        
        return KeyboardActionResponse(
            success=True,
            action="key_down",
            key=key,
            message=f"Holding {key} down"
        )
    except Exception as e:
        logger.error(f"Key down failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/key-up", response_model=KeyboardActionResponse)
async def key_up(key: str):
    """Release key
    
    Args:
        key: Key to release
        
    Returns:
        Action result
    """
    try:
        automation_service.key_up(key)
        
        return KeyboardActionResponse(
            success=True,
            action="key_up",
            key=key,
            message=f"Released {key}"
        )
    except Exception as e:
        logger.error(f"Key up failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))