#!/usr/bin/env python3
"""
GGWave Mock API Server
Provides REST endpoints for audio-based data transmission simulation
"""
import http.server
import socketserver
import json
import base64
import sys
from urllib.parse import urlparse, parse_qs

class GGWaveHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        """Handle GET requests"""
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                "status": "healthy",
                "version": "0.4.0",
                "modes_available": ["normal", "fast", "dt", "ultrasonic"]
            }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_error(404, "Not found")
    
    def do_POST(self):
        """Handle POST requests"""
        try:
            content_length = int(self.headers.get('Content-Length', 0))
            if content_length > 0:
                post_data = self.rfile.read(content_length)
                data = json.loads(post_data.decode('utf-8'))
            else:
                data = {}
            
            if self.path == '/api/encode':
                input_text = data.get('data', '')
                mode = data.get('mode', 'normal')
                format_type = data.get('format', 'base64')
                
                # Simulate FSK encoding
                encoded_data = base64.b64encode(input_text.encode()).decode()
                
                response = {
                    "audio": encoded_data,
                    "duration_ms": len(input_text) * 100,
                    "mode": mode,
                    "bytes": len(input_text)
                }
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
                
            elif self.path == '/api/decode':
                audio_data = data.get('audio', '')
                mode = data.get('mode', 'auto')
                
                # Simulate FSK decoding
                try:
                    decoded_text = base64.b64decode(audio_data).decode('utf-8')
                except Exception:
                    decoded_text = "Error decoding"
                
                response = {
                    "data": decoded_text,
                    "confidence": 0.95,
                    "mode_detected": mode if mode != 'auto' else 'normal',
                    "errors_corrected": 0
                }
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
                
            else:
                self.send_error(404, "Endpoint not found")
                
        except json.JSONDecodeError as e:
            self.send_error(400, f"Invalid JSON: {str(e)}")
        except Exception as e:
            self.send_error(500, f"Server error: {str(e)}")
    
    def log_message(self, format, *args):
        """Log to stdout for Docker logs"""
        sys.stdout.write("%s - - [%s] %s\n" %
                         (self.address_string(),
                          self.log_date_time_string(),
                          format % args))
        sys.stdout.flush()

def run_server(port=8196):
    """Run the HTTP server"""
    try:
        with socketserver.TCPServer(("", port), GGWaveHandler) as httpd:
            httpd.allow_reuse_address = True
            print(f"GGWave API server running on port {port}", flush=True)
            httpd.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down server...", flush=True)
        sys.exit(0)
    except Exception as e:
        print(f"Error starting server: {e}", flush=True)
        sys.exit(1)

if __name__ == '__main__':
    run_server()