"""AI Service Manager for Agent S2

Handles service discovery, initialization, and API communication with AI providers.
"""

import os
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime

try:
    import requests
except ImportError:
    requests = None

from ...config import Config

logger = logging.getLogger(__name__)


class AIServiceManager:
    """Manages AI service discovery, initialization, and communication"""
    
    def __init__(self):
        """Initialize AI service manager"""
        self.initialized = False
        self.enabled = Config.AI_ENABLED
        self.provider = Config.AI_PROVIDER
        self.model = Config.AI_MODEL
        self.api_url = Config.AI_API_URL
        
        # Parse Ollama base URL from API URL
        if self.api_url:
            if "/api/" in self.api_url:
                self.ollama_base_url = self.api_url.split("/api/")[0]
            else:
                self.ollama_base_url = self.api_url
        else:
            self.ollama_base_url = "http://localhost:11434"
    
    async def discover_ollama_service(self) -> Optional[Dict[str, Any]]:
        """Auto-detect available Ollama services
        
        Returns:
            Dict with discovered service info or None if not found
        """
        logger.info("🔍 Auto-detecting Ollama services...")
        
        # Common Ollama locations to check
        ollama_locations = [
            ("http://localhost:11434", "Local Ollama"),
            ("http://ollama:11434", "Docker service 'ollama'"),
            ("http://host.docker.internal:11434", "Host machine (Docker Desktop)"),
            ("http://172.17.0.1:11434", "Docker host gateway")
        ]
        
        # Add current configured URL if different
        if self.ollama_base_url not in [loc[0] for loc in ollama_locations]:
            ollama_locations.insert(0, (self.ollama_base_url, "Configured URL"))
        
        for url, description in ollama_locations:
            try:
                logger.debug(f"Checking {description} at {url}")
                response = requests.get(f"{url}/api/tags", timeout=2)
                
                if response.status_code == 200:
                    models_data = response.json()
                    models = models_data.get("models", [])
                    model_names = [m["name"] for m in models]
                    
                    logger.info(f"✅ Found Ollama at {url} ({description}) with {len(models)} models")
                    
                    # Look for vision models first
                    vision_models = [m for m in model_names if "vision" in m.lower() or "llava" in m.lower()]
                    
                    return {
                        "url": url,
                        "description": description,
                        "models": model_names,
                        "vision_models": vision_models,
                        "model_count": len(models)
                    }
                    
            except requests.RequestException as e:
                logger.debug(f"Failed to connect to {url}: {type(e).__name__}")
                continue
            except Exception as e:
                logger.debug(f"Unexpected error checking {url}: {e}")
                continue
        
        logger.warning("❌ No Ollama service found at any common location")
        return None

    async def initialize(self) -> bool:
        """Initialize AI service with error handling and auto-detection
        
        Returns:
            True if initialization successful, False otherwise
        """
        if not self.enabled:
            logger.info("AI disabled in configuration")
            return False
            
        logger.info(f"Initializing AI service manager with provider: {self.provider}")
        logger.info(f"Configured model: {self.model}")
        logger.info(f"Initial Ollama base URL: {self.ollama_base_url}")
        
        try:
            # Validate provider support
            if self.provider != "ollama":
                logger.error(f"Provider '{self.provider}' is not yet implemented. Only 'ollama' is currently supported.")
                logger.error("To use Ollama, set AGENTS2_LLM_PROVIDER=ollama or remove the environment variable (defaults to ollama)")
                return False
                
            # Check if requests library is available
            if requests is None:
                logger.error("requests library not available. Install with: pip install requests")
                return False
            
            # Try auto-detection first
            discovered = await self.discover_ollama_service()
            
            if discovered:
                # Update configuration with discovered service
                self.ollama_base_url = discovered["url"]
                logger.info(f"🎯 Using discovered Ollama at: {discovered['url']} ({discovered['description']})")
                
                # Auto-select best model if current model not available
                if self.model not in discovered["models"]:
                    if discovered["vision_models"]:
                        # Prefer vision models for Agent-S2
                        self.model = discovered["vision_models"][0]
                        logger.info(f"🎨 Auto-selected vision model: {self.model}")
                    elif discovered["models"]:
                        # Fall back to first available model
                        self.model = discovered["models"][0]
                        logger.info(f"🤖 Auto-selected model: {self.model}")
                    else:
                        logger.error("No models available in Ollama. Please pull a model.")
                        logger.error("Recommended: ollama pull llama3.2-vision:11b")
                        return False
                else:
                    logger.info(f"✅ Configured model '{self.model}' is available")
            else:
                # No auto-detection successful, try configured URL anyway
                logger.warning("Auto-detection failed, trying configured URL...")
                
            # Check Ollama health using direct HTTP call
            logger.info("Checking Ollama connectivity...")
            try:
                health_response = requests.get(f"{self.ollama_base_url}/api/tags", timeout=5)
                if health_response.status_code != 200:
                    logger.error(f"Ollama health check failed: {health_response.status_code}")
                    logger.error(f"Make sure Ollama is running and accessible at {self.ollama_base_url}")
                    return False
                logger.info("✅ Ollama is reachable")
            except requests.RequestException as e:
                logger.error(f"Ollama is not reachable at {self.ollama_base_url}: {e}")
                logger.error("Ensure Ollama is running and listening on all interfaces (OLLAMA_HOST=0.0.0.0)")
                return False
                
            # Test API connectivity and validate model
            try:
                response = requests.get(f"{self.ollama_base_url}/api/tags", timeout=5)
                if response.status_code != 200:
                    logger.error(f"Ollama API not responding: {response.status_code}")
                    return False
                    
                # Check if our model is available
                models = response.json().get("models", [])
                available_models = [model["name"] for model in models]
                
                logger.info(f"Found {len(available_models)} models in Ollama")
                
                if self.model not in available_models:
                    logger.warning(f"Configured model '{self.model}' not found")
                    logger.info(f"Available models: {', '.join(available_models) if available_models else 'None'}")
                    
                    # Look for vision models as preferred fallback
                    vision_models = [m for m in available_models if 'vision' in m.lower() or 'llava' in m.lower()]
                    if vision_models:
                        self.model = vision_models[0]
                        logger.info(f"Using vision model as fallback: {self.model}")
                    elif available_models:
                        self.model = available_models[0]
                        logger.info(f"Using first available model as fallback: {self.model}")
                    else:
                        logger.error("No models available in Ollama. Please pull a model using: ollama pull llama3.2-vision:11b")
                        return False
                else:
                    logger.info(f"✅ Model '{self.model}' is available")
                        
            except requests.RequestException as e:
                logger.error(f"Failed to connect to Ollama API: {e}")
                return False
                
            self.initialized = True
            logger.info(f"✅ AI service manager initialized successfully")
            logger.info(f"   Provider: {self.provider}")
            logger.info(f"   Model: {self.model}")
            logger.info(f"   Base URL: {self.ollama_base_url}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to initialize AI service manager: {e}")
            self.initialized = False
            return False
            
    async def shutdown(self):
        """Shutdown AI service manager"""
        self.initialized = False
        logger.info("AI service manager shut down")
        
    def call_ollama(self, prompt: str, model: Optional[str] = None, system: Optional[str] = None, images: Optional[List[str]] = None) -> Dict[str, Any]:
        """Make a call to Ollama API
        
        Args:
            prompt: The prompt to send
            model: Model to use (defaults to self.model)
            system: Optional system prompt
            images: Optional list of base64-encoded images
            
        Returns:
            Response from Ollama
            
        Raises:
            RuntimeError: If service not initialized
            requests.RequestException: If API call fails
        """
        if not self.initialized:
            raise RuntimeError("AI service manager not initialized")
            
        model = model or self.model
        
        payload = {
            "model": model,
            "prompt": prompt,
            "stream": False
        }
        
        if system:
            payload["system"] = system
            
        if images:
            # Ensure images is a proper list
            if not isinstance(images, list):
                logger.warning(f"images parameter is not a list: {type(images)}")
                images = [images] if images else []
            
            # Validate each image is a string
            validated_images = []
            for img in images:
                if isinstance(img, str):
                    validated_images.append(img)
                else:
                    logger.warning(f"Skipping non-string image: {type(img)}")
            
            payload["images"] = validated_images
            
        try:
            # Log payload structure for debugging
            logger.debug(f"Ollama payload keys: {list(payload.keys())}")
            if images:
                logger.debug(f"Number of images: {len(payload.get('images', []))}")
            
            response = requests.post(
                f"{self.ollama_base_url}/api/generate",
                json=payload,
                timeout=Config.AI_TIMEOUT
            )
            response.raise_for_status()
            return response.json()
            
        except requests.RequestException as e:
            logger.error(f"Ollama API call failed: {e}")
            if hasattr(e, 'response') and e.response:
                logger.error(f"Response status: {e.response.status_code}")
                logger.error(f"Response text: {e.response.text}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error in Ollama call: {type(e).__name__}: {e}")
            raise
            
    def get_service_info(self) -> Dict[str, Any]:
        """Get current service information
        
        Returns:
            Dict with service status and configuration
        """
        return {
            "initialized": self.initialized,
            "enabled": self.enabled,
            "provider": self.provider,
            "model": self.model,
            "ollama_base_url": self.ollama_base_url,
            "api_url": self.api_url
        }
        
    def is_ready(self) -> bool:
        """Check if service is ready for use
        
        Returns:
            True if service is initialized and enabled
        """
        return self.initialized and self.enabled