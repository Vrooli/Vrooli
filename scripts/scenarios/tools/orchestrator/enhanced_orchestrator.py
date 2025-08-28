#!/usr/bin/env python3
"""
Enhanced Vrooli App Orchestrator with FastAPI Management
Provides granular app control with HTTP API for scalable app management
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
import threading
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Set, Any
import psutil

# FastAPI imports
try:
    from fastapi import FastAPI, HTTPException, BackgroundTasks
    from fastapi.responses import JSONResponse
    import uvicorn
except ImportError:
    print("ERROR: FastAPI and uvicorn are required", file=sys.stderr)
    print("Install with: pip3 install fastapi uvicorn", file=sys.stderr)
    sys.exit(1)

# Import shared utilities
from orchestrator_utils import ForkBombDetector, MAX_APPS, ORCHESTRATOR_LOCK_FILE

# API Configuration
API_PORT = int(os.environ.get('ORCHESTRATOR_PORT', '9500'))  # Use environment variable with fallback
API_HOST = "0.0.0.0"

@dataclass 
class EnhancedApp:
    """Enhanced app representation with granular state tracking"""
    name: str
    enabled: bool
    allocated_ports: Dict[str, int] = field(default_factory=dict)
    process: Optional[asyncio.subprocess.Process] = None
    pid: Optional[int] = None
    log_file: Optional[Path] = None
    status: str = "stopped"  # stopped, starting, running, stopping, error
    started_at: Optional[datetime] = None
    stopped_at: Optional[datetime] = None
    restart_count: int = 0
    health_check_failures: int = 0
    config_path: Optional[Path] = None
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for API responses"""
        return {
            "name": self.name,
            "enabled": self.enabled,
            "allocated_ports": self.allocated_ports,
            "pid": self.pid,
            "status": self.status,
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "stopped_at": self.stopped_at.isoformat() if self.stopped_at else None,
            "restart_count": self.restart_count,
            "health_check_failures": self.health_check_failures,
            "log_file": str(self.log_file) if self.log_file else None
        }
    
    def is_running(self) -> bool:
        """Check if app is currently running"""
        if self.status != "running" or not self.pid:
            return False
        
        try:
            # Check if process actually exists
            process = psutil.Process(self.pid)
            return process.is_running()
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            return False


class EnhancedAppOrchestrator:
    """Enhanced orchestrator with granular app management and FastAPI"""
    
    def __init__(self, vrooli_root: Path, verbose: bool = False, fast_mode: bool = True):
        self.vrooli_root = vrooli_root
        self.verbose = verbose
        self.fast_mode = fast_mode
        
        # App management
        self.apps: Dict[str, EnhancedApp] = {}
        self.generated_apps_dir = Path.home() / "generated-apps"
        
        # Safety and monitoring
        self.fork_bomb_detector = ForkBombDetector()
        self.allocated_ports: Set[int] = set()
        
        # API components
        self.api_app = FastAPI(
            title="Vrooli App Orchestrator",
            description="Granular app management API",
            version="2.0.0"
        )
        self.api_server = None
        self.api_thread = None
        
        # Setup logging
        self.setup_logging()
        
        # Initialize API routes
        self.setup_api_routes()
        
        # Load apps
        self.discover_apps()
        
        # Detect already-running apps
        self.detect_running_apps()

    def setup_logging(self):
        """Setup structured logging"""
        log_level = logging.DEBUG if self.verbose else logging.INFO
        logging.basicConfig(
            level=log_level,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
        )
        self.logger = logging.getLogger('EnhancedOrchestrator')
    
    def discover_apps(self):
        """Discover all available apps and load their configurations"""
        if not self.generated_apps_dir.exists():
            self.logger.warning(f"Generated apps directory not found: {self.generated_apps_dir}")
            return
        
        self.logger.info(f"Discovering apps in {self.generated_apps_dir}")
        
        for app_dir in self.generated_apps_dir.iterdir():
            if app_dir.is_dir() and not app_dir.name.startswith('.'):
                config_path = app_dir / ".vrooli" / "service.json"
                
                if config_path.exists():
                    try:
                        with open(config_path) as f:
                            config = json.load(f)
                        
                        app_name = app_dir.name
                        enabled = config.get("enabled", True)  # Default to enabled
                        
                        # Pre-allocate ports from config
                        allocated_ports = self.pre_allocate_ports_for_app(app_name, config)
                        
                        app = EnhancedApp(
                            name=app_name,
                            enabled=enabled,
                            allocated_ports=allocated_ports,
                            config_path=config_path,
                            log_file=Path.home() / ".vrooli" / "logs" / f"{app_name}.log"
                        )
                        
                        self.apps[app_name] = app
                        self.logger.debug(f"Discovered app: {app_name} (enabled: {enabled})")
                        
                    except Exception as e:
                        self.logger.error(f"Error loading config for {app_dir.name}: {e}")
                else:
                    self.logger.debug(f"No config found for {app_dir.name}")
        
        self.logger.info(f"Discovered {len(self.apps)} apps ({sum(1 for a in self.apps.values() if a.enabled)} enabled)")
    
    def detect_running_apps(self):
        """Detect apps that are already running (started outside orchestrator)"""
        self.logger.info("Detecting already-running apps...")
        detected_count = 0
        
        for app_name, app in self.apps.items():
            app_dir = self.generated_apps_dir / app_name
            
            # Check for running processes related to this app
            # Method 1: Check if ports are in use
            for port_type, port_num in app.allocated_ports.items():
                if not self.is_port_available(port_num):
                    # Port is in use, likely app is running
                    try:
                        # Try to find the PID using lsof
                        import subprocess
                        result = subprocess.run(
                            ["lsof", "-t", f"-i:{port_num}"],
                            capture_output=True,
                            text=True,
                            timeout=2
                        )
                        if result.returncode == 0 and result.stdout.strip():
                            pid = int(result.stdout.strip().split('\n')[0])
                            # Verify it's actually our app
                            proc = psutil.Process(pid)
                            if app_name in proc.cwd() or app_name in ' '.join(proc.cmdline()):
                                app.pid = pid
                                app.status = "running"
                                app.started_at = datetime.fromtimestamp(proc.create_time())
                                detected_count += 1
                                self.logger.info(f"Detected {app_name} already running (PID: {pid}, Port: {port_num})")
                                break
                    except (subprocess.TimeoutExpired, ValueError, psutil.NoSuchProcess, psutil.AccessDenied):
                        pass
            
            # Method 2: Check for manage.sh or start.sh processes
            if app.status != "running":
                try:
                    for proc in psutil.process_iter(['pid', 'cmdline', 'cwd', 'create_time']):
                        try:
                            cmdline = ' '.join(proc.info['cmdline'] or [])
                            cwd = proc.info['cwd'] or ''
                            
                            # Check if this process is related to our app
                            if (f"generated-apps/{app_name}" in cmdline or 
                                f"generated-apps/{app_name}" in cwd or
                                (app_name in cwd and "generated-apps" in cwd)):
                                
                                # Found a process for this app
                                app.pid = proc.info['pid']
                                app.status = "running"
                                app.started_at = datetime.fromtimestamp(proc.info['create_time'])
                                detected_count += 1
                                self.logger.info(f"Detected {app_name} already running (PID: {proc.info['pid']})")
                                break
                        except (psutil.NoSuchProcess, psutil.AccessDenied, KeyError):
                            continue
                except Exception as e:
                    self.logger.debug(f"Error detecting process for {app_name}: {e}")
        
        if detected_count > 0:
            self.logger.info(f"Detected {detected_count} apps already running")
        else:
            self.logger.info("No previously running apps detected")
    
    def pre_allocate_ports_for_app(self, app_name: str, config: Dict) -> Dict[str, int]:
        """Pre-allocate ports for a specific app"""
        ports_config = config.get("ports", {})
        allocated = {}
        
        for port_type, port_info in ports_config.items():
            if isinstance(port_info, dict):
                if "fixed" in port_info:
                    fixed_port = int(port_info["fixed"])
                    if self.is_port_available(fixed_port):
                        allocated[port_type] = fixed_port
                        self.allocated_ports.add(fixed_port)
                elif "range" in port_info:
                    range_str = port_info["range"]
                    if "-" in range_str:
                        start, end = map(int, range_str.split("-"))
                        for port in range(start, end + 1):
                            if self.is_port_available(port):
                                allocated[port_type] = port
                                self.allocated_ports.add(port)
                                break
        
        return allocated
    
    def is_port_available(self, port: int) -> bool:
        """Check if a port is available for allocation"""
        if port in self.allocated_ports:
            return False
        
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
                sock.bind(('', port))
                return True
        except OSError:
            return False
    
    async def start_app(self, app_name: str) -> bool:
        """Start a specific app"""
        if app_name not in self.apps:
            raise HTTPException(status_code=404, detail=f"App '{app_name}' not found")
        
        app = self.apps[app_name]
        
        if app.status == "running" and app.is_running():
            self.logger.info(f"App {app_name} is already running")
            return True
        
        if not app.enabled:
            raise HTTPException(status_code=400, detail=f"App '{app_name}' is not enabled")
        
        self.logger.info(f"Starting app: {app_name}")
        app.status = "starting"
        app.started_at = datetime.now()
        
        try:
            # Check fork bomb protection
            if not self.fork_bomb_detector.record_start():
                raise Exception("Fork bomb protection triggered")
            
            # Build app if needed
            if not await self.ensure_app_built(app):
                raise Exception("Failed to build app")
            
            # Start the app process using manage.sh develop
            app_path = self.generated_apps_dir / app_name
            manage_script = app_path / "scripts" / "manage.sh"
            
            if not manage_script.exists():
                self.logger.error(f"manage.sh not found for {app_name}")
                return False
            
            # Create setup markers to skip redundant setup (manage.sh checks these)
            data_dir = app_path / "data"
            data_dir.mkdir(parents=True, exist_ok=True)
            
            # Create the state file that manage.sh checks for
            state_file = data_dir / ".setup-state"
            state_data = {
                "git_commit": "bypass",
                "timestamp": time.time(),
                "setup_complete": True
            }
            state_file.write_text(json.dumps(state_data, indent=2))
            
            # Also create old marker for compatibility
            setup_marker = app_path / ".vrooli" / ".setup_complete"
            setup_marker.parent.mkdir(parents=True, exist_ok=True)
            setup_marker.touch()
            
            # Prepare environment
            env = os.environ.copy()
            
            # Remove any existing port variables to prevent conflicts
            for key in list(env.keys()):
                if key.endswith('_PORT'):
                    del env[key]
            
            env['VROOLI_ORCHESTRATOR_RUNNING'] = '1'  # Prevent recursive orchestrator calls
            env['VROOLI_ROOT'] = str(self.vrooli_root)
            env['GENERATED_APPS_DIR'] = str(self.generated_apps_dir)
            env['FAST_MODE'] = 'true'  # Skip heavy setup
            env['PORTS_PREALLOCATED'] = '1'  # Ports are pre-allocated
            env['APP_ROOT'] = str(app_path)
            env['VROOLI_APP_NAME'] = app_name
            env['VROOLI_TRACKED'] = '1'
            env['VROOLI_SAFE'] = '1'
            
            # Set port environment variables
            for port_type, port_num in app.allocated_ports.items():
                if port_type == "api":
                    env['SERVICE_PORT'] = str(port_num)
                    self.logger.debug(f"Set SERVICE_PORT={port_num} for {app_name}")
                elif port_type == "ui":
                    env['UI_PORT'] = str(port_num)
                    self.logger.debug(f"Set UI_PORT={port_num} for {app_name}")
                else:
                    env[f"{port_type.upper()}_PORT"] = str(port_num)
            
            # Ensure log directory exists
            app.log_file.parent.mkdir(parents=True, exist_ok=True)
            
            # Start using manage.sh develop (reads service.json and executes lifecycle)
            with open(app.log_file, 'a') as log_f:
                app.process = await asyncio.create_subprocess_exec(
                    "bash", str(manage_script), "develop", "--fast",
                    cwd=str(app_path),
                    env=env,
                    stdout=log_f,
                    stderr=log_f
                )
            
            app.pid = app.process.pid
            
            # Wait a moment and check if process started successfully
            await asyncio.sleep(1)
            
            if app.process.returncode is None:
                # Process is still running
                app.status = "running"
                self.logger.info(f"✓ {app_name} started successfully (PID: {app.pid})")
                return True
            else:
                # Process exited immediately
                app.status = "error"
                app.pid = None
                app.process = None
                self.logger.error(f"✗ {app_name} failed to start (exited immediately)")
                return False
                
        except Exception as e:
            app.status = "error"
            app.pid = None 
            app.process = None
            self.logger.error(f"Failed to start {app_name}: {e}")
            return False
    
    async def stop_app(self, app_name: str, force: bool = False) -> bool:
        """Stop a specific app"""
        if app_name not in self.apps:
            raise HTTPException(status_code=404, detail=f"App '{app_name}' not found")
        
        app = self.apps[app_name]
        
        if app.status == "stopped" or not app.is_running():
            self.logger.info(f"App {app_name} is already stopped")
            app.status = "stopped"
            app.stopped_at = datetime.now()
            return True
        
        self.logger.info(f"Stopping app: {app_name}")
        app.status = "stopping"
        
        try:
            if app.pid:
                if force:
                    # Force kill
                    os.kill(app.pid, signal.SIGKILL)
                else:
                    # Graceful shutdown
                    os.kill(app.pid, signal.SIGTERM)
                    
                    # Wait for graceful shutdown
                    for _ in range(5):
                        if not app.is_running():
                            break
                        await asyncio.sleep(1)
                    
                    # Force kill if still running
                    if app.is_running():
                        os.kill(app.pid, signal.SIGKILL)
            
            app.status = "stopped"
            app.stopped_at = datetime.now()
            app.pid = None
            app.process = None
            
            self.logger.info(f"✓ {app_name} stopped successfully")
            return True
            
        except Exception as e:
            self.logger.error(f"Error stopping {app_name}: {e}")
            return False
    
    async def ensure_app_built(self, app: EnhancedApp) -> bool:
        """Ensure app is built and ready to run"""
        app_path = self.generated_apps_dir / app.name
        
        # Check for Go API binary
        api_source = app_path / "api" / "main.go" 
        if api_source.exists():
            cache_dir = Path.home() / ".vrooli" / "build-cache"
            cache_dir.mkdir(parents=True, exist_ok=True)
            
            cached_binary = cache_dir / f"{app.name}-api"
            target_binary = app_path / "api" / f"{app.name}-api"
            
            # Check if we need to rebuild
            need_rebuild = True
            if cached_binary.exists() and target_binary.exists():
                source_mtime = api_source.stat().st_mtime
                cache_mtime = cached_binary.stat().st_mtime
                if cache_mtime >= source_mtime:
                    need_rebuild = False
            
            if need_rebuild:
                self.logger.info(f"Building {app.name} binary")
                build_cmd = f"cd {app_path}/api && go build -o {cached_binary} main.go"
                result = os.system(build_cmd)
                if result != 0:
                    return False
            
            # Copy to target location
            if cached_binary.exists():
                import shutil
                shutil.copy2(cached_binary, target_binary)
                os.chmod(target_binary, 0o755)
        
        return True
    
    def get_app_status(self, app_name: str) -> Dict[str, Any]:
        """Get detailed status of a specific app"""
        if app_name not in self.apps:
            raise HTTPException(status_code=404, detail=f"App '{app_name}' not found")
        
        app = self.apps[app_name]
        
        # Update status based on actual process state
        if app.status == "running" and not app.is_running():
            app.status = "stopped"
            app.stopped_at = datetime.now()
            app.pid = None
            app.process = None
        
        status = app.to_dict()
        
        # Add port availability info
        port_status = {}
        for port_type, port_num in app.allocated_ports.items():
            port_status[port_type] = {
                "port": port_num,
                "listening": not self.is_port_available(port_num)
            }
        status["port_status"] = port_status
        
        return status

    def get_all_apps(self) -> List[Dict[str, Any]]:
        """Get status of all apps"""
        return [self.get_app_status(name) for name in sorted(self.apps.keys())]
    
    def get_running_apps(self) -> List[Dict[str, Any]]:
        """Get status of only running apps"""
        running = []
        for app_name, app in self.apps.items():
            if app.is_running():
                running.append(self.get_app_status(app_name))
        return running
    
    def setup_api_routes(self):
        """Setup FastAPI routes"""
        
        @self.api_app.get("/health")
        async def health():
            """Orchestrator health check"""
            return {
                "status": "healthy",
                "timestamp": datetime.now().isoformat(),
                "apps_total": len(self.apps),
                "apps_running": len([a for a in self.apps.values() if a.is_running()]),
                "fork_bomb_rate": self.fork_bomb_detector.get_rate()
            }
        
        @self.api_app.get("/apps")
        async def list_apps():
            """List all available apps"""
            return {
                "apps": self.get_all_apps(),
                "total": len(self.apps),
                "running": len([a for a in self.apps.values() if a.is_running()])
            }
        
        @self.api_app.get("/apps/running")
        async def list_running_apps():
            """List only running apps"""
            running = self.get_running_apps()
            return {
                "apps": running,
                "count": len(running)
            }
        
        @self.api_app.get("/apps/{app_name}/status")
        async def app_status(app_name: str):
            """Get status of specific app"""
            return self.get_app_status(app_name)
        
        @self.api_app.post("/apps/{app_name}/start")
        async def start_app_endpoint(app_name: str, background_tasks: BackgroundTasks):
            """Start a specific app"""
            success = await self.start_app(app_name)
            if success:
                return {"message": f"App '{app_name}' started successfully"}
            else:
                raise HTTPException(status_code=500, detail=f"Failed to start app '{app_name}'")
        
        @self.api_app.post("/apps/{app_name}/stop")
        async def stop_app_endpoint(app_name: str, force: bool = False):
            """Stop a specific app"""
            success = await self.stop_app(app_name, force)
            if success:
                return {"message": f"App '{app_name}' stopped successfully"}
            else:
                raise HTTPException(status_code=500, detail=f"Failed to stop app '{app_name}'")
        
        @self.api_app.post("/apps/start-all")
        async def start_all_apps():
            """Start all enabled apps"""
            results = []
            for app_name, app in self.apps.items():
                if app.enabled:
                    try:
                        success = await self.start_app(app_name)
                        results.append({"app": app_name, "success": success})
                    except Exception as e:
                        results.append({"app": app_name, "success": False, "error": str(e)})
            
            successful = sum(1 for r in results if r["success"])
            return {
                "message": f"Started {successful}/{len(results)} apps",
                "results": results
            }
        
        @self.api_app.post("/apps/stop-all")
        async def stop_all_apps(force: bool = False):
            """Stop all running apps"""
            results = []
            for app_name, app in self.apps.items():
                if app.is_running():
                    try:
                        success = await self.stop_app(app_name, force)
                        results.append({"app": app_name, "success": success})
                    except Exception as e:
                        results.append({"app": app_name, "success": False, "error": str(e)})
            
            successful = sum(1 for r in results if r["success"])
            return {
                "message": f"Stopped {successful}/{len(results)} apps", 
                "results": results
            }
        
        @self.api_app.get("/status")
        async def system_status():
            """Get comprehensive system status"""
            return {
                "orchestrator": {
                    "status": "running",
                    "api_port": API_PORT,
                    "vrooli_root": str(self.vrooli_root)
                },
                "apps": {
                    "total": len(self.apps),
                    "running": len([a for a in self.apps.values() if a.is_running()]),
                    "enabled": len([a for a in self.apps.values() if a.enabled])
                },
                "system": {
                    "process_count": len(psutil.pids()),
                    "cpu_percent": psutil.cpu_percent(interval=0.1),
                    "memory_percent": psutil.virtual_memory().percent,
                    "fork_bomb_rate": self.fork_bomb_detector.get_rate()
                }
            }

    async def start_api_server(self):
        """Start the FastAPI server in a separate thread"""
        def run_server():
            config = uvicorn.Config(
                self.api_app, 
                host=API_HOST, 
                port=API_PORT,
                log_level="info" if self.verbose else "warning"
            )
            server = uvicorn.Server(config)
            asyncio.run(server.serve())
        
        self.api_thread = threading.Thread(target=run_server, daemon=True)
        self.api_thread.start()
        
        # Wait for server to start
        await asyncio.sleep(2)
        self.logger.info(f"API server started on http://{API_HOST}:{API_PORT}")

    async def shutdown(self):
        """Shutdown orchestrator and all apps"""
        self.logger.info("Shutting down orchestrator...")
        
        # Stop all running apps
        for app_name, app in self.apps.items():
            if app.is_running():
                await self.stop_app(app_name, force=True)
        
        self.logger.info("Orchestrator shutdown complete")


async def main():
    """Main entry point for enhanced orchestrator"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Enhanced Vrooli App Orchestrator")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose logging")
    parser.add_argument("--fast", action="store_true", help="Enable fast mode")
    parser.add_argument("--background", action="store_true", help="Run in background")
    parser.add_argument("--start-all", action="store_true", help="Start all enabled apps")
    
    args = parser.parse_args()
    
    # Setup
    vrooli_root = Path.cwd()
    if not vrooli_root.name == "Vrooli":
        vrooli_root = Path.home() / "Vrooli"
    
    # Create orchestrator
    orchestrator = EnhancedAppOrchestrator(
        vrooli_root=vrooli_root,
        verbose=args.verbose,
        fast_mode=args.fast
    )
    
    try:
        # Start API server
        await orchestrator.start_api_server()
        
        # Start all apps if requested
        if args.start_all:
            print("Starting all enabled apps...")
            for app_name, app in orchestrator.apps.items():
                if app.enabled:
                    success = await orchestrator.start_app(app_name)
                    status = "✓" if success else "✗"
                    print(f"  {status} {app_name}")
        
        # Summary
        total_apps = len(orchestrator.apps)
        running_apps = len([a for a in orchestrator.apps.values() if a.is_running()])
        enabled_apps = len([a for a in orchestrator.apps.values() if a.enabled])
        
        print(f"\n{'=' * 60}")
        print(f"Enhanced Orchestrator Running:")
        print(f"  API Server: http://localhost:{API_PORT}")
        print(f"  Apps: {running_apps}/{total_apps} running ({enabled_apps} enabled)")
        print(f"  Management: Use API or CLI commands")
        print(f"{'=' * 60}\n")
        
        print("Orchestrator running. Press Ctrl+C to stop all apps and exit.")
        await asyncio.Event().wait()
        
    except KeyboardInterrupt:
        await orchestrator.shutdown()
        sys.exit(130)
    except Exception as e:
        orchestrator.logger.error(f"Fatal error: {e}")
        await orchestrator.shutdown()
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())