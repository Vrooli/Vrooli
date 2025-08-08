#!/bin/bash
#
# Routine Import Script using Vrooli CLI
# Imports generated routines from staged directory into local Vrooli instance
#
# Usage: ./routine-import.sh [directory]
#

set -e

# Default directory
IMPORT_DIR="${1:-./docs/ai-creation/routine/staged}"
SUBROUTINES_DIR="$IMPORT_DIR/subroutines"
MAIN_ROUTINES_DIR="$IMPORT_DIR/main-routines"

# Check if CLI is available and auto-build if needed
if ! command -v vrooli &> /dev/null; then
    echo "🔨 Vrooli CLI not found. Building and installing..."
    
    # Ensure we're in the right directory
    SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
    
    # Source var.sh first to get all directory variables
    # shellcheck disable=SC1091
    source "${SCRIPT_DIR}/../../../lib/utils/var.sh"
    
    PROJECT_ROOT="${var_ROOT_DIR}"
    
    # Verify CLI directory exists
    if [[ ! -d "$PROJECT_ROOT/packages/cli" ]]; then
        echo "❌ CLI package directory not found at: $PROJECT_ROOT/packages/cli"
        exit 1
    fi
    
    cd "$PROJECT_ROOT/packages/cli"
    
    # Build and link CLI with better error handling
    echo "📦 Installing dependencies..."
    if ! timeout 300 pnpm install --frozen-lockfile 2>&1; then
        echo "❌ Failed to install dependencies or operation timed out"
        echo "💡 Try running 'cd packages/cli && pnpm install' manually"
        exit 1
    fi
    
    echo "🏗️  Building CLI..."
    if ! timeout 300 pnpm run build 2>&1; then
        echo "❌ Failed to build CLI or operation timed out"
        echo "💡 Try running 'cd packages/cli && pnpm run build' manually"
        exit 1
    fi
    
    echo "🔗 Linking CLI globally..."
    if ! npm link 2>&1; then
        echo "❌ Failed to link CLI globally"
        echo "💡 You may need to run with sudo or check npm permissions"
        echo "💡 Alternative: Add packages/cli/dist to your PATH"
        exit 1
    fi
    
    # Verify installation with timeout
    if ! timeout 10 command -v vrooli &> /dev/null; then
        echo "❌ CLI installation failed. Please check the build logs above."
        echo "💡 Try running 'vrooli --help' to test manually"
        exit 1
    fi
    
    echo "✅ CLI installed successfully"
    cd "$PROJECT_ROOT"
fi

# Check if logged in
if ! vrooli auth status &> /dev/null; then
    echo "🔐 Not authenticated. Logging in..."
    vrooli auth login
fi

# Check if directories exist and count files
if [[ ! -d "$IMPORT_DIR" ]]; then
    echo "❌ Import directory does not exist: $IMPORT_DIR"
    exit 1
fi

# Count files in each subdirectory
SUBROUTINES_COUNT=0
MAIN_ROUTINES_COUNT=0
LEGACY_COUNT=0

if [[ -d "$SUBROUTINES_DIR" ]]; then
    SUBROUTINES_COUNT=$(find "$SUBROUTINES_DIR" -name "*.json" -type f | wc -l)
fi

if [[ -d "$MAIN_ROUTINES_DIR" ]]; then
    MAIN_ROUTINES_COUNT=$(find "$MAIN_ROUTINES_DIR" -name "*.json" -type f | wc -l)
fi

# Count legacy files in root staged directory
LEGACY_COUNT=$(find "$IMPORT_DIR" -maxdepth 1 -name "*.json" -type f | wc -l)

TOTAL_COUNT=$((SUBROUTINES_COUNT + MAIN_ROUTINES_COUNT + LEGACY_COUNT))

if [[ $TOTAL_COUNT -eq 0 ]]; then
    echo "⚠️  No JSON files found in: $IMPORT_DIR"
    echo "Run ./routine-generate-enhanced.sh first to create routines"
    exit 0
fi

echo "📋 Found routine files:"
if [[ $SUBROUTINES_COUNT -gt 0 ]]; then
    echo "  - Subroutines: $SUBROUTINES_COUNT"
fi
if [[ $MAIN_ROUTINES_COUNT -gt 0 ]]; then
    echo "  - Main routines: $MAIN_ROUTINES_COUNT"
fi
if [[ $LEGACY_COUNT -gt 0 ]]; then
    echo "  - Legacy files: $LEGACY_COUNT"
fi
echo "  - Total: $TOTAL_COUNT"

# Function: Validate files in a directory
validate_directory() {
    local dir="$1"
    local dir_name="$2"
    local validation_failed=0
    local files_validated=0
    
    if [[ ! -d "$dir" ]]; then
        return 0
    fi
    
    echo "📂 Validating $dir_name..."
    
    while IFS= read -r -d '' json_file; do
        filename=$(basename "$json_file")
        echo "  Validating: $filename"
        
        # First check if it's valid JSON
        if ! jq empty "$json_file" 2>/dev/null; then
            echo "    ❌ Invalid JSON: $filename"
            validation_failed=1
        elif ! timeout 30 vrooli routine validate "$json_file" --quiet 2>/dev/null; then
            echo "    ❌ Validation failed: $filename"
            validation_failed=1
        else
            echo "    ✅ Valid: $filename"
        fi
        ((files_validated++))
    done < <(find "$dir" -maxdepth 1 -name "*.json" -type f -print0 2>/dev/null)
    
    if [[ $files_validated -gt 0 ]]; then
        if [[ $validation_failed -eq 1 ]]; then
            echo "  ❌ Some $dir_name failed validation"
        else
            echo "  ✅ All $files_validated $dir_name passed validation"
        fi
    fi
    
    return $validation_failed
}

# Function: Import files from a directory
import_directory() {
    local dir="$1"
    local dir_name="$2"
    local import_failed=0
    
    if [[ ! -d "$dir" ]]; then
        return 0
    fi
    
    local file_count
    file_count=$(find "$dir" -maxdepth 1 -name "*.json" -type f | wc -l)
    
    if [[ $file_count -eq 0 ]]; then
        return 0
    fi
    
    echo "📦 Importing $file_count $dir_name..."
    
    # Check if import-dir command is available
    if vrooli routine import-dir --help &>/dev/null 2>&1; then
        if timeout 300 vrooli routine import-dir "$dir" --fail-fast 2>&1; then
            echo "  ✅ $dir_name imported successfully"
        else
            echo "  ❌ Failed to import $dir_name"
            import_failed=1
        fi
    else
        # Fallback to individual file imports
        echo "  ⚠️  Using individual file import (import-dir not available)"
        
        while IFS= read -r -d '' json_file; do
            filename=$(basename "$json_file")
            echo "    Importing: $filename"
            
            if timeout 60 vrooli routine import "$json_file" 2>/dev/null; then
                echo "      ✅ $filename"
            else
                echo "      ❌ $filename"
                import_failed=1
            fi
        done < <(find "$dir" -maxdepth 1 -name "*.json" -type f -print0 2>/dev/null)
    fi
    
    return $import_failed
}

# Pre-import validation
echo ""
echo "🔍 Validating routines before import..."

VALIDATION_FAILED=0

# Check if validate command is available
if vrooli routine validate --help &>/dev/null 2>&1; then
    # Validate subroutines first
    if ! validate_directory "$SUBROUTINES_DIR" "subroutines"; then
        VALIDATION_FAILED=1
    fi
    
    # Then validate main routines
    if ! validate_directory "$MAIN_ROUTINES_DIR" "main routines"; then
        VALIDATION_FAILED=1
    fi
    
    # Finally validate any legacy files
    if ! validate_directory "$IMPORT_DIR" "legacy files"; then
        VALIDATION_FAILED=1
    fi
    
    if [[ $VALIDATION_FAILED -eq 1 ]]; then
        echo ""
        echo "❌ Some routines failed validation. Fix errors before importing."
        echo "💡 Use 'vrooli routine validate <file>' for detailed error messages"
        exit 1
    else
        echo "✅ All files passed validation"
    fi
else
    echo "⚠️  CLI validate command not available - skipping pre-import validation"
    echo "💡 Files will be validated during import process"
fi

# Hierarchical import: subroutines first, then main routines
echo ""
echo "🚀 Starting hierarchical import (subroutines → main routines → legacy)..."

IMPORT_FAILED=0

# Phase 1: Import subroutines first
if [[ $SUBROUTINES_COUNT -gt 0 ]]; then
    echo ""
    echo "📋 Phase 1: Importing subroutines (atomic operations)..."
    if ! import_directory "$SUBROUTINES_DIR" "subroutines"; then
        IMPORT_FAILED=1
    fi
else
    echo ""
    echo "📋 Phase 1: No subroutines to import"
fi

# Phase 2: Import main routines (depend on subroutines)
if [[ $MAIN_ROUTINES_COUNT -gt 0 ]]; then
    echo ""
    echo "📋 Phase 2: Importing main routines (orchestrated workflows)..."
    if ! import_directory "$MAIN_ROUTINES_DIR" "main routines"; then
        IMPORT_FAILED=1
    fi
else
    echo ""
    echo "📋 Phase 2: No main routines to import"
fi

# Phase 3: Import any legacy files in root directory
if [[ $LEGACY_COUNT -gt 0 ]]; then
    echo ""
    echo "📋 Phase 3: Importing legacy files..."
    if ! import_directory "$IMPORT_DIR" "legacy files"; then
        IMPORT_FAILED=1
    fi
else
    echo ""
    echo "📋 Phase 3: No legacy files to import"
fi

# Final results
echo ""
if [[ $IMPORT_FAILED -eq 1 ]]; then
    echo "⚠️  Import completed with some failures"
    echo "💡 Check the error messages above for details"
    echo "💡 You can retry failed imports individually"
else
    echo "✅ Hierarchical import completed successfully!"
fi

# Show imported routines if list command is available
echo ""
if vrooli routine list --help &>/dev/null 2>&1; then
    echo "📋 Recently imported routines:"
    if ! timeout 30 vrooli routine list --limit 20 --mine 2>/dev/null; then
        echo "⚠️  Could not list imported routines (command failed or timed out)"
    fi
else
    echo "💡 Use 'vrooli routine list' to see imported routines"
fi

# Exit with appropriate code
if [[ $IMPORT_FAILED -eq 1 ]]; then
    exit 1
fi