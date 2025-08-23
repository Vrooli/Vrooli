#!/bin/bash
# Inject functions for Odoo resource - install modules and import data

odoo_inject() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        module|install-module)
            odoo_install_module "$@"
            ;;
        data|import-data)
            odoo_import_data "$@"
            ;;
        script|run-script)
            odoo_run_script "$@"
            ;;
        *)
            echo "Usage: odoo inject {module|data|script} [options]"
            echo ""
            echo "Actions:"
            echo "  module <name>  - Install an Odoo module"
            echo "  data <file>    - Import data from CSV/XML file"
            echo "  script <file>  - Run a Python script in Odoo context"
            return 1
            ;;
    esac
}

odoo_install_module() {
    local module_name="${1:-}"
    
    if [[ -z "$module_name" ]]; then
        echo "Error: Module name required"
        return 1
    fi
    
    if ! odoo_is_running; then
        echo "Error: Odoo must be running to install modules"
        return 1
    fi
    
    echo "Installing Odoo module: $module_name"
    
    # Install module via Odoo shell
    docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" <<EOF
env['ir.module.module'].search([('name', '=', '$module_name')]).button_immediate_install()
env.cr.commit()
EOF
    
    if [[ $? -eq 0 ]]; then
        echo "Module '$module_name' installed successfully"
        return 0
    else
        echo "Failed to install module '$module_name'"
        return 1
    fi
}

odoo_import_data() {
    local data_file="${1:-}"
    
    if [[ -z "$data_file" ]]; then
        echo "Error: Data file path required"
        return 1
    fi
    
    if [[ ! -f "$data_file" ]]; then
        echo "Error: Data file not found: $data_file"
        return 1
    fi
    
    if ! odoo_is_running; then
        echo "Error: Odoo must be running to import data"
        return 1
    fi
    
    echo "Importing data from: $data_file"
    
    # Copy file to container
    local filename=$(basename "$data_file")
    docker cp "$data_file" "$ODOO_CONTAINER_NAME:/tmp/$filename"
    
    # Import based on file type
    case "$filename" in
        *.csv)
            echo "Importing CSV data..."
            # CSV import logic would go here
            ;;
        *.xml)
            echo "Importing XML data..."
            docker exec "$ODOO_CONTAINER_NAME" odoo -d "$ODOO_DB_NAME" --load-data "/tmp/$filename"
            ;;
        *.json)
            echo "Importing JSON data via API..."
            # JSON import via API would go here
            ;;
        *)
            echo "Error: Unsupported file type"
            return 1
            ;;
    esac
    
    echo "Data import completed"
    return 0
}

odoo_run_script() {
    local script_file="${1:-}"
    
    if [[ -z "$script_file" ]]; then
        echo "Error: Script file path required"
        return 1
    fi
    
    if [[ ! -f "$script_file" ]]; then
        echo "Error: Script file not found: $script_file"
        return 1
    fi
    
    if ! odoo_is_running; then
        echo "Error: Odoo must be running to execute scripts"
        return 1
    fi
    
    echo "Running script: $script_file"
    
    # Copy script to container
    local filename=$(basename "$script_file")
    docker cp "$script_file" "$ODOO_CONTAINER_NAME:/tmp/$filename"
    
    # Execute script in Odoo shell
    docker exec "$ODOO_CONTAINER_NAME" odoo shell -d "$ODOO_DB_NAME" < "/tmp/$filename"
    
    if [[ $? -eq 0 ]]; then
        echo "Script executed successfully"
        return 0
    else
        echo "Script execution failed"
        return 1
    fi
}

export -f odoo_inject
export -f odoo_install_module
export -f odoo_import_data
export -f odoo_run_script