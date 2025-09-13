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
    numpy \
    reedsolo

# Create app directory
WORKDIR /app

# Create a simple API server (MVP implementation with Reed-Solomon)
RUN cat > server.py << 'EOF'
#!/usr/bin/env python3
from flask import Flask, request, jsonify
from flask_cors import CORS
import base64
import json
import time
import reedsolo
import random

app = Flask(__name__)
CORS(app)

# Simulated GGWave functionality for MVP
MODES = ["normal", "fast", "dt", "ultrasonic"]

# Reed-Solomon codec for error correction
rs_codec = reedsolo.RSCodec(10)  # 10 bytes of error correction

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        "status": "healthy",
        "version": "0.5.0",
        "modes_available": MODES,
        "error_correction": "reed-solomon",
        "timestamp": time.time()
    })

@app.route('/api/encode', methods=['POST'])
def encode():
    try:
        data = request.json
        input_data = data.get('data', '')
        mode = data.get('mode', 'normal')
        use_error_correction = data.get('error_correction', True)
        
        if mode not in MODES and mode != 'auto':
            return jsonify({"error": f"Invalid mode: {mode}"}), 400
        
        # Apply Reed-Solomon error correction if enabled
        if use_error_correction:
            # Encode with Reed-Solomon
            encoded_bytes = rs_codec.encode(input_data.encode())
            encoded = base64.b64encode(encoded_bytes).decode()
            error_correction_bytes = 10
        else:
            # Simple base64 encoding without error correction
            encoded = base64.b64encode(input_data.encode()).decode()
            error_correction_bytes = 0
        
        return jsonify({
            "audio": encoded,
            "duration_ms": len(input_data) * 100,  # Simulated duration
            "mode": mode if mode != 'auto' else 'normal',
            "bytes": len(input_data),
            "error_correction_bytes": error_correction_bytes,
            "total_bytes": len(base64.b64decode(encoded))
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route('/api/decode', methods=['POST'])
def decode():
    try:
        data = request.json
        audio_data = data.get('audio', '')
        mode = data.get('mode', 'auto')
        use_error_correction = data.get('error_correction', True)
        simulate_errors = data.get('simulate_errors', False)
        
        # Decode base64
        try:
            decoded_bytes = base64.b64decode(audio_data)
            
            # Simulate transmission errors if requested (for testing)
            if simulate_errors and len(decoded_bytes) > 5:
                decoded_bytes = bytearray(decoded_bytes)
                # Introduce random errors
                num_errors = min(3, len(decoded_bytes) // 10)
                for _ in range(num_errors):
                    pos = random.randint(0, len(decoded_bytes) - 1)
                    decoded_bytes[pos] ^= random.randint(1, 255)
                decoded_bytes = bytes(decoded_bytes)
            
            if use_error_correction:
                # Try to decode with Reed-Solomon error correction
                try:
                    corrected_bytes = rs_codec.decode(decoded_bytes)[0]
                    decoded = corrected_bytes.decode()
                    errors_corrected = len(decoded_bytes) - len(corrected_bytes)
                except reedsolo.ReedSolomonError:
                    # Too many errors to correct
                    decoded = "Error: Unable to correct transmission errors"
                    errors_corrected = -1
            else:
                decoded = decoded_bytes.decode()
                errors_corrected = 0
                
        except Exception as e:
            decoded = f"Invalid audio data: {str(e)}"
            errors_corrected = -1
        
        return jsonify({
            "data": decoded,
            "confidence": 0.95 if errors_corrected >= 0 else 0.0,
            "mode_detected": "normal",
            "errors_corrected": errors_corrected
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