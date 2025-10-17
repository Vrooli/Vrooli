#!/bin/bash

# Calculation functions for SageMath

sagemath_calculate() {
    local expression="${1:-}"
    
    if [ -z "$expression" ]; then
        echo "Error: No expression provided" >&2
        echo "Usage: sagemath calculate \"expression\"" >&2
        return 1
    fi
    
    if ! sagemath_container_running; then
        echo "Error: SageMath is not running" >&2
        echo "Starting SageMath..."
        sagemath_start
    fi
    
    # Create temporary script
    local temp_script="$SAGEMATH_SCRIPTS_DIR/temp_calc_$$.sage"
    cat > "$temp_script" << EOF
try:
    result = $expression
    print("Result:", result)
except Exception as e:
    print("Error:", str(e))
EOF
    
    # Execute calculation
    echo "Calculating: $expression"
    local output=$(docker exec "$SAGEMATH_CONTAINER_NAME" sage "/home/sage/scripts/$(basename "$temp_script")" 2>&1)
    
    # Clean up temp files (both .sage and .sage.py)
    rm -f "$temp_script"
    rm -f "${temp_script}.py"
    
    # Display result
    echo "$output"
    
    return 0
}

# Open Jupyter notebook interface
sagemath_notebook() {
    if ! sagemath_container_running; then
        echo "SageMath is not running. Starting..."
        sagemath_start
    fi
    
    # Get Jupyter token
    local token=$(docker logs "$SAGEMATH_CONTAINER_NAME" 2>&1 | grep -o 'token=[a-z0-9]*' | tail -1 | cut -d= -f2)
    local url="http://localhost:${SAGEMATH_PORT_JUPYTER}"
    
    if [ -n "$token" ]; then
        url="${url}/?token=${token}"
    fi
    
    echo "Opening SageMath Jupyter notebook..."
    echo "URL: $url"
    
    # Try to open in browser if available
    if command -v xdg-open >/dev/null 2>&1; then
        xdg-open "$url" 2>/dev/null &
    elif command -v open >/dev/null 2>&1; then
        open "$url" 2>/dev/null &
    else
        echo "Please open this URL in your browser: $url"
    fi
    
    return 0
}

# Run tests
sagemath_test() {
    echo "Running SageMath tests..."
    
    if ! sagemath_container_running; then
        echo "Starting SageMath for tests..."
        sagemath_start
    fi
    
    # Run test script
    local test_output="$SAGEMATH_OUTPUTS_DIR/test_$(date +%Y%m%d_%H%M%S).out"
    
    if docker exec "$SAGEMATH_CONTAINER_NAME" sage "/home/sage/scripts/test.sage" > "$test_output" 2>&1; then
        echo "Tests passed!"
        cat "$test_output"
        touch "$SAGEMATH_OUTPUTS_DIR/.last_test"
        return 0
    else
        echo "Tests failed!" >&2
        cat "$test_output" >&2
        return 1
    fi
}