#!/usr/bin/env python3
"""
Safe Vrooli App Orchestrator - Fork Bomb Prevention Edition
Implements multiple safety layers to prevent recursive process spawning
"""

import asyncio
import json
import logging
import os
import signal
import socket
import subprocess
import sys
import time
import fcntl
import shutil
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Set
import psutil

# Constants for safety
MAX_APPS = 100
MAX_CONCURRENT_STARTS = 3  # Maximum apps starting simultaneously
MAX_PROCESSES_PER_APP = 5  # Max processes an app can spawn
ORCHESTRATOR_LOCK_FILE = "/tmp/vrooli-orchestrator.lock"
ORCHESTRATOR_PID_FILE = "/tmp/vrooli-orchestrator.pid"
APP_PID_DIR = "/tmp/vrooli-apps"
FORK_BOMB_THRESHOLD = 30  # Reduced threshold for safety
FORK_BOMB_WINDOW = 10  # Time window in seconds
# Adjusted for 32-core server with ~730 normal processes (415 kernel + 315 user)
SYSTEM_PROCESS_LIMIT = 2000  # Increased - your system normally runs 730+ processes
SYSTEM_WARNING_LIMIT = 1500  # Warn but don't stop

@dataclass
class App:
    """Represents a Vrooli app to be started"""
    name: str
    enabled: bool
    allocated_ports: Dict[str, int]
    process: Optional[asyncio.subprocess.Process] = None
    pid: Optional[int] = None
    log_file: Optional[Path] = None
    status: str = "pending"


class ForkBombDetector:
    """Detects and prevents fork bomb scenarios by monitoring TOTAL system processes"""
    
    def __init__(self, threshold: int = FORK_BOMB_THRESHOLD, window: int = FORK_BOMB_WINDOW):
        self.threshold = threshold
        self.window = window
        self.process_starts = []
        self.start_time = time.time()
        self.initial_process_count = len(psutil.pids())
        self.last_process_count = self.initial_process_count
    
    def record_start(self):
        """Record a process start event and check TOTAL system processes"""
        current_time = time.time()
        current_process_count = len(psutil.pids())
        
        # CRITICAL: Check total system process count
        if current_process_count > SYSTEM_PROCESS_LIMIT:
            logging.error(f"SYSTEM OVERLOAD: {current_process_count} processes (limit: {SYSTEM_PROCESS_LIMIT})")
            return False
        
        # Check process growth rate
        process_growth = current_process_count - self.last_process_count
        if process_growth > 10:  # More than 10 new processes since last check
            logging.warning(f"Rapid process growth detected: {process_growth} new processes")
            time.sleep(2)  # Slow down
        
        self.last_process_count = current_process_count
        
        # Remove old entries outside the window
        self.process_starts = [t for t in self.process_starts if current_time - t < self.window]
        self.process_starts.append(current_time)
        
        # Check if we've exceeded threshold
        if len(self.process_starts) > self.threshold:
            return False  # Fork bomb detected!
        return True  # Safe to continue
    
    def get_rate(self) -> float:
        """Get current process spawn rate per second"""
        current_time = time.time()
        recent_starts = [t for t in self.process_starts if current_time - t < self.window]
        if not recent_starts:
            return 0.0
        time_span = current_time - min(recent_starts)
        if time_span == 0:
            return float(len(recent_starts))
        return len(recent_starts) / time_span
    
    def get_system_load(self) -> dict:
        """Get current system load information"""
        return {
            'process_count': len(psutil.pids()),
            'process_growth': len(psutil.pids()) - self.initial_process_count,
            'cpu_percent': psutil.cpu_percent(interval=0.1),
            'memory_percent': psutil.virtual_memory().percent
        }


class SafeAppOrchestrator:
    """Safe orchestrator with comprehensive fork bomb prevention"""
    
    def __init__(self, vrooli_root: Path, verbose: bool = False, fast_mode: bool = True):
        self.vrooli_root = vrooli_root
        self.verbose = verbose
        self.fast_mode = fast_mode
        self.generated_apps_dir = Path.home() / "generated-apps"
        self.apps: Dict[str, App] = {}
        self.log_dir = Path("/tmp/vrooli-orchestrator")
        self.log_dir.mkdir(exist_ok=True)
        
        # Safety mechanisms
        self.fork_bomb_detector = ForkBombDetector()
        self.lock_file = None
        self.running_processes: List[asyncio.subprocess.Process] = []
        self.allocated_ports: Set[int] = set()
        self.port_allocations: Dict[tuple, int] = {}  # (app_name, port_type) -> port
        self.resource_ports: Set[int] = set()  # Reserved ports from port_registry.sh
        
        # Setup logging first (before any logger calls)
        log_level = logging.DEBUG if verbose else logging.INFO
        logging.basicConfig(
            level=log_level,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(self.log_dir / "orchestrator-safe.log"),
                logging.StreamHandler()
            ]
        )
        self.logger = logging.getLogger(__name__)
        
        # Load reserved resource ports (after logger is initialized)
        self._load_resource_ports()
        
        # Ensure PID directory exists
        Path(APP_PID_DIR).mkdir(exist_ok=True)
        
        # Setup signal handlers
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully"""
        self.logger.info(f"Received signal {signum}, shutting down gracefully...")
        asyncio.create_task(self.shutdown())
    
    def acquire_lock(self) -> bool:
        """Acquire exclusive lock to prevent multiple orchestrators"""
        try:
            # Try to open and lock the file
            self.lock_file = open(ORCHESTRATOR_LOCK_FILE, 'w')
            fcntl.flock(self.lock_file.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
            
            # Write our PID to the lock file
            self.lock_file.write(str(os.getpid()))
            self.lock_file.flush()
            
            # Also write to separate PID file for redundancy
            with open(ORCHESTRATOR_PID_FILE, 'w') as f:
                f.write(str(os.getpid()))
            
            self.logger.info("Acquired orchestrator lock")
            return True
            
        except (IOError, OSError) as e:
            # Lock is held by another process
            if self.lock_file:
                self.lock_file.close()
                self.lock_file = None
            
            # Check if the lock holder is still alive
            try:
                with open(ORCHESTRATOR_PID_FILE, 'r') as f:
                    other_pid = int(f.read().strip())
                
                # Check if that process is actually running
                if psutil.pid_exists(other_pid):
                    self.logger.error(f"Another orchestrator is already running (PID: {other_pid})")
                    return False
                else:
                    # Stale lock, remove it
                    self.logger.warning("Removing stale orchestrator lock")
                    try:
                        os.remove(ORCHESTRATOR_LOCK_FILE)
                        os.remove(ORCHESTRATOR_PID_FILE)
                    except:
                        pass
                    # Try again
                    return self.acquire_lock()
                    
            except (FileNotFoundError, ValueError):
                self.logger.error("Cannot acquire lock, another orchestrator may be running")
                return False
    
    def release_lock(self):
        """Release the orchestrator lock"""
        if self.lock_file:
            try:
                # Release the flock
                fcntl.flock(self.lock_file.fileno(), fcntl.LOCK_UN)
                self.lock_file.close()
                self.lock_file = None
            except:
                pass
        
        # Clean up lock files
        try:
            os.remove(ORCHESTRATOR_LOCK_FILE)
        except:
            pass
        
        try:
            os.remove(ORCHESTRATOR_PID_FILE)
        except:
            pass
    
    def is_app_running(self, app_name: str) -> Optional[int]:
        """Check if an app is already running by checking its PID file"""
        pid_file = Path(APP_PID_DIR) / f"{app_name}.pid"
        
        if not pid_file.exists():
            return None
        
        try:
            with open(pid_file, 'r') as f:
                pid = int(f.read().strip())
            
            # Check if process is actually running
            if psutil.pid_exists(pid):
                try:
                    proc = psutil.Process(pid)
                    # Verify it's actually our app (check command line)
                    cmdline = ' '.join(proc.cmdline())
                    if app_name in cmdline or 'manage.sh' in cmdline:
                        return pid
                except (psutil.NoSuchProcess, psutil.AccessDenied):
                    pass
            
            # PID file exists but process is dead - clean up
            pid_file.unlink()
            return None
            
        except (ValueError, FileNotFoundError):
            return None
    
    def write_app_pid(self, app_name: str, pid: int):
        """Write app PID to file for tracking"""
        pid_file = Path(APP_PID_DIR) / f"{app_name}.pid"
        with open(pid_file, 'w') as f:
            f.write(str(pid))
    
    def remove_app_pid(self, app_name: str):
        """Remove app PID file"""
        pid_file = Path(APP_PID_DIR) / f"{app_name}.pid"
        try:
            pid_file.unlink()
        except FileNotFoundError:
            pass
    
    def is_port_available(self, port: int) -> bool:
        """Check if a port is available"""
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                s.bind(('localhost', port))
                return True
        except OSError:
            return False
    
    def find_available_port(self, start: int = 3001, end: int = 5000) -> Optional[int]:
        """Find an available port in range, avoiding resource ports"""
        for port in range(start, end):
            if (port not in self.allocated_ports and 
                port not in self.resource_ports and 
                self.is_port_available(port)):
                self.allocated_ports.add(port)
                return port
        return None
    
    def _load_resource_ports(self):
        """Load reserved ports from port_registry.sh"""
        # Common resource ports that should be avoided
        # Based on scripts/resources/port_registry.sh
        reserved = {
            8092,   # vrooli-api
            11434,  # ollama
            11435,  # litellm
            8090,   # whisper (conflicts with some apps!)
            5678,   # n8n
            8188,   # comfyui
            1880,   # node-red
            5681,   # windmill
            9000,   # minio
            8200,   # vault (conflicts with picker-wheel!)
            6333,   # qdrant
            9009,   # questdb
            5433,   # postgres
            6380,   # redis
            7474,   # neo4j
            7687,   # neo4j-bolt
            4110,   # browserless
            4113,   # agent-s2
            8280,   # searxng
            2358,   # judge0
            8070,   # keycloak
            8020,   # erpnext
        }
        self.resource_ports = reserved
        self.logger.debug(f"Loaded {len(self.resource_ports)} reserved resource ports")
    
    def _load_all_app_configs(self) -> Dict[str, dict]:
        """Load all app service.json configs for port planning"""
        app_configs = {}
        
        if not self.generated_apps_dir.exists():
            return app_configs
        
        for app_dir in self.generated_apps_dir.iterdir():
            if not app_dir.is_dir():
                continue
                
            service_json = app_dir / ".vrooli" / "service.json"
            if service_json.exists():
                try:
                    with open(service_json) as f:
                        config = json.load(f)
                        app_configs[app_dir.name] = config
                except Exception as e:
                    self.logger.warning(f"Could not load config for {app_dir.name}: {e}")
        
        return app_configs
    
    def _pre_allocate_all_ports(self, enabled_apps: List[str]) -> Dict[tuple, int]:
        """
        Pre-allocate all ports using smart strategy:
        1. Fixed ports first (must have exact port)
        2. Smallest ranges next (less flexibility)  
        3. Largest ranges last (most flexibility)
        
        Returns dict of (app_name, port_type) -> allocated_port
        """
        allocations = {}
        all_configs = self._load_all_app_configs()
        
        # Step 1: Collect all port requirements
        fixed_requirements = []  # [(app, port_type, port)]
        range_requirements = []  # [(app, port_type, start, end, size)]
        
        for app_name in enabled_apps:
            if app_name not in all_configs:
                continue
                
            ports_config = all_configs[app_name].get("ports", {})
            
            for port_type, port_info in ports_config.items():
                if isinstance(port_info, dict):
                    # Check for fixed port
                    if "fixed" in port_info:
                        fixed_port = int(port_info["fixed"])
                        fixed_requirements.append((app_name, port_type, fixed_port))
                    # Check for range
                    elif "range" in port_info:
                        range_str = port_info["range"]
                        if "-" in range_str:
                            start, end = map(int, range_str.split("-"))
                            size = end - start + 1
                            range_requirements.append((app_name, port_type, start, end, size))
                        else:
                            self.logger.warning(f"Invalid range format for {app_name}:{port_type}: {range_str}")
        
        # Step 2: Sort ranges by size (smallest first)
        range_requirements.sort(key=lambda x: x[4])
        
        # Step 3: Allocate fixed ports first
        for app_name, port_type, fixed_port in fixed_requirements:
            if fixed_port in self.resource_ports:
                self.logger.error(f"Fixed port {fixed_port} for {app_name}:{port_type} conflicts with resource port!")
                # Try to find alternative in nearby range
                alt_port = self.find_available_port(fixed_port + 1, fixed_port + 100)
                if alt_port:
                    allocations[(app_name, port_type)] = alt_port
                    self.logger.warning(f"Using alternative port {alt_port} for {app_name}:{port_type}")
            elif fixed_port not in self.allocated_ports and self.is_port_available(fixed_port):
                allocations[(app_name, port_type)] = fixed_port
                self.allocated_ports.add(fixed_port)
                self.logger.debug(f"Allocated fixed port {fixed_port} for {app_name}:{port_type}")
            else:
                self.logger.error(f"Fixed port {fixed_port} unavailable for {app_name}:{port_type}")
        
        # Step 4: Allocate ranges (smallest to largest for optimal packing)
        for app_name, port_type, start, end, size in range_requirements:
            allocated = False
            
            # Try to allocate within requested range
            for port in range(start, end + 1):
                if (port not in self.allocated_ports and 
                    port not in self.resource_ports and 
                    self.is_port_available(port)):
                    allocations[(app_name, port_type)] = port
                    self.allocated_ports.add(port)
                    allocated = True
                    self.logger.debug(f"Allocated port {port} (from range {start}-{end}) for {app_name}:{port_type}")
                    break
            
            if not allocated:
                # Fallback: Find any available port
                fallback_port = self.find_available_port(3001, 9999)
                if fallback_port:
                    allocations[(app_name, port_type)] = fallback_port
                    self.logger.warning(f"Range {start}-{end} exhausted for {app_name}:{port_type}, using fallback port {fallback_port}")
                else:
                    self.logger.error(f"Could not allocate any port for {app_name}:{port_type}")
        
        # Step 5: Handle apps without port configs
        for app_name in enabled_apps:
            if app_name not in all_configs:
                # App without config - allocate default ports
                api_port = self.find_available_port(3001, 9999)
                if api_port:
                    allocations[(app_name, "api")] = api_port
                    ui_port = self.find_available_port(api_port + 1, 9999)
                    if ui_port:
                        allocations[(app_name, "ui")] = ui_port
        
        self.logger.info(f"Pre-allocated {len(allocations)} ports for {len(enabled_apps)} apps")
        return allocations
    
    def load_enabled_apps(self) -> List[App]:
        """Load enabled apps from catalog with smart port pre-allocation"""
        catalog_path = self.vrooli_root / "scripts" / "scenarios" / "catalog.json"
        
        if not catalog_path.exists():
            self.logger.error(f"Catalog not found: {catalog_path}")
            return []
        
        try:
            with open(catalog_path) as f:
                catalog = json.load(f)
            
            # Step 1: Collect all enabled apps
            enabled_apps = []
            app_paths = {}
            
            for scenario in catalog.get("scenarios", []):
                if not scenario.get("enabled", False):
                    continue
                
                app_name = scenario["name"]
                app_path = self.generated_apps_dir / app_name
                
                if not app_path.exists():
                    self.logger.warning(f"App directory not found: {app_path}")
                    continue
                
                # Check if app is already running
                existing_pid = self.is_app_running(app_name)
                if existing_pid:
                    self.logger.info(f"App {app_name} is already running (PID: {existing_pid})")
                    continue
                
                # Safety check: limit number of apps
                if len(enabled_apps) >= MAX_APPS:
                    self.logger.warning(f"Maximum app limit ({MAX_APPS}) reached")
                    break
                
                enabled_apps.append(app_name)
                app_paths[app_name] = app_path
            
            # Step 2: Pre-allocate all ports using smart strategy
            self.port_allocations = self._pre_allocate_all_ports(enabled_apps)
            
            # Step 3: Create App objects with allocated ports
            apps = []
            for app_name in enabled_apps:
                # Gather all allocated ports for this app
                allocated_ports = {}
                for (name, port_type), port in self.port_allocations.items():
                    if name == app_name:
                        allocated_ports[port_type] = port
                
                # Create app only if we allocated at least one port
                if allocated_ports:
                    app = App(
                        name=app_name,
                        enabled=True,
                        allocated_ports=allocated_ports,
                        log_file=self.log_dir / f"{app_name}.log"
                    )
                    apps.append(app)
                    self.apps[app_name] = app
                    
                    self.logger.info(f"Prepared {app_name} with ports: {allocated_ports}")
                else:
                    self.logger.warning(f"No ports allocated for {app_name}, skipping")
            
            self.logger.info(f"Loaded {len(apps)} apps with pre-allocated ports")
            return apps
            
        except Exception as e:
            self.logger.error(f"Error loading catalog: {e}")
            return []
    
    def prepare_app_binaries(self, app_name: str) -> bool:
        """Pre-build or use cached binaries to avoid spawning build processes"""
        app_path = self.generated_apps_dir / app_name
        cache_dir = Path.home() / ".vrooli" / "build-cache"
        cache_dir.mkdir(parents=True, exist_ok=True)
        
        # Check for Go API binary
        api_source = app_path / "api" / "main.go"
        if api_source.exists():
            cached_binary = cache_dir / f"{app_name}-api"
            target_binary = app_path / "api" / f"{app_name}-api"
            
            # Check if we need to rebuild
            need_rebuild = True
            if cached_binary.exists() and target_binary.exists():
                # Compare modification times
                source_mtime = api_source.stat().st_mtime
                cache_mtime = cached_binary.stat().st_mtime
                if cache_mtime >= source_mtime:
                    need_rebuild = False
                    self.logger.info(f"Using cached binary for {app_name}")
            
            if need_rebuild:
                self.logger.info(f"Building {app_name} binary (one-time)")
                # Build directly to cache to avoid duplicate builds
                build_cmd = f"cd {app_path}/api && go build -o {cached_binary} main.go"
                result = os.system(build_cmd)
                if result != 0:
                    self.logger.error(f"Failed to build {app_name}")
                    return False
            
            # Copy from cache to target
            shutil.copy2(cached_binary, target_binary)
            os.chmod(target_binary, 0o755)
        
        # Check for Node modules (just verify, don't install here)
        package_json = app_path / "ui" / "package.json"
        if package_json.exists():
            node_modules = app_path / "ui" / "node_modules"
            if not node_modules.exists():
                # Create marker to trigger install on first real run only
                self.logger.info(f"Node modules missing for {app_name}, will install on first run")
        
        return True
    
    async def start_app(self, app: App) -> bool:
        """Start a single app with safety checks"""
        try:
            # Enhanced fork bomb detection with system monitoring
            if not self.fork_bomb_detector.record_start():
                system_load = self.fork_bomb_detector.get_system_load()
                spawn_rate = self.fork_bomb_detector.get_rate()
                self.logger.error(f"FORK BOMB/OVERLOAD DETECTED!")
                self.logger.error(f"  Process count: {system_load['process_count']}")
                self.logger.error(f"  Process growth: {system_load['process_growth']}")
                self.logger.error(f"  Spawn rate: {spawn_rate:.1f}/sec")
                self.logger.error(f"  CPU: {system_load['cpu_percent']:.1f}%")
                self.logger.error(f"  Memory: {system_load['memory_percent']:.1f}%")
                await self.emergency_shutdown()
                return False
            
            self.logger.info(f"Starting {app.name} with ports: {app.allocated_ports}")
            
            # Pre-build binaries if needed (BEFORE starting the app)
            if not self.prepare_app_binaries(app.name):
                self.logger.error(f"Failed to prepare binaries for {app.name}")
                return False
            
            app_path = self.generated_apps_dir / app.name
            manage_script = app_path / "scripts" / "manage.sh"
            
            if not manage_script.exists():
                self.logger.error(f"manage.sh not found for {app.name}")
                return False
            
            # Create CORRECT setup markers to skip redundant setup
            # The manage.sh script checks for data/.setup-state, NOT .vrooli/.setup_complete!
            data_dir = app_path / "data"
            data_dir.mkdir(parents=True, exist_ok=True)
            
            # Create the state file that manage.sh actually checks for
            state_file = data_dir / ".setup-state"
            state_data = {
                "git_commit": "bypass",
                "timestamp": time.time(),
                "setup_complete": True
            }
            state_file.write_text(json.dumps(state_data, indent=2))
            
            # Also create the old marker for compatibility
            setup_marker = app_path / ".vrooli" / ".setup_complete"
            setup_marker.parent.mkdir(parents=True, exist_ok=True)
            setup_marker.touch()
            
            # CRITICAL: Set environment variable to prevent recursive orchestrator calls
            env = os.environ.copy()
            
            # Remove any existing port variables to prevent conflicts
            for key in list(env.keys()):
                if key.endswith('_PORT'):
                    del env[key]
            
            env['VROOLI_ORCHESTRATOR_RUNNING'] = '1'  # Apps check this to skip orchestrator
            env['VROOLI_ROOT'] = str(self.vrooli_root)
            env['GENERATED_APPS_DIR'] = str(self.generated_apps_dir)
            env['FAST_MODE'] = 'true'  # Force fast mode to skip heavy setup
            
            # CRITICAL: Tell app that ports are pre-allocated
            env['PORTS_PREALLOCATED'] = '1'
            
            # PRE-CALCULATE APP_ROOT to eliminate subshells in bash scripts
            env['APP_ROOT'] = str(app_path)
            
            # Set tracking environment variables for stop-all script
            env['VROOLI_APP_NAME'] = app.name
            env['VROOLI_TRACKED'] = '1'
            env['VROOLI_SAFE'] = '1'
            
            # Set port environment variables with explicit values
            for port_type, port_num in app.allocated_ports.items():
                if port_type == "api":
                    env['SERVICE_PORT'] = str(port_num)
                    self.logger.debug(f"Set SERVICE_PORT={port_num} for {app.name}")
                elif port_type == "ui":
                    env['UI_PORT'] = str(port_num)
                    self.logger.debug(f"Set UI_PORT={port_num} for {app.name}")
                else:
                    env[f"{port_type.upper()}_PORT"] = str(port_num)
                    self.logger.debug(f"Set {port_type.upper()}_PORT={port_num} for {app.name}")
            
            # Build command
            cmd = ["bash", str(manage_script), "develop", "--fast"]
            
            # Start the process
            process = await asyncio.create_subprocess_exec(
                *cmd,
                env=env,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.STDOUT,
                start_new_session=True,
                cwd=str(app_path)
            )
            
            app.process = process
            app.pid = process.pid
            self.running_processes.append(process)
            
            # Write PID file
            if app.pid:
                self.write_app_pid(app.name, app.pid)
            
            # Wait a bit and verify it started
            await asyncio.sleep(2)
            
            if process.returncode is not None:
                # Process already exited
                app.status = "failed"
                self.logger.error(f"App {app.name} exited immediately with code {process.returncode}")
                self.remove_app_pid(app.name)
                return False
            
            # Check if port is listening
            for port_type, port_num in app.allocated_ports.items():
                if not self.is_port_available(port_num):
                    # Port is now in use - good sign
                    app.status = "running"
                    self.logger.info(f"✓ {app.name} started successfully on port {port_num}")
                    return True
            
            # No ports listening yet, but process is still running
            app.status = "starting"
            self.logger.info(f"App {app.name} is starting (PID: {app.pid})")
            return True
            
        except Exception as e:
            self.logger.error(f"Failed to start {app.name}: {e}")
            app.status = "error"
            return False
    
    async def start_all_apps(self) -> tuple[int, int]:
        """Start all enabled apps with safety checks"""
        if not self.acquire_lock():
            self.logger.error("Cannot start - another orchestrator is running")
            return 0, 0
        
        apps = self.load_enabled_apps()
        
        if not apps:
            self.logger.info("No enabled apps to start")
            return 0, 0
        
        # Check for preflight limit file
        limit_file = Path("/tmp/vrooli-max-apps-limit")
        if limit_file.exists():
            try:
                preflight_limit = int(limit_file.read_text().strip())
                if preflight_limit < len(apps):
                    self.logger.warning(f"Preflight check limited apps to {preflight_limit} (found {len(apps)})")
                    apps = apps[:preflight_limit]
            except:
                pass
        
        self.logger.info(f"Starting {len(apps)} apps with safety mechanisms enabled")
        
        success_count = 0
        fail_count = 0
        
        # Start apps with concurrency limiting and system monitoring
        concurrent_starts = 0
        app_futures = []
        
        for i, app in enumerate(apps):
            # Check system health before each app
            system_load = self.fork_bomb_detector.get_system_load()
            if system_load['process_count'] > SYSTEM_PROCESS_LIMIT:
                self.logger.error(f"CRITICAL: Process limit exceeded ({system_load['process_count']} > {SYSTEM_PROCESS_LIMIT})")
                break
            elif system_load['process_count'] > SYSTEM_WARNING_LIMIT:
                self.logger.warning(f"Approaching process limit ({system_load['process_count']} processes)")
                # Continue but slow down
                await asyncio.sleep(1.0)
            
            # Wait if too many concurrent starts
            while concurrent_starts >= MAX_CONCURRENT_STARTS:
                await asyncio.sleep(1)
                # Check completed futures
                done_futures = [f for f in app_futures if f.done()]
                concurrent_starts = len(app_futures) - len(done_futures)
            
            # Start the app
            self.logger.info(f"Starting app {i+1}/{len(apps)}: {app.name}")
            future = asyncio.create_task(self.start_app(app))
            app_futures.append(future)
            concurrent_starts += 1
            
            # Longer delay between batches
            if (i + 1) % MAX_CONCURRENT_STARTS == 0:
                await asyncio.sleep(10.0)  # Wait for batch to stabilize
            else:
                await asyncio.sleep(0.5)
            
            # Check fork bomb detector with new system monitoring
            if self.fork_bomb_detector.get_rate() > 10 or system_load['cpu_percent'] > 90:
                self.logger.warning(f"High process spawn rate detected: {self.fork_bomb_detector.get_rate():.1f}/sec")
                self.logger.warning("Slowing down app starts for safety")
                await asyncio.sleep(2)
        
        # Wait for all remaining apps to finish starting with timeout
        if app_futures:
            self.logger.info(f"Waiting for {len(app_futures)} apps to finish starting...")
            try:
                # Give apps 60 seconds total to start
                results = await asyncio.wait_for(
                    asyncio.gather(*app_futures, return_exceptions=True),
                    timeout=60.0
                )
            except asyncio.TimeoutError:
                self.logger.error("Timeout waiting for apps to start (60s exceeded)")
                # Cancel remaining futures
                for future in app_futures:
                    if not future.done():
                        future.cancel()
                # Count timeouts as failures
                results = []
                for future in app_futures:
                    if future.done():
                        try:
                            results.append(future.result())
                        except:
                            results.append(None)
                    else:
                        results.append(None)
                        fail_count += 1
            
            # Count results
            for result in results:
                if isinstance(result, Exception):
                    fail_count += 1
                    self.logger.error(f"App failed with: {result}")
                elif result:
                    success_count += 1
                else:
                    fail_count += 1
        
        return success_count, fail_count
    
    async def emergency_shutdown(self):
        """Emergency shutdown when fork bomb is detected"""
        self.logger.critical("EMERGENCY SHUTDOWN - Killing all spawned processes")
        
        # Kill all tracked processes
        for process in self.running_processes:
            try:
                process.terminate()
            except:
                pass
        
        # Give them a moment to terminate
        await asyncio.sleep(1)
        
        # Force kill any remaining
        for process in self.running_processes:
            try:
                process.kill()
            except:
                pass
        
        # Clean up PID files
        for pid_file in Path(APP_PID_DIR).glob("*.pid"):
            try:
                pid_file.unlink()
            except:
                pass
        
        self.release_lock()
        sys.exit(1)
    
    async def shutdown(self):
        """Graceful shutdown"""
        self.logger.info("Shutting down orchestrator...")
        
        # Stop all running processes
        for app in self.apps.values():
            if app.process:
                try:
                    app.process.terminate()
                    await asyncio.wait_for(app.process.wait(), timeout=5)
                except asyncio.TimeoutError:
                    app.process.kill()
                except:
                    pass
                
                # Remove PID file
                self.remove_app_pid(app.name)
        
        self.release_lock()
        self.logger.info("Orchestrator shutdown complete")


async def main():
    """Main entry point with safety checks"""
    # Check if we're being called recursively (fork bomb prevention)
    if os.environ.get('VROOLI_ORCHESTRATOR_RUNNING') == '1':
        print("ERROR: Orchestrator is already running (recursive call detected)", file=sys.stderr)
        print("This is a safety mechanism to prevent fork bombs", file=sys.stderr)
        sys.exit(1)
    
    # Check system process count as additional safety
    process_count = len(psutil.pids())
    if process_count > 1000:
        print(f"WARNING: High system process count detected: {process_count}", file=sys.stderr)
        print("This may indicate a fork bomb in progress", file=sys.stderr)
        response = input("Continue anyway? (yes/no): ")
        if response.lower() != 'yes':
            sys.exit(1)
    
    # Parse arguments
    verbose = "--verbose" in sys.argv or "-v" in sys.argv
    fast_mode = "--fast" in sys.argv or os.environ.get("FAST_MODE", "true").lower() == "true"
    
    # Determine Vrooli root
    script_path = Path(__file__).resolve()
    vrooli_root = script_path.parent.parent.parent.parent.parent
    
    # Create safe orchestrator
    orchestrator = SafeAppOrchestrator(
        vrooli_root=vrooli_root,
        verbose=verbose,
        fast_mode=fast_mode
    )
    
    try:
        # Start all apps with safety mechanisms
        success_count, fail_count = await orchestrator.start_all_apps()
        
        # Print summary
        print(f"\n{'=' * 60}")
        print(f"Orchestrator Summary:")
        print(f"  Successfully started: {success_count} apps")
        if fail_count > 0:
            print(f"  Failed to start: {fail_count} apps")
        print(f"  Fork bomb prevention: ACTIVE")
        print(f"  Process spawn rate: {orchestrator.fork_bomb_detector.get_rate():.1f}/sec")
        print(f"{'=' * 60}\n")
        
        # Keep orchestrator running to maintain child processes
        if success_count == 0:
            sys.exit(1)
        elif fail_count > 0:
            sys.exit(2)
        else:
            # Keep the event loop running indefinitely
            print("Orchestrator keeping apps alive. Press Ctrl+C to stop all apps.")
            await asyncio.Event().wait()
            
    except KeyboardInterrupt:
        await orchestrator.shutdown()
        sys.exit(130)
    except Exception as e:
        orchestrator.logger.error(f"Fatal error: {e}")
        await orchestrator.shutdown()
        sys.exit(1)


if __name__ == "__main__":
    # Check if psutil is available
    try:
        import psutil
    except ImportError:
        print("ERROR: psutil is required for safe orchestrator", file=sys.stderr)
        print("Install with: pip3 install psutil", file=sys.stderr)
        sys.exit(1)
    
    asyncio.run(main())