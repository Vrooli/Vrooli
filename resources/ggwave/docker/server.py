#!/usr/bin/env python3
from flask import Flask, request, jsonify
from flask_cors import CORS
from flask_socketio import SocketIO, emit, join_room, leave_room
import base64
import json
import time
import reedsolo
import random
import threading
import queue

app = Flask(__name__)
CORS(app)
socketio = SocketIO(app, cors_allowed_origins="*", async_mode='eventlet')

# Simulated GGWave functionality for MVP
MODES = ["normal", "fast", "dt", "ultrasonic"]

# Reed-Solomon codec for error correction
rs_codec = reedsolo.RSCodec(10)  # 10 bytes of error correction

# Active streaming sessions
active_sessions = {}
session_lock = threading.Lock()

def select_optimal_mode(data, environment):
    """
    Intelligently select the best transmission mode based on:
    - Data size
    - Noise level
    - Privacy requirements
    - Distance estimate
    """
    data_size = len(data)
    noise_level = environment.get('noise_level', 'medium')
    privacy_required = environment.get('privacy', False)
    distance = environment.get('distance_meters', 2.0)
    speed_priority = environment.get('speed_priority', False)
    
    # Ultrasonic for privacy
    if privacy_required:
        return 'ultrasonic'
    
    # Small data in quiet environment - use fast mode
    if data_size < 50 and noise_level == 'low':
        return 'fast' if speed_priority else 'normal'
    
    # Large data with speed priority - use DT mode
    if data_size > 100 and speed_priority:
        return 'dt'
    
    # Long distance or high noise - use normal (most robust)
    if distance > 3 or noise_level == 'high':
        return 'normal'
    
    # Default to normal for reliability
    return 'normal'

def get_mode_selection_reason(mode, environment):
    """Explain why a particular mode was selected"""
    reasons = {
        'ultrasonic': 'Selected for privacy - inaudible transmission',
        'normal': 'Selected for reliability - robust against noise',
        'fast': 'Selected for speed in quiet environment',
        'dt': 'Selected for maximum throughput'
    }
    
    if environment:
        if environment.get('privacy'):
            return reasons.get('ultrasonic')
        if environment.get('noise_level') == 'high':
            return 'High noise detected - using most robust mode'
        if environment.get('speed_priority'):
            return 'Speed prioritized - using fastest available mode'
    
    return reasons.get(mode, 'Default mode selected')

def calculate_transmission_duration(data_size, mode):
    """Calculate estimated transmission duration based on mode"""
    # Bytes per second for each mode
    rates = {
        'normal': 36,     # Average of 8-64
        'fast': 48,       # Average of 32-64  
        'dt': 250,        # Average of 64-500
        'ultrasonic': 36  # Same as normal but inaudible
    }
    
    rate = rates.get(mode, 36)
    duration_seconds = data_size / rate
    return int(duration_seconds * 1000)  # Convert to milliseconds

def detect_transmission_mode(audio_data):
    """
    Detect transmission mode from audio signal characteristics
    In production, this would analyze frequency spectrum
    """
    # Simulated detection based on data patterns
    data_len = len(audio_data)
    
    # Simple heuristic for demo purposes
    if data_len % 7 == 0:
        return 'ultrasonic'
    elif data_len % 5 == 0:
        return 'dt'
    elif data_len % 3 == 0:
        return 'fast'
    else:
        return 'normal'

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        "status": "healthy",
        "version": "0.6.0",
        "modes_available": MODES,
        "error_correction": "reed-solomon",
        "websocket_support": True,
        "timestamp": time.time()
    })

@app.route('/api/encode', methods=['POST'])
def encode():
    try:
        data = request.json
        input_data = data.get('data', '')
        mode = data.get('mode', 'normal')
        use_error_correction = data.get('error_correction', True)
        environment = data.get('environment', {})
        
        if mode not in MODES and mode != 'auto':
            return jsonify({"error": f"Invalid mode: {mode}"}), 400
        
        # Automatic protocol selection based on environment
        if mode == 'auto':
            mode = select_optimal_mode(input_data, environment)
        
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
            "duration_ms": calculate_transmission_duration(len(input_data), mode),
            "mode": mode,
            "mode_selection_reason": get_mode_selection_reason(mode, environment),
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
        
        # Auto-detect mode from signal characteristics (simulated)
        if mode == 'auto':
            mode = detect_transmission_mode(audio_data)
        
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
            "mode_detected": mode,
            "errors_corrected": errors_corrected
        })
    except Exception as e:
        return jsonify({"error": str(e)}), 500

# WebSocket event handlers for real-time streaming
@socketio.on('connect')
def handle_connect():
    """Handle client connection"""
    session_id = request.sid
    with session_lock:
        active_sessions[session_id] = {
            'connected': True,
            'mode': 'normal',
            'error_correction': True,
            'stream_queue': queue.Queue()
        }
    emit('connected', {'session_id': session_id, 'status': 'connected'})

@socketio.on('disconnect')
def handle_disconnect():
    """Handle client disconnection"""
    session_id = request.sid
    with session_lock:
        if session_id in active_sessions:
            del active_sessions[session_id]
    print(f"Client disconnected: {session_id}")

@socketio.on('configure')
def handle_configure(data):
    """Configure streaming parameters"""
    session_id = request.sid
    with session_lock:
        if session_id in active_sessions:
            session = active_sessions[session_id]
            session['mode'] = data.get('mode', session['mode'])
            session['error_correction'] = data.get('error_correction', session['error_correction'])
    emit('configured', {
        'mode': session['mode'],
        'error_correction': session['error_correction']
    })

@socketio.on('stream_encode')
def handle_stream_encode(data):
    """Handle streaming encode requests"""
    session_id = request.sid
    try:
        with session_lock:
            if session_id not in active_sessions:
                emit('error', {'error': 'Session not found'})
                return
            session = active_sessions[session_id]
        
        # Get data chunk to encode
        chunk_data = data.get('data', '')
        mode = session['mode']
        use_error_correction = session['error_correction']
        
        # Apply Reed-Solomon if enabled
        if use_error_correction:
            encoded_bytes = rs_codec.encode(chunk_data.encode())
            encoded = base64.b64encode(encoded_bytes).decode()
            error_correction_bytes = 10
        else:
            encoded = base64.b64encode(chunk_data.encode()).decode()
            error_correction_bytes = 0
        
        # Emit encoded chunk back to client
        emit('encoded_chunk', {
            'audio': encoded,
            'chunk_id': data.get('chunk_id', 0),
            'mode': mode,
            'bytes': len(chunk_data),
            'error_correction_bytes': error_correction_bytes
        })
        
    except Exception as e:
        emit('error', {'error': str(e)})

@socketio.on('stream_decode')
def handle_stream_decode(data):
    """Handle streaming decode requests"""
    session_id = request.sid
    try:
        with session_lock:
            if session_id not in active_sessions:
                emit('error', {'error': 'Session not found'})
                return
            session = active_sessions[session_id]
        
        # Get audio chunk to decode
        audio_chunk = data.get('audio', '')
        use_error_correction = session['error_correction']
        simulate_errors = data.get('simulate_errors', False)
        
        # Decode base64
        decoded_bytes = base64.b64decode(audio_chunk)
        
        # Simulate errors if requested
        if simulate_errors and len(decoded_bytes) > 5:
            decoded_bytes = bytearray(decoded_bytes)
            num_errors = min(3, len(decoded_bytes) // 10)
            for _ in range(num_errors):
                pos = random.randint(0, len(decoded_bytes) - 1)
                decoded_bytes[pos] ^= random.randint(1, 255)
            decoded_bytes = bytes(decoded_bytes)
        
        if use_error_correction:
            try:
                corrected_bytes = rs_codec.decode(decoded_bytes)[0]
                decoded = corrected_bytes.decode()
                errors_corrected = len(decoded_bytes) - len(corrected_bytes)
            except reedsolo.ReedSolomonError:
                decoded = "Error: Too many transmission errors"
                errors_corrected = -1
        else:
            decoded = decoded_bytes.decode()
            errors_corrected = 0
        
        # Emit decoded chunk back to client
        emit('decoded_chunk', {
            'data': decoded,
            'chunk_id': data.get('chunk_id', 0),
            'confidence': 0.95 if errors_corrected >= 0 else 0.0,
            'errors_corrected': errors_corrected
        })
        
    except Exception as e:
        emit('error', {'error': str(e)})

@socketio.on('join_room')
def handle_join_room(data):
    """Join a specific room for group transmission"""
    room = data.get('room', 'default')
    join_room(room)
    emit('joined_room', {'room': room})

@socketio.on('leave_room')
def handle_leave_room(data):
    """Leave a specific room"""
    room = data.get('room', 'default')
    leave_room(room)
    emit('left_room', {'room': room})

@socketio.on('broadcast')
def handle_broadcast(data):
    """Broadcast encoded data to all clients in a room"""
    room = data.get('room', 'default')
    audio_data = data.get('audio', '')
    emit('broadcast_data', {'audio': audio_data, 'sender': request.sid}, room=room, include_self=False)

if __name__ == '__main__':
    port = 8196
    print(f"Starting GGWave API server with WebSocket support on port {port}", flush=True)
    socketio.run(app, host='0.0.0.0', port=port, debug=False)