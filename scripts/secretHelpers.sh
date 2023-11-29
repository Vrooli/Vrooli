#!/bin/bash
# These functions help with setting secret variables

# Function to store a secret in /run/secrets, converting multi-line to single-line if necessary
convert_secret() {
    local secret_value="$1"

    if is_multiline "$secret_value"; then
        # Convert multi-line secrets to single-line
        echo "$secret_value" | convert_to_single_line
    else
        echo "$secret_value"
    fi
}

# Function to check if a string is multi-line
is_multiline() {
    local string="$1"
    test "$(echo "$string" | wc -l)" -gt 1
}

# Function to convert multi-line string to single-line.
# Also removes BEGIN/END lines from PEM files.
convert_to_single_line() {
    sed -ne '/-BEGIN /,/-END /p' | sed -e '/-BEGIN /d' -e '/-END /d' | awk 'NR > 1 { printf "\\n" } { printf "%s", $0 }'
}
