#!/usr/bin/env python3
"""
Enhanced Vrooli App Orchestrator - Clean Architecture
Single source of truth for scenario lifecycle management with robust port allocation.
"""

import asyncio
import json
import logging
import logging.handlers
import os
import psutil
import signal
import subprocess
import sys
import time
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from pathlib import Path
from typing import Dict, List, Optional, Set, Any, Tuple

# FastAPI imports
try:
    from fastapi import FastAPI, HTTPException, BackgroundTasks
    from fastapi.responses import JSONResponse
    import uvicorn
except ImportError:
    print("ERROR: FastAPI and uvicorn are required", file=sys.stderr)
    print("Install with: pip3 install fastapi uvicorn", file=sys.stderr)
    sys.exit(1)

# Import existing safety mechanisms
from orchestrator_utils import (
    ForkBombDetector, MAX_APPS, MAX_CONCURRENT_STARTS,
    ORCHESTRATOR_LOCK_FILE, ORCHESTRATOR_PID_FILE
)
from pid_manager import OrchestrationLockManager

# API Configuration
API_PORT = int(os.environ.get('ORCHESTRATOR_PORT', '9500'))
API_HOST = "0.0.0.0"

@dataclass
class ProcessInfo:
    """Information about a running process"""
    pid: int
    status: str = "running"  # running, failed, stopped
    log_file: Optional[Path] = None

@dataclass
class ScenarioState:
    """Complete state of a scenario"""
    scenario_name: str
    phase: str = "develop"  # develop, test, etc.
    processes: Dict[str, ProcessInfo] = field(default_factory=dict)  # step_name -> ProcessInfo
    status: str = "stopped"  # stopped, starting, running, stopping, error
    started_at: Optional[datetime] = None
    config_path: Optional[Path] = None

class ScenarioDiscovery:
    """Discovers all running vrooli scenarios"""
    
    def __init__(self):
        self.process_base = Path.home() / ".vrooli/processes/scenarios"
        self.logger = logging.getLogger(__name__)
    
    def discover_running_scenarios(self) -> Dict[str, ScenarioState]:
        """
        Discover all running scenarios by scanning process manager directories.
        """
        scenarios = {}
        
        if not self.process_base.exists():
            return scenarios
        
        # Phase 1: Collect all scenario info and PIDs
        pid_to_info = {}  # pid -> (scenario_name, step_name, process_name)
        
        for scenario_dir in self.process_base.iterdir():
            if not scenario_dir.is_dir():
                continue
                
            scenario_name = scenario_dir.name
            
            # Find all processes for this scenario
            for process_dir in scenario_dir.iterdir():
                if not process_dir.is_dir():
                    continue
                    
                pid_file = process_dir / "pid"
                if not pid_file.exists():
                    continue
                    
                try:
                    pid = int(pid_file.read_text().strip())
                    # Quick check if process exists (no lsof yet)
                    if self._is_process_running(pid):
                        process_name = process_dir.name
                        if process_name.startswith("vrooli."):
                            parts = process_name.split('.')
                            if len(parts) >= 4:
                                step_name = parts[-1]  # Last part is step name
                                pid_to_info[pid] = (scenario_name, step_name, process_name)
                except (ValueError, OSError) as e:
                    self.logger.debug(f"Skipping PID file {pid_file}: {e}")
                    continue
        
        # Phase 2: Build scenario states from collected data
            for pid, (scenario_name, step_name, process_name) in pid_to_info.items():
                if scenario_name not in scenarios:
                    scenarios[scenario_name] = ScenarioState(
                        scenario_name=scenario_name,
                        status="running"
                    )
                
                
                # Get process start time
                process_start_time = self._get_process_start_time(pid)
                
                # Update scenario's started_at to the earliest process start time
                if process_start_time:
                    if scenarios[scenario_name].started_at is None or process_start_time < scenarios[scenario_name].started_at:
                        scenarios[scenario_name].started_at = process_start_time
                
                # Set log file path
                log_file = Path.home() / ".vrooli/logs/scenarios" / scenario_name / f"{process_name}.log"
                
                scenarios[scenario_name].processes[step_name] = ProcessInfo(
                    pid=pid,
                    status="running",
                    log_file=log_file if log_file.exists() else None
                )
                
                
        return scenarios

    def _is_process_running(self, pid: int) -> bool:
        """Check if process is running"""
        try:
            os.kill(pid, 0)  # Signal 0 just checks if process exists
            return True
        except OSError:
            return False
    
    def _get_process_start_time(self, pid: int) -> Optional[datetime]:
        """Get the start time of a process"""
        try:
            process = psutil.Process(pid)
            # psutil returns timestamp in seconds since epoch
            return datetime.fromtimestamp(process.create_time())
        except (psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
            # Fallback to using stat time of the pid file as approximation
            try:
                # Try to get from /proc/<pid>/stat if psutil fails
                with open(f'/proc/{pid}/stat', 'r') as f:
                    fields = f.read().split(')')
                    if len(fields) >= 2:
                        # Field 22 (0-indexed: 21) after the ) is start time in clock ticks
                        stats = fields[1].split()
                        if len(stats) >= 20:
                            start_ticks = int(stats[19])
                            # Get system boot time
                            with open('/proc/stat', 'r') as stat_file:
                                for line in stat_file:
                                    if line.startswith('btime'):
                                        boot_time = int(line.split()[1])
                                        # Calculate actual start time
                                        clock_ticks = os.sysconf(os.sysconf_names['SC_CLK_TCK'])
                                        start_time = boot_time + (start_ticks / clock_ticks)
                                        return datetime.fromtimestamp(start_time)
            except (OSError, ValueError, IndexError):
                pass
            return None


# PortAllocator class removed - replaced by unified get_scenario_environment function

class EnhancedOrchestrator:
    """Single source of truth for scenario lifecycle management"""
    
    def __init__(self):
        self.setup_logging()
        self.discovery = ScenarioDiscovery()
        self.scenarios: Dict[str, ScenarioState] = {}
        self.fork_bomb_detector = ForkBombDetector()
        self.lock_manager = OrchestrationLockManager(API_PORT)
        self.start_time = time.time()  # Track when orchestrator started
        # Path to Vrooli root
        self.vrooli_root = Path(os.environ.get('VROOLI_ROOT', Path.home() / 'Vrooli'))
        
        # Performance optimization: Add caching
        self.last_discovery_time = 0
        self.discovery_cache_duration = 10  # Cache for 10 seconds
        self.discovery_in_progress = False
        self.refresh_lock = asyncio.Lock()
        
        # Setup FastAPI
        self.app = FastAPI(title="Vrooli Scenario Orchestrator", version="2.0.0")
        self.setup_routes()
        
        # Always refresh state on startup
        self.refresh_state()
        
    def setup_logging(self):
        """Configure logging with file output"""
        # Create logs directory if it doesn't exist
        log_dir = Path.home() / ".vrooli/logs"
        log_dir.mkdir(parents=True, exist_ok=True)
        log_file = log_dir / "orchestrator.log"
        
        # Configure root logger
        logging.basicConfig(
            level=logging.INFO,
            format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
            handlers=[
                logging.StreamHandler(sys.stderr),  # Console output
                logging.handlers.RotatingFileHandler(
                    log_file,
                    maxBytes=10*1024*1024,  # 10MB
                    backupCount=5
                )
            ]
        )
        self.logger = logging.getLogger(__name__)
        self.logger.info(f"Orchestrator logging initialized to {log_file}")
        
    def refresh_state(self, force: bool = False):
        """Rediscover all running scenarios - single source of truth with caching"""
        current_time = time.time()
        
        # Use cache if data is fresh enough and not forced
        if (not force and 
            current_time - self.last_discovery_time < self.discovery_cache_duration and 
            self.scenarios):
            self.logger.debug(f"Using cached scenario data ({len(self.scenarios)} scenarios)")
            return
            
        # Prevent concurrent discoveries  
        if self.discovery_in_progress:
            self.logger.debug("Discovery already in progress, skipping")
            return
            
        self.discovery_in_progress = True
        try:
            self.logger.info("Refreshing scenario state from system...")
            self.scenarios = self.discovery.discover_running_scenarios()
            self.last_discovery_time = current_time
            self.logger.info(f"Discovered {len(self.scenarios)} running scenarios")
        finally:
            self.discovery_in_progress = False
        
    def _calculate_runtime(self, started_at: Optional[datetime]) -> Optional[str]:
        """Calculate runtime from start time to now"""
        if not started_at:
            return None
        
        runtime = datetime.now() - started_at
        days = runtime.days
        hours, remainder = divmod(runtime.seconds, 3600)
        minutes, seconds = divmod(remainder, 60)
        
        if days > 0:
            return f"{days}d {hours}h {minutes}m"
        elif hours > 0:
            return f"{hours}h {minutes}m {seconds}s"
        elif minutes > 0:
            return f"{minutes}m {seconds}s"
        else:
            return f"{seconds}s"
    
    def setup_routes(self):
        """Setup FastAPI routes"""
        
        @self.app.get("/health")
        async def health_check():
            return {"status": "healthy", "scenarios": len(self.scenarios)}
        
        @self.app.get("/status")
        async def status():
            """System status endpoint (legacy compatibility)"""
            self.refresh_state()
            
            # Count scenarios by status
            running = len([s for s in self.scenarios.values() if s.status == "running"])
            stopped = len([s for s in self.scenarios.values() if s.status == "stopped"])
            error = len([s for s in self.scenarios.values() if s.status == "error"])
            
            # Get system load info
            load_info = self.fork_bomb_detector.get_system_load() if hasattr(self, 'fork_bomb_detector') else {}
            
            return {
                "status": "healthy",
                "scenarios": {
                    "total": len(self.scenarios),
                    "running": running,
                    "stopped": stopped,
                    "error": error
                },
                "system": load_info,
                "orchestrator": {
                    "pid": os.getpid(),
                    "port": API_PORT,
                    "uptime": time.time() - self.start_time if hasattr(self, 'start_time') else None
                }
            }
            
        @self.app.get("/scenarios")
        async def list_scenarios():
            """List all scenarios with their current state"""
            self.refresh_state()  # Use cached data when possible
            
            result = []
            for scenario_name, scenario_state in self.scenarios.items():
                result.append({
                    "name": scenario_name,
                    "status": scenario_state.status,
                    "processes": len(scenario_state.processes),
                    "started_at": scenario_state.started_at.isoformat() if scenario_state.started_at else None,
                    "runtime": self._calculate_runtime(scenario_state.started_at)
                })
                
            return {"success": True, "data": result}
            
        @self.app.get("/scenarios/{scenario_name}")
        async def get_scenario_status(scenario_name: str):
            """Get detailed status for a specific scenario"""
            self.refresh_state()  # Use cached data when possible
            
            if scenario_name not in self.scenarios:
                raise HTTPException(status_code=404, detail=f"Scenario '{scenario_name}' not found or not running")
                
            scenario = self.scenarios[scenario_name]
            
            # Build detailed response
            processes = []
            for step_name, process_info in scenario.processes.items():
                processes.append({
                    "step_name": step_name,
                    "pid": process_info.pid,
                    "status": process_info.status,
                    "ports": process_info.ports,
                    "log_file": str(process_info.log_file) if process_info.log_file else None
                })
                
            return {
                "success": True,
                "data": {
                    "name": scenario_name,
                    "status": scenario.status,
                    "phase": scenario.phase,
                    "processes": processes,
                    "started_at": scenario.started_at.isoformat() if scenario.started_at else None,
                    "runtime": self._calculate_runtime(scenario.started_at)
                }
            }
            
        @self.app.post("/scenarios/{scenario_name}/start")
        async def start_scenario(scenario_name: str):
            """Start a scenario with proper port allocation"""
            try:
                # Check fork bomb protection
                if not self.fork_bomb_detector.record_start():
                    raise HTTPException(
                        status_code=429, 
                        detail="System overload detected - too many processes starting"
                    )
                
                # Force refresh state before starting
                self.refresh_state(force=True)
                
                # Load scenario configuration and get service.json path
                config, service_json_path = self._load_scenario_config_with_path(scenario_name)
                if not config:
                    raise HTTPException(
                        status_code=404, 
                        detail=f"Scenario configuration not found: {scenario_name}"
                    )
                
                # NEW: Start scenario WITHOUT pre-allocating ports
                # Let lifecycle.sh handle port allocation on-demand
                result = await self._execute_lifecycle(scenario_name, 'develop', {})
                
                if not result:
                    raise HTTPException(status_code=500, detail=f"Failed to start scenario: {scenario_name}")
                
                # Force refresh state to pick up new processes immediately
                self.refresh_state(force=True)
                
                return {
                    "success": True,
                    "message": f"Scenario '{scenario_name}' started successfully - ports allocated on-demand"
                }
                
            except Exception as e:
                self.logger.error(f"Failed to start scenario {scenario_name}: {e}")
                raise HTTPException(status_code=500, detail=str(e))
                
        @self.app.post("/scenarios/start-all")
        async def start_all_scenarios(background_tasks: BackgroundTasks):
            """Start all scenarios found in the scenarios directory"""
            try:
                # Check fork bomb protection
                if not self.fork_bomb_detector.record_start():
                    raise HTTPException(
                        status_code=429, 
                        detail="System overload detected - too many processes starting"
                    )
                
                # Find all available scenarios
                vrooli_root = Path(os.environ.get('VROOLI_ROOT', Path.home() / 'Vrooli'))
                scenarios_dir = vrooli_root / 'scenarios'
                
                if not scenarios_dir.exists():
                    raise HTTPException(status_code=404, detail="Scenarios directory not found")
                
                started = []
                failed = []
                
                # Start each scenario
                for scenario_dir in scenarios_dir.iterdir():
                    if not scenario_dir.is_dir():
                        continue
                        
                    scenario_name = scenario_dir.name
                    
                    # Skip meta-scenarios and templates
                    if scenario_name.startswith('.') or scenario_name == 'templates':
                        continue
                    
                    # Check if service.json exists
                    service_json = scenario_dir / '.vrooli/service.json'
                    if not service_json.exists():
                        service_json = scenario_dir / 'service.json'
                    
                    if not service_json.exists():
                        self.logger.debug(f"Skipping {scenario_name}: no service.json")
                        continue
                    
                    try:
                        # Load configuration and get service.json path
                        config, service_json_path = self._load_scenario_config_with_path(scenario_name)
                        if not config:
                            failed.append({"name": scenario_name, "error": "Configuration not found"})
                            continue
                        
                        # NEW: Start scenario WITHOUT pre-allocating ports
                        # Let lifecycle.sh handle port allocation on-demand
                        result = await self._execute_lifecycle(scenario_name, 'develop', {})
                        
                        if result:
                            started.append({
                                "name": scenario_name, 
                                "message": "Started - ports allocated on-demand"
                            })
                        else:
                            failed.append({"name": scenario_name, "error": "Lifecycle execution failed"})
                            
                    except Exception as e:
                        self.logger.error(f"Failed to start {scenario_name}: {e}")
                        failed.append({"name": scenario_name, "error": str(e)})
                
                # Refresh state after bulk start
                self.refresh_state(force=True)
                
                return {
                    "success": True,
                    "started": started,
                    "failed": failed,
                    "message": f"Started {len(started)} scenarios, {len(failed)} failed"
                }
                
            except HTTPException:
                raise
            except Exception as e:
                self.logger.error(f"Failed to start all scenarios: {e}")
                raise HTTPException(status_code=500, detail=str(e))
        
        @self.app.post("/scenarios/{scenario_name}/stop")
        async def stop_scenario(scenario_name: str, background_tasks: BackgroundTasks):
            """Stop a scenario and all its processes (async to prevent timeout)"""
            try:
                self.refresh_state()
                
                if scenario_name not in self.scenarios:
                    raise HTTPException(status_code=404, detail=f"Scenario '{scenario_name}' not running")
                
                scenario = self.scenarios[scenario_name]
                
                # Schedule the actual stopping in the background to prevent timeout
                async def do_stop():
                    stopped_processes = []
                    
                    # Stop all processes for this scenario
                    for step_name, process_info in scenario.processes.items():
                        try:
                            # Try graceful shutdown first
                            os.kill(process_info.pid, signal.SIGTERM)
                            stopped_processes.append(step_name)
                            
                            # Check if process stopped (with short timeout)
                            for _ in range(5):  # Wait up to 0.5 seconds
                                await asyncio.sleep(0.1)
                                if not self.discovery._is_process_running(process_info.pid):
                                    break
                            else:
                                # Force kill if still running
                                try:
                                    os.kill(process_info.pid, signal.SIGKILL)
                                except ProcessLookupError:
                                    pass  # Already dead
                                    
                        except ProcessLookupError:
                            # Process already dead
                            stopped_processes.append(step_name)
                        except Exception as e:
                            self.logger.warning(f"Failed to stop process {step_name} (PID {process_info.pid}): {e}")
                    
                    # Refresh state after stopping
                    self.refresh_state()
                    
                    # Port deallocation not needed - lifecycle.sh handles cleanup
                    
                    self.logger.info(f"Successfully stopped scenario '{scenario_name}' with {len(stopped_processes)} processes")
                
                # Add to background tasks
                background_tasks.add_task(do_stop)
                
                # Return immediately to prevent timeout
                return {
                    "success": True,
                    "message": f"Scenario '{scenario_name}' stop initiated",
                    "status": "stopping"
                }
                
            except HTTPException:
                raise
            except Exception as e:
                self.logger.error(f"Failed to initiate stop for scenario {scenario_name}: {e}")
                raise HTTPException(status_code=500, detail=str(e))

    def _load_scenario_config(self, scenario_name: str) -> Optional[Dict]:
        """Load scenario configuration from service.json"""
        vrooli_root = Path(os.environ.get('VROOLI_ROOT', Path.home() / 'Vrooli'))
        
        # Try both possible locations
        config_paths = [
            vrooli_root / 'scenarios' / scenario_name / '.vrooli/service.json',
            vrooli_root / 'scenarios' / scenario_name / 'service.json'
        ]
        
        for config_path in config_paths:
            if config_path.exists():
                try:
                    with open(config_path, 'r') as f:
                        return json.load(f)
                except (json.JSONDecodeError, OSError) as e:
                    self.logger.error(f"Failed to load config from {config_path}: {e}")
                    
        return None

    def _load_scenario_config_with_path(self, scenario_name: str) -> Tuple[Optional[Dict], Optional[str]]:
        """Load scenario configuration from service.json and return both config and path"""
        vrooli_root = Path(os.environ.get('VROOLI_ROOT', Path.home() / 'Vrooli'))
        
        # Try both possible locations
        config_paths = [
            vrooli_root / 'scenarios' / scenario_name / '.vrooli/service.json',
            vrooli_root / 'scenarios' / scenario_name / 'service.json'
        ]
        
        for config_path in config_paths:
            if config_path.exists():
                try:
                    with open(config_path, 'r') as f:
                        config = json.load(f)
                        return config, str(config_path)
                except (json.JSONDecodeError, OSError) as e:
                    self.logger.error(f"Failed to load config from {config_path}: {e}")
                    
        return None, None

    # Port allocation/deallocation removed - scenarios handle their own ports via lifecycle.sh

    async def _execute_lifecycle(self, scenario_name: str, phase: str, env_vars: Dict[str, str]) -> bool:
        """Execute lifecycle script for scenario without blocking"""
        try:
            vrooli_root = Path(os.environ.get('VROOLI_ROOT', Path.home() / 'Vrooli'))
            lifecycle_script = vrooli_root / 'scripts/lib/utils/lifecycle.sh'
            
            if not lifecycle_script.exists():
                self.logger.error(f"Lifecycle script not found: {lifecycle_script}")
                return False
                
            # Build command environment
            env = os.environ.copy()
            env.update(env_vars)
            env['SCENARIO_NAME'] = scenario_name
            env['SCENARIO_MODE'] = 'true'
            env['VROOLI_ROOT'] = str(vrooli_root)
            
            self.logger.info(f"Starting lifecycle script for {scenario_name} in {phase} mode")
            self.logger.debug(f"Environment variables: {env_vars}")
            
            # Execute lifecycle script without waiting for completion
            # The lifecycle script starts background processes, so we don't wait
            process = await asyncio.create_subprocess_exec(
                'bash', str(lifecycle_script), scenario_name, phase,
                env=env,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            
            # Wait briefly to check if process started successfully
            # If it exits immediately with error, we can catch it
            try:
                return_code = await asyncio.wait_for(process.wait(), timeout=3.0)
                # If we get here, process exited quickly (likely an error)
                if return_code != 0:
                    stderr = await process.stderr.read()
                    self.logger.error(f"Lifecycle script failed immediately for {scenario_name}: {stderr.decode()}")
                    return False
                else:
                    # Successful quick completion (unlikely for develop phase)
                    self.logger.info(f"Lifecycle script completed quickly for {scenario_name}")
                    return True
            except asyncio.TimeoutError:
                # This is expected - lifecycle script is running in background
                self.logger.info(f"Lifecycle script started successfully for {scenario_name} (running in background)")
                
                # Give it a moment to establish processes
                await asyncio.sleep(2)
                
                # Check if process is still running (sanity check)
                if process.returncode is None:
                    self.logger.debug(f"Lifecycle process still running for {scenario_name} (PID: {process.pid})")
                    return True
                else:
                    self.logger.warning(f"Lifecycle process exited unexpectedly for {scenario_name} with code {process.returncode}")
                    return False
                    
        except Exception as e:
            self.logger.error(f"Failed to execute lifecycle script for {scenario_name}: {e}", exc_info=True)
            return False

    async def start_server(self):
        """Start the FastAPI server"""
        config = uvicorn.Config(
            self.app,
            host=API_HOST,
            port=API_PORT,
            log_level="info"
        )
        server = uvicorn.Server(config)
        await server.serve()

def main():
    """Main entry point"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Enhanced Vrooli Scenario Orchestrator")
    parser.add_argument("--force", action="store_true", help="Force start even if another instance is running")
    global API_PORT
    parser.add_argument("--port", type=int, default=API_PORT, help=f"API port (default: {API_PORT})")
    
    # Use parse_known_args to ignore extra arguments from legacy scripts
    args, unknown = parser.parse_known_args()
    if unknown:
        print(f"Ignoring unknown arguments: {unknown}")
    
    # Update API_PORT if specified
    API_PORT = args.port
    
    # Initialize orchestrator
    orchestrator = EnhancedOrchestrator()
    
    # Acquire lock
    if not orchestrator.lock_manager.acquire_lock(force=args.force):
        sys.exit(1)
    
    # Setup signal handlers for graceful shutdown
    def signal_handler(signum, frame):
        orchestrator.logger.info(f"Received signal {signum}, shutting down...")
        orchestrator.lock_manager.release_lock()
        sys.exit(0)
    
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)
    
    try:
        orchestrator.logger.info(f"Starting Enhanced Orchestrator on port {API_PORT}")
        asyncio.run(orchestrator.start_server())
    except KeyboardInterrupt:
        orchestrator.logger.info("Orchestrator stopped by user")
    finally:
        orchestrator.lock_manager.release_lock()

if __name__ == "__main__":
    main()