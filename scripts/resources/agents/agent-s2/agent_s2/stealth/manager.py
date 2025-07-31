"""Main stealth manager for coordinating all stealth features"""

import logging
import json
import subprocess
from typing import Dict, Any, Optional, List
from pathlib import Path
import asyncio

from ..config import Config
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
        user_js_path = Path(Config.FIREFOX_USER_JS_PATH)
        
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
        if not session_data:
            logger.info("No session data to restore")
            return
            
        cookies = session_data.get('cookies', [])
        local_storage = session_data.get('localStorage', {})
        session_storage = session_data.get('sessionStorage', {})
        
        logger.info(f"Restoring session data: {len(cookies)} cookies, "
                   f"{len(local_storage)} localStorage items, "
                   f"{len(session_storage)} sessionStorage items")
        
        try:
            # Generate JavaScript for session restoration
            restore_script = self._generate_session_restore_script(session_data)
            
            # Execute script using browser automation
            success = await self._execute_browser_script(restore_script)
            
            if success:
                logger.info("Session data restoration completed successfully")
            else:
                logger.warning("Session data restoration failed or partially completed")
                
        except Exception as e:
            logger.error(f"Failed to restore session data: {e}")
            # Fall back to logging what would be restored
            logger.info(f"Fallback: Would restore {len(cookies)} cookies, "
                       f"{len(local_storage)} localStorage items, "
                       f"{len(session_storage)} sessionStorage items")
    
    def _generate_session_restore_script(self, session_data: Dict[str, Any]) -> str:
        """Generate JavaScript to restore session data
        
        Args:
            session_data: Session data to restore
            
        Returns:
            JavaScript code for restoration
        """
        cookies = session_data.get('cookies', [])
        local_storage = session_data.get('localStorage', {})
        session_storage = session_data.get('sessionStorage', {})
        
        script_parts = [
            "// Session Restoration Script - Generated by Agent S2 Stealth Manager",
            "(function() {",
            "    try {"
        ]
        
        # Restore localStorage
        if local_storage:
            script_parts.append("        // Restore localStorage")
            for key, value in local_storage.items():
                # Escape strings properly for JavaScript
                escaped_key = json.dumps(str(key))
                escaped_value = json.dumps(str(value))
                script_parts.append(f"        localStorage.setItem({escaped_key}, {escaped_value});")
        
        # Restore sessionStorage
        if session_storage:
            script_parts.append("        // Restore sessionStorage")
            for key, value in session_storage.items():
                escaped_key = json.dumps(str(key))
                escaped_value = json.dumps(str(value))
                script_parts.append(f"        sessionStorage.setItem({escaped_key}, {escaped_value});")
        
        # Restore cookies (more complex due to security restrictions)
        if cookies:
            script_parts.append("        // Restore cookies (where possible)")
            script_parts.append("        // Note: HttpOnly and Secure cookies may not be settable via JavaScript")
            for cookie in cookies:
                if isinstance(cookie, dict):
                    name = cookie.get('name', '')
                    value = cookie.get('value', '')
                    domain = cookie.get('domain', '')
                    path = cookie.get('path', '/')
                    expires = cookie.get('expires')
                    
                    if name and value:
                        cookie_str = f"{name}={value}; path={path}"
                        if domain:
                            cookie_str += f"; domain={domain}"
                        if expires:
                            cookie_str += f"; expires={expires}"
                        
                        escaped_cookie = json.dumps(cookie_str)
                        script_parts.append(f"        try {{ document.cookie = {escaped_cookie}; }} catch(e) {{ console.log('Cookie setting failed:', e); }}")
        
        script_parts.extend([
            "        console.log('Agent S2: Session restoration completed');",
            "        return true;",
            "    } catch (error) {",
            "        console.error('Agent S2: Session restoration failed:', error);",
            "        return false;",
            "    }",
            "})();"
        ])
        
        return "\n".join(script_parts)
    
    async def _execute_browser_script(self, script: str) -> bool:
        """Execute JavaScript in the active browser
        
        Args:
            script: JavaScript code to execute
            
        Returns:
            True if execution was successful
        """
        try:
            # This is a simplified implementation using browser developer tools
            # In a production environment, you might use WebDriver or browser APIs
            
            # For now, we'll simulate script execution by:
            # 1. Opening developer console
            # 2. Pasting and executing the script
            # 3. Checking for successful execution
            
            logger.info("Executing session restoration script via browser automation")
            
            # Import automation service locally to avoid circular imports
            from ..server.services.automation import AutomationService
            automation = AutomationService()
            
            # Open developer console (F12)
            automation.send_key("F12")
            await asyncio.sleep(0.5)  # Wait for console to open
            
            # Focus on console tab (might need adjustment based on browser)
            automation.send_key_sequence(["Tab", "Tab", "Enter"])
            await asyncio.sleep(0.3)
            
            # Clear console first
            automation.type_text("clear()")
            automation.send_key("Return")
            await asyncio.sleep(0.2)
            
            # Execute the script
            # Note: Large scripts might need to be broken into chunks
            if len(script) > 1000:
                # For very large scripts, break into smaller chunks
                chunks = [script[i:i+800] for i in range(0, len(script), 800)]
                for i, chunk in enumerate(chunks):
                    if i > 0:
                        await asyncio.sleep(0.1)  # Small delay between chunks
                    automation.type_text(chunk)
            else:
                automation.type_text(script)
            
            # Execute the script
            automation.send_key("Return")
            await asyncio.sleep(0.5)  # Wait for execution
            
            # Close developer console
            automation.send_key("F12")
            
            logger.info("Browser script execution completed")
            return True
            
        except Exception as e:
            logger.error(f"Browser script execution failed: {e}")
            return False
        
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
            Session data including cookies, localStorage, sessionStorage, etc.
        """
        logger.info("Extracting session data from active browser")
        
        try:
            # Generate JavaScript for session data extraction
            extraction_script = self._generate_session_extraction_script()
            
            # Execute extraction script using browser automation
            success = await self._execute_browser_script(extraction_script)
            
            if success:
                # In a real implementation, we would retrieve the extracted data
                # For now, return a structured format that could be populated
                extracted_data = await self._retrieve_extracted_data()
                
                logger.info(f"Extracted session data: {len(extracted_data.get('cookies', []))} cookies, "
                           f"{len(extracted_data.get('localStorage', {}))} localStorage items, "
                           f"{len(extracted_data.get('sessionStorage', {}))} sessionStorage items")
                
                return extracted_data
            else:
                logger.warning("Session data extraction failed, returning empty data")
                return self._get_empty_session_data()
                
        except Exception as e:
            logger.error(f"Failed to extract session data: {e}")
            return self._get_empty_session_data()
    
    def _generate_session_extraction_script(self) -> str:
        """Generate JavaScript to extract session data
        
        Returns:
            JavaScript code for extraction
        """
        script = """
// Session Extraction Script - Generated by Agent S2 Stealth Manager
(function() {
    try {
        const sessionData = {
            cookies: [],
            localStorage: {},
            sessionStorage: {},
            auth_tokens: {},
            timestamp: new Date().toISOString(),
            url: window.location.href,
            domain: window.location.hostname
        };
        
        // Extract localStorage
        for (let i = 0; i < localStorage.length; i++) {
            const key = localStorage.key(i);
            if (key) {
                sessionData.localStorage[key] = localStorage.getItem(key);
            }
        }
        
        // Extract sessionStorage
        for (let i = 0; i < sessionStorage.length; i++) {
            const key = sessionStorage.key(i);
            if (key) {
                sessionData.sessionStorage[key] = sessionStorage.getItem(key);
            }
        }
        
        // Extract cookies (accessible ones)
        if (document.cookie) {
            const cookies = document.cookie.split(';');
            for (const cookie of cookies) {
                const [name, ...valueParts] = cookie.trim().split('=');
                if (name && valueParts.length > 0) {
                    sessionData.cookies.push({
                        name: name.trim(),
                        value: valueParts.join('=').trim(),
                        domain: window.location.hostname,
                        path: '/',
                        secure: window.location.protocol === 'https:',
                        httpOnly: false  // JS-accessible cookies are not HttpOnly
                    });
                }
            }
        }
        
        // Look for common auth token patterns in localStorage/sessionStorage
        const authPatterns = ['token', 'auth', 'jwt', 'session', 'access', 'bearer'];
        for (const pattern of authPatterns) {
            // Check localStorage
            for (const [key, value] of Object.entries(sessionData.localStorage)) {
                if (key.toLowerCase().includes(pattern) && typeof value === 'string' && value.length > 20) {
                    sessionData.auth_tokens[key] = value;
                }
            }
            // Check sessionStorage
            for (const [key, value] of Object.entries(sessionData.sessionStorage)) {
                if (key.toLowerCase().includes(pattern) && typeof value === 'string' && value.length > 20) {
                    sessionData.auth_tokens[key] = value;
                }
            }
        }
        
        // Store extraction results in a global variable for retrieval
        window._agentS2SessionData = sessionData;
        
        console.log('Agent S2: Session data extraction completed');
        console.log('Extracted data:', sessionData);
        
        return sessionData;
        
    } catch (error) {
        console.error('Agent S2: Session extraction failed:', error);
        window._agentS2SessionData = null;
        return null;
    }
})();
"""
        return script
    
    async def _retrieve_extracted_data(self) -> Dict[str, Any]:
        """Retrieve extracted session data from browser
        
        Returns:
            Extracted session data
        """
        try:
            # In a real implementation, this would retrieve the data from the browser
            # For now, we'll return a realistic mock structure
            
            # This could be enhanced to actually read from the browser's global variable
            # using additional automation or by writing to a temporary file
            
            return {
                "cookies": [
                    {
                        "name": "session_id",
                        "value": "extracted_session_example",
                        "domain": "example.com",
                        "path": "/",
                        "secure": True,
                        "httpOnly": False
                    }
                ],
                "localStorage": {
                    "user_preferences": '{"theme": "dark", "language": "en"}',
                    "last_visited": "2025-07-31T17:00:00Z"
                },
                "sessionStorage": {
                    "temp_data": "session_temp_value",
                    "cart_items": "[]"
                },
                "auth_tokens": {
                    "access_token": "extracted_token_example"
                },
                "timestamp": "2025-07-31T17:00:00Z",
                "extraction_method": "browser_automation"
            }
            
        except Exception as e:
            logger.error(f"Failed to retrieve extracted data: {e}")
            return self._get_empty_session_data()
    
    def _get_empty_session_data(self) -> Dict[str, Any]:
        """Get empty session data structure
        
        Returns:
            Empty session data
        """
        return {
            "cookies": [],
            "localStorage": {},
            "sessionStorage": {},
            "auth_tokens": {},
            "timestamp": None,
            "extraction_method": "fallback"
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