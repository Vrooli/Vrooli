"""Response models for Agent S2 API"""

from typing import Dict, Any, Optional, List
from pydantic import BaseModel, Field


class BaseResponse(BaseModel):
    """Base response model"""
    success: bool = True
    message: Optional[str] = None


class TaskResponse(BaseModel):
    """Task execution response"""
    task_id: str
    status: str
    result: Optional[Dict[str, Any]] = None
    error: Optional[str] = None
    created_at: str
    completed_at: Optional[str] = None


class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    display: str
    screen_size: Dict[str, int]
    startup_time: str
    tasks_processed: int
    ai_status: Dict[str, Any]
    mode_info: Optional[Dict[str, Any]] = None


class CapabilitiesResponse(BaseModel):
    """Capabilities response"""
    capabilities: Dict[str, bool]
    supported_tasks: List[str]
    ai_status: Dict[str, Any]
    display_info: Dict[str, Any]
    mode_info: Optional[Dict[str, Any]] = None


class ScreenshotResponse(BaseModel):
    """Screenshot response"""
    success: bool
    format: str
    size: Dict[str, int]
    data: str = Field(..., description="Base64 encoded image data")


class MousePositionResponse(BaseModel):
    """Mouse position response"""
    x: int
    y: int


class TargetedActionResponse(BaseModel):
    """Response for targeted automation actions"""
    success: bool
    action: str
    target_app: Optional[str] = Field(default=None, description="Target application that was focused")
    focused_window: Optional[str] = Field(default=None, description="Title of window that was focused")
    window_id: Optional[str] = Field(default=None, description="ID of window that was focused")
    focus_time: Optional[float] = Field(default=None, description="Time taken to focus target (seconds)")
    execution_time: float = Field(..., description="Total execution time (seconds)")
    message: str = Field(..., description="Description of action performed")
    error: Optional[str] = Field(default=None, description="Error message if action failed")


class MouseActionResponse(TargetedActionResponse):
    """Mouse action response with target awareness"""
    position: Optional[Dict[str, int]] = Field(default=None, description="Mouse position after action")
    button: Optional[str] = Field(default=None, description="Mouse button used")
    clicks: Optional[int] = Field(default=None, description="Number of clicks performed")


class KeyboardActionResponse(TargetedActionResponse):
    """Keyboard action response with target awareness"""
    text: Optional[str] = Field(default=None, description="Text that was typed")
    key: Optional[str] = Field(default=None, description="Key combination that was pressed")
    interval: Optional[float] = Field(default=None, description="Typing interval used")


class FindElementResponse(BaseModel):
    """Find element response"""
    found: bool
    x: Optional[int] = None
    y: Optional[int] = None
    width: Optional[int] = None
    height: Optional[int] = None
    text: Optional[str] = None
    confidence: Optional[float] = None


class AICommandResponse(BaseModel):
    """AI command execution response"""
    task_id: str
    status: str
    command: str
    executed: bool
    actions_taken: Optional[List[Dict[str, Any]]] = None
    message: Optional[str] = None
    error: Optional[str] = None


class AIPlanResponse(BaseModel):
    """AI planning response"""
    goal: str
    constraints: List[str]
    steps: List[Dict[str, Any]]
    estimated_duration: str
    complexity: str


class AIAnalyzeResponse(BaseModel):
    """AI screen analysis response"""
    screen_size: Dict[str, int]
    question: Optional[str]
    analysis: str
    elements_detected: List[Dict[str, Any]]
    suggested_actions: List[Dict[str, Any]]


class AIActionResponse(BaseModel):
    """AI action response"""
    success: bool
    task: str
    summary: str
    actions_taken: Optional[List[Dict[str, Any]]] = None
    reasoning: Optional[str] = None
    error: Optional[str] = None