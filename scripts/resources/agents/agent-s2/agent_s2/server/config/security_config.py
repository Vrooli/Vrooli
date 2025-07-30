"""Security configuration management for Agent S2"""

import os
import json
import logging
from typing import Dict, Any, Optional
from pathlib import Path

logger = logging.getLogger(__name__)


class SecurityConfig:
    """Manages security configuration including API keys and profiles"""
    
    DEFAULT_CONFIG = {
        "security": {
            "virustotal_api_key": None,
            "default_profile": "moderate",
            "cache_reputation_results": True,
            "reputation_cache_ttl_hours": 24,
            "enable_browser_monitoring": True,
            "block_unsafe_navigation": True,
            "incident_log_retention_days": 30,
            "profiles": {
                "strict": {
                    "allowed_domains": [
                        "wikipedia.org", "*.wikipedia.org",
                        "github.com", "*.github.com",
                        "stackoverflow.com", "docs.python.org",
                        "developer.mozilla.org", "google.com"
                    ],
                    "blocked_domains": ["*"],
                    "require_https": True,
                    "check_reputation": True
                },
                "moderate": {
                    "blocked_domains": [
                        "*.tk", "*.ml", "*.ga", "*.cf",
                        "bit.ly", "tinyurl.com", "goo.gl",
                        "*.click", "*.download", "*.loan"
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
        }
    }
    
    def __init__(self, config_path: Optional[str] = None):
        self.config_path = config_path or self._get_default_config_path()
        self.config = self._load_config()
        self._apply_environment_overrides()
        
    def _get_default_config_path(self) -> str:
        """Get default configuration file path"""
        config_dir = Path.home() / ".agent-s2" / "config"
        config_dir.mkdir(parents=True, exist_ok=True)
        return str(config_dir / "security.json")
        
    def _load_config(self) -> Dict[str, Any]:
        """Load configuration from file"""
        if os.path.exists(self.config_path):
            try:
                with open(self.config_path, 'r') as f:
                    loaded_config = json.load(f)
                    # Merge with defaults
                    return self._merge_configs(self.DEFAULT_CONFIG, loaded_config)
            except Exception as e:
                logger.error(f"Failed to load config from {self.config_path}: {e}")
                logger.info("Using default configuration")
                
        # Create default config file
        self._save_config(self.DEFAULT_CONFIG)
        return self.DEFAULT_CONFIG.copy()
        
    def _merge_configs(self, default: Dict, loaded: Dict) -> Dict:
        """Recursively merge loaded config with defaults"""
        result = default.copy()
        
        for key, value in loaded.items():
            if key in result and isinstance(result[key], dict) and isinstance(value, dict):
                result[key] = self._merge_configs(result[key], value)
            else:
                result[key] = value
                
        return result
        
    def _save_config(self, config: Dict[str, Any]):
        """Save configuration to file"""
        try:
            os.makedirs(os.path.dirname(self.config_path), exist_ok=True)
            with open(self.config_path, 'w') as f:
                json.dump(config, f, indent=2)
            logger.info(f"Configuration saved to: {self.config_path}")
        except Exception as e:
            logger.error(f"Failed to save config: {e}")
            
    def _apply_environment_overrides(self):
        """Apply environment variable overrides"""
        env_mappings = {
            "AGENT_S2_VIRUSTOTAL_API_KEY": ("security", "virustotal_api_key"),
            "AGENT_S2_SECURITY_PROFILE": ("security", "default_profile"),
            "AGENT_S2_REPUTATION_CACHE_TTL": ("security", "reputation_cache_ttl_hours"),
            "AGENT_S2_ENABLE_BROWSER_MONITORING": ("security", "enable_browser_monitoring"),
            "AGENT_S2_BLOCK_UNSAFE_NAVIGATION": ("security", "block_unsafe_navigation")
        }
        
        for env_var, config_path in env_mappings.items():
            value = os.environ.get(env_var)
            if value is not None:
                # Navigate to the correct config location
                current = self.config
                for key in config_path[:-1]:
                    current = current.setdefault(key, {})
                    
                # Convert value types as needed
                if env_var.endswith("_TTL"):
                    value = int(value)
                elif env_var.endswith("_ENABLE") or env_var.endswith("_BLOCK"):
                    value = value.lower() in ["true", "yes", "1"]
                    
                current[config_path[-1]] = value
                logger.debug(f"Applied environment override: {env_var}")
                
        # Handle domain lists from environment
        if os.environ.get("AGENT_S2_DEFAULT_ALLOWED_DOMAINS"):
            domains = os.environ["AGENT_S2_DEFAULT_ALLOWED_DOMAINS"].split(",")
            profile = self.config["security"]["default_profile"]
            if profile in self.config["security"]["profiles"]:
                self.config["security"]["profiles"][profile]["allowed_domains"] = domains
                
        if os.environ.get("AGENT_S2_DEFAULT_BLOCKED_DOMAINS"):
            domains = os.environ["AGENT_S2_DEFAULT_BLOCKED_DOMAINS"].split(",")
            profile = self.config["security"]["default_profile"]
            if profile in self.config["security"]["profiles"]:
                self.config["security"]["profiles"][profile]["blocked_domains"] = domains
                
    def get_virustotal_api_key(self) -> Optional[str]:
        """Get VirusTotal API key"""
        return self.config["security"].get("virustotal_api_key")
        
    def get_security_profile(self, profile_name: Optional[str] = None) -> Dict[str, Any]:
        """Get security profile configuration"""
        profile_name = profile_name or self.config["security"]["default_profile"]
        profiles = self.config["security"]["profiles"]
        
        if profile_name not in profiles:
            logger.warning(f"Unknown profile '{profile_name}', using 'moderate'")
            profile_name = "moderate"
            
        return profiles[profile_name].copy()
        
    def get_reputation_config(self) -> Dict[str, Any]:
        """Get reputation checking configuration"""
        security = self.config["security"]
        return {
            "api_key": security.get("virustotal_api_key"),
            "cache_enabled": security.get("cache_reputation_results", True),
            "cache_ttl_hours": security.get("reputation_cache_ttl_hours", 24),
            "check_reputation": True  # Can be overridden by profile
        }
        
    def get_monitoring_config(self) -> Dict[str, Any]:
        """Get browser monitoring configuration"""
        security = self.config["security"]
        return {
            "enabled": security.get("enable_browser_monitoring", True),
            "block_unsafe": security.get("block_unsafe_navigation", True),
            "log_retention_days": security.get("incident_log_retention_days", 30)
        }
        
    def update_api_key(self, service: str, api_key: str):
        """Update an API key"""
        if service == "virustotal":
            self.config["security"]["virustotal_api_key"] = api_key
            self._save_config(self.config)
            logger.info("VirusTotal API key updated")
        else:
            logger.error(f"Unknown service: {service}")
            
    def add_allowed_domain(self, domain: str, profile: Optional[str] = None):
        """Add domain to allowed list"""
        profile = profile or self.config["security"]["default_profile"]
        if profile in self.config["security"]["profiles"]:
            allowed = self.config["security"]["profiles"][profile].setdefault("allowed_domains", [])
            if domain not in allowed:
                allowed.append(domain)
                self._save_config(self.config)
                logger.info(f"Added {domain} to allowed domains for profile {profile}")
                
    def add_blocked_domain(self, domain: str, profile: Optional[str] = None):
        """Add domain to blocked list"""
        profile = profile or self.config["security"]["default_profile"]
        if profile in self.config["security"]["profiles"]:
            blocked = self.config["security"]["profiles"][profile].setdefault("blocked_domains", [])
            if domain not in blocked:
                blocked.append(domain)
                self._save_config(self.config)
                logger.info(f"Added {domain} to blocked domains for profile {profile}")
                
    def get_all_config(self) -> Dict[str, Any]:
        """Get complete configuration"""
        return self.config.copy()
        
    def reset_to_defaults(self):
        """Reset configuration to defaults"""
        self.config = self.DEFAULT_CONFIG.copy()
        self._save_config(self.config)
        logger.info("Configuration reset to defaults")