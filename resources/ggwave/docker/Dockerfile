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
    numpy

# Create app directory
WORKDIR /app

# Create a simple API server (MVP implementation)
RUN cat > server.py << 'EOF'
#!/usr/bin/env python3
from flask import Flask, request, jsonify
from flask_cors import CORS
import base64
import json
import time

app = Flask(__name__)
CORS(app)

# Simulated GGWave functionality for MVP
MODES = ["normal", "fast", "dt", "ultrasonic"]

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        "status": "healthy",
        "version": "0.4.0",
        "modes_available": MODES,
        "timestamp": time.time()
    })

@app.route('/api/encode', methods=['POST'])
def encode():
    try:
        data = request.json
        input_data = data.get('data', '')
        mode = data.get('mode', 'normal')
        
        if mode not in MODES and mode != 'auto':
            return jsonify({"error": f"Invalid mode: {mode}"}), 400
        
        # Simulate encoding (in production, would use actual ggwave)
        # For MVP, just base64 encode the data
        encoded = base64.b64encode(input_data.encode()).decode()
        
        return jsonify({
            "audio": encoded,
            "duration_ms": len(input_data) * 100,  # Simulated duration
            "mode": mode if mode != 'auto' else 'normal',
            "bytes": len(input_data)
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route('/api/decode', methods=['POST'])
def decode():
    try:
        data = request.json
        audio_data = data.get('audio', '')
        mode = data.get('mode', 'auto')
        
        # Simulate decoding (in production, would use actual ggwave)
        # For MVP, just base64 decode
        try:
            decoded = base64.b64decode(audio_data).decode()
        except:
            decoded = "Invalid audio data"
        
        return jsonify({
            "data": decoded,
            "confidence": 0.95,
            "mode_detected": "normal",
            "errors_corrected": 0
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8196, debug=False)
EOF

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