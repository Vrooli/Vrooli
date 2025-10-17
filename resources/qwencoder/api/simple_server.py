#!/usr/bin/env python3
"""Simple QwenCoder API Server for testing without dependencies"""

import http.server
import json
import argparse
from urllib.parse import urlparse, parse_qs

class QwenCoderHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'status': 'healthy',
                'model_loaded': False,
                'model_name': 'qwencoder-1.5b'
            }
            self.wfile.write(json.dumps(response).encode())
        
        elif parsed_path.path == '/models':
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            response = {
                'models': [
                    {'id': 'qwencoder-0.5b', 'object': 'model'},
                    {'id': 'qwencoder-1.5b', 'object': 'model'},
                    {'id': 'qwencoder-7b', 'object': 'model'}
                ]
            }
            self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/v1/completions':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            
            self.send_response(200)
            self.send_header('Content-Type', 'application/json')
            self.end_headers()
            
            response = {
                'id': 'cmpl-test',
                'object': 'text_completion',
                'model': 'qwencoder-1.5b',
                'choices': [{
                    'text': '\n    # Generated code\n    pass',
                    'index': 0,
                    'finish_reason': 'stop'
                }],
                'usage': {
                    'prompt_tokens': 10,
                    'completion_tokens': 5,
                    'total_tokens': 15
                }
            }
            self.wfile.write(json.dumps(response).encode())
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def log_message(self, format, *args):
        # Suppress default logging
        pass

def main():
    parser = argparse.ArgumentParser(description='Simple QwenCoder Server')
    parser.add_argument('--port', type=int, default=11452)
    parser.add_argument('--model', type=str, default='qwencoder-1.5b')
    parser.add_argument('--device', type=str, default='cpu')
    
    args = parser.parse_args()
    
    server_address = ('', args.port)
    httpd = http.server.HTTPServer(server_address, QwenCoderHandler)
    
    print(f'Starting QwenCoder simple server on port {args.port}')
    httpd.serve_forever()

if __name__ == '__main__':
    main()