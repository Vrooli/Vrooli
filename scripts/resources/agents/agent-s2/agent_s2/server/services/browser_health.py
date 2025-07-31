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


# Global instance
browser_monitor = BrowserHealthMonitor()