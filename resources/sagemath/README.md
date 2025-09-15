# SageMath Resource

## Overview
SageMath is a comprehensive open-source mathematics software system that provides symbolic computation, numerical analysis, algebra, calculus, number theory, and scientific computing capabilities.

## Features
- **Symbolic Mathematics**: Algebraic manipulation, calculus, differential equations
- **Numerical Computing**: Linear algebra, optimization, statistics  
- **Graph Theory**: Network analysis, combinatorics
- **Number Theory**: Cryptography, prime numbers, factorization
- **Visualization**: 2D/3D plotting, animations, interactive graphics
- **Export Capabilities**: LaTeX, MathML, and PNG image export for equations
- **GPU Acceleration**: CUDA support for numerical computations
- **Parallel Computing**: Multi-core distributed processing
- **v2.0 Contract Compliant**: Full CLI standardization with manage/test/content groups
- **Health Monitoring**: JSON health endpoint with component status
- **Extended Math CLI**: Specialized commands for common mathematical operations
- **Result Caching**: Automatic caching of computation results for performance

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

### Export Capabilities
```bash
# Export to LaTeX format
vrooli resource sagemath export latex "solve(x^2 - 4 == 0, x)"
vrooli resource sagemath export latex "integrate(sin(x), x, 0, pi)" output.tex

# Export to MathML format (web-friendly)
vrooli resource sagemath export mathml "x^2 + y^2"

# Render equation to PNG image
vrooli resource sagemath export image "sum(1/n^2, n, 1, infinity)" equation.png

# Export to all formats at once
vrooli resource sagemath export all "factorial(10)" factorial_outputs

# List available export formats
vrooli resource sagemath export formats
```

### GPU Acceleration
```bash
# Check GPU availability
vrooli resource sagemath gpu check

# Enable GPU acceleration in container
vrooli resource sagemath gpu enable

# Run GPU-accelerated computation
vrooli resource sagemath gpu compute "matrix_multiply(A, B)"

# Benchmark GPU vs CPU performance
vrooli resource sagemath gpu benchmark
```

### Parallel Computing
```bash
# Check parallel computing capabilities
vrooli resource sagemath parallel status

# Run parallel computation with 8 cores
vrooli resource sagemath parallel compute "@parallel\ndef f(n): return is_prime(n)" 8

# Test parallel processing
vrooli resource sagemath test parallel
```

### Visualization & Plotting
```bash
# Create 2D plots
vrooli resource sagemath plot 2d "sin(x) + cos(2*x)" -pi pi "Trigonometric Functions"
vrooli resource sagemath plot 2d "x^3 - 3*x" -3 3 "Cubic Function"

# Create 3D plots
vrooli resource sagemath plot 3d "x^2 + y^2" "(-3,3)" "(-3,3)" "Paraboloid"
vrooli resource sagemath plot 3d "sin(x)*cos(y)" "(-pi,pi)" "(-pi,pi)" "Wave Surface"

# Create parametric plots
vrooli resource sagemath plot parametric "cos(t)" "sin(t)" 0 "2*pi" "Circle"
vrooli resource sagemath plot parametric "t*cos(t)" "t*sin(t)" 0 "6*pi" "Spiral"

# Create polar plots
vrooli resource sagemath plot polar "1 + cos(theta)" 0 "2*pi" "Cardioid"
vrooli resource sagemath plot polar "sin(3*theta)" 0 "2*pi" "Rose Curve"
```

### Cache Management
```bash
# View cache statistics
vrooli resource sagemath cache stats

# Clear all cached results
vrooli resource sagemath cache clear
```

### Testing
```bash
# Run tests
vrooli resource sagemath test smoke       # Quick health check
vrooli resource sagemath test integration # Full functionality test
vrooli resource sagemath test parallel     # Parallel computing test
vrooli resource sagemath test performance  # Performance benchmarks
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