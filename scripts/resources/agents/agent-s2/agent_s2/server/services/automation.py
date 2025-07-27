"""Automation service for Agent S2 - handles mouse and keyboard operations"""

import logging
import time
from typing import Optional, Tuple, List, Union

import pyautogui

logger = logging.getLogger(__name__)


class AutomationService:
    """Service for handling automation operations"""
    
    def __init__(self):
        """Initialize automation service"""
        # Configure pyautogui
        pyautogui.FAILSAFE = False  # Disable failsafe in container
        pyautogui.PAUSE = 0.1  # Small pause between actions
        
    # Mouse operations
    def get_mouse_position(self) -> Tuple[int, int]:
        """Get current mouse position
        
        Returns:
            Tuple of (x, y) coordinates
        """
        return pyautogui.position()
        
    def move_mouse(self, x: int, y: int, duration: float = 0.0) -> None:
        """Move mouse to position
        
        Args:
            x: Target X coordinate
            y: Target Y coordinate
            duration: Movement duration in seconds
        """
        if duration > 0:
            pyautogui.moveTo(x, y, duration=duration)
        else:
            pyautogui.moveTo(x, y)
            
    def move_mouse_relative(self, dx: int, dy: int, duration: float = 0.0) -> None:
        """Move mouse relative to current position
        
        Args:
            dx: X offset
            dy: Y offset
            duration: Movement duration in seconds
        """
        if duration > 0:
            pyautogui.moveRel(dx, dy, duration=duration)
        else:
            pyautogui.moveRel(dx, dy)
            
    def click(self, 
              x: Optional[int] = None,
              y: Optional[int] = None,
              button: str = "left",
              clicks: int = 1) -> None:
        """Click mouse button
        
        Args:
            x: X coordinate (uses current if None)
            y: Y coordinate (uses current if None)
            button: Mouse button (left, right, middle)
            clicks: Number of clicks
            
        Raises:
            ValueError: If button is invalid
        """
        if button not in ["left", "right", "middle"]:
            raise ValueError(f"Invalid button: {button}")
            
        if x is not None and y is not None:
            pyautogui.click(x, y, button=button, clicks=clicks)
        else:
            pyautogui.click(button=button, clicks=clicks)
            
    def double_click(self, x: Optional[int] = None, y: Optional[int] = None) -> None:
        """Double click at position
        
        Args:
            x: X coordinate (uses current if None)
            y: Y coordinate (uses current if None)
        """
        if x is not None and y is not None:
            pyautogui.doubleClick(x, y)
        else:
            pyautogui.doubleClick()
            
    def right_click(self, x: Optional[int] = None, y: Optional[int] = None) -> None:
        """Right click at position
        
        Args:
            x: X coordinate (uses current if None)
            y: Y coordinate (uses current if None)
        """
        if x is not None and y is not None:
            pyautogui.rightClick(x, y)
        else:
            pyautogui.rightClick()
            
    def drag(self,
             start_x: int,
             start_y: int,
             end_x: int,
             end_y: int,
             duration: float = 1.0,
             button: str = "left") -> None:
        """Drag from one position to another
        
        Args:
            start_x: Starting X coordinate
            start_y: Starting Y coordinate
            end_x: Ending X coordinate
            end_y: Ending Y coordinate
            duration: Drag duration in seconds
            button: Mouse button to hold
        """
        pyautogui.moveTo(start_x, start_y)
        pyautogui.dragTo(end_x, end_y, duration=duration, button=button)
        
    def scroll(self, clicks: int, x: Optional[int] = None, y: Optional[int] = None) -> None:
        """Scroll mouse wheel
        
        Args:
            clicks: Number of scroll clicks (positive=up, negative=down)
            x: X coordinate (uses current if None)
            y: Y coordinate (uses current if None)
        """
        if x is not None and y is not None:
            pyautogui.scroll(clicks, x=x, y=y)
        else:
            pyautogui.scroll(clicks)
            
    # Keyboard operations
    def type_text(self, text: str, interval: float = 0.0) -> None:
        """Type text
        
        Args:
            text: Text to type
            interval: Delay between keystrokes in seconds
        """
        if interval > 0:
            pyautogui.typewrite(text, interval=interval)
        else:
            pyautogui.typewrite(text)
            
    def press_key(self, key: Union[str, List[str]]) -> None:
        """Press key(s)
        
        Args:
            key: Single key or list of keys for hotkey
        """
        if isinstance(key, list):
            pyautogui.hotkey(*key)
        else:
            pyautogui.press(key)
            
    def key_down(self, key: str) -> None:
        """Hold key down
        
        Args:
            key: Key to hold down
        """
        pyautogui.keyDown(key)
        
    def key_up(self, key: str) -> None:
        """Release key
        
        Args:
            key: Key to release
        """
        pyautogui.keyUp(key)
        
    def hotkey(self, *keys: str) -> None:
        """Press hotkey combination
        
        Args:
            *keys: Keys to press together
        """
        pyautogui.hotkey(*keys)
        
    # Combined operations
    def click_and_type(self,
                      x: int,
                      y: int,
                      text: str,
                      clear_first: bool = False) -> None:
        """Click at position and type text
        
        Args:
            x: X coordinate
            y: Y coordinate
            text: Text to type
            clear_first: Whether to select all before typing
        """
        self.click(x, y)
        time.sleep(0.1)
        
        if clear_first:
            self.hotkey('ctrl', 'a')
            time.sleep(0.1)
            
        self.type_text(text)
        
    # Utility functions
    def get_screen_size(self) -> Tuple[int, int]:
        """Get screen size
        
        Returns:
            Tuple of (width, height)
        """
        return pyautogui.size()
        
    def is_on_screen(self, x: int, y: int) -> bool:
        """Check if coordinates are on screen
        
        Args:
            x: X coordinate
            y: Y coordinate
            
        Returns:
            True if on screen, False otherwise
        """
        width, height = self.get_screen_size()
        return 0 <= x < width and 0 <= y < height
        
    def safe_click(self,
                  x: int,
                  y: int,
                  button: str = "left",
                  clicks: int = 1) -> bool:
        """Click only if coordinates are on screen
        
        Args:
            x: X coordinate
            y: Y coordinate
            button: Mouse button
            clicks: Number of clicks
            
        Returns:
            True if clicked, False if coordinates were off-screen
        """
        if self.is_on_screen(x, y):
            self.click(x, y, button, clicks)
            return True
        return False