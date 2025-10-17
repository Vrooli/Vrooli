"""Search Engine Service for Agent S2

Provides intelligent search engine selection with SearXNG as primary and DuckDuckGo as fallback.
Handles search URL generation, health checking, and privacy-focused search routing.
"""

import logging
import requests
import time
from typing import Dict, Any, Optional, List, Tuple
from urllib.parse import quote_plus
from datetime import datetime, timedelta

logger = logging.getLogger(__name__)


class SearchEngineService:
    """Service for managing search engine selection and URL generation"""
    
    def __init__(self):
        """Initialize search engine service"""
        self.searxng_url = "http://localhost:9200"
        self.searxng_healthy = None
        self.last_health_check = None
        self.health_check_interval = 300  # 5 minutes
        self.request_timeout = 5  # seconds
        
        # Search engine configurations
        self.engines = {
            "searxng": {
                "name": "SearXNG",
                "url": self.searxng_url,
                "search_path": "/search",
                "privacy_score": 10,
                "aggregates_sources": True
            },
            "duckduckgo": {
                "name": "DuckDuckGo", 
                "url": "https://duckduckgo.com",
                "search_path": "/",
                "privacy_score": 9,
                "aggregates_sources": False
            },
            "google": {
                "name": "Google",
                "url": "https://www.google.com", 
                "search_path": "/search",
                "privacy_score": 3,
                "aggregates_sources": False
            }
        }
        
        logger.info("SearchEngineService initialized")
    
    def check_searxng_health(self) -> bool:
        """Check if SearXNG is running and healthy
        
        Returns:
            True if SearXNG is healthy, False otherwise
        """
        now = datetime.now()
        
        # Use cached result if recent
        if (self.last_health_check and 
            self.searxng_healthy is not None and
            now - self.last_health_check < timedelta(seconds=self.health_check_interval)):
            return self.searxng_healthy
        
        try:
            # Test SearXNG health endpoint
            health_url = f"{self.searxng_url}/stats"
            response = requests.get(health_url, timeout=self.request_timeout)
            
            if response.status_code == 200:
                # Try a simple search to verify full functionality
                test_url = f"{self.searxng_url}/search"
                test_params = {"q": "test", "format": "json", "categories": "general"}
                test_response = requests.get(test_url, params=test_params, timeout=self.request_timeout)
                
                if test_response.status_code == 200:
                    self.searxng_healthy = True
                    logger.info("SearXNG health check: HEALTHY")
                else:
                    self.searxng_healthy = False
                    logger.warning(f"SearXNG search test failed: {test_response.status_code}")
            else:
                self.searxng_healthy = False
                logger.warning(f"SearXNG health endpoint failed: {response.status_code}")
                
        except Exception as e:
            self.searxng_healthy = False
            logger.warning(f"SearXNG health check failed: {e}")
        
        self.last_health_check = now
        return self.searxng_healthy
    
    def get_primary_search_engine(self) -> str:
        """Get the primary search engine to use
        
        Returns:
            Search engine key ('searxng' or 'duckduckgo')
        """
        if self.searxng_healthy:
            return "searxng"
        else:
            return "duckduckgo"
    
    def generate_search_url(self, query: str, engine: Optional[str] = None, 
                          categories: Optional[List[str]] = None,
                          language: str = "en") -> str:
        """Generate search URL for the given query
        
        Args:
            query: Search query
            engine: Specific engine to use (None for automatic selection)
            categories: Search categories (for SearXNG)
            language: Search language
            
        Returns:
            Generated search URL
        """
        if not engine:
            engine = self.get_primary_search_engine()
        
        engine_config = self.engines.get(engine, self.engines["duckduckgo"])
        base_url = engine_config["url"]
        search_path = engine_config["search_path"]
        
        encoded_query = quote_plus(query)
        
        if engine == "searxng":
            # SearXNG format
            url = f"{base_url}{search_path}?q={encoded_query}&format=html"
            if categories:
                url += f"&categories={','.join(categories)}"
            if language != "en":
                url += f"&language={language}"
        elif engine == "duckduckgo":
            # DuckDuckGo format
            url = f"{base_url}{search_path}?q={encoded_query}"
        elif engine == "google":
            # Google format  
            url = f"{base_url}{search_path}?q={encoded_query}"
        else:
            # Fallback to DuckDuckGo
            url = f"https://duckduckgo.com/?q={encoded_query}"
        
        logger.info(f"Generated search URL using {engine_config['name']}: {url}")
        return url
    
    def generate_image_search_url(self, query: str, engine: Optional[str] = None) -> str:
        """Generate image search URL for the given query
        
        Args:
            query: Search query
            engine: Specific engine to use (None for automatic selection)
            
        Returns:
            Generated image search URL
        """
        if not engine:
            engine = self.get_primary_search_engine()
            
        engine_config = self.engines.get(engine, self.engines["duckduckgo"])
        encoded_query = quote_plus(query)
        
        if engine == "searxng":
            # SearXNG image search
            url = f"{engine_config['url']}/search?q={encoded_query}&categories=images&format=html"
        elif engine == "duckduckgo":
            # DuckDuckGo image search
            url = f"{engine_config['url']}/?q={encoded_query}&iax=images&ia=images"
        elif engine == "google":
            # Google image search
            url = f"https://images.google.com/search?q={encoded_query}"
        else:
            # Fallback to DuckDuckGo images
            url = f"https://duckduckgo.com/?q={encoded_query}&iax=images&ia=images"
        
        logger.info(f"Generated image search URL using {engine_config['name']}: {url}")
        return url
    
    def classify_search_intent(self, query: str) -> Dict[str, Any]:
        """Classify the search intent based on the query
        
        Args:
            query: Search query to classify
            
        Returns:
            Dictionary with search intent classification
        """
        query_lower = query.lower()
        
        # Image search indicators
        image_keywords = ["show me", "pictures of", "images of", "photos of", "puppies", "cats", "dogs", 
                         "artwork", "screenshots", "wallpapers", "diagrams"]
        is_image_search = any(keyword in query_lower for keyword in image_keywords)
        
        # News search indicators  
        news_keywords = ["news", "latest", "breaking", "headlines", "today", "current events"]
        is_news_search = any(keyword in query_lower for keyword in news_keywords)
        
        # Academic/research indicators
        academic_keywords = ["research", "study", "academic", "paper", "journal", "thesis", "analysis"]
        is_academic_search = any(keyword in query_lower for keyword in academic_keywords)
        
        # Shopping indicators
        shopping_keywords = ["buy", "purchase", "price", "store", "shop", "deal", "sale"]
        is_shopping_search = any(keyword in query_lower for keyword in shopping_keywords)
        
        # Local search indicators
        local_keywords = ["near me", "nearby", "local", "restaurant", "hotel", "directions"]
        is_local_search = any(keyword in query_lower for keyword in local_keywords)
        
        # Determine primary intent
        if is_image_search:
            search_type = "images"
            categories = ["images"] 
        elif is_news_search:
            search_type = "news"
            categories = ["news"]
        elif is_academic_search:
            search_type = "academic"
            categories = ["science", "general"]
        elif is_shopping_search:
            search_type = "shopping"
            categories = ["general"]
        elif is_local_search:
            search_type = "local"
            categories = ["general", "map"]
        else:
            search_type = "general"
            categories = ["general"]
        
        return {
            "search_type": search_type,
            "categories": categories,
            "is_image_search": is_image_search,
            "is_news_search": is_news_search,
            "is_academic_search": is_academic_search,
            "is_shopping_search": is_shopping_search,
            "is_local_search": is_local_search,
            "query": query,
            "processed_query": query  # Could add query enhancement here
        }
    
    def get_appropriate_search_url(self, query: str) -> Tuple[str, Dict[str, Any]]:
        """Get the most appropriate search URL for a query
        
        Args:
            query: Search query
            
        Returns:
            Tuple of (search_url, intent_analysis)
        """
        # First check SearXNG health
        is_healthy = self.check_searxng_health()
        
        # Classify the search intent
        intent = self.classify_search_intent(query)
        
        # Generate appropriate URL based on intent
        if intent["is_image_search"]:
            url = self.generate_image_search_url(query)
        else:
            url = self.generate_search_url(
                query,
                categories=intent["categories"]
            )
        
        # Add metadata about the decision
        intent["selected_engine"] = self.get_primary_search_engine()
        intent["searxng_healthy"] = is_healthy
        intent["generated_url"] = url
        
        return url, intent
    
    def get_service_status(self) -> Dict[str, Any]:
        """Get service status information
        
        Returns:
            Dictionary with service status
        """
        return {
            "searxng_url": self.searxng_url,
            "searxng_healthy": self.searxng_healthy,
            "last_health_check": self.last_health_check.isoformat() if self.last_health_check else None,
            "primary_engine": self.get_primary_search_engine(),
            "available_engines": list(self.engines.keys()),
            "health_check_interval": self.health_check_interval
        }


# Global instance
search_service = SearchEngineService()