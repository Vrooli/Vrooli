"""Profile persistence: create, load, list, delete."""

import json
import logging
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

import numpy as np

from config import settings
from models import ProfileMetadata, ProfileResponse

logger = logging.getLogger(__name__)


def _profile_dir(profile_id: str) -> Path:
    """Get the directory for a profile."""
    return settings.PROFILES_DIR / profile_id


def _metadata_path(profile_id: str) -> Path:
    """Get the metadata file path for a profile."""
    return _profile_dir(profile_id) / "profile.json"


def _embedding_path(profile_id: str) -> Path:
    """Get the embedding file path for a profile."""
    return _profile_dir(profile_id) / "embedding.npy"


def exists(profile_id: str) -> bool:
    """Check if a profile exists."""
    return _metadata_path(profile_id).exists()


def save(
    profile_id: str,
    embedding: np.ndarray,
    display_name: str = "",
    enrollment_audio_seconds: float = 0.0,
    model_name: str = "",
    notes: str = "",
) -> ProfileMetadata:
    """Save a profile using staged file replacement.

    Returns the saved profile metadata.
    """
    profile_dir = _profile_dir(profile_id)
    profile_dir.mkdir(parents=True, exist_ok=True)
    now = datetime.now(timezone.utc).isoformat()

    metadata = ProfileMetadata(
        id=profile_id,
        display_name=display_name or profile_id,
        created_at=now,
        updated_at=now,
        model_name=model_name or settings.MODEL,
        embedding_dim=len(embedding),
        sample_rate=settings.SAMPLE_RATE,
        enrollment_audio_seconds=enrollment_audio_seconds,
        notes=notes,
    )

    # If profile already exists, preserve created_at
    if exists(profile_id):
        existing = load_metadata(profile_id)
        if existing:
            metadata.created_at = existing.created_at

    # Stage files in the target directory and replace file-by-file so an
    # interrupted update does not delete the existing profile outright.
    fd_meta, meta_tmp_path = tempfile.mkstemp(
        dir=profile_dir, prefix="profile.", suffix=".json.tmp"
    )
    fd_emb, emb_tmp_path = tempfile.mkstemp(
        dir=profile_dir, prefix="embedding.", suffix=".npy.tmp"
    )
    try:
        with open(fd_meta, "w", encoding="utf-8", closefd=True) as meta_file:
            json.dump(metadata.model_dump(), meta_file, indent=2)
            meta_file.write("\n")

        with open(fd_emb, "wb", closefd=True) as emb_file:
            np.save(emb_file, embedding)

        Path(meta_tmp_path).replace(_metadata_path(profile_id))
        Path(emb_tmp_path).replace(_embedding_path(profile_id))
    except Exception:
        Path(meta_tmp_path).unlink(missing_ok=True)
        Path(emb_tmp_path).unlink(missing_ok=True)
        raise

    logger.info("Saved profile '%s' (dim=%d)", profile_id, len(embedding))
    return metadata


def load_metadata(profile_id: str) -> Optional[ProfileMetadata]:
    """Load profile metadata."""
    meta_path = _metadata_path(profile_id)
    if not meta_path.exists():
        return None

    try:
        data = json.loads(meta_path.read_text())
        return ProfileMetadata(**data)
    except Exception:
        logger.exception("Failed to load metadata for '%s'", profile_id)
        return None


def load_embedding(profile_id: str) -> Optional[np.ndarray]:
    """Load a stored embedding."""
    emb_path = _embedding_path(profile_id)
    if not emb_path.exists():
        return None

    try:
        return np.load(str(emb_path))
    except Exception:
        logger.exception("Failed to load embedding for '%s'", profile_id)
        return None


def list_all() -> list[ProfileResponse]:
    """List all stored profiles."""
    profiles = []

    if not settings.PROFILES_DIR.exists():
        return profiles

    for profile_dir in sorted(settings.PROFILES_DIR.iterdir()):
        if not profile_dir.is_dir():
            continue
        meta = load_metadata(profile_dir.name)
        if meta:
            profiles.append(ProfileResponse(**meta.model_dump()))

    return profiles


def delete(profile_id: str) -> bool:
    """Delete a profile and its data."""
    profile_dir = _profile_dir(profile_id)
    if not profile_dir.exists():
        return False

    import shutil

    shutil.rmtree(profile_dir)
    logger.info("Deleted profile '%s'", profile_id)
    return True


def is_store_accessible() -> bool:
    """Check if the profile store directory is readable and writable."""
    try:
        settings.PROFILES_DIR.mkdir(parents=True, exist_ok=True)
        # Test write
        test_file = settings.PROFILES_DIR / ".health_check"
        test_file.write_text("ok")
        test_file.unlink()
        return True
    except Exception:
        return False
