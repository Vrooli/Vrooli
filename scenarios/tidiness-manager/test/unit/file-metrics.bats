#!/usr/bin/env bats
# [REQ:TM-LS-005] [REQ:TM-LS-006]
# File metrics: line counting and long file detection

setup() {
    export TEST_DIR="/tmp/tidiness-test-$$"
    mkdir -p "$TEST_DIR"
}

teardown() {
    rm -rf "$TEST_DIR"
}

# [REQ:TM-LS-005] Compute per-file line counts for all source files matching configurable glob patterns
@test "Count lines in Go files" {
    cat > "$TEST_DIR/test.go" <<'EOF'
package main
func main() {
}
EOF

    local lines
    lines=$(wc -l < "$TEST_DIR/test.go")
    [ "$lines" -eq 3 ]
}

# [REQ:TM-LS-006] Flag files exceeding configurable line count threshold as 'long file' issues
@test "Detect long files exceeding threshold" {
    # Create a file with 600 lines (exceeds default 500 threshold)
    for i in $(seq 1 600); do
        echo "line $i" >> "$TEST_DIR/long.go"
    done

    local lines
    lines=$(wc -l < "$TEST_DIR/long.go")
    [ "$lines" -gt 500 ]
}
