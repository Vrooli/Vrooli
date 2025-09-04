#!/bin/bash

# Scenario to Android Converter
# Transforms any Vrooli scenario into a native Android application

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TEMPLATES_DIR="${SCRIPT_DIR}/../initialization/templates/android"
OUTPUT_DIR="/tmp/android-builds"
SCENARIO_NAME=""
SCENARIO_PATH=""
APP_NAME=""
PACKAGE_SUFFIX=""
VERSION_NAME="1.0.0"
VERSION_CODE=1
SIGN_APK=false
KEYSTORE_PATH=""
KEY_ALIAS=""
OPTIMIZE=false

# Function to print colored output
print_color() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 --scenario <name> [options]

Convert a Vrooli scenario to an Android application.

Options:
    --scenario <name>      Name of the scenario to convert (required)
    --output <dir>         Output directory (default: /tmp/android-builds)
    --app-name <name>      App display name (default: scenario name)
    --version <version>    App version (default: 1.0.0)
    --sign                 Sign the APK with a keystore
    --keystore <path>      Path to keystore file (required if --sign)
    --key-alias <alias>    Key alias in keystore (required if --sign)
    --optimize            Optimize APK with ProGuard/R8
    --help                Show this help message

Examples:
    $0 --scenario study-buddy
    $0 --scenario notes --app-name "My Notes" --version 2.1.0 --sign
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --scenario)
            SCENARIO_NAME="$2"
            shift 2
            ;;
        --output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --app-name)
            APP_NAME="$2"
            shift 2
            ;;
        --version)
            VERSION_NAME="$2"
            shift 2
            ;;
        --sign)
            SIGN_APK=true
            shift
            ;;
        --keystore)
            KEYSTORE_PATH="$2"
            shift 2
            ;;
        --key-alias)
            KEY_ALIAS="$2"
            shift 2
            ;;
        --optimize)
            OPTIMIZE=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_color $RED "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [ -z "$SCENARIO_NAME" ]; then
    print_color $RED "Error: --scenario is required"
    show_usage
    exit 1
fi

# Find scenario path
SCENARIO_PATH="${SCRIPT_DIR}/../../${SCENARIO_NAME}"
if [ ! -d "$SCENARIO_PATH" ]; then
    # Try absolute path
    SCENARIO_PATH="/home/matthalloran8/Vrooli/scenarios/${SCENARIO_NAME}"
    if [ ! -d "$SCENARIO_PATH" ]; then
        print_color $RED "Error: Scenario '${SCENARIO_NAME}' not found"
        exit 1
    fi
fi

# Set default app name if not provided
if [ -z "$APP_NAME" ]; then
    APP_NAME="${SCENARIO_NAME}"
    # Convert kebab-case to Title Case
    APP_NAME=$(echo "$APP_NAME" | sed 's/-/ /g' | sed 's/\b\(.\)/\u\1/g')
fi

# Generate package suffix from scenario name
PACKAGE_SUFFIX=$(echo "$SCENARIO_NAME" | sed 's/-/_/g' | tr '[:upper:]' '[:lower:]')

# Extract version code from version name (last number)
VERSION_CODE=$(echo "$VERSION_NAME" | sed 's/[^0-9]//g' | tail -c 3)
if [ -z "$VERSION_CODE" ]; then
    VERSION_CODE=1
fi

# Validate signing configuration
if $SIGN_APK; then
    if [ -z "$KEYSTORE_PATH" ] || [ -z "$KEY_ALIAS" ]; then
        print_color $RED "Error: --keystore and --key-alias are required when using --sign"
        exit 1
    fi
    if [ ! -f "$KEYSTORE_PATH" ]; then
        print_color $RED "Error: Keystore file not found: $KEYSTORE_PATH"
        exit 1
    fi
fi

print_color $BLUE "========================================="
print_color $BLUE "   Scenario to Android Converter"
print_color $BLUE "========================================="
echo ""
print_color $GREEN "Scenario: $SCENARIO_NAME"
print_color $GREEN "App Name: $APP_NAME"
print_color $GREEN "Version: $VERSION_NAME (code: $VERSION_CODE)"
print_color $GREEN "Package: com.vrooli.scenario.$PACKAGE_SUFFIX"
echo ""

# Create output directory
PROJECT_DIR="${OUTPUT_DIR}/${SCENARIO_NAME}-android"
rm -rf "$PROJECT_DIR"
mkdir -p "$PROJECT_DIR"

print_color $YELLOW "→ Copying Android template..."
cp -r "$TEMPLATES_DIR"/* "$PROJECT_DIR/"

# Function to replace template variables
replace_vars() {
    local file="$1"
    
    # Create temporary file for replacements
    local temp_file="${file}.tmp"
    
    # Perform replacements
    sed -e "s/{{SCENARIO_NAME}}/${SCENARIO_NAME}/g" \
        -e "s/{{SCENARIO_NAME_PACKAGE}}/${PACKAGE_SUFFIX}/g" \
        -e "s/{{APP_NAME}}/${APP_NAME}/g" \
        -e "s/{{VERSION_NAME}}/${VERSION_NAME}/g" \
        -e "s/{{VERSION_CODE}}/${VERSION_CODE}/g" \
        -e "s/{{THEME_COLOR}}/2196F3/g" \
        -e "s/{{API_URL}}/http:\/\/localhost:8080/g" \
        -e "s/{{OFFLINE_MODE}}/true/g" \
        -e "s/{{ENABLE_ABI_SPLIT}}/false/g" \
        -e "s/{{ABI_FILTERS}}/\"arm64-v8a\", \"armeabi-v7a\"/g" \
        "$file" > "$temp_file"
    
    # Handle conditional sections (remove markers)
    sed -i 's/{{#if [A-Z_]*}}//g' "$temp_file"
    sed -i 's/{{\/if}}//g' "$temp_file"
    
    mv "$temp_file" "$file"
}

print_color $YELLOW "→ Processing template files..."

# Process all template files
find "$PROJECT_DIR" -type f \( -name "*.kt" -o -name "*.java" -o -name "*.xml" -o -name "*.gradle" -o -name "*.html" \) | while read file; do
    replace_vars "$file"
done

# Copy scenario UI files to assets
print_color $YELLOW "→ Copying scenario UI files..."
SCENARIO_UI_DIR="${SCENARIO_PATH}/ui"
if [ -d "$SCENARIO_UI_DIR" ]; then
    # Create assets directory structure
    ASSETS_DIR="${PROJECT_DIR}/app/src/main/assets"
    mkdir -p "$ASSETS_DIR"
    
    # Copy UI files
    cp -r "$SCENARIO_UI_DIR"/* "$ASSETS_DIR/" 2>/dev/null || true
    
    # Check for index.html
    if [ ! -f "$ASSETS_DIR/index.html" ]; then
        print_color $YELLOW "  Note: No index.html found in scenario UI, using template"
    fi
else
    print_color $YELLOW "  Warning: No UI directory found in scenario"
fi

# Copy scenario data/config if exists
SCENARIO_DATA_DIR="${SCENARIO_PATH}/data"
if [ -d "$SCENARIO_DATA_DIR" ]; then
    print_color $YELLOW "→ Copying scenario data..."
    cp -r "$SCENARIO_DATA_DIR" "$ASSETS_DIR/data" 2>/dev/null || true
fi

# Generate gradle.properties
print_color $YELLOW "→ Generating gradle configuration..."
cat > "$PROJECT_DIR/gradle.properties" << EOF
# Project-wide Gradle settings.
org.gradle.jvmargs=-Xmx2048m -Dfile.encoding=UTF-8
android.useAndroidX=true
kotlin.code.style=official
android.nonTransitiveRClass=true
android.enableJetifier=true
EOF

# Generate local.properties if Android SDK is available
if [ -n "$ANDROID_HOME" ]; then
    cat > "$PROJECT_DIR/local.properties" << EOF
sdk.dir=$ANDROID_HOME
EOF
elif [ -d "$HOME/Android/Sdk" ]; then
    cat > "$PROJECT_DIR/local.properties" << EOF
sdk.dir=$HOME/Android/Sdk
EOF
fi

# Add signing configuration if requested
if $SIGN_APK; then
    print_color $YELLOW "→ Configuring APK signing..."
    
    # Copy keystore to project
    cp "$KEYSTORE_PATH" "$PROJECT_DIR/keystore.jks"
    
    # Update build.gradle with signing config
    sed -i "s/{{#if SIGNING_CONFIG}}//" "$PROJECT_DIR/app/build.gradle"
    sed -i "s/{{\/if}}//" "$PROJECT_DIR/app/build.gradle"
    sed -i "s/{{KEYSTORE_PATH}}/..\/keystore.jks/g" "$PROJECT_DIR/app/build.gradle"
    
    # Prompt for passwords if not in environment
    if [ -z "$KEYSTORE_PASSWORD" ]; then
        read -sp "Enter keystore password: " KEYSTORE_PASSWORD
        echo
    fi
    if [ -z "$KEY_PASSWORD" ]; then
        read -sp "Enter key password: " KEY_PASSWORD
        echo
    fi
    
    sed -i "s/{{KEYSTORE_PASSWORD}}/$KEYSTORE_PASSWORD/g" "$PROJECT_DIR/app/build.gradle"
    sed -i "s/{{KEY_ALIAS}}/$KEY_ALIAS/g" "$PROJECT_DIR/app/build.gradle"
    sed -i "s/{{KEY_PASSWORD}}/$KEY_PASSWORD/g" "$PROJECT_DIR/app/build.gradle"
fi

# Create gradlew wrapper
print_color $YELLOW "→ Setting up Gradle wrapper..."
cd "$PROJECT_DIR"
if command -v gradle &> /dev/null; then
    gradle wrapper --gradle-version=8.2 --distribution-type=all &> /dev/null || true
fi

# Make gradlew executable
if [ -f "./gradlew" ]; then
    chmod +x ./gradlew
    GRADLE_CMD="./gradlew"
else
    GRADLE_CMD="gradle"
fi

# Check if we can build
if command -v $GRADLE_CMD &> /dev/null || [ -f "./gradlew" ]; then
    print_color $YELLOW "→ Building Android APK..."
    
    BUILD_TYPE="assembleDebug"
    if $SIGN_APK; then
        BUILD_TYPE="assembleRelease"
    fi
    
    # Add optimization flags if requested
    if $OPTIMIZE; then
        export ANDROID_OPTIMIZE=true
    fi
    
    # Attempt to build
    if $GRADLE_CMD $BUILD_TYPE; then
        print_color $GREEN "✓ Build successful!"
        
        # Find the generated APK
        APK_PATH=$(find "$PROJECT_DIR/app/build/outputs/apk" -name "*.apk" -type f | head -1)
        if [ -n "$APK_PATH" ]; then
            APK_NAME="${SCENARIO_NAME}-${VERSION_NAME}.apk"
            FINAL_APK="${OUTPUT_DIR}/${APK_NAME}"
            cp "$APK_PATH" "$FINAL_APK"
            
            print_color $GREEN ""
            print_color $GREEN "========================================="
            print_color $GREEN "   Conversion Complete!"
            print_color $GREEN "========================================="
            print_color $GREEN "APK Location: $FINAL_APK"
            print_color $GREEN "Project Location: $PROJECT_DIR"
            
            # Show APK info
            if command -v aapt &> /dev/null; then
                print_color $BLUE ""
                print_color $BLUE "APK Information:"
                aapt dump badging "$FINAL_APK" | grep -E "package:|application-label:|launchable-activity:" | head -3
            fi
            
            print_color $YELLOW ""
            print_color $YELLOW "To install on device:"
            print_color $YELLOW "  adb install $FINAL_APK"
            print_color $YELLOW ""
            print_color $YELLOW "To test in emulator:"
            print_color $YELLOW "  emulator -avd <AVD_NAME>"
            print_color $YELLOW "  adb install $FINAL_APK"
        else
            print_color $YELLOW "Warning: APK built but not found in expected location"
        fi
    else
        print_color $RED "✗ Build failed. Check the error messages above."
        print_color $YELLOW "Project created at: $PROJECT_DIR"
        print_color $YELLOW "You can manually build it with Android Studio"
    fi
else
    print_color $YELLOW "→ Gradle not found. Skipping build step."
    print_color $GREEN ""
    print_color $GREEN "========================================="
    print_color $GREEN "   Project Created Successfully!"
    print_color $GREEN "========================================="
    print_color $GREEN "Project Location: $PROJECT_DIR"
    print_color $YELLOW ""
    print_color $YELLOW "To build the APK:"
    print_color $YELLOW "  1. Install Android Studio"
    print_color $YELLOW "  2. Open project: $PROJECT_DIR"
    print_color $YELLOW "  3. Build → Build APK"
fi

print_color $BLUE ""
print_color $BLUE "Next steps:"
print_color $BLUE "  - Test the app on various Android devices"
print_color $BLUE "  - Customize the UI and features as needed"
print_color $BLUE "  - Prepare for Google Play Store submission"

exit 0