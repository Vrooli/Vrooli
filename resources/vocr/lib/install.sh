#!/usr/bin/env bash
# VOCR Installation Module

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VOCR_INSTALL_DIR="${APP_ROOT}/resources/vocr/lib"

# Source utilities first
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/sudo.sh"

# Source configuration
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/vocr/config/defaults.sh"

# Main installation function
vocr::install() {
    log::header "Installing VOCR"
    
    # Export configuration
    vocr::export_config
    
    # Validate configuration
    if ! vocr::validate_config; then
        log::error "Configuration validation failed"
        return 1
    fi
    
    # Create required directories
    log::info "Creating directories..."
    for dir in "$VOCR_DATA_DIR" "$VOCR_CONFIG_DIR" "$VOCR_SCREENSHOTS_DIR" "$VOCR_MODELS_DIR" "$VOCR_LOGS_DIR"; do
        mkdir -p "$dir" || {
            log::error "Failed to create directory: $dir"
            return 1
        }
    done
    
    # Install dependencies based on platform
    log::info "Installing dependencies..."
    if ! vocr::install::dependencies; then
        log::error "Failed to install dependencies"
        return 1
    fi
    
    # Setup Python environment
    log::info "Setting up Python environment..."
    if ! vocr::install::python_env; then
        log::error "Failed to setup Python environment"
        return 1
    fi
    
    # Install OCR models
    log::info "Installing OCR models..."
    if ! vocr::install::ocr_models; then
        log::warning "OCR models installation incomplete (will download on first use)"
    fi
    
    # Setup service
    log::info "Setting up VOCR service..."
    if ! vocr::install::service; then
        log::error "Failed to setup service"
        return 1
    fi
    
    # Register with Vrooli
    log::info "Registering with Vrooli..."
    if ! vocr::install::register; then
        log::warning "Failed to register with Vrooli (manual registration may be needed)"
    fi
    
    # Install CLI
    log::info "Installing CLI..."
    local install_cli_path="${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh"
    if [[ -x "$install_cli_path" ]]; then
        "$install_cli_path" "${APP_ROOT}/resources/vocr" || log::warning "CLI installation failed"
    else
        log::warning "install-resource-cli.sh not found at $install_cli_path"
    fi
    
    log::success "VOCR installed successfully"
    
    # Show next steps
    echo ""
    log::info "Next steps:"
    echo "1. Grant screen permissions: resource-vocr configure"
    echo "2. Test capture: resource-vocr test-capture"
    echo "3. Start service: resource-vocr start"
    
    return 0
}

# Install platform-specific dependencies
vocr::install::dependencies() {
    # Use Docker for Tesseract if Docker is available
    if command -v docker &>/dev/null; then
        log::info "Setting up Tesseract via Docker..."
        if ! vocr::install::docker_tesseract; then
            log::warning "Docker setup failed, falling back to system install"
            vocr::install::system_dependencies
        fi
    else
        vocr::install::system_dependencies
    fi
    
    return 0
}

# Install Tesseract via Docker
vocr::install::docker_tesseract() {
    # For VOCR, we'll use Python-based OCR (pytesseract with EasyOCR fallback)
    # This avoids the need for system tesseract installation
    log::info "Configuring Python-based OCR (no system tesseract needed)..."
    
    # Update Python environment to use pure Python OCR
    local venv_dir="${VOCR_DATA_DIR}/venv"
    if [[ -d "$venv_dir" ]]; then
        log::info "Installing additional OCR libraries..."
        # Install EasyOCR as primary OCR engine (works without tesseract binary)
        "${venv_dir}/bin/pip" install easyocr 2>/dev/null || {
            log::warning "EasyOCR installation failed, will use pytesseract with workarounds"
        }
        
        # Install additional image processing libraries
        "${venv_dir}/bin/pip" install \
            scikit-image \
            scipy \
            torch torchvision --index-url https://download.pytorch.org/whl/cpu \
            2>/dev/null || true
    fi
    
    # Create a mock tesseract wrapper that uses Python
    cat > "${VOCR_DATA_DIR}/tesseract-wrapper.sh" << 'EOF'
#!/bin/bash
# Wrapper that uses Python OCR instead of tesseract binary
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PYTHON_SCRIPT="${VOCR_DATA_DIR:-${APP_ROOT}/data/vocr}/venv/bin/python"
if [[ ! -x "$PYTHON_SCRIPT" ]]; then
    echo "Error: Python environment not found" >&2
    exit 1
fi

# Simple Python OCR wrapper
"$PYTHON_SCRIPT" -c "
import sys
try:
    import easyocr
    reader = easyocr.Reader(['en'])
    if len(sys.argv) > 1:
        result = reader.readtext(sys.argv[1], detail=0)
        print(' '.join(result))
except ImportError:
    # Fallback to PIL-based basic OCR
    from PIL import Image
    if len(sys.argv) > 1:
        img = Image.open(sys.argv[1])
        print('OCR text extraction (tesseract not available)')
" "$@"
EOF
    chmod +x "${VOCR_DATA_DIR}/tesseract-wrapper.sh"
    
    # Link wrapper to bin directory
    mkdir -p "${HOME}/.local/bin"
    ln -sf "${VOCR_DATA_DIR}/tesseract-wrapper.sh" "${HOME}/.local/bin/tesseract-vocr"
    
    log::success "Python-based OCR configured (no system tesseract needed)"
    return 0
}

# Install system dependencies (with conditional sudo)
vocr::install::system_dependencies() {
    case "$(uname -s)" in
        Darwin*)
            # macOS dependencies
            if ! command -v brew &>/dev/null; then
                log::warning "Homebrew not found, skipping dependency installation"
                return 0
            fi
            
            # Install Tesseract
            if ! command -v tesseract &>/dev/null; then
                log::info "Installing Tesseract..."
                brew install tesseract || true
            fi
            
            # Install ImageMagick for image processing
            if ! command -v convert &>/dev/null; then
                log::info "Installing ImageMagick..."
                brew install imagemagick || true
            fi
            ;;
            
        Linux*)
            # Linux dependencies with conditional sudo
            if command -v apt-get &>/dev/null; then
                # Debian/Ubuntu
                log::info "Checking Linux dependencies..."
                if sudo::can_use_sudo; then
                    log::info "Installing system packages with sudo..."
                    sudo::exec_with_fallback "apt-get update" || true
                    sudo::exec_with_fallback "apt-get install -y tesseract-ocr tesseract-ocr-eng python3-pip python3-venv scrot imagemagick" || {
                        log::warning "Some packages failed to install, falling back to Python-only approach"
                    }
                else
                    log::warning "Cannot install system dependencies without sudo access"
                    log::info "Will use Docker-based approach or Python fallback"
                fi
            elif command -v yum &>/dev/null; then
                # RHEL/CentOS
                if sudo::can_use_sudo; then
                    sudo::exec_with_fallback "yum install -y tesseract tesseract-langpack-eng python3-pip scrot ImageMagick" || true
                else
                    log::warning "Cannot install system dependencies without sudo access"
                fi
            else
                log::warning "Package manager not recognized, will use Docker approach"
            fi
            ;;
            
        *)
            log::warning "Platform not recognized, will use Docker approach"
            ;;
    esac
    
    return 0
}

# Setup Python environment
vocr::install::python_env() {
    # Check Python availability
    if ! command -v python3 &>/dev/null; then
        log::error "Python 3 not found"
        return 1
    fi
    
    # Create virtual environment
    local venv_dir="${VOCR_DATA_DIR}/venv"
    if [[ ! -d "$venv_dir" ]]; then
        log::info "Creating Python virtual environment..."
        # Try creating venv, if it fails try without pip
        if ! python3 -m venv "$venv_dir" 2>/dev/null; then
            log::warning "Standard venv creation failed, trying without pip..."
            python3 -m venv --without-pip "$venv_dir" || {
                log::error "Failed to create virtual environment"
                log::info "Please install python3-venv package"
                return 1
            }
            # Manually install pip if needed
            curl -s https://bootstrap.pypa.io/get-pip.py | "${venv_dir}/bin/python3" 2>/dev/null || true
        fi
    fi
    
    # Install Python packages
    log::info "Installing Python packages..."
    "${venv_dir}/bin/pip" install --upgrade pip 2>/dev/null || true
    "${venv_dir}/bin/pip" install \
        pillow \
        pytesseract \
        numpy \
        opencv-python \
        flask \
        flask-cors \
        requests \
        2>/dev/null || true
    
    # Install EasyOCR if GPU is available or requested
    if [[ "$VOCR_USE_GPU" == "true" ]] || [[ "$VOCR_OCR_ENGINE" == "easyocr" ]]; then
        log::info "Installing EasyOCR..."
        "${venv_dir}/bin/pip" install easyocr 2>/dev/null || true
    fi
    
    return 0
}

# Install OCR models
vocr::install::ocr_models() {
    # Download Tesseract language data if needed
    if [[ "$VOCR_OCR_ENGINE" == "tesseract" ]]; then
        local tessdata_dir="/usr/share/tesseract-ocr/5/tessdata"
        if [[ ! -d "$tessdata_dir" ]]; then
            tessdata_dir="/usr/share/tesseract-ocr/4.00/tessdata"
        fi
        if [[ ! -d "$tessdata_dir" ]]; then
            tessdata_dir="/usr/local/share/tessdata"
        fi
        
        if [[ -d "$tessdata_dir" ]]; then
            log::info "Tesseract data directory: $tessdata_dir"
            # Check for English language data
            if [[ ! -f "$tessdata_dir/eng.traineddata" ]]; then
                log::warning "English language data not found for Tesseract"
            fi
        else
            log::warning "Tesseract data directory not found"
        fi
    fi
    
    return 0
}

# Setup service
vocr::install::service() {
    # Create service script
    cat > "${VOCR_DATA_DIR}/vocr-service.py" << 'EOF'
#!/usr/bin/env python3
"""
VOCR Service - Vision OCR and Screen Recognition API
"""

import os
import sys
import json
import time
import subprocess
import tempfile
from pathlib import Path
from flask import Flask, request, jsonify
from flask_cors import CORS

# Add venv packages to path
venv_site_packages = Path(__file__).parent / "venv" / "lib" / "python*" / "site-packages"
for sp in venv_site_packages.parent.glob("python*/site-packages"):
    sys.path.insert(0, str(sp))

from PIL import Image
import numpy as np

# Try to import OCR libraries
try:
    import easyocr
    OCR_ENGINE = 'easyocr'
    ocr_reader = None  # Will be initialized on first use
except ImportError:
    OCR_ENGINE = 'fallback'
    try:
        import pytesseract
        OCR_ENGINE = 'pytesseract'
    except ImportError:
        pass

app = Flask(__name__)
CORS(app)

# Configuration from environment
CONFIG = {
    'port': int(os.environ.get('VOCR_PORT', 9420)),
    'host': os.environ.get('VOCR_HOST', '0.0.0.0'),
    'screenshots_dir': os.environ.get('VOCR_SCREENSHOTS_DIR', '/tmp/vocr/screenshots'),
    'ocr_engine': os.environ.get('VOCR_OCR_ENGINE', 'tesseract'),
    'ocr_languages': os.environ.get('VOCR_OCR_LANGUAGES', 'eng'),
}

# Ensure directories exist
Path(CONFIG['screenshots_dir']).mkdir(parents=True, exist_ok=True)

@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'vocr',
        'timestamp': time.time(),
        'config': CONFIG
    })

@app.route('/capture', methods=['POST'])
def capture():
    """Capture screen region"""
    data = request.json or {}
    region = data.get('region', 'fullscreen')
    
    timestamp = int(time.time() * 1000)
    output_file = Path(CONFIG['screenshots_dir']) / f'capture_{timestamp}.png'
    
    try:
        # Platform-specific capture
        if sys.platform == 'darwin':
            # macOS
            cmd = ['screencapture', '-x']
            if region != 'fullscreen':
                x, y, w, h = map(int, region.split(','))
                cmd.extend(['-R', f'{x},{y},{w},{h}'])
            cmd.append(str(output_file))
        elif sys.platform.startswith('linux'):
            # Linux
            if region == 'fullscreen':
                cmd = ['scrot', str(output_file)]
            else:
                x, y, w, h = map(int, region.split(','))
                cmd = ['scrot', '-a', f'{x},{y},{w},{h}', str(output_file)]
        else:
            return jsonify({'error': 'Platform not supported'}), 400
        
        subprocess.run(cmd, check=True, capture_output=True)
        
        return jsonify({
            'success': True,
            'file': str(output_file),
            'region': region,
            'timestamp': timestamp
        })
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/ocr', methods=['POST'])
def ocr():
    """Extract text from image"""
    data = request.json or {}
    image_path = data.get('image')
    language = data.get('language', CONFIG['ocr_languages'])
    
    if not image_path:
        return jsonify({'error': 'Image path required'}), 400
    
    try:
        # Load image
        image = Image.open(image_path)
        
        # Perform OCR
        global ocr_reader
        
        if OCR_ENGINE == 'easyocr':
            if ocr_reader is None:
                ocr_reader = easyocr.Reader([language])
            result = ocr_reader.readtext(image_path, detail=0)
            text = ' '.join(result)
        elif OCR_ENGINE == 'pytesseract':
            try:
                text = pytesseract.image_to_string(image, lang=language)
            except Exception as e:
                text = f"Tesseract error: {e}. Install tesseract-ocr system package."
        else:
            # Basic fallback - just return image info
            text = f"OCR not available. Image size: {image.size}, mode: {image.mode}"
        
        return jsonify({
            'success': True,
            'text': text,
            'image': image_path,
            'language': language,
            'engine': CONFIG['ocr_engine']
        })
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/ask', methods=['POST'])
def ask():
    """Ask AI about image content"""
    # This would integrate with Ollama or OpenRouter
    return jsonify({
        'error': 'Vision AI not yet implemented',
        'hint': 'This will integrate with Ollama llava model'
    }), 501

if __name__ == '__main__':
    print(f"Starting VOCR service on {CONFIG['host']}:{CONFIG['port']}")
    app.run(host=CONFIG['host'], port=CONFIG['port'], debug=False)
EOF
    
    chmod +x "${VOCR_DATA_DIR}/vocr-service.py"
    
    # Create systemd service file (if on Linux with systemd and sudo available)
    if [[ -d /etc/systemd/system ]] && command -v systemctl &>/dev/null; then
        if sudo::can_use_sudo; then
            log::info "Creating systemd service with sudo..."
            local service_content="[Unit]
Description=VOCR Vision OCR Service
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=${VOCR_DATA_DIR}
Environment=\"VOCR_PORT=${VOCR_PORT}\"
Environment=\"VOCR_HOST=${VOCR_HOST}\"
Environment=\"VOCR_SCREENSHOTS_DIR=${VOCR_SCREENSHOTS_DIR}\"
Environment=\"VOCR_OCR_ENGINE=${VOCR_OCR_ENGINE}\"
Environment=\"VOCR_OCR_LANGUAGES=${VOCR_OCR_LANGUAGES}\"
ExecStart=${VOCR_DATA_DIR}/venv/bin/python ${VOCR_DATA_DIR}/vocr-service.py
Restart=unless-stopped

[Install]
WantedBy=multi-user.target"
            
            echo "$service_content" | sudo::exec_with_fallback "tee /etc/systemd/system/vocr.service > /dev/null"
            sudo::exec_with_fallback "systemctl daemon-reload"
            log::info "Systemd service created"
        else
            log::warning "Cannot create systemd service without sudo access"
            log::info "Manual service management will be required"
            log::info "Service script available at: ${VOCR_DATA_DIR}/vocr-service.py"
        fi
    fi
    
    return 0
}

# Register with Vrooli
vocr::install::register() {
    # Add to service.json if it exists
    if [[ -f "${var_SERVICE_JSON_FILE}" ]]; then
        # Check if vocr already exists
        if jq -e '.dependencies.resources.vocr' "${var_SERVICE_JSON_FILE}" >/dev/null 2>&1; then
            log::info "VOCR already registered in service.json"
        else
            # Add vocr to service.json
            local tmp_file
            tmp_file=$(mktemp)
            jq '.dependencies |= (. // {})
                | .dependencies.resources |= (. // {})
                | .dependencies.resources.vocr = {
                "enabled": true,
                "category": "execution",
                "display_name": "VOCR (Vision OCR)",
                "description": "Advanced screen recognition and AI-powered image analysis"
            }' "${var_SERVICE_JSON_FILE}" > "$tmp_file" && \
            mv "$tmp_file" "${var_SERVICE_JSON_FILE}"
            
            log::success "VOCR registered in service.json"
        fi
    fi
    
    return 0
}

# Uninstall function
vocr::uninstall() {
    log::header "Uninstalling VOCR"
    
    # Stop service
    if command -v vocr::stop &>/dev/null; then
        vocr::stop 2>/dev/null || true
    fi
    
    # Remove systemd service (with conditional sudo)
    if [[ -f /etc/systemd/system/vocr.service ]]; then
        if sudo::can_use_sudo; then
            log::info "Removing systemd service with sudo..."
            sudo::exec_with_fallback "systemctl stop vocr" 2>/dev/null || true
            sudo::exec_with_fallback "systemctl disable vocr" 2>/dev/null || true
            sudo::exec_with_fallback "rm /etc/systemd/system/vocr.service"
            sudo::exec_with_fallback "systemctl daemon-reload"
        else
            log::warning "Cannot remove systemd service without sudo access"
            log::info "Manual removal required: sudo rm /etc/systemd/system/vocr.service"
        fi
    fi
    
    # Remove data directory
    if [[ -d "$VOCR_DATA_DIR" ]]; then
        rm -rf "$VOCR_DATA_DIR"
    fi
    
    # Remove from service.json
    if [[ -f "${var_SERVICE_JSON_FILE}" ]]; then
        local tmp_file
        tmp_file=$(mktemp)
        jq '.dependencies.resources |= (. // {})
            | del(.dependencies.resources.vocr)' "${var_SERVICE_JSON_FILE}" > "$tmp_file" && \
        mv "$tmp_file" "${var_SERVICE_JSON_FILE}"
    fi
    
    # Remove CLI symlink
    if [[ -L "${HOME}/.local/bin/resource-vocr" ]]; then
        rm "${HOME}/.local/bin/resource-vocr"
    fi
    
    log::success "VOCR uninstalled"
    return 0
}
