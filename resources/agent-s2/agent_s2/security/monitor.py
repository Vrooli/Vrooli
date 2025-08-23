"""Security Monitoring and Audit Logging

Provides runtime security monitoring, suspicious activity detection,
and comprehensive audit logging for Agent S2 operations.
"""

import os
import re
import json
import hashlib
import logging
import asyncio
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional, Tuple
from pathlib import Path
from dataclasses import dataclass, asdict
from collections import defaultdict, deque

from ..config import Config, AgentMode

logger = logging.getLogger(__name__)


@dataclass
class SecurityEvent:
    """Security event data structure"""
    timestamp: str
    event_type: str
    severity: str  # low, medium, high, critical
    source: str
    target: str
    action: str
    details: Dict[str, Any]
    user_context: Dict[str, Any]
    risk_score: int  # 0-100


@dataclass
class AuditLogEntry:
    """Audit log entry structure"""
    timestamp: str
    session_id: str
    mode: str
    action: str
    target: str
    result: str
    metadata: Dict[str, Any]
    risk_indicators: List[str]


class SecurityMonitor:
    """Runtime security monitoring and threat detection"""
    
    def __init__(self):
        self.mode = Config.CURRENT_MODE
        self.events = deque(maxlen=1000)  # Keep last 1000 events
        self.threat_patterns = self._load_threat_patterns()
        self.suspicious_activity_counts = defaultdict(int)
        self.last_activity_time = {}
        self.session_id = self._generate_session_id()
        
    def _generate_session_id(self) -> str:
        """Generate unique session ID"""
        timestamp = datetime.utcnow().isoformat()
        content = f"{timestamp}-{os.getpid()}-{Config.get_mode_name()}"
        return hashlib.md5(content.encode()).hexdigest()[:12]
    
    def _load_threat_patterns(self) -> Dict[str, List[str]]:
        """Load threat detection patterns"""
        return {
            "suspicious_paths": [
                r"/etc/passwd",
                r"/etc/shadow", 
                r"/root/",
                r"\.ssh/",
                r"\.gnupg/",
                r"/var/log/auth\.log",
                r"/var/log/secure",
                r"/proc/[0-9]+/mem",
                r"\.key$",
                r"\.pem$",
                r"password",
                r"secret",
                r"token",
                r"credential"
            ],
            "suspicious_commands": [
                r"sudo\s+",
                r"su\s+",
                r"passwd\s+",
                r"usermod\s+",
                r"chmod\s+[0-9]*7[0-9]*",
                r"nc\s+.*-l",  # netcat listener
                r"bash\s+-i",  # interactive bash
                r"sh\s+-i",
                r"/bin/sh",
                r"eval\s+",
                r"exec\s+",
                r"wget\s+.*\|.*sh",
                r"curl\s+.*\|.*sh"
            ],
            "privilege_escalation": [
                r"setuid",
                r"setgid", 
                r"sudo",
                r"pkexec",
                r"doas"
            ],
            "network_anomalies": [
                r"0\.0\.0\.0",
                r"127\.0\.0\.1",
                r"localhost",
                r"192\.168\.",
                r"10\.",
                r"172\.(1[6-9]|2[0-9]|3[0-1])\."
            ]
        }
    
    def log_action(self, action: str, target: str, details: Optional[Dict[str, Any]] = None) -> None:
        """Log and analyze user action for security threats"""
        try:
            event = SecurityEvent(
                timestamp=datetime.utcnow().isoformat(),
                event_type="user_action",
                severity="info",
                source="agent_s2",
                target=target,
                action=action,
                details=details or {},
                user_context={
                    "mode": self.mode.value,
                    "session_id": self.session_id,
                    "pid": os.getpid()
                },
                risk_score=0
            )
            
            # Analyze for threats
            self._analyze_event(event)
            
            # Store event
            self.events.append(event)
            
            # Update activity tracking
            self.last_activity_time[action] = datetime.utcnow()
            
            # Log to audit system if enabled
            if Config.HOST_AUDIT_LOGGING:
                AuditLogger.log_action(action, target, details, self.mode)
                
        except Exception as e:
            logger.error(f"Failed to log security action: {e}")
    
    def _analyze_event(self, event: SecurityEvent) -> None:
        """Analyze event for security threats"""
        risk_score = 0
        threats = []
        
        # Check suspicious paths
        for pattern in self.threat_patterns["suspicious_paths"]:
            if re.search(pattern, event.target, re.IGNORECASE):
                threats.append(f"suspicious_path:{pattern}")
                risk_score += 20
        
        # Check suspicious commands  
        combined_text = f"{event.action} {event.target} {json.dumps(event.details)}"
        for pattern in self.threat_patterns["suspicious_commands"]:
            if re.search(pattern, combined_text, re.IGNORECASE):
                threats.append(f"suspicious_command:{pattern}")
                risk_score += 30
        
        # Check privilege escalation attempts
        for pattern in self.threat_patterns["privilege_escalation"]:
            if re.search(pattern, combined_text, re.IGNORECASE):
                threats.append(f"privilege_escalation:{pattern}")
                risk_score += 40
        
        # Check for rapid repeated actions (potential automation attack)
        if self._detect_rapid_actions(event.action):
            threats.append("rapid_repeated_actions")
            risk_score += 25
        
        # Mode-specific checks
        if self.mode == AgentMode.HOST:
            risk_score += self._analyze_host_mode_risks(event)
        
        # Update event with risk assessment
        event.risk_score = min(risk_score, 100)
        if risk_score > 70:
            event.severity = "critical"
        elif risk_score > 40:
            event.severity = "high"
        elif risk_score > 20:
            event.severity = "medium"
        else:
            event.severity = "low"
        
        # Add threat indicators to details
        if threats:
            event.details["threat_indicators"] = threats
        
        # Alert on high-risk events
        if event.risk_score > 40:
            self._alert_suspicious_activity(event)
    
    def _analyze_host_mode_risks(self, event: SecurityEvent) -> int:
        """Analyze additional risks specific to host mode"""
        risk_score = 0
        
        # Host mode inherently higher risk
        risk_score += 10
        
        # Check if accessing forbidden paths
        forbidden_paths = Config.HOST_FORBIDDEN_PATHS
        for forbidden in forbidden_paths:
            if forbidden in event.target:
                risk_score += 35
                event.details["forbidden_path_access"] = forbidden
                break
        
        # Check for attempts to access host user directories
        host_user_patterns = [
            r"/home/[^/]+/\.ssh",
            r"/home/[^/]+/\.gnupg", 
            r"/home/[^/]+/\.config/.*/password",
            r"/root/"
        ]
        
        for pattern in host_user_patterns:
            if re.search(pattern, event.target):
                risk_score += 30
                event.details["host_user_access_attempt"] = pattern
                break
        
        return risk_score
    
    def _detect_rapid_actions(self, action: str) -> bool:
        """Detect rapid repeated actions that might indicate automated attacks"""
        now = datetime.utcnow()
        
        # Count actions in last 60 seconds
        recent_events = [
            e for e in self.events 
            if e.action == action and 
            datetime.fromisoformat(e.timestamp) > now - timedelta(seconds=60)
        ]
        
        # Flag if more than 20 of same action in 60 seconds
        if len(recent_events) > 20:
            return True
        
        # Flag if more than 5 of same action in 10 seconds
        very_recent = [
            e for e in recent_events
            if datetime.fromisoformat(e.timestamp) > now - timedelta(seconds=10)
        ]
        
        return len(very_recent) > 5
    
    def _alert_suspicious_activity(self, event: SecurityEvent) -> None:
        """Alert on suspicious activity"""
        alert_msg = (
            f"SECURITY ALERT: {event.severity.upper()} risk activity detected\n"
            f"Action: {event.action}\n"
            f"Target: {event.target}\n"
            f"Risk Score: {event.risk_score}/100\n"
            f"Mode: {event.user_context['mode']}\n"
            f"Session: {event.user_context['session_id']}\n"
            f"Threats: {event.details.get('threat_indicators', [])}\n"
            f"Timestamp: {event.timestamp}"
        )
        
        logger.warning(alert_msg)
        
        # In production, could send to SIEM, security team, etc.
        self._notify_security_team(event)
    
    def _notify_security_team(self, event: SecurityEvent) -> None:
        """Notify security team of high-risk events"""
        try:
            # Log security event with structured data
            security_log_entry = {
                "timestamp": event.timestamp.isoformat(),
                "event_type": event.event_type,
                "severity": event.severity,
                "source": event.source,
                "details": event.details
            }
            
            # Log to security-specific logger
            security_logger = logging.getLogger("agent_s2.security")
            security_logger.warning(f"Security Event: {event.event_type} - {event.details}", extra=security_log_entry)
            
            # Send email notification if configured
            self._send_email_notification(event)
            
            # Send webhook notification if configured
            self._send_webhook_notification(event)
            
            # Write to security incident file
            self._write_security_incident(event)
            
        except Exception as e:
            logger.error(f"Failed to notify security team: {e}")
    
    def _send_email_notification(self, event: SecurityEvent) -> None:
        """Send email notification for security events"""
        try:
            import smtplib
            from email.mime.text import MIMEText
            from email.mime.multipart import MIMEMultipart
            import os
            
            # Check if email is configured
            smtp_server = os.getenv("AGENT_S2_SMTP_SERVER")
            smtp_port = int(os.getenv("AGENT_S2_SMTP_PORT", "587"))
            smtp_user = os.getenv("AGENT_S2_SMTP_USER")
            smtp_password = os.getenv("AGENT_S2_SMTP_PASSWORD")
            security_email = os.getenv("AGENT_S2_SECURITY_EMAIL")
            
            if not all([smtp_server, smtp_user, smtp_password, security_email]):
                logger.debug("Email notification not configured, skipping")
                return
            
            # Create email message
            msg = MIMEMultipart()
            msg["From"] = smtp_user
            msg["To"] = security_email
            msg["Subject"] = f"Agent S2 Security Alert: {event.event_type}"
            
            # Email body
            body = f"""
            Security Event Detected
            
            Time: {event.timestamp.isoformat()}
            Event Type: {event.event_type}
            Severity: {event.severity}
            Source: {event.source}
            
            Details:
            {event.details}
            
            This is an automated notification from Agent S2 Security Monitor.
            """
            
            msg.attach(MIMEText(body, "plain"))
            
            # Send email
            with smtplib.SMTP(smtp_server, smtp_port) as server:
                server.starttls()
                server.login(smtp_user, smtp_password)
                server.send_message(msg)
                
            logger.info(f"Security email notification sent for event: {event.event_type}")
            
        except Exception as e:
            logger.error(f"Failed to send email notification: {e}")
    
    def _send_webhook_notification(self, event: SecurityEvent) -> None:
        """Send webhook notification for security events"""
        try:
            import requests
            import json
            import os
            
            webhook_url = os.getenv("AGENT_S2_SECURITY_WEBHOOK_URL")
            if not webhook_url:
                logger.debug("Webhook notification not configured, skipping")
                return
            
            # Prepare webhook payload
            payload = {
                "timestamp": event.timestamp.isoformat(),
                "event_type": event.event_type,
                "severity": event.severity,
                "source": event.source,
                "details": event.details,
                "service": "agent-s2",
                "host": os.getenv("HOSTNAME", "unknown")
            }
            
            # Send webhook
            response = requests.post(
                webhook_url,
                json=payload,
                headers={"Content-Type": "application/json"},
                timeout=10
            )
            
            if response.status_code == 200:
                logger.info(f"Security webhook notification sent for event: {event.event_type}")
            else:
                logger.warning(f"Webhook notification failed with status {response.status_code}")
                
        except Exception as e:
            logger.error(f"Failed to send webhook notification: {e}")
    
    def _write_security_incident(self, event: SecurityEvent) -> None:
        """Write security incident to file for audit trail"""
        try:
            import json
            import os
            from pathlib import Path
            
            # Create security incidents directory
            incidents_dir = Path("/tmp/agent-s2-security-incidents")
            incidents_dir.mkdir(exist_ok=True)
            
            # Create incident file with timestamp
            incident_file = incidents_dir / f"incident_{event.timestamp.strftime('%Y%m%d_%H%M%S')}.json"
            
            # Prepare incident data
            incident_data = {
                "timestamp": event.timestamp.isoformat(),
                "event_type": event.event_type,
                "severity": event.severity,
                "source": event.source,
                "details": event.details,
                "session_id": getattr(self, 'session_id', 'unknown'),
                "user_agent": getattr(self, 'user_agent', 'unknown'),
                "host": os.getenv("HOSTNAME", "unknown")
            }
            
            # Write incident to file
            with open(incident_file, 'w') as f:
                json.dump(incident_data, f, indent=2)
                
            logger.info(f"Security incident written to: {incident_file}")
            
        except Exception as e:
            logger.error(f"Failed to write security incident: {e}")
    
    def get_security_summary(self) -> Dict[str, Any]:
        """Get security summary for current session"""
        now = datetime.utcnow()
        
        # Events in last hour
        recent_events = [
            e for e in self.events
            if datetime.fromisoformat(e.timestamp) > now - timedelta(hours=1)
        ]
        
        # Count by severity
        severity_counts = defaultdict(int)
        for event in recent_events:
            severity_counts[event.severity] += 1
        
        # High-risk events
        high_risk_events = [
            e for e in recent_events 
            if e.risk_score > 40
        ]
        
        return {
            "session_id": self.session_id,
            "mode": self.mode.value,
            "total_events": len(self.events),
            "recent_events_1h": len(recent_events),
            "severity_distribution": dict(severity_counts),
            "high_risk_events": len(high_risk_events),
            "latest_high_risk": [
                {
                    "timestamp": e.timestamp,
                    "action": e.action,
                    "target": e.target,
                    "risk_score": e.risk_score,
                    "threats": e.details.get("threat_indicators", [])
                }
                for e in high_risk_events[-5:]  # Last 5 high-risk events
            ],
            "uptime": str(now - datetime.fromisoformat(recent_events[0].timestamp) if recent_events else timedelta(0))
        }
    
    def export_events(self, start_time: Optional[datetime] = None, end_time: Optional[datetime] = None) -> List[Dict[str, Any]]:
        """Export security events for analysis"""
        events = list(self.events)
        
        if start_time:
            events = [e for e in events if datetime.fromisoformat(e.timestamp) >= start_time]
        
        if end_time:
            events = [e for e in events if datetime.fromisoformat(e.timestamp) <= end_time]
        
        return [asdict(event) for event in events]


class AuditLogger:
    """Comprehensive audit logging system"""
    
    @staticmethod
    def log_action(action: str, target: str, details: Optional[Dict[str, Any]], mode: AgentMode) -> None:
        """Log action to audit trail"""
        if not Config.HOST_AUDIT_LOGGING:
            return
        
        try:
            audit_entry = AuditLogEntry(
                timestamp=datetime.utcnow().isoformat(),
                session_id=SecurityMonitor()._generate_session_id(),
                mode=mode.value,
                action=action,
                target=target,
                result="attempted",  # Could be updated based on actual result
                metadata=details or {},
                risk_indicators=[]
            )
            
            # Write to audit log file
            AuditLogger._write_audit_log(audit_entry)
            
        except Exception as e:
            logger.error(f"Failed to write audit log: {e}")
    
    @staticmethod
    def _write_audit_log(entry: AuditLogEntry) -> None:
        """Write audit entry to log file"""
        try:
            # Create audit log directory
            audit_dir = Path("/var/log/agent-s2-audit")
            audit_dir.mkdir(exist_ok=True, parents=True)
            
            # Daily log files
            log_file = audit_dir / f"audit-{datetime.utcnow().strftime('%Y-%m-%d')}.log"
            
            # JSON format for easy parsing
            log_entry = json.dumps(asdict(entry))
            
            with open(log_file, "a") as f:
                f.write(log_entry + "\n")
                
        except Exception as e:
            # Fallback to container logs
            logger.info(f"AUDIT: {json.dumps(asdict(entry))}")
    
    @staticmethod
    def get_audit_summary(days: int = 7) -> Dict[str, Any]:
        """Get audit log summary for specified days"""
        try:
            audit_dir = Path("/var/log/agent-s2-audit")
            if not audit_dir.exists():
                return {"error": "Audit directory not found"}
            
            summary = {
                "total_entries": 0,
                "actions_by_type": defaultdict(int),
                "modes_used": defaultdict(int),
                "risk_indicators": defaultdict(int),
                "date_range": {
                    "start": (datetime.utcnow() - timedelta(days=days)).isoformat(),
                    "end": datetime.utcnow().isoformat()
                }
            }
            
            # Process recent log files
            for i in range(days):
                date = datetime.utcnow() - timedelta(days=i)
                log_file = audit_dir / f"audit-{date.strftime('%Y-%m-%d')}.log"
                
                if log_file.exists():
                    with open(log_file, "r") as f:
                        for line in f:
                            try:
                                entry = json.loads(line.strip())
                                summary["total_entries"] += 1
                                summary["actions_by_type"][entry["action"]] += 1
                                summary["modes_used"][entry["mode"]] += 1
                                
                                for indicator in entry.get("risk_indicators", []):
                                    summary["risk_indicators"][indicator] += 1
                                    
                            except json.JSONDecodeError:
                                continue
            
            # Convert defaultdicts to regular dicts
            summary["actions_by_type"] = dict(summary["actions_by_type"])
            summary["modes_used"] = dict(summary["modes_used"])
            summary["risk_indicators"] = dict(summary["risk_indicators"])
            
            return summary
            
        except Exception as e:
            logger.error(f"Failed to get audit summary: {e}")
            return {"error": str(e)}


# Global security monitor instance
_security_monitor = None

def get_security_monitor() -> SecurityMonitor:
    """Get global security monitor instance"""
    global _security_monitor
    if _security_monitor is None:
        _security_monitor = SecurityMonitor()
    return _security_monitor