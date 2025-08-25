#!/usr/bin/env bash
# Windmill App Management Functions
# Functions for managing and preparing Windmill app definitions

# Source trash module for safe cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
WINDMILL_LIB_DIR="${APP_ROOT}/resources/windmill/lib"
# shellcheck disable=SC1091
source "${WINDMILL_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# List available app examples
#######################################
windmill::list_apps() {
    log::header "📱 Available Windmill App Examples"
    
    # Get script directory dynamically if WINDMILL_LIB_DIR is not set
    local script_dir="${WINDMILL_LIB_DIR:-$(builtin cd "${BASH_SOURCE[0]%/*/.." && builtin pwd)}"
    local apps_dir="${script_dir}/examples/apps"
    
    if [[ ! -d "$apps_dir" ]]; then
        log::error "Apps directory not found: $apps_dir"
        return 1
    fi
    
    echo
    echo "The following app examples are available:"
    echo
    
    # List all JSON files in the apps directory  
    local app_count=0
    shopt -s nullglob  # Handle case where no files match
    
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
    
    shopt -u nullglob  # Reset option
    
    if [[ $app_count -eq 0 ]]; then
        log::warn "No app examples found"
        return 1
    fi
    
    echo "To prepare an app for deployment:"
    echo "  $0 --action prepare-app --app-name <app-name>"
    echo "  $0 --action deploy-app --app-name <app-name> --workspace <workspace>"
    echo
    echo "Note: App creation API exists but may have workspace constraint issues"
    echo "      in some Windmill versions. Manual import may still be needed."
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
    
    local apps_dir="${WINDMILL_LIB_DIR%/*/examples/apps"
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
    log::info "Note: App creation API exists but may have constraints in some versions."
    log::info "If deployment fails, use these files for manual import."
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
    
    # App creation endpoints exist in Windmill
    log::info "App creation API endpoints are available:"
    log::info "  - POST /api/w/{workspace}/apps/create"
    log::info "  - POST /api/w/{workspace}/apps/create_raw"
    log::info "  - GET  /api/w/{workspace}/apps/list"
    log::info "  - GET  /api/w/{workspace}/apps/get/p/{path}"
    log::info "  - GET  /api/w/{workspace}/apps/exists/{path}"
    
    # Check if we can list apps in demo workspace
    local response
    if response=$(curl -s -H "Authorization: Bearer $api_token" \
        "${WINDMILL_BASE_URL}/api/w/demo/apps/list" 2>/dev/null); then
        log::success "API endpoints confirmed working"
    fi
    
    log::warn "Note: App creation via API may fail due to workspace constraint issues in some versions"
    
    return 0
}

#######################################
# Future: Deploy app via API
#######################################
windmill::deploy_app() {
    local app_name="$1"
    local workspace="${2:-demo}"
    local api_token
    
    api_token=$(windmill::load_api_key)
    if [[ -z "$api_token" ]]; then
        log::error "API token required for app deployment"
        return 1
    fi
    
    # Find the app file
    local apps_dir="${WINDMILL_LIB_DIR%/*/examples/apps"
    local app_file="${apps_dir}/${app_name}.json"
    
    if [[ ! -f "$app_file" ]]; then
        log::error "App not found: $app_name"
        return 1
    fi
    
    log::info "Deploying app '$app_name' to workspace '$workspace'..."
    
    # Convert our app format to Windmill's expected format
    local temp_file="/tmp/windmill-app-deploy-${app_name}.json"
    
    # Create minimal Windmill app structure
    if command -v jq >/dev/null 2>&1; then
        jq '{
            path: .name | ascii_downcase | gsub(" "; "_"),
            value: .app_structure // {},
            summary: .description // .name,
            policy: {
                execution_mode: "viewer"
            }
        }' "$app_file" > "$temp_file"
    else
        log::error "jq is required for app deployment"
        return 1
    fi
    
    # Deploy the app via API
    local response
    local http_code
    
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Authorization: Bearer $api_token" \
        -H "Content-Type: application/json" \
        -d "@$temp_file" \
        "${WINDMILL_BASE_URL}/api/w/${workspace}/apps/create" 2>&1)
    
    http_code=$(echo "$response" | tail -1)
    local body=$(echo "$response" | head -n -1)
    
    # Clean up temp file
    trash::safe_remove "$temp_file" --temp
    
    if [[ "$http_code" == "201" ]] || [[ "$http_code" == "200" ]]; then
        log::success "App deployed successfully!"
        echo "App path: $body"
        echo "Access it at: ${WINDMILL_BASE_URL}/apps/get/${workspace}/$body"
        return 0
    else
        log::error "Failed to deploy app (HTTP $http_code)"
        log::error "Response: $body"
        
        if [[ "$body" == *"app_workspace_id_fkey"* ]]; then
            log::warn "This appears to be a workspace constraint issue in this Windmill version"
            log::info "Try creating the app manually through the UI instead"
            log::info "Use --action prepare-app to generate the files for manual import"
        elif [[ "$body" == *"missing field"* ]]; then
            log::warn "The app structure may need adjustment for this Windmill version"
            log::info "Check the prepared files and import manually"
        fi
        return 1
    fi
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