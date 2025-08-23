"""Domain reputation checking service for Agent S2"""

import os
import time
import json
import logging
import hashlib
import base64
from typing import Dict, Any, Optional, List
from dataclasses import dataclass
from datetime import datetime, timedelta
from urllib.parse import urlparse

try:
    import aiohttp
except ImportError:
    aiohttp = None
    
try:
    import asyncio
except ImportError:
    asyncio = None

logger = logging.getLogger(__name__)


@dataclass
class ReputationResult:
    """Result of domain reputation check"""
    safe: bool
    malicious_count: int = 0
    suspicious_count: int = 0
    harmless_count: int = 0
    undetected_count: int = 0
    reputation_score: int = 0
    scan_date: Optional[datetime] = None
    cached: bool = False
    error: Optional[str] = None
    details: Optional[Dict[str, Any]] = None


@dataclass
class ScanResult:
    """VirusTotal scan result"""
    malicious_count: int
    suspicious_count: int
    harmless_count: int = 0
    undetected_count: int = 0
    reputation_score: int = 0
    error: Optional[str] = None


class ReputationCache:
    """Simple in-memory cache for reputation results"""
    
    def __init__(self, ttl_hours: int = 24):
        self.cache: Dict[str, tuple[ReputationResult, datetime]] = {}
        self.ttl = timedelta(hours=ttl_hours)
        
    def get(self, domain: str) -> Optional[ReputationResult]:
        """Get cached result if still valid"""
        if domain in self.cache:
            result, timestamp = self.cache[domain]
            if datetime.now() - timestamp < self.ttl:
                result.cached = True
                return result
            else:
                # Remove expired entry
                del self.cache[domain]
        return None
        
    def set(self, domain: str, result: ReputationResult):
        """Cache a result"""
        self.cache[domain] = (result, datetime.now())
        
    def clear(self):
        """Clear all cached results"""
        self.cache.clear()
        
    def stats(self) -> Dict[str, Any]:
        """Get cache statistics"""
        total = len(self.cache)
        expired = sum(
            1 for _, (_, timestamp) in self.cache.items()
            if datetime.now() - timestamp >= self.ttl
        )
        return {
            "total_entries": total,
            "active_entries": total - expired,
            "expired_entries": expired,
            "cache_size_bytes": len(str(self.cache))
        }


class VirusTotalClient:
    """VirusTotal API v3 client"""
    
    def __init__(self, api_key: str):
        self.api_key = api_key
        self.base_url = "https://www.virustotal.com/api/v3"
        
    async def scan_url(self, url: str) -> ScanResult:
        """Scan URL for malicious content"""
        if not aiohttp:
            logger.error("aiohttp not available for VirusTotal API")
            return ScanResult(0, 0, error="aiohttp not installed")
            
        try:
            # Create URL ID for VirusTotal
            url_id = base64.urlsafe_b64encode(url.encode()).decode().strip("=")
            
            async with aiohttp.ClientSession() as session:
                headers = {"x-apikey": self.api_key}
                
                # First check if URL already has results
                analysis_url = f"{self.base_url}/urls/{url_id}"
                
                async with session.get(analysis_url, headers=headers) as resp:
                    if resp.status == 200:
                        # URL already analyzed
                        analysis_result = await resp.json()
                        return self._parse_scan_result(analysis_result)
                    elif resp.status == 404:
                        # Need to submit URL for scanning
                        return await self._submit_and_scan(session, headers, url)
                    else:
                        error_text = await resp.text()
                        logger.error(f"VirusTotal API error: {resp.status} - {error_text}")
                        return ScanResult(0, 0, error=f"API error: {resp.status}")
                        
        except Exception as e:
            logger.error(f"VirusTotal scan failed: {e}")
            return ScanResult(0, 0, error=str(e))
            
    async def _submit_and_scan(self, session: aiohttp.ClientSession, 
                              headers: Dict[str, str], url: str) -> ScanResult:
        """Submit URL for scanning and get results"""
        try:
            # Submit URL for scanning
            submit_url = f"{self.base_url}/urls"
            submit_data = {"url": url}
            
            async with session.post(submit_url, headers=headers, data=submit_data) as resp:
                if resp.status not in [200, 201]:
                    error_text = await resp.text()
                    return ScanResult(0, 0, error=f"Submit failed: {error_text}")
                    
                submit_result = await resp.json()
                
            # Get analysis ID from submission
            analysis_id = submit_result.get("data", {}).get("id")
            if not analysis_id:
                return ScanResult(0, 0, error="No analysis ID returned")
                
            # Poll for analysis completion (with timeout)
            analysis_url = f"{self.base_url}/analyses/{analysis_id}"
            max_attempts = 30  # 30 seconds timeout
            
            for attempt in range(max_attempts):
                await asyncio.sleep(1)  # Wait 1 second between polls
                
                async with session.get(analysis_url, headers=headers) as resp:
                    if resp.status == 200:
                        analysis_data = await resp.json()
                        status = analysis_data.get("data", {}).get("attributes", {}).get("status")
                        
                        if status == "completed":
                            # Get final results
                            stats = analysis_data.get("data", {}).get("attributes", {}).get("stats", {})
                            return ScanResult(
                                malicious_count=stats.get("malicious", 0),
                                suspicious_count=stats.get("suspicious", 0),
                                harmless_count=stats.get("harmless", 0),
                                undetected_count=stats.get("undetected", 0)
                            )
                        elif status == "failed":
                            return ScanResult(0, 0, error="Analysis failed")
                            
            return ScanResult(0, 0, error="Analysis timeout")
            
        except Exception as e:
            logger.error(f"Submit and scan failed: {e}")
            return ScanResult(0, 0, error=str(e))
            
    def _parse_scan_result(self, vt_response: dict) -> ScanResult:
        """Parse VirusTotal response into ScanResult"""
        try:
            stats = vt_response.get("data", {}).get("attributes", {}).get("last_analysis_stats", {})
            reputation = vt_response.get("data", {}).get("attributes", {}).get("reputation", 0)
            
            return ScanResult(
                malicious_count=stats.get("malicious", 0),
                suspicious_count=stats.get("suspicious", 0),
                harmless_count=stats.get("harmless", 0),
                undetected_count=stats.get("undetected", 0),
                reputation_score=reputation
            )
        except Exception as e:
            logger.error(f"Failed to parse VirusTotal response: {e}")
            return ScanResult(0, 0, error="Parse error")


class DomainReputationService:
    """Service for checking domain reputation"""
    
    def __init__(self, virustotal_api_key: Optional[str] = None,
                 cache_ttl_hours: int = 24,
                 check_reputation: bool = True):
        self.virustotal_api_key = virustotal_api_key or os.environ.get("AGENT_S2_VIRUSTOTAL_API_KEY")
        self.check_reputation = check_reputation
        self.cache = ReputationCache(ttl_hours=cache_ttl_hours)
        self.vt_client = VirusTotalClient(self.virustotal_api_key) if self.virustotal_api_key else None
        
        # Hardcoded malicious domains for immediate blocking
        self.known_malicious = {
            "malware-test.com",
            "phishing-test.com",
            "scam-site.tk"
        }
        
        logger.info(f"DomainReputationService initialized (API key: {'configured' if self.virustotal_api_key else 'not configured'})")
        
    async def check_domain(self, domain: str) -> ReputationResult:
        """Check domain reputation"""
        # Normalize domain
        domain = domain.lower().strip()
        
        # Check cache first
        cached_result = self.cache.get(domain)
        if cached_result:
            logger.debug(f"Cache hit for domain: {domain}")
            return cached_result
            
        # Check known malicious list
        if domain in self.known_malicious:
            result = ReputationResult(
                safe=False,
                malicious_count=1,
                details={"reason": "Known malicious domain"}
            )
            self.cache.set(domain, result)
            return result
            
        # Check VirusTotal if enabled and API key available
        if self.check_reputation and self.vt_client:
            try:
                logger.info(f"Checking VirusTotal for domain: {domain}")
                # Construct full URL for scanning
                url = f"https://{domain}"
                scan_result = await self.vt_client.scan_url(url)
                
                # Determine safety based on scan results
                is_safe = (
                    scan_result.malicious_count == 0 and 
                    scan_result.suspicious_count <= 1 and
                    not scan_result.error
                )
                
                result = ReputationResult(
                    safe=is_safe,
                    malicious_count=scan_result.malicious_count,
                    suspicious_count=scan_result.suspicious_count,
                    harmless_count=scan_result.harmless_count,
                    undetected_count=scan_result.undetected_count,
                    reputation_score=scan_result.reputation_score,
                    scan_date=datetime.now(),
                    error=scan_result.error,
                    details={
                        "virustotal_scan": True,
                        "url_scanned": url
                    }
                )
                
                # Cache the result
                self.cache.set(domain, result)
                return result
                
            except Exception as e:
                logger.error(f"Reputation check failed for {domain}: {e}")
                # Return neutral result on error
                return ReputationResult(
                    safe=True,
                    error=str(e),
                    details={"exception": type(e).__name__}
                )
        else:
            # No reputation check available
            logger.debug(f"Reputation check skipped for {domain} (not configured)")
            return ReputationResult(
                safe=True,
                details={"reputation_check": "disabled"}
            )
            
    async def check_url(self, url: str) -> ReputationResult:
        """Check reputation of a full URL"""
        try:
            parsed = urlparse(url)
            domain = parsed.netloc.lower()
            if not domain:
                return ReputationResult(
                    safe=False,
                    error="Invalid URL: no domain found"
                )
            return await self.check_domain(domain)
        except Exception as e:
            logger.error(f"Failed to parse URL {url}: {e}")
            return ReputationResult(
                safe=False,
                error=f"URL parse error: {e}"
            )
            
    def get_cache_stats(self) -> Dict[str, Any]:
        """Get cache statistics"""
        return self.cache.stats()
        
    def clear_cache(self):
        """Clear reputation cache"""
        self.cache.clear()
        logger.info("Reputation cache cleared")