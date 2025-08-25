#!/usr/bin/env bash
# VOCR Screen Capture Module

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
VOCR_CAPTURE_DIR="${APP_ROOT}/resources/vocr/lib"

# Source utilities
# shellcheck disable=SC1091
source "${VOCR_CAPTURE_DIR}/../../../../lib/utils/log.sh"
# shellcheck disable=SC1091
source "${VOCR_CAPTURE_DIR}/../../../../lib/utils/var.sh"

# Source configuration
# shellcheck disable=SC1091
source "${VOCR_CAPTURE_DIR}/../config/defaults.sh"

# Capture screen region
vocr::capture::screen() {
    local region="${1:-}"
    local output="${2:-}"
    
    # Default to full screen if no region specified
    if [[ -z "$region" ]]; then
        region="full"
    fi
    
    # Generate default output filename if not specified
    if [[ -z "$output" ]]; then
        output="${VOCR_SCREENSHOTS_DIR}/capture_$(date +%s).png"
    fi
    
    # Ensure output directory exists
    local output_dir
    output_dir=${output%/*
    mkdir -p "$output_dir"
    
    # Platform-specific capture
    case "$(uname -s)" in
        Darwin*)
            # macOS using screencapture
            if [[ "$region" == "full" ]]; then
                screencapture "$output"
            else
                # Parse region format: x,y,width,height
                IFS=',' read -r x y width height <<< "$region"
                screencapture -R"$x,$y,$width,$height" "$output"
            fi
            ;;
            
        Linux*)
            # Linux using scrot, import, or Python fallback
            local capture_cmd=""
            local fallback_script="/home/matthalloran8/Vrooli/data/vocr/scrot-fallback"
            
            # Check for available capture tools
            if command -v scrot &>/dev/null; then
                capture_cmd="scrot"
            elif command -v import &>/dev/null; then
                capture_cmd="import"
            elif [[ -x "$fallback_script" ]]; then
                capture_cmd="$fallback_script"
            else
                log::error "No screen capture tool found (install scrot or imagemagick, or ensure Python fallback is available)"
                return 1
            fi
            
            # Perform capture based on available tool
            case "$capture_cmd" in
                scrot)
                    if [[ "$region" == "full" ]]; then
                        scrot "$output"
                    else
                        # scrot doesn't support region directly, use import if available
                        if command -v import &>/dev/null; then
                            # Parse region format: x,y,width,height
                            IFS=',' read -r x y width height <<< "$region"
                            import -window root -crop "${width}x${height}+${x}+${y}" "$output"
                        else
                            # Fallback to full screen with scrot
                            log::warning "Region capture not available, capturing full screen"
                            scrot "$output"
                        fi
                    fi
                    ;;
                import)
                    # Use ImageMagick's import
                    if [[ "$region" == "full" ]]; then
                        import -window root "$output"
                    else
                        # Parse region format: x,y,width,height
                        IFS=',' read -r x y width height <<< "$region"
                        import -window root -crop "${width}x${height}+${x}+${y}" "$output"
                    fi
                    ;;
                "$fallback_script")
                    # Use Python fallback (full screen only for now)
                    if [[ "$region" != "full" ]]; then
                        log::warning "Region capture not available with Python fallback, capturing full screen"
                    fi
                    "$fallback_script" "$output"
                    ;;
            esac
            ;;
            
        *)
            log::error "Platform not supported for screen capture"
            return 1
            ;;
    esac
    
    # Verify capture succeeded
    if [[ -f "$output" ]]; then
        log::success "Screen captured to: $output"
        echo "$output"
        return 0
    else
        log::error "Screen capture failed"
        return 1
    fi
}

# OCR text extraction from image
vocr::ocr::extract() {
    local input="${1:-}"
    local language="${2:-eng}"
    
    if [[ -z "$input" ]]; then
        log::error "Input image required"
        return 1
    fi
    
    if [[ ! -f "$input" ]]; then
        log::error "Input file not found: $input"
        return 1
    fi
    
    # Check if tesseract is available
    if command -v tesseract &>/dev/null; then
        # Use tesseract directly
        tesseract "$input" stdout -l "$language" 2>/dev/null
    elif [[ -f "${VOCR_DATA_DIR}/venv/bin/python" ]]; then
        # Use Python with pytesseract
        "${VOCR_DATA_DIR}/venv/bin/python" -c "
import sys
try:
    from PIL import Image
    import pytesseract
    img = Image.open('$input')
    text = pytesseract.image_to_string(img, lang='$language')
    print(text)
except Exception as e:
    print(f'OCR failed: {e}', file=sys.stderr)
    sys.exit(1)
"
    else
        log::error "No OCR engine available"
        return 1
    fi
}

# Monitor screen region for changes
vocr::monitor::region() {
    local region="${1:-}"
    local interval="${2:-5}"
    local callback="${3:-}"
    
    if [[ -z "$region" ]]; then
        log::error "Region required for monitoring"
        return 1
    fi
    
    log::info "Monitoring region: $region (interval: ${interval}s)"
    
    local last_hash=""
    local current_hash
    local temp_file="${VOCR_SCREENSHOTS_DIR}/monitor_temp.png"
    
    while true; do
        # Capture current state
        if vocr::capture::screen "$region" "$temp_file" >/dev/null 2>&1; then
            # Calculate hash of the image
            if command -v md5sum &>/dev/null; then
                current_hash=$(md5sum "$temp_file" | cut -d' ' -f1)
            elif command -v md5 &>/dev/null; then
                current_hash=$(md5 -q "$temp_file")
            else
                log::warning "No hash utility found, skipping change detection"
                current_hash="$last_hash"
            fi
            
            # Check for changes
            if [[ "$current_hash" != "$last_hash" ]]; then
                log::info "Change detected in region"
                
                # Execute callback if provided
                if [[ -n "$callback" ]]; then
                    eval "$callback" "$temp_file"
                fi
                
                last_hash="$current_hash"
            fi
        else
            log::warning "Failed to capture screen"
        fi
        
        sleep "$interval"
    done
}

# Export functions
export -f vocr::capture::screen
export -f vocr::ocr::extract
export -f vocr::monitor::region