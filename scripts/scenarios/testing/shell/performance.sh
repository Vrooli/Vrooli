#!/usr/bin/env bash
# Performance validation orchestrator for phased testing
# Integrates Lighthouse, bundle size, benchmarks, response time, and load testing
set -euo pipefail

# Main entry point for performance validation
# Reads configuration from .vrooli/testing.json and runs enabled performance checks
# Usage: testing::performance::validate_all --scenario SCENARIO_NAME
testing::performance::validate_all() {
  local scenario_name=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --scenario)
        scenario_name="$2"
        shift 2
        ;;
      *)
        echo "Unknown option to testing::performance::validate_all: $1" >&2
        return 1
        ;;
    esac
  done

  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"

  # Fallback: if no testing.json, check for standalone lighthouse config
  if [ ! -f "$config_file" ]; then
    log::warning "No .vrooli/testing.json found; checking for standalone Lighthouse config"
    _performance_basic_checks
    return 0
  fi

  # Check if performance section exists (it's always enabled if present)
  local perf_exists
  perf_exists=$(jq -e '.performance' "$config_file" >/dev/null 2>&1 && echo "true" || echo "false")

  if [ "$perf_exists" != "true" ]; then
    log::info "No performance section in testing.json; checking for standalone Lighthouse config"
    _performance_basic_checks
    return 0
  fi

  local overall_status=0
  local continue_on_failure
  continue_on_failure=$(jq -r '.performance.options.continue_on_failure // true' "$config_file")

  # Run enabled performance checks in order
  # Each function returns 0 on success/skip, 1 on failure

  if ! _performance_run_lighthouse; then
    overall_status=1
    if [ "$continue_on_failure" != "true" ]; then
      log::error "Lighthouse failed; stopping performance tests (continue_on_failure=false)"
      return 1
    fi
  fi

  if ! _performance_run_bundle_size; then
    overall_status=1
    if [ "$continue_on_failure" != "true" ]; then
      log::error "Bundle size check failed; stopping performance tests"
      return 1
    fi
  fi

  if ! _performance_run_api_benchmarks; then
    overall_status=1
    if [ "$continue_on_failure" != "true" ]; then
      log::error "API benchmarks failed; stopping performance tests"
      return 1
    fi
  fi

  if ! _performance_run_response_time; then
    overall_status=1
    if [ "$continue_on_failure" != "true" ]; then
      log::error "Response time tests failed; stopping performance tests"
      return 1
    fi
  fi

  if ! _performance_run_concurrent_load; then
    overall_status=1
    if [ "$continue_on_failure" != "true" ]; then
      log::error "Concurrent load test failed; stopping performance tests"
      return 1
    fi
  fi

  if ! _performance_run_memory_usage; then
    overall_status=1
    if [ "$continue_on_failure" != "true" ]; then
      log::error "Memory usage check failed; stopping performance tests"
      return 1
    fi
  fi

  return $overall_status
}

# Run Lighthouse audits if enabled
_performance_run_lighthouse() {
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"
  local enabled
  enabled=$(jq -r '.performance.checks.lighthouse.enabled // false' "$config_file" 2>/dev/null)

  if [ "$enabled" != "true" ]; then
    log::info "📊 Lighthouse: disabled"
    return 0
  fi

  log::info "📊 Running Lighthouse audits..."

  # Convention: .vrooli/lighthouse.json, but allow override
  local lighthouse_config
  lighthouse_config=$(jq -r '.performance.checks.lighthouse.config // ".vrooli/lighthouse.json"' "$config_file")
  local lighthouse_path="${TESTING_PHASE_SCENARIO_DIR}/${lighthouse_config}"

  if [ ! -f "$lighthouse_path" ]; then
    log::warning "Lighthouse config not found: $lighthouse_path"
    log::info "To enable Lighthouse testing, create .vrooli/lighthouse.json"
    log::info "See docs/testing/guides/lighthouse-integration.md for details"
    testing::phase::add_test skipped
    return 0
  fi

  # Source and run Lighthouse runner
  source "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/lighthouse/runner.sh"

  if lighthouse::run_audits; then
    log::success "✅ Lighthouse audits passed"
    testing::phase::add_test passed
    return 0
  else
    log::error "❌ Lighthouse audits failed (see test/artifacts/lighthouse/)"
    testing::phase::add_test failed
    return 1
  fi
}

# Check bundle sizes against configured limits
_performance_run_bundle_size() {
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"
  local enabled
  enabled=$(jq -r '.performance.checks.bundle_size.enabled // false' "$config_file" 2>/dev/null)

  if [ "$enabled" != "true" ]; then
    log::info "📦 Bundle size: disabled"
    return 0
  fi

  log::info "📦 Checking bundle sizes..."

  local targets_json
  targets_json=$(jq -c '.performance.checks.bundle_size.targets // []' "$config_file")

  if [ "$targets_json" = "[]" ]; then
    log::warning "Bundle size enabled but no targets defined"
    testing::phase::add_test skipped
    return 0
  fi

  local target_count
  target_count=$(echo "$targets_json" | jq 'length')
  local failed=0
  local warnings=0

  for ((i=0; i<target_count; i++)); do
    local target
    target=$(echo "$targets_json" | jq ".[$i]")

    local name path max_size warn_size requirement
    name=$(echo "$target" | jq -r '.name')
    path=$(echo "$target" | jq -r '.path')
    max_size=$(echo "$target" | jq -r '.max_size_kb')
    warn_size=$(echo "$target" | jq -r '.warn_size_kb // (.max_size_kb * 0.8 | floor)')
    requirement=$(echo "$target" | jq -r '.requirement // ""')

    local full_path="${TESTING_PHASE_SCENARIO_DIR}/${path}"

    if [ ! -e "$full_path" ]; then
      log::warning "Bundle not found: $path (may need to build UI first)"
      testing::phase::add_warning "Bundle not built: $path"
      testing::phase::add_test skipped
      continue
    fi

    # Calculate size (excluding patterns if specified)
    local exclude_patterns
    exclude_patterns=$(echo "$target" | jq -r '.exclude_patterns[]?' 2>/dev/null || echo "")

    local size_kb
    if [ -d "$full_path" ]; then
      # Directory - sum all files, excluding patterns
      local find_cmd="find '$full_path' -type f"
      if [ -n "$exclude_patterns" ]; then
        while IFS= read -r pattern; do
          [ -n "$pattern" ] && find_cmd="$find_cmd ! -name '$pattern'"
        done <<< "$exclude_patterns"
      fi
      size_kb=$(eval "$find_cmd -exec du -k {} \;" 2>/dev/null | awk '{sum+=$1} END{print sum}' || echo "0")
    else
      # Single file
      size_kb=$(du -k "$full_path" 2>/dev/null | awk '{print $1}' || echo "0")
    fi

    # Compare against thresholds
    if [ "$size_kb" -gt "$max_size" ]; then
      log::error "❌ $name: ${size_kb}KB exceeds limit (${max_size}KB)"
      testing::phase::add_error "$name bundle too large: ${size_kb}KB > ${max_size}KB"
      [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status failed --evidence "Bundle size ${size_kb}KB > ${max_size}KB"
      ((failed++))
    elif [ "$size_kb" -gt "$warn_size" ]; then
      log::warning "⚠️  $name: ${size_kb}KB approaching limit (warn: ${warn_size}KB, max: ${max_size}KB)"
      testing::phase::add_warning "$name bundle near limit: ${size_kb}KB (warn threshold: ${warn_size}KB)"
      [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Bundle size ${size_kb}KB (warn threshold)"
      ((warnings++))
    else
      log::success "✅ $name: ${size_kb}KB (limit: ${max_size}KB)"
      [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Bundle size ${size_kb}KB"
    fi
  done

  if [ $failed -gt 0 ]; then
    testing::phase::add_test failed
    return 1
  else
    testing::phase::add_test passed
    return 0
  fi
}

# Run API benchmarks (currently supports Go)
_performance_run_api_benchmarks() {
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"
  local enabled
  enabled=$(jq -r '.performance.checks.api_benchmarks.enabled // false' "$config_file" 2>/dev/null)

  if [ "$enabled" != "true" ]; then
    log::info "⚡ API benchmarks: disabled"
    return 0
  fi

  log::info "⚡ Running API benchmarks..."

  local language dir timeout requirement
  language=$(jq -r '.performance.checks.api_benchmarks.language // "go"' "$config_file")
  dir=$(jq -r '.performance.checks.api_benchmarks.dir // "api"' "$config_file")
  timeout=$(jq -r '.performance.checks.api_benchmarks.timeout // 180' "$config_file")
  requirement=$(jq -r '.performance.checks.api_benchmarks.requirement // ""' "$config_file")

  local benchmark_dir="${TESTING_PHASE_SCENARIO_DIR}/${dir}"

  if [ ! -d "$benchmark_dir" ]; then
    log::warning "Benchmark directory not found: $dir"
    testing::phase::add_test skipped
    return 0
  fi

  cd "$benchmark_dir"

  case "$language" in
    go)
      if ! command -v go >/dev/null 2>&1; then
        log::warning "Go not available; skipping benchmarks"
        testing::phase::add_test skipped
        cd - >/dev/null
        return 0
      fi

      local bench_output="/tmp/benchmark-${TESTING_PHASE_SCENARIO_NAME}.log"

      if go test -bench=. -benchmem -timeout "${timeout}s" -run=^$ ./... 2>&1 | tee "$bench_output"; then
        log::success "✅ Go benchmarks completed"

        # Display benchmark results (top 10)
        if grep -q "Benchmark" "$bench_output"; then
          log::info "Benchmark results (top 10):"
          grep "Benchmark" "$bench_output" | head -10 | while read -r line; do
            log::info "  $line"
          done
        fi

        [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Go benchmarks completed"
        testing::phase::add_test passed
        cd - >/dev/null
        return 0
      else
        log::error "❌ Go benchmarks failed"
        [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status failed --evidence "Go benchmarks failed"
        testing::phase::add_test failed
        cd - >/dev/null
        return 1
      fi
      ;;
    node|javascript|typescript)
      log::warning "Node.js benchmarks not yet implemented"
      testing::phase::add_test skipped
      cd - >/dev/null
      return 0
      ;;
    python)
      log::warning "Python benchmarks not yet implemented"
      testing::phase::add_test skipped
      cd - >/dev/null
      return 0
      ;;
    *)
      log::warning "Unsupported benchmark language: $language"
      testing::phase::add_test skipped
      cd - >/dev/null
      return 0
      ;;
  esac
}

# Test API endpoint response times
_performance_run_response_time() {
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"
  local enabled
  enabled=$(jq -r '.performance.checks.response_time.enabled // false' "$config_file" 2>/dev/null)

  if [ "$enabled" != "true" ]; then
    log::info "⏱️  Response time: disabled"
    return 0
  fi

  log::info "⏱️  Testing response times..."

  # Get API port from scenario
  local api_port
  if ! api_port=$(vrooli scenario port "${TESTING_PHASE_SCENARIO_NAME}" API_PORT 2>/dev/null); then
    log::warning "Unable to determine API_PORT; skipping response time tests"
    testing::phase::add_test skipped
    return 0
  fi

  if [ -z "$api_port" ] || [ "$api_port" = "Error" ]; then
    log::warning "Invalid API_PORT: '$api_port'; skipping response time tests"
    testing::phase::add_test skipped
    return 0
  fi

  local endpoints_json
  endpoints_json=$(jq -c '.performance.checks.response_time.endpoints // []' "$config_file")

  if [ "$endpoints_json" = "[]" ]; then
    log::warning "Response time enabled but no endpoints defined"
    testing::phase::add_test skipped
    return 0
  fi

  local endpoint_count
  endpoint_count=$(echo "$endpoints_json" | jq 'length')
  local failed=0
  local warnings=0

  for ((i=0; i<endpoint_count; i++)); do
    local endpoint
    endpoint=$(echo "$endpoints_json" | jq ".[$i]")

    local path max_ms warn_ms iterations requirement
    path=$(echo "$endpoint" | jq -r '.path')
    max_ms=$(echo "$endpoint" | jq -r '.max_ms')
    warn_ms=$(echo "$endpoint" | jq -r '.warn_ms // (.max_ms * 0.7 | floor)')
    iterations=$(echo "$endpoint" | jq -r '.iterations // 10')
    requirement=$(echo "$endpoint" | jq -r '.requirement // ""')

    local url="http://localhost:${api_port}${path}"
    local total_time=0
    local successful_requests=0

    # Run multiple iterations and average
    for ((j=0; j<iterations; j++)); do
      local response_time http_code
      response_time=$(curl -s -w "%{time_total}" -o /dev/null -w "%{http_code}" "$url" 2>/dev/null | tail -1)
      http_code="${response_time: -3}"

      # Only count successful requests (2xx status)
      if [[ "$http_code" =~ ^2[0-9]{2}$ ]]; then
        response_time=$(curl -s -w "%{time_total}" -o /dev/null "$url" 2>/dev/null || echo "999")
        total_time=$(echo "$total_time + $response_time" | bc 2>/dev/null || echo "$total_time")
        ((successful_requests++))
      fi
    done

    if [ $successful_requests -eq 0 ]; then
      log::error "❌ $path: All requests failed"
      testing::phase::add_error "Endpoint unreachable: $path"
      [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status failed --evidence "Endpoint unreachable"
      ((failed++))
      continue
    fi

    # Calculate average time in milliseconds
    local avg_time_s avg_time_ms
    avg_time_s=$(echo "scale=3; $total_time / $successful_requests" | bc 2>/dev/null || echo "999")
    avg_time_ms=$(echo "$avg_time_s * 1000 / 1" | bc 2>/dev/null || echo "999")

    # Compare against thresholds
    if [ "$avg_time_ms" -gt "$max_ms" ]; then
      log::error "❌ $path: ${avg_time_ms}ms exceeds limit (${max_ms}ms)"
      testing::phase::add_error "Response time too high: ${path} ${avg_time_ms}ms > ${max_ms}ms"
      [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status failed --evidence "Response time ${avg_time_ms}ms > ${max_ms}ms"
      ((failed++))
    elif [ "$avg_time_ms" -gt "$warn_ms" ]; then
      log::warning "⚠️  $path: ${avg_time_ms}ms approaching limit (warn: ${warn_ms}ms, max: ${max_ms}ms)"
      testing::phase::add_warning "Response time near limit: ${path} ${avg_time_ms}ms"
      [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Response time ${avg_time_ms}ms (warn threshold)"
      ((warnings++))
    else
      log::success "✅ $path: ${avg_time_ms}ms (limit: ${max_ms}ms, ${successful_requests}/${iterations} successful)"
      [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Response time ${avg_time_ms}ms"
    fi
  done

  if [ $failed -gt 0 ]; then
    testing::phase::add_test failed
    return 1
  else
    testing::phase::add_test passed
    return 0
  fi
}

# Test concurrent request handling
_performance_run_concurrent_load() {
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"
  local enabled
  enabled=$(jq -r '.performance.checks.concurrent_load.enabled // false' "$config_file" 2>/dev/null)

  if [ "$enabled" != "true" ]; then
    log::info "🔀 Concurrent load: disabled"
    return 0
  fi

  if ! command -v parallel >/dev/null 2>&1; then
    log::warning "GNU parallel not available; skipping concurrent load test"
    log::info "Install GNU parallel: apt-get install parallel (or brew install parallel)"
    testing::phase::add_test skipped
    return 0
  fi

  log::info "🔀 Testing concurrent load..."

  # Get API port
  local api_port
  if ! api_port=$(vrooli scenario port "${TESTING_PHASE_SCENARIO_NAME}" API_PORT 2>/dev/null); then
    log::warning "Unable to determine API_PORT; skipping concurrent load test"
    testing::phase::add_test skipped
    return 0
  fi

  local requests concurrency endpoint min_success_rate requirement
  requests=$(jq -r '.performance.checks.concurrent_load.requests // 100' "$config_file")
  concurrency=$(jq -r '.performance.checks.concurrent_load.concurrency // 10' "$config_file")
  endpoint=$(jq -r '.performance.checks.concurrent_load.endpoint // "/health"' "$config_file")
  min_success_rate=$(jq -r '.performance.checks.concurrent_load.min_success_rate // 0.95' "$config_file")
  requirement=$(jq -r '.performance.checks.concurrent_load.requirement // ""' "$config_file")

  local url="http://localhost:${api_port}${endpoint}"
  local output_file="/tmp/concurrent-test-${TESTING_PHASE_SCENARIO_NAME}.log"

  log::info "Sending $requests concurrent requests to $endpoint (concurrency: $concurrency)..."

  # Run concurrent requests
  seq "$requests" | parallel --no-notice -j "$concurrency" "curl -s -w '\\n%{http_code}\\n' '$url'" > "$output_file" 2>&1

  # Count successful responses (2xx status codes)
  local success_count total_count success_rate
  success_count=$(grep -E '^2[0-9]{2}$' "$output_file" 2>/dev/null | wc -l || echo 0)
  total_count=$requests
  success_rate=$(echo "scale=3; $success_count / $total_count" | bc 2>/dev/null || echo "0")

  log::info "Success rate: ${success_rate} ($success_count/$total_count)"

  # Check against minimum success rate
  if (( $(echo "$success_rate >= $min_success_rate" | bc -l 2>/dev/null || echo 0) )); then
    log::success "✅ Concurrent load test passed (${success_rate} ≥ ${min_success_rate})"
    [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Concurrent load ${success_rate} success rate"
    testing::phase::add_test passed
    return 0
  else
    log::error "❌ Concurrent load test failed (${success_rate} < ${min_success_rate})"
    [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status failed --evidence "Concurrent load ${success_rate} < ${min_success_rate}"
    testing::phase::add_test failed
    return 1
  fi
}

# Monitor memory usage of running process
_performance_run_memory_usage() {
  local config_file="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/testing.json"
  local enabled
  enabled=$(jq -r '.performance.checks.memory_usage.enabled // false' "$config_file" 2>/dev/null)

  if [ "$enabled" != "true" ]; then
    log::info "💾 Memory usage: disabled"
    return 0
  fi

  log::info "💾 Checking memory usage..."

  local process_pattern max_mb warn_mb requirement
  process_pattern=$(jq -r '.performance.checks.memory_usage.process_pattern // ""' "$config_file")
  max_mb=$(jq -r '.performance.checks.memory_usage.max_mb // 500' "$config_file")
  warn_mb=$(jq -r '.performance.checks.memory_usage.warn_mb // (.max_mb * 0.8 | floor)' "$config_file")
  requirement=$(jq -r '.performance.checks.memory_usage.requirement // ""' "$config_file")

  if [ -z "$process_pattern" ]; then
    log::warning "Memory usage enabled but no process_pattern defined"
    testing::phase::add_test skipped
    return 0
  fi

  # Check if process is running
  if ! pgrep -f "$process_pattern" >/dev/null 2>&1; then
    log::warning "Process not running: $process_pattern"
    testing::phase::add_test skipped
    return 0
  fi

  # Get memory usage (RSS in MB)
  local pid mem_usage
  pid=$(pgrep -f "$process_pattern" | head -1)
  mem_usage=$(ps -p "$pid" -o rss= 2>/dev/null | awk '{printf "%.0f", $1/1024}' || echo "0")

  log::info "Process memory usage: ${mem_usage}MB (PID: $pid)"

  # Compare against thresholds
  if [ "$mem_usage" -gt "$max_mb" ]; then
    log::error "❌ Memory usage exceeds limit: ${mem_usage}MB > ${max_mb}MB"
    testing::phase::add_error "Memory usage too high: ${mem_usage}MB > ${max_mb}MB"
    [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status failed --evidence "Memory usage ${mem_usage}MB > ${max_mb}MB"
    testing::phase::add_test failed
    return 1
  elif [ "$mem_usage" -gt "$warn_mb" ]; then
    log::warning "⚠️  Memory usage approaching limit: ${mem_usage}MB (warn: ${warn_mb}MB, max: ${max_mb}MB)"
    testing::phase::add_warning "Memory usage near limit: ${mem_usage}MB"
    [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Memory usage ${mem_usage}MB (warn threshold)"
    testing::phase::add_test passed
    return 0
  else
    log::success "✅ Memory usage within limits: ${mem_usage}MB (max: ${max_mb}MB)"
    [ -n "$requirement" ] && testing::phase::add_requirement --id "$requirement" --status passed --evidence "Memory usage ${mem_usage}MB"
    testing::phase::add_test passed
    return 0
  fi
}

# Fallback for scenarios without testing.json
_performance_basic_checks() {
  log::info "Running basic performance checks (no .vrooli/testing.json found)..."

  # Check for standalone Lighthouse config (old pattern)
  local lighthouse_path="${TESTING_PHASE_SCENARIO_DIR}/.vrooli/lighthouse.json"

  if [ -f "$lighthouse_path" ]; then
    log::info "Found standalone Lighthouse config"
    source "${TESTING_PHASE_APP_ROOT}/scripts/scenarios/testing/lighthouse/runner.sh"

    if lighthouse::run_audits; then
      log::success "✅ Lighthouse audits passed"
      testing::phase::add_test passed
      return 0
    else
      log::error "❌ Lighthouse audits failed"
      testing::phase::add_test failed
      return 1
    fi
  else
    log::info "No performance checks configured (create .vrooli/testing.json or .vrooli/lighthouse.json)"
    testing::phase::add_test skipped
    return 0
  fi
}

# Export main function
export -f testing::performance::validate_all
