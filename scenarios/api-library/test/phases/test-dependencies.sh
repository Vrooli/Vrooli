#!/bin/bash
# Ensures language runtimes and manifests resolve without mutating the workspace.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

run_go_dependency_checks() {
  if [ ! -f "api/go.mod" ]; then
    testing::phase::add_warning "Go module not found; skipping Go dependency checks"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    testing::phase::add_warning "Go toolchain missing; cannot verify Go dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  local go_dir="$TESTING_PHASE_SCENARIO_DIR/api"
  testing::phase::check "Go module graph resolves" bash -c "cd '$go_dir' && go list ./... >/dev/null"
}

run_node_dependency_checks() {
  if [ ! -f "ui/package.json" ]; then
    testing::phase::add_warning "UI package.json not found; skipping Node dependency checks"
    testing::phase::add_test skipped
    return 0
  fi

  if ! command -v node >/dev/null 2>&1; then
    testing::phase::add_warning "Node runtime missing; cannot verify frontend dependencies"
    testing::phase::add_test skipped
    return 0
  fi

  local ui_dir="$TESTING_PHASE_SCENARIO_DIR/ui"
  local package_manager
  package_manager=$(bash -c "cd '$ui_dir' && node -e \"const pkg=require('./package.json');process.stdout.write(pkg.packageManager||'')\"" 2>/dev/null)

  local manager_cmd
  if [[ "${package_manager}" == pnpm* ]]; then
    if ! command -v pnpm >/dev/null 2>&1; then
      testing::phase::add_warning "pnpm not installed; skipping pnpm dependency check"
      testing::phase::add_test skipped
      return 0
    fi
    manager_cmd="pnpm list --depth -1"
  elif [[ "${package_manager}" == yarn* ]]; then
    if ! command -v yarn >/dev/null 2>&1; then
      testing::phase::add_warning "yarn not installed; skipping yarn dependency check"
      testing::phase::add_test skipped
      return 0
    fi
    manager_cmd="yarn install --immutable --dry-run"
  else
    if ! command -v npm >/dev/null 2>&1; then
      testing::phase::add_warning "npm CLI missing; cannot verify Node dependencies"
      testing::phase::add_test skipped
      return 0
    fi
    manager_cmd="npm install --dry-run"
  fi

  testing::phase::check "${manager_cmd%% *}" bash -c "cd '$ui_dir' && ${manager_cmd} >/dev/null"
}

run_go_dependency_checks || true
run_node_dependency_checks || true

testing::phase::end_with_summary "Dependency validation completed"
