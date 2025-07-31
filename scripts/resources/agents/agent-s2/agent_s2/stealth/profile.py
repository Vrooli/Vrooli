"""Stealth profile management for Agent S2"""

import json
import random
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, asdict
from pathlib import Path

from ..config import Config


@dataclass
class StealthConfig:
    """Configuration for stealth mode features"""
    enabled: bool = True
    fingerprint_randomization: bool = True
    webdriver_hiding: bool = True
    user_agent_rotation: bool = True
    timezone_spoofing: bool = True
    webgl_noise: bool = True
    audio_context_noise: bool = True
    canvas_fingerprint_defense: bool = True
    webrtc_leak_prevention: bool = True
    battery_api_spoofing: bool = True
    hardware_concurrency_spoofing: bool = True
    
    # Session persistence
    session_data_persistence: bool = True
    session_state_persistence: bool = False
    session_storage_path: str = Config.SESSION_STORAGE_PATH if 'Config' in globals() else "/home/agents2/.agent-s2/sessions"
    session_encryption: bool = True
    session_ttl_days: int = 30
    
    # Profile settings
    profile_type: str = "residential"  # residential, mobile, datacenter
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return asdict(self)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "StealthConfig":
        """Create from dictionary"""
        return cls(**data)


class StealthProfile:
    """Manages browser stealth profiles and configurations"""
    
    # User agent pools by profile type
    USER_AGENTS = {
        "residential": [
            # Windows Chrome
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
            # Windows Firefox
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
            # macOS Chrome
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
            # macOS Safari
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15",
            # Linux Chrome
            "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
        ],
        "mobile": [
            # Android Chrome
            "Mozilla/5.0 (Linux; Android 13; SM-S908B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
            "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
            # iOS Safari
            "Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
            "Mozilla/5.0 (iPad; CPU OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
        ],
        "datacenter": [
            # Generic Chrome
            "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            # Generic Firefox
            "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/120.0",
        ]
    }
    
    # Viewport sizes by profile type
    VIEWPORTS = {
        "residential": [
            {"width": 1920, "height": 1080},
            {"width": 1366, "height": 768},
            {"width": 1536, "height": 864},
            {"width": 1440, "height": 900},
            {"width": 1280, "height": 720},
        ],
        "mobile": [
            {"width": 390, "height": 844},   # iPhone 14
            {"width": 412, "height": 915},   # Pixel 7
            {"width": 414, "height": 896},   # iPhone 11
            {"width": 375, "height": 812},   # iPhone X
        ],
        "datacenter": [
            {"width": 1920, "height": 1080},
            {"width": 1024, "height": 768},
        ]
    }
    
    # Timezones
    TIMEZONES = [
        "America/New_York",
        "America/Chicago", 
        "America/Denver",
        "America/Los_Angeles",
        "America/Phoenix",
        "America/Toronto",
        "Europe/London",
        "Europe/Paris",
        "Europe/Berlin",
        "Asia/Tokyo",
        "Asia/Shanghai",
        "Australia/Sydney",
    ]
    
    # Languages/Locales
    LOCALES = [
        "en-US",
        "en-GB",
        "en-CA",
        "es-ES",
        "es-MX",
        "fr-FR",
        "de-DE",
        "it-IT",
        "pt-BR",
        "ja-JP",
        "zh-CN",
        "ko-KR",
    ]
    
    def __init__(self, config: Optional[StealthConfig] = None):
        """Initialize stealth profile
        
        Args:
            config: Stealth configuration
        """
        self.config = config or StealthConfig()
        self._current_profile = None
        
    def generate_profile(self, profile_type: Optional[str] = None) -> Dict[str, Any]:
        """Generate a new stealth profile
        
        Args:
            profile_type: Type of profile (residential, mobile, datacenter)
            
        Returns:
            Complete profile configuration
        """
        profile_type = profile_type or self.config.profile_type
        
        profile = {
            "type": profile_type,
            "user_agent": random.choice(self.USER_AGENTS[profile_type]),
            "viewport": random.choice(self.VIEWPORTS[profile_type]),
            "timezone": random.choice(self.TIMEZONES),
            "locale": random.choice(self.LOCALES),
            "webgl_vendor": self._random_webgl_vendor(),
            "webgl_renderer": self._random_webgl_renderer(),
            "hardware_concurrency": self._random_hardware_concurrency(profile_type),
            "device_memory": self._random_device_memory(profile_type),
            "max_touch_points": self._random_touch_points(profile_type),
            "color_depth": random.choice([24, 32]),
            "pixel_ratio": self._random_pixel_ratio(profile_type),
            "plugins": self._random_plugins(profile_type),
            "fonts": self._random_fonts(),
        }
        
        self._current_profile = profile
        return profile
        
    def _random_webgl_vendor(self) -> str:
        """Generate random WebGL vendor"""
        vendors = [
            "Intel Inc.",
            "NVIDIA Corporation",
            "AMD",
            "Apple Inc.",
            "Google Inc.",
            "Mesa/X.org",
        ]
        return random.choice(vendors)
        
    def _random_webgl_renderer(self) -> str:
        """Generate random WebGL renderer"""
        renderers = [
            "ANGLE (Intel(R) UHD Graphics Direct3D11 vs_5_0 ps_5_0)",
            "ANGLE (NVIDIA GeForce GTX 1660 Ti Direct3D11 vs_5_0 ps_5_0)",
            "ANGLE (AMD Radeon RX 580 Direct3D11 vs_5_0 ps_5_0)",
            "ANGLE (Intel(R) Iris(R) Xe Graphics Direct3D11 vs_5_0 ps_5_0)",
            "Apple GPU",
            "Mesa DRI Intel(R) HD Graphics",
        ]
        return random.choice(renderers)
        
    def _random_hardware_concurrency(self, profile_type: str) -> int:
        """Generate random hardware concurrency (CPU cores)"""
        if profile_type == "mobile":
            return random.choice([4, 6, 8])
        elif profile_type == "datacenter":
            return random.choice([16, 32, 64])
        else:  # residential
            return random.choice([4, 6, 8, 12, 16])
            
    def _random_device_memory(self, profile_type: str) -> int:
        """Generate random device memory (GB)"""
        if profile_type == "mobile":
            return random.choice([4, 6, 8])
        elif profile_type == "datacenter":
            return random.choice([32, 64, 128])
        else:  # residential
            return random.choice([8, 16, 32])
            
    def _random_touch_points(self, profile_type: str) -> int:
        """Generate random max touch points"""
        if profile_type == "mobile":
            return random.choice([5, 10])
        else:
            return 0
            
    def _random_pixel_ratio(self, profile_type: str) -> float:
        """Generate random device pixel ratio"""
        if profile_type == "mobile":
            return random.choice([2.0, 2.5, 3.0])
        else:
            return random.choice([1.0, 1.25, 1.5, 2.0])
            
    def _random_plugins(self, profile_type: str) -> List[str]:
        """Generate random plugin list"""
        if profile_type == "mobile":
            return []
        
        plugins = [
            "Chrome PDF Plugin",
            "Chrome PDF Viewer",
            "Native Client",
        ]
        # Randomly include some plugins
        return random.sample(plugins, k=random.randint(0, len(plugins)))
        
    def _random_fonts(self) -> List[str]:
        """Generate random font list"""
        base_fonts = [
            "Arial", "Arial Black", "Comic Sans MS", "Courier New",
            "Georgia", "Impact", "Times New Roman", "Trebuchet MS",
            "Verdana", "Helvetica", "Tahoma", "Segoe UI"
        ]
        # Add some random system fonts
        system_fonts = [
            "Calibri", "Cambria", "Consolas", "Lucida Console",
            "MS Sans Serif", "MS Serif", "Palatino Linotype"
        ]
        
        # Randomly select subset
        selected_fonts = base_fonts.copy()
        selected_fonts.extend(random.sample(system_fonts, k=random.randint(0, len(system_fonts))))
        
        return selected_fonts
        
    def get_firefox_preferences(self) -> Dict[str, Any]:
        """Get Firefox preferences for current profile
        
        Returns:
            Dictionary of Firefox preferences
        """
        if not self._current_profile:
            self.generate_profile()
            
        prefs = {}
        
        # Basic stealth preferences
        if self.config.webdriver_hiding:
            # Disable webdriver detection
            prefs["dom.webdriver.enabled"] = False
            prefs["useAutomationExtension"] = False
            
        if self.config.user_agent_rotation:
            prefs["general.useragent.override"] = self._current_profile["user_agent"]
            
        if self.config.timezone_spoofing:
            # Firefox doesn't support direct timezone override via prefs
            # This would need to be done at system level or via extension
            pass
            
        if self.config.webrtc_leak_prevention:
            prefs["media.peerconnection.enabled"] = False
            prefs["media.navigator.enabled"] = False
            
        if self.config.battery_api_spoofing:
            prefs["dom.battery.enabled"] = False
            
        # Canvas fingerprinting protection
        if self.config.canvas_fingerprint_defense:
            prefs["privacy.resistFingerprinting"] = True
            prefs["privacy.resistFingerprinting.autoDeclineNoUserInputCanvasPrompts"] = False
            
        # Disable various tracking features
        prefs["privacy.trackingprotection.enabled"] = True
        prefs["network.cookie.cookieBehavior"] = 1
        prefs["network.http.referer.XOriginPolicy"] = 2
        prefs["network.http.referer.XOriginTrimmingPolicy"] = 2
        
        # Performance settings
        prefs["dom.ipc.plugins.flash.subprocess.crashreporter.enabled"] = False
        prefs["browser.safebrowsing.downloads.remote.enabled"] = False
        
        return prefs
        
    def save_profile(self, profile_id: str, path: Optional[Path] = None) -> None:
        """Save current profile to disk
        
        Args:
            profile_id: Unique profile identifier
            path: Optional custom path
        """
        if not self._current_profile:
            raise ValueError("No profile generated yet")
            
        save_path = path or Path(self.config.session_storage_path) / "profiles" / f"{profile_id}.json"
        save_path.parent.mkdir(parents=True, exist_ok=True)
        
        with open(save_path, "w") as f:
            json.dump(self._current_profile, f, indent=2)
            
    def load_profile(self, profile_id: str, path: Optional[Path] = None) -> Dict[str, Any]:
        """Load profile from disk
        
        Args:
            profile_id: Profile identifier
            path: Optional custom path
            
        Returns:
            Profile configuration
        """
        load_path = path or Path(self.config.session_storage_path) / "profiles" / f"{profile_id}.json"
        
        if not load_path.exists():
            raise FileNotFoundError(f"Profile not found: {profile_id}")
            
        with open(load_path, "r") as f:
            self._current_profile = json.load(f)
            
        return self._current_profile