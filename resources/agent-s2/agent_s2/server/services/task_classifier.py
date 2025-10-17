"""Task Classification Service for Agent S2

Classifies user tasks to determine appropriate actions and prevent nonsensical navigation decisions.
Provides semantic understanding of user intent to route tasks correctly.
"""

import logging
import re
from typing import Dict, Any, Optional, List, Tuple
from enum import Enum
from urllib.parse import urlparse

logger = logging.getLogger(__name__)


class TaskType(Enum):
    """Task type classifications"""
    SEARCH = "search"           # General information search
    IMAGE_SEARCH = "image_search"  # Image/visual content search
    NAVIGATION = "navigation"   # Navigate to specific website
    APPLICATION = "application" # Launch/interact with applications
    SYSTEM = "system"          # System operations (screenshot, etc.)
    AUTOMATION = "automation"  # UI automation tasks
    UNKNOWN = "unknown"        # Unclear intent


class TaskClassifier:
    """Service for classifying user tasks and determining appropriate actions"""
    
    def __init__(self):
        """Initialize task classifier"""
        
        # Search intent patterns
        self.search_patterns = [
            r'\b(?:search|find|look\s+(?:up|for)|show\s+me|tell\s+me\s+about)\b',
            r'\b(?:what\s+is|how\s+to|where\s+can|when\s+did)\b',
            r'\b(?:information\s+about|details\s+on|learn\s+about)\b'
        ]
        
        # Image search patterns
        self.image_patterns = [
            r'\b(?:show\s+me|pictures?\s+of|images?\s+of|photos?\s+of)\b',
            r'\b(?:puppies|kittens|cats|dogs|animals|artwork|wallpapers)\b',
            r'\b(?:screenshot|visual|diagram|chart|graph)\b'
        ]
        
        # Navigation patterns
        self.navigation_patterns = [
            r'\b(?:go\s+to|navigate\s+to|visit|open)\s+(?:website\s+)?([^\s]+\.(?:com|org|net|edu|gov|io))\b',
            r'\b(?:go\s+to|visit)\s+(reddit|youtube|github|stackoverflow|wikipedia)\b',
            r'\bhttps?://[^\s]+\b'
        ]
        
        # Application patterns
        self.application_patterns = [
            r'\b(?:launch|open|start|run)\s+(?:application\s+|app\s+|program\s+)?([^\s]+)\b',
            r'\b(?:firefox|chrome|calculator|terminal|editor|notepad)\b'
        ]
        
        # System operation patterns
        self.system_patterns = [
            r'\b(?:take\s+a?\s*screenshot|capture\s+screen|screenshot)\b',
            r'\b(?:system\s+status|check\s+health|show\s+processes)\b'
        ]
        
        # Automation patterns
        self.automation_patterns = [
            r'\b(?:click|type|press|scroll|drag)\b',
            r'\b(?:fill\s+(?:out|in)|submit|select|choose)\b'
        ]
        
        # Known safe domains for navigation
        self.safe_domains = {
            "reddit.com", "reddit", "youtube.com", "youtube", "github.com", "github",
            "stackoverflow.com", "stackoverflow", "wikipedia.org", "wikipedia",
            "google.com", "duckduckgo.com", "bing.com", "startpage.com",
            "mozilla.org", "python.org", "nodejs.org", "developer.mozilla.org",
            "docs.python.org", "npmjs.com", "pypi.org"
        }
        
        logger.info("TaskClassifier initialized")
    
    def classify_task(self, task: str) -> Dict[str, Any]:
        """Classify a user task to determine appropriate actions
        
        Args:
            task: User task description
            
        Returns:
            Dictionary with classification results
        """
        task_lower = task.lower()
        
        # Initialize classification result
        classification = {
            "original_task": task,
            "task_type": TaskType.UNKNOWN,
            "confidence": 0.0,
            "recommended_action": "search",  # Safe default
            "reasoning": "",
            "requires_browser": False,
            "safe_to_execute": True,
            "metadata": {}
        }
        
        # Check each task type
        scores = {}
        
        # 1. Check for image search
        image_score = self._calculate_pattern_score(task_lower, self.image_patterns)
        scores[TaskType.IMAGE_SEARCH] = image_score
        
        # 2. Check for navigation
        nav_score, nav_metadata = self._check_navigation_intent(task, task_lower)
        scores[TaskType.NAVIGATION] = nav_score
        if nav_metadata:
            classification["metadata"].update(nav_metadata)
        
        # 3. Check for general search
        search_score = self._calculate_pattern_score(task_lower, self.search_patterns)
        scores[TaskType.SEARCH] = search_score
        
        # 4. Check for application tasks
        app_score, app_metadata = self._check_application_intent(task_lower)
        scores[TaskType.APPLICATION] = app_score
        if app_metadata:
            classification["metadata"].update(app_metadata)
        
        # 5. Check for system operations
        system_score = self._calculate_pattern_score(task_lower, self.system_patterns)
        scores[TaskType.SYSTEM] = system_score
        
        # 6. Check for automation tasks
        automation_score = self._calculate_pattern_score(task_lower, self.automation_patterns)
        scores[TaskType.AUTOMATION] = automation_score
        
        # Determine best classification
        best_type = max(scores.keys(), key=lambda k: scores[k])
        best_score = scores[best_type]
        
        if best_score > 0.0:
            classification["task_type"] = best_type
            classification["confidence"] = best_score
        
        # Set recommended action and reasoning
        classification = self._set_recommended_action(classification, task_lower)
        
        logger.info(f"Task classified: '{task}' -> {best_type.value} (confidence: {best_score:.2f})")
        return classification
    
    def _calculate_pattern_score(self, text: str, patterns: List[str]) -> float:
        """Calculate pattern matching score
        
        Args:
            text: Text to analyze
            patterns: List of regex patterns
            
        Returns:
            Score between 0.0 and 1.0
        """
        matches = 0
        total_patterns = len(patterns)
        
        for pattern in patterns:
            if re.search(pattern, text, re.IGNORECASE):
                matches += 1
        
        return matches / total_patterns if total_patterns > 0 else 0.0
    
    def _check_navigation_intent(self, original_task: str, task_lower: str) -> Tuple[float, Dict[str, Any]]:
        """Check for navigation intent and validate safety
        
        Args:
            original_task: Original task text
            task_lower: Lowercase task text
            
        Returns:
            Tuple of (score, metadata)
        """
        metadata = {}
        score = 0.0
        
        # Check navigation patterns
        for pattern in self.navigation_patterns:
            match = re.search(pattern, task_lower, re.IGNORECASE)
            if match:
                score = 0.8  # High confidence for navigation
                
                # Extract target
                if match.groups():
                    target = match.group(1)
                    metadata["target"] = target
                    metadata["is_safe_domain"] = self._is_safe_domain(target)
                else:
                    # Look for domain patterns in the original task
                    domain_match = re.search(r'\b([a-zA-Z0-9.-]+\.(?:com|org|net|edu|gov|io))\b', original_task)
                    if domain_match:
                        target = domain_match.group(1)
                        metadata["target"] = target
                        metadata["is_safe_domain"] = self._is_safe_domain(target)
                
                break
        
        # Additional navigation indicators
        nav_words = ["go to", "navigate", "visit", "open website"]
        for word in nav_words:
            if word in task_lower:
                score = max(score, 0.6)
                break
        
        return score, metadata
    
    def _check_application_intent(self, task_lower: str) -> Tuple[float, Dict[str, Any]]:
        """Check for application launch intent
        
        Args:
            task_lower: Lowercase task text
            
        Returns:
            Tuple of (score, metadata)
        """
        metadata = {}
        score = 0.0
        
        # Check application patterns
        for pattern in self.application_patterns:
            match = re.search(pattern, task_lower, re.IGNORECASE)
            if match:
                score = 0.7
                if match.groups():
                    app_name = match.group(1)
                    metadata["app_name"] = app_name
                    metadata["is_browser"] = app_name.lower() in ["firefox", "chrome", "browser"]
                break
        
        return score, metadata
    
    def _is_safe_domain(self, domain: str) -> bool:
        """Check if a domain is in the safe list
        
        Args:
            domain: Domain to check
            
        Returns:
            True if domain is safe, False otherwise
        """
        if not domain:
            return False
            
        domain_lower = domain.lower().strip()
        
        # Remove www. prefix if present
        if domain_lower.startswith("www."):
            domain_lower = domain_lower[4:]
        
        # Check against safe domains
        return domain_lower in self.safe_domains
    
    def _set_recommended_action(self, classification: Dict[str, Any], task_lower: str) -> Dict[str, Any]:
        """Set recommended action based on classification
        
        Args:
            classification: Current classification
            task_lower: Lowercase task text
            
        Returns:
            Updated classification with recommended action
        """
        task_type = classification["task_type"]
        
        if task_type == TaskType.IMAGE_SEARCH:
            classification["recommended_action"] = "image_search"
            classification["requires_browser"] = True
            classification["reasoning"] = "Detected image search intent - will use image search engine"
            
        elif task_type == TaskType.SEARCH:
            classification["recommended_action"] = "search"
            classification["requires_browser"] = True
            classification["reasoning"] = "Detected general search intent - will use default search engine"
            
        elif task_type == TaskType.NAVIGATION:
            if classification["metadata"].get("is_safe_domain", False):
                classification["recommended_action"] = "navigate"
                classification["requires_browser"] = True
                classification["reasoning"] = f"Navigate to safe domain: {classification['metadata'].get('target', 'unknown')}"
            else:
                # Fallback to search for unsafe/unknown domains
                classification["recommended_action"] = "search"
                classification["requires_browser"] = True
                classification["safe_to_execute"] = True  # Search is always safe
                classification["reasoning"] = "Unknown domain - converting to search for safety"
                
        elif task_type == TaskType.APPLICATION:
            classification["recommended_action"] = "launch_app"
            classification["requires_browser"] = classification["metadata"].get("is_browser", False)
            app_name = classification["metadata"].get("app_name", "unknown")
            classification["reasoning"] = f"Launch application: {app_name}"
            
        elif task_type == TaskType.SYSTEM:
            classification["recommended_action"] = "system_operation"
            classification["requires_browser"] = False
            classification["reasoning"] = "System operation detected"
            
        elif task_type == TaskType.AUTOMATION:
            classification["recommended_action"] = "automation"
            classification["requires_browser"] = False
            classification["reasoning"] = "UI automation task detected"
            
        else:
            # Unknown task - default to search (safest option)
            classification["recommended_action"] = "search"
            classification["requires_browser"] = True
            classification["reasoning"] = "Unknown task type - defaulting to search for safety"
            
        return classification
    
    def validate_url_safety(self, url: str, task: str) -> Dict[str, Any]:
        """Validate if a URL makes sense for the given task
        
        Args:
            url: URL to validate
            task: Original task description
            
        Returns:
            Dictionary with validation results
        """
        validation = {
            "url": url,
            "task": task,
            "is_safe": True,
            "is_relevant": True,
            "issues": [],
            "alternative_suggestion": None
        }
        
        try:
            parsed = urlparse(url)
            domain = parsed.netloc.lower()
            
            # Remove www. prefix
            if domain.startswith("www."):
                domain = domain[4:]
            
            # Check if domain is safe
            if not self._is_safe_domain(domain):
                validation["is_safe"] = False
                validation["issues"].append(f"Unknown/unsafe domain: {domain}")
                
                # Suggest search alternative
                from .search_engine_service import search_service
                search_url, _ = search_service.get_appropriate_search_url(task)
                validation["alternative_suggestion"] = search_url
            
            # Check semantic relevance
            task_lower = task.lower()
            domain_parts = domain.split(".")
            
            # Simple relevance check
            is_relevant = False
            for part in domain_parts:
                if part in task_lower or any(word in part for word in task_lower.split()):
                    is_relevant = True
                    break
            
            # Special cases
            if "reddit" in task_lower and "reddit" not in domain:
                is_relevant = False
            elif "youtube" in task_lower and "youtube" not in domain:
                is_relevant = False
            elif "google" in task_lower and "google" not in domain:
                is_relevant = False
            
            # If not relevant, suggest search
            if not is_relevant and not self._is_safe_domain(domain):
                validation["is_relevant"] = False
                validation["issues"].append(f"Domain '{domain}' does not seem relevant to task '{task}'")
                
                # Suggest search alternative
                from .search_engine_service import search_service
                search_url, _ = search_service.get_appropriate_search_url(task)
                validation["alternative_suggestion"] = search_url
            
        except Exception as e:
            validation["is_safe"] = False
            validation["issues"].append(f"URL parsing error: {e}")
            
            # Suggest search as fallback
            from .search_engine_service import search_service
            search_url, _ = search_service.get_appropriate_search_url(task)
            validation["alternative_suggestion"] = search_url
        
        logger.info(f"URL validation: {url} for task '{task}' -> safe: {validation['is_safe']}, relevant: {validation['is_relevant']}")
        return validation
    
    def suggest_better_url(self, task: str, problematic_url: str) -> str:
        """Suggest a better URL for a task
        
        Args:
            task: Original task
            problematic_url: URL that was deemed problematic
            
        Returns:
            Better URL suggestion
        """
        classification = self.classify_task(task)
        
        if classification["task_type"] in [TaskType.SEARCH, TaskType.IMAGE_SEARCH]:
            from .search_engine_service import search_service
            better_url, _ = search_service.get_appropriate_search_url(task)
            return better_url
        elif classification["task_type"] == TaskType.NAVIGATION:
            target = classification["metadata"].get("target")
            if target and self._is_safe_domain(target):
                return f"https://{target}"
        
        # Default fallback to search
        from .search_engine_service import search_service
        search_url, _ = search_service.get_appropriate_search_url(task)
        return search_url


# Global instance
task_classifier = TaskClassifier()