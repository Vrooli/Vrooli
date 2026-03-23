"""Verification and embedding extraction endpoints."""

import time

from fastapi import APIRouter, File, Form, HTTPException, UploadFile

import audio
import embeddings
import profiles
import verification
from config import settings
from models import ErrorResponse, InfoResponse, VerificationResponse

router = APIRouter(prefix="/v1")


@router.post(
    "/verify",
    response_model=VerificationResponse,
    responses={400: {"model": ErrorResponse}, 404: {"model": ErrorResponse}},
)
async def verify_speaker(
    audio_file: UploadFile = File(..., alias="audio"),
    profile_id: str = Form(...),
    threshold: float | None = Form(None),
    return_embedding: bool = Form(False),
):
    """Compare verification audio against a stored profile."""
    overall_start = time.monotonic()

    # Load stored profile
    profile_embedding = profiles.load_embedding(profile_id)
    if profile_embedding is None:
        raise HTTPException(
            status_code=404, detail=f"Profile '{profile_id}' not found"
        )

    # Read and process audio
    audio_bytes = await audio_file.read()
    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    try:
        waveform, duration = audio.load_and_normalize(audio_bytes)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    try:
        audio.validate_duration(duration, settings.VERIFY_MIN_SECONDS)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    # Extract candidate embedding
    candidate = embeddings.extract_embedding(waveform)

    # Verify
    result = verification.verify(profile_embedding, candidate, threshold)

    overall_duration_ms = (time.monotonic() - overall_start) * 1000

    return VerificationResponse(
        profile_id=profile_id,
        matched=result["matched"],
        score=result["score"],
        threshold=result["threshold"],
        duration_ms=round(overall_duration_ms, 2),
        backend=settings.BACKEND,
        model=settings.MODEL,
        audio_seconds=round(duration, 2),
    )


@router.post("/embeddings")
async def extract_embeddings_endpoint(
    audio_file: UploadFile = File(..., alias="audio"),
):
    """Debug endpoint: extract raw embedding from audio."""
    audio_bytes = await audio_file.read()
    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    try:
        waveform, duration = audio.load_and_normalize(audio_bytes)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    embedding = embeddings.extract_embedding(waveform)

    return {
        "embedding": embedding.tolist(),
        "embedding_dim": len(embedding),
        "audio_seconds": round(duration, 2),
        "model": settings.MODEL,
        "backend": settings.BACKEND,
    }


@router.get("/info", response_model=InfoResponse)
async def info():
    """Service backend and model information."""
    return InfoResponse(
        backend=settings.BACKEND,
        model=settings.MODEL,
        device=embeddings.get_device(),
        sample_rate=settings.SAMPLE_RATE,
        version=settings.VERSION,
        embedding_dim=embeddings.get_embedding_dim(),
    )
