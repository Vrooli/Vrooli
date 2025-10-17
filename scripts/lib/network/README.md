# Network Utilities and Diagnostics

This directory contains network-related utilities and comprehensive diagnostic tools for the Vrooli platform.

## Directory Structure

```
/scripts/lib/network/
├── authorize_key.sh          # SSH key authorization utilities
├── enableSsh.sh              # SSH configuration and password authentication
├── firewall.sh               # Firewall configuration and management
├── keyless_ssh.sh            # Keyless SSH setup and configuration
├── diagnostics/              # Network diagnostic modules
│   ├── network_diagnostics.sh           # Main orchestrator that loads all modules
│   ├── network_diagnostics_core.sh      # Core testing functions (ping, DNS, TCP, HTTPS)
│   ├── network_diagnostics_tcp.sh       # TCP optimizations (TSO, MTU, ECN, PMTU)
│   ├── network_diagnostics_analysis.sh  # Advanced analysis (TLS, IPv4/IPv6, time sync)
│   ├── network_diagnostics_fixes.sh     # Automatic network issue fixes
│   ├── network_diagnostics_modular.sh   # Integrated modular version
│   ├── network_diagnostics_core.bats    # Core functionality tests
│   └── network_diagnostics.bats         # Integration tests
└── README.md                 # This file
```

## Components

### Network Utilities
- **`authorize_key.sh`** - Manages SSH key authorization and deployment
- **`enableSsh.sh`** - Configures SSH daemon for password and key authentication
- **`firewall.sh`** - Handles firewall rules and UFW configuration
- **`keyless_ssh.sh`** - Sets up passwordless SSH access between nodes

### Diagnostic Modules
Located in the `diagnostics/` subdirectory:
- **`network_diagnostics.sh`** - Main orchestrator script that loads and coordinates all diagnostic modules
- **`network_diagnostics_core.sh`** - Essential network tests including connectivity, DNS resolution, and basic protocol checks
- **`network_diagnostics_tcp.sh`** - TCP-specific optimizations and performance tuning
- **`network_diagnostics_analysis.sh`** - Deep protocol analysis including TLS handshakes and dual-stack connectivity
- **`network_diagnostics_fixes.sh`** - Automated remediation for common network issues
- **`network_diagnostics_modular.sh`** - Alternative integrated modular version with all functionality in one file

## Usage

### Network Utilities
Each utility script can be sourced or executed directly:
```bash
# Enable SSH password authentication
./enableSsh.sh

# Configure firewall rules
./firewall.sh

# Setup keyless SSH
./keyless_ssh.sh
```

### Network Diagnostics
The diagnostic modules can be accessed through multiple entry points:
```bash
# Run diagnostics through the main orchestrator
/scripts/lib/network/diagnostics/network_diagnostics.sh

# Or source the modular version directly
source /scripts/lib/network/diagnostics/network_diagnostics_modular.sh

# Alternative entry point from app lifecycle
/scripts/app/lifecycle/setup/network_diagnostics.sh
```

## Testing

Test files are located alongside their corresponding modules:
- **`diagnostics/network_diagnostics_core.bats`** - Unit tests for core functionality
- **`diagnostics/network_diagnostics.bats`** - Integration tests for the complete diagnostic suite

Run tests using:
```bash
bats diagnostics/network_diagnostics_core.bats
bats diagnostics/network_diagnostics.bats
```

## Architecture

The modular design separates concerns while maintaining backward compatibility:

```
Entry Points:
├── /scripts/lib/network/diagnostics/network_diagnostics.sh    (Main orchestrator)
└── /scripts/app/lifecycle/setup/network_diagnostics.sh        (Lifecycle integration)
    
    Both load modules from /scripts/lib/network/diagnostics/:
    ├── network_diagnostics_core.sh      (Layer 1: Basic connectivity)
    ├── network_diagnostics_tcp.sh       (Layer 2: TCP optimization)
    ├── network_diagnostics_analysis.sh  (Layer 3: Protocol analysis)
    └── network_diagnostics_fixes.sh     (Layer 4: Auto-remediation)

Alternative: network_diagnostics_modular.sh (All-in-one integrated version)
```

This modular approach:
- Reduces the monolithic script from 2,198 lines to manageable modules
- Enables selective loading of functionality
- Maintains full backward compatibility
- Improves testability and maintainability

## Dependencies

All scripts depend on common utilities from `/scripts/lib/utils/`:
- `log.sh` - Logging functions
- `flow.sh` - Flow control and sudo management
- `exit_codes.sh` - Standardized exit codes
- `var.sh` - Variable utilities

## Notes

- Scripts respect `SUDO_MODE` environment variable for privilege escalation
- Network diagnostics can run in standalone mode without sudo access
- All scripts are POSIX-compliant for maximum portability