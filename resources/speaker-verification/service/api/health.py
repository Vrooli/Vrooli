"""Health and readiness endpoints."""

import tempfile

from fastapi import APIRouter, Response, status

import embeddings
import profiles
from config import settings
from models import HealthResponse, ReadyResponse

router = APIRouter()


@router.get("/health", response_model=HealthResponse)
async def health():
    """Liveness check: process is running and can respond."""
    return HealthResponse(status="ok", version=settings.VERSION)


@router.get("/ready", response_model=ReadyResponse)
async def ready(response: Response):
    """Readiness check: model loaded, store accessible, temp dir writable."""
    model_loaded = embeddings.is_model_loaded()
    profile_store_ok = profiles.is_store_accessible()

    # Check temp dir
    temp_dir_ok = False
    try:
        with tempfile.NamedTemporaryFile(delete=True):
            temp_dir_ok = True
    except Exception:
        pass

    is_ready = model_loaded and profile_store_ok and temp_dir_ok
    if not is_ready:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE

    readiness = "ready" if is_ready else "not_ready"

    return ReadyResponse(
        status=readiness,
        model_loaded=model_loaded,
        profile_store_ok=profile_store_ok,
        temp_dir_ok=temp_dir_ok,
    )
