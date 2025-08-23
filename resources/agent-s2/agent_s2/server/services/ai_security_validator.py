"""AI Security Validator for Agent S2

Handles security validation for AI-driven actions, particularly URL validation and content filtering.
"""

import os
import logging
import re
from typing import Dict, Any, Optional, List, Tuple
from enum import Enum

from .url_security import TypeAction, TypeActionType, URLValidator, get_security_config

logger = logging.getLogger(__name__)


class ActionSecurityResult:
    """Result of action security validation"""
    
    def __init__(self, valid: bool, reason: str = "", suggested_action: Optional[Dict[str, Any]] = None, blocked_reason: Optional[str] = None):
        self.valid = valid
        self.reason = reason
        self.suggested_action = suggested_action
        self.blocked_reason = blocked_reason


class AISecurityValidator:
    """Handles security validation for AI-driven actions"""
    
    def __init__(self):
        """Initialize security validator"""
        self.default_security_profile = "moderate"
    
    def validate_text_action(self, text: str, security_config: Optional[Dict[str, Any]] = None) -> ActionSecurityResult:
        """Validate a text input action for security
        
        Args:
            text: Text to be typed
            security_config: Optional security configuration
            
        Returns:
            ActionSecurityResult with validation outcome
        """
        try:
            # Determine action type based on content
            action_type = self._detect_action_type(text)
            
            # Validate URL-type actions
            if action_type == TypeActionType.URL:
                logger.info(f"Detected URL type action for text: {text}")
                logger.info(f"Security config passed: {security_config}")
                
                # Get security config
                sec_config = self._get_security_config(security_config)
                
                type_action = TypeAction(text, action_type, sec_config)
                logger.info(f"Using security config: {sec_config}")
                validation_result = type_action.validate()
                logger.info(f"Validation result: valid={validation_result.valid}, reason={validation_result.reason}")
                
                if not validation_result.valid:
                    logger.warning(f"URL validation failed: {validation_result.reason}")
                    
                    # Create suggested action if we have a suggested URL
                    suggested_action = None
                    if validation_result.suggested_url:
                        logger.info(f"Using suggested URL: {validation_result.suggested_url}")
                        suggested_action = {
                            "type": "type",
                            "text": validation_result.suggested_url,
                            "description": f"Use suggested URL: {validation_result.suggested_url}"
                        }
                    
                    return ActionSecurityResult(
                        valid=False,
                        reason=validation_result.reason,
                        suggested_action=suggested_action,
                        blocked_reason=f"BLOCKED: {validation_result.reason}"
                    )
            
            # Text is valid
            return ActionSecurityResult(valid=True, reason="Text action is safe")
            
        except Exception as e:
            logger.error(f"Error validating text action: {e}")
            return ActionSecurityResult(
                valid=False,
                reason=f"Security validation error: {e}",
                blocked_reason="Security validation failed"
            )
    
    def validate_action(self, action: Dict[str, Any], security_config: Optional[Dict[str, Any]] = None) -> ActionSecurityResult:
        """Validate any action for security
        
        Args:
            action: Action to validate
            security_config: Optional security configuration
            
        Returns:
            ActionSecurityResult with validation outcome
        """
        action_type = action.get("type", "unknown")
        
        # Validate text/type actions
        if action_type == "type":
            text = action.get("text", "")
            if text:
                return self.validate_text_action(text, security_config)
        
        # For other action types, perform basic validation
        if action_type in ["click", "key", "wait", "scroll", "drag"]:
            return ActionSecurityResult(valid=True, reason="Standard automation action is safe")
        
        if action_type == "launch_app":
            app_name = action.get("app_name", "")
            if app_name:
                # Basic app name validation (could be enhanced)
                if self._is_safe_app_name(app_name):
                    return ActionSecurityResult(valid=True, reason="App launch is safe")
                else:
                    return ActionSecurityResult(
                        valid=False,
                        reason=f"Potentially unsafe app name: {app_name}",
                        blocked_reason="App launch blocked for security"
                    )
        
        # Unknown or potentially unsafe action types
        logger.warning(f"Unknown action type for security validation: {action_type}")
        return ActionSecurityResult(
            valid=True,  # Default to allowing unknown actions (could be made more restrictive)
            reason=f"Unknown action type '{action_type}' - defaulting to allow"
        )
    
    def _detect_action_type(self, text: str) -> TypeActionType:
        """Detect the type of action based on text content
        
        Args:
            text: Text to analyze
            
        Returns:
            TypeActionType enum value
        """
        text_lower = text.lower()
        
        # Enhanced URL detection
        url_indicators = [
            "http://", "https://", "www.", 
            ".com", ".org", ".net", ".io", ".edu", ".gov",
            ".co", ".uk", ".de", ".fr", ".jp", ".cn",
            ".ly", ".tk", ".ml", ".ga"  # Include shorteners and suspicious TLDs
        ]
        
        # Check if it looks like a URL
        if any(indicator in text_lower for indicator in url_indicators):
            return TypeActionType.URL
        
        # Also check for domain/path pattern (e.g., "site.com/path" or "bit.ly/xyz")
        if re.match(r'^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/|$)', text):
            return TypeActionType.URL
        
        # Check for email
        if "@" in text and "." in text:
            return TypeActionType.EMAIL
        
        # Default to text
        return TypeActionType.TEXT
    
    def _get_security_config(self, security_config: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Get security configuration
        
        Args:
            security_config: Optional custom security config
            
        Returns:
            Security configuration dictionary
        """
        if security_config:
            # Use passed security config
            return get_security_config(
                profile_name=security_config.get("security_profile", self.default_security_profile),
                custom_allowed=security_config.get("allowed_domains"),
                custom_blocked=security_config.get("blocked_domains")
            )
        else:
            # Fall back to environment or defaults
            return get_security_config(
                profile_name=os.environ.get("AGENTS2_SECURITY_PROFILE", self.default_security_profile)
            )
    
    def _is_safe_app_name(self, app_name: str) -> bool:
        """Check if an app name is safe to launch
        
        Args:
            app_name: Application name to check
            
        Returns:
            True if app name appears safe
        """
        # Basic validation - could be enhanced with whitelist/blacklist
        app_name_lower = app_name.lower()
        
        # Block obviously dangerous commands
        dangerous_patterns = [
            "rm ", "del ", "format", "fdisk", "dd if=", "mkfs",
            "shutdown", "reboot", "halt", "poweroff",
            "sudo ", "su ", "chmod 777", "passwd",
            "; ", " && ", " || ", "`", "$("
        ]
        
        for pattern in dangerous_patterns:
            if pattern in app_name_lower:
                logger.warning(f"Blocked potentially dangerous app name: {app_name}")
                return False
        
        return True
    
    def get_security_summary(self, actions: List[Dict[str, Any]], security_config: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Get security summary for a list of actions
        
        Args:
            actions: List of actions to analyze
            security_config: Optional security configuration
            
        Returns:
            Security analysis summary
        """
        summary = {
            "total_actions": len(actions),
            "safe_actions": 0,
            "blocked_actions": 0,
            "warnings": [],
            "blocked_details": [],
            "security_profile": security_config.get("security_profile", self.default_security_profile) if security_config else self.default_security_profile
        }
        
        for i, action in enumerate(actions):
            result = self.validate_action(action, security_config)
            
            if result.valid:
                summary["safe_actions"] += 1
            else:
                summary["blocked_actions"] += 1
                summary["blocked_details"].append({
                    "action_index": i,
                    "action_type": action.get("type", "unknown"),
                    "reason": result.reason,
                    "blocked_reason": result.blocked_reason
                })
            
            if result.reason and not result.valid:
                summary["warnings"].append(f"Action {i}: {result.reason}")
        
        return summary