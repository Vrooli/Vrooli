"""Main stealth manager for coordinating all stealth features"""

import logging
import json
import subprocess
from typing import Dict, Any, Optional, List
from pathlib import Path
import asyncio

from .profile import StealthProfile, StealthConfig
from .fingerprints import FingerprintGenerator
from .session import SessionManager

logger = logging.getLogger(__name__)


class StealthManager:
    """Manages all stealth mode features and coordinates between components"""
    
    def __init__(self, config: Optional[StealthConfig] = None):
        """Initialize stealth manager
        
        Args:
            config: Stealth configuration
        """
        self.config = config or StealthConfig()
        self.profile_manager = StealthProfile(self.config)
        self.fingerprint_generator = FingerprintGenerator()
        self.session_manager = SessionManager(
            storage_path=self.config.session_storage_path,
            encryption_enabled=self.config.session_encryption,
            ttl_days=self.config.session_ttl_days
        )
        
        self._current_profile_id = None
        self._browser_pid = None
        
    async def initialize(self, profile_id: Optional[str] = None) -> Dict[str, Any]:
        """Initialize stealth mode with optional profile
        
        Args:
            profile_id: Optional profile ID to load
            
        Returns:
            Initialization result
        """
        result = {
            "success": False,
            "profile_id": profile_id,
            "features_enabled": {},
            "errors": []
        }
        
        try:
            # Load or generate profile
            if profile_id and self._profile_exists(profile_id):
                logger.info(f"Loading existing profile: {profile_id}")
                profile = self.profile_manager.load_profile(profile_id)
                self._current_profile_id = profile_id
            else:
                logger.info("Generating new stealth profile")
                profile = self.profile_manager.generate_profile()
                if profile_id:
                    self.profile_manager.save_profile(profile_id)
                    self._current_profile_id = profile_id
                    
            # Set fingerprint seed for consistency
            if profile_id:
                self.fingerprint_generator.set_seed(profile_id)
                
            # Apply Firefox preferences
            await self._apply_firefox_preferences()
            
            # Load session data if persistence is enabled
            if self.config.session_data_persistence and profile_id:
                session_data = self.session_manager.load_session_data(profile_id)
                if session_data:
                    await self._restore_session_data(session_data)
                    
            # Update result
            result["success"] = True
            result["features_enabled"] = {
                "webdriver_hiding": self.config.webdriver_hiding,
                "user_agent_rotation": self.config.user_agent_rotation,
                "fingerprint_randomization": self.config.fingerprint_randomization,
                "session_persistence": self.config.session_data_persistence,
            }
            
        except Exception as e:
            logger.error(f"Failed to initialize stealth mode: {e}")
            result["errors"].append(str(e))
            
        return result
        
    def _profile_exists(self, profile_id: str) -> bool:
        """Check if profile exists
        
        Args:
            profile_id: Profile identifier
            
        Returns:
            True if exists
        """
        profile_path = Path(self.config.session_storage_path) / "profiles" / f"{profile_id}.json"
        return profile_path.exists()
        
    async def _apply_firefox_preferences(self) -> None:
        """Apply Firefox preferences for stealth mode"""
        prefs = self.profile_manager.get_firefox_preferences()
        
        # Write additional preferences to user.js
        user_js_path = Path("/home/agents2/.mozilla/firefox/agent-s2/user.js")
        
        if not user_js_path.exists():
            logger.warning("Firefox profile not found, creating directory")
            user_js_path.parent.mkdir(parents=True, exist_ok=True)
            
        # Read existing preferences
        existing_prefs = []
        if user_js_path.exists():
            with open(user_js_path, "r") as f:
                existing_prefs = f.readlines()
                
        # Add stealth preferences
        with open(user_js_path, "a") as f:
            f.write("\n// Agent S2 Stealth Mode Preferences\n")
            
            for pref_name, pref_value in prefs.items():
                if isinstance(pref_value, bool):
                    value_str = "true" if pref_value else "false"
                elif isinstance(pref_value, str):
                    value_str = f'"{pref_value}"'
                else:
                    value_str = str(pref_value)
                    
                f.write(f'user_pref("{pref_name}", {value_str});\n')
                
        logger.info("Applied Firefox stealth preferences")
        
    async def _restore_session_data(self, session_data: Dict[str, Any]) -> None:
        """Restore session data to browser
        
        Args:
            session_data: Session data to restore
        """
        # This would require browser automation to inject cookies, storage, etc.
        # For now, we'll log what would be restored
        logger.info(f"Would restore session data: {len(session_data.get('cookies', []))} cookies")
        
        # TODO: Implement actual restoration using browser automation
        
    async def save_current_session(self, profile_id: Optional[str] = None) -> bool:
        """Save current browser session
        
        Args:
            profile_id: Profile ID to save to (uses current if None)
            
        Returns:
            True if successful
        """
        profile_id = profile_id or self._current_profile_id
        
        if not profile_id:
            logger.error("No profile ID specified for saving session")
            return False
            
        try:
            # Extract session data from browser
            session_data = await self._extract_session_data()
            
            # Save to disk
            self.session_manager.save_session_data(profile_id, session_data)
            
            return True
            
        except Exception as e:
            logger.error(f"Failed to save session: {e}")
            return False
            
    async def _extract_session_data(self) -> Dict[str, Any]:
        """Extract current session data from browser
        
        Returns:
            Session data
        """
        # This would use browser automation to extract cookies, storage, etc.
        # For now, return mock data
        return {
            "cookies": [],
            "localStorage": {},
            "sessionStorage": {},
            "auth_tokens": {}
        }
        
    def get_canvas_injection_script(self) -> str:
        """Get JavaScript to inject for canvas fingerprinting defense
        
        Returns:
            JavaScript code
        """
        noise_data = self.fingerprint_generator.generate_canvas_noise()
        
        return f"""
        // Canvas fingerprinting defense
        (function() {{
            const originalGetContext = HTMLCanvasElement.prototype.getContext;
            HTMLCanvasElement.prototype.getContext = function(type, ...args) {{
                const context = originalGetContext.apply(this, [type, ...args]);
                
                if (type === '2d' && context) {{
                    // Add noise to canvas operations
                    const originalFillText = context.fillText;
                    context.fillText = function(text, x, y, ...args) {{
                        // Add tiny offset
                        x += Math.random() * 0.1 - 0.05;
                        y += Math.random() * 0.1 - 0.05;
                        return originalFillText.apply(this, [text, x, y, ...args]);
                    }};
                }}
                
                return context;
            }};
        }})();
        """
        
    def get_webgl_injection_script(self) -> str:
        """Get JavaScript to inject for WebGL fingerprinting defense
        
        Returns:
            JavaScript code
        """
        webgl_params = self.fingerprint_generator.generate_webgl_noise()
        
        return f"""
        // WebGL fingerprinting defense
        (function() {{
            const params = {json.dumps(webgl_params)};
            
            const originalGetParameter = WebGLRenderingContext.prototype.getParameter;
            WebGLRenderingContext.prototype.getParameter = function(param) {{
                // Override specific parameters
                if (param === this.RENDERER) return params.RENDERER;
                if (param === this.VENDOR) return params.VENDOR;
                if (param === this.VERSION) return params.VERSION;
                
                return originalGetParameter.apply(this, arguments);
            }};
        }})();
        """
        
    def get_audio_injection_script(self) -> str:
        """Get JavaScript to inject for audio fingerprinting defense
        
        Returns:
            JavaScript code
        """
        audio_params = self.fingerprint_generator.generate_audio_noise()
        
        return f"""
        // Audio fingerprinting defense
        (function() {{
            const params = {json.dumps(audio_params)};
            
            const OriginalAudioContext = window.AudioContext || window.webkitAudioContext;
            if (OriginalAudioContext) {{
                window.AudioContext = window.webkitAudioContext = function(...args) {{
                    const context = new OriginalAudioContext(...args);
                    
                    // Override properties
                    Object.defineProperty(context, 'sampleRate', {{
                        get: () => params.sampleRate
                    }});
                    
                    return context;
                }};
            }}
        }})();
        """
        
    def get_timing_injection_script(self) -> str:
        """Get JavaScript to inject for timing attack defense
        
        Returns:
            JavaScript code
        """
        return """
        // Timing attack defense
        (function() {
            const originalNow = performance.now;
            const originalDateNow = Date.now;
            
            // Add small random noise to timing
            performance.now = function() {
                return originalNow.call(this) + (Math.random() * 0.1);
            };
            
            Date.now = function() {
                return originalDateNow.call(this) + Math.floor(Math.random() * 10);
            };
        })();
        """
        
    def get_status(self) -> Dict[str, Any]:
        """Get current stealth mode status
        
        Returns:
            Status information
        """
        return {
            "enabled": self.config.enabled,
            "current_profile": self._current_profile_id,
            "features": {
                "webdriver_hiding": self.config.webdriver_hiding,
                "fingerprint_randomization": self.config.fingerprint_randomization,
                "session_persistence": self.config.session_data_persistence,
                "user_agent_rotation": self.config.user_agent_rotation,
            },
            "profiles": self.session_manager.list_profiles(),
        }
        
    def test_stealth_effectiveness(self, url: str = "https://bot.sannysoft.com/") -> Dict[str, Any]:
        """Test stealth mode effectiveness against detection service
        
        Args:
            url: Bot detection test URL
            
        Returns:
            Test results
        """
        # This would automate browser to visit detection site and check results
        logger.info(f"Testing stealth effectiveness at: {url}")
        
        return {
            "test_url": url,
            "status": "pending",
            "message": "Stealth test functionality to be implemented"
        }
        
    def cleanup(self) -> None:
        """Clean up resources and expired sessions"""
        try:
            # Clean expired sessions
            cleaned = self.session_manager.cleanup_expired()
            logger.info(f"Cleaned up {cleaned} expired sessions")
            
            # Save current session if configured
            if self.config.session_data_persistence and self._current_profile_id:
                asyncio.create_task(self.save_current_session())
                
        except Exception as e:
            logger.error(f"Error during cleanup: {e}")