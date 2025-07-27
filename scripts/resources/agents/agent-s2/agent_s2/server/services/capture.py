"""Screenshot capture service for Agent S2"""

import os
import io
import base64
import logging
from typing import Optional, List, Dict, Any, Tuple

import pyautogui
from PIL import Image

from ...config import Config

logger = logging.getLogger(__name__)


class ScreenshotService:
    """Service for handling screenshot operations"""
    
    def __init__(self):
        """Initialize screenshot service"""
        # Configure pyautogui
        pyautogui.FAILSAFE = False  # Disable failsafe in container
        pyautogui.PAUSE = 0.1  # Small pause between actions
        
    def capture(self, 
                format: str = "png",
                quality: int = 95,
                region: Optional[List[int]] = None) -> Dict[str, Any]:
        """Capture a screenshot
        
        Args:
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            
        Returns:
            Dictionary with screenshot data and metadata
            
        Raises:
            ValueError: If parameters are invalid
        """
        # Validate format
        format = format.lower()
        if format not in ["png", "jpeg", "jpg"]:
            raise ValueError(f"Invalid format: {format}. Must be png or jpeg")
            
        # Normalize jpeg/jpg
        if format == "jpg":
            format = "jpeg"
            
        # Take screenshot
        if region and len(region) == 4:
            # Convert [x, y, width, height] to (x, y, width, height) for pyautogui
            x, y, width, height = region
            screenshot = pyautogui.screenshot(region=(x, y, width, height))
        else:
            screenshot = pyautogui.screenshot()
            
        # Convert to bytes
        img_buffer = io.BytesIO()
        if format == "jpeg":
            # Convert RGBA to RGB for JPEG
            screenshot = screenshot.convert('RGB')
            screenshot.save(img_buffer, format="JPEG", quality=quality)
        else:
            screenshot.save(img_buffer, format="PNG")
            
        img_buffer.seek(0)
        
        # Encode to base64
        img_base64 = base64.b64encode(img_buffer.getvalue()).decode('utf-8')
        
        return {
            "format": format,
            "size": {"width": screenshot.width, "height": screenshot.height},
            "data": f"data:image/{format};base64,{img_base64}"
        }
        
    def capture_to_file(self, 
                       filename: str,
                       format: Optional[str] = None,
                       quality: int = 95,
                       region: Optional[List[int]] = None) -> str:
        """Capture screenshot and save to file
        
        Args:
            filename: Output filename
            format: Image format (auto-detected from filename if None)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            
        Returns:
            Path to saved file
        """
        # Auto-detect format from filename
        if format is None:
            ext = os.path.splitext(filename)[1].lower()
            format = ext[1:] if ext else "png"
            
        # Capture screenshot
        data = self.capture(format=format, quality=quality, region=region)
        
        # Decode base64
        img_data = data["data"].split(",")[1]
        img_bytes = base64.b64decode(img_data)
        
        # Save to file
        with open(filename, "wb") as f:
            f.write(img_bytes)
            
        return filename
        
    def get_screen_info(self) -> Dict[str, Any]:
        """Get screen information
        
        Returns:
            Dictionary with screen information
        """
        width, height = pyautogui.size()
        
        return {
            "size": {"width": width, "height": height},
            "display": os.environ.get("DISPLAY", ":99"),
            "color_depth": Config.SCREEN_DEPTH
        }
        
    def find_on_screen(self, 
                      image_path: str,
                      confidence: float = 0.8,
                      grayscale: bool = False) -> Optional[Tuple[int, int, int, int]]:
        """Find an image on screen
        
        Args:
            image_path: Path to image to find
            confidence: Matching confidence (0-1)
            grayscale: Use grayscale matching
            
        Returns:
            Tuple of (x, y, width, height) if found, None otherwise
        """
        try:
            location = pyautogui.locateOnScreen(
                image_path,
                confidence=confidence,
                grayscale=grayscale
            )
            return location
        except Exception as e:
            logger.error(f"Image search failed: {e}")
            return None
            
    def find_all_on_screen(self,
                          image_path: str,
                          confidence: float = 0.8,
                          grayscale: bool = False) -> List[Tuple[int, int, int, int]]:
        """Find all instances of an image on screen
        
        Args:
            image_path: Path to image to find
            confidence: Matching confidence (0-1)
            grayscale: Use grayscale matching
            
        Returns:
            List of (x, y, width, height) tuples
        """
        try:
            locations = list(pyautogui.locateAllOnScreen(
                image_path,
                confidence=confidence,
                grayscale=grayscale
            ))
            return locations
        except Exception as e:
            logger.error(f"Image search failed: {e}")
            return []
            
    def wait_for_image(self,
                      image_path: str,
                      timeout: float = 30.0,
                      confidence: float = 0.8) -> Optional[Tuple[int, int, int, int]]:
        """Wait for an image to appear on screen
        
        Args:
            image_path: Path to image to find
            timeout: Maximum wait time in seconds
            confidence: Matching confidence (0-1)
            
        Returns:
            Tuple of (x, y, width, height) if found, None if timeout
        """
        import time
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            location = self.find_on_screen(image_path, confidence)
            if location:
                return location
            time.sleep(0.5)
            
        return None