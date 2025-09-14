#!/usr/bin/env python3
"""
MCRcon Python Library
Provides programmatic access to Minecraft servers via RCON protocol
"""

import os
import json
import subprocess
import time
import threading
from typing import Optional, List, Dict, Any, Callable
from pathlib import Path


class MCRconClient:
    """Minecraft RCON client for Python"""
    
    def __init__(self, host: str = "localhost", port: int = 25575, password: str = None):
        """
        Initialize MCRcon client
        
        Args:
            host: Server hostname or IP
            port: RCON port (default 25575)
            password: RCON password (can also be set via MCRCON_PASSWORD env var)
        """
        self.host = host
        self.port = port
        self.password = password or os.environ.get('MCRCON_PASSWORD', '')
        self.data_dir = Path.home() / '.mcrcon'
        self.binary_path = self.data_dir / 'bin' / 'mcrcon'
        self.timeout = 30
        self.retry_attempts = 3
        
        if not self.password:
            raise ValueError("RCON password required (set via parameter or MCRCON_PASSWORD env var)")
        
        if not self.binary_path.exists():
            raise RuntimeError(f"mcrcon binary not found at {self.binary_path}. Run 'vrooli resource mcrcon manage install' first.")
    
    def execute(self, command: str) -> str:
        """
        Execute a single RCON command
        
        Args:
            command: Minecraft command to execute
            
        Returns:
            Command output as string
        """
        cmd = [
            str(self.binary_path),
            '-H', self.host,
            '-P', str(self.port),
            '-p', self.password,
            command
        ]
        
        for attempt in range(self.retry_attempts):
            try:
                result = subprocess.run(
                    cmd,
                    capture_output=True,
                    text=True,
                    timeout=self.timeout
                )
                
                if result.returncode == 0:
                    return result.stdout.strip()
                else:
                    if attempt < self.retry_attempts - 1:
                        time.sleep(2 ** attempt)  # Exponential backoff
                        continue
                    raise RuntimeError(f"Command failed: {result.stderr}")
                    
            except subprocess.TimeoutExpired:
                if attempt < self.retry_attempts - 1:
                    time.sleep(2 ** attempt)
                    continue
                raise TimeoutError(f"Command timed out after {self.timeout} seconds")
        
        raise RuntimeError(f"Command failed after {self.retry_attempts} attempts")
    
    def list_players(self) -> List[str]:
        """Get list of online players"""
        output = self.execute("list")
        # Parse player list from output
        if "There are" in output:
            lines = output.split('\n')
            for line in lines:
                if ":" in line:
                    players_str = line.split(":", 1)[1].strip()
                    if players_str:
                        return [p.strip() for p in players_str.split(",")]
        return []
    
    def save_world(self) -> bool:
        """Save the world"""
        try:
            self.execute("save-all")
            return True
        except Exception as e:
            print(f"Failed to save world: {e}")
            return False
    
    def set_difficulty(self, difficulty: str) -> bool:
        """
        Set world difficulty
        
        Args:
            difficulty: peaceful, easy, normal, or hard
        """
        if difficulty not in ['peaceful', 'easy', 'normal', 'hard']:
            raise ValueError("Difficulty must be: peaceful, easy, normal, or hard")
        
        try:
            self.execute(f"difficulty {difficulty}")
            return True
        except Exception:
            return False
    
    def teleport_player(self, player: str, x: float, y: float, z: float) -> bool:
        """Teleport a player to coordinates"""
        try:
            self.execute(f"tp {player} {x} {y} {z}")
            return True
        except Exception:
            return False
    
    def give_item(self, player: str, item: str, count: int = 1) -> bool:
        """Give items to a player"""
        try:
            self.execute(f"give {player} {item} {count}")
            return True
        except Exception:
            return False
    
    def kick_player(self, player: str, reason: str = "Kicked by administrator") -> bool:
        """Kick a player from the server"""
        try:
            self.execute(f"kick {player} {reason}")
            return True
        except Exception:
            return False
    
    def ban_player(self, player: str, reason: str = "Banned by administrator") -> bool:
        """Ban a player from the server"""
        try:
            self.execute(f"ban {player} {reason}")
            return True
        except Exception:
            return False
    
    def set_weather(self, weather: str, duration: Optional[int] = None) -> bool:
        """
        Set weather
        
        Args:
            weather: clear, rain, or thunder
            duration: Duration in seconds (optional)
        """
        if weather not in ['clear', 'rain', 'thunder']:
            raise ValueError("Weather must be: clear, rain, or thunder")
        
        try:
            cmd = f"weather {weather}"
            if duration:
                cmd += f" {duration}"
            self.execute(cmd)
            return True
        except Exception:
            return False
    
    def set_time(self, time_value: str) -> bool:
        """
        Set world time
        
        Args:
            time_value: day, night, noon, midnight, or numeric value
        """
        try:
            self.execute(f"time set {time_value}")
            return True
        except Exception:
            return False
    
    def say(self, message: str) -> bool:
        """Broadcast a message to all players"""
        try:
            self.execute(f"say {message}")
            return True
        except Exception:
            return False


class MCRconEventMonitor:
    """Monitor Minecraft server events"""
    
    def __init__(self, client: MCRconClient):
        """
        Initialize event monitor
        
        Args:
            client: MCRconClient instance
        """
        self.client = client
        self.running = False
        self.thread = None
        self.callbacks = {
            'join': [],
            'leave': [],
            'chat': [],
            'death': [],
            'achievement': []
        }
        self._player_count = 0
    
    def on(self, event: str, callback: Callable):
        """
        Register event callback
        
        Args:
            event: Event type (join, leave, chat, death, achievement)
            callback: Function to call when event occurs
        """
        if event in self.callbacks:
            self.callbacks[event].append(callback)
    
    def start(self):
        """Start monitoring events"""
        if self.running:
            return
        
        self.running = True
        self.thread = threading.Thread(target=self._monitor_loop)
        self.thread.daemon = True
        self.thread.start()
    
    def stop(self):
        """Stop monitoring events"""
        self.running = False
        if self.thread:
            self.thread.join(timeout=5)
    
    def _monitor_loop(self):
        """Internal monitoring loop"""
        while self.running:
            try:
                # Monitor player count changes
                players = self.client.list_players()
                current_count = len(players)
                
                if current_count > self._player_count:
                    # Player joined
                    for callback in self.callbacks['join']:
                        callback({'type': 'join', 'player_count': current_count, 'players': players})
                elif current_count < self._player_count:
                    # Player left
                    for callback in self.callbacks['leave']:
                        callback({'type': 'leave', 'player_count': current_count, 'players': players})
                
                self._player_count = current_count
                
            except Exception as e:
                print(f"Monitor error: {e}")
            
            time.sleep(2)


class MCRconServer:
    """Manage multiple Minecraft servers"""
    
    def __init__(self, config_file: Optional[str] = None):
        """
        Initialize server manager
        
        Args:
            config_file: Path to server configuration JSON
        """
        self.config_file = config_file or str(Path.home() / '.mcrcon' / 'servers.json')
        self.servers = self._load_servers()
    
    def _load_servers(self) -> Dict[str, MCRconClient]:
        """Load server configurations"""
        servers = {}
        
        if os.path.exists(self.config_file):
            with open(self.config_file, 'r') as f:
                config = json.load(f)
                for server in config.get('servers', []):
                    name = server['name']
                    servers[name] = MCRconClient(
                        host=server['host'],
                        port=server['port'],
                        password=server['password']
                    )
        
        return servers
    
    def add_server(self, name: str, host: str, port: int, password: str):
        """Add a server configuration"""
        self.servers[name] = MCRconClient(host, port, password)
        self._save_servers()
    
    def remove_server(self, name: str):
        """Remove a server configuration"""
        if name in self.servers:
            del self.servers[name]
            self._save_servers()
    
    def get_server(self, name: str) -> Optional[MCRconClient]:
        """Get a server client by name"""
        return self.servers.get(name)
    
    def list_servers(self) -> List[str]:
        """List configured server names"""
        return list(self.servers.keys())
    
    def execute_all(self, command: str) -> Dict[str, str]:
        """
        Execute command on all servers
        
        Returns:
            Dictionary of server_name -> output
        """
        results = {}
        for name, client in self.servers.items():
            try:
                results[name] = client.execute(command)
            except Exception as e:
                results[name] = f"Error: {e}"
        return results
    
    def _save_servers(self):
        """Save server configurations"""
        config = {
            'servers': [
                {
                    'name': name,
                    'host': client.host,
                    'port': client.port,
                    'password': client.password
                }
                for name, client in self.servers.items()
            ]
        }
        
        os.makedirs(os.path.dirname(self.config_file), exist_ok=True)
        with open(self.config_file, 'w') as f:
            json.dump(config, f, indent=2)


# Convenience functions for quick usage
def quick_execute(command: str, host: str = "localhost", port: int = 25575, password: str = None) -> str:
    """
    Quick command execution without creating client instance
    
    Args:
        command: Minecraft command
        host: Server host
        port: RCON port
        password: RCON password
        
    Returns:
        Command output
    """
    client = MCRconClient(host, port, password)
    return client.execute(command)


def discover_servers() -> List[Dict[str, Any]]:
    """
    Discover Minecraft servers on local network
    
    Returns:
        List of discovered servers with connection details
    """
    servers = []
    
    # Check for Docker containers
    try:
        result = subprocess.run(
            ['docker', 'ps', '--format', '{{json .}}'],
            capture_output=True,
            text=True
        )
        
        for line in result.stdout.strip().split('\n'):
            if line:
                container = json.loads(line)
                if 'minecraft' in container.get('Image', '').lower():
                    servers.append({
                        'type': 'docker',
                        'name': container.get('Names', ''),
                        'image': container.get('Image', ''),
                        'ports': container.get('Ports', '')
                    })
    except Exception:
        pass
    
    # Check for local processes
    try:
        result = subprocess.run(
            ['pgrep', '-a', 'java'],
            capture_output=True,
            text=True
        )
        
        for line in result.stdout.strip().split('\n'):
            if 'minecraft' in line.lower():
                servers.append({
                    'type': 'process',
                    'command': line
                })
    except Exception:
        pass
    
    return servers


if __name__ == "__main__":
    # Example usage
    print("MCRcon Python Library")
    print("=====================")
    print()
    print("Example usage:")
    print("  from mcrcon import MCRconClient")
    print("  client = MCRconClient('localhost', 25575, 'password')")
    print("  players = client.list_players()")
    print("  client.say('Hello from Python!')")
    print()
    print("For more examples, see the documentation.")