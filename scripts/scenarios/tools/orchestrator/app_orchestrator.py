#!/usr/bin/env python3
"""
Enhanced Vrooli App Orchestrator
Reliable Python-based app starter with sophisticated port allocation
"""

import asyncio
import json
import logging
import os
import signal
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Tuple, Set
import re

# Optional imports - gracefully handle if not available
try:
    from colorama import Fore, Style, init as colorama_init
    colorama_init(autoreset=True)
    HAS_COLOR = True
except ImportError:
    HAS_COLOR = False
    # Fallback - no colors
    class Fore:
        RED = GREEN = YELLOW = BLUE = CYAN = RESET = ''
    class Style:
        BRIGHT = RESET_ALL = ''

try:
    import psutil
    HAS_PSUTIL = True
except ImportError:
    HAS_PSUTIL = False

try:
    import jsonschema
    HAS_JSONSCHEMA = True
except ImportError:
    HAS_JSONSCHEMA = False


@dataclass
class PortRequirement:
    """Represents a port requirement for a scenario"""
    scenario_name: str
    port_type: str          # "api", "ui", "websocket", etc.
    env_var: str           # "SERVICE_PORT", "UI_PORT", etc.
    specification: str     # "39001" or "8200-8299"
    description: str       # Human-readable description
    priority: int          # Lower = higher priority (0=exact port, 1=small range, 2=large range)
    range_size: int        # Size of the port range (1 for exact ports)
    allocated_port: Optional[int] = None


@dataclass 
class App:
    """Represents a Vrooli app to be started"""
    name: str
    enabled: bool
    allocated_ports: Dict[str, int]  # port_type -> port number
    process: Optional[asyncio.subprocess.Process] = None
    pid: Optional[int] = None
    log_file: Optional[Path] = None
    status: str = "pending"


class PortAllocator:
    """Sophisticated port allocation engine"""
    
    def __init__(self, generated_apps_dir: Path):
        self.generated_apps_dir = generated_apps_dir
        self.allocated_ports: Set[int] = set()
        
    def parse_port_requirements(self, enabled_scenarios: List[str]) -> List[PortRequirement]:
        """Parse port requirements from all enabled scenarios"""
        requirements = []
        
        for scenario_name in enabled_scenarios:
            scenario_dir = self.generated_apps_dir / scenario_name
            service_json = scenario_dir / ".vrooli" / "service.json"
            
            if not service_json.exists():
                continue
                
            try:
                with open(service_json) as f:
                    config = json.load(f)
                
                # Parse ports section
                ports = config.get("ports", {})
                for port_type, port_config in ports.items():
                    if isinstance(port_config, dict):
                        env_var = port_config.get("env_var", f"{port_type.upper()}_PORT")
                        range_spec = port_config.get("range", "auto")
                        description = port_config.get("description", f"{port_type} port")
                        
                        # Calculate priority based on range specification
                        priority, range_size = self._calculate_priority(range_spec)
                        
                        requirement = PortRequirement(
                            scenario_name=scenario_name,
                            port_type=port_type,
                            env_var=env_var,
                            specification=str(range_spec),
                            description=description,
                            priority=priority,
                            range_size=range_size
                        )
                        requirements.append(requirement)
                        
            except Exception as e:
                print(f"⚠️ Failed to parse ports for {scenario_name}: {e}")
                continue
        
        # Sort by priority (exact ports first, then shortest ranges)
        requirements.sort(key=lambda r: (r.priority, r.range_size, r.scenario_name))
        return requirements
    
    def _calculate_priority(self, range_spec: str) -> Tuple[int, int]:
        """Calculate priority and range size from range specification"""
        spec_str = str(range_spec).strip()
        
        # Exact port (highest priority)
        if spec_str.isdigit():
            return (0, 1)  # Priority 0, range size 1
        
        # Port range like "8200-8299"
        if "-" in spec_str:
            try:
                start, end = spec_str.split("-", 1)
                start_port = int(start.strip())
                end_port = int(end.strip())
                range_size = end_port - start_port + 1
                
                # Priority based on range size - smaller ranges get higher priority
                if range_size <= 10:
                    priority = 1  # Small range
                elif range_size <= 100: 
                    priority = 2  # Medium range
                else:
                    priority = 3  # Large range
                    
                return (priority, range_size)
            except ValueError:
                pass
        
        # Auto or unknown - lowest priority, treat as needing 1 port
        return (4, 1)
    
    def allocate_ports(self, requirements: List[PortRequirement]) -> Dict[str, Dict[str, int]]:
        """Allocate ports using sophisticated algorithm"""
        allocations = {}  # scenario_name -> {port_type: port_number}
        
        print(f"🔢 Allocating ports for {len(requirements)} requirements...")
        if requirements:
            print("   Priority order:")
            for req in requirements[:10]:  # Show first 10
                print(f"     {req.scenario_name:25} {req.port_type:8} {req.specification:12} (priority {req.priority})")
            if len(requirements) > 10:
                print(f"     ... and {len(requirements) - 10} more")
            print()
        
        for req in requirements:
            allocated_port = self._allocate_single_port(req)
            if allocated_port:
                if req.scenario_name not in allocations:
                    allocations[req.scenario_name] = {}
                allocations[req.scenario_name][req.port_type] = allocated_port
                req.allocated_port = allocated_port
                print(f"   ✅ {req.scenario_name:25} {req.port_type:8} → {allocated_port}")
            else:
                print(f"   ❌ {req.scenario_name:25} {req.port_type:8} → FAILED")
        
        return allocations
    
    def _allocate_single_port(self, req: PortRequirement) -> Optional[int]:
        """Allocate a single port based on requirement specification"""
        spec = req.specification.strip()
        
        # Exact port
        if spec.isdigit():
            port = int(spec)
            if port not in self.allocated_ports and self._is_port_available(port):
                self.allocated_ports.add(port)
                return port
            else:
                # Exact port not available - this is a hard failure for exact ports
                return None
        
        # Port range like "8200-8299"  
        if "-" in spec:
            try:
                start, end = spec.split("-", 1)
                start_port = int(start.strip())
                end_port = int(end.strip())
                
                # Try to find an available port in the range
                for port in range(start_port, end_port + 1):
                    if port not in self.allocated_ports and self._is_port_available(port):
                        self.allocated_ports.add(port)
                        return port
                        
                # No port available in range
                return None
                
            except ValueError:
                pass
        
        # Auto allocation - find any available port starting from 3001
        return self._find_auto_port()
    
    def _find_auto_port(self, start_port: int = 3001) -> Optional[int]:
        """Find any available port starting from start_port"""
        for port in range(start_port, start_port + 2000):  # Search 2000 ports
            if port not in self.allocated_ports and self._is_port_available(port):
                self.allocated_ports.add(port)
                return port
        return None
    
    def _is_port_available(self, port: int) -> bool:
        """Check if a port is actually available on the system"""
        try:
            import socket
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                s.bind(('localhost', port))
                return True
        except OSError:
            return False


class AppOrchestrator:
    """Enhanced orchestrator with sophisticated port allocation"""
    
    def __init__(self, vrooli_root: Path, verbose: bool = False, fast_mode: bool = True):
        self.vrooli_root = vrooli_root
        self.verbose = verbose
        self.fast_mode = fast_mode
        self.generated_apps_dir = Path.home() / "generated-apps"
        self.apps: Dict[str, App] = {}
        self.port_allocator = PortAllocator(self.generated_apps_dir)
        self.log_dir = Path("/tmp/vrooli-orchestrator")
        self.log_dir.mkdir(exist_ok=True)
        
        # Setup logging
        log_level = logging.DEBUG if verbose else logging.INFO
        logging.basicConfig(
            level=log_level,
            format='%(asctime)s - %(levelname)s - %(message)s',
            handlers=[
                logging.FileHandler(self.log_dir / "orchestrator.log"),
                logging.StreamHandler()
            ]
        )
        self.logger = logging.getLogger(__name__)
        
        # Track subprocesses for cleanup
        self.running_processes: List[asyncio.subprocess.Process] = []
        
        # Setup signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals gracefully"""
        self.logger.info(f"Received signal {signum}, shutting down gracefully...")
        asyncio.create_task(self.shutdown())
    
    def _print(self, message: str, color: str = Fore.RESET):
        """Print colored message to console"""
        if HAS_COLOR:
            print(f"{color}{message}{Style.RESET_ALL}")
        else:
            print(message)
    
    def load_enabled_apps(self) -> List[App]:
        """Load enabled apps with sophisticated port allocation"""
        catalog_path = self.vrooli_root / "scripts" / "scenarios" / "catalog.json"
        
        if not catalog_path.exists():
            self.logger.error(f"Catalog not found: {catalog_path}")
            return []
        
        try:
            with open(catalog_path) as f:
                catalog = json.load(f)
            
            # Get enabled scenario names
            enabled_scenarios = []
            for scenario in catalog.get("scenarios", []):
                if scenario.get("enabled", False):
                    app_name = scenario["name"]
                    app_path = self.generated_apps_dir / app_name
                    
                    # Validate app exists
                    if self._validate_app(app_name, app_path):
                        enabled_scenarios.append(app_name)
            
            if not enabled_scenarios:
                return []
            
            # Parse port requirements and allocate ports smartly
            port_requirements = self.port_allocator.parse_port_requirements(enabled_scenarios)
            port_allocations = self.port_allocator.allocate_ports(port_requirements)
            
            # Create App objects with allocated ports
            apps = []
            for app_name in enabled_scenarios:
                allocated_ports = port_allocations.get(app_name, {})
                
                # Ensure at least basic ports if none were specified
                if not allocated_ports:
                    # Fall back to auto allocation for basic ports
                    api_port = self.port_allocator._find_auto_port()
                    ui_port = self.port_allocator._find_auto_port()
                    if api_port and ui_port:
                        allocated_ports = {"api": api_port, "ui": ui_port}
                
                if allocated_ports:
                    app = App(
                        name=app_name,
                        enabled=True,
                        allocated_ports=allocated_ports,
                        log_file=self.log_dir / f"{app_name}.log"
                    )
                    apps.append(app)
                    self.apps[app_name] = app
            
            return apps
            
        except Exception as e:
            self.logger.error(f"Failed to load catalog: {e}")
            return []
    
    def _validate_app(self, app_name: str, app_path: Path) -> bool:
        """Validate that app directory has required files"""
        if not app_path.exists():
            self.logger.warning(f"App directory not found: {app_path}")
            return False
        
        manage_script = app_path / "scripts" / "manage.sh"
        if not manage_script.exists():
            self.logger.warning(f"Missing manage.sh for {app_name}")
            return False
        
        service_json = app_path / ".vrooli" / "service.json"
        if not service_json.exists():
            self.logger.warning(f"Missing service.json for {app_name}")
            return False
        
        return True
    
    async def start_app(self, app: App) -> bool:
        """Start a single app with proper port allocation"""
        try:
            self._print(f"Starting {app.name}...", Fore.BLUE)
            
            # Build command - call manage.sh directly to avoid process manager issues
            app_path = self.generated_apps_dir / app.name
            manage_script = app_path / "scripts" / "manage.sh"
            
            cmd = [
                "bash",
                str(manage_script),
                "develop"
            ]
            
            # Setup environment with allocated ports
            env = os.environ.copy()
            env['VROOLI_ROOT'] = str(self.vrooli_root)
            env['GENERATED_APPS_DIR'] = str(self.generated_apps_dir)
            
            # Set port environment variables
            for port_type, port_num in app.allocated_ports.items():
                if port_type == "api":
                    env['SERVICE_PORT'] = str(port_num)
                elif port_type == "ui":
                    env['UI_PORT'] = str(port_num)
                else:
                    # Generic mapping for other port types
                    env[f"{port_type.upper()}_PORT"] = str(port_num)
            
            # Open log file for this app
            log_file = open(app.log_file, 'w')
            
            # Create subprocess with proper isolation
            process = await asyncio.create_subprocess_exec(
                *cmd,
                env=env,
                stdout=log_file,
                stderr=asyncio.subprocess.STDOUT,
                start_new_session=True,
                cwd=str(app_path)
            )
            
            app.process = process
            app.pid = process.pid
            self.running_processes.append(process)
            
            # Give it a moment to start
            await asyncio.sleep(1)
            
            # Check if process is still running
            if process.returncode is None:
                if await self._verify_app_running(app):
                    app.status = "running"
                    ports_str = ", ".join([f"{k}:{v}" for k, v in app.allocated_ports.items()])
                    self._print(f"  ✓ {app.name} started ({ports_str})", Fore.GREEN)
                    return True
                else:
                    app.status = "verification_failed"
                    self._print(f"  ⚠ {app.name} started but verification failed", Fore.YELLOW)
                    return True
            else:
                app.status = "failed"
                self._print(f"  ✗ {app.name} failed with code {process.returncode}", Fore.RED)
                
                # Show last few lines of log
                if app.log_file.exists():
                    with open(app.log_file) as f:
                        lines = f.readlines()
                        if lines:
                            self._print("    Last output:", Fore.YELLOW)
                            for line in lines[-3:]:
                                self._print(f"      {line.rstrip()}", Fore.YELLOW)
                
                return False
                
        except Exception as e:
            self.logger.error(f"Failed to start {app.name}: {e}")
            app.status = "error"
            return False
    
    async def _verify_app_running(self, app: App, max_attempts: int = 5) -> bool:
        """Verify app is actually running"""
        for attempt in range(max_attempts):
            try:
                if app.process and app.process.returncode is None:
                    if HAS_PSUTIL and app.pid:
                        try:
                            proc = psutil.Process(app.pid)
                            if proc.is_running() and proc.status() != psutil.STATUS_ZOMBIE:
                                return True
                        except psutil.NoSuchProcess:
                            pass
                    else:
                        return True
                
                # Check with Vrooli's process manager
                result = await asyncio.create_subprocess_exec(
                    str(self.vrooli_root / "cli" / "vrooli"),
                    "app", "status", app.name, "--json",
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.DEVNULL
                )
                stdout, _ = await result.communicate()
                
                if result.returncode == 0 and stdout:
                    status = json.loads(stdout)
                    if status.get("status") in ["running", "starting"]:
                        return True
                
            except Exception as e:
                self.logger.debug(f"Verification attempt {attempt + 1} failed: {e}")
            
            await asyncio.sleep(1)
        
        return False
    
    async def start_all_apps(self) -> Tuple[int, int]:
        """Start all enabled apps with smart port allocation"""
        apps = self.load_enabled_apps()
        
        if not apps:
            self._print("No enabled apps found", Fore.YELLOW)
            return 0, 0
        
        self._print(f"\n{'=' * 60}", Fore.CYAN)
        self._print(f"Starting {len(apps)} apps with smart port allocation...", Fore.CYAN)
        self._print(f"{'=' * 60}\n", Fore.CYAN)
        
        # Start apps concurrently with controlled concurrency
        semaphore = asyncio.Semaphore(5)  # Max 5 concurrent starts
        
        async def start_with_limit(app):
            async with semaphore:
                return await self.start_app(app)
        
        # Create all start tasks
        tasks = [start_with_limit(app) for app in apps]
        
        # Wait for all to complete
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # Count successes and failures
        success_count = sum(1 for r in results if r is True)
        fail_count = len(results) - success_count
        
        return success_count, fail_count
    
    async def shutdown(self):
        """Gracefully shutdown all running apps"""
        self._print("\nShutting down apps...", Fore.YELLOW)
        
        for app_name, app in self.apps.items():
            if app.process and app.process.returncode is None:
                try:
                    app.process.terminate()
                    await asyncio.sleep(0.5)
                    
                    if app.process.returncode is None:
                        app.process.kill()
                    
                    self._print(f"  Stopped {app_name}", Fore.YELLOW)
                except Exception as e:
                    self.logger.error(f"Error stopping {app_name}: {e}")
    
    def print_summary(self, success_count: int, fail_count: int):
        """Print final summary with running apps"""
        self._print(f"\n{'=' * 60}", Fore.CYAN)
        self._print("Startup Summary:", Fore.CYAN)
        self._print(f"  Started: {success_count} apps", Fore.GREEN if success_count > 0 else Fore.YELLOW)
        self._print(f"  Failed:  {fail_count} apps", Fore.RED if fail_count > 0 else Fore.GREEN)
        
        if success_count > 0:
            running_apps = [app for app in self.apps.values() if app.status in ["running", "verification_failed"]]
            if running_apps:
                self._print(f"\n{'=' * 60}", Fore.CYAN)
                self._print("Running Applications:", Fore.CYAN)
                self._print(f"{'=' * 60}", Fore.CYAN)
                
                for app in running_apps:
                    # Show primary port (usually api port)
                    primary_port = app.allocated_ports.get("api") or next(iter(app.allocated_ports.values()))
                    port_details = ", ".join([f"{k}:{v}" for k, v in app.allocated_ports.items()])
                    self._print(f"  {app.name:30} → http://localhost:{primary_port} ({port_details})", Fore.BLUE)
                
                self._print(f"\n{'=' * 60}", Fore.CYAN)
        
        self._print(f"{'=' * 60}\n", Fore.CYAN)


async def main():
    """Main entry point"""
    # Parse arguments
    verbose = "--verbose" in sys.argv
    fast_mode = "--fast" in sys.argv or os.environ.get("FAST_MODE", "true").lower() == "true"
    
    # Determine Vrooli root
    script_path = Path(__file__).resolve()
    vrooli_root = script_path.parent.parent.parent.parent.parent
    
    # Create enhanced orchestrator
    orchestrator = AppOrchestrator(
        vrooli_root=vrooli_root,
        verbose=verbose,
        fast_mode=fast_mode
    )
    
    try:
        # Start all apps with smart port allocation
        success_count, fail_count = await orchestrator.start_all_apps()
        
        # Print summary
        orchestrator.print_summary(success_count, fail_count)
        
        # Exit with appropriate code
        if success_count == 0:
            sys.exit(1)
        elif fail_count > 0:
            sys.exit(2)  # Partial success
        else:
            sys.exit(0)
            
    except KeyboardInterrupt:
        await orchestrator.shutdown()
        sys.exit(130)
    except Exception as e:
        orchestrator.logger.error(f"Fatal error: {e}")
        await orchestrator.shutdown()
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())