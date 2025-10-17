#!/usr/bin/env python3
"""
FFmpeg RESTful API Server
Provides secure endpoints for media processing operations
"""

import http.server
import socketserver
import json
import os
import subprocess
import tempfile
import sys
import hashlib
import time
import re
import shutil
from urllib.parse import parse_qs, urlparse
from pathlib import Path

# Configuration from environment
PORT = int(os.environ.get('API_PORT', 8097))
WEB_ROOT = os.environ.get('WEB_ROOT', '.')
UPLOAD_DIR = os.environ.get('UPLOAD_DIR', '/tmp/ffmpeg-uploads')
OUTPUT_DIR = os.environ.get('OUTPUT_DIR', '/tmp/ffmpeg-output')
SCRIPT_DIR = os.environ.get('SCRIPT_DIR', '.')
MAX_FILE_SIZE = int(os.environ.get('MAX_FILE_SIZE', 500 * 1024 * 1024))  # 500MB default
ALLOWED_EXTENSIONS = {'.mp4', '.avi', '.mov', '.mkv', '.mp3', '.wav', '.flac', '.webm', '.ogg', '.m4a', '.aac'}

# Create required directories
Path(UPLOAD_DIR).mkdir(parents=True, exist_ok=True)
Path(OUTPUT_DIR).mkdir(parents=True, exist_ok=True)

# Stats tracking
STATS_FILE = '/tmp/ffmpeg-stats.json'
if not os.path.exists(STATS_FILE):
    with open(STATS_FILE, 'w') as f:
        json.dump({'activeJobs': 0, 'completedJobs': 0, 'totalProcessingTime': 0, 'errors': 0}, f)

def update_stats(field, value):
    """Update statistics atomically"""
    try:
        with open(STATS_FILE, 'r') as f:
            stats = json.load(f)
        stats[field] = stats.get(field, 0) + value
        with open(STATS_FILE, 'w') as f:
            json.dump(stats, f)
    except:
        pass

def validate_filename(filename):
    """Validate and sanitize filename"""
    # Remove path traversal attempts
    filename = os.path.basename(filename)
    # Remove dangerous characters
    filename = re.sub(r'[^a-zA-Z0-9._-]', '_', filename)
    # Check extension
    ext = os.path.splitext(filename)[1].lower()
    if ext not in ALLOWED_EXTENSIONS:
        return None
    return filename

def validate_preset(preset):
    """Validate preset name"""
    valid_presets = [
        'mp4-h264', 'mp4-h265', 'webm-vp9', 'mp3-320', 'mp3-192',
        'aac-256', 'flac', 'wav', 'gif', 'thumbnail', 'optimize-web',
        'optimize-mobile', 'hdr-to-sdr', 'upscale-1080p', 'upscale-4k'
    ]
    return preset if preset in valid_presets else None

def validate_ffmpeg_options(options):
    """Validate custom ffmpeg options for security"""
    # Block dangerous patterns
    dangerous_patterns = [
        r'[;&|`$]',  # Command injection attempts
        r'file://',  # File protocol
        r'data://',  # Data protocol
        r'\.\./',    # Path traversal
        r'^/',       # Absolute paths
    ]
    
    for pattern in dangerous_patterns:
        if re.search(pattern, options):
            return None
    
    # Only allow safe ffmpeg options
    allowed_patterns = [
        r'^-c:v\s+\w+$',  # Video codec
        r'^-c:a\s+\w+$',  # Audio codec
        r'^-b:v\s+\d+[kM]?$',  # Video bitrate
        r'^-b:a\s+\d+[kM]?$',  # Audio bitrate
        r'^-r\s+\d+$',    # Frame rate
        r'^-s\s+\d+x\d+$',  # Resolution
        r'^-crf\s+\d+$',  # Quality level
        r'^-preset\s+(ultrafast|superfast|veryfast|faster|fast|medium|slow|slower|veryslow)$',
    ]
    
    # Split options and validate each
    option_parts = options.strip().split()
    validated_parts = []
    i = 0
    while i < len(option_parts):
        if i + 1 < len(option_parts):
            option_pair = f"{option_parts[i]} {option_parts[i+1]}"
            if any(re.match(pattern, option_pair) for pattern in allowed_patterns):
                validated_parts.extend([option_parts[i], option_parts[i+1]])
                i += 2
                continue
        i += 1
    
    return ' '.join(validated_parts) if validated_parts else ''

class FFmpegAPIHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=WEB_ROOT, **kwargs)
    
    def do_GET(self):
        """Handle GET requests"""
        parsed_path = urlparse(self.path)
        
        # API endpoints
        if parsed_path.path == '/api/health':
            self.send_json_response({'status': 'healthy', 'service': 'ffmpeg-api', 'version': '2.0'})
        
        elif parsed_path.path == '/api/stats':
            self.handle_stats()
        
        elif parsed_path.path == '/api/presets':
            self.handle_presets_list()
        
        elif parsed_path.path.startswith('/api/download/'):
            self.handle_download(parsed_path.path)
        
        elif parsed_path.path.startswith('/api/'):
            self.send_error_response(404, 'Endpoint not found')
        
        else:
            # Serve static files
            super().do_GET()
    
    def do_POST(self):
        """Handle POST requests"""
        parsed_path = urlparse(self.path)
        
        if not parsed_path.path.startswith('/api/'):
            self.send_error_response(404, 'Not an API endpoint')
            return
        
        # Parse content
        content_type = self.headers.get('Content-Type', '')
        
        if parsed_path.path == '/api/convert':
            self.handle_convert()
        
        elif parsed_path.path == '/api/extract':
            self.handle_extract()
        
        elif parsed_path.path == '/api/info':
            self.handle_info()
        
        elif parsed_path.path == '/api/stream':
            self.handle_stream()
        
        elif parsed_path.path == '/api/batch':
            self.handle_batch()
        
        else:
            self.send_error_response(404, 'Endpoint not found')
    
    def do_OPTIONS(self):
        """Handle CORS preflight"""
        self.send_response(200)
        self.send_header('Access-Control-Allow-Origin', '*')
        self.send_header('Access-Control-Allow-Methods', 'GET, POST, OPTIONS')
        self.send_header('Access-Control-Allow-Headers', 'Content-Type')
        self.end_headers()
    
    def handle_stats(self):
        """Get system and processing stats"""
        try:
            with open(STATS_FILE, 'r') as f:
                stats = json.load(f)
            
            # Add system info
            cpu_info = subprocess.run(['top', '-bn1'], capture_output=True, text=True, timeout=2)
            cpu_usage = 0
            if cpu_info.returncode == 0:
                for line in cpu_info.stdout.split('\n'):
                    if 'Cpu(s)' in line or '%Cpu' in line:
                        match = re.search(r'(\d+\.\d+).*id', line)
                        if match:
                            cpu_usage = 100 - float(match.group(1))
                            break
            
            stats['cpuUsage'] = round(cpu_usage, 1)
            stats['gpuAvailable'] = os.path.exists('/dev/nvidia0') or os.path.exists('/dev/dri/renderD128')
            
            self.send_json_response(stats)
        except Exception as e:
            self.send_error_response(500, f'Failed to get stats: {str(e)}')
    
    def handle_presets_list(self):
        """List available presets"""
        presets = {
            'video': [
                {'id': 'mp4-h264', 'name': 'MP4 H.264', 'description': 'Standard MP4 with H.264 codec'},
                {'id': 'mp4-h265', 'name': 'MP4 H.265', 'description': 'MP4 with H.265/HEVC for better compression'},
                {'id': 'webm-vp9', 'name': 'WebM VP9', 'description': 'WebM format with VP9 codec'},
                {'id': 'optimize-web', 'name': 'Web Optimized', 'description': 'Optimized for web streaming'},
                {'id': 'optimize-mobile', 'name': 'Mobile Optimized', 'description': 'Optimized for mobile devices'},
            ],
            'audio': [
                {'id': 'mp3-320', 'name': 'MP3 320kbps', 'description': 'High quality MP3'},
                {'id': 'mp3-192', 'name': 'MP3 192kbps', 'description': 'Standard quality MP3'},
                {'id': 'aac-256', 'name': 'AAC 256kbps', 'description': 'AAC audio format'},
                {'id': 'flac', 'name': 'FLAC Lossless', 'description': 'Lossless audio compression'},
                {'id': 'wav', 'name': 'WAV Uncompressed', 'description': 'Uncompressed WAV format'},
            ],
            'special': [
                {'id': 'gif', 'name': 'Animated GIF', 'description': 'Convert video to GIF'},
                {'id': 'thumbnail', 'name': 'Thumbnail', 'description': 'Extract thumbnail image'},
                {'id': 'hdr-to-sdr', 'name': 'HDR to SDR', 'description': 'Convert HDR to SDR'},
                {'id': 'upscale-1080p', 'name': 'Upscale to 1080p', 'description': 'Upscale to Full HD'},
                {'id': 'upscale-4k', 'name': 'Upscale to 4K', 'description': 'Upscale to 4K resolution'},
            ]
        }
        self.send_json_response(presets)
    
    def handle_convert(self):
        """Handle media conversion request"""
        try:
            # Parse multipart form data
            content_type = self.headers.get('Content-Type', '')
            if not content_type.startswith('multipart/form-data'):
                # Try JSON format
                content_length = int(self.headers.get('Content-Length', 0))
                if content_length > MAX_FILE_SIZE:
                    self.send_error_response(413, 'File too large')
                    return
                
                body = self.rfile.read(content_length)
                data = json.loads(body)
                
                input_path = data.get('input')
                preset = data.get('preset', 'mp4-h264')
                options = data.get('options', '')
            else:
                # Handle multipart upload
                boundary = content_type.split('boundary=')[1]
                content_length = int(self.headers.get('Content-Length', 0))
                
                if content_length > MAX_FILE_SIZE:
                    self.send_error_response(413, 'File too large')
                    return
                
                # Simple multipart parser (avoiding deprecated cgi module)
                body = self.rfile.read(content_length)
                parts = body.split(f'--{boundary}'.encode())
                
                input_path = None
                preset = 'mp4-h264'
                options = ''
                
                for part in parts[1:-1]:
                    if b'Content-Disposition' in part:
                        lines = part.split(b'\r\n')
                        header = lines[1].decode('utf-8')
                        
                        if 'name="file"' in header and 'filename=' in header:
                            # Extract filename
                            filename_match = re.search(r'filename="([^"]+)"', header)
                            if filename_match:
                                filename = validate_filename(filename_match.group(1))
                                if not filename:
                                    self.send_error_response(400, 'Invalid file type')
                                    return
                                
                                # Save file
                                file_content = b'\r\n'.join(lines[4:])
                                input_path = os.path.join(UPLOAD_DIR, f"{int(time.time())}_{filename}")
                                with open(input_path, 'wb') as f:
                                    f.write(file_content)
                        
                        elif 'name="preset"' in header:
                            preset_value = lines[-1].decode('utf-8').strip()
                            preset = validate_preset(preset_value) or preset
                        
                        elif 'name="options"' in header:
                            options_value = lines[-1].decode('utf-8').strip()
                            options = validate_ffmpeg_options(options_value) or ''
            
            if not input_path or not os.path.exists(input_path):
                self.send_error_response(400, 'No input file provided')
                return
            
            # Validate preset
            preset = validate_preset(preset) or 'mp4-h264'
            options = validate_ffmpeg_options(options) or ''
            
            # Update stats
            update_stats('activeJobs', 1)
            start_time = time.time()
            
            # Build ffmpeg command based on preset
            output_ext = {
                'mp4-h264': '.mp4', 'mp4-h265': '.mp4', 'webm-vp9': '.webm',
                'mp3-320': '.mp3', 'mp3-192': '.mp3', 'aac-256': '.m4a',
                'flac': '.flac', 'wav': '.wav', 'gif': '.gif',
                'thumbnail': '.jpg'
            }.get(preset, '.mp4')
            
            output_filename = f"converted_{int(time.time())}{output_ext}"
            output_path = os.path.join(OUTPUT_DIR, output_filename)
            
            # Build ffmpeg command
            ffmpeg_cmd = ['ffmpeg', '-i', input_path]
            
            # Add preset-specific options
            if preset == 'mp4-h264':
                ffmpeg_cmd.extend(['-c:v', 'libx264', '-preset', 'fast', '-crf', '23', '-c:a', 'aac'])
            elif preset == 'mp4-h265':
                ffmpeg_cmd.extend(['-c:v', 'libx265', '-preset', 'fast', '-crf', '28', '-c:a', 'aac'])
            elif preset == 'webm-vp9':
                ffmpeg_cmd.extend(['-c:v', 'libvpx-vp9', '-crf', '30', '-b:v', '0'])
            elif preset == 'mp3-320':
                ffmpeg_cmd.extend(['-ab', '320k'])
            elif preset == 'mp3-192':
                ffmpeg_cmd.extend(['-ab', '192k'])
            elif preset == 'aac-256':
                ffmpeg_cmd.extend(['-c:a', 'aac', '-ab', '256k'])
            elif preset == 'flac':
                ffmpeg_cmd.extend(['-c:a', 'flac'])
            elif preset == 'wav':
                ffmpeg_cmd.extend(['-c:a', 'pcm_s16le'])
            elif preset == 'gif':
                ffmpeg_cmd.extend(['-vf', 'fps=10,scale=320:-1:flags=lanczos', '-c:v', 'gif'])
            elif preset == 'thumbnail':
                ffmpeg_cmd.extend(['-vframes', '1', '-vf', 'select=eq(n\\,0)'])
            elif preset == 'optimize-web':
                ffmpeg_cmd.extend(['-c:v', 'libx264', '-preset', 'fast', '-crf', '23', '-movflags', '+faststart', '-c:a', 'aac'])
            elif preset == 'optimize-mobile':
                ffmpeg_cmd.extend(['-c:v', 'libx264', '-profile:v', 'baseline', '-level', '3.0', '-preset', 'fast', '-crf', '23', '-c:a', 'aac'])
            elif preset == 'hdr-to-sdr':
                ffmpeg_cmd.extend(['-vf', 'zscale=t=linear:npl=100,format=gbrpf32le,zscale=p=bt709,tonemap=tonemap=hable:desat=0,zscale=t=bt709:m=bt709:r=tv,format=yuv420p', '-c:v', 'libx264', '-crf', '23'])
            elif preset == 'upscale-1080p':
                ffmpeg_cmd.extend(['-vf', 'scale=1920:1080:flags=lanczos', '-c:v', 'libx264', '-preset', 'slow', '-crf', '20'])
            elif preset == 'upscale-4k':
                ffmpeg_cmd.extend(['-vf', 'scale=3840:2160:flags=lanczos', '-c:v', 'libx265', '-preset', 'slow', '-crf', '22'])
            
            # Add custom options if validated
            if options:
                ffmpeg_cmd.extend(options.split())
            
            # Add output file
            ffmpeg_cmd.extend(['-y', output_path])
            
            # Execute conversion with timeout
            result = subprocess.run(ffmpeg_cmd, capture_output=True, text=True, timeout=300)
            
            # Update stats
            update_stats('activeJobs', -1)
            duration = time.time() - start_time
            
            if result.returncode == 0 and os.path.exists(output_path):
                update_stats('completedJobs', 1)
                update_stats('totalProcessingTime', int(duration))
                
                # Get output file info
                file_size = os.path.getsize(output_path)
                
                response = {
                    'success': True,
                    'output': output_filename,
                    'downloadUrl': f'/api/download/{output_filename}',
                    'size': file_size,
                    'duration': round(duration, 2),
                    'preset': preset
                }
                self.send_json_response(response)
            else:
                update_stats('errors', 1)
                self.send_error_response(500, f'Conversion failed: {result.stderr[:500]}')
            
            # Clean up input file if it was uploaded
            if input_path.startswith(UPLOAD_DIR):
                try:
                    os.remove(input_path)
                except:
                    pass
                    
        except subprocess.TimeoutExpired:
            update_stats('activeJobs', -1)
            update_stats('errors', 1)
            self.send_error_response(408, 'Conversion timeout')
        except Exception as e:
            update_stats('activeJobs', -1)
            update_stats('errors', 1)
            self.send_error_response(500, f'Conversion error: {str(e)}')
    
    def handle_extract(self):
        """Handle extraction request (audio, frames, etc)"""
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length)
            data = json.loads(body)
            
            input_path = data.get('input')
            extract_type = data.get('type', 'audio')
            
            if not input_path or not os.path.exists(input_path):
                self.send_error_response(400, 'Input file not found')
                return
            
            # Validate extract type
            valid_types = ['audio', 'audio-wav', 'frames', 'thumbnail', 'subtitles']
            if extract_type not in valid_types:
                self.send_error_response(400, 'Invalid extraction type')
                return
            
            update_stats('activeJobs', 1)
            start_time = time.time()
            
            output_path = None
            ffmpeg_cmd = ['ffmpeg', '-i', input_path]
            
            if extract_type == 'audio':
                output_path = os.path.join(OUTPUT_DIR, f"audio_{int(time.time())}.mp3")
                ffmpeg_cmd.extend(['-vn', '-ab', '192k', '-y', output_path])
            
            elif extract_type == 'audio-wav':
                output_path = os.path.join(OUTPUT_DIR, f"audio_{int(time.time())}.wav")
                ffmpeg_cmd.extend(['-vn', '-c:a', 'pcm_s16le', '-y', output_path])
            
            elif extract_type == 'frames':
                output_dir = os.path.join(OUTPUT_DIR, f"frames_{int(time.time())}")
                os.makedirs(output_dir, exist_ok=True)
                output_path = output_dir
                frame_rate = data.get('fps', 1)
                ffmpeg_cmd.extend(['-vf', f'fps={frame_rate}', f'{output_dir}/frame_%04d.jpg'])
            
            elif extract_type == 'thumbnail':
                output_path = os.path.join(OUTPUT_DIR, f"thumb_{int(time.time())}.jpg")
                position = data.get('position', 0)
                ffmpeg_cmd.extend(['-ss', str(position), '-vframes', '1', '-y', output_path])
            
            elif extract_type == 'subtitles':
                output_path = os.path.join(OUTPUT_DIR, f"subs_{int(time.time())}.srt")
                stream = data.get('stream', 0)
                ffmpeg_cmd.extend(['-map', f'0:s:{stream}', '-c:s', 'srt', '-y', output_path])
            
            # Execute extraction
            result = subprocess.run(ffmpeg_cmd, capture_output=True, text=True, timeout=120)
            
            update_stats('activeJobs', -1)
            duration = time.time() - start_time
            
            if result.returncode == 0:
                update_stats('completedJobs', 1)
                update_stats('totalProcessingTime', int(duration))
                
                response = {
                    'success': True,
                    'output': os.path.basename(output_path) if output_path else 'frames',
                    'type': extract_type,
                    'duration': round(duration, 2)
                }
                
                if extract_type != 'frames':
                    response['downloadUrl'] = f'/api/download/{os.path.basename(output_path)}'
                
                self.send_json_response(response)
            else:
                update_stats('errors', 1)
                self.send_error_response(500, f'Extraction failed: {result.stderr[:500]}')
                
        except Exception as e:
            update_stats('activeJobs', -1)
            update_stats('errors', 1)
            self.send_error_response(500, f'Extraction error: {str(e)}')
    
    def handle_info(self):
        """Get media file information"""
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length)
            data = json.loads(body)
            
            input_path = data.get('input')
            if not input_path or not os.path.exists(input_path):
                self.send_error_response(400, 'Input file not found')
                return
            
            # Use ffprobe to get media info
            result = subprocess.run(
                ['ffprobe', '-v', 'quiet', '-print_format', 'json', '-show_format', '-show_streams', input_path],
                capture_output=True, text=True, timeout=10
            )
            
            if result.returncode == 0:
                info = json.loads(result.stdout)
                
                # Extract key information
                summary = {
                    'format': info.get('format', {}).get('format_name', 'unknown'),
                    'duration': float(info.get('format', {}).get('duration', 0)),
                    'size': int(info.get('format', {}).get('size', 0)),
                    'bitrate': int(info.get('format', {}).get('bit_rate', 0)),
                    'streams': []
                }
                
                for stream in info.get('streams', []):
                    stream_info = {
                        'type': stream.get('codec_type'),
                        'codec': stream.get('codec_name'),
                        'profile': stream.get('profile', ''),
                    }
                    
                    if stream['codec_type'] == 'video':
                        stream_info.update({
                            'width': stream.get('width'),
                            'height': stream.get('height'),
                            'fps': eval(stream.get('r_frame_rate', '0/1')) if '/' in stream.get('r_frame_rate', '') else 0,
                            'bitrate': int(stream.get('bit_rate', 0))
                        })
                    elif stream['codec_type'] == 'audio':
                        stream_info.update({
                            'channels': stream.get('channels'),
                            'sample_rate': int(stream.get('sample_rate', 0)),
                            'bitrate': int(stream.get('bit_rate', 0))
                        })
                    
                    summary['streams'].append(stream_info)
                
                self.send_json_response({'success': True, 'info': summary, 'raw': info})
            else:
                self.send_error_response(500, 'Failed to get media info')
                
        except Exception as e:
            self.send_error_response(500, f'Info error: {str(e)}')
    
    def handle_stream(self):
        """Handle stream processing"""
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length)
            data = json.loads(body)
            
            url = data.get('url', '')
            action = data.get('action', 'info')
            duration = min(int(data.get('duration', 60)), 300)  # Max 5 minutes
            
            # Validate URL
            if not url or not (url.startswith('http://') or url.startswith('https://') or url.startswith('rtsp://')):
                self.send_error_response(400, 'Invalid stream URL')
                return
            
            update_stats('activeJobs', 1)
            start_time = time.time()
            
            if action == 'info':
                # Get stream info
                result = subprocess.run(
                    ['ffprobe', '-v', 'quiet', '-print_format', 'json', '-show_format', '-show_streams', url],
                    capture_output=True, text=True, timeout=15
                )
                
                update_stats('activeJobs', -1)
                
                if result.returncode == 0:
                    info = json.loads(result.stdout)
                    self.send_json_response({'success': True, 'info': info})
                else:
                    self.send_error_response(500, 'Failed to get stream info')
            
            elif action == 'capture':
                # Capture stream
                output_path = os.path.join(OUTPUT_DIR, f"stream_capture_{int(time.time())}.mp4")
                result = subprocess.run(
                    ['ffmpeg', '-i', url, '-t', str(duration), '-c', 'copy', '-y', output_path],
                    capture_output=True, text=True, timeout=duration + 30
                )
                
                update_stats('activeJobs', -1)
                process_time = time.time() - start_time
                
                if result.returncode == 0:
                    update_stats('completedJobs', 1)
                    update_stats('totalProcessingTime', int(process_time))
                    
                    self.send_json_response({
                        'success': True,
                        'output': os.path.basename(output_path),
                        'downloadUrl': f'/api/download/{os.path.basename(output_path)}',
                        'duration': round(process_time, 2)
                    })
                else:
                    update_stats('errors', 1)
                    self.send_error_response(500, 'Stream capture failed')
            
            elif action == 'transcode':
                # Live transcode stream
                output_path = os.path.join(OUTPUT_DIR, f"stream_transcode_{int(time.time())}.mp4")
                result = subprocess.run(
                    ['ffmpeg', '-i', url, '-t', str(duration), '-c:v', 'libx264', '-preset', 'veryfast', '-c:a', 'aac', '-y', output_path],
                    capture_output=True, text=True, timeout=duration + 60
                )
                
                update_stats('activeJobs', -1)
                process_time = time.time() - start_time
                
                if result.returncode == 0:
                    update_stats('completedJobs', 1)
                    update_stats('totalProcessingTime', int(process_time))
                    
                    self.send_json_response({
                        'success': True,
                        'output': os.path.basename(output_path),
                        'downloadUrl': f'/api/download/{os.path.basename(output_path)}',
                        'duration': round(process_time, 2)
                    })
                else:
                    update_stats('errors', 1)
                    self.send_error_response(500, 'Stream transcode failed')
            
            else:
                update_stats('activeJobs', -1)
                self.send_error_response(400, 'Invalid stream action')
                
        except Exception as e:
            update_stats('activeJobs', -1)
            update_stats('errors', 1)
            self.send_error_response(500, f'Stream error: {str(e)}')
    
    def handle_batch(self):
        """Handle batch processing request"""
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length)
            data = json.loads(body)
            
            jobs = data.get('jobs', [])
            if not jobs or len(jobs) > 10:  # Limit batch size
                self.send_error_response(400, 'Invalid batch request (1-10 jobs required)')
                return
            
            results = []
            
            for job in jobs:
                job_type = job.get('type', 'convert')
                job_result = {'id': job.get('id'), 'type': job_type}
                
                try:
                    if job_type == 'convert':
                        # Process conversion
                        preset = validate_preset(job.get('preset', 'mp4-h264')) or 'mp4-h264'
                        input_path = job.get('input')
                        
                        if input_path and os.path.exists(input_path):
                            output_ext = '.mp4'
                            output_path = os.path.join(OUTPUT_DIR, f"batch_{int(time.time())}_{job.get('id', 0)}{output_ext}")
                            
                            ffmpeg_cmd = ['ffmpeg', '-i', input_path, '-c:v', 'libx264', '-preset', 'fast', '-c:a', 'aac', '-y', output_path]
                            result = subprocess.run(ffmpeg_cmd, capture_output=True, text=True, timeout=120)
                            
                            if result.returncode == 0:
                                job_result['success'] = True
                                job_result['output'] = os.path.basename(output_path)
                            else:
                                job_result['success'] = False
                                job_result['error'] = 'Conversion failed'
                        else:
                            job_result['success'] = False
                            job_result['error'] = 'Input file not found'
                    
                    else:
                        job_result['success'] = False
                        job_result['error'] = 'Unknown job type'
                        
                except Exception as e:
                    job_result['success'] = False
                    job_result['error'] = str(e)
                
                results.append(job_result)
            
            self.send_json_response({'success': True, 'results': results})
            
        except Exception as e:
            self.send_error_response(500, f'Batch processing error: {str(e)}')
    
    def handle_download(self, path):
        """Handle file download request"""
        try:
            filename = os.path.basename(path.replace('/api/download/', ''))
            filepath = os.path.join(OUTPUT_DIR, filename)
            
            if not os.path.exists(filepath):
                self.send_error_response(404, 'File not found')
                return
            
            # Send file
            self.send_response(200)
            self.send_header('Content-Type', 'application/octet-stream')
            self.send_header('Content-Disposition', f'attachment; filename="{filename}"')
            self.send_header('Content-Length', str(os.path.getsize(filepath)))
            self.end_headers()
            
            with open(filepath, 'rb') as f:
                shutil.copyfileobj(f, self.wfile)
                
        except Exception as e:
            self.send_error_response(500, f'Download error: {str(e)}')
    
    def send_json_response(self, data):
        """Send JSON response"""
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())
    
    def send_error_response(self, code, message, details=None):
        """Send detailed error response"""
        self.send_response(code)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        
        error_response = {
            'success': False,
            'error': message,
            'code': code,
            'timestamp': int(time.time())
        }
        
        # Add specific error details based on code
        if code == 400:
            error_response['type'] = 'bad_request'
            error_response['hint'] = 'Check your request parameters and try again'
        elif code == 404:
            error_response['type'] = 'not_found'
            error_response['hint'] = 'The requested resource or endpoint does not exist'
        elif code == 413:
            error_response['type'] = 'payload_too_large'
            error_response['hint'] = f'File size must be less than {MAX_FILE_SIZE // (1024*1024)}MB'
        elif code == 415:
            error_response['type'] = 'unsupported_media_type'
            error_response['hint'] = 'Only video and audio files are supported'
        elif code == 422:
            error_response['type'] = 'unprocessable_entity'
            error_response['hint'] = 'The file format is not supported or corrupted'
        elif code == 500:
            error_response['type'] = 'internal_server_error'
            error_response['hint'] = 'An unexpected error occurred. Please try again later'
        elif code == 503:
            error_response['type'] = 'service_unavailable'
            error_response['hint'] = 'The service is temporarily unavailable. Please try again later'
        
        if details:
            error_response['details'] = details
            
        self.wfile.write(json.dumps(error_response).encode())
    
    def log_message(self, format, *args):
        """Override to reduce logging noise"""
        if '/api/' in args[0]:
            return  # Don't log API requests
        super().log_message(format, *args)

def main():
    """Main server entry point"""
    try:
        # Set socket reuse to prevent address in use errors
        socketserver.TCPServer.allow_reuse_address = True
        
        with socketserver.TCPServer(("", PORT), FFmpegAPIHandler) as httpd:
            print(f"[INFO] FFmpeg API server running on port {PORT}")
            print(f"[INFO] Upload dir: {UPLOAD_DIR}")
            print(f"[INFO] Output dir: {OUTPUT_DIR}")
            print(f"[INFO] Max file size: {MAX_FILE_SIZE / (1024*1024)}MB")
            httpd.serve_forever()
            
    except OSError as e:
        if "Address already in use" in str(e):
            print(f"[ERROR] Port {PORT} is already in use. Try a different port.")
            sys.exit(1)
        else:
            raise
    except KeyboardInterrupt:
        print("\n[INFO] Server shutting down...")
        sys.exit(0)

if __name__ == "__main__":
    main()