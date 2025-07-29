"""Session data and state persistence for Agent S2"""

import json
import os
import shutil
from pathlib import Path
from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.kdf.pbkdf2 import PBKDF2HMAC
import base64
import logging

logger = logging.getLogger(__name__)


class SessionManager:
    """Manages browser session data and state persistence"""
    
    def __init__(self, storage_path: str = "/data/sessions", 
                 encryption_enabled: bool = True,
                 ttl_days: int = 30):
        """Initialize session manager
        
        Args:
            storage_path: Base path for session storage
            encryption_enabled: Whether to encrypt session data
            ttl_days: Time to live for sessions in days
        """
        self.storage_path = Path(storage_path)
        self.encryption_enabled = encryption_enabled
        self.ttl_days = ttl_days
        
        # Create directory structure
        self._init_storage()
        
        # Initialize encryption if enabled
        self._cipher = None
        if encryption_enabled:
            self._init_encryption()
            
    def _init_storage(self) -> None:
        """Initialize storage directory structure"""
        directories = [
            self.storage_path / "profiles",
            self.storage_path / "states", 
            self.storage_path / "fingerprints",
        ]
        
        for directory in directories:
            directory.mkdir(parents=True, exist_ok=True)
            
    def _init_encryption(self) -> None:
        """Initialize encryption cipher"""
        key_file = self.storage_path / ".session_key"
        
        if key_file.exists():
            # Load existing key
            with open(key_file, "rb") as f:
                key = f.read()
        else:
            # Generate new key
            key = Fernet.generate_key()
            with open(key_file, "wb") as f:
                f.write(key)
            # Secure permissions
            os.chmod(key_file, 0o600)
            
        self._cipher = Fernet(key)
        
    def _encrypt_data(self, data: str) -> str:
        """Encrypt data if encryption is enabled
        
        Args:
            data: String data to encrypt
            
        Returns:
            Encrypted data or original if encryption disabled
        """
        if not self.encryption_enabled or not self._cipher:
            return data
            
        return self._cipher.encrypt(data.encode()).decode()
        
    def _decrypt_data(self, data: str) -> str:
        """Decrypt data if encryption is enabled
        
        Args:
            data: Encrypted data
            
        Returns:
            Decrypted data or original if encryption disabled
        """
        if not self.encryption_enabled or not self._cipher:
            return data
            
        return self._cipher.decrypt(data.encode()).decode()
        
    def save_session_data(self, profile_id: str, data: Dict[str, Any]) -> None:
        """Save session data for a profile
        
        Args:
            profile_id: Profile identifier
            data: Session data including cookies, storage, etc.
        """
        profile_dir = self.storage_path / "profiles" / profile_id
        profile_dir.mkdir(exist_ok=True)
        
        # Add metadata
        data["_metadata"] = {
            "saved_at": datetime.now().isoformat(),
            "expires_at": (datetime.now() + timedelta(days=self.ttl_days)).isoformat(),
            "version": "1.0"
        }
        
        # Save individual components
        components = {
            "cookies": data.get("cookies", []),
            "localStorage": data.get("localStorage", {}),
            "sessionStorage": data.get("sessionStorage", {}), 
            "indexedDB": data.get("indexedDB", {}),
            "auth_tokens": data.get("auth_tokens", {}),
            "headers": data.get("headers", {}),
            "_metadata": data["_metadata"]
        }
        
        for name, component_data in components.items():
            file_path = profile_dir / f"{name}.json"
            json_data = json.dumps(component_data, indent=2)
            
            if self.encryption_enabled and name != "_metadata":
                # Encrypt sensitive data
                encrypted = self._encrypt_data(json_data)
                with open(file_path, "w") as f:
                    f.write(encrypted)
            else:
                with open(file_path, "w") as f:
                    f.write(json_data)
                    
        logger.info(f"Saved session data for profile: {profile_id}")
        
    def load_session_data(self, profile_id: str) -> Optional[Dict[str, Any]]:
        """Load session data for a profile
        
        Args:
            profile_id: Profile identifier
            
        Returns:
            Session data or None if not found/expired
        """
        profile_dir = self.storage_path / "profiles" / profile_id
        
        if not profile_dir.exists():
            logger.warning(f"Profile not found: {profile_id}")
            return None
            
        # Check metadata first
        metadata_file = profile_dir / "_metadata.json"
        if metadata_file.exists():
            with open(metadata_file, "r") as f:
                metadata = json.load(f)
                
            # Check expiration
            expires_at = datetime.fromisoformat(metadata["expires_at"])
            if datetime.now() > expires_at:
                logger.warning(f"Session expired for profile: {profile_id}")
                self.delete_session_data(profile_id)
                return None
                
        # Load components
        data = {}
        components = ["cookies", "localStorage", "sessionStorage", 
                     "indexedDB", "auth_tokens", "headers"]
        
        for name in components:
            file_path = profile_dir / f"{name}.json"
            if file_path.exists():
                with open(file_path, "r") as f:
                    content = f.read()
                    
                if self.encryption_enabled:
                    try:
                        content = self._decrypt_data(content)
                    except Exception as e:
                        logger.error(f"Failed to decrypt {name}: {e}")
                        continue
                        
                try:
                    data[name] = json.loads(content)
                except json.JSONDecodeError as e:
                    logger.error(f"Failed to parse {name}: {e}")
                    
        logger.info(f"Loaded session data for profile: {profile_id}")
        return data
        
    def delete_session_data(self, profile_id: str) -> None:
        """Delete session data for a profile
        
        Args:
            profile_id: Profile identifier
        """
        profile_dir = self.storage_path / "profiles" / profile_id
        
        if profile_dir.exists():
            shutil.rmtree(profile_dir)
            logger.info(f"Deleted session data for profile: {profile_id}")
            
    def save_session_state(self, state: Dict[str, Any]) -> None:
        """Save browser window/tab state
        
        Args:
            state: Window and tab state data
        """
        state_file = self.storage_path / "states" / "current_state.json"
        
        # Add timestamp
        state["_saved_at"] = datetime.now().isoformat()
        
        with open(state_file, "w") as f:
            json.dump(state, f, indent=2)
            
        logger.info("Saved session state")
        
    def load_session_state(self) -> Optional[Dict[str, Any]]:
        """Load browser window/tab state
        
        Returns:
            State data or None if not found
        """
        state_file = self.storage_path / "states" / "current_state.json"
        
        if not state_file.exists():
            return None
            
        with open(state_file, "r") as f:
            state = json.load(f)
            
        logger.info("Loaded session state")
        return state
        
    def delete_session_state(self) -> None:
        """Delete saved session state"""
        state_file = self.storage_path / "states" / "current_state.json"
        
        if state_file.exists():
            state_file.unlink()
            logger.info("Deleted session state")
            
    def list_profiles(self) -> List[Dict[str, Any]]:
        """List all saved profiles
        
        Returns:
            List of profile information
        """
        profiles = []
        profiles_dir = self.storage_path / "profiles"
        
        if not profiles_dir.exists():
            return profiles
            
        for profile_dir in profiles_dir.iterdir():
            if profile_dir.is_dir():
                metadata_file = profile_dir / "_metadata.json"
                
                if metadata_file.exists():
                    with open(metadata_file, "r") as f:
                        metadata = json.load(f)
                        
                    profiles.append({
                        "id": profile_dir.name,
                        "saved_at": metadata["saved_at"],
                        "expires_at": metadata["expires_at"],
                        "size": sum(f.stat().st_size for f in profile_dir.glob("*.json"))
                    })
                    
        return sorted(profiles, key=lambda x: x["saved_at"], reverse=True)
        
    def cleanup_expired(self) -> int:
        """Clean up expired sessions
        
        Returns:
            Number of profiles cleaned up
        """
        profiles = self.list_profiles()
        cleaned = 0
        
        for profile in profiles:
            expires_at = datetime.fromisoformat(profile["expires_at"])
            if datetime.now() > expires_at:
                self.delete_session_data(profile["id"])
                cleaned += 1
                
        if cleaned > 0:
            logger.info(f"Cleaned up {cleaned} expired profiles")
            
        return cleaned
        
    def export_profile(self, profile_id: str, output_path: str) -> None:
        """Export profile to a file
        
        Args:
            profile_id: Profile to export
            output_path: Output file path
        """
        data = self.load_session_data(profile_id)
        
        if not data:
            raise ValueError(f"Profile not found: {profile_id}")
            
        # Remove metadata before export
        data.pop("_metadata", None)
        
        with open(output_path, "w") as f:
            json.dump(data, f, indent=2)
            
        logger.info(f"Exported profile {profile_id} to {output_path}")
        
    def import_profile(self, profile_id: str, input_path: str) -> None:
        """Import profile from a file
        
        Args:
            profile_id: Profile identifier to use
            input_path: Input file path
        """
        with open(input_path, "r") as f:
            data = json.load(f)
            
        self.save_session_data(profile_id, data)
        logger.info(f"Imported profile {profile_id} from {input_path}")
        
    def get_profile_size(self, profile_id: str) -> int:
        """Get storage size of a profile
        
        Args:
            profile_id: Profile identifier
            
        Returns:
            Size in bytes
        """
        profile_dir = self.storage_path / "profiles" / profile_id
        
        if not profile_dir.exists():
            return 0
            
        return sum(f.stat().st_size for f in profile_dir.glob("*.json"))