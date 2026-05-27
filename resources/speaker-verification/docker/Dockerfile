# Speaker Verification server image.
# CPU-first: ECAPA-TDNN is a small model and runs comfortably on CPU. GPU is an
# optional overlay (docker-compose.gpu.yml) that flips SPEAKER_VERIFICATION_DEVICE
# to "cuda"; this base image bundles the CPU torch wheels.
FROM python:3.11-slim-bookworm

# ffmpeg is required to robustly decode WebM/Opus uploads from the browser embed
# before handing PCM to torchaudio. curl backs the compose healthcheck.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ffmpeg \
        curl \
        libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install torch / torchaudio from the CPU wheel index first so the resolver does
# not pull CUDA wheels, then the remaining pinned deps.
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir --upgrade pip==24.0 \
    && pip install --no-cache-dir \
        --index-url https://download.pytorch.org/whl/cpu \
        torch==2.2.2 torchaudio==2.2.2 \
    && pip install --no-cache-dir -r requirements.txt

COPY server.py /app/server.py

# Persisted across container restarts via compose volumes.
ENV SPEAKER_VERIFICATION_PROFILE_DIR=/data/profiles \
    SPEAKER_VERIFICATION_MODEL_CACHE=/data/model-cache \
    SPEAKER_VERIFICATION_TEMP_DIR=/tmp/speaker-verification \
    SPEAKER_VERIFICATION_DEVICE=cpu \
    HF_HOME=/data/model-cache/hf

EXPOSE 8000

CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]
