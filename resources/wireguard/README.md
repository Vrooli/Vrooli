# WireGuard Resource

Modern, high-performance VPN networking for secure Vrooli resource connectivity.

## Overview

WireGuard provides state-of-the-art cryptographic VPN tunnels for:
- Secure point-to-point connections between resources
- Remote access to local Vrooli deployments
- Network isolation for containerized workloads
- Multi-cloud and edge computing connectivity

## Quick Start

```bash
# Install WireGuard
vrooli resource wireguard manage install

# Start the service
vrooli resource wireguard manage start --wait

# Check status
vrooli resource wireguard status

# Create a tunnel configuration
vrooli resource wireguard content add my-tunnel
```

## Features

### Core Capabilities
- **High Performance**: Kernel-level implementation with minimal overhead
- **Modern Cryptography**: ChaCha20, Poly1305, Curve25519, BLAKE2s
- **Simple Configuration**: Public key-based authentication
- **Cross-Platform**: Works across Linux, Windows, macOS, mobile devices
- **Low Latency**: <1ms overhead for encrypted tunnels
- **Traffic Monitoring**: Real-time bandwidth and packet statistics
- **Content Management**: Store and retrieve tunnel configurations

### Security Features
- End-to-end encryption for all traffic
- Perfect forward secrecy
- Resistance to key compromise impersonation
- Built-in DoS protection
- Minimal attack surface
- Automatic key generation for tunnels
- **Key rotation system** for enhanced security

### Key Rotation
WireGuard supports comprehensive key rotation for enhanced security:

```bash
# Rotate keys immediately
vrooli resource wireguard rotate keys my-tunnel

# Schedule automatic rotation (every 30 days)
vrooli resource wireguard rotate schedule my-tunnel --interval 30d

# Check rotation status and history
vrooli resource wireguard rotate status
```

**Key Rotation Features:**
- Automatic backup of old configurations
- Scheduled rotations with configurable intervals
- Rotation history tracking
- Seamless key replacement without downtime

### Monitoring & Statistics
- Per-interface traffic statistics
- Real-time bandwidth monitoring
- Packet counts and error tracking
- Container resource usage metrics
- Peer connection status and handshake timing

## Configuration

### Environment Variables
```bash
WIREGUARD_PORT=51820           # UDP port for connections
WIREGUARD_NETWORK=10.13.13.0/24  # Internal VPN network
WIREGUARD_DNS=1.1.1.1,8.8.8.8    # DNS servers for clients
WIREGUARD_KEEPALIVE=25           # Keepalive interval (seconds)
```

### Creating Tunnels

1. **Generate a tunnel configuration**:
```bash
vrooli resource wireguard content add site-to-site
```

2. **Add peers to the configuration**:
```bash
# On peer machine, generate keys
wg genkey | tee private.key | wg pubkey > public.key

# Add peer public key to server config
# Edit ~/.vrooli/resources/wireguard/wg_site-to-site.conf
```

3. **Activate the tunnel**:
```bash
vrooli resource wireguard content execute site-to-site
```

## Usage Examples

### Remote Access VPN
```bash
# Create remote access tunnel
vrooli resource wireguard content add remote-access

# Configure for remote users
# Each user gets unique keys and IP
```

### Site-to-Site VPN
```bash
# Connect two Vrooli deployments
vrooli resource wireguard content add datacenter-link

# Configure both endpoints
# Exchange public keys between sites
```

### Container Network Isolation

WireGuard now provides Docker-based network isolation for enhanced container security:

```bash
# Create an isolated network
vrooli resource wireguard namespace create secure-app

# Create network with WireGuard tunnel routing
vrooli resource wireguard namespace create prod-net site-tunnel

# List isolated networks
vrooli resource wireguard namespace list

# Check network status
vrooli resource wireguard namespace status secure-app

# Connect container to isolated network
vrooli resource wireguard namespace connect my-container secure-app

# Run new container in isolated network
docker run --network=wg-secure-app nginx

# Delete network (must be empty)
vrooli resource wireguard namespace delete secure-app
```

**Benefits of Network Isolation:**
- Complete container-to-container isolation
- Optional WireGuard tunnel routing for secure traffic
- Automatic subnet allocation to prevent conflicts
- Easy container attachment to isolated networks
- Full Docker compatibility

## Testing

```bash
# Run all tests
vrooli resource wireguard test all

# Quick health check
vrooli resource wireguard test smoke

# Integration tests
vrooli resource wireguard test integration

# Unit tests
vrooli resource wireguard test unit
```

## Monitoring

```bash
# View status and active tunnels
vrooli resource wireguard status

# View detailed traffic statistics
vrooli resource wireguard stats

# View logs
vrooli resource wireguard logs --tail 50

# Check tunnel statistics
docker exec vrooli-wireguard wg show

# Monitor bandwidth usage in real-time
# Run stats command multiple times to see bandwidth rates
vrooli resource wireguard stats
```

## Troubleshooting

### Common Issues

**Port already in use**:
```bash
# Check what's using port 51820
sudo ss -tulpn | grep 51820

# Use alternative port
export WIREGUARD_PORT=51821
vrooli resource wireguard manage restart
```

**No connectivity through tunnel**:
```bash
# Check firewall rules
sudo iptables -L -n | grep 51820

# Verify IP forwarding
sysctl net.ipv4.ip_forward

# Check WireGuard interface
docker exec vrooli-wireguard wg show
```

**Key exchange failures**:
```bash
# Regenerate keys
wg genkey | tee new-private.key | wg pubkey > new-public.key

# Update configurations with new keys
```

## Performance Tuning

### Optimize for throughput:
```bash
# Increase MTU for local networks
export WIREGUARD_MTU=1420

# Adjust keepalive for stable connections
export WIREGUARD_KEEPALIVE=60
```

### Optimize for low latency:
```bash
# Decrease keepalive interval
export WIREGUARD_KEEPALIVE=10

# Use closer DNS servers
export WIREGUARD_DNS=192.168.1.1
```

## Key Rotation

WireGuard now includes a comprehensive key rotation system for enhanced security:

### Manual Key Rotation
```bash
# Rotate keys for a specific tunnel immediately
vrooli resource wireguard rotate keys my-tunnel

# Rotate without backup (not recommended)
vrooli resource wireguard rotate keys my-tunnel --no-backup
```

### Scheduled Key Rotation
```bash
# Schedule automatic rotation every 30 days
vrooli resource wireguard rotate schedule my-tunnel --interval 30d

# Schedule with different intervals
vrooli resource wireguard rotate schedule my-tunnel --interval 7d   # Weekly
vrooli resource wireguard rotate schedule my-tunnel --interval 720h # 30 days in hours
```

### Rotation Status
```bash
# Check rotation history and schedules
vrooli resource wireguard rotate status
```

Key rotation automatically:
- Backs up old configurations
- Generates new key pairs
- Preserves peer configurations
- Tracks rotation history
- Provides fallback mechanisms

## Security Best Practices

1. **Regular key rotation**: Use the rotation scheduler for automatic key updates
2. **Limit allowed IPs**: Restrict peer access to required subnets only
3. **Monitor logs**: Check for unauthorized connection attempts
4. **Firewall rules**: Only allow WireGuard port through firewall
5. **Secure key storage**: Store private keys in Vault resource when available

## Integration with Other Resources

### With Vault
```bash
# Store WireGuard keys securely
vrooli resource vault content add wireguard/keys/server-private
```

### With monitoring resources
```bash
# Export metrics for monitoring
docker exec vrooli-wireguard wg show all dump
```

## API Reference

See `vrooli resource wireguard help` for complete command reference.

## License

WireGuard® is a registered trademark of Jason A. Donenfeld.