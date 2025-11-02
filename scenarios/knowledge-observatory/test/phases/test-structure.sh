#!/bin/bash
# Structure validation for knowledge-observatory using shared helpers
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "45s"

# Required filesystem layout
required_dirs=(api cli data initialization test ui)
if testing::phase::check_directories "${required_dirs[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

required_files=(
  .vrooli/service.json
  PRD.md
  README.md
  api/main.go
  api/go.mod
  cli/install.sh
  initialization/postgres/schema.sql
  initialization/postgres/seed.sql
)
if testing::phase::check_files "${required_files[@]}"; then
  testing::phase::add_test passed
else
  testing::phase::add_test failed
fi

# Validate service.json structure
if testing::phase::check "service.json is valid" bash -c 'jq empty .vrooli/service.json >/dev/null'; then
  lifecycle_version=$(jq -r '.lifecycle.version // ""' .vrooli/service.json)
  if [ "$lifecycle_version" != "2.0.0" ]; then
    testing::phase::add_warning "Lifecycle version should be 2.0.0 (found ${lifecycle_version:-unknown})"
  fi
else
  testing::phase::end_with_summary "service.json validation failed"
fi

# Go source should compile
if testing::phase::check "Go API builds" bash -c 'cd api && go build -o /tmp/knowledge-observatory-struct .'; then
  rm -f /tmp/knowledge-observatory-struct
else
  testing::phase::end_with_summary "Go build failed"
fi

# CLI artefacts
if [ -f cli/knowledge-observatory ]; then
  testing::phase::add_test passed
  log::success "CLI binary present"
else
  testing::phase::add_warning "CLI binary missing; run setup to build it"
fi

# UI manifest
if [ -f ui/package.json ]; then
  testing::phase::add_test passed
  log::success "UI package.json present"
else
  testing::phase::add_warning "UI package.json missing"
fi

# Ensure docs are non-empty
for doc in README.md PRD.md PROBLEMS.md; do
  if testing::phase::check "${doc} exists and is not empty" bash -c "[ -s '$doc' ]"; then
    :
  else
    testing::phase::end_with_summary "Documentation check failed"
  fi
done

testing::phase::end_with_summary "Structure validation completed"
