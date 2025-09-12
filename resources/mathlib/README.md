# Mathlib (Lean) Resource

Mathlib4 integration for formal mathematical theorem proving and verification using Lean 4.

## Overview

This resource provides access to the Lean 4 theorem prover with the complete Mathlib4 mathematical library, enabling formal verification of mathematical proofs and theorems. It powers research automation, educational tools, and ensures absolute correctness in mathematical reasoning.

## Quick Start

```bash
# Install the resource
vrooli resource mathlib manage install

# Start the service
vrooli resource mathlib manage start

# Check health
vrooli resource mathlib status

# Run tests
vrooli resource mathlib test smoke
```

## Key Features

- **Lean 4 Runtime**: Complete Lean 4 compiler and Lake build system
- **Mathlib4 Library**: Comprehensive mathematical formalization library
- **Proof Verification**: Execute and verify mathematical proofs
- **Health Monitoring**: Robust health checks with timeout handling
- **v2.0 Compliant**: Full universal resource contract implementation

## Usage Examples

### Basic Commands

```bash
# Get help
vrooli resource mathlib help

# Show configuration
vrooli resource mathlib info

# Manage lifecycle
vrooli resource mathlib manage start
vrooli resource mathlib manage stop
vrooli resource mathlib manage restart

# Run tests
vrooli resource mathlib test all
```

### Proof Verification (Future)

```bash
# Submit a proof for verification
vrooli resource mathlib content execute proof.lean

# List available tactics
vrooli resource mathlib content list
```

## Configuration

### Environment Variables

- `MATHLIB_PORT`: API port (default: 11458)
- `MATHLIB_WORK_DIR`: Working directory for proofs (default: /tmp/mathlib)
- `MATHLIB_TIMEOUT`: Proof verification timeout in seconds (default: 30)
- `MATHLIB_CACHE_DIR`: Cache directory for compiled modules

### Resource Information

- **Port**: 11458
- **Startup Time**: 30-60 seconds
- **Memory Requirements**: 4-8GB
- **Disk Requirements**: 10GB for full Mathlib

## Technical Details

### Architecture
- Lean 4 compiler with Lake build system
- Mathlib4 mathematical library
- REST API for proof submission
- Compiled module cache for performance

### Dependencies
- Linux x86_64
- Git
- Python 3.8+
- Minimum 8GB RAM

## Troubleshooting

### Common Issues

1. **Slow startup**: First run compiles Mathlib modules (can take 5-10 minutes)
2. **Memory errors**: Ensure at least 8GB RAM available
3. **Compilation failures**: Check disk space (needs 10GB free)

## Integration with Scenarios

This resource enables scenarios to:
- Formally verify mathematical proofs
- Build proof assistants and educational tools
- Automate mathematical research workflows
- Ensure correctness in scientific computations

## Development Status

Current implementation provides basic scaffolding with:
- ✅ v2.0 contract structure
- ✅ Basic lifecycle management
- ✅ Health monitoring
- ⏳ Lean 4 installation (minimal)
- ⏳ Mathlib4 integration (pending)
- ⏳ Proof verification API (future)

See [PRD.md](./PRD.md) for detailed requirements and progress tracking.