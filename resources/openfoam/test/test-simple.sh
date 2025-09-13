#!/bin/bash

# Simple test script for OpenFOAM

echo "Testing OpenFOAM resource..."

# Test 1: CLI help
echo -n "1. CLI help command... "
if ./cli.sh help &>/dev/null; then
    echo "PASS"
else
    echo "FAIL"
fi

# Test 2: CLI info
echo -n "2. CLI info command... "
if ./cli.sh info &>/dev/null; then
    echo "PASS"
else
    echo "FAIL"
fi

# Test 3: CLI status
echo -n "3. CLI status command... "
if ./cli.sh status &>/dev/null; then
    echo "PASS"
else
    echo "FAIL"
fi

echo "Test complete!"