"""Speaker Verification Service Configuration."""

import os
from pathlib import Path


class Settings:
    """Service configuration loaded from environment variables."""

    DEVICE: str = os.getenv("SPEAKER_VERIFICATION_DEVICE", "auto")
    MODEL: str = os.getenv(
        "SPEAKER_VERIFICATION_MODEL",
        "nvidia/speakerverification_en_titanet_large",
    )
    DEFAULT_THRESHOLD: float = float(
        os.getenv("SPEAKER_VERIFICATION_DEFAULT_THRESHOLD", "0.7")
    )
    ENROLLMENT_MIN_SECONDS: float = float(
        os.getenv("SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS", "3")
    )
    VERIFY_MIN_SECONDS: float = float(
        os.getenv("SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS", "1")
    )
    SAMPLE_RATE: int = int(os.getenv("SPEAKER_VERIFICATION_SAMPLE_RATE", "16000"))
    MAX_UPLOAD_MB: int = int(
        os.getenv("SPEAKER_VERIFICATION_MAX_UPLOAD_MB", "50")
    )

    PROFILES_DIR: Path = Path(
        os.getenv("SPEAKER_VERIFICATION_PROFILES_DIR", "/data/profiles")
    )
    CACHE_DIR: Path = Path(
        os.getenv("SPEAKER_VERIFICATION_CACHE_DIR", "/data/cache")
    )
    LOG_DIR: Path = Path(
        os.getenv("SPEAKER_VERIFICATION_LOG_DIR", "/data/logs")
    )

    VERSION: str = "1.0.0"
    BACKEND: str = "nemo-titanet"


settings = Settings()
