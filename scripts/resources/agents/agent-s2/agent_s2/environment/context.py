"""Mode Context Management

Manages agent context and system prompts based on environment discovery
and current operating mode.
"""

import logging
from typing import Dict, Any, List, Optional
from datetime import datetime

from ..config import Config, AgentMode
from .discovery import EnvironmentDiscovery

logger = logging.getLogger(__name__)


class ModeContext:
    """Manages agent context for different operating modes"""
    
    def __init__(self, mode: Optional[AgentMode] = None, discovery: Optional[EnvironmentDiscovery] = None):
        """Initialize mode context
        
        Args:
            mode: Operating mode (defaults to current config mode)
            discovery: Environment discovery instance (creates new if None)
        """
        self.mode = mode or Config.CURRENT_MODE
        self.discovery = discovery or EnvironmentDiscovery(self.mode)
        self.context_data = self._build_context_data()
        self.system_prompt = self._generate_system_prompt()
        
    def _build_context_data(self) -> Dict[str, Any]:
        """Build comprehensive context data"""
        capabilities = self.discovery.capabilities
        
        return {
            "mode": {
                "name": self.mode.value,
                "display_name": self.mode.value.title(),
                "security_level": capabilities.get("security", {}).get("isolation_level", "unknown")
            },
            "environment": {
                "display": capabilities.get("display", {}),
                "window_manager": capabilities.get("window_manager", {}),
                "desktop_environment": capabilities.get("display", {}).get("desktop_environment", {})
            },
            "applications": capabilities.get("applications", {}),
            "capabilities": capabilities.get("capabilities", []),
            "limitations": capabilities.get("limitations", []),
            "security": capabilities.get("security", {}),
            "filesystem": capabilities.get("filesystem", {}),
            "network": capabilities.get("network", {}),
            "available_actions": self.discovery.get_available_actions(),
            "timestamp": datetime.utcnow().isoformat(),
            "config_summary": self._get_config_summary()
        }
        
    def _get_config_summary(self) -> Dict[str, Any]:
        """Get relevant configuration summary"""
        return {
            "api_port": Config.API_PORT,
            "display": Config.DISPLAY,
            "ai_enabled": Config.AI_ENABLED,
            "cors_enabled": Config.ENABLE_CORS,
            "log_level": Config.LOG_LEVEL,
            "mode_specific": self._get_mode_specific_config()
        }
        
    def _get_mode_specific_config(self) -> Dict[str, Any]:
        """Get mode-specific configuration details"""
        if self.mode == AgentMode.SANDBOX:
            return {
                "allowed_applications": Config.SANDBOX_APPLICATIONS,
                "output_directory": Config.OUTPUT_DIR,
                "screen_resolution": f"{Config.SCREEN_WIDTH}x{Config.SCREEN_HEIGHT}"
            }
        elif self.mode == AgentMode.HOST:
            return {
                "host_mode_enabled": Config.HOST_MODE_ENABLED,
                "host_display_access": Config.HOST_DISPLAY_ACCESS,
                "allowed_applications": Config.get_allowed_applications(),
                "filesystem_mounts": Config.get_host_mounts(),
                "security_profile": Config.HOST_SECURITY_PROFILE,
                "audit_logging": Config.HOST_AUDIT_LOGGING,
                "forbidden_paths": Config.HOST_FORBIDDEN_PATHS
            }
        return {}
        
    def _generate_system_prompt(self) -> str:
        """Generate comprehensive system prompt for AI agents"""
        if self.mode == AgentMode.SANDBOX:
            return self._generate_sandbox_prompt()
        elif self.mode == AgentMode.HOST:
            return self._generate_host_prompt()
        else:
            return "Unknown operating mode - functionality limited."
            
    def _generate_sandbox_prompt(self) -> str:
        """Generate system prompt for sandbox mode"""
        context = self.context_data
        apps = context["applications"]
        
        # Build applications section
        apps_section = self._format_applications_list(apps, sandbox_mode=True)
        
        # Build capabilities section
        capabilities_list = ", ".join(context["capabilities"])
        
        # Build limitations section
        limitations_list = "\n".join([f"- {limitation}" for limitation in context["limitations"]])
        
        # Build shortcuts section
        shortcuts = context["environment"]["window_manager"].get("shortcuts", {})
        shortcuts_list = "\n".join([f"- {key}: {desc}" for key, desc in shortcuts.items()])
        
        return f"""# Agent S2 - SANDBOX MODE CONTEXT

## OPERATING ENVIRONMENT
You are operating in **SANDBOX MODE** - a secure, isolated environment designed for safe automation tasks.

**Mode**: {context["mode"]["display_name"]}
**Security Level**: {context["mode"]["security_level"].title()}
**Display**: {context["environment"]["display"]["type"]} ({context["environment"]["display"]["resolution"]})
**Window Manager**: {context["environment"]["window_manager"]["type"]}

## AVAILABLE APPLICATIONS
{apps_section}

## DESKTOP NAVIGATION
**Keyboard Shortcuts:**
{shortcuts_list}

**Mouse Controls:**
- Right-click desktop: Open context menu
- Middle-click desktop: Workspace menu (if available)

## CAPABILITIES
You can perform the following actions:
{capabilities_list}

## SECURITY LIMITATIONS
{limitations_list}

## BEST PRACTICES
1. **Application Launch**: Use Alt+F1 to open terminal, then type application name
2. **Run Dialog**: Use Alt+F2 for quick application launch
3. **Window Management**: Use Alt+Tab to switch between applications
4. **File Operations**: Limited to container directories only
5. **Network Access**: Internet browsing available, localhost blocked for security

## TROUBLESHOOTING
- If applications don't respond: Try Alt+F1 → terminal → application name
- If display seems frozen: Take a screenshot to assess current state
- For text input: Click target area first, then type
- For web browsing: Use firefox command or launcher

Remember: You're in a sandboxed environment. Focus on the available applications and use standard GUI automation techniques."""

    def _generate_host_prompt(self) -> str:
        """Generate system prompt for host mode"""
        context = self.context_data
        apps = context["applications"]
        
        # Build applications section
        apps_section = self._format_applications_list(apps, sandbox_mode=False)
        
        # Build filesystem access section
        mounts = context["filesystem"].get("mounts", [])
        if mounts:
            mounts_section = "\n".join([
                f"- {mount.get('host', 'unknown')} → {mount.get('container', 'unknown')} ({mount.get('mode', 'rw')})"
                for mount in mounts
            ])
        else:
            mounts_section = "- No custom filesystem mounts configured"
            
        # Build security constraints
        security = context["security"]
        forbidden_paths = security.get("forbidden_paths", [])
        forbidden_list = ", ".join(forbidden_paths[:5]) + ("..." if len(forbidden_paths) > 5 else "")
        
        # Build capabilities
        capabilities_list = ", ".join(context["capabilities"])
        
        # Desktop environment info
        desktop_env = context["environment"]["desktop_environment"]
        desktop_info = f"{desktop_env.get('name', 'Unknown')} ({desktop_env.get('type', 'unknown')})"
        
        return f"""# Agent S2 - HOST MODE CONTEXT

## OPERATING ENVIRONMENT  
You are operating in **HOST MODE** - extended access to the host system with native desktop integration.

**Mode**: {context["mode"]["display_name"]}
**Security Level**: {context["mode"]["security_level"].title()}
**Desktop Environment**: {desktop_info}
**Display**: {context["environment"]["display"]["type"]} ({context["environment"]["display"]["resolution"]})
**Applications Available**: {len(apps)} detected

## FILESYSTEM ACCESS
**Mounted Directories:**
{mounts_section}

**Security Constraints:**
- Security Profile: {security.get("security_profile", "default")}
- Forbidden Paths: {forbidden_list}
- Audit Logging: {'Enabled' if security.get("audit_logging") else 'Disabled'}
- Max Mount Size: {context["config_summary"]["mode_specific"].get("host_max_mount_size_gb", "Unknown")} GB

## AVAILABLE APPLICATIONS
{apps_section}

## CAPABILITIES
You have access to these advanced capabilities:
{capabilities_list}

## NATIVE DESKTOP INTEGRATION
- **Application Launch**: Use native desktop methods (menus, launchers, command line)
- **File Management**: Access configured mounted directories
- **Window Management**: Full native window manager support
- **System Integration**: Can interact with host system services (within security constraints)

## SECURITY GUIDELINES
1. **Respect Mount Boundaries**: Only access configured mounted directories
2. **Avoid Forbidden Paths**: System directories are protected
3. **Monitor Resource Usage**: Be mindful of system resource consumption
4. **Audit Compliance**: Actions may be logged for security review

## HOST SYSTEM INTERACTION
- **Network Access**: Full network access including localhost
- **Process Management**: Can launch and manage applications
- **File Operations**: Read/write within mounted directories
- **System Integration**: Native desktop notifications and services

## TROUBLESHOOTING
- **Permission Errors**: Check if path is within mounted directories
- **Application Issues**: Verify application is in allowed list
- **Display Problems**: Ensure proper X11/Wayland permissions
- **Network Issues**: Host networking should provide full access

Remember: You have significant system access. Use responsibly and respect security boundaries."""

    def _format_applications_list(self, apps: Dict[str, Dict[str, Any]], sandbox_mode: bool) -> str:
        """Format applications list for display in prompts"""
        if not apps:
            return "No applications detected"
            
        if sandbox_mode:
            # Group by category for sandbox
            categorized = {}
            for name, info in apps.items():
                category = info.get("category", "other")
                if category not in categorized:
                    categorized[category] = []
                categorized[category].append(info)
                
            sections = []
            for category, app_list in categorized.items():
                section = f"**{category.replace('_', ' ').title()}:**\n"
                for app in app_list:
                    section += f"- {app['name']}: {app['launcher']} - {app.get('description', 'No description')}\n"
                sections.append(section)
            
            return "\n".join(sections)
        else:
            # Simpler list for host mode (too many to categorize)
            if len(apps) > 20:
                # Show top applications + summary
                important_apps = []
                for name, info in list(apps.items())[:15]:
                    important_apps.append(f"- {info['name']}: {info.get('description', 'No description')}")
                
                return "\n".join(important_apps) + f"\n... and {len(apps) - 15} more applications available"
            else:
                return "\n".join([
                    f"- {info['name']}: {info.get('description', 'No description')}"
                    for info in apps.values()
                ])
    
    def get_context_summary(self) -> Dict[str, Any]:
        """Get a summary of the current context"""
        return {
            "mode": self.mode.value,
            "applications_count": len(self.context_data["applications"]),
            "capabilities_count": len(self.context_data["capabilities"]),
            "limitations_count": len(self.context_data["limitations"]),
            "security_level": self.context_data["mode"]["security_level"],
            "display_type": self.context_data["environment"]["display"]["type"],
            "timestamp": self.context_data["timestamp"]
        }
    
    def get_application_info(self, app_name: str) -> Optional[Dict[str, Any]]:
        """Get detailed information about a specific application"""
        apps = self.context_data["applications"]
        
        # Direct match
        if app_name in apps:
            return apps[app_name]
            
        # Case-insensitive search
        for name, info in apps.items():
            if name.lower() == app_name.lower():
                return info
                
        # Partial match
        for name, info in apps.items():
            if app_name.lower() in name.lower() or app_name.lower() in info.get("command", "").lower():
                return info
                
        return None
    
    def get_launch_instructions(self, app_name: str) -> Optional[str]:
        """Get specific launch instructions for an application"""
        app_info = self.get_application_info(app_name)
        if not app_info:
            return None
            
        if self.mode == AgentMode.SANDBOX:
            return f"Launch {app_info['name']}: {app_info['launcher']}"
        elif self.mode == AgentMode.HOST:
            return f"Launch {app_info['name']}: Use native desktop or run '{app_info['command']}'"
            
        return None
    
    def is_action_available(self, action: str) -> bool:
        """Check if a specific action is available in current mode"""
        return action in self.context_data["available_actions"]
    
    def get_security_constraints(self) -> Dict[str, Any]:
        """Get current security constraints"""
        return self.context_data["security"]
    
    def refresh(self) -> None:
        """Refresh context data (re-discover environment)"""
        self.discovery = EnvironmentDiscovery(self.mode)
        self.context_data = self._build_context_data()
        self.system_prompt = self._generate_system_prompt()
        logger.info(f"Context refreshed for {self.mode.value} mode")
    
    def switch_mode(self, new_mode: AgentMode) -> 'ModeContext':
        """Create new context for different mode"""
        return ModeContext(new_mode)
    
    def export_context(self) -> Dict[str, Any]:
        """Export full context data for external use"""
        return {
            "mode": self.mode.value,
            "context_data": self.context_data,
            "system_prompt": self.system_prompt,
            "discovery_capabilities": self.discovery.capabilities,
            "generated_at": datetime.utcnow().isoformat()
        }