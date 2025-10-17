"""Proxy service for Agent S2 network-level security"""

import os
import sys
import asyncio
import logging
import subprocess
from typing import Optional
from pathlib import Path

try:
    from mitmproxy import options
    from mitmproxy.tools.dump import DumpMaster
    MITMPROXY_AVAILABLE = True
except ImportError:
    MITMPROXY_AVAILABLE = False
    
from .proxy_addon import SecurityProxyAddon

logger = logging.getLogger(__name__)


class ProxyService:
    """Manages the mitmproxy regular proxy service for URL security"""
    
    def __init__(self, port: int = 8085):
        self.port = port
        self.master: Optional[DumpMaster] = None
        self.proxy_task: Optional[asyncio.Task] = None
        self.running = False
        
        if not MITMPROXY_AVAILABLE:
            logger.warning("mitmproxy not available - proxy service disabled")
        else:
            # Check if proxy is already running (e.g., started by supervisor)
            self.running = self._check_proxy_running()
            
    async def start(self):
        """Start the proxy service"""
        if not MITMPROXY_AVAILABLE:
            logger.error("Cannot start proxy service - mitmproxy not installed")
            return False
            
        if self.running:
            logger.warning("Proxy service already running")
            return True
            
        try:
            # Create mitmproxy options
            # Changed from transparent to regular proxy mode to prevent system-wide traffic hijacking
            opts = options.Options(
                listen_port=self.port,
                mode=["regular"],  # Regular proxy mode (safer than transparent)
                ssl_insecure=True,  # Accept self-signed certificates
                confdir=os.path.expanduser("~/.mitmproxy"),
                # Add connection management to prevent TCP leaks
                connection_strategy="lazy",  # Use lazy connection strategy
                stream_large_bodies="50m",   # Stream large bodies to prevent memory issues
                body_size_limit="50m",       # Limit body size
                keep_host_header=True,       # Preserve original host headers
            )
            
            # Create master with our security addon
            self.master = DumpMaster(opts)
            addon = SecurityProxyAddon()
            self.master.addons.add(addon)
            
            # Run proxy in background task
            self.proxy_task = asyncio.create_task(self._run_proxy())
            self.running = True
            
            # Give proxy time to start
            await asyncio.sleep(1)
            
            logger.info(f"Proxy service started on port {self.port}")
            return True
            
        except Exception as e:
            logger.error(f"Failed to start proxy service: {e}")
            self.running = False
            return False
    
    async def _run_proxy(self):
        """Run the proxy (blocking)"""
        try:
            await self.master.run()
        except Exception as e:
            logger.error(f"Proxy error: {e}")
        finally:
            self.running = False
    
    async def stop(self):
        """Stop the proxy service"""
        if not self.running:
            return
            
        logger.info("Stopping proxy service...")
        
        if self.master:
            self.master.shutdown()
            
        if self.proxy_task:
            self.proxy_task.cancel()
            try:
                await self.proxy_task
            except asyncio.CancelledError:
                pass
                
        self.running = False
        logger.info("Proxy service stopped")
    
    def setup_transparent_proxy(self) -> bool:
        """DEPRECATED: Configure system for transparent proxy (requires root)
        
        NOTE: This method is deprecated since we switched to regular proxy mode.
        Regular proxy mode is safer and doesn't require system-wide iptables rules.
        Applications should be configured to use HTTP_PROXY/HTTPS_PROXY environment variables instead.
        """
        if os.geteuid() != 0:
            logger.warning("Transparent proxy setup requires root privileges")
            return False
            
        try:
            # Enable IP forwarding
            subprocess.run(
                ["sysctl", "-w", "net.ipv4.ip_forward=1"],
                check=True,
                capture_output=True
            )
            
            # Setup iptables rules for transparent proxy
            # Redirect HTTP
            subprocess.run([
                "iptables", "-t", "nat", "-A", "OUTPUT",
                "-p", "tcp", "--dport", "80",
                "-j", "REDIRECT", "--to-port", str(self.port)
            ], check=True)
            
            # Redirect HTTPS
            subprocess.run([
                "iptables", "-t", "nat", "-A", "OUTPUT",
                "-p", "tcp", "--dport", "443",
                "-j", "REDIRECT", "--to-port", str(self.port)
            ], check=True)
            
            # Exclude proxy's own traffic (prevent loops)
            subprocess.run([
                "iptables", "-t", "nat", "-I", "OUTPUT",
                "-p", "tcp", "--dport", "80",
                "-m", "owner", "--uid-owner", str(os.getuid()),
                "-j", "RETURN"
            ], check=True)
            
            subprocess.run([
                "iptables", "-t", "nat", "-I", "OUTPUT",
                "-p", "tcp", "--dport", "443",
                "-m", "owner", "--uid-owner", str(os.getuid()),
                "-j", "RETURN"
            ], check=True)
            
            logger.info("Transparent proxy iptables rules configured")
            return True
            
        except subprocess.CalledProcessError as e:
            logger.error(f"Failed to setup iptables rules: {e}")
            return False
    
    def cleanup_iptables(self):
        """Remove iptables rules (requires root)"""
        if os.geteuid() != 0:
            return
            
        try:
            # Remove our rules (inverse of setup)
            rules = [
                ["iptables", "-t", "nat", "-D", "OUTPUT",
                 "-p", "tcp", "--dport", "80",
                 "-j", "REDIRECT", "--to-port", str(self.port)],
                ["iptables", "-t", "nat", "-D", "OUTPUT",
                 "-p", "tcp", "--dport", "443",
                 "-j", "REDIRECT", "--to-port", str(self.port)],
                ["iptables", "-t", "nat", "-D", "OUTPUT",
                 "-p", "tcp", "--dport", "80",
                 "-m", "owner", "--uid-owner", str(os.getuid()),
                 "-j", "RETURN"],
                ["iptables", "-t", "nat", "-D", "OUTPUT",
                 "-p", "tcp", "--dport", "443",
                 "-m", "owner", "--uid-owner", str(os.getuid()),
                 "-j", "RETURN"]
            ]
            
            for rule in rules:
                try:
                    subprocess.run(rule, check=True, capture_output=True)
                except subprocess.CalledProcessError:
                    # Rule might not exist
                    pass
                    
            logger.info("Cleaned up iptables rules")
            
        except Exception as e:
            logger.error(f"Failed to cleanup iptables: {e}")
    
    def _check_proxy_running(self) -> bool:
        """Check if mitmdump is already running on our port"""
        try:
            result = subprocess.run(
                ["pgrep", "-f", f"mitmdump.*--listen-port {self.port}"],
                capture_output=True,
                text=True
            )
            is_running = result.returncode == 0
            if is_running:
                logger.info(f"Detected mitmdump already running on port {self.port}")
            return is_running
        except Exception as e:
            logger.debug(f"Failed to check proxy status: {e}")
            return False
    
    def is_proxy_configured(self) -> bool:
        """Check if system is configured for transparent proxy (iptables rules)"""
        try:
            # Check if redirect rules exist
            result = subprocess.run(
                ["sudo", "iptables", "-t", "nat", "-L", "OUTPUT", "-n"],
                capture_output=True,
                text=True
            )
            # Look for redirect rules to our proxy port
            return f"redir ports {self.port}" in result.stdout and "dpt:80" in result.stdout
        except Exception:
            return False
    
    def install_ca_certificate(self) -> bool:
        """Install mitmproxy CA certificate in Firefox"""
        try:
            # Find mitmproxy CA certificate
            ca_cert = os.path.expanduser("~/.mitmproxy/mitmproxy-ca-cert.pem")
            
            if not os.path.exists(ca_cert):
                # Generate certificates using proper non-blocking approach
                logger.info("Generating mitmproxy CA certificate...")
                try:
                    # Use mitmproxy's cert generation with timeout
                    result = subprocess.run(
                        ["mitmdump", "--set", "confdir=~/.mitmproxy", "-s", "/dev/null"],
                        timeout=5,  # 5 second timeout
                        capture_output=True,
                        input=b"",  # Send empty input to make it exit
                        text=False
                    )
                    # Process will timeout and be killed, but certs should be generated
                except subprocess.TimeoutExpired:
                    logger.debug("mitmdump timed out as expected after generating certificates")
                except Exception as e:
                    logger.warning(f"Certificate generation failed: {e}")
                
            # Check if certificate was generated or already exists
            if not os.path.exists(ca_cert):
                logger.info("CA certificate not found - proxy will work without Firefox HTTPS interception")
                return False
                
            # Find Firefox profile directory
            firefox_dir = os.path.expanduser("~/.mozilla/firefox")
            if not os.path.exists(firefox_dir):
                logger.warning("Firefox profile directory not found")
                return False
                
            # Look for default profile
            profiles = []
            for item in os.listdir(firefox_dir):
                if item.endswith(".default") or item.endswith(".default-release"):
                    profiles.append(os.path.join(firefox_dir, item))
                    
            if not profiles:
                logger.warning("No Firefox profiles found")
                return False
                
            # Install certificate in each profile
            for profile in profiles:
                try:
                    # Use certutil to add certificate
                    subprocess.run([
                        "certutil", "-A", "-n", "mitmproxy",
                        "-t", "CT,c,c", "-i", ca_cert,
                        "-d", f"sql:{profile}"
                    ], check=True, capture_output=True)
                    
                    logger.info(f"Installed mitmproxy CA in Firefox profile: {profile}")
                    
                except subprocess.CalledProcessError as e:
                    logger.error(f"Failed to install certificate in {profile}: {e}")
                    
            return True
            
        except Exception as e:
            logger.error(f"Failed to install CA certificate: {e}")
            return False
    
    def get_proxy_config(self) -> dict:
        """Get proxy configuration for applications in regular proxy mode"""
        proxy_url = f"http://localhost:{self.port}"
        return {
            "HTTP_PROXY": proxy_url,
            "HTTPS_PROXY": proxy_url,
            "http_proxy": proxy_url,
            "https_proxy": proxy_url,
            # Don't proxy localhost and internal services
            "NO_PROXY": "localhost,127.0.0.1,::1,host.docker.internal",
            "no_proxy": "localhost,127.0.0.1,::1,host.docker.internal"
        }
    
    def configure_environment_for_proxy(self):
        """Configure current process environment for proxy usage"""
        proxy_config = self.get_proxy_config()
        for key, value in proxy_config.items():
            os.environ[key] = value
        logger.info(f"Configured environment for regular proxy mode on port {self.port}")
    
    def is_proxy_configured(self) -> bool:
        """Check if transparent proxy is configured"""
        try:
            # Check if our iptables rules exist
            result = subprocess.run(
                ["iptables", "-t", "nat", "-L", "OUTPUT", "-n"],
                capture_output=True,
                text=True
            )
            
            return f"REDIRECT tcp -- 0.0.0.0/0 0.0.0.0/0 tcp dpt:80 redir ports {self.port}" in result.stdout
            
        except (subprocess.CalledProcessError, FileNotFoundError, PermissionError):
            return False


class ProxyManager:
    """Singleton manager for proxy service"""
    
    _instance: Optional['ProxyManager'] = None
    _proxy_service: Optional[ProxyService] = None
    
    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance
    
    def get_proxy_service(self) -> ProxyService:
        """Get or create proxy service instance"""
        if self._proxy_service is None:
            self._proxy_service = ProxyService()
        return self._proxy_service
    
    async def ensure_proxy_running(self) -> bool:
        """Ensure proxy is running"""
        service = self.get_proxy_service()
        
        if not service.running:
            return await service.start()
            
        return True