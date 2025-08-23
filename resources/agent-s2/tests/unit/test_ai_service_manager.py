#!/usr/bin/env python3
"""
Unit tests for Agent S2 AI Service Manager

Tests AI service management functionality including:
- Service initialization
- URL parsing and configuration
- Service discovery
- Error handling
"""

import os
import sys
import pytest
from unittest.mock import Mock, patch, AsyncMock
from pathlib import Path

# Add the agent_s2 package to Python path
sys.path.insert(0, str(Path(__file__).parent.parent.parent / "agent_s2"))

from agent_s2.server.services.ai_service_manager import AIServiceManager


class TestAIServiceManager:
    """Test AI service manager functionality"""
    
    def setup_method(self):
        """Set up test fixtures"""
        with patch('agent_s2.server.services.ai_service_manager.Config') as mock_config:
            mock_config.AI_ENABLED = True
            mock_config.AI_PROVIDER = "ollama"
            mock_config.AI_MODEL = "llama3.2-vision:11b"
            mock_config.AI_API_URL = "http://localhost:11434"
            self.manager = AIServiceManager()
    
    def test_init_with_base_url(self):
        """Test initialization with base URL"""
        with patch('agent_s2.server.services.ai_service_manager.Config') as mock_config:
            mock_config.AI_ENABLED = True
            mock_config.AI_PROVIDER = "ollama"
            mock_config.AI_MODEL = "llama3.2-vision:11b"
            mock_config.AI_API_URL = "http://localhost:11434"
            
            manager = AIServiceManager()
            
            assert manager.initialized is False
            assert manager.enabled is True
            assert manager.provider == "ollama"
            assert manager.model == "llama3.2-vision:11b"
            assert manager.api_url == "http://localhost:11434"
            assert manager.ollama_base_url == "http://localhost:11434"
    
    def test_init_with_full_api_url(self):
        """Test initialization with full API URL"""
        with patch('agent_s2.server.services.ai_service_manager.Config') as mock_config:
            mock_config.AI_ENABLED = True
            mock_config.AI_PROVIDER = "ollama"
            mock_config.AI_MODEL = "llama3.2-vision:11b"
            mock_config.AI_API_URL = "http://localhost:11434/api/generate"
            
            manager = AIServiceManager()
            
            assert manager.api_url == "http://localhost:11434/api/generate"
            assert manager.ollama_base_url == "http://localhost:11434"
    
    def test_init_with_empty_api_url(self):
        """Test initialization with empty API URL"""
        with patch('agent_s2.server.services.ai_service_manager.Config') as mock_config:
            mock_config.AI_ENABLED = True
            mock_config.AI_PROVIDER = "ollama"
            mock_config.AI_MODEL = "llama3.2-vision:11b"
            mock_config.AI_API_URL = ""
            
            manager = AIServiceManager()
            
            assert manager.api_url == ""
            assert manager.ollama_base_url == "http://localhost:11434"  # Default fallback
    
    def test_init_with_none_api_url(self):
        """Test initialization with None API URL"""
        with patch('agent_s2.server.services.ai_service_manager.Config') as mock_config:
            mock_config.AI_ENABLED = True
            mock_config.AI_PROVIDER = "ollama"
            mock_config.AI_MODEL = "llama3.2-vision:11b"
            mock_config.AI_API_URL = None
            
            manager = AIServiceManager()
            
            assert manager.api_url is None
            assert manager.ollama_base_url == "http://localhost:11434"  # Default fallback
    
    def test_init_disabled(self):
        """Test initialization when AI is disabled"""
        with patch('agent_s2.server.services.ai_service_manager.Config') as mock_config:
            mock_config.AI_ENABLED = False
            mock_config.AI_PROVIDER = "ollama"
            mock_config.AI_MODEL = "llama3.2-vision:11b"
            mock_config.AI_API_URL = "http://localhost:11434"
            
            manager = AIServiceManager()
            
            assert manager.enabled is False
    
    @pytest.mark.asyncio
    @patch('agent_s2.server.services.ai_service_manager.requests')
    async def test_discover_ollama_service_success(self, mock_requests):
        """Test successful Ollama service discovery"""
        # Mock successful response
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {
            "models": [
                {"name": "llama3.2-vision:11b"},
                {"name": "mistral:7b"}
            ]
        }
        mock_requests.get.return_value = mock_response
        
        result = await self.manager.discover_ollama_service()
        
        assert result is not None
        assert result["service_type"] == "ollama"
        assert result["base_url"] == "http://localhost:11434"
        assert result["status"] == "available"
        assert len(result["models"]) == 2
        assert "llama3.2-vision:11b" in result["models"]
    
    @pytest.mark.asyncio
    @patch('agent_s2.server.services.ai_service_manager.requests')
    async def test_discover_ollama_service_failure(self, mock_requests):
        """Test failed Ollama service discovery"""
        # Mock failed response for all locations
        mock_requests.get.side_effect = Exception("Connection failed")
        
        result = await self.manager.discover_ollama_service()
        
        assert result is None
    
    @pytest.mark.asyncio
    @patch('agent_s2.server.services.ai_service_manager.requests')
    async def test_discover_ollama_service_no_models(self, mock_requests):
        """Test Ollama service discovery with no models"""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"models": []}
        mock_requests.get.return_value = mock_response
        
        result = await self.manager.discover_ollama_service()
        
        assert result is not None
        assert result["status"] == "available"
        assert len(result["models"]) == 0
    
    @pytest.mark.asyncio
    @patch('agent_s2.server.services.ai_service_manager.requests')
    async def test_discover_ollama_service_partial_failure(self, mock_requests):
        """Test Ollama service discovery with some locations failing"""
        def side_effect(url, timeout=None):
            if "localhost:11434" in url:
                raise Exception("Local connection failed")
            else:
                mock_response = Mock()
                mock_response.status_code = 200
                mock_response.json.return_value = {"models": [{"name": "test-model"}]}
                return mock_response
        
        mock_requests.get.side_effect = side_effect
        
        result = await self.manager.discover_ollama_service()
        
        # Should find the working service despite local failure
        assert result is not None
        assert result["status"] == "available"
    
    def test_url_parsing_edge_cases(self):
        """Test URL parsing with various edge cases"""
        test_cases = [
            ("http://localhost:11434", "http://localhost:11434"),
            ("http://localhost:11434/", "http://localhost:11434"),
            ("http://localhost:11434/api/generate", "http://localhost:11434"),
            ("http://localhost:11434/api/chat", "http://localhost:11434"),
            ("https://api.openai.com/v1/api/completions", "https://api.openai.com/v1"),
        ]
        
        for api_url, expected_base in test_cases:
            with patch('agent_s2.server.services.ai_service_manager.Config') as mock_config:
                mock_config.AI_ENABLED = True
                mock_config.AI_PROVIDER = "ollama"
                mock_config.AI_MODEL = "test"
                mock_config.AI_API_URL = api_url
                
                manager = AIServiceManager()
                assert manager.ollama_base_url == expected_base, f"Failed for {api_url}"


if __name__ == "__main__":
    pytest.main([__file__])