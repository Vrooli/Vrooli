#!/bin/bash
# Dependencies test phase for scenario-to-android
# Validates external dependencies and tools

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "30s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# ============================================================================
# Shell Dependencies
# ============================================================================

log::info "Validating shell dependencies"

REQUIRED_COMMANDS=(
    "bash"
    "sed"
    "grep"
    "find"
    "awk"
)

MISSING_COMMANDS=()
for cmd in "${REQUIRED_COMMANDS[@]}"; do
    if ! command -v "$cmd" &> /dev/null; then
        MISSING_COMMANDS+=("$cmd")
    fi
done

if [ ${#MISSING_COMMANDS[@]} -eq 0 ]; then
    log::success "All required shell commands available"
else
    testing::phase::add_error "Missing required commands: ${MISSING_COMMANDS[*]}"
fi

# ============================================================================
# Optional Development Tools
# ============================================================================

log::info "Checking optional development tools"

OPTIONAL_COMMANDS=(
    "jq:JSON processing"
    "shellcheck:Shell script linting"
    "bats:CLI testing"
    "bc:Calculations"
)

for item in "${OPTIONAL_COMMANDS[@]}"; do
    cmd="${item%%:*}"
    desc="${item##*:}"

    if command -v "$cmd" &> /dev/null; then
        version=$("$cmd" --version 2>&1 | head -1 || echo "unknown")
        log::success "✓ $cmd available ($desc) - $version"
    else
        log::warning "⚠ $cmd not available ($desc) - some tests may be skipped"
    fi
done

# ============================================================================
# Android SDK Dependencies
# ============================================================================

log::info "Checking Android SDK dependencies"

# Check for ANDROID_HOME
if [ -n "${ANDROID_HOME:-}" ]; then
    log::success "ANDROID_HOME is set: $ANDROID_HOME"

    if [ -d "$ANDROID_HOME" ]; then
        log::success "Android SDK directory exists"

        # Check for SDK components
        SDK_COMPONENTS=(
            "platform-tools:Platform Tools"
            "build-tools:Build Tools"
            "platforms:SDK Platforms"
        )

        for component in "${SDK_COMPONENTS[@]}"; do
            dir="${component%%:*}"
            desc="${component##*:}"

            if [ -d "$ANDROID_HOME/$dir" ]; then
                count=$(ls -1 "$ANDROID_HOME/$dir" 2>/dev/null | wc -l)
                log::success "✓ $desc available ($count versions)"
            else
                log::warning "⚠ $desc not found"
            fi
        done
    else
        log::warning "ANDROID_HOME points to non-existent directory"
    fi
else
    log::warning "ANDROID_HOME not set - Android builds will not work"
    log::info "Set ANDROID_HOME to enable Android app building"
fi

# Check for Android command-line tools
ANDROID_TOOLS=(
    "adb:Android Debug Bridge"
    "aapt:Android Asset Packaging Tool"
    "emulator:Android Emulator"
)

for tool in "${ANDROID_TOOLS[@]}"; do
    cmd="${tool%%:*}"
    desc="${tool##*:}"

    if command -v "$cmd" &> /dev/null; then
        log::success "✓ $cmd available ($desc)"
    else
        log::warning "⚠ $cmd not available ($desc)"
    fi
done

# ============================================================================
# Java Dependencies
# ============================================================================

log::info "Checking Java dependencies"

if command -v java &> /dev/null; then
    java_version=$(java -version 2>&1 | head -1)
    log::success "Java available: $java_version"

    # Check Java version (need 11+)
    if java -version 2>&1 | grep -q "version \"1[1-9]\|version \"[2-9]"; then
        log::success "Java version is 11 or higher"
    else
        log::warning "Java version may be too old (need 11+)"
    fi
else
    log::warning "Java not available - required for Android builds"
fi

# Check for javac
if command -v javac &> /dev/null; then
    log::success "Java compiler (javac) available"
else
    log::warning "Java compiler not available"
fi

# ============================================================================
# Gradle Dependencies
# ============================================================================

log::info "Checking Gradle dependencies"

if command -v gradle &> /dev/null; then
    gradle_version=$(gradle --version 2>/dev/null | grep "Gradle" | awk '{print $2}')
    log::success "Gradle available: $gradle_version"

    # Check Gradle version (need 7.0+)
    if [ -n "$gradle_version" ]; then
        major_version=$(echo "$gradle_version" | cut -d. -f1)
        if [ "$major_version" -ge 7 ]; then
            log::success "Gradle version is 7.0 or higher"
        else
            log::warning "Gradle version may be too old (need 7.0+)"
        fi
    fi
else
    log::warning "Gradle not available - will use wrapper if available"
fi

# ============================================================================
# Kotlin Dependencies
# ============================================================================

log::info "Checking Kotlin dependencies"

if command -v kotlinc &> /dev/null; then
    kotlin_version=$(kotlinc -version 2>&1)
    log::success "Kotlin compiler available: $kotlin_version"
else
    log::warning "Kotlin compiler not available (Gradle will download if needed)"
fi

# ============================================================================
# Template Dependencies
# ============================================================================

log::info "Checking template dependencies"

TEMPLATES_DIR="initialization/templates/android"

if [ -d "$TEMPLATES_DIR" ]; then
    # Count template files
    template_count=$(find "$TEMPLATES_DIR" -type f | wc -l)
    log::success "Android templates present ($template_count files)"

    # Check for Gradle wrapper in templates
    if [ -f "$TEMPLATES_DIR/gradlew" ]; then
        log::success "Gradle wrapper included in templates"
    else
        log::warning "Gradle wrapper not in templates (will be generated)"
    fi
else
    testing::phase::add_error "Templates directory not found"
fi

# ============================================================================
# Resource Dependencies
# ============================================================================

log::info "Checking resource dependencies (from service.json)"

if [ -f ".vrooli/service.json" ]; then
    if command -v jq &> /dev/null; then
        # Check required resources
        required_resources=$(jq -r '.resources.required[]?.name // empty' .vrooli/service.json 2>/dev/null)

        if [ -n "$required_resources" ]; then
            log::info "Required resources:"
            echo "$required_resources" | while read -r resource; do
                log::info "  - $resource"
            done
        else
            log::info "No required resources specified"
        fi

        # Check optional resources
        optional_resources=$(jq -r '.resources.optional[]?.name // empty' .vrooli/service.json 2>/dev/null)

        if [ -n "$optional_resources" ]; then
            log::info "Optional resources:"
            echo "$optional_resources" | while read -r resource; do
                log::info "  - $resource"
            done
        fi
    fi
else
    testing::phase::add_error "service.json not found"
fi

# ============================================================================
# Disk Space Check
# ============================================================================

log::info "Checking disk space"

# Get available disk space in the current directory
available_space=$(df -BG . | tail -1 | awk '{print $4}' | sed 's/G//')

if [ "$available_space" -ge 5 ]; then
    log::success "Sufficient disk space available (${available_space}GB >= 5GB)"
else
    log::warning "Low disk space (${available_space}GB < 5GB recommended)"
fi

# ============================================================================
# Memory Check
# ============================================================================

log::info "Checking system memory"

if [ -f /proc/meminfo ]; then
    total_mem=$(grep MemTotal /proc/meminfo | awk '{print int($2/1024/1024)}')
    log::info "Total system memory: ${total_mem}GB"

    if [ "$total_mem" -ge 2 ]; then
        log::success "Sufficient memory for builds (${total_mem}GB >= 2GB)"
    else
        log::warning "Low memory (${total_mem}GB < 2GB recommended)"
    fi
else
    log::info "Cannot determine system memory (not on Linux)"
fi

# ============================================================================
# Network Connectivity Check
# ============================================================================

log::info "Checking network connectivity (for dependency downloads)"

if command -v ping &> /dev/null; then
    if ping -c 1 -W 2 google.com &> /dev/null; then
        log::success "Network connectivity available"
    else
        log::warning "No network connectivity - may affect dependency downloads"
    fi
else
    log::info "ping command not available, skipping network check"
fi

# ============================================================================
# Dependency Summary
# ============================================================================

log::info "Generating dependency summary"

cat > "$TESTING_PHASE_SCENARIO_DIR/test/dependency-report.txt" << EOF
Scenario to Android - Dependency Check Report
Generated: $(date)

Required Dependencies:
- bash, sed, grep, find, awk: Core shell utilities

Android Build Dependencies:
- Android SDK: ${ANDROID_HOME:-"Not set"}
- Java: $(command -v java &> /dev/null && java -version 2>&1 | head -1 || echo "Not found")
- Gradle: $(command -v gradle &> /dev/null && gradle --version 2>&1 | grep Gradle || echo "Not found (will use wrapper)")

Optional Tools:
- jq: $(command -v jq &> /dev/null && echo "Available" || echo "Not available")
- shellcheck: $(command -v shellcheck &> /dev/null && echo "Available" || echo "Not available")
- bats: $(command -v bats &> /dev/null && echo "Available" || echo "Not available")

System Resources:
- Disk Space: $(df -BG . | tail -1 | awk '{print $4}')
- Memory: $([ -f /proc/meminfo ] && grep MemTotal /proc/meminfo | awk '{print int($2/1024/1024)"GB"}' || echo "Unknown")

Status:
- Core functionality: $([ ${#MISSING_COMMANDS[@]} -eq 0 ] && echo "Ready" || echo "Missing dependencies")
- Android builds: $([ -n "${ANDROID_HOME:-}" ] && [ -d "${ANDROID_HOME:-}" ] && echo "Ready" || echo "Not configured")

EOF

log::success "Dependency report saved to test/dependency-report.txt"

# End with summary
testing::phase::end_with_summary "Dependency tests completed"
