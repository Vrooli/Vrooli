# SageMath Resource

## Overview
SageMath is a comprehensive open-source mathematics software system that provides symbolic computation, numerical analysis, algebra, calculus, number theory, and scientific computing capabilities.

## Features
- **Symbolic Mathematics**: Algebraic manipulation, calculus, differential equations
- **Numerical Computing**: Linear algebra, optimization, statistics  
- **Graph Theory**: Network analysis, combinatorics
- **Number Theory**: Cryptography, prime numbers, factorization
- **Visualization**: 2D/3D plotting, animations, interactive graphics
- **v2.0 Contract Compliant**: Full CLI standardization with manage/test/content groups
- **Health Monitoring**: JSON health endpoint with component status
- **Extended Math CLI**: Specialized commands for common mathematical operations

## Architecture
- Docker-based deployment using official SageMath images
- Python API for programmatic access
- Jupyter notebook integration for interactive sessions
- REST API via SageMath Cell Server

## Usage

### Lifecycle Management
```bash
# Install and start SageMath
vrooli resource sagemath manage install
vrooli resource sagemath manage start --wait

# Check status and health
vrooli resource sagemath status
vrooli resource sagemath health

# View logs
vrooli resource sagemath logs

# Stop and restart
vrooli resource sagemath manage stop
vrooli resource sagemath manage restart
```

### Mathematical Operations
```bash
# Direct calculations
vrooli resource sagemath content calculate "solve(x^2 - 4 == 0, x)"
vrooli resource sagemath content calculate "integrate(sin(x), x, 0, pi)"
vrooli resource sagemath content calculate "factor(2025)"

# Specialized operations
vrooli resource sagemath solve "x^3 - 1 == 0" x
vrooli resource sagemath differentiate "sin(x) * cos(x)" x
vrooli resource sagemath integrate "x^2" x 0 1
vrooli resource sagemath matrix det "[[1,2],[3,4]]"
vrooli resource sagemath prime check 1234567891
vrooli resource sagemath polynomial expand "(x+1)^5"
```

### Content Management
```bash
# Add and execute scripts
vrooli resource sagemath content add --file my_calculation.sage
vrooli resource sagemath content execute my_calculation.sage
vrooli resource sagemath content list
vrooli resource sagemath content get my_calculation.sage
vrooli resource sagemath content remove --name my_calculation.sage

# Work with notebooks
vrooli resource sagemath content add --file analysis.ipynb
vrooli resource sagemath content notebook  # Opens Jupyter interface
```

### Testing
```bash
# Run tests
vrooli resource sagemath test smoke       # Quick health check
vrooli resource sagemath test integration # Full functionality test
vrooli resource sagemath test all         # Complete test suite
```

## Integration with Vrooli
SageMath enables scenarios requiring advanced mathematical computation:
- Research automation with symbolic algebra
- Engineering calculations for circuit design
- Financial modeling and risk analysis
- Scientific simulations and data analysis
- Cryptographic protocol verification

## Resource Configuration
- Default port: 8888 (Jupyter interface)
- Memory: 2GB minimum, 4GB recommended
- Storage: 5GB for base installation
- CPU: 2 cores recommended

## See Also
- [Installation Guide](docs/installation.md)
- [API Reference](docs/api.md)
- [Examples](examples/)
- [Integration Patterns](docs/integration.md)