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

- **Lean 4 Runtime**: Complete Lean 4.23.0 compiler and Lake build system with elan version manager
- **Mathlib4 Library**: Comprehensive mathematical formalization library installation
- **Proof Verification API**: REST endpoints for submitting and verifying mathematical proofs
- **Cache Management**: LRU cache for compiled proofs with hit/miss metrics and configurable size
- **Batch Processing**: Process multiple proofs in parallel with job tracking
- **Performance Metrics**: Real-time statistics on proof verification times and cache efficiency
- **Interactive REPL**: Interactive theorem proving with step-by-step proof development
- **Custom Tactics**: Load, manage, and test custom proof tactics
- **Content Management**: Add, list, execute, and manage proof files
- **Health Monitoring**: Robust health checks with timeout handling and Lean version detection
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

### Proof Verification

```bash
# Add a proof file to the resource
vrooli resource mathlib content add myproof.lean

# List available proofs and tactics
vrooli resource mathlib content list

# Execute a proof for verification
vrooli resource mathlib content execute myproof.lean

# Get a stored proof
vrooli resource mathlib content get myproof

# Remove a proof
vrooli resource mathlib content remove myproof
```

### API Usage

```bash
# Submit single proof
curl -X POST http://localhost:11458/prove \
  -H "Content-Type: application/json" \
  -d '{"proof": "theorem my_theorem : 2 + 2 = 4 := rfl"}'

# Submit batch of proofs
curl -X POST http://localhost:11458/batch \
  -H "Content-Type: application/json" \
  -d '{"proofs": [
    {"name": "theorem1", "proof": "theorem add_comm (a b : Nat) : a + b = b + a := sorry"},
    {"name": "theorem2", "proof": "theorem mul_comm (a b : Nat) : a * b = b * a := sorry"}
  ]}'

# Check proof status
curl http://localhost:11458/status/{job_id}

# Check batch status
curl http://localhost:11458/batch/status/{batch_id}

# Get available tactics
curl http://localhost:11458/tactics

# View performance metrics
curl http://localhost:11458/metrics
```

### Interactive REPL

```bash
# Start interactive theorem proving session
vrooli resource mathlib interactive start

# Replay a proof file step-by-step
vrooli resource mathlib interactive replay proof.lean

# Get REPL help
vrooli resource mathlib interactive help
```

REPL commands:
- `:help` - Show help
- `:tactics` - List available tactics
- `:check <expr>` - Check type of expression
- `:eval <expr>` - Evaluate expression
- `:quit` - Exit REPL
- `:clear` - Clear screen

### Custom Tactics Management

```bash
# Load a custom tactic file
vrooli resource mathlib content tactics load my_tactic.lean

# List loaded custom tactics
vrooli resource mathlib content tactics list

# Test a custom tactic
vrooli resource mathlib content tactics test my_tactic "theorem test : 2 + 2 = 4 := by my_tactic"

# Remove a custom tactic
vrooli resource mathlib content tactics remove my_tactic

# Export tactics to bundle
vrooli resource mathlib content tactics export tactics.tar.gz

# Import tactics from bundle
vrooli resource mathlib content tactics import tactics.tar.gz
```

## Configuration

### Environment Variables

- `MATHLIB_PORT`: API port (default: 11458)
- `MATHLIB_WORK_DIR`: Working directory for proofs (default: /tmp/mathlib)
- `MATHLIB_TIMEOUT`: Proof verification timeout in seconds (default: 30)
- `MATHLIB_CACHE_DIR`: Cache directory for compiled modules (default: ~/.cache/mathlib)
- `MATHLIB_ENABLE_CACHE`: Enable proof caching (default: true)
- `MATHLIB_MAX_MEMORY`: Maximum memory usage in MB (default: 4096)
- `MATHLIB_MAX_THREADS`: Maximum parallel proof threads (default: 4)

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

Current implementation provides:
- ✅ v2.0 contract structure
- ✅ Complete lifecycle management  
- ✅ Lean 4.23.0 installed via elan
- ✅ Mathlib4 integration with Lake
- ✅ Proof verification API with async job processing
- ✅ Batch processing for multiple proofs
- ✅ Cache management with LRU eviction and metrics
- ✅ Performance monitoring and statistics
- ✅ Content management for proof files
- ✅ Error diagnostics and detailed reporting
- ⏳ Interactive REPL mode (future)
- ⏳ Custom tactics loading (future)

See [PRD.md](./PRD.md) for detailed requirements and progress tracking.