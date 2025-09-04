#!/bin/bash

# Test script for scenario-to-ios
# Validates the iOS app generation functionality

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_SCENARIO="${1:-hello-world}"
OUTPUT_DIR="/tmp/ios-test-build"
TEST_RESULTS=()
FAILED_TESTS=0

# Function to log test results
log_test() {
    local test_name="$1"
    local result="$2"
    local details="$3"
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓ ${test_name}${NC}"
        TEST_RESULTS+=("✓ ${test_name}")
    else
        echo -e "${RED}✗ ${test_name}${NC}"
        [ -n "$details" ] && echo -e "  ${YELLOW}${details}${NC}"
        TEST_RESULTS+=("✗ ${test_name}: ${details}")
        ((FAILED_TESTS++))
    fi
}

# Function to check prerequisites
check_prerequisites() {
    echo -e "${BLUE}Checking prerequisites...${NC}"
    
    # Check for macOS
    if [[ "$OSTYPE" != "darwin"* ]]; then
        log_test "macOS Check" "FAIL" "iOS development requires macOS"
        return 1
    fi
    log_test "macOS Check" "PASS"
    
    # Check for Xcode
    if command -v xcodebuild &> /dev/null; then
        log_test "Xcode Installation" "PASS"
    else
        log_test "Xcode Installation" "FAIL" "Xcode not found"
        return 1
    fi
    
    # Check Xcode version
    XCODE_VERSION=$(xcodebuild -version | head -n 1 | awk '{print $2}')
    XCODE_MAJOR=$(echo $XCODE_VERSION | cut -d. -f1)
    
    if [ "$XCODE_MAJOR" -ge 14 ]; then
        log_test "Xcode Version" "PASS"
    else
        log_test "Xcode Version" "FAIL" "Xcode 14+ required (found $XCODE_VERSION)"
    fi
    
    # Check for Swift
    if command -v swift &> /dev/null; then
        log_test "Swift Installation" "PASS"
    else
        log_test "Swift Installation" "FAIL" "Swift not found"
    fi
}

# Test 1: Directory Structure
test_directory_structure() {
    echo -e "\n${BLUE}Testing directory structure...${NC}"
    
    # Check required directories
    [ -d "$SCENARIO_DIR/api" ] && log_test "API directory" "PASS" || log_test "API directory" "FAIL" "Missing api/"
    [ -d "$SCENARIO_DIR/cli" ] && log_test "CLI directory" "PASS" || log_test "CLI directory" "FAIL" "Missing cli/"
    [ -d "$SCENARIO_DIR/initialization" ] && log_test "Initialization directory" "PASS" || log_test "Initialization directory" "FAIL" "Missing initialization/"
    [ -d "$SCENARIO_DIR/initialization/templates/ios" ] && log_test "iOS templates" "PASS" || log_test "iOS templates" "FAIL" "Missing templates/ios/"
    [ -d "$SCENARIO_DIR/initialization/prompts" ] && log_test "Prompts directory" "PASS" || log_test "Prompts directory" "FAIL" "Missing prompts/"
}

# Test 2: Required Files
test_required_files() {
    echo -e "\n${BLUE}Testing required files...${NC}"
    
    # Check documentation
    [ -f "$SCENARIO_DIR/PRD.md" ] && log_test "PRD.md" "PASS" || log_test "PRD.md" "FAIL" "Missing PRD.md"
    [ -f "$SCENARIO_DIR/README.md" ] && log_test "README.md" "PASS" || log_test "README.md" "FAIL" "Missing README.md"
    
    # Check CLI files
    [ -f "$SCENARIO_DIR/cli/scenario-to-ios" ] && log_test "CLI executable" "PASS" || log_test "CLI executable" "FAIL" "Missing cli/scenario-to-ios"
    [ -f "$SCENARIO_DIR/cli/install.sh" ] && log_test "Install script" "PASS" || log_test "Install script" "FAIL" "Missing cli/install.sh"
    
    # Check prompts
    [ -f "$SCENARIO_DIR/initialization/prompts/ios-app-creator.md" ] && log_test "Creator prompt" "PASS" || log_test "Creator prompt" "FAIL"
    [ -f "$SCENARIO_DIR/initialization/prompts/ios-app-debugger.md" ] && log_test "Debugger prompt" "PASS" || log_test "Debugger prompt" "FAIL"
    
    # Check template files
    [ -f "$SCENARIO_DIR/initialization/templates/ios/project.xcodeproj/project.pbxproj" ] && log_test "Xcode project" "PASS" || log_test "Xcode project" "FAIL"
    [ -f "$SCENARIO_DIR/initialization/templates/ios/project/VrooliScenarioApp.swift" ] && log_test "App Swift file" "PASS" || log_test "App Swift file" "FAIL"
    [ -f "$SCENARIO_DIR/initialization/templates/ios/project/ContentView.swift" ] && log_test "ContentView" "PASS" || log_test "ContentView" "FAIL"
    [ -f "$SCENARIO_DIR/initialization/templates/ios/project/WebView.swift" ] && log_test "WebView" "PASS" || log_test "WebView" "FAIL"
    [ -f "$SCENARIO_DIR/initialization/templates/ios/project/JSBridge.swift" ] && log_test "JSBridge" "PASS" || log_test "JSBridge" "FAIL"
}

# Test 3: CLI Functionality
test_cli_functionality() {
    echo -e "\n${BLUE}Testing CLI functionality...${NC}"
    
    # Make CLI executable
    chmod +x "$SCENARIO_DIR/cli/scenario-to-ios"
    
    # Test help command
    if "$SCENARIO_DIR/cli/scenario-to-ios" help &> /dev/null; then
        log_test "CLI help command" "PASS"
    else
        log_test "CLI help command" "FAIL" "Help command failed"
    fi
    
    # Test version command
    if "$SCENARIO_DIR/cli/scenario-to-ios" version &> /dev/null; then
        log_test "CLI version command" "PASS"
    else
        log_test "CLI version command" "FAIL" "Version command failed"
    fi
    
    # Test status command
    if "$SCENARIO_DIR/cli/scenario-to-ios" status &> /dev/null; then
        log_test "CLI status command" "PASS"
    else
        log_test "CLI status command" "FAIL" "Status command failed"
    fi
}

# Test 4: Template Validation
test_template_validation() {
    echo -e "\n${BLUE}Testing iOS template validation...${NC}"
    
    # Check Swift syntax in template files
    local swift_files=(
        "$SCENARIO_DIR/initialization/templates/ios/project/VrooliScenarioApp.swift"
        "$SCENARIO_DIR/initialization/templates/ios/project/ContentView.swift"
        "$SCENARIO_DIR/initialization/templates/ios/project/WebView.swift"
        "$SCENARIO_DIR/initialization/templates/ios/project/JSBridge.swift"
    )
    
    for file in "${swift_files[@]}"; do
        if [ -f "$file" ]; then
            # Check for basic Swift syntax markers
            if grep -q "import SwiftUI\|import UIKit\|import Foundation" "$file"; then
                log_test "$(basename $file) syntax" "PASS"
            else
                log_test "$(basename $file) syntax" "FAIL" "Missing Swift imports"
            fi
        fi
    done
    
    # Validate Info.plist structure
    if [ -f "$SCENARIO_DIR/initialization/templates/ios/project/Info.plist" ]; then
        if xmllint --noout "$SCENARIO_DIR/initialization/templates/ios/project/Info.plist" 2> /dev/null; then
            log_test "Info.plist validation" "PASS"
        else
            log_test "Info.plist validation" "FAIL" "Invalid XML structure"
        fi
    else
        log_test "Info.plist validation" "FAIL" "File not found"
    fi
}

# Test 5: Build Test (if possible)
test_build_process() {
    echo -e "\n${BLUE}Testing build process...${NC}"
    
    # Only run if Xcode is available
    if ! command -v xcodebuild &> /dev/null; then
        log_test "Build test" "SKIP" "Xcode not available"
        return
    fi
    
    # Create temporary build directory
    rm -rf "$OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR"
    
    # Copy template to test directory
    cp -r "$SCENARIO_DIR/initialization/templates/ios/" "$OUTPUT_DIR/"
    
    # Replace placeholders
    find "$OUTPUT_DIR" -type f \( -name "*.swift" -o -name "*.plist" -o -name "*.pbxproj" \) | while read file; do
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/{{SCENARIO_NAME}}/test-scenario/g" "$file"
            sed -i '' "s/{{APP_NAME}}/Test App/g" "$file"
            sed -i '' "s/{{APP_VERSION}}/1.0.0/g" "$file"
            sed -i '' "s/{{BUILD_NUMBER}}/1/g" "$file"
        else
            sed -i "s/{{SCENARIO_NAME}}/test-scenario/g" "$file"
            sed -i "s/{{APP_NAME}}/Test App/g" "$file"
            sed -i "s/{{APP_VERSION}}/1.0.0/g" "$file"
            sed -i "s/{{BUILD_NUMBER}}/1/g" "$file"
        fi
    done
    
    # Try to build for simulator (doesn't require signing)
    cd "$OUTPUT_DIR"
    if xcodebuild -project project.xcodeproj \
        -scheme VrooliScenario \
        -configuration Debug \
        -sdk iphonesimulator \
        -derivedDataPath ./build \
        CODE_SIGN_IDENTITY="" \
        CODE_SIGNING_REQUIRED=NO \
        build &> /tmp/ios-build.log; then
        log_test "Xcode build" "PASS"
    else
        log_test "Xcode build" "FAIL" "Check /tmp/ios-build.log for details"
    fi
    cd - > /dev/null
}

# Test 6: JavaScript Bridge Validation
test_javascript_bridge() {
    echo -e "\n${BLUE}Testing JavaScript bridge...${NC}"
    
    local bridge_file="$SCENARIO_DIR/initialization/templates/ios/project/JSBridge.swift"
    
    if [ -f "$bridge_file" ]; then
        # Check for required bridge methods
        local methods=("getDeviceInfo" "requestNotificationPermission" "saveToKeychain" "authenticateWithBiometrics" "hapticFeedback")
        
        for method in "${methods[@]}"; do
            if grep -q "$method" "$bridge_file"; then
                log_test "Bridge method: $method" "PASS"
            else
                log_test "Bridge method: $method" "FAIL" "Method not found in JSBridge.swift"
            fi
        done
    else
        log_test "JavaScript bridge" "FAIL" "JSBridge.swift not found"
    fi
}

# Test 7: Prompt Validation
test_prompts() {
    echo -e "\n${BLUE}Testing AI prompts...${NC}"
    
    # Check creator prompt
    local creator_prompt="$SCENARIO_DIR/initialization/prompts/ios-app-creator.md"
    if [ -f "$creator_prompt" ]; then
        # Check for required sections
        if grep -q "Your Task\|Required Deliverables\|Success Criteria" "$creator_prompt"; then
            log_test "Creator prompt structure" "PASS"
        else
            log_test "Creator prompt structure" "FAIL" "Missing required sections"
        fi
    else
        log_test "Creator prompt" "FAIL" "File not found"
    fi
    
    # Check debugger prompt
    local debugger_prompt="$SCENARIO_DIR/initialization/prompts/ios-app-debugger.md"
    if [ -f "$debugger_prompt" ]; then
        if grep -q "Common Issues and Solutions\|Debugging Tools" "$debugger_prompt"; then
            log_test "Debugger prompt structure" "PASS"
        else
            log_test "Debugger prompt structure" "FAIL" "Missing required sections"
        fi
    else
        log_test "Debugger prompt" "FAIL" "File not found"
    fi
}

# Main test execution
main() {
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}          scenario-to-ios Test Suite${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    
    # Run all tests
    check_prerequisites
    test_directory_structure
    test_required_files
    test_cli_functionality
    test_template_validation
    test_javascript_bridge
    test_prompts
    
    # Only run build test on macOS with Xcode
    if [[ "$OSTYPE" == "darwin"* ]] && command -v xcodebuild &> /dev/null; then
        test_build_process
    fi
    
    # Print summary
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}                    Test Summary${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════${NC}"
    
    local total_tests=${#TEST_RESULTS[@]}
    local passed_tests=$((total_tests - FAILED_TESTS))
    
    echo -e "\nTotal Tests: ${total_tests}"
    echo -e "${GREEN}Passed: ${passed_tests}${NC}"
    echo -e "${RED}Failed: ${FAILED_TESTS}${NC}"
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "\n${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}✗ Some tests failed. See details above.${NC}"
        
        # List failed tests
        echo -e "\n${YELLOW}Failed tests:${NC}"
        for result in "${TEST_RESULTS[@]}"; do
            if [[ "$result" == *"✗"* ]]; then
                echo -e "  ${result}"
            fi
        done
        exit 1
    fi
}

# Cleanup function
cleanup() {
    if [ -d "$OUTPUT_DIR" ]; then
        echo -e "\n${BLUE}Cleaning up test files...${NC}"
        rm -rf "$OUTPUT_DIR"
    fi
}

# Set up trap for cleanup
trap cleanup EXIT

# Run main function
main "$@"