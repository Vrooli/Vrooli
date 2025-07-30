"""Request models for Agent S2 API"""

from typing import Dict, Any, Optional, List
from pydantic import BaseModel, Field


class WindowCriteria(BaseModel):
    """Criteria for selecting specific window of an application"""
    title_contains: Optional[str] = Field(default=None, description="Window title must contain this text")
    title_matches: Optional[str] = Field(default=None, description="Window title must match this regex pattern")
    url_contains: Optional[str] = Field(default=None, description="Browser window URL must contain this text")
    window_id: Optional[str] = Field(default=None, description="Explicit window ID to target")
    prefer_recent: bool = Field(default=True, description="Prefer most recently active window")


class TargetedAutomationRequest(BaseModel):
    """Base class for target-aware automation requests"""
    target_app: Optional[str] = Field(default=None, description="Target application name (firefox, terminal, calculator)")
    window_criteria: Optional[WindowCriteria] = Field(default=None, description="Criteria for window selection")
    ensure_focus: bool = Field(default=True, description="Ensure target has focus before action")
    focus_timeout: float = Field(default=2.0, description="Timeout for focus operations in seconds")


class TaskRequest(BaseModel):
    """Task execution request"""
    task_type: str = Field(..., description="Type of task: screenshot, click, type, automation")
    parameters: Dict[str, Any] = Field(default_factory=dict, description="Task parameters")
    async_execution: bool = Field(default=False, description="Execute task asynchronously")


class ScreenshotRequest(BaseModel):
    """Screenshot request parameters"""
    format: str = Field(default="png", description="Image format (png or jpeg)")
    quality: int = Field(default=95, ge=1, le=100, description="JPEG quality (1-100)")
    region: Optional[List[int]] = Field(default=None, description="Region to capture [x, y, width, height]")


class MouseMoveRequest(TargetedAutomationRequest):
    """Mouse movement request with target awareness"""
    x: int = Field(..., description="Target X coordinate")
    y: int = Field(..., description="Target Y coordinate")
    duration: float = Field(default=0.0, ge=0, description="Movement duration in seconds")


class MouseClickRequest(TargetedAutomationRequest):
    """Mouse click request with target awareness"""
    button: str = Field(default="left", description="Mouse button (left, right, middle)")
    x: Optional[int] = Field(default=None, description="X coordinate (uses current position if None)")
    y: Optional[int] = Field(default=None, description="Y coordinate (uses current position if None)")
    clicks: int = Field(default=1, ge=1, description="Number of clicks")


class KeyboardTypeRequest(TargetedAutomationRequest):
    """Keyboard typing request with target awareness"""
    text: str = Field(..., description="Text to type")
    interval: float = Field(default=0.0, ge=0, description="Delay between keystrokes in seconds")


class KeyboardPressRequest(TargetedAutomationRequest):
    """Keyboard key press request with target awareness"""
    key: str = Field(..., description="Key to press")
    modifiers: Optional[List[str]] = Field(default=None, description="Modifier keys (ctrl, alt, shift)")


class FindElementRequest(BaseModel):
    """Find UI element request"""
    screenshot: Optional[str] = Field(default=None, description="Base64 screenshot data")
    text: Optional[str] = Field(default=None, description="Text to search for")
    element_type: Optional[str] = Field(default=None, description="Type of element to find")


class AICommandRequest(BaseModel):
    """AI command execution request"""
    command: str = Field(..., description="Natural language command to execute")
    context: Optional[str] = Field(default=None, description="Additional context for the command")
    async_execution: bool = Field(default=False, description="Execute command asynchronously")


class AIPlanRequest(BaseModel):
    """AI planning request"""
    goal: str = Field(..., description="High-level goal to achieve")
    constraints: Optional[List[str]] = Field(default=None, description="Constraints or limitations")


class AIAnalyzeRequest(BaseModel):
    """AI screen analysis request"""
    question: Optional[str] = Field(default=None, description="Specific question about the screen")
    screenshot: Optional[str] = Field(default=None, description="Base64 screenshot to analyze")


class AIActionRequest(BaseModel):
    """AI action request"""
    task: str = Field(..., description="Task description for AI")
    screenshot: Optional[str] = Field(default=None, description="Optional screenshot data")
    context: Optional[Dict[str, Any]] = Field(default=None, description="Optional context information")