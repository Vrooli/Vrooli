import http.server
import json
from datetime import datetime

class MockMesaHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {"status": "healthy", "timestamp": datetime.utcnow().isoformat()}
            self.wfile.write(json.dumps(response).encode())
        elif self.path == '/models':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = {"models": ["schelling", "virus", "forest_fire"]}
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        return  # Suppress logs

if __name__ == '__main__':
    import sys
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 9512
    server = http.server.HTTPServer(('0.0.0.0', port), MockMesaHandler)
    server.serve_forever()
