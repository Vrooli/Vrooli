"""Agent S2 Base Client

Provides a unified interface for interacting with the Agent S2 API.
"""

import requests
from typing import Optional, Dict, Any, List, Union, Tuple
import base64
import io
from PIL import Image

from ..config import Config


class AgentS2Client:
    """Base client for Agent S2 API interactions"""
    
    def __init__(self, base_url: Optional[str] = None, timeout: int = 30):
        """Initialize Agent S2 client
        
        Args:
            base_url: Base URL for API (defaults to Config.API_BASE_URL)
            timeout: Request timeout in seconds
        """
        self.base_url = (base_url or Config.API_BASE_URL).rstrip('/')
        self.timeout = timeout
        self.session = requests.Session()
        
    def _request(self, method: str, endpoint: str, **kwargs) -> requests.Response:
        """Make HTTP request to API
        
        Args:
            method: HTTP method
            endpoint: API endpoint (without base URL)
            **kwargs: Additional request arguments
            
        Returns:
            Response object
        """
        url = f"{self.base_url}/{endpoint.lstrip('/')}"
        kwargs.setdefault('timeout', self.timeout)
        return self.session.request(method, url, **kwargs)
    
    def health_check(self) -> bool:
        """Check if Agent S2 service is healthy
        
        Returns:
            True if service is healthy, False otherwise
        """
        try:
            response = self._request('GET', '/health')
            return response.ok
        except (requests.RequestException, ConnectionError, TimeoutError):
            return False
    
    def screenshot(self, 
                   format: str = "png", 
                   quality: int = 95,
                   region: Optional[List[int]] = None) -> Dict[str, Any]:
        """Capture screenshot
        
        Args:
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            
        Returns:
            Dictionary with screenshot data and metadata
            
        Raises:
            requests.HTTPError: If request fails
        """
        params = {"format": format}
        if format == "jpeg" and quality is not None:
            params["quality"] = quality
            
        json_data = region if region else None
        
        response = self._request('POST', '/screenshot', params=params, json=json_data)
        response.raise_for_status()
        return response.json()
    
    def save_screenshot(self, 
                       filename: str,
                       format: str = "png", 
                       quality: int = 95,
                       region: Optional[List[int]] = None) -> str:
        """Capture and save screenshot to file
        
        Args:
            filename: Output filename
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            
        Returns:
            Path to saved file
            
        Raises:
            requests.HTTPError: If request fails
        """
        data = self.screenshot(format=format, quality=quality, region=region)
        
        # Extract base64 data
        image_data = data['data']
        if image_data.startswith('data:image'):
            image_data = image_data.split(',')[1]
            
        # Decode and save
        img_bytes = base64.b64decode(image_data)
        img = Image.open(io.BytesIO(img_bytes))
        img.save(filename)
        
        return filename
    
    def mouse_position(self) -> Dict[str, int]:
        """Get current mouse position
        
        Returns:
            Dictionary with x and y coordinates
            
        Raises:
            requests.HTTPError: If request fails
        """
        response = self._request('GET', '/mouse/position')
        response.raise_for_status()
        return response.json()
    
    def mouse_move(self, x: int, y: int, duration: float = 0.0) -> Dict[str, Any]:
        """Move mouse to position
        
        Args:
            x: Target X coordinate
            y: Target Y coordinate
            duration: Movement duration in seconds
            
        Returns:
            Response data
            
        Raises:
            requests.HTTPError: If request fails
        """
        task_data = {
            "task_type": "mouse_move",
            "parameters": {
                "x": x,
                "y": y,
                "duration": duration
            }
        }
        response = self._request('POST', '/execute', json=task_data)
        response.raise_for_status()
        return response.json()
    
    def mouse_click(self, 
                    button: str = "left",
                    x: Optional[int] = None,
                    y: Optional[int] = None,
                    clicks: int = 1) -> Dict[str, Any]:
        """Click mouse button
        
        Args:
            button: Mouse button (left, right, middle)
            x: Optional X coordinate (uses current position if None)
            y: Optional Y coordinate (uses current position if None)
            clicks: Number of clicks
            
        Returns:
            Response data
            
        Raises:
            requests.HTTPError: If request fails
        """
        parameters = {
            "button": button,
            "clicks": clicks
        }
        if x is not None and y is not None:
            parameters["x"] = x
            parameters["y"] = y
            
        task_data = {
            "task_type": "click",
            "parameters": parameters
        }
        response = self._request('POST', '/execute', json=task_data)
        response.raise_for_status()
        return response.json()
    
    def keyboard_type(self, text: str, interval: float = 0.0) -> Dict[str, Any]:
        """Type text using keyboard
        
        Args:
            text: Text to type
            interval: Delay between keystrokes in seconds
            
        Returns:
            Response data
            
        Raises:
            requests.HTTPError: If request fails
        """
        task_data = {
            "task_type": "type_text",
            "parameters": {
                "text": text,
                "interval": interval
            }
        }
        response = self._request('POST', '/execute', json=task_data)
        response.raise_for_status()
        return response.json()
    
    def keyboard_press(self, key: Union[str, List[str]]) -> Dict[str, Any]:
        """Press keyboard key(s)
        
        Args:
            key: Single key or list of keys for hotkey
            
        Returns:
            Response data
            
        Raises:
            requests.HTTPError: If request fails
        """
        task_data = {
            "task_type": "key_press",
            "parameters": {
                "key": key
            }
        }
        response = self._request('POST', '/execute', json=task_data)
        response.raise_for_status()
        return response.json()
    
    def find_element(self, 
                    screenshot_data: Optional[str] = None,
                    text: Optional[str] = None,
                    element_type: Optional[str] = None) -> Dict[str, Any]:
        """Find UI element using OCR
        
        Args:
            screenshot_data: Base64 screenshot data (captures new if None)
            text: Text to search for
            element_type: Type of element to find
            
        Returns:
            Element location data
            
        Raises:
            requests.HTTPError: If request fails
        """
        data = {}
        if screenshot_data:
            data["screenshot"] = screenshot_data
        if text:
            data["text"] = text
        if element_type:
            data["element_type"] = element_type
            
        response = self._request('POST', '/find-element', json=data)
        response.raise_for_status()
        return response.json()
    
    def ai_action(self, 
                  task: str,
                  screenshot: Optional[str] = None,
                  context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Execute AI-driven action
        
        Args:
            task: Task description for AI
            screenshot: Optional screenshot data
            context: Optional context information
            
        Returns:
            AI response with actions taken
            
        Raises:
            requests.HTTPError: If request fails
        """
        data = {"command": task}
        if context:
            data["context"] = str(context)
            
        response = self._request('POST', '/execute/ai', json=data)
        response.raise_for_status()
        return response.json()
    
    def __enter__(self):
        """Context manager entry"""
        return self
        
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit - close session"""
        self.session.close()