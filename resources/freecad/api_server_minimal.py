#!/usr/bin/env python3
"""
FreeCAD API Server - Minimal Implementation
Provides health check and basic structure for future FreeCAD integration
"""

import os
import json
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import uuid
from pathlib import Path

# Configuration
PORT = int(os.environ.get('PORT', 8080))
DATA_DIR = Path('/data')
PROJECTS_DIR = Path('/projects')
SCRIPTS_DIR = Path('/scripts')
EXPORTS_DIR = Path('/exports')

# Ensure directories exist
for directory in [DATA_DIR, PROJECTS_DIR, SCRIPTS_DIR, EXPORTS_DIR]:
    directory.mkdir(parents=True, exist_ok=True)

class FreeCADHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        """Handle GET requests"""
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            health_status = {
                'status': 'healthy',
                'service': 'freecad',
                'version': '0.1.0',
                'message': 'FreeCAD API server is running (minimal implementation)'
            }
            self.wfile.write(json.dumps(health_status).encode())
            
        elif parsed_path.path == '/status':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            status = {
                'service': 'freecad',
                'operational': True,
                'features': {
                    'health_check': True,
                    'python_api': False,  # Not yet implemented
                    'step_export': False,  # Not yet implemented
                    'fem_analysis': False  # Not yet implemented
                },
                'directories': {
                    'data': str(DATA_DIR),
                    'projects': str(PROJECTS_DIR),
                    'scripts': str(SCRIPTS_DIR),
                    'exports': str(EXPORTS_DIR)
                }
            }
            self.wfile.write(json.dumps(status, indent=2).encode())
            
        elif parsed_path.path.startswith('/list/'):
            # List directory contents
            dir_name = parsed_path.path.split('/')[-1]
            dir_map = {
                'scripts': SCRIPTS_DIR,
                'projects': PROJECTS_DIR,
                'exports': EXPORTS_DIR
            }
            
            if dir_name in dir_map:
                target_dir = dir_map[dir_name]
                files = [f.name for f in target_dir.glob('*') if f.is_file()]
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps({dir_name: files}).encode())
            else:
                self.send_error(404, 'Directory not found')
                
        else:
            self.send_error(404, 'Path not found')
    
    def do_POST(self):
        """Handle POST requests"""
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/generate':
            # Placeholder for CAD generation
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            try:
                data = json.loads(post_data)
                script_content = data.get('script', '')
                
                # For now, just save the script
                script_id = str(uuid.uuid4())
                script_path = SCRIPTS_DIR / f'{script_id}.py'
                script_path.write_text(script_content)
                
                response = {
                    'success': True,
                    'message': 'Script saved (FreeCAD execution not yet implemented)',
                    'id': script_id,
                    'path': str(script_path)
                }
                
                self.send_response(200)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(json.dumps(response).encode())
                
            except Exception as e:
                self.send_error(500, str(e))
                
        elif parsed_path.path == '/export':
            # Placeholder for export functionality
            self.send_response(501)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'error': 'Export functionality not yet implemented',
                'message': 'This is a placeholder for future FreeCAD integration'
            }
            self.wfile.write(json.dumps(response).encode())
            
        elif parsed_path.path == '/analyze':
            # Placeholder for analysis functionality
            self.send_response(501)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'error': 'Analysis functionality not yet implemented',
                'message': 'This is a placeholder for future FEM analysis'
            }
            self.wfile.write(json.dumps(response).encode())
            
        else:
            self.send_error(404, 'Path not found')
    
    def log_message(self, format, *args):
        """Suppress default logging"""
        pass

def run_server():
    """Run the HTTP server"""
    server_address = ('0.0.0.0', PORT)
    httpd = HTTPServer(server_address, FreeCADHandler)
    print(f"FreeCAD API server running on port {PORT}")
    print("Health check available at /health")
    print("Status available at /status")
    httpd.serve_forever()

if __name__ == '__main__':
    run_server()