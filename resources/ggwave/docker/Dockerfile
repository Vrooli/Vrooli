# GGWave Docker Container
# Minimal implementation for MVP

FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    python3 \
    python3-pip \
    alsa-utils \
    pulseaudio \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies for API server
RUN pip3 install --no-cache-dir \
    flask \
    flask-cors \
    flask-socketio \
    python-socketio \
    eventlet \
    numpy \
    reedsolo

# Create app directory
WORKDIR /app

# Copy server file
COPY server.py /app/

# Make server executable
RUN chmod +x server.py

# Create data directory
RUN mkdir -p /data/logs

# Expose port
EXPOSE 8196

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8196/health || exit 1

# Run the server
CMD ["python3", "server.py"]