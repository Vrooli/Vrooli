"""Browser health monitoring service for Agent S2"""

import os
import psutil
import logging
import time
from typing import Dict, Any, Optional, List
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


class BrowserHealthMonitor:
    """Monitor browser processes for crashes and health issues"""
    
    def __init__(self):
        """Initialize browser health monitor"""
        self.crash_log = []
        self.max_crash_log_entries = 100
        self.restart_count = 0
        self.last_restart_time = None
        self.memory_threshold_mb = 1500
        self.cpu_threshold_percent = 80
        
    def check_firefox_health(self) -> Dict[str, Any]:
        """Check Firefox process health
        
        Returns:
            Dictionary with health status and metrics
        """
        firefox_processes = []
        total_memory_mb = 0
        total_cpu_percent = 0
        
        try:
            # Find all Firefox processes
            for proc in psutil.process_iter(['pid', 'name', 'memory_info', 'cpu_percent']):
                try:
                    if 'firefox' in proc.info['name'].lower():
                        memory_mb = proc.info['memory_info'].rss / (1024 * 1024)
                        cpu_percent = proc.info['cpu_percent']
                        
                        firefox_processes.append({
                            'pid': proc.info['pid'],
                            'name': proc.info['name'],
                            'memory_mb': round(memory_mb, 2),
                            'cpu_percent': round(cpu_percent, 2)
                        })
                        
                        total_memory_mb += memory_mb
                        total_cpu_percent += cpu_percent
                        
                except (psutil.NoSuchProcess, psutil.AccessDenied):
                    continue
                    
            # Analyze health
            is_running = len(firefox_processes) > 0
            memory_warning = total_memory_mb > self.memory_threshold_mb
            cpu_warning = total_cpu_percent > self.cpu_threshold_percent
            
            health_status = {
                'healthy': is_running and not memory_warning and not cpu_warning,
                'running': is_running,
                'process_count': len(firefox_processes),
                'total_memory_mb': round(total_memory_mb, 2),
                'total_cpu_percent': round(total_cpu_percent, 2),
                'memory_warning': memory_warning,
                'cpu_warning': cpu_warning,
                'processes': firefox_processes,
                'restart_count': self.restart_count,
                'last_restart': self.last_restart_time.isoformat() if self.last_restart_time else None,
                'crash_log_entries': len(self.crash_log)
            }
            
            # Check for recent crashes
            recent_crashes = self._get_recent_crashes(minutes=5)
            if recent_crashes:
                health_status['recent_crashes'] = recent_crashes
                health_status['healthy'] = False
                
            return health_status
            
        except Exception as e:
            logger.error(f"Error checking Firefox health: {e}")
            return {
                'healthy': False,
                'running': False,
                'error': str(e)
            }
            
    def log_crash(self, details: Dict[str, Any]) -> None:
        """Log a browser crash event
        
        Args:
            details: Crash details including PID, signal, timestamp
        """
        crash_entry = {
            'timestamp': datetime.now().isoformat(),
            'details': details
        }
        
        self.crash_log.append(crash_entry)
        
        # Trim log if too large
        if len(self.crash_log) > self.max_crash_log_entries:
            self.crash_log = self.crash_log[-self.max_crash_log_entries:]
            
        logger.warning(f"Browser crash logged: {details}")
        
    def log_restart(self) -> None:
        """Log a browser restart event"""
        self.restart_count += 1
        self.last_restart_time = datetime.now()
        logger.info(f"Browser restarted (count: {self.restart_count})")
        
    def _get_recent_crashes(self, minutes: int = 10) -> List[Dict[str, Any]]:
        """Get crashes from the last N minutes
        
        Args:
            minutes: Time window to check
            
        Returns:
            List of recent crash entries
        """
        cutoff_time = datetime.now() - timedelta(minutes=minutes)
        recent = []
        
        for entry in reversed(self.crash_log):
            try:
                crash_time = datetime.fromisoformat(entry['timestamp'])
                if crash_time > cutoff_time:
                    recent.append(entry)
                else:
                    break
            except:
                continue
                
        return recent
        
    def get_crash_statistics(self) -> Dict[str, Any]:
        """Get crash statistics
        
        Returns:
            Dictionary with crash statistics
        """
        if not self.crash_log:
            return {
                'total_crashes': 0,
                'crashes_last_hour': 0,
                'crashes_last_24h': 0,
                'most_common_signal': None
            }
            
        # Count crashes by time window
        now = datetime.now()
        hour_ago = now - timedelta(hours=1)
        day_ago = now - timedelta(hours=24)
        
        crashes_last_hour = 0
        crashes_last_24h = 0
        signal_counts = {}
        
        for entry in self.crash_log:
            try:
                crash_time = datetime.fromisoformat(entry['timestamp'])
                
                if crash_time > day_ago:
                    crashes_last_24h += 1
                    
                    if crash_time > hour_ago:
                        crashes_last_hour += 1
                        
                    # Count signals
                    signal = entry['details'].get('signal', 'unknown')
                    signal_counts[signal] = signal_counts.get(signal, 0) + 1
                    
            except:
                continue
                
        # Find most common signal
        most_common_signal = None
        if signal_counts:
            most_common_signal = max(signal_counts, key=signal_counts.get)
            
        return {
            'total_crashes': len(self.crash_log),
            'crashes_last_hour': crashes_last_hour,
            'crashes_last_24h': crashes_last_24h,
            'most_common_signal': most_common_signal,
            'signal_distribution': signal_counts,
            'restart_count': self.restart_count,
            'uptime_since_last_restart': (
                str(datetime.now() - self.last_restart_time) 
                if self.last_restart_time else None
            )
        }
        
    def should_restart_browser(self) -> bool:
        """Determine if browser should be restarted
        
        Returns:
            True if browser should be restarted
        """
        health = self.check_firefox_health()
        
        # Not running
        if not health['running']:
            return True
            
        # Memory threshold exceeded
        if health['memory_warning']:
            logger.warning(f"Browser memory high: {health['total_memory_mb']}MB")
            return True
            
        # Too many recent crashes
        recent_crashes = self._get_recent_crashes(minutes=5)
        if len(recent_crashes) >= 3:
            logger.warning(f"Too many recent crashes: {len(recent_crashes)}")
            return True
            
        return False
        
    def get_browser_info(self) -> Dict[str, Any]:
        """Get comprehensive browser information
        
        Returns:
            Dictionary with browser info and health metrics
        """
        health = self.check_firefox_health()
        stats = self.get_crash_statistics()
        
        return {
            'browser': 'Firefox ESR',
            'health': health,
            'statistics': stats,
            'thresholds': {
                'memory_mb': self.memory_threshold_mb,
                'cpu_percent': self.cpu_threshold_percent
            }
        }
    
    def cleanup_firefox_state(self) -> Dict[str, Any]:
        """Clean up Firefox state before starting a new task
        
        This method:
        1. Kills all Firefox processes
        2. Clears session restore data
        3. Resets crash logs
        4. Ensures clean state for next task
        
        Returns:
            Dictionary with cleanup results
        """
        logger.info("Starting Firefox state cleanup...")
        cleanup_results = {
            'processes_killed': 0,
            'session_cleared': False,
            'errors': [],
            'cleanup_time': None
        }
        
        start_time = time.time()
        
        try:
            # Step 1: Kill all Firefox processes
            firefox_pids = []
            for proc in psutil.process_iter(['pid', 'name']):
                try:
                    if 'firefox' in proc.info['name'].lower():
                        firefox_pids.append(proc.info['pid'])
                        proc.terminate()  # Try graceful termination first
                except (psutil.NoSuchProcess, psutil.AccessDenied) as e:
                    cleanup_results['errors'].append(f"Failed to terminate PID {proc.info['pid']}: {e}")
                    continue
            
            # Wait a moment for graceful termination
            time.sleep(1)
            
            # Force kill any remaining processes
            for pid in firefox_pids:
                try:
                    proc = psutil.Process(pid)
                    if proc.is_running():
                        proc.kill()  # Force kill if still running
                        cleanup_results['processes_killed'] += 1
                        logger.info(f"Force killed Firefox process: {pid}")
                except (psutil.NoSuchProcess, psutil.AccessDenied):
                    # Process already gone or no permission
                    cleanup_results['processes_killed'] += 1
            
            # Step 2: Clear Firefox session data and profile locks
            try:
                # Look for Firefox profile directories
                home_dir = os.path.expanduser("~")
                firefox_dirs = [
                    os.path.join(home_dir, ".mozilla", "firefox"),
                    os.path.join(home_dir, ".cache", "mozilla", "firefox"),
                    "/tmp/firefox-*"
                ]
                
                for firefox_dir in firefox_dirs:
                    if os.path.exists(firefox_dir):
                        # Clear session restore files
                        import glob
                        session_files = glob.glob(os.path.join(firefox_dir, "*", "sessionstore*"))
                        session_files.extend(glob.glob(os.path.join(firefox_dir, "*", "recovery*")))
                        
                        for session_file in session_files:
                            try:
                                os.remove(session_file)
                                logger.debug(f"Removed session file: {session_file}")
                                cleanup_results['session_cleared'] = True
                            except Exception as e:
                                cleanup_results['errors'].append(f"Failed to remove {session_file}: {e}")
                        
                        # CRITICAL: Remove profile lock files
                        profile_dirs = glob.glob(os.path.join(firefox_dir, "*"))
                        for profile_dir in profile_dirs:
                            if os.path.isdir(profile_dir):
                                # Remove lock files that prevent Firefox from starting
                                lock_files = ["lock", ".parentlock", "parent.lock"]
                                for lock_file in lock_files:
                                    lock_path = os.path.join(profile_dir, lock_file)
                                    if os.path.exists(lock_path):
                                        try:
                                            # Handle both files and symlinks
                                            if os.path.islink(lock_path):
                                                os.unlink(lock_path)
                                            else:
                                                os.remove(lock_path)
                                            logger.info(f"Removed Firefox lock file: {lock_path}")
                                            cleanup_results['locks_cleared'] = True
                                        except Exception as e:
                                            cleanup_results['errors'].append(f"Failed to remove lock {lock_path}: {e}")
                
            except Exception as e:
                cleanup_results['errors'].append(f"Failed to clear session data: {e}")
            
            # Step 3: Reset internal state
            self.crash_log = []
            self.restart_count = 0
            self.last_restart_time = None
            
            # Step 4: Give system time to release resources
            time.sleep(0.5)
            
            cleanup_results['cleanup_time'] = round(time.time() - start_time, 2)
            cleanup_results['success'] = True
            
            logger.info(f"Firefox cleanup completed: {cleanup_results['processes_killed']} processes killed, "
                       f"session cleared: {cleanup_results['session_cleared']}, "
                       f"time: {cleanup_results['cleanup_time']}s")
            
            return cleanup_results
            
        except Exception as e:
            cleanup_results['errors'].append(f"Unexpected error during cleanup: {e}")
            cleanup_results['success'] = False
            cleanup_results['cleanup_time'] = round(time.time() - start_time, 2)
            logger.error(f"Firefox cleanup failed: {e}")
            return cleanup_results
    
    def ensure_clean_state(self) -> bool:
        """Ensure Firefox is in a clean state for task execution
        
        Returns:
            True if state is clean and ready, False otherwise
        """
        try:
            # Check current health
            health = self.check_firefox_health()
            
            # Check for multiple Firefox processes (indicates problems)
            process_count = health.get('process_count', 0)
            if process_count > 5:  # More than 5 processes suggests issues
                logger.warning(f"Too many Firefox processes detected ({process_count}), performing cleanup...")
                cleanup_result = self.cleanup_firefox_state()
                return cleanup_result.get('success', False)
            
            # If Firefox is running with issues or has crashed windows, clean it up
            if (health['running'] and not health['healthy']) or health.get('crash_log_entries', 0) > 0:
                logger.info("Firefox in unhealthy state, performing cleanup...")
                cleanup_result = self.cleanup_firefox_state()
                return cleanup_result.get('success', False)
            
            # Check if profile is locked (common issue)
            if self._check_profile_locked():
                logger.warning("Firefox profile is locked, performing cleanup...")
                cleanup_result = self.cleanup_firefox_state()
                return cleanup_result.get('success', False)
            
            # If Firefox is running healthy, leave it alone
            if health['running'] and health['healthy']:
                logger.info("Firefox is running and healthy, no cleanup needed")
                return True
            
            # If Firefox is not running, we're good to go
            if not health['running']:
                logger.info("Firefox not running, state is clean")
                return True
                
            return True
            
        except Exception as e:
            logger.error(f"Error ensuring clean state: {e}")
            # Try cleanup as last resort
            self.cleanup_firefox_state()
            return False
    
    def _check_profile_locked(self) -> bool:
        """Check if Firefox profile is locked
        
        Returns:
            True if profile is locked, False otherwise
        """
        try:
            home_dir = os.path.expanduser("~")
            firefox_dir = os.path.join(home_dir, ".mozilla", "firefox")
            
            if os.path.exists(firefox_dir):
                import glob
                # Check all profile directories
                profile_dirs = glob.glob(os.path.join(firefox_dir, "*"))
                for profile_dir in profile_dirs:
                    if os.path.isdir(profile_dir):
                        # Check for lock files
                        lock_path = os.path.join(profile_dir, "lock")
                        parent_lock_path = os.path.join(profile_dir, ".parentlock")
                        
                        if os.path.exists(lock_path) or os.path.exists(parent_lock_path):
                            logger.debug(f"Found Firefox lock in profile: {profile_dir}")
                            return True
            
            return False
        except Exception as e:
            logger.warning(f"Error checking profile lock: {e}")
            return False


# Global instance
browser_monitor = BrowserHealthMonitor()