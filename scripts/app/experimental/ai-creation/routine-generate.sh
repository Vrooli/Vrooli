#!/usr/bin/env bash

# Process routine backlog items into staged routine definitions
# This script orchestrates AI-powered routine generation with dynamic subroutine discovery
#
# Usage:
#   routine-generate.sh [--direct]   # Use --direct to generate prompt for manual use

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

# Source var.sh first to get all directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

PROJECT_ROOT="${var_ROOT_DIR}"
ROUTINE_DIR="$PROJECT_ROOT/docs/ai-creation/routine"
BACKLOG_FILE="$ROUTINE_DIR/backlog.md"
STAGED_DIR="$ROUTINE_DIR/staged"
PROMPT_FILE="$ROUTINE_DIR/prompt.md"
SUBROUTINES_FILE="$ROUTINE_DIR/available-subroutines.txt"

# Check for --direct flag
DIRECT_MODE=false
if [[ "${1:-}" == "--direct" ]]; then
    DIRECT_MODE=true
    # Use the direct generation script
    exec "$SCRIPT_DIR/routine-generate-direct.sh" --prompt-only --subroutines
fi

# Configuration
AI_COMMAND=${AI_COMMAND:-"claude-code"}  # Can be overridden for other LLMs
AI_FALLBACK=${AI_FALLBACK:-"claude"}
AI_PROMPT_FLAG=${AI_PROMPT_FLAG:-"--prompt"}  # claude-code uses --prompt
AI_FALLBACK_FLAG=${AI_FALLBACK_FLAG:-"-p"}     # claude uses -p

# Check if required files exist
if [[ ! -f "$BACKLOG_FILE" ]]; then
    echo "Error: Backlog file not found at $BACKLOG_FILE"
    exit 1
fi

if [[ ! -f "$PROMPT_FILE" ]]; then
    echo "Error: Prompt file not found at $PROMPT_FILE"
    exit 1
fi

# Create staged directory with error handling
echo "📁 Setting up staging directory..."
if ! mkdir -p "$STAGED_DIR" 2>/dev/null; then
    echo "❌ Failed to create staging directory: $STAGED_DIR"
    echo "💡 Check permissions and disk space"
    exit 1
fi

# Verify directory is writable
if ! [[ -w "$STAGED_DIR" ]]; then
    echo "❌ Staging directory is not writable: $STAGED_DIR"
    echo "💡 Check directory permissions"
    exit 1
fi
echo "✅ Staging directory ready"

# Find the AI generation tool (modular approach for future LLM support)
AI_TOOL=""
AI_FLAG=""

if command -v "$AI_COMMAND" >/dev/null 2>&1; then
    AI_TOOL="$AI_COMMAND"
    AI_FLAG="$AI_PROMPT_FLAG"
elif command -v "$AI_FALLBACK" >/dev/null 2>&1; then
    AI_TOOL="$AI_FALLBACK"
    AI_FLAG="$AI_FALLBACK_FLAG"
else
    echo "⛔️  No AI generation tool found. Please install one of:"
    echo "  - Claude Code CLI: https://docs.anthropic.com/claude/docs/claude-cli"
    echo "  - Claude CLI: npm install -g @anthropic-ai/claude-cli"
    echo ""
    echo "Or set AI_COMMAND environment variable to use a different LLM tool"
    exit 1
fi

# maintenance-agent.sh is in the same directory as this script
MAINTENANCE_AGENT="$SCRIPT_DIR/maintenance-agent.sh"

if [[ ! -x "$MAINTENANCE_AGENT" ]]; then
    echo "⛔️  Agent script not found or not executable: $MAINTENANCE_AGENT" >&2
    exit 1
fi

echo "🤖 AI Routine Generation System"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📁 Backlog: $BACKLOG_FILE"
echo "📁 Staged: $STAGED_DIR"
echo "🔧 AI Tool: $AI_TOOL"
echo "🔧 Agent: $MAINTENANCE_AGENT"

# Check if Vrooli CLI is available for validation and subroutine discovery
VROOLI_AVAILABLE=false
ENABLE_VALIDATION=false
ENABLE_DISCOVERY=false

if ! command -v vrooli >/dev/null 2>&1; then
    echo "⚠️  Warning: Vrooli CLI not found."
    echo "   - Generated routines won't be pre-validated"
    echo "   - Subroutine discovery will be unavailable"
    echo "   To enable: cd packages/cli && pnpm run build && npm link"
else
    VROOLI_AVAILABLE=true
    # Check if CLI is authenticated
    if ! vrooli auth status &> /dev/null; then
        echo "⚠️  Warning: Vrooli CLI not authenticated."
        echo "   - Subroutine discovery will be unavailable"
        echo "   - Generated routines won't be pre-validated"
        echo "   To enable: vrooli auth login"
    else
        echo "✅ Vrooli CLI authenticated"
        ENABLE_VALIDATION=true
        ENABLE_DISCOVERY=true
    fi
fi

# Discover available subroutines if CLI is available and authenticated
if [[ "$ENABLE_DISCOVERY" == "true" ]]; then
    echo ""
    echo "🔍 Discovering available subroutines..."
    
    # Check if discover command is available
    if vrooli routine discover --help &>/dev/null 2>&1; then
        # Create a comprehensive subroutine mapping with timeout
        if timeout 60 vrooli routine discover --format mapping > "$SUBROUTINES_FILE" 2>/dev/null; then
            if [[ -f "$SUBROUTINES_FILE" ]] && [[ -s "$SUBROUTINES_FILE" ]]; then
                SUBROUTINE_COUNT=$(grep -c "^\"" "$SUBROUTINES_FILE" 2>/dev/null || echo "0")
                echo "✅ Found $SUBROUTINE_COUNT available subroutines"
                
                # Also create a JSON version for structured access
                timeout 60 vrooli routine discover --format json > "$ROUTINE_DIR/available-subroutines.json" 2>/dev/null || true
            else
                echo "⚠️  Discovery returned empty result - generation will proceed without dynamic discovery"
                ENABLE_DISCOVERY=false
            fi
        else
            echo "⚠️  Failed to discover subroutines or command timed out - generation will proceed without dynamic discovery"
            ENABLE_DISCOVERY=false
        fi
    else
        echo "⚠️  CLI discover command not available - generation will proceed without dynamic discovery"
        ENABLE_DISCOVERY=false
    fi
fi

echo ""

# Check if there are any unprocessed items (items not in the "Completed Items" section)
if ! grep -q "### [0-9]" "$BACKLOG_FILE"; then
    echo "No unprocessed items found in backlog."
    exit 0
fi

echo "📋 Found unprocessed items in backlog - AI will select and process one"

# Create a temporary file with the processing request
TEMP_REQUEST=$(mktemp)

# Build the AI request with optional subroutine context
cat > "$TEMP_REQUEST" << 'EOF'
Process backlog routine items into staged routine definitions using the AI creation system.

Follow these steps:
1. Read docs/ai-creation/routine/prompt.md for complete routine generation instructions
2. Read docs/ai-creation/routine/backlog.md to see available routine items to process
3. Select the first unprocessed item from the backlog
EOF

# Add subroutine discovery instructions if available
if [[ "$ENABLE_DISCOVERY" == "true" ]] && [[ -f "$SUBROUTINES_FILE" ]]; then
    cat >> "$TEMP_REQUEST" << 'EOF'
4. IMPORTANT: Read docs/ai-creation/routine/available-subroutines.txt for ACTUAL subroutine IDs
   - Use ONLY IDs from this file (format: "publicId" # Name (Type))
   - Search for relevant subroutines based on functionality needed
   - NEVER use placeholder or made-up IDs
EOF
else
    cat >> "$TEMP_REQUEST" << 'EOF'
4. Since subroutine discovery is unavailable, you'll need to:
   - Use generic subroutine references that will need manual updating
   - Add TODO comments in the JSON where subroutine IDs are needed
   - Include clear descriptions of what type of subroutine is required
EOF
fi

cat >> "$TEMP_REQUEST" << 'EOF'
5. Generate a complete routine JSON following the prompt specifications
6. Save the routine to docs/ai-creation/routine/staged/ with a descriptive filename
7. Update the backlog to mark the item as processed

Focus on creating a valid, importable routine with:
- Proper data flow validation with unique variable names
- Correct ID formats (19-digit snowflake IDs)
- Complete form definitions
- Actual subroutine references from available-subroutines.txt (if available)
- Clear TODO comments for any subroutines that need manual selection

Generate one routine at a time and ensure it follows all validation rules from the prompt.
EOF

# Call the maintenance agent with the processing request
echo "🤖 Starting AI routine generation..."
if "$MAINTENANCE_AGENT" "$(cat "$TEMP_REQUEST")" "50"; then
    echo "✅ Successfully processed backlog item"
    echo "📁 Check the staged/ directory for new routine files"
    
    # Post-generation validation if CLI is available and authenticated
    if [[ "${ENABLE_VALIDATION:-false}" == "true" ]]; then
        echo ""
        echo "🔍 Validating generated routines..."
        
        # Check if validate command is available
        if vrooli routine validate --help &>/dev/null 2>&1; then
            # Find the most recently created JSON files in staged directory
            NEWEST_FILES=$(find "$STAGED_DIR" -name "*.json" -type f -newerct "1 minute ago" 2>/dev/null || true)
            
            if [[ -n "$NEWEST_FILES" ]]; then
                VALIDATION_PASSED=true
                files_validated=0
                
                while IFS= read -r json_file; do
                    [[ -f "$json_file" ]] || continue
                    filename=$(basename "$json_file")
                    echo "  Validating: $filename"
                    
                    # First check if it's valid JSON
                    if ! jq empty "$json_file" 2>/dev/null; then
                        echo "    ❌ Invalid JSON: $filename"
                        VALIDATION_PASSED=false
                        ((files_validated++))
                        continue
                    fi
                    
                    # Then check with CLI validation
                    if timeout 30 vrooli routine validate "$json_file" --quiet 2>/dev/null; then
                        echo "    ✅ Valid: $filename"
                    else
                        echo "    ❌ Validation failed: $filename"
                        echo "    💡 Run 'vrooli routine validate $json_file' for details"
                        
                        # Also try standalone validation for more details
                        if [[ -x "$ROUTINE_DIR/validate-routine.sh" ]]; then
                            echo "    🔍 Running standalone validation..."
                            "$ROUTINE_DIR/validate-routine.sh" "$json_file" 2>/dev/null || true
                        fi
                        VALIDATION_PASSED=false
                    fi
                    ((files_validated++))
                done <<< "$NEWEST_FILES"
                
                if [[ $files_validated -eq 0 ]]; then
                    echo "⚠️  No files found to validate"
                elif [[ "$VALIDATION_PASSED" == "true" ]]; then
                    echo "✅ All $files_validated generated routines passed validation"
                else
                    echo "⚠️  Some generated routines failed validation"
                    echo "   You may need to regenerate or manually fix them before import"
                fi
            else
                echo "⚠️  No recently generated files found for validation"
            fi
        else
            echo "⚠️  CLI validate command not available - skipping validation"
        fi
    fi
else
    echo "❌ Failed to process backlog item" >&2
    exit 1
fi

# Clean up
rm -f "$TEMP_REQUEST"

echo "🎉 Routine processing complete!"