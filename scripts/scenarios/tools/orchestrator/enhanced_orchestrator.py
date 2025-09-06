#!/usr/bin/env python3
"""
Enhanced Vrooli App Orchestrator with FastAPI Management
Provides granular app control with HTTP API for scalable app management
"""

import asyncio
import http.client
import json
import logging
import os
import signal
import socket
import subprocess
import sys
import time
import threading
import urllib.parse
from dataclasses import dataclass, field
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Set, Any, Tuple
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
from orchestrator_utils import (
    ForkBombDetector, MAX_APPS, MAX_CONCURRENT_STARTS,
    ORCHESTRATOR_LOCK_FILE, ORCHESTRATOR_PID_FILE
)

# Import the new PID manager
from pid_manager import OrchestrationLockManager

# API Configuration
API_PORT = int(os.environ.get('ORCHESTRATOR_PORT', '9500'))  # Use environment variable with fallback
API_HOST = "0.0.0.0"

@dataclass 
class EnhancedApp:
    """Enhanced app representation with granular state tracking"""
    name: str
    enabled: bool
    allocated_ports: Dict[str, Dict[str, Any]] = field(default_factory=dict)  # Now stores full port config
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
        # Simplify allocated_ports for API response (just port numbers)
        simple_ports = {}
        for port_type, port_config in self.allocated_ports.items():
            if isinstance(port_config, dict):
                simple_ports[port_type] = port_config.get('port')
            else:
                simple_ports[port_type] = port_config
        
        # Calculate runtime duration if the app is running
        runtime_seconds = None
        runtime_formatted = None
        if self.started_at and self.status == "running":
            duration = datetime.now() - self.started_at
            runtime_seconds = int(duration.total_seconds())
            
            # Format duration as human-readable string
            days = duration.days
            hours, remainder = divmod(duration.seconds, 3600)
            minutes, seconds = divmod(remainder, 60)
            
            parts = []
            if days > 0:
                parts.append(f"{days}d")
            if hours > 0:
                parts.append(f"{hours}h")
            if minutes > 0:
                parts.append(f"{minutes}m")
            if seconds > 0 or not parts:  # Always show seconds if nothing else
                parts.append(f"{seconds}s")
            runtime_formatted = " ".join(parts)
        
        return {
            "name": self.name,
            "enabled": self.enabled,
            "allocated_ports": simple_ports,
            "pid": self.pid,
            "status": self.status,
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "stopped_at": self.stopped_at.isoformat() if self.stopped_at else None,
            "runtime_seconds": runtime_seconds,
            "runtime_formatted": runtime_formatted,
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
        self.scenarios_dir = self.vrooli_root / "scenarios"
        self.project_root = self.vrooli_root  # Add project_root attribute
        
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
        
        # Load resource ports for RESOURCE_PORTS expansion
        self.resource_ports = self.load_resource_ports()
        
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
        """Discover available scenarios"""
        if not self.scenarios_dir.exists():
            self.logger.warning(f"Scenarios directory not found: {self.scenarios_dir}")
            return
        
        self.logger.info(f"Discovering scenarios in {self.scenarios_dir}")
        
        for scenario_dir in self.scenarios_dir.iterdir():
            if scenario_dir.is_dir() and not scenario_dir.name.startswith('.'):
                config_path = scenario_dir / ".vrooli" / "service.json"
                
                if config_path.exists():
                    try:
                        config = self.load_service_json(config_path)
                        
                        app_name = scenario_dir.name
                        enabled = config.get("enabled", True)  # Default to enabled
                        
                        # Load all configured ports (for discovery, include even if in use)
                        ports_config = config.get("ports", {})
                        allocated_ports = {}
                        for port_type, port_info in ports_config.items():
                            if isinstance(port_info, dict):
                                env_var = port_info.get("env_var", f"{port_type.upper()}_PORT")
                                port_num = port_info.get("port")
                                if port_num:
                                    allocated_ports[port_type] = {
                                        "port": int(port_num),
                                        "env_var": env_var
                                    }
                        
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
                        self.logger.error(f"Error loading config for {scenario_dir.name}: {e}")
                else:
                    self.logger.debug(f"No config found for {scenario_dir.name}")
        
        self.logger.info(f"Discovered {len(self.apps)} apps ({sum(1 for a in self.apps.values() if a.enabled)} enabled)")
    
    def detect_running_apps(self, verbose_log=True):
        """Detect apps that are already running (started outside orchestrator)"""
        if verbose_log:
            self.logger.info("Detecting already-running apps...")
        detected_count = 0
        status_changes = []
        
        for app_name, app in self.apps.items():
            app_dir = self.scenarios_dir / app_name
            app_detected = False
            detected_ports = []
            detected_pids = []
            
            # Look for actual service processes (not lifecycle.sh)
            try:
                for proc in psutil.process_iter(['pid', 'cmdline', 'cwd', 'create_time']):
                    try:
                        cmdline = ' '.join(proc.info['cmdline'] or [])
                        cwd = proc.info['cwd'] or ''
                        pid = proc.info['pid']
                        
                        # Look for actual API/UI processes
                        if (f"{app_name}-api" in cmdline or  # Go API binary
                            (f"scenarios/{app_name}/ui" in cwd and "node server.js" in cmdline) or  # Node UI
                            (f"scenarios/{app_name}/api" in cwd and any(x in cmdline for x in ["go run", "./", app_name]))):  # API process
                            
                            detected_pids.append(pid)
                            if not app_detected:
                                # Use first found PID as primary
                                prev_status = app.status
                                app.pid = pid
                                app.status = "running"
                                # Only set started_at if status is changing from non-running to running
                                if prev_status != "running":
                                    app.started_at = datetime.now()
                                
                                # Perform health check to determine actual status
                                if app.allocated_ports:
                                    total_ports = len(app.allocated_ports)
                                    responding_ports = 0
                                    listening_ports = 0
                                    
                                    for port_type, port_config in app.allocated_ports.items():
                                        port_num = port_config.get("port") if isinstance(port_config, dict) else port_config
                                        if port_num and not self.is_port_available(port_num):
                                            listening_ports += 1
                                            health_check = self.check_service_health(port_num, port_type, app_name)
                                            if health_check["responding"]:
                                                responding_ports += 1
                                    
                                    # Determine final status based on health
                                    if responding_ports == total_ports:
                                        # All ports responding - keep as running
                                        final_status = "running"
                                    elif responding_ports > 0 or listening_ports > 0:
                                        # Some ports responding or listening - mark as degraded
                                        app.status = "degraded"
                                        final_status = "degraded"
                                    else:
                                        # No ports responding - mark as degraded
                                        app.status = "degraded"
                                        final_status = "degraded"
                                else:
                                    # No ports configured - assume healthy
                                    final_status = "running"
                                
                                status_changes.append((app_name, prev_status, final_status))
                                app_detected = True
                                
                    except (psutil.NoSuchProcess, psutil.AccessDenied, KeyError):
                        continue
            except Exception as e:
                self.logger.debug(f"Error detecting process for {app_name}: {e}")
            
            # Check for running processes related to this app via ports
            if not app_detected:
                for port_type, port_config in app.allocated_ports.items():
                    # Handle both old and new format
                    port_num = port_config.get('port') if isinstance(port_config, dict) else port_config
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
                                    if not app_detected:
                                        prev_status = app.status
                                        app.pid = pid
                                        app.status = "running"
                                        # Only set started_at if status is changing from non-running to running
                                        if prev_status != "running":
                                            app.started_at = datetime.now()
                                            status_changes.append((app_name, prev_status, "running"))
                                        app_detected = True
                                    detected_ports.append(f"{port_type}:{port_num}")
                        except (subprocess.TimeoutExpired, ValueError, psutil.NoSuchProcess, psutil.AccessDenied):
                            pass
            
            if app_detected:
                detected_count += 1
                # Reload full port configuration from service.json for already-running apps
                config_path = self.scenarios_dir / app_name / ".vrooli" / "service.json"
                if config_path.exists():
                    try:
                        config = self.load_service_json(config_path)
                        # For running apps, load ALL ports from config (not just available ones)
                        ports_config = config.get("ports", {})
                        allocated = {}
                        for port_type, port_info in ports_config.items():
                            if isinstance(port_info, dict):
                                env_var = port_info.get("env_var", f"{port_type.upper()}_PORT")
                                port_num = port_info.get("port")
                                if port_num:
                                    allocated[port_type] = {
                                        "port": int(port_num),
                                        "env_var": env_var
                                    }
                        app.allocated_ports = allocated
                    except Exception as e:
                        self.logger.debug(f"Could not reload port config for {app_name}: {e}")
                
                if verbose_log or status_changes:
                    self.logger.info(f"Detected {app_name} already running (PID: {app.pid}, Ports: {', '.join(detected_ports) if detected_ports else 'N/A'})")
            elif app.status == "running" and not app_detected:
                # App was marked as running but no process found - mark as stopped
                prev_status = app.status
                app.status = "stopped"
                app.pid = None
                status_changes.append((app_name, prev_status, "stopped"))
                if verbose_log:
                    self.logger.info(f"App {app_name} no longer running (was {prev_status})")
        
        if verbose_log:
            if detected_count > 0:
                self.logger.info(f"Detected {detected_count} apps already running")
            else:
                self.logger.info("No previously running apps detected")
        
        return status_changes
    
    def pre_allocate_ports_for_app(self, app_name: str, config: Dict) -> Dict[str, Dict[str, Any]]:
        """Pre-allocate ports for a specific app"""
        ports_config = config.get("ports", {})
        allocated = {}
        
        for port_type, port_info in ports_config.items():
            if isinstance(port_info, dict):
                # Get env_var from config, with smart defaults
                if port_type == "api":
                    default_env_var = "API_PORT"
                elif port_type == "ui":
                    default_env_var = "UI_PORT"
                else:
                    default_env_var = f"{port_type.upper()}_PORT"
                
                env_var = port_info.get("env_var", default_env_var)
                
                # Handle new service.json format with "port" key
                if "port" in port_info:
                    desired_port = int(port_info["port"])
                    if self.is_port_available(desired_port):
                        allocated[port_type] = {
                            "port": desired_port,
                            "env_var": env_var
                        }
                        self.allocated_ports.add(desired_port)
                    elif port_info.get("fallback") == "auto":
                        # Auto-allocate a port if desired one is taken
                        for port in range(30000, 40000):
                            if self.is_port_available(port):
                                allocated[port_type] = {
                                    "port": port,
                                    "env_var": env_var
                                }
                                self.allocated_ports.add(port)
                                break
                # Legacy format support
                elif "fixed" in port_info:
                    fixed_port = int(port_info["fixed"])
                    if self.is_port_available(fixed_port):
                        allocated[port_type] = {
                            "port": fixed_port,
                            "env_var": env_var
                        }
                        self.allocated_ports.add(fixed_port)
                elif "range" in port_info:
                    range_str = port_info["range"]
                    if "-" in range_str:
                        start, end = map(int, range_str.split("-"))
                        for port in range(start, end + 1):
                            if self.is_port_available(port):
                                allocated[port_type] = {
                                    "port": port,
                                    "env_var": env_var
                                }
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
    
    async def start_app(self, app_name: str, retry_count: int = 0, max_retries: int = 2) -> bool:
        """Start a specific app using direct lifecycle execution with retry logic"""
        if app_name not in self.apps:
            raise HTTPException(status_code=404, detail=f"App '{app_name}' not found")
        
        app = self.apps[app_name]
        
        if app.status == "running" and app.is_running():
            self.logger.info(f"App {app_name} is already running")
            return True
        
        if not app.enabled:
            raise HTTPException(status_code=400, detail=f"App '{app_name}' is not enabled")
        
        if retry_count > 0:
            self.logger.info(f"Starting app: {app_name} (retry {retry_count}/{max_retries})")
        else:
            self.logger.info(f"Starting app: {app_name}")
        app.status = "starting"
        app.started_at = datetime.now()
        
        try:
            # Check fork bomb protection
            if not self.fork_bomb_detector.record_start():
                raise Exception("Fork bomb protection triggered")
            
            # Use direct lifecycle execution instead of manage.sh
            app_path = self.scenarios_dir / app_name
            config_path = app_path / ".vrooli" / "service.json"
            
            if not app_path.exists():
                self.logger.error(f"Scenario path not found: {app_path}")
                return False
            
            # Let lifecycle.sh handle setup state properly
            # Don't create fake setup markers - this prevents npm install from running
            
            # Prepare environment with allocated ports
            env = os.environ.copy()
            
            # Clean existing port variables
            for key in list(env.keys()):
                if key.endswith('_PORT'):
                    del env[key]
            
            # Set allocated ports using the shared function
            port_env = self.get_port_environment(app)
            env.update(port_env)
            
            # Call lifecycle.sh directly (it handles resource environment loading internally)
            lifecycle_script = self.vrooli_root / "scripts" / "lib" / "utils" / "lifecycle.sh"
            
            # Ensure log directory exists
            app.log_file.parent.mkdir(parents=True, exist_ok=True)
            
            # Start the lifecycle execution in background
            # lifecycle.sh automatically loads resource environments via lifecycle::load_resource_environments()
            with open(app.log_file, 'a') as log_f:
                app.process = await asyncio.create_subprocess_exec(
                    "bash", str(lifecycle_script), app_name, "develop", "--fast",
                    env=env,
                    stdout=log_f,
                    stderr=log_f,
                    preexec_fn=os.setsid  # Create new process group
                )
            
            app.pid = app.process.pid
            
            # Wait a moment for processes to start
            await asyncio.sleep(3)
            
            # Wait for lifecycle to complete and find actual service processes
            # lifecycle.sh should exit after starting background services
            # Get timeout from service.json or use default
            timeout = 60.0  # Default timeout increased from 10 to 60 seconds
            if config_path.exists():
                try:
                    config = self.load_service_json(config_path)
                    # Check for lifecycle.timeout or startup_timeout in config
                        timeout = float(config.get('lifecycle', {}).get('timeout', 
                                      config.get('startup_timeout', timeout)))
                except:
                    pass  # Use default timeout if config read fails
            
            try:
                await asyncio.wait_for(app.process.wait(), timeout=timeout)
            except asyncio.TimeoutError:
                # Lifecycle is still running, probably needs more time
                self.logger.warning(f"{app_name} lifecycle taking longer than expected (>{timeout}s)")
                # Check if process actually terminated with an error
                if app.process.returncode is not None and app.process.returncode > 0:
                    app.status = "error"
                    app.pid = None
                    self.logger.error(f"✗ {app_name} lifecycle failed with exit code {app.process.returncode}")
                    return False
            
            # Always check for actual service processes (lifecycle may have exited)
            actual_pids = []
            for proc in psutil.process_iter(['pid', 'cmdline', 'cwd']):
                try:
                    cmdline = ' '.join(proc.info['cmdline'] or [])
                    cwd = proc.info['cwd'] or ''
                    
                    # Look for actual service processes (API servers, not lifecycle wrappers)
                    if (f"scenarios/{app_name}" in cwd and 
                        any(svc in cmdline for svc in [app_name + "-api", "npm start", "node", "go run", "./"])):
                        actual_pids.append(proc.info['pid'])
                except (psutil.NoSuchProcess, psutil.AccessDenied):
                    continue
            
            if actual_pids:
                # Track the first service process
                app.pid = actual_pids[0]
                app.status = "running"
                self.logger.info(f"✓ {app_name} started successfully ({len(actual_pids)} processes)")
                return True
            else:
                # Check if lifecycle process failed with error exit code
                if app.process and app.process.returncode is not None and app.process.returncode > 0:
                    exit_code = app.process.returncode
                    app.process = None
                    
                    # Determine if we should retry based on exit code and retry count
                    should_retry = (exit_code == 1 and retry_count < max_retries)
                    
                    if should_retry:
                        app.status = "error"
                        app.pid = None
                        self.logger.warning(f"✗ {app_name} failed (exit code: {exit_code}), will retry in {5 * (retry_count + 1)}s")
                        app.restart_count += 1
                        
                        # Exponential backoff
                        await asyncio.sleep(5 * (retry_count + 1))
                        
                        # Recursive retry
                        return await self.start_app(app_name, retry_count + 1, max_retries)
                    else:
                        app.status = "error"
                        app.pid = None
                        self.logger.error(f"✗ {app_name} failed to start (exit code: {exit_code})")
                        return False
                else:
                    # Lifecycle completed but no services detected
                    # Check if lifecycle had a non-zero exit code (like 124 for timeout)
                    if app.process and app.process.returncode is not None and app.process.returncode > 0:
                        app.status = "error"
                        app.pid = None
                        self.logger.error(f"✗ {app_name} lifecycle failed with exit code {app.process.returncode}")
                        return False
                    else:
                        # Lifecycle completed successfully but services might be delayed
                        app.status = "running"
                        self.logger.info(f"✓ {app_name} lifecycle executed (services may be starting)")
                        return True
                
        except Exception as e:
            app.status = "error"
            app.pid = None 
            app.process = None
            
            # Retry on certain exceptions
            if retry_count < max_retries and "Connection" in str(e):
                self.logger.warning(f"Failed to start {app_name}: {e}, will retry in {5 * (retry_count + 1)}s")
                app.restart_count += 1
                await asyncio.sleep(5 * (retry_count + 1))
                return await self.start_app(app_name, retry_count + 1, max_retries)
            else:
                self.logger.error(f"Failed to start {app_name}: {e}")
                return False
    
    async def stop_app(self, app_name: str, force: bool = False) -> bool:
        """Stop a specific app using lifecycle executor"""
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
            # Stop processes
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
    
    
    def get_health_config(self, app_name: str) -> Dict[str, Any]:
        """Get health configuration from service.json"""
        config_path = self.scenarios_dir / app_name / ".vrooli" / "service.json"
        
        if not config_path.exists():
            return None
            
        try:
            config = self.load_service_json(config_path)
            return config.get("lifecycle", {}).get("health", None)
        except:
            return None
    
    def check_service_health(self, port: int, port_type: str, app_name: str = None) -> Dict[str, Any]:
        """Check if a service on a port is actually responding"""
        import socket
        import http.client
        
        result = {
            "port": port,
            "listening": False,
            "responding": False,
            "status_code": None
        }
        
        # First check if port is bound
        if self.is_port_available(port):
            return result  # Port not even bound
        
        result["listening"] = True
        
        # Get health configuration if app_name provided
        health_config = None
        health_endpoint = None
        health_timeout = 1  # Default 1 second
        
        if app_name:
            health_config = self.get_health_config(app_name)
            if health_config:
                # Get endpoint path from health config
                endpoints = health_config.get("endpoints", {})
                if port_type == "api":
                    health_endpoint = endpoints.get("api", "/health")
                elif port_type == "ui":
                    health_endpoint = endpoints.get("ui", "/")
                elif port_type == "metrics":
                    health_endpoint = endpoints.get("metrics", "/metrics")
                else:
                    health_endpoint = "/"
                
                # Get timeout in seconds (convert from milliseconds)
                health_timeout = health_config.get("timeout", 5000) / 1000.0
        
        # Use defaults if no config
        if not health_endpoint:
            if port_type == "api":
                health_endpoint = "/health"
            else:  # UI or other
                health_endpoint = "/"
        
        # Try HTTP health check
        try:
            conn = http.client.HTTPConnection("localhost", port, timeout=health_timeout)
            conn.request("GET", health_endpoint)
            response = conn.getresponse()
            result["responding"] = True
            result["status_code"] = response.status
            conn.close()
            
        except (socket.timeout, socket.error, http.client.HTTPException):
            result["responding"] = False
        
        return result
    
    def get_port_environment(self, app: 'EnhancedApp') -> Dict[str, str]:
        """Get environment variables for port substitution"""
        port_env = {}
        for port_type, port_config in app.allocated_ports.items():
            if isinstance(port_config, dict):
                # New format with env_var from service.json
                env_var = port_config.get('env_var', f"{port_type.upper()}_PORT")
                port_num = port_config.get('port')
                if env_var and port_num:
                    port_env[env_var] = str(port_num)
            else:
                # Legacy format (just the port number)
                port_env[f"{port_type.upper()}_PORT"] = str(port_config)
        return port_env
    
    def load_resource_ports(self) -> Dict[str, str]:
        """Load RESOURCE_PORTS mapping from port_registry.sh"""
        resource_ports = {}
        port_registry_path = self.vrooli_root / "scripts" / "resources" / "port_registry.sh"
        
        if not port_registry_path.exists():
            self.logger.warning(f"Port registry not found at {port_registry_path}")
            return resource_ports
        
        try:
            # Execute the port registry script to extract RESOURCE_PORTS array
            import subprocess
            
            script = f"""
            source {port_registry_path}
            # Print all RESOURCE_PORTS entries
            for key in "${{!RESOURCE_PORTS[@]}}"; do
                echo "$key=${{RESOURCE_PORTS[$key]}}"
            done
            """
            
            result = subprocess.run(
                ["bash", "-c", script],
                capture_output=True,
                text=True,
                cwd=str(self.vrooli_root)
            )
            
            if result.returncode == 0:
                for line in result.stdout.strip().split('\n'):
                    if '=' in line and line.strip():
                        key, value = line.split('=', 1)
                        resource_ports[key] = value
                        
                self.logger.debug(f"Loaded {len(resource_ports)} resource ports")
            else:
                self.logger.warning(f"Failed to load port registry: {result.stderr}")
                
        except Exception as e:
            self.logger.warning(f"Failed to load port registry: {e}")
        
        return resource_ports
    
    def expand_resource_ports(self, content: str) -> str:
        """Expand ${RESOURCE_PORTS[...]} syntax in service.json content"""
        import re
        
        def replace_match(match):
            resource_name = match.group(1)
            if resource_name in self.resource_ports:
                return self.resource_ports[resource_name]
            else:
                self.logger.warning(f"Unknown resource port: {resource_name}")
                return match.group(0)  # Return original if not found
        
        # Replace ${RESOURCE_PORTS[resource_name]} with actual port
        expanded = re.sub(r'\$\{RESOURCE_PORTS\[([^}]+)\]\}', replace_match, content)
        return expanded
    
    def load_service_json(self, config_path: Path) -> dict:
        """Load and preprocess service.json with RESOURCE_PORTS expansion"""
        try:
            with open(config_path, 'r') as f:
                raw_content = f.read()
            
            # Expand RESOURCE_PORTS references
            expanded_content = self.expand_resource_ports(raw_content)
            
            # Parse JSON
            import json
            config = json.loads(expanded_content)
            return config
            
        except Exception as e:
            self.logger.error(f"Failed to load service.json from {config_path}: {e}")
            return {}
    
    def _check_http(self, target: str, timeout_ms: int = 5000, method: str = "GET", expected_status: List[int] = None, app_name: str = None) -> Tuple[str, str]:
        """Perform HTTP health check"""
        import re
        import urllib.parse
        
        if expected_status is None:
            expected_status = [200, 204]
            
        try:
            # Substitute port variables if app_name is provided
            if app_name and app_name in self.apps:
                port_env = self.get_port_environment(self.apps[app_name])
                for env_var, port_value in port_env.items():
                    target = target.replace(f"${{{env_var}}}", port_value)
                    target = target.replace(f"${env_var}", port_value)
            
            # Parse URL and substitute any remaining environment variables
            target = os.path.expandvars(target)
            parsed = urllib.parse.urlparse(target)
            
            # Extract host and port
            host = parsed.hostname or "localhost"
            port = parsed.port or (443 if parsed.scheme == "https" else 80)
            path = parsed.path or "/"
            
            # Make HTTP request
            timeout = timeout_ms / 1000.0
            conn = http.client.HTTPConnection(host, port, timeout=timeout)
            conn.request(method, path)
            response = conn.getresponse()
            status_code = response.status
            conn.close()
            
            if status_code in expected_status:
                return "passed", f"HTTP {method} {target} returned {status_code}"
            else:
                return "failed", f"HTTP {method} {target} returned {status_code}, expected {expected_status}"
                
        except Exception as e:
            return "failed", f"HTTP check failed: {str(e)}"
    
    def _check_tcp(self, target: str, timeout_ms: int = 3000) -> Tuple[str, str]:
        """Perform TCP port check"""
        import re
        
        try:
            # Parse target (format: "host:port" or just "port")
            target = os.path.expandvars(target)
            
            if ":" in target:
                host, port_str = target.rsplit(":", 1)
            else:
                host = "localhost"
                port_str = target
                
            port = int(port_str)
            timeout = timeout_ms / 1000.0
            
            # Try to connect
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.settimeout(timeout)
            result = sock.connect_ex((host, port))
            sock.close()
            
            if result == 0:
                return "passed", f"TCP port {host}:{port} is open"
            else:
                return "failed", f"TCP port {host}:{port} is not reachable"
                
        except Exception as e:
            return "failed", f"TCP check failed: {str(e)}"
    
    def _check_process(self, target: str, match: str = "exact") -> Tuple[str, str]:
        """Perform process check"""
        try:
            if match == "exact":
                cmd = ["pgrep", "-x", target]
            elif match == "contains":
                cmd = ["pgrep", "-f", target]
            else:  # regex
                cmd = ["pgrep", target]
                
            result = subprocess.run(cmd, capture_output=True, text=True)
            
            if result.returncode == 0:
                pids = result.stdout.strip().split("\n")
                return "passed", f"Process '{target}' is running (PIDs: {', '.join(pids)})"
            else:
                return "failed", f"Process '{target}' is not running"
                
        except Exception as e:
            return "failed", f"Process check failed: {str(e)}"
    
    def _check_file(self, target: str, check: str = "exists") -> Tuple[str, str]:
        """Perform file/directory check"""
        try:
            # Expand path relative to scenario directory
            if not os.path.isabs(target):
                # Find the scenario directory for this app
                for app_name, app in self.apps.items():
                    if app.is_running():
                        scenario_dir = self.scenarios_dir / app_name
                        target = str(scenario_dir / target)
                        break
                        
            path = Path(target)
            
            if check == "exists":
                if path.exists():
                    return "passed", f"Path '{target}' exists"
                else:
                    return "failed", f"Path '{target}' does not exist"
            elif check == "readable":
                if path.exists() and os.access(path, os.R_OK):
                    return "passed", f"Path '{target}' is readable"
                else:
                    return "failed", f"Path '{target}' is not readable"
            elif check == "executable":
                if path.exists() and os.access(path, os.X_OK):
                    return "passed", f"Path '{target}' is executable"
                else:
                    return "failed", f"Path '{target}' is not executable"
            else:
                return "failed", f"Unknown file check type: {check}"
                
        except Exception as e:
            return "failed", f"File check failed: {str(e)}"
    
    def _check_resource(self, resource: str, schema: str = None, timeout_ms: int = 5000) -> Tuple[str, str]:
        """Perform resource health check using vrooli resource status --json"""
        try:
            # Call vrooli resource status with JSON output
            cmd = ["vrooli", "resource", "status", resource, "--json"]
            timeout = timeout_ms / 1000.0
            
            result = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                timeout=timeout,
                cwd=str(self.project_root)
            )
            
            if result.returncode != 0:
                return "failed", f"Resource '{resource}' status check failed: {result.stderr}"
            
            try:
                data = json.loads(result.stdout)
            except json.JSONDecodeError:
                return "failed", f"Invalid JSON response from resource status"
            
            # Check overall resource status using 'running' field
            resource_running = data.get("running", False)
            if not resource_running:
                return "failed", f"Resource '{resource}' is not running"
            
            # Check overall health using 'healthy' field
            resource_healthy = data.get("healthy", False)
            if not resource_healthy:
                return "failed", f"Resource '{resource}' is unhealthy"
            
            # If schema specified, check schema-specific health
            if schema:
                schemas = data.get("schemas", {})
                if schema in schemas:
                    schema_info = schemas[schema]
                    if schema_info.get("healthy", False):
                        return "passed", f"Resource '{resource}' schema '{schema}' is healthy"
                    else:
                        return "failed", f"Resource '{resource}' schema '{schema}' is unhealthy"
                else:
                    return "failed", f"Schema '{schema}' not found in resource '{resource}'"
            
            return "passed", f"Resource '{resource}' is healthy"
            
        except subprocess.TimeoutExpired:
            return "failed", f"Resource status check timed out after {timeout_ms}ms"
        except Exception as e:
            return "failed", f"Resource check failed: {str(e)}"
    
    def _check_command(self, run: str, expected_exit: int = 0, timeout_ms: int = 3000) -> Tuple[str, str]:
        """Perform custom command check"""
        try:
            # Expand environment variables
            run = os.path.expandvars(run)
            timeout = timeout_ms / 1000.0
            
            result = subprocess.run(
                run,
                shell=True,
                capture_output=True,
                text=True,
                timeout=timeout,
                cwd=str(self.project_root)
            )
            
            if result.returncode == expected_exit:
                return "passed", f"Command exited with expected code {expected_exit}"
            else:
                return "failed", f"Command exited with {result.returncode}, expected {expected_exit}"
                
        except subprocess.TimeoutExpired:
            return "failed", f"Command timed out after {timeout_ms}ms"
        except Exception as e:
            return "failed", f"Command check failed: {str(e)}"
    
    def perform_health_checks(self, app_name: str) -> List[Dict[str, Any]]:
        """Perform comprehensive health checks based on configuration"""
        health_config = self.get_health_config(app_name)
        check_results = []
        
        if health_config and "checks" in health_config:
            for check in health_config.get("checks", []):
                check_type = check.get("type")
                check_name = check.get("name", f"{check_type}_check")
                timeout = check.get("timeout", 5000)
                
                result = {
                    "name": check_name,
                    "type": check_type,
                    "critical": check.get("critical", True),
                    "status": "unknown",
                    "message": ""
                }
                
                # Perform the appropriate check based on type
                try:
                    if check_type == "http":
                        target = check.get("target", "")
                        method = check.get("method", "GET")
                        expected_status = check.get("expected_status", [200, 204])
                        status, message = self._check_http(target, timeout, method, expected_status, app_name)
                        
                    elif check_type == "tcp":
                        target = check.get("target", "")
                        status, message = self._check_tcp(target, timeout)
                        
                    elif check_type == "process":
                        target = check.get("target", "")
                        match = check.get("match", "exact")
                        status, message = self._check_process(target, match)
                        
                    elif check_type == "file":
                        target = check.get("target", "")
                        check_kind = check.get("check", "exists")
                        status, message = self._check_file(target, check_kind)
                        
                    elif check_type == "resource":
                        resource = check.get("resource", check.get("target", ""))
                        schema = check.get("schema")
                        status, message = self._check_resource(resource, schema, timeout)
                        
                    elif check_type in ["postgres", "redis", "qdrant", "n8n", "ollama"]:
                        # Resource-specific health check - just check if resource is healthy
                        # Ignore target/schema for these checks as they're not supported yet
                        status, message = self._check_resource(check_type, None, timeout)
                        
                    elif check_type == "command":
                        run = check.get("run", "")
                        expected_exit = check.get("expected_exit", 0)
                        status, message = self._check_command(run, expected_exit, timeout)
                        
                    else:
                        status = "skipped"
                        message = f"Unknown check type: {check_type}"
                    
                    result["status"] = status
                    result["message"] = message
                    
                except Exception as e:
                    result["status"] = "error"
                    result["message"] = f"Check failed with error: {str(e)}"
                
                check_results.append(result)
        
        return check_results
    
    def get_app_status(self, app_name: str) -> Dict[str, Any]:
        """Get detailed status of a specific app"""
        if app_name not in self.apps:
            raise HTTPException(status_code=404, detail=f"App '{app_name}' not found")
        
        app = self.apps[app_name]
        
        # Update status based on actual process state
        is_running = app.is_running()
        if is_running and app.status != "running":
            # App is running but status says stopped - update it
            app.status = "running"
            if not app.started_at:
                app.started_at = datetime.now()
        elif not is_running and app.status == "running":
            # App is not running but status says running - update it
            app.status = "stopped"
            app.stopped_at = datetime.now()
            app.pid = None
            app.process = None
        
        status = app.to_dict()
        
        # Add enhanced port status with health checks
        port_status = {}
        actual_health = "healthy"  # Assume healthy unless proven otherwise
        total_ports = len(app.allocated_ports)
        listening_ports = 0
        responding_ports = 0
        
        for port_type, port_config in app.allocated_ports.items():
            # Handle both old and new format
            port_num = port_config.get('port') if isinstance(port_config, dict) else port_config
            health_check = self.check_service_health(port_num, port_type, app_name)
            port_status[port_type] = health_check
            
            # Count port statuses
            if health_check["listening"]:
                listening_ports += 1
                if health_check["responding"]:
                    responding_ports += 1
        
        # Determine overall health based on port statuses
        if app.status == "running" and total_ports > 0:
            if responding_ports == total_ports:
                # All ports responding
                actual_health = "healthy"
            elif responding_ports > 0:
                # Some ports responding
                actual_health = "degraded"
                app.status = "degraded"  # Update app status to match health
            elif listening_ports > 0:
                # Ports bound but not responding
                actual_health = "unhealthy"
                app.status = "degraded"  # Update app status to match health
            else:
                # No ports bound
                actual_health = "degraded"
                app.status = "degraded"  # Update app status to match health
        elif app.status == "running" and total_ports == 0:
            # App running but no ports configured - assume healthy
            actual_health = "healthy"
        
        status["port_status"] = port_status
        status["actual_health"] = actual_health
        
        # Add comprehensive health check results if available
        health_config = self.get_health_config(app_name)
        if health_config:
            status["health_checks"] = self.perform_health_checks(app_name)
            status["health_config"] = {
                "endpoints": health_config.get("endpoints", {}),
                "timeout": health_config.get("timeout", 5000),
                "interval": health_config.get("interval", 30000)
            }
        
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
        
        @self.api_app.get("/apps/{app_name}/health")
        async def app_health(app_name: str):
            """Get health status and configuration of specific app"""
            if app_name not in self.apps:
                raise HTTPException(status_code=404, detail=f"App '{app_name}' not found")
            
            status = self.get_app_status(app_name)
            return {
                "name": app_name,
                "status": status.get("status"),
                "health": status.get("actual_health"),
                "health_checks": status.get("health_checks", []),
                "health_config": status.get("health_config", {}),
                "port_status": status.get("port_status", {})
            }
        
        @self.api_app.get("/health/configs")
        async def get_all_health_configs():
            """Get health configurations for all apps"""
            configs = {}
            for app_name in self.apps.keys():
                health_config = self.get_health_config(app_name)
                if health_config:
                    configs[app_name] = health_config
            return configs
        
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
            """Start all enabled apps using rolling window concurrency"""
            results = []
            enabled_apps = [(name, app) for name, app in self.apps.items() if app.enabled]
            
            # Use semaphore for rolling window concurrency
            semaphore = asyncio.Semaphore(MAX_CONCURRENT_STARTS)
            
            async def start_with_semaphore(name):
                async with semaphore:
                    try:
                        self.logger.info(f"Starting {name}...")
                        success = await self.start_app(name)
                        result = {"app": name, "success": success}
                        if success:
                            self.logger.info(f"✓ {name} started successfully")
                        else:
                            self.logger.warning(f"✗ {name} failed to start")
                        return result
                    except Exception as e:
                        self.logger.error(f"✗ {name} error: {str(e)}")
                        return {"app": name, "success": False, "error": str(e)}
            
            # Start all apps with rolling window concurrency
            tasks = [start_with_semaphore(app_name) for app_name, _ in enabled_apps]
            results = await asyncio.gather(*tasks)
            
            successful = sum(1 for r in results if r["success"])
            self.logger.info(f"Completed: {successful}/{len(results)} apps started successfully")
            
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


# Legacy functions kept for compatibility but now use the new lock manager
def check_existing_orchestrator():
    """Check if another orchestrator instance is already running"""
    manager = OrchestrationLockManager(port=API_PORT)
    existing = manager._find_running_orchestrator()
    if existing:
        return existing.get('pid', -1)
    return None

def write_pid_file():
    """Compatibility function - actual lock management is handled by OrchestrationLockManager"""
    pass  # Now handled by lock manager

def cleanup_pid_file():
    """Compatibility function - cleanup is handled by OrchestrationLockManager"""
    pass  # Now handled by lock manager

async def main():
    """Main entry point for enhanced orchestrator"""
    import argparse
    import atexit
    
    parser = argparse.ArgumentParser(description="Enhanced Vrooli App Orchestrator")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose logging")
    parser.add_argument("--fast", action="store_true", help="Enable fast mode")
    parser.add_argument("--background", action="store_true", help="Run in background")
    parser.add_argument("--start-all", action="store_true", help="Start all enabled apps")
    parser.add_argument("--force", action="store_true", help="Force start even if another instance exists")
    
    args = parser.parse_args()
    
    # Use the new lock manager with robust PID handling
    lock_manager = OrchestrationLockManager(port=API_PORT)
    
    # Try to acquire lock (handles all edge cases internally)
    if not lock_manager.acquire_lock(force=args.force):
        # Lock manager already printed appropriate error messages
        sys.exit(1)
    
    # Register cleanup on exit - ensures PID files are always cleaned up
    def cleanup():
        try:
            lock_manager.release_lock()
        except Exception:
            pass  # Ignore cleanup errors on exit
    
    atexit.register(cleanup)
    
    # Register signal handlers for graceful shutdown
    def signal_handler(signum, frame):
        cleanup()
        if signum == signal.SIGINT:
            sys.exit(130)  # Standard exit code for Ctrl+C
        else:
            sys.exit(0)
    
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    
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
            print(f"Starting all enabled apps (max {MAX_CONCURRENT_STARTS} concurrent)...")
            enabled_apps = [(name, app) for name, app in orchestrator.apps.items() if app.enabled]
            print(f"  Total scenarios to start: {len(enabled_apps)}")
            
            # Use semaphore for rolling window concurrency
            semaphore = asyncio.Semaphore(MAX_CONCURRENT_STARTS)
            start_times = {}
            
            async def start_with_semaphore(app_name):
                async with semaphore:
                    start_time = time.time()
                    start_times[app_name] = start_time
                    try:
                        print(f"  ⟳ Starting {app_name}...")
                        success = await orchestrator.start_app(app_name)
                        elapsed = time.time() - start_time
                        if success:
                            print(f"  ✓ {app_name} started ({elapsed:.1f}s)")
                        else:
                            print(f"  ✗ {app_name} failed ({elapsed:.1f}s)")
                        return app_name, success
                    except Exception as e:
                        elapsed = time.time() - start_time
                        print(f"  ✗ {app_name} error: {str(e)} ({elapsed:.1f}s)")
                        return app_name, False
            
            # Start all apps with rolling window concurrency
            tasks = [start_with_semaphore(app_name) for app_name, _ in enabled_apps]
            results = await asyncio.gather(*tasks)
            
            # Summary
            successful = sum(1 for _, success in results if success)
            print(f"\nCompleted: {successful}/{len(results)} scenarios started successfully")
        
        # Summary
        total_apps = len(orchestrator.apps)
        running_apps = len([a for a in orchestrator.apps.values() if a.is_running()])
        enabled_apps = len([a for a in orchestrator.apps.values() if a.enabled])
        
        print(f"\n{'=' * 60}")
        print(f"Enhanced Orchestrator Running:")
        print(f"  API Server: http://localhost:{API_PORT}")
        print(f"  Apps: {running_apps}/{total_apps} running ({enabled_apps} enabled)")
        print(f"  Management: Use API or CLI commands")
        print(f"  Auto-detection: Checking for externally started apps every 10s")
        print(f"{'=' * 60}\n")
        
        print("Orchestrator running. Press Ctrl+C to stop all apps and exit.")
        
        # Periodic detection loop - check for externally started apps
        detection_interval = 10  # seconds
        last_detection_time = time.time()
        
        while True:
            try:
                # Wait for the detection interval
                await asyncio.sleep(detection_interval)
                
                # Detect externally started apps (silent unless changes detected)
                status_changes = orchestrator.detect_running_apps(verbose_log=False)
                
                # Log any status changes
                if status_changes:
                    for app_name, old_status, new_status in status_changes:
                        if new_status == "running":
                            orchestrator.logger.info(f"✓ Detected {app_name} started externally (status: {old_status} → {new_status})")
                        else:
                            orchestrator.logger.info(f"✗ Detected {app_name} stopped (status: {old_status} → {new_status})")
                    
                    # Update running count
                    running_apps = len([a for a in orchestrator.apps.values() if a.is_running()])
                    orchestrator.logger.debug(f"Updated app count: {running_apps}/{total_apps} running")
                
            except asyncio.CancelledError:
                break
            except Exception as e:
                orchestrator.logger.error(f"Error in detection loop: {e}")
                # Continue loop even if detection fails
        
    except KeyboardInterrupt:
        await orchestrator.shutdown()
        cleanup_pid_file()
        sys.exit(130)
    except Exception as e:
        orchestrator.logger.error(f"Fatal error: {e}")
        await orchestrator.shutdown()
        cleanup_pid_file()
        sys.exit(1)
    finally:
        # Cleanup is handled by atexit and signal handlers
        pass


if __name__ == "__main__":
    asyncio.run(main())