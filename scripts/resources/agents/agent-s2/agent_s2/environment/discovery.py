"""Environment Discovery Module

Detects available capabilities, applications, and environment characteristics
for both sandbox and host modes.
"""

import os
import glob
import shutil
import subprocess
import logging
from typing import Dict, List, Any, Optional, Tuple
from pathlib import Path

from ..config import Config, AgentMode

logger = logging.getLogger(__name__)


class EnvironmentDiscovery:
    """Discovers environment capabilities and applications"""
    
    def __init__(self, mode: Optional[AgentMode] = None):
        """Initialize environment discovery
        
        Args:
            mode: Operating mode (defaults to current config mode)
        """
        self.mode = mode or Config.CURRENT_MODE
        self.capabilities = self._discover_capabilities()
        
    def _discover_capabilities(self) -> Dict[str, Any]:
        """Discover all environment capabilities"""
        if self.mode == AgentMode.SANDBOX:
            return self._discover_sandbox()
        elif self.mode == AgentMode.HOST:
            return self._discover_host()
        else:
            logger.error(f"Unknown mode: {self.mode}")
            return {}
    
    def _discover_sandbox(self) -> Dict[str, Any]:
        """Discover sandbox mode capabilities"""
        return {
            "mode": "sandbox",
            "display": {
                "type": "virtual",
                "server": "Xvfb",
                "display_id": Config.DISPLAY,
                "resolution": f"{Config.SCREEN_WIDTH}x{Config.SCREEN_HEIGHT}",
                "depth": Config.SCREEN_DEPTH
            },
            "window_manager": {
                "type": "fluxbox",
                "config_path": "/home/agents2/.fluxbox",
                "shortcuts": self._get_fluxbox_shortcuts(),
                "menu_system": "minimal"
            },
            "applications": self._discover_sandbox_applications(),
            "filesystem": {
                "access_level": "container_only",
                "writable_dirs": ["/tmp", "/home/agents2", "/opt/agent-s2"],
                "output_dir": Config.OUTPUT_DIR
            },
            "network": {
                "type": "bridge",
                "access_level": "internet_only",
                "localhost_blocked": True
            },
            "security": {
                "isolation_level": "high",
                "container_security": True,
                "host_access": False,
                "privileged": False
            },
            "limitations": [
                "No host filesystem access",
                "No host display access", 
                "Limited to pre-installed applications",
                "No localhost network access",
                "Virtual display only"
            ],
            "capabilities": [
                "screenshot",
                "mouse_control",
                "keyboard_control",
                "basic_applications",
                "web_browsing",
                "text_editing",
                "calculations"
            ]
        }
    
    def _discover_host(self) -> Dict[str, Any]:
        """Discover host mode capabilities"""
        try:
            desktop_env = self._detect_desktop_environment()
            host_apps = self._scan_host_applications()
            display_info = self._get_host_display_info()
            
            return {
                "mode": "host",
                "display": {
                    "type": "host" if Config.HOST_DISPLAY_ACCESS else "virtual_with_host_apps",
                    "server": display_info.get("server", "unknown"),
                    "display_id": display_info.get("display", Config.DISPLAY),
                    "resolution": display_info.get("resolution", "unknown"),
                    "desktop_environment": desktop_env
                },
                "window_manager": desktop_env,
                "applications": host_apps,
                "filesystem": {
                    "access_level": "mounted",
                    "mounts": Config.get_host_mounts(),
                    "forbidden_paths": Config.HOST_FORBIDDEN_PATHS,
                    "max_mount_size_gb": Config.HOST_MAX_MOUNT_SIZE_GB
                },
                "network": {
                    "type": "host",
                    "access_level": "full",
                    "localhost_available": True
                },
                "security": {
                    "isolation_level": "medium",
                    "security_profile": Config.HOST_SECURITY_PROFILE,
                    "audit_logging": Config.HOST_AUDIT_LOGGING,
                    "host_access": True,
                    "privileged": True
                },
                "capabilities": [
                    "screenshot",
                    "mouse_control", 
                    "keyboard_control",
                    "host_applications",
                    "file_system_access",
                    "native_desktop_integration",
                    "full_network_access",
                    "system_automation"
                ],
                "limitations": [
                    f"Limited to configured mounts: {len(Config.get_host_mounts())} paths",
                    f"Security profile: {Config.HOST_SECURITY_PROFILE}",
                    "Host access requires elevated permissions"
                ]
            }
        except Exception as e:
            logger.error(f"Failed to discover host capabilities: {e}")
            # Fallback to basic host configuration
            return self._get_fallback_host_config()
    
    def _discover_sandbox_applications(self) -> Dict[str, Dict[str, Any]]:
        """Discover applications available in sandbox mode"""
        apps = {}
        
        for app_name in Config.SANDBOX_APPLICATIONS:
            app_info = self._get_sandbox_app_info(app_name)
            if app_info:
                apps[app_name] = app_info
                
        return apps
    
    def _get_sandbox_app_info(self, app_name: str) -> Optional[Dict[str, Any]]:
        """Get information about a sandbox application"""
        app_configs = {
            "firefox-esr": {
                "name": "Firefox ESR",
                "command": "firefox",
                "category": "web_browser", 
                "launcher": "Alt+F1 → firefox",
                "description": "Mozilla Firefox web browser",
                "capabilities": ["web_browsing", "javascript", "downloads"]
            },
            "mousepad": {
                "name": "Mousepad",
                "command": "mousepad",
                "category": "text_editor",
                "launcher": "Alt+F1 → mousepad",
                "description": "Simple text editor",
                "capabilities": ["text_editing", "syntax_highlighting"]
            },
            "gedit": {
                "name": "GEdit", 
                "command": "gedit",
                "category": "text_editor",
                "launcher": "Alt+F1 → gedit",
                "description": "GNOME text editor",
                "capabilities": ["text_editing", "plugins", "syntax_highlighting"]
            },
            "gnome-calculator": {
                "name": "Calculator",
                "command": "gnome-calculator", 
                "category": "utility",
                "launcher": "Alt+F1 → gnome-calculator",
                "description": "GNOME calculator",
                "capabilities": ["calculations", "scientific_mode"]
            },
            "xterm": {
                "name": "Terminal",
                "command": "xterm",
                "category": "terminal",
                "launcher": "Alt+F1 → xterm", 
                "description": "X terminal emulator",
                "capabilities": ["command_line", "shell_access"]
            }
        }
        
        return app_configs.get(app_name)
    
    def _scan_host_applications(self) -> Dict[str, Dict[str, Any]]:
        """Scan for applications available on the host system"""
        apps = {}
        
        # Scan desktop files
        desktop_dirs = [
            "/usr/share/applications",
            "/usr/local/share/applications", 
            "~/.local/share/applications"
        ]
        
        for desktop_dir in desktop_dirs:
            expanded_dir = os.path.expanduser(desktop_dir)
            if os.path.exists(expanded_dir):
                for desktop_file in glob.glob(f"{expanded_dir}/*.desktop"):
                    app_info = self._parse_desktop_file(desktop_file)
                    if app_info and self._is_application_allowed(app_info["name"]):
                        apps[app_info["name"]] = app_info
        
        # Scan common binary locations
        binary_apps = self._scan_binary_applications()
        apps.update(binary_apps)
        
        return apps
    
    def _parse_desktop_file(self, desktop_file: str) -> Optional[Dict[str, Any]]:
        """Parse a .desktop file to extract application information"""
        try:
            app_info = {
                "desktop_file": desktop_file,
                "category": "unknown",
                "capabilities": []
            }
            
            with open(desktop_file, 'r', encoding='utf-8') as f:
                in_desktop_entry = False
                for line in f:
                    line = line.strip()
                    
                    if line == "[Desktop Entry]":
                        in_desktop_entry = True
                        continue
                    elif line.startswith("[") and line.endswith("]"):
                        in_desktop_entry = False
                        continue
                    
                    if not in_desktop_entry or "=" not in line:
                        continue
                    
                    key, value = line.split("=", 1)
                    
                    if key == "Name":
                        app_info["name"] = value
                    elif key == "Exec":
                        app_info["command"] = value.split()[0]  # First word is the command
                    elif key == "Comment":
                        app_info["description"] = value
                    elif key == "Categories":
                        app_info["category"] = value.split(";")[0].lower()
                    elif key == "NoDisplay" and value.lower() == "true":
                        return None  # Skip hidden applications
            
            # Ensure required fields exist
            if "name" not in app_info or "command" not in app_info:
                return None
            
            # Add launcher method (host mode uses native desktop)
            app_info["launcher"] = f"Native desktop → {app_info['name']}"
            
            # Determine capabilities based on category
            app_info["capabilities"] = self._get_app_capabilities(app_info["category"])
            
            return app_info
            
        except Exception as e:
            logger.debug(f"Failed to parse desktop file {desktop_file}: {e}")
            return None
    
    def _scan_binary_applications(self) -> Dict[str, Dict[str, Any]]:
        """Scan for applications in common binary locations"""
        apps = {}
        common_binaries = [
            "firefox", "chrome", "chromium", "code", "vim", "nano", 
            "git", "python3", "node", "npm", "docker", "kubectl"
        ]
        
        for binary in common_binaries:
            if self._is_application_allowed(binary):
                path = shutil.which(binary)
                if path:
                    apps[binary] = {
                        "name": binary.title(),
                        "command": binary,
                        "path": path,
                        "category": self._guess_category(binary),
                        "launcher": f"Command line → {binary}",
                        "description": f"{binary} command line application",
                        "capabilities": self._get_app_capabilities(self._guess_category(binary))
                    }
        
        return apps
    
    def _is_application_allowed(self, app_name: str) -> bool:
        """Check if application is allowed in current configuration"""
        allowed_apps = Config.get_allowed_applications()
        return "*" in allowed_apps or app_name.lower() in [app.lower() for app in allowed_apps]
    
    def _get_app_capabilities(self, category: str) -> List[str]:
        """Get capabilities based on application category"""
        capability_map = {
            "webbrowser": ["web_browsing", "downloads", "javascript"],
            "texteditor": ["text_editing", "syntax_highlighting"],
            "development": ["code_editing", "debugging", "version_control"],
            "terminal": ["command_line", "shell_access", "system_admin"],
            "graphics": ["image_editing", "drawing", "design"],
            "office": ["document_editing", "spreadsheets", "presentations"],
            "multimedia": ["media_playback", "audio_editing", "video_editing"],
            "utility": ["system_tools", "file_management"],
            "game": ["entertainment"],
            "unknown": ["basic_application"]
        }
        
        return capability_map.get(category.lower(), ["basic_application"])
    
    def _guess_category(self, binary_name: str) -> str:
        """Guess application category from binary name"""
        category_patterns = {
            "webbrowser": ["firefox", "chrome", "chromium", "safari"],
            "texteditor": ["vim", "nano", "emacs", "gedit"],
            "development": ["code", "git", "python", "node", "npm", "docker", "kubectl"],
            "terminal": ["bash", "zsh", "fish", "sh"],
            "multimedia": ["vlc", "mpv", "gimp", "blender"]
        }
        
        for category, patterns in category_patterns.items():
            if any(pattern in binary_name.lower() for pattern in patterns):
                return category
        
        return "utility"
    
    def _detect_desktop_environment(self) -> Dict[str, Any]:
        """Detect the desktop environment on the host system"""
        desktop_env = {
            "name": "unknown",
            "type": "unknown",
            "window_manager": "unknown",
            "capabilities": []
        }
        
        # Check environment variables
        desktop_session = os.environ.get("DESKTOP_SESSION", "").lower()
        xdg_current_desktop = os.environ.get("XDG_CURRENT_DESKTOP", "").lower()
        
        if "gnome" in desktop_session or "gnome" in xdg_current_desktop:
            desktop_env.update({
                "name": "GNOME",
                "type": "wayland/x11",
                "window_manager": "mutter",
                "capabilities": ["activities", "extensions", "workspaces"]
            })
        elif "kde" in desktop_session or "plasma" in xdg_current_desktop:
            desktop_env.update({
                "name": "KDE Plasma", 
                "type": "x11/wayland",
                "window_manager": "kwin",
                "capabilities": ["activities", "widgets", "effects"]
            })
        elif "xfce" in desktop_session:
            desktop_env.update({
                "name": "XFCE",
                "type": "x11",
                "window_manager": "xfwm4",
                "capabilities": ["panels", "workspaces", "thunar"]
            })
        elif "fluxbox" in desktop_session:
            desktop_env.update({
                "name": "Fluxbox",
                "type": "x11", 
                "window_manager": "fluxbox",
                "capabilities": ["minimal", "lightweight"]
            })
        
        return desktop_env
    
    def _get_host_display_info(self) -> Dict[str, Any]:
        """Get host display information"""
        display_info = {
            "display": os.environ.get("DISPLAY", ":0"),
            "server": "unknown",
            "resolution": "unknown"
        }
        
        try:
            # Try to get display information
            if shutil.which("xdpyinfo"):
                result = subprocess.run(
                    ["xdpyinfo"], 
                    capture_output=True, 
                    text=True, 
                    timeout=5
                )
                if result.returncode == 0:
                    # Parse xdpyinfo output
                    lines = result.stdout.split("\n")
                    for line in lines:
                        if "version number" in line:
                            display_info["server"] = "X11"
                        elif "dimensions:" in line:
                            # Extract resolution
                            parts = line.split()
                            for part in parts:
                                if "x" in part and part.replace("x", "").replace("pixels", "").isdigit():
                                    display_info["resolution"] = part.split("x")[0] + "x" + part.split("x")[1].split()[0]
                                    break
        except (subprocess.TimeoutExpired, subprocess.SubprocessError) as e:
            logger.debug(f"Failed to get display info: {e}")
        
        return display_info
    
    def _get_fluxbox_shortcuts(self) -> Dict[str, str]:
        """Get Fluxbox keyboard shortcuts"""
        return {
            "Alt+F1": "Open terminal",
            "Alt+F2": "Run command dialog", 
            "Alt+Tab": "Switch windows",
            "Right-click desktop": "Context menu",
            "Middle-click desktop": "Workspace menu"
        }
    
    def _get_fallback_host_config(self) -> Dict[str, Any]:
        """Get fallback configuration when host discovery fails"""
        return {
            "mode": "host",
            "display": {
                "type": "host_fallback",
                "server": "unknown",
                "display_id": Config.DISPLAY,
                "resolution": "unknown"
            },
            "window_manager": {"name": "unknown", "type": "unknown"},
            "applications": {},
            "filesystem": {
                "access_level": "mounted",
                "mounts": Config.get_host_mounts()
            },
            "network": {"type": "host", "access_level": "full"},
            "security": {
                "isolation_level": "medium",
                "security_profile": Config.HOST_SECURITY_PROFILE
            },
            "capabilities": ["basic_host_access"],
            "limitations": ["Host discovery failed - limited functionality"]
        }
    
    def get_available_actions(self) -> List[str]:
        """Get list of available actions for current mode"""
        base_actions = [
            "screenshot",
            "mouse_click", 
            "mouse_move",
            "keyboard_type",
            "keyboard_key",
            "window_management"
        ]
        
        if self.mode == AgentMode.HOST:
            base_actions.extend([
                "launch_native_application",
                "file_system_operations", 
                "system_integration",
                "host_automation"
            ])
        
        return base_actions
    
    def get_system_prompt_context(self) -> str:
        """Generate context information for AI system prompt"""
        caps = self.capabilities
        
        if self.mode == AgentMode.SANDBOX:
            apps_list = "\n".join([
                f"- {info['name']}: {info['launcher']}" 
                for info in caps['applications'].values()
            ])
            
            return f"""
OPERATING MODE: SANDBOX (Secure, Isolated)

AVAILABLE APPLICATIONS:
{apps_list}

DESKTOP ENVIRONMENT:
- Window Manager: {caps['window_manager']['type']}
- Display: {caps['display']['type']} ({caps['display']['resolution']})
- Shortcuts: {', '.join(caps['window_manager']['shortcuts'].keys())}

CAPABILITIES:
{', '.join(caps['capabilities'])}

LIMITATIONS:
{chr(10).join([f'- {limit}' for limit in caps['limitations']])}

INSTRUCTIONS:
Use keyboard shortcuts (Alt+F1 for terminal, Alt+F2 for run dialog) to launch applications.
Right-click desktop for context menu. Applications are pre-installed and isolated.
"""
        
        elif self.mode == AgentMode.HOST:
            apps_count = len(caps['applications'])
            mounts_list = "\n".join([
                f"- {mount.get('host', 'unknown')} → {mount.get('container', 'unknown')}"
                for mount in caps['filesystem']['mounts']
            ])
            
            return f"""
OPERATING MODE: HOST (Extended Access)

SYSTEM INFORMATION:
- Desktop Environment: {caps['display']['desktop_environment']['name']}
- Display: {caps['display']['type']} ({caps['display']['resolution']})
- Available Applications: {apps_count} detected

FILESYSTEM ACCESS:
{mounts_list if mounts_list.strip() else '- No custom mounts configured'}

CAPABILITIES:
{', '.join(caps['capabilities'])}

SECURITY CONSTRAINTS:
- Profile: {caps['security']['security_profile']}
- Forbidden paths: {', '.join(caps['security']['forbidden_paths'][:3])}{'...' if len(caps['security']['forbidden_paths']) > 3 else ''}
- Audit logging: {'enabled' if caps['security']['audit_logging'] else 'disabled'}

INSTRUCTIONS:
You have access to the host system with native desktop integration.
Use standard desktop methods to launch applications and manage files.
Respect security constraints and mounted directory limitations.
"""
        
        return "Unknown operating mode - limited functionality available."