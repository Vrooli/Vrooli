#!/usr/bin/env python3
"""
Unit tests for Agent S2 screenshot capture service

Tests capture service functionality including:
- Screenshot capture with different formats
- Region-based screenshots
- Input validation
- Error handling
"""

import os
import sys
import pytest
from unittest.mock import Mock, patch, MagicMock
from pathlib import Path
from PIL import Image
import io

# Add the agent_s2 package to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "agent_s2"))

from agent_s2.server.services.capture import ScreenshotService


class TestScreenshotService:
    """Test screenshot capture service functionality"""
    
    def setup_method(self):
        """Set up test fixtures"""
        self.service = ScreenshotService()
    
    def test_init(self):
        """Test service initialization"""
        service = ScreenshotService()
        assert service is not None
    
    @patch('agent_s2.server.services.capture.pyautogui')
    def test_capture_png_format(self, mock_pyautogui):
        """Test PNG screenshot capture"""
        # Mock pyautogui.screenshot to return a PIL Image
        mock_image = Mock(spec=Image.Image)
        mock_image.size = (1920, 1080)
        mock_pyautogui.screenshot.return_value = mock_image
        
        # Mock the image save method
        mock_buffer = io.BytesIO()
        mock_image.save = Mock()
        
        with patch('io.BytesIO', return_value=mock_buffer):
            with patch('base64.b64encode', return_value=b'base64data'):
                result = self.service.capture(format="png")
        
        # Verify the result structure
        assert isinstance(result, dict)
        assert 'image' in result
        assert 'metadata' in result
        assert result['metadata']['format'] == 'png'
        assert result['metadata']['size'] == [1920, 1080]
        
        # Verify pyautogui was called correctly
        mock_pyautogui.screenshot.assert_called_once_with(region=None)
    
    @patch('agent_s2.server.services.capture.pyautogui')
    def test_capture_jpeg_format(self, mock_pyautogui):
        """Test JPEG screenshot capture"""
        mock_image = Mock(spec=Image.Image)
        mock_image.size = (1920, 1080)
        mock_pyautogui.screenshot.return_value = mock_image
        
        mock_buffer = io.BytesIO()
        mock_image.save = Mock()
        
        with patch('io.BytesIO', return_value=mock_buffer):
            with patch('base64.b64encode', return_value=b'base64data'):
                result = self.service.capture(format="jpeg", quality=85)
        
        assert result['metadata']['format'] == 'jpeg'
        mock_image.save.assert_called()
    
    @patch('agent_s2.server.services.capture.pyautogui')
    def test_capture_with_region(self, mock_pyautogui):
        """Test screenshot capture with specific region"""
        mock_image = Mock(spec=Image.Image)
        mock_image.size = (500, 300)
        mock_pyautogui.screenshot.return_value = mock_image
        
        region = [100, 200, 500, 300]
        
        mock_buffer = io.BytesIO()
        mock_image.save = Mock()
        
        with patch('io.BytesIO', return_value=mock_buffer):
            with patch('base64.b64encode', return_value=b'base64data'):
                result = self.service.capture(format="png", region=region)
        
        # Verify region was passed to pyautogui
        mock_pyautogui.screenshot.assert_called_once_with(region=tuple(region))
        assert result['metadata']['region'] == region
    
    def test_capture_invalid_format(self):
        """Test error handling for invalid format"""
        with pytest.raises(ValueError, match="Invalid format: gif"):
            self.service.capture(format="gif")
    
    def test_capture_invalid_quality_low(self):
        """Test error handling for quality too low"""
        with pytest.raises(ValueError, match="Quality must be 1-100"):
            self.service.capture(format="jpeg", quality=0)
    
    def test_capture_invalid_quality_high(self):
        """Test error handling for quality too high"""
        with pytest.raises(ValueError, match="Quality must be 1-100"):
            self.service.capture(format="jpeg", quality=101)
    
    def test_capture_invalid_region_format(self):
        """Test error handling for invalid region format"""
        with pytest.raises(ValueError, match="Region must be a list of 4 integers"):
            self.service.capture(region=[100, 200, 300])  # Only 3 elements
    
    def test_capture_invalid_region_type(self):
        """Test error handling for invalid region values"""
        with pytest.raises(ValueError, match="Region values must be positive integers"):
            self.service.capture(region=[100, 200, -50, 300])  # Negative width
    
    @patch('agent_s2.server.services.capture.pyautogui')
    def test_capture_jpg_format_normalized(self, mock_pyautogui):
        """Test that 'jpg' gets normalized to 'jpeg'"""
        mock_image = Mock(spec=Image.Image)
        mock_image.size = (1920, 1080)
        mock_pyautogui.screenshot.return_value = mock_image
        
        mock_buffer = io.BytesIO()
        mock_image.save = Mock()
        
        with patch('io.BytesIO', return_value=mock_buffer):
            with patch('base64.b64encode', return_value=b'base64data'):
                result = self.service.capture(format="jpg")
        
        assert result['metadata']['format'] == 'jpeg'
    
    @patch('agent_s2.server.services.capture.pyautogui')
    def test_capture_to_file_png(self, mock_pyautogui):
        """Test saving screenshot to file"""
        mock_image = Mock(spec=Image.Image)
        mock_pyautogui.screenshot.return_value = mock_image
        
        filename = "/tmp/test_screenshot.png"
        
        self.service.capture_to_file(filename, format="png")
        
        mock_image.save.assert_called_once_with(filename, format="PNG")
    
    @patch('agent_s2.server.services.capture.pyautogui')
    def test_capture_to_file_jpeg(self, mock_pyautogui):
        """Test saving JPEG screenshot to file"""
        mock_image = Mock(spec=Image.Image)
        mock_pyautogui.screenshot.return_value = mock_image
        
        filename = "/tmp/test_screenshot.jpg"
        
        self.service.capture_to_file(filename, format="jpeg", quality=90)
        
        mock_image.save.assert_called_once_with(filename, format="JPEG", quality=90)
    
    def test_capture_to_file_invalid_format(self):
        """Test error handling for invalid format in file capture"""
        with pytest.raises(ValueError, match="Invalid format: bmp"):
            self.service.capture_to_file("/tmp/test.bmp", format="bmp")


if __name__ == "__main__":
    pytest.main([__file__])