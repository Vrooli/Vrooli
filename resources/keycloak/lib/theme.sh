#!/usr/bin/env bash
################################################################################
# Keycloak Theme Customization Functions
################################################################################

set -euo pipefail

# Determine script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/common.sh"

# Get Keycloak port from registry
KEYCLOAK_PORT=$(source "${APP_ROOT}/scripts/resources/port_registry.sh" && port_registry::get keycloak)

# Theme directory configuration
THEME_DIR="${RESOURCE_DIR}/themes"
CONTAINER_THEME_DIR="/opt/keycloak/themes"

################################################################################
# Theme Management Functions
################################################################################

theme::init() {
    # Create theme directory if it doesn't exist
    if [[ ! -d "$THEME_DIR" ]]; then
        mkdir -p "$THEME_DIR"
        log::info "Created theme directory: $THEME_DIR"
    fi
}

theme::create() {
    local theme_name="${1:-}"
    local base_theme="${2:-keycloak}"
    
    if [[ -z "$theme_name" ]]; then
        log::error "Theme name is required"
        return 1
    fi
    
    theme::init
    
    local theme_path="${THEME_DIR}/${theme_name}"
    
    if [[ -d "$theme_path" ]]; then
        log::warning "Theme already exists: $theme_name"
        return 1
    fi
    
    log::info "Creating theme: $theme_name (based on: $base_theme)"
    
    # Create theme directory structure
    mkdir -p "$theme_path/login"
    mkdir -p "$theme_path/account"
    mkdir -p "$theme_path/admin"
    mkdir -p "$theme_path/email"
    mkdir -p "$theme_path/common/resources/css"
    mkdir -p "$theme_path/common/resources/img"
    
    # Create theme.properties for login theme
    cat > "$theme_path/login/theme.properties" <<EOF
parent=$base_theme
import=common/keycloak

# Custom theme properties
styles=css/login.css

# Branding
displayName=$theme_name Login Theme
displayNameHtml=$theme_name

# Colors and styling can be customized here
EOF
    
    # Create basic custom CSS
    cat > "$theme_path/common/resources/css/login.css" <<'EOF'
/* Custom theme styles */
.login-pf body {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.card-pf {
    border-radius: 10px;
    box-shadow: 0 10px 30px rgba(0,0,0,0.1);
}

.btn-primary {
    background: #667eea;
    border-color: #667eea;
    border-radius: 5px;
    transition: all 0.3s ease;
}

.btn-primary:hover {
    background: #5a67d8;
    border-color: #5a67d8;
    transform: translateY(-2px);
    box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
}

/* Custom logo placeholder */
.login-pf-page-header {
    font-size: 24px;
    font-weight: bold;
    color: #fff;
    text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
}
EOF
    
    # Create account theme properties
    cat > "$theme_path/account/theme.properties" <<EOF
parent=$base_theme
import=common/keycloak
EOF
    
    # Create email theme properties
    cat > "$theme_path/email/theme.properties" <<EOF
parent=$base_theme
import=common/keycloak
EOF
    
    log::success "Theme created: $theme_name"
    log::info "Theme location: $theme_path"
    
    # Copy theme to container if running
    if docker ps --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$"; then
        theme::deploy "$theme_name"
    fi
    
    return 0
}

theme::list() {
    theme::init
    
    log::info "Available custom themes:"
    
    local count=0
    if [[ -d "$THEME_DIR" ]]; then
        while IFS= read -r theme_dir; do
            if [[ -n "$theme_dir" && -d "$theme_dir" ]]; then
                local theme_name=$(basename "$theme_dir")
                local size=$(du -sh "$theme_dir" | cut -f1)
                
                log::info "  🎨 $theme_name"
                log::info "     Size: $size"
                
                # Check if deployed to container
                if docker ps --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$"; then
                    if docker exec "${KEYCLOAK_CONTAINER_NAME}" test -d "${CONTAINER_THEME_DIR}/${theme_name}" 2>/dev/null; then
                        log::info "     Status: Deployed ✓"
                    else
                        log::info "     Status: Not deployed"
                    fi
                fi
                
                ((count++))
            fi
        done < <(find "$THEME_DIR" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)
        
        if [[ $count -eq 0 ]]; then
            log::warning "No custom themes found"
        else
            log::success "Found $count theme(s)"
        fi
    else
        log::warning "Theme directory does not exist"
    fi
}

theme::deploy() {
    local theme_name="${1:-}"
    
    if [[ -z "$theme_name" ]]; then
        log::error "Theme name is required"
        return 1
    fi
    
    local theme_path="${THEME_DIR}/${theme_name}"
    
    if [[ ! -d "$theme_path" ]]; then
        log::error "Theme not found: $theme_name"
        return 1
    fi
    
    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$"; then
        log::error "Keycloak container is not running"
        return 1
    fi
    
    log::info "Deploying theme: $theme_name"
    
    # Copy theme to container
    docker cp "$theme_path" "${KEYCLOAK_CONTAINER_NAME}:${CONTAINER_THEME_DIR}/" || {
        log::error "Failed to copy theme to container"
        return 1
    }
    
    log::success "Theme deployed: $theme_name"
    log::info "Theme is now available in Keycloak admin console"
    
    return 0
}

theme::apply() {
    local realm="${1:-}"
    local theme_name="${2:-}"
    local theme_type="${3:-login}"  # login, account, admin, or email
    
    if [[ -z "$realm" || -z "$theme_name" ]]; then
        log::error "Realm and theme name are required"
        return 1
    fi
    
    log::info "Applying theme '$theme_name' to realm '$realm' ($theme_type)"
    
    # Get admin token
    local access_token
    access_token=$(keycloak::get_admin_token) || {
        log::error "Failed to get admin token"
        return 1
    }
    
    # Get current realm configuration
    local realm_config
    realm_config=$(timeout 5 curl -sf \
        -H "Authorization: Bearer $access_token" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}") || {
        log::error "Failed to get realm configuration - checking authentication"
        # Try to re-authenticate
        access_token=$(keycloak::get_admin_token --force) || {
            log::error "Authentication failed"
            return 1
        }
        # Retry with new token
        realm_config=$(timeout 5 curl -sf \
            -H "Authorization: Bearer $access_token" \
            "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}") || {
            log::error "Failed to get realm configuration"
            return 1
        }
    }
    
    # Update theme in configuration
    local theme_field="${theme_type}Theme"
    realm_config=$(echo "$realm_config" | jq ".${theme_field} = \"${theme_name}\"")
    
    # Update realm with new theme
    if curl -sf \
        -X PUT \
        -H "Authorization: Bearer $access_token" \
        -H "Content-Type: application/json" \
        -d "$realm_config" \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}"; then
        log::success "Theme applied successfully"
        log::info "Realm '$realm' now uses '$theme_name' for $theme_type"
    else
        log::error "Failed to apply theme"
        return 1
    fi
    
    return 0
}

theme::remove() {
    local theme_name="${1:-}"
    
    if [[ -z "$theme_name" ]]; then
        log::error "Theme name is required"
        return 1
    fi
    
    local theme_path="${THEME_DIR}/${theme_name}"
    
    if [[ ! -d "$theme_path" ]]; then
        log::error "Theme not found: $theme_name"
        return 1
    fi
    
    log::warning "Removing theme: $theme_name"
    
    # Remove from container if deployed
    if docker ps --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$"; then
        if docker exec "${KEYCLOAK_CONTAINER_NAME}" test -d "${CONTAINER_THEME_DIR}/${theme_name}" 2>/dev/null; then
            docker exec "${KEYCLOAK_CONTAINER_NAME}" rm -rf "${CONTAINER_THEME_DIR}/${theme_name}" || {
                log::warning "Failed to remove theme from container"
            }
        fi
    fi
    
    # Remove local theme directory
    rm -rf "$theme_path"
    
    log::success "Theme removed: $theme_name"
    return 0
}

theme::customize() {
    local theme_name="${1:-}"
    local property="${2:-}"
    local value="${3:-}"
    
    if [[ -z "$theme_name" || -z "$property" || -z "$value" ]]; then
        log::error "Theme name, property, and value are required"
        log::info "Usage: theme customize <theme-name> <property> <value>"
        log::info "Properties: logo, primary-color, background-color, font-family"
        return 1
    fi
    
    local theme_path="${THEME_DIR}/${theme_name}"
    
    if [[ ! -d "$theme_path" ]]; then
        log::error "Theme not found: $theme_name"
        return 1
    fi
    
    case "$property" in
        logo)
            # Handle logo upload
            if [[ -f "$value" ]]; then
                cp "$value" "$theme_path/common/resources/img/logo.png"
                # Update CSS to use custom logo
                echo ".login-pf-brand { background-image: url('../img/logo.png'); }" >> "$theme_path/common/resources/css/login.css"
                log::success "Logo updated for theme: $theme_name"
            else
                log::error "Logo file not found: $value"
                return 1
            fi
            ;;
            
        primary-color)
            # Update primary color in CSS
            sed -i "s/background: #[0-9a-fA-F]\{6\}/background: $value/g" "$theme_path/common/resources/css/login.css" 2>/dev/null || {
                echo ".btn-primary { background: $value !important; }" >> "$theme_path/common/resources/css/login.css"
            }
            log::success "Primary color updated to: $value"
            ;;
            
        background-color)
            # Update background color
            sed -i "s/background: linear-gradient.*/background: $value;/g" "$theme_path/common/resources/css/login.css" 2>/dev/null || {
                echo ".login-pf body { background: $value !important; }" >> "$theme_path/common/resources/css/login.css"
            }
            log::success "Background color updated to: $value"
            ;;
            
        font-family)
            # Update font family
            echo "* { font-family: $value !important; }" >> "$theme_path/common/resources/css/login.css"
            log::success "Font family updated to: $value"
            ;;
            
        *)
            log::error "Unknown property: $property"
            log::info "Valid properties: logo, primary-color, background-color, font-family"
            return 1
            ;;
    esac
    
    # Redeploy theme if container is running
    if docker ps --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$"; then
        theme::deploy "$theme_name"
    fi
    
    return 0
}

theme::export() {
    local theme_name="${1:-}"
    local output_file="${2:-}"
    
    if [[ -z "$theme_name" ]]; then
        log::error "Theme name is required"
        return 1
    fi
    
    local theme_path="${THEME_DIR}/${theme_name}"
    
    if [[ ! -d "$theme_path" ]]; then
        log::error "Theme not found: $theme_name"
        return 1
    fi
    
    if [[ -z "$output_file" ]]; then
        output_file="/tmp/${theme_name}-theme-$(date +%Y%m%d_%H%M%S).tar.gz"
    fi
    
    log::info "Exporting theme: $theme_name"
    
    # Create tar archive of theme
    tar -czf "$output_file" -C "$THEME_DIR" "$theme_name" || {
        log::error "Failed to export theme"
        return 1
    }
    
    log::success "Theme exported to: $output_file"
    echo "$output_file"
    
    return 0
}

theme::import() {
    local archive_file="${1:-}"
    
    if [[ -z "$archive_file" ]]; then
        log::error "Archive file is required"
        return 1
    fi
    
    if [[ ! -f "$archive_file" ]]; then
        log::error "Archive file not found: $archive_file"
        return 1
    fi
    
    theme::init
    
    log::info "Importing theme from: $archive_file"
    
    # Extract theme archive
    tar -xzf "$archive_file" -C "$THEME_DIR" || {
        log::error "Failed to import theme"
        return 1
    }
    
    # Get theme name from archive
    local theme_name=$(tar -tzf "$archive_file" | head -1 | cut -d'/' -f1)
    
    log::success "Theme imported: $theme_name"
    
    # Deploy if container is running
    if docker ps --format "{{.Names}}" | grep -q "^${KEYCLOAK_CONTAINER_NAME}$"; then
        theme::deploy "$theme_name"
    fi
    
    return 0
}