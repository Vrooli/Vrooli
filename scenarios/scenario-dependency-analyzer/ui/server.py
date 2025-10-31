#!/usr/bin/env python3
"""
Simple HTTP server for scenario-dependency-analyzer UI with health endpoint
"""
import json
import os
import http.server
import socketserver
from datetime import datetime
from pathlib import Path

PORT = int(os.environ.get('UI_PORT', 36897))
API_PORT = int(os.environ.get('API_PORT', 15533))


SERVE_ROOT = Path(__file__).resolve().parent
DIST_DIR = SERVE_ROOT / "dist"


class HealthCheckHandler(http.server.SimpleHTTPRequestHandler):
    """HTTP handler with health check endpoint"""

    def __init__(self, *args, **kwargs):
        directory = DIST_DIR if DIST_DIR.exists() else SERVE_ROOT
        super().__init__(*args, directory=str(directory), **kwargs)

    def do_GET(self):
        if self.path == '/health':
            self.send_health_response()
        else:
            candidate = Path(self.translate_path(self.path))
            if not candidate.exists() or candidate.is_dir():
                self.path = '/index.html'
            super().do_GET()

    def send_health_response(self):
        """Send health check response conforming to UI health schema"""
        # Check if API is accessible
        api_connected = False
        try:
            import urllib.request
            response = urllib.request.urlopen(f'http://localhost:{API_PORT}/health', timeout=2)
            api_connected = (response.status == 200)
        except:
            api_connected = False

        status = "healthy" if api_connected else "degraded"

        health_data = {
            "status": status,
            "service": "scenario-dependency-analyzer-ui",
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "readiness": True,  # UI is always ready once started
            "api_connectivity": {
                "connected": api_connected,
                "api_url": f"http://localhost:{API_PORT}",
                "error": None if api_connected else {
                    "code": "CONNECTION_REFUSED",
                    "message": "Unable to connect to API service",
                    "category": "network",
                    "retryable": True
                }
            }
        }

        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(health_data, indent=2).encode())

    def log_message(self, format, *args):
        """Suppress request logging except for errors"""
        if self.path != '/health':
            super().log_message(format, *args)


if __name__ == '__main__':
    os.chdir(str(DIST_DIR if DIST_DIR.exists() else SERVE_ROOT))

    # Allow socket reuse to prevent "Address already in use" errors
    socketserver.TCPServer.allow_reuse_address = True

    with socketserver.TCPServer(("", PORT), HealthCheckHandler) as httpd:
        print(f"🌐 UI Server running at http://localhost:{PORT}")
        print(f"📊 API endpoint: http://localhost:{API_PORT}")
        httpd.serve_forever()
