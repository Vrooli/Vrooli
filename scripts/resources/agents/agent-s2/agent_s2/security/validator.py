"""Security Validation and Constraint Checking

Validates user inputs, file paths, and operations against security policies
before execution to prevent unauthorized access and malicious activities.
"""

import os
import re
import logging
from typing import List, Tuple, Optional, Dict, Any
from pathlib import Path, PurePath
from urllib.parse import urlparse

from ..config import Config, AgentMode

logger = logging.getLogger(__name__)


class SecurityValidationError(Exception):
    """Exception raised when security validation fails"""
    pass


class SecurityValidator:
    """Security validation and constraint checking"""
    
    def __init__(self):
        self.mode = Config.CURRENT_MODE
        self.forbidden_patterns = self._load_forbidden_patterns()
        
    def _load_forbidden_patterns(self) -> Dict[str, List[str]]:
        """Load forbidden patterns for different validation types"""
        return {
            "paths": [
                r"^/etc/passwd",
                r"^/etc/shadow",
                r"^/root/",
                r"\.ssh/",
                r"\.gnupg/",
                r"/proc/[0-9]+/mem$",
                r"\.key$",
                r"\.pem$",
                r"/var/log/auth\.log",
                r"/var/log/secure",
                r"/boot/",
                r"^/sys/",
                r"/proc/sys/",
            ],
            "commands": [
                r"sudo\s+",
                r"^su\s",
                r"passwd\s+",
                r"usermod\s+",
                r"userdel\s+", 
                r"groupadd\s+",
                r"rm\s+-rf\s+/",
                r"mkfs\.",
                r"fdisk\s+",
                r"dd\s+.*of=/dev/",
                r"nc\s+.*-l.*-e",
                r"bash\s+-i\s+>&",
                r"sh\s+-i\s+>&",
                r"/bin/sh.*>&",
                r"python.*-c.*exec",
                r"perl.*-e.*exec",
                r"eval\s*\(",
                r"exec\s*\(",
                r"\|.*sh$",
                r"wget.*\|.*sh",
                r"curl.*\|.*sh",
            ],
            "network": [
                r"^https?://10\.",
                r"^https?://172\.(1[6-9]|2[0-9]|3[0-1])\.",
                r"^https?://192\.168\.",
                r"^https?://127\.",
                r"^https?://localhost",
                r"^ftp://",
                r"^sftp://",
                r"^ssh://",
            ]
        }
    
    def validate_file_path(self, path: str, operation: str = "read") -> Tuple[bool, Optional[str]]:
        """Validate file path access according to current mode constraints
        
        Args:
            path: File path to validate
            operation: Type of operation (read, write, execute)
            
        Returns:
            Tuple of (is_valid, error_message)
        """
        try:
            normalized_path = os.path.normpath(os.path.abspath(path))
            
            # Check forbidden patterns
            for pattern in self.forbidden_patterns["paths"]:
                if re.search(pattern, normalized_path, re.IGNORECASE):
                    return False, f"Access to path matching pattern '{pattern}' is forbidden"
            
            # Mode-specific validation
            if self.mode == AgentMode.SANDBOX:
                return self._validate_sandbox_path(normalized_path, operation)
            elif self.mode == AgentMode.HOST:
                return self._validate_host_path(normalized_path, operation)
            
            return True, None
            
        except Exception as e:
            logger.error(f"Path validation error: {e}")
            return False, f"Path validation failed: {str(e)}"
    
    def _validate_sandbox_path(self, path: str, operation: str) -> Tuple[bool, Optional[str]]:
        """Validate path access for sandbox mode"""
        allowed_prefixes = [
            "/home/agents2",
            "/tmp",
            "/opt/agent-s2",
            "/usr/share/applications",  # For reading desktop files
            "/usr/share/pixmaps",       # For icons
            "/usr/bin",                 # For executing allowed applications
            "/bin"                      # For basic utilities
        ]
        
        # Allow read access to system libraries and configs
        if operation == "read":
            allowed_prefixes.extend([
                "/lib",
                "/lib64", 
                "/usr/lib",
                "/etc/fonts",
                "/etc/mime.types",
                "/usr/share/fonts",
                "/usr/share/themes",
                "/usr/share/icons"
            ])
        
        for prefix in allowed_prefixes:
            if path.startswith(prefix):
                return True, None
        
        return False, f"Sandbox mode: Access to '{path}' not allowed. Only container directories are accessible."
    
    def _validate_host_path(self, path: str, operation: str) -> Tuple[bool, Optional[str]]:
        """Validate path access for host mode"""
        # Check explicitly forbidden paths
        for forbidden in Config.HOST_FORBIDDEN_PATHS:
            if path.startswith(forbidden):
                return False, f"Host mode: Access to forbidden path '{forbidden}' is not allowed"
        
        # Allow container paths
        container_paths = [
            "/home/agents2",
            "/tmp", 
            "/opt/agent-s2"
        ]
        
        for container_path in container_paths:
            if path.startswith(container_path):
                return True, None
        
        # Check mounted directories
        mounted_dirs = Config.get_host_mounts()
        for mount in mounted_dirs:
            container_path = mount.get("container", "")
            if path.startswith(container_path):
                # Check if operation is allowed
                mount_mode = mount.get("mode", "ro")
                if operation == "write" and "w" not in mount_mode:
                    return False, f"Write access not allowed to read-only mount '{container_path}'"
                return True, None
        
        # Allow read access to system files for host integration
        if operation == "read":
            system_read_paths = [
                "/usr/share/applications",
                "/usr/share/pixmaps",
                "/usr/share/icons",
                "/usr/share/themes",
                "/usr/share/fonts",
                "/usr/bin",
                "/bin",
                "/lib",
                "/lib64",
                "/usr/lib",
                "/etc/fonts",
                "/etc/mime.types"
            ]
            
            for system_path in system_read_paths:
                if path.startswith(system_path):
                    return True, None
        
        return False, f"Host mode: Path '{path}' is not in allowed mounted directories"
    
    def validate_command(self, command: str) -> Tuple[bool, Optional[str]]:
        """Validate command execution for security risks
        
        Args:
            command: Command string to validate
            
        Returns:
            Tuple of (is_valid, error_message)
        """
        try:
            # Check against forbidden command patterns
            for pattern in self.forbidden_patterns["commands"]:
                if re.search(pattern, command, re.IGNORECASE | re.MULTILINE):
                    return False, f"Command contains forbidden pattern: '{pattern}'"
            
            # Mode-specific validation
            if self.mode == AgentMode.SANDBOX:
                return self._validate_sandbox_command(command)
            elif self.mode == AgentMode.HOST:
                return self._validate_host_command(command)
            
            return True, None
            
        except Exception as e:
            logger.error(f"Command validation error: {e}")
            return False, f"Command validation failed: {str(e)}"
    
    def _validate_sandbox_command(self, command: str) -> Tuple[bool, Optional[str]]:
        """Validate command for sandbox mode"""
        # Extract base command
        base_command = command.split()[0] if command.split() else ""
        
        # Allow specific applications
        allowed_commands = Config.SANDBOX_APPLICATIONS + [
            "bash", "sh", "ls", "cat", "echo", "grep", "find", "which",
            "python3", "python", "curl", "wget"
        ]
        
        if base_command in allowed_commands:
            return True, None
        
        # Check if it's a path-based command
        if base_command.startswith("/"):
            is_valid, error = self.validate_file_path(base_command, "execute")
            if not is_valid:
                return False, f"Command execution not allowed: {error}"
            return True, None
        
        return False, f"Sandbox mode: Command '{base_command}' is not in allowed application list"
    
    def _validate_host_command(self, command: str) -> Tuple[bool, Optional[str]]:
        """Validate command for host mode"""
        # Extract base command
        base_command = command.split()[0] if command.split() else ""
        
        # Check allowed applications configuration
        allowed_apps = Config.get_allowed_applications()
        
        if "*" in allowed_apps:
            # All applications allowed, but still check forbidden patterns
            return True, None
        
        if base_command in allowed_apps:
            return True, None
        
        # Check if it's a system command that should be generally allowed
        system_commands = [
            "ls", "cat", "echo", "grep", "find", "which", "ps", "top",
            "df", "du", "free", "uname", "whoami", "pwd", "date"
        ]
        
        if base_command in system_commands:
            return True, None
        
        return False, f"Host mode: Application '{base_command}' is not in allowed application list"
    
    def validate_network_access(self, url: str) -> Tuple[bool, Optional[str]]:
        """Validate network access for URLs
        
        Args:
            url: URL to validate
            
        Returns:
            Tuple of (is_valid, error_message)
        """
        try:
            parsed = urlparse(url)
            
            # Check forbidden network patterns
            for pattern in self.forbidden_patterns["network"]:
                if re.search(pattern, url, re.IGNORECASE):
                    return False, f"Network access to '{url}' matches forbidden pattern: '{pattern}'"
            
            # Mode-specific validation
            if self.mode == AgentMode.SANDBOX:
                return self._validate_sandbox_network(url, parsed)
            elif self.mode == AgentMode.HOST:
                return self._validate_host_network(url, parsed)
            
            return True, None
            
        except Exception as e:
            logger.error(f"Network validation error: {e}")
            return False, f"Network validation failed: {str(e)}"
    
    def _validate_sandbox_network(self, url: str, parsed) -> Tuple[bool, Optional[str]]:
        """Validate network access for sandbox mode"""
        # Sandbox mode blocks localhost/internal networks for security
        if parsed.hostname in ["localhost", "127.0.0.1", "::1"]:
            return False, "Sandbox mode: Localhost access is blocked for security"
        
        # Block private IP ranges
        if parsed.hostname:
            if (parsed.hostname.startswith("10.") or 
                parsed.hostname.startswith("192.168.") or
                any(parsed.hostname.startswith(f"172.{i}.") for i in range(16, 32))):
                return False, "Sandbox mode: Private network access is blocked"
        
        # Allow public internet access
        return True, None
    
    def _validate_host_network(self, url: str, parsed) -> Tuple[bool, Optional[str]]:
        """Validate network access for host mode"""
        # Host mode allows more network access but still has some restrictions
        
        # Could add specific restrictions here if needed
        # For now, allow all network access in host mode
        return True, None
    
    def validate_input_size(self, input_data: str, max_size: int = 10000) -> Tuple[bool, Optional[str]]:
        """Validate input size to prevent DoS attacks
        
        Args:
            input_data: Input string to validate
            max_size: Maximum allowed size in characters
            
        Returns:
            Tuple of (is_valid, error_message)
        """
        if len(input_data) > max_size:
            return False, f"Input size {len(input_data)} exceeds maximum allowed size {max_size}"
        
        return True, None
    
    def validate_resource_limits(self, operation: str, **kwargs) -> Tuple[bool, Optional[str]]:
        """Validate resource usage limits
        
        Args:
            operation: Type of operation
            **kwargs: Operation-specific parameters
            
        Returns:
            Tuple of (is_valid, error_message)
        """
        try:
            if operation == "file_upload":
                file_size = kwargs.get("file_size", 0)
                max_size = Config.HOST_MAX_MOUNT_SIZE_GB * 1024 * 1024 * 1024  # Convert GB to bytes
                
                if file_size > max_size:
                    return False, f"File size {file_size} bytes exceeds limit {max_size} bytes"
            
            elif operation == "process_creation":
                # Could check process limits, memory usage, etc.
                pass
            
            return True, None
            
        except Exception as e:
            logger.error(f"Resource validation error: {e}")
            return False, f"Resource validation failed: {str(e)}"
    
    def get_security_constraints(self) -> Dict[str, Any]:
        """Get current security constraints for the mode"""
        return {
            "mode": self.mode.value,
            "forbidden_path_patterns": self.forbidden_patterns["paths"],
            "forbidden_command_patterns": self.forbidden_patterns["commands"],
            "forbidden_network_patterns": self.forbidden_patterns["network"],
            "allowed_applications": Config.get_allowed_applications() if self.mode == AgentMode.HOST else Config.SANDBOX_APPLICATIONS,
            "host_constraints": {
                "forbidden_paths": Config.HOST_FORBIDDEN_PATHS,
                "mounted_directories": Config.get_host_mounts(),
                "max_mount_size_gb": Config.HOST_MAX_MOUNT_SIZE_GB
            } if self.mode == AgentMode.HOST else None
        }
    
    def check_security_policy(self, action: str, target: str, details: Optional[Dict[str, Any]] = None) -> Tuple[bool, Optional[str]]:
        """Comprehensive security policy check
        
        Args:
            action: Action being performed
            target: Target of the action (file, URL, command, etc.)
            details: Additional details about the action
            
        Returns:
            Tuple of (is_allowed, error_message)
        """
        try:
            details = details or {}
            
            # Route to appropriate validator based on action type
            if action in ["file_read", "file_write", "file_execute"]:
                operation = action.split("_")[1]  # extract read/write/execute
                return self.validate_file_path(target, operation)
            
            elif action in ["command_execute", "process_start"]:
                return self.validate_command(target)
            
            elif action in ["network_request", "url_access"]:
                return self.validate_network_access(target)
            
            elif action == "input_validation":
                max_size = details.get("max_size", 10000)
                return self.validate_input_size(target, max_size)
            
            elif action.startswith("resource_"):
                return self.validate_resource_limits(action, **details)
            
            else:
                # Default allow for unrecognized actions, but log
                logger.warning(f"Unknown action type for security validation: {action}")
                return True, None
                
        except Exception as e:
            logger.error(f"Security policy check failed: {e}")
            return False, f"Security policy check failed: {str(e)}"


# Global validator instance
_security_validator = None

def get_security_validator() -> SecurityValidator:
    """Get global security validator instance"""
    global _security_validator
    if _security_validator is None:
        _security_validator = SecurityValidator()
    return _security_validator