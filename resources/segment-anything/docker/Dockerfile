FROM python:3.10-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    git \
    wget \
    curl \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Set work directory
WORKDIR /app

# Install Python dependencies
RUN pip install --no-cache-dir \
    torch==2.1.2 \
    torchvision==0.16.2 \
    fastapi==0.109.0 \
    uvicorn==0.27.0 \
    numpy==1.24.3 \
    opencv-python-headless==4.9.0.80 \
    Pillow==10.2.0 \
    pydantic==2.5.3 \
    httpx==0.26.0 \
    python-multipart==0.0.6 \
    aiofiles==23.2.1 \
    psutil==5.9.8 \
    tqdm==4.66.1 \
    requests==2.31.0

# Install SAM2
RUN git clone https://github.com/facebookresearch/sam2.git /app/sam2 && \
    cd /app/sam2 && \
    pip install -e .

# Create directories for models and data
RUN mkdir -p /app/models /app/data /app/outputs

# Copy application files
COPY api_server.py /app/
COPY download_models.py /app/
COPY entrypoint.sh /app/

# Make entrypoint executable
RUN chmod +x /app/entrypoint.sh

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV SAM_MODEL_PATH=/app/models
ENV DEFAULT_MODEL=sam2_hiera_small

# Expose API port
EXPOSE 11454

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=120s --retries=5 \
    CMD curl -f http://localhost:11454/health || exit 1

# Run entrypoint script
ENTRYPOINT ["/app/entrypoint.sh"]