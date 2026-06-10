# SearXNG Resource - Known Problems and Solutions

## Operational Notes (2026-06-10 overhaul)

### Image pin policy: deliberate bumps, never "set and forget"
SearXNG is a rolling-release project whose engine scrapers rot fast. The previous pin
(`searxng/searxng:2025.1.31`) sat for ~17 months and degraded the instance to
bing-only (google rate-suspended since an upstream-fixed `arc_id` breakage, duckduckgo
CAPTCHA'd pending an upstream engine overhaul, wikidata 403'd pending an upstream
UA-policy fix). Policy now:
- Pin a dated tag from `ghcr.io/searxng/searxng` (GHCR over Docker Hub to dodge pull
  rate limits). The pin lives in THREE places that must stay in sync:
  `resource.json` (runtime.image — the authoritative one for `vrooli resource start`),
  `config/defaults.sh` (SEARXNG_CUSTOM_IMAGE fallback), `docker/docker-compose.yml`.
- `settings.yml.template` stays a minimal `use_default_settings: true` overlay so
  upstream engine fixes flow in on every bump.
- The integration suite asserts >=2 distinct engines respond (test 13); run
  `resource-searxng engine-health --json` to diagnose engine-level degradation.
- Engine suspensions are tuned down (`search.suspended_times`: 1800s/1800s/900s vs
  upstream's 86400s/86400s/3600s) so transient 403s self-heal in minutes, not a day.

### Engine decisions (2026-06)
- **qwant: permanently disabled.** Datadome bot protection; upstream-confirmed
  unfixable (searxng issue #3929) and disabled by upstream default.
- **bing: disabled.** Keyword-matching navigational junk (searxng issue #4964);
  upstream disabled it by default too. Revisit only if result volume drops.
- Enabled set: google, duckduckgo, brave, startpage, mojeek, wikipedia, wikidata.

### Migration notes from the 2025.1.31 → 2026.6.8 bump
- uWSGI → **Granian** (since 2025.7): `uwsgi.ini` is dead weight and is no longer
  generated; listen host/port come from GRANIAN_* env baked into the image
  (settings.yml `server.port`/`bind_address` are ignored).
- `redis:` settings key was renamed `valkey:` upstream. We sidestep it entirely: the
  limiter is now default-OFF (`SEARXNG_LIMITER_ENABLED=no`) since it requires a
  valkey backend and adds nothing for a single-user local instance; the `redis`
  resource dependency was dropped from resource.json.
- The image expects a `/var/cache/searxng` data volume (faviconcache/engine cache);
  resource.json now binds `${RESOURCE_DATA_DIR}` there.
- `~/.searxng` is a LEGACY config dir from the old shell-managed container. The
  platform docker-service driver mounts `${RESOURCE_CONFIG_DIR}` =
  `~/.config/vrooli/resources/searxng`. Config regeneration
  (lib/config.sh `searxng::generate_config`) writes to the XDG path by default.
- `vrooli resource start` does NOT regenerate settings.yml — it reuses an existing
  container, or `docker run`s the resource.json runtime spec against whatever config
  is on disk. After editing `config/settings.yml.template`, regenerate explicitly
  (source config/defaults.sh + lib/config.sh, call `searxng::generate_config`) and
  restart the container.
- KNOWN DELTA: a fresh platform-driver `docker run` binds the host port on
  `0.0.0.0` (the resource manifest has no bind-address field), whereas the legacy
  shell path bound `127.0.0.1`. Acceptable on a firewalled dev host; revisit if the
  manifest grows a bind-address field.

## Resolved Issues

### 0. Recent Fixes (2025-09-14)

#### Benchmark Command Parameter Parsing
**Problem**: The benchmark command failed when using --iterations parameter.
**Symptoms**: `searxng::benchmark: line 748: iterations: unbound variable`
**Root Cause**: CLI framework consumed parameters before they reached the benchmark function.
**Solution**: Created a wrapper function `searxng::benchmark_cli()` that properly parses CLI arguments and passes them to the actual benchmark function.

#### Advanced Search Time Range Parameter
**Problem**: The advanced-search command didn't recognize --time-range parameter.
**Symptoms**: `[ERROR] Unknown option: --time-range`
**Root Cause**: Only --time was accepted, not the more intuitive --time-range.
**Solution**: Updated parameter parsing to accept both --time and --time-range for better user experience.

#### Redis Readonly Variable Error
**Problem**: Enabling/disabling Redis threw readonly variable error.
**Symptoms**: `/lib/docker.sh: line 296: SEARXNG_ENABLE_REDIS: readonly variable`
**Root Cause**: Attempting to export a variable that was declared as readonly in defaults.sh.
**Solution**: Removed the export statements and just logged the status changes instead.

## Previously Resolved Issues

### 1. Benchmark Loop Execution Failure
**Problem**: The benchmark command would hang when executed due to bash C-style for loop incompatibility with `set -euo pipefail`.
**Symptoms**: Command would print "Running benchmark..." and then hang indefinitely.
**Root Cause**: The syntax `for ((i=0; i<num_queries; i++))` doesn't work reliably with strict error handling enabled.
**Solution**: Replaced C-style for loop with while loop and used safe arithmetic operations.
```bash
# Before (problematic)
for ((i=0; i<num_queries; i++)); do

# After (working)
local i=0
while [[ $i -lt $num_queries ]]; do
    # loop body
    ((i++)) || true
done
```

### 2. Integration Test JSON Parsing Timeouts
**Problem**: Integration tests would hang when parsing JSON responses with jq.
**Symptoms**: Test would hang at "Testing JSON search API with multiple queries..." and never complete.
**Root Cause**: 
1. Large JSON responses could cause jq to hang
2. Arithmetic operations with `((var++))` would fail with strict error handling
3. Passing JSON through bash -c with single quotes could fail with JSON containing single quotes

**Solution**: 
1. Added timeout to jq commands to prevent hanging
2. Replaced all `((var++))` with `var=$((var + 1))` 
3. Removed unnecessary bash -c wrapper for jq operations

### 3. Arithmetic Operations Failing Silently
**Problem**: Increment operations like `((successful_queries++))` would cause script to exit when the value was 0.
**Symptoms**: Tests would exit unexpectedly without error messages.
**Root Cause**: With `set -e`, arithmetic operations that evaluate to 0 are treated as failures.
**Solution**: Used explicit arithmetic assignment or added `|| true` to ignore the exit code.
```bash
# Before
((successful_queries++))

# After (option 1)
successful_queries=$((successful_queries + 1))

# After (option 2)
((successful_queries++)) || true
```

## Best Practices Learned

1. **Always use timeouts with external commands**: Especially for JSON parsing operations
2. **Avoid C-style for loops in bash with strict error handling**: Use while loops instead
3. **Be careful with arithmetic in bash**: Increment operations can fail unexpectedly
4. **Test with verbose output**: Helps identify where scripts hang
5. **Use simple fallback parsing**: When jq fails or times out, have grep-based alternatives

## Testing Commands

To verify all issues are resolved:
```bash
# Test benchmark functionality
vrooli resource searxng content benchmark 5

# Test integration tests
vrooli resource searxng test integration

# Run full test suite
vrooli resource searxng test all
```

All tests should complete without hanging and show passing results.