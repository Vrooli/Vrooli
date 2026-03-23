"""Speaker Verification FastAPI Application."""

import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

import embeddings
import extraction
from api import extract, health, profiles, verify
from config import settings

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Load model on startup."""
    logger.info("Starting speaker verification service v%s", settings.VERSION)
    logger.info("Model: %s", settings.MODEL)
    logger.info("Device: %s", settings.DEVICE)
    logger.info("Profiles dir: %s", settings.PROFILES_DIR)

    # Ensure directories exist
    settings.PROFILES_DIR.mkdir(parents=True, exist_ok=True)
    settings.CACHE_DIR.mkdir(parents=True, exist_ok=True)
    settings.LOG_DIR.mkdir(parents=True, exist_ok=True)

    # Load the TitaNet embedding model
    try:
        embeddings.load_model()
        logger.info(
            "Model loaded: dim=%d, device=%s",
            embeddings.get_embedding_dim(),
            embeddings.get_device(),
        )
    except Exception:
        logger.exception("Failed to load model - service will start but not be ready")

    # Load TSE separation model (if enabled)
    if settings.TSE_ENABLED:
        try:
            extraction.load_model()
            logger.info("TSE model loaded: %s", settings.TSE_MODEL)
        except Exception:
            logger.exception("Failed to load TSE model - extraction will be unavailable")
    else:
        logger.info("TSE disabled via config — skipping separation model load")

    yield

    logger.info("Shutting down speaker verification service")


app = FastAPI(
    title="Speaker Verification",
    description="Local speaker verification using NeMo TitaNet",
    version=settings.VERSION,
    lifespan=lifespan,
)

# Register routes
app.include_router(health.router)
app.include_router(profiles.router)
app.include_router(verify.router)
app.include_router(extract.router)
