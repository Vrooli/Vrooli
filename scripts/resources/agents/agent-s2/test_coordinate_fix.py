#!/usr/bin/env python3
"""
Test script to understand Agent-S2 URL security filtering behavior
"""

import sys
import os
from pathlib import Path

# Add the agent_s2 package to Python path
sys.path.insert(0, str(Path(__file__).parent / "agent_s2"))

from agent_s2.server.services.url_security import URLValidator, TypeActionType, SecurityProfile

def test_url_validation():
    """Test URL validation with various inputs"""
    
    test_cases = [
        # Valid URLs
        ("https://reddit.com", "Valid reddit URL"),
        ("https://www.reddit.com", "Valid reddit with www"),
        
        # Typosquatting patterns
        ("reddit.comreddit.com", "Doubled domain pattern"),
        ("redditreddit.com", "Doubled site name"),
        ("reddit.com.reddit", "Subdomain pattern"),
        
        # Other test cases
        ("reddit.com", "No protocol"),
        ("http://reddit.com", "HTTP instead of HTTPS"),
    ]
    
    print("Testing URL Validation with Agent-S2 Security")
    print("=" * 60)
    
    for url, description in test_cases:
        print(f"\nTesting: {url}")
        print(f"Description: {description}")
        
        # Test with moderate security profile (default)
        result = URLValidator.validate(
            url, 
            action_type=TypeActionType.URL,  
            security_profile=SecurityProfile.MODERATE
        )
        
        print(f"Valid: {result.valid}")
        if not result.valid:
            print(f"Reason: {result.reason}")
            if result.suggested_url:
                print(f"Suggested: {result.suggested_url}")
        
        print("-" * 40)

if __name__ == "__main__":
    test_url_validation()