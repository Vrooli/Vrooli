#!/usr/bin/env python3
import os
import json
from http.server import HTTPServer, SimpleHTTPRequestHandler
from urllib.parse import urlparse
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class UIHandler(SimpleHTTPRequestHandler):
    def do_GET(self):
        """Handle GET requests"""
        parsed_path = urlparse(self.path)
        
        # Health check endpoint
        if parsed_path.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = json.dumps({
                'status': 'healthy',
                'port': PORT
            })
            self.wfile.write(response.encode())
            return
        
        # API configuration endpoint
        if parsed_path.path == '/api/config':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = json.dumps({
                'apiUrl': f'http://localhost:{API_PORT}',
                'uiPort': PORT
            })
            self.wfile.write(response.encode())
            return
        
        # Serve static files
        if parsed_path.path == '/':
            # Check for dashboard.html first, then index.html
            if os.path.exists('dashboard.html'):
                self.path = '/dashboard.html'
            elif os.path.exists('index.html'):
                self.path = '/index.html'
        
        return SimpleHTTPRequestHandler.do_GET(self)
    
    def log_message(self, format, *args):
        """Override to use logger"""
        logger.info("%s - %s", self.address_string(), format % args)

if __name__ == '__main__':
    PORT = int(os.environ.get('UI_PORT', os.environ.get('PORT', 3000)))
    API_PORT = int(os.environ.get('API_PORT', 8080))
    
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    
    server = HTTPServer(('', PORT), UIHandler)
    logger.info(f'UI server running on http://localhost:{PORT}')
    logger.info(f'API expected at http://localhost:{API_PORT}')
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        logger.info('Server stopped')
