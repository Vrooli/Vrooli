#!/usr/bin/env bash
set -euo pipefail

# Structure Verification Script
# Verifies all resources follow the standardized structure

RESOURCES_DIR="${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources"
ISSUES_FOUND=0

echo "🔍 Resource Structure Verification"
echo "=================================="
echo

# Function to check resource structure
check_resource() {
    local resource_path=$1
    local resource_name=$(basename "$resource_path")
    local category=$(basename $(dirname "$resource_path"))
    local issues=""
    
    # Check for non-standard root files
    for file in $(ls -1 "$resource_path" 2>/dev/null); do
        if [[ ! "$file" =~ ^(manage\.sh|manage\.bats|README\.md|inject\.sh|lib|config|docs|examples|docker|templates|schemas|instances|agent_s2|flows|nodes|policies|scripts|sandbox|integrations)$ ]]; then
            # Special cases for docker projects
            if [[ "$file" =~ ^(\.dockerignore|\.env\.example|\.gitignore|setup\.py|docker-compose.*\.yml)$ ]]; then
                continue  # These are acceptable for docker/python projects
            fi
            issues="${issues}    ⚠️  Non-standard file: $file\n"
            ((ISSUES_FOUND++))
        fi
    done
    
    # Check for test directories
    for testdir in test tests testing fixtures test_fixtures test-fixtures; do
        if [ -d "$resource_path/$testdir" ]; then
            issues="${issues}    ❌ Test directory found: $testdir/\n"
            ((ISSUES_FOUND++))
        fi
    done
    
    # Print results
    if [ -n "$issues" ]; then
        echo "📦 $category/$resource_name:"
        echo -e "$issues"
    fi
}

# Check all resources
for category in ai automation agents search storage execution; do
    if [ -d "$RESOURCES_DIR/$category" ]; then
        for resource in "$RESOURCES_DIR/$category"/*; do
            if [ -d "$resource" ]; then
                check_resource "$resource"
            fi
        done
    fi
done

echo
echo "=================================="
echo "📊 Verification Summary:"
echo "   Issues found: $ISSUES_FOUND"

if [ $ISSUES_FOUND -eq 0 ]; then
    echo "   ✅ All resources follow standard structure!"
else
    echo "   ⚠️  Some resources need attention"
fi