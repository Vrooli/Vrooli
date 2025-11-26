#!/usr/bin/env bash
# =============================================================================
# System Resource Checks
# Disk space, inodes, swap, zombie processes, port exhaustion, certificates
# =============================================================================

check_disk_space() {
    [[ "$CHECK_DISK_SPACE" != "true" ]] && return 0

    log "DEBUG" "Checking disk space usage"

    local issues=0
    local partition

    for partition in $DISK_PARTITIONS; do
        if [[ ! -d "$partition" ]]; then
            log "DEBUG" "Partition $partition not found, skipping"
            continue
        fi

        local usage
        usage=$(df "$partition" | awk 'NR==2 {print $5}' | sed 's/%//')

        if [[ -n "$usage" ]] && [[ "$usage" -ge "$DISK_THRESHOLD" ]]; then
            log "ERROR" "Disk usage on $partition is ${usage}% (threshold: ${DISK_THRESHOLD}%)"
            log "WARN" "Consider cleaning up disk space: tmp files, old logs, docker images"
            log "INFO" "Largest directories: $(du -sh "$partition"/* 2>/dev/null | sort -rh | head -5 | tr '\n' ' ')"
            issues=$((issues + 1))
        else
            log "DEBUG" "Disk usage on $partition: ${usage}%"
        fi
    done

    return "$issues"
}

check_inode_usage() {
    [[ "$CHECK_INODE" != "true" ]] && return 0

    log "DEBUG" "Checking inode usage"

    local issues=0
    local partition

    for partition in $DISK_PARTITIONS; do
        if [[ ! -d "$partition" ]]; then
            continue
        fi

        local usage
        usage=$(df -i "$partition" | awk 'NR==2 {print $5}' | sed 's/%//')

        if [[ -n "$usage" ]] && [[ "$usage" -ge "$INODE_THRESHOLD" ]]; then
            log "ERROR" "Inode usage on $partition is ${usage}% (threshold: ${INODE_THRESHOLD}%)"
            log "WARN" "Too many files! Find directories with most files: find $partition -xdev -type d -exec sh -c 'echo \$(ls -a \"{}\" | wc -l) \"{}\"' \; | sort -rn | head -10"
            issues=$((issues + 1))
        else
            log "DEBUG" "Inode usage on $partition: ${usage}%"
        fi
    done

    return "$issues"
}

check_swap_usage() {
    [[ "$CHECK_SWAP" != "true" ]] && return 0

    log "DEBUG" "Checking swap usage"

    local swap_total swap_used swap_percent

    swap_total=$(free | awk '/^Swap:/ {print $2}')
    swap_used=$(free | awk '/^Swap:/ {print $3}')

    if [[ -z "$swap_total" ]] || [[ "$swap_total" -eq 0 ]]; then
        log "DEBUG" "No swap configured"
        return 0
    fi

    swap_percent=$((swap_used * 100 / swap_total))

    if [[ "$swap_percent" -ge "$SWAP_THRESHOLD" ]]; then
        log "ERROR" "Swap usage is ${swap_percent}% (threshold: ${SWAP_THRESHOLD}%)"
        log "WARN" "High swap usage indicates memory pressure"
        log "INFO" "Top memory consumers: $(ps aux --sort=-%mem | awk 'NR<=6 {print $11}' | tail -5 | tr '\n' ' ')"
        return 1
    else
        log "DEBUG" "Swap usage: ${swap_percent}%"
        return 0
    fi
}

check_zombie_processes() {
    [[ "$CHECK_ZOMBIES" != "true" ]] && return 0

    log "DEBUG" "Checking for zombie processes"

    local zombie_count
    zombie_count=$(ps aux | awk '$8 ~ /Z/ {print}' | wc -l)

    if [[ "$zombie_count" -ge "$ZOMBIE_THRESHOLD" ]]; then
        log "WARN" "Found $zombie_count zombie processes (threshold: $ZOMBIE_THRESHOLD)"

        if [[ "$ZOMBIE_AUTO_CLEANUP" == "true" ]]; then
            log "INFO" "Attempting to clean up zombie processes"
            # Call cleanup function from recovery/cleanup.sh
            cleanup_zombie_processes
        else
            log "INFO" "Zombie auto-cleanup disabled - showing zombie parent processes"
            ps aux | awk '$8 ~ /Z/ {print}' | head -5 | log_stream "INFO"
        fi

        return 1
    else
        log "DEBUG" "Zombie process count: $zombie_count"
        return 0
    fi
}

check_port_exhaustion() {
    [[ "$CHECK_PORT_EXHAUSTION" != "true" ]] && return 0

    log "DEBUG" "Checking port exhaustion"

    local port_range_file="/proc/sys/net/ipv4/ip_local_port_range"

    if [[ ! -f "$port_range_file" ]]; then
        log "DEBUG" "Port range file not found, skipping"
        return 0
    fi

    read -r port_min port_max < "$port_range_file"
    local port_range=$((port_max - port_min))

    # Count established connections using ephemeral ports
    local connections
    connections=$(ss -tan | awk -v min="$port_min" -v max="$port_max" '
        NR>1 {
            split($4, a, ":")
            port = a[length(a)]
            if (port >= min && port <= max) count++
        }
        END {print count+0}
    ')

    local usage_percent=$((connections * 100 / port_range))

    if [[ "$usage_percent" -ge "$PORT_USAGE_THRESHOLD" ]]; then
        log "ERROR" "Ephemeral port usage is ${usage_percent}% ($connections/$port_range used)"
        log "WARN" "High port usage - may cause connection failures"
        log "INFO" "Top connection consumers: $(ss -tanp | awk 'NR>1 {print $7}' | cut -d',' -f2 | sort | uniq -c | sort -rn | head -5 | tr '\n' ' ')"
        return 1
    else
        log "DEBUG" "Ephemeral port usage: ${usage_percent}% ($connections/$port_range)"
        return 0
    fi
}

check_certificate_expiration() {
    [[ "$CHECK_CERTIFICATES" != "true" ]] && return 0

    log "DEBUG" "Checking certificate expiration"

    local cloudflared_cert="${HOME}/.cloudflared"

    if [[ -d "$cloudflared_cert" ]]; then
        local cert_files
        cert_files=$(find "$cloudflared_cert" -name "*.pem" -o -name "*.crt" 2>/dev/null)

        for cert_file in $cert_files; do
            if [[ -f "$cert_file" ]]; then
                local expiry_date
                expiry_date=$(openssl x509 -in "$cert_file" -noout -enddate 2>/dev/null | cut -d= -f2)

                if [[ -n "$expiry_date" ]]; then
                    local expiry_epoch
                    expiry_epoch=$(date -d "$expiry_date" +%s 2>/dev/null || echo "0")
                    local now_epoch
                    now_epoch=$(date +%s)
                    local days_until_expiry=$(( (expiry_epoch - now_epoch) / 86400 ))

                    if [[ "$days_until_expiry" -le "$CERT_WARNING_DAYS" ]]; then
                        log "WARN" "Certificate $cert_file expires in $days_until_expiry days"
                    else
                        log "DEBUG" "Certificate $cert_file valid for $days_until_expiry days"
                    fi
                fi
            fi
        done
    fi

    return 0
}
