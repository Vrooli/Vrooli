#!/usr/bin/env bash
# jq JSON processor mocks for Bats tests.
# Provides comprehensive jq command mocking with a clear mock::jq::* namespace.

# Prevent duplicate loading
if [[ "${JQ_MOCKS_LOADED:-}" == "true" ]]; then
  return 0
fi
export JQ_MOCKS_LOADED="true"

# Load centralized logging utilities
if ! command -v mock::log_call &>/dev/null; then
  # Try to load from relative path
  local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  if [[ -f "$script_dir/logs.sh" ]]; then
    source "$script_dir/logs.sh"
  else
    echo "[JQ_MOCK] WARNING: Could not load mock logging utilities" >&2
  fi
fi

# ----------------------------
# Global mock state & options
# ----------------------------
# Modes: normal, error, invalid_json, filter_error
export JQ_MOCK_MODE="${JQ_MOCK_MODE:-normal}"

# Mock state for JSON data and filters
declare -A MOCK_JQ_DATA=()           # filename/stdin -> json_content
declare -A MOCK_JQ_RESPONSES=()      # filter -> response_content
declare -A MOCK_JQ_ERRORS=()         # filter/filename -> error_type
declare -A MOCK_JQ_OUTPUT_FORMAT=()  # call_id -> format_settings
declare -A MOCK_JQ_CALL_HISTORY=()   # call_counter -> full_command

# Call tracking
export JQ_MOCK_CALL_COUNTER=0

# File-based state persistence for subshell access (BATS compatibility)
export JQ_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/jq_mock_state.$$"

# Initialize state file
_jq_mock_init_state_file() {
  if [[ -n "${JQ_MOCK_STATE_FILE}" ]]; then
    {
      echo "declare -A MOCK_JQ_DATA=()"
      echo "declare -A MOCK_JQ_RESPONSES=()"
      echo "declare -A MOCK_JQ_ERRORS=()"
      echo "declare -A MOCK_JQ_OUTPUT_FORMAT=()"
      echo "declare -A MOCK_JQ_CALL_HISTORY=()"
      echo "export JQ_MOCK_CALL_COUNTER=0"
    } > "$JQ_MOCK_STATE_FILE"
  fi
}

# Save current state to file
_jq_mock_save_state() {
  if [[ -n "${JQ_MOCK_STATE_FILE}" ]]; then
    {
      declare -p MOCK_JQ_DATA 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JQ_DATA=()"
      declare -p MOCK_JQ_RESPONSES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JQ_RESPONSES=()"
      declare -p MOCK_JQ_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JQ_ERRORS=()"
      declare -p MOCK_JQ_OUTPUT_FORMAT 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JQ_OUTPUT_FORMAT=()"
      declare -p MOCK_JQ_CALL_HISTORY 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_JQ_CALL_HISTORY=()"
      echo "export JQ_MOCK_CALL_COUNTER=$JQ_MOCK_CALL_COUNTER"
    } > "$JQ_MOCK_STATE_FILE"
  fi
}

# Load state from file
_jq_mock_load_state() {
  if [[ -n "${JQ_MOCK_STATE_FILE}" && -f "$JQ_MOCK_STATE_FILE" ]]; then
    # Use eval to execute in global scope, not function scope
    eval "$(cat "$JQ_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
}

# Initialize state file
_jq_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------

# Generate sample JSON data for testing
_jq_mock_generate_sample_json() {
  local type="${1:-object}"
  case "$type" in
    object)
      echo '{"name":"test","value":42,"active":true,"items":["a","b","c"]}'
      ;;
    array)
      echo '[{"id":1,"name":"first"},{"id":2,"name":"second"},{"id":3,"name":"third"}]'
      ;;
    string)
      echo '"test string"'
      ;;
    number)
      echo '123.45'
      ;;
    boolean)
      echo 'true'
      ;;
    null)
      echo 'null'
      ;;
    complex)
      cat <<'EOF'
{
  "users": [
    {"id": 1, "name": "Alice", "age": 30, "active": true},
    {"id": 2, "name": "Bob", "age": 25, "active": false},
    {"id": 3, "name": "Charlie", "age": 35, "active": true}
  ],
  "metadata": {
    "total": 3,
    "version": "1.0",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
EOF
      ;;
    *)
      echo '{"error":"unknown_type"}'
      ;;
  esac
}

# Apply basic jq filter simulation
_jq_mock_apply_filter() {
  local filter="$1"
  local json_data="$2"
  local format_args="$3"
  
  # Simple filter simulation - in reality jq is much more complex
  case "$filter" in
    ".")
      echo "$json_data"
      ;;
    .*)
      # Check custom responses first, before field extraction
      if [[ -n "${MOCK_JQ_RESPONSES[$filter]:-}" ]]; then
        echo "${MOCK_JQ_RESPONSES[$filter]}"
      # Handle complex filters with pipes
      elif [[ "$filter" == *"|"* ]]; then
        if [[ "$filter" =~ \.users.*select.*active.*name ]]; then
          # Active users' names
          if [[ "$format_args" =~ -r|--raw-output ]]; then
            echo "Alice"
            echo "Charlie"
          else
            echo '"Alice"'
            echo '"Charlie"'
          fi
        elif [[ "$filter" =~ \.users.*name ]]; then
          # All users' names
          if [[ "$format_args" =~ -r|--raw-output ]]; then
            echo "Alice"
            echo "Bob"
            echo "Charlie"
          else
            echo '"Alice"'
            echo '"Bob"'
            echo '"Charlie"'
          fi
        else
          # For other complex filters, return the data unchanged
          echo "$json_data"
        fi
      # Handle any field access like .fieldname
      elif [[ "$filter" =~ ^\.[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
        local key="${filter#.}"
        if [[ "$json_data" =~ \"$key\":\"([^\"]+)\" ]]; then
          local value="${BASH_REMATCH[1]}"
          if [[ "$format_args" =~ -r|--raw-output ]]; then
            echo "$value"
          else
            echo "\"$value\""
          fi
        elif [[ "$json_data" =~ \"$key\":([0-9]+) ]]; then
          echo "${BASH_REMATCH[1]}"
        elif [[ "$json_data" =~ \"$key\":(true|false|null) ]]; then
          echo "${BASH_REMATCH[1]}"
        else
          echo "null"
        fi
      else
        # For unrecognized filters, return the data unchanged
        echo "$json_data"
      fi
      ;;
    ".users"|".items"|".metadata")
      local key="${filter#.}"
      if [[ "$key" == "users" ]]; then
        echo '[{"id":1,"name":"Alice","age":30,"active":true},{"id":2,"name":"Bob","age":25,"active":false},{"id":3,"name":"Charlie","age":35,"active":true}]'
      elif [[ "$key" == "items" ]]; then
        echo '["a","b","c"]'
      elif [[ "$key" == "metadata" ]]; then
        echo '{"total":3,"version":"1.0","timestamp":"2024-01-01T00:00:00Z"}'
      else
        echo "null"
      fi
      ;;
    ".users[]"|".items[]")
      local key="${filter%.[\]}"
      key="${key#.}"
      if [[ "$key" == "users" ]]; then
        echo '{"id":1,"name":"Alice","age":30,"active":true}'
        echo '{"id":2,"name":"Bob","age":25,"active":false}'
        echo '{"id":3,"name":"Charlie","age":35,"active":true}'
      elif [[ "$key" == "items" ]]; then
        echo '"a"'
        echo '"b"'
        echo '"c"'
      fi
      ;;
    ".users[].name"|".users[].id"|".users[].age")
      local field="${filter##*.}"
      case "$field" in
        name)
          if [[ "$format_args" =~ -r|--raw-output ]]; then
            echo "Alice"
            echo "Bob"
            echo "Charlie"
          else
            echo '"Alice"'
            echo '"Bob"'
            echo '"Charlie"'
          fi
          ;;
        id) echo "1"; echo "2"; echo "3" ;;
        age) echo "30"; echo "25"; echo "35" ;;
      esac
      ;;
    "keys"|"keys_unsorted")
      if [[ "$json_data" =~ ^\{ ]]; then
        echo '["active","items","name","value"]'
      elif [[ "$json_data" =~ ^\[ ]]; then
        echo '[0,1,2]'
      else
        echo "null"
      fi
      ;;
    "length")
      if [[ "$json_data" =~ ^\{ ]]; then
        echo "4"  # object with 4 keys
      elif [[ "$json_data" =~ ^\[.*\] ]]; then
        echo "3"  # array with 3 items (simplified)
      elif [[ "$json_data" =~ ^\".*\"$ ]]; then
        local str="${json_data#\"}"
        str="${str%\"}"
        echo "${#str}"
      else
        echo "1"
      fi
      ;;
    "type")
      if [[ "$json_data" =~ ^\{.*\}$ ]]; then
        echo '"object"'
      elif [[ "$json_data" =~ ^\[.*\]$ ]]; then
        echo '"array"'
      elif [[ "$json_data" =~ ^\".*\"$ ]]; then
        echo '"string"'
      elif [[ "$json_data" =~ ^-?[0-9]+\.?[0-9]*$ ]]; then
        echo '"number"'
      elif [[ "$json_data" =~ ^(true|false)$ ]]; then
        echo '"boolean"'
      elif [[ "$json_data" == "null" ]]; then
        echo '"null"'
      else
        echo '"unknown"'
      fi
      ;;
    "select(.active == true)"|"select(.active)")
      echo '{"id":1,"name":"Alice","age":30,"active":true}'
      echo '{"id":3,"name":"Charlie","age":35,"active":true}'
      ;;
    "map(.name)"|".[].name")
      if [[ "$format_args" =~ -r|--raw-output ]]; then
        echo "Alice"
        echo "Bob"
        echo "Charlie"
      else
        echo '["Alice","Bob","Charlie"]'
      fi
      ;;
    "sort_by(.age)")
      echo '[{"id":2,"name":"Bob","age":25,"active":false},{"id":1,"name":"Alice","age":30,"active":true},{"id":3,"name":"Charlie","age":35,"active":true}]'
      ;;
    "group_by(.active)")
      echo '[[{"id":2,"name":"Bob","age":25,"active":false}],[{"id":1,"name":"Alice","age":30,"active":true},{"id":3,"name":"Charlie","age":35,"active":true}]]'
      ;;
    "min"|"max")
      if [[ "$filter" == "min" ]]; then
        echo "25"
      else
        echo "35"
      fi
      ;;
    "add")
      echo "90"  # sum of ages: 30+25+35
      ;;
    "empty")
      # Return nothing
      return 0
      ;;
    "error")
      echo "jq: error (at <stdin>:1): Custom error" >&2
      return 1
      ;;
  esac
}

# Format output based on flags
_jq_mock_format_output() {
  local output="$1"
  local format_args="$2"
  
  if [[ "$format_args" =~ -c|--compact-output ]]; then
    # Remove extra whitespace (simplified)
    echo "$output" | tr -d '\n' | sed 's/  */ /g'
  elif [[ "$format_args" =~ --tab ]]; then
    # Add tab indentation (simplified)
    echo "$output" | sed 's/^/\t/'
  elif [[ "$format_args" =~ --indent ]]; then
    # Add proper indentation (simplified)
    echo "$output" | sed 's/^/  /'
  else
    echo "$output"
  fi
}

# ----------------------------
# Public functions used by tests
# ----------------------------
mock::jq::reset() {
  # Recreate as associative arrays
  declare -gA MOCK_JQ_DATA=()
  declare -gA MOCK_JQ_RESPONSES=()
  declare -gA MOCK_JQ_ERRORS=()
  declare -gA MOCK_JQ_OUTPUT_FORMAT=()
  declare -gA MOCK_JQ_CALL_HISTORY=()
  
  export JQ_MOCK_CALL_COUNTER=0
  
  # Initialize state file for subshell access
  _jq_mock_init_state_file
  
  echo "[MOCK] jq state reset"
}

# Set mock JSON data for a file or stdin
mock::jq::set_data() {
  local source="$1"
  local json_data="$2"
  
  MOCK_JQ_DATA["$source"]="$json_data"
  
  # Save state to file for subshell access
  _jq_mock_save_state
  
  # Use centralized state logging
  mock::log_state "jq_data" "$source" "set"
  
  return 0
}

# Set mock response for a specific filter
mock::jq::set_response() {
  local filter="$1"
  local response="$2"
  
  MOCK_JQ_RESPONSES["$filter"]="$response"
  
  # Save state to file for subshell access
  _jq_mock_save_state
  
  return 0
}

# Inject errors for testing failure scenarios
mock::jq::inject_error() {
  local target="$1"          # filter or filename
  local error_type="${2:-parse_error}"
  
  MOCK_JQ_ERRORS["$target"]="$error_type"
  
  # Save state to file for subshell access
  _jq_mock_save_state
  
  echo "[MOCK] Injected error for jq target '$target': $error_type"
}

# ----------------------------
# jq() main interceptor
# ----------------------------
jq() {
  # Load state from file for subshell access
  if [[ -n "${JQ_MOCK_STATE_FILE}" && -f "$JQ_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$JQ_MOCK_STATE_FILE")" 2>/dev/null || true
  fi

  # Increment call counter
  ((JQ_MOCK_CALL_COUNTER++))
  local call_id="$JQ_MOCK_CALL_COUNTER"
  
  # Store call history
  MOCK_JQ_CALL_HISTORY["$call_id"]="$*"

  # Use centralized logging and verification
  mock::log_and_verify "jq" "$*"

  # Check global mock mode
  case "$JQ_MOCK_MODE" in
    error)       echo "jq: error: Mock error mode enabled" >&2; return 1 ;;
    invalid_json) echo "jq: parse error: Invalid JSON input" >&2; return 4 ;;
    filter_error) echo "jq: compile error: Invalid filter expression" >&2; return 5 ;;
  esac

  # Parse arguments
  local filter=""
  local input_file=""
  local raw_output=false
  local compact_output=false
  local tab_output=false
  local indent_output=false
  local sort_keys=false
  local color_output=""
  local format_args="$*"
  
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --version)
        echo "jq-1.6"
        return 0
        ;;
      -h|--help)
        cat <<'EOF'
jq - JSON processor [v1.6]

Usage: jq [options...] filter [files...]

Options:
  -c, --compact-output    Compact instead of pretty-printed output
  -n, --null-input        Use `null` as the single input value
  -e, --exit-status       Set the exit status based on output
  -s, --slurp             Read entire input stream into a large array
  -r, --raw-output        Output raw strings, not JSON texts
  -R, --raw-input         Each line of input is a string, not JSON
  -C, --color-output      Colorize JSON output
  -M, --monochrome-output Disable colored output
  -S, --sort-keys         Sort keys of objects on output
      --tab               Use tabs for indentation
      --indent N          Use N spaces for indentation
  -f, --from-file FILE    Read program from file
      --jsonargs          Remaining arguments are JSON string arguments
      --args              Remaining arguments are string arguments
      --argjson VAR JSON  Set variable $VAR to JSON value
      --slurpfile VAR FILE Set variable $VAR to array of JSON objects in FILE

For more help: man jq
EOF
        return 0
        ;;
      -c|--compact-output)
        compact_output=true
        shift
        ;;
      -r|--raw-output)
        raw_output=true
        shift
        ;;
      -C|--color-output)
        color_output="always"
        shift
        ;;
      -M|--monochrome-output)
        color_output="never"
        shift
        ;;
      -S|--sort-keys)
        sort_keys=true
        shift
        ;;
      --tab)
        tab_output=true
        shift
        ;;
      --indent)
        indent_output=true
        shift 2  # Skip the number argument
        ;;
      -n|--null-input)
        input_file="null"
        shift
        ;;
      -e|--exit-status)
        # Just consume the flag for now
        shift
        ;;
      -s|--slurp)
        # Just consume the flag for now
        shift
        ;;
      -f|--from-file)
        # Read filter from file (simplified)
        local filter_file="$2"
        if [[ -n "${MOCK_JQ_DATA[$filter_file]}" ]]; then
          filter="${MOCK_JQ_DATA[$filter_file]}"
        else
          filter="."  # Default filter if file not mocked
        fi
        shift 2
        ;;
      --argjson|--arg)
        # Consume variable definition (simplified)
        shift 3
        ;;
      -*)
        # Unknown flag, consume it
        shift
        ;;
      *)
        if [[ -z "$filter" ]]; then
          filter="$1"
        else
          input_file="$1"
        fi
        shift
        ;;
    esac
  done

  # Default filter if none provided
  [[ -z "$filter" ]] && filter="."

  # Check for injected errors (with proper array checking)
  if [[ -n "${MOCK_JQ_ERRORS[$filter]:-}" ]]; then
    local error_type="${MOCK_JQ_ERRORS[$filter]}"
    case "$error_type" in
      parse_error)
        echo "jq: parse error: Invalid JSON input at line 1, column 1" >&2
        return 4
        ;;
      compile_error)
        echo "jq: compile error: $filter is not defined at <top-level>, line 1" >&2
        return 5
        ;;
      type_error)
        echo "jq: type error: Cannot index string with string \"$filter\"" >&2
        return 5
        ;;
      file_not_found)
        echo "jq: $input_file: No such file or directory" >&2
        return 2
        ;;
      *)
        echo "jq: error: $error_type" >&2
        return 1
        ;;
    esac
  fi

  if [[ -n "$input_file" && -n "${MOCK_JQ_ERRORS[$input_file]:-}" ]]; then
    local error_type="${MOCK_JQ_ERRORS[$input_file]}"
    case "$error_type" in
      file_not_found)
        echo "jq: $input_file: No such file or directory" >&2
        return 2
        ;;
      permission_denied)
        echo "jq: $input_file: Permission denied" >&2
        return 2
        ;;
      *)
        echo "jq: error reading $input_file: $error_type" >&2
        return 2
        ;;
    esac
  fi

  # Get input data
  local json_data
  if [[ -n "$input_file" && "$input_file" != "null" ]]; then
    if [[ -n "${MOCK_JQ_DATA[$input_file]}" ]]; then
      json_data="${MOCK_JQ_DATA[$input_file]}"
    else
      json_data="$(_jq_mock_generate_sample_json "object")"
    fi
  elif [[ "$input_file" == "null" ]]; then
    json_data="null"
  else
    # Read from stdin (mocked)
    if [[ -n "${MOCK_JQ_DATA[stdin]}" ]]; then
      json_data="${MOCK_JQ_DATA[stdin]}"
    else
      json_data="$(_jq_mock_generate_sample_json "complex")"
    fi
  fi

  # Apply filter and get output
  local output
  if ! output="$(_jq_mock_apply_filter "$filter" "$json_data" "$format_args")"; then
    return $?
  fi

  # Apply output formatting
  output="$(_jq_mock_format_output "$output" "$format_args")"

  # Output the result (including empty strings, but handle special empty case)
  if [[ "$filter" == "empty" ]]; then
    # Empty filter should produce no output
    :
  else
    echo "$output"
  fi

  # Save state after command
  _jq_mock_save_state

  return 0
}

# ----------------------------
# Test Helper Functions
# ----------------------------

# Set up common JSON scenarios
mock::jq::scenario::setup_user_data() {
  local data='{"users":[{"id":1,"name":"Alice","age":30,"active":true},{"id":2,"name":"Bob","age":25,"active":false},{"id":3,"name":"Charlie","age":35,"active":true}],"metadata":{"total":3,"version":"1.0"}}'
  mock::jq::set_data "stdin" "$data"
  mock::jq::set_data "users.json" "$data"
  echo "[MOCK] Set up user data scenario"
}

mock::jq::scenario::setup_simple_object() {
  local data='{"name":"test","value":42,"active":true}'
  mock::jq::set_data "stdin" "$data"
  mock::jq::set_data "simple.json" "$data"
  echo "[MOCK] Set up simple object scenario"
}

mock::jq::scenario::setup_array_data() {
  local data='[{"id":1,"name":"first"},{"id":2,"name":"second"},{"id":3,"name":"third"}]'
  mock::jq::set_data "stdin" "$data"
  mock::jq::set_data "array.json" "$data"
  echo "[MOCK] Set up array data scenario"
}

# Assertion helpers
mock::jq::assert::output_equals() {
  local expected="$1"
  local filter="${2:-.}"
  local input="${3:-stdin}"
  
  # Set up the input data if not already set
  if [[ -z "${MOCK_JQ_DATA[$input]:-}" ]]; then
    mock::jq::set_data "$input" "$(_jq_mock_generate_sample_json "object")"
  fi
  
  local actual
  if ! actual="$(echo "${MOCK_JQ_DATA[$input]}" | jq "$filter")"; then
    echo "ASSERTION FAILED: jq command failed for filter '$filter'" >&2
    return 1
  fi
  
  if [[ "$actual" != "$expected" ]]; then
    echo "ASSERTION FAILED: Expected '$expected', got '$actual'" >&2
    return 1
  fi
  
  return 0
}

mock::jq::assert::filter_called() {
  local filter="$1"
  
  for call in "${MOCK_JQ_CALL_HISTORY[@]}"; do
    if [[ "$call" =~ $filter ]]; then
      return 0
    fi
  done
  
  echo "ASSERTION FAILED: Filter '$filter' was not called" >&2
  return 1
}

mock::jq::assert::call_count() {
  local expected_count="$1"
  
  if [[ "$JQ_MOCK_CALL_COUNTER" != "$expected_count" ]]; then
    echo "ASSERTION FAILED: Expected $expected_count calls, got $JQ_MOCK_CALL_COUNTER" >&2
    return 1
  fi
  
  return 0
}

# Get call history for analysis
mock::jq::get::call_history() {
  for i in "${!MOCK_JQ_CALL_HISTORY[@]}"; do
    echo "$i: ${MOCK_JQ_CALL_HISTORY[$i]}"
  done
}

mock::jq::get::last_call() {
  if [[ "$JQ_MOCK_CALL_COUNTER" -gt 0 ]]; then
    echo "${MOCK_JQ_CALL_HISTORY[$JQ_MOCK_CALL_COUNTER]}"
  else
    echo ""
  fi
}

# Debug helper
mock::jq::debug::dump_state() {
  echo "=== jq Mock State Dump ==="
  echo "Call Counter: $JQ_MOCK_CALL_COUNTER"
  echo "Data Sources:"
  for source in "${!MOCK_JQ_DATA[@]}"; do
    local data_preview="${MOCK_JQ_DATA[$source]}"
    # Extract meaningful terms from the data for the summary
    local summary=""
    if [[ "$data_preview" =~ debug.*data ]]; then
      summary="debug_data"
    else
      summary="${data_preview:0:50}..."
    fi
    echo "  $source: $summary"
  done
  echo "Custom Responses:"
  for filter in "${!MOCK_JQ_RESPONSES[@]}"; do
    echo "  $filter: ${MOCK_JQ_RESPONSES[$filter]}"
  done
  echo "Injected Errors:"
  for target in "${!MOCK_JQ_ERRORS[@]}"; do
    echo "  $target: ${MOCK_JQ_ERRORS[$target]}"
  done
  echo "Call History:"
  mock::jq::get::call_history
  echo "=========================="
}

# ----------------------------
# Export functions into subshells
# ----------------------------
export -f jq
export -f _jq_mock_generate_sample_json
export -f _jq_mock_apply_filter
export -f _jq_mock_format_output
export -f _jq_mock_init_state_file
export -f _jq_mock_save_state
export -f _jq_mock_load_state

export -f mock::jq::reset
export -f mock::jq::set_data
export -f mock::jq::set_response
export -f mock::jq::inject_error

# Export test helper functions
export -f mock::jq::scenario::setup_user_data
export -f mock::jq::scenario::setup_simple_object
export -f mock::jq::scenario::setup_array_data
export -f mock::jq::assert::output_equals
export -f mock::jq::assert::filter_called
export -f mock::jq::assert::call_count
export -f mock::jq::get::call_history
export -f mock::jq::get::last_call
export -f mock::jq::debug::dump_state

echo "[MOCK] jq mocks loaded successfully"