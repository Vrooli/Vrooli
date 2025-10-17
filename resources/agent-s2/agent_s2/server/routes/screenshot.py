"""Screenshot routes for Agent S2 API"""

import logging
from typing import Optional, List, Dict, Any
from fastapi import APIRouter, Query, Body, HTTPException, Request
from fastapi.responses import Response
from fastapi.exceptions import RequestValidationError
from starlette.exceptions import HTTPException as StarletteHTTPException

from ..models.responses import ScreenshotResponse
from ..services.capture import ScreenshotService
from ..services.window_manager import WindowManagerError

logger = logging.getLogger(__name__)
router = APIRouter()

# Initialize service
screenshot_service = ScreenshotService()


@router.post("/")
async def take_screenshot(
    format: str = Query(default="png", description="Image format (png or jpeg)", regex="^(png|jpeg|jpg)$"),
    quality: int = Query(default=95, ge=1, le=100, description="JPEG quality"),
    response_format: str = Query(default="json", description="Response format (json or binary)", regex="^(json|binary)$"),
    region: Optional[List[int]] = Body(default=None, description="Region [x, y, width, height]", embed=True),
    max_size_mb: Optional[float] = Query(default=10.0, description="Maximum screenshot size in MB (for json format)")
):
    """Take a screenshot of the current display
    
    Args:
        format: Image format (png or jpeg)
        quality: JPEG quality (1-100)
        response_format: Response format (json or binary)
        region: Optional region to capture [x, y, width, height]
        max_size_mb: Maximum size in MB for json response (to prevent context overflow)
        
    Returns:
        For json format: Screenshot data with base64 encoded image
        For binary format: Raw image bytes
    """
    try:
        # Validate response_format (redundant with regex, but provides clear error)
        if response_format not in ["json", "binary"]:
            raise ValueError(f"Invalid response_format: {response_format}. Must be 'json' or 'binary'")
            
        # Validate region if provided
        if region:
            if len(region) != 4:
                raise HTTPException(status_code=400, detail="Region must be [x, y, width, height] with exactly 4 values")
            if any(v < 0 for v in region):
                raise HTTPException(status_code=400, detail="Region values must be non-negative")
            if region[2] <= 0 or region[3] <= 0:
                raise HTTPException(status_code=400, detail="Region width and height must be positive")
            
        result = screenshot_service.capture(
            format=format,
            quality=quality,
            region=region
        )
        
        # Check size for json response to prevent context window issues
        if response_format == "json":
            # Calculate approximate size of base64 data
            base64_size = len(result["data"]) / (1024 * 1024)  # Convert to MB
            if base64_size > max_size_mb:
                logger.warning(f"Screenshot too large: {base64_size:.2f}MB > {max_size_mb}MB limit")
                raise ValueError(f"Screenshot too large ({base64_size:.2f}MB). Maximum allowed: {max_size_mb}MB. Use binary format or smaller region.")
        
        if response_format == "binary":
            # Return raw binary data
            image_data = result["data"]
            if image_data.startswith('data:image'):
                # Extract base64 part and decode
                base64_data = image_data.split(',')[1]
                import base64
                binary_data = base64.b64decode(base64_data)
                
                # Determine MIME type
                mime_type = f"image/{result['format']}"
                
                return Response(
                    content=binary_data,
                    media_type=mime_type,
                    headers={
                        "Content-Disposition": f"attachment; filename=screenshot.{result['format']}"
                    }
                )
            else:
                raise ValueError("Invalid image data format")
        else:
            # Return JSON response (default behavior)
            return ScreenshotResponse(
                success=True,
                format=result["format"],
                size=result["size"],
                data=result["data"]
            )
        
    except ValueError as e:
        logger.error(f"Screenshot validation error: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except MemoryError as e:
        logger.error(f"Screenshot memory error: {e}")
        raise HTTPException(status_code=413, detail="Screenshot too large to process. Try a smaller region or lower quality.")
    except Exception as e:
        logger.error(f"Screenshot failed: {e}", exc_info=True)
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


@router.post("/window")
async def capture_window(
    window_title: str = Query(description="Window title to capture"),
    format: str = Query(default="png", description="Image format (png or jpeg)"),
    quality: int = Query(default=95, ge=1, le=100, description="JPEG quality")
):
    """Capture specific window by title"""
    try:
        result = screenshot_service.capture_window(
            window_title=window_title,
            format=format,
            quality=quality
        )
        
        return ScreenshotResponse(
            success=True,
            format=result["format"],
            size=result["size"],
            data=result["data"],
            window_info=result.get("window_info")
        )
        
    except WindowManagerError as e:
        logger.error(f"Window capture error: {e}")
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error(f"Window screenshot failed: {e}")
        raise HTTPException(status_code=500, detail=f"Window screenshot failed: {str(e)}")


@router.post("/window-by-id")
async def capture_window_by_id(
    window_id: str = Query(description="Window ID to capture"),
    format: str = Query(default="png", description="Image format (png or jpeg)"),
    quality: int = Query(default=95, ge=1, le=100, description="JPEG quality")
):
    """Capture specific window by ID"""
    try:
        result = screenshot_service.capture_window_by_id(
            window_id=window_id,
            format=format,
            quality=quality
        )
        
        return ScreenshotResponse(
            success=True,
            format=result["format"],
            size=result["size"],
            data=result["data"],
            window_info=result.get("window_info")
        )
        
    except WindowManagerError as e:
        logger.error(f"Window capture by ID error: {e}")
        raise HTTPException(status_code=404, detail=str(e))
    except Exception as e:
        logger.error(f"Window screenshot by ID failed: {e}")
        raise HTTPException(status_code=500, detail=f"Window screenshot by ID failed: {str(e)}")


@router.post("/compare")
async def compare_screenshots(
    data: Dict[str, Any] = Body(description="Screenshots and comparison parameters")
):
    """Compare two screenshots for similarity"""
    try:
        screenshot1 = data.get("screenshot1")
        screenshot2 = data.get("screenshot2")
        method = data.get("method", "mse")
        
        if not screenshot1 or not screenshot2:
            raise ValueError("Both screenshot1 and screenshot2 are required")
        
        result = screenshot_service.compare_screenshots(
            screenshot1=screenshot1,
            screenshot2=screenshot2,
            method=method
        )
        
        return {
            "success": True,
            "comparison": result
        }
        
    except ValueError as e:
        logger.error(f"Screenshot comparison validation error: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Screenshot comparison failed: {e}")
        raise HTTPException(status_code=500, detail=f"Screenshot comparison failed: {str(e)}")


@router.post("/detect-changes")
async def detect_changes(
    data: Dict[str, Any] = Body(description="Change detection parameters")
):
    """Monitor screen for changes"""
    try:
        interval = data.get("interval", 1.0)
        timeout = data.get("timeout", 30.0)
        threshold = data.get("threshold", 0.05)
        
        changes = screenshot_service.detect_changes(
            interval=interval,
            timeout=timeout,
            threshold=threshold
        )
        
        return {
            "success": True,
            "changes": changes,
            "change_count": len(changes)
        }
        
    except Exception as e:
        logger.error(f"Change detection failed: {e}")
        raise HTTPException(status_code=500, detail=f"Change detection failed: {str(e)}")


@router.post("/stop-change-detection")
async def stop_change_detection():
    """Stop active change detection"""
    try:
        screenshot_service.stop_change_detection()
        return {"success": True, "message": "Change detection stopped"}
    except Exception as e:
        logger.error(f"Failed to stop change detection: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/pixel-color")
async def get_pixel_color(
    x: int = Query(description="X coordinate"),
    y: int = Query(description="Y coordinate")
):
    """Get color of specific pixel"""
    try:
        color = screenshot_service.get_pixel_color(x, y)
        return {
            "success": True,
            "coordinates": {"x": x, "y": y},
            "color": list(color)
        }
        
    except ValueError as e:
        logger.error(f"Pixel color validation error: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to get pixel color: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/pixel-colors-region")
async def get_pixel_colors_region(
    x: int = Query(description="Starting X coordinate"),
    y: int = Query(description="Starting Y coordinate"),
    width: int = Query(description="Region width"),
    height: int = Query(description="Region height")
):
    """Get colors of all pixels in a region"""
    try:
        colors = screenshot_service.get_pixel_colors_region(x, y, width, height)
        return {
            "success": True,
            "region": {"x": x, "y": y, "width": width, "height": height},
            "colors": colors
        }
        
    except ValueError as e:
        logger.error(f"Pixel colors region validation error: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to get pixel colors region: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/difference")
async def create_difference_image(
    data: Dict[str, Any] = Body(description="Screenshots to compare")
):
    """Create visual difference image between two screenshots"""
    try:
        screenshot1 = data.get("screenshot1")
        screenshot2 = data.get("screenshot2")
        
        if not screenshot1 or not screenshot2:
            raise ValueError("Both screenshot1 and screenshot2 are required")
        
        result = screenshot_service.create_difference_image(
            screenshot1=screenshot1,
            screenshot2=screenshot2
        )
        
        return {
            "success": True,
            "difference_image": result
        }
        
    except ValueError as e:
        logger.error(f"Difference image validation error: {e}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"Failed to create difference image: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to create difference image: {str(e)}")