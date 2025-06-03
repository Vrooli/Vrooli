#!/bin/sh

# This script modifies JavaScript files in the specified directory to add 'assert { type: "json" }'
# to import statements that reference JSON files.

# Check if two arguments are provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 <directory>"
    exit 1
fi

DIR=$1

# Verify that the provided argument is a directory
if [ ! -d "$DIR" ]; then
    echo "Error: $DIR is not a directory"
    exit 1
fi

# Find all .js files and update JSON import statements
find "$DIR" -type f -name '*.js' -exec sed -i.bak 's/\(import .* from ".*\.json"\);/\1 with { type: "json" };/g' {} \;
find "$DIR" -type f -name '*.js' -exec sed -i.bak 's/\(import .* from ".*\.json"\) assert { type: "json" };/\1 with { type: "json" };/g' {} \; # Convert existing asserts to with

# Clean up backup files created by sed
find "$DIR" -type f -name '*.bak' -delete

echo "Finished fixing imports using 'with { type: \"json\" }'."