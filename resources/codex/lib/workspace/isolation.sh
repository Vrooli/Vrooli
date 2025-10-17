#!/usr/bin/env bash
################################################################################
# Workspace Isolation - Advanced sandboxing and process isolation
# 
# Provides container-like isolation for workspace operations including:
# - Filesystem namespace isolation
# - Process tree isolation
# - Resource containerization
# - Environment variable isolation
################################################################################

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

################################################################################
# Isolation Management
################################################################################

#######################################
# Create isolated environment for workspace
# Arguments:
#   $1 - Workspace ID
#   $2 - Isolation level (basic, advanced, strict)
#   $3 - Execution context (JSON)
# Returns:
#   Isolation setup result JSON
#######################################
workspace_isolation::create_environment() {
    local workspace_id="$1"
    local isolation_level="${2:-basic}"
    local context="${3:-{}}"
    
    log::info "Creating isolated environment for workspace: $workspace_id (level: $isolation_level)"
    
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ ! -d "$workspace_dir" ]]; then
        echo '{"success": false, "error": "Workspace not found"}'
        return 1
    fi
    
    # Create isolation configuration
    local isolation_config
    isolation_config=$(workspace_isolation::create_config "$isolation_level" "$context")
    
    # Save configuration
    echo "$isolation_config" > "$workspace_dir/config/isolation.json"
    
    # Set up isolation based on level
    case "$isolation_level" in
        basic)
            workspace_isolation::setup_basic_isolation "$workspace_id"
            ;;
        advanced)
            workspace_isolation::setup_advanced_isolation "$workspace_id"
            ;;
        strict)
            workspace_isolation::setup_strict_isolation "$workspace_id"
            ;;
        *)
            echo '{"success": false, "error": "Invalid isolation level"}'
            return 1
            ;;
    esac
    
    # Create execution wrapper
    workspace_isolation::create_execution_wrapper "$workspace_id" "$isolation_level"
    
    log::success "Isolation environment created for workspace: $workspace_id"
    
    cat << EOF
{
  "success": true,
  "workspace_id": "$workspace_id",
  "isolation_level": "$isolation_level",
  "environment_created_at": "$(date -Iseconds)",
  "isolation_config": $isolation_config
}
EOF
}

#######################################
# Create isolation configuration
# Arguments:
#   $1 - Isolation level
#   $2 - Context (JSON)
# Returns:
#   Isolation configuration JSON
#######################################
workspace_isolation::create_config() {
    local isolation_level="$1"
    local context="$2"
    
    case "$isolation_level" in
        basic)
            cat << 'EOF'
{
  "isolation_level": "basic",
  "filesystem": {
    "chroot": false,
    "bind_mounts": [],
    "read_only_mounts": ["/usr", "/lib", "/lib64"],
    "temp_dirs": ["/tmp", "/var/tmp"]
  },
  "processes": {
    "new_pid_namespace": false,
    "process_group_isolation": true,
    "signal_isolation": true,
    "max_processes": 20
  },
  "network": {
    "namespace_isolation": false,
    "localhost_only": true,
    "port_restrictions": true
  },
  "environment": {
    "clean_environment": true,
    "restricted_variables": ["HOME", "PATH", "USER"],
    "allowed_variables": ["LANG", "LC_ALL", "TZ"]
  },
  "resources": {
    "memory_limit": "256M",
    "cpu_limit": "50%",
    "disk_quota": "100M",
    "file_descriptor_limit": 1024
  }
}
EOF
            ;;
        advanced)
            cat << 'EOF'
{
  "isolation_level": "advanced",
  "filesystem": {
    "chroot": true,
    "bind_mounts": ["/lib", "/lib64", "/usr/lib", "/bin", "/usr/bin"],
    "read_only_mounts": ["/usr", "/lib", "/lib64", "/bin", "/sbin"],
    "temp_dirs": ["/tmp", "/var/tmp"],
    "dev_restrictions": true
  },
  "processes": {
    "new_pid_namespace": true,
    "process_group_isolation": true,
    "signal_isolation": true,
    "max_processes": 10
  },
  "network": {
    "namespace_isolation": true,
    "localhost_only": true,
    "port_restrictions": true,
    "dns_restrictions": true
  },
  "environment": {
    "clean_environment": true,
    "restricted_variables": ["HOME", "PATH", "USER", "SHELL"],
    "allowed_variables": ["LANG", "LC_ALL", "TZ"]
  },
  "resources": {
    "memory_limit": "128M",
    "cpu_limit": "25%",
    "disk_quota": "50M",
    "file_descriptor_limit": 512,
    "cgroup_isolation": true
  }
}
EOF
            ;;
        strict)
            cat << 'EOF'
{
  "isolation_level": "strict",
  "filesystem": {
    "chroot": true,
    "bind_mounts": ["/lib", "/lib64"],
    "read_only_mounts": ["/lib", "/lib64"],
    "temp_dirs": [],
    "dev_restrictions": true,
    "proc_restrictions": true
  },
  "processes": {
    "new_pid_namespace": true,
    "process_group_isolation": true,
    "signal_isolation": true,
    "max_processes": 5,
    "seccomp_filter": true
  },
  "network": {
    "namespace_isolation": true,
    "localhost_only": false,
    "network_disabled": true,
    "port_restrictions": true,
    "dns_restrictions": true
  },
  "environment": {
    "clean_environment": true,
    "restricted_variables": ["*"],
    "allowed_variables": ["LANG"]
  },
  "resources": {
    "memory_limit": "64M",
    "cpu_limit": "10%",
    "disk_quota": "25M",
    "file_descriptor_limit": 256,
    "cgroup_isolation": true,
    "rlimit_enforcement": true
  }
}
EOF
            ;;
    esac
}

################################################################################
# Isolation Levels Implementation
################################################################################

#######################################
# Setup basic isolation (process and environment only)
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_isolation::setup_basic_isolation() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create basic isolation wrapper
    cat > "$workspace_dir/config/basic_isolator.sh" << 'EOF'
#!/bin/bash
# Basic isolation wrapper

WORKSPACE_ID="$1"
WORKSPACE_DIR="$2"
shift 2
COMMAND="$*"

# Set up basic environment isolation
export HOME="$WORKSPACE_DIR/data"
export PATH="/usr/local/bin:/usr/bin:/bin"
export USER="workspace"
export SHELL="/bin/bash"
export TMPDIR="$WORKSPACE_DIR/tmp"

# Unset potentially dangerous variables
unset LD_PRELOAD
unset LD_LIBRARY_PATH
unset SUDO_USER
unset SUDO_COMMAND

# Create new process group
set -m

# Execute command with resource limits
ulimit -v 262144  # 256MB virtual memory
ulimit -f 102400  # 100MB file size
ulimit -n 1024    # 1024 file descriptors

# Change to workspace directory
cd "$WORKSPACE_DIR/data" || exit 1

# Execute command in isolated environment
exec setsid bash -c "$COMMAND"
EOF
    
    chmod +x "$workspace_dir/config/basic_isolator.sh"
}

#######################################
# Setup advanced isolation (chroot + namespaces)
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_isolation::setup_advanced_isolation() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create chroot environment
    workspace_isolation::create_chroot_environment "$workspace_dir"
    
    # Create advanced isolation wrapper
    cat > "$workspace_dir/config/advanced_isolator.sh" << 'EOF'
#!/bin/bash
# Advanced isolation wrapper with chroot and namespaces

WORKSPACE_ID="$1"
WORKSPACE_DIR="$2"
shift 2
COMMAND="$*"

CHROOT_DIR="$WORKSPACE_DIR/chroot"

# Verify chroot environment
if [[ ! -d "$CHROOT_DIR" ]]; then
    echo "ERROR: Chroot environment not found"
    exit 1
fi

# Set up advanced isolation with unshare if available
UNSHARE_OPTS=""
if command -v unshare >/dev/null 2>&1; then
    # Create new PID namespace
    UNSHARE_OPTS="$UNSHARE_OPTS --pid --fork"
    
    # Create new mount namespace
    UNSHARE_OPTS="$UNSHARE_OPTS --mount"
    
    # Create new network namespace (limited)
    if [[ "$(id -u)" == "0" ]]; then
        UNSHARE_OPTS="$UNSHARE_OPTS --net"
    fi
fi

# Create cgroup for resource control if available
CGROUP_DIR=""
if [[ -d "/sys/fs/cgroup" && "$(id -u)" == "0" ]]; then
    CGROUP_DIR="/sys/fs/cgroup/workspace-$WORKSPACE_ID"
    workspace_isolation::setup_cgroup "$CGROUP_DIR"
fi

# Execute with advanced isolation
if [[ -n "$UNSHARE_OPTS" ]]; then
    exec unshare $UNSHARE_OPTS chroot "$CHROOT_DIR" /bin/bash -c "
        export HOME=/data
        export PATH=/bin:/usr/bin
        export USER=workspace
        export TMPDIR=/tmp
        cd /data || exit 1
        exec $COMMAND
    "
else
    # Fallback to basic chroot
    exec chroot "$CHROOT_DIR" /bin/bash -c "
        export HOME=/data
        export PATH=/bin:/usr/bin
        export USER=workspace
        export TMPDIR=/tmp
        cd /data || exit 1
        exec $COMMAND
    "
fi
EOF
    
    chmod +x "$workspace_dir/config/advanced_isolator.sh"
}

#######################################
# Setup strict isolation (full containerization)
# Arguments:
#   $1 - Workspace ID
#######################################
workspace_isolation::setup_strict_isolation() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create minimal chroot environment
    workspace_isolation::create_minimal_chroot "$workspace_dir"
    
    # Create strict isolation wrapper
    cat > "$workspace_dir/config/strict_isolator.sh" << 'EOF'
#!/bin/bash
# Strict isolation wrapper with full containerization

WORKSPACE_ID="$1"
WORKSPACE_DIR="$2"
shift 2
COMMAND="$*"

CHROOT_DIR="$WORKSPACE_DIR/chroot"

# Verify minimal chroot environment
if [[ ! -d "$CHROOT_DIR" ]]; then
    echo "ERROR: Minimal chroot environment not found"
    exit 1
fi

# Set up strict isolation with all namespaces
UNSHARE_OPTS=""
if command -v unshare >/dev/null 2>&1; then
    UNSHARE_OPTS="--pid --fork --mount --ipc --uts"
    
    # Add network namespace isolation if running as root
    if [[ "$(id -u)" == "0" ]]; then
        UNSHARE_OPTS="$UNSHARE_OPTS --net"
    fi
fi

# Set up seccomp filter if available
SECCOMP_FILE="$WORKSPACE_DIR/config/seccomp_filter.json"
if command -v bwrap >/dev/null 2>&1 && [[ -f "$SECCOMP_FILE" ]]; then
    # Use bubblewrap for strict sandboxing
    exec bwrap \
        --bind "$WORKSPACE_DIR/data" /data \
        --tmpfs /tmp \
        --ro-bind /lib /lib \
        --ro-bind /lib64 /lib64 \
        --ro-bind /bin /bin \
        --ro-bind /usr/bin /usr/bin \
        --dev /dev \
        --proc /proc \
        --unshare-all \
        --share-net \
        --die-with-parent \
        --seccomp 10 10<"$SECCOMP_FILE" \
        /bin/bash -c "
            export HOME=/data
            export PATH=/bin:/usr/bin
            export USER=workspace
            cd /data || exit 1
            exec $COMMAND
        "
elif [[ -n "$UNSHARE_OPTS" ]]; then
    # Fallback to unshare with chroot
    exec unshare $UNSHARE_OPTS chroot "$CHROOT_DIR" /bin/bash -c "
        export HOME=/data
        export PATH=/bin
        cd /data || exit 1
        ulimit -v 65536   # 64MB virtual memory
        ulimit -f 25600   # 25MB file size
        ulimit -n 256     # 256 file descriptors
        exec $COMMAND
    "
else
    echo "ERROR: Strict isolation not available - unshare not found"
    exit 1
fi
EOF
    
    chmod +x "$workspace_dir/config/strict_isolator.sh"
    
    # Create seccomp filter for strict isolation
    workspace_isolation::create_seccomp_filter "$workspace_dir"
}

################################################################################
# Chroot Environment Setup
################################################################################

#######################################
# Create chroot environment for workspace
# Arguments:
#   $1 - Workspace directory
#######################################
workspace_isolation::create_chroot_environment() {
    local workspace_dir="$1"
    local chroot_dir="$workspace_dir/chroot"
    
    log::debug "Creating chroot environment: $chroot_dir"
    
    # Create directory structure
    mkdir -p "$chroot_dir"/{bin,lib,lib64,usr/bin,usr/lib,tmp,data,dev,proc,sys}
    
    # Copy essential binaries
    local essential_bins=("/bin/bash" "/bin/sh" "/bin/cat" "/bin/echo" "/bin/ls" "/bin/pwd")
    for bin in "${essential_bins[@]}"; do
        if [[ -f "$bin" ]]; then
            cp "$bin" "$chroot_dir$bin" 2>/dev/null || true
        fi
    done
    
    # Copy essential libraries
    workspace_isolation::copy_libraries "$chroot_dir" "/bin/bash"
    
    # Set up devices
    if [[ "$(id -u)" == "0" ]]; then
        mknod "$chroot_dir/dev/null" c 1 3 2>/dev/null || true
        mknod "$chroot_dir/dev/zero" c 1 5 2>/dev/null || true
        mknod "$chroot_dir/dev/random" c 1 8 2>/dev/null || true
        mknod "$chroot_dir/dev/urandom" c 1 9 2>/dev/null || true
    else
        # Non-root fallback - bind mount if possible
        touch "$chroot_dir/dev/null" "$chroot_dir/dev/zero"
    fi
    
    # Link data directory
    ln -sf "$workspace_dir/data" "$chroot_dir/data" 2>/dev/null || true
    
    # Set permissions
    chmod 755 "$chroot_dir"
    chmod 1777 "$chroot_dir/tmp"
}

#######################################
# Create minimal chroot environment for strict isolation
# Arguments:
#   $1 - Workspace directory
#######################################
workspace_isolation::create_minimal_chroot() {
    local workspace_dir="$1"
    local chroot_dir="$workspace_dir/chroot"
    
    log::debug "Creating minimal chroot environment: $chroot_dir"
    
    # Create minimal directory structure
    mkdir -p "$chroot_dir"/{bin,lib,lib64,tmp,data,dev}
    
    # Copy only essential binaries
    local minimal_bins=("/bin/sh" "/bin/bash")
    for bin in "${minimal_bins[@]}"; do
        if [[ -f "$bin" ]]; then
            cp "$bin" "$chroot_dir$bin" 2>/dev/null || true
            workspace_isolation::copy_libraries "$chroot_dir" "$bin"
        fi
    done
    
    # Minimal device setup
    touch "$chroot_dir/dev/null"
    
    # Link data directory
    ln -sf "$workspace_dir/data" "$chroot_dir/data" 2>/dev/null || true
    
    # Strict permissions
    chmod 700 "$chroot_dir"
    chmod 700 "$chroot_dir/tmp"
}

#######################################
# Copy required libraries for a binary
# Arguments:
#   $1 - Chroot directory
#   $2 - Binary path
#######################################
workspace_isolation::copy_libraries() {
    local chroot_dir="$1"
    local binary="$2"
    
    if command -v ldd >/dev/null 2>&1; then
        # Get library dependencies
        ldd "$binary" 2>/dev/null | while read -r line; do
            # Extract library path
            local lib_path
            lib_path=$(echo "$line" | grep -o '/[^ ]*' | head -1)
            
            if [[ -f "$lib_path" ]]; then
                local lib_dir
                lib_dir=$(dirname "$lib_path")
                mkdir -p "$chroot_dir$lib_dir"
                cp "$lib_path" "$chroot_dir$lib_path" 2>/dev/null || true
            fi
        done
    fi
}

################################################################################
# Resource Control
################################################################################

#######################################
# Setup cgroup for resource control
# Arguments:
#   $1 - Cgroup directory path
#######################################
workspace_isolation::setup_cgroup() {
    local cgroup_dir="$1"
    
    if [[ "$(id -u)" != "0" ]]; then
        log::debug "Skipping cgroup setup - not running as root"
        return 1
    fi
    
    # Create cgroup directory
    mkdir -p "$cgroup_dir" 2>/dev/null || return 1
    
    # Set memory limit (128MB)
    echo "134217728" > "$cgroup_dir/memory.limit_in_bytes" 2>/dev/null || true
    
    # Set CPU limit (25%)
    echo "25000" > "$cgroup_dir/cpu.cfs_quota_us" 2>/dev/null || true
    echo "100000" > "$cgroup_dir/cpu.cfs_period_us" 2>/dev/null || true
    
    # Add current process to cgroup
    echo "$$" > "$cgroup_dir/cgroup.procs" 2>/dev/null || true
}

#######################################
# Create seccomp filter for strict isolation
# Arguments:
#   $1 - Workspace directory
#######################################
workspace_isolation::create_seccomp_filter() {
    local workspace_dir="$1"
    
    # Create basic seccomp filter JSON
    cat > "$workspace_dir/config/seccomp_filter.json" << 'EOF'
{
  "defaultAction": "SCMP_ACT_ERRNO",
  "syscalls": [
    {"names": ["read", "write", "open", "close", "stat", "fstat", "lstat"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["mmap", "mprotect", "munmap", "brk"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["rt_sigaction", "rt_sigprocmask", "rt_sigreturn"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["ioctl"], "action": "SCMP_ACT_ALLOW", "args": [{"index": 1, "value": 21505, "op": "SCMP_CMP_EQ"}]},
    {"names": ["exit", "exit_group"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["getpid", "getppid", "getuid", "getgid"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["access", "execve", "wait4"], "action": "SCMP_ACT_ALLOW"},
    {"names": ["clone"], "action": "SCMP_ACT_ERRNO"},
    {"names": ["fork", "vfork"], "action": "SCMP_ACT_ERRNO"},
    {"names": ["socket", "connect", "accept", "bind", "listen"], "action": "SCMP_ACT_ERRNO"}
  ]
}
EOF
}

################################################################################
# Execution Interface
################################################################################

#######################################
# Create unified execution wrapper
# Arguments:
#   $1 - Workspace ID
#   $2 - Isolation level
#######################################
workspace_isolation::create_execution_wrapper() {
    local workspace_id="$1"
    local isolation_level="$2"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    # Create main execution wrapper
    cat > "$workspace_dir/config/isolated_exec.sh" << EOF
#!/bin/bash
# Unified isolated execution wrapper

WORKSPACE_ID="$workspace_id"
WORKSPACE_DIR="$workspace_dir"
ISOLATION_LEVEL="$isolation_level"
COMMAND="\$*"

# Log execution
echo "\$(date -Iseconds) EXEC level=\$ISOLATION_LEVEL command='\$COMMAND'" >> "\$WORKSPACE_DIR/logs/execution.log"

# Execute based on isolation level
case "\$ISOLATION_LEVEL" in
    basic)
        exec "\$WORKSPACE_DIR/config/basic_isolator.sh" "\$WORKSPACE_ID" "\$WORKSPACE_DIR" \$COMMAND
        ;;
    advanced)
        exec "\$WORKSPACE_DIR/config/advanced_isolator.sh" "\$WORKSPACE_ID" "\$WORKSPACE_DIR" \$COMMAND
        ;;
    strict)
        exec "\$WORKSPACE_DIR/config/strict_isolator.sh" "\$WORKSPACE_ID" "\$WORKSPACE_DIR" \$COMMAND
        ;;
    *)
        echo "ERROR: Unknown isolation level: \$ISOLATION_LEVEL"
        exit 1
        ;;
esac
EOF
    
    chmod +x "$workspace_dir/config/isolated_exec.sh"
}

#######################################
# Execute command in isolated environment
# Arguments:
#   $1 - Workspace ID
#   $2 - Command to execute
#   $3 - Execution options (JSON)
# Returns:
#   Execution result JSON
#######################################
workspace_isolation::execute() {
    local workspace_id="$1"
    local command="$2"
    local options="${3:-{}}"
    
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    local exec_wrapper="$workspace_dir/config/isolated_exec.sh"
    
    if [[ ! -f "$exec_wrapper" ]]; then
        echo '{"success": false, "error": "Isolation environment not configured"}'
        return 1
    fi
    
    # Extract execution options
    local timeout
    timeout=$(echo "$options" | jq -r '.timeout // 300')
    
    log::debug "Executing isolated command in workspace $workspace_id: $command"
    
    # Execute with timeout
    local start_time=$(date +%s)
    local output exit_code
    
    output=$(timeout "$timeout" "$exec_wrapper" "$command" 2>&1)
    exit_code=$?
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Handle timeout
    if [[ $exit_code -eq 124 ]]; then
        echo "{\"success\": false, \"error\": \"Command timed out after $timeout seconds\", \"exit_code\": $exit_code, \"duration\": $duration}"
        return 1
    fi
    
    # Escape output for JSON
    local escaped_output
    escaped_output=$(echo "$output" | jq -Rs .)
    
    # Determine success
    local success
    if [[ $exit_code -eq 0 ]]; then
        success="true"
    else
        success="false"
    fi
    
    cat << EOF
{
  "success": $success,
  "exit_code": $exit_code,
  "output": $escaped_output,
  "duration": $duration,
  "workspace_id": "$workspace_id",
  "command": $(echo "$command" | jq -Rs .),
  "executed_at": "$(date -Iseconds)"
}
EOF
}

#######################################
# Get isolation status for workspace
# Arguments:
#   $1 - Workspace ID
# Returns:
#   Isolation status JSON
#######################################
workspace_isolation::get_status() {
    local workspace_id="$1"
    local workspace_dir="$WORKSPACE_BASE_DIR/$workspace_id"
    
    if [[ ! -d "$workspace_dir" ]]; then
        echo '{"success": false, "error": "Workspace not found"}'
        return 1
    fi
    
    # Load isolation configuration
    local config="{}"
    local config_file="$workspace_dir/config/isolation.json"
    if [[ -f "$config_file" ]]; then
        config=$(cat "$config_file")
    fi
    
    # Check if isolation components are available
    local capabilities="{}"
    capabilities=$(jq -n \
        --argjson unshare "$(command -v unshare >/dev/null && echo true || echo false)" \
        --argjson chroot "$(command -v chroot >/dev/null && echo true || echo false)" \
        --argjson bwrap "$(command -v bwrap >/dev/null && echo true || echo false)" \
        --argjson cgroups "$([[ -d /sys/fs/cgroup ]] && echo true || echo false)" \
        '{
            unshare: $unshare,
            chroot: $chroot,
            bubblewrap: $bwrap,
            cgroups: $cgroups
        }')
    
    # Check if chroot environment exists
    local chroot_status="not_configured"
    if [[ -d "$workspace_dir/chroot" ]]; then
        chroot_status="configured"
    fi
    
    cat << EOF
{
  "success": true,
  "workspace_id": "$workspace_id",
  "isolation_level": $(echo "$config" | jq -r '.isolation_level // "none"'),
  "chroot_status": "$chroot_status",
  "system_capabilities": $capabilities,
  "configuration": $config
}
EOF
}

# Export functions
export -f workspace_isolation::create_environment
export -f workspace_isolation::execute
export -f workspace_isolation::get_status