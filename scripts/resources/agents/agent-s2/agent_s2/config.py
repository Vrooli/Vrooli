"""Agent S2 Configuration Module

Centralized configuration for all Agent S2 components.
All configuration values can be overridden via environment variables.
"""

import os
from typing import Optional, Dict, List, Any
from enum import Enum


class AgentMode(Enum):
    """Available Agent S2 operation modes"""
    SANDBOX = "sandbox"
    HOST = "host"


class Config:
    """Central configuration class for Agent S2"""
    
    # API Configuration
    API_HOST: str = os.getenv("AGENT_S2_HOST", "0.0.0.0")
    API_PORT: int = int(os.getenv("AGENT_S2_PORT", "4113"))
    API_BASE_URL: str = f"http://localhost:{API_PORT}"
    
    # Display Configuration  
    DISPLAY: str = os.getenv("DISPLAY", ":99")
    SCREEN_WIDTH: int = int(os.getenv("AGENT_S2_SCREEN_WIDTH", "1920"))
    SCREEN_HEIGHT: int = int(os.getenv("AGENT_S2_SCREEN_HEIGHT", "1080"))
    SCREEN_DEPTH: int = int(os.getenv("AGENT_S2_SCREEN_DEPTH", "24"))
    
    # VNC Configuration
    VNC_PASSWORD: str = os.getenv("AGENT_S2_VNC_PASSWORD", "agents2vnc")
    VNC_PORT: int = int(os.getenv("AGENT_S2_VNC_PORT", "5900"))
    
    # AI Configuration
    AI_ENABLED: bool = os.getenv("AGENTS2_ENABLE_AI", "true").lower() == "true"
    AI_PROVIDER: str = os.getenv("AGENTS2_LLM_PROVIDER", "ollama")
    AI_API_URL: str = os.getenv("AGENTS2_OLLAMA_BASE_URL", "http://localhost:11434").rstrip("/") + "/api/generate"
    AI_MODEL: str = os.getenv("AGENTS2_LLM_MODEL", "llama3.2-vision:11b")
    AI_TIMEOUT: int = int(os.getenv("AGENTS2_TIMEOUT", "120"))
    
    # Output Configuration
    OUTPUT_DIR: str = os.getenv("AGENT_S2_OUTPUT_DIR", "/tmp/agent-s2-outputs")
    MAX_OUTPUT_SIZE_MB: int = int(os.getenv("AGENT_S2_MAX_OUTPUT_SIZE_MB", "100"))
    
    # Security Configuration
    ENABLE_CORS: bool = os.getenv("AGENT_S2_ENABLE_CORS", "true").lower() == "true"
    CORS_ORIGINS: list = os.getenv("AGENT_S2_CORS_ORIGINS", "*").split(",")
    
    # Performance Configuration
    SCREENSHOT_CACHE_TTL: int = int(os.getenv("AGENT_S2_SCREENSHOT_CACHE_TTL", "0"))  # seconds
    MAX_CONCURRENT_REQUESTS: int = int(os.getenv("AGENT_S2_MAX_CONCURRENT_REQUESTS", "10"))
    
    # Logging Configuration
    LOG_LEVEL: str = os.getenv("AGENT_S2_LOG_LEVEL", "INFO")
    LOG_FORMAT: str = os.getenv("AGENT_S2_LOG_FORMAT", "json")  # json or text
    
    # Docker Configuration
    CONTAINER_NAME: str = "agent-s2"
    DOCKER_IMAGE: str = "agent-s2:latest"
    
    # Mode Configuration
    CURRENT_MODE: AgentMode = AgentMode(os.getenv("AGENT_S2_MODE", "sandbox"))
    DEFAULT_MODE: AgentMode = AgentMode(os.getenv("AGENT_S2_DEFAULT_MODE", "sandbox"))
    
    # Sandbox Mode Configuration
    SANDBOX_APPLICATIONS: List[str] = [
        "firefox-esr", "mousepad", "gedit", "gnome-calculator", "xterm"
    ]
    
    # Host Mode Configuration  
    HOST_MODE_ENABLED: bool = os.getenv("AGENT_S2_HOST_MODE_ENABLED", "false").lower() == "true"
    HOST_DISPLAY_ACCESS: bool = os.getenv("AGENT_S2_HOST_DISPLAY_ACCESS", "false").lower() == "true"
    HOST_FILESYSTEM_MOUNTS: str = os.getenv("AGENT_S2_HOST_MOUNTS", "")  # JSON string
    HOST_ALLOWED_APPLICATIONS: str = os.getenv("AGENT_S2_HOST_APPS", "*")  # comma-separated or "*"
    HOST_SECURITY_PROFILE: str = os.getenv("AGENT_S2_HOST_SECURITY_PROFILE", "agent-s2-host")
    
    # Security Configuration for Host Mode
    HOST_FORBIDDEN_PATHS: List[str] = ["/etc", "/var", "/usr", "/boot", "/sys", "/proc"]
    HOST_MAX_MOUNT_SIZE_GB: int = int(os.getenv("AGENT_S2_HOST_MAX_MOUNT_SIZE_GB", "10"))
    HOST_AUDIT_LOGGING: bool = os.getenv("AGENT_S2_HOST_AUDIT_LOGGING", "true").lower() == "true"
    
    # Stealth Mode Configuration
    STEALTH_MODE_ENABLED: bool = os.getenv("AGENT_S2_STEALTH_MODE_ENABLED", "true").lower() == "true"
    SESSION_STORAGE_PATH: str = os.getenv("AGENT_S2_SESSION_STORAGE_PATH", "/home/agents2/.agent-s2/sessions")
    SESSION_DATA_PERSISTENCE: bool = os.getenv("AGENT_S2_SESSION_DATA_PERSISTENCE", "true").lower() == "true"
    SESSION_STATE_PERSISTENCE: bool = os.getenv("AGENT_S2_SESSION_STATE_PERSISTENCE", "false").lower() == "true"
    SESSION_ENCRYPTION: bool = os.getenv("AGENT_S2_SESSION_ENCRYPTION", "true").lower() == "true"
    SESSION_TTL_DAYS: int = int(os.getenv("AGENT_S2_SESSION_TTL_DAYS", "30"))
    STEALTH_PROFILE_TYPE: str = os.getenv("AGENT_S2_STEALTH_PROFILE_TYPE", "residential")  # residential, mobile, datacenter
    
    @classmethod
    def validate(cls) -> None:
        """Validate configuration values"""
        errors = []
        
        # Validate port ranges
        if not 1 <= cls.API_PORT <= 65535:
            errors.append(f"API_PORT must be between 1 and 65535, got {cls.API_PORT}")
            
        if not 1 <= cls.VNC_PORT <= 65535:
            errors.append(f"VNC_PORT must be between 1 and 65535, got {cls.VNC_PORT}")
            
        # Validate screen dimensions
        if cls.SCREEN_WIDTH <= 0:
            errors.append(f"SCREEN_WIDTH must be positive, got {cls.SCREEN_WIDTH}")
            
        if cls.SCREEN_HEIGHT <= 0:
            errors.append(f"SCREEN_HEIGHT must be positive, got {cls.SCREEN_HEIGHT}")
            
        # Validate log level
        valid_log_levels = ["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]
        if cls.LOG_LEVEL.upper() not in valid_log_levels:
            errors.append(f"LOG_LEVEL must be one of {valid_log_levels}, got {cls.LOG_LEVEL}")
        
        # Validate mode configuration
        if cls.CURRENT_MODE == AgentMode.HOST and not cls.HOST_MODE_ENABLED:
            errors.append("HOST mode requested but HOST_MODE_ENABLED is false")
        
        # Validate host mode security
        if cls.HOST_MODE_ENABLED:
            if cls.HOST_MAX_MOUNT_SIZE_GB <= 0:
                errors.append(f"HOST_MAX_MOUNT_SIZE_GB must be positive, got {cls.HOST_MAX_MOUNT_SIZE_GB}")
            
            if cls.HOST_DISPLAY_ACCESS:
                import warnings
                warnings.warn("Host display access enabled - this is a security risk", UserWarning)
            
        if errors:
            raise ValueError(f"Configuration validation failed:\n" + "\n".join(errors))
    
    @classmethod
    def get_display_geometry(cls) -> str:
        """Get X11 display geometry string"""
        return f"{cls.SCREEN_WIDTH}x{cls.SCREEN_HEIGHT}x{cls.SCREEN_DEPTH}"
    
    @classmethod
    def get_api_url(cls, path: str = "") -> str:
        """Get full API URL with optional path"""
        base = cls.API_BASE_URL.rstrip("/")
        path = path.lstrip("/")
        return f"{base}/{path}" if path else base
    
    @classmethod
    def to_dict(cls) -> dict:
        """Export configuration as dictionary"""
        return {
            key: value 
            for key, value in cls.__dict__.items() 
            if not key.startswith("_") and not callable(value)
        }
    
    @classmethod
    def get(cls, key: str, default: Any = None) -> Any:
        """Get configuration value with optional default"""
        return getattr(cls, key, default)
    
    @classmethod
    def is_sandbox_mode(cls) -> bool:
        """Check if currently in sandbox mode"""
        return cls.CURRENT_MODE == AgentMode.SANDBOX
    
    @classmethod 
    def is_host_mode(cls) -> bool:
        """Check if currently in host mode"""
        return cls.CURRENT_MODE == AgentMode.HOST
    
    @classmethod
    def get_mode_name(cls) -> str:
        """Get current mode as string"""
        return cls.CURRENT_MODE.value
    
    @classmethod
    def get_allowed_applications(cls) -> List[str]:
        """Get list of allowed applications for current mode"""
        if cls.is_sandbox_mode():
            return cls.SANDBOX_APPLICATIONS
        elif cls.is_host_mode():
            if cls.HOST_ALLOWED_APPLICATIONS == "*":
                return ["*"]  # All applications allowed
            return [app.strip() for app in cls.HOST_ALLOWED_APPLICATIONS.split(",") if app.strip()]
        return []
    
    @classmethod
    def get_host_mounts(cls) -> List[Dict[str, str]]:
        """Parse and return host filesystem mounts configuration"""
        if not cls.HOST_FILESYSTEM_MOUNTS:
            return []
        
        try:
            import json
            mounts = json.loads(cls.HOST_FILESYSTEM_MOUNTS)
            return mounts if isinstance(mounts, list) else []
        except (json.JSONDecodeError, ImportError):
            # Fallback to simple parsing
            return []
    
    @classmethod
    def get_security_constraints(cls) -> Dict[str, Any]:
        """Get security constraints for current mode"""
        if cls.is_sandbox_mode():
            return {
                "filesystem_access": "none",
                "display_access": "virtual",
                "network_access": "bridge",
                "applications": cls.SANDBOX_APPLICATIONS,
                "isolation_level": "high"
            }
        elif cls.is_host_mode():
            return {
                "filesystem_access": "mounted_only",
                "display_access": "host" if cls.HOST_DISPLAY_ACCESS else "virtual",
                "network_access": "host",
                "applications": cls.get_allowed_applications(),
                "isolation_level": "medium",
                "security_profile": cls.HOST_SECURITY_PROFILE,
                "forbidden_paths": cls.HOST_FORBIDDEN_PATHS,
                "audit_logging": cls.HOST_AUDIT_LOGGING
            }
        return {}
    
    @classmethod
    def switch_mode(cls, new_mode: AgentMode) -> None:
        """Switch to a different mode (requires restart)"""
        if new_mode == AgentMode.HOST and not cls.HOST_MODE_ENABLED:
            raise ValueError("Host mode is not enabled in configuration")
        
        # Update environment variable for persistence across restarts
        os.environ["AGENT_S2_MODE"] = new_mode.value
        cls.CURRENT_MODE = new_mode


# Validate configuration on module import
try:
    Config.validate()
except ValueError as e:
    print(f"Warning: {e}")