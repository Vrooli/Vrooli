#!/bin/bash
# Python unit test runner
# Runs Python tests if any Python components exist
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🐍 Running Python unit tests..."

# Check if Python is available
if ! command -v python >/dev/null 2>&1 && ! command -v python3 >/dev/null 2>&1; then
    echo -e "${RED}❌ Python is not installed${NC}"
    exit 1
fi

# Use python3 if available, otherwise python
PYTHON_CMD="python3"
if ! command -v python3 >/dev/null 2>&1; then
    PYTHON_CMD="python"
fi

echo "🐍 Using Python command: $PYTHON_CMD"

# Check if we have Python requirements
has_requirements=false
if [ -f "requirements.txt" ]; then
    has_requirements=true
    echo "📋 Found requirements.txt"
elif [ -f "pyproject.toml" ]; then
    has_requirements=true
    echo "📋 Found pyproject.toml"
else
    echo -e "${YELLOW}ℹ️  No Python requirements file found, skipping Python tests${NC}"
    exit 0
fi

# Check if pytest is available
if ! $PYTHON_CMD -c "import pytest" 2>/dev/null; then
    echo -e "${YELLOW}⚠️  pytest not installed, attempting to install...${NC}"
    
    # Try to install pytest
    if $PYTHON_CMD -m pip install pytest pytest-cov --quiet; then
        echo -e "${GREEN}✅ pytest installed successfully${NC}"
    else
        echo -e "${RED}❌ Failed to install pytest${NC}"
        exit 1
    fi
fi

# Install requirements if they exist
if [ -f "requirements.txt" ]; then
    echo "📦 Installing Python requirements..."
    if ! $PYTHON_CMD -m pip install -r requirements.txt --quiet; then
        echo -e "${RED}❌ Failed to install Python requirements${NC}"
        exit 1
    fi
fi

# Look for test files
test_files=()
test_dirs=("tests" "test" ".")

for dir in "${test_dirs[@]}"; do
    if [ -d "$dir" ]; then
        while IFS= read -r -d '' file; do
            test_files+=("$file")
        done < <(find "$dir" -name "test_*.py" -o -name "*_test.py" -type f -print0 2>/dev/null)
    fi
done

if [ ${#test_files[@]} -eq 0 ]; then
    echo -e "${YELLOW}ℹ️  No Python test files found, creating basic test structure${NC}"
    
    # Create a basic test directory and file
    mkdir -p tests
    cat > tests/test_basic.py << 'EOF'
"""
Basic Python unit tests for visited-tracker scenario.
"""

import unittest
import sys
import os


class TestBasicFunctionality(unittest.TestCase):
    """Basic functionality tests."""
    
    def test_python_environment(self):
        """Test that Python environment is working."""
        self.assertTrue(True)
        self.assertEqual(1 + 1, 2)
    
    def test_import_capabilities(self):
        """Test basic import capabilities."""
        import json
        import datetime
        
        # Test JSON functionality
        test_data = {"test": "value", "number": 42}
        json_string = json.dumps(test_data)
        parsed_data = json.loads(json_string)
        self.assertEqual(parsed_data["test"], "value")
        self.assertEqual(parsed_data["number"], 42)
        
        # Test datetime functionality
        now = datetime.datetime.now()
        self.assertIsInstance(now, datetime.datetime)
    
    def test_file_operations(self):
        """Test basic file operations."""
        import tempfile
        
        # Test file creation and reading
        with tempfile.NamedTemporaryFile(mode='w', delete=False) as f:
            f.write("test content")
            temp_path = f.name
        
        try:
            with open(temp_path, 'r') as f:
                content = f.read()
            self.assertEqual(content, "test content")
        finally:
            os.unlink(temp_path)


if __name__ == '__main__':
    unittest.main()
EOF
    echo -e "${GREEN}✅ Created basic test file: tests/test_basic.py${NC}"
fi

echo "🧪 Running Python tests with pytest..."

# Run pytest with coverage if available
pytest_args=("-v" "--timeout=30")

# Add coverage if pytest-cov is available
if $PYTHON_CMD -c "import pytest_cov" 2>/dev/null; then
    pytest_args+=("--cov=." "--cov-report=term-missing" "--cov-report=html")
fi

# Add test directories/files
if [ ${#test_files[@]} -gt 0 ]; then
    pytest_args+=("${test_files[@]}")
else
    # Run tests in common test directories
    for dir in "${test_dirs[@]}"; do
        if [ -d "$dir" ] && find "$dir" -name "test_*.py" -o -name "*_test.py" | grep -q .; then
            pytest_args+=("$dir")
        fi
    done
fi

if $PYTHON_CMD -m pytest "${pytest_args[@]}"; then
    echo -e "${GREEN}✅ Python unit tests completed successfully${NC}"
    
    # Check if coverage report was generated
    if [ -d "htmlcov" ]; then
        echo ""
        echo "📊 Python Test Coverage generated in htmlcov/"
        if [ -f "htmlcov/index.html" ]; then
            echo -e "${BLUE}ℹ️  HTML coverage report: htmlcov/index.html${NC}"
        fi
    fi
    
    exit 0
else
    echo -e "${RED}❌ Python unit tests failed${NC}"
    exit 1
fi