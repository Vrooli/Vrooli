FROM python:3.9-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip install --no-cache-dir \
    torch==2.1.0 \
    torchaudio \
    audiocraft \
    flask \
    flask-cors \
    numpy \
    scipy

# Create app directory
WORKDIR /app

# Copy API server script
COPY api_server.py /app/

# Expose port
EXPOSE 7862

# Run API server
CMD ["python", "api_server.py"]
