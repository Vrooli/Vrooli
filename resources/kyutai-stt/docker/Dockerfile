# Kyutai STT streaming server image.
#
# Based on the official PyTorch CUDA runtime image so torch ships with a CUDA
# build that matches the host driver class. The moshi streaming stack is
# installed on top with pinned versions (see requirements.txt). torch itself is
# intentionally NOT in requirements.txt to avoid clobbering the image's CUDA
# build with a CPU wheel.
#
# Base image is pinned by digest-friendly tag. pytorch 2.5.1 + cuda 12.4 is
# compatible with moshi 0.2.6 and the RTX 40-series (Ada / sm_89).
FROM pytorch/pytorch@sha256:c8268a92a69bd500f8be0e665b2630ee006dadaf7bfbc24249141b15ff622755

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    HF_HOME=/models \
    KYUTAI_STT_DEVICE=cuda \
    KYUTAI_STT_CONTAINER_PORT=8000

# System deps: curl for the container healthcheck, libsndfile for audio I/O,
# build-essential because moshi's streaming generator uses torch.compile
# (inductor), which shells out to a host C/C++ compiler to build kernels at
# first inference — the slim CUDA *runtime* image ships without one.
RUN apt-get update \
    && apt-get install -y --no-install-recommends curl libsndfile1 build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

COPY server.py /app/server.py

# HF cache lives here and is bind-mounted from the host so model weights persist
# across container recreations (multi-GB first-run download).
RUN mkdir -p /models
VOLUME ["/models"]

EXPOSE 8000

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 \
    CMD curl -f http://localhost:8000/ready || exit 1

CMD ["python", "-m", "uvicorn", "server:app", "--host", "0.0.0.0", "--port", "8000"]
