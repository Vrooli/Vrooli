# Environment Management

This page describes the current project-level environment and secrets posture.

## Current Truth

Environment and secrets handling in Vrooli is a mix of:

- project-level configuration under `.vrooli/`
- project and scenario manifests
- host-tool and setup requirements declared in manifests and setup flows
- deployment-tier-specific secrets behavior described in the Deployment Hub

This is not a stable “one environment model for everything” story yet, so avoid treating older environment docs as universally canonical.

## What To Use Today

For setup and development:

```bash
vrooli setup --help
vrooli develop --help
vrooli status
vrooli doctor
```

These commands are the best current entrypoints for understanding environment expectations, tool availability, and setup posture.

For deployment-tier-specific secrets thinking:

- [../deployment/README.md](../deployment/README.md)

For scenario-specific runtime expectations:

- the scenario's own `.vrooli/service.json`
- scenario-local docs when the behavior is specific to that scenario

## Guidance

- Treat project configuration under `.vrooli/` as part of the project-level operational truth.
- Treat scenario-local `.vrooli/service.json` files as scenario-specific operational truth.
- Avoid documenting environment behavior in old shell-script terms unless that specific path still exists and is intentionally supported.
- Be careful with secrets claims: current behavior varies by setup path and deployment tier.
- Keep host requirements, setup behavior, and secret handling distinct. They overlap, but they are not one single unified concern.

## Status

This page is intentionally conservative until a fuller environment-and-secrets rewrite happens, but it is still the canonical project-level reference for the current posture.

## Platform evidence

Platform-specific setup and lifecycle claims use the canonical
[platform support matrix](platform-support.md). In particular, macOS is not
hardware-qualified merely because a setup or cross-build command is available.

## Host disk floor (operator steps)

> Added after the 2026-07-31 disk-exhaustion incident, in which the host root
> filesystem reached 100 percent, the runtime supervisor crash-looped about 110
> times in nine minutes, rsyslog lost log data, and the emergency watchdog
> itself failed to write.

The Go safeguards now detect and remediate disk pressure on their own. The
steps below are the host-level floor underneath them, and they need root, so
**an operator applies them deliberately — no Vrooli process does this silently.**

### 1. Age Vrooli build staging in `/tmp` at three days

The default policy cannot work. `systemd-tmpfiles-clean.timer` fires daily, but
`/usr/lib/tmpfiles.d/tmp.conf` ages `/tmp` at 30 days. At incident time exactly
one entry in `/tmp` was older than thirty days while 116 GB across 43,647
entries was younger: build-staging directories of about 2.4 GB each are created
far faster than a 30-day TTL can ever retire them.

This override shortens **only** the Vrooli staging patterns and leaves every
other `/tmp` path on the default 30-day rule.

```bash
sudo tee /etc/tmpfiles.d/vrooli-tmp.conf >/dev/null <<'EOF'
# Vrooli build-staging debris ages at 3 days, not the /tmp default of 30.
# These directories are ~2.4 GB each and are recreated on every release build,
# so a 30-day TTL can never retire them faster than they accumulate.
# Everything else in /tmp keeps the default rule from /usr/lib/tmpfiles.d/tmp.conf.
d /tmp/vrooli-release          1777 root root 3d
e /tmp/vrooli-.*-release       -    -    -    3d
e /tmp/vrooli-resource-release -    -    -    3d
e /tmp/tmp.*                   -    -    -    3d
EOF

# Read back what the new rule would clean, without cleaning anything:
sudo systemd-tmpfiles --dry-run --clean 2>&1 | grep -E 'vrooli|tmp\.' | head -20

# Apply when the dry run looks right:
sudo systemd-tmpfiles --clean
```

**Never shorten `systemd-private-*`.** Those directories carry the current boot
id despite old mtimes, and deleting them breaks running services.

### 2. Reserve a floor on the root filesystem

Filling the data areas must not make the root filesystem unwritable for the
supervisor and the watchdog. ext4 reserves 5 percent for root by default; on a
1.8 TB filesystem that is about 93 GB, which is what kept `df` reporting 93
percent while a `Bfree`-based check reported 87.

Confirm the reserve is intact:

```bash
sudo tune2fs -l /dev/nvme0n1p2 | grep -i 'reserved block count'
```

If the reserve has been reduced to zero (a common space-saving change), restore
it to 1 percent — enough for root-owned writes without giving up much capacity:

```bash
sudo tune2fs -m 1 /dev/nvme0n1p2
```

The watchdog enforces a second, unprivileged floor above this one. It is
configurable and defaults to 10 GiB:

```bash
EMERGENCY_WATCHDOG_DISK_FLOOR_MB=10240   # request cleanup below this
EMERGENCY_WATCHDOG_DISK_THRESHOLD=120    # seconds below the floor before escalating
EMERGENCY_WATCHDOG_MOUNT=/               # root and home share one filesystem here
```

Below the floor the watchdog requests a `high`-band cleanup; below half the
floor it requests `critical`, which permits unattended safe-tier reclamation.

### 3. Bound the journal

The journal was never vacuumed during the incident because it needs root:

```bash
sudo journalctl --vacuum-size=200M
sudo sed -i 's/^#\?SystemMaxUse=.*/SystemMaxUse=500M/' /etc/systemd/journald.conf
sudo systemctl restart systemd-journald
```

### 4. Remove the dead cron entry

`*/5 * * * * ./automation/scheduled-monitor.sh main` has never executed: the
path is relative and cron runs from `$HOME`, while the script lived under
`scenarios/system-monitor/`. Repairing it would be wrong even so — it targeted
port 8080 (the live port is 16914) and checked only CPU and memory, never disk.
The system-monitor threshold loop now owns that job.

```bash
crontab -l | grep -v 'automation/scheduled-monitor.sh' | crontab -
crontab -l   # confirm no entry references a relative path
```

### Rehearsal

Never rehearse against the live root filesystem. Drive the watchdog's disk
logic against a synthetic `df` and a stubbed `storage-manager` instead, or a
loopback mount if one can be created:

```bash
truncate -s 200M /tmp/vrooli-rehearsal.img
mkfs.ext4 -q /tmp/vrooli-rehearsal.img
mkdir -p /tmp/vrooli-rehearsal-mnt
sudo mount -o loop /tmp/vrooli-rehearsal.img /tmp/vrooli-rehearsal-mnt

# Fill past each band and observe the watchdog's decisions:
EMERGENCY_WATCHDOG_MOUNT=/tmp/vrooli-rehearsal-mnt \
EMERGENCY_WATCHDOG_DISK_FLOOR_MB=100 \
EMERGENCY_WATCHDOG_DISK_THRESHOLD=0 \
  bash scripts/emergency-watchdog.sh

sudo umount /tmp/vrooli-rehearsal-mnt && rm -f /tmp/vrooli-rehearsal.img
```

### Reading current disk pressure

One command reports usage, the active band, the last violation, and the last
remediation result:

```bash
curl -s http://localhost:16914/api/v1/disk-pressure | jq
```
