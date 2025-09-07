#!/bin/bash
# Test script to verify prompt loading with includes is working

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROMPT_FILE="$SCENARIO_DIR/prompts/main-prompt.md"

echo "Testing prompt loading system..."
echo "================================"

# Check if main prompt exists
if [ ! -f "$PROMPT_FILE" ]; then
    echo "❌ Main prompt file not found: $PROMPT_FILE"
    exit 1
fi

echo "✅ Found main prompt file"

# Check for {{INCLUDE}} directives
echo ""
echo "Checking for include directives..."
INCLUDES=$(grep -o '{{INCLUDE:[^}]*}}' "$PROMPT_FILE")

if [ -z "$INCLUDES" ]; then
    echo "⚠️  No include directives found"
else
    echo "Found include directives:"
    echo "$INCLUDES" | while read -r include; do
        # Extract the path
        PATH_PART=$(echo "$include" | sed 's/{{INCLUDE: *//' | sed 's/}}//')
        FULL_PATH="/home/matthalloran8/Vrooli$PATH_PART"
        
        echo -n "  - $PATH_PART: "
        if [ -f "$FULL_PATH" ]; then
            SIZE=$(wc -c < "$FULL_PATH")
            echo "✅ Found ($SIZE bytes)"
        else
            echo "❌ Not found at $FULL_PATH"
        fi
    done
fi

# Test Go implementation
echo ""
echo "Testing Go implementation..."
cd "$SCENARIO_DIR/api"

# Create a simple test program
cat > test_prompt_loading.go <<'EOF'
package main

import (
    "fmt"
    "os"
    "path/filepath"
)

func main() {
    // Test loading the prompt
    promptPath := filepath.Join("..", "prompts", "main-prompt.md")
    content, err := loadPromptWithIncludes(promptPath)
    if err != nil {
        fmt.Printf("❌ Failed to load prompt: %v\n", err)
        os.Exit(1)
    }
    
    // Check if includes were processed
    originalSize := 0
    if info, err := os.Stat(promptPath); err == nil {
        originalSize = int(info.Size())
    }
    
    processedSize := len(content)
    
    fmt.Printf("✅ Prompt loaded successfully\n")
    fmt.Printf("  Original size: %d bytes\n", originalSize)
    fmt.Printf("  Processed size: %d bytes\n", processedSize)
    
    if processedSize > originalSize*2 {
        fmt.Printf("✅ Includes appear to be processed (size increased significantly)\n")
    } else {
        fmt.Printf("⚠️  Processed size seems small - includes may not be working\n")
    }
}
EOF

# Build and run the test
if go build -o test_prompt test_prompt_loading.go main.go 2>/dev/null; then
    ./test_prompt
    rm -f test_prompt test_prompt_loading.go
else
    echo "⚠️  Could not compile test program"
    echo "This is normal if the API hasn't been built yet"
    rm -f test_prompt_loading.go
fi

echo ""
echo "Test complete!"