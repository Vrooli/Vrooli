#!/usr/bin/env bash
# System Command Mocks for Bats tests.
# Provides comprehensive system command mocking (systemctl, ps, kill, etc.) with consistent state management.

# Prevent duplicate loading
if [[ "${SYSTEM_MOCKS_LOADED:-}" == "true" ]]; then
  return 0
fi
export SYSTEM_MOCKS_LOADED="true"

# Load mock utilities for standardized logging
MOCK_UTILS_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ -f "$MOCK_UTILS_SCRIPT_DIR/logs.sh" ]]; then
    source "$MOCK_UTILS_SCRIPT_DIR/logs.sh"
else
    echo "[SYSTEMCTL_MOCK] WARNING: Could not load mock utilities from $MOCK_UTILS_SCRIPT_DIR/logs.sh" >&2
fi

# ----------------------------
# Global mock state & options
# ----------------------------
# Modes: normal, offline, error, permission_denied
export SYSTEMCTL_MOCK_MODE="${SYSTEMCTL_MOCK_MODE:-normal}"

# In-memory state storage - SystemCtl
declare -A MOCK_SYSTEMCTL_SERVICES=()      # service -> "state|enabled|substate|main_pid|..."
declare -A MOCK_SYSTEMCTL_UNITS=()         # unit -> "loaded|active|state|description"
declare -A MOCK_SYSTEMCTL_ERRORS=()        # command -> error_type
declare -A MOCK_SYSTEMCTL_TARGETS=()       # target -> "active|inactive"
declare -A MOCK_SYSTEMCTL_TIMERS=()        # timer -> "active|inactive|next"
declare -A MOCK_SYSTEMCTL_SOCKETS=()       # socket -> "listening|closed"
declare -A MOCK_SYSTEMCTL_MOUNTS=()        # mount -> "mounted|unmounted"

# In-memory state storage - System Commands  
declare -A MOCK_SYSTEM_PROCESSES=()        # [pid] -> "name|status|ppid|command|user|cpu|mem|start_time"
declare -A MOCK_SYSTEM_PROCESS_PATTERNS=() # [pattern] -> "pid1,pid2,pid3" for pgrep optimization
declare -A MOCK_SYSTEM_INFO=()             # [key] -> "value" for system properties
declare -A MOCK_SYSTEM_USERS=()            # [username] -> "uid|gid|groups"
declare -A MOCK_SYSTEM_COMMANDS=()         # [command] -> "path" for which/command/type
declare -A MOCK_SYSTEM_ERRORS=()           # [command] -> error_type

# File-based state persistence for subshell access (BATS compatibility)
export SYSTEM_MOCK_STATE_FILE="${MOCK_LOG_DIR:-/tmp}/system_mock_state.$$"

# Initialize state file
_system_mock_init_state_file() {
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" ]]; then
    {
      echo "declare -A MOCK_SYSTEMCTL_SERVICES=()"
      echo "declare -A MOCK_SYSTEMCTL_UNITS=()"
      echo "declare -A MOCK_SYSTEMCTL_ERRORS=()"
      echo "declare -A MOCK_SYSTEMCTL_TARGETS=()"
      echo "declare -A MOCK_SYSTEMCTL_TIMERS=()"
      echo "declare -A MOCK_SYSTEMCTL_SOCKETS=()"
      echo "declare -A MOCK_SYSTEMCTL_MOUNTS=()"
      echo "declare -A MOCK_SYSTEM_PROCESSES=()"
      echo "declare -A MOCK_SYSTEM_PROCESS_PATTERNS=()"
      echo "declare -A MOCK_SYSTEM_INFO=()"
      echo "declare -A MOCK_SYSTEM_USERS=()"
      echo "declare -A MOCK_SYSTEM_COMMANDS=()"
      echo "declare -A MOCK_SYSTEM_ERRORS=()"
    } > "$SYSTEM_MOCK_STATE_FILE"
  fi
}

# Save current state to file
_system_mock_save_state() {
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" ]]; then
    {
      # Use declare -gA for global associative arrays - SystemCtl
      declare -p MOCK_SYSTEMCTL_SERVICES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEMCTL_SERVICES=()"
      declare -p MOCK_SYSTEMCTL_UNITS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEMCTL_UNITS=()"
      declare -p MOCK_SYSTEMCTL_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEMCTL_ERRORS=()"
      declare -p MOCK_SYSTEMCTL_TARGETS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEMCTL_TARGETS=()"
      declare -p MOCK_SYSTEMCTL_TIMERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEMCTL_TIMERS=()"
      declare -p MOCK_SYSTEMCTL_SOCKETS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEMCTL_SOCKETS=()"
      declare -p MOCK_SYSTEMCTL_MOUNTS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEMCTL_MOUNTS=()"
      # System Commands
      declare -p MOCK_SYSTEM_PROCESSES 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEM_PROCESSES=()"
      declare -p MOCK_SYSTEM_PROCESS_PATTERNS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEM_PROCESS_PATTERNS=()"
      declare -p MOCK_SYSTEM_INFO 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEM_INFO=()"
      declare -p MOCK_SYSTEM_USERS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEM_USERS=()"
      declare -p MOCK_SYSTEM_COMMANDS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEM_COMMANDS=()"
      declare -p MOCK_SYSTEM_ERRORS 2>/dev/null | sed 's/declare -A/declare -gA/' || echo "declare -gA MOCK_SYSTEM_ERRORS=()"
    } > "$SYSTEM_MOCK_STATE_FILE"
  fi
}

# Load state from file  
_system_mock_load_state() {
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    # Use eval to execute in global scope, not function scope
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
}

# Initialize state file
_system_mock_init_state_file

# ----------------------------
# Utilities
# ----------------------------

_mock_service_name() {
  local name="$1"
  # Strip .service suffix if present
  echo "${name%.service}"
}

_mock_unit_name() {
  local name="$1"
  # Add .service suffix if no extension present
  if [[ "$name" != *.* ]]; then
    echo "${name}.service"
  else
    echo "$name"
  fi
}

_mock_systemd_time() {
  # Generate realistic systemd timestamp
  date '+%a %Y-%m-%d %H:%M:%S %Z' 2>/dev/null || echo "Mon 2024-01-01 12:00:00 UTC"
}

_mock_pid() {
  # Generate mock PID
  echo "$((RANDOM % 9000 + 1000))"
}

_mock_since_time() {
  # Generate "since" timestamp
  local seconds_ago=$((RANDOM % 3600))
  if command -v date >/dev/null 2>&1; then
    date -d "$seconds_ago seconds ago" '+%a %Y-%m-%d %H:%M:%S %Z' 2>/dev/null || echo "Mon 2024-01-01 12:00:00 UTC"
  else
    echo "Mon 2024-01-01 12:00:00 UTC"
  fi
}

# ----------------------------
# Public functions used by tests
# ----------------------------
mock::systemctl::reset() {
  # Recreate as associative arrays (not indexed arrays)
  declare -gA MOCK_SYSTEMCTL_SERVICES=()
  declare -gA MOCK_SYSTEMCTL_UNITS=()
  declare -gA MOCK_SYSTEMCTL_ERRORS=()
  declare -gA MOCK_SYSTEMCTL_TARGETS=()
  declare -gA MOCK_SYSTEMCTL_TIMERS=()
  declare -gA MOCK_SYSTEMCTL_SOCKETS=()
  declare -gA MOCK_SYSTEMCTL_MOUNTS=()
  
  # Initialize state file for subshell access
  _system_mock_init_state_file
  
  echo "[MOCK] SystemCtl state reset"
}

# System-wide reset for all commands
mock::system::reset() {
  # Reset SystemCtl state
  declare -gA MOCK_SYSTEMCTL_SERVICES=()
  declare -gA MOCK_SYSTEMCTL_UNITS=()
  declare -gA MOCK_SYSTEMCTL_ERRORS=()
  declare -gA MOCK_SYSTEMCTL_TARGETS=()
  declare -gA MOCK_SYSTEMCTL_TIMERS=()
  declare -gA MOCK_SYSTEMCTL_SOCKETS=()
  declare -gA MOCK_SYSTEMCTL_MOUNTS=()
  
  # Reset System Command state
  declare -gA MOCK_SYSTEM_PROCESSES=()
  declare -gA MOCK_SYSTEM_PROCESS_PATTERNS=()
  declare -gA MOCK_SYSTEM_INFO=()
  declare -gA MOCK_SYSTEM_USERS=()
  declare -gA MOCK_SYSTEM_COMMANDS=()
  declare -gA MOCK_SYSTEM_ERRORS=()
  
  # Initialize default system state
  mock::system::init_defaults
  
  # Initialize and save state file for subshell access
  _system_mock_init_state_file
  _system_mock_save_state
  
  echo "[MOCK] System state reset (all commands)"
}

# Enable automatic cleanup between tests
mock::systemctl::enable_auto_cleanup() {
  export SYSTEMCTL_MOCK_AUTO_CLEANUP=true
}

# ----------------------------
# System Command Utilities
# ----------------------------

# Initialize default system state
mock::system::init_defaults() {
  # Default system info
  MOCK_SYSTEM_INFO["hostname"]="testhost"
  MOCK_SYSTEM_INFO["kernel"]="5.4.0-test"
  MOCK_SYSTEM_INFO["arch"]="x86_64"
  MOCK_SYSTEM_INFO["os"]="Linux"
  MOCK_SYSTEM_INFO["uptime"]="1704067200"
  MOCK_SYSTEM_INFO["system_running"]="running"
  
  # Default users
  MOCK_SYSTEM_USERS["root"]="0|0|root"
  MOCK_SYSTEM_USERS["testuser"]="1000|1000|testuser,adm,sudo,docker"
  
  # Set current user (avoid circular dependency with whoami mock)
  MOCK_SYSTEM_INFO["current_user"]="testuser"
  
  # Default commands
  MOCK_SYSTEM_COMMANDS["bash"]="/bin/bash"
  MOCK_SYSTEM_COMMANDS["sh"]="/bin/sh"
  MOCK_SYSTEM_COMMANDS["ps"]="/bin/ps"
  MOCK_SYSTEM_COMMANDS["kill"]="/bin/kill"
  MOCK_SYSTEM_COMMANDS["systemctl"]="/usr/bin/systemctl"
  MOCK_SYSTEM_COMMANDS["which"]="/usr/bin/which"
  MOCK_SYSTEM_COMMANDS["id"]="/usr/bin/id"
  MOCK_SYSTEM_COMMANDS["whoami"]="/usr/bin/whoami"
  MOCK_SYSTEM_COMMANDS["uname"]="/bin/uname"
  MOCK_SYSTEM_COMMANDS["date"]="/bin/date"
  
  # Default processes - use static values to avoid circular dependencies
  MOCK_SYSTEM_PROCESSES["1"]="systemd|running|0|/sbin/init|root|0.0|0.1|1704067200"
  MOCK_SYSTEM_PROCESSES["2"]="kthreadd|sleeping|0|[kthreadd]|root|0.0|0.0|1704067200"
  MOCK_SYSTEM_PROCESSES["10001"]="bash|running|1|/bin/bash|testuser|0.1|0.5|1704067200"
  
  # Update pattern cache
  mock::system::_update_pattern_cache
  
  echo "[MOCK] System defaults initialized"
}

# Update process pattern cache for pgrep optimization
mock::system::_update_pattern_cache() {
  # Clear existing patterns
  declare -gA MOCK_SYSTEM_PROCESS_PATTERNS=()
  
  # Build pattern cache from current processes
  for pid in "${!MOCK_SYSTEM_PROCESSES[@]}"; do
    IFS='|' read -r name status ppid command user cpu mem start_time <<< "${MOCK_SYSTEM_PROCESSES[$pid]}"
    
    # Add to patterns based on name
    if [[ -n "${MOCK_SYSTEM_PROCESS_PATTERNS[$name]}" ]]; then
      MOCK_SYSTEM_PROCESS_PATTERNS["$name"]+=",$pid"
    else
      MOCK_SYSTEM_PROCESS_PATTERNS["$name"]="$pid"
    fi
    
    # Add to patterns based on command basename
    local cmd_basename="$(basename "$command" 2>/dev/null || echo "$command")"
    if [[ "$cmd_basename" != "$name" ]]; then
      if [[ -n "${MOCK_SYSTEM_PROCESS_PATTERNS[$cmd_basename]}" ]]; then
        MOCK_SYSTEM_PROCESS_PATTERNS["$cmd_basename"]+=",$pid"
      else
        MOCK_SYSTEM_PROCESS_PATTERNS["$cmd_basename"]="$pid"
      fi
    fi
  done
}

# Generate mock PID
_mock_system_pid() {
  local pid
  # Generate PIDs in mock range to avoid conflicts with real processes
  while true; do
    pid=$((10000 + RANDOM % 50000))
    # Ensure PID doesn't already exist
    if [[ -z "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
      echo "$pid"
      return 0
    fi
  done
}

# Get current timestamp
_mock_system_timestamp() {
  # Use builtin date command to avoid circular dependency with our mock
  command date '+%s' 2>/dev/null || echo '1704067200'
}

# Inject errors for testing failure scenarios
mock::systemctl::inject_error() {
  local cmd="$1"
  local error_type="${2:-generic}"
  MOCK_SYSTEMCTL_ERRORS["$cmd"]="$error_type"
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  echo "[MOCK] Injected error for systemctl $cmd: $error_type"
}

# Inject errors for system commands
mock::system::inject_error() {
  local cmd="$1"
  local error_type="${2:-generic}"
  MOCK_SYSTEM_ERRORS["$cmd"]="$error_type"
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  echo "[MOCK] Injected error for system command $cmd: $error_type"
}

# ----------------------------
# Public setters used by tests
# ----------------------------
mock::systemctl::set_service_state() {
  local service="$1" 
  local state="$2" 
  local enabled="${3:-enabled}"
  local substate="${4:-running}"
  local main_pid="${5:-$(_mock_pid)}"
  
  local service_name="$(_mock_service_name "$service")"
  MOCK_SYSTEMCTL_SERVICES["$service_name"]="$state|$enabled|$substate|$main_pid"

  # Save state to file for subshell access
  _system_mock_save_state

  # Use centralized state logging
  if command -v mock::log_state &>/dev/null; then
    mock::log_state "systemctl_service_state" "$service_name" "$state"
  fi

  return 0
}

# ----------------------------
# System Command Setters
# ----------------------------

# Set process state - shared by ps, pgrep, pkill, kill
mock::system::set_process_state() {
  local pid="$1"
  local name="$2"
  local status="${3:-running}"
  local ppid="${4:-1}"
  local command="${5:-/usr/bin/$name}"
  local user="${6:-testuser}"
  local cpu="${7:-0.0}"
  local mem="${8:-0.1}"
  local start_time="${9:-$(_mock_system_timestamp)}"
  
  MOCK_SYSTEM_PROCESSES["$pid"]="$name|$status|$ppid|$command|$user|$cpu|$mem|$start_time"
  
  # Update pattern cache
  mock::system::_update_pattern_cache
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  # Use centralized state logging
  if command -v mock::log_state &>/dev/null; then
    mock::log_state "system_process_state" "$pid" "$name:$status"
  fi
  
  return 0
}

# Remove process (used by kill/pkill)
mock::system::remove_process() {
  local pid="$1"
  
  if [[ -n "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
    unset MOCK_SYSTEM_PROCESSES["$pid"]
    
    # Update pattern cache
    mock::system::_update_pattern_cache
    
    # Save state to file for subshell access
    _system_mock_save_state
    
    if command -v mock::log_state &>/dev/null; then
      mock::log_state "system_process_removed" "$pid" "killed"
    fi
    
    return 0
  fi
  
  return 1
}

# Set system info
mock::system::set_info() {
  local key="$1"
  local value="$2"
  
  MOCK_SYSTEM_INFO["$key"]="$value"
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  return 0
}

# Set user info
mock::system::set_user() {
  local username="$1"
  local uid="$2"
  local gid="$3"
  local groups="${4:-$username}"
  
  MOCK_SYSTEM_USERS["$username"]="$uid|$gid|$groups"
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  return 0
}

# Set command path
mock::system::set_command() {
  local command="$1"
  local path="$2"
  
  MOCK_SYSTEM_COMMANDS["$command"]="$path"
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  return 0
}

mock::systemctl::set_unit_state() {
  local unit="$1"
  local loaded="$2"
  local active="$3" 
  local state="$4"
  local description="${5:-Mock Unit}"
  
  local unit_name="$(_mock_unit_name "$unit")"
  MOCK_SYSTEMCTL_UNITS["$unit_name"]="$loaded|$active|$state|$description"
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  return 0
}

mock::systemctl::set_target_state() {
  local target="$1" 
  local state="$2"
  MOCK_SYSTEMCTL_TARGETS["$target"]="$state"
  
  # Save state to file for subshell access
  _system_mock_save_state
  
  return 0
}

# ----------------------------
# systemctl() main interceptor
# ----------------------------
systemctl() {
  # Load state from file for subshell access (inline to avoid function scoping issues)
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi

  # Use centralized logging and verification
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "$*"
  elif command -v mock::log_call &>/dev/null; then
    mock::log_call "systemctl" "$*"
  fi

  case "$SYSTEMCTL_MOCK_MODE" in
    offline) echo "Failed to connect to bus: No such file or directory" >&2; return 1 ;;
    error)   echo "Failed to execute operation: Connection timed out" >&2; return 1 ;;
    permission_denied) echo "Access denied" >&2; return 1 ;;
  esac

  # Check for injected errors
  local cmd_check="${1:-}"
  if [[ -n "${MOCK_SYSTEMCTL_ERRORS[$cmd_check]}" ]]; then
    local error_type="${MOCK_SYSTEMCTL_ERRORS[$cmd_check]}"
    case "$error_type" in
      service_not_found)
        echo "Unit ${2:-unknown}.service could not be found." >&2
        return 5
        ;;
      permission_denied)
        echo "==== AUTHENTICATING FOR org.freedesktop.systemd1.manage-units ====" >&2
        echo "Authentication required to manage system services or other units" >&2
        return 1
        ;;
      failed_to_start)
        echo "Job for ${2:-unknown}.service failed because the control process exited with error code." >&2
        return 1
        ;;
      failed_to_stop)
        echo "Job for ${2:-unknown}.service failed because a timeout was exceeded." >&2
        return 1
        ;;
      *)
        echo "SystemCtl operation failed: $error_type" >&2
        return 1
        ;;
    esac
  fi

  # Handle global flags first
  local no_pager=false
  local long_output=false
  local quiet=false
  local user_mode=false
  
  # Parse global flags and filter them out
  local args=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --no-pager) no_pager=true; shift ;;
      -l|--full) long_output=true; shift ;;
      -q|--quiet) quiet=true; shift ;;
      --user) user_mode=true; shift ;;
      --system) shift ;;  # default mode, ignore
      --) shift; args+=("$@"); break ;;
      *) args+=("$1"); shift ;;
    esac
  done
  
  # Restore arguments
  set -- "${args[@]}"

  local cmd="$1"; shift || true
  local exit_code=0
  
  case "$cmd" in
    start|stop|restart|reload|reload-or-restart)
        mock::systemctl::lifecycle "$cmd" "$@" || exit_code=$?
        ;;
    enable|disable|mask|unmask|preset)
        mock::systemctl::configuration "$cmd" "$@" || exit_code=$?
        ;;
    status)     mock::systemctl::status "$@" || exit_code=$? ;;
    is-active)  mock::systemctl::is_active "$@" || exit_code=$? ;;
    is-enabled) mock::systemctl::is_enabled "$@" || exit_code=$? ;;
    is-failed)  mock::systemctl::is_failed "$@" || exit_code=$? ;;
    is-system-running) mock::systemctl::is_system_running "$@" || exit_code=$? ;;
    list-units) mock::systemctl::list_units "$@" || exit_code=$? ;;
    list-unit-files) mock::systemctl::list_unit_files "$@" || exit_code=$? ;;
    show)       mock::systemctl::show "$@" || exit_code=$? ;;
    cat)        mock::systemctl::cat "$@" || exit_code=$? ;;
    daemon-reload|daemon-reexec) mock::systemctl::daemon_control "$cmd" "$@" || exit_code=$? ;;
    default|rescue|emergency|halt|poweroff|reboot) mock::systemctl::system_control "$cmd" "$@" || exit_code=$? ;;
    --version) echo "systemd 249 (249.11-0ubuntu3.12)"; echo "+PAM +AUDIT +SELINUX +IMA +APPARMOR +SMACK +SYSVINIT +UTMP +LIBCRYPTSETUP +GCRYPT +GNUTLS +ACL +XZ +LZ4 +ZSTD +SECCOMP +BLKID +ELFUTILS +KMOD +IDN2 -IDN +PCRE2 default-hierarchy=unified" ;;
    --help)
      cat <<'EOF'
systemctl [OPTIONS...] COMMAND ...

Query or send control commands to the systemd service manager.

  -h --help           Show this help
     --version        Show package version
     --system         Connect to system manager
     --user           Connect to user service manager
  -H --host=[USER@]HOST
                      Operate on remote host
  -M --machine=CONTAINER
                      Operate on local container
  -t --type=TYPE      List units of a particular type
     --state=STATE    List units with particular LOAD or SUB or ACTIVE state
  -p --property=NAME  Show only properties by this name
  -a --all            Show all loaded units/properties, including dead empty
                      ones. To list all units installed on the system, use
                      the 'list-unit-files' command instead.
     --reverse        Show reverse dependencies with 'list-dependencies'
  -l --full           Don't ellipsize unit names on output
     --no-pager       Do not pipe output into a pager
     --no-legend      Do not print a legend (column headers and hints)
     --no-wall        Don't send wall message before halt/power-off/reboot
     --dry-run        Only print what would be done
  -q --quiet          Suppress output
     --wait           For (re)start, wait until service stopped again
                      For is-system-running, wait until startup is completed
     --no-block       Do not wait until operation finished
     --no-reload      Don't reload daemon after en-/dis-abling unit files
     --no-ask-password
                      Do not ask for system passwords
     --global         Enable/disable/mask unit files globally
     --runtime        Enable/disable/mask unit files temporarily until next
                      reboot
  -f --force          When enabling unit files, override existing symlinks
                      When shutting down, execute action immediately
     --preset-mode=   Apply only enable, only disable, or all presets
     --root=PATH      Enable/disable/mask unit files in the specified root
                      directory
  -n --lines=INTEGER  Number of journal entries to show
  -o --output=STRING  Change journal output mode (short, short-precise,
                             short-iso, short-full, short-monotonic, short-unix,
                             verbose, export, json, json-pretty, json-sse, cat)
     --firmware-setup Tell the firmware to show the setup menu on next boot
     --plain          Print unit dependencies as a list instead of a tree
EOF
      return 0
      ;;
    *)
      echo "Unknown operation '$cmd'." >&2
      exit_code=1
      ;;
  esac
  
  # Save state after any command that might have modified it
  _system_mock_save_state
  
  return $exit_code
}

# ----------------------------
# System Command Implementations
# ----------------------------

# ps command - process status
ps() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "ps" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[ps]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[ps]}"
    case "$error_type" in
      permission_denied) echo "ps: permission denied" >&2; return 1 ;;
      no_processes) echo "ps: no processes found" >&2; return 1 ;;
      *) echo "ps: $error_type" >&2; return 1 ;;
    esac
  fi
  
  # Parse options
  local show_all=false
  local format="standard"
  local specific_pid=""
  
  while [[ $# -gt 0 ]]; do
    case "$1" in
      aux|axu|-ef) show_all=true; format="full"; shift ;;
      -p) specific_pid="$2"; shift 2 ;;
      -p*) specific_pid="${1#-p}"; shift ;;
      *) shift ;;
    esac
  done
  
  # If specific PID requested, show only that process
  if [[ -n "$specific_pid" ]]; then
    if [[ -n "${MOCK_SYSTEM_PROCESSES[$specific_pid]}" ]]; then
      IFS='|' read -r name status ppid command user cpu mem start_time <<< "${MOCK_SYSTEM_PROCESSES[$specific_pid]}"
      if [[ "$format" == "full" ]]; then
        echo "USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND"
        printf "%-10s %5s %4s %4s %6s %5s %-8s %-4s %8s %8s %s\n" \
          "$user" "$specific_pid" "$cpu" "$mem" "1234" "567" "pts/0" "S+" "08:00" "00:01" "$command"
      else
        echo "  PID TTY          TIME CMD"
        printf "%5s %-12s %8s %s\n" "$specific_pid" "pts/0" "00:01" "$name"
      fi
      return 0
    else
      return 1
    fi
  fi
  
  # Show header based on format
  if [[ "$format" == "full" ]]; then
    echo "USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND"
  else
    echo "  PID TTY          TIME CMD"
  fi
  
  # Show processes
  for pid in "${!MOCK_SYSTEM_PROCESSES[@]}"; do
    IFS='|' read -r name status ppid command user cpu mem start_time <<< "${MOCK_SYSTEM_PROCESSES[$pid]}"
    
    # Skip if not showing all and this is a background process
    if [[ "$show_all" != "true" && "$status" != "running" ]]; then
      continue
    fi
    
    if [[ "$format" == "full" ]]; then
      local stat_char="S+"
      [[ "$status" == "running" ]] && stat_char="R+"
      [[ "$status" == "sleeping" ]] && stat_char="S"
      [[ "$status" == "zombie" ]] && stat_char="Z"
      
      printf "%-10s %5s %4s %4s %6s %5s %-8s %-4s %8s %8s %s\n" \
        "$user" "$pid" "$cpu" "$mem" "1234" "567" "pts/0" "$stat_char" "08:00" "00:01" "$command"
    else
      printf "%5s %-12s %8s %s\n" "$pid" "pts/0" "00:01" "$name"
    fi
  done
  
  return 0
}

# pgrep command - find processes by pattern
pgrep() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "pgrep" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[pgrep]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[pgrep]}"
    case "$error_type" in
      no_matches) return 1 ;;
      *) echo "pgrep: $error_type" >&2; return 1 ;;
    esac
  fi
  
  local pattern="$1"
  local found_pids=()
  
  if [[ -z "$pattern" ]]; then
    echo "pgrep: no pattern specified" >&2
    return 1
  fi
  
  # Check pattern cache first (optimization)
  if [[ -n "${MOCK_SYSTEM_PROCESS_PATTERNS[$pattern]}" ]]; then
    IFS=',' read -ra found_pids <<< "${MOCK_SYSTEM_PROCESS_PATTERNS[$pattern]}"
    for pid in "${found_pids[@]}"; do
      echo "$pid"
    done
    return 0
  fi
  
  # Fallback: search through all processes
  for pid in "${!MOCK_SYSTEM_PROCESSES[@]}"; do
    IFS='|' read -r name status ppid command user cpu mem start_time <<< "${MOCK_SYSTEM_PROCESSES[$pid]}"
    
    # Match against process name or command
    if [[ "$name" =~ $pattern ]] || [[ "$command" =~ $pattern ]]; then
      found_pids+=("$pid")
    fi
  done
  
  if [[ ${#found_pids[@]} -eq 0 ]]; then
    return 1
  fi
  
  for pid in "${found_pids[@]}"; do
    echo "$pid"
  done
  
  return 0
}

# pkill command - kill processes by pattern
pkill() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "pkill" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[pkill]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[pkill]}"
    case "$error_type" in
      permission_denied) echo "pkill: permission denied" >&2; return 1 ;;
      no_matches) return 1 ;;
      *) echo "pkill: $error_type" >&2; return 1 ;;
    esac
  fi
  
  local signal="TERM"
  local pattern=""
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -[0-9]*) signal="${1#-}"; shift ;;
      -[A-Z]*) signal="${1#-}"; shift ;;
      -*) shift ;; # ignore other flags
      *) pattern="$1"; shift ;;
    esac
  done
  
  if [[ -z "$pattern" ]]; then
    echo "pkill: no pattern specified" >&2
    return 1
  fi
  
  local killed_count=0
  local pids_to_kill=()
  
  # Find matching processes
  for pid in "${!MOCK_SYSTEM_PROCESSES[@]}"; do
    IFS='|' read -r name status ppid command user cpu mem start_time <<< "${MOCK_SYSTEM_PROCESSES[$pid]}"
    
    if [[ "$name" =~ $pattern ]] || [[ "$command" =~ $pattern ]]; then
      pids_to_kill+=("$pid")
    fi
  done
  
  # Kill matching processes
  for pid in "${pids_to_kill[@]}"; do
    if mock::system::remove_process "$pid"; then
      ((killed_count++))
    fi
  done
  
  if [[ $killed_count -eq 0 ]]; then
    return 1
  fi
  
  return 0
}

# kill command - terminate processes by PID
kill() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "kill" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[kill]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[kill]}"
    case "$error_type" in
      permission_denied) echo "kill: permission denied" >&2; return 1 ;;
      no_such_process) echo "kill: no such process" >&2; return 1 ;;
      *) echo "kill: $error_type" >&2; return 1 ;;
    esac
  fi
  
  local signal="TERM"
  local pids=()
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -[0-9]*) signal="${1#-}"; shift ;;
      -[A-Z]*) signal="${1#-}"; shift ;;
      -*) shift ;; # ignore other flags
      *) pids+=("$1"); shift ;;
    esac
  done
  
  if [[ ${#pids[@]} -eq 0 ]]; then
    echo "kill: no process ID specified" >&2
    return 1
  fi
  
  local failed_count=0
  
  for pid in "${pids[@]}"; do
    # Special handling for kill -0 (test if process exists)
    if [[ "$signal" == "0" ]]; then
      if [[ -n "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
        continue  # Process exists
      else
        ((failed_count++))
        continue
      fi
    fi
    
    # For real PIDs outside mock range, try to kill them for real
    if [[ "$pid" =~ ^[0-9]+$ && "$pid" -lt 10000 ]]; then
      # This is likely a real PID, try to kill it (carefully)
      if builtin kill "-$signal" "$pid" 2>/dev/null; then
        continue
      else
        ((failed_count++))
        continue
      fi
    fi
    
    # Kill mock process
    if mock::system::remove_process "$pid"; then
      continue
    else
      ((failed_count++))
    fi
  done
  
  return $failed_count
}

# which command - locate command
which() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "which" "$*"
  fi
  
  local command="$1"
  
  if [[ -z "$command" ]]; then
    echo "which: no command specified" >&2
    return 1
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[which]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[which]}"
    case "$error_type" in
      not_found) return 1 ;;
      *) echo "which: $error_type" >&2; return 1 ;;
    esac
  fi
  
  # Return configured path or default
  if [[ -n "${MOCK_SYSTEM_COMMANDS[$command]}" ]]; then
    echo "${MOCK_SYSTEM_COMMANDS[$command]}"
    return 0
  fi
  
  # Default paths for common commands
  case "$command" in
    bash|sh|ps|kill|uname|date) echo "/bin/$command" ;;
    systemctl|which|id|whoami) echo "/usr/bin/$command" ;;
    *) return 1 ;;
  esac
  
  return 0
}

# id command - user identity
id() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "id" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[id]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[id]}"
    case "$error_type" in
      no_such_user) echo "id: no such user" >&2; return 1 ;;
      *) echo "id: $error_type" >&2; return 1 ;;
    esac
  fi
  
  local target_user=""
  local format="full"
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -u) format="uid"; shift ;;
      -g) format="gid"; shift ;;
      -un) format="username"; shift ;;
      -gn) format="groupname"; shift ;;
      *) target_user="$1"; shift ;;
    esac
  done
  
  # Default to current user if none specified
  if [[ -z "$target_user" ]]; then
    target_user="${MOCK_SYSTEM_INFO[current_user]:-testuser}"
  fi
  
  # Get user info
  local user_info="${MOCK_SYSTEM_USERS[$target_user]}"
  if [[ -z "$user_info" ]]; then
    # Default for unknown users
    user_info="1000|1000|testuser"
  fi
  
  IFS='|' read -r uid gid groups <<< "$user_info"
  
  case "$format" in
    uid) echo "$uid" ;;
    gid) echo "$gid" ;;
    username) echo "$target_user" ;;
    groupname) echo "${groups%%,*}" ;;
    full) echo "uid=$uid($target_user) gid=$gid(${groups%%,*}) groups=$uid($target_user),$(echo "$groups" | tr ',' ' ')" ;;
  esac
  
  return 0
}

# whoami command - current username
whoami() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "whoami" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[whoami]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[whoami]}"
    echo "whoami: $error_type" >&2
    return 1
  fi
  
  echo "${MOCK_SYSTEM_INFO[current_user]:-testuser}"
  return 0
}

# uname command - system information
uname() {
  # Load state from file for subshell access
  if [[ -n "${SYSTEM_MOCK_STATE_FILE}" && -f "$SYSTEM_MOCK_STATE_FILE" ]]; then
    eval "$(cat "$SYSTEM_MOCK_STATE_FILE")" 2>/dev/null || true
  fi
  
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "uname" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[uname]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[uname]}"
    echo "uname: $error_type" >&2
    return 1
  fi
  
  local format="kernel_name"
  
  case "${1:-}" in
    -a|--all) format="all" ;;
    -s|--kernel-name) format="kernel_name" ;;
    -r|--kernel-release) format="kernel_release" ;;
    -m|--machine) format="machine" ;;
    -n|--nodename) format="nodename" ;;
    -v|--kernel-version) format="kernel_version" ;;
  esac
  
  local os="${MOCK_SYSTEM_INFO[os]:-Linux}"
  local hostname="${MOCK_SYSTEM_INFO[hostname]:-testhost}"
  local kernel="${MOCK_SYSTEM_INFO[kernel]:-5.4.0-test}"
  local arch="${MOCK_SYSTEM_INFO[arch]:-x86_64}"
  local kernel_version="#1 SMP Mon Jan 1 12:00:00 UTC 2024"
  
  case "$format" in
    all) echo "$os $hostname $kernel $kernel_version $arch $arch $arch GNU/Linux" ;;
    kernel_name) echo "$os" ;;
    kernel_release) echo "$kernel" ;;
    machine) echo "$arch" ;;
    nodename) echo "$hostname" ;;
    kernel_version) echo "$kernel_version" ;;
  esac
  
  return 0
}

# date command - current date/time
date() {
  # Use centralized logging
  if command -v mock::log_call &>/dev/null; then
    mock::log_call "date" "$*"
  fi
  
  # Check for injected errors
  if [[ -n "${MOCK_SYSTEM_ERRORS[date]}" ]]; then
    local error_type="${MOCK_SYSTEM_ERRORS[date]}"
    echo "date: $error_type" >&2
    return 1
  fi
  
  # For simplicity, provide consistent mock output
  case "${1:-}" in
    "+%s") echo "1704067200" ;;  # 2024-01-01 00:00:00 UTC
    "+%Y-%m-%d") echo "2024-01-01" ;;
    "+%b %d %H:%M:%S") echo "Jan 01 12:00:00" ;;
    "+%a %Y-%m-%d %H:%M:%S %Z") echo "Mon 2024-01-01 12:00:00 UTC" ;;
    *) echo "Mon Jan  1 12:00:00 UTC 2024" ;;
  esac
  
  return 0
}

# ----------------------------
# SystemCtl Subcommand implementations
# ----------------------------
mock::systemctl::lifecycle() {
  local action="$1"; shift
  local service unit_name service_name
  
  for service in "$@"; do
    service_name="$(_mock_service_name "$service")"
    unit_name="$(_mock_unit_name "$service")"
    
    if command -v mock::log_and_verify &>/dev/null; then
      mock::log_and_verify "systemctl" "$action $service"
    fi
    
    # Check if service exists (if not in our state, assume it exists)
    if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
      # Default to stopped service if not defined
      mock::systemctl::set_service_state "$service_name" "inactive" "enabled" "dead"
    fi
    
    IFS='|' read -r current_state enabled substate main_pid <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
    
    case "$action" in
      start)
        if [[ "$current_state" == "active" ]]; then
          echo "Job for $unit_name finished, started already."
        else
          mock::systemctl::set_service_state "$service_name" "active" "$enabled" "running"
          echo "Job for $unit_name finished, started successfully."
        fi
        ;;
      stop)
        if [[ "$current_state" == "inactive" ]]; then
          echo "Job for $unit_name finished, stopped already."
        else
          mock::systemctl::set_service_state "$service_name" "inactive" "$enabled" "dead"
          echo "Job for $unit_name finished, stopped successfully."
        fi
        ;;
      restart)
        mock::systemctl::set_service_state "$service_name" "active" "$enabled" "running"
        echo "Job for $unit_name finished, restarted successfully."
        ;;
      reload)
        if [[ "$current_state" == "active" ]]; then
          echo "Job for $unit_name finished, reloaded successfully."
        else
          echo "Job for $unit_name failed because the unit is not active." >&2
          return 3
        fi
        ;;
      reload-or-restart)
        mock::systemctl::set_service_state "$service_name" "active" "$enabled" "running"
        echo "Job for $unit_name finished, reloaded or restarted successfully."
        ;;
    esac
  done
  
  return 0
}

mock::systemctl::configuration() {
  local action="$1"; shift
  local service unit_name service_name
  
  for service in "$@"; do
    service_name="$(_mock_service_name "$service")"
    unit_name="$(_mock_unit_name "$service")"
    
    if command -v mock::log_and_verify &>/dev/null; then
      mock::log_and_verify "systemctl" "$action $service"
    fi
    
    # Get current state or create default
    if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
      mock::systemctl::set_service_state "$service_name" "inactive" "disabled" "dead"
    fi
    
    IFS='|' read -r current_state current_enabled substate main_pid <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
    
    case "$action" in
      enable)
        mock::systemctl::set_service_state "$service_name" "$current_state" "enabled" "$substate" "$main_pid"
        echo "Created symlink /etc/systemd/system/multi-user.target.wants/$unit_name → /lib/systemd/system/$unit_name."
        ;;
      disable)
        mock::systemctl::set_service_state "$service_name" "$current_state" "disabled" "$substate" "$main_pid"
        echo "Removed symlink /etc/systemd/system/multi-user.target.wants/$unit_name."
        ;;
      mask)
        mock::systemctl::set_service_state "$service_name" "$current_state" "masked" "$substate" "$main_pid"
        echo "Created symlink /etc/systemd/system/$unit_name → /dev/null."
        ;;
      unmask)
        mock::systemctl::set_service_state "$service_name" "$current_state" "enabled" "$substate" "$main_pid"
        echo "Removed symlink /etc/systemd/system/$unit_name."
        ;;
      preset)
        # Preset enables services that are enabled by default, disables others
        mock::systemctl::set_service_state "$service_name" "$current_state" "enabled" "$substate" "$main_pid"
        echo "Preset for $unit_name: enabled."
        ;;
    esac
  done
  
  return 0
}

mock::systemctl::status() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  local unit_name="$(_mock_unit_name "$service")"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "status $service"
  fi
  
  # Get service state or default to active
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    mock::systemctl::set_service_state "$service_name" "active" "enabled" "running"
  fi
  
  IFS='|' read -r state enabled substate main_pid <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  
  local status_symbol since_time exit_code=0
  case "$state" in
    active)
      status_symbol="●"
      since_time="$(_mock_since_time)"
      ;;
    inactive)
      status_symbol="○"
      since_time="$(_mock_since_time)"
      exit_code=3
      ;;
    failed)
      status_symbol="●"
      since_time="$(_mock_since_time)"
      exit_code=3
      ;;
    *)
      status_symbol="○"
      since_time="$(_mock_since_time)"
      exit_code=3
      ;;
  esac
  
  cat <<EOF
${status_symbol} ${unit_name} - Mock ${service_name} Service
   Loaded: loaded (/lib/systemd/system/${unit_name}; ${enabled}; vendor preset: enabled)
   Active: ${state} (${substate}) since ${since_time}
     Docs: man:${service_name}(8)
 Main PID: ${main_pid} (${service_name})
    Tasks: 1 (limit: 4615)
   Memory: 2.3M
      CPU: 123ms
   CGroup: /system.slice/${unit_name}
           └─${main_pid} /usr/bin/${service_name} --config /etc/${service_name}/${service_name}.conf

$(date '+%b %d %H:%M:%S') $(hostname) ${service_name}[${main_pid}]: Mock ${service_name} service started
$(date '+%b %d %H:%M:%S') $(hostname) ${service_name}[${main_pid}]: Ready to accept connections
$(date '+%b %d %H:%M:%S') $(hostname) systemd[1]: Started Mock ${service_name} Service.
EOF

  return $exit_code
}

mock::systemctl::is_active() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "is-active $service"
  fi
  
  # Get service state or default to active
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    echo "active"
    return 0
  fi
  
  IFS='|' read -r state _ _ _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  
  case "$state" in
    active)
      echo "active"
      return 0
      ;;
    inactive)
      echo "inactive"
      return 3
      ;;
    failed)
      echo "failed"
      return 3
      ;;
    *)
      echo "unknown"
      return 3
      ;;
  esac
}

mock::systemctl::is_enabled() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "is-enabled $service"
  fi
  
  # Get service state or default to enabled
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    echo "enabled"
    return 0
  fi
  
  IFS='|' read -r _ enabled _ _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  
  case "$enabled" in
    enabled)
      echo "enabled"
      return 0
      ;;
    disabled)
      echo "disabled"
      return 1
      ;;
    masked)
      echo "masked"
      return 1
      ;;
    static)
      echo "static"
      return 0
      ;;
    *)
      echo "unknown"
      return 1
      ;;
  esac
}

mock::systemctl::is_failed() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "is-failed $service"
  fi
  
  # Get service state or default to active (not failed)
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    echo "active"
    return 1
  fi
  
  IFS='|' read -r state _ _ _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  
  if [[ "$state" == "failed" ]]; then
    echo "failed"
    return 0
  else
    echo "$state"
    return 1
  fi
}

mock::systemctl::is_system_running() {
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "is-system-running"
  fi
  
  # System is always running in mock
  echo "running"
  return 0
}

mock::systemctl::list_units() {
  local type_filter="" state_filter="" all_units=false
  
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --type=*) type_filter="${1#--type=}"; shift ;;
      -t) type_filter="$2"; shift 2 ;;
      --state=*) state_filter="${1#--state=}"; shift ;;
      --all|-a) all_units=true; shift ;;
      *) shift ;;
    esac
  done
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "list-units${type_filter:+ --type=$type_filter}${state_filter:+ --state=$state_filter}"
  fi
  
  echo "  UNIT                           LOAD   ACTIVE SUB     DESCRIPTION"
  
  # List mock services
  for service_name in "${!MOCK_SYSTEMCTL_SERVICES[@]}"; do
    IFS='|' read -r state enabled substate main_pid <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
    local unit_name="$(_mock_unit_name "$service_name")"
    
    # Apply type filter
    if [[ -n "$type_filter" && "$type_filter" != "service" ]]; then
      continue
    fi
    
    # Apply state filter
    if [[ -n "$state_filter" && "$state" != "$state_filter" ]]; then
      continue
    fi
    
    # Skip inactive units unless --all specified
    if [[ "$all_units" != "true" && "$state" == "inactive" ]]; then
      continue
    fi
    
    printf "  %-30s %-6s %-6s %-7s Mock %s Service\n" \
      "$unit_name" "loaded" "$state" "$substate" "$service_name"
  done
  
  # Add some default system units if no services defined
  if [[ ${#MOCK_SYSTEMCTL_SERVICES[@]} -eq 0 ]]; then
    printf "  %-30s %-6s %-6s %-7s System and Service Manager\n" \
      "systemd.service" "loaded" "active" "running"
    printf "  %-30s %-6s %-6s %-7s OpenSSH Daemon\n" \
      "ssh.service" "loaded" "active" "running"
    printf "  %-30s %-6s %-6s %-7s Network Manager\n" \
      "NetworkManager.service" "loaded" "active" "running"
  fi
  
  echo ""
  echo "LOAD   = Reflects whether the unit definition was properly loaded."
  echo "ACTIVE = The high-level unit activation state, i.e. generalization of SUB."
  echo "SUB    = The low-level unit activation state, values depend on unit type."
  
  local count=${#MOCK_SYSTEMCTL_SERVICES[@]}
  [[ $count -eq 0 ]] && count=3
  echo ""
  echo "$count loaded units listed."
  
  return 0
}

mock::systemctl::list_unit_files() {
  local type_filter="" state_filter=""
  
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --type=*) type_filter="${1#--type=}"; shift ;;
      -t) type_filter="$2"; shift 2 ;;
      --state=*) state_filter="${1#--state=}"; shift ;;
      *) shift ;;
    esac
  done
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "list-unit-files${type_filter:+ --type=$type_filter}${state_filter:+ --state=$state_filter}"
  fi
  
  echo "UNIT FILE                    STATE"
  
  # List mock services
  for service_name in "${!MOCK_SYSTEMCTL_SERVICES[@]}"; do
    IFS='|' read -r _ enabled _ _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
    local unit_name="$(_mock_unit_name "$service_name")"
    
    # Apply type filter
    if [[ -n "$type_filter" && "$type_filter" != "service" ]]; then
      continue
    fi
    
    # Apply state filter
    if [[ -n "$state_filter" && "$enabled" != "$state_filter" ]]; then
      continue
    fi
    
    printf "%-28s %s\n" "$unit_name" "$enabled"
  done
  
  # Add some default system units if no services defined
  if [[ ${#MOCK_SYSTEMCTL_SERVICES[@]} -eq 0 ]]; then
    printf "%-28s %s\n" "ssh.service" "enabled"
    printf "%-28s %s\n" "NetworkManager.service" "enabled"
    printf "%-28s %s\n" "systemd-resolved.service" "enabled"
  fi
  
  local count=${#MOCK_SYSTEMCTL_SERVICES[@]}
  [[ $count -eq 0 ]] && count=3
  echo ""
  echo "$count unit files listed."
  
  return 0
}

mock::systemctl::show() {
  local service=""
  local property=""
  
  # Parse arguments to handle --property=VALUE format
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --property=*)
        property="${1#--property=}"
        shift
        ;;
      --property)
        property="$2"
        shift 2
        ;;
      -p)
        property="$2"
        shift 2
        ;;
      *)
        if [[ -z "$service" ]]; then
          service="$1"
        fi
        shift
        ;;
    esac
  done
  
  local service_name="$(_mock_service_name "$service")"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "show $service${property:+ --property=$property}"
  fi
  
  # Get service state or create default
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    mock::systemctl::set_service_state "$service_name" "active" "enabled" "running"
  fi
  
  IFS='|' read -r state enabled substate main_pid <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  
  # If property specified, show only that property
  if [[ -n "$property" ]]; then
    case "$property" in
      ActiveState) echo "ActiveState=$state" ;;
      UnitFileState) echo "UnitFileState=$enabled" ;;
      SubState) echo "SubState=$substate" ;;
      MainPID) echo "MainPID=$main_pid" ;;
      LoadState) echo "LoadState=loaded" ;;
      Id) echo "Id=$(_mock_unit_name "$service")" ;;
      *) echo "$property=mock-value" ;;
    esac
    return 0
  fi
  
  # Show all properties
  cat <<EOF
Id=$(_mock_unit_name "$service")
LoadState=loaded
ActiveState=$state
SubState=$substate
UnitFileState=$enabled
MainPID=$main_pid
ExecStart={ path=/usr/bin/$service_name ; argv[]=/usr/bin/$service_name --config /etc/$service_name/$service_name.conf ; ignore_errors=no ; start_time=[n/a] ; stop_time=[n/a] ; pid=0 ; code=(null) ; status=0/0 }
Type=simple
Restart=always
TimeoutStartUSec=1min 30s
TimeoutStopUSec=1min 30s
RuntimeMaxUSec=infinity
WatchdogUSec=0
KillMode=control-group
KillSignal=15
RestartKillSignal=15
FinalKillSignal=9
SendSIGKILL=yes
SendSIGHUP=no
UMask=0022
LimitCORE=infinity
LimitNOFILE=524288
StandardInput=null
StandardOutput=journal
StandardError=inherit
DynamicUser=no
RemainAfterExit=no
GuessMainPID=yes
JobRunningTimeoutUSec=infinity
JobTimeoutUSec=infinity
StatusText=Mock $service_name service
Description=Mock $service_name Service
SourcePath=/lib/systemd/system/$(_mock_unit_name "$service")
EOF
  
  return 0
}

mock::systemctl::cat() {
  local service="$1"
  local unit_name="$(_mock_unit_name "$service")"
  local service_name="$(_mock_service_name "$service")"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "cat $service"
  fi
  
  echo "# /lib/systemd/system/$unit_name"
  cat <<EOF
[Unit]
Description=Mock $service_name Service
Documentation=man:$service_name(8)
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/$service_name --config /etc/$service_name/$service_name.conf
Restart=always
RestartSec=5
TimeoutStartSec=infinity
TimeoutStopSec=30
User=$service_name
Group=$service_name
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
  
  return 0
}

mock::systemctl::daemon_control() {
  local action="$1"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "$action"
  fi
  
  case "$action" in
    daemon-reload)
      # Simulate reload delay
      sleep 0.1 2>/dev/null || true
      # No output for daemon-reload on success
      ;;
    daemon-reexec)
      echo "Reexecuting systemd..."
      ;;
  esac
  
  return 0
}

mock::systemctl::system_control() {
  local action="$1"
  
  if command -v mock::log_and_verify &>/dev/null; then
    mock::log_and_verify "systemctl" "$action"
  fi
  
  case "$action" in
    default)
      echo "Switching to default target."
      ;;
    rescue)
      echo "Switching to rescue mode."
      ;;
    emergency)
      echo "Switching to emergency mode."
      ;;
    halt|poweroff|reboot)
      echo "Mock $action initiated - system would $action now."
      ;;
  esac
  
  return 0
}

# ----------------------------
# Test Helper Functions
# ----------------------------
# Scenario builders for common test patterns
mock::systemctl::scenario::create_running_services() {
  local prefix="${1:-test}"
  local count="${2:-3}"
  
  for ((i=1; i<=count; i++)); do
    local service_name="${prefix}_service_$i"
    mock::systemctl::set_service_state "$service_name" "active" "enabled" "running"
  done
  
  echo "[MOCK] Created $count running services with prefix '$prefix'"
}

mock::systemctl::scenario::create_mixed_services() {
  local prefix="${1:-test}"
  
  # Create services in different states
  mock::systemctl::set_service_state "${prefix}_web" "active" "enabled" "running"
  mock::systemctl::set_service_state "${prefix}_db" "active" "enabled" "running"
  mock::systemctl::set_service_state "${prefix}_cache" "inactive" "disabled" "dead"
  mock::systemctl::set_service_state "${prefix}_backup" "failed" "enabled" "failed"
  mock::systemctl::set_service_state "${prefix}_monitor" "active" "masked" "running"
  
  echo "[MOCK] Created mixed service states with prefix '$prefix'"
}

# Assertion helpers
mock::systemctl::assert::service_active() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  local state="${MOCK_SYSTEMCTL_SERVICES[$service_name]%%|*}"
  
  if [[ "$state" != "active" ]]; then
    echo "ASSERTION FAILED: Service '$service' is not active (state: ${state:-not found})" >&2
    return 1
  fi
  return 0
}

mock::systemctl::assert::service_inactive() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  local state="${MOCK_SYSTEMCTL_SERVICES[$service_name]%%|*}"
  
  if [[ "$state" != "inactive" ]]; then
    echo "ASSERTION FAILED: Service '$service' is not inactive (state: ${state:-not found})" >&2
    return 1
  fi
  return 0
}

mock::systemctl::assert::service_enabled() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    echo "ASSERTION FAILED: Service '$service' does not exist" >&2
    return 1
  fi
  
  IFS='|' read -r _ enabled _ _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  
  if [[ "$enabled" != "enabled" ]]; then
    echo "ASSERTION FAILED: Service '$service' is not enabled (state: $enabled)" >&2
    return 1
  fi
  return 0
}

mock::systemctl::assert::service_disabled() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    echo "ASSERTION FAILED: Service '$service' does not exist" >&2
    return 1
  fi
  
  IFS='|' read -r _ enabled _ _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  
  if [[ "$enabled" != "disabled" ]]; then
    echo "ASSERTION FAILED: Service '$service' is not disabled (state: $enabled)" >&2
    return 1
  fi
  return 0
}

mock::systemctl::assert::service_failed() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  local state="${MOCK_SYSTEMCTL_SERVICES[$service_name]%%|*}"
  
  if [[ "$state" != "failed" ]]; then
    echo "ASSERTION FAILED: Service '$service' is not failed (state: ${state:-not found})" >&2
    return 1
  fi
  return 0
}

mock::systemctl::assert::service_exists() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  
  if [[ -z "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    echo "ASSERTION FAILED: Service '$service' does not exist" >&2
    return 1
  fi
  return 0
}

mock::systemctl::assert::service_not_exists() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  
  if [[ -n "${MOCK_SYSTEMCTL_SERVICES[$service_name]}" ]]; then
    echo "ASSERTION FAILED: Service '$service' exists but should not" >&2
    return 1
  fi
  return 0
}

# Get service info helpers
mock::systemctl::get::service_state() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  echo "${MOCK_SYSTEMCTL_SERVICES[$service_name]%%|*}"
}

mock::systemctl::get::service_enabled() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  IFS='|' read -r _ enabled _ _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  echo "$enabled"
}

mock::systemctl::get::service_substate() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  IFS='|' read -r _ _ substate _ <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  echo "$substate"
}

mock::systemctl::get::service_pid() {
  local service="$1"
  local service_name="$(_mock_service_name "$service")"
  IFS='|' read -r _ _ _ main_pid <<<"${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  echo "$main_pid"
}

# Debug helper
mock::systemctl::debug::dump_state() {
  echo "=== SystemCtl Mock State Dump ==="
  echo "Services:"
  for service_name in "${!MOCK_SYSTEMCTL_SERVICES[@]}"; do
    echo "  $service_name: ${MOCK_SYSTEMCTL_SERVICES[$service_name]}"
  done
  echo "Units:"
  for unit_name in "${!MOCK_SYSTEMCTL_UNITS[@]}"; do
    echo "  $unit_name: ${MOCK_SYSTEMCTL_UNITS[$unit_name]}"
  done
  echo "Targets:"
  for target_name in "${!MOCK_SYSTEMCTL_TARGETS[@]}"; do
    echo "  $target_name: ${MOCK_SYSTEMCTL_TARGETS[$target_name]}"
  done
  echo "Errors:"
  for cmd in "${!MOCK_SYSTEMCTL_ERRORS[@]}"; do
    echo "  $cmd: ${MOCK_SYSTEMCTL_ERRORS[$cmd]}"
  done
  echo "=========================="
}

# ----------------------------
# System Command Test Helpers
# ----------------------------

# Scenario builders for process testing
mock::system::scenario::create_processes() {
  local prefix="${1:-test}"
  local count="${2:-3}"
  
  for ((i=1; i<=count; i++)); do
    local pid="$(_mock_system_pid)"
    local name="${prefix}_process_$i"
    mock::system::set_process_state "$pid" "$name" "running" "1" "/usr/bin/$name" "testuser" "0.1" "0.2"
  done
  
  echo "[MOCK] Created $count processes with prefix '$prefix'"
}

mock::system::scenario::create_mixed_processes() {
  local prefix="${1:-test}"
  
  # Create processes in different states
  local pid1="$(_mock_system_pid)"
  local pid2="$(_mock_system_pid)"
  local pid3="$(_mock_system_pid)"
  local pid4="$(_mock_system_pid)"
  
  mock::system::set_process_state "$pid1" "${prefix}_web" "running" "1" "/usr/bin/${prefix}_web" "www-data" "2.1" "5.3"
  mock::system::set_process_state "$pid2" "${prefix}_db" "sleeping" "1" "/usr/bin/${prefix}_db" "mysql" "0.8" "15.2"
  mock::system::set_process_state "$pid3" "${prefix}_worker" "running" "$pid1" "/usr/bin/${prefix}_worker" "testuser" "1.2" "2.1"
  mock::system::set_process_state "$pid4" "${prefix}_zombie" "zombie" "1" "[${prefix}_zombie]" "testuser" "0.0" "0.0"
  
  echo "[MOCK] Created mixed process states with prefix '$prefix'"
}

# Assertion helpers for system commands
mock::system::assert::process_exists() {
  local pid="$1"
  
  if [[ -z "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
    echo "ASSERTION FAILED: Process '$pid' does not exist" >&2
    return 1
  fi
  return 0
}

mock::system::assert::process_not_exists() {
  local pid="$1"
  
  if [[ -n "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
    echo "ASSERTION FAILED: Process '$pid' exists but should not" >&2
    return 1
  fi
  return 0
}

mock::system::assert::process_running() {
  local pid="$1"
  
  if [[ -z "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
    echo "ASSERTION FAILED: Process '$pid' does not exist" >&2
    return 1
  fi
  
  local status="${MOCK_SYSTEM_PROCESSES[$pid]}"
  status="${status#*|}"; status="${status%%|*}"  # Extract status field
  
  if [[ "$status" != "running" ]]; then
    echo "ASSERTION FAILED: Process '$pid' is not running (status: $status)" >&2
    return 1
  fi
  return 0
}

mock::system::assert::command_exists() {
  local command="$1"
  
  if [[ -z "${MOCK_SYSTEM_COMMANDS[$command]}" ]]; then
    # Check default paths
    case "$command" in
      bash|sh|ps|kill|uname|date) return 0 ;;  # These have defaults
      systemctl|which|id|whoami) return 0 ;;   # These have defaults
      *) 
        echo "ASSERTION FAILED: Command '$command' does not exist" >&2
        return 1
        ;;
    esac
  fi
  return 0
}

# Get helpers for system commands
mock::system::get::process_name() {
  local pid="$1"
  if [[ -n "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
    echo "${MOCK_SYSTEM_PROCESSES[$pid]%%|*}"
  fi
}

mock::system::get::process_status() {
  local pid="$1"
  if [[ -n "${MOCK_SYSTEM_PROCESSES[$pid]}" ]]; then
    local status="${MOCK_SYSTEM_PROCESSES[$pid]}"
    status="${status#*|}"; echo "${status%%|*}"
  fi
}

mock::system::get::process_count() {
  echo "${#MOCK_SYSTEM_PROCESSES[@]}"
}

mock::system::get::user_info() {
  local username="$1"
  echo "${MOCK_SYSTEM_USERS[$username]}"
}

# Debug helper for system state
mock::system::debug::dump_state() {
  echo "=== System Mock State Dump ==="
  echo "Processes:"
  for pid in "${!MOCK_SYSTEM_PROCESSES[@]}"; do
    echo "  $pid: ${MOCK_SYSTEM_PROCESSES[$pid]}"
  done
  echo "Process Patterns:"
  for pattern in "${!MOCK_SYSTEM_PROCESS_PATTERNS[@]}"; do
    echo "  $pattern: ${MOCK_SYSTEM_PROCESS_PATTERNS[$pattern]}"
  done
  echo "System Info:"
  for key in "${!MOCK_SYSTEM_INFO[@]}"; do
    echo "  $key: ${MOCK_SYSTEM_INFO[$key]}"
  done
  echo "Users:"
  for user in "${!MOCK_SYSTEM_USERS[@]}"; do
    echo "  $user: ${MOCK_SYSTEM_USERS[$user]}"
  done
  echo "Commands:"
  for cmd in "${!MOCK_SYSTEM_COMMANDS[@]}"; do
    echo "  $cmd: ${MOCK_SYSTEM_COMMANDS[$cmd]}"
  done
  echo "System Errors:"
  for cmd in "${!MOCK_SYSTEM_ERRORS[@]}"; do
    echo "  $cmd: ${MOCK_SYSTEM_ERRORS[$cmd]}"
  done
  echo "========================"
}

# ----------------------------
# Export functions into subshells
# ----------------------------
# Exporting lets child bash processes (spawned by scripts under test) inherit mocks.

# Core system commands
export -f systemctl ps pgrep pkill kill which id whoami uname date

# Utility functions
export -f _mock_service_name _mock_unit_name _mock_systemd_time _mock_pid _mock_since_time
export -f _mock_system_pid _mock_system_timestamp
export -f _system_mock_init_state_file _system_mock_save_state _system_mock_load_state

# System state management
export -f mock::system::reset mock::system::init_defaults mock::system::inject_error
export -f mock::system::set_process_state mock::system::remove_process
export -f mock::system::set_info mock::system::set_user mock::system::set_command
export -f mock::system::_update_pattern_cache

# SystemCtl functions
export -f mock::systemctl::reset mock::systemctl::enable_auto_cleanup mock::systemctl::inject_error
export -f mock::systemctl::set_service_state mock::systemctl::set_unit_state mock::systemctl::set_target_state
export -f mock::systemctl::lifecycle mock::systemctl::configuration mock::systemctl::status
export -f mock::systemctl::is_active mock::systemctl::is_enabled mock::systemctl::is_failed mock::systemctl::is_system_running
export -f mock::systemctl::list_units mock::systemctl::list_unit_files mock::systemctl::show mock::systemctl::cat
export -f mock::systemctl::daemon_control mock::systemctl::system_control

# SystemCtl test helpers
export -f mock::systemctl::scenario::create_running_services mock::systemctl::scenario::create_mixed_services
export -f mock::systemctl::assert::service_active mock::systemctl::assert::service_inactive
export -f mock::systemctl::assert::service_enabled mock::systemctl::assert::service_disabled
export -f mock::systemctl::assert::service_failed mock::systemctl::assert::service_exists mock::systemctl::assert::service_not_exists
export -f mock::systemctl::get::service_state mock::systemctl::get::service_enabled
export -f mock::systemctl::get::service_substate mock::systemctl::get::service_pid
export -f mock::systemctl::debug::dump_state

# System command test helpers
export -f mock::system::scenario::create_processes mock::system::scenario::create_mixed_processes
export -f mock::system::assert::process_exists mock::system::assert::process_not_exists
export -f mock::system::assert::process_running mock::system::assert::command_exists
export -f mock::system::get::process_name mock::system::get::process_status
export -f mock::system::get::process_count mock::system::get::user_info
export -f mock::system::debug::dump_state

# Note: Defaults will be initialized on first mock::system::reset call
# This avoids circular dependencies during initial load

echo "[MOCK] System command mocks loaded successfully (systemctl, ps, kill, which, id, whoami, uname, date)"