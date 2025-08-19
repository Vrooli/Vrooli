#!/bin/bash
# TARS-desktop installation functionality

# Get script directory
TARS_DESKTOP_INSTALL_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${TARS_DESKTOP_INSTALL_LIB_DIR}/core.sh"

# Install TARS-desktop
tars_desktop::install() {
    local verbose="${1:-false}"
    
    log::header "Installing TARS-desktop"
    
    # Initialize
    tars_desktop::init "$verbose"
    
    # Check if already installed
    if tars_desktop::is_installed; then
        log::success "TARS-desktop already installed"
        return 0
    fi
    
    # Check prerequisites
    log::info "Checking prerequisites..."
    
    # Check Python
    if ! command -v python3 >/dev/null 2>&1; then
        log::error "Python 3 is required but not installed"
        return 1
    fi
    
    local python_version
    python_version=$(python3 --version 2>&1 | grep -oP '\d+\.\d+' | head -1)
    log::info "Python version: $python_version"
    
    # Check git
    if ! command -v git >/dev/null 2>&1; then
        log::error "Git is required but not installed"
        return 1
    fi
    
    # Create installation directory
    log::info "Creating installation directory..."
    if [[ -d "$TARS_DESKTOP_INSTALL_DIR" ]]; then
        log::info "Installation directory already exists, cleaning..."
        rm -rf "$TARS_DESKTOP_INSTALL_DIR"
    fi
    mkdir -p "$TARS_DESKTOP_INSTALL_DIR"
    
    # Try to create virtual environment, fall back to user install if fails
    log::info "Setting up Python environment..."
    
    # Try venv first
    if python3 -m venv "$TARS_DESKTOP_VENV_DIR" 2>/dev/null; then
        log::info "Virtual environment created successfully"
        PYTHON_CMD="${TARS_DESKTOP_VENV_DIR}/bin/python"
        PIP_CMD="${TARS_DESKTOP_VENV_DIR}/bin/pip"
        
        # Upgrade pip in venv
        $PIP_CMD install --upgrade pip >/dev/null 2>&1
    else
        log::warning "Cannot create virtual environment, using user installation"
        PYTHON_CMD="python3"
        PIP_CMD="python3 -m pip"
        
        # Ensure pip is available
        if ! $PIP_CMD --version >/dev/null 2>&1; then
            log::error "pip is not available"
            return 1
        fi
    fi
    
    # Install dependencies
    log::info "Installing Python dependencies..."
    
    # Create requirements file
    cat > "${TARS_DESKTOP_INSTALL_DIR}/requirements.txt" <<EOF
pyautogui>=0.9.54
opencv-python>=4.8.0
pillow>=10.0.0
numpy>=1.24.0
fastapi>=0.104.0
uvicorn>=0.24.0
pydantic>=2.4.0
python-multipart>=0.0.6
screeninfo>=0.8.1
mss>=9.0.1
pytesseract>=0.3.10
EOF
    
    # Install requirements
    $PIP_CMD install --user -r "${TARS_DESKTOP_INSTALL_DIR}/requirements.txt" >/dev/null 2>&1
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to install Python dependencies"
        return 1
    fi
    
    # Create server script
    log::info "Creating server script..."
    cat > "${TARS_DESKTOP_INSTALL_DIR}/server.py" <<'EOF'
#!/usr/bin/env python3
"""TARS Desktop UI Automation Server"""

import os
import json
import base64
from typing import Dict, Any, Optional
from datetime import datetime
import asyncio

from fastapi import FastAPI, HTTPException, Header
from fastapi.responses import JSONResponse, FileResponse
from pydantic import BaseModel
import pyautogui
import mss
import cv2
import numpy as np
from PIL import Image

# Configure PyAutoGUI
pyautogui.FAILSAFE = True
pyautogui.PAUSE = 0.5

app = FastAPI(title="TARS Desktop", version="1.0.0")

class ActionRequest(BaseModel):
    action: str
    target: Optional[str] = None
    coordinates: Optional[Dict[str, int]] = None
    text: Optional[str] = None
    duration: Optional[float] = 1.0

class HealthResponse(BaseModel):
    status: str = "ok"
    timestamp: str
    capabilities: Dict[str, bool]

@app.get("/health")
async def health_check():
    """Check service health"""
    return HealthResponse(
        status="ok",
        timestamp=datetime.now().isoformat(),
        capabilities={
            "screenshot": True,
            "click": True,
            "type": True,
            "scroll": True,
            "drag": True,
            "hotkey": True
        }
    )

@app.get("/capabilities")
async def get_capabilities():
    """Get available capabilities"""
    screen_width, screen_height = pyautogui.size()
    return {
        "screen": {
            "width": screen_width,
            "height": screen_height
        },
        "mouse": {
            "position": pyautogui.position()
        },
        "actions": [
            "click", "doubleClick", "rightClick",
            "moveTo", "dragTo", "scroll",
            "typewrite", "hotkey", "screenshot"
        ]
    }

@app.post("/execute")
async def execute_action(request: ActionRequest):
    """Execute a UI action"""
    try:
        action = request.action.lower()
        
        if action == "click":
            if request.coordinates:
                pyautogui.click(
                    x=request.coordinates.get("x"),
                    y=request.coordinates.get("y")
                )
            else:
                pyautogui.click()
                
        elif action == "doubleclick":
            if request.coordinates:
                pyautogui.doubleClick(
                    x=request.coordinates.get("x"),
                    y=request.coordinates.get("y")
                )
            else:
                pyautogui.doubleClick()
                
        elif action == "rightclick":
            if request.coordinates:
                pyautogui.rightClick(
                    x=request.coordinates.get("x"),
                    y=request.coordinates.get("y")
                )
            else:
                pyautogui.rightClick()
                
        elif action == "moveto":
            if request.coordinates:
                pyautogui.moveTo(
                    x=request.coordinates.get("x"),
                    y=request.coordinates.get("y"),
                    duration=request.duration
                )
                
        elif action == "dragto":
            if request.coordinates:
                pyautogui.dragTo(
                    x=request.coordinates.get("x"),
                    y=request.coordinates.get("y"),
                    duration=request.duration
                )
                
        elif action == "scroll":
            clicks = request.coordinates.get("clicks", 3) if request.coordinates else 3
            pyautogui.scroll(clicks)
            
        elif action == "typewrite":
            if request.text:
                pyautogui.typewrite(request.text)
                
        elif action == "hotkey":
            if request.text:
                keys = request.text.split("+")
                pyautogui.hotkey(*keys)
                
        else:
            raise ValueError(f"Unknown action: {action}")
            
        return {"success": True, "action": action}
        
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@app.get("/screenshot")
async def capture_screenshot():
    """Capture a screenshot"""
    try:
        with mss.mss() as sct:
            monitor = sct.monitors[0]
            screenshot = sct.grab(monitor)
            
            # Convert to PIL Image
            img = Image.frombytes("RGB", screenshot.size, screenshot.bgra, "raw", "BGRX")
            
            # Save to temp file
            temp_path = f"/tmp/tars_screenshot_{datetime.now().timestamp()}.png"
            img.save(temp_path)
            
            return FileResponse(temp_path, media_type="image/png")
            
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/find_element")
async def find_element(image_base64: str, template_base64: str):
    """Find an element on screen using template matching"""
    try:
        # Decode images
        image_bytes = base64.b64decode(image_base64)
        template_bytes = base64.b64decode(template_base64)
        
        # Convert to OpenCV format
        image_np = np.frombuffer(image_bytes, np.uint8)
        template_np = np.frombuffer(template_bytes, np.uint8)
        
        image = cv2.imdecode(image_np, cv2.IMREAD_COLOR)
        template = cv2.imdecode(template_np, cv2.IMREAD_COLOR)
        
        # Template matching
        result = cv2.matchTemplate(image, template, cv2.TM_CCOEFF_NORMED)
        min_val, max_val, min_loc, max_loc = cv2.minMaxLoc(result)
        
        if max_val > 0.8:  # Threshold for match confidence
            h, w = template.shape[:2]
            return {
                "found": True,
                "confidence": float(max_val),
                "location": {
                    "x": int(max_loc[0]),
                    "y": int(max_loc[1]),
                    "width": w,
                    "height": h
                }
            }
        else:
            return {"found": False, "confidence": float(max_val)}
            
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("TARS_DESKTOP_PORT", 11570))
    uvicorn.run(app, host="0.0.0.0", port=port)
EOF
    
    chmod +x "${TARS_DESKTOP_INSTALL_DIR}/server.py"
    
    # Create systemd service file (optional)
    if [[ -d "/etc/systemd/system" ]] && [[ "$EUID" -eq 0 ]]; then
        log::info "Creating systemd service..."
        
        # Determine python path
        if [[ -f "${TARS_DESKTOP_VENV_DIR}/bin/python" ]]; then
            EXEC_PYTHON="${TARS_DESKTOP_VENV_DIR}/bin/python"
            EXEC_PATH="${TARS_DESKTOP_VENV_DIR}/bin:/usr/local/bin:/usr/bin:/bin"
        else
            EXEC_PYTHON="python3"
            EXEC_PATH="/usr/local/bin:/usr/bin:/bin"
        fi
        
        cat > /etc/systemd/system/tars-desktop.service <<EOF
[Unit]
Description=TARS Desktop UI Automation Server
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$TARS_DESKTOP_INSTALL_DIR
Environment="PATH=$EXEC_PATH"
Environment="TARS_DESKTOP_PORT=$TARS_DESKTOP_PORT"
ExecStart=$EXEC_PYTHON ${TARS_DESKTOP_INSTALL_DIR}/server.py
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
        
        systemctl daemon-reload
        log::success "Systemd service created"
    fi
    
    log::success "TARS-desktop installed successfully"
    return 0
}

# Uninstall TARS-desktop
tars_desktop::uninstall() {
    local verbose="${1:-false}"
    
    log::header "Uninstalling TARS-desktop"
    
    # Stop if running
    if tars_desktop::is_running; then
        tars_desktop::stop "$verbose"
    fi
    
    # Remove installation directory
    if [[ -d "$TARS_DESKTOP_INSTALL_DIR" ]]; then
        rm -rf "$TARS_DESKTOP_INSTALL_DIR"
        log::info "Removed installation directory"
    fi
    
    # Remove cache directory
    if [[ -d "$TARS_DESKTOP_CACHE_DIR" ]]; then
        rm -rf "$TARS_DESKTOP_CACHE_DIR"
        log::info "Removed cache directory"
    fi
    
    # Remove systemd service if exists
    if [[ -f "/etc/systemd/system/tars-desktop.service" ]] && [[ "$EUID" -eq 0 ]]; then
        systemctl stop tars-desktop 2>/dev/null
        systemctl disable tars-desktop 2>/dev/null
        rm -f /etc/systemd/system/tars-desktop.service
        systemctl daemon-reload
        log::info "Removed systemd service"
    fi
    
    # Unregister from resource registry
    resource_registry::unregister "$TARS_DESKTOP_NAME"
    
    log::success "TARS-desktop uninstalled successfully"
    return 0
}

# Export functions
export -f tars_desktop::install
export -f tars_desktop::uninstall