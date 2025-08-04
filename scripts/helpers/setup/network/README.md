# Network Diagnostics Modules

This directory contains modular components for network diagnostics functionality.

## Structure

- **`network_diagnostics_core.sh`** - Core network testing functions (ping, DNS, TCP, HTTPS)
- **`network_diagnostics_tcp.sh`** - TCP-specific optimizations (TSO, MTU, ECN, PMTU)
- **`network_diagnostics_analysis.sh`** - Advanced analysis (TLS handshake, IPv4/IPv6, time sync)
- **`network_diagnostics_fixes.sh`** - Network issue fixes (DNS, IPv6, UFW, host overrides)
- **`network_diagnostics_modular.sh`** - Integrated modular version combining all components

## Usage

The main entry point is `../network_diagnostics.sh` which loads these modules and provides backward compatibility with existing code.

## Testing

- **`network_diagnostics_core.bats`** - Tests for core functionality

The main test file `../network_diagnostics.bats` tests the integrated functionality.

## Architecture

```
network_diagnostics.sh (main wrapper)
    ├── network/network_diagnostics_core.sh (essential tests)
    ├── network/network_diagnostics_tcp.sh (TCP optimizations)
    ├── network/network_diagnostics_analysis.sh (detailed analysis)
    └── network/network_diagnostics_fixes.sh (automatic fixes)
```

This modular design reduces the main script from 2,198 lines to 284 lines while maintaining full backward compatibility.