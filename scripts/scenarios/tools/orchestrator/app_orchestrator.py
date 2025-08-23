#!/usr/bin/env python3
"""
Vrooli App Orchestrator
Reliable Python-based app starter that replaces simple-multi-app-starter.sh
Handles subprocess management without shell interpretation issues.
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
from typing import Dict, List, Optional, Tuple

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
class App:
    """Represents a Vrooli app to be started"""
    name: str
    enabled: bool
    port: int
    ui_port: int
    process: Optional[asyncio.subprocess.Process] = None
    pid: Optional[int] = None
    log_file: Optional[Path] = None
    status: str = "pending"


class AppOrchestrator:
    """Orchestrates starting and managing Vrooli apps"""
    
    def __init__(self, vrooli_root: Path, verbose: bool = False, fast_mode: bool = True):
        self.vrooli_root = vrooli_root
        self.verbose = verbose
        self.fast_mode = fast_mode
        self.generated_apps_dir = Path.home() / "generated-apps"
        self.apps: Dict[str, App] = {}
        self.base_port = int(os.environ.get("VROOLI_DEV_PORT_START", "3001"))
        self.max_apps = int(os.environ.get("VROOLI_DEV_MAX_APPS", "100"))
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
        """Load enabled apps from catalog.json"""
        catalog_path = self.vrooli_root / "scripts" / "scenarios" / "catalog.json"
        
        if not catalog_path.exists():
            self.logger.error(f"Catalog not found: {catalog_path}")
            return []
        
        try:
            with open(catalog_path) as f:
                catalog = json.load(f)
            
            apps = []
            port = self.base_port
            
            for scenario in catalog.get("scenarios", []):
                if scenario.get("enabled", False):
                    app_name = scenario["name"]
                    app_path = self.generated_apps_dir / app_name
                    
                    # Validate app exists
                    if not self._validate_app(app_name, app_path):
                        continue
                    
                    app = App(
                        name=app_name,
                        enabled=True,
                        port=port,
                        ui_port=port + 1000,
                        log_file=self.log_dir / f"{app_name}.log"
                    )
                    apps.append(app)
                    self.apps[app_name] = app
                    port += 1
                    
                    if len(apps) >= self.max_apps:
                        self.logger.warning(f"Reached max apps limit: {self.max_apps}")
                        break
            
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
        """Start a single app as a detached subprocess"""
        try:
            self._print(f"Starting {app.name}...", Fore.BLUE)
            
            # Build command - use direct path to avoid exec issues
            cmd = [
                str(self.vrooli_root / "cli" / "commands" / "app-commands.sh"),
                "start",
                app.name
            ]
            
            if self.fast_mode:
                cmd.append("--fast")
            
            # Setup environment
            env = os.environ.copy()
            env['SERVICE_PORT'] = str(app.port)
            env['UI_PORT'] = str(app.ui_port)
            env['VROOLI_ROOT'] = str(self.vrooli_root)
            env['GENERATED_APPS_DIR'] = str(self.generated_apps_dir)
            
            # Open log file for this app
            log_file = open(app.log_file, 'w')
            
            # Create subprocess with proper isolation
            # Key: use start_new_session to fully detach from parent
            process = await asyncio.create_subprocess_exec(
                *cmd,
                env=env,
                stdout=log_file,
                stderr=asyncio.subprocess.STDOUT,
                start_new_session=True,  # This is crucial - creates new session
                cwd=str(self.vrooli_root)
            )
            
            app.process = process
            app.pid = process.pid
            self.running_processes.append(process)
            
            # Give it a moment to start
            await asyncio.sleep(1)
            
            # Check if process is still running
            if process.returncode is None:
                # Still running - verify with process manager if available
                if await self._verify_app_running(app):
                    app.status = "running"
                    self._print(f"  ✓ {app.name} started on port {app.port}", Fore.GREEN)
                    return True
                else:
                    app.status = "verification_failed"
                    self._print(f"  ⚠ {app.name} started but verification failed", Fore.YELLOW)
                    # Still consider it a success if process is running
                    return True
            else:
                # Process exited
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
        """Verify app is actually running using process manager"""
        for attempt in range(max_attempts):
            try:
                # Method 1: Check if our subprocess is still alive
                if app.process and app.process.returncode is None:
                    # Process is running
                    
                    # Method 2: If we have psutil, do deeper check
                    if HAS_PSUTIL and app.pid:
                        try:
                            proc = psutil.Process(app.pid)
                            if proc.is_running() and proc.status() != psutil.STATUS_ZOMBIE:
                                return True
                        except psutil.NoSuchProcess:
                            pass
                    else:
                        # No psutil, but process is alive
                        return True
                
                # Method 3: Check with Vrooli's process manager
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
        """Start all enabled apps concurrently"""
        apps = self.load_enabled_apps()
        
        if not apps:
            self._print("No enabled apps found", Fore.YELLOW)
            return 0, 0
        
        self._print(f"\n{'=' * 60}", Fore.CYAN)
        self._print(f"Starting {len(apps)} apps...", Fore.CYAN)
        self._print(f"{'=' * 60}\n", Fore.CYAN)
        
        # Start apps concurrently with controlled concurrency
        # Limit concurrent starts to avoid overwhelming the system
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
                    # Try graceful termination first
                    app.process.terminate()
                    await asyncio.sleep(0.5)
                    
                    # Force kill if still running
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
                    self._print(f"  {app.name:30} → http://localhost:{app.port}", Fore.BLUE)
                
                self._print(f"\n{'=' * 60}", Fore.CYAN)
                self._print("Management Commands:", Fore.CYAN)
                self._print(f"{'=' * 60}", Fore.CYAN)
                self._print("  vrooli app list                # Show all app status")
                self._print("  vrooli app stop <name>         # Stop specific app")
                self._print("  vrooli app logs <name>         # Show app logs")
                self._print("  vrooli app restart <name>      # Restart specific app")
        
        self._print(f"{'=' * 60}\n", Fore.CYAN)


async def main():
    """Main entry point"""
    # Parse arguments
    verbose = "--verbose" in sys.argv
    fast_mode = "--fast" in sys.argv or os.environ.get("FAST_MODE", "true").lower() == "true"
    
    # Determine Vrooli root
    script_path = Path(__file__).resolve()
    # orchestrator is in scripts/scenarios/tools/orchestrator/
    # So we need to go up 5 levels to get to Vrooli root
    vrooli_root = script_path.parent.parent.parent.parent.parent
    
    # Create orchestrator
    orchestrator = AppOrchestrator(
        vrooli_root=vrooli_root,
        verbose=verbose,
        fast_mode=fast_mode
    )
    
    try:
        # Start all apps
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