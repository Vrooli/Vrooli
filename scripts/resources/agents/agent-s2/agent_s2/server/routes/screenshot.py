"""Screenshot routes for Agent S2 API"""

import logging
from typing import Optional, List
from fastapi import APIRouter, Query, Body, HTTPException

from ..models.responses import ScreenshotResponse
from ..services.capture import ScreenshotService

logger = logging.getLogger(__name__)
router = APIRouter()

# Initialize service
screenshot_service = ScreenshotService()


@router.post("", response_model=ScreenshotResponse)
@router.post("/", response_model=ScreenshotResponse)
async def take_screenshot(
    format: str = Query(default="png", description="Image format (png or jpeg)"),
    quality: int = Query(default=95, ge=1, le=100, description="JPEG quality"),
    region: Optional[List[int]] = Body(default=None, description="Region [x, y, width, height]")
):
    """Take a screenshot of the current display
    
    Args:
        format: Image format (png or jpeg)
        quality: JPEG quality (1-100)
        region: Optional region to capture [x, y, width, height]
        
    Returns:
        Screenshot data with base64 encoded image
    """
    try:
        result = screenshot_service.capture(
            format=format,
            quality=quality,
            region=region
        )
        
        return ScreenshotResponse(
            success=True,
            format=result["format"],
            size=result["size"],
            data=result["data"]
        )
        
    except ValueError as e:
        logger.error(f"Screenshot validation error: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Screenshot failed: {e}")
        raise HTTPException(status_code=500, detail=f"Screenshot failed: {str(e)}")


@router.get("/info")
async def get_screen_info():
    """Get screen information without taking a screenshot"""
    try:
        info = screenshot_service.get_screen_info()
        return {
            "screen_size": info["size"],
            "display": info["display"],
            "color_depth": info.get("color_depth", 24)
        }
    except Exception as e:
        logger.error(f"Failed to get screen info: {e}")
        raise HTTPException(status_code=500, detail=str(e))