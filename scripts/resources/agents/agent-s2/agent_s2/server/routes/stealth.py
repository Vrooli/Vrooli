"""Stealth mode API routes for Agent S2"""

from fastapi import APIRouter, HTTPException, Request
from typing import Dict, Any, Optional
from pydantic import BaseModel

from ..services.ai_handler import AIHandler

router = APIRouter(tags=["stealth"])


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


class StealthConfigureRequest(BaseModel):
    """Request to configure stealth settings"""
    enabled: Optional[bool] = None
    features: Optional[Dict[str, bool]] = None


class StealthProfileRequest(BaseModel):
    """Request for profile operations"""
    profile_id: str
    profile_type: Optional[str] = "residential"


class StealthTestRequest(BaseModel):
    """Request to test stealth effectiveness"""
    url: Optional[str] = "https://bot.sannysoft.com/"


@router.get("/status")
async def get_stealth_status(request: Request) -> Dict[str, Any]:
    """Get current stealth mode status
    
    Returns:
        Stealth mode status and configuration
    """
    handler = get_ai_handler(request)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    return handler.stealth_manager.get_status()


@router.post("/configure")
async def configure_stealth(request: StealthConfigureRequest, req: Request) -> Dict[str, Any]:
    """Configure stealth mode settings
    
    Args:
        request: Configuration settings
        
    Returns:
        Updated configuration
    """
    handler = get_ai_handler(req)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    # Update configuration
    if request.enabled is not None:
        handler.stealth_manager.config.enabled = request.enabled
        
    if request.features:
        for feature, enabled in request.features.items():
            if hasattr(handler.stealth_manager.config, feature):
                setattr(handler.stealth_manager.config, feature, enabled)
                
    return {
        "success": True,
        "config": handler.stealth_manager.config.to_dict()
    }


@router.post("/test")
async def test_stealth(request: StealthTestRequest, req: Request) -> Dict[str, Any]:
    """Test stealth mode effectiveness
    
    Args:
        request: Test configuration
        
    Returns:
        Test results
    """
    handler = get_ai_handler(req)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    return handler.stealth_manager.test_stealth_effectiveness(request.url)


@router.post("/profile/create")
async def create_profile(request: StealthProfileRequest, req: Request) -> Dict[str, Any]:
    """Create a new stealth profile
    
    Args:
        request: Profile configuration
        
    Returns:
        Created profile information
    """
    handler = get_ai_handler(req)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    try:
        # Generate and save profile
        profile = handler.stealth_manager.profile_manager.generate_profile(request.profile_type)
        handler.stealth_manager.profile_manager.save_profile(request.profile_id)
        
        return {
            "success": True,
            "profile_id": request.profile_id,
            "profile": profile
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to create profile: {str(e)}")


@router.get("/profile/list")
async def list_profiles(request: Request) -> Dict[str, Any]:
    """List all saved profiles
    
    Returns:
        List of profiles
    """
    handler = get_ai_handler(request)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    profiles = handler.stealth_manager.session_manager.list_profiles()
    
    return {
        "success": True,
        "profiles": profiles,
        "count": len(profiles)
    }


@router.put("/profile/{profile_id}/activate")
async def activate_profile(profile_id: str, request: Request) -> Dict[str, Any]:
    """Activate a stealth profile
    
    Args:
        profile_id: Profile to activate
        
    Returns:
        Activation result
    """
    handler = get_ai_handler(request)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    try:
        result = await handler.stealth_manager.initialize(profile_id)
        
        if not result["success"]:
            raise HTTPException(status_code=500, detail=f"Failed to activate profile: {result['errors']}")
            
        return result
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to activate profile: {str(e)}")


@router.delete("/profile/{profile_id}")
async def delete_profile(profile_id: str, request: Request) -> Dict[str, Any]:
    """Delete a stealth profile and restart browser
    
    Args:
        profile_id: Profile to delete
        
    Returns:
        Deletion result
    """
    handler = get_ai_handler(request)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    try:
        # Pass automation service for browser control
        automation_service = getattr(handler, 'automation_service', None)
        handler.stealth_manager.session_manager.delete_session_data(
            profile_id, automation_service, restart_browser=True
        )
        
        return {
            "success": True,
            "message": f"Profile {profile_id} deleted and browser restarted"
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to delete profile: {str(e)}")


@router.post("/session/save")
async def save_session(request: Request) -> Dict[str, Any]:
    """Save current browser session
    
    Returns:
        Save result
    """
    handler = get_ai_handler(request)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    success = await handler.stealth_manager.save_current_session()
    
    return {
        "success": success,
        "profile_id": handler.stealth_manager._current_profile_id
    }


@router.delete("/session/reset")
async def reset_session(request: Request) -> Dict[str, Any]:
    """Reset session state and close all browser windows/tabs
    
    Returns:
        Reset result
    """
    handler = get_ai_handler(request)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    # Pass automation service for browser control
    automation_service = getattr(handler, 'automation_service', None)
    handler.stealth_manager.session_manager.delete_session_state(automation_service)
    
    return {
        "success": True,
        "message": "Session state reset and browser windows closed"
    }


@router.delete("/session/data")
async def reset_session_data(request: Request) -> Dict[str, Any]:
    """Reset all session data and restart browser
    
    Returns:
        Reset result
    """
    handler = get_ai_handler(request)
    if not hasattr(handler, 'stealth_manager'):
        raise HTTPException(status_code=500, detail="Stealth mode not initialized")
        
    try:
        # Pass automation service for browser control, clear all profiles
        automation_service = getattr(handler, 'automation_service', None)
        handler.stealth_manager.session_manager.delete_session_data(
            profile_id=None, automation_service=automation_service, restart_browser=True
        )
        
        return {
            "success": True,
            "message": "All session data cleared and browser restarted"
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to reset session data: {str(e)}")