#!/usr/bin/env python3
"""
Robust PID and lock management for Vrooli orchestrator
Handles all edge cases and provides atomic operations
"""

import os
import sys
import time
import json
import fcntl
import socket
import psutil
from pathlib import Path
from typing import Optional, Dict

class OrchestrationLockManager:
    """Manages PID files and locks with atomic operations and comprehensive validation"""
    
    def __init__(self, port: int = 9500):
        self.port = port
        self.pid_file = Path("/tmp/vrooli-orchestrator.pid")
        self.lock_file = Path("/tmp/vrooli-orchestrator.lock")
        self.state_file = Path("/tmp/vrooli-orchestrator.state")
        
    def acquire_lock(self, force: bool = False) -> bool:
        """
        Atomically acquire orchestrator lock with comprehensive checks
        Returns True if lock acquired, False otherwise
        """
        
        # Step 1: Clean up any stale files
        self._cleanup_stale_files()
        
        # Step 2: Check if another orchestrator is genuinely running
        existing = self._find_running_orchestrator()
        if existing and not force:
            print(f"ERROR: Another orchestrator instance is already running (PID: {existing['pid']})", file=sys.stderr)
            print(f"Stop it first with: kill {existing['pid']}", file=sys.stderr)
            print(f"Or use --force to override", file=sys.stderr)
            return False
            
        if existing and force:
            print(f"Force mode: Stopping existing orchestrator (PID: {existing['pid']})")
            self._stop_orchestrator(existing['pid'])
            time.sleep(2)
        
        # Step 3: Atomic lock acquisition
        try:
            # Create lock file with exclusive access
            lock_fd = os.open(str(self.lock_file), os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o644)
            
            # Write our PID atomically
            os.write(lock_fd, str(os.getpid()).encode())
            os.close(lock_fd)
            
            # Write detailed state file
            state = {
                'pid': os.getpid(),
                'port': self.port,
                'start_time': time.time(),
                'cmdline': ' '.join(sys.argv)
            }
            self.state_file.write_text(json.dumps(state, indent=2))
            
            # Also write simple PID file for compatibility
            self.pid_file.write_text(str(os.getpid()))
            
            return True
            
        except FileExistsError:
            # Lock file exists - another process got there first
            return False
        except Exception as e:
            print(f"ERROR: Failed to acquire lock: {e}", file=sys.stderr)
            return False
    
    def release_lock(self):
        """Release orchestrator lock and clean up files"""
        try:
            # Only remove if we own the lock
            if self.lock_file.exists():
                try:
                    current_pid = int(self.lock_file.read_text().strip())
                    if current_pid == os.getpid():
                        self.lock_file.unlink()
                        self.pid_file.unlink(missing_ok=True)
                        self.state_file.unlink(missing_ok=True)
                except (ValueError, FileNotFoundError):
                    pass
        except Exception as e:
            print(f"Warning: Error releasing lock: {e}", file=sys.stderr)
    
    def _cleanup_stale_files(self):
        """Remove stale PID/lock files from dead processes"""
        for file in [self.pid_file, self.lock_file]:
            if file.exists():
                try:
                    pid = int(file.read_text().strip())
                    if not self._is_process_running(pid):
                        print(f"Cleaning up stale file: {file}")
                        file.unlink()
                except (ValueError, FileNotFoundError):
                    # Invalid or missing file, remove it
                    file.unlink(missing_ok=True)
    
    def _find_running_orchestrator(self) -> Optional[Dict]:
        """
        Find if an orchestrator is genuinely running
        Checks both PID files and port usage
        """
        
        # Method 1: Check state file
        if self.state_file.exists():
            try:
                state = json.loads(self.state_file.read_text())
                pid = state['pid']
                if self._is_orchestrator_process(pid):
                    return state
            except (json.JSONDecodeError, KeyError, ValueError):
                pass
        
        # Method 2: Check who's using the port
        try:
            for conn in psutil.net_connections():
                if conn.laddr.port == self.port and conn.status == 'LISTEN':
                    try:
                        proc = psutil.Process(conn.pid)
                        cmdline = ' '.join(proc.cmdline())
                        if 'orchestrator' in cmdline.lower():
                            return {
                                'pid': conn.pid,
                                'port': self.port,
                                'cmdline': cmdline
                            }
                    except (psutil.NoSuchProcess, psutil.AccessDenied):
                        continue
        except psutil.AccessDenied:
            # Fallback: Try to bind to port
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            try:
                sock.bind(('', self.port))
                sock.close()
                return None  # Port is free
            except OSError:
                # Port is in use, but we can't determine by whom
                return {'pid': 'unknown', 'port': self.port}
        
        return None
    
    def _is_process_running(self, pid: int) -> bool:
        """Check if a process with given PID is running"""
        try:
            proc = psutil.Process(pid)
            return proc.is_running()
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            return False
    
    def _is_orchestrator_process(self, pid: int) -> bool:
        """Check if a PID belongs to an orchestrator process"""
        try:
            proc = psutil.Process(pid)
            cmdline = ' '.join(proc.cmdline())
            return 'orchestrator' in cmdline.lower()
        except (psutil.NoSuchProcess, psutil.AccessDenied):
            return False
    
    def _stop_orchestrator(self, pid: int):
        """Stop an orchestrator process gracefully"""
        try:
            os.kill(pid, 15)  # SIGTERM
            time.sleep(2)
            if self._is_process_running(pid):
                os.kill(pid, 9)  # SIGKILL
        except ProcessLookupError:
            pass

# Standalone functions for backward compatibility
def check_and_clean():
    """Check for stale PIDs and clean them up"""
    manager = OrchestrationLockManager()
    manager._cleanup_stale_files()
    return manager._find_running_orchestrator()

def acquire_orchestrator_lock(port: int = 9500, force: bool = False) -> bool:
    """Main entry point for lock acquisition"""
    manager = OrchestrationLockManager(port)
    return manager.acquire_lock(force)

def release_orchestrator_lock():
    """Main entry point for lock release"""
    manager = OrchestrationLockManager()
    manager.release_lock()

if __name__ == "__main__":
    # Command-line utility for testing and maintenance
    import argparse
    
    parser = argparse.ArgumentParser(description="Orchestrator PID/Lock Manager")
    parser.add_argument("command", choices=["check", "clean", "acquire", "release"],
                       help="Command to execute")
    parser.add_argument("--force", action="store_true", help="Force operation")
    parser.add_argument("--port", type=int, default=9500, help="Orchestrator port")
    
    args = parser.parse_args()
    
    if args.command == "check":
        manager = OrchestrationLockManager(args.port)
        running = manager._find_running_orchestrator()
        if running:
            print(f"Orchestrator is running: {running}")
        else:
            print("No orchestrator running")
    
    elif args.command == "clean":
        manager = OrchestrationLockManager(args.port)
        manager._cleanup_stale_files()
        print("Cleaned up stale files")
    
    elif args.command == "acquire":
        if acquire_orchestrator_lock(args.port, args.force):
            print("Lock acquired successfully")
        else:
            print("Failed to acquire lock")
            sys.exit(1)
    
    elif args.command == "release":
        release_orchestrator_lock()
        print("Lock released")