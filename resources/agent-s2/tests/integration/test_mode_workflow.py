#!/usr/bin/env python3
"""
Integration tests for Agent S2 dual-mode workflow

Tests complete workflows including:
- Installation and setup
- Mode switching
- Security enforcement
- API functionality
- Error handling and recovery
"""

import os
import sys
import time
import json
import pytest
import requests
import subprocess
from typing import Dict, Any, Optional
from pathlib import Path

# Add the agent_s2 package to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "agent_s2"))

from agent_s2.config import Config, AgentMode


class AgentS2IntegrationTest:
    """Integration test fixture for Agent S2"""
    
    def __init__(self):
        self.base_url = "http://localhost:4113"
        self.script_dir = Path(__file__).parent.parent.parent
        self.manage_script = self.script_dir / "manage.sh"
        self.initial_mode = None
        self.container_was_running = False
        
    def setup(self):
        """Set up integration test environment"""
        # Check if Agent S2 is already running
        self.container_was_running = self.is_container_running()
        
        if self.container_was_running:
            # Store current mode
            try:
                response = requests.get(f"{self.base_url}/modes/current", timeout=5)
                if response.status_code == 200:
                    self.initial_mode = response.json().get("current_mode")
            except requests.RequestException:
                pass
        
        # Ensure Agent S2 is running for tests
        if not self.is_agent_healthy():
            self.start_agent_sandbox()
    
    def teardown(self):
        """Clean up integration test environment"""
        # Restore initial state if we started Agent S2
        if not self.container_was_running:
            self.stop_agent()
        elif self.initial_mode:
            self.switch_to_mode(self.initial_mode)
    
    def is_container_running(self) -> bool:
        """Check if Agent S2 container is running"""
        try:
            result = subprocess.run(
                ["docker", "ps", "--filter", "name=agent-s2", "--format", "{{.Names}}"],
                capture_output=True,
                text=True,
                timeout=10
            )
            return "agent-s2" in result.stdout
        except (subprocess.TimeoutExpired, subprocess.SubprocessError):
            return False
    
    def is_agent_healthy(self) -> bool:
        """Check if Agent S2 API is healthy"""
        try:
            response = requests.get(f"{self.base_url}/health", timeout=5)
            return response.status_code == 200
        except requests.RequestException:
            return False
    
    def start_agent_sandbox(self) -> bool:
        """Start Agent S2 in sandbox mode"""
        try:
            result = subprocess.run(
                [str(self.manage_script), "--action", "start", "--mode", "sandbox", "--yes", "yes"],
                capture_output=True,
                text=True,
                timeout=120  # 2 minutes timeout
            )
            
            if result.returncode == 0:
                # Wait for agent to be ready
                return self.wait_for_agent_ready(timeout=60)
            else:
                print(f"Failed to start Agent S2: {result.stderr}")
                return False
                
        except (subprocess.TimeoutExpired, subprocess.SubprocessError) as e:
            print(f"Error starting Agent S2: {e}")
            return False
    
    def stop_agent(self) -> bool:
        """Stop Agent S2"""
        try:
            result = subprocess.run(
                [str(self.manage_script), "--action", "stop", "--yes", "yes"],
                capture_output=True,
                text=True,
                timeout=30
            )
            return result.returncode == 0
        except (subprocess.TimeoutExpired, subprocess.SubprocessError):
            return False
    
    def switch_to_mode(self, mode: str) -> bool:
        """Switch Agent S2 to specified mode"""
        try:
            # Use API to switch mode
            response = requests.post(
                f"{self.base_url}/modes/switch",
                json={"new_mode": mode},
                timeout=30
            )
            
            if response.status_code == 200:
                # Wait for mode switch to complete
                time.sleep(10)
                return self.wait_for_agent_ready(timeout=30)
            else:
                # Try using script
                result = subprocess.run(
                    [str(self.manage_script), "--action", "switch-mode", "--target-mode", mode, "--yes", "yes"],
                    capture_output=True,
                    text=True,
                    timeout=60
                )
                return result.returncode == 0
                
        except (requests.RequestException, subprocess.TimeoutExpired, subprocess.SubprocessError):
            return False
    
    def wait_for_agent_ready(self, timeout: int = 30) -> bool:
        """Wait for Agent S2 to be ready"""
        for _ in range(timeout):
            if self.is_agent_healthy():
                return True
            time.sleep(1)
        return False
    
    def get_current_mode(self) -> Optional[str]:
        """Get current Agent S2 mode"""
        try:
            response = requests.get(f"{self.base_url}/modes/current", timeout=5)
            if response.status_code == 200:
                return response.json().get("current_mode")
        except requests.RequestException:
            pass
        return None


@pytest.fixture(scope="module")
def integration_fixture():
    """Pytest fixture for integration testing"""
    fixture = AgentS2IntegrationTest()
    fixture.setup()
    yield fixture
    fixture.teardown()


class TestInstallationWorkflow:
    """Test installation and setup workflow"""
    
    @pytest.mark.slow
    def test_fresh_installation(self, integration_fixture):
        """Test fresh installation workflow"""
        # This test assumes we can do a fresh install
        # In practice, this might require stopping existing containers
        
        # Stop existing Agent S2 if running
        integration_fixture.stop_agent()
        
        # Test installation
        result = subprocess.run([
            str(integration_fixture.manage_script),
            "--action", "install",
            "--mode", "sandbox",
            "--yes", "yes"
        ], capture_output=True, text=True, timeout=180)
        
        # Installation might fail if already exists, that's ok
        if result.returncode == 0:
            assert integration_fixture.wait_for_agent_ready(timeout=60)
            assert integration_fixture.get_current_mode() == "sandbox"
    
    @pytest.mark.slow  
    def test_host_mode_installation(self, integration_fixture):
        """Test installation with host mode enabled"""
        # Set environment for host mode
        env = os.environ.copy()
        env["AGENT_S2_HOST_MODE_ENABLED"] = "true"
        
        # Stop existing container
        integration_fixture.stop_agent()
        
        # Install with host mode enabled
        result = subprocess.run([
            str(integration_fixture.manage_script),
            "--action", "install", 
            "--mode", "sandbox",  # Start in sandbox, but enable host mode
            "--yes", "yes"
        ], capture_output=True, text=True, timeout=180, env=env)
        
        # Verify Agent S2 is running
        if result.returncode == 0:
            assert integration_fixture.wait_for_agent_ready(timeout=60)
            
            # Check that host mode is available
            response = requests.get(f"{integration_fixture.base_url}/modes/current", timeout=5)
            if response.status_code == 200:
                data = response.json()
                assert "can_switch_to_host" in data


class TestModeWorkflow:
    """Test mode switching workflow"""
    
    def test_sandbox_mode_functionality(self, integration_fixture):
        """Test sandbox mode basic functionality"""
        # Ensure we're in sandbox mode
        current_mode = integration_fixture.get_current_mode()
        if current_mode != "sandbox":
            integration_fixture.switch_to_mode("sandbox")
        
        # Test basic API endpoints
        response = requests.get(f"{integration_fixture.base_url}/health")
        assert response.status_code == 200
        
        health_data = response.json()
        assert health_data["status"] == "healthy"
        assert health_data.get("mode_info", {}).get("current_mode") == "sandbox"
        
        # Test screenshot functionality
        response = requests.get(f"{integration_fixture.base_url}/screenshot")
        assert response.status_code == 200
        
        screenshot_data = response.json()
        assert screenshot_data["success"] == True
        assert "data" in screenshot_data
    
    def test_mode_information_consistency(self, integration_fixture):
        """Test that mode information is consistent across endpoints"""
        current_mode = integration_fixture.get_current_mode()
        
        # Get mode info from different endpoints
        endpoints = [
            "/health",
            "/capabilities", 
            "/modes/current",
            "/modes/environment"
        ]
        
        mode_info = {}
        for endpoint in endpoints:
            response = requests.get(f"{integration_fixture.base_url}{endpoint}")
            if response.status_code == 200:
                data = response.json()
                
                # Extract mode information from different response formats
                if endpoint == "/health":
                    mode_info[endpoint] = data.get("mode_info", {}).get("current_mode")
                elif endpoint == "/capabilities":
                    mode_info[endpoint] = data.get("mode_info", {}).get("current_mode")
                elif endpoint == "/modes/current":
                    mode_info[endpoint] = data.get("current_mode")
                elif endpoint == "/modes/environment":
                    mode_info[endpoint] = data.get("mode")
        
        # All endpoints should report the same mode
        reported_modes = [mode for mode in mode_info.values() if mode is not None]
        assert len(set(reported_modes)) <= 1, f"Inconsistent mode reporting: {mode_info}"
    
    @pytest.mark.slow
    def test_mode_switching_workflow(self, integration_fixture):
        """Test complete mode switching workflow"""
        # Check if host mode is available
        response = requests.get(f"{integration_fixture.base_url}/modes/current")
        if response.status_code != 200:
            pytest.skip("Cannot get current mode information")
        
        current_data = response.json()
        if not current_data.get("can_switch_to_host", False):
            pytest.skip("Host mode not available for testing")
        
        initial_mode = current_data["current_mode"]
        target_mode = "host" if initial_mode == "sandbox" else "sandbox"
        
        # Attempt mode switch
        switch_response = requests.post(
            f"{integration_fixture.base_url}/modes/switch",
            json={"new_mode": target_mode},
            timeout=30
        )
        
        if switch_response.status_code == 200:
            # Wait for switch to complete
            time.sleep(15)
            
            # Verify mode changed
            if integration_fixture.wait_for_agent_ready(timeout=30):
                new_mode = integration_fixture.get_current_mode()
                assert new_mode == target_mode, f"Mode switch failed: expected {target_mode}, got {new_mode}"
                
                # Switch back to original mode
                integration_fixture.switch_to_mode(initial_mode)
        else:
            pytest.skip(f"Mode switch request failed: {switch_response.text}")


class TestSecurityWorkflow:
    """Test security enforcement workflow"""
    
    def test_security_endpoint_access(self, integration_fixture):
        """Test access to security information endpoints"""
        # Test security info endpoint
        response = requests.get(f"{integration_fixture.base_url}/modes/security")
        assert response.status_code == 200
        
        security_data = response.json()
        assert "security_level" in security_data
        assert "constraints" in security_data
        assert "limitations" in security_data
        
        current_mode = integration_fixture.get_current_mode()
        if current_mode == "sandbox":
            assert security_data["security_level"] == "high"
        elif current_mode == "host":
            assert security_data["security_level"] == "medium"
    
    def test_application_discovery(self, integration_fixture):
        """Test application discovery in current mode"""
        response = requests.get(f"{integration_fixture.base_url}/modes/applications")
        assert response.status_code == 200
        
        apps_data = response.json()
        assert "applications" in apps_data
        assert "applications_count" in apps_data
        assert isinstance(apps_data["applications"], list)
        assert apps_data["applications_count"] == len(apps_data["applications"])
        
        # Verify application data structure
        if apps_data["applications"]:
            app = apps_data["applications"][0]
            assert "name" in app
            assert "command" in app
            assert "category" in app
    
    def test_security_constraints_mode_specific(self, integration_fixture):
        """Test that security constraints are mode-specific"""
        current_mode = integration_fixture.get_current_mode()
        
        # Get environment info
        response = requests.get(f"{integration_fixture.base_url}/modes/environment")
        assert response.status_code == 200
        
        env_data = response.json()
        security_constraints = env_data.get("security_constraints", {})
        
        if current_mode == "sandbox":
            # Sandbox should have strict constraints
            assert security_constraints.get("isolation_level") == "high"
            assert security_constraints.get("host_access") == False
        elif current_mode == "host":
            # Host mode should have medium constraints
            assert security_constraints.get("isolation_level") == "medium"
            assert security_constraints.get("host_access") == True


class TestErrorHandling:
    """Test error handling and recovery"""
    
    def test_invalid_mode_switch(self, integration_fixture):
        """Test handling of invalid mode switch requests"""
        # Try to switch to invalid mode
        response = requests.post(
            f"{integration_fixture.base_url}/modes/switch",
            json={"new_mode": "invalid_mode"}
        )
        
        assert response.status_code == 400
        error_data = response.json()
        assert "detail" in error_data
        assert "invalid" in error_data["detail"].lower()
    
    def test_api_error_responses(self, integration_fixture):
        """Test API error responses are properly formatted"""
        # Test non-existent endpoint
        response = requests.get(f"{integration_fixture.base_url}/nonexistent")
        assert response.status_code == 404
        
        # Test invalid application lookup
        response = requests.get(f"{integration_fixture.base_url}/modes/applications/nonexistent-app")
        assert response.status_code == 404
        
        error_data = response.json()
        assert "detail" in error_data
    
    def test_agent_recovery_after_error(self, integration_fixture):
        """Test that Agent S2 recovers properly after errors"""
        # Verify agent is healthy before test
        assert integration_fixture.is_agent_healthy()
        
        # Cause some API errors
        requests.post(f"{integration_fixture.base_url}/modes/switch", json={"new_mode": "invalid"})
        requests.get(f"{integration_fixture.base_url}/nonexistent")
        
        # Agent should still be healthy
        assert integration_fixture.is_agent_healthy()
        
        # Basic functionality should still work
        response = requests.get(f"{integration_fixture.base_url}/health")
        assert response.status_code == 200


class TestPerformance:
    """Test performance characteristics"""
    
    def test_api_response_times(self, integration_fixture):
        """Test that API responses are reasonably fast"""
        endpoints = [
            "/health",
            "/capabilities",
            "/modes/current",
            "/modes/applications"
        ]
        
        for endpoint in endpoints:
            start_time = time.time()
            response = requests.get(f"{integration_fixture.base_url}{endpoint}")
            response_time = time.time() - start_time
            
            assert response.status_code == 200
            assert response_time < 5.0, f"Endpoint {endpoint} took {response_time:.2f}s (too slow)"
    
    def test_concurrent_requests(self, integration_fixture):
        """Test handling of concurrent requests"""
        import threading
        import queue
        
        results = queue.Queue()
        
        def make_request():
            try:
                response = requests.get(f"{integration_fixture.base_url}/health", timeout=10)
                results.put(("success", response.status_code))
            except Exception as e:
                results.put(("error", str(e)))
        
        # Start multiple concurrent requests
        threads = []
        for _ in range(5):
            thread = threading.Thread(target=make_request)
            thread.start()
            threads.append(thread)
        
        # Wait for all threads to complete
        for thread in threads:
            thread.join(timeout=15)
        
        # Check results
        success_count = 0
        while not results.empty():
            result_type, result_value = results.get()
            if result_type == "success" and result_value == 200:
                success_count += 1
        
        # At least most requests should succeed
        assert success_count >= 3, f"Only {success_count}/5 concurrent requests succeeded"


def run_integration_tests():
    """Run integration tests with proper configuration"""
    # Set test environment
    os.environ.setdefault("AGENT_S2_MODE", "sandbox")
    
    # Run pytest with appropriate settings
    exit_code = pytest.main([
        __file__,
        "-v",
        "--tb=short",
        "-x",  # Stop on first failure
        "--durations=10",  # Show slowest tests
        "-m", "not slow"  # Skip slow tests by default
    ])
    
    return exit_code


def run_full_integration_tests():
    """Run full integration tests including slow tests"""
    os.environ.setdefault("AGENT_S2_MODE", "sandbox")
    
    exit_code = pytest.main([
        __file__,
        "-v",
        "--tb=short",
        "--durations=20"
    ])
    
    return exit_code


if __name__ == "__main__":
    if "--full" in sys.argv:
        exit_code = run_full_integration_tests()
    else:
        exit_code = run_integration_tests()
    sys.exit(exit_code)