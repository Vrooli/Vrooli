#!/usr/bin/env python3
"""
Validate and update audio metadata YAML file.
Helps prevent drift between actual audio files and metadata.
"""

import os
import yaml
from pathlib import Path
import subprocess
import json
import sys

def get_audio_info(file_path):
    """Extract actual audio properties using ffprobe."""
    try:
        # Skip non-audio files
        if file_path.suffix == '.txt':
            return {
                'format': 'txt',
                'duration': 0,
                'fileSize': os.path.getsize(file_path),
                'sampleRate': 0,
                'bitRate': 0,
                'channels': 0
            }
        
        # Use ffprobe to get audio metadata
        cmd = [
            'ffprobe',
            '-v', 'quiet',
            '-print_format', 'json',
            '-show_format',
            '-show_streams',
            str(file_path)
        ]
        
        result = subprocess.run(cmd, capture_output=True, text=True)
        if result.returncode != 0:
            print(f"Error running ffprobe on {file_path}")
            return None
        
        data = json.loads(result.stdout)
        
        # Extract format info
        format_info = data.get('format', {})
        streams = data.get('streams', [])
        audio_stream = next((s for s in streams if s.get('codec_type') == 'audio'), {})
        
        # Get file extension as format
        file_format = file_path.suffix[1:].lower()
        if file_format == 'opus':
            file_format = 'ogg'
        
        return {
            'format': file_format,
            'duration': float(format_info.get('duration', 0)),
            'fileSize': os.path.getsize(file_path),
            'sampleRate': int(audio_stream.get('sample_rate', 0)),
            'bitRate': int(format_info.get('bit_rate', 0)) // 1000,  # Convert to kbps
            'channels': int(audio_stream.get('channels', 0))
        }
        
    except Exception as e:
        print(f"Error reading {file_path}: {e}")
        return None

def validate_metadata(yaml_file='audio.yaml'):
    """Validate that metadata matches actual files."""
    with open(yaml_file, 'r') as f:
        data = yaml.safe_load(f)
    
    base_dir = Path(__file__).parent
    errors = []
    warnings = []
    
    # Flatten all audio entries
    all_audio = []
    def extract_audio(obj):
        if isinstance(obj, list):
            all_audio.extend(obj)
        elif isinstance(obj, dict):
            for key, value in obj.items():
                if key not in ['testSuites', 'schema', 'version', 'transcriptionExpectations']:
                    extract_audio(value)
    
    extract_audio(data.get('audio', {}))
    
    # Check each audio file
    for audio_meta in all_audio:
        path = audio_meta.get('path')
        if not path:
            errors.append("Audio entry missing 'path' field")
            continue
            
        full_path = base_dir / path
        
        # Check if file exists
        if not full_path.exists():
            errors.append(f"File not found: {path}")
            continue
        
        # Get actual properties
        actual = get_audio_info(full_path)
        if not actual:
            continue
        
        # Compare key properties
        if audio_meta.get('format') != actual['format']:
            errors.append(f"{path}: format mismatch - YAML: {audio_meta.get('format')}, Actual: {actual['format']}")
        
        # Duration can vary slightly, warn if > 5% difference
        yaml_duration = audio_meta.get('duration', 0)
        actual_duration = actual['duration']
        if yaml_duration > 0 and actual_duration > 0 and abs(yaml_duration - actual_duration) / actual_duration > 0.05:
            warnings.append(f"{path}: duration differs > 5% - YAML: {yaml_duration}s, Actual: {actual_duration:.1f}s")
        
        # File size can vary slightly, warn if > 10% difference
        yaml_size = audio_meta.get('fileSize', 0)
        actual_size = actual['fileSize']
        if yaml_size > 0 and abs(yaml_size - actual_size) / actual_size > 0.1:
            warnings.append(f"{path}: file size differs > 10% - YAML: {yaml_size}, Actual: {actual_size}")
        
        # Sample rate should match exactly for audio files
        if actual['sampleRate'] > 0 and audio_meta.get('sampleRate') != actual['sampleRate']:
            errors.append(f"{path}: sample rate mismatch - YAML: {audio_meta.get('sampleRate')}, Actual: {actual['sampleRate']}")
    
    # Find orphaned files (audio without metadata)
    yaml_paths = {audio.get('path') for audio in all_audio if audio.get('path')}
    audio_extensions = ['.wav', '.mp3', '.ogg', '.opus', '.flac', '.m4a', '.aac']
    
    for audio_file in base_dir.iterdir():
        if audio_file.is_file() and audio_file.suffix.lower() in audio_extensions:
            rel_path = audio_file.relative_to(base_dir)
            if str(rel_path) not in yaml_paths:
                warnings.append(f"Audio file without metadata: {rel_path}")
    
    return errors, warnings

def main():
    """Run validation and report results."""
    # Check if ffprobe is available
    try:
        subprocess.run(['ffprobe', '-version'], capture_output=True, check=True)
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("ERROR: ffprobe not found. Please install ffmpeg.")
        print("  Ubuntu/Debian: sudo apt-get install ffmpeg")
        print("  macOS: brew install ffmpeg")
        sys.exit(1)
    
    print("Validating audio metadata...")
    errors, warnings = validate_metadata()
    
    if errors:
        print(f"\n❌ Found {len(errors)} errors:")
        for error in errors:
            print(f"  - {error}")
    
    if warnings:
        print(f"\n⚠️  Found {len(warnings)} warnings:")
        for warning in warnings:
            print(f"  - {warning}")
    
    if not errors and not warnings:
        print("✅ All metadata is valid!")
    
    # Exit with error code if there are errors
    sys.exit(1 if errors else 0)

if __name__ == '__main__':
    main()