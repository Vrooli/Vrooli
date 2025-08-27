#!/usr/bin/env bash
################################################################################
# Content operations for Odoo resource
# Business functionality for ERP operations
################################################################################

# Source common functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/resources/odoo/lib/common.sh"

# Add content (modules, data) to Odoo
odoo::content::add() {
    local content_type="${1:-module}"
    local content_path="${2:-}"
    
    if [[ -z "$content_path" ]]; then
        log::error "Content path required"
        log::info "Usage: odoo content add <module|data|addon> <path>"
        return 1
    fi
    
    if ! odoo_is_running; then
        log::error "Odoo must be running to add content"
        return 1
    fi
    
    case "$content_type" in
        module)
            log::info "Installing Odoo module: $content_path"
            odoo_install_module "$content_path"
            ;;
        data)
            log::info "Importing data from: $content_path"
            odoo_import_data "$content_path"
            ;;
        addon)
            log::info "Adding addon from: $content_path"
            if [[ -d "$content_path" ]]; then
                cp -r "$content_path" "$ODOO_DATA_DIR/addons/"
                log::success "Addon copied to addons directory"
            else
                log::error "Addon path must be a directory"
                return 1
            fi
            ;;
        *)
            log::error "Unknown content type: $content_type"
            log::info "Supported types: module, data, addon"
            return 1
            ;;
    esac
}

# List content in Odoo
odoo::content::list() {
    local content_type="${1:-modules}"
    
    if ! odoo_is_running; then
        log::error "Odoo must be running to list content"
        return 1
    fi
    
    case "$content_type" in
        modules)
            log::info "Listing installed Odoo modules..."
            docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<'EOF'
modules = env['ir.module.module'].search([('state', '=', 'installed')])
for module in modules:
    print(f"{module.name}: {module.summary or 'No description'}")
EOF
            ;;
        databases)
            log::info "Listing Odoo databases..."
            docker exec "$ODOO_PG_CONTAINER_NAME" psql -U "$ODOO_DB_USER" -c "SELECT datname FROM pg_database WHERE datname NOT IN ('template0', 'template1', 'postgres');"
            ;;
        users)
            log::info "Listing Odoo users..."
            docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<'EOF'
users = env['res.users'].search([])
for user in users:
    print(f"{user.login}: {user.name} ({user.email or 'no email'})")
EOF
            ;;
        companies)
            log::info "Listing companies..."
            docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<'EOF'
companies = env['res.company'].search([])
for company in companies:
    print(f"{company.name}: {company.email or 'no email'}")
EOF
            ;;
        *)
            log::error "Unknown content type: $content_type"
            log::info "Supported types: modules, databases, users, companies"
            return 1
            ;;
    esac
}

# Get specific content from Odoo
odoo::content::get() {
    local content_type="${1:-}"
    local identifier="${2:-}"
    
    if [[ -z "$content_type" || -z "$identifier" ]]; then
        log::error "Content type and identifier required"
        log::info "Usage: odoo content get <module|user|company> <name/id>"
        return 1
    fi
    
    if ! odoo_is_running; then
        log::error "Odoo must be running to get content"
        return 1
    fi
    
    case "$content_type" in
        module)
            log::info "Getting module info: $identifier"
            docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<EOF
module = env['ir.module.module'].search([('name', '=', '$identifier')], limit=1)
if module:
    print(f"Name: {module.name}")
    print(f"State: {module.state}")
    print(f"Summary: {module.summary or 'No description'}")
    print(f"Version: {module.latest_version}")
else:
    print("Module not found")
EOF
            ;;
        user)
            log::info "Getting user info: $identifier"
            docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<EOF
user = env['res.users'].search(['|', ('login', '=', '$identifier'), ('name', 'ilike', '$identifier')], limit=1)
if user:
    print(f"Login: {user.login}")
    print(f"Name: {user.name}")
    print(f"Email: {user.email or 'No email'}")
    print(f"Active: {user.active}")
    print(f"Groups: {', '.join([g.name for g in user.groups_id])}")
else:
    print("User not found")
EOF
            ;;
        company)
            log::info "Getting company info: $identifier"
            docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<EOF
company = env['res.company'].search([('name', 'ilike', '$identifier')], limit=1)
if company:
    print(f"Name: {company.name}")
    print(f"Email: {company.email or 'No email'}")
    print(f"Website: {company.website or 'No website'}")
    print(f"Currency: {company.currency_id.name}")
else:
    print("Company not found")
EOF
            ;;
        *)
            log::error "Unknown content type: $content_type"
            log::info "Supported types: module, user, company"
            return 1
            ;;
    esac
}

# Remove content from Odoo
odoo::content::remove() {
    local content_type="${1:-}"
    local identifier="${2:-}"
    
    if [[ -z "$content_type" || -z "$identifier" ]]; then
        log::error "Content type and identifier required"
        log::info "Usage: odoo content remove <module|user> <name/id>"
        return 1
    fi
    
    if ! odoo_is_running; then
        log::error "Odoo must be running to remove content"
        return 1
    fi
    
    case "$content_type" in
        module)
            log::info "Uninstalling module: $identifier"
            docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<EOF
module = env['ir.module.module'].search([('name', '=', '$identifier')], limit=1)
if module and module.state == 'installed':
    module.button_immediate_uninstall()
    env.cr.commit()
    print(f"Module {identifier} uninstalled")
else:
    print(f"Module {identifier} not found or not installed")
EOF
            ;;
        user)
            log::warn "WARNING: This will permanently remove the user!"
            read -p "Are you sure you want to remove user '$identifier'? [y/N]: " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                log::info "Removing user: $identifier"
                docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<EOF
user = env['res.users'].search(['|', ('login', '=', '$identifier'), ('name', 'ilike', '$identifier')], limit=1)
if user and user.login != 'admin':
    user.unlink()
    env.cr.commit()
    print(f"User {identifier} removed")
else:
    print(f"User {identifier} not found or is admin (cannot remove admin)")
EOF
            else
                log::info "User removal cancelled"
            fi
            ;;
        *)
            log::error "Unknown content type: $content_type"
            log::info "Supported types: module, user"
            return 1
            ;;
    esac
}

# Execute business operations in Odoo
odoo::content::execute() {
    local operation="${1:-}"
    shift || true
    
    if [[ -z "$operation" ]]; then
        log::error "Operation required"
        log::info "Usage: odoo content execute <backup|script|shell> [args]"
        return 1
    fi
    
    if ! odoo_is_running; then
        log::error "Odoo must be running to execute operations"
        return 1
    fi
    
    case "$operation" in
        backup)
            local backup_name="${1:-odoo_backup_$(date +%Y%m%d_%H%M%S)}"
            log::info "Creating database backup: $backup_name"
            docker exec "$ODOO_PG_CONTAINER_NAME" pg_dump -U "$ODOO_DB_USER" "$ODOO_DB_NAME" > "$ODOO_DATA_DIR/$backup_name.sql"
            log::success "Backup saved to: $ODOO_DATA_DIR/$backup_name.sql"
            ;;
        script)
            local script_file="${1:-}"
            if [[ -z "$script_file" ]]; then
                log::error "Script file required"
                return 1
            fi
            odoo_run_script "$script_file"
            ;;
        shell)
            log::info "Starting Odoo shell (interactive mode)..."
            docker exec -it "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME"
            ;;
        *)
            log::error "Unknown operation: $operation"
            log::info "Supported operations: backup, script, shell"
            return 1
            ;;
    esac
}

# Export functions
export -f odoo::content::add
export -f odoo::content::list
export -f odoo::content::get
export -f odoo::content::remove
export -f odoo::content::execute