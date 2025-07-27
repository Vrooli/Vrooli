"""Request models for Agent S2 API"""

from typing import Dict, Any, Optional, List
from pydantic import BaseModel, Field


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


class MouseMoveRequest(BaseModel):
    """Mouse movement request"""
    x: int = Field(..., description="Target X coordinate")
    y: int = Field(..., description="Target Y coordinate")
    duration: float = Field(default=0.0, ge=0, description="Movement duration in seconds")


class MouseClickRequest(BaseModel):
    """Mouse click request"""
    button: str = Field(default="left", description="Mouse button (left, right, middle)")
    x: Optional[int] = Field(default=None, description="X coordinate (uses current position if None)")
    y: Optional[int] = Field(default=None, description="Y coordinate (uses current position if None)")
    clicks: int = Field(default=1, ge=1, description="Number of clicks")


class KeyboardTypeRequest(BaseModel):
    """Keyboard typing request"""
    text: str = Field(..., description="Text to type")
    interval: float = Field(default=0.0, ge=0, description="Delay between keystrokes in seconds")


class KeyboardPressRequest(BaseModel):
    """Keyboard key press request"""
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