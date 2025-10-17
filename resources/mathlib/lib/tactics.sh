#!/bin/bash
# Custom tactics management for Mathlib

# Source core functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Custom tactics directory
TACTICS_DIR="${MATHLIB_WORK_DIR}/custom_tactics"

# Initialize tactics directory
init_tactics_dir() {
    mkdir -p "$TACTICS_DIR"
    
    # Create example custom tactic if directory is empty
    if [[ -z "$(ls -A "$TACTICS_DIR" 2>/dev/null)" ]]; then
        cat > "$TACTICS_DIR/example.lean" << 'EOF'
-- Example custom tactic
import Mathlib.Tactic

namespace CustomTactics

-- A simple tactic that tries reflexivity and simplification
macro "easy" : tactic => `(tactic| first | rfl | simp)

-- A tactic for basic arithmetic
macro "arith" : tactic => `(tactic| norm_num <;> ring)

-- A tactic that combines multiple approaches
macro "auto" : tactic => `(tactic| 
  first | rfl | simp | linarith | norm_num | ring | omega)

end CustomTactics
EOF
        echo "Created example custom tactic in $TACTICS_DIR/example.lean"
    fi
}

# Load a custom tactic file
load_tactic() {
    local tactic_file="$1"
    local tactic_name="${2:-$(basename "$tactic_file" .lean)}"
    
    if [[ ! -f "$tactic_file" ]]; then
        echo "Error: Tactic file not found: $tactic_file"
        return 1
    fi
    
    # Copy to tactics directory
    local dest_file="$TACTICS_DIR/${tactic_name}.lean"
    cp "$tactic_file" "$dest_file"
    
    # Validate the tactic file
    local lean_path="${HOME}/.elan/bin/lean"
    if [[ ! -x "$lean_path" ]]; then
        lean_path="lean"
    fi
    
    echo "Validating custom tactic..."
    if timeout 10 "$lean_path" "$dest_file" &>/dev/null; then
        echo "✓ Custom tactic '$tactic_name' loaded successfully"
        echo "  Location: $dest_file"
        return 0
    else
        echo "✗ Failed to validate tactic file"
        rm -f "$dest_file"
        return 1
    fi
}

# List loaded custom tactics
list_custom_tactics() {
    init_tactics_dir
    
    echo "Custom Tactics Directory: $TACTICS_DIR"
    echo ""
    
    if [[ -z "$(ls -A "$TACTICS_DIR" 2>/dev/null)" ]]; then
        echo "No custom tactics loaded."
        echo "Use 'vrooli resource mathlib content tactics load <file>' to add custom tactics."
        return 0
    fi
    
    echo "Loaded Custom Tactics:"
    echo "======================"
    
    for tactic_file in "$TACTICS_DIR"/*.lean; do
        if [[ -f "$tactic_file" ]]; then
            local name=$(basename "$tactic_file" .lean)
            echo "  • $name"
            
            # Extract macro definitions
            grep -E "^macro " "$tactic_file" | while read -r line; do
                if [[ "$line" =~ macro[[:space:]]+\"([^\"]+)\" ]]; then
                    echo "      - ${BASH_REMATCH[1]}"
                fi
            done
        fi
    done
    
    echo ""
    echo "To use custom tactics in proofs, import them with:"
    echo "  import CustomTactics.<name>"
}

# Remove a custom tactic
remove_tactic() {
    local tactic_name="$1"
    
    if [[ -z "$tactic_name" ]]; then
        echo "Error: Please provide a tactic name to remove"
        return 1
    fi
    
    local tactic_file="$TACTICS_DIR/${tactic_name}.lean"
    
    if [[ ! -f "$tactic_file" ]]; then
        echo "Error: Custom tactic not found: $tactic_name"
        return 1
    fi
    
    rm -f "$tactic_file"
    echo "Custom tactic '$tactic_name' removed"
}

# Test a custom tactic with a proof
test_tactic() {
    local tactic_name="$1"
    local proof_code="$2"
    
    if [[ -z "$tactic_name" ]] || [[ -z "$proof_code" ]]; then
        echo "Error: Please provide both tactic name and proof code"
        echo "Usage: test_tactic <tactic_name> <proof_code>"
        return 1
    fi
    
    local tactic_file="$TACTICS_DIR/${tactic_name}.lean"
    
    if [[ ! -f "$tactic_file" ]]; then
        echo "Error: Custom tactic not found: $tactic_name"
        return 1
    fi
    
    # Create test file
    local test_file="/tmp/test_tactic_$$.lean"
    
    # Copy tactic content
    cp "$tactic_file" "$test_file"
    echo "" >> "$test_file"
    echo "-- Test proof" >> "$test_file"
    echo "$proof_code" >> "$test_file"
    
    # Run test
    local lean_path="${HOME}/.elan/bin/lean"
    if [[ ! -x "$lean_path" ]]; then
        lean_path="lean"
    fi
    
    echo "Testing custom tactic '$tactic_name'..."
    if timeout 10 "$lean_path" "$test_file" 2>&1; then
        echo "✓ Test passed"
        rm -f "$test_file"
        return 0
    else
        echo "✗ Test failed"
        rm -f "$test_file"
        return 1
    fi
}

# Export custom tactics to a bundle
export_tactics() {
    local output_file="${1:-custom_tactics.tar.gz}"
    
    init_tactics_dir
    
    if [[ -z "$(ls -A "$TACTICS_DIR" 2>/dev/null)" ]]; then
        echo "No custom tactics to export"
        return 1
    fi
    
    echo "Exporting custom tactics..."
    tar -czf "$output_file" -C "$TACTICS_DIR" .
    echo "Custom tactics exported to: $output_file"
}

# Import custom tactics from a bundle
import_tactics() {
    local bundle_file="$1"
    
    if [[ ! -f "$bundle_file" ]]; then
        echo "Error: Bundle file not found: $bundle_file"
        return 1
    fi
    
    init_tactics_dir
    
    echo "Importing custom tactics from: $bundle_file"
    tar -xzf "$bundle_file" -C "$TACTICS_DIR"
    echo "Custom tactics imported successfully"
    
    list_custom_tactics
}

# Main entry point
main() {
    local command="${1:-list}"
    shift || true
    
    case "$command" in
        init)
            init_tactics_dir
            ;;
        load)
            load_tactic "$@"
            ;;
        list)
            list_custom_tactics
            ;;
        remove)
            remove_tactic "$@"
            ;;
        test)
            test_tactic "$@"
            ;;
        export)
            export_tactics "$@"
            ;;
        import)
            import_tactics "$@"
            ;;
        help)
            echo "Custom Tactics Management"
            echo "Usage: $0 [command] [options]"
            echo ""
            echo "Commands:"
            echo "  init    - Initialize tactics directory"
            echo "  load    - Load a custom tactic file"
            echo "  list    - List loaded custom tactics"
            echo "  remove  - Remove a custom tactic"
            echo "  test    - Test a custom tactic"
            echo "  export  - Export tactics to bundle"
            echo "  import  - Import tactics from bundle"
            echo "  help    - Show this help"
            ;;
        *)
            echo "Unknown command: $command"
            echo "Run '$0 help' for usage information"
            return 1
            ;;
    esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi