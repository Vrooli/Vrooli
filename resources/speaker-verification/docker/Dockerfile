# Speaker Verification server image.
#
# GPU-capable. Based on the official PyTorch CUDA runtime image so torch ships
# with a CUDA build that matches the host driver class (Ada / sm_89 on the RTX
# 40-series). torch is intentionally NOT in requirements.txt — reinstalling it
# from PyPI would clobber the CUDA build with a CPU-only wheel (the bug this
# image previously shipped, which silently pinned the service to CPU).
#
# The resource manager applies the GPU overlay + SPEAKER_VERIFICATION_DEVICE=cuda
# automatically when the nvidia probe passes (see resource.json `gpu` block). The
# server falls back to CPU when torch.cuda.is_available() is False, so the same
# image runs unchanged on CPU-only hosts — no separate CPU image to maintain.
#
# Base tag matches the kyutai-stt resource so the multi-GB CUDA layers are shared
# in the local image cache. cuda 12.4 + torch 2.5.1 supports the RTX 40-series.
FROM pytorch/pytorch@sha256:c8268a92a69bd500f8be0e665b2630ee006dadaf7bfbc24249141b15ff622755

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1

# ffmpeg robustly decodes WebM/Opus uploads from the browser embed before handing
# PCM to torchaudio; libsndfile backs soundfile; curl backs the compose healthcheck.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ffmpeg \
        curl \
        libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# The base image ships torch + torchvision but not torchaudio, so install
# torchaudio from the matching CUDA wheel index (its only dependency is the
# already-present torch, so this does NOT pull a CPU wheel). Then the remaining
# pinned deps; torch/torchaudio are kept out of requirements.txt for the same
# clobber-protection reason.
COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir --upgrade pip==24.0 \
    && pip install --no-cache-dir \
        --index-url https://download.pytorch.org/whl/cu124 \
        torchaudio==2.5.1 \
    && pip install --no-cache-dir -r requirements.txt

COPY vad.py /app/vad.py
COPY server.py /app/server.py
# Unit tests ride along so `python -m unittest` runs inside the image.
COPY test_vad.py /app/test_vad.py
COPY test_profiles.py /app/test_profiles.py

# Persisted across container restarts via compose volumes. Device defaults to
# cuda; the server downgrades to cpu at startup when no GPU is visible.
ENV SPEAKER_VERIFICATION_PROFILE_DIR=/data/profiles \
    SPEAKER_VERIFICATION_MODEL_CACHE=/data/model-cache \
    SPEAKER_VERIFICATION_TEMP_DIR=/tmp/speaker-verification \
    SPEAKER_VERIFICATION_DEVICE=cuda \
    HF_HOME=/data/model-cache/hf

EXPOSE 8000

CMD ["python", "-m", "uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]
