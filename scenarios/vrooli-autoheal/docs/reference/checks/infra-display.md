# Display Manager Check (infra-display)

Monitors the display manager (GDM, LightDM, SDDM, etc.) and X11/Wayland responsiveness on Linux systems.

## Overview

| Property | Value |
|----------|-------|
| Check ID | `infra-display` |
| Category | Infrastructure |
| Interval | 5 minutes |
| Platforms | Linux only |

## What It Monitors

This check verifies that the graphical login system is functioning:

1. **Display Manager Service**: GDM, LightDM, SDDM, LXDM, or XDM
2. **X11/Wayland Responsiveness**: Verifies the display server is responding (when DISPLAY is set)

## Supported Display Managers

- **gdm/gdm3** - GNOME Display Manager
- **lightdm** - Light Display Manager (Ubuntu, Xubuntu)
- **sddm** - Simple Desktop Display Manager (KDE/Plasma)
- **lxdm** - LXDE Display Manager
- **xdm** - X Display Manager (legacy)

## Status Meanings

| Status | Meaning |
|--------|---------|
| **OK** | Display manager running and X11/Wayland responsive |
| **Warning** | Display manager running but X11 not responsive |
| **Critical** | Display manager service not running |
| **OK (headless)** | Headless server detected - no display manager expected |

## Why It Matters

Display manager failures cause:
- Inability to log into graphical sessions
- Black screen after boot
- Loss of remote desktop access (VNC, RDP via xrdp)
- Disconnected existing sessions on restart

## Recovery Actions

| Action | Description | Risk |
|--------|-------------|------|
| **Restart Display Manager** | Restarts the DM service | **HIGH** - Disconnects active sessions |
| **Check Status** | Gets detailed service status | Safe |
| **View Logs** | Shows recent DM logs | Safe |

## Troubleshooting Steps

### 1. Check Service Status
```bash
# Identify which display manager is in use
systemctl list-units --type=service | grep -E "gdm|lightdm|sddm"

# Check status
systemctl status gdm  # or lightdm, sddm
```

### 2. View Logs
```bash
# Display manager logs
journalctl -u gdm -n 50

# Xorg logs
cat /var/log/Xorg.0.log | tail -100
```

### 3. Test X11 Responsiveness
```bash
# Check if DISPLAY is set
echo $DISPLAY

# Test X11 connection
xdpyinfo | head -20
```

### 4. Restart Display Manager
```bash
# WARNING: This will disconnect all graphical sessions!
sudo systemctl restart gdm  # or lightdm, sddm
```

## Headless Server Detection

This check automatically detects headless servers and reports OK status. A server is considered headless if:
- No graphical.target is the default systemd target
- No display manager service is installed/enabled
- Platform capabilities indicate headless mode

## Common Failure Causes

### GDM Won't Start
```bash
# Check for GPU driver issues
dmesg | grep -i gpu
dmesg | grep -i drm

# Check for Xorg configuration issues
cat /etc/X11/xorg.conf
```

### Login Loop
```bash
# Often caused by ~/.Xauthority permission issues
ls -la ~/.Xauthority
rm ~/.Xauthority  # Forces regeneration
```

### Black Screen
```bash
# Switch to TTY
Ctrl+Alt+F2

# Check logs from there
journalctl -u gdm -b
```

## Related Checks

- **infra-rdp**: RDP/xrdp remote access (depends on display manager)
- **system-disk**: Disk space issues can cause DM failures

---

*Back to [Check Catalog](../check-catalog.md)*
