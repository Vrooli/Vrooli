#!/usr/bin/env python3
"""
Mock RCON server for testing mcrcon functionality
Implements a subset of the Minecraft RCON protocol
"""

import socket
import struct
import sys
import threading
import time
import json
import os

class MockRCONServer:
    def __init__(self, host='0.0.0.0', port=25575, password='test123'):
        self.host = host
        self.port = port
        self.password = password
        self.socket = None
        self.running = False
        self.request_id = 0
        
        # Mock data
        self.players = ['Steve', 'Alex', 'Notch']
        self.world_data = {
            'seed': '1234567890',
            'difficulty': 'normal',
            'gamemode': 'survival',
            'spawn': {'x': 0, 'y': 64, 'z': 0}
        }
        self.mods = ['WorldEdit', 'Essentials', 'Vault']
        
    def start(self):
        """Start the mock RCON server"""
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.socket.bind((self.host, self.port))
        self.socket.listen(5)
        self.running = True
        
        print(f"Mock RCON server listening on {self.host}:{self.port}")
        
        while self.running:
            try:
                client, addr = self.socket.accept()
                print(f"Connection from {addr}")
                thread = threading.Thread(target=self.handle_client, args=(client,))
                thread.daemon = True
                thread.start()
            except OSError:
                break
                
    def handle_client(self, client):
        """Handle a client connection"""
        authenticated = False
        
        try:
            while True:
                # Read packet length
                length_data = client.recv(4)
                if not length_data:
                    break
                    
                length = struct.unpack('<i', length_data)[0]
                
                # Read packet data
                data = client.recv(length)
                if not data:
                    break
                    
                request_id, packet_type = struct.unpack('<ii', data[:8])
                payload = data[8:-2].decode('utf-8', errors='ignore')
                
                # Handle packet types
                if packet_type == 3:  # Auth
                    if payload == self.password:
                        authenticated = True
                        response = struct.pack('<iii', 10, request_id, 2) + b'\x00\x00'
                    else:
                        response = struct.pack('<iii', 10, -1, 2) + b'\x00\x00'
                    client.send(response)
                    
                elif packet_type == 2 and authenticated:  # Command
                    response_text = self.handle_command(payload)
                    response_data = response_text.encode('utf-8') + b'\x00'
                    response_len = len(response_data) + 10
                    response = struct.pack('<iiii', response_len, request_id, 0, 0) + response_data + b'\x00'
                    client.send(response)
                    
        except Exception as e:
            print(f"Client error: {e}")
        finally:
            client.close()
            
    def handle_command(self, command):
        """Handle RCON commands and return responses"""
        parts = command.split()
        if not parts:
            return ""
            
        cmd = parts[0].lower()
        
        # Player commands
        if cmd == 'list':
            return f"There are {len(self.players)} players online: {', '.join(self.players)}"
        elif cmd == 'tp' and len(parts) >= 3:
            return f"Teleported {parts[1]} to {parts[2]}"
        elif cmd == 'kick' and len(parts) >= 2:
            return f"Kicked {parts[1]}"
        elif cmd == 'ban' and len(parts) >= 2:
            return f"Banned {parts[1]}"
        elif cmd == 'give' and len(parts) >= 3:
            return f"Gave {parts[2]} to {parts[1]}"
            
        # World commands
        elif cmd == 'save-all':
            return "Saving the game..."
        elif cmd == 'seed':
            return f"Seed: {self.world_data['seed']}"
        elif cmd == 'difficulty' and len(parts) >= 2:
            self.world_data['difficulty'] = parts[1]
            return f"Set difficulty to {parts[1]}"
        elif cmd == 'setworldspawn' and len(parts) >= 4:
            self.world_data['spawn'] = {'x': parts[1], 'y': parts[2], 'z': parts[3]}
            return f"Set world spawn to {parts[1]} {parts[2]} {parts[3]}"
            
        # Server commands
        elif cmd == 'say' and len(parts) >= 2:
            message = ' '.join(parts[1:])
            return f"[Server] {message}"
        elif cmd == 'help':
            return "Available commands: list, tp, kick, ban, give, save-all, seed, difficulty, setworldspawn, say"
            
        # Mod commands
        elif cmd == 'mods':
            return f"Installed mods: {', '.join(self.mods)}"
            
        else:
            return f"Unknown command: {cmd}"
            
    def stop(self):
        """Stop the server"""
        self.running = False
        if self.socket:
            self.socket.close()
            
if __name__ == '__main__':
    # Get configuration from environment or use defaults
    host = os.environ.get('MOCK_RCON_HOST', '127.0.0.1')
    port = int(os.environ.get('MOCK_RCON_PORT', '25575'))
    password = os.environ.get('MOCK_RCON_PASSWORD', 'test123')
    
    server = MockRCONServer(host, port, password)
    
    try:
        server.start()
    except KeyboardInterrupt:
        print("\nShutting down mock RCON server...")
        server.stop()