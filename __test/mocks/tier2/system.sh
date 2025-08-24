#!/usr/bin/env bash
# System Mock - Tier 2 (Stateful)
# 
# Provides stateful system command mocking for testing:
# - SystemCtl service management (start, stop, status, enable, disable)
# - Process management (ps, pgrep, pkill, kill)
# - System information (uname, whoami, id, which)
# - Basic command existence and path resolution
# - Error injection for resilience testing
#
# Coverage: ~80% of common system operations in 800 lines

# === Configuration ===
declare -gA SYSTEM_SERVICES=()         # Service management: name -> "state|enabled|pid"
declare -gA SYSTEM_PROCESSES=()        # Process tracking: pid -> "name|status|user|command"  
declare -gA SYSTEM_COMMANDS=()         # Command paths: command -> "/path/to/command"
declare -gA SYSTEM_USERS=()            # User info: username -> "uid|gid|groups"
declare -gA SYSTEM_INFO=(              # System information
    [hostname]="testhost"
    [kernel]="5.4.0-test"
    [arch]="x86_64"
    [os]="Linux"
    [current_user]="testuser"
    [system_running]="running"
)
declare -gA SYSTEM_CONFIG=(            # Mock configuration
    [mode]="normal"
    [error_mode]=""
)

# Debug mode
declare -g SYSTEM_DEBUG="${SYSTEM_DEBUG:-}"

# === Helper Functions ===
system_debug() {
    [[ -n "$SYSTEM_DEBUG" ]] && echo "[MOCK:SYSTEM] $*" >&2
}

system_check_error() {
    case "${SYSTEM_CONFIG[error_mode]}" in
        "permission_denied")
            echo "Permission denied" >&2
            return 1
            ;;
        "command_not_found")
            echo "Command not found" >&2
            return 127
            ;;
        "system_failure")
            echo "System error" >&2
            return 1
            ;;
    esac
    return 0
}

system_mock_pid() {
    echo $((10000 + RANDOM % 50000))
}

system_mock_timestamp() {
    date '+%s' 2>/dev/null || echo '1704067200'
}

# === SystemCtl Mock ===
systemctl() {
    system_debug "systemctl called with: $*"
    
    # Check for error injection
    if ! system_check_error; then
        return $?
    fi
    
    if [[ $# -eq 0 ]]; then
        echo "systemctl: missing command" >&2
        return 1
    fi
    
    local cmd="$1"
    shift
    
    case "$cmd" in
        start)
            system_cmd_start "$@"
            ;;
        stop) 
            system_cmd_stop "$@"
            ;;
        restart)
            system_cmd_restart "$@"
            ;;
        status)
            system_cmd_status "$@"
            ;;
        enable)
            system_cmd_enable "$@"
            ;;
        disable)
            system_cmd_disable "$@"
            ;;
        is-active)
            system_cmd_is_active "$@"
            ;;
        is-enabled)
            system_cmd_is_enabled "$@"
            ;;
        list-units)
            system_cmd_list_units "$@"
            ;;
        --version)
            echo "systemd 247 (247.3-7ubuntu4)"
            ;;
        *)
            echo "systemctl: Unknown command '$cmd'" >&2
            return 1
            ;;
    esac
}

# SystemCtl command implementations
system_cmd_start() {
    local service="$1"
    if [[ -z "$service" ]]; then
        echo "systemctl: missing service name" >&2
        return 1
    fi
    
    # Remove .service suffix if present
    service="${service%.service}"
    
    local pid=$(system_mock_pid)
    SYSTEM_SERVICES[$service]="active|enabled|$pid"
    
    # Add corresponding process
    SYSTEM_PROCESSES[$pid]="$service|running|root|/usr/lib/systemd/$service"
    
    system_debug "Started service: $service (PID $pid)"
}

system_cmd_stop() {
    local service="$1"
    if [[ -z "$service" ]]; then
        echo "systemctl: missing service name" >&2
        return 1
    fi
    
    service="${service%.service}"
    
    if [[ -n "${SYSTEM_SERVICES[$service]}" ]]; then
        local service_data="${SYSTEM_SERVICES[$service]}"
        IFS='|' read -r state enabled pid <<< "$service_data"
        
        SYSTEM_SERVICES[$service]="inactive|$enabled|"
        
        # Remove corresponding process if exists
        if [[ -n "$pid" ]]; then
            unset SYSTEM_PROCESSES[$pid]
        fi
        
        system_debug "Stopped service: $service"
    else
        echo "Failed to stop $service.service: Unit $service.service not loaded." >&2
        return 5
    fi
}

system_cmd_restart() {
    local service="$1"
    system_cmd_stop "$service" 2>/dev/null
    system_cmd_start "$service"
}

system_cmd_status() {
    local service="$1"
    if [[ -z "$service" ]]; then
        echo "systemctl: missing service name" >&2
        return 1
    fi
    
    service="${service%.service}"
    
    local service_data="${SYSTEM_SERVICES[$service]}"
    if [[ -z "$service_data" ]]; then
        echo "Unit $service.service could not be found." >&2
        return 4
    fi
    
    IFS='|' read -r state enabled pid <<< "$service_data"
    
    local status_color=""
    local status_symbol=""
    case "$state" in
        active)
            status_color="32"  # green
            status_symbol="●"
            ;;
        inactive)
            status_color="37"  # white
            status_symbol="○"
            ;;
        failed)
            status_color="31"  # red
            status_symbol="×"
            ;;
    esac
    
    echo "● $service.service - Mock $service service"
    echo "   Loaded: loaded (/usr/lib/systemd/system/$service.service; $enabled; vendor preset: enabled)"
    echo "   Active: \033[0;${status_color}m$status_symbol $state\033[0m since $(date)"
    if [[ -n "$pid" && "$state" == "active" ]]; then
        echo " Main PID: $pid ($service)"
    fi
    echo "   CGroup: /system.slice/$service.service"
    if [[ -n "$pid" ]]; then
        echo "           └─$pid $service"
    fi
    
    # Return appropriate exit code
    case "$state" in
        active) return 0 ;;
        inactive) return 3 ;;
        failed) return 3 ;;
        *) return 1 ;;
    esac
}

system_cmd_enable() {
    local service="$1"
    service="${service%.service}"
    
    local service_data="${SYSTEM_SERVICES[$service]:-inactive|disabled|}"
    IFS='|' read -r state enabled pid <<< "$service_data"
    
    SYSTEM_SERVICES[$service]="$state|enabled|$pid"
    echo "Created symlink /etc/systemd/system/multi-user.target.wants/$service.service → /usr/lib/systemd/system/$service.service."
    system_debug "Enabled service: $service"
}

system_cmd_disable() {
    local service="$1"  
    service="${service%.service}"
    
    local service_data="${SYSTEM_SERVICES[$service]:-inactive|enabled|}"
    IFS='|' read -r state enabled pid <<< "$service_data"
    
    SYSTEM_SERVICES[$service]="$state|disabled|$pid"
    echo "Removed /etc/systemd/system/multi-user.target.wants/$service.service."
    system_debug "Disabled service: $service"
}

system_cmd_is_active() {
    local service="$1"
    service="${service%.service}"
    
    local service_data="${SYSTEM_SERVICES[$service]}"
    if [[ -z "$service_data" ]]; then
        echo "inactive"
        return 1
    fi
    
    local state="${service_data%%|*}"
    echo "$state"
    [[ "$state" == "active" ]]
}

system_cmd_is_enabled() {
    local service="$1"
    service="${service%.service}"
    
    local service_data="${SYSTEM_SERVICES[$service]}"
    if [[ -z "$service_data" ]]; then
        echo "disabled"
        return 1
    fi
    
    IFS='|' read -r state enabled pid <<< "$service_data"
    echo "$enabled"
    [[ "$enabled" == "enabled" ]]
}

system_cmd_list_units() {
    echo "UNIT                          LOAD   ACTIVE SUB       DESCRIPTION"
    for service in "${!SYSTEM_SERVICES[@]}"; do
        local service_data="${SYSTEM_SERVICES[$service]}"
        IFS='|' read -r state enabled pid <<< "$service_data"
        printf "%-28s  loaded %-6s %-9s Mock %s service\n" \
            "$service.service" "$state" "running" "$service"
    done
}

# === Process Management Mocks ===
ps() {
    system_debug "ps called with: $*"
    
    if ! system_check_error; then
        return $?
    fi
    
    # Parse basic options
    local show_all=false format="default"
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -a|aux|ax) show_all=true; shift ;;
            -o) format="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    # Header
    case "$format" in
        "pid,ppid,cmd")
            echo "PID  PPID CMD"
            ;;
        "pid,user,cmd")
            echo "PID  USER     CMD"
            ;;
        *)
            echo "  PID TTY          TIME CMD"
            ;;
    esac
    
    # Show processes
    for pid in "${!SYSTEM_PROCESSES[@]}"; do
        local process_data="${SYSTEM_PROCESSES[$pid]}"
        IFS='|' read -r name status user command <<< "$process_data"
        
        case "$format" in
            "pid,ppid,cmd")
                printf "%5s %4s %s\n" "$pid" "1" "$command"
                ;;
            "pid,user,cmd")
                printf "%5s %-8s %s\n" "$pid" "$user" "$command"
                ;;
            *)
                printf "%5s pts/0    00:00:00 %s\n" "$pid" "$name"
                ;;
        esac
    done
}

pgrep() {
    system_debug "pgrep called with: $*"
    
    if ! system_check_error; then
        return $?
    fi
    
    local pattern="$1"
    if [[ -z "$pattern" ]]; then
        echo "pgrep: no pattern specified" >&2
        return 1
    fi
    
    local found=false
    for pid in "${!SYSTEM_PROCESSES[@]}"; do
        local process_data="${SYSTEM_PROCESSES[$pid]}"
        local name="${process_data%%|*}"
        
        if [[ "$name" == *"$pattern"* ]]; then
            echo "$pid"
            found=true
        fi
    done
    
    if [[ "$found" != "true" ]]; then
        return 1
    fi
}

pkill() {
    system_debug "pkill called with: $*"
    
    if ! system_check_error; then
        return $?
    fi
    
    local pattern="$1"
    if [[ -z "$pattern" ]]; then
        echo "pkill: no pattern specified" >&2
        return 1
    fi
    
    local killed=false
    for pid in "${!SYSTEM_PROCESSES[@]}"; do
        local process_data="${SYSTEM_PROCESSES[$pid]}"
        local name="${process_data%%|*}"
        
        if [[ "$name" == *"$pattern"* ]]; then
            unset SYSTEM_PROCESSES[$pid]
            killed=true
            system_debug "Killed process: $name (PID $pid)"
        fi
    done
    
    if [[ "$killed" != "true" ]]; then
        return 1
    fi
}

kill() {
    system_debug "kill called with: $*"
    
    if ! system_check_error; then
        return $?
    fi
    
    local signal="-TERM"
    local pids=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -[0-9]*|-SIG*)
                signal="$1"
                shift
                ;;
            [0-9]*)
                pids+=("$1")
                shift
                ;;
            *)
                echo "kill: invalid argument '$1'" >&2
                return 1
                ;;
        esac
    done
    
    if [[ ${#pids[@]} -eq 0 ]]; then
        echo "kill: usage: kill [-s sigspec | -signum] [pid ...]" >&2
        return 1
    fi
    
    for pid in "${pids[@]}"; do
        if [[ -n "${SYSTEM_PROCESSES[$pid]}" ]]; then
            unset SYSTEM_PROCESSES[$pid]
            system_debug "Killed process PID $pid with signal $signal"
        else
            echo "kill: ($pid) - No such process" >&2
            return 1
        fi
    done
}

# === System Information Mocks ===
uname() {
    system_debug "uname called with: $*"
    
    if ! system_check_error; then
        return $?
    fi
    
    local option="${1:--s}"
    
    case "$option" in
        -s|--kernel-name) echo "${SYSTEM_INFO[os]}" ;;
        -r|--kernel-release) echo "${SYSTEM_INFO[kernel]}" ;;
        -m|--machine) echo "${SYSTEM_INFO[arch]}" ;;
        -a|--all) echo "${SYSTEM_INFO[os]} ${SYSTEM_INFO[hostname]} ${SYSTEM_INFO[kernel]} #1 SMP $(date) ${SYSTEM_INFO[arch]} ${SYSTEM_INFO[arch]} ${SYSTEM_INFO[arch]} GNU/Linux" ;;
        *) echo "${SYSTEM_INFO[os]}" ;;
    esac
}

whoami() {
    system_debug "whoami called"
    
    if ! system_check_error; then
        return $?
    fi
    
    echo "${SYSTEM_INFO[current_user]}"
}

id() {
    system_debug "id called with: $*"
    
    if ! system_check_error; then
        return $?
    fi
    
    local user="${1:-${SYSTEM_INFO[current_user]}}"
    local user_data="${SYSTEM_USERS[$user]}"
    
    if [[ -z "$user_data" ]]; then
        echo "id: '$user': no such user" >&2
        return 1
    fi
    
    IFS='|' read -r uid gid groups <<< "$user_data"
    echo "uid=$uid($user) gid=$gid($user) groups=$gid($user),$groups"
}

which() {
    system_debug "which called with: $*"
    
    if ! system_check_error; then
        return $?
    fi
    
    local command="$1"
    if [[ -z "$command" ]]; then
        return 1
    fi
    
    local path="${SYSTEM_COMMANDS[$command]}"
    if [[ -n "$path" ]]; then
        echo "$path"
        return 0
    else
        return 1
    fi
}

# === Convention-based Test Functions ===
test_system_connection() {
    system_debug "Testing connection..."
    
    # Test basic system commands
    local result
    result=$(uname 2>&1)
    
    if [[ "$result" =~ "Linux" ]]; then
        system_debug "Connection test passed"
        return 0
    else
        system_debug "Connection test failed: $result"
        return 1
    fi
}

test_system_health() {
    system_debug "Testing health..."
    
    # Test connection
    test_system_connection || return 1
    
    # Test systemctl operations
    systemctl start test-service >/dev/null 2>&1 || return 1
    systemctl status test-service >/dev/null 2>&1 || return 1
    systemctl stop test-service >/dev/null 2>&1 || return 1
    
    # Test process operations
    local pid=$(system_mock_pid)
    SYSTEM_PROCESSES[$pid]="test-process|running|testuser|/usr/bin/test"
    pgrep test-process >/dev/null 2>&1 || return 1
    kill $pid >/dev/null 2>&1 || return 1
    
    system_debug "Health test passed"
    return 0
}

test_system_basic() {
    system_debug "Testing basic operations..."
    
    # Test system info commands
    uname >/dev/null 2>&1 || return 1
    whoami >/dev/null 2>&1 || return 1
    id >/dev/null 2>&1 || return 1
    which bash >/dev/null 2>&1 || return 1
    
    # Test service lifecycle
    systemctl start basic-test >/dev/null 2>&1 || return 1
    systemctl is-active basic-test >/dev/null 2>&1 || return 1
    systemctl enable basic-test >/dev/null 2>&1 || return 1
    systemctl is-enabled basic-test >/dev/null 2>&1 || return 1
    systemctl stop basic-test >/dev/null 2>&1 || return 1
    
    # Test process management
    local pid=$(system_mock_pid)
    SYSTEM_PROCESSES[$pid]="basic-test|running|testuser|/usr/bin/basic"
    ps | grep -q "basic-test" || return 1
    kill $pid >/dev/null 2>&1 || return 1
    
    system_debug "Basic test passed"
    return 0
}

# === State Management ===
system_mock_reset() {
    system_debug "Resetting mock state (called from: ${BASH_SOURCE[1]:-unknown}:${BASH_LINENO[0]:-unknown})"
    
    SYSTEM_SERVICES=()
    SYSTEM_PROCESSES=()
    SYSTEM_CONFIG[error_mode]=""
    SYSTEM_CONFIG[mode]="normal"
    
    # Reinitialize defaults
    system_mock_init_defaults
}

system_mock_init_defaults() {
    # Default users
    SYSTEM_USERS["root"]="0|0|root"
    SYSTEM_USERS["testuser"]="1000|1000|testuser,adm,sudo,docker"
    
    # Default commands
    SYSTEM_COMMANDS["bash"]="/bin/bash"
    SYSTEM_COMMANDS["sh"]="/bin/sh"
    SYSTEM_COMMANDS["systemctl"]="/usr/bin/systemctl"
    SYSTEM_COMMANDS["ps"]="/bin/ps"
    SYSTEM_COMMANDS["kill"]="/bin/kill"
    SYSTEM_COMMANDS["which"]="/usr/bin/which"
    SYSTEM_COMMANDS["id"]="/usr/bin/id"
    SYSTEM_COMMANDS["whoami"]="/usr/bin/whoami"
    SYSTEM_COMMANDS["uname"]="/bin/uname"
    
    # Default processes
    SYSTEM_PROCESSES["1"]="systemd|running|root|/sbin/init"
    SYSTEM_PROCESSES["2"]="kthreadd|sleeping|root|[kthreadd]"
    
    # Default services
    SYSTEM_SERVICES["ssh"]="active|enabled|$(system_mock_pid)"
    SYSTEM_SERVICES["nginx"]="inactive|enabled|"
}

system_mock_set_error() {
    SYSTEM_CONFIG[error_mode]="$1"
    system_debug "Set error mode: $1"
}

system_mock_set_mode() {
    SYSTEM_CONFIG[mode]="$1"
    system_debug "Set mode: $1"
}

system_mock_dump_state() {
    echo "=== System Mock State ==="
    echo "Mode: ${SYSTEM_CONFIG[mode]}"
    echo "Services: ${#SYSTEM_SERVICES[@]}"
    for service in "${!SYSTEM_SERVICES[@]}"; do
        echo "  $service: ${SYSTEM_SERVICES[$service]}"
    done
    echo "Processes: ${#SYSTEM_PROCESSES[@]}"
    for pid in "${!SYSTEM_PROCESSES[@]}"; do
        echo "  $pid: ${SYSTEM_PROCESSES[$pid]}"
    done
    echo "Users: ${#SYSTEM_USERS[@]}"
    for user in "${!SYSTEM_USERS[@]}"; do
        echo "  $user: ${SYSTEM_USERS[$user]}"
    done
    echo "Error Mode: ${SYSTEM_CONFIG[error_mode]:-none}"
    echo "=================="
}

system_mock_create_service() {
    local name="${1:-test_service}"
    local state="${2:-active}"
    local enabled="${3:-enabled}"
    
    local pid=""
    if [[ "$state" == "active" ]]; then
        pid=$(system_mock_pid)
        SYSTEM_PROCESSES[$pid]="$name|running|root|/usr/lib/systemd/$name"
    fi
    
    SYSTEM_SERVICES[$name]="$state|$enabled|$pid"
    system_debug "Created service: $name ($state, $enabled)"
    echo "$name"
}

system_mock_create_process() {
    local name="${1:-test_process}"
    local user="${2:-testuser}"
    local command="${3:-/usr/bin/$name}"
    
    local pid=$(system_mock_pid)
    SYSTEM_PROCESSES[$pid]="$name|running|$user|$command"
    system_debug "Created process: $name (PID $pid, User $user)"
    echo "$pid"
}

# === Export Functions ===
export -f systemctl ps pgrep pkill kill uname whoami id which
export -f test_system_connection test_system_health test_system_basic
export -f system_mock_reset system_mock_set_error system_mock_set_mode
export -f system_mock_dump_state system_mock_create_service system_mock_create_process
export -f system_debug system_check_error

# Initialize with defaults
system_mock_reset
system_debug "System Tier 2 mock initialized"