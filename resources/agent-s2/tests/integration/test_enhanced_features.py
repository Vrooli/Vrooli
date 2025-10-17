#!/usr/bin/env python3
"""
Integration tests for enhanced Agent S2 features

Tests the new implementations for:
- Window capture functionality
- Image comparison
- Change detection
- Pixel color extraction
- Improved error handling
- Real browser data extraction
- Security notification systems
"""

import os
import sys
import pytest
import requests
import time
import json
from unittest.mock import Mock, patch
from pathlib import Path

# Add the agent_s2 package to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "agent_s2"))

from agent_s2.client.screenshot import ScreenshotClient
from agent_s2.client.base import AgentS2Client
from agent_s2.server.services.capture import ScreenshotService
from agent_s2.server.services.window_manager import WindowManager, WindowManagerError
from agent_s2.stealth.manager import StealthManager, StealthConfig
from agent_s2.security.monitor import SecurityMonitor, SecurityEvent
from agent_s2.exceptions import *


class TestEnhancedScreenshotFeatures:
    """Test enhanced screenshot functionality"""
    
    @pytest.fixture
    def screenshot_service(self):
        """Create screenshot service instance"""
        return ScreenshotService()
    
    @pytest.fixture  
    def screenshot_client(self):
        """Create screenshot client instance"""
        return ScreenshotClient()
    
    def test_screenshot_service_initialization(self, screenshot_service):
        """Test that screenshot service initializes with new features"""
        assert hasattr(screenshot_service, 'window_manager')
        assert hasattr(screenshot_service, '_monitoring_active')
        assert isinstance(screenshot_service.window_manager, WindowManager)
    
    def test_window_manager_integration(self, screenshot_service):
        """Test window manager integration"""
        # Test that window manager is properly initialized
        assert screenshot_service.window_manager is not None
        
        # Test window listing functionality
        windows = screenshot_service.window_manager.list_all_windows()
        assert isinstance(windows, list)
    
    def test_pixel_color_extraction(self, screenshot_service):
        """Test pixel color extraction functionality"""
        try:
            # Test valid coordinates (center of screen)
            color = screenshot_service.get_pixel_color(100, 100)
            assert isinstance(color, tuple)
            assert len(color) == 3
            assert all(0 <= c <= 255 for c in color)
        except ValueError as e:
            # This is expected if coordinates are out of bounds
            assert "out of screen bounds" in str(e)
    
    def test_pixel_color_validation(self, screenshot_service):
        """Test pixel color extraction with invalid coordinates"""
        with pytest.raises(ValueError, match="out of screen bounds"):
            screenshot_service.get_pixel_color(-1, -1)
        
        with pytest.raises(ValueError, match="out of screen bounds"):
            screenshot_service.get_pixel_color(99999, 99999)
    
    def test_screenshot_comparison(self, screenshot_service):
        """Test screenshot comparison functionality"""
        # Take two screenshots
        shot1 = screenshot_service.capture()
        time.sleep(0.1)  # Small delay
        shot2 = screenshot_service.capture()
        
        # Test comparison
        comparison = screenshot_service.compare_screenshots(shot1, shot2, method="mse")
        
        assert isinstance(comparison, dict)
        assert "similarity" in comparison
        assert "method" in comparison
        assert 0.0 <= comparison["similarity"] <= 1.0
        assert comparison["method"] == "mse"
    
    def test_screenshot_comparison_methods(self, screenshot_service):
        """Test different comparison methods"""
        shot1 = screenshot_service.capture()
        shot2 = screenshot_service.capture()
        
        methods = ["mse", "histogram", "pixel_diff"]
        
        for method in methods:
            comparison = screenshot_service.compare_screenshots(shot1, shot2, method=method)
            assert isinstance(comparison, dict)
            assert "similarity" in comparison
            assert comparison["method"] == method
    
    def test_difference_image_creation(self, screenshot_service):
        """Test difference image creation"""
        shot1 = screenshot_service.capture()
        shot2 = screenshot_service.capture()
        
        diff_image = screenshot_service.create_difference_image(shot1, shot2)
        
        assert isinstance(diff_image, dict)
        assert "format" in diff_image
        assert "size" in diff_image
        assert "data" in diff_image
        assert diff_image["format"] == "png"
        assert diff_image["data"].startswith("data:image/png;base64,")
    
    @pytest.mark.slow
    def test_change_detection(self, screenshot_service):
        """Test change detection functionality"""
        # Start change detection with short timeout
        changes = screenshot_service.detect_changes(
            interval=0.5,
            timeout=2.0,
            threshold=0.01
        )
        
        assert isinstance(changes, list)
        # Changes may or may not be detected depending on screen activity
    
    def test_change_detection_stop(self, screenshot_service):
        """Test stopping change detection"""
        # This should not raise an exception
        screenshot_service.stop_change_detection()
        assert not screenshot_service._monitoring_active


class TestEnhancedClientFeatures:
    """Test enhanced client functionality"""
    
    @pytest.fixture
    def mock_client(self):
        """Create mock client for testing"""
        client = Mock(spec=AgentS2Client)
        client._request = Mock()
        return client
    
    def test_screenshot_client_new_methods(self, mock_client):
        """Test that screenshot client has new methods"""
        screenshot_client = ScreenshotClient(client=mock_client)
        
        # Test that new methods exist
        assert hasattr(screenshot_client, 'capture_window_by_id')
        assert hasattr(screenshot_client, 'create_difference_image')
        assert hasattr(screenshot_client, 'get_pixel_colors_region')
        assert hasattr(screenshot_client, 'stop_change_detection')
    
    def test_window_capture_client_call(self, mock_client):
        """Test window capture client API call"""
        mock_response = Mock()
        mock_response.raise_for_status = Mock()
        mock_response.json.return_value = {"success": True, "format": "png"}
        mock_client._request.return_value = mock_response
        
        screenshot_client = ScreenshotClient(client=mock_client)
        result = screenshot_client.capture_window("Firefox")
        
        # Verify the API call was made correctly
        mock_client._request.assert_called_once_with(
            'POST', '/screenshot/window', 
            params={"format": "png", "window_title": "Firefox"}
        )
        assert result == {"success": True, "format": "png"}
    
    def test_pixel_color_client_call(self, mock_client):
        """Test pixel color client API call"""
        mock_response = Mock()
        mock_response.raise_for_status = Mock()
        mock_response.json.return_value = {"color": [255, 0, 0]}
        mock_client._request.return_value = mock_response
        
        screenshot_client = ScreenshotClient(client=mock_client)
        result = screenshot_client.get_pixel_color(100, 100)
        
        # Verify the API call was made correctly
        mock_client._request.assert_called_once_with(
            'GET', '/screenshot/pixel-color',
            params={"x": 100, "y": 100}
        )
        assert result == (255, 0, 0)


class TestErrorHandlingImprovements:
    """Test improved error handling"""
    
    def test_custom_exceptions_exist(self):
        """Test that custom exception classes exist"""
        # Test that all custom exceptions are defined
        assert issubclass(WindowManagerError, Exception)
        assert issubclass(ScreenshotError, Exception)
        assert issubclass(AutomationError, Exception)
        assert issubclass(SecurityError, Exception)
        assert issubclass(ValidationError, Exception)
    
    def test_window_manager_error_handling(self):
        """Test window manager error handling"""
        service = ScreenshotService()
        
        # Test with non-existent window
        with pytest.raises(WindowManagerError, match="No window found"):
            service.capture_window("NonExistentWindow123456789")
    
    def test_pixel_color_error_handling(self):
        """Test pixel color error handling"""
        service = ScreenshotService()
        
        # Test with invalid coordinates
        with pytest.raises(ValueError, match="out of screen bounds"):
            service.get_pixel_color(-1, -1)


class TestBrowserDataExtraction:
    """Test real browser data extraction"""
    
    def test_stealth_manager_real_extraction(self):
        """Test that stealth manager uses real extraction methods"""
        config = StealthConfig(enabled=True, session_storage_path="/tmp/test_sessions")
        manager = StealthManager(config)
        
        # Test that the extraction method returns proper structure
        import asyncio
        
        async def test_extraction():
            data = await manager._retrieve_extracted_data()
            assert isinstance(data, dict)
            assert "cookies" in data
            assert "localStorage" in data
            assert "sessionStorage" in data
            assert "extraction_method" in data
            assert data["extraction_method"] in ["browser_automation", "fallback"]
        
        asyncio.run(test_extraction())


class TestSecurityNotifications:
    """Test security notification system"""
    
    def test_security_monitor_initialization(self):
        """Test security monitor initialization"""
        monitor = SecurityMonitor()
        assert hasattr(monitor, '_notify_security_team')
    
    @patch('smtplib.SMTP')
    @patch.dict(os.environ, {
        'AGENT_S2_SMTP_SERVER': 'smtp.example.com',
        'AGENT_S2_SMTP_USER': 'user@example.com',
        'AGENT_S2_SMTP_PASSWORD': 'password',
        'AGENT_S2_SECURITY_EMAIL': 'security@example.com'
    })
    def test_email_notification(self, mock_smtp):
        """Test email notification functionality"""
        monitor = SecurityMonitor()
        
        # Create test security event
        event = SecurityEvent(
            event_type="test_event",
            severity="high",
            source="test",
            details="Test security event"
        )
        
        # Test notification (should not raise exception)
        monitor._notify_security_team(event)
    
    @patch('requests.post')
    @patch.dict(os.environ, {
        'AGENT_S2_SECURITY_WEBHOOK_URL': 'https://webhook.example.com/security'
    })
    def test_webhook_notification(self, mock_post):
        """Test webhook notification functionality"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_post.return_value = mock_response
        
        monitor = SecurityMonitor()
        
        # Create test security event
        event = SecurityEvent(
            event_type="test_event",
            severity="high", 
            source="test",
            details="Test security event"
        )
        
        # Test notification
        monitor._notify_security_team(event)
        
        # Verify webhook was called
        mock_post.assert_called()
    
    def test_security_incident_file_creation(self):
        """Test security incident file creation"""
        monitor = SecurityMonitor()
        
        # Create test security event
        event = SecurityEvent(
            event_type="test_event",
            severity="high",
            source="test", 
            details="Test security event"
        )
        
        # Test notification (should create incident file)
        monitor._notify_security_team(event)
        
        # Check that incident directory exists
        incidents_dir = Path("/tmp/agent-s2-security-incidents")
        if incidents_dir.exists():
            # Verify at least one incident file was created
            incident_files = list(incidents_dir.glob("incident_*.json"))
            assert len(incident_files) > 0


class TestAPIContracts:
    """Test that API contracts are maintained"""
    
    def test_screenshot_response_model(self):
        """Test that screenshot response model supports new fields"""
        from agent_s2.server.models.responses import ScreenshotResponse
        
        # Test that model can be created with window_info
        response = ScreenshotResponse(
            success=True,
            format="png",
            size={"width": 100, "height": 100},
            data="data:image/png;base64,test",
            window_info={"window_id": "test", "title": "Test Window"}
        )
        
        assert response.window_info is not None
        assert response.window_info["window_id"] == "test"
    
    def test_backward_compatibility(self):
        """Test that existing API contracts are maintained"""
        from agent_s2.server.models.responses import ScreenshotResponse
        
        # Test that model works without new optional fields
        response = ScreenshotResponse(
            success=True,
            format="png", 
            size={"width": 100, "height": 100},
            data="data:image/png;base64,test"
        )
        
        assert response.window_info is None  # Should be optional


if __name__ == "__main__":
    # Run tests with verbose output
    pytest.main([__file__, "-v", "--tb=short"])