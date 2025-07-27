"""Agent S2 Configuration Module

Centralized configuration for all Agent S2 components.
All configuration values can be overridden via environment variables.
"""

import os
from typing import Optional


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
    AI_ENABLED: bool = os.getenv("AGENT_S2_AI_ENABLED", "true").lower() == "true"
    AI_API_URL: str = os.getenv("AGENT_S2_AI_API_URL", "http://localhost:11434/api/chat")
    AI_MODEL: str = os.getenv("AGENT_S2_AI_MODEL", "llama3.2-vision:11b")
    AI_TIMEOUT: int = int(os.getenv("AGENT_S2_AI_TIMEOUT", "120"))
    
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


# Validate configuration on module import
try:
    Config.validate()
except ValueError as e:
    print(f"Warning: {e}")