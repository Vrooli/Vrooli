"""Agent S2 - Autonomous Computer Interaction Service

A sandboxed environment for GUI automation and screenshot capture with AI integration.
"""

__version__ = "1.0.0"
__author__ = "Vrooli Team"

from .config import Config
from .client import AgentS2Client, ScreenshotClient, AutomationClient, AIClient

__all__ = [
    "Config",
    "AgentS2Client", 
    "ScreenshotClient",
    "AutomationClient", 
    "AIClient"
]