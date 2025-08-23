"""Constants for Agent S2 examples

This file centralizes all configuration values used across examples
to avoid hardcoding URLs, ports, and other settings.
"""

import os

# Agent S2 API Configuration
AGENT_S2_HOST = os.getenv("AGENT_S2_HOST", "localhost")
AGENT_S2_PORT = int(os.getenv("AGENT_S2_PORT", "4113"))
AGENT_S2_BASE_URL = f"http://{AGENT_S2_HOST}:{AGENT_S2_PORT}"

# VNC Configuration  
VNC_HOST = os.getenv("VNC_HOST", "localhost")
VNC_PORT = int(os.getenv("VNC_PORT", "5900"))
VNC_URL = f"vnc://{VNC_HOST}:{VNC_PORT}"
VNC_PASSWORD = os.getenv("VNC_PASSWORD", "agents2vnc")

# Ollama Configuration
OLLAMA_HOST = os.getenv("OLLAMA_HOST", "localhost")
OLLAMA_PORT = int(os.getenv("OLLAMA_PORT", "11434"))
OLLAMA_BASE_URL = f"http://{OLLAMA_HOST}:{OLLAMA_PORT}"
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "llama3.2-vision:11b")

# Example Defaults
DEFAULT_SCREENSHOT_FORMAT = "png"
DEFAULT_SCREENSHOT_QUALITY = 95
DEFAULT_TIMEOUT = 60  # seconds
DEFAULT_MAX_RETRIES = 3

# Directory paths
OUTPUTS_DIR = os.getenv("AGENT_S2_OUTPUTS_DIR", "./outputs")
SCREENSHOTS_DIR = os.path.join(OUTPUTS_DIR, "screenshots")
LOGS_DIR = os.path.join(OUTPUTS_DIR, "logs")

# Ensure output directories exist
os.makedirs(OUTPUTS_DIR, exist_ok=True)
os.makedirs(SCREENSHOTS_DIR, exist_ok=True)
os.makedirs(LOGS_DIR, exist_ok=True)

# Headers
DEFAULT_HEADERS = {
    "Content-Type": "application/json"
}

# Logging Configuration
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")
LOG_FORMAT = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"