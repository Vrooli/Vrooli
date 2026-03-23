"""Pydantic models for request/response validation."""

from datetime import datetime
from typing import Optional

from pydantic import BaseModel, Field


class ProfileMetadata(BaseModel):
    """Speaker profile metadata."""

    id: str
    display_name: str = ""
    created_at: str = ""
    updated_at: str = ""
    model_name: str = ""
    embedding_dim: int = 0
    sample_rate: int = 16000
    enrollment_audio_seconds: float = 0.0
    notes: str = ""


class ProfileResponse(BaseModel):
    """Response for a single profile."""

    id: str
    display_name: str
    created_at: str
    updated_at: str
    model_name: str
    embedding_dim: int
    sample_rate: int
    enrollment_audio_seconds: float
    notes: str = ""


class ProfileListResponse(BaseModel):
    """Response for listing profiles."""

    profiles: list[ProfileResponse]
    count: int


class EnrollmentResponse(BaseModel):
    """Response after enrolling a speaker profile."""

    profile_id: str
    display_name: str
    embedding_dim: int
    sample_rate: int
    enrollment_audio_seconds: float
    model_name: str
    created_at: str


class VerificationResponse(BaseModel):
    """Response from a verification request."""

    profile_id: str
    matched: bool
    score: float
    threshold: float
    duration_ms: float
    backend: str
    model: str
    audio_seconds: float


class InfoResponse(BaseModel):
    """Response for service info."""

    backend: str
    model: str
    device: str
    sample_rate: int
    version: str
    embedding_dim: int = 0


class HealthResponse(BaseModel):
    """Response for health check."""

    status: str
    version: str


class ReadyResponse(BaseModel):
    """Response for readiness check."""

    status: str
    model_loaded: bool
    tse_model_loaded: bool = False
    profile_store_ok: bool
    temp_dir_ok: bool


class ErrorResponse(BaseModel):
    """Standard error response."""

    error: str
    detail: str = ""
