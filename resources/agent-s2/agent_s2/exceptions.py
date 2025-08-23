"""Custom exceptions for Agent S2

This module defines specific exception classes to replace generic Exception
and bare except handlers throughout the codebase.
"""


class AgentS2Error(Exception):
    """Base exception class for all Agent S2 errors"""
    pass


class ServiceError(AgentS2Error):
    """Base exception for service-related errors"""
    pass


class ServiceUnavailableError(ServiceError):
    """Raised when a required service is not available or accessible"""
    pass


class ServiceInitializationError(ServiceError):
    """Raised when service initialization fails"""
    pass


class AutomationError(AgentS2Error):
    """Base exception for automation-related errors"""
    pass


class ElementNotFoundError(AutomationError):
    """Raised when a UI element cannot be found"""
    pass


class ClickError(AutomationError):
    """Raised when click operations fail"""
    pass


class TypeError(AutomationError):
    """Raised when keyboard typing operations fail"""
    pass


class ScreenshotError(AgentS2Error):
    """Base exception for screenshot-related errors"""
    pass


class ScreenCaptureError(ScreenshotError):
    """Raised when screen capture operations fail"""
    pass


class ImageProcessingError(ScreenshotError):
    """Raised when image processing operations fail"""
    pass


class WindowManagerError(AgentS2Error):
    """Exception for window management operations"""
    pass


class WindowNotFoundError(WindowManagerError):
    """Raised when a specified window cannot be found"""
    pass


class WindowFocusError(WindowManagerError):
    """Raised when window focus operations fail"""
    pass


class SecurityError(AgentS2Error):
    """Base exception for security-related errors"""
    pass


class URLSecurityError(SecurityError):
    """Raised when URL security validation fails"""
    pass


class ProxyError(SecurityError):
    """Base exception for proxy-related errors"""
    pass


class ProxyConfigurationError(ProxyError):
    """Raised when proxy configuration fails"""
    pass


class ProxyConnectionError(ProxyError):
    """Raised when proxy connection fails"""
    pass


class BrowserMonitorError(SecurityError):
    """Raised when browser monitoring operations fail"""
    pass


class StealthError(AgentS2Error):
    """Base exception for stealth mode operations"""
    pass


class StealthConfigurationError(StealthError):
    """Raised when stealth configuration fails"""
    pass


class StealthOperationError(StealthError):
    """Raised when stealth operations fail"""
    pass


class AIError(AgentS2Error):
    """Base exception for AI-related errors"""
    pass


class AIServiceError(AIError):
    """Raised when AI service operations fail"""
    pass


class AIConnectionError(AIError):
    """Raised when AI service connection fails"""
    pass


class AIModelError(AIError):
    """Raised when AI model operations fail"""
    pass


class ValidationError(AgentS2Error):
    """Raised when input validation fails"""
    pass


class ConfigurationError(AgentS2Error):
    """Raised when configuration is invalid or missing"""
    pass


class NetworkError(AgentS2Error):
    """Base exception for network-related errors"""
    pass


class ConnectionError(NetworkError):
    """Raised when network connections fail"""
    pass


class TimeoutError(NetworkError):
    """Raised when operations timeout"""
    pass


class ProcessError(AgentS2Error):
    """Raised when subprocess operations fail"""
    pass