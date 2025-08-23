"""Agent S2 API Models"""

from .requests import (
    TaskRequest,
    ScreenshotRequest,
    MouseMoveRequest,
    MouseClickRequest,
    KeyboardTypeRequest,
    KeyboardPressRequest,
    FindElementRequest,
    AICommandRequest,
    AIPlanRequest,
    AIAnalyzeRequest,
    AIActionRequest
)

from .responses import (
    TaskResponse,
    HealthResponse,
    CapabilitiesResponse,
    ScreenshotResponse,
    MousePositionResponse,
    MouseActionResponse,
    KeyboardActionResponse,
    FindElementResponse,
    AICommandResponse,
    AIPlanResponse,
    AIAnalyzeResponse,
    AIActionResponse
)

__all__ = [
    # Requests
    "TaskRequest",
    "ScreenshotRequest",
    "MouseMoveRequest",
    "MouseClickRequest",
    "KeyboardTypeRequest",
    "KeyboardPressRequest",
    "FindElementRequest",
    "AICommandRequest",
    "AIPlanRequest",
    "AIAnalyzeRequest",
    "AIActionRequest",
    # Responses
    "TaskResponse",
    "HealthResponse",
    "CapabilitiesResponse",
    "ScreenshotResponse",
    "MousePositionResponse",
    "MouseActionResponse",
    "KeyboardActionResponse",
    "FindElementResponse",
    "AICommandResponse",
    "AIPlanResponse",
    "AIAnalyzeResponse",
    "AIActionResponse"
]