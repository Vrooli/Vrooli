"""Agent S2 Screenshot Client

Specialized client for screenshot operations.
"""

from typing import Optional, List, Dict, Any, Tuple
import os
from datetime import datetime

from .base import AgentS2Client


class ScreenshotClient:
    """Client specialized for screenshot operations"""
    
    def __init__(self, client: Optional[AgentS2Client] = None):
        """Initialize screenshot client
        
        Args:
            client: Base Agent S2 client (creates new if None)
        """
        self.client = client or AgentS2Client()
        
    def capture(self, 
                format: str = "png", 
                quality: int = 95,
                region: Optional[List[int]] = None) -> Dict[str, Any]:
        """Capture screenshot with specified parameters
        
        Args:
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            
        Returns:
            Screenshot data dictionary
        """
        return self.client.screenshot(format=format, quality=quality, region=region)
    
    def capture_full_screen(self, format: str = "png") -> Dict[str, Any]:
        """Capture full screen screenshot
        
        Args:
            format: Image format (png or jpeg)
            
        Returns:
            Screenshot data dictionary
        """
        return self.capture(format=format)
    
    def capture_region(self, x: int, y: int, width: int, height: int, 
                      format: str = "png") -> Dict[str, Any]:
        """Capture specific screen region
        
        Args:
            x: Starting X coordinate
            y: Starting Y coordinate
            width: Region width
            height: Region height
            format: Image format (png or jpeg)
            
        Returns:
            Screenshot data dictionary
        """
        return self.capture(format=format, region=[x, y, width, height])
    
    def capture_window(self, window_title: str, format: str = "png") -> Dict[str, Any]:
        """Capture specific window by title
        
        Args:
            window_title: Window title to capture
            format: Image format (png or jpeg)
            
        Returns:
            Screenshot data dictionary
        """
        params = {"format": format, "window_title": window_title}
        response = self.client._request('POST', '/screenshot/window', params=params)
        response.raise_for_status()
        return response.json()
    
    def save(self, 
             filename: Optional[str] = None,
             directory: Optional[str] = None,
             format: str = "png", 
             quality: int = 95,
             region: Optional[List[int]] = None) -> str:
        """Capture and save screenshot with auto-generated filename
        
        Args:
            filename: Output filename (auto-generated if None)
            directory: Output directory (current dir if None)
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            
        Returns:
            Path to saved file
        """
        if filename is None:
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            filename = f"screenshot_{timestamp}.{format}"
            
        if directory:
            os.makedirs(directory, exist_ok=True)
            filepath = os.path.join(directory, filename)
        else:
            filepath = filename
            
        return self.client.save_screenshot(
            filepath, 
            format=format, 
            quality=quality, 
            region=region
        )
    
    def compare(self, screenshot1: Dict[str, Any], 
                screenshot2: Dict[str, Any], 
                method: str = "mse") -> Dict[str, Any]:
        """Compare two screenshots for similarity
        
        Args:
            screenshot1: First screenshot data
            screenshot2: Second screenshot data
            method: Comparison method (mse, histogram, pixel_diff)
            
        Returns:
            Comparison results with similarity score
        """
        data = {
            "screenshot1": screenshot1,
            "screenshot2": screenshot2,
            "method": method
        }
        response = self.client._request('POST', '/screenshot/compare', json=data)
        response.raise_for_status()
        return response.json()
    
    def find_changes(self, interval: float = 1.0, 
                    timeout: float = 30.0,
                    threshold: float = 0.05) -> List[Dict[str, Any]]:
        """Monitor screen for changes
        
        Args:
            interval: Check interval in seconds
            timeout: Maximum monitoring time in seconds
            threshold: Change detection threshold (0-1, lower = more sensitive)
            
        Returns:
            List of detected changes with timestamps
        """
        data = {
            "interval": interval,
            "timeout": timeout,
            "threshold": threshold
        }
        response = self.client._request('POST', '/screenshot/detect-changes', json=data)
        response.raise_for_status()
        return response.json()
    
    def capture_series(self, count: int, interval: float = 1.0,
                      format: str = "png", quality: int = 95) -> List[Dict[str, Any]]:
        """Capture a series of screenshots
        
        Args:
            count: Number of screenshots to capture
            interval: Delay between captures in seconds
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            
        Returns:
            List of screenshot data dictionaries
        """
        screenshots = []
        import time
        
        for i in range(count):
            screenshot = self.capture(format=format, quality=quality)
            screenshots.append(screenshot)
            
            if i < count - 1:
                time.sleep(interval)
                
        return screenshots
    
    def capture_window_by_id(self, window_id: str, format: str = "png") -> Dict[str, Any]:
        """Capture specific window by ID
        
        Args:
            window_id: Window ID to capture
            format: Image format (png or jpeg)
            
        Returns:
            Screenshot data dictionary
        """
        params = {"format": format, "window_id": window_id}
        response = self.client._request('POST', '/screenshot/window-by-id', params=params)
        response.raise_for_status()
        return response.json()
    
    def create_difference_image(self, screenshot1: Dict[str, Any], 
                               screenshot2: Dict[str, Any]) -> Dict[str, Any]:
        """Create visual difference image between two screenshots
        
        Args:
            screenshot1: First screenshot data
            screenshot2: Second screenshot data
            
        Returns:
            Difference image data
        """
        data = {
            "screenshot1": screenshot1,
            "screenshot2": screenshot2
        }
        response = self.client._request('POST', '/screenshot/difference', json=data)
        response.raise_for_status()
        return response.json()
    
    def get_pixel_colors_region(self, x: int, y: int, width: int, height: int) -> List[List[Tuple[int, int, int]]]:
        """Get colors of all pixels in a region
        
        Args:
            x: Starting X coordinate
            y: Starting Y coordinate
            width: Region width
            height: Region height
            
        Returns:
            2D list of RGB color tuples [row][col] = (r, g, b)
        """
        params = {"x": x, "y": y, "width": width, "height": height}
        response = self.client._request('GET', '/screenshot/pixel-colors-region', params=params)
        response.raise_for_status()
        result = response.json()
        return result["colors"]
    
    def stop_change_detection(self) -> bool:
        """Stop active change detection
        
        Returns:
            True if successfully stopped
        """
        response = self.client._request('POST', '/screenshot/stop-change-detection')
        response.raise_for_status()
        result = response.json()
        return result.get("success", False)
    
    def get_pixel_color(self, x: int, y: int) -> Tuple[int, int, int]:
        """Get color of specific pixel
        
        Args:
            x: X coordinate
            y: Y coordinate
            
        Returns:
            RGB color tuple
        """
        params = {"x": x, "y": y}
        response = self.client._request('GET', '/screenshot/pixel-color', params=params)
        response.raise_for_status()
        result = response.json()
        return tuple(result["color"])