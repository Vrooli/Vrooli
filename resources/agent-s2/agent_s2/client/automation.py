"""Agent S2 Automation Client

Specialized client for mouse and keyboard automation.
"""

from typing import Optional, List, Dict, Any, Union
import time

from .base import AgentS2Client


class AutomationClient:
    """Client specialized for automation operations"""
    
    def __init__(self, client: Optional[AgentS2Client] = None):
        """Initialize automation client
        
        Args:
            client: Base Agent S2 client (creates new if None)
        """
        self.client = client or AgentS2Client()
        
    # Mouse operations
    def get_mouse_position(self) -> tuple[int, int]:
        """Get current mouse position
        
        Returns:
            Tuple of (x, y) coordinates
        """
        pos = self.client.mouse_position()
        return pos['x'], pos['y']
    
    def move_mouse(self, x: int, y: int, duration: float = 0.0) -> None:
        """Move mouse to specific position
        
        Args:
            x: Target X coordinate
            y: Target Y coordinate
            duration: Movement duration in seconds (smooth movement)
        """
        self.client.mouse_move(x, y, duration)
        
    def move_mouse_relative(self, dx: int, dy: int, duration: float = 0.0) -> None:
        """Move mouse relative to current position
        
        Args:
            dx: X offset
            dy: Y offset
            duration: Movement duration in seconds
        """
        current_x, current_y = self.get_mouse_position()
        self.move_mouse(current_x + dx, current_y + dy, duration)
        
    def click(self, x: Optional[int] = None, y: Optional[int] = None,
              button: str = "left", clicks: int = 1) -> None:
        """Click mouse button
        
        Args:
            x: X coordinate (uses current position if None)
            y: Y coordinate (uses current position if None)
            button: Mouse button (left, right, middle)
            clicks: Number of clicks
        """
        self.client.mouse_click(button=button, x=x, y=y, clicks=clicks)
        
    def double_click(self, x: Optional[int] = None, y: Optional[int] = None) -> None:
        """Double click at position
        
        Args:
            x: X coordinate (uses current position if None)
            y: Y coordinate (uses current position if None)
        """
        self.click(x, y, clicks=2)
        
    def right_click(self, x: Optional[int] = None, y: Optional[int] = None) -> None:
        """Right click at position
        
        Args:
            x: X coordinate (uses current position if None)
            y: Y coordinate (uses current position if None)
        """
        self.click(x, y, button="right")
        
    def drag(self, start_x: int, start_y: int, end_x: int, end_y: int,
             duration: float = 1.0, button: str = "left") -> None:
        """Drag from one position to another
        
        Args:
            start_x: Starting X coordinate
            start_y: Starting Y coordinate
            end_x: Ending X coordinate
            end_y: Ending Y coordinate
            duration: Drag duration in seconds
            button: Mouse button to hold
        """
        # Move to start position
        self.move_mouse(start_x, start_y)
        time.sleep(0.1)
        
        # TODO: Implement mouse drag when API supports it
        # For now, we simulate with move and click
        self.move_mouse(end_x, end_y, duration)
        
    # Keyboard operations
    def type_text(self, text: str, interval: float = 0.0) -> None:
        """Type text with optional delay between keystrokes
        
        Args:
            text: Text to type
            interval: Delay between keystrokes in seconds
        """
        self.client.keyboard_type(text, interval)
        
    def press_key(self, key: Union[str, List[str]]) -> None:
        """Press single key or key combination
        
        Args:
            key: Single key name or list of keys for combination
        """
        self.client.keyboard_press(key)
        
    def hotkey(self, *keys: str) -> None:
        """Press key combination (hotkey)
        
        Args:
            *keys: Keys to press together (e.g., 'ctrl', 'a')
        """
        self.press_key(list(keys))
    
    def key_combination(self, keys: List[str]) -> None:
        """Press key combination (alias for hotkey)
        
        Args:
            keys: List of keys to press together (e.g., ['ctrl', 'a'])
        """
        self.press_key(keys)
        
    # Common keyboard shortcuts
    def select_all(self) -> None:
        """Select all (Ctrl+A)"""
        self.hotkey('ctrl', 'a')
        
    def copy(self) -> None:
        """Copy (Ctrl+C)"""
        self.hotkey('ctrl', 'c')
        
    def paste(self) -> None:
        """Paste (Ctrl+V)"""
        self.hotkey('ctrl', 'v')
        
    def cut(self) -> None:
        """Cut (Ctrl+X)"""
        self.hotkey('ctrl', 'x')
        
    def undo(self) -> None:
        """Undo (Ctrl+Z)"""
        self.hotkey('ctrl', 'z')
        
    def redo(self) -> None:
        """Redo (Ctrl+Y)"""
        self.hotkey('ctrl', 'y')
        
    def save(self) -> None:
        """Save (Ctrl+S)"""
        self.hotkey('ctrl', 's')
        
    def open_file(self) -> None:
        """Open file dialog (Ctrl+O)"""
        self.hotkey('ctrl', 'o')
        
    def new_tab(self) -> None:
        """New tab (Ctrl+T)"""
        self.hotkey('ctrl', 't')
        
    def close_tab(self) -> None:
        """Close tab (Ctrl+W)"""
        self.hotkey('ctrl', 'w')
        
    def switch_tab(self, tab_number: int) -> None:
        """Switch to specific tab (Ctrl+1-9)
        
        Args:
            tab_number: Tab number (1-9)
        """
        if 1 <= tab_number <= 9:
            self.hotkey('ctrl', str(tab_number))
        else:
            raise ValueError("Tab number must be between 1 and 9")
            
    # Special keys
    def press_enter(self) -> None:
        """Press Enter key"""
        self.press_key('enter')
        
    def press_escape(self) -> None:
        """Press Escape key"""
        self.press_key('escape')
        
    def press_tab(self) -> None:
        """Press Tab key"""
        self.press_key('tab')
        
    def press_backspace(self) -> None:
        """Press Backspace key"""
        self.press_key('backspace')
        
    def press_delete(self) -> None:
        """Press Delete key"""
        self.press_key('delete')
        
    def press_space(self) -> None:
        """Press Space key"""
        self.press_key('space')
        
    # Arrow keys
    def press_up(self) -> None:
        """Press Up arrow key"""
        self.press_key('up')
        
    def press_down(self) -> None:
        """Press Down arrow key"""
        self.press_key('down')
        
    def press_left(self) -> None:
        """Press Left arrow key"""
        self.press_key('left')
        
    def press_right(self) -> None:
        """Press Right arrow key"""
        self.press_key('right')
        
    # Page navigation
    def page_up(self) -> None:
        """Press Page Up key"""
        self.press_key('pageup')
        
    def page_down(self) -> None:
        """Press Page Down key"""
        self.press_key('pagedown')
        
    def home(self) -> None:
        """Press Home key"""
        self.press_key('home')
        
    def end(self) -> None:
        """Press End key"""
        self.press_key('end')
        
    # Function keys
    def press_function_key(self, number: int) -> None:
        """Press function key (F1-F12)
        
        Args:
            number: Function key number (1-12)
        """
        if 1 <= number <= 12:
            self.press_key(f'f{number}')
        else:
            raise ValueError("Function key number must be between 1 and 12")
            
    # Combined operations
    def click_and_type(self, x: int, y: int, text: str, 
                      clear_first: bool = False) -> None:
        """Click at position and type text
        
        Args:
            x: X coordinate
            y: Y coordinate
            text: Text to type
            clear_first: Whether to clear existing text first
        """
        self.click(x, y)
        time.sleep(0.1)
        
        if clear_first:
            self.select_all()
            time.sleep(0.1)
            
        self.type_text(text)
        
    def find_and_click(self, text: str, element_type: Optional[str] = None) -> bool:
        """Find element by text and click it
        
        Args:
            text: Text to search for
            element_type: Optional element type filter
            
        Returns:
            True if element was found and clicked, False otherwise
        """
        try:
            result = self.client.find_element(text=text, element_type=element_type)
            if 'x' in result and 'y' in result:
                self.click(result['x'], result['y'])
                return True
        except (requests.RequestException, KeyError, ValueError) as e:
            import logging
            logger = logging.getLogger(__name__)
            logger.debug(f"Failed to find and click element: {e}")
        return False