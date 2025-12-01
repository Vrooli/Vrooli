#!/usr/bin/env bash
# Scenario-local phased test orchestrator for Test Genie.

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/runtime.sh"

if [[ -n "${TEST_GENIE_ORCHESTRATOR_LOADED:-}" ]]; then
  return
fi
TEST_GENIE_ORCHESTRATOR_LOADED=1

declare -Ar TG_ORCHESTRATOR_PRESETS=(
  [quick]="structure unit"
  [smoke]="structure integration"
  [comprehensive]="structure dependencies unit integration business performance"
)

tg::orchestrator::usage() {
  cat <<'USAGE'
Usage: ./run-tests.sh [options]

Options:
  --preset <quick|smoke|comprehensive>  Run a predefined subset of phases
  --only <phase1,phase2>                Run only the specified phases
  --skip <phase1,phase2>                Skip the specified phases
  --list                                List available phases and exit
  --fail-fast                           Stop after the first failed phase
  --help                                Show this help message
USAGE
}

tg::orchestrator::discover_phases() {
  local phase_dir="${TEST_GENIE_SCENARIO_DIR}/test/phases"
  if [[ ! -d "$phase_dir" ]]; then
    tg::fail "Phase directory '${phase_dir}' is missing"
  fi
  local -a entries=()
  while IFS= read -r -d '' script; do
    local base
    base="$(basename "$script")"
    local phase="${base#test-}"
    phase="${phase%.sh}"
    entries+=("${phase}|${script}")
  done < <(find "$phase_dir" -maxdepth 1 -type f -name 'test-*.sh' -print0 | sort -z)
  printf "%s\n" "${entries[@]}"
}

tg::orchestrator::normalize_list() {
  local raw="${1:-}"
  raw="${raw//,/ }"
  for token in $raw; do
    printf "%s\n" "$token"
  done
}

tg::orchestrator::resolve_run_list() {
  local -n _dest="$1"
  local -n _order="$2"
  local only_list="${3:-}"
  local skip_list="${4:-}"
  local preset="${5:-}"

  local -A phase_selection=()
  local -a resolved=()

  if [[ -n "$preset" ]]; then
    local preset_phases="${TG_ORCHESTRATOR_PRESETS[$preset]:-}"
    if [[ -z "$preset_phases" ]]; then
      tg::fail "Unknown preset '${preset}'. Valid presets: quick, smoke, comprehensive"
    fi
    if [[ -z "$only_list" ]]; then
      only_list="$preset_phases"
    fi
  fi

  local -A allowed=()
  local phase
  for phase in "${_order[@]}"; do
    allowed["$phase"]=1
  done

  if [[ -n "$only_list" ]]; then
    while IFS= read -r phase; do
      [[ -z "$phase" ]] && continue
      if [[ -z "${allowed[$phase]:-}" ]]; then
        tg::fail "Phase '${phase}' is not defined in test/phases"
      fi
      phase_selection["$phase"]=1
    done < <(tg::orchestrator::normalize_list "$only_list")
  else
    for phase in "${_order[@]}"; do
      phase_selection["$phase"]=1
    done
  fi

  if [[ -n "$skip_list" ]]; then
    while IFS= read -r phase; do
      [[ -z "$phase" ]] && continue
      unset "phase_selection[$phase]"
    done < <(tg::orchestrator::normalize_list "$skip_list")
  fi

  for phase in "${_order[@]}"; do
    if [[ -n "${phase_selection[$phase]:-}" ]]; then
      resolved+=("$phase")
    fi
  done

  if [[ ${#resolved[@]} -eq 0 ]]; then
    tg::fail "No phases selected for execution"
  fi

  _dest=("${resolved[@]}")
}

tg::orchestrator::print_phase_list() {
  local -n _order="$1"
  tg::log_info "Available phases:"
  local phase
  for phase in "${_order[@]}"; do
    printf "  - %s\n" "$phase"
  done
}

tg::orchestrator::run() {
  local preset=""
  local only=""
  local skip=""
  local list_only=false
  local fail_fast=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --preset)
        preset="${2:-}"
        shift
        ;;
      --only)
        only="${2:-}"
        shift
        ;;
      --skip)
        skip="${2:-}"
        shift
        ;;
      --list)
        list_only=true
        ;;
      --fail-fast)
        fail_fast=true
        ;;
      --help|-h)
        tg::orchestrator::usage
        return 0
        ;;
      *)
        tg::fail "Unknown option '$1'"
        ;;
    esac
    shift || true
  done

  local -a phase_entries=()
  mapfile -t phase_entries < <(tg::orchestrator::discover_phases)
  if [[ ${#phase_entries[@]} -eq 0 ]]; then
    tg::fail "No phase scripts found in test/phases"
  fi

  declare -A scripts=()
  local -a order=()
  local entry
  for entry in "${phase_entries[@]}"; do
    local phase="${entry%%|*}"
    local script="${entry#*|}"
    scripts["$phase"]="$script"
    order+=("$phase")
  done

  if [[ "$list_only" = true ]]; then
    tg::orchestrator::print_phase_list order
    return 0
  fi

  local -a selected=()
  tg::orchestrator::resolve_run_list selected order "$only" "$skip" "$preset"

  tg::log_info "Running phases: ${selected[*]}"
  local -a results=()
  local phase
  local any_failure=false

  for phase in "${selected[@]}"; do
    local script="${scripts[$phase]}"
    tg::log_info "⏱  Phase '${phase}' started"
    local start_ts
    start_ts="$(date +%s)"
    set +e
    bash "$script"
    local exit_code=$?
    set -e
    local duration=$(( $(date +%s) - start_ts ))
    if [[ $exit_code -eq 0 ]]; then
      results+=("${phase}|passed|${duration}")
      tg::log_success "✓ Phase '${phase}' completed in ${duration}s"
    else
      results+=("${phase}|failed|${duration}")
      tg::log_error "✗ Phase '${phase}' failed in ${duration}s"
      any_failure=true
      if [[ "$fail_fast" = true ]]; then
        break
      fi
    fi
  done

  tg::log_info "Phase summary:"
  for entry in "${results[@]}"; do
    IFS='|' read -r phase status duration <<< "$entry"
    printf "  - %-12s status=%-7s duration=%ss\n" "$phase" "$status" "$duration"
  done

  if [[ "$any_failure" = true ]]; then
    return 1
  fi
  return 0
}
