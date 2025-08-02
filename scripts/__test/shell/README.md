# Test Shell Scripts

This directory contains shell scripts for running and managing tests in the Vrooli project.

## Overview

The test infrastructure is divided into three main components:
- **Shell scripts** (this directory) - Test runners and utilities
- **Fixtures** (`../fixtures/`) - BATS testing infrastructure with mocks
- **Resources** (`../resources/`) - Integration testing framework

## Directory Structure

```
shell/
├── core/               # Active test infrastructure
│   ├── run-tests.sh    # Main test runner with caching and optimization
│   ├── cache-manager.sh # Manages test cache for performance
│   ├── test-profiler.sh # Analyzes test performance
│   └── lib/            # Shared utilities and helpers
│       ├── test-helper.bash # Common test utilities
│       ├── stub.bash   # Mock/stub utilities
│       ├── stub.bats   # Tests for stub utilities
│       └── binstub     # Binary stub helper
├── archive/            # Historical scripts (DO NOT USE)
│   └── 2024-07-phase3-optimization/ # Past optimization efforts
└── helpers/            # External dependencies
    └── bats-core/      # BATS testing framework
```

## Usage

### Running Tests

Run all tests with optimizations:
```bash
./core/run-tests.sh
```

Run tests for changed files only:
```bash
./core/run-tests.sh --changed-only
```

Run tests with custom timeout:
```bash
./core/run-tests.sh --timeout 120
```

### Performance Analysis

Profile test performance:
```bash
./core/test-profiler.sh
```

Analyze specific test files:
```bash
./core/test-profiler.sh path/to/test.bats
```

### Cache Management

The test runner uses intelligent caching to improve performance:
- Caches are automatically managed by `run-tests.sh`
- Cache TTL: 24 hours for most tests
- Invalidated on: file changes, dependency updates, environment changes

To clear cache manually:
```bash
rm -rf /tmp/bats-cache-*
```

## Test Timeouts

The test runner automatically scales timeouts based on test type:
- Basic tests: 2x base timeout (120s)
- API/Model tests: 4x base timeout (240s)
- AI service tests: 5x base timeout (300s)

Base timeout: 60 seconds (configurable via `--timeout`)

## Performance History

In July 2024, a major optimization effort (Phase 3) reduced test runtime from 84+ minutes to 2-5 minutes through:
- Intelligent caching
- Parallel execution
- Resource-aware setup
- Optimized test discovery

See `archive/2024-07-phase3-optimization/README.md` for details.

## Development Guidelines

1. **Adding New Tests**: Place in appropriate resource directory, not here
2. **Modifying Test Runner**: Update `core/run-tests.sh` with care
3. **Performance Issues**: Use `test-profiler.sh` to identify bottlenecks
4. **Cache Issues**: Check cache-manager.sh logs or clear cache

## Troubleshooting

### Tests Hanging
- Check timeout settings (some AI tests need 300s+)
- Look for resource contention
- Use profiler to identify slow tests

### Cache Problems
- Clear cache directory
- Check disk space
- Verify file permissions

### Failed Tests
- Run with `--max-failures 100` to see more failures
- Check test logs in output
- Verify resource availability

## Migration Notes

This directory was reorganized in August 2025 to improve maintainability:
- Removed double underscore prefix convention
- Moved one-time fixes to archive
- Standardized naming to kebab-case
- Created clear separation between active and historical scripts

For questions or issues, see the main project documentation.