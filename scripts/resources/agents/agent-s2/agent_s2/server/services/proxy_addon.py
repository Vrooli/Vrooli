"""mitmproxy addon for Agent S2 security filtering"""

import os
import asyncio
import logging
from typing import Optional
from datetime import datetime
from urllib.parse import urlparse

from mitmproxy import http, ctx
from mitmproxy.script import concurrent

try:
    # Try relative imports first (when loaded as module)
    from .url_security import URLValidator, ValidationResult, TypeActionType, get_security_config
    from .reputation import DomainReputationService, ReputationResult
    from ...config import Config
except ImportError:
    # Fall back to absolute imports (when loaded as mitmproxy script)
    import sys
    import os
    # Add the agent_s2 directory to Python path
    sys.path.insert(0, '/opt/agent-s2')
    from agent_s2.server.services.url_security import URLValidator, ValidationResult, TypeActionType, get_security_config
    from agent_s2.server.services.reputation import DomainReputationService, ReputationResult
    from agent_s2.config import Config

logger = logging.getLogger(__name__)


class SecurityProxyAddon:
    """mitmproxy addon that implements URL security filtering"""
    
    def __init__(self):
        """Initialize the security proxy addon"""
        self.validator = URLValidator()
        # Use the main Config class for security configuration
        self.config = Config
        
        # Initialize reputation service with API key from env/config
        api_key = self.config.VIRUSTOTAL_API_KEY
        
        self.reputation_service = DomainReputationService(
            virustotal_api_key=api_key,
            cache_ttl_hours=24,
            check_reputation=bool(api_key)
        )
        
        # Load security profile
        profile_name = os.environ.get("AGENT_S2_SECURITY_PROFILE", "moderate")
        self.security_profile = self.security_config.get_profile(profile_name)
        
        # Stats
        self.requests_checked = 0
        self.requests_blocked = 0
        self.start_time = datetime.now()
        
        logger.info(f"SecurityProxyAddon initialized (profile: {profile_name}, "
                   f"VirusTotal: {'enabled' if api_key else 'disabled'})")
    
    def load(self, loader):
        """Called when addon is first loaded"""
        ctx.log.info("Agent S2 Security Proxy loaded")
    
    @concurrent
    async def request(self, flow: http.HTTPFlow) -> None:
        """Intercept and validate outgoing requests"""
        try:
            url = flow.request.pretty_url
            host = flow.request.pretty_host
            
            # Skip internal/local requests
            if self._is_internal_request(host):
                return
            
            self.requests_checked += 1
            
            # Step 1: URL validation (typosquatting, patterns)
            validation_result = self.validator.validate(
                url,
                action_type=TypeActionType.URL,
                security_profile=self.security_profile
            )
            
            if not validation_result.valid:
                logger.warning(f"URL validation failed for {url}: {validation_result.reason}")
                await self._block_request(flow, validation_result.reason, "high")
                return
            
            # Step 2: Domain reputation check (if enabled)
            if self.reputation_service.check_reputation:
                reputation_result = await self.reputation_service.check_domain(host)
                
                if not reputation_result.safe:
                    severity = "critical" if reputation_result.malicious_count > 0 else "high"
                    reason = self._format_reputation_reason(reputation_result)
                    logger.warning(f"Reputation check failed for {host}: {reason}")
                    await self._block_request(flow, reason, severity, reputation_result)
                    return
                    
            # Request passed all checks
            logger.debug(f"Request allowed: {url}")
            
        except Exception as e:
            logger.error(f"Error processing request: {e}")
            # On error, allow request to proceed
    
    def _is_internal_request(self, host: str) -> bool:
        """Check if request is to internal/local addresses"""
        internal_patterns = [
            "localhost",
            "127.0.0.1",
            "::1",
            "0.0.0.0",
            ".local",
            ".internal"
        ]
        
        host_lower = host.lower()
        return any(pattern in host_lower for pattern in internal_patterns)
    
    def _format_reputation_reason(self, result: ReputationResult) -> str:
        """Format reputation check result into human-readable reason"""
        parts = []
        
        if result.malicious_count > 0:
            parts.append(f"{result.malicious_count} security vendors flagged as malicious")
        
        if result.suspicious_count > 0:
            parts.append(f"{result.suspicious_count} vendors flagged as suspicious")
            
        if result.error:
            parts.append(f"Check error: {result.error}")
            
        return "; ".join(parts) if parts else "Failed reputation check"
    
    async def _block_request(self, flow: http.HTTPFlow, reason: str, 
                           severity: str = "high", 
                           reputation_result: Optional[ReputationResult] = None):
        """Block request with custom error page"""
        self.requests_blocked += 1
        
        # Generate block page HTML
        block_page = self._generate_block_page(
            url=flow.request.url,
            domain=flow.request.pretty_host,
            reason=reason,
            severity=severity,
            reputation_result=reputation_result
        )
        
        # Create response
        flow.response = http.Response.make(
            403,  # Forbidden
            block_page,
            {"Content-Type": "text/html; charset=utf-8"}
        )
        
        # Log the incident
        await self._log_blocked_request(flow, reason, severity)
    
    def _generate_block_page(self, url: str, domain: str, reason: str, 
                           severity: str, reputation_result: Optional[ReputationResult] = None) -> str:
        """Generate HTML block page"""
        
        # VirusTotal details section
        vt_section = ""
        if reputation_result and not reputation_result.error:
            vt_section = f"""
            <div class="virustotal-info">
                <h3>🔍 VirusTotal Analysis</h3>
                <table>
                    <tr><td>Malicious detections:</td><td class="bad">{reputation_result.malicious_count}</td></tr>
                    <tr><td>Suspicious detections:</td><td class="warning">{reputation_result.suspicious_count}</td></tr>
                    <tr><td>Clean detections:</td><td class="good">{reputation_result.harmless_count}</td></tr>
                    <tr><td>No detection:</td><td>{reputation_result.undetected_count}</td></tr>
                </table>
            </div>
            """
        
        severity_class = {
            "critical": "critical",
            "high": "high",
            "medium": "medium",
            "low": "low"
        }.get(severity, "medium")
        
        return f"""
<!DOCTYPE html>
<html>
<head>
    <title>Security Warning - Navigation Blocked</title>
    <meta charset="utf-8">
    <style>
        body {{
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Arial, sans-serif;
            margin: 0;
            padding: 0;
            background: #f5f5f7;
            color: #1d1d1f;
            line-height: 1.6;
        }}
        .container {{
            max-width: 700px;
            margin: 50px auto;
            padding: 20px;
        }}
        .warning-box {{
            background: white;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.08);
            padding: 40px;
            border-top: 4px solid #ff3b30;
        }}
        h1 {{
            color: #ff3b30;
            font-size: 28px;
            margin: 0 0 20px 0;
            display: flex;
            align-items: center;
            gap: 10px;
        }}
        .icon {{
            font-size: 32px;
        }}
        .details {{
            background: #f5f5f7;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
            font-family: "SF Mono", Monaco, monospace;
            font-size: 14px;
        }}
        .details strong {{
            color: #1d1d1f;
            font-weight: 600;
        }}
        .severity {{
            display: inline-block;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            margin-left: 10px;
        }}
        .severity.critical {{
            background: #ff3b30;
            color: white;
        }}
        .severity.high {{
            background: #ff9500;
            color: white;
        }}
        .severity.medium {{
            background: #ffcc00;
            color: #1d1d1f;
        }}
        .virustotal-info {{
            background: #f5f5f7;
            border-radius: 8px;
            padding: 20px;
            margin: 20px 0;
        }}
        .virustotal-info h3 {{
            margin: 0 0 15px 0;
            color: #1d1d1f;
            font-size: 18px;
        }}
        .virustotal-info table {{
            width: 100%;
            border-collapse: collapse;
        }}
        .virustotal-info td {{
            padding: 8px 0;
            border-bottom: 1px solid #e5e5e7;
        }}
        .virustotal-info td:last-child {{
            text-align: right;
            font-weight: 600;
        }}
        .virustotal-info .bad {{
            color: #ff3b30;
        }}
        .virustotal-info .warning {{
            color: #ff9500;
        }}
        .virustotal-info .good {{
            color: #34c759;
        }}
        .actions {{
            margin-top: 30px;
            display: flex;
            gap: 15px;
        }}
        .actions a {{
            display: inline-block;
            padding: 12px 24px;
            border-radius: 8px;
            text-decoration: none;
            font-weight: 500;
            transition: all 0.2s;
        }}
        .actions .primary {{
            background: #007aff;
            color: white;
        }}
        .actions .primary:hover {{
            background: #0051d5;
        }}
        .actions .secondary {{
            background: #f5f5f7;
            color: #007aff;
            border: 1px solid #d1d1d6;
        }}
        .actions .secondary:hover {{
            background: #e5e5e7;
        }}
        .footer {{
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #d1d1d6;
            font-size: 14px;
            color: #86868b;
            text-align: center;
        }}
        .url-display {{
            word-break: break-all;
            color: #86868b;
            font-size: 13px;
        }}
    </style>
</head>
<body>
    <div class="container">
        <div class="warning-box">
            <h1>
                <span class="icon">⛔</span>
                Navigation Blocked
                <span class="severity {severity_class}">{severity}</span>
            </h1>
            
            <p>The website you're trying to visit has been blocked by Agent S2 security protection.</p>
            
            <div class="details">
                <strong>Domain:</strong> {domain}<br>
                <strong>Reason:</strong> {reason}<br>
                <div class="url-display"><strong>Full URL:</strong> {url}</div>
            </div>
            
            {vt_section}
            
            <div class="actions">
                <a href="about:home" class="primary">Return to Home</a>
                <a href="#" onclick="history.back(); return false;" class="secondary">Go Back</a>
            </div>
            
            <div class="footer">
                <p>This request was blocked at {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
                <p>If you believe this is a mistake, please check your security settings.</p>
            </div>
        </div>
    </div>
</body>
</html>
        """
    
    async def _log_blocked_request(self, flow: http.HTTPFlow, reason: str, severity: str):
        """Log blocked request to security log"""
        try:
            log_entry = {
                "timestamp": datetime.now().isoformat(),
                "url": flow.request.url,
                "domain": flow.request.pretty_host,
                "method": flow.request.method,
                "reason": reason,
                "severity": severity,
                "user_agent": flow.request.headers.get("User-Agent", "Unknown"),
                "action": "blocked"
            }
            
            # Write to security log
            log_dir = os.path.expanduser("~/.agent-s2/security_logs")
            os.makedirs(log_dir, exist_ok=True)
            
            log_file = os.path.join(log_dir, f"proxy_blocks_{datetime.now().strftime('%Y%m%d')}.json")
            
            # Append to log file
            import json
            with open(log_file, "a") as f:
                f.write(json.dumps(log_entry) + "\n")
                
        except Exception as e:
            logger.error(f"Failed to log blocked request: {e}")
    
    def get_stats(self) -> dict:
        """Get proxy statistics"""
        uptime = (datetime.now() - self.start_time).total_seconds()
        return {
            "requests_checked": self.requests_checked,
            "requests_blocked": self.requests_blocked,
            "block_rate": self.requests_blocked / max(self.requests_checked, 1),
            "uptime_seconds": uptime,
            "virustotal_enabled": self.reputation_service.check_reputation
        }