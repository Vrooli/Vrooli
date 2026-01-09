#!/usr/bin/env python3
import os
import json
from flask import Flask, request, jsonify, send_file
from flask_cors import CORS
import torch
from audiocraft.models import MusicGen, AudioGen
from audiocraft.data.audio import audio_write
import tempfile
import traceback

app = Flask(__name__)
CORS(app)

# Load models on startup
print("Loading AudioCraft models...")
try:
    musicgen_model = MusicGen.get_pretrained('facebook/musicgen-medium')
    print("MusicGen model loaded")
except Exception as e:
    print(f"Failed to load MusicGen: {e}")
    musicgen_model = None

try:
    audiogen_model = AudioGen.get_pretrained('facebook/audiogen-medium')
    print("AudioGen model loaded")
except Exception as e:
    print(f"Failed to load AudioGen: {e}")
    audiogen_model = None

@app.route('/health', methods=['GET'])
def health():
    return jsonify({
        'status': 'healthy',
        'musicgen': musicgen_model is not None,
        'audiogen': audiogen_model is not None
    })

@app.route('/api/generate/music', methods=['POST'])
def generate_music():
    try:
        if not musicgen_model:
            return jsonify({'error': 'MusicGen model not loaded'}), 503
            
        data = request.json
        prompt = data.get('prompt', 'relaxing ambient music')
        duration = min(data.get('duration', 10), 120)
        
        # Set generation parameters
        musicgen_model.set_generation_params(duration=duration)
        
        # Generate music
        wav = musicgen_model.generate([prompt])
        
        # Save to temporary file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as f:
            audio_write(f.name[:-4], wav[0].cpu(), musicgen_model.sample_rate)
            return send_file(f.name, mimetype='audio/wav')
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/generate/sound', methods=['POST'])
def generate_sound():
    try:
        if not audiogen_model:
            return jsonify({'error': 'AudioGen model not loaded'}), 503
            
        data = request.json
        prompt = data.get('prompt', 'rain sound')
        duration = min(data.get('duration', 5), 30)
        
        # Set generation parameters
        audiogen_model.set_generation_params(duration=duration)
        
        # Generate sound
        wav = audiogen_model.generate([prompt])
        
        # Save to temporary file
        with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as f:
            audio_write(f.name[:-4], wav[0].cpu(), audiogen_model.sample_rate)
            return send_file(f.name, mimetype='audio/wav')
            
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/api/models', methods=['GET'])
def list_models():
    return jsonify({
        'musicgen': {
            'loaded': musicgen_model is not None,
            'variants': ['small', 'medium', 'large', 'melody']
        },
        'audiogen': {
            'loaded': audiogen_model is not None,
            'variants': ['medium']
        }
    })

if __name__ == '__main__':
    port = int(os.environ.get('AUDIOCRAFT_PORT', 7862))
    app.run(host='0.0.0.0', port=port, debug=False)
