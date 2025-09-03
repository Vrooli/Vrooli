#!/usr/bin/env python3
"""
Shared utilities for Vrooli app orchestrators
Contains safety mechanisms and constants used by orchestrator implementations
"""

import time
import logging
from typing import List
import psutil

# Constants for safety
MAX_APPS = 100
MAX_CONCURRENT_STARTS = 20  # Maximum apps starting simultaneously
MAX_PROCESSES_PER_APP = 5  # Max processes an app can spawn
ORCHESTRATOR_LOCK_FILE = "/tmp/vrooli-orchestrator.lock"
ORCHESTRATOR_PID_FILE = "/tmp/vrooli-orchestrator.pid"
APP_PID_DIR = "/tmp/vrooli-apps"
FORK_BOMB_THRESHOLD = 30  # Reduced threshold for safety
FORK_BOMB_WINDOW = 10  # Time window in seconds
# Adjusted for 32-core server with ~730 normal processes (415 kernel + 315 user)
SYSTEM_PROCESS_LIMIT = 2000  # Increased - your system normally runs 730+ processes
SYSTEM_WARNING_LIMIT = 1500  # Warn but don't stop


class ForkBombDetector:
    """Detects and prevents fork bomb scenarios by monitoring TOTAL system processes"""
    
    def __init__(self, threshold: int = FORK_BOMB_THRESHOLD, window: int = FORK_BOMB_WINDOW):
        self.threshold = threshold
        self.window = window
        self.process_starts: List[float] = []
        self.start_time = time.time()
        self.initial_process_count = len(psutil.pids())
        self.last_process_count = self.initial_process_count
    
    def record_start(self) -> bool:
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