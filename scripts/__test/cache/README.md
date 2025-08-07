# Shell Test Caching System

## Overview
The shell test caching system speeds up test runs by skipping tests that have recently passed and haven't changed. This is particularly useful during development when you're focusing on fixing specific failing tests.

## Features
- **Smart Caching**: Tracks test results with file checksums to detect changes
- **Configurable TTL**: Set how long test results remain valid (default: 2 hours)
- **Time Tracking**: Shows estimated time saved by skipping cached tests
- **Cache Management**: Clear, inspect, and manage cached results

## Usage

### Basic Usage
```bash
# Run tests normally (caching enabled by default)
pnpm run test:shell

# Force run all tests, ignoring cache
pnpm run test:shell -- --force

# Disable cache for this run
pnpm run test:shell -- --no-cache

# Preview what would be skipped
pnpm run test:shell -- --cache-only

# Clear cache before running
pnpm run test:shell -- --clear-cache
```

### Cache Management
```bash
# View cache statistics
./scripts/__test/cache/test-cache.sh stats

# List cached test results
./scripts/__test/cache/test-cache.sh list          # All entries
./scripts/__test/cache/test-cache.sh list passed   # Only passed tests
./scripts/__test/cache/test-cache.sh list failed   # Only failed tests
./scripts/__test/cache/test-cache.sh list valid    # Only valid (non-expired) entries
./scripts/__test/cache/test-cache.sh list expired  # Only expired entries

# Clear cache
./scripts/__test/cache/test-cache.sh clear         # Clear all
./scripts/__test/cache/test-cache.sh clear "*.bats" # Clear matching pattern
```

### Configuration
Environment variables to control caching behavior:

```bash
# Set cache TTL to 60 minutes (default: 120)
export VROOLI_TEST_CACHE_TTL=60
pnpm run test:shell

# Disable caching globally
export VROOLI_TEST_CACHE_ENABLED=false
pnpm run test:shell

# Force run all tests
export VROOLI_TEST_FORCE_RUN=true
pnpm run test:shell
```

### Command-Line Options
- `--no-cache`: Disable caching for this run
- `--cache-ttl N`: Set cache TTL in minutes
- `--force`: Force run all tests, ignoring cache
- `--cache-only`: Preview what would be skipped (dry run)
- `--clear-cache`: Clear cache before running

## How It Works

1. **Before Each Test**: The system checks if:
   - A cached result exists for the test file
   - The test previously passed
   - The file hasn't changed (via checksum)
   - The cache entry isn't expired

2. **Skip or Run**:
   - If all conditions are met, the test is skipped
   - Otherwise, the test runs normally

3. **After Each Test**:
   - Test results are stored with timestamp and file checksum
   - Both passed and failed tests are cached

4. **Cache Location**:
   - Cache is stored in `~/.cache/vrooli/test-results/shell-tests.json`
   - Each entry contains: checksum, timestamp, result, duration

## Benefits

- **Faster Development**: Focus on failing tests without re-running passed ones
- **Time Savings**: Skip 100+ tests in seconds when working on specific failures
- **Smart Invalidation**: Automatically re-runs tests when files change
- **Transparency**: Always shows what's being skipped and time saved

## Examples

### Typical Development Workflow
```bash
# Initial run - all tests execute
pnpm run test:shell
# Output: 150 tests, 148 passed, 2 failed

# Fix one failing test, re-run
pnpm run test:shell
# Output: 150 tests, 1 passed, 1 failed, 148 skipped (cached)
# Time saved: ~296s

# Fix the last failure
pnpm run test:shell
# Output: 150 tests, 1 passed, 0 failed, 149 skipped (cached)
# Time saved: ~298s

# Verify all tests still pass
pnpm run test:shell -- --force
# Output: 150 tests, 150 passed, 0 failed
```

### Cache Analysis
```bash
# See what would be skipped without running tests
pnpm run test:shell -- --cache-only --verbose

# Check cache statistics
./scripts/__test/cache/test-cache.sh stats
# Output:
# Cache file: ~/.cache/vrooli/test-results/shell-tests.json
# Cache TTL: 120 minutes
# Total entries: 150
# Valid entries: 148
# Expired entries: 2
# Time saved (estimated): 296s
```

## Troubleshooting

### Tests Not Being Skipped
- Check if cache is enabled: `echo $VROOLI_TEST_CACHE_ENABLED`
- Verify cache TTL: `echo $VROOLI_TEST_CACHE_TTL`
- Check if files have changed (cache uses checksums)
- View cache stats: `./scripts/__test/cache/test-cache.sh stats`

### Clear Stale Cache
```bash
# Clear all cache
./scripts/__test/cache/test-cache.sh clear

# Clear specific tests
./scripts/__test/cache/test-cache.sh clear "permissions"
```

### Force Fresh Run
```bash
# One-time force run
pnpm run test:shell -- --force

# Or clear cache first
pnpm run test:shell -- --clear-cache
```