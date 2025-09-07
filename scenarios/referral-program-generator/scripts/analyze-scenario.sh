#!/bin/bash

# Referral Program Generator - Scenario Analysis Script
# Analyzes scenarios for branding, pricing, and structural information

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../.."

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS] SCENARIO_PATH

Analyze a Vrooli scenario for branding, pricing, and structural information.

OPTIONS:
    -m, --mode MODE     Analysis mode: 'local' (default) or 'deployed'
    -o, --output FORMAT Output format: 'json' (default) or 'summary'
    -v, --verbose       Enable verbose output
    -h, --help         Show this help message

EXAMPLES:
    $0 ../test-scenario-1
    $0 --mode local --output json ../my-scenario
    $0 --mode deployed --output summary https://github.com/user/repo

EOF
}

# Parse command line arguments
MODE="local"
OUTPUT_FORMAT="json"
VERBOSE=false
SCENARIO_PATH=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -m|--mode)
            MODE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_FORMAT="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            SCENARIO_PATH="$1"
            shift
            ;;
    esac
done

# Validate arguments
if [[ -z "$SCENARIO_PATH" ]]; then
    log_error "SCENARIO_PATH is required"
    usage
    exit 1
fi

if [[ "$MODE" != "local" && "$MODE" != "deployed" ]]; then
    log_error "Invalid mode: $MODE. Must be 'local' or 'deployed'"
    exit 1
fi

if [[ "$OUTPUT_FORMAT" != "json" && "$OUTPUT_FORMAT" != "summary" ]]; then
    log_error "Invalid output format: $OUTPUT_FORMAT. Must be 'json' or 'summary'"
    exit 1
fi

# Analysis functions
analyze_branding_local() {
    local scenario_path="$1"
    local branding_data="{}"
    
    if [[ $VERBOSE == true ]]; then
        log_info "Analyzing branding for local scenario: $scenario_path"
    fi
    
    # Look for CSS files with color definitions
    local css_files
    css_files=$(find "$scenario_path" -name "*.css" -type f 2>/dev/null || true)
    
    local primary_color="#007bff"
    local secondary_color="#6c757d" 
    local accent_color="#28a745"
    local fonts="[]"
    local logo_path=""
    
    if [[ -n "$css_files" ]]; then
        while IFS= read -r css_file; do
            if [[ -f "$css_file" ]]; then
                # Extract primary colors (looking for CSS custom properties or common color patterns)
                local found_primary
                found_primary=$(grep -oE "(--primary-color|--main-color|\.primary.*color):\s*#[0-9a-fA-F]{6}" "$css_file" | head -1 | grep -oE "#[0-9a-fA-F]{6}" || true)
                if [[ -n "$found_primary" ]]; then
                    primary_color="$found_primary"
                fi
                
                # Look for font families
                local found_fonts
                found_fonts=$(grep -oE "font-family:\s*[^;]+" "$css_file" | sed 's/font-family:\s*//' | head -3 || true)
                if [[ -n "$found_fonts" ]]; then
                    fonts="[\"$(echo "$found_fonts" | tr '\n' '","' | sed 's/,$//' | sed 's/,$//g')\"]"
                fi
            fi
        done <<< "$css_files"
    fi
    
    # Look for logo files
    local logo_candidates
    logo_candidates=$(find "$scenario_path" \( -name "*logo*" -o -name "*brand*" \) \( -name "*.png" -o -name "*.svg" -o -name "*.jpg" -o -name "*.jpeg" \) -type f 2>/dev/null | head -1 || true)
    if [[ -n "$logo_candidates" ]]; then
        logo_path="$logo_candidates"
    fi
    
    # Look for package.json or HTML title for brand name
    local brand_name=""
    if [[ -f "$scenario_path/package.json" ]]; then
        brand_name=$(jq -r '.name // empty' "$scenario_path/package.json" 2>/dev/null || true)
    fi
    
    if [[ -z "$brand_name" ]]; then
        local html_files
        html_files=$(find "$scenario_path" -name "*.html" -type f 2>/dev/null | head -1 || true)
        if [[ -n "$html_files" && -f "$html_files" ]]; then
            brand_name=$(grep -oE "<title>([^<]+)" "$html_files" | sed 's/<title>//' || true)
        fi
    fi
    
    # Build branding JSON
    branding_data=$(jq -n \
        --arg primary_color "$primary_color" \
        --arg secondary_color "$secondary_color" \
        --arg accent_color "$accent_color" \
        --argjson fonts "$fonts" \
        --arg logo_path "$logo_path" \
        --arg brand_name "$brand_name" \
        '{
            colors: {
                primary: $primary_color,
                secondary: $secondary_color,
                accent: $accent_color
            },
            fonts: $fonts,
            logo_path: $logo_path,
            brand_name: $brand_name
        }')
    
    echo "$branding_data"
}

analyze_pricing_local() {
    local scenario_path="$1"
    local pricing_data="{}"
    
    if [[ $VERBOSE == true ]]; then
        log_info "Analyzing pricing for local scenario: $scenario_path"
    fi
    
    local model="freemium"
    local tiers="[]"
    
    # Look for pricing information in API files
    local api_files
    api_files=$(find "$scenario_path" -path "*/api/*" -name "*.go" -type f 2>/dev/null || true)
    
    if [[ -n "$api_files" ]]; then
        while IFS= read -r api_file; do
            if [[ -f "$api_file" ]]; then
                # Look for pricing-related constants or structures
                if grep -q -i "subscription\|recurring\|monthly\|yearly" "$api_file"; then
                    model="subscription"
                elif grep -q -i "one.*time\|purchase\|buy.*once" "$api_file"; then
                    model="one-time"
                fi
                
                # Look for price values (simple heuristic)
                local price_matches
                price_matches=$(grep -oE '\$[0-9]+(\.[0-9]{2})?|\b[0-9]+\.[0-9]{2}\s*(USD|usd|\$)' "$api_file" || true)
                if [[ -n "$price_matches" ]]; then
                    # Create simple tier structure
                    tiers='[{"name": "Standard", "price": "29.99", "period": "monthly"}]'
                fi
            fi
        done <<< "$api_files"
    fi
    
    # Look for pricing in configuration files
    if [[ -f "$scenario_path/.vrooli/service.json" ]]; then
        local service_config
        service_config=$(cat "$scenario_path/.vrooli/service.json")
        if echo "$service_config" | jq -e '.pricing' >/dev/null 2>&1; then
            local extracted_pricing
            extracted_pricing=$(echo "$service_config" | jq '.pricing // {}')
            if [[ "$extracted_pricing" != "{}" ]]; then
                model=$(echo "$extracted_pricing" | jq -r '.model // "freemium"')
                tiers=$(echo "$extracted_pricing" | jq '.tiers // []')
            fi
        fi
    fi
    
    # Build pricing JSON
    pricing_data=$(jq -n \
        --arg model "$model" \
        --argjson tiers "$tiers" \
        '{
            model: $model,
            tiers: $tiers,
            currency: "USD",
            billing_cycle: "monthly"
        }')
    
    echo "$pricing_data"
}

analyze_structure_local() {
    local scenario_path="$1"
    local structure_data="{}"
    
    if [[ $VERBOSE == true ]]; then
        log_info "Analyzing structure for local scenario: $scenario_path"
    fi
    
    local has_api=false
    local has_ui=false
    local has_cli=false
    local has_database=false
    local has_referral=false
    local api_framework=""
    local ui_framework=""
    
    # Check for API
    if [[ -d "$scenario_path/api" ]]; then
        has_api=true
        if [[ -f "$scenario_path/api/go.mod" ]]; then
            api_framework="go"
        elif [[ -f "$scenario_path/api/package.json" ]]; then
            api_framework="nodejs"
        elif [[ -f "$scenario_path/api/requirements.txt" ]]; then
            api_framework="python"
        fi
    fi
    
    # Check for UI
    if [[ -d "$scenario_path/ui" ]]; then
        has_ui=true
        if [[ -f "$scenario_path/ui/package.json" ]]; then
            local ui_deps
            ui_deps=$(jq -r '.dependencies // {} | keys[]' "$scenario_path/ui/package.json" 2>/dev/null || true)
            if echo "$ui_deps" | grep -q "react"; then
                ui_framework="react"
            elif echo "$ui_deps" | grep -q "vue"; then
                ui_framework="vue"
            elif echo "$ui_deps" | grep -q "angular"; then
                ui_framework="angular"
            else
                ui_framework="javascript"
            fi
        fi
    fi
    
    # Check for CLI
    if [[ -d "$scenario_path/cli" ]]; then
        has_cli=true
    fi
    
    # Check for database initialization
    if [[ -d "$scenario_path/initialization/storage" || -f "$scenario_path/initialization"*"/schema.sql" ]]; then
        has_database=true
    fi
    
    # Check for existing referral logic
    local referral_indicators
    referral_indicators=$(find "$scenario_path" -type f \( -name "*.go" -o -name "*.js" -o -name "*.ts" -o -name "*.py" \) -exec grep -l -i "referral\|affiliate" {} \; 2>/dev/null || true)
    if [[ -n "$referral_indicators" ]]; then
        has_referral=true
    fi
    
    # Build structure JSON
    structure_data=$(jq -n \
        --argjson has_api "$has_api" \
        --argjson has_ui "$has_ui" \
        --argjson has_cli "$has_cli" \
        --argjson has_database "$has_database" \
        --argjson has_referral "$has_referral" \
        --arg api_framework "$api_framework" \
        --arg ui_framework "$ui_framework" \
        '{
            has_api: $has_api,
            has_ui: $has_ui,
            has_cli: $has_cli,
            has_database: $has_database,
            has_existing_referral: $has_referral,
            api_framework: $api_framework,
            ui_framework: $ui_framework
        }')
    
    echo "$structure_data"
}

analyze_deployed() {
    local url="$1"
    log_warning "Deployed analysis not yet implemented - using browserless resource"
    
    # Placeholder for browserless integration
    echo '{
        "branding": {
            "colors": {
                "primary": "#007bff",
                "secondary": "#6c757d",
                "accent": "#28a745"
            },
            "fonts": ["Arial", "sans-serif"],
            "logo_path": "",
            "brand_name": "Unknown"
        },
        "pricing": {
            "model": "unknown",
            "tiers": [],
            "currency": "USD"
        },
        "structure": {
            "has_api": true,
            "has_ui": true,
            "has_cli": false,
            "has_database": false,
            "has_existing_referral": false
        }
    }'
}

# Main analysis function
analyze_scenario() {
    local scenario_path="$1"
    local mode="$2"
    
    if [[ "$mode" == "local" ]]; then
        if [[ ! -d "$scenario_path" ]]; then
            log_error "Scenario directory not found: $scenario_path"
            exit 1
        fi
        
        local branding
        local pricing
        local structure
        
        branding=$(analyze_branding_local "$scenario_path")
        pricing=$(analyze_pricing_local "$scenario_path")
        structure=$(analyze_structure_local "$scenario_path")
        
        # Combine results
        jq -n \
            --argjson branding "$branding" \
            --argjson pricing "$pricing" \
            --argjson structure "$structure" \
            --arg scenario_path "$scenario_path" \
            --arg analysis_timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            '{
                scenario_path: $scenario_path,
                analysis_timestamp: $analysis_timestamp,
                mode: "local",
                branding: $branding,
                pricing: $pricing,
                structure: $structure
            }'
    else
        analyze_deployed "$scenario_path"
    fi
}

# Output formatting
format_output() {
    local analysis_result="$1"
    local format="$2"
    
    if [[ "$format" == "json" ]]; then
        echo "$analysis_result" | jq '.'
    else
        # Summary format
        echo "=== Scenario Analysis Summary ==="
        echo ""
        
        local scenario_path
        scenario_path=$(echo "$analysis_result" | jq -r '.scenario_path')
        echo "Scenario: $scenario_path"
        echo "Analyzed: $(echo "$analysis_result" | jq -r '.analysis_timestamp')"
        echo ""
        
        echo "🎨 BRANDING:"
        echo "  Primary Color: $(echo "$analysis_result" | jq -r '.branding.colors.primary')"
        echo "  Brand Name: $(echo "$analysis_result" | jq -r '.branding.brand_name')"
        echo "  Logo: $(echo "$analysis_result" | jq -r '.branding.logo_path' | sed 's/^$/None found/')"
        echo ""
        
        echo "💰 PRICING:"
        echo "  Model: $(echo "$analysis_result" | jq -r '.pricing.model')"
        echo "  Tiers: $(echo "$analysis_result" | jq '.pricing.tiers | length') found"
        echo ""
        
        echo "🏗️  STRUCTURE:"
        echo "  API: $(echo "$analysis_result" | jq -r '.structure.has_api')"
        echo "  UI: $(echo "$analysis_result" | jq -r '.structure.has_ui')"
        echo "  CLI: $(echo "$analysis_result" | jq -r '.structure.has_cli')"
        echo "  Database: $(echo "$analysis_result" | jq -r '.structure.has_database')"
        echo "  Existing Referral: $(echo "$analysis_result" | jq -r '.structure.has_existing_referral')"
        echo ""
        
        if [[ $(echo "$analysis_result" | jq -r '.structure.has_existing_referral') == "true" ]]; then
            echo "⚠️  WARNING: This scenario already has referral logic implemented."
        fi
    fi
}

# Main execution
main() {
    if [[ $VERBOSE == true ]]; then
        log_info "Starting scenario analysis"
        log_info "Mode: $MODE"
        log_info "Output format: $OUTPUT_FORMAT"
        log_info "Scenario path: $SCENARIO_PATH"
    fi
    
    local result
    result=$(analyze_scenario "$SCENARIO_PATH" "$MODE")
    
    format_output "$result" "$OUTPUT_FORMAT"
    
    if [[ $VERBOSE == true ]]; then
        log_success "Analysis completed successfully"
    fi
}

# Run main function
main