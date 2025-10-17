#!/bin/bash
# Interactive REPL interface for Mathlib theorem proving

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Check if Lean is available
check_lean_available() {
    local lean_path=""
    
    # Try elan installation first
    if [[ -x "${HOME}/.elan/bin/lean" ]]; then
        lean_path="${HOME}/.elan/bin/lean"
    elif command -v lean &>/dev/null; then
        lean_path="lean"
    else
        echo "Error: Lean not found. Please install Lean 4 first."
        return 1
    fi
    
    echo "$lean_path"
}

# Start interactive REPL session
start_repl() {
    local lean_path
    lean_path=$(check_lean_available) || return 1
    
    echo "==================================="
    echo "Mathlib Interactive REPL"
    echo "==================================="
    echo "Type Lean theorems and proofs interactively."
    echo "Commands:"
    echo "  :help    - Show this help"
    echo "  :tactics - List available tactics"
    echo "  :check   - Check type of expression"
    echo "  :eval    - Evaluate expression"
    echo "  :quit    - Exit REPL"
    echo "  :clear   - Clear screen"
    echo ""
    echo "Example: theorem simple : 2 + 2 = 4 := by rfl"
    echo "==================================="
    echo ""
    
    # Create temporary workspace
    local work_dir="/tmp/mathlib_repl_$$"
    mkdir -p "$work_dir"
    
    # Create a persistent Lean file for the session
    local session_file="$work_dir/session.lean"
    echo "-- Mathlib REPL Session" > "$session_file"
    echo "import Mathlib.Data.Nat.Basic" >> "$session_file"
    echo "" >> "$session_file"
    
    local line_number=4
    local history_file="${HOME}/.mathlib_repl_history"
    
    # Main REPL loop
    while true; do
        # Show prompt
        echo -n "lean> "
        
        # Read user input
        read -r input
        
        # Save to history
        echo "$input" >> "$history_file"
        
        # Handle special commands
        case "$input" in
            ":quit"|":exit"|":q")
                echo "Goodbye!"
                rm -rf "$work_dir"
                return 0
                ;;
            ":help"|":h")
                echo "Commands:"
                echo "  :help    - Show this help"
                echo "  :tactics - List available tactics"
                echo "  :check   - Check type of expression (e.g., :check 2 + 2)"
                echo "  :eval    - Evaluate expression (e.g., :eval 2 + 2)"
                echo "  :quit    - Exit REPL"
                echo "  :clear   - Clear screen"
                echo ""
                echo "Enter Lean code directly to verify it."
                continue
                ;;
            ":tactics"|":t")
                echo "Basic tactics:"
                echo "  rfl        - Reflexivity"
                echo "  simp       - Simplification"
                echo "  ring       - Ring algebra"
                echo "  linarith   - Linear arithmetic"
                echo "  norm_num   - Normalize numbers"
                echo ""
                echo "Logic tactics:"
                echo "  intro      - Introduce hypothesis"
                echo "  apply      - Apply theorem"
                echo "  exact      - Exact proof"
                echo "  constructor - Apply constructor"
                echo "  cases      - Case analysis"
                echo ""
                echo "Advanced tactics:"
                echo "  omega      - Omega decision procedure"
                echo "  field_simp - Field simplification"
                echo "  polyrith   - Polynomial arithmetic"
                continue
                ;;
            ":clear"|":c")
                clear
                continue
                ;;
            ":check "*)
                # Extract expression to check
                local expr="${input#:check }"
                
                # Create temporary file with check command
                local check_file="$work_dir/check.lean"
                echo "import Mathlib.Data.Nat.Basic" > "$check_file"
                echo "#check $expr" >> "$check_file"
                
                # Run Lean to check type
                echo "Checking: $expr"
                timeout 5 "$lean_path" "$check_file" 2>&1 | grep -v "^$" | head -5
                continue
                ;;
            ":eval "*)
                # Extract expression to evaluate
                local expr="${input#:eval }"
                
                # Create temporary file with eval command
                local eval_file="$work_dir/eval.lean"
                echo "import Mathlib.Data.Nat.Basic" > "$eval_file"
                echo "#eval $expr" >> "$eval_file"
                
                # Run Lean to evaluate
                echo "Evaluating: $expr"
                timeout 5 "$lean_path" "$eval_file" 2>&1 | grep -v "^$" | head -5
                continue
                ;;
            "")
                # Empty input, just continue
                continue
                ;;
            *)
                # Regular Lean code - add to session and verify
                echo "$input" >> "$session_file"
                
                # Try to verify the current session
                echo "Verifying..."
                
                # Run Lean on the session file
                local output
                output=$(timeout 10 "$lean_path" "$session_file" 2>&1)
                local result=$?
                
                if [[ $result -eq 0 ]]; then
                    # Success
                    echo "✓ Verified successfully"
                    ((line_number++))
                else
                    # Error - show diagnostics and remove the bad line
                    echo "✗ Verification failed:"
                    echo "$output" | grep -E "error:|warning:" | head -5
                    
                    # Remove the last line from session file
                    sed -i '$d' "$session_file"
                fi
                ;;
        esac
    done
}

# Function to replay a proof file interactively
replay_proof() {
    local proof_file="$1"
    
    if [[ ! -f "$proof_file" ]]; then
        echo "Error: File not found: $proof_file"
        return 1
    fi
    
    echo "Replaying proof from: $proof_file"
    echo "Press Enter to step through each line..."
    echo ""
    
    local lean_path
    lean_path=$(check_lean_available) || return 1
    
    # Create temporary workspace
    local work_dir="/tmp/mathlib_replay_$$"
    mkdir -p "$work_dir"
    local replay_file="$work_dir/replay.lean"
    
    # Start with empty file
    echo "-- Replay of $proof_file" > "$replay_file"
    echo "" >> "$replay_file"
    
    # Read file line by line
    while IFS= read -r line; do
        # Skip empty lines and comments
        if [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*-- ]]; then
            continue
        fi
        
        # Show the line
        echo ">>> $line"
        
        # Add to replay file
        echo "$line" >> "$replay_file"
        
        # Verify current state
        local output
        output=$(timeout 5 "$lean_path" "$replay_file" 2>&1)
        local result=$?
        
        if [[ $result -eq 0 ]]; then
            echo "✓ OK"
        else
            echo "✗ Error:"
            echo "$output" | grep -E "error:" | head -3
        fi
        
        # Wait for user to continue
        read -r -p "Press Enter to continue..."
        echo ""
    done < "$proof_file"
    
    echo "Replay complete!"
    rm -rf "$work_dir"
}

# Main entry point
main() {
    local command="${1:-start}"
    shift
    
    case "$command" in
        start)
            start_repl "$@"
            ;;
        replay)
            replay_proof "$@"
            ;;
        help)
            echo "Mathlib Interactive REPL"
            echo "Usage: $0 [command] [options]"
            echo ""
            echo "Commands:"
            echo "  start   - Start interactive REPL session (default)"
            echo "  replay  - Replay a proof file interactively"
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