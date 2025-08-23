"""Window Manager Service for Target-Aware Automation

Provides window detection, management, and focusing capabilities
for reliable application targeting in automation tasks.
"""

import logging
import subprocess
import time
import re
from datetime import datetime
from typing import Dict, List, Optional, Any
from dataclasses import dataclass

logger = logging.getLogger(__name__)


@dataclass
class WindowInfo:
    """Information about a window"""
    window_id: str
    title: str
    app_name: str
    process_id: int
    geometry: Dict[str, int]  # x, y, width, height
    is_focused: bool
    last_active: datetime


class WindowManagerError(Exception):
    """Window manager specific exceptions"""
    pass


class WindowManager:
    """Window management service for target-aware automation"""
    
    def __init__(self):
        self.known_apps = {
            "firefox": ["Firefox", "Mozilla Firefox", "firefox", "firefox-esr"],
            "terminal": ["xterm", "Terminal", "Xterm", "gnome-terminal"],
            "calculator": ["Calculator", "gnome-calculator", "calc"],
            "mousepad": ["Mousepad", "mousepad"],
            "gedit": ["Gedit", "gedit", "Text Editor"]
        }
        self._window_cache = {}
        self._last_cache_update = 0
        self.cache_timeout = 1.0  # Cache windows for 1 second
    
    def _run_command(self, command: List[str]) -> str:
        """Run system command and return output"""
        try:
            result = subprocess.run(
                command, 
                capture_output=True, 
                text=True, 
                timeout=10
            )
            if result.returncode != 0:
                logger.warning(f"Command {' '.join(command)} failed: {result.stderr}")
                return ""
            return result.stdout.strip()
        except subprocess.TimeoutExpired:
            logger.error(f"Command {' '.join(command)} timed out")
            return ""
        except Exception as e:
            logger.error(f"Command {' '.join(command)} failed: {e}")
            return ""
    
    def _parse_wmctrl_output(self, output: str) -> List[WindowInfo]:
        """Parse wmctrl window list output"""
        windows = []
        current_time = datetime.now()
        
        for line in output.split('\n'):
            if not line.strip():
                continue
                
            # wmctrl -lG format: window_id desktop x y width height hostname title
            parts = line.split(None, 7)
            if len(parts) < 8:
                continue
                
            try:
                window_id = parts[0]
                x, y, width, height = map(int, parts[2:6])
                title = parts[7] if len(parts) > 7 else ""
                
                # Get process info for this window
                process_id = self._get_window_process_id(window_id)
                app_name = self._detect_app_name(title, process_id)
                is_focused = self._is_window_focused(window_id)
                
                window_info = WindowInfo(
                    window_id=window_id,
                    title=title,
                    app_name=app_name,
                    process_id=process_id,
                    geometry={"x": x, "y": y, "width": width, "height": height},
                    is_focused=is_focused,
                    last_active=current_time
                )
                windows.append(window_info)
                
            except (ValueError, IndexError) as e:
                logger.warning(f"Failed to parse window line: {line}, error: {e}")
                continue
                
        return windows
    
    def _get_window_process_id(self, window_id: str) -> int:
        """Get process ID for a window"""
        try:
            output = self._run_command(["xprop", "-id", window_id, "_NET_WM_PID"])
            if output:
                match = re.search(r'_NET_WM_PID\(CARDINAL\) = (\d+)', output)
                if match:
                    return int(match.group(1))
        except Exception as e:
            logger.debug(f"Could not get PID for window {window_id}: {e}")
        return 0
    
    def _detect_app_name(self, title: str, process_id: int) -> str:
        """Detect application name from window title and process"""
        # Try to match against known applications
        title_lower = title.lower()
        for app_key, app_names in self.known_apps.items():
            for app_name in app_names:
                if app_name.lower() in title_lower:
                    return app_key
        
        # Try to get process name
        if process_id > 0:
            try:
                with open(f"/proc/{process_id}/comm", "r") as f:
                    process_name = f.read().strip()
                    for app_key, app_names in self.known_apps.items():
                        if process_name in app_names:
                            return app_key
                    return process_name
            except Exception:
                pass
        
        # Fallback to extracting from title
        return title.split()[0] if title else "unknown"
    
    def _is_window_focused(self, window_id: str) -> bool:
        """Check if window is currently focused"""
        try:
            output = self._run_command(["xprop", "-root", "_NET_ACTIVE_WINDOW"])
            if output:
                match = re.search(r'_NET_ACTIVE_WINDOW\(WINDOW\): window id # (0x[0-9a-f]+)', output)
                if match:
                    active_id = match.group(1)
                    return active_id == window_id
        except Exception as e:
            logger.debug(f"Could not check focus for window {window_id}: {e}")
        return False
    
    def _update_window_cache(self) -> None:
        """Update the window cache if needed"""
        current_time = time.time()
        if current_time - self._last_cache_update > self.cache_timeout:
            output = self._run_command(["wmctrl", "-lG"])
            if output:
                self._window_cache = {w.window_id: w for w in self._parse_wmctrl_output(output)}
                self._last_cache_update = current_time
    
    def get_running_applications(self) -> List[str]:
        """Get list of running GUI applications"""
        self._update_window_cache()
        apps = set()
        for window in self._window_cache.values():
            if window.app_name and window.app_name != "unknown":
                apps.add(window.app_name)
        return sorted(list(apps))
    
    def get_application_windows(self, app_name: str) -> List[WindowInfo]:
        """Get all windows for specific application"""
        self._update_window_cache()
        windows = []
        
        # Check both direct matches and known app aliases
        app_aliases = self.known_apps.get(app_name.lower(), [app_name])
        
        for window in self._window_cache.values():
            if (window.app_name.lower() == app_name.lower() or 
                any(alias.lower() in window.title.lower() for alias in app_aliases)):
                windows.append(window)
        
        # Sort by last active time (most recent first)
        windows.sort(key=lambda w: w.last_active, reverse=True)
        return windows
    
    def get_focused_window(self) -> Optional[WindowInfo]:
        """Get currently focused window"""
        self._update_window_cache()
        for window in self._window_cache.values():
            if window.is_focused:
                return window
        return None
    
    def focus_application(self, app_name: str, window_criteria: Dict = None) -> bool:
        """Focus specific application with optional window selection"""
        try:
            windows = self.get_application_windows(app_name)
            if not windows:
                logger.warning(f"No windows found for application: {app_name}")
                return False
            
            # Select window based on criteria
            target_window = self._select_window_by_criteria(windows, window_criteria or {})
            if not target_window:
                logger.warning(f"No window matches criteria for {app_name}")
                return False
            
            return self.focus_window_by_id(target_window.window_id)
            
        except Exception as e:
            logger.error(f"Failed to focus application {app_name}: {e}")
            return False
    
    def _select_window_by_criteria(self, windows: List[WindowInfo], criteria: Dict) -> Optional[WindowInfo]:
        """Select window based on criteria"""
        if not windows:
            return None
        
        # If no criteria, prefer most recent or focused window
        if not criteria:
            focused = next((w for w in windows if w.is_focused), None)
            return focused or windows[0]
        
        candidates = windows.copy()
        
        # Filter by title contains
        if "title_contains" in criteria:
            title_filter = criteria["title_contains"].lower()
            candidates = [w for w in candidates if title_filter in w.title.lower()]
        
        # Filter by title regex
        if "title_matches" in criteria:
            try:
                pattern = re.compile(criteria["title_matches"], re.IGNORECASE)
                candidates = [w for w in candidates if pattern.search(w.title)]
            except re.error as e:
                logger.warning(f"Invalid regex pattern: {e}")
        
        # Filter by URL (for browser windows)
        if "url_contains" in criteria and candidates:
            # This would require additional logic to get URL from browser windows
            # For now, just log and continue
            logger.debug("URL filtering not yet implemented")
        
        # Explicit window ID
        if "window_id" in criteria:
            window_id = criteria["window_id"]
            candidates = [w for w in candidates if w.window_id == window_id]
        
        if not candidates:
            return None
        
        # Prefer focused window, then most recent
        if criteria.get("prefer_recent", True):
            focused = next((w for w in candidates if w.is_focused), None)
            return focused or candidates[0]
        
        return candidates[0]
    
    def focus_window_by_id(self, window_id: str) -> bool:
        """Focus specific window by ID"""
        try:
            # Use wmctrl to activate window
            result = self._run_command(["wmctrl", "-ia", window_id])
            
            # Give it a moment to process
            time.sleep(0.1)
            
            # Verify focus was successful
            return self.verify_focus_by_id(window_id, timeout=1.0)
            
        except Exception as e:
            logger.error(f"Failed to focus window {window_id}: {e}")
            return False
    
    def verify_focus_by_id(self, window_id: str, timeout: float = 2.0) -> bool:
        """Verify window gained focus within timeout"""
        start_time = time.time()
        while time.time() - start_time < timeout:
            if self._is_window_focused(window_id):
                return True
            time.sleep(0.1)
        return False
    
    def start_application(self, app_name: str, args: List[str] = None) -> bool:
        """Start application if not running"""
        try:
            # Check if already running
            if self.get_application_windows(app_name):
                logger.info(f"Application {app_name} is already running")
                return True
            
            # Map app names to executable commands
            app_commands = {
                "firefox": ["firefox-esr"],
                "terminal": ["xterm"],
                "calculator": ["gnome-calculator"],
                "mousepad": ["mousepad"],
                "gedit": ["gedit"]
            }
            
            command = app_commands.get(app_name.lower())
            if not command:
                logger.warning(f"Don't know how to start application: {app_name}")
                return False
            
            if args:
                command.extend(args)
            
            # Start application in background
            subprocess.Popen(
                command,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL
            )
            
            # Wait a moment for application to start
            time.sleep(2.0)
            
            # Verify it started
            return bool(self.get_application_windows(app_name))
            
        except Exception as e:
            logger.error(f"Failed to start application {app_name}: {e}")
            return False
    
    def verify_focus(self, app_name: str, timeout: float = 2.0) -> bool:
        """Verify application gained focus within timeout"""
        start_time = time.time()
        while time.time() - start_time < timeout:
            focused_window = self.get_focused_window()
            if focused_window and focused_window.app_name.lower() == app_name.lower():
                return True
            time.sleep(0.1)
        return False
    
    def get_window_info(self, window_id: str) -> Optional[WindowInfo]:
        """Get information about specific window"""
        self._update_window_cache()
        return self._window_cache.get(window_id)
    
    def list_all_windows(self) -> List[WindowInfo]:
        """Get list of all windows"""
        self._update_window_cache()
        return list(self._window_cache.values())