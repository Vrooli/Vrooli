"""Semantic URL Validation Service for Agent S2

Prevents nonsensical navigation decisions by validating that generated URLs
make semantic sense for the given task. Provides intelligent fallbacks when
AI generates inappropriate URLs.
"""

import logging
import re
from typing import Dict, Any, Optional, List, Tuple
from urllib.parse import urlparse
from difflib import SequenceMatcher

logger = logging.getLogger(__name__)


class SemanticValidator:
    """Service for validating semantic appropriateness of URLs for tasks"""
    
    def __init__(self):
        """Initialize semantic validator"""
        
        # Known legitimate domains and their associated keywords
        self.legitimate_domains = {
            # Search engines
            "google.com": ["search", "find", "look", "query"],
            "duckduckgo.com": ["search", "find", "look", "query", "privacy"],
            "bing.com": ["search", "find", "look", "query"],
            "startpage.com": ["search", "find", "look", "query", "privacy"],
            
            # Social media and content
            "reddit.com": ["reddit", "discussion", "community", "forum"],
            "youtube.com": ["video", "watch", "tutorial", "show"],
            "twitter.com": ["tweet", "social", "news", "update"],
            "instagram.com": ["photos", "images", "pictures", "social"],
            "facebook.com": ["social", "friends", "community"],
            
            # Development and tech
            "github.com": ["code", "programming", "development", "repository"],
            "stackoverflow.com": ["programming", "code", "help", "development"],
            "developer.mozilla.org": ["web", "html", "css", "javascript"],
            "python.org": ["python", "programming", "language"],
            "nodejs.org": ["node", "javascript", "development"],
            
            # Reference and information
            "wikipedia.org": ["information", "encyclopedia", "learn", "facts"],
            "mozilla.org": ["browser", "firefox", "web", "privacy"],
            "w3schools.com": ["web", "tutorial", "html", "css", "javascript"],
            
            # News and media
            "bbc.com": ["news", "current", "breaking", "world"],
            "cnn.com": ["news", "current", "breaking", "politics"],
            "reuters.com": ["news", "current", "breaking", "business"],
            
            # Shopping and commerce
            "amazon.com": ["buy", "purchase", "shop", "product"],
            "ebay.com": ["buy", "purchase", "shop", "auction"],
            "etsy.com": ["buy", "craft", "handmade", "art"],
            
            # Animals and pets (relevant for "show me puppies")
            "petfinder.com": ["pets", "dogs", "cats", "puppies", "kittens", "adopt"],
            "akc.org": ["dogs", "breeds", "puppies", "american kennel club"],
            "aspca.org": ["animals", "pets", "dogs", "cats", "welfare"],
            "rover.com": ["dogs", "pets", "walking", "sitting"],
            
            # Image sources
            "unsplash.com": ["photos", "images", "pictures", "free"],
            "pexels.com": ["photos", "images", "pictures", "free"],
            "pixabay.com": ["photos", "images", "pictures", "free"],
            "flickr.com": ["photos", "images", "pictures", "sharing"]
        }
        
        # Suspicious domain patterns that should be flagged
        self.suspicious_patterns = [
            r'^[a-z]+\-[a-z]+\.com$',  # hyphenated domains like "paw-paw.com"
            r'^[a-z]{3,6}[0-9]+\.com$',  # domains with numbers
            r'^[a-z]+[0-9]{2,}\.com$',   # domains ending with multiple numbers
            r'^[a-z]+\.tk$',             # .tk domains are often suspicious
            r'^[a-z]+\.ml$',             # .ml domains are often suspicious
            r'^[a-z]+\.cf$',             # .cf domains are often suspicious
            r'^[a-z]+\.ga$',             # .ga domains are often suspicious
        ]
        
        # Task-to-domain mapping for common queries
        self.task_domain_mappings = {
            "puppies": ["unsplash.com", "google.com", "petfinder.com", "akc.org"],
            "kittens": ["unsplash.com", "google.com", "petfinder.com"],
            "cats": ["unsplash.com", "google.com", "petfinder.com"],
            "dogs": ["unsplash.com", "google.com", "petfinder.com", "akc.org"],
            "animals": ["unsplash.com", "google.com", "petfinder.com"],
            "pets": ["petfinder.com", "google.com", "aspca.org"],
            "news": ["google.com", "bbc.com", "cnn.com", "reuters.com"],
            "tutorial": ["youtube.com", "google.com", "w3schools.com"],
            "programming": ["stackoverflow.com", "github.com", "google.com"],
            "code": ["github.com", "stackoverflow.com", "google.com"],
            "images": ["google.com", "unsplash.com", "pexels.com"],
            "photos": ["google.com", "unsplash.com", "flickr.com"],
            "search": ["google.com", "duckduckgo.com", "bing.com"]
        }
        
        logger.info("SemanticValidator initialized")
    
    def validate_url_for_task(self, url: str, task: str) -> Dict[str, Any]:
        """Validate if a URL makes semantic sense for the given task
        
        Args:
            url: URL to validate
            task: Original task description
            
        Returns:
            Dictionary with validation results and suggestions
        """
        validation_result = {
            "url": url,
            "task": task,
            "is_valid": True,
            "confidence": 1.0,
            "issues": [],
            "suggestions": [],
            "alternative_url": None,
            "reasoning": ""
        }
        
        try:
            parsed_url = urlparse(url)
            domain = parsed_url.netloc.lower()
            
            # Remove www. prefix
            if domain.startswith("www."):
                domain = domain[4:]
            
            task_lower = task.lower()
            task_words = re.findall(r'\b\w+\b', task_lower)
            
            # Check 1: Is domain suspicious?
            if self._is_suspicious_domain(domain):
                validation_result["is_valid"] = False
                validation_result["confidence"] = 0.2
                validation_result["issues"].append(f"Suspicious domain pattern: '{domain}'")
                validation_result["reasoning"] = f"Domain '{domain}' matches suspicious patterns"
            
            # Check 2: Is domain known and legitimate?
            elif domain in self.legitimate_domains:
                domain_keywords = self.legitimate_domains[domain]
                relevance_score = self._calculate_relevance(task_words, domain_keywords)
                
                if relevance_score > 0.3:
                    validation_result["confidence"] = min(1.0, relevance_score + 0.3)
                    validation_result["reasoning"] = f"Known legitimate domain with good relevance"
                else:
                    validation_result["confidence"] = 0.5
                    validation_result["issues"].append(f"Domain '{domain}' is legitimate but may not be most relevant")
                    validation_result["reasoning"] = f"Legitimate domain but low relevance to task"
            
            # Check 3: Unknown domain - more scrutiny needed
            else:
                validation_result["confidence"] = 0.3
                validation_result["issues"].append(f"Unknown domain: '{domain}' - cannot verify legitimacy")
                validation_result["reasoning"] = f"Unknown domain requires verification"
            
            # Check 4: Semantic appropriateness
            semantic_score = self._check_semantic_appropriateness(domain, task_words)
            if semantic_score < 0.4:
                if validation_result["is_valid"]:
                    validation_result["confidence"] *= 0.5  # Reduce confidence
                validation_result["issues"].append(f"Poor semantic match between domain and task")
            
            # Generate suggestions if needed
            if validation_result["confidence"] < 0.7 or not validation_result["is_valid"]:
                suggestions = self._generate_better_suggestions(task, task_words)
                validation_result["suggestions"] = suggestions
                if suggestions:
                    validation_result["alternative_url"] = suggestions[0]["url"]
            
            # Final validation decision
            if validation_result["confidence"] < 0.5 or not validation_result["is_valid"]:
                validation_result["is_valid"] = False
                if validation_result["alternative_url"]:
                    validation_result["reasoning"] += f" -> Suggested alternative: {validation_result['alternative_url']}"
            
        except Exception as e:
            validation_result["is_valid"] = False
            validation_result["confidence"] = 0.0
            validation_result["issues"].append(f"URL parsing error: {e}")
            validation_result["reasoning"] = f"Failed to parse URL: {e}"
            
            # Generate search fallback
            suggestions = self._generate_search_fallback(task)
            validation_result["suggestions"] = suggestions
            if suggestions:
                validation_result["alternative_url"] = suggestions[0]["url"]
        
        logger.info(f"URL validation: '{url}' for task '{task}' -> valid: {validation_result['is_valid']}, confidence: {validation_result['confidence']:.2f}")
        return validation_result
    
    def _is_suspicious_domain(self, domain: str) -> bool:
        """Check if domain matches suspicious patterns
        
        Args:
            domain: Domain to check
            
        Returns:
            True if domain appears suspicious
        """
        for pattern in self.suspicious_patterns:
            if re.match(pattern, domain):
                return True
        return False
    
    def _calculate_relevance(self, task_words: List[str], domain_keywords: List[str]) -> float:
        """Calculate relevance score between task words and domain keywords
        
        Args:
            task_words: Words from the task
            domain_keywords: Keywords associated with the domain
            
        Returns:
            Relevance score between 0.0 and 1.0
        """
        if not task_words or not domain_keywords:
            return 0.0
        
        matches = 0
        for task_word in task_words:
            for keyword in domain_keywords:
                # Direct match
                if task_word == keyword:
                    matches += 2
                # Partial match
                elif task_word in keyword or keyword in task_word:
                    matches += 1
                # Similar words
                elif SequenceMatcher(None, task_word, keyword).ratio() > 0.7:
                    matches += 0.5
        
        # Normalize by total possible matches
        max_possible = len(task_words) * 2
        return min(1.0, matches / max_possible)
    
    def _check_semantic_appropriateness(self, domain: str, task_words: List[str]) -> float:
        """Check semantic appropriateness of domain for task
        
        Args:
            domain: Domain to check
            task_words: Words from task
            
        Returns:
            Appropriateness score between 0.0 and 1.0
        """
        score = 0.0
        
        # Check domain parts against task words
        domain_parts = domain.replace('.', ' ').replace('-', ' ').split()
        
        for task_word in task_words:
            for domain_part in domain_parts:
                if task_word == domain_part:
                    score += 0.3
                elif task_word in domain_part or domain_part in task_word:
                    score += 0.2
                elif SequenceMatcher(None, task_word, domain_part).ratio() > 0.7:
                    score += 0.1
        
        # Check against task-domain mappings
        for task_word in task_words:
            if task_word in self.task_domain_mappings:
                appropriate_domains = self.task_domain_mappings[task_word]
                if domain in appropriate_domains:
                    score += 0.5
                else:
                    # Check if domain is similar to appropriate ones
                    for appropriate_domain in appropriate_domains:
                        if SequenceMatcher(None, domain, appropriate_domain).ratio() > 0.6:
                            score += 0.2
        
        return min(1.0, score)
    
    def _generate_better_suggestions(self, task: str, task_words: List[str]) -> List[Dict[str, Any]]:
        """Generate better URL suggestions for the task
        
        Args:
            task: Original task
            task_words: Words from task
            
        Returns:
            List of suggestion dictionaries
        """
        suggestions = []
        
        # Use search engine service for appropriate search URL
        try:
            from .search_engine_service import search_service
            from .task_classifier import task_classifier
            
            # Get appropriate search URL
            search_url, intent = search_service.get_appropriate_search_url(task)
            suggestions.append({
                "url": search_url,
                "reason": f"Search for '{task}' using {intent['selected_engine']}",
                "type": "search",
                "confidence": 0.9
            })
            
            # Classify task for additional suggestions
            classification = task_classifier.classify_task(task)
            
            if classification["task_type"].value == "image_search":
                image_url = search_service.generate_image_search_url(task)
                if image_url != search_url:
                    suggestions.append({
                        "url": image_url,
                        "reason": f"Image search for '{task}'",
                        "type": "image_search",
                        "confidence": 0.8
                    })
            
        except ImportError:
            # Fallback if services not available
            pass
        
        # Check task-domain mappings for direct suggestions
        for task_word in task_words:
            if task_word in self.task_domain_mappings:
                appropriate_domains = self.task_domain_mappings[task_word]
                for domain in appropriate_domains[:2]:  # Top 2 suggestions
                    url = f"https://{domain}"
                    if domain == "google.com":
                        url = f"https://www.google.com/search?q={'+'.join(task_words)}"
                    elif domain == "duckduckgo.com":
                        url = f"https://duckduckgo.com/?q={'+'.join(task_words)}"
                    
                    suggestions.append({
                        "url": url,
                        "reason": f"Appropriate site for '{task_word}' queries",
                        "type": "navigation",
                        "confidence": 0.7
                    })
        
        # Remove duplicates and sort by confidence
        unique_suggestions = []
        seen_urls = set()
        
        for suggestion in suggestions:
            if suggestion["url"] not in seen_urls:
                unique_suggestions.append(suggestion)
                seen_urls.add(suggestion["url"])
        
        unique_suggestions.sort(key=lambda x: x["confidence"], reverse=True)
        return unique_suggestions[:3]  # Top 3 suggestions
    
    def _generate_search_fallback(self, task: str) -> List[Dict[str, Any]]:
        """Generate search fallback when URL validation fails completely
        
        Args:
            task: Original task
            
        Returns:
            List with search fallback suggestion
        """
        # Simple fallback to DuckDuckGo search
        encoded_task = '+'.join(task.split())
        fallback_url = f"https://duckduckgo.com/?q={encoded_task}"
        
        return [{
            "url": fallback_url,
            "reason": f"Safe search fallback for '{task}'",
            "type": "search_fallback",
            "confidence": 0.8
        }]
    
    def get_validation_stats(self) -> Dict[str, Any]:
        """Get validation service statistics
        
        Returns:
            Dictionary with service statistics
        """
        return {
            "legitimate_domains_count": len(self.legitimate_domains),
            "suspicious_patterns_count": len(self.suspicious_patterns),
            "task_mappings_count": len(self.task_domain_mappings),
            "supported_categories": list(self.task_domain_mappings.keys())
        }


# Global instance
semantic_validator = SemanticValidator()