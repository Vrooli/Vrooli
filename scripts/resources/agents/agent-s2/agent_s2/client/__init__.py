"""Agent S2 Client Library

Provides a unified interface for interacting with the Agent S2 API.
"""

from .base import AgentS2Client
from .screenshot import ScreenshotClient
from .automation import AutomationClient
from .ai import AIClient

__all__ = [
    "AgentS2Client",
    "ScreenshotClient", 
    "AutomationClient",
    "AIClient"
]