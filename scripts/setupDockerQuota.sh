#!/bin/bash
# This script sets up a dedicated systemd slice for the Docker daemon
# and assigns the docker.service to this slice. It calculates resource
# limits based on system characteristics:
#  - CPUQuota: (total cores - 0.5)*100 (%)
#  - MemoryMax: 80% of the total memory (in MB)
#
# The Docker slice will be defined in /etc/systemd/system/docker.slice,
# and the docker.service override will be in
# /etc/systemd/system/docker.service.d/slice.conf.
#
# Run this script as root.

set -euo pipefail

# -------------------------------
# Check for required utilities
# -------------------------------
required_utils=(nproc bc free awk sed grep mkdir systemctl)

for util in "${required_utils[@]}"; do
    if ! command -v "$util" &>/dev/null; then
        echo "Error: Required utility '$util' is not installed. Aborting." >&2
        exit 1
    fi
done

# -------------------------------
# Calculate resource limit values
# -------------------------------

# Get total number of CPU cores and calculate CPU quota.
N=$(nproc)
# Calculate quota: (N - 0.5) * 100. This value is later appended with '%' .
QUOTA=$(echo "($N - 0.5) * 100" | bc)
CPU_QUOTA="${QUOTA}%" 

# Get total memory (in MB) and calculate 80% of it.
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
MEM_LIMIT=$(echo "$TOTAL_MEM * 0.8" | bc | cut -d. -f1)M

# -------------------------------
# Define file paths for configuration
# -------------------------------

# This slice unit will hold the Docker daemon’s resource limits.
SLICE_FILE="/etc/systemd/system/docker.slice"

# The service override drop-in file ensures docker.service is placed in the Docker slice.
SERVICE_OVERRIDE_DIR="/etc/systemd/system/docker.service.d"
SERVICE_OVERRIDE_FILE="${SERVICE_OVERRIDE_DIR}/slice.conf"

# Flag to track if any changes are made
changed=false

# -------------------------------
# Update/create the slice unit file
# -------------------------------

# We want the slice file to contain a [Slice] section with CPUQuota and MemoryMax.
if [[ ! -f "$SLICE_FILE" ]]; then
    cat <<EOF > "$SLICE_FILE"
[Slice]
CPUQuota=${CPU_QUOTA}
MemoryMax=${MEM_LIMIT}
EOF
    changed=true
else
    # Ensure the [Slice] header exists.
    if ! grep -q "^\[Slice\]" "$SLICE_FILE"; then
        sed -i "1i[Slice]" "$SLICE_FILE"
        changed=true
    fi
    # Update or add CPUQuota setting.
    if grep -q '^CPUQuota=' "$SLICE_FILE"; then
        old_cpu=$(grep '^CPUQuota=' "$SLICE_FILE" | head -n 1 | cut -d= -f2-)
        if [[ "$old_cpu" != "$CPU_QUOTA" ]]; then
            sed -i "s/^CPUQuota=.*/CPUQuota=${CPU_QUOTA}/" "$SLICE_FILE"
            changed=true
        fi
    else
        sed -i "/^\[Slice\]/a CPUQuota=${CPU_QUOTA}" "$SLICE_FILE"
        changed=true
    fi

    # Update or add MemoryMax setting.
    if grep -q '^MemoryMax=' "$SLICE_FILE"; then
        old_mem=$(grep '^MemoryMax=' "$SLICE_FILE" | head -n 1 | cut -d= -f2-)
        if [[ "$old_mem" != "$MEM_LIMIT" ]]; then
            sed -i "s/^MemoryMax=.*/MemoryMax=${MEM_LIMIT}/" "$SLICE_FILE"
            changed=true
        fi
    else
        sed -i "/^\[Slice\]/a MemoryMax=${MEM_LIMIT}" "$SLICE_FILE"
        changed=true
    fi
fi

# -------------------------------
# Update/create the docker.service drop-in file
# -------------------------------

# Make sure the directory exists.
mkdir -p "$SERVICE_OVERRIDE_DIR"

# The override file should assign docker.service to the docker.slice.
if [[ ! -f "$SERVICE_OVERRIDE_FILE" ]]; then
    cat <<EOF > "$SERVICE_OVERRIDE_FILE"
[Service]
Slice=docker.slice
EOF
    changed=true
else
    # Ensure the [Service] header exists.
    if ! grep -q "^\[Service\]" "$SERVICE_OVERRIDE_FILE"; then
       sed -i "1i[Service]" "$SERVICE_OVERRIDE_FILE"
       changed=true
    fi

    # Check and update the Slice directive.
    if grep -q '^Slice=' "$SERVICE_OVERRIDE_FILE"; then
         old_slice=$(grep '^Slice=' "$SERVICE_OVERRIDE_FILE" | head -n 1 | cut -d= -f2-)
         if [[ "$old_slice" != "docker.slice" ]]; then
             sed -i "s/^Slice=.*/Slice=docker.slice/" "$SERVICE_OVERRIDE_FILE"
             changed=true
         fi
    else
         sed -i "/^\[Service\]/a Slice=docker.slice" "$SERVICE_OVERRIDE_FILE"
         changed=true
    fi
fi

# -------------------------------
# Reload systemd if changes were made
# -------------------------------

if [ "$changed" = true ]; then
    echo "Docker slice configuration updated."
    systemctl daemon-reload
    systemctl restart docker.service
else
    echo "Docker slice configuration unchanged. No action taken."
fi
