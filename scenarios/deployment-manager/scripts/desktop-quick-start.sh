#!/usr/bin/env bash
#
# Desktop Deployment Quick Start
# Automates the complete workflow for deploying a scenario as a bundled desktop app (Tier 2)
#
# Usage:
#   ./desktop-quick-start.sh <scenario-name> [options]
#
# Options:
#   --profile-name <name>    Name for the deployment profile (default: <scenario>-desktop)
#   --output <path>          Output directory for bundle.json (default: scenario's platforms/electron/)
#   --platforms <list>       Comma-separated platforms: linux-x64,darwin-arm64,win-x64 (default: all)
#   --skip-validation        Skip pre-flight validation
#   --dry-run                Show what would be done without executing
#   --help                   Show this help message
#
# Prerequisites:
#   - deployment-manager scenario must be running
#   - scenario-to-desktop scenario must be running (for bundled mode)
#   - Go 1.21+ (for compiling Go services)
#   - Node.js 18+ and pnpm (for UI build)
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
VROOLI_ROOT="${VROOLI_ROOT:-${HOME}/Vrooli}"
SCENARIO_NAME=""
PROFILE_NAME=""
OUTPUT_PATH=""
PLATFORMS="linux-x64,darwin-arm64,darwin-x64,win-x64"
SKIP_VALIDATION=false
DRY_RUN=false

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "\n${GREEN}==>${NC} ${BLUE}$1${NC}"
}

show_help() {
    cat << 'EOF'
Desktop Deployment Quick Start
==============================

Automates the complete workflow for deploying a scenario as a bundled desktop app.

USAGE:
    ./desktop-quick-start.sh <scenario-name> [options]

ARGUMENTS:
    scenario-name    Name of the scenario to deploy (required)

OPTIONS:
    --profile-name <name>    Name for the deployment profile
                             Default: <scenario>-desktop

    --output <path>          Output directory for bundle.json and build artifacts
                             Default: scenarios/<scenario>/platforms/electron/

    --platforms <list>       Comma-separated list of target platforms
                             Default: linux-x64,darwin-arm64,darwin-x64,win-x64
                             Available: linux-x64, linux-arm64, darwin-x64, darwin-arm64, win-x64

    --skip-validation        Skip pre-flight validation checks

    --dry-run                Show what would be done without making changes

    --help                   Show this help message

WORKFLOW:
    This script performs the following steps:

    1. Check Prerequisites
       - Verify deployment-manager is running
       - Verify scenario exists

    2. Analyze Compatibility
       - Run fitness check for desktop (tier 2)
       - Identify required dependency swaps

    3. Create/Update Profile
       - Create deployment profile if needed
       - Apply necessary swaps (postgres->sqlite, etc.)

    4. Generate Bundle Manifest
       - Export production-ready bundle.json with checksum
       - Include secret configuration

    5. Build Instructions
       - Output next steps for building installers

EXAMPLES:
    # Deploy browser-automation-studio as a desktop app
    ./desktop-quick-start.sh browser-automation-studio

    # Deploy with custom profile name
    ./desktop-quick-start.sh my-scenario --profile-name my-desktop-app

    # Deploy for specific platforms only
    ./desktop-quick-start.sh my-scenario --platforms linux-x64,darwin-arm64

    # Dry run to see what would happen
    ./desktop-quick-start.sh my-scenario --dry-run

REQUIREMENTS:
    - deployment-manager scenario running (vrooli scenario start deployment-manager)
    - Go 1.21+ for compiling Go services
    - Node.js 18+ and pnpm for UI builds
    - Rust/Cargo (optional, for Rust services)
EOF
}

check_prerequisites() {
    log_step "Checking Prerequisites"

    # Check if deployment-manager is running
    local api_port
    api_port=$(vrooli scenario port deployment-manager API_PORT 2>/dev/null || echo "")

    if [[ -z "$api_port" ]]; then
        log_error "deployment-manager is not running"
        log_info "Start it with: vrooli scenario start deployment-manager"
        exit 1
    fi
    log_success "deployment-manager is running on port $api_port"

    # Check if scenario exists
    local scenario_path="${VROOLI_ROOT}/scenarios/${SCENARIO_NAME}"
    if [[ ! -d "$scenario_path" ]]; then
        log_error "Scenario '$SCENARIO_NAME' not found at $scenario_path"
        exit 1
    fi
    log_success "Scenario '$SCENARIO_NAME' found"

    # Check for Go
    if command -v go &> /dev/null; then
        log_success "Go $(go version | awk '{print $3}')"
    else
        log_warn "Go not installed - Go services cannot be compiled"
    fi

    # Check for Node.js
    if command -v node &> /dev/null; then
        log_success "Node.js $(node --version)"
    else
        log_warn "Node.js not installed - UI cannot be built"
    fi

    export DM_API_PORT="$api_port"
}

check_fitness() {
    log_step "Analyzing Compatibility (Tier 2 - Desktop)"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would run: deployment-manager fitness $SCENARIO_NAME --tier 2"
        return 0
    fi

    local result
    result=$(deployment-manager fitness "$SCENARIO_NAME" --tier 2 2>&1) || {
        log_error "Failed to analyze fitness"
        echo "$result"
        exit 1
    }

    echo "$result"

    # Extract overall score
    local score
    score=$(echo "$result" | jq -r '.scores.overall // 0' 2>/dev/null || echo "0")

    if [[ "$score" -lt 40 ]]; then
        log_warn "Low fitness score ($score). Significant swaps may be required."
    elif [[ "$score" -lt 60 ]]; then
        log_info "Moderate fitness score ($score). Some swaps may be needed."
    else
        log_success "Good fitness score ($score)"
    fi

    # Check for blockers
    local blockers
    blockers=$(echo "$result" | jq -r '.blockers[]? // empty' 2>/dev/null || echo "")
    if [[ -n "$blockers" ]]; then
        log_warn "Blockers detected:"
        echo "$blockers" | while read -r blocker; do
            echo "  - $blocker"
        done
    fi
}

list_swaps() {
    log_step "Checking Available Swaps"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would run: deployment-manager swaps list $SCENARIO_NAME"
        return 0
    fi

    local result
    result=$(deployment-manager swaps list "$SCENARIO_NAME" 2>&1) || {
        log_warn "Could not list swaps (this is normal for scenarios without dependencies)"
        return 0
    }

    local swap_count
    swap_count=$(echo "$result" | jq -r 'length // 0' 2>/dev/null || echo "0")

    if [[ "$swap_count" -gt 0 ]]; then
        log_info "Available swaps:"
        echo "$result" | jq -r '.[] | "  \(.from) -> \(.to): \(.reason)"' 2>/dev/null || echo "$result"
    else
        log_success "No swaps required"
    fi
}

create_or_update_profile() {
    log_step "Creating Deployment Profile"

    local profile_id=""

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would create profile: $PROFILE_NAME for $SCENARIO_NAME --tier 2"
        echo "PROFILE_ID=dry-run-profile-id"
        return 0
    fi

    # Check if profile already exists
    local existing
    existing=$(deployment-manager profiles --format json 2>/dev/null | jq -r ".[] | select(.name == \"$PROFILE_NAME\") | .id" 2>/dev/null || echo "")

    if [[ -n "$existing" ]]; then
        log_info "Profile '$PROFILE_NAME' already exists (ID: $existing)"
        profile_id="$existing"
    else
        # Create new profile
        local result
        result=$(deployment-manager profile create "$PROFILE_NAME" "$SCENARIO_NAME" --tier 2 2>&1) || {
            log_error "Failed to create profile"
            echo "$result"
            exit 1
        }

        profile_id=$(echo "$result" | jq -r '.id // empty' 2>/dev/null || echo "")

        if [[ -z "$profile_id" ]]; then
            log_error "Failed to extract profile ID from response"
            echo "$result"
            exit 1
        fi

        log_success "Created profile '$PROFILE_NAME' (ID: $profile_id)"
    fi

    # Apply common swaps for desktop bundling
    log_info "Applying desktop-compatible swaps..."

    # Try postgres -> sqlite swap
    deployment-manager swaps apply "$profile_id" postgres sqlite 2>/dev/null && \
        log_success "Applied swap: postgres -> sqlite" || \
        log_info "postgres swap not applicable"

    # Try redis -> in-process swap
    deployment-manager swaps apply "$profile_id" redis in-process 2>/dev/null && \
        log_success "Applied swap: redis -> in-process" || \
        log_info "redis swap not applicable"

    echo "PROFILE_ID=$profile_id"
}

validate_profile() {
    local profile_id="$1"

    if [[ "$SKIP_VALIDATION" == "true" ]]; then
        log_info "Skipping validation (--skip-validation)"
        return 0
    fi

    log_step "Validating Profile"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would run: deployment-manager validate $profile_id --verbose"
        return 0
    fi

    local result
    result=$(deployment-manager validate "$profile_id" --verbose 2>&1) || {
        log_error "Validation failed"
        echo "$result"
        exit 1
    }

    local valid
    valid=$(echo "$result" | jq -r '.valid // false' 2>/dev/null || echo "false")

    if [[ "$valid" == "true" ]]; then
        log_success "Profile validation passed"
    else
        log_error "Profile validation failed"
        echo "$result" | jq -r '.checks[] | select(.status != "pass") | "  [\(.status)] \(.name): \(.details)"' 2>/dev/null || echo "$result"
        exit 1
    fi
}

generate_bundle() {
    log_step "Generating Bundle Manifest"

    # Determine output path
    if [[ -z "$OUTPUT_PATH" ]]; then
        OUTPUT_PATH="${VROOLI_ROOT}/scenarios/${SCENARIO_NAME}/platforms/electron"
    fi

    # Create output directory
    mkdir -p "$OUTPUT_PATH"

    local bundle_file="${OUTPUT_PATH}/bundle.json"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would export bundle to: $bundle_file"
        return 0
    fi

    log_info "Exporting bundle manifest to: $bundle_file"

    local result
    result=$(deployment-manager bundle export "$SCENARIO_NAME" --output "$bundle_file" 2>&1) || {
        log_error "Failed to export bundle"
        echo "$result"
        exit 1
    }

    echo "$result"

    if [[ -f "$bundle_file" ]]; then
        log_success "Bundle manifest generated: $bundle_file"
        log_info "Checksum: $(sha256sum "$bundle_file" | cut -d' ' -f1)"
    else
        log_error "Bundle file was not created"
        exit 1
    fi
}

print_next_steps() {
    log_step "Next Steps"

    local electron_path="${OUTPUT_PATH:-${VROOLI_ROOT}/scenarios/${SCENARIO_NAME}/platforms/electron}"

    cat << EOF

Your bundle manifest has been generated. To complete the desktop build:

1. Build the UI (if not already built):
   ${BLUE}cd ${VROOLI_ROOT}/scenarios/${SCENARIO_NAME}/ui${NC}
   ${BLUE}pnpm install && pnpm run build${NC}

2. Build binaries (if using build config - automatic during packaging):
   The bundle packager will compile services with 'build' config automatically.

3. Generate Electron wrapper (if not exists):
   ${BLUE}POST http://localhost:\${API_PORT}/api/v1/desktop/generate/quick${NC}
   ${BLUE}{${NC}
   ${BLUE}  "scenario_name": "${SCENARIO_NAME}",${NC}
   ${BLUE}  "deployment_mode": "bundled",${NC}
   ${BLUE}  "bundle_manifest_path": "${electron_path}/bundle.json"${NC}
   ${BLUE}}${NC}

4. Build desktop installers:
   ${BLUE}cd ${electron_path}${NC}
   ${BLUE}pnpm install${NC}
   ${BLUE}pnpm run dist:all${NC}   # Or dist:win, dist:mac, dist:linux

5. Find your installers in:
   ${BLUE}${electron_path}/dist-electron/${NC}

For more details, see:
  - Deployment Guide: ${VROOLI_ROOT}/scenarios/deployment-manager/docs/DEPLOYMENT-GUIDE.md
  - Desktop Workflow: ${VROOLI_ROOT}/scenarios/deployment-manager/docs/workflows/desktop-deployment.md

EOF
}

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --profile-name)
                PROFILE_NAME="$2"
                shift 2
                ;;
            --output)
                OUTPUT_PATH="$2"
                shift 2
                ;;
            --platforms)
                PLATFORMS="$2"
                shift 2
                ;;
            --skip-validation)
                SKIP_VALIDATION=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
            *)
                if [[ -z "$SCENARIO_NAME" ]]; then
                    SCENARIO_NAME="$1"
                else
                    log_error "Unexpected argument: $1"
                    show_help
                    exit 1
                fi
                shift
                ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$SCENARIO_NAME" ]]; then
        log_error "Scenario name is required"
        show_help
        exit 1
    fi

    # Set default profile name
    if [[ -z "$PROFILE_NAME" ]]; then
        PROFILE_NAME="${SCENARIO_NAME}-desktop"
    fi

    echo ""
    echo "========================================"
    echo "  Desktop Deployment Quick Start"
    echo "========================================"
    echo ""
    echo "Scenario:  $SCENARIO_NAME"
    echo "Profile:   $PROFILE_NAME"
    echo "Platforms: $PLATFORMS"
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}Mode:      DRY RUN${NC}"
    fi
    echo ""

    # Execute workflow
    check_prerequisites
    check_fitness
    list_swaps

    # Create profile and capture output
    local profile_output
    profile_output=$(create_or_update_profile)
    eval "$profile_output"

    if [[ -n "${PROFILE_ID:-}" ]]; then
        validate_profile "$PROFILE_ID"
    fi

    generate_bundle
    print_next_steps

    log_success "Quick start complete!"
}

main "$@"
