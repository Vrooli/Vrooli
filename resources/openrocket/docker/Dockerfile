FROM openjdk:11-jre-slim

# Install dependencies
RUN apt-get update && apt-get install -y \
    wget \
    xvfb \
    python3 \
    python3-pip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/openrocket

# Download OpenRocket JAR (latest stable)
RUN wget -q https://github.com/openrocket/openrocket/releases/download/release-22.02/OpenRocket-22.02.jar \
    -O OpenRocket.jar

# Install Python dependencies for API
RUN pip3 install flask flask-cors pyyaml

# Create data directories
RUN mkdir -p /data/designs /data/simulations /data/models

# Add API server script
COPY api_server.py /opt/openrocket/

# Add atmosphere models
COPY ../models/standard_atmosphere.json /data/models/

EXPOSE 9513

CMD ["python3", "/opt/openrocket/api_server.py"]