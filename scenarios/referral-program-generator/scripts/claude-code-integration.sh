#!/bin/bash

# Referral Program Generator - Claude Code Integration
# Automates referral pattern implementation using resource-claude-code

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../.."
LIB_DIR="${SCRIPT_DIR}/../lib"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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
Usage: $0 [OPTIONS] SCENARIO_PATH PROGRAM_CONFIG

Integrate referral program into a scenario using resource-claude-code.

ARGUMENTS:
    SCENARIO_PATH     Path to the target scenario directory
    PROGRAM_CONFIG    JSON file containing referral program configuration

OPTIONS:
    --dry-run         Show what would be done without making changes
    --force           Overwrite existing referral implementation
    --verbose         Enable detailed output
    --help            Show this help message

EXAMPLES:
    $0 ../my-scenario program-config.json
    $0 --dry-run --verbose ../my-scenario program-config.json

PROGRAM_CONFIG JSON Format:
{
    "program_id": "uuid",
    "scenario_name": "my-scenario",
    "brand_name": "My App",
    "commission_rate": 0.20,
    "branding": {
        "colors": {"primary": "#007bff", "secondary": "#6c757d"},
        "fonts": ["Inter", "sans-serif"]
    },
    "structure": {
        "api_framework": "go",
        "ui_framework": "react"
    }
}

EOF
}

# Default values
DRY_RUN=false
FORCE_OVERWRITE=false
VERBOSE=false
SCENARIO_PATH=""
PROGRAM_CONFIG=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force)
            FORCE_OVERWRITE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            if [[ -z "$SCENARIO_PATH" ]]; then
                SCENARIO_PATH="$1"
            elif [[ -z "$PROGRAM_CONFIG" ]]; then
                PROGRAM_CONFIG="$1"
            else
                log_error "Too many arguments"
                usage
                exit 1
            fi
            shift
            ;;
    esac
done

# Validate arguments
if [[ -z "$SCENARIO_PATH" || -z "$PROGRAM_CONFIG" ]]; then
    log_error "Both SCENARIO_PATH and PROGRAM_CONFIG are required"
    usage
    exit 1
fi

if [[ ! -d "$SCENARIO_PATH" ]]; then
    log_error "Scenario directory not found: $SCENARIO_PATH"
    exit 1
fi

if [[ ! -f "$PROGRAM_CONFIG" ]]; then
    log_error "Program config file not found: $PROGRAM_CONFIG"
    exit 1
fi

# Load and validate configuration
load_config() {
    if ! command -v jq >/dev/null 2>&1; then
        log_error "jq is required but not installed"
        exit 1
    fi
    
    local config_data
    config_data=$(cat "$PROGRAM_CONFIG")
    
    # Validate required fields
    local required_fields=("program_id" "scenario_name" "brand_name" "commission_rate")
    for field in "${required_fields[@]}"; do
        if ! echo "$config_data" | jq -e ".$field" >/dev/null 2>&1; then
            log_error "Missing required field in config: $field"
            exit 1
        fi
    done
    
    # Extract configuration values
    PROGRAM_ID=$(echo "$config_data" | jq -r '.program_id')
    SCENARIO_NAME=$(echo "$config_data" | jq -r '.scenario_name')
    BRAND_NAME=$(echo "$config_data" | jq -r '.brand_name')
    COMMISSION_RATE=$(echo "$config_data" | jq -r '.commission_rate')
    PRIMARY_COLOR=$(echo "$config_data" | jq -r '.branding.colors.primary // "#007bff"')
    SECONDARY_COLOR=$(echo "$config_data" | jq -r '.branding.colors.secondary // "#6c757d"')
    API_FRAMEWORK=$(echo "$config_data" | jq -r '.structure.api_framework // "go"')
    UI_FRAMEWORK=$(echo "$config_data" | jq -r '.structure.ui_framework // "react"')
    
    if [[ "$VERBOSE" == true ]]; then
        log_info "Configuration loaded:"
        log_info "  Program ID: $PROGRAM_ID"
        log_info "  Scenario: $SCENARIO_NAME ($BRAND_NAME)"
        log_info "  Commission Rate: $(echo "$COMMISSION_RATE * 100" | bc)%"
        log_info "  Colors: $PRIMARY_COLOR / $SECONDARY_COLOR"
        log_info "  Frameworks: $API_FRAMEWORK / $UI_FRAMEWORK"
    fi
}

# Check if referral implementation already exists
check_existing_implementation() {
    local referral_indicators=()
    
    # Look for referral-related files
    if [[ -d "$SCENARIO_PATH/src/referral" ]]; then
        referral_indicators+=("src/referral directory exists")
    fi
    
    # Look for referral code in API files
    local api_files
    api_files=$(find "$SCENARIO_PATH" -path "*/api/*" -name "*.go" -type f 2>/dev/null || true)
    if [[ -n "$api_files" ]]; then
        while IFS= read -r api_file; do
            if [[ -f "$api_file" ]] && grep -q -i "referral\|affiliate" "$api_file"; then
                referral_indicators+=("referral code found in $(basename "$api_file")")
            fi
        done <<< "$api_files"
    fi
    
    # Look for referral UI components
    local ui_files
    ui_files=$(find "$SCENARIO_PATH" -path "*/ui/*" -name "*.jsx" -o -name "*.js" -o -name "*.tsx" -o -name "*.ts" 2>/dev/null || true)
    if [[ -n "$ui_files" ]]; then
        while IFS= read -r ui_file; do
            if [[ -f "$ui_file" ]] && grep -q -i "referral\|affiliate" "$ui_file"; then
                referral_indicators+=("referral UI found in $(basename "$ui_file")")
            fi
        done <<< "$ui_files"
    fi
    
    # Check database schema
    local schema_files
    schema_files=$(find "$SCENARIO_PATH" -name "schema.sql" -o -name "*schema*.sql" 2>/dev/null || true)
    if [[ -n "$schema_files" ]]; then
        while IFS= read -r schema_file; do
            if [[ -f "$schema_file" ]] && grep -q -i "referral" "$schema_file"; then
                referral_indicators+=("referral tables found in $(basename "$schema_file")")
            fi
        done <<< "$schema_files"
    fi
    
    if [[ ${#referral_indicators[@]} -gt 0 ]]; then
        log_warning "Existing referral implementation detected:"
        for indicator in "${referral_indicators[@]}"; do
            log_warning "  - $indicator"
        done
        
        if [[ "$FORCE_OVERWRITE" == false ]]; then
            log_error "Use --force to overwrite existing implementation"
            exit 1
        else
            log_warning "Proceeding with --force flag (will overwrite existing implementation)"
        fi
    fi
}

# Generate Claude Code instructions
generate_claude_instructions() {
    cat << EOF
# Referral Program Implementation Task

You are implementing a referral program for the **$BRAND_NAME** scenario located at \`$SCENARIO_PATH\`.

## Task Overview
Integrate a complete referral system that allows users to:
1. Create referral links
2. Track clicks and conversions  
3. Earn commissions on successful referrals
4. View referral performance in a dashboard

## Configuration
- **Commission Rate**: $(echo "$COMMISSION_RATE * 100" | bc)%
- **Brand Colors**: Primary: $PRIMARY_COLOR, Secondary: $SECONDARY_COLOR
- **API Framework**: $API_FRAMEWORK
- **UI Framework**: $UI_FRAMEWORK

## Implementation Requirements

### 1. Database Schema (CRITICAL)
Add referral tables to the existing database schema. If \`initialization/storage/postgres/schema.sql\` exists, extend it with:

\`\`\`sql
-- Referral program tables
CREATE TABLE IF NOT EXISTS referral_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) NOT NULL,
    commission_rate DECIMAL(5,4) NOT NULL,
    tracking_code VARCHAR(50) UNIQUE NOT NULL,
    branding_config JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS referral_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    program_id UUID NOT NULL REFERENCES referral_programs(id) ON DELETE CASCADE,
    referrer_id UUID NOT NULL,
    tracking_code VARCHAR(32) UNIQUE NOT NULL,
    clicks INTEGER DEFAULT 0,
    conversions INTEGER DEFAULT 0,
    total_commission DECIMAL(12,2) DEFAULT 0.00,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS commissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    link_id UUID NOT NULL REFERENCES referral_links(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL,
    amount DECIMAL(12,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    transaction_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_referral_links_tracking_code ON referral_links(tracking_code);
CREATE INDEX IF NOT EXISTS idx_referral_links_referrer_id ON referral_links(referrer_id);
CREATE INDEX IF NOT EXISTS idx_commissions_link_id ON commissions(link_id);
\`\`\`

### 2. API Implementation (Go)
Add these endpoints to the main API router in \`api/main.go\`:

- \`POST /api/referral/links\` - Create referral link
- \`GET /ref/{tracking_code}\` - Track click and redirect  
- \`POST /api/referral/conversions\` - Record conversion
- \`GET /api/referral/user/{user_id}/stats\` - Get user stats

Key requirements:
- Set referral cookie (30-day expiration) on click tracking
- Calculate commission as: \`purchase_amount * $COMMISSION_RATE\`
- Store all events for analytics
- Handle fraud prevention (rate limiting, duplicate detection)

### 3. UI Integration (React)
Add a referral dashboard component that shows:
- User's referral links with performance stats
- Total earnings, clicks, conversions
- "Create New Link" button
- Copy-to-clipboard functionality
- Professional styling using brand colors

### 4. Integration Points
- **User Authentication**: Use existing user system for referrer_id
- **Payment/Signup Success**: Add conversion tracking after successful transactions
- **Navigation**: Add "Referral Program" to main navigation
- **Styling**: Match existing design system and use brand colors

### 5. Configuration
Create \`src/referral/config.json\` with:
\`\`\`json
{
  "commission_rate": $COMMISSION_RATE,
  "brand_colors": {
    "primary": "$PRIMARY_COLOR",
    "secondary": "$SECONDARY_COLOR"
  },
  "tracking": {
    "cookie_duration_days": 30,
    "attribution_window_days": 30
  }
}
\`\`\`

## Critical Success Criteria
1. ✅ Database schema successfully migrated
2. ✅ API endpoints functional and tested
3. ✅ UI component renders and integrates properly
4. ✅ Referral tracking works end-to-end
5. ✅ Commission calculations are accurate
6. ✅ No breaking changes to existing functionality

## Testing Instructions
After implementation:
1. Create a referral link via API
2. Visit the tracking URL
3. Complete a signup/purchase flow
4. Verify commission is recorded correctly
5. Check that UI shows updated statistics

## Important Notes
- **Preserve existing code**: Don't modify unrelated functionality
- **Follow scenario conventions**: Match existing code style and patterns
- **Error handling**: Fail gracefully if referral system has issues
- **Performance**: Add database indexes for referral queries
- **Security**: Validate all inputs and prevent referral fraud

## Brand Integration
- Use "$BRAND_NAME" in all user-facing text
- Apply brand colors: $PRIMARY_COLOR (primary), $SECONDARY_COLOR (secondary)
- Match the scenario's existing design language
- Ensure referral UI feels native to the app

Generate clean, production-ready code that follows the scenario's existing patterns and conventions.
EOF
}

# Execute Claude Code integration
execute_claude_code() {
    local instructions_file="/tmp/referral-implementation-$PROGRAM_ID.md"
    
    # Generate detailed instructions
    generate_claude_instructions > "$instructions_file"
    
    if [[ "$DRY_RUN" == true ]]; then
        log_info "=== DRY RUN MODE ==="
        log_info "Would execute: resource-claude-code implement --scenario '$SCENARIO_PATH' --instructions '$instructions_file'"
        log_info ""
        log_info "Claude Code would receive these instructions:"
        echo ""
        cat "$instructions_file"
        echo ""
        log_info "=== END DRY RUN ==="
        return 0
    fi
    
    # Check if resource-claude-code is available
    if ! command -v resource-claude-code >/dev/null 2>&1; then
        log_error "resource-claude-code is not available"
        log_error "Please ensure Claude Code resource is installed and accessible"
        return 1
    fi
    
    log_info "Executing Claude Code implementation..."
    log_info "Instructions file: $instructions_file"
    
    # Execute the integration with a timeout
    local timeout_seconds=1800  # 30 minutes
    
    if timeout "$timeout_seconds" resource-claude-code implement \
        --scenario "$SCENARIO_PATH" \
        --instructions "$instructions_file" \
        --verbose 2>&1; then
        
        log_success "Claude Code implementation completed successfully"
        
        # Clean up instructions file
        rm -f "$instructions_file"
        
        return 0
    else
        local exit_code=$?
        log_error "Claude Code implementation failed (exit code: $exit_code)"
        
        if [[ $exit_code -eq 124 ]]; then
            log_error "Implementation timed out after $timeout_seconds seconds"
            log_error "Consider breaking the task into smaller parts"
        fi
        
        log_info "Instructions file preserved for debugging: $instructions_file"
        return 1
    fi
}

# Validate implementation
validate_implementation() {
    if [[ "$DRY_RUN" == true ]]; then
        log_info "Skipping validation in dry run mode"
        return 0
    fi
    
    log_info "Validating referral implementation..."
    
    local validation_errors=()
    
    # Check for required files
    local required_files=(
        "src/referral/config.json"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$SCENARIO_PATH/$file" ]]; then
            validation_errors+=("Missing required file: $file")
        fi
    done
    
    # Check database schema
    local schema_files
    schema_files=$(find "$SCENARIO_PATH" -name "*schema*.sql" -type f 2>/dev/null || true)
    local schema_has_referral=false
    
    if [[ -n "$schema_files" ]]; then
        while IFS= read -r schema_file; do
            if [[ -f "$schema_file" ]] && grep -q "referral_programs" "$schema_file"; then
                schema_has_referral=true
                break
            fi
        done <<< "$schema_files"
    fi
    
    if [[ "$schema_has_referral" == false ]]; then
        validation_errors+=("Database schema missing referral tables")
    fi
    
    # Check API implementation
    local api_files
    api_files=$(find "$SCENARIO_PATH" -path "*/api/*" -name "*.go" -type f 2>/dev/null || true)
    local api_has_referral=false
    
    if [[ -n "$api_files" ]]; then
        while IFS= read -r api_file; do
            if [[ -f "$api_file" ]] && grep -q "referral" "$api_file"; then
                api_has_referral=true
                break
            fi
        done <<< "$api_files"
    fi
    
    if [[ "$api_has_referral" == false ]]; then
        validation_errors+=("API implementation missing referral endpoints")
    fi
    
    # Report validation results
    if [[ ${#validation_errors[@]} -eq 0 ]]; then
        log_success "Implementation validation passed"
        return 0
    else
        log_error "Implementation validation failed:"
        for error in "${validation_errors[@]}"; do
            log_error "  - $error"
        done
        return 1
    fi
}

# Generate post-implementation report
generate_report() {
    local report_file="$SCENARIO_PATH/REFERRAL_IMPLEMENTATION_REPORT.md"
    
    cat > "$report_file" << EOF
# Referral Program Implementation Report

**Generated**: $(date)  
**Scenario**: $BRAND_NAME ($SCENARIO_NAME)  
**Program ID**: $PROGRAM_ID  

## Configuration
- **Commission Rate**: $(echo "$COMMISSION_RATE * 100" | bc)%
- **Brand Colors**: $PRIMARY_COLOR / $SECONDARY_COLOR
- **Frameworks**: $API_FRAMEWORK / $UI_FRAMEWORK

## Implementation Status
$(if [[ "$DRY_RUN" == true ]]; then
    echo "- **Status**: DRY RUN - No changes made"
else
    echo "- **Status**: Implementation completed"
fi)

## Next Steps
1. Test referral link creation: \`POST /api/referral/links\`
2. Test click tracking: \`GET /ref/{tracking_code}\`
3. Test conversion recording: \`POST /api/referral/conversions\`
4. Verify UI integration in main application
5. Monitor referral performance in production

## Files Modified/Created
- Database schema updated with referral tables
- API endpoints added for referral functionality
- UI components integrated for referral dashboard
- Configuration files created in \`src/referral/\`

## Maintenance
- Commission rates can be adjusted in database
- UI styling can be customized in component CSS
- Analytics can be extended with additional tracking

---
*Implementation completed by Referral Program Generator with resource-claude-code*
EOF
    
    log_info "Implementation report saved: $report_file"
}

# Main execution function
main() {
    log_info "Starting Claude Code integration for referral program"
    
    # Load and validate configuration
    load_config
    
    # Check for existing implementation
    check_existing_implementation
    
    # Execute Claude Code integration
    if execute_claude_code; then
        # Validate the implementation
        if validate_implementation; then
            # Generate report
            generate_report
            
            log_success "Referral program integration completed successfully!"
            log_info "Review the implementation report: $SCENARIO_PATH/REFERRAL_IMPLEMENTATION_REPORT.md"
            
            if [[ "$DRY_RUN" == false ]]; then
                log_info "Next steps:"
                log_info "1. Build and test the scenario"
                log_info "2. Verify referral functionality works end-to-end"
                log_info "3. Deploy and monitor performance"
            fi
        else
            log_error "Implementation validation failed"
            exit 1
        fi
    else
        log_error "Claude Code integration failed"
        exit 1
    fi
}

# Run main function
main