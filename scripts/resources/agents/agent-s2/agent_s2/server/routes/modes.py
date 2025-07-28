"""Mode management routes for Agent S2 API"""

import logging
from typing import Dict, Any
from fastapi import APIRouter, Request, HTTPException, Depends
from pydantic import BaseModel

from ..models.responses import BaseResponse
from ...config import Config, AgentMode
from ...environment import EnvironmentDiscovery, ModeContext

logger = logging.getLogger(__name__)
router = APIRouter()


class ModeInfo(BaseModel):
    """Mode information response model"""
    current_mode: str
    available_modes: list[str] 
    host_mode_enabled: bool
    can_switch_to_host: bool
    security_level: str
    capabilities_count: int
    applications_count: int


class EnvironmentInfo(BaseModel):
    """Environment information response model"""
    mode: str
    display_type: str
    window_manager: str
    applications_count: int
    capabilities: list[str]
    limitations: list[str]
    security_constraints: Dict[str, Any]
    system_context: str


class ModeSwitchRequest(BaseModel):
    """Mode switch request model"""
    new_mode: str
    force: bool = False


class ModeSwitchResponse(BaseResponse):
    """Mode switch response model"""
    previous_mode: str
    new_mode: str
    restart_required: bool
    message: str


@router.get("/current", response_model=ModeInfo)
async def get_current_mode():
    """Get current operating mode information"""
    try:
        context = ModeContext()
        summary = context.get_context_summary()
        
        return ModeInfo(
            current_mode=Config.get_mode_name(),
            available_modes=["sandbox", "host"],
            host_mode_enabled=Config.HOST_MODE_ENABLED,
            can_switch_to_host=Config.HOST_MODE_ENABLED,
            security_level=summary["security_level"], 
            capabilities_count=summary["capabilities_count"],
            applications_count=summary["applications_count"]
        )
    except Exception as e:
        logger.error(f"Failed to get mode info: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get mode information: {str(e)}")


@router.get("/environment", response_model=EnvironmentInfo)
async def get_environment_info():
    """Get detailed environment information for current mode"""
    try:
        context = ModeContext()
        capabilities = context.discovery.capabilities
        
        return EnvironmentInfo(
            mode=context.mode.value,
            display_type=capabilities.get("display", {}).get("type", "unknown"),
            window_manager=capabilities.get("window_manager", {}).get("type", "unknown"),
            applications_count=len(capabilities.get("applications", {})),
            capabilities=capabilities.get("capabilities", []),
            limitations=capabilities.get("limitations", []),
            security_constraints=capabilities.get("security", {}),
            system_context=context.system_prompt
        )
    except Exception as e:
        logger.error(f"Failed to get environment info: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get environment information: {str(e)}")


@router.post("/switch", response_model=ModeSwitchResponse)
async def switch_mode(request: ModeSwitchRequest):
    """Switch between operating modes"""
    try:
        current_mode = Config.get_mode_name()
        new_mode_name = request.new_mode.lower()
        
        # Validate new mode
        if new_mode_name not in ["sandbox", "host"]:
            raise HTTPException(status_code=400, detail=f"Invalid mode: {new_mode_name}")
        
        if current_mode == new_mode_name:
            return ModeSwitchResponse(
                success=True,
                previous_mode=current_mode,
                new_mode=new_mode_name,
                restart_required=False,
                message=f"Already in {new_mode_name} mode"
            )
        
        # Check if host mode is enabled
        if new_mode_name == "host" and not Config.HOST_MODE_ENABLED:
            raise HTTPException(
                status_code=403, 
                detail="Host mode is not enabled. Update configuration to enable host mode."
            )
        
        # Perform mode switch
        try:
            new_mode = AgentMode(new_mode_name)
            Config.switch_mode(new_mode)
            
            return ModeSwitchResponse(
                success=True,
                previous_mode=current_mode,
                new_mode=new_mode_name,
                restart_required=True,
                message=f"Mode switched from {current_mode} to {new_mode_name}. Container restart required for changes to take effect."
            )
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))
            
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to switch mode: {e}")
        raise HTTPException(status_code=500, detail=f"Mode switch failed: {str(e)}")


@router.get("/capabilities")
async def get_mode_capabilities():
    """Get detailed capabilities for current mode"""
    try:
        discovery = EnvironmentDiscovery()
        context = ModeContext(discovery=discovery)
        
        return {
            "mode": Config.get_mode_name(),
            "capabilities": discovery.capabilities,
            "available_actions": discovery.get_available_actions(),
            "context_summary": context.get_context_summary(),
            "security_constraints": context.get_security_constraints()
        }
    except Exception as e:
        logger.error(f"Failed to get capabilities: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get capabilities: {str(e)}")


@router.get("/applications")
async def get_available_applications():
    """Get list of available applications for current mode"""
    try:
        context = ModeContext()
        applications = context.context_data["applications"]
        
        # Format for API response
        formatted_apps = []
        for name, info in applications.items():
            formatted_apps.append({
                "name": info["name"],
                "command": info.get("command", "unknown"),
                "category": info.get("category", "unknown"),
                "launcher": info.get("launcher", "unknown"),
                "description": info.get("description", "No description"),
                "capabilities": info.get("capabilities", [])
            })
        
        return {
            "mode": Config.get_mode_name(),
            "applications_count": len(formatted_apps),
            "applications": formatted_apps
        }
    except Exception as e:
        logger.error(f"Failed to get applications: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get applications: {str(e)}")


@router.get("/applications/{app_name}")
async def get_application_info(app_name: str):
    """Get detailed information about a specific application"""
    try:
        context = ModeContext()
        app_info = context.get_application_info(app_name)
        
        if not app_info:
            raise HTTPException(status_code=404, detail=f"Application '{app_name}' not found")
        
        launch_instructions = context.get_launch_instructions(app_name)
        
        return {
            "mode": Config.get_mode_name(),
            "application": app_info,
            "launch_instructions": launch_instructions,
            "available_in_mode": True
        }
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to get application info: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get application info: {str(e)}")


@router.get("/security")
async def get_security_info():
    """Get security information for current mode"""
    try:
        context = ModeContext()
        security_constraints = context.get_security_constraints()
        
        return {
            "mode": Config.get_mode_name(),
            "security_level": context.context_data["mode"]["security_level"],
            "constraints": security_constraints,
            "limitations": context.context_data["limitations"],
            "config": {
                "host_mode_enabled": Config.HOST_MODE_ENABLED,
                "host_display_access": Config.HOST_DISPLAY_ACCESS,
                "audit_logging": Config.HOST_AUDIT_LOGGING,
                "security_profile": Config.HOST_SECURITY_PROFILE
            }
        }
    except Exception as e:
        logger.error(f"Failed to get security info: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get security info: {str(e)}")


@router.post("/refresh")
async def refresh_environment():
    """Refresh environment discovery and context"""
    try:
        # Create new context to force re-discovery
        context = ModeContext()
        summary = context.get_context_summary()
        
        return {
            "success": True,
            "message": "Environment refreshed successfully",
            "mode": Config.get_mode_name(),
            "timestamp": summary["timestamp"],
            "applications_count": summary["applications_count"],
            "capabilities_count": summary["capabilities_count"]
        }
    except Exception as e:
        logger.error(f"Failed to refresh environment: {e}")
        raise HTTPException(status_code=500, detail=f"Environment refresh failed: {str(e)}")


@router.get("/context")
async def get_ai_context():
    """Get AI system context for current mode"""
    try:
        context = ModeContext()
        
        return {
            "mode": Config.get_mode_name(),
            "system_prompt": context.system_prompt,
            "context_data": context.context_data,
            "available_actions": context.context_data["available_actions"],
            "generated_at": context.context_data["timestamp"]
        }
    except Exception as e:
        logger.error(f"Failed to get AI context: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get AI context: {str(e)}")