#!/bin/bash
# KiCad Desktop Application Management Functions

# Desktop app-specific handlers (not Docker-based)
kicad::desktop::uninstall() {
    log::info "Note: KiCad is a system package. Uninstallation requires manual action."
    echo "To uninstall KiCad, run: sudo apt-get remove --purge kicad kicad-*"
    echo "User data remains at: ${KICAD_DATA_DIR:-$HOME/.local/share/kicad}"
}

kicad::desktop::start() {
    if kicad::is_installed; then
        log::success "KiCad is installed and ready to use"
        echo "Launch KiCad with: kicad"
    else
        log::error "KiCad is not installed. Run: resource-kicad manage install"
        return 1
    fi
}

kicad::desktop::stop() {
    log::info "KiCad is a desktop application - close it from the GUI"
}

kicad::desktop::restart() {
    log::info "KiCad is a desktop application - restart it from the GUI"
}

kicad::desktop::logs() {
    local log_file="${KICAD_DATA_DIR:-$HOME/.local/share/kicad}/kicad.log"
    if [[ -f "$log_file" ]]; then
        tail -n 50 "$log_file"
    else
        log::info "No KiCad logs found at $log_file"
    fi
}