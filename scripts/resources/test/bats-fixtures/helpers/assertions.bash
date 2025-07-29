#!/usr/bin/env bash
# Common Test Assertions for Bats Tests
# Provides additional assertion helpers beyond bats-assert

# Assert output contains text (case-sensitive)
assert_output_contains() {
    local expected="$1"
    if [[ ! "$output" =~ $expected ]]; then
        echo "Expected output to contain: $expected" >&2
        echo "Actual output: $output" >&2
        return 1
    fi
}

# Assert output does not contain text
assert_output_not_contains() {
    local unexpected="$1"
    if [[ "$output" =~ $unexpected ]]; then
        echo "Expected output NOT to contain: $unexpected" >&2
        echo "Actual output: $output" >&2
        return 1
    fi
}

# Assert output matches regex
assert_output_matches() {
    local pattern="$1"
    if [[ ! "$output" =~ $pattern ]]; then
        echo "Expected output to match pattern: $pattern" >&2
        echo "Actual output: $output" >&2
        return 1
    fi
}

# Assert file exists
assert_file_exists() {
    local file="$1"
    if [[ ! -f "$file" ]]; then
        echo "Expected file to exist: $file" >&2
        return 1
    fi
}

# Assert file does not exist
assert_file_not_exists() {
    local file="$1"
    if [[ -f "$file" ]]; then
        echo "Expected file NOT to exist: $file" >&2
        return 1
    fi
}

# Assert directory exists
assert_dir_exists() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        echo "Expected directory to exist: $dir" >&2
        return 1
    fi
}

# Assert directory does not exist
assert_dir_not_exists() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        echo "Expected directory NOT to exist: $dir" >&2
        return 1
    fi
}

# Assert file contains text
assert_file_contains() {
    local file="$1"
    local expected="$2"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist: $file" >&2
        return 1
    fi
    
    if ! grep -q "$expected" "$file"; then
        echo "Expected file $file to contain: $expected" >&2
        echo "File contents:" >&2
        cat "$file" >&2
        return 1
    fi
}

# Assert environment variable is set
assert_env_set() {
    local var="$1"
    if [[ -z "${!var:-}" ]]; then
        echo "Expected environment variable to be set: $var" >&2
        return 1
    fi
}

# Assert environment variable equals value
assert_env_equals() {
    local var="$1"
    local expected="$2"
    local actual="${!var:-}"
    
    if [[ "$actual" != "$expected" ]]; then
        echo "Environment variable $var:" >&2
        echo "  Expected: $expected" >&2
        echo "  Actual: $actual" >&2
        return 1
    fi
}

# Assert command exists
assert_command_exists() {
    local cmd="$1"
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "Expected command to exist: $cmd" >&2
        return 1
    fi
}

# Assert function exists
assert_function_exists() {
    local func="$1"
    if ! declare -f "$func" >/dev/null 2>&1; then
        echo "Expected function to exist: $func" >&2
        return 1
    fi
}

# Assert arrays are equal
assert_arrays_equal() {
    local -n array1=$1
    local -n array2=$2
    
    if [[ ${#array1[@]} -ne ${#array2[@]} ]]; then
        echo "Arrays have different lengths:" >&2
        echo "  $1: ${#array1[@]} elements" >&2
        echo "  $2: ${#array2[@]} elements" >&2
        return 1
    fi
    
    for i in "${!array1[@]}"; do
        if [[ "${array1[$i]}" != "${array2[$i]}" ]]; then
            echo "Arrays differ at index $i:" >&2
            echo "  $1[$i]: ${array1[$i]}" >&2
            echo "  $2[$i]: ${array2[$i]}" >&2
            return 1
        fi
    done
}

# Assert JSON field equals value
assert_json_field_equals() {
    local json="$1"
    local field="$2"
    local expected="$3"
    
    local actual=$(echo "$json" | jq -r "$field" 2>/dev/null)
    if [[ $? -ne 0 ]]; then
        echo "Failed to parse JSON or extract field: $field" >&2
        echo "JSON: $json" >&2
        return 1
    fi
    
    if [[ "$actual" != "$expected" ]]; then
        echo "JSON field $field:" >&2
        echo "  Expected: $expected" >&2
        echo "  Actual: $actual" >&2
        return 1
    fi
}

# Assert port is available
assert_port_available() {
    local port="$1"
    if lsof -i ":$port" >/dev/null 2>&1; then
        echo "Expected port to be available: $port" >&2
        return 1
    fi
}

# Assert port is in use
assert_port_in_use() {
    local port="$1"
    if ! lsof -i ":$port" >/dev/null 2>&1; then
        echo "Expected port to be in use: $port" >&2
        return 1
    fi
}