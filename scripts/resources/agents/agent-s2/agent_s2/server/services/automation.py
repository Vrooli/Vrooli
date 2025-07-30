"""Automation service for Agent S2 - handles mouse and keyboard operations"""

import logging
import time
from typing import Optional, Tuple, List, Union, Dict, Any

import pyautogui

from .window_manager import WindowManager, WindowInfo, WindowManagerError

logger = logging.getLogger(__name__)


class AutomationError(Exception):
    """Automation specific exceptions"""
    pass


class AutomationService:
    """Service for handling automation operations"""
    
    def __init__(self):
        """Initialize automation service"""
        # Configure pyautogui
        pyautogui.FAILSAFE = False  # Disable failsafe in container
        pyautogui.PAUSE = 0.1  # Small pause between actions
        
        # Initialize window manager for target-aware automation
        self.window_manager = WindowManager()
    
    # Target-aware automation methods
    def _ensure_target_focus(self, target_app: str, window_criteria: Dict = None) -> Tuple[bool, Optional[WindowInfo], float]:
        """Ensure target application has focus before automation
        
        Args:
            target_app: Target application name
            window_criteria: Optional window selection criteria
            
        Returns:
            Tuple of (success, focused_window_info, focus_time)
        """
        if not target_app:
            return True, None, 0.0  # No target specified, proceed
        
        start_time = time.time()
        
        try:
            # Check if already focused
            focused = self.window_manager.get_focused_window()
            if focused and self._matches_target(focused, target_app):
                return True, focused, time.time() - start_time
            
            # Need to focus target app
            success = self.window_manager.focus_application(target_app, window_criteria or {})
            focus_time = time.time() - start_time
            
            if success:
                # Get the focused window info
                focused_window = self.window_manager.get_focused_window()
                return True, focused_window, focus_time
            else:
                logger.warning(f"Could not focus target application: {target_app}")
                return False, None, focus_time
                
        except Exception as e:
            focus_time = time.time() - start_time
            logger.error(f"Error ensuring target focus for {target_app}: {e}")
            return False, None, focus_time
    
    def _matches_target(self, window: WindowInfo, target_app: str) -> bool:
        """Check if window matches target application"""
        return window.app_name.lower() == target_app.lower()
    
    def click_targeted(self, x: int, y: int, target_app: str = None, 
                      button: str = "left", clicks: int = 1, 
                      window_criteria: Dict = None) -> Tuple[bool, Optional[WindowInfo], float]:
        """Enhanced click with target awareness
        
        Args:
            x: X coordinate
            y: Y coordinate  
            target_app: Target application name
            button: Mouse button
            clicks: Number of clicks
            window_criteria: Window selection criteria
            
        Returns:
            Tuple of (success, focused_window_info, focus_time)
        """
        if target_app:
            success, focused_window, focus_time = self._ensure_target_focus(target_app, window_criteria)
            if not success:
                raise AutomationError(f"Could not focus target application: {target_app}")
        else:
            focused_window, focus_time = None, 0.0
        
        # Proceed with click
        self.click(x, y, button, clicks)
        return True, focused_window, focus_time
    
    def type_text_targeted(self, text: str, target_app: str = None, 
                          interval: float = 0.0, window_criteria: Dict = None) -> Tuple[bool, Optional[WindowInfo], float]:
        """Enhanced typing with target awareness
        
        Args:
            text: Text to type
            target_app: Target application name
            interval: Typing interval
            window_criteria: Window selection criteria
            
        Returns:
            Tuple of (success, focused_window_info, focus_time)
        """
        if target_app:
            success, focused_window, focus_time = self._ensure_target_focus(target_app, window_criteria)
            if not success:
                raise AutomationError(f"Could not focus target application: {target_app}")
        else:
            focused_window, focus_time = None, 0.0
        
        # Proceed with typing
        self.type_text(text, interval)
        return True, focused_window, focus_time
    
    def press_key_targeted(self, keys: Union[str, List[str]], target_app: str = None,
                          window_criteria: Dict = None) -> Tuple[bool, Optional[WindowInfo], float]:
        """Enhanced key press with target awareness
        
        Args:
            keys: Key(s) to press
            target_app: Target application name
            window_criteria: Window selection criteria
            
        Returns:
            Tuple of (success, focused_window_info, focus_time)
        """
        if target_app:
            success, focused_window, focus_time = self._ensure_target_focus(target_app, window_criteria)
            if not success:
                raise AutomationError(f"Could not focus target application: {target_app}")
        else:
            focused_window, focus_time = None, 0.0
        
        # Proceed with key press
        self.press_key(keys)
        return True, focused_window, focus_time
    
    def hotkey_targeted(self, *keys: str, target_app: str = None,
                       window_criteria: Dict = None) -> Tuple[bool, Optional[WindowInfo], float]:
        """Enhanced hotkey with target awareness
        
        Args:
            *keys: Keys to press together
            target_app: Target application name
            window_criteria: Window selection criteria
            
        Returns:
            Tuple of (success, focused_window_info, focus_time)
        """
        if target_app:
            success, focused_window, focus_time = self._ensure_target_focus(target_app, window_criteria)
            if not success:
                raise AutomationError(f"Could not focus target application: {target_app}")
        else:
            focused_window, focus_time = None, 0.0
        
        # Proceed with hotkey
        self.hotkey(*keys)
        return True, focused_window, focus_time
        
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