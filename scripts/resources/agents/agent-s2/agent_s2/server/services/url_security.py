"""URL security validation service for Agent S2"""

import re
import logging
import asyncio
from typing import Dict, Any, Optional, List
from enum import Enum
from urllib.parse import urlparse, urlunparse

logger = logging.getLogger(__name__)


class TypeActionType(Enum):
    """Semantic types for text input actions"""
    TEXT = "text"           # Generic text input
    URL = "url"             # Website URL
    SEARCH = "search"       # Search query
    PASSWORD = "password"   # Sensitive input
    EMAIL = "email"         # Email address
    FILENAME = "filename"   # File path/name


class SecurityProfile(Enum):
    """Security profile levels"""
    STRICT = "strict"       # Whitelist only
    MODERATE = "moderate"   # Block suspicious domains
    PERMISSIVE = "permissive"  # Minimal blocking


class ValidationResult:
    """Result of URL validation"""
    def __init__(self, valid: bool, reason: Optional[str] = None, 
                 suggested_url: Optional[str] = None):
        self.valid = valid
        self.reason = reason
        self.suggested_url = suggested_url


class URLValidator:
    """URL validation and security checks"""
    
    # Common typosquatting patterns for popular sites
    COMMON_SITES = ["reddit", "google", "facebook", "amazon", "github", 
                    "stackoverflow", "twitter", "youtube", "linkedin", 
                    "microsoft", "apple", "netflix", "paypal", "ebay"]
    
    # Suspicious TLDs often used for scams
    SUSPICIOUS_TLDS = [".tk", ".ml", ".ga", ".cf", ".click", ".download",
                       ".loan", ".racing", ".review", ".stream", ".trade"]
    
    # Default blocked patterns
    DEFAULT_BLOCKED_PATTERNS = [
        r".*\.tk$",           # Free TLDs
        r".*\.ml$",
        r".*\.ga$", 
        r".*\.cf$",
        r"bit\.ly",           # URL shorteners
        r"tinyurl\.com",
        r"goo\.gl",
        r"ow\.ly"
    ]
    
    @staticmethod
    def validate(url: str, action_type: TypeActionType = TypeActionType.URL,
                 allowed_domains: Optional[List[str]] = None,
                 blocked_domains: Optional[List[str]] = None,
                 security_profile: SecurityProfile = SecurityProfile.MODERATE) -> ValidationResult:
        """Validate a URL for security issues
        
        Args:
            url: URL to validate
            action_type: Type of action (only validates if URL type)
            allowed_domains: Whitelist of allowed domains
            blocked_domains: Blacklist of blocked domains
            security_profile: Security profile to apply
            
        Returns:
            ValidationResult with validation status
        """
        # Only validate URL-type actions
        if action_type != TypeActionType.URL:
            return ValidationResult(valid=True)
            
        # Ensure URL has protocol
        if not url.startswith(("http://", "https://")):
            # Suggest HTTPS version
            suggested_url = f"https://{url}"
            return ValidationResult(
                valid=False, 
                reason="URL must include protocol (https:// recommended)",
                suggested_url=suggested_url
            )
            
        # Basic URL format validation
        try:
            parsed = urlparse(url)
            if not parsed.netloc:
                return ValidationResult(valid=False, reason="Invalid URL format")
        except Exception as e:
            return ValidationResult(valid=False, reason=f"URL parsing error: {e}")
            
        # Extract domain
        domain = parsed.netloc.lower()
        
        # Apply security profile rules
        if security_profile == SecurityProfile.STRICT:
            # In strict mode, only whitelisted domains allowed
            if not allowed_domains:
                return ValidationResult(
                    valid=False, 
                    reason="Strict mode requires whitelist but none provided"
                )
            if not URLValidator._domain_in_list(domain, allowed_domains):
                return ValidationResult(
                    valid=False,
                    reason=f"Domain {domain} not in whitelist"
                )
                
        # Check for typosquatting
        typo_result = URLValidator._check_typosquatting(domain)
        if not typo_result.valid:
            return typo_result
            
        # Check against blacklist
        if blocked_domains:
            if URLValidator._domain_in_list(domain, blocked_domains):
                return ValidationResult(
                    valid=False,
                    reason=f"Domain {domain} is blocked"
                )
                
        # Check suspicious TLDs
        if security_profile in [SecurityProfile.STRICT, SecurityProfile.MODERATE]:
            for tld in URLValidator.SUSPICIOUS_TLDS:
                if domain.endswith(tld):
                    return ValidationResult(
                        valid=False,
                        reason=f"Suspicious TLD: {tld}"
                    )
                    
        # Check default blocked patterns
        if security_profile != SecurityProfile.PERMISSIVE:
            for pattern in URLValidator.DEFAULT_BLOCKED_PATTERNS:
                if re.match(pattern, domain):
                    return ValidationResult(
                        valid=False,
                        reason=f"Domain matches blocked pattern: {pattern}"
                    )
                    
        # Prefer HTTPS
        if parsed.scheme == "http" and security_profile == SecurityProfile.STRICT:
            suggested_url = urlunparse(parsed._replace(scheme="https"))
            return ValidationResult(
                valid=False,
                reason="HTTPS required in strict mode",
                suggested_url=suggested_url
            )
            
        return ValidationResult(valid=True)
    
    @staticmethod
    def _check_typosquatting(domain: str) -> ValidationResult:
        """Check for common typosquatting patterns
        
        Args:
            domain: Domain to check
            
        Returns:
            ValidationResult
        """
        # Remove port if present
        if ":" in domain:
            domain = domain.split(":")[0]
            
        # Check for doubled domain pattern (e.g., reddit.comreddit.com)
        for site in URLValidator.COMMON_SITES:
            # Pattern: site.comsite
            if f"{site}.com{site}" in domain:
                return ValidationResult(
                    valid=False,
                    reason=f"Typosquatting detected: doubled domain pattern ({site}.com{site})",
                    suggested_url=f"https://{site}.com"
                )
            # Pattern: sitesite.com  
            if f"{site}{site}.com" in domain:
                return ValidationResult(
                    valid=False,
                    reason=f"Typosquatting detected: doubled site name ({site}{site}.com)",
                    suggested_url=f"https://{site}.com"
                )
            # Pattern: site.com.site
            if f"{site}.com.{site}" in domain:
                return ValidationResult(
                    valid=False,
                    reason=f"Typosquatting detected: suspicious subdomain pattern",
                    suggested_url=f"https://{site}.com"
                )
                
        # Check for character substitution
        substitutions = {
            '0': 'o', '1': 'l', '3': 'e', '5': 's', '4': 'a',
            '@': 'a', '!': 'i', '$': 's'
        }
        
        for site in URLValidator.COMMON_SITES:
            # Check if domain contains suspicious substitutions
            for char, legit in substitutions.items():
                test_domain = domain.replace(char, legit)
                if site in test_domain and site not in domain:
                    return ValidationResult(
                        valid=False,
                        reason=f"Possible typosquatting: character substitution detected",
                        suggested_url=f"https://{site}.com"
                    )
                    
        # Check for extra characters (e.g., reddit-com.com)
        for site in URLValidator.COMMON_SITES:
            if site in domain and f"{site}.com" not in domain:
                # Domain contains the site name but not the correct format
                if re.search(f"{site}[\\-_.]com", domain):
                    return ValidationResult(
                        valid=False,
                        reason=f"Possible typosquatting: suspicious domain variation",
                        suggested_url=f"https://{site}.com"
                    )
                    
        return ValidationResult(valid=True)
    
    @staticmethod
    def _domain_in_list(domain: str, domain_list: List[str]) -> bool:
        """Check if domain matches any pattern in list
        
        Args:
            domain: Domain to check
            domain_list: List of domain patterns (supports wildcards)
            
        Returns:
            True if domain matches any pattern
        """
        for pattern in domain_list:
            if URLValidator._domain_matches(domain, pattern):
                return True
        return False
    
    @staticmethod
    def _domain_matches(domain: str, pattern: str) -> bool:
        """Check if domain matches pattern
        
        Args:
            domain: Domain to check
            pattern: Pattern to match (supports wildcards)
            
        Returns:
            True if domain matches pattern
        """
        # Handle wildcards
        if pattern == "*":
            return True
        if pattern.startswith("*."):
            # Subdomain wildcard
            return domain.endswith(pattern[2:]) or domain == pattern[2:]
        if pattern.endswith(".*"):
            # TLD wildcard
            return domain.startswith(pattern[:-2])
            
        # Exact match
        return domain == pattern


class TypeAction:
    """Enhanced type action with validation"""
    
    def __init__(self, text: str, action_type: TypeActionType = TypeActionType.TEXT,
                 security_config: Optional[Dict[str, Any]] = None):
        self.text = text
        self.action_type = action_type
        self.security_config = security_config or {}
        
    def validate(self) -> ValidationResult:
        """Validate the action based on its type
        
        Returns:
            ValidationResult
        """
        if self.action_type == TypeActionType.URL:
            return URLValidator.validate(
                self.text,
                action_type=self.action_type,
                allowed_domains=self.security_config.get("allowed_domains"),
                blocked_domains=self.security_config.get("blocked_domains"),
                security_profile=SecurityProfile(
                    self.security_config.get("security_profile", "moderate")
                )
            )
            
        # Other action types pass through
        return ValidationResult(valid=True)


# Security profile configurations
SECURITY_PROFILES = {
    "strict": {
        "allowed_domains": [
            "wikipedia.org", "*.wikipedia.org",
            "github.com", "*.github.com", 
            "stackoverflow.com", "*.stackoverflow.com",
            "docs.python.org", "developer.mozilla.org",
            "google.com", "*.google.com"
        ],
        "blocked_domains": ["*"],  # Block all except allowed
        "require_https": True,
        "check_reputation": True
    },
    "moderate": {
        "blocked_domains": [
            "*.tk", "*.ml", "*.ga", "*.cf",  # Free TLDs
            "bit.ly", "tinyurl.com", "goo.gl",  # URL shorteners
            "*.click", "*.download", "*.loan"  # Suspicious TLDs
        ],
        "require_https": False,
        "check_reputation": True
    },
    "permissive": {
        "blocked_domains": [],
        "require_https": False,
        "check_reputation": False
    }
}


def get_security_config(profile_name: str = "moderate",
                       custom_allowed: Optional[List[str]] = None,
                       custom_blocked: Optional[List[str]] = None) -> Dict[str, Any]:
    """Get security configuration for a profile
    
    Args:
        profile_name: Name of security profile
        custom_allowed: Additional allowed domains
        custom_blocked: Additional blocked domains
        
    Returns:
        Security configuration dict
    """
    base_config = SECURITY_PROFILES.get(profile_name, SECURITY_PROFILES["moderate"]).copy()
    
    # Merge custom lists
    if custom_allowed:
        base_config["allowed_domains"] = base_config.get("allowed_domains", []) + custom_allowed
        
    if custom_blocked:
        base_config["blocked_domains"] = base_config.get("blocked_domains", []) + custom_blocked
        
    base_config["security_profile"] = profile_name
    
    return base_config