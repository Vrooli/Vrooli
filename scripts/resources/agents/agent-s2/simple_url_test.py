#!/usr/bin/env python3
"""
Simple test to check URL validation logic directly
"""

import re
from enum import Enum
from typing import Optional, List
from dataclasses import dataclass

class TypeActionType(Enum):
    """Semantic types for text input actions"""
    TEXT = "text"
    URL = "url"
    SEARCH = "search"
    PASSWORD = "password"
    EMAIL = "email"
    FILENAME = "filename"

class SecurityProfile(Enum):
    """Security profile levels"""
    STRICT = "strict"
    MODERATE = "moderate"
    PERMISSIVE = "permissive"

@dataclass
class ValidationResult:
    """Result of URL validation"""
    valid: bool
    reason: Optional[str] = None
    suggested_url: Optional[str] = None

def check_typosquatting(domain: str) -> ValidationResult:
    """Check for common typosquatting patterns - simplified version"""
    
    # Common sites to check against
    COMMON_SITES = ["reddit", "google", "facebook", "amazon", "github", 
                    "stackoverflow", "twitter", "youtube", "linkedin", 
                    "microsoft", "apple", "netflix", "paypal", "ebay"]
    
    # Remove port if present
    if ":" in domain:
        domain = domain.split(":")[0]
        
    # Check for doubled domain pattern (e.g., reddit.comreddit.com)
    for site in COMMON_SITES:
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
            
    return ValidationResult(valid=True)

def test_url_patterns():
    """Test specific URL patterns we're interested in"""
    
    test_cases = [
        "reddit.com",
        "reddit.comreddit.com", 
        "redditreddit.com",
        "reddit.com.reddit",
        "www.reddit.com",
        "facebook.comfacebook.com",
        "googlegoogle.com"
    ]
    
    print("Testing Typosquatting Detection")
    print("=" * 50)
    
    for domain in test_cases:
        print(f"\nTesting domain: {domain}")
        result = check_typosquatting(domain)
        print(f"Valid: {result.valid}")
        if not result.valid:
            print(f"Reason: {result.reason}")
            print(f"Suggested: {result.suggested_url}")
        print("-" * 30)

if __name__ == "__main__":
    test_url_patterns()