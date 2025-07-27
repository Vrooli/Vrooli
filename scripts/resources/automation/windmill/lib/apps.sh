#!/usr/bin/env bash
# Windmill App Management Functions
# Functions for managing and preparing Windmill app definitions

#######################################
# List available app examples
#######################################
windmill::list_apps() {
    log::header "📱 Available Windmill App Examples"
    
    local apps_dir="${SCRIPT_DIR}/examples/apps"
    
    if [[ ! -d "$apps_dir" ]]; then
        log::error "Apps directory not found: $apps_dir"
        return 1
    fi
    
    echo
    echo "The following app examples are available:"
    echo
    
    # List all JSON files in the apps directory
    local app_count=0
    for app_file in "${apps_dir}"/*.json; do
        if [[ -f "$app_file" ]]; then
            local app_name=$(basename "$app_file" .json)
            local app_desc=""
            
            # Try to extract description from JSON
            if command -v jq >/dev/null 2>&1; then
                app_desc=$(jq -r '.description // ""' "$app_file" 2>/dev/null || echo "")
            fi
            
            ((app_count++))
            printf "  %d. %s\n" "$app_count" "$app_name"
            if [[ -n "$app_desc" ]]; then
                printf "     └─ %s\n" "$app_desc"
            fi
            printf "\n"
        fi
    done
    
    if [[ $app_count -eq 0 ]]; then
        log::warn "No app examples found"
        return 1
    fi
    
    echo "To prepare an app for import:"
    echo "  $0 --action prepare-app --app-name <app-name>"
    echo
    echo "Note: Apps must be imported manually through the Windmill UI"
    echo "      as there is no programmatic API for app creation yet."
}

#######################################
# Prepare an app for manual import
#######################################
windmill::prepare_app() {
    local app_name="$1"
    local output_dir="$2"
    
    if [[ -z "$app_name" ]]; then
        log::error "App name is required"
        echo
        echo "Usage: $0 --action prepare-app --app-name <app-name>"
        echo
        windmill::list_apps
        return 1
    fi
    
    local apps_dir="${SCRIPT_DIR}/examples/apps"
    local app_file="${apps_dir}/${app_name}.json"
    
    if [[ ! -f "$app_file" ]]; then
        log::error "App not found: $app_name"
        echo
        echo "Available apps:"
        for f in "${apps_dir}"/*.json; do
            if [[ -f "$f" ]]; then
                echo "  - $(basename "$f" .json)"
            fi
        done
        return 1
    fi
    
    log::header "📦 Preparing Windmill App: $app_name"
    
    # Create output directory
    mkdir -p "$output_dir"
    
    # Copy app definition
    local output_file="${output_dir}/${app_name}.json"
    cp "$app_file" "$output_file"
    
    # Extract required scripts
    local scripts_file="${output_dir}/${app_name}-required-scripts.txt"
    if command -v jq >/dev/null 2>&1; then
        jq -r '.required_scripts[]? | "Path: \(.path)\nDescription: \(.description // "No description")\nLanguage: \(.language // "typescript")\n"' \
            "$app_file" > "$scripts_file" 2>/dev/null || true
    fi
    
    # Create instructions file
    local instructions_file="${output_dir}/${app_name}-instructions.md"
    cat > "$instructions_file" << EOF
# Windmill App Import Instructions: $app_name

## Overview
This directory contains the prepared files for importing the "$app_name" app into Windmill.

## Files
- **${app_name}.json**: The app definition file
- **${app_name}-required-scripts.txt**: List of scripts that need to be created first
- **${app_name}-instructions.md**: This file

## Import Process

### Step 1: Create Required Scripts
Before importing the app, create the following scripts in Windmill:

$(if [[ -f "$scripts_file" ]]; then cat "$scripts_file"; else echo "No required scripts listed"; fi)

### Step 2: Import the App

#### Option A: Manual Recreation (Recommended)
1. Access Windmill at http://localhost:5681
2. Navigate to Apps → New App
3. Use the JSON file as a reference to recreate the app structure:
   - Add components matching the layout structure
   - Configure data bindings using the expressions in the JSON
   - Set up event handlers as specified
   - Configure state management

#### Option B: Check for Import Feature
1. Check if Windmill has added an import feature since this was written
2. Look for "Import App" or similar option in the Apps section
3. If available, upload the JSON file directly

### Step 3: Configure the App
1. Update any workspace-specific paths in scripts
2. Configure required resources (API keys, databases, etc.)
3. Test all functionality before deploying

## App Structure Reference
The JSON file contains:
- **layout**: Component hierarchy and styling
- **state**: Initial state configuration
- **scripts**: Connected scripts and their triggers
- **modals**: Modal dialog definitions
- **features**: App capabilities and settings

## Testing
1. Test each component interaction
2. Verify data loading and updates
3. Check all forms and validations
4. Test error handling
5. Verify responsive design

## Notes
- Windmill currently doesn't have a direct API for programmatic app creation
- Apps must be created through the UI
- This preparation step helps organize the required components
- Check Windmill's documentation for updates on app import features

## Support
- Windmill Documentation: https://docs.windmill.dev
- GitHub: https://github.com/windmill-labs/windmill
EOF

    log::success "✅ App prepared successfully!"
    echo
    echo "Output directory: $output_dir"
    echo "Files created:"
    echo "  - $(basename "$output_file")"
    echo "  - $(basename "$scripts_file")"
    echo "  - $(basename "$instructions_file")"
    echo
    echo "Next steps:"
    echo "  1. Review the instructions in: $instructions_file"
    echo "  2. Create required scripts in Windmill"
    echo "  3. Import the app through Windmill UI"
    echo
    log::info "Note: Windmill doesn't yet support programmatic app creation."
    log::info "This tool prepares the files for manual import."
}

#######################################
# Check if Windmill has app API (future)
#######################################
windmill::check_app_api() {
    local api_token
    api_token=$(windmill::load_api_key)
    
    if [[ -z "$api_token" ]]; then
        log::warn "No API token configured"
        return 1
    fi
    
    log::info "Checking for app management API endpoints..."
    
    # Try to access potential app endpoints
    local endpoints=(
        "/api/apps/list"
        "/api/w/list/apps"
        "/api/apps"
    )
    
    for endpoint in "${endpoints[@]}"; do
        local response
        if response=$(curl -s -H "Authorization: Bearer $api_token" \
            "${WINDMILL_BASE_URL}${endpoint}" 2>/dev/null); then
            
            if [[ "$response" != *"not found"* ]] && [[ "$response" != *"404"* ]]; then
                log::info "Found potential app endpoint: $endpoint"
                echo "$response" | jq . 2>/dev/null || echo "$response"
            fi
        fi
    done
    
    log::info "Check Windmill documentation for app API updates"
}

#######################################
# Future: Deploy app via API
#######################################
windmill::deploy_app() {
    local app_name="$1"
    local workspace="$2"
    
    log::warn "App deployment via API is not yet available in Windmill"
    echo
    echo "This is a placeholder for future functionality."
    echo "Currently, apps must be created through the Windmill UI."
    echo
    echo "To track this feature request:"
    echo "  - https://github.com/windmill-labs/windmill/issues"
    echo
    echo "For now, use: $0 --action prepare-app --app-name $app_name"
    
    return 1
}

#######################################
# Create app deployment script (future)
#######################################
windmill::create_app_deployer() {
    local output_file="${OUTPUT_DIR}/windmill-app-deployer.js"
    
    log::header "🔧 Creating App Deployment Helper Script"
    
    cat > "$output_file" << 'EOF'
#!/usr/bin/env node
/**
 * Windmill App Deployment Helper
 * 
 * This script helps automate app deployment when Windmill adds API support.
 * Currently, this is a template for future use.
 */

const fs = require('fs');
const https = require('https');

class WindmillAppDeployer {
    constructor(apiUrl, apiToken) {
        this.apiUrl = apiUrl;
        this.apiToken = apiToken;
    }

    async deployApp(appDefinition, workspace) {
        console.log('Windmill App Deployment Helper');
        console.log('==============================');
        console.log('');
        console.log('This script is prepared for when Windmill adds app deployment API.');
        console.log('Currently, apps must be created through the UI.');
        console.log('');
        console.log('App to deploy:', appDefinition.name);
        console.log('Workspace:', workspace);
        console.log('');
        console.log('Future API endpoint might be:');
        console.log(`POST ${this.apiUrl}/api/w/${workspace}/apps/create`);
        console.log('');
        console.log('Check for updates at:');
        console.log('https://docs.windmill.dev/docs/core_concepts/apps_and_scripts#apps');
        
        // Future implementation would go here
        return false;
    }
}

// Usage example
if (require.main === module) {
    const args = process.argv.slice(2);
    if (args.length < 2) {
        console.log('Usage: node windmill-app-deployer.js <app-file.json> <workspace>');
        process.exit(1);
    }
    
    const appFile = args[0];
    const workspace = args[1];
    const apiUrl = process.env.WINDMILL_API_URL || 'http://localhost:5681';
    const apiToken = process.env.WINDMILL_API_TOKEN || '';
    
    if (!apiToken) {
        console.error('Error: WINDMILL_API_TOKEN environment variable not set');
        process.exit(1);
    }
    
    try {
        const appDef = JSON.parse(fs.readFileSync(appFile, 'utf8'));
        const deployer = new WindmillAppDeployer(apiUrl, apiToken);
        deployer.deployApp(appDef, workspace);
    } catch (error) {
        console.error('Error:', error.message);
        process.exit(1);
    }
}

module.exports = WindmillAppDeployer;
EOF

    chmod +x "$output_file"
    
    log::success "✅ App deployment helper script created"
    echo
    echo "Location: $output_file"
    echo
    echo "This script is a template for future use when Windmill adds app API support."
    echo "Monitor Windmill releases for app deployment API availability."
}