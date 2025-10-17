# Use pre-built FreeCAD image with AppImage
FROM amancevice/freecad:latest

# Install additional dependencies for API server
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    curl \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies for API server
RUN pip3 install \
    flask \
    flask-cors \
    requests

# Create necessary directories
RUN mkdir -p /data /projects /scripts /exports /app

# Copy API server script
COPY api_server.py /app/

# Set environment variables
ENV DISPLAY=:99
ENV PYTHONPATH=/usr/lib/freecad/lib

# Create a simple health check script
RUN echo '#!/bin/bash\necho "{\"status\":\"healthy\",\"service\":\"freecad\"}"' > /app/health.sh && \
    chmod +x /app/health.sh

# Expose port
EXPOSE 8080

# Set working directory
WORKDIR /app

# Start script to run Xvfb and API server
CMD Xvfb :99 -screen 0 1920x1080x24 & \
    python3 /app/api_server.py