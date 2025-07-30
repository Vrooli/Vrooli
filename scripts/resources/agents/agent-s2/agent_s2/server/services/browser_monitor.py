"""Browser navigation monitoring service for Agent S2"""

import os
import asyncio
import logging
import json
from typing import Dict, Any, Optional, Callable, List
from datetime import datetime
from urllib.parse import urlparse

from .reputation import DomainReputationService, ReputationResult
from .url_security import URLValidator, SecurityProfile, TypeActionType

logger = logging.getLogger(__name__)


class NavigationEvent:
    """Represents a browser navigation event"""
    
    def __init__(self, url: str, timestamp: datetime, 
                 window_id: Optional[str] = None,
                 tab_id: Optional[str] = None):
        self.url = url
        self.timestamp = timestamp
        self.window_id = window_id
        self.tab_id = tab_id
        self.domain = self._extract_domain(url)
        
    def _extract_domain(self, url: str) -> str:
        """Extract domain from URL"""
        try:
            parsed = urlparse(url)
            return parsed.netloc.lower()
        except:
            return ""
            
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "url": self.url,
            "domain": self.domain,
            "timestamp": self.timestamp.isoformat(),
            "window_id": self.window_id,
            "tab_id": self.tab_id
        }


class SecurityIncident:
    """Represents a security incident"""
    
    def __init__(self, event: NavigationEvent, reason: str,
                 severity: str = "medium", 
                 action_taken: str = "blocked"):
        self.event = event
        self.reason = reason
        self.severity = severity
        self.action_taken = action_taken
        self.timestamp = datetime.now()
        
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return {
            "timestamp": self.timestamp.isoformat(),
            "event": self.event.to_dict(),
            "reason": self.reason,
            "severity": self.severity,
            "action_taken": self.action_taken
        }


class BrowserSecurityMonitor:
    """Monitor browser navigation for security threats
    
    Note: This class provides security validation framework.
    Actual browser monitoring requires network-level proxy or browser extension.
    """
    
    def __init__(self, 
                 reputation_service: Optional[DomainReputationService] = None,
                 security_profile: SecurityProfile = SecurityProfile.MODERATE,
                 incident_callback: Optional[Callable] = None):
        self.reputation_service = reputation_service or DomainReputationService()
        self.security_profile = security_profile
        self.incident_callback = incident_callback
        
        self.last_url = None
        self.monitoring = False
        self.incidents: List[SecurityIncident] = []
        self.navigation_history: List[NavigationEvent] = []
        
        # Configuration
        self.check_interval = 0.5  # seconds
        self.max_history = 1000
        self.block_unsafe_navigation = True
        
        logger.info(f"BrowserSecurityMonitor initialized (profile: {security_profile.value})")
        
    async def start_monitoring(self):
        """Start monitoring browser navigation events
        
        Note: This is a placeholder. Actual monitoring requires:
        - Network-level proxy (mitmproxy)
        - Browser extension
        - DNS filtering
        """
        logger.info("Browser monitoring placeholder - requires network proxy integration")
        return
            
        self.monitoring = True
        logger.info("Started browser security monitoring")
        
        try:
            while self.monitoring:
                await self._check_navigation()
                await asyncio.sleep(self.check_interval)
        except Exception as e:
            logger.error(f"Monitoring error: {e}")
        finally:
            self.monitoring = False
            logger.info("Stopped browser security monitoring")
            
    def stop_monitoring(self):
        """Stop monitoring"""
        self.monitoring = False
        
    async def _check_navigation(self):
        """Check current browser URL"""
        # Placeholder for network proxy integration
        # In a real implementation, the proxy would call this method
        # with captured navigation events
        return
                
            # New navigation detected
            logger.info(f"Navigation detected: {current_url}")
            event = NavigationEvent(
                url=current_url,
                timestamp=datetime.now()
            )
            
            # Add to history
            self.navigation_history.append(event)
            if len(self.navigation_history) > self.max_history:
                self.navigation_history.pop(0)
                
            # Validate navigation
            await self._validate_navigation(event)
            
            self.last_url = current_url
            
        except Exception as e:
            logger.debug(f"Failed to check navigation: {e}")
            
    async def _validate_navigation(self, event: NavigationEvent):
        """Validate a navigation event"""
        url = event.url
        
        # First, check URL format and patterns
        validation_result = URLValidator.validate(
            url,
            action_type=TypeActionType.URL,
            security_profile=self.security_profile
        )
        
        if not validation_result.valid:
            logger.warning(f"URL validation failed: {validation_result.reason}")
            await self._handle_security_incident(
                event,
                reason=f"URL validation: {validation_result.reason}",
                severity="high"
            )
            return
            
        # Then check domain reputation if available
        if self.reputation_service and event.domain:
            reputation_result = await self.reputation_service.check_domain(event.domain)
            
            if not reputation_result.safe:
                threat_info = f"Malicious: {reputation_result.malicious_count}, Suspicious: {reputation_result.suspicious_count}"
                await self._handle_security_incident(
                    event,
                    reason=f"Reputation check failed: {threat_info}",
                    severity="critical" if reputation_result.malicious_count > 0 else "high"
                )
                
    async def _handle_security_incident(self, event: NavigationEvent, 
                                      reason: str, severity: str = "medium"):
        """Handle a security incident"""
        incident = SecurityIncident(
            event=event,
            reason=reason,
            severity=severity,
            action_taken="blocked" if self.block_unsafe_navigation else "logged"
        )
        
        self.incidents.append(incident)
        logger.warning(f"Security incident: {reason} for URL: {event.url}")
        
        # Log incident details
        await self._log_security_incident(incident)
        
        # Take action if configured
        if self.block_unsafe_navigation:
            await self._block_navigation(event)
            
        # Notify callback if configured
        if self.incident_callback:
            try:
                await self.incident_callback(incident)
            except Exception as e:
                logger.error(f"Incident callback failed: {e}")
                
    async def _block_navigation(self, event: NavigationEvent):
        """Block unsafe navigation"""
        try:
            logger.info(f"Blocking navigation to: {event.url}")
            
            # In a network proxy implementation:
            # 1. Return HTTP 403 with custom block page
            # 2. Log the incident
            # 3. Prevent request from reaching destination
            logger.info(f"Would block navigation to {event.url}")
            
            # Future: Return block page HTML to proxy
            # block_page = self._generate_block_page(event, reason, severity)
            # return block_page
            
        except Exception as e:
            logger.error(f"Failed to block navigation: {e}")
            
    async def _log_security_incident(self, incident: SecurityIncident):
        """Log security incident to file"""
        try:
            log_dir = os.path.expanduser("~/.agent-s2/security_logs")
            os.makedirs(log_dir, exist_ok=True)
            
            log_file = os.path.join(log_dir, f"incidents_{datetime.now().strftime('%Y%m%d')}.json")
            
            # Read existing incidents
            incidents = []
            if os.path.exists(log_file):
                try:
                    with open(log_file, 'r') as f:
                        incidents = json.load(f)
                except:
                    incidents = []
                    
            # Add new incident
            incidents.append(incident.to_dict())
            
            # Write back
            with open(log_file, 'w') as f:
                json.dump(incidents, f, indent=2)
                
            logger.info(f"Security incident logged to: {log_file}")
            
        except Exception as e:
            logger.error(f"Failed to log security incident: {e}")
            
    def get_incidents(self, limit: Optional[int] = None) -> List[Dict[str, Any]]:
        """Get recent security incidents"""
        incidents = [inc.to_dict() for inc in self.incidents]
        if limit:
            return incidents[-limit:]
        return incidents
        
    def get_navigation_history(self, limit: Optional[int] = None) -> List[Dict[str, Any]]:
        """Get navigation history"""
        history = [event.to_dict() for event in self.navigation_history]
        if limit:
            return history[-limit:]
        return history
        
    def get_stats(self) -> Dict[str, Any]:
        """Get monitoring statistics"""
        incident_by_severity = {}
        for incident in self.incidents:
            severity = incident.severity
            incident_by_severity[severity] = incident_by_severity.get(severity, 0) + 1
            
        return {
            "monitoring": self.monitoring,
            "total_navigations": len(self.navigation_history),
            "total_incidents": len(self.incidents),
            "incidents_by_severity": incident_by_severity,
            "current_url": self.last_url,
            "security_profile": self.security_profile.value
        }
        
    def clear_history(self):
        """Clear navigation history"""
        self.navigation_history.clear()
        logger.info("Navigation history cleared")
        
    def clear_incidents(self):
        """Clear incident history"""
        self.incidents.clear()
        logger.info("Incident history cleared")