#!/bin/bash

# Install script for scenario-to-ios CLI
# This script creates necessary symlinks and sets up the environment

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Installation directories
VROOLI_BIN_DIR="$HOME/.vrooli/bin"
CLI_NAME="scenario-to-ios"
CLI_PATH="$SCRIPT_DIR/$CLI_NAME"

echo -e "${BLUE}Installing scenario-to-ios CLI...${NC}"

# Check if running on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo -e "${RED}Error: scenario-to-ios requires macOS with Xcode installed${NC}"
    echo "iOS development is only supported on macOS"
    exit 1
fi

# Check for Xcode
if ! command -v xcodebuild &> /dev/null; then
    echo -e "${RED}Error: Xcode is not installed${NC}"
    echo "Please install Xcode from the Mac App Store before running this installer"
    echo "After installing Xcode, run: xcode-select --install"
    exit 1
fi

# Check Xcode command line tools
if ! xcode-select -p &> /dev/null; then
    echo -e "${YELLOW}Xcode command line tools not configured${NC}"
    echo "Running: xcode-select --install"
    xcode-select --install
    echo "Please complete the installation and run this script again"
    exit 1
fi

# Create the .vrooli/bin directory if it doesn't exist
mkdir -p "$VROOLI_BIN_DIR"

# Make the CLI executable
chmod +x "$CLI_PATH"

# Create symlink
if [ -L "$VROOLI_BIN_DIR/$CLI_NAME" ] || [ -f "$VROOLI_BIN_DIR/$CLI_NAME" ]; then
    echo -e "${YELLOW}Removing existing $CLI_NAME installation...${NC}"
    rm -f "$VROOLI_BIN_DIR/$CLI_NAME"
fi

ln -s "$CLI_PATH" "$VROOLI_BIN_DIR/$CLI_NAME"
echo -e "${GREEN}✓ Created symlink: $VROOLI_BIN_DIR/$CLI_NAME${NC}"

# Check if .vrooli/bin is in PATH
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    echo -e "${YELLOW}Adding $VROOLI_BIN_DIR to PATH...${NC}"
    
    # Determine which shell configuration file to use
    if [ -f "$HOME/.zshrc" ]; then
        SHELL_RC="$HOME/.zshrc"
    elif [ -f "$HOME/.bashrc" ]; then
        SHELL_RC="$HOME/.bashrc"
    elif [ -f "$HOME/.bash_profile" ]; then
        SHELL_RC="$HOME/.bash_profile"
    else
        SHELL_RC="$HOME/.profile"
    fi
    
    # Add PATH export to shell configuration
    echo "" >> "$SHELL_RC"
    echo "# Added by scenario-to-ios installer" >> "$SHELL_RC"
    echo "export PATH=\"\$HOME/.vrooli/bin:\$PATH\"" >> "$SHELL_RC"
    
    echo -e "${GREEN}✓ Added to PATH in $SHELL_RC${NC}"
    echo -e "${YELLOW}Please run: source $SHELL_RC${NC}"
    echo -e "${YELLOW}Or restart your terminal for PATH changes to take effect${NC}"
else
    echo -e "${GREEN}✓ $VROOLI_BIN_DIR already in PATH${NC}"
fi

# Check for iOS development prerequisites
echo -e "\n${BLUE}Checking iOS development prerequisites...${NC}"

# Check Xcode version
XCODE_VERSION=$(xcodebuild -version | head -n 1 | awk '{print $2}')
XCODE_MAJOR=$(echo $XCODE_VERSION | cut -d. -f1)

if [ "$XCODE_MAJOR" -lt 14 ]; then
    echo -e "${YELLOW}Warning: Xcode $XCODE_VERSION detected. Xcode 14+ recommended${NC}"
else
    echo -e "${GREEN}✓ Xcode $XCODE_VERSION${NC}"
fi

# Check for Swift
if command -v swift &> /dev/null; then
    SWIFT_VERSION=$(swift --version | head -n 1 | awk '{print $4}')
    echo -e "${GREEN}✓ Swift $SWIFT_VERSION${NC}"
else
    echo -e "${RED}✗ Swift not found${NC}"
fi

# Check for valid developer certificate
if security find-identity -p codesigning -v 2>/dev/null | grep -q "valid identities found"; then
    echo -e "${GREEN}✓ Code signing certificates found${NC}"
else
    echo -e "${YELLOW}⚠ No valid code signing certificates found${NC}"
    echo -e "${YELLOW}  You'll need to set up certificates for App Store distribution${NC}"
    echo -e "${YELLOW}  Visit: https://developer.apple.com/account${NC}"
fi

# Check for CocoaPods (optional but useful)
if command -v pod &> /dev/null; then
    POD_VERSION=$(pod --version)
    echo -e "${GREEN}✓ CocoaPods $POD_VERSION${NC}"
else
    echo -e "${YELLOW}⚠ CocoaPods not installed (optional)${NC}"
    echo -e "${YELLOW}  Install with: sudo gem install cocoapods${NC}"
fi

# Check for fastlane (optional but useful)
if command -v fastlane &> /dev/null; then
    FASTLANE_VERSION=$(fastlane --version | head -n 1 | awk '{print $2}')
    echo -e "${GREEN}✓ fastlane $FASTLANE_VERSION${NC}"
else
    echo -e "${YELLOW}⚠ fastlane not installed (optional)${NC}"
    echo -e "${YELLOW}  Install with: sudo gem install fastlane${NC}"
fi

# Create configuration directory
CONFIG_DIR="$HOME/.vrooli/scenario-to-ios"
mkdir -p "$CONFIG_DIR"

# Create default configuration file if it doesn't exist
CONFIG_FILE="$CONFIG_DIR/config.yaml"
if [ ! -f "$CONFIG_FILE" ]; then
    cat > "$CONFIG_FILE" << EOF
# scenario-to-ios configuration
# Edit these values for your Apple Developer account

# Apple Developer Team ID (found in Apple Developer Portal)
developer_team: ""

# Default build settings
build:
  configuration: Release
  sdk: iphoneos
  deployment_target: "15.0"

# TestFlight settings
testflight:
  # Your Apple ID email
  apple_id: ""
  # App-specific password (generate at appleid.apple.com)
  app_password: ""

# Default device for simulator testing
simulator:
  device: "iPhone 14"
  ios_version: "latest"
EOF
    echo -e "${GREEN}✓ Created configuration file: $CONFIG_FILE${NC}"
    echo -e "${YELLOW}  Edit this file to add your Apple Developer credentials${NC}"
fi

echo -e "\n${GREEN}════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ scenario-to-ios installation complete!${NC}"
echo -e "${GREEN}════════════════════════════════════════════════════════${NC}"
echo -e "\n${BLUE}Quick Start:${NC}"
echo -e "  1. Configure your Apple Developer account:"
echo -e "     ${YELLOW}nano $CONFIG_FILE${NC}"
echo -e "  2. Build your first iOS app:"
echo -e "     ${YELLOW}scenario-to-ios build <scenario-name>${NC}"
echo -e "  3. Test in simulator:"
echo -e "     ${YELLOW}scenario-to-ios simulator <scenario-name>${NC}"
echo -e "\n${BLUE}For help:${NC}"
echo -e "  ${YELLOW}scenario-to-ios help${NC}"

# Check if we need to source the shell config
if [[ ":$PATH:" != *":$VROOLI_BIN_DIR:"* ]]; then
    echo -e "\n${RED}IMPORTANT: Run this command to update your PATH:${NC}"
    echo -e "  ${YELLOW}source $SHELL_RC${NC}"
fi