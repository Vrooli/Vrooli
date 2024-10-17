#!/bin/bash
# Runs all *.bats files in the current directory
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Run all tests
for test_file in "${HERE}"/*.bats; do
    bats "${test_file}"
done
