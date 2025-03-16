#!/bin/sh

# This script modifies JavaScript files in the specified directory to add 'with { type: "json" }' or 'assert { type: "json" }'
# to import statements that reference JSON files.

# Check if a directory argument is provided
if [ $# -ne 2 ]; then
    echo "Usage: $0 <with|assert> <directory>"
    exit 1
fi


WORD=$1
DIR=$2

# Verify that the provided argument is a directory
if [ ! -d "$DIR" ]; then
    echo "Error: $DIR is not a directory"
    exit 1
fi

# Find all .js files and update JSON import statements
if [ "$WORD" = "with" ]; then
    find "$DIR" -type f -name '*.js' -exec sed -i.bak 's/\(import .* from ".*\.json"\);/\1 with { type: "json" };/g' {} \;
elif [ "$WORD" = "assert" ]; then
    find "$DIR" -type f -name '*.js' -exec sed -i.bak 's/\(import .* from ".*\.json"\);/\1 assert { type: "json" };/g' {} \;
else
    echo "Error: Invalid argument. Use 'with' or 'assert'."
    exit 1
fi
# Clean up backup files created by sed
find "$DIR" -type f -name '*.bak' -delete

echo "Done."