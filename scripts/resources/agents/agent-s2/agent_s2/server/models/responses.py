"""Response models for Agent S2 API"""

from typing import Dict, Any, Optional, List
from pydantic import BaseModel, Field


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


class CapabilitiesResponse(BaseModel):
    """Capabilities response"""
    capabilities: Dict[str, bool]
    supported_tasks: List[str]
    ai_status: Dict[str, Any]
    display_info: Dict[str, Any]


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


class MouseActionResponse(BaseModel):
    """Mouse action response"""
    success: bool
    action: str
    position: Optional[Dict[str, int]] = None
    message: Optional[str] = None


class KeyboardActionResponse(BaseModel):
    """Keyboard action response"""
    success: bool
    action: str
    text: Optional[str] = None
    key: Optional[str] = None
    message: Optional[str] = None


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