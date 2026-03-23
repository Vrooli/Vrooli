"""Profile management endpoints."""

from fastapi import APIRouter, File, Form, HTTPException, UploadFile

import audio
import embeddings
import profiles
from config import settings
from models import (
    EnrollmentResponse,
    ErrorResponse,
    ProfileListResponse,
    ProfileResponse,
)

router = APIRouter(prefix="/v1")


@router.post(
    "/profiles",
    response_model=EnrollmentResponse,
    responses={400: {"model": ErrorResponse}, 422: {"model": ErrorResponse}},
)
async def enroll_profile(
    audio_file: UploadFile = File(..., alias="audio"),
    profile_id: str = Form(...),
    display_name: str = Form(""),
    notes: str = Form(""),
):
    """Create or replace a speaker profile from enrollment audio."""
    # Read audio
    audio_bytes = await audio_file.read()
    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    # Load and normalize
    try:
        waveform, duration = audio.load_and_normalize(audio_bytes)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    # Validate duration
    try:
        audio.validate_duration(duration, settings.ENROLLMENT_MIN_SECONDS)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    # Check signal presence
    try:
        audio.check_signal_presence(waveform)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    # Extract embedding(s) - structured for future multi-sample support
    embedding_list = embeddings.extract_embeddings_multi([waveform])
    canonical = embeddings.aggregate_embeddings(embedding_list)

    # Persist
    metadata = profiles.save(
        profile_id=profile_id,
        embedding=canonical,
        display_name=display_name or profile_id,
        enrollment_audio_seconds=duration,
        model_name=settings.MODEL,
        notes=notes,
    )

    return EnrollmentResponse(
        profile_id=metadata.id,
        display_name=metadata.display_name,
        embedding_dim=metadata.embedding_dim,
        sample_rate=metadata.sample_rate,
        enrollment_audio_seconds=metadata.enrollment_audio_seconds,
        model_name=metadata.model_name,
        created_at=metadata.created_at,
    )


@router.get("/profiles", response_model=ProfileListResponse)
async def list_profiles():
    """List all stored profiles."""
    profile_list = profiles.list_all()
    return ProfileListResponse(profiles=profile_list, count=len(profile_list))


@router.get(
    "/profiles/{profile_id}",
    response_model=ProfileResponse,
    responses={404: {"model": ErrorResponse}},
)
async def get_profile(profile_id: str):
    """Get a single profile's metadata."""
    meta = profiles.load_metadata(profile_id)
    if meta is None:
        raise HTTPException(status_code=404, detail=f"Profile '{profile_id}' not found")
    return ProfileResponse(**meta.model_dump())


@router.delete(
    "/profiles/{profile_id}",
    responses={404: {"model": ErrorResponse}},
)
async def delete_profile(profile_id: str):
    """Remove a profile and its data."""
    if not profiles.delete(profile_id):
        raise HTTPException(status_code=404, detail=f"Profile '{profile_id}' not found")
    return {"status": "deleted", "profile_id": profile_id}
